package main

import (
	"fmt"

	"github.com/funfunpayer/SamNPlayer/funscript"
)

func (a *App) SuggestPolarity() (funscript.PolarityHint, error) {
	if a.currentScript == nil {
		return funscript.PolarityHint{}, fmt.Errorf("kein Skript geladen")
	}
	return funscript.SuggestPolarity(a.currentScript.Actions), nil
}

func (a *App) InvertLoadedScript() error {
	if a.currentScript == nil {
		return fmt.Errorf("kein Skript geladen")
	}
	if a.scriptPath == "" {
		return fmt.Errorf("kein Skriptpfad")
	}
	actions := make([]funscript.Action, len(a.currentScript.Actions))
	for i, act := range a.currentScript.Actions {
		actions[i] = funscript.Action{At: act.At, Pos: 100 - act.Pos}
	}
	if err := funscript.SaveActions(a.scriptPath, actions); err != nil {
		return err
	}
	script, err := funscript.Load(a.scriptPath)
	if err != nil {
		return err
	}
	a.currentScript = script
	return nil
}

func (a *App) SuggestOZone() (funscript.OZoneSuggestion, error) {
	if a.currentScript == nil {
		return funscript.OZoneSuggestion{}, fmt.Errorf("kein Skript geladen")
	}
	return funscript.SuggestOZone(a.currentScript.Actions), nil
}

func (a *App) ApplySuggestedOZone() (funscript.OZoneSuggestion, error) {
	zone, err := a.SuggestOZone()
	if err != nil {
		return zone, err
	}
	if !zone.OK {
		return zone, nil
	}
	path := a.scriptPath
	if path == "" {
		return zone, fmt.Errorf("kein Skriptpfad")
	}
	if err := a.SaveMarker(path, zone.StartMs, zone.EndMs); err != nil {
		return zone, err
	}
	markers, err := a.GetOMarkers(path)
	if err != nil {
		return zone, err
	}
	markers = append(markers, funscript.OMarker{
		StartMs:   zone.StartMs,
		EndMs:     zone.EndMs,
		Kind:      funscript.OMarkerPrimary,
		Intensity: 1,
	})
	// Optionale schwächere Marker vor dem Hauptmarker (docs/NEXT.md
	// Priorität 7: "primary marker + optionally one or two secondary
	// markers earlier in the scene at lower intensity") - klassisch aus
	// demselben Signal, kein KI-Modell. Meist leer, das ist der normale
	// Fall, kein Fehler.
	for _, sec := range funscript.SuggestSecondaryOZones(a.currentScript.Actions, zone, 2) {
		markers = append(markers, secondaryMarkerFromSuggestion(sec, zone))
	}
	return zone, a.SaveOMarkers(path, markers)
}

// secondaryMarkerFromSuggestion leitet die Intensität eines sekundären
// O-Markers aus dem Stärkeverhältnis zum Hauptmarker ab (sec.Mean/
// primary.Mean) statt eine feste Zahl zu erfinden - "nicht so doll" (der
// Nutzer, 14. September 2026) wird so vom tatsächlichen Signal bestimmt,
// nicht geraten. Geklemmt auf 0.2-0.8: unter 0.2 wäre kaum wahrnehmbar,
// SuggestSecondaryOZones' eigener Filter schließt alles über 0.85 des
// Hauptmarkers ohnehin schon aus.
func secondaryMarkerFromSuggestion(sec, primary funscript.OZoneSuggestion) funscript.OMarker {
	intensity := 0.5
	if primary.Mean > 0 {
		intensity = sec.Mean / primary.Mean
		if intensity < 0.2 {
			intensity = 0.2
		}
		if intensity > 0.8 {
			intensity = 0.8
		}
	}
	return funscript.OMarker{
		StartMs:   sec.StartMs,
		EndMs:     sec.EndMs,
		Kind:      funscript.OMarkerSecondary,
		Intensity: intensity,
	}
}

// ApplyRingDown hängt nach atMs gedämpfte Halbzyklen an (siehe funscript.RingDown)
// und speichert das Skript. cycles wird auf 1–2 geklemmt. Position unter 10
// wird abgelehnt, damit keine unnötigen Null-Zyklen entstehen. Bestätigung
// bleibt der UI überlassen.
func (a *App) ApplyRingDown(atMs int64, cycles int) error {
	if a.currentScript == nil {
		return fmt.Errorf("kein Skript geladen")
	}
	if a.scriptPath == "" {
		return fmt.Errorf("kein Skriptpfad")
	}
	if atMs < 0 {
		atMs = 0
	}
	actions := a.currentScript.Actions
	lastPos := 0
	for _, act := range actions {
		if act.At > atMs {
			break
		}
		lastPos = act.Pos
	}
	if lastPos < 10 {
		return fmt.Errorf("Position zu niedrig für Ring-down (%d)", lastPos)
	}
	newActions := funscript.RingDown(actions, atMs, lastPos, cycles)
	if err := funscript.SaveActions(a.scriptPath, newActions); err != nil {
		return err
	}
	script, err := funscript.Load(a.scriptPath)
	if err != nil {
		return err
	}
	a.currentScript = script
	return nil
}
