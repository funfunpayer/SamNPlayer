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
	return zone, a.SaveOMarkers(path, markers)
}

// ApplyRingDown hängt gedämpfte Halbzyklen nach atMs an das geladene Skript
// und speichert. cycles 1 oder 2. Prefix bleibt unverändert; Ende ist 0.
// Nicht automatisch während live Extended-O — nur explizit vom UI.
func (a *App) ApplyRingDown(atMs int64, cycles int) error {
	if a.currentScript == nil {
		return fmt.Errorf("kein Skript geladen")
	}
	if a.scriptPath == "" {
		return fmt.Errorf("kein Skriptpfad")
	}
	if cycles < 1 {
		cycles = 1
	}
	if cycles > 2 {
		cycles = 2
	}
	actions := a.currentScript.Actions
	lastPos := 50
	for _, act := range actions {
		if act.At > atMs {
			break
		}
		lastPos = act.Pos
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
