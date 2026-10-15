package funscript

import (
	"math"
	"testing"
)

var zigzag = []Action{{0, 0}, {500, 100}, {1000, 0}, {1500, 100}, {2000, 0}}

func TestStrokeFilterOffChangesNothing(t *testing.T) {
	out := ApplyStrokeFilter(zigzag, StrokeFilter{})
	if len(out) != len(zigzag) {
		t.Fatalf("len %d", len(out))
	}
	out[0].Pos = 55
	if zigzag[0].Pos != 0 {
		t.Fatal("input modified")
	}
}

func TestStrokeFilterSmoothKeepsPointsAndStaysInRange(t *testing.T) {
	for _, mode := range []string{SmoothPCHIP, SmoothMakima} {
		out := ApplyStrokeFilter(zigzag, StrokeFilter{Smooth: mode, StepMs: 50})
		if len(out) <= len(zigzag) {
			t.Fatalf("%s: no points added (%d)", mode, len(out))
		}
		orig := map[int64]int{}
		for _, a := range zigzag {
			orig[a.At] = a.Pos
		}
		for i, a := range out {
			if a.Pos < 0 || a.Pos > 100 {
				t.Fatalf("%s: out of range %+v", mode, a)
			}
			if i > 0 && a.At <= out[i-1].At {
				t.Fatalf("%s: time not increasing at %d", mode, i)
			}
			if p, ok := orig[a.At]; ok && p != a.Pos {
				t.Fatalf("%s: original point moved %+v", mode, a)
			}
		}
		// Rounded: the curve leaves the top at 500 ms flat, so 100 ms later it
		// is still above the straight line's 80.
		var q Action
		for _, a := range out {
			if a.At == 600 {
				q = a
			}
		}
		if q.Pos <= 80 {
			t.Fatalf("%s: not rounded after the top: %+v", mode, q)
		}
	}
}

func TestPCHIPNoOvershoot(t *testing.T) {
	// Monotone data: pchip must stay monotone between the points.
	a := []Action{{0, 0}, {300, 10}, {600, 90}, {900, 100}}
	out := ApplyStrokeFilter(a, StrokeFilter{Smooth: SmoothPCHIP, StepMs: 25})
	for i := 1; i < len(out); i++ {
		if out[i].Pos < out[i-1].Pos {
			t.Fatalf("not monotone at %d: %+v -> %+v", i, out[i-1], out[i])
		}
	}
}

func TestStrokeFilterSpeedLimit(t *testing.T) {
	a := []Action{{0, 0}, {100, 100}, {1000, 100}}
	out := ApplyStrokeFilter(a, StrokeFilter{MaxSpeed: 400})
	if out[1].Pos != 40 {
		t.Fatalf("100 ms at 400/s = 40 units, got %+v", out[1])
	}
	if out[2].Pos != 100 {
		t.Fatalf("then it catches up: %+v", out[2])
	}
	for i := 1; i < len(out); i++ {
		dt := float64(out[i].At-out[i-1].At) / 1000
		if math.Abs(float64(out[i].Pos-out[i-1].Pos)) > 400*dt+1e-9 {
			t.Fatalf("too fast at %d", i)
		}
	}
}

func TestStrokeFilterIdleFill(t *testing.T) {
	a := []Action{{0, 50}, {10000, 50}, {10500, 0}}
	out := ApplyStrokeFilter(a, StrokeFilter{IdleFillMs: 4000})
	if len(out) < 8 {
		t.Fatalf("pause not filled: %+v", out)
	}
	for _, p := range out[1 : len(out)-2] {
		if p.At <= 0 || p.At >= 10000 || p.Pos < 50-idleWaveAmp || p.Pos > 50+idleWaveAmp {
			t.Fatalf("wave outside the pause or too strong: %+v", p)
		}
	}
	// A short gap stays as it is.
	if got := ApplyStrokeFilter([]Action{{0, 0}, {1000, 100}}, StrokeFilter{IdleFillMs: 4000}); len(got) != 2 {
		t.Fatalf("short gap filled: %+v", got)
	}
}

func TestNormalizeSmooth(t *testing.T) {
	if NormalizeSmooth("pchip") != SmoothPCHIP || NormalizeSmooth("makima") != SmoothMakima || NormalizeSmooth("cubic") != SmoothOff {
		t.Fatal("normalize")
	}
}
