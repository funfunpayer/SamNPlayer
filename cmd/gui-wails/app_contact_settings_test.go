package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/funfunpayer/SamNPlayer/funscript"
)

func TestSaveContactSettingsPersistsRecipe(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tj.funscript")
	body := `{"actions":[{"at":0,"pos":20},{"at":800,"pos":90}],"metadata":{"profile":"tj","device_recipe":{"sync":"suction_position","min_suction":0.2,"tick_ms":50,"max_speed":0.5,"smoothing":0.22,"contact_vibration":true,"contact_vibration_span":0.75,"contact_vibration_curve":"linear"}}}`
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
	a := NewApp()
	script, err := funscript.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	a.setLoadedScript(path, script)

	if err := a.SaveContactSettings(true, 0.5, "soft"); err != nil {
		t.Fatalf("SaveContactSettings: %v", err)
	}
	got := a.loadedScript()
	dr := got.Metadata.DeviceRecipe
	if dr == nil || !dr.ContactVibration || dr.ContactVibrationSpan != 0.5 || dr.ContactVibrationCurve != "soft" {
		t.Fatalf("unerwartetes Rezept: %+v", dr)
	}
}

func TestSaveContactSettingsStrokeProfile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "stroke.funscript")
	body := `{"actions":[{"at":0,"pos":20},{"at":800,"pos":90}],"metadata":{"profile":"standard","device_recipe":{"sync":"suction_position","min_suction":0.2,"tick_ms":50,"max_speed":0.5,"smoothing":0.22,"contact_vibration":true,"contact_vibration_span":0.75,"contact_vibration_curve":"soft"}}}`
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
	a := NewApp()
	script, err := funscript.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	a.setLoadedScript(path, script)

	if err := a.SaveContactSettings(true, 0.55, "peak"); err != nil {
		t.Fatalf("SaveContactSettings stroke: %v", err)
	}
	got := a.loadedScript()
	dr := got.Metadata.DeviceRecipe
	if dr == nil || !dr.ContactVibration || dr.ContactVibrationSpan != 0.55 || dr.ContactVibrationCurve != "peak" {
		t.Fatalf("stroke contact recipe: %+v", dr)
	}
}
