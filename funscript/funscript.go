// Package funscript implementiert einen Parser für das .funscript-Dateiformat.
//
// Format (vereinfacht):
//
//	{
//	  "actions": [
//	    {"at": 0,    "pos": 100},
//	    {"at": 1000, "pos": 0}
//	  ]
//	}
//
// "at" ist ein Zeitstempel in Millisekunden ab Start, "pos" eine Position
// zwischen 0 (unten/eingefahren) und 100 (oben/ausgefahren). Das Format
// wurde ursprünglich für lineare Stroker (z.B. The Handy, OSR2) entworfen.
package funscript

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
)

// Action ist ein einzelner Punkt im Skript.
type Action struct {
	At  int64 `json:"at"`  // Zeitstempel in Millisekunden
	Pos int   `json:"pos"` // Position 0-100
}

// Script ist das geparste .funscript-Dokument. Nur die für die Wiedergabe
// relevanten Felder werden abgebildet; unbekannte Felder (inverted, range,
// ...) werden stillschweigend ignoriert.
type Script struct {
	Actions  []Action `json:"actions"`
	Metadata struct {
		Duration     int64    `json:"duration"`
		Creator      string   `json:"creator"`
		QualityScore *float64 `json:"quality_score,omitempty"`
		// QualityPassed ist das Gesamturteil des Quality Doctor. Es ist NICHT
		// dasselbe wie QualityScore >= 0.5: der Quality Doctor kennt zusätzlich
		// harte Ausschlusskriterien (z.B. >40% der Videolänge ohne Tracking-
		// Daten), die unabhängig vom Punktestand zum Scheitern führen. Wer nur
		// den Score auswertet, zeigt solche Skripte fälschlich als unauffällig
		// an. Pointer, weil ältere Skripte das Feld nicht enthalten.
		QualityPassed   *bool    `json:"quality_passed,omitempty"`
		QualityWarnings []string `json:"quality_warnings,omitempty"`
	} `json:"metadata,omitempty"`
}

// Load liest und parst eine .funscript-Datei von der Festplatte.
func Load(path string) (*Script, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("funscript: Datei konnte nicht gelesen werden: %w", err)
	}
	return Parse(data)
}

// Parse parst rohe .funscript-JSON-Bytes.
func Parse(data []byte) (*Script, error) {
	var s Script
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("funscript: ungültiges JSON: %w", err)
	}
	if len(s.Actions) == 0 {
		return nil, fmt.Errorf("funscript: keine actions im Skript gefunden")
	}

	// Pos-Werte auf den gültigen Bereich klemmen (Format-Konvention: 0-100).
	// Manche Editoren/Generatoren exportieren durch Rundungsfehler oder Bugs
	// leicht außerhalb liegende Werte (z.B. -1 oder 101) - klemmen statt
	// abzulehnen, damit ein sonst brauchbares Skript nicht komplett verworfen
	// werden muss.
	for i := range s.Actions {
		if s.Actions[i].Pos < 0 {
			s.Actions[i].Pos = 0
		} else if s.Actions[i].Pos > 100 {
			s.Actions[i].Pos = 100
		}
	}

	// Actions müssen zeitlich sortiert sein - manche Generatoren/Editoren
	// garantieren das nicht zuverlässig.
	sort.Slice(s.Actions, func(i, j int) bool {
		return s.Actions[i].At < s.Actions[j].At
	})

	return &s, nil
}

// Duration gibt die Gesamtlänge des Skripts zurück (Zeitpunkt der letzten Action).
func (s *Script) Duration() int64 {
	if len(s.Actions) == 0 {
		return 0
	}
	return s.Actions[len(s.Actions)-1].At
}
