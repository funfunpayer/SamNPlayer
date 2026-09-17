package generator

import "testing"

func TestNativePipelineEligible(t *testing.T) {
	roi := ROI{X: 10, Y: 10, W: 40, H: 40}
	ok := Options{Backend: "csrt", DisableCache: true}
	// Eligible is option-only: OpenCV optional (Windows uses simpletrack).
	if !NativePipelineEligible(ok, roi) {
		t.Fatal("expected eligible for plain CSRT options")
	}
	// Default GUI has AutoRetry on — must still be eligible (retry is in-Go).
	if !NativePipelineEligible(Options{Backend: "csrt", AutoRetry: true}, roi) {
		t.Fatal("AutoRetry must not block the Go path")
	}
	if NativePipelineEligible(Options{Backend: "flow"}, roi) {
		t.Fatal("flow must not be eligible")
	}
	if NativePipelineEligible(Options{Backend: "csrt", ROI2: ROI{W: 10, H: 10}}, roi) {
		t.Fatal("roi2 must not be eligible")
	}
	if NativePipelineEligible(Options{Backend: "csrt", PerSceneROI: true}, roi) {
		t.Fatal("per-scene must not be eligible")
	}
	if NativePipelineEligible(Options{Backend: "csrt", AIQualityOpinion: true}, roi) {
		t.Fatal("AI opinion must not be eligible")
	}
	if NativePipelineEligible(Options{Backend: "csrt"}, ROI{}) {
		t.Fatal("empty ROI must not be eligible")
	}
	if !SimpleTrackingAvailable() {
		t.Fatal("simpletrack must always be available")
	}
}
