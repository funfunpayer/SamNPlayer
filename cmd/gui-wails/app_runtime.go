package main

import (
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/funfunpayer/SamNPlayer/generator"
	"github.com/funfunpayer/SamNPlayer/logging"
)

// RuntimeHealth fasst den Startcheck zusammen: fehlende Ordner wurden
// angelegt (soweit möglich), fehlende optionale/pflichtige Werkzeuge
// werden nur gemeldet - der Start bricht nicht ab.
type RuntimeHealth struct {
	DirsCreated []string         `json:"dirsCreated"`
	DirsFailed  []string         `json:"dirsFailed"`
	Deps        []RuntimeDepInfo `json:"deps"`
	OK          bool             `json:"ok"`
}

// RuntimeDepInfo beschreibt ein geprüftes externes Werkzeug.
type RuntimeDepInfo struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	Required bool   `json:"required"`
	Found    bool   `json:"found"`
	Path     string `json:"path,omitempty"`
	Hint     string `json:"hint,omitempty"`
}

// ensureRuntimeReady legt fehlende Nutzer-Ordner an und prüft Abhängigkeiten
// (ffmpeg Pflicht für Generator/Video, python3 optional). Ergebnis wird
// geloggt und als Event "runtime:health" ans Frontend geschickt.
func (a *App) ensureRuntimeReady() RuntimeHealth {
	h := RuntimeHealth{OK: true}

	for _, dir := range runtimeDirs() {
		if dir == "" {
			continue
		}
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			continue
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			logging.Warn("startup: Ordner konnte nicht angelegt werden", "pfad", dir, "fehler", err)
			h.DirsFailed = append(h.DirsFailed, dir)
			h.OK = false
			continue
		}
		logging.Info("startup: Ordner angelegt", "pfad", dir)
		h.DirsCreated = append(h.DirsCreated, dir)
	}

	h.Deps = checkRuntimeDeps()
	for _, d := range h.Deps {
		if d.Found {
			logging.Info("startup: Abhängigkeit ok", "id", d.ID, "pfad", d.Path)
			continue
		}
		if d.Required {
			logging.Warn("startup: benötigte Abhängigkeit fehlt", "id", d.ID, "hinweis", d.Hint)
			h.OK = false
		} else {
			logging.Info("startup: optionale Abhängigkeit fehlt", "id", d.ID, "hinweis", d.Hint)
		}
	}

	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "runtime:health", h)
	}
	return h
}

// GetRuntimeHealth führt denselben Check erneut aus (z.B. aus den
// Einstellungen nach Installation von ffmpeg).
func (a *App) GetRuntimeHealth() RuntimeHealth {
	return a.ensureRuntimeReady()
}

func runtimeDirs() []string {
	var dirs []string
	if base, err := os.UserConfigDir(); err == nil {
		root := filepath.Join(base, "SamNPlayer")
		dirs = append(dirs,
			root,
			filepath.Join(root, "logs"),
			filepath.Join(root, "logs", "sessions"),
			filepath.Join(root, "models"),
			filepath.Join(root, "roi_training_dataset"),
			filepath.Join(root, "roi_training_dataset", "images", "train"),
			filepath.Join(root, "roi_training_dataset", "labels", "train"),
		)
	}
	if cache := generator.DefaultCacheDir(); cache != "" {
		dirs = append(dirs, cache)
	}
	if model := defaultAIRoiModelPath(); model != "" {
		dirs = append(dirs, filepath.Dir(model))
	}
	if ds := generator.DefaultRoiDatasetDir(); ds != "" {
		dirs = append(dirs, ds)
	}
	return dirs
}

func checkRuntimeDeps() []RuntimeDepInfo {
	deps := []RuntimeDepInfo{
		{
			ID:       "ffmpeg",
			Label:    "ffmpeg",
			Required: true,
			Hint:     "Für Video-Dekodierung und Generator nötig. Bitte ffmpeg installieren und im PATH bereitstellen.",
		},
		{
			ID:       "ffprobe",
			Label:    "ffprobe",
			Required: false,
			Hint:     "Optional für Bildgrößen/Metadaten. Kommt üblicherweise mit ffmpeg.",
		},
		{
			ID:       "python3",
			Label:    "Python 3",
			Required: false,
			Hint:     "Optional für den klassischen Python-Generator und KI-Training. Der Go-Pfad braucht kein Python.",
		},
	}
	if goruntime.GOOS == "windows" {
		deps[2].ID = "python"
		deps[2].Label = "Python"
	}
	for i := range deps {
		name := deps[i].ID
		if path, err := exec.LookPath(name); err == nil {
			deps[i].Found = true
			deps[i].Path = path
		}
	}
	return deps
}
