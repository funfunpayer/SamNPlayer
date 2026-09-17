package generator

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/funfunpayer/SamNPlayer/funscript"
	"github.com/funfunpayer/SamNPlayer/generator/posttrack"
	"github.com/funfunpayer/SamNPlayer/logging"
)

// NativePipelineEligible reports whether opts+roi can run on the pure-Go
// CSRT path (trackcv + posttrack). Anything outside this set still needs
// Python: other backends, two-point Tf/Tj, per-scene ROI, AI/audio extras,
// auto-retry (needs Quality Doctor), OpenCL, cache.
func NativePipelineEligible(opts Options, roi ROI) bool {
	if !NativeTrackingAvailable() {
		return false
	}
	if roi.W <= 0 || roi.H <= 0 {
		return false
	}
	if opts.ROI2.W > 0 && opts.ROI2.H > 0 {
		return false
	}
	backend := opts.Backend
	if backend == "" {
		backend = "csrt"
	}
	if backend != "csrt" {
		return false
	}
	if opts.PerSceneROI || opts.AutoRetry || opts.UseOpenCL {
		return false
	}
	if opts.AIQualityOpinion || opts.AudioCheck {
		return false
	}
	return true
}

// GenerateNativeCSRT runs trackcv.TrackROI + posttrack.PositionsToActions and
// writes a .funscript. No Python subprocess. Quality Doctor / AI / audio are
// not applied — the metadata.creator string marks the file as native so the
// GUI can tell. Returns errNativeUnavailable when OpenCV/cgo is not linked.
func GenerateNativeCSRT(videoPath string, roi ROI, outputPath string, opts Options, onProgress func(line string), onPercent func(pct int)) error {
	if !NativeTrackingAvailable() {
		return errNativeUnavailable
	}
	if !NativePipelineEligible(opts, roi) {
		return fmt.Errorf("generator: native pipeline not eligible for these options (CSRT + single ROI only; no Tf/Tj, per-scene, AI, audio, auto-retry, OpenCL)")
	}

	progress := func(line string) {
		logging.Info("generator/native: " + line)
		if onProgress != nil {
			onProgress(line)
		}
	}
	progress("Go-native CSRT-Pipeline (trackcv + posttrack), ohne Python")

	start := time.Now()
	trackOpts := nativeTrackOptions{
		MaxFrames:          opts.MaxFrames,
		CameraCompensation: !opts.DisableCameraCompensation,
		SceneCutDetection:  !opts.DisableSceneCutDetection,
		AppearanceMemory:   true,
		Axis:               opts.Axis,
	}
	if trackOpts.Axis == "" {
		trackOpts.Axis = "auto"
	}

	tr, err := nativeTrackROI(videoPath, roi, trackOpts, onPercent)
	if err != nil {
		return fmt.Errorf("generator/native: tracking: %w", err)
	}
	progress(fmt.Sprintf("%d Frames getrackt (%dx%d) in %s",
		len(tr.TimestampsMs), tr.Width, tr.Height, time.Since(start).Round(time.Millisecond)))

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
	// Match generate_funscript.py profile='weich' defaults when the caller
	// left the corresponding fields at zero.
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

	if err := writeNativeFunscript(outputPath, actions, opts, tr); err != nil {
		return err
	}
	progress(fmt.Sprintf("geschrieben: %s (gesamt %s)", outputPath, time.Since(start).Round(time.Millisecond)))
	if onPercent != nil {
		onPercent(100)
	}
	return nil
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
}

type nativeTrackOptions struct {
	MaxFrames          int
	CameraCompensation bool
	SceneCutDetection  bool
	AppearanceMemory   bool
	Axis               string
}

func writeNativeFunscript(path string, actions []funscript.Action, opts Options, tr nativeTrackResult) error {
	if len(actions) == 0 {
		return fmt.Errorf("generator/native: keine Actions erzeugt")
	}
	duration := actions[len(actions)-1].At
	meta := map[string]any{
		"creator":  "SamNPlayer generator/native (trackcv+posttrack, kein Python)",
		"duration": duration,
		"native_pipeline": map[string]any{
			"tracking": "trackcv",
			"signal":   "posttrack",
			"backend":  "csrt",
			"frames":   tr.TotalFrames,
			"lost":     tr.LostFrames,
			"range_px": tr.VertRange,
		},
	}
	if opts.Profile != "" && opts.Profile != "standard" {
		recipe := funscript.RecipeMeta(opts.Profile)
		if opts.ContactVibration && funscript.IsDistanceProfile(opts.Profile) {
			recipe.ContactVibration = true
		}
		meta["profile"] = funscript.NormalizeProfile(opts.Profile)
		if funscript.IsDistanceProfile(opts.Profile) {
			meta["device_recipe"] = recipe
		}
	}
	doc := map[string]any{
		"actions":  actions,
		"metadata": meta,
	}
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
