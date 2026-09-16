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

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
