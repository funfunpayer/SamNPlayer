package trackutil

import "testing"

func TestCoastBridgesBriefLoss(t *testing.T) {
	var c Coast
	c.ObserveOK(10, 20, 40, 40)
	c.ObserveOK(12, 22, 40, 40) // vx=2, vy=2
	x, y, w, h, ok := c.OnLost()
	if !ok {
		t.Fatal("expected include during coast")
	}
	if w != 40 || h != 40 {
		t.Fatalf("size %dx%d", w, h)
	}
	if x != 14 || y != 24 {
		t.Fatalf("coasted to %d,%d want 14,24", x, y)
	}
}

func TestCoastExhausts(t *testing.T) {
	c := Coast{MaxFrames: 2}
	c.ObserveOK(0, 0, 10, 10)
	c.ObserveOK(1, 0, 10, 10)
	if _, _, _, _, ok := c.OnLost(); !ok {
		t.Fatal("frame 1 should coast")
	}
	if _, _, _, _, ok := c.OnLost(); !ok {
		t.Fatal("frame 2 should coast")
	}
	if _, _, _, _, ok := c.OnLost(); ok {
		t.Fatal("frame 3 should exclude")
	}
}

func TestCoastHoldsWithoutVelocity(t *testing.T) {
	var c Coast
	c.ObserveOK(5, 5, 10, 10) // one sample — hold box, do not invent motion
	x, y, _, _, ok := c.OnLost()
	if !ok {
		t.Fatal("should hold last box")
	}
	if x != 5 || y != 5 {
		t.Fatalf("held %d,%d want 5,5", x, y)
	}
}

func TestCoastNoPosition(t *testing.T) {
	var c Coast
	if _, _, _, _, ok := c.OnLost(); ok {
		t.Fatal("empty coast must not include")
	}
}
