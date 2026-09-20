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
	}, nil
}

func nativeTrackTwoPoints(videoPath string, roi, roi2 ROI, opts nativeTrackOptions, onPercent func(int)) (nativeTrackResult, error) {
	tr, err := trackcv.TrackTwoPoints(videoPath,
		trackcv.Rect{X: roi.X, Y: roi.Y, W: roi.W, H: roi.H},
		trackcv.Rect{X: roi2.X, Y: roi2.Y, W: roi2.W, H: roi2.H},
		trackcv.Options{
			MaxFrames:    opts.MaxFrames,
			StartTimeSec: opts.StartTimeSec,
			Cancel:       opts.Cancel,
			FixedB:       opts.FixedB,
			OnProgress:   percentFromProgress(onPercent),
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
	}, nil
}
