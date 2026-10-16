package funscript

import (
	"math"
	"testing"
)

func TestBeatStrokesDownOnEveryBeat(t *testing.T) {
	beats := []float64{1000, 1500, 2000, 2500}
	out := BeatStrokes(beats, BeatStrokeOptions{})
	bottoms := map[int64]bool{}
	for _, a := range out {
		if a.Pos == 10 {
			bottoms[a.At] = true
		}
	}
	for _, b := range beats {
		if !bottoms[int64(b)] {
			t.Fatalf("no bottom at beat %v: %+v", b, out)
		}
	}
	// 250 ms up at 500/s = 125 units, capped at Top 90.
	for _, a := range out {
		if a.At == 1250 && a.Pos != 90 {
			t.Fatalf("full stroke expected between slow beats: %+v", a)
		}
	}
}

func TestBeatStrokesSpeedLimitAndHold(t *testing.T) {
	// Fast beats (100 ms apart) get short strokes; a 5 s pause holds.
	out := BeatStrokes([]float64{0, 100, 200, 5200}, BeatStrokeOptions{MaxSpeed: 400})
	for i := 1; i < len(out); i++ {
		dt := float64(out[i].At-out[i-1].At) / 1000
		if math.Abs(float64(out[i].Pos-out[i-1].Pos)) > 400*dt+1 {
			t.Fatalf("too fast between %+v and %+v", out[i-1], out[i])
		}
		if out[i].At <= out[i-1].At {
			t.Fatalf("time not increasing: %+v", out)
		}
	}
	held := 0
	for _, a := range out {
		if a.At > 200 && a.At < 5200 {
			held++
		}
	}
	if held != 2 {
		t.Fatalf("pause should rise once and hold (2 points), got %d: %+v", held, out)
	}
	if len(BeatStrokes(nil, BeatStrokeOptions{})) != 0 {
		t.Fatal("no beats, no actions")
	}
}
