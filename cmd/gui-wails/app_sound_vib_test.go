package main

import (
	"testing"

	"github.com/funfunpayer/SamNPlayer/funscript"
)

func TestEnvelopeToVibrationActionsThins(t *testing.T) {
	// Constant 0.5 loudness, 50 ms steps, 10 s → should thin heavily.
	env := make([]float64, 200) // 10 s
	for i := range env {
		env[i] = 0.5
	}
	acts := envelopeToVibrationActions(env, 50)
	if len(acts) < 2 {
		t.Fatalf("expected at least start+end, got %d", len(acts))
	}
	if acts[0].Pos != 50 || acts[len(acts)-1].Pos != 50 {
		t.Errorf("pos: %d … %d", acts[0].Pos, acts[len(acts)-1].Pos)
	}
	// Thinning should keep it well under one point per 250 ms window.
	if len(acts) > 10 {
		t.Errorf("expected heavy thinning on constant signal, got %d points", len(acts))
	}
}

func TestMaxVibrationActionsTakesHigher(t *testing.T) {
	a := []funscript.Action{{At: 0, Pos: 20}, {At: 1000, Pos: 40}}
	b := []funscript.Action{{At: 0, Pos: 60}, {At: 1000, Pos: 10}}
	out := maxVibrationActions(a, b)
	if len(out) != 2 {
		t.Fatalf("len %d", len(out))
	}
	if out[0].Pos != 60 || out[1].Pos != 40 {
		t.Errorf("got %+v", out)
	}
}
