package aiscript

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
)

// LoadedSample is one imitation JSON file on disk plus its path.
type LoadedSample struct {
	Path   string
	Sample ImitationSample
}

// LoadImitationSamples reads every *.json under dir (non-recursive).
// Corrupt or short samples are skipped; empty dir → empty slice, no error.
func LoadImitationSamples(dir string) ([]LoadedSample, error) {
	if dir == "" {
		return nil, fmt.Errorf("aiscript: imitation dir required")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("aiscript: read imitation dir: %w", err)
	}
	out := make([]LoadedSample, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		path := filepath.Join(dir, e.Name())
		raw, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var sample ImitationSample
		if err := json.Unmarshal(raw, &sample); err != nil {
			continue
		}
		if len(sample.Actions) < 2 {
			continue
		}
		if sample.DurationMs <= 0 {
			sample.DurationMs = sample.Actions[len(sample.Actions)-1].At - sample.Actions[0].At
		}
		out = append(out, LoadedSample{Path: path, Sample: sample})
	}
	return out, nil
}

// CountValidSamples returns how many usable imitation files sit under dir.
func CountValidSamples(dir string) int {
	samples, err := LoadImitationSamples(dir)
	if err != nil {
		return 0
	}
	return len(samples)
}

// PickBestSample chooses the closest duration match. Prefer qdPassed=true when
// durations are similar (within 25%). targetDurationMs ≤ 0 → first QD-passed
// or first sample.
func PickBestSample(samples []LoadedSample, targetDurationMs int64) (LoadedSample, error) {
	if len(samples) == 0 {
		return LoadedSample{}, fmt.Errorf("aiscript: no imitation samples")
	}
	if targetDurationMs <= 0 {
		for _, s := range samples {
			if s.Sample.QDPassed != nil && *s.Sample.QDPassed {
				return s, nil
			}
		}
		return samples[0], nil
	}
	type scored struct {
		s     LoadedSample
		dist  int64
		qdOk  bool
		index int
	}
	ranked := make([]scored, 0, len(samples))
	for i, s := range samples {
		dur := s.Sample.DurationMs
		if dur <= 0 {
			dur = 1
		}
		d := dur - targetDurationMs
		if d < 0 {
			d = -d
		}
		qd := s.Sample.QDPassed != nil && *s.Sample.QDPassed
		ranked = append(ranked, scored{s: s, dist: d, qdOk: qd, index: i})
	}
	sort.SliceStable(ranked, func(i, j int) bool {
		// Prefer smaller duration distance; break ties with QD pass, then index.
		if ranked[i].dist != ranked[j].dist {
			return ranked[i].dist < ranked[j].dist
		}
		if ranked[i].qdOk != ranked[j].qdOk {
			return ranked[i].qdOk
		}
		return ranked[i].index < ranked[j].index
	})
	best := ranked[0]
	// If a QD-passed sample is within 25% of the best distance, prefer it.
	for _, c := range ranked[1:] {
		if !c.qdOk {
			continue
		}
		limit := best.dist + best.dist/4
		if best.dist == 0 {
			limit = targetDurationMs / 4
			if limit < 500 {
				limit = 500
			}
		}
		if c.dist <= limit {
			return c.s, nil
		}
		break
	}
	return best.s, nil
}

// StretchActions linearly remaps action times so the span matches targetMs.
// Positions unchanged. targetMs ≤ 0 or single-span source → copy as-is.
func StretchActions(actions []Action, targetMs int64) []Action {
	if len(actions) == 0 {
		return nil
	}
	out := make([]Action, len(actions))
	copy(out, actions)
	if len(out) < 2 || targetMs <= 0 {
		return out
	}
	t0 := out[0].At
	t1 := out[len(out)-1].At
	span := t1 - t0
	if span <= 0 {
		return out
	}
	if span == targetMs {
		// Normalize to start at 0 when already matching span.
		if t0 == 0 {
			return out
		}
		for i := range out {
			out[i].At -= t0
		}
		return out
	}
	scale := float64(targetMs) / float64(span)
	for i := range out {
		rel := out[i].At - t0
		out[i].At = int64(math.Round(float64(rel) * scale))
		if out[i].Pos < 0 {
			out[i].Pos = 0
		}
		if out[i].Pos > 100 {
			out[i].Pos = 100
		}
	}
	// Ensure strictly non-decreasing times after rounding.
	for i := 1; i < len(out); i++ {
		if out[i].At < out[i-1].At {
			out[i].At = out[i-1].At
		}
	}
	out[len(out)-1].At = targetMs
	return out
}
