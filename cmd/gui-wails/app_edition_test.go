//go:build !studio

package main

import (
	"errors"
	"testing"
)

// Customer build: training/learning bindings refuse with errStudioOnly and
// report studio=false, so the GUI hides them.
func TestCustomerBuildRefusesStudioTools(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	a := NewApp()
	if a.IsStudioBuild() {
		t.Fatal("default build must be the customer edition")
	}
	if err := a.RunRoiModelTraining(10, "cpu"); !errors.Is(err, errStudioOnly) {
		t.Fatalf("AI Train must be refused: %v", err)
	}
	if err := a.RunGoldenClipBenchmark("m.json"); !errors.Is(err, errStudioOnly) {
		t.Fatalf("benchmark must be refused: %v", err)
	}
	if _, err := a.TrainQualityModel(); !errors.Is(err, errStudioOnly) {
		t.Fatalf("quality model training must be refused: %v", err)
	}
	if _, err := a.ExportSceneMapLearning("x.samn"); !errors.Is(err, errStudioOnly) {
		t.Fatalf("learning export must be refused: %v", err)
	}
	if a.CheckRoiTrainingAvailable() || a.CancelRoiTraining() || a.CancelGoldenClipBenchmark() {
		t.Fatal("availability/cancel must be false in the customer build")
	}
}
