package funscript

import (
	"fmt"
	"math"
	"sort"
)

const (
	// DefaultFillGapMs: fill segments longer than this when MaxGapMs is 0
	// and the Script-Doctor-style auto threshold is smaller.
	DefaultFillGapMs int64 = 800
	// DefaultFillStepMs: spacing between inserted points when no audio Hz hint.
	DefaultFillStepMs int64 = 100
)

// ImproveOpts controls post-generate script polish (FunGen-like end step).
// Mutators only — never invent a full curve from audio.
type ImproveOpts struct {
	// StartMs / EndMs trim the script. 0 = no bound on that side.
	// EndMs < 0 also means no end bound.
	StartMs int64
	EndMs   int64
	// FillGaps inserts linear points across long action gaps.
	FillGaps bool
	// MaxGapMs: fill only when Δt exceeds this. 0 = auto (max(median*8, 2000)
	// or DefaultFillGapMs, whichever is smaller — catches doctor-flagged gaps
	// and shorter tracker holes).
	MaxGapMs int64
	// StepMs between inserted points. 0 = DefaultFillStepMs, or derived from
	// AudioHz when AudioHz > 0 (half-period of the audio tempo).
	StepMs int64
	// AudioHz optional tempo hint for fill spacing (from CheckAudioTempo).
	AudioHz float64
}

// ImproveResult summarizes what changed.
type ImproveResult struct {
	Actions      []Action `json:"-"`
	BeforeCount  int      `json:"beforeCount"`
	AfterCount   int      `json:"afterCount"`
	Trimmed      bool     `json:"trimmed"`
	GapsFilled   int      `json:"gapsFilled"`
	PointsAdded  int      `json:"pointsAdded"`
	FillGapMs    int64    `json:"fillGapMs"`
	FillStepMs   int64    `json:"fillStepMs"`
}

// TrimActions keeps actions in [startMs, endMs] (inclusive).
// startMs < 0 or endMs < 0 means unbounded on that side.
// Ensures at least two points; interpolates endpoints when the cut falls
// between existing actions.
func TrimActions(actions []Action, startMs, endMs int64) ([]Action, error) {
	if len(actions) < 2 {
		return nil, fmt.Errorf("funscript: trim needs at least 2 points")
	}
	sorted := append([]Action(nil), actions...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].At < sorted[j].At })

	lo := sorted[0].At
	hi := sorted[len(sorted)-1].At
	if startMs < 0 {
		startMs = lo
	}
	if endMs < 0 {
		endMs = hi
	}
	if endMs < startMs {
		startMs, endMs = endMs, startMs
	}
	if startMs <= lo && endMs >= hi {
		return sorted, nil
	}

	out := make([]Action, 0, len(sorted)+2)
	// Leading endpoint
	if startMs > lo {
		out = append(out, Action{At: startMs, Pos: clampPos(interpAt(sorted, startMs))})
	}
	for _, a := range sorted {
		if a.At < startMs || a.At > endMs {
			continue
		}
		if len(out) > 0 && out[len(out)-1].At == a.At {
			out[len(out)-1] = a
			continue
		}
		out = append(out, a)
	}
	if endMs < hi {
		ep := Action{At: endMs, Pos: clampPos(interpAt(sorted, endMs))}
		if len(out) == 0 || out[len(out)-1].At != endMs {
			out = append(out, ep)
		} else {
			out[len(out)-1] = ep
		}
	}
	if len(out) < 2 {
		return nil, fmt.Errorf("funscript: trim would leave fewer than 2 points")
	}
	return out, nil
}

// FillGaps inserts linearly interpolated actions across long time gaps.
// Does not invent motion from audio — optional AudioHz only sets step spacing.
func FillGaps(actions []Action, maxGapMs, stepMs int64, audioHz float64) (out []Action, gapsFilled, pointsAdded int) {
	if len(actions) < 2 {
		return append([]Action(nil), actions...), 0, 0
	}
	sorted := append([]Action(nil), actions...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].At < sorted[j].At })

	if maxGapMs <= 0 {
		maxGapMs = autoFillGapThreshold(sorted)
	}
	if stepMs <= 0 {
		stepMs = DefaultFillStepMs
		if audioHz > 0.2 && audioHz < 5 {
			// Half-period so a full stroke (up+down) can land near audio tempo.
			half := int64(math.Round(1000.0 / (2.0 * audioHz)))
			if half >= 40 && half <= 800 {
				stepMs = half
			}
		}
	}
	if stepMs < 20 {
		stepMs = 20
	}

	out = make([]Action, 0, len(sorted)*2)
	out = append(out, sorted[0])
	for i := 1; i < len(sorted); i++ {
		prev := out[len(out)-1]
		next := sorted[i]
		dt := next.At - prev.At
		if dt > maxGapMs && dt > stepMs {
			gapsFilled++
			n := int(dt / stepMs)
			for k := 1; k < n; k++ {
				t := prev.At + int64(k)*stepMs
				if t >= next.At {
					break
				}
				frac := float64(t-prev.At) / float64(dt)
				pos := int(math.Round(float64(prev.Pos) + frac*float64(next.Pos-prev.Pos)))
				out = append(out, Action{At: t, Pos: clampPos(pos)})
				pointsAdded++
			}
		}
		if out[len(out)-1].At == next.At {
			out[len(out)-1] = next
		} else {
			out = append(out, next)
		}
	}
	return out, gapsFilled, pointsAdded
}

// ImproveScript applies trim then fill-gaps. Order matches FunGen-like polish:
// cut ends first, then bridge holes.
func ImproveScript(actions []Action, opts ImproveOpts) (ImproveResult, error) {
	res := ImproveResult{BeforeCount: len(actions)}
	if len(actions) < 2 {
		return res, fmt.Errorf("funscript: improve needs at least 2 points")
	}
	cur := append([]Action(nil), actions...)

	startMs, endMs := opts.StartMs, opts.EndMs
	doTrim := startMs > 0 || endMs > 0
	if doTrim {
		if startMs <= 0 {
			startMs = -1
		}
		if endMs <= 0 {
			endMs = -1
		}
		trimmed, err := TrimActions(cur, startMs, endMs)
		if err != nil {
			return res, err
		}
		if len(trimmed) != len(cur) ||
			(len(trimmed) > 0 && len(cur) > 0 &&
				(trimmed[0].At != cur[0].At || trimmed[len(trimmed)-1].At != cur[len(cur)-1].At)) {
			res.Trimmed = true
		}
		cur = trimmed
	}

	if opts.FillGaps {
		maxGap := opts.MaxGapMs
		if maxGap <= 0 {
			maxGap = autoFillGapThreshold(cur)
		}
		step := opts.StepMs
		filled, nGaps, nPts := FillGaps(cur, maxGap, step, opts.AudioHz)
		res.GapsFilled = nGaps
		res.PointsAdded = nPts
		res.FillGapMs = maxGap
		if step <= 0 {
			step = DefaultFillStepMs
			if opts.AudioHz > 0.2 && opts.AudioHz < 5 {
				half := int64(math.Round(1000.0 / (2.0 * opts.AudioHz)))
				if half >= 40 && half <= 800 {
					step = half
				}
			}
		}
		res.FillStepMs = step
		cur = filled
	}

	res.Actions = cur
	res.AfterCount = len(cur)
	return res, nil
}

func autoFillGapThreshold(actions []Action) int64 {
	var dts []float64
	for i := 1; i < len(actions); i++ {
		d := float64(actions[i].At - actions[i-1].At)
		if d > 0 {
			dts = append(dts, d)
		}
	}
	if len(dts) == 0 {
		return DefaultFillGapMs
	}
	med := medianFloat(dts)
	doctor := int64(math.Max(med*8, 2000))
	if doctor < DefaultFillGapMs {
		return doctor
	}
	// Prefer filling shorter tracker holes than the doctor threshold alone.
	return DefaultFillGapMs
}

func interpAt(actions []Action, t int64) int {
	if len(actions) == 0 {
		return 50
	}
	if t <= actions[0].At {
		return actions[0].Pos
	}
	last := actions[len(actions)-1]
	if t >= last.At {
		return last.Pos
	}
	for i := 1; i < len(actions); i++ {
		a, b := actions[i-1], actions[i]
		if t >= a.At && t <= b.At {
			if b.At == a.At {
				return b.Pos
			}
			frac := float64(t-a.At) / float64(b.At-a.At)
			return int(math.Round(float64(a.Pos) + frac*float64(b.Pos-a.Pos)))
		}
	}
	return last.Pos
}

func clampPos(p int) int {
	if p < 0 {
		return 0
	}
	if p > 100 {
		return 100
	}
	return p
}
