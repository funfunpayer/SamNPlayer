package generator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/funfunpayer/SamNPlayer/funscript"
	"github.com/funfunpayer/SamNPlayer/logging"
)

// errNativeCanceled is the build-tag-agnostic cancel sentinel from nativeTrackROI.
var errNativeCanceled = errors.New("generator/native: tracking canceled")

// percentFromProgress maps tracker (done,total) callbacks onto the GUI's
// 0–100 (or -1 when total is unknown) percent channel.
func percentFromProgress(onPercent func(int)) func(done, total int) {
	if onPercent == nil {
		return nil
	}
	return func(done, total int) {
		onPercent(percentOf(done, total))
	}
}

// NativePipelineEligible reports whether opts+roi can run on a Python-free
// path (CSRT via trackcv when OpenCV is linked, otherwise simpletrack/NCC
// via videox). Covers single-ROI and Tf/Tj two-point (ROI2). Other backends
// / per-scene / AI opinion still need Python. Audio-tempo check is post-hoc
// in Go (CheckAudioTempo) and does not force the Python path.
func NativePipelineEligible(opts Options, roi ROI) bool {
	return nativeOptionsEligible(opts, roi)
}

func nativeOptionsEligible(opts Options, roi ROI) bool {
	if roi.W <= 0 || roi.H <= 0 {
		return false
	}
	backend := opts.Backend
	if backend == "" {
		backend = "csrt"
	}
	// "csrt" and empty mean "native Go pipeline preferred"; simpletrack
	// covers the same cases when OpenCV is missing. Two-point Tf/Tj uses
	// the same CSRT/NCC trackers on both ROIs.
	if backend != "csrt" {
		return false
	}
	// AutoRetry is handled inside finishNativeGenerate (signal-param retry),
	// so it no longer blocks the Go path — default GUI has AutoRetry on.
	// UseOpenCL is ignored for eligibility: Go CSRT does not use OpenCL;
	// Python enables OpenCL automatically when that path is taken anyway.
	if opts.PerSceneROI {
		return false
	}
	// AI quality opinion still shells out to Python; audio check runs in Go
	// after tracking and no longer blocks eligibility.
	if opts.AIQualityOpinion {
		return false
	}
	// Soft masks still need Python (feature punch-outs on camera/grid paths).
	// Extra Tf/Tj targets run on Go TrackMultiPoints (CSRT / simpletrack).
	if len(opts.MaskROIs) > 0 {
		return false
	}
	return true
}

// GenerateNativeCSRT runs trackcv.TrackROI (or TrackTwoPoints when ROI2 is
// set) + posttrack.PositionsToActions and writes a .funscript. No Python
// subprocess. Dense Quality Doctor runs in Go when dense curve + tracker
// stats are available; AI/audio stay Python-only.
// Cancel ctx to abort CSRT mid-loop (same Abort path as Python CommandContext).
// Returns errNativeUnavailable when OpenCV/cgo is not linked; context.Canceled
// when aborted.
func GenerateNativeCSRT(ctx context.Context, videoPath string, roi ROI, outputPath string, opts Options, onProgress func(line string), onPercent func(pct int)) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if !NativeTrackingAvailable() {
		return errNativeUnavailable
	}
	if !nativeOptionsEligible(opts, roi) {
		return fmt.Errorf("generator: native pipeline not eligible for these options (CSRT; single ROI or Tf/Tj ROI2/+targets; no per-scene, AI opinion, soft masks)")
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	progress := func(line string) {
		logging.Info("generator/native: " + line)
		if onProgress != nil {
			onProgress(line)
		}
	}
	twoPoint := distancePartnersActive(opts)
	if opts.ROI2.W > 0 && opts.ROI2.H > 0 && !twoPoint {
		progress("ignoring Zone 2 — Autotune/stroke profiles use tip ROI only (switch to Tf/Tj for distance)")
	}
	multi := twoPoint && len(opts.ExtraTargets) > 0
	if multi {
		progress("Go-native multi-partner pipeline (trackcv TrackMultiPoints + posttrack), no Python")
	} else if twoPoint {
		progress("Go-native two-point pipeline (trackcv TrackTwoPoints + posttrack), no Python")
	} else {
		progress("Go-native CSRT pipeline (trackcv + posttrack), no Python")
	}

	start := time.Now()
	trackOpts := nativeTrackOptions{
		MaxFrames:          opts.MaxFrames,
		StartTimeSec:       opts.StartTimeSec,
		CameraCompensation: !opts.DisableCameraCompensation,
		SceneCutDetection:  !opts.DisableSceneCutDetection,
		AppearanceMemory:   true,
		Axis:               opts.Axis,
		Cancel:             func() bool { return ctx.Err() != nil },
		FixedB:             opts.ROI2Fixed,
	}
	if trackOpts.Axis == "" {
		trackOpts.Axis = "auto"
	}

	var tr nativeTrackResult
	var err error
	if twoPoint {
		partners := []nativePartner{{ROI: opts.ROI2, Fixed: opts.ROI2Fixed}}
		for _, t := range opts.ExtraTargets {
			if t.W <= 0 || t.H <= 0 {
				continue
			}
			// Zone 3+ extras are fixed contact anchors (GUI +Target).
			partners = append(partners, nativePartner{
				ROI:   ROI{X: t.X, Y: t.Y, W: t.W, H: t.H},
				Fixed: true,
			})
		}
		tr, err = nativeTrackMultiPoints(videoPath, roi, partners, trackOpts, onPercent)
	} else {
		tr, err = nativeTrackROI(videoPath, roi, trackOpts, onPercent)
	}
	if err != nil {
		if errors.Is(err, errNativeCanceled) || errors.Is(ctx.Err(), context.Canceled) {
			return context.Canceled
		}
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return context.DeadlineExceeded
		}
		return fmt.Errorf("generator/native: tracking: %w", err)
	}
	progress(fmt.Sprintf("%d frames tracked (%dx%d) in %s",
		len(tr.TimestampsMs), tr.Width, tr.Height, time.Since(start).Round(time.Millisecond)))
	backend := "csrt"
	if multi {
		backend = "multi_point"
	} else if twoPoint {
		backend = "two_point"
	}
	return finishNativeGenerate(ctx, videoPath, outputPath, opts, tr, "trackcv", backend, progress, onPercent, start)
}

type nativePartner struct {
	ROI   ROI
	Fixed bool
}

type nativeTrackResult struct {
	TimestampsMs []int
	Positions    []float64
	Width        int
	Height       int
	SceneCuts    []int
	LostFrames   int
	TotalFrames  int
	VertRange    float64
	ValidFrames  int
	Confidence   float64
	Reason       string
	LostFlags    []bool // two-point: per-frame tracker loss
}

type nativeTrackOptions struct {
	MaxFrames          int
	StartTimeSec       float64
	CameraCompensation bool
	SceneCutDetection  bool
	AppearanceMemory   bool
	Axis               string
	Cancel             func() bool
	FixedB             bool
}

func writeNativeFunscript(path string, actions []funscript.Action, opts Options, tr nativeTrackResult, quality funscript.ScriptQualityResult) error {
	return writeNativeFunscriptNamed(path, actions, opts, tr, quality, nil, "trackcv", "csrt")
}

func writeNativeFunscriptNamed(path string, actions []funscript.Action, opts Options, tr nativeTrackResult, quality funscript.ScriptQualityResult, audio *funscript.AudioCheck, tracking, backend string) error {
	if len(actions) == 0 {
		return fmt.Errorf("generator/native: keine Actions erzeugt")
	}
	duration := actions[len(actions)-1].At
	nativeMeta := map[string]any{
		"tracking": tracking,
		"signal":   "posttrack",
		"backend":  backend,
		"frames":   tr.TotalFrames,
		"lost":     tr.LostFrames,
		"range_px": tr.VertRange,
	}
	if tr.ValidFrames > 0 || tr.TotalFrames > 0 {
		nativeMeta["valid_frames"] = tr.ValidFrames
		nativeMeta["confidence"] = tr.Confidence
	}
	if tr.Reason != "" {
		nativeMeta["reason"] = tr.Reason
	}
	creator := "SamNPlayer generator/native (trackcv+posttrack, kein Python)"
	if tracking == "simpletrack" {
		creator = "SamNPlayer generator/native (simpletrack+posttrack, kein Python/OpenCV)"
	}
	meta := map[string]any{
		"creator":                    creator,
		"duration":                   duration,
		"native_pipeline":            nativeMeta,
		"quality_score":              quality.Score,
		"quality_passed":             quality.Passed,
		"quality_warnings":           quality.Warnings,
		"quality_kind":               quality.Kind,
		"estimated_from_script_only": quality.EstimatedFromScriptOnly,
	}
	gaps := trackingGapsFromFlags(tr.TimestampsMs, tr.LostFlags, 100)
	if len(gaps) > 0 {
		meta["tracking_gaps"] = gaps
	}
	if (opts.Profile != "" && opts.Profile != "standard") || opts.ContactVibration {
		recipe := funscript.RecipeMeta(opts.Profile)
		if opts.Profile == "" || opts.Profile == "standard" {
			recipe = funscript.DeviceRecipe{
				Sync:      funscript.SyncIndependent.String(),
				Smoothing: 0.3,
			}
		}
		if opts.ContactVibration {
			recipe.ContactVibration = true
			if opts.ContactVibrationSpan > 0 {
				span := funscript.EffectiveContactSpan(opts.ContactVibrationSpan)
				if span != funscript.DefaultContactVibrationSpan {
					recipe.ContactVibrationSpan = span
				}
			}
			curve := funscript.NormalizeContactCurve(opts.ContactVibrationCurve)
			if curve != funscript.ContactCurveLinear {
				recipe.ContactVibrationCurve = curve
			}
		}
		if opts.Profile != "" && opts.Profile != "standard" {
			meta["profile"] = funscript.NormalizeProfile(opts.Profile)
		} else if opts.ContactVibration {
			meta["profile"] = "standard"
		}
		if funscript.IsDistanceProfile(opts.Profile) || opts.ContactVibration {
			meta["device_recipe"] = recipe
		}
	}
	if audio != nil {
		meta["audio_check"] = audio
	}
	if opts.StrokePreviewHint != nil {
		meta["stroke_preview"] = opts.StrokePreviewHint
	}
	doc := map[string]any{
		"actions":  actions,
		"metadata": meta,
	}
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return err
	}
	return writeCompanionSamn(path, actions, opts, gaps, quality)
}
