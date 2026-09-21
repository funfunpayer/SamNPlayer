package funscript

import (
	"math"
	"testing"
)

func sineActions(durationMs, periodMs, stepMs int, t0 int64, amplitude, center float64) []Action {
	var actions []Action
	for t := 0; t <= durationMs; t += stepMs {
		pos := center + amplitude*math.Sin(2*math.Pi*float64(t)/float64(periodMs))
		actions = append(actions, Action{At: t0 + int64(t), Pos: int(math.Round(pos))})
	}
	return actions
}

func sineActionsPhase(durationMs, periodMs, stepMs int, t0 int64, amplitude, center float64, phaseMs int) []Action {
	var actions []Action
	for t := 0; t <= durationMs; t += stepMs {
		pos := center + amplitude*math.Sin(2*math.Pi*float64(t+phaseMs)/float64(periodMs))
		actions = append(actions, Action{At: t0 + int64(t), Pos: int(math.Round(pos))})
	}
	return actions
}

func driftingLagActions(durationMs, periodMs, maxDriftMs, stepMs int, amplitude, center float64) []Action {
	var actions []Action
	for t := 0; t <= durationMs; t += stepMs {
		drift := float64(maxDriftMs) * float64(t) / float64(durationMs)
		pos := center + amplitude*math.Sin(2*math.Pi*(float64(t)-drift)/float64(periodMs))
		actions = append(actions, Action{At: int64(t), Pos: int(math.Round(pos))})
	}
	return actions
}

func irregularActions(t0 int64) []Action {
	var actions []Action
	t := 0
	for t < 4000 {
		pos := 50 + 15*math.Sin(2*math.Pi*float64(t)/500)
		actions = append(actions, Action{At: t0 + int64(t), Pos: int(math.Round(pos))})
		t += 40
	}
	for t < 6000 {
		actions = append(actions, Action{At: t0 + int64(t), Pos: 50})
		t += 40
	}
	for t < 12000 {
		pos := 50 + 45*math.Sin(2*math.Pi*float64(t)/2000)
		actions = append(actions, Action{At: t0 + int64(t), Pos: int(math.Round(pos))})
		t += 40
	}
	return actions
}

func TestBestLagCorrelationShiftedSine(t *testing.T) {
	reference := sineActions(20000, 2000, 40, 0, 40, 50)
	shifted := sineActions(20000, 2000, 40, 837, 40, 50)
	result := BestLagCorrelation(reference, shifted, 2000, 50, 100)
	if result == nil {
		t.Fatal("expected a correlation result")
	}
	if absInt(result.LagMs-(-837)) > 50 {
		t.Fatalf("lag_ms=%d, want ~-837 (±50)", result.LagMs)
	}
	if result.R <= 0.95 {
		t.Fatalf("aligned r=%.4f, want >0.95", result.R)
	}
	if result.RZeroLag == nil {
		t.Fatal("expected r_zero_lag")
	}
	if result.R-*result.RZeroLag <= 0.3 {
		t.Fatalf("aligned should beat raw by >0.3; r=%.4f r0=%.4f", result.R, *result.RZeroLag)
	}
	if result.LowConfidence {
		t.Fatal("long overlap should not be low_confidence")
	}
}

func TestBestLagCorrelationInverted(t *testing.T) {
	reference := sineActions(20000, 2000, 40, 0, 40, 50)
	inverted := make([]Action, len(reference))
	for i, a := range reference {
		inverted[i] = Action{At: a.At, Pos: 100 - a.Pos}
	}
	result := BestLagCorrelation(reference, inverted, 200, 50, 100)
	if result == nil {
		t.Fatal("expected a correlation result")
	}
	if result.Orientation != "inverted" {
		t.Fatalf("orientation=%q, want inverted", result.Orientation)
	}
	if result.R <= 0.95 {
		t.Fatalf("r=%.4f, want >0.95", result.R)
	}
}

func TestBestLagCorrelationConstantUndefined(t *testing.T) {
	reference := sineActions(20000, 2000, 40, 0, 40, 50)
	constant := make([]Action, 0, 50)
	for tms := 0; tms < 5000; tms += 100 {
		constant = append(constant, Action{At: int64(tms), Pos: 100})
	}
	result := BestLagCorrelation(reference, constant, 500, 100, 100)
	if result != nil {
		t.Fatalf("constant series must yield nil, got %+v", result)
	}
	if pearson([]float64{1, 1, 1, 1}, []float64{1, 2, 3, 4}) != nil {
		t.Fatal("pearson must return nil on zero variance, not 0.0")
	}
}

func TestBestLagCorrelationIrregular(t *testing.T) {
	ref := irregularActions(0)
	same := irregularActions(250)
	result := BestLagCorrelation(ref, same, 1000, 50, 100)
	if result == nil {
		t.Fatal("expected a correlation result")
	}
	if result.R <= 0.9 {
		t.Fatalf("r=%.4f, want >0.9", result.R)
	}
	if result.ShapeError == nil || *result.ShapeError >= 0.3 {
		t.Fatalf("shape_error=%v, want <0.3", result.ShapeError)
	}
}

func TestBestLagCorrelationLowConfidence(t *testing.T) {
	shortRef := sineActions(2000, 500, 40, 0, 40, 50)
	shortMatch := sineActions(2000, 500, 40, 50, 40, 50)
	result := BestLagCorrelation(shortRef, shortMatch, 1000, 50, 100)
	if result == nil {
		t.Fatal("expected a correlation result")
	}
	if !result.LowConfidence {
		t.Fatalf("short overlap should be low_confidence; n_samples=%d", result.NSamples)
	}
}

func TestDiagnosePhaseTimingVsShape(t *testing.T) {
	reference := sineActions(20000, 2000, 40, 0, 40, 50)
	shifted := sineActions(20000, 2000, 40, 837, 40, 50)
	corr := BestLagCorrelation(reference, shifted, 2000, 50, 100)
	d := DiagnosePhase(corr, 0, 0)
	if d.Verdict != PhaseTiming {
		t.Fatalf("verdict=%s, want timing (detail=%s)", d.Verdict, d.Detail)
	}

	// Unrelated noise → shape
	noise := make([]Action, 0, 200)
	for i := 0; i < 200; i++ {
		// Deterministic but not matching the sine — sawtooth-ish.
		noise = append(noise, Action{At: int64(i * 100), Pos: (i * 17) % 101})
	}
	corrNoise := BestLagCorrelation(reference, noise, 500, 100, 100)
	dNoise := DiagnosePhase(corrNoise, 0, 0)
	if dNoise.Verdict != PhaseShape && dNoise.Verdict != PhaseUndefined {
		t.Fatalf("noise verdict=%s, want shape or undefined", dNoise.Verdict)
	}

	if DiagnosePhase(nil, 0, 0).Verdict != PhaseUndefined {
		t.Fatal("nil correlation → undefined")
	}
}

func TestWindowedConstantLag(t *testing.T) {
	ref := sineActions(120000, 4000, 40, 0, 40, 50)
	shifted := sineActionsPhase(120000, 4000, 40, 0, 40, 50, 500)
	results := WindowedBestLagCorrelation(ref, shifted, 30000, 1000, 100, 100)
	if len(results) == 0 {
		t.Fatal("expected windows")
	}
	var confident []WindowResult
	for _, r := range results {
		if r.R != nil {
			confident = append(confident, r)
		}
	}
	if len(confident) != len(results) {
		t.Fatalf("all windows should yield a result, got %d/%d", len(confident), len(results))
	}
	lagSet := map[int]struct{}{}
	for _, r := range confident {
		lagSet[r.LagMs] = struct{}{}
		if *r.R <= 0.9 {
			t.Fatalf("window r=%.4f, want >0.9", *r.R)
		}
	}
	if len(lagSet) != 1 {
		t.Fatalf("constant lag should yield one lag across windows, got %v", lagSet)
	}
}

func TestWindowedDriftingLag(t *testing.T) {
	ref := sineActions(120000, 4000, 40, 0, 40, 50)
	drift := driftingLagActions(120000, 4000, 2000, 40, 40, 50)
	results := WindowedBestLagCorrelation(ref, drift, 15000, 2500, 100, 100)
	var confident []WindowResult
	for _, r := range results {
		if r.R != nil {
			confident = append(confident, r)
		}
	}
	if len(confident) < 6 {
		t.Fatalf("expected >=6 confident windows, got %d", len(confident))
	}
	early := confident[0].LagMs
	late := confident[len(confident)-1].LagMs
	if absInt(late-early) <= 500 {
		t.Fatalf("drifting lag: early=%d late=%d, want |delta| > 500", early, late)
	}
	for _, r := range confident {
		if *r.R <= 0.8 {
			t.Fatalf("window r=%.4f, want >0.8", *r.R)
		}
	}
}

func TestWindowedTooFewActions(t *testing.T) {
	sparseRef := []Action{{At: 0, Pos: 50}, {At: 200000, Pos: 60}}
	sparseVariant := sineActions(200000, 4000, 40, 0, 40, 50)
	results := WindowedBestLagCorrelation(sparseRef, sparseVariant, 20000, 1000, 100, 100)
	if len(results) != 10 {
		t.Fatalf("expected 10 windows, got %d", len(results))
	}
	found := false
	for _, r := range results {
		if r.R == nil && r.Reason == "too_few_actions_in_window" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected at least one too_few_actions_in_window window")
	}
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
