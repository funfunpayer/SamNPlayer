package simpletrack

import (
	"math"
	"testing"
)

func TestTipPartnerDistance_largePenisUsesNearEdge(t *testing.T) {
	tip := Rect{X: 10, Y: 40, W: 100, H: 20}
	partner := Rect{X: 120, Y: 45, W: 20, H: 10}
	d := tipPartnerDistance(tip, partner)
	if math.Abs(d-20) > 0.01 {
		t.Fatalf("got %v want 20", d)
	}
}

func TestTipPartnerDistance_overlapIsZero(t *testing.T) {
	if d := tipPartnerDistance(Rect{0, 0, 50, 50}, Rect{10, 10, 10, 10}); d != 0 {
		t.Fatalf("got %v want 0", d)
	}
}
