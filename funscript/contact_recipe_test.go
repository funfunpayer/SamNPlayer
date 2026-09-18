package funscript

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveContactRecipeRoundtrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "clip.funscript")
	body := `{"actions":[{"at":0,"pos":20},{"at":1000,"pos":90}],"metadata":{"profile":"tj","device_recipe":{"sync":"suction_position","min_suction":0.2,"tick_ms":50,"max_speed":0.5,"smoothing":0.22}}}`
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
	if err := SaveContactRecipe(path, true, 0.55, "soft"); err != nil {
		t.Fatalf("SaveContactRecipe: %v", err)
	}
	s, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	dr := s.Metadata.DeviceRecipe
	if dr == nil {
		t.Fatal("device_recipe fehlt")
	}
	if !dr.ContactVibration {
		t.Error("ContactVibration nicht gesetzt")
	}
	if dr.ContactVibrationSpan != 0.55 {
		t.Errorf("span=%v", dr.ContactVibrationSpan)
	}
	if dr.ContactVibrationCurve != ContactCurveSoft {
		t.Errorf("curve=%q", dr.ContactVibrationCurve)
	}
	if dr.Sync != SyncSuctionPosition.String() {
		t.Errorf("sync verloren: %q", dr.Sync)
	}
	if len(s.Actions) != 2 {
		t.Errorf("actions verändert: %d", len(s.Actions))
	}
}
