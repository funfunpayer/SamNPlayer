//go:build !cgo || windows

package generator

import "errors"

// NativeTrackingAvailable is false on Windows cross-builds and any build
// without cgo — see native_track_opencv.go for the OpenCV-backed path.
func NativeTrackingAvailable() bool { return false }

var errNativeUnavailable = errors.New("generator: native CSRT tracking is not available in this build (needs cgo + OpenCV)")

func nativeTrackROI(videoPath string, roi ROI, opts nativeTrackOptions, onPercent func(int)) (nativeTrackResult, error) {
	return nativeTrackResult{}, errNativeUnavailable
}
