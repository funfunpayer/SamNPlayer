package main

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// Video first (Owner 14 Oct): the Player opens a video or a script. For a
// video SamNPlayer looks for its script itself; only when there is none does
// it offer to create one.

var videoExts = []string{".mp4", ".mkv", ".avi", ".mov", ".wmv", ".m4v", ".webm", ".ts", ".m2ts", ".flv", ".mpg", ".mpeg", ".3gp", ".ogv"}

// PickMediaFile opens one file dialog for a video or a script.
func (a *App) PickMediaFile() (string, error) {
	vids := "*" + strings.Join(videoExts, ";*")
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Open video or script",
		Filters: []runtime.FileFilter{
			{DisplayName: "Video or script", Pattern: vids + ";*.samn;*.funscript"},
			{DisplayName: "Videos", Pattern: vids},
			{DisplayName: "Scripts (*.samn, *.funscript)", Pattern: "*.samn;*.funscript"},
		},
	})
}

// VideoScriptMatch is the script found for a video (Found false: none yet).
type VideoScriptMatch struct {
	VideoPath  string `json:"videoPath"`
	ScriptPath string `json:"scriptPath"`
	Found      bool   `json:"found"`
}

// FindScriptForVideo looks beside the video for a script with the same name
// (case does not matter): the Emotion Script (.samn) first, then .funscript.
// Axis companions (name.surge.funscript, …) do not count.
func (a *App) FindScriptForVideo(videoPath string) (VideoScriptMatch, error) {
	return findScriptForVideo(videoPath), nil
}

func findScriptForVideo(videoPath string) VideoScriptMatch {
	out := VideoScriptMatch{VideoPath: videoPath}
	if videoPath == "" {
		return out
	}
	dir := filepath.Dir(videoPath)
	base := strings.TrimSuffix(filepath.Base(videoPath), filepath.Ext(videoPath))
	entries, err := os.ReadDir(dir)
	if err != nil {
		return out
	}
	best, bestRank := "", 99
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		ext := strings.ToLower(filepath.Ext(name))
		rank := 0
		switch ext {
		case ".samn":
			rank = 0
		case ".funscript":
			rank = 2
		default:
			continue
		}
		stem := strings.TrimSuffix(name, filepath.Ext(name))
		if !strings.EqualFold(stem, base) {
			continue
		}
		if stem != base {
			rank++ // exact case wins
		}
		if rank < bestRank {
			best, bestRank = filepath.Join(dir, name), rank
		}
	}
	if best != "" {
		out.ScriptPath, out.Found = best, true
	}
	return out
}
