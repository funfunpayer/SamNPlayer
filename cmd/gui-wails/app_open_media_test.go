package main

import (
	"os"
	"path/filepath"
	"testing"
)

func touch(t *testing.T, p string) {
	t.Helper()
	if err := os.WriteFile(p, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestFindScriptForVideo(t *testing.T) {
	dir := t.TempDir()
	video := filepath.Join(dir, "Clip One.mp4")
	touch(t, video)

	if m := findScriptForVideo(video); m.Found {
		t.Fatalf("no script yet, got %+v", m)
	}

	// Axis companions do not count as the script.
	touch(t, filepath.Join(dir, "Clip One.surge.funscript"))
	if m := findScriptForVideo(video); m.Found {
		t.Fatalf("axis companion taken as script: %+v", m)
	}

	touch(t, filepath.Join(dir, "clip one.funscript"))
	m := findScriptForVideo(video)
	if !m.Found || filepath.Base(m.ScriptPath) != "clip one.funscript" {
		t.Fatalf("case-insensitive funscript not found: %+v", m)
	}

	touch(t, filepath.Join(dir, "Clip One.funscript"))
	if m := findScriptForVideo(video); filepath.Base(m.ScriptPath) != "Clip One.funscript" {
		t.Fatalf("exact case should win: %+v", m)
	}

	touch(t, filepath.Join(dir, "Clip One.samn"))
	if m := findScriptForVideo(video); filepath.Base(m.ScriptPath) != "Clip One.samn" {
		t.Fatalf("Emotion Script should win: %+v", m)
	}

	if m := findScriptForVideo(""); m.Found {
		t.Fatal("empty path matched")
	}
}
