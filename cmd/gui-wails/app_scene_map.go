package main

import (
	"github.com/funfunpayer/SamNPlayer/generator"
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
