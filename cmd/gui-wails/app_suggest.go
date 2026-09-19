package main

import (
	"fmt"

	"github.com/funfunpayer/SamNPlayer/funscript"
)

func (a *App) SuggestPolarity() (funscript.PolarityHint, error) {
	script := a.loadedScript()
	if script == nil {
		return funscript.PolarityHint{}, fmt.Errorf("no script loaded")
	}
	return funscript.SuggestPolarity(script.Actions), nil
}

func (a *App) InvertLoadedScript() error {
	script := a.loadedScript()
	path := a.loadedScriptPath()
	if script == nil {
		return fmt.Errorf("no script loaded")
	}
	if path == "" {
		return fmt.Errorf("no script path")
	}
	actions := make([]funscript.Action, len(script.Actions))
	for i, act := range script.Actions {
		actions[i] = funscript.Action{At: act.At, Pos: 100 - act.Pos}
	}
	return a.SaveScriptAxisActions(string(funscript.AxisGeneral), actions)
}

func (a *App) SuggestOZone() (funscript.OZoneSuggestion, error) {
	script := a.loadedScript()
	if script == nil {
		return funscript.OZoneSuggestion{}, fmt.Errorf("no script loaded")
	}
	return funscript.SuggestOZone(script.Actions), nil
}

func (a *App) ApplySuggestedOZone() (funscript.OZoneSuggestion, error) {
	zone, err := a.SuggestOZone()
	if err != nil {
		return zone, err
	}
	if !zone.OK {
		return zone, nil
	}
	path := a.loadedScriptPath()
	if path == "" {
		return zone, fmt.Errorf("no script path")
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
	script := a.loadedScript()
	for _, sec := range funscript.SuggestSecondaryOZones(script.Actions, zone, 2) {
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
	script := a.loadedScript()
	path := a.loadedScriptPath()
	if script == nil {
		return fmt.Errorf("no script loaded")
	}
	if path == "" {
		return fmt.Errorf("no script path")
	}
	if atMs < 0 {
		atMs = 0
	}
	actions := script.Actions
	lastPos := 0
	for _, act := range actions {
		if act.At > atMs {
			break
		}
		lastPos = act.Pos
	}
	if lastPos < 10 {
		return fmt.Errorf("position too low for ring-down (%d)", lastPos)
	}
	newActions := funscript.RingDown(actions, atMs, lastPos, cycles)
	return a.SaveScriptAxisActions(string(funscript.AxisGeneral), newActions)
}
