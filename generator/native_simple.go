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
// Python/OpenCV. Used when NativePipeline is on and CSRT/OpenCV is not
// linked — notably Windows release builds.
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
	progress("Go-Pipeline ohne OpenCV (simpletrack/NCC über ffmpeg) — schwächer als CSRT, kein Python")

	start := time.Now()
	axis := opts.Axis
	if axis == "" {
		axis = "auto"
	}
	tr, err := simpletrack.TrackROI(ctx, videoPath, simpletrack.Rect{X: roi.X, Y: roi.Y, W: roi.W, H: roi.H}, simpletrack.Options{
		MaxFrames: opts.MaxFrames,
		Axis:      axis,
		Cancel:    func() bool { return ctx.Err() != nil },
	})
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
	}
	return finishNativeGenerate(ctx, outputPath, opts, ntr, "simpletrack", "ncc", progress, onPercent, start)
}

// finishNativeGenerate shared posttrack + quality + write for CSRT and simple.
func finishNativeGenerate(
	ctx context.Context,
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

	postOpts := posttrack.DefaultOptions()
	postOpts.Invert = opts.Invert
	if opts.SmoothWindow > 0 {
		postOpts.SmoothWindow = opts.SmoothWindow
	}
	if opts.MinPeakDistanceMs > 0 {
		postOpts.MinPeakDistanceMs = opts.MinPeakDistanceMs
	}
	postOpts.RDPTolerance = opts.RDPTolerance
	if opts.NormPercentile > 0 {
		postOpts.NormPercentile = opts.NormPercentile
	} else if opts.NormPercentile < 0 {
		postOpts.NormPercentile = 0
	}
	postOpts.AdaptiveError = opts.AdaptiveKeyframeError
	if opts.MinActionIntervalMs > 0 {
		postOpts.MinIntervalMs = opts.MinActionIntervalMs
	} else if opts.MinActionIntervalMs < 0 {
		postOpts.MinIntervalMs = 0
	}
	postOpts.DynamicRangeMs = opts.DynamicRangeMs
	postOpts.PeakProminence = opts.PeakProminence
	if opts.Profile == "weich" {
		if postOpts.PeakProminence == 0 {
			postOpts.PeakProminence = 0.35
		}
		if opts.MinPeakDistanceMs == 0 {
			postOpts.MinPeakDistanceMs = 200
		}
		if postOpts.DynamicRangeMs == 0 {
			postOpts.DynamicRangeMs = 3000
		}
	}

	result, err := posttrack.PositionsToActions(ts, tr.Positions, postOpts)
	if err != nil {
		return fmt.Errorf("generator/native: signal path: %w", err)
	}
	actions := result.Actions
	if opts.MaxSpeed > 0 {
		var changed int
		actions, changed = posttrack.LimitSpeed(actions, opts.MaxSpeed)
		if changed > 0 {
			progress(fmt.Sprintf("%d Aktion(en) auf %.0f Einheiten/s begrenzt", changed, opts.MaxSpeed))
		}
	}
	if funscript.IsDistanceProfile(opts.Profile) {
		actions = posttrack.ClampActionsPos(actions, 20, 90)
	}
	progress(fmt.Sprintf("%d Keyframes aus %d Frames", len(actions), len(tr.TimestampsMs)))

	if err := ctx.Err(); err != nil {
		return err
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
	progress(fmt.Sprintf("Quality Doctor (dense): score=%.2f passed=%v", quality.Score, quality.Passed))

	if err := writeNativeFunscriptNamed(outputPath, actions, opts, tr, quality, tracking, backend); err != nil {
		return err
	}
	progress(fmt.Sprintf("geschrieben: %s (gesamt %s)", outputPath, time.Since(start).Round(time.Millisecond)))
	if onPercent != nil {
		onPercent(100)
	}
	return nil
}
