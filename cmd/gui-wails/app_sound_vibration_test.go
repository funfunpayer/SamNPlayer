package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func soundVibApp(t *testing.T, env []float64) *App {
	t.Helper()
	dir := t.TempDir()
	script := filepath.Join(dir, "clip.funscript")
	var acts []map[string]any
	for i := 0; i <= 20; i++ {
		pos := 10
		if i%2 == 1 {
			pos = 90
		}
		acts = append(acts, map[string]any{"at": i * 250, "pos": pos})
	}
	raw, _ := json.Marshal(map[string]any{"actions": acts})
	if err := os.WriteFile(script, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	video := filepath.Join(dir, "clip.mp4")
	if err := os.WriteFile(video, []byte("not really a video"), 0o644); err != nil {
		t.Fatal(err)
	}
	old := audioVibrationEnvelopeFn
	audioVibrationEnvelopeFn = func(context.Context, string, int64) ([]float64, error) { return env, nil }
	t.Cleanup(func() { audioVibrationEnvelopeFn = old })
	a := NewApp()
	if _, err := a.LoadFunscript(script); err != nil {
		t.Fatal(err)
	}
	a.stateMu.Lock()
	a.videoPath = video
	a.stateMu.Unlock()
	return a
}

// A loud first second and silence after: the written vibration follows it.
func TestBakeSoundVibrationFollowsLoudness(t *testing.T) {
	env := make([]float64, 100) // 5 s at 50 ms
	for i := 0; i < 20; i++ {
		env[i] = 1
	}
	a := soundVibApp(t, env)
	res, err := a.BakeSoundVibration(SoundVibSoundOnly)
	if err != nil {
		t.Fatal(err)
	}
	if res.Points == 0 || res.Message == "" {
		t.Fatalf("result %+v", res)
	}
	axes := a.loadedScript().Metadata.SamnAxes
	if axes == nil || len(axes.Vibration) == 0 || len(axes.Suction) == 0 {
		t.Fatalf("lanes not written: %+v", axes)
	}
	at := func(ms int64) int {
		best := axes.Vibration[0]
		for _, p := range axes.Vibration {
			if p.At <= ms {
				best = p
			}
		}
		return best.Pos
	}
	if at(500) < 90 || at(3000) > 10 {
		t.Fatalf("vibration does not follow the sound: 0.5 s=%d, 3 s=%d", at(500), at(3000))
	}
}

// Sound + contact keeps the script's own vibration where the sound is quiet.
func TestBakeSoundVibrationWithContactKeepsScriptVibration(t *testing.T) {
	env := make([]float64, 100) // silence
	a := soundVibApp(t, env)
	if _, err := a.BakeSoundVibration(SoundVibSoundContact); err != nil {
		t.Fatal(err)
	}
	max := 0
	for _, p := range a.loadedScript().Metadata.SamnAxes.Vibration {
		if p.Pos > max {
			max = p.Pos
		}
	}
	if max == 0 {
		t.Fatal("with contact the script's vibration should stay when the sound is silent")
	}
}

func TestBakeSoundVibrationRejects(t *testing.T) {
	a := soundVibApp(t, []float64{0.5})
	if _, err := a.BakeSoundVibration("noise"); err == nil {
		t.Fatal("unknown mode accepted")
	}
	b := NewApp()
	if _, err := b.BakeSoundVibration(SoundVibSoundOnly); err == nil {
		t.Fatal("no script accepted")
	}
}
