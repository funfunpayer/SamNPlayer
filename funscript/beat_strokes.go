package funscript

import (
	"math"
	"sort"
)

// BeatStrokes turns beat times (a Cock Hero / Fap Hero beat bar, or any
// beat list) into a stroke script: down to Bottom on every beat, up to Top
// halfway to the next one. Fast beats get shorter strokes so no move is
// faster than MaxSpeed; a long pause rises once and holds until shortly
// before the next beat.
type BeatStrokeOptions struct {
	Top, Bottom int     // 0–100; defaults 90 / 10
	MaxSpeed    float64 // units per second; default 500
	HoldAfterMs float64 // gaps longer than this hold instead of a slow stroke; default 2000
}

const beatLeadMs = 400 // rise / start before a beat after a pause

func BeatStrokes(beatsMs []float64, o BeatStrokeOptions) []Action {
	if o.Top == 0 && o.Bottom == 0 {
		o.Top, o.Bottom = 90, 10
	}
	o.Top, o.Bottom = clampPos(o.Top), clampPos(o.Bottom)
	if o.Top < o.Bottom {
		o.Top, o.Bottom = o.Bottom, o.Top
	}
	if o.MaxSpeed <= 0 {
		o.MaxSpeed = 500
	}
	if o.HoldAfterMs <= 0 {
		o.HoldAfterMs = 2000
	}
	beats := append([]float64(nil), beatsMs...)
	sort.Float64s(beats)
	var out []Action
	add := func(at float64, pos int) {
		t := int64(math.Round(at))
		if t < 0 {
			t = 0
		}
		if n := len(out); n > 0 && t <= out[n-1].At {
			return
		}
		out = append(out, Action{At: t, Pos: pos})
	}
	// up returns the height reachable from Bottom in ms at MaxSpeed.
	up := func(ms float64) int {
		h := float64(o.Bottom) + o.MaxSpeed*ms/1000
		if h > float64(o.Top) {
			h = float64(o.Top)
		}
		return int(math.Floor(h))
	}
	for i, b := range beats {
		if lead := math.Min(beatLeadMs, b); i == 0 && lead > 0 {
			add(b-lead, up(lead))
		}
		add(b, o.Bottom)
		if i+1 >= len(beats) {
			break
		}
		gap := beats[i+1] - b
		if gap > o.HoldAfterMs {
			// Rise, hold, then come down on the next beat.
			h := up(beatLeadMs)
			add(b+beatLeadMs, h)
			add(beats[i+1]-beatLeadMs, h)
			continue
		}
		add(b+gap/2, up(gap/2))
	}
	return out
}
