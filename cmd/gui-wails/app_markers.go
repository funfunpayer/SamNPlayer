package main

import (
	"encoding/json"
	"os"
)

// Marker ist ein vom Nutzer markierter Zeitbereich im Skript (z.B. der
// Höhepunkt gegen Ende), in dem Extended-O automatisch ausgelöst werden
// kann. Liegt als kleine JSON-Datei neben dem Skript
// ("szene.funscript" -> "szene.funscript.marker.json"), bewusst nicht im
// funscript selbst - das Format hat kein Standardfeld dafür, und ein
// zusätzliches, unbekanntes Feld in der .funscript-Datei könnte andere
// Player stören.
type Marker struct {
	StartMs int64 `json:"startMs"`
	EndMs   int64 `json:"endMs"`
}

func markerPath(scriptPath string) string {
	return scriptPath + ".marker.json"
}

// SaveMarker speichert einen Zeitbereich für das aktuell geladene Skript.
// EndMs <= StartMs löscht eine vorhandene Markierung wieder - "man muss
// die Funktion auch ausschlagen können".
func (a *App) SaveMarker(scriptPath string, startMs, endMs int64) error {
	path := markerPath(scriptPath)
	if endMs <= startMs {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	b, err := json.Marshal(Marker{StartMs: startMs, EndMs: endMs})
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0644)
}

// GetMarker lädt eine vorhandene Markierung, falls es eine gibt. Liefert
// (nil, nil) wenn keine existiert - kein Fehler, das ist der Normalfall.
func (a *App) GetMarker(scriptPath string) (*Marker, error) {
	b, err := os.ReadFile(markerPath(scriptPath))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var m Marker
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return &m, nil
}
