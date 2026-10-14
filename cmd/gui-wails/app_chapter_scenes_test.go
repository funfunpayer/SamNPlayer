package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/funfunpayer/SamNPlayer/funscript"
)

// Two chapters; the scene detection says blowjob in the first and titjob in
// the second → Oral / Breast suggested; saving gives each chapter its feel.
func TestChapterScenesSuggestAndSave(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "clip.funscript")
	var acts []map[string]any
	for i := 0; i <= 40; i++ {
		pos := 10
		if i%2 == 1 {
			pos = 90
		}
		acts = append(acts, map[string]any{"at": i * 500, "pos": pos})
	}
	raw, _ := json.Marshal(map[string]any{"actions": acts})
	if err := os.WriteFile(script, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	video := filepath.Join(dir, "clip.mp4")
	if err := os.WriteFile(video, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	box := map[string]any{"x": 10, "y": 10, "w": 50, "h": 50}
	scene, _ := json.Marshal(map[string]any{"version": 1, "width": 640, "height": 360, "proposals": []map[string]any{
		{"t_ms": 2000, "start_ms": 0, "end_ms": 9000, "scene_type": "blowjob", "confidence": 0.9, "primary": box},
		{"t_ms": 14000, "start_ms": 11000, "end_ms": 20000, "scene_type": "titjob", "confidence": 0.8, "primary": box},
	}})
	if err := os.WriteFile(filepath.Join(dir, "clip.scene.json"), scene, 0o644); err != nil {
		t.Fatal(err)
	}
	a := NewApp()
	if _, err := a.LoadFunscript(script); err != nil {
		t.Fatal(err)
	}
	a.stateMu.Lock()
	a.videoPath = video
	a.stateMu.Unlock()
	if err := a.SaveScriptChapterMarks([]funscript.ChapterMark{{Name: "A", StartTime: 0}, {Name: "B", StartTime: 10000}}); err != nil {
		t.Fatal(err)
	}
	got, err := a.SuggestChapterScenes()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("chapters %+v", got)
	}
	if got[0].Suggested != funscript.SceneFeelOral || got[0].Source != "scene detection" || got[0].Confidence <= 0.5 {
		t.Fatalf("chapter A %+v", got[0])
	}
	if got[1].Suggested != funscript.SceneFeelBreast || got[1].EndMs != 20000 {
		t.Fatalf("chapter B %+v", got[1])
	}
	if got[0].HasFeel {
		t.Fatal("nothing applied before saving")
	}
	ranges := []funscript.SceneFeelRange{
		{StartMs: got[0].StartMs, EndMs: got[0].EndMs, Feel: got[0].Suggested},
		{StartMs: got[1].StartMs, EndMs: got[1].EndMs, Feel: got[1].Suggested},
	}
	if err := a.SaveChapterScenes(ranges); err != nil {
		t.Fatal(err)
	}
	got, _ = a.SuggestChapterScenes()
	if !got[0].HasFeel || got[0].Feel != funscript.SceneFeelOral || got[1].Feel != funscript.SceneFeelBreast {
		t.Fatalf("saved %+v", got)
	}
	// The mapper uses the chapter feel: oral keeps a seal at the shallow point.
	opts := funscript.MapOptionsFromScript(a.loadedScript())
	frames := a.loadedScript().ToIntensityCurve(opts)
	minSuc := 1.0
	for _, f := range frames {
		if f.At < 9000 && f.Suction < minSuc {
			minSuc = f.Suction
		}
	}
	if minSuc < 0.25 {
		t.Fatalf("oral chapter should keep a seal, min suction %v", minSuc)
	}
	if err := a.SaveChapterScenes([]funscript.SceneFeelRange{{StartMs: 0, EndMs: 10, Feel: "dance"}}); err == nil {
		t.Fatal("unknown feel accepted")
	}
}

// Without scene detection the contact mark suggests a feel for every chapter.
func TestChapterScenesFromContactMark(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "c.funscript")
	raw, _ := json.Marshal(map[string]any{
		"actions":  []map[string]any{{"at": 0, "pos": 10}, {"at": 500, "pos": 90}, {"at": 1000, "pos": 10}},
		"metadata": map[string]any{"contact_marks": map[string]any{"primary": map[string]any{"x": 1, "y": 1, "w": 5, "h": 5, "class": "mouth"}, "drive_stroke": false}},
	})
	if err := os.WriteFile(script, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	a := NewApp()
	if _, err := a.LoadFunscript(script); err != nil {
		t.Fatal(err)
	}
	if err := a.SaveScriptChapterMarks([]funscript.ChapterMark{{Name: "All", StartTime: 0}}); err != nil {
		t.Fatal(err)
	}
	got, err := a.SuggestChapterScenes()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Suggested != funscript.SceneFeelOral || got[0].Source != "contact mark" {
		t.Fatalf("%+v", got)
	}
}
