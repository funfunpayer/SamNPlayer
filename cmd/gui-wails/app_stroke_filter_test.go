package main

import (
	"path/filepath"
	"testing"
)

func TestStrokeFilterSettingsRoundTrip(t *testing.T) {
	a := NewApp()
	a.settings = &settingsStore{path: filepath.Join(t.TempDir(), "settings.json"), data: map[string]any{}}
	if got := a.GetStrokeFilter(); got.Smooth != "" || got.MaxSpeed != 0 || got.IdleFill || got.HomeOnPause {
		t.Fatalf("default off: %+v", got)
	}
	got, err := a.SetStrokeFilter(StrokeFilterSettings{Smooth: "makima", MaxSpeed: 20, IdleFill: true, HomeOnPause: true})
	if err != nil {
		t.Fatal(err)
	}
	if got.Smooth != "makima" || got.MaxSpeed != 50 || !got.IdleFill || !got.HomeOnPause {
		t.Fatalf("stored and clamped: %+v", got)
	}
	// A fresh App on the same settings reads it back.
	b := NewApp()
	b.settings = a.settings
	if got := b.GetStrokeFilter(); got.Smooth != "makima" || got.MaxSpeed != 50 || !got.IdleFill || !got.HomeOnPause {
		t.Fatalf("read back: %+v", got)
	}
	if got, _ := a.SetStrokeFilter(StrokeFilterSettings{Smooth: "cubic", MaxSpeed: 5000}); got.Smooth != "" || got.MaxSpeed != 1000 {
		t.Fatalf("unknown mode off, speed capped: %+v", got)
	}
}
