package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// Video first (Owner 14 Oct): the Player opens a video or a script. For a
// video SamNPlayer looks for its script itself; only when there is none does
// it offer to create one.
//
// v0.8 (docs/ROADMAP_2026-10.md §3 item 1, seen in all other players): the
// search also covers the user's script folders, tolerates differently
// written names ("Clip (2019) [1080p]" ↔ "clip"), and returns every variant
// of one video (Soft / Hard / another author) so the Player can switch.

var videoExts = []string{".mp4", ".mkv", ".avi", ".mov", ".wmv", ".m4v", ".webm", ".ts", ".m2ts", ".flv", ".mpg", ".mpeg", ".3gp", ".ogv"}

// prefScriptFolders: extra folders searched for scripts, one path per line.
const prefScriptFolders = "library.script_folders"

// Limits for walking a script folder: deep libraries stay responsive.
const (
	scriptFolderMaxDepth = 6
	scriptFolderMaxFiles = 50000
)

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

// ScriptVariant is one script that fits a video. Label is "" for the
// script with the video's own name, else what follows it ("Soft",
// "by Someone", …). Beside: it lies next to the video.
type ScriptVariant struct {
	Path   string `json:"path"`
	Label  string `json:"label"`
	Beside bool   `json:"beside"`
}

// VideoScriptMatch is the script found for a video (Found false: none yet).
// ScriptPath is the best one; Variants lists all, best first.
type VideoScriptMatch struct {
	VideoPath  string          `json:"videoPath"`
	ScriptPath string          `json:"scriptPath"`
	Found      bool            `json:"found"`
	Variants   []ScriptVariant `json:"variants"`
}

// FindScriptForVideo looks beside the video and in the script folders for
// scripts with the video's name, written loosely or with a variant label.
func (a *App) FindScriptForVideo(videoPath string) (VideoScriptMatch, error) {
	return findScriptForVideo(videoPath, a.scriptFolders()), nil
}

// GetScriptFolders lists the user's script folders.
func (a *App) GetScriptFolders() []string { return a.scriptFolders() }

// AddScriptFolder asks for a folder and adds it; it returns the new list.
func (a *App) AddScriptFolder() ([]string, error) {
	dir, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{Title: "Add a script folder"})
	if err != nil || dir == "" {
		return a.scriptFolders(), err
	}
	return a.setScriptFolders(append(a.scriptFolders(), dir))
}

// RemoveScriptFolder removes one folder from the list.
func (a *App) RemoveScriptFolder(dir string) ([]string, error) {
	var keep []string
	for _, d := range a.scriptFolders() {
		if d != dir {
			keep = append(keep, d)
		}
	}
	return a.setScriptFolders(keep)
}

func (a *App) scriptFolders() []string {
	if a.settings == nil {
		return nil
	}
	return splitFolders(a.settings.GetString(prefScriptFolders, ""))
}

func (a *App) setScriptFolders(dirs []string) ([]string, error) {
	clean := splitFolders(strings.Join(dirs, "\n"))
	if err := a.settings.Set(prefScriptFolders, strings.Join(clean, "\n")); err != nil {
		return a.scriptFolders(), err
	}
	return clean, nil
}

// splitFolders: one path per line, trimmed, without blanks or duplicates.
func splitFolders(s string) []string {
	var out []string
	seen := map[string]bool{}
	for _, line := range strings.Split(s, "\n") {
		d := strings.TrimSpace(line)
		if d == "" || seen[d] {
			continue
		}
		seen[d] = true
		out = append(out, d)
	}
	return out
}

var (
	reBrackets = regexp.MustCompile(`[\[{][^\]}]*[\]}]`)
	reYear     = regexp.MustCompile(`\((19|20)\d\d\)`)
	reSep      = regexp.MustCompile(`[\s._\-]+`)
	// Release tags that differ between a video and its script's name.
	reTags = regexp.MustCompile(`(?i)\b(2160p|1440p|1080p|720p|480p|4k|8k|uhd|fhd|hd|x264|x265|h264|h265|hevc|av1|60fps|30fps|vr180|vr|sbs|lr|180x180|3dh)\b`)
)

// normalizeTitle makes names comparable: lower case, no brackets, years,
// release tags or separators.
func normalizeTitle(stem string) string {
	s := strings.ToLower(stem)
	s = reBrackets.ReplaceAllString(s, " ")
	s = reYear.ReplaceAllString(s, " ")
	s = reSep.ReplaceAllString(s, " ")
	s = reTags.ReplaceAllString(s, " ")
	s = strings.Join(strings.Fields(s), " ")
	return strings.Trim(s, " ()")
}

// Labels that name an axis file, not a variant (MultiFunPlayer / OFS names).
var axisLabels = map[string]bool{
	"surge": true, "sway": true, "twist": true, "roll": true, "pitch": true,
	"l0": true, "l1": true, "l2": true, "r0": true, "r1": true, "r2": true,
	"v0": true, "v1": true, "v2": true, "a0": true, "a1": true, "a2": true,
	"vib": true, "vibe": true, "vibrate": true, "suck": true, "suction": true,
	"lube": true, "valve": true, "stroke": true, "raw": true,
}

// variantLabel returns the label of script stem against video stem, or
// ok=false when the script does not belong to that video.
func variantLabel(scriptStem, videoStem string) (label string, ok bool) {
	if strings.EqualFold(scriptStem, videoStem) {
		return "", true
	}
	s, v := normalizeTitle(scriptStem), normalizeTitle(videoStem)
	if v == "" {
		return "", false
	}
	if s == v {
		return "", true
	}
	if !strings.HasPrefix(s, v+" ") {
		return "", false
	}
	rest := strings.TrimSpace(strings.TrimPrefix(s, v))
	rest = strings.Trim(rest, " ()-")
	if rest == "" {
		return "", true
	}
	// "Clip 2" is another clip, not a variant of "Clip"; an axis file is no
	// variant either.
	if strings.Trim(rest, "0123456789 ") == "" || axisLabels[rest] || len(rest) > 40 {
		return "", false
	}
	return rest, true
}

type scriptCand struct {
	ScriptVariant
	rank int
}

func findScriptForVideo(videoPath string, folders []string) VideoScriptMatch {
	out := VideoScriptMatch{VideoPath: videoPath}
	if videoPath == "" {
		return out
	}
	videoDir := filepath.Dir(videoPath)
	videoStem := strings.TrimSuffix(filepath.Base(videoPath), filepath.Ext(videoPath))
	var cands []scriptCand
	seen := map[string]bool{}
	consider := func(path string, beside bool) {
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".samn" && ext != ".funscript" {
			return
		}
		key := strings.ToLower(filepath.Clean(path))
		if seen[key] {
			return
		}
		stem := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		label, ok := variantLabel(stem, videoStem)
		if !ok {
			return
		}
		seen[key] = true
		// Lower rank = better: beside the video, the video's own name, exact
		// spelling, then the Emotion Script before .funscript.
		rank := 0
		if !beside {
			rank += 100
		}
		if label != "" {
			rank += 50
		}
		if stem != videoStem {
			rank += 10
		}
		if ext == ".funscript" {
			rank += 2
		}
		cands = append(cands, scriptCand{ScriptVariant{Path: path, Label: label, Beside: beside}, rank})
	}
	if entries, err := os.ReadDir(videoDir); err == nil {
		for _, e := range entries {
			if !e.IsDir() {
				consider(filepath.Join(videoDir, e.Name()), true)
			}
		}
	}
	for _, root := range folders {
		walkScriptFolder(root, func(p string) { consider(p, filepath.Dir(p) == videoDir) })
	}
	sort.SliceStable(cands, func(i, j int) bool {
		if cands[i].rank != cands[j].rank {
			return cands[i].rank < cands[j].rank
		}
		if cands[i].Label != cands[j].Label {
			return cands[i].Label < cands[j].Label
		}
		return cands[i].Path < cands[j].Path
	})
	// One entry per variant: a .samn and its .funscript of the same name
	// count once (the better one, already first).
	byName := map[string]bool{}
	for _, c := range cands {
		k := strings.ToLower(strings.TrimSuffix(c.Path, filepath.Ext(c.Path)))
		if byName[k] {
			continue
		}
		byName[k] = true
		out.Variants = append(out.Variants, c.ScriptVariant)
	}
	if len(out.Variants) > 0 {
		out.ScriptPath, out.Found = out.Variants[0].Path, true
	}
	return out
}

// walkScriptFolder calls fn for every script under root, within the depth
// and file limits; unreadable parts are skipped.
func walkScriptFolder(root string, fn func(string)) {
	root = filepath.Clean(root)
	baseDepth := strings.Count(root, string(filepath.Separator))
	files := 0
	_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			if d != nil && d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			if strings.Count(p, string(filepath.Separator))-baseDepth >= scriptFolderMaxDepth {
				return filepath.SkipDir
			}
			return nil
		}
		files++
		if files > scriptFolderMaxFiles {
			return filepath.SkipAll
		}
		ext := strings.ToLower(filepath.Ext(p))
		if ext == ".samn" || ext == ".funscript" {
			fn(p)
		}
		return nil
	})
}
