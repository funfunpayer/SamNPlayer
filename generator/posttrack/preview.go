package posttrack

import (
	"fmt"
	"math"
)

// PreviewRequest holds Expert-tuning knobs for a live postprocess estimate.
// Values mirror GUI Expert fields; PeakProminence 0 = off (no prominence gate).
type PreviewRequest struct {
	SmoothWindow      int
	MinPeakDistanceMs int
	PeakProminence    float64
	RDPTolerance      float64
	AdaptiveError     float64
	MaxSpeed          float64
	// DurationMs of the synthetic probe clip (default 10s).
	DurationMs float64
}

// PreviewResult summarises what the knobs would do on a fixed synthetic probe.
// Not a substitute for tracking — only live feedback for smooth / prominence /
// peak spacing / RDP (DeepFunGen-style postprocess displays).
type PreviewResult struct {
	KeyframeCount int     `json:"keyframeCount"`
	PeakCount     int     `json:"peakCount"`
	ValleyCount   int     `json:"valleyCount"`
	MeanHz        float64 `json:"meanHz"`
	Hint          string  `json:"hint"`
}

// PreviewStats runs PositionsToActions on a fixed multi-stroke probe signal
// so the GUI can show live-ish keyframe/peak feedback when Expert knobs change.
func PreviewStats(req PreviewRequest) (PreviewResult, error) {
	opts := DefaultOptions()
	if req.SmoothWindow > 0 {
		opts.SmoothWindow = req.SmoothWindow
	}
	if req.MinPeakDistanceMs > 0 {
		opts.MinPeakDistanceMs = req.MinPeakDistanceMs
	}
	opts.PeakProminence = req.PeakProminence
	opts.RDPTolerance = req.RDPTolerance
	opts.AdaptiveError = req.AdaptiveError

	dur := req.DurationMs
	if dur <= 0 {
		dur = 10000
	}
	ts, pos := syntheticProbe(dur, 30.0)
	res, err := PositionsToActions(ts, pos, opts)
	if err != nil {
		return PreviewResult{}, err
	}

	actions := res.Actions
	if req.MaxSpeed > 0 {
		actions, _ = LimitSpeed(actions, req.MaxSpeed)
	}

	// Peak/valley counts from dense normalised curve (same prominence gate).
	stepMs := medianDiff(ts)
	fps := 1000.0 / math.Max(stepMs, 1.0)
	minDist := int(float64(opts.MinPeakDistanceMs) / 1000.0 * fps)
	if minDist < 1 {
		minDist = 1
	}
	prom := 0.0
	if opts.PeakProminence > 0 {
		span := ptp(res.DensePos)
		if span > 1e-9 {
			prom = span * opts.PeakProminence
		}
	}
	peaks := findPeaks(res.DensePos, minDist, prom)
	neg := make([]float64, len(res.DensePos))
	for i, v := range res.DensePos {
		neg[i] = -v
	}
	valleys := findPeaks(neg, minDist, prom)

	meanHz := 0.0
	if len(actions) >= 2 {
		spanMs := float64(actions[len(actions)-1].At - actions[0].At)
		if spanMs > 0 {
			// Half-cycles ≈ peaks+valleys; report stroke Hz ≈ keyframe rate / 2.
			meanHz = (float64(len(actions)-1) / 2.0) / (spanMs / 1000.0)
		}
	}

	hint := previewHint(req, len(actions), len(peaks))
	return PreviewResult{
		KeyframeCount: len(actions),
		PeakCount:     len(peaks),
		ValleyCount:   len(valleys),
		MeanHz:        meanHz,
		Hint:          hint,
	}, nil
}

func previewHint(req PreviewRequest, keyframes, peaks int) string {
	bits := []string{
		fmt.Sprintf("Probe: %d keyframes, %d peaks", keyframes, peaks),
	}
	if req.PeakProminence <= 0 {
		bits = append(bits, "prominence off (0 = profile default on Create)")
	} else if req.PeakProminence >= 0.4 {
		bits = append(bits, "high prominence — fewer soft strokes kept")
	} else if req.PeakProminence <= 0.15 {
		bits = append(bits, "low prominence — keeps small wiggles")
	}
	if req.SmoothWindow >= 21 {
		bits = append(bits, "wide smooth — calmer, may lag")
	} else if req.SmoothWindow > 0 && req.SmoothWindow <= 5 {
		bits = append(bits, "narrow smooth — more noise")
	}
	if req.MinPeakDistanceMs >= 300 {
		bits = append(bits, "wide peak spacing — slower tempo")
	}
	if req.RDPTolerance > 0 {
		bits = append(bits, fmt.Sprintf("RDP %.1f thins the curve", req.RDPTolerance))
	}
	if req.MaxSpeed > 0 {
		bits = append(bits, fmt.Sprintf("max speed %.0f", req.MaxSpeed))
	}
	out := bits[0]
	for i := 1; i < len(bits); i++ {
		out += " · " + bits[i]
	}
	return out
}

// syntheticProbe builds a noisy multi-Hz stroke-like signal (px-ish amplitude)
// so prominence and smooth window have a visible effect.
func syntheticProbe(durationMs, fps float64) (timestamps []float64, positions []float64) {
	if fps < 1 {
		fps = 30
	}
	n := int(durationMs/1000.0*fps) + 1
	if n < 32 {
		n = 32
	}
	timestamps = make([]float64, n)
	positions = make([]float64, n)
	for i := 0; i < n; i++ {
		t := float64(i) / fps
		timestamps[i] = t * 1000.0
		// Main ~1.2 Hz stroke + softer ~2.4 Hz overtone + tiny jitter.
		main := 40.0 * math.Sin(2*math.Pi*1.2*t)
		soft := 8.0 * math.Sin(2*math.Pi*2.4*t+0.3)
		jitter := 1.5 * math.Sin(2*math.Pi*7.0*t+1.1)
		positions[i] = 100.0 + main + soft + jitter
	}
	return timestamps, positions
}
