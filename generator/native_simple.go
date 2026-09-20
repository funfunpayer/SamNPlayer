package generator

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/funfunpayer/SamNPlayer/funscript"
	"github.com/funfunpayer/SamNPlayer/generator/posttrack"
	"github.com/funfunpayer/SamNPlayer/generator/simpletrack"
	"github.com/funfunpayer/SamNPlayer/logging"
)

// SimpleTrackingAvailable is always true: tracking uses videox/ffmpeg (no
// cgo/OpenCV). Runtime still needs ffmpeg on PATH — same as videox tests.
func SimpleTrackingAvailable() bool { return true }

// NativeGoGenerateAvailable reports whether any Python-free generate path
// can run for CSRT-style single-ROI options (OpenCV CSRT and/or simpletrack).
func NativeGoGenerateAvailable() bool {
	return NativeTrackingAvailable() || SimpleTrackingAvailable()
}

// GenerateNativeSimple runs simpletrack (videox NCC) + posttrack without
// Python/OpenCV. Used automatically when NativePipeline is eligible and
// CSRT/OpenCV is not linked — notably Windows release builds. Supports
// single-ROI and Tf/Tj two-point (ROI2).
func GenerateNativeSimple(ctx context.Context, videoPath string, roi ROI, outputPath string, opts Options, onProgress func(line string), onPercent func(pct int)) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if !nativeOptionsEligible(opts, roi) {
		return fmt.Errorf("generator: native simple pipeline not eligible for these options")
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	progress := func(line string) {
		logging.Info("generator/native-simple: " + line)
		if onProgress != nil {
			onProgress(line)
		}
	}
	twoPoint := opts.ROI2.W > 0 && opts.ROI2.H > 0
	if twoPoint {
		progress("Go-Pipeline Zwei-Punkt ohne OpenCV (simpletrack/NCC) — schwächer als CSRT, kein Python")
	} else {
		progress("Go-Pipeline ohne OpenCV (simpletrack/NCC über ffmpeg) — schwächer als CSRT, kein Python")
	}

	start := time.Now()
	axis := opts.Axis
	if axis == "" {
		axis = "auto"
	}
	stOpts := simpletrack.Options{
		MaxFrames:    opts.MaxFrames,
		StartTimeSec: opts.StartTimeSec,
		Axis:         axis,
		Cancel:       func() bool { return ctx.Err() != nil },
		FixedB:       opts.ROI2Fixed,
	}

	var tr simpletrack.Result
	var err error
	if twoPoint {
		tr, err = simpletrack.TrackTwoPoints(ctx, videoPath,
			simpletrack.Rect{X: roi.X, Y: roi.Y, W: roi.W, H: roi.H},
			simpletrack.Rect{X: opts.ROI2.X, Y: opts.ROI2.Y, W: opts.ROI2.W, H: opts.ROI2.H},
			stOpts)
	} else {
		tr, err = simpletrack.TrackROI(ctx, videoPath, simpletrack.Rect{X: roi.X, Y: roi.Y, W: roi.W, H: roi.H}, stOpts)
	}
	if err != nil {
		if errors.Is(err, simpletrack.ErrCanceled) || errors.Is(ctx.Err(), context.Canceled) {
			return context.Canceled
		}
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return context.DeadlineExceeded
		}
		return fmt.Errorf("generator/native-simple: tracking: %w", err)
	}
	progress(fmt.Sprintf("%d Frames getrackt (%dx%d) in %s",
		len(tr.TimestampsMs), tr.Width, tr.Height, time.Since(start).Round(time.Millisecond)))

	ntr := nativeTrackResult{
		TimestampsMs: tr.TimestampsMs,
		Positions:    tr.Positions,
		Width:        tr.Width,
		Height:       tr.Height,
		LostFrames:   tr.Stats.TrackerLostFrames,
		TotalFrames:  tr.Stats.TotalFrames,
		VertRange:    tr.Stats.VerticalRange,
		ValidFrames:  tr.Stats.ValidFrames,
		Confidence:   tr.Stats.Confidence,
		Reason:       tr.Stats.Reason,
		LostFlags:    tr.LostFlags,
	}
	backend := "ncc"
	if twoPoint {
		backend = "two_point_ncc"
	}
	return finishNativeGenerate(ctx, videoPath, outputPath, opts, ntr, "simpletrack", backend, progress, onPercent, start)
}

// finishNativeGenerate shared posttrack + quality + write for CSRT and simple.
func finishNativeGenerate(
	ctx context.Context,
	videoPath string,
	outputPath string,
	opts Options,
	tr nativeTrackResult,
	tracking, backend string,
	progress func(string),
	onPercent func(int),
	start time.Time,
) error {
	ts := make([]float64, len(tr.TimestampsMs))
	for i, v := range tr.TimestampsMs {
		ts[i] = float64(v)
	}

	buildPost := func(smooth, peakDist int, adaptive, normPct float64) (posttrack.Options, error) {
		postOpts := posttrack.DefaultOptions()
		postOpts.Invert = opts.Invert
		if smooth > 0 {
			postOpts.SmoothWindow = smooth
		}
		if peakDist > 0 {
			postOpts.MinPeakDistanceMs = peakDist
		}
		postOpts.RDPTolerance = opts.RDPTolerance
		if normPct > 0 {
			postOpts.NormPercentile = normPct
		} else if normPct < 0 {
			postOpts.NormPercentile = 0
		}
		postOpts.AdaptiveError = adaptive
		if opts.MinActionIntervalMs > 0 {
			postOpts.MinIntervalMs = opts.MinActionIntervalMs
		} else if opts.MinActionIntervalMs < 0 {
			postOpts.MinIntervalMs = 0
		}
		postOpts.DynamicRangeMs = opts.DynamicRangeMs
		postOpts.PeakProminence = opts.PeakProminence
		postOpts.DetrendWindowMs = opts.DetrendWindowMs
		postOpts.BandpassLowHz = opts.BandpassLowHz
		postOpts.BandpassHighHz = opts.BandpassHighHz
		if opts.Profile == "weich" {
			if postOpts.PeakProminence == 0 {
				postOpts.PeakProminence = 0.35
			}
			if peakDist == 0 {
				postOpts.MinPeakDistanceMs = 200
			}
			if postOpts.DynamicRangeMs == 0 {
				postOpts.DynamicRangeMs = 3000
			}
		}
		if opts.Profile == "autotune" {
			// Multi-filter recipe inspired by FunGen Ultimate Autotune /
			// Funscript-Flow — classical track still writes the script.
			if postOpts.DetrendWindowMs == 0 {
				postOpts.DetrendWindowMs = 3000
			}
			if postOpts.BandpassLowHz == 0 && postOpts.BandpassHighHz == 0 {
				postOpts.BandpassLowHz = 0.5
				postOpts.BandpassHighHz = 4.0
			}
			if postOpts.DynamicRangeMs == 0 {
				postOpts.DynamicRangeMs = 3000
			}
			if postOpts.PeakProminence == 0 {
				postOpts.PeakProminence = 0.2
			}
		}
		return postOpts, nil
	}

	smooth := opts.SmoothWindow
	peakDist := opts.MinPeakDistanceMs
	adaptive := opts.AdaptiveKeyframeError
	normPct := opts.NormPercentile

	runSignal := func(smooth, peakDist int, adaptive, normPct float64) ([]funscript.Action, posttrack.Result, funscript.ScriptQualityResult, error) {
		postOpts, _ := buildPost(smooth, peakDist, adaptive, normPct)
		result, err := posttrack.PositionsToActions(ts, tr.Positions, postOpts)
		if err != nil {
			return nil, posttrack.Result{}, funscript.ScriptQualityResult{}, err
		}
		actions := result.Actions
		maxSpeed := opts.MaxSpeed
		if maxSpeed <= 0 && opts.Profile == "autotune" {
			maxSpeed = 400
		}
		if maxSpeed > 0 {
			actions, _ = posttrack.LimitSpeed(actions, maxSpeed)
		}
		if funscript.IsDistanceProfile(opts.Profile) {
			actions = posttrack.ClampActionsPos(actions, 20, 90)
		}
		var lostFrac *float64
		if tr.TotalFrames > 0 {
			f := float64(tr.LostFrames) / float64(tr.TotalFrames)
			lostFrac = &f
		}
		var motionFrac *float64
		if tr.Height > 0 && tr.VertRange > 0 {
			f := tr.VertRange / float64(tr.Height)
			motionFrac = &f
		}
		var durationMs *float64
		if len(actions) > 0 {
			d := float64(actions[len(actions)-1].At)
			durationMs = &d
		}
		quality := funscript.EvaluateDenseQuality(actions, funscript.DenseQualityInput{
			DenseAt:             result.DenseAt,
			DensePos:            result.DensePos,
			TrackerLostFraction: lostFrac,
			MotionRangeFraction: motionFrac,
			VideoDurationMs:     durationMs,
			ValidFrames:         tr.ValidFrames,
			Confidence:          tr.Confidence,
			Reason:              tr.Reason,
		})
		return actions, result, quality, nil
	}

	actions, _, quality, err := runSignal(smooth, peakDist, adaptive, normPct)
	if err != nil {
		return fmt.Errorf("generator/native: signal path: %w", err)
	}
	if opts.MaxSpeed > 0 {
		progress(fmt.Sprintf("Geschwindigkeitsbegrenzung aktiv (%.0f Einheiten/s)", opts.MaxSpeed))
	}

	// Auto-Retry: vary signal params only (same as Python). Tracking issues
	// are not retriable here.
	if opts.AutoRetry && !quality.Passed {
		lostHeavy := tr.TotalFrames > 0 && float64(tr.LostFrames)/float64(tr.TotalFrames) > 0.5
		motionTiny := tr.Height > 0 && tr.VertRange/float64(tr.Height) < 0.03
		if lostHeavy || motionTiny {
			progress("Auto-Retry übersprungen: Problem liegt im Tracking, nicht in der Signalverarbeitung")
		} else {
			type cand struct {
				smooth, peakDist int
				adaptive, norm   float64
			}
			baseSmooth := smooth
			if baseSmooth <= 0 {
				baseSmooth = 11
			}
			basePeak := peakDist
			if basePeak <= 0 {
				basePeak = 150
			}
			candidates := []cand{
				{max(5, baseSmooth-4), basePeak, adaptive, normPct},
				{baseSmooth + 10, basePeak, adaptive, normPct},
				{baseSmooth + 20, max(250, basePeak*2), adaptive, normPct},
				{baseSmooth, basePeak, adaptive, 0},
			}
			progress(fmt.Sprintf("Qualitätsprüfung nicht bestanden (Score %.2f) — probiere %d alternative Signalparameter",
				quality.Score, len(candidates)))
			bestActions, bestQuality := actions, quality
			for _, c := range candidates {
				if err := ctx.Err(); err != nil {
					return err
				}
				acts, _, q, err := runSignal(c.smooth, c.peakDist, c.adaptive, c.norm)
				if err != nil {
					continue
				}
				if q.Score > bestQuality.Score {
					bestActions, bestQuality = acts, q
				}
				if q.Passed {
					break
				}
			}
			actions, quality = bestActions, bestQuality
			progress(fmt.Sprintf("Bestes Ergebnis nach Auto-Retry: Score %.2f passed=%v", quality.Score, quality.Passed))
		}
	}

	progress(fmt.Sprintf("%d Keyframes aus %d Frames", len(actions), len(tr.TimestampsMs)))

	if err := ctx.Err(); err != nil {
		return err
	}

	progress(fmt.Sprintf("Quality Doctor (dense): score=%.2f passed=%v", quality.Score, quality.Passed))

	var audioMeta *funscript.AudioCheck
	if opts.AudioCheck {
		progress("Audio-Tempo-Prüfung (Go, post-hoc)…")
		audioMeta = CheckAudioTempo(videoPath, actions)
		if audioMeta == nil {
			progress("Audio-Tempo-Prüfung nicht möglich (kein ffmpeg / keine Audiospur)")
		} else {
			switch {
			case audioMeta.ScriptHz != nil && audioMeta.AudioHz != nil:
				progress(fmt.Sprintf("Audio-Tempo-Prüfung: Skript %.2fHz, Audio %.2fHz",
					*audioMeta.ScriptHz, *audioMeta.AudioHz))
			case audioMeta.AudioHz != nil:
				progress(fmt.Sprintf("Audio-Tempo-Prüfung: Audio %.2fHz (Skript-Tempo nicht schätzbar)",
					*audioMeta.AudioHz))
			case audioMeta.ScriptHz != nil:
				progress(fmt.Sprintf("Audio-Tempo-Prüfung: Skript %.2fHz (Audio-Tempo nicht schätzbar)",
					*audioMeta.ScriptHz))
			}
			for _, w := range audioMeta.Warnings {
				progress("WARNUNG: " + w)
			}
		}
	}

	if err := writeNativeFunscriptNamed(outputPath, actions, opts, tr, quality, audioMeta, tracking, backend); err != nil {
		return err
	}
	progress(fmt.Sprintf("geschrieben: %s (gesamt %s)", outputPath, time.Since(start).Round(time.Millisecond)))
	if onPercent != nil {
		onPercent(100)
	}
	return nil
}
