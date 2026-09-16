//go:build cgo && !windows

package generator

import (
	"errors"

	"github.com/funfunpayer/SamNPlayer/generator/trackcv"
)

// NativeTrackingAvailable is true when this binary was built with cgo against
// system OpenCV (Linux/macOS). Windows release builds keep CGO off for the
// cross-compile, so the stub returns false there.
func NativeTrackingAvailable() bool { return true }

var errNativeUnavailable = errors.New("generator: native CSRT tracking is not available in this build (needs cgo + OpenCV)")

func nativeTrackROI(videoPath string, roi ROI, opts nativeTrackOptions, onPercent func(int)) (nativeTrackResult, error) {
	tr, err := trackcv.TrackROI(videoPath, trackcv.Rect{X: roi.X, Y: roi.Y, W: roi.W, H: roi.H}, trackcv.Options{
		MaxFrames:          opts.MaxFrames,
		CameraCompensation: opts.CameraCompensation,
		SceneCutDetection:  opts.SceneCutDetection,
		AppearanceMemory:   opts.AppearanceMemory,
		Axis:               opts.Axis,
	})
	if err != nil {
		return nativeTrackResult{}, err
	}
	if onPercent != nil && tr.Stats.TotalFrames > 0 {
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
	}, nil
}
