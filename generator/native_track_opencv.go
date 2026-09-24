//go:build cgo && opencv

package generator

import (
	"errors"

	"github.com/funfunpayer/SamNPlayer/generator/trackcv"
)

// NativeTrackingAvailable is true when this binary was built with
// `-tags opencv` and cgo against system OpenCV (Linux/macOS). Default
// builds and Windows cross-compiles use the stub (false).
func NativeTrackingAvailable() bool { return true }

var errNativeUnavailable = errors.New("generator: native CSRT tracking is not available in this build (needs -tags opencv + cgo + OpenCV)")

func nativeTrackROI(videoPath string, roi ROI, opts nativeTrackOptions, onPercent func(int)) (nativeTrackResult, error) {
	tr, err := trackcv.TrackROI(videoPath, trackcv.Rect{X: roi.X, Y: roi.Y, W: roi.W, H: roi.H}, trackcv.Options{
		MaxFrames:          opts.MaxFrames,
		StartTimeSec:       opts.StartTimeSec,
		CameraCompensation: opts.CameraCompensation,
		SceneCutDetection:  opts.SceneCutDetection,
		AppearanceMemory:   opts.AppearanceMemory,
		Axis:               opts.Axis,
		Cancel:             opts.Cancel,
		OnProgress:         percentFromProgress(onPercent),
		CaptureTrajectory:  opts.CaptureTrajectory,
		RhythmGrid:         opts.RhythmGrid,
		SceneMarks:         toTrackcvSceneMarks(opts.SceneMarks),
	})
	if err != nil {
		if errors.Is(err, trackcv.ErrCanceled) || tr.Canceled {
			return nativeTrackResult{}, errNativeCanceled
		}
		return nativeTrackResult{}, err
	}
	if onPercent != nil {
		onPercent(100)
	}
	return nativeTrackResult{
		TimestampsMs: tr.TimestampsMs,
		Positions:    tr.Positions,
		Width:        tr.Width,
		Height:       tr.Height,
		SceneCuts:    tr.SceneCuts,
		LostFrames:   tr.Stats.TrackerLostFrames,
		TotalFrames:  tr.Stats.TotalFrames,
		VertRange:    tr.Stats.VerticalRange,
		ValidFrames:  tr.Stats.ValidFrames,
		Confidence:   tr.Stats.Confidence,
		Reason:       tr.Stats.Reason,
		TrajectoryA:  convertTrackcvPoints(tr.TrajectoryA),
	}, nil
}

func toTrackcvSceneMarks(marks []SceneMark) []trackcv.SceneMark {
	if len(marks) == 0 {
		return nil
	}
	out := make([]trackcv.SceneMark, len(marks))
	for i, m := range marks {
		out[i] = trackcv.SceneMark{
			Kind:   m.Kind,
			ID:     m.ID,
			Rect:   trackcv.Rect{X: m.Rect.X, Y: m.Rect.Y, W: m.Rect.W, H: m.Rect.H},
			FromMs: m.FromMs,
			ToMs:   m.ToMs,
			Class:  m.Class,
		}
	}
	return out
}

// convertTrackcvPoints maps trackcv.Point (only meaningful inside this
// cgo+opencv-tagged file) onto the tag-free NativePoint used by native.go.
func convertTrackcvPoints(pts []trackcv.Point) []NativePoint {
	if len(pts) == 0 {
		return nil
	}
	out := make([]NativePoint, len(pts))
	for i, p := range pts {
		out[i] = NativePoint{X: p.X, Y: p.Y}
	}
	return out
}

func nativeTrackTwoPoints(videoPath string, roi, roi2 ROI, opts nativeTrackOptions, onPercent func(int)) (nativeTrackResult, error) {
	return nativeTrackMultiPoints(videoPath, roi, []nativePartner{{ROI: roi2, Fixed: opts.FixedB}}, opts, onPercent)
}

func nativeTrackMultiPoints(videoPath string, tip ROI, partners []nativePartner, opts nativeTrackOptions, onPercent func(int)) (nativeTrackResult, error) {
	tps := make([]trackcv.Partner, len(partners))
	for i, p := range partners {
		tps[i] = trackcv.Partner{
			ROI:   trackcv.Rect{X: p.ROI.X, Y: p.ROI.Y, W: p.ROI.W, H: p.ROI.H},
			Fixed: p.Fixed,
		}
	}
	tr, err := trackcv.TrackMultiPoints(videoPath,
		trackcv.Rect{X: tip.X, Y: tip.Y, W: tip.W, H: tip.H},
		tps,
		trackcv.Options{
			MaxFrames:         opts.MaxFrames,
			StartTimeSec:      opts.StartTimeSec,
			AppearanceMemory:  opts.AppearanceMemory,
			Cancel:            opts.Cancel,
			OnProgress:        percentFromProgress(onPercent),
			CaptureTrajectory: opts.CaptureTrajectory,
		})
	if err != nil {
		if errors.Is(err, trackcv.ErrCanceled) || tr.Canceled {
			return nativeTrackResult{}, errNativeCanceled
		}
		return nativeTrackResult{}, err
	}
	if onPercent != nil {
		onPercent(100)
	}
	return nativeTrackResult{
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
		TrajectoryA:  convertTrackcvPoints(tr.TrajectoryA),
		TrajectoryB:  convertTrackcvPoints(tr.TrajectoryB),
	}, nil
}
