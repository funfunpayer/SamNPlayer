package funscript

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
)

// Project is a lightweight SamNPlayer session sidecar (OFS-inspired, JSON).
// Stored next to the video or script as "<name>.snp.json".
type Project struct {
	Version      int     `json:"version"`
	VideoPath    string  `json:"videoPath,omitempty"`
	ScriptPath   string  `json:"scriptPath,omitempty"`
	OffsetMs     int64   `json:"offsetMs,omitempty"`
	SeekMs       int64   `json:"seekMs,omitempty"`
	LoopMarker   *Marker `json:"loopMarker,omitempty"`
	Bookmarks    []Bookmark    `json:"bookmarks,omitempty"`
	Chapters     []ChapterMark `json:"chapters,omitempty"`
	LastOpenedMs int64   `json:"lastOpenedMs,omitempty"`
}

// Marker is a simple time range (playback loop / selection).
type Marker struct {
	StartMs int64 `json:"startMs"`
	EndMs   int64 `json:"endMs"`
}

const projectVersion = 1

// ProjectPathFor returns the sidecar path beside mediaPath.
func ProjectPathFor(mediaPath string) string {
	ext := filepath.Ext(mediaPath)
	base := mediaPath[:len(mediaPath)-len(ext)]
	return base + ".snp.json"
}

// SaveProject writes the sidecar atomically-ish (write then rename via temp).
func SaveProject(path string, p Project) error {
	if p.Version == 0 {
		p.Version = projectVersion
	}
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// LoadProject reads a sidecar.
func LoadProject(path string) (Project, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Project{}, err
	}
	var p Project
	if err := json.Unmarshal(data, &p); err != nil {
		return Project{}, fmt.Errorf("project: ungültiges JSON: %w", err)
	}
	if p.Version == 0 {
		p.Version = projectVersion
	}
	return p, nil
}

// SnapMs snaps t to the nearest frame for fps (OFS frame overlay aid).
// fps <= 0 returns t unchanged.
func SnapMs(t int64, fps float64) int64 {
	if fps <= 0 || math.IsNaN(fps) || math.IsInf(fps, 0) {
		return t
	}
	frame := math.Round(float64(t) * fps / 1000.0)
	return int64(math.Round(frame * 1000.0 / fps))
}

// DeleteRange removes actions with at in [start,end] (inclusive).
// Keeps at least endpoints if the range would empty the script — caller
// should validate; returns error if fewer than 2 points remain.
func DeleteRange(actions []Action, start, end int64) ([]Action, error) {
	if end < start {
		start, end = end, start
	}
	out := make([]Action, 0, len(actions))
	for _, a := range actions {
		if a.At < start || a.At > end {
			out = append(out, a)
		}
	}
	if len(out) < 2 {
		return nil, fmt.Errorf("funscript: Löschen würde weniger als 2 Punkte lassen")
	}
	return out, nil
}

// CapSpeedRange clamps segment intensity in [start,end] to maxIntensity
// by stretching time on too-fast segments; subsequent points shift right.
func CapSpeedRange(actions []Action, start, end int64, maxIntensity float64) ([]Action, error) {
	if maxIntensity <= 0 {
		maxIntensity = DefaultMaxIntensity
	}
	if len(actions) < 2 {
		return nil, fmt.Errorf("funscript: zu wenige Punkte")
	}
	if end < start {
		start, end = end, start
	}
	out := append([]Action(nil), actions...)
	var shift int64
	for i := 0; i < len(out)-1; i++ {
		out[i].At += shift
		nextAt := out[i+1].At + shift
		dt := float64(nextAt - out[i].At)
		if dt <= 0 {
			continue
		}
		// Segment overlaps edit range if it starts before end and ends after start.
		if nextAt < start || out[i].At > end {
			continue
		}
		dpos := math.Abs(float64(out[i+1].Pos - out[i].Pos))
		inten := 500.0 * dpos / dt
		if inten <= maxIntensity {
			continue
		}
		need := 500.0 * dpos / maxIntensity
		extra := int64(math.Ceil(need - dt))
		if extra > 0 {
			shift += extra
		}
	}
	out[len(out)-1].At += shift
	return out, nil
}

// ScaleRangePos scales positions in [start,end] around 50 by factor.
func ScaleRangePos(actions []Action, start, end int64, factor float64) []Action {
	if end < start {
		start, end = end, start
	}
	out := append([]Action(nil), actions...)
	for i := range out {
		if out[i].At < start || out[i].At > end {
			continue
		}
		p := 50 + (float64(out[i].Pos)-50)*factor
		if p < 0 {
			p = 0
		}
		if p > 100 {
			p = 100
		}
		out[i].Pos = int(math.Round(p))
	}
	return out
}
