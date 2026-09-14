package main

import (
	"encoding/json"
	"os"

	"github.com/funfunpayer/SamNPlayer/funscript"
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

// GetOMarkers/SaveOMarkers sind dünne Bindungen um funscript.LoadOMarkers/
// SaveOMarkers (siehe dort für das Format und warum es NICHT über den
// typisierten Script-Typ läuft) - im Unterschied zu Marker oben liegen
// O-Marker direkt IM Skript, nicht in einer Sidecar-Datei: sie sind
// authored data, die mit dem Skript geteilt/exportiert werden soll
// (docs/NEXT.md Priorität 7), nicht nur lokaler Player-Zustand.
func (a *App) GetOMarkers(scriptPath string) ([]funscript.OMarker, error) {
	markers, err := funscript.LoadOMarkers(scriptPath)
	if err != nil {
		return nil, err
	}
	if markers == nil {
		markers = []funscript.OMarker{}
	}
	return markers, nil
}

func (a *App) SaveOMarkers(scriptPath string, markers []funscript.OMarker) error {
	return funscript.SaveOMarkers(scriptPath, markers)
}
