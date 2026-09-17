package main

import (
	"fmt"

	"github.com/funfunpayer/SamNPlayer/funscript"
)

// GetScriptActions liefert die vollen, nicht resampleten Punkte des
// aktuell geladenen Skripts - anders als GetScriptCurve, das für die
// Anzeige auf maxPoints herunterrechnet. Grundlage für den Kurven-Editor:
// bearbeitet werden muss der echte Punkt, nicht ein Anzeige-Kompromiss.
func (a *App) GetScriptActions() ([]funscript.Action, error) {
	script := a.loadedScript()
	if script == nil {
		return nil, fmt.Errorf("kein Skript geladen")
	}
	return script.Actions, nil
}

// SaveScriptActions schreibt eine im Editor geänderte Punktliste zurück und
// lädt das Skript danach über denselben Pfad wie LoadFunscript neu ein - so
// bleibt a.currentScript exakt das, was auf der Platte steht (geklemmte
// Positionswerte, sortierte Reihenfolge), statt dass Speicher- und
// Dateizustand leicht auseinanderlaufen.
func (a *App) SaveScriptActions(actions []funscript.Action) error {
	path := a.loadedScriptPath()
	if path == "" {
		return fmt.Errorf("kein Skript geladen")
	}
	if err := funscript.SaveActions(path, actions); err != nil {
		return err
	}
	reloaded, err := funscript.Load(path)
	if err != nil {
		return err
	}
	a.setLoadedScript(path, reloaded)
	return nil
}
