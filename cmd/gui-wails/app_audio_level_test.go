package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestAudioLevelEnvelopeSharesDecodeWithAudioVibration(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	a := NewApp()
	if env, err := a.GetAudioLevelEnvelope(); err != nil || len(env) != 0 {
		t.Fatalf("no video: want empty, got %v %v", env, err)
	}
	video := filepath.Join(t.TempDir(), "clip.mp4")
	if err := os.WriteFile(video, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	calls := 0
	stubAudioVibration(t, func(_ context.Context, path string, stepMs int64) ([]float64, error) {
		calls++
		if path != video || stepMs != audioLevelStepMs {
			t.Fatalf("decode %q step %d", path, stepMs)
		}
		return []float64{0.1, 0.9, 0.4}, nil
	})
	a.stateMu.Lock()
	a.setVideoPathLocked(video)
	a.stateMu.Unlock()
	env, err := a.GetAudioLevelEnvelope()
	if err != nil || len(env) != 3 || env[1] != 0.9 {
		t.Fatalf("envelope %v err %v", env, err)
	}
	// Audio vibration at the default 50 ms tick reuses the cached decode.
	if _, err := a.audioVibrationEnvelope(context.Background(), video, 50); err != nil || calls != 1 {
		t.Fatalf("expected one shared decode, calls %d err %v", calls, err)
	}
}
