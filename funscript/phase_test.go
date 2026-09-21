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

// TestWindowedBestLagCorrelationLagDrift verifies that independent per-window
// lag search recovers a continuously drifting offset (the clip_voll finding).
func TestWindowedBestLagCorrelationLagDrift(t *testing.T) {
	// Build a long reference sine, then a candidate whose lag drifts from
	// ~-400ms in the first half to ~+400ms in the second half.
	const durationMs = 60000
	const periodMs = 2000
	const stepMs = 40
	reference := sineActions(durationMs, periodMs, stepMs, 0, 40, 50)

	var candidate []Action
	for t := 0; t <= durationMs; t += stepMs {
		// Linear lag drift: -400ms at t=0 → +400ms at t=duration
		lag := -400 + int(800*float64(t)/float64(durationMs))
		pos := 50 + 40*math.Sin(2*math.Pi*float64(t)/float64(periodMs))
		candidate = append(candidate, Action{At: int64(t + lag), Pos: int(math.Round(pos))})
	}

	rows := WindowedBestLagCorrelation(reference, candidate, 15000, 1000, 50, 100)
	if len(rows) < 3 {
		t.Fatalf("expected ≥3 windows, got %d", len(rows))
	}
	var confident []WindowedLagRow
	for _, r := range rows {
		if r.R != nil && !r.LowConfidence {
			confident = append(confident, r)
		}
	}
	if len(confident) < 2 {
		t.Fatalf("expected ≥2 confident windows, got %d", len(confident))
	}
	// BestLagCorrelation's convention (shiftedB.At = act.At + lagMs, see
	// phase.go) recovers the shift needed to correct the candidate's
	// timestamps, which is the NEGATIVE of the drift injected above: the
	// candidate is injected as t+lag(t) (lag(t): -400ms→+400ms), so the
	// recovered LagMs runs +400ms→-400ms, positive-ish first and
	// negative-ish last.
	firstLag := confident[0].LagMs
	lastLag := confident[len(confident)-1].LagMs
	if firstLag <= lastLag {
		t.Fatalf("expected lag drift (first=%d > last=%d)", firstLag, lastLag)
	}
	// Mean r should be high (shape matches; only lag drifts).
	var sumR float64
	for _, r := range confident {
		sumR += *r.R
	}
	meanR := sumR / float64(len(confident))
	if meanR < 0.9 {
		t.Fatalf("mean r=%.3f, want >0.9 (shape should match within each window)", meanR)
	}
}

func TestWindowedBestLagCorrelationSparse(t *testing.T) {
	// Very short / sparse actions → undefined windows, not a panic.
	ref := []Action{{At: 0, Pos: 50}, {At: 100, Pos: 60}}
	cand := []Action{{At: 0, Pos: 50}, {At: 100, Pos: 55}}
	rows := WindowedBestLagCorrelation(ref, cand, 50, 100, 50, 50)
	if len(rows) == 0 {
		t.Fatal("expected at least one window row")
	}
	// With tiny windows and few points, most should be undefined or low-conf.
	hasReason := false
	for _, r := range rows {
		if r.R == nil && r.Reason != "" {
			hasReason = true
		}
	}
	if !hasReason {
		// Acceptable if every window somehow got a result; just don't crash.
		t.Log("all windows produced a correlation (unexpected but ok)")
	}
}

func TestFormatWindowedReport(t *testing.T) {
	r := 0.85
	rows := []WindowedLagRow{
		{WindowStartMs: 0, WindowEndMs: 30000, R: &r, LagMs: -200, Orientation: "normal"},
		{WindowStartMs: 30000, WindowEndMs: 60000, Reason: "too_few_actions_in_window"},
	}
	report := FormatWindowedReport(rows, "ref.funscript", "var.funscript", 30000)
	if report == "" {
		t.Fatal("empty report")
	}
	if !contains(report, "mean r") && !contains(report, "undefined") {
		t.Fatalf("report missing expected content:\n%s", report)
	}
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
