package player

import (
	"testing"
	"time"
)

func TestClampSpeed(t *testing.T) {
	for in, want := range map[float64]float64{0: 1, -1: 1, 0.2: MinSpeed, 1.5: 1.5, 9: MaxSpeed} {
		if got := ClampSpeed(in); got != want {
			t.Fatalf("ClampSpeed(%v) = %v, want %v", in, got, want)
		}
	}
}

func TestSpeedScalesClockAndReanchors(t *testing.T) {
	p := New(nil)
	t0 := time.Now()
	p.startClock(t0)
	p.SetSpeed(2)
	// After re-anchoring at ~t0, 1000 ms of script is due ~500 ms later.
	d := p.realTimeFor(1000).Sub(t0)
	if d < 480*time.Millisecond || d > 540*time.Millisecond {
		t.Fatalf("2× speed: 1000 ms script due after %v, want ~500ms", d)
	}
	p.SetSpeed(0.5)
	d = p.realTimeFor(1000).Sub(t0)
	if d < 1950*time.Millisecond || d > 2100*time.Millisecond {
		t.Fatalf("0.5× speed: 1000 ms script due after %v, want ~2000ms", d)
	}
	if p.Speed() != 0.5 {
		t.Fatalf("Speed() = %v", p.Speed())
	}
}
