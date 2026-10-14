package main

import (
	"fmt"

	"github.com/funfunpayer/SamNPlayer/funscript"
	"github.com/funfunpayer/SamNPlayer/logging"
	"github.com/funfunpayer/SamNPlayer/samn"
)

// FeelSettings is the scene feel of the loaded script (feel_scene.go): how
// suction follows the stroke per scene, vibration following suction, and the
// vibration ↔ suction balance (−1 … 1, >0 more suction).
type FeelSettings struct {
	SceneFeel               string   `json:"sceneFeel"`
	VibrationFollowsSuction bool     `json:"vibrationFollowsSuction"`
	Balance                 float64  `json:"balance"`
	SceneFeels              []string `json:"sceneFeels"`
}

// GetFeelSettings reads the loaded script's feel recipe.
func (a *App) GetFeelSettings() (FeelSettings, error) {
	out := FeelSettings{SceneFeels: funscript.SceneFeels}
	script := a.loadedScript()
	if script == nil {
		return out, fmt.Errorf("no script loaded")
	}
	o := funscript.MapOptionsFromScript(script)
	out.SceneFeel, out.VibrationFollowsSuction, out.Balance = o.SceneFeel, o.VibrationFollowsSuction, o.Balance
	return out, nil
}

// SaveFeelSettings writes the feel recipe into the loaded script and
// reloads it. An Emotion Script that plays from baked axes is re-baked so the
// device feels the new setting. Empty scene, follow off and balance 0 restore
// the old mapping.
func (a *App) SaveFeelSettings(sceneFeel string, followSuction bool, balance float64) error {
	path := a.loadedScriptPath()
	if path == "" || a.loadedScript() == nil {
		return fmt.Errorf("no script loaded")
	}
	if sceneFeel != "" && funscript.NormalizeSceneFeel(sceneFeel) == "" {
		return fmt.Errorf("unknown scene feel %q", sceneFeel)
	}
	if samn.IsSamnPath(path) {
		doc, err := samn.Load(path)
		if err != nil {
			return err
		}
		doc.ApplyFeelRecipe(sceneFeel, followSuction, balance)
		if funscript.NormalizePlaybackSource(doc.PlaybackSource) == samn.PlaybackAxes {
			if err := doc.BakeNeoAxes(); err != nil {
				return err
			}
		}
		if err := samn.Save(path, doc); err != nil {
			return err
		}
		if err := doc.ExportFunscript(samn.CompanionFunscriptPath(path)); err != nil {
			logging.Warn("app: companion funscript export failed after feel save", "path", path, "error", err)
		}
		return a.reloadLoadedScript()
	}
	if err := funscript.SaveFeelRecipe(path, sceneFeel, followSuction, balance); err != nil {
		return err
	}
	reloaded, err := funscript.Load(path)
	if err != nil {
		return err
	}
	a.setLoadedScript(path, reloaded)
	return nil
}
