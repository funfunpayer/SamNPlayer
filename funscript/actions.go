package funscript

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
)

// SaveActions schreibt eine bearbeitete Punktliste (z.B. aus dem
// Kurven-Editor) in ein bestehendes .funscript zurück - nach demselben
// Prinzip wie SaveOMarkers: nur der "actions"-Schlüssel wird ersetzt, alles
// andere (metadata, auch von anderen Werkzeugen dort abgelegte unbekannte
// Felder) bleibt Byte für Byte erhalten statt über den typisierten
// Script-Typ neu serialisiert zu werden.
//
// Positionswerte außerhalb 0-100 werden geklemmt (wie beim Einlesen in
// Parse), keine negative Zeit erlaubt, und es wird nach der Zeit sortiert -
// ein Ziehen im Editor kann die Reihenfolge während der Bewegung kurzzeitig
// durcheinanderbringen, das Ergebnis muss trotzdem eine gültige, aufsteigend
// sortierte Punktliste sein. Eine leere Liste wird abgelehnt: ein Skript
// ohne jeden Punkt lässt sich anschließend nicht mehr laden (siehe Parse).
func SaveActions(path string, actions []Action) error {
	if len(actions) == 0 {
		return fmt.Errorf("funscript: mindestens ein Punkt wird benötigt")
	}
	sorted := append([]Action(nil), actions...)
	for i := range sorted {
		if sorted[i].At < 0 {
			return fmt.Errorf("funscript: Punkt %d hat eine negative Zeit (%dms)", i, sorted[i].At)
		}
		if sorted[i].Pos < 0 {
			sorted[i].Pos = 0
		} else if sorted[i].Pos > 100 {
			sorted[i].Pos = 100
		}
	}
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].At < sorted[j].At })

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("funscript: Datei konnte nicht gelesen werden: %w", err)
	}
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(data, &doc); err != nil {
		return fmt.Errorf("funscript: ungültiges JSON: %w", err)
	}
	actionsJSON, err := json.Marshal(sorted)
	if err != nil {
		return err
	}
	doc["actions"] = actionsJSON

	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, out, 0644)
}
