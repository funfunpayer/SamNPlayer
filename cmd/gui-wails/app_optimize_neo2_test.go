package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/funfunpayer/SamNPlayer/samn"
)

func TestOptimizeLoadedForNeo2FromFunscript(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "imported.funscript")
	body := map[string]any{
		"actions": []map[string]any{
			{"at": 0, "pos": 0},
			{"at": 100, "pos": 20},
			{"at": 4000, "pos": 80},
			{"at": 4100, "pos": 100},
		},
		"metadata": map[string]any{"creator": "other-tool"},
	}
	raw, _ := json.Marshal(body)
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}

	a := NewApp()
	if _, err := a.LoadFunscript(path); err != nil {
		t.Fatal(err)
	}

	res, err := a.OptimizeLoadedForNeo2(true)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Baked || !res.ContactOn || !samn.IsSamnPath(res.Path) {
		t.Fatalf("result %+v", res)
	}
	doc, err := samn.Load(res.Path)
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.General) < 2 {
		t.Fatalf("general empty")
	}
	if len(doc.Vibration) < 2 || len(doc.Suction) < 2 {
		t.Fatalf("neo axes missing vib=%d suc=%d", len(doc.Vibration), len(doc.Suction))
	}
	if !doc.Recipe.ContactVibration {
		t.Fatal("contact recipe expected")
	}
}

func TestOptimizeLoadedForGeneralKeepsStrokeOnly(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "imported.funscript")
	body := map[string]any{"actions": []map[string]any{
		{"at": 0, "pos": 0}, {"at": 100, "pos": 20}, {"at": 4000, "pos": 80}, {"at": 4100, "pos": 100},
	}}
	raw, _ := json.Marshal(body)
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	a := NewApp()
	if _, err := a.LoadFunscript(path); err != nil {
		t.Fatal(err)
	}
	res, err := a.OptimizeLoadedFor(OptimizeTargetGeneral)
	if err != nil {
		t.Fatal(err)
	}
	if res.Baked || res.ContactOn || res.HasNeoAxes {
		t.Fatalf("general must not add the Neo 2 layer: %+v", res)
	}
	if res.Path == "" || res.Message == "" {
		t.Fatalf("result %+v", res)
	}
}

func TestOptimizeLoadedForRejectsUnknownTarget(t *testing.T) {
	a := NewApp()
	if _, err := a.OptimizeLoadedFor("toaster"); err == nil {
		t.Fatal("unknown target accepted")
	}
}

func TestOptimizeLoadedForNeo2Target(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "imported.funscript")
	raw, _ := json.Marshal(map[string]any{"actions": []map[string]any{{"at": 0, "pos": 0}, {"at": 400, "pos": 90}, {"at": 800, "pos": 10}}})
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	a := NewApp()
	if _, err := a.LoadFunscript(path); err != nil {
		t.Fatal(err)
	}
	res, err := a.OptimizeLoadedFor(OptimizeTargetNeo2)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Baked || !samn.IsSamnPath(res.Path) {
		t.Fatalf("neo2 target %+v", res)
	}
}
