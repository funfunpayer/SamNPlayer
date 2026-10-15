package main

import (
	"os"
	"path/filepath"
	"testing"
)

func touch(t *testing.T, p string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestFindScriptForVideo(t *testing.T) {
	dir := t.TempDir()
	video := filepath.Join(dir, "Clip One.mp4")
	touch(t, video)

	if m := findScriptForVideo(video, nil); m.Found {
		t.Fatalf("no script yet, got %+v", m)
	}

	// Axis companions do not count as the script.
	touch(t, filepath.Join(dir, "Clip One.surge.funscript"))
	if m := findScriptForVideo(video, nil); m.Found {
		t.Fatalf("axis companion taken as script: %+v", m)
	}

	touch(t, filepath.Join(dir, "clip one.funscript"))
	m := findScriptForVideo(video, nil)
	if !m.Found || filepath.Base(m.ScriptPath) != "clip one.funscript" {
		t.Fatalf("case-insensitive funscript not found: %+v", m)
	}

	touch(t, filepath.Join(dir, "Clip One.funscript"))
	if m := findScriptForVideo(video, nil); filepath.Base(m.ScriptPath) != "Clip One.funscript" {
		t.Fatalf("exact case should win: %+v", m)
	}

	touch(t, filepath.Join(dir, "Clip One.samn"))
	if m := findScriptForVideo(video, nil); filepath.Base(m.ScriptPath) != "Clip One.samn" {
		t.Fatalf("Emotion Script should win: %+v", m)
	}

	if m := findScriptForVideo("", nil); m.Found {
		t.Fatal("empty path matched")
	}
}

func TestFindScriptForVideoFoldersNamesVariants(t *testing.T) {
	videos := t.TempDir()
	scripts := t.TempDir()
	video := filepath.Join(videos, "Some_Studio - Clip.Title (2019) [1080p].mp4")
	touch(t, video)

	// Only in the script folder, written differently, one level down.
	touch(t, filepath.Join(scripts, "authorA", "some studio clip title.funscript"))
	m := findScriptForVideo(video, []string{scripts})
	if !m.Found || filepath.Base(m.ScriptPath) != "some studio clip title.funscript" || m.Variants[0].Beside {
		t.Fatalf("tolerant name in a script folder: %+v", m)
	}

	// Variants: Soft, another author; not a sequel, not an axis file.
	touch(t, filepath.Join(scripts, "Some Studio - Clip Title (Soft).funscript"))
	touch(t, filepath.Join(scripts, "authorB", "Some Studio - Clip Title - by Bee.funscript"))
	touch(t, filepath.Join(scripts, "Some Studio - Clip Title 2.funscript"))
	touch(t, filepath.Join(scripts, "Some Studio - Clip Title.twist.funscript"))
	m = findScriptForVideo(video, []string{scripts})
	labels := map[string]bool{}
	for _, v := range m.Variants {
		labels[v.Label] = true
	}
	if len(m.Variants) != 3 || !labels[""] || !labels["soft"] || !labels["by bee"] {
		t.Fatalf("variants: %+v", m.Variants)
	}
	if m.Variants[0].Label != "" {
		t.Fatalf("the script with the video's own name comes first: %+v", m.Variants)
	}

	// A script beside the video beats the folder.
	touch(t, filepath.Join(videos, "Some_Studio - Clip.Title (2019) [1080p].funscript"))
	m = findScriptForVideo(video, []string{scripts})
	if !m.Variants[0].Beside || filepath.Dir(m.ScriptPath) != videos {
		t.Fatalf("beside the video first: %+v", m.Variants)
	}

	// A .samn and its .funscript count once.
	touch(t, filepath.Join(videos, "Some_Studio - Clip.Title (2019) [1080p].samn"))
	m = findScriptForVideo(video, []string{scripts})
	if filepath.Ext(m.ScriptPath) != ".samn" || len(m.Variants) != 4 {
		t.Fatalf("one entry per name, .samn first: %+v", m.Variants)
	}
}

func TestVariantLabel(t *testing.T) {
	cases := []struct {
		script, video, label string
		ok                   bool
	}{
		{"Clip", "Clip", "", true},
		{"clip [4K]", "Clip (2020)", "", true},
		{"Clip (Hard)", "Clip", "hard", true},
		{"Clip 10", "Clip 1", "", false},
		{"Clip 2", "Clip", "", false},
		{"Clip.roll", "Clip", "", false},
		{"Other", "Clip", "", false},
	}
	for _, c := range cases {
		label, ok := variantLabel(c.script, c.video)
		if ok != c.ok || label != c.label {
			t.Errorf("variantLabel(%q, %q) = %q,%v want %q,%v", c.script, c.video, label, ok, c.label, c.ok)
		}
	}
}

func TestSplitFolders(t *testing.T) {
	got := splitFolders(" /a \n\n/b\n/a\n")
	if len(got) != 2 || got[0] != "/a" || got[1] != "/b" {
		t.Fatalf("%v", got)
	}
}
