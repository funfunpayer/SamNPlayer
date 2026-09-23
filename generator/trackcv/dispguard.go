//go:build cgo && opencv

package trackcv

import "sort"

// dispGuard flags an implausible single-frame CSRT jump: OpenCV's CSRT
// tracker can report Update() as successful ("ok=true") while the
// returned box has actually snapped onto a completely different,
// unrelated location - not a scene cut, not a reported loss, just the
// tracker's own internal correlation filter locking onto a different,
// higher-scoring patch elsewhere in the frame. Confirmed on a real clip
// (docs/AGENT_COORD.md, 23 Sep "CSRT long-clip drift"): two single-frame
// jumps of 107px and 295px, both with SceneCutDetection reporting no cut
// and no other lost frame anywhere nearby, while every genuine motion
// across that same ~6700-frame clip stayed under ~50px in a single
// ~40ms frame.
//
// A fixed pixel threshold would need separate tuning per clip/ROI size -
// too low and it clips real fast motion, too high and it misses a slow
// clip's teleport. Comparing each frame's displacement against a short
// rolling median of its own recent (accepted) history adapts
// automatically instead: real motion raises the local median together
// with the frame that's part of it, so the ratio test stays sane; an
// isolated teleport frame stands far outside whatever the surrounding
// motion actually was, regardless of the clip's general pace.
type dispGuard struct {
	recent []float64
}

const (
	// dispGuardWindow: how many recent accepted per-frame displacements
	// feed the rolling median.
	dispGuardWindow = 20
	// dispGuardRatio: a displacement must exceed this multiple of the
	// recent median to be flagged.
	dispGuardRatio = 8.0
	// dispGuardFloorPx: never flag a displacement below this many px,
	// regardless of ratio - protects near-static passages where the
	// median sits close to 0 (a tiny ratio there would flag ordinary
	// jitter). Comfortably above the largest genuine single-frame motion
	// measured on the real clip above (~50px) and comfortably below the
	// smallest confirmed violation (107px).
	dispGuardFloorPx = 60.0
)

// implausible reports whether d is too large a jump to be genuine tracker
// motion, given the recent accepted displacement history.
func (g *dispGuard) implausible(d float64) bool {
	if d < dispGuardFloorPx || len(g.recent) == 0 {
		return false
	}
	return d > dispGuardRatio*medianOf(g.recent)
}

// accept records a legitimate displacement into the rolling window. Call
// only for a displacement implausible did NOT flag, so a rejected jump
// never skews the baseline it was judged against.
func (g *dispGuard) accept(d float64) {
	g.recent = append(g.recent, d)
	if len(g.recent) > dispGuardWindow {
		g.recent = g.recent[1:]
	}
}

func medianOf(v []float64) float64 {
	s := append([]float64(nil), v...)
	sort.Float64s(s)
	n := len(s)
	if n%2 == 1 {
		return s[n/2]
	}
	return (s[n/2-1] + s[n/2]) / 2
}
