package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/funfunpayer/SamNPlayer/funscript"
)

func writeTestScript(t *testing.T, path string, actions []funscript.Action) {
	t.Helper()
	raw, err := json.Marshal(funscript.Script{Actions: actions})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0644); err != nil {
		t.Fatal(err)
	}
}

func TestApplyAutoOZoneMarkerWritesPrimaryMarker(t *testing.T) {
	path := filepath.Join(t.TempDir(), "g.funscript")
	actions := make([]funscript.Action, 0, 101)
	for i := 0; i <= 100; i++ {
		pos := 20
		if i >= 90 {
			pos = 90
		}
		actions = append(actions, funscript.Action{At: int64(i * 100), Pos: pos})
	}
	writeTestScript(t, path, actions)

	zone, err := applyAutoOZoneMarker(path, actions)
	if err != nil {
		t.Fatal(err)
	}
	if !zone.OK {
		t.Fatalf("erwartete Zone bei hohem Ende: %+v", zone)
	}

	markers, err := funscript.LoadOMarkers(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(markers) != 1 {
		t.Fatalf("erwartete genau einen Marker, got %d", len(markers))
	}
	if markers[0].Kind != funscript.OMarkerPrimary {
		t.Fatalf("erwartete primary, got %s", markers[0].Kind)
	}
	if markers[0].StartMs != zone.StartMs || markers[0].EndMs != zone.EndMs {
		t.Fatalf("Marker-Zeiten weichen von der Suggestion ab: %+v vs %+v", markers[0], zone)
	}
}

func TestApplyAutoOZoneMarkerFlatScriptWritesNothing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "flat.funscript")
	actions := make([]funscript.Action, 0, 51)
	for i := 0; i <= 50; i++ {
		actions = append(actions, funscript.Action{At: int64(i * 100), Pos: 20})
	}
	writeTestScript(t, path, actions)

	zone, err := applyAutoOZoneMarker(path, actions)
	if err != nil {
		t.Fatal(err)
	}
	if zone.OK {
		t.Fatalf("flaches Ende darf keine Zone liefern: %+v", zone)
	}

	markers, err := funscript.LoadOMarkers(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(markers) != 0 {
		t.Fatalf("ohne Vorschlag darf kein Marker geschrieben werden, got %d", len(markers))
	}
}
