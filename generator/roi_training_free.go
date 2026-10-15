//go:build !suite

package generator

import (
	"context"
	"fmt"
)

// Stubs for Free builds: training is Suite-only (ENG-01 / HANDOFF task 3).

func RoiTrainingAvailable() bool { return false }

type RoiTrainingStatus struct {
	Python      bool   `json:"python"`
	OpenCV      bool   `json:"opencv"`
	Ultralytics bool   `json:"ultralytics"`
	Detail      string `json:"detail"`
}

func GetRoiTrainingStatus() RoiTrainingStatus {
	return RoiTrainingStatus{Detail: "Suite only"}
}

func InstallRoiTrainingDeps(onProgress func(line string)) error {
	return fmt.Errorf("generator: ROI training is Suite-only")
}

func WriteAIRequirementFiles(dir string) error {
	return fmt.Errorf("generator: ROI training is Suite-only")
}

type RoiTrainingDevice struct {
	ID        string `json:"id"`
	Label     string `json:"label"`
	Available bool   `json:"available"`
}

func ListRoiTrainingDevices() []RoiTrainingDevice { return nil }

type RoiTrainingRegion struct {
	ROI       ROI    `json:"ROI"`
	ClassName string `json:"ClassName"`
}

func BootstrapRoiTrainingSample(videoPath string, regions []RoiTrainingRegion, outputDir, samplePrefix string, onProgress func(line string)) error {
	return fmt.Errorf("generator: ROI training is Suite-only")
}

func BootstrapRoiTrainingSampleOpts(videoPath string, regions []RoiTrainingRegion, outputDir, samplePrefix string, sampleEvery int, extractAudio bool, startSeconds, boxScale float64, onProgress func(line string), onPercent func(pct int)) error {
	return fmt.Errorf("generator: ROI training is Suite-only")
}

func BootstrapRoiTrainingSampleOptsCtx(ctx context.Context, videoPath string, regions []RoiTrainingRegion, outputDir, samplePrefix string, sampleEvery int, extractAudio bool, startSeconds, boxScale float64, onProgress func(line string), onPercent func(pct int)) error {
	return fmt.Errorf("generator: ROI training is Suite-only")
}

func RunRoiModelTraining(datasetDir, outputModelPath string, epochs int, device string, onProgress func(line string)) error {
	return fmt.Errorf("generator: ROI training is Suite-only")
}

func RunRoiModelTrainingWithProgress(datasetDir, outputModelPath string, epochs int, device string, onProgress func(line string), onPercent func(pct int)) error {
	return fmt.Errorf("generator: ROI training is Suite-only")
}

func RunRoiModelTrainingCtx(ctx context.Context, datasetDir, outputModelPath string, epochs int, device string, onProgress func(line string), onPercent func(pct int)) error {
	return fmt.Errorf("generator: ROI training is Suite-only")
}

func ClampRoiTrainingEpochs(epochs int) int { return epochs }

func DefaultRoiDatasetDir() string { return "" }
