package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/funfunpayer/SamNPlayer/funscript"
	"github.com/funfunpayer/SamNPlayer/generator"
	"github.com/funfunpayer/SamNPlayer/videox"
)

func TestCreateBeatBarScript(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	oldDetect, oldProbe := detectBeatBarFn, probeVideoFn
	t.Cleanup(func() { detectBeatBarFn, probeVideoFn = oldDetect, oldProbe })
	probeVideoFn = func(context.Context, string) (videox.Info, error) {
		return videox.Info{Duration: 60 * time.Second}, nil
	}
	found := true
	detectBeatBarFn = func(_ context.Context, _ string, dur float64) (generator.BeatBarResult, error) {
		if dur != 60 {
			t.Fatalf("duration %v", dur)
		}
		if !found {
			return generator.BeatBarResult{}, nil
		}
		return generator.BeatBarResult{Found: true, BeatsMs: []float64{1000, 1500, 2000}, BPM: 120}, nil
	}
	dir := t.TempDir()
	video := filepath.Join(dir, "Hero Round.mp4")
	if err := os.WriteFile(video, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	// The author's own script must stay untouched.
	own := filepath.Join(dir, "Hero Round.funscript")
	if err := os.WriteFile(own, []byte(`{"actions":[{"at":0,"pos":0},{"at":9,"pos":9}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	a := NewApp()
	if _, err := a.CreateBeatBarScript(""); err == nil {
		t.Fatal("no video must be an error")
	}
	res, err := a.CreateBeatBarScript(video)
	if err != nil || !res.Found || res.Beats != 3 || res.BPM != 120 {
		t.Fatalf("got %+v %v", res, err)
	}
	if res.ScriptPath != filepath.Join(dir, "Hero Round (Beat bar).funscript") {
		t.Fatalf("path %q", res.ScriptPath)
	}
	s, err := funscript.Load(res.ScriptPath)
	if err != nil || len(s.Actions) < 3 {
		t.Fatalf("written script: %+v %v", s, err)
	}
	if label, ok := variantLabel("Hero Round (Beat bar)", "Hero Round"); !ok || label != "beat bar" {
		t.Fatalf("not a variant: %q %v", label, ok)
	}
	// A second run keeps the first.
	res2, _ := a.CreateBeatBarScript(video)
	if res2.ScriptPath != filepath.Join(dir, "Hero Round (Beat bar 2).funscript") {
		t.Fatalf("second path %q", res2.ScriptPath)
	}
	if b, _ := os.ReadFile(own); string(b) != `{"actions":[{"at":0,"pos":0},{"at":9,"pos":9}]}` {
		t.Fatal("own script changed")
	}
	found = false
	if res, err := a.CreateBeatBarScript(video); err != nil || res.Found || res.Message == "" {
		t.Fatalf("no bar: %+v %v", res, err)
	}
}
