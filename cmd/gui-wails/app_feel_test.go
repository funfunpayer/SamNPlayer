package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/funfunpayer/SamNPlayer/funscript"
)

func TestSaveFeelSettingsOnFunscript(t *testing.T) {
	path := filepath.Join(t.TempDir(), "s.funscript")
	raw, _ := json.Marshal(map[string]any{"actions": []map[string]any{{"at": 0, "pos": 10}, {"at": 400, "pos": 90}, {"at": 800, "pos": 10}}})
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	a := NewApp()
	if _, err := a.LoadFunscript(path); err != nil {
		t.Fatal(err)
	}
	if err := a.SaveFeelSettings("oral", true, 0.3); err != nil {
		t.Fatal(err)
	}
	got, err := a.GetFeelSettings()
	if err != nil {
		t.Fatal(err)
	}
	if got.SceneFeel != funscript.SceneFeelOral || !got.VibrationFollowsSuction || got.Balance != 0.3 || len(got.SceneFeels) != 4 {
		t.Fatalf("feel %+v", got)
	}
	if err := a.SaveFeelSettings("", false, 0); err != nil {
		t.Fatal(err)
	}
	got, _ = a.GetFeelSettings()
	if got.SceneFeel != "" || got.VibrationFollowsSuction || got.Balance != 0 {
		t.Fatalf("reset feel %+v", got)
	}
	if err := a.SaveFeelSettings("dance", false, 0); err == nil {
		t.Fatal("unknown scene accepted")
	}
}

func TestSaveFeelSettingsRebakesEmotionScript(t *testing.T) {
	path := filepath.Join(t.TempDir(), "imported.funscript")
	raw, _ := json.Marshal(map[string]any{"actions": []map[string]any{{"at": 0, "pos": 10}, {"at": 400, "pos": 90}, {"at": 800, "pos": 10}, {"at": 1200, "pos": 90}}})
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	a := NewApp()
	if _, err := a.LoadFunscript(path); err != nil {
		t.Fatal(err)
	}
	res, err := a.OptimizeLoadedFor(OptimizeTargetNeo2) // .samn, plays from baked axes
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.LoadFunscript(res.Path); err != nil {
		t.Fatal(err)
	}
	before := a.loadedScript().Metadata.SamnAxes
	if before == nil {
		t.Fatal("no baked axes")
	}
	if err := a.SaveFeelSettings("oral", true, 0); err != nil {
		t.Fatal(err)
	}
	after := a.loadedScript().Metadata.SamnAxes
	if after == nil || len(after.Vibration) == 0 {
		t.Fatal("axes lost")
	}
	same := len(before.Vibration) == len(after.Vibration)
	if same {
		for i := range before.Vibration {
			if before.Vibration[i] != after.Vibration[i] {
				same = false
				break
			}
		}
	}
	if same {
		t.Fatal("baked vibration did not change with the feel")
	}
}
