package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/funfunpayer/SamNPlayer/generator"
	"github.com/funfunpayer/SamNPlayer/samn"
)

// ScanSceneMap runs the quick rhythm heatmap (P1 / docs/SCENE_MAP_PLAN.md).
// Explicit Advanced control only — not automatic before Generate (Owner 24 Sep).
// n <= 0 uses generator.DefaultSceneMapWindows (6).
func (a *App) ScanSceneMap(videoPath string, n int) (generator.SceneMapDTO, error) {
	return generator.ScanSceneMap(videoPath, n)
}

// SceneMapAvailable reports whether the OpenCV-backed quick scan can run.
func (a *App) SceneMapAvailable() bool {
	return generator.NativeTrackingAvailable()
}

// LoadSceneMapForVideo restores Advanced scene-map state from a companion
// .samn beside the video (P4 persist → Create). Missing/old files return
// Found=false without error. Does not change Generate defaults.
func (a *App) LoadSceneMapForVideo(videoPath string) (generator.SceneMapLoad, error) {
	return generator.LoadSceneMapBesideVideo(videoPath)
}

// ExportSceneMapLearning writes local scene_map_learning artifacts for one
// .samn (or companion beside a video/funscript). Requires Settings
// Collect learning data. Never writes YOLO images/labels/train.
func (a *App) ExportSceneMapLearning(path string) (generator.LearningExportResult, error) {
	samnPath, err := resolveSamnForLearning(path)
	if err != nil {
		return generator.LearningExportResult{}, err
	}
	optIn := false
	if a.settings != nil {
		optIn = a.settings.GetBool(prefCollectLearningData, false)
	}
	outDir := ""
	if a.settings != nil {
		if root := strings.TrimSpace(a.settings.GetString(prefRoiDatasetDir, "")); root != "" {
			outDir = filepath.Join(root, generator.SceneMapLearningSubdir)
		}
	}
	return generator.ExportSceneMapLearning(samnPath, generator.LearningExportOptions{
		OptIn:     optIn,
		OutputDir: outDir,
	})
}

// DeleteSceneMapLearningData removes only scene_map_learning under the
// configured ROI dataset root (or the default). Hand YOLO samples stay.
func (a *App) DeleteSceneMapLearningData() error {
	root := ""
	if a.settings != nil {
		root = strings.TrimSpace(a.settings.GetString(prefRoiDatasetDir, ""))
	}
	return generator.DeleteSceneMapLearningData(root)
}

func resolveSamnForLearning(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("no video or .samn path")
	}
	if strings.EqualFold(filepath.Ext(path), ".samn") {
		return path, nil
	}
	return samn.CompanionSamnPath(path), nil
}
