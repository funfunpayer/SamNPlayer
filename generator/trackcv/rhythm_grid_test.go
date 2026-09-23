package trackcv

import (
	"math"
	"math/rand"
	"testing"
)

// synthClip builds per-frame cell flow for a 16x9 grid over a 1280x720 frame:
// cell `target` moves with the stroke (velocity of sin at hz, times
// targetSign), every other cell carries small noise, and `distractor` (if
// >=0) oscillates stronger than the target but a quarter period out of phase.
func synthClip(frames int, fps, hz float64, target, distractor int, targetSign float64) (cellV [][]float32, stroke []float64) {
	rng := rand.New(rand.NewSource(1))
	cells := 16 * 9
	cellV = make([][]float32, frames)
	stroke = make([]float64, frames)
	for i := range cellV {
		t := float64(i) / fps
		stroke[i] = 40 * math.Sin(2*math.Pi*hz*t)
		cellV[i] = make([]float32, cells)
		if i == 0 {
			continue
		}
		dv := stroke[i] - stroke[i-1]
		for c := range cellV[i] {
			cellV[i][c] = float32(rng.NormFloat64() * 0.6)
		}
		cellV[i][target] += float32(targetSign * dv)
		if distractor >= 0 {
			cellV[i][distractor] += float32(3 * 40 * (math.Cos(2*math.Pi*hz*t) - math.Cos(2*math.Pi*hz*(t-1/fps))))
		}
	}
	return cellV, stroke
}

func cellCenter(c int) (float64, float64) {
	return (float64(c%16) + 0.5) * 80, (float64(c/16) + 0.5) * 80
}

// A drifting box whose own motion only weakly follows the stroke: the grid
// must recover the stroke from the rhythmic cell near the box.
func TestRhythmGridRecoversStrokeDespiteDrift(t *testing.T) {
	const fps, hz, frames = 24.0, 1.1, 24 * 60
	target := 5*16 + 8
	// Far-away (> 3 cells) stronger rhythm must be ignored: it's another body part.
	distractor := 1*16 + 1
	cellV, stroke := synthClip(frames, fps, hz, target, distractor, -1)

	tx, ty := cellCenter(target)
	rng := rand.New(rand.NewSource(2))
	cx, cy, anchor := make([]float64, frames), make([]float64, frames), make([]float64, frames)
	for i := range anchor {
		frac := float64(i) / frames
		drift := 150 * frac // box walks ~2 cells off target
		cx[i], cy[i] = tx-drift, ty
		// The box's own motion follows the stroke less the further it
		// drifts - the real failure mode - plus jitter and baseline drift.
		anchor[i] = ty + drift + 0.3*(1-frac)*stroke[i] + 4*rng.NormFloat64()
	}

	got := rhythmGridPositions(cellV, 16, 9, 1280, 720, cx, cy, anchor, fps)
	k := int(2 * fps)
	rGrid := pearson(subtractRollingMean(got, k), stroke)
	rCSRT := pearson(subtractRollingMean(anchor, k), stroke)
	if rGrid < 0.9 {
		t.Errorf("grid curve r=%.3f vs stroke, want >= 0.9 (tracker alone %.3f)", rGrid, rCSRT)
	}
	if rGrid <= rCSRT {
		t.Errorf("grid r=%.3f not better than tracker r=%.3f", rGrid, rCSRT)
	}
}

// No rhythm anywhere (static scene): output falls back to the tracker.
func TestRhythmGridFallsBackToTracker(t *testing.T) {
	const fps, frames = 24.0, 24 * 20
	cellV := make([][]float32, frames)
	cx, cy, anchor := make([]float64, frames), make([]float64, frames), make([]float64, frames)
	for i := range cellV {
		cellV[i] = make([]float32, 16*9)
		cx[i], cy[i] = 640, 360
		anchor[i] = 360 + 10*math.Sin(float64(i)/5)
	}
	got := rhythmGridPositions(cellV, 16, 9, 1280, 720, cx, cy, anchor, fps)
	for i := range got {
		if math.Abs(got[i]-anchor[i]) > 1e-9 {
			t.Fatalf("frame %d: got %v, want tracker %v", i, got[i], anchor[i])
		}
	}
}

func TestRhythmGridBadInputReturnsTracker(t *testing.T) {
	anchor := []float64{1, 2, 3}
	got := rhythmGridPositions(make([][]float32, 2), 16, 9, 1280, 720, anchor, anchor, anchor, 24)
	if len(got) != 3 || got[2] != 3 {
		t.Fatalf("got %v, want copy of tracker positions", got)
	}
	got[0] = 99
	if anchor[0] != 1 {
		t.Fatal("result aliases the tracker slice")
	}
}

func TestRhythmGridRows(t *testing.T) {
	for _, c := range []struct{ w, h, want int }{{1280, 720, 9}, {1920, 1080, 9}, {1080, 1920, 28}, {640, 480, 12}, {0, 0, 9}} {
		if got := rhythmGridRows(c.w, c.h); got != c.want {
			t.Errorf("%dx%d: got %d rows, want %d", c.w, c.h, got, c.want)
		}
	}
}
