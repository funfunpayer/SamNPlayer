package main

import (
	"fmt"
	"os"

	"github.com/funfunpayer/SamNPlayer/funscript"
	"github.com/funfunpayer/SamNPlayer/samn"
)

// GeneratedReview is the post-generate check that does not need a loaded player script.
type GeneratedReview struct {
	Path     string                    `json:"path"`
	Polarity funscript.PolarityHint    `json:"polarity"`
	OZone    funscript.OZoneSuggestion `json:"ozone"`
}

// ReviewGeneratedScript liest eine gerade erzeugte Datei und liefert
// Polaritäts- und O-Zonen-Vorschlag. Ändert nichts an der Datei.
func (a *App) ReviewGeneratedScript(path string) (GeneratedReview, error) {
	if path == "" {
		return GeneratedReview{}, fmt.Errorf("no path")
	}
	script, err := a.loadScriptDocument(preferSamnCompanion(path))
	if err != nil {
		return GeneratedReview{}, err
	}
	return GeneratedReview{
		Path:     path,
		Polarity: funscript.SuggestPolarity(script.Actions),
		OZone:    funscript.SuggestOZone(script.Actions),
	}, nil
}

// InvertScriptAtPath spiegelt Positionen einer Datei, auch wenn sie nicht
// die gerade geladene Wiedergabe ist (Generator-Nachlauf).
func (a *App) InvertScriptAtPath(path string) error {
	if path == "" {
		return fmt.Errorf("no path")
	}
	path = preferSamnCompanion(path)
	script, err := a.loadScriptDocument(path)
	if err != nil {
		return err
	}
	actions := make([]funscript.Action, len(script.Actions))
	for i, act := range script.Actions {
		actions[i] = funscript.Action{At: act.At, Pos: 100 - act.Pos}
	}
	if samn.IsSamnPath(path) {
		doc, err := samn.Load(path)
		if err != nil {
			return err
		}
		doc.General = actions
		if err := samn.Save(path, doc); err != nil {
			return err
		}
		_ = doc.ExportFunscript(samn.CompanionFunscriptPath(path))
	} else if err := funscript.SaveActions(path, actions); err != nil {
		return err
	}
	if a.loadedScriptPath() == path {
		return a.reloadLoadedScript()
	}
	return nil
}

// preferSamnCompanion returns path's companion .samn when it exists next to a .funscript.
func preferSamnCompanion(path string) string {
	if path == "" || samn.IsSamnPath(path) {
		return path
	}
	companion := samn.CompanionSamnPath(path)
	if st, err := os.Stat(companion); err == nil && !st.IsDir() {
		return companion
	}
	return path
}
