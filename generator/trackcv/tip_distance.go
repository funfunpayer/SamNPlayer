//go:build cgo && opencv

package trackcv

import "math"

// TipPartnerDistance measures how close the tip ROI is to a contact partner.
//
// Uses the point on the tip box nearest the partner center (not tip center).
// Marking the whole penis still reads contact from the end toward the partner
// (typically the glans), so nipple/mouth contact vibration and Tf/Tj suction
// fire when the tip end approaches — not when the shaft center does.
func TipPartnerDistance(tip, partner Rect) float64 {
	pcx := float64(partner.X) + float64(partner.W)/2
	pcy := float64(partner.Y) + float64(partner.H)/2
	tx := clampFloat(pcx, float64(tip.X), float64(tip.X+tip.W))
	ty := clampFloat(pcy, float64(tip.Y), float64(tip.Y+tip.H))
	return math.Hypot(pcx-tx, pcy-ty)
}

func clampFloat(v, lo, hi float64) float64 {
	if lo > hi {
		lo, hi = hi, lo
	}
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
