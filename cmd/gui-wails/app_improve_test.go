package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/funfunpayer/SamNPlayer/funscript"
)

func TestImproveGeneratedScriptFillGaps(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "clip.funscript")
	body := map[string]any{
		"actions": []map[string]any{
			{"at": 0, "pos": 0},
			{"at": 100, "pos": 10},
			{"at": 5000, "pos": 90},
			{"at": 5100, "pos": 100},
		},
		"metadata": map[string]any{"creator": "test"},
	}
	raw, _ := json.Marshal(body)
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}

	a := &App{}
	res, err := a.ImproveGeneratedScript(ImproveScriptRequest{
		Path:       path,
		FillGaps:   true,
		MaxGapMs:   800,
		AudioCheck: false,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.GapsFilled < 1 || res.PointsAdded < 1 {
		t.Fatalf("expected fill, got %+v", res)
	}
	script, err := funscript.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(script.Actions) <= 4 {
		t.Fatalf("actions not expanded: %d", len(script.Actions))
	}
}

func TestImproveGeneratedScriptTrim(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "clip.funscript")
	body := `{"actions":[{"at":0,"pos":0},{"at":1000,"pos":50},{"at":2000,"pos":100},{"at":3000,"pos":50}],"metadata":{}}`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	a := &App{}
	res, err := a.ImproveGeneratedScript(ImproveScriptRequest{
		Path:     path,
		StartSec: 0.5,
		EndSec:   2.5,
		FillGaps: false,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Trimmed {
		t.Fatalf("expected trim: %+v", res)
	}
	script, err := funscript.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if script.Actions[0].At != 500 || script.Actions[len(script.Actions)-1].At != 2500 {
		t.Fatalf("ends %+v", script.Actions)
	}
}
