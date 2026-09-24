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

// Where the box's own motion carries no stroke at all, the tracker's sign is
// a coin toss per chunk; orientation must then come from the curve already
// written, or the stroke flips back and forth.
func TestRhythmGridKeepsOrientationWhereTrackerIsUninformative(t *testing.T) {
	const fps, hz, frames = 24.0, 1.1, 24 * 90
	target := 5*16 + 8
	cellV, stroke := synthClip(frames, fps, hz, target, -1, -1)
	tx, ty := cellCenter(target)
	rng := rand.New(rand.NewSource(3))
	cx, cy, anchor := make([]float64, frames), make([]float64, frames), make([]float64, frames)
	for i := range anchor {
		cx[i], cy[i] = tx, ty
		follow := 0.3
		if i > frames/3 && i < 2*frames/3 {
			follow = 0 // middle third: box sits on something that doesn't stroke
		}
		anchor[i] = ty + follow*stroke[i] + 4*rng.NormFloat64()
	}
	got := rhythmGridPositions(cellV, 16, 9, 1280, 720, cx, cy, anchor, fps)
	k := int(2 * fps)
	if r := pearson(subtractRollingMean(got, k), stroke); r < 0.95 {
		t.Errorf("grid curve r=%.3f vs stroke, want >= 0.95 (orientation flips where the tracker is uninformative)", r)
	}
}

// P1: scoreWindows + chooseAndStitch must produce the same curve as the
// combined rhythmGridPositions path (bit-identical on synthetic data).
func TestRhythmGridSplitBitIdentical(t *testing.T) {
	const fps, hz, frames = 24.0, 1.1, 24 * 60
	target := 5*16 + 8
	cellV, stroke := synthClip(frames, fps, hz, target, -1, -1)
	tx, ty := cellCenter(target)
	rng := rand.New(rand.NewSource(4))
	cx, cy, anchor := make([]float64, frames), make([]float64, frames), make([]float64, frames)
	for i := range anchor {
		cx[i], cy[i] = tx, ty
		anchor[i] = ty + 0.4*stroke[i] + 3*rng.NormFloat64()
	}
	viaWrapper := rhythmGridPositions(cellV, 16, 9, 1280, 720, cx, cy, anchor, fps)
	viaMap, m := rhythmGridPositionsWithMap(cellV, 16, 9, 1280, 720, cx, cy, anchor, fps)
	if len(viaWrapper) != len(viaMap) {
		t.Fatalf("length mismatch: wrapper %d map %d", len(viaWrapper), len(viaMap))
	}
	for i := range viaWrapper {
		if viaWrapper[i] != viaMap[i] {
			t.Fatalf("frame %d: wrapper %v != map path %v (not bit-identical)", i, viaWrapper[i], viaMap[i])
		}
	}
	if m.Cols != 16 || m.Rows != 9 || m.Width != 1280 || m.Height != 720 {
		t.Fatalf("map shape: cols=%d rows=%d %dx%d", m.Cols, m.Rows, m.Width, m.Height)
	}
	if len(m.Windows) == 0 {
		t.Fatal("expected scored windows")
	}
	for _, w := range m.Windows {
		if len(w.Score) != 16*9 {
			t.Fatalf("window score len %d, want %d", len(w.Score), 16*9)
		}
		if w.TempoHz < rhythmTempoLoHz || w.TempoHz > rhythmTempoHiHz {
			t.Errorf("tempo %.3f outside stroke band", w.TempoHz)
		}
		// Full-run path should pick a cell near the target.
		if w.ChosenCell < 0 {
			continue
		}
		if w.SignRule != "tracker" && w.SignRule != "continuity" {
			t.Errorf("unexpected SignRule %q", w.SignRule)
		}
	}
}

func TestNormalizeScores(t *testing.T) {
	s := normalizeScores([]float64{0, 1, 0.5})
	if s[1] != 255 || s[2] != 128 {
		t.Fatalf("got %v, want max=255 mid≈128", s)
	}
	z := normalizeScores([]float64{0, 0, 0})
	for _, v := range z {
		if v != 0 {
			t.Fatalf("zero scores should stay 0, got %v", z)
		}
	}
}

func TestScoreWindowsQuickScanNoBox(t *testing.T) {
	const fps, frames = 24.0, 24 * 30
	target := 5*16 + 8
	cellV, _ := synthClip(frames, fps, 1.0, target, -1, 1)
	m := scoreWindows(cellV, 16, 9, 1280, 720, nil, nil, fps)
	if m.Cols != 16 || m.Rows != 9 {
		t.Fatalf("shape %dx%d", m.Cols, m.Rows)
	}
	if len(m.Windows) == 0 {
		t.Fatal("expected windows")
	}
	for _, w := range m.Windows {
		if w.ChosenCell != -1 {
			t.Errorf("quick-scan ChosenCell should stay -1, got %d", w.ChosenCell)
		}
		if len(w.Score) != 144 {
			t.Fatalf("score len %d", len(w.Score))
		}
	}
}
