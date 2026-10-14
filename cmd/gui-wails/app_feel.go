package main

import (
	"fmt"
	"strings"

	"github.com/funfunpayer/SamNPlayer/funscript"
	"github.com/funfunpayer/SamNPlayer/generator"
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

// ChapterScene is one chapter as a scene with its feel and a suggestion
// (step 4: scene per chapter, suggested — you confirm before it is used).
type ChapterScene struct {
	Index      int     `json:"index"`
	Name       string  `json:"name"`
	StartMs    int64   `json:"startMs"`
	EndMs      int64   `json:"endMs"`
	Feel       string  `json:"feel"`       // saved feel for this chapter ("" = Classic / script-wide)
	HasFeel    bool    `json:"hasFeel"`    // a feel is saved for exactly this chapter
	Suggested  string  `json:"suggested"`  // "" = no suggestion
	Source     string  `json:"source"`     // "scene detection" | "contact mark"
	Confidence float64 `json:"confidence"` // 0–1
}

// SuggestChapterScenes lists the loaded script's chapters with a suggested
// scene feel each. Sources, strongest first:
//   - scene detection (<clip>.scene.json beside the video): the scene types
//     overlapping the chapter, weighted by overlap × confidence;
//   - the contact mark's class (mouth → Oral, nipple / breast → Breast), for
//     the whole script, so with a lower confidence.
//
// Nothing is applied here.
func (a *App) SuggestChapterScenes() ([]ChapterScene, error) {
	script := a.loadedScript()
	if script == nil {
		return nil, fmt.Errorf("no script loaded")
	}
	marks, err := a.GetScriptChapterMarks()
	if err != nil {
		return nil, err
	}
	end := int64(0)
	if n := len(script.Actions); n > 0 {
		end = script.Actions[n-1].At
	}
	opts := funscript.MapOptionsFromScript(script)
	var props []generator.SceneProposal
	if v := a.currentVideoPath(); v != "" {
		if sp, err := generator.LoadSceneProposals(SceneProposalsPathBesideVideo(v)); err == nil {
			props = sp.Proposals
		}
	}
	markFeel := ""
	if cm := script.Metadata.ContactMarks; cm != nil && cm.Primary != nil {
		markFeel = funscript.NormalizeSceneFeel(contactClassScene(cm.Primary.Class))
	}

	out := make([]ChapterScene, 0, len(marks))
	for i, m := range marks {
		start, stop := m.StartTime, m.EndTime
		if stop <= start {
			stop = end
			if i+1 < len(marks) && marks[i+1].StartTime > start {
				stop = marks[i+1].StartTime
			}
		}
		cs := ChapterScene{Index: i, Name: m.Name, StartMs: start, EndMs: stop}
		for _, r := range opts.SceneRanges {
			if r.StartMs == start && r.EndMs == stop {
				cs.Feel, cs.HasFeel = r.Feel, true
			}
		}
		// Scene detection: overlap × confidence per scene feel.
		weight := map[string]float64{}
		total := 0.0
		for _, p := range props {
			lo, hi := max64(p.StartMs, start), min64(p.EndMs, stop)
			if hi <= lo {
				continue
			}
			w := float64(hi-lo) * clampUnit01(p.Confidence)
			total += float64(hi - lo)
			if f := funscript.NormalizeSceneFeel(p.SceneType); f != "" {
				weight[f] += w
			}
		}
		best, bestW := "", 0.0
		for f, w := range weight {
			if w > bestW {
				best, bestW = f, w
			}
		}
		if best != "" && total > 0 {
			cs.Suggested, cs.Source, cs.Confidence = best, "scene detection", bestW/total
		} else if markFeel != "" {
			cs.Suggested, cs.Source, cs.Confidence = markFeel, "contact mark", 0.5
		}
		out = append(out, cs)
	}
	return out, nil
}

// contactClassScene: the contact mark's class as a scene name.
func contactClassScene(class string) string {
	c := strings.ToLower(class)
	switch {
	case strings.Contains(c, "mouth"), strings.Contains(c, "lips"):
		return "oral"
	case strings.Contains(c, "nipple"), strings.Contains(c, "breast"):
		return "breast"
	case strings.Contains(c, "hand"):
		return "hand"
	}
	return ""
}

func max64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func min64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func clampUnit01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// SaveChapterScenes stores a scene feel per chapter (only the ranges given;
// an empty list removes them) and re-bakes an Emotion Script that plays from
// axes.
func (a *App) SaveChapterScenes(ranges []funscript.SceneFeelRange) error {
	path := a.loadedScriptPath()
	if path == "" || a.loadedScript() == nil {
		return fmt.Errorf("no script loaded")
	}
	for _, r := range ranges {
		if r.Feel != "" && funscript.NormalizeSceneFeel(r.Feel) == "" {
			return fmt.Errorf("unknown scene feel %q", r.Feel)
		}
	}
	if samn.IsSamnPath(path) {
		doc, err := samn.Load(path)
		if err != nil {
			return err
		}
		doc.Recipe.SceneRanges = funscript.NormalizeSceneRanges(ranges)
		if funscript.NormalizePlaybackSource(doc.PlaybackSource) == samn.PlaybackAxes {
			if err := doc.BakeNeoAxes(); err != nil {
				return err
			}
		}
		if err := samn.Save(path, doc); err != nil {
			return err
		}
		if err := doc.ExportFunscript(samn.CompanionFunscriptPath(path)); err != nil {
			logging.Warn("app: companion funscript export failed after chapter scenes", "path", path, "error", err)
		}
		return a.reloadLoadedScript()
	}
	if err := funscript.SaveSceneRanges(path, ranges); err != nil {
		return err
	}
	reloaded, err := funscript.Load(path)
	if err != nil {
		return err
	}
	a.setLoadedScript(path, reloaded)
	return nil
}
