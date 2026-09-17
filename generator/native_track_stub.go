//go:build !cgo || !opencv || windows

package generator

import "errors"

// NativeTrackingAvailable is false unless the binary was built with
// `-tags opencv` (and cgo + system OpenCV). See native_track_opencv.go.
func NativeTrackingAvailable() bool { return false }

var errNativeUnavailable = errors.New("generator: native CSRT tracking is not available in this build (needs -tags opencv + cgo + OpenCV)")

func nativeTrackROI(videoPath string, roi ROI, opts nativeTrackOptions, onPercent func(int)) (nativeTrackResult, error) {
	return nativeTrackResult{}, errNativeUnavailable
}
