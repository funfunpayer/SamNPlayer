package funscript

import (
	"math"
	"sort"
)

// Phase-Analyzer defaults — mirrored from generator/fungen_compare.py so
// Go and Python agree on the same lag search when diagnosing a clip.
const (
	DefaultMaxLagMs       = 1000
	DefaultLagStepMs      = 100
	DefaultResampleStepMs = 100
	MinOverlapSamples     = 10
	LowConfidenceSamples  = 30
	DefaultTimingGainMin  = 0.30 // aligned r must beat raw by this much → "timing"
	DefaultMinAlignedR    = 0.70 // below this after alignment → shape/perception
)

// LagCorrelation is the result of BestLagCorrelation — same fields as
// fungen_compare.best_lag_correlation(). Nil pointer fields mean
// "undefined" (never 0.0 for a missing value).
type LagCorrelation struct {
	R             float64  `json:"r"`
	LagMs         int      `json:"lag_ms"`
	Orientation   string   `json:"orientation"` // "normal" or "inverted"
	NSamples      int      `json:"n_samples"`
	ShapeError    *float64 `json:"shape_error"`
	RZeroLag      *float64 `json:"r_zero_lag"`
	LowConfidence bool     `json:"low_confidence"`
}

// PhaseVerdict is the diagnose step from docs/ENGINE.md (timing vs shape):
// after best-lag, decide whether to fix timing or perception/shape.
type PhaseVerdict string

const (
	PhaseUndefined PhaseVerdict = "undefined" // constant / too-short overlap
	PhaseTiming    PhaseVerdict = "timing"    // shape OK, phase wrong
	PhaseShape     PhaseVerdict = "shape"     // aligned correlation still weak
	PhaseOK        PhaseVerdict = "ok"        // raw already matches well
)

// PhaseDiagnosis pairs the lag search with the research-doc decision tree.
type PhaseDiagnosis struct {
	Correlation *LagCorrelation `json:"correlation"`
	Verdict     PhaseVerdict    `json:"verdict"`
	Detail      string          `json:"detail"`
}

// MotionFidelityResult is a labeled Motion Fidelity report
// (docs/SIGNAL_VS_FIDELITY.md) — reference vs candidate via BestLagCorrelation.
// Distinct from ScriptQualityResult (Signal Quality only).
type MotionFidelityResult struct {
	Kind          string         `json:"kind"` // always "motion_fidelity"
	Diagnosis     PhaseDiagnosis `json:"diagnosis"`
	R             *float64       `json:"r"`
	RZeroLag      *float64       `json:"r_zero_lag"`
	LagMs         *int           `json:"lag_ms"`
	Orientation   string         `json:"orientation,omitempty"`
	LowConfidence bool           `json:"low_confidence"`
}

// EvaluateMotionFidelity compares reference A against candidate B.
// Defaults for lag search match BestLagCorrelation / fungen_compare.
func EvaluateMotionFidelity(reference, candidate []Action, maxLagMs, lagStepMs, resampleStepMs int) MotionFidelityResult {
	corr := BestLagCorrelation(reference, candidate, maxLagMs, lagStepMs, resampleStepMs)
	diag := DiagnosePhase(corr, 0, 0)
	out := MotionFidelityResult{
		Kind:      "motion_fidelity",
		Diagnosis: diag,
	}
	if corr != nil {
		r := corr.R
		out.R = &r
		out.RZeroLag = corr.RZeroLag
		lag := corr.LagMs
		out.LagMs = &lag
		out.Orientation = corr.Orientation
		out.LowConfidence = corr.LowConfidence
	}
	return out
}

// BestLagCorrelation searches lag and orientation for the best match of
// two action lists. Port of generator/fungen_compare.best_lag_correlation:
// shared absolute timeline, ±maxLagMs search, normal + inverted (100-pos).
// Returns nil when no usable overlap exists or Pearson is undefined
// (constant series) for every candidate lag.
func BestLagCorrelation(a, b []Action, maxLagMs, lagStepMs, resampleStepMs int) *LagCorrelation {
	if maxLagMs < 0 {
		maxLagMs = DefaultMaxLagMs
	}
	if lagStepMs <= 0 {
		lagStepMs = DefaultLagStepMs
	}
	if resampleStepMs <= 0 {
		resampleStepMs = DefaultResampleStepMs
	}
	if len(a) < 2 || len(b) < 2 {
		return nil
	}

	aSorted := sortActions(a)
	bSorted := sortActions(b)
	aT0, aT1 := aSorted[0].At, aSorted[len(aSorted)-1].At
	bT0, bT1 := bSorted[0].At, bSorted[len(bSorted)-1].At

	var best *LagCorrelation
	var zeroLagR *float64

	for lagMs := -maxLagMs; lagMs <= maxLagMs; lagMs += lagStepMs {
		overlapStart := max64(aT0, bT0+int64(lagMs))
		overlapEnd := min64(aT1, bT1+int64(lagMs))
		if overlapEnd <= overlapStart {
			continue
		}
		nSamples := int((overlapEnd-overlapStart)/int64(resampleStepMs)) + 1
		if nSamples < MinOverlapSamples {
			continue
		}

		seriesA := resampleAbsolute(aSorted, overlapStart, overlapEnd, resampleStepMs)
		shiftedB := make([]Action, len(bSorted))
		for i, act := range bSorted {
			shiftedB[i] = Action{At: act.At + int64(lagMs), Pos: act.Pos}
		}
		seriesB := resampleAbsolute(shiftedB, overlapStart, overlapEnd, resampleStepMs)

		rNormal := pearson(seriesA, seriesB)
		var rInverted *float64
		invertedB := invertSeries(seriesB)
		if rNormal != nil {
			rInverted = pearson(seriesA, invertedB)
		}

		candidates := []struct {
			r           *float64
			orientation string
			sb          []float64
		}{
			{rNormal, "normal", seriesB},
			{rInverted, "inverted", invertedB},
		}
		for _, c := range candidates {
			if c.r == nil {
				continue
			}
			if best == nil || *c.r > best.R {
				se := shapeNormalizedError(seriesA, c.sb)
				best = &LagCorrelation{
					R:           *c.r,
					LagMs:       lagMs,
					Orientation: c.orientation,
					NSamples:    len(seriesA),
					ShapeError:  se,
				}
			}
		}
		if lagMs == 0 && rNormal != nil {
			v := *rNormal
			zeroLagR = &v
		}
	}

	if best == nil {
		return nil
	}
	best.RZeroLag = zeroLagR
	best.LowConfidence = best.NSamples < LowConfidenceSamples
	return best
}

// DiagnosePhase applies the research-doc flow: if aligned correlation is
// much better than raw → timing; if aligned stays weak → shape/perception;
// otherwise OK. timingGainMin / minAlignedR ≤ 0 use the package defaults.
func DiagnosePhase(corr *LagCorrelation, timingGainMin, minAlignedR float64) PhaseDiagnosis {
	if timingGainMin <= 0 {
		timingGainMin = DefaultTimingGainMin
	}
	if minAlignedR <= 0 {
		minAlignedR = DefaultMinAlignedR
	}
	if corr == nil {
		return PhaseDiagnosis{
			Verdict: PhaseUndefined,
			Detail:  "keine verwertbare Überlappung oder konstante Serie (Pearson undefiniert)",
		}
	}
	d := PhaseDiagnosis{Correlation: corr}
	if corr.R < minAlignedR {
		d.Verdict = PhaseShape
		d.Detail = "aligned correlation bleibt schwach — Perception/Normalisierung/ROI prüfen"
		return d
	}
	raw := 0.0
	hasRaw := corr.RZeroLag != nil
	if hasRaw {
		raw = *corr.RZeroLag
	}
	if hasRaw && corr.R-raw >= timingGainMin {
		d.Verdict = PhaseTiming
		d.Detail = "Kurve passt nach Lag-Ausrichtung — primär Timing/Phase reparieren"
		return d
	}
	d.Verdict = PhaseOK
	d.Detail = "Rohkorrelation und Alignierung sind beide brauchbar"
	return d
}

func sortActions(in []Action) []Action {
	out := append([]Action(nil), in...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].At < out[j].At })
	return out
}

func resampleAbsolute(actions []Action, tStart, tEnd int64, stepMs int) []float64 {
	times := make([]float64, len(actions))
	positions := make([]float64, len(actions))
	for i, a := range actions {
		times[i] = float64(a.At)
		positions[i] = float64(a.Pos)
	}
	n := int((tEnd-tStart)/int64(stepMs)) + 1
	out := make([]float64, 0, n)
	for t := tStart; t <= tEnd; t += int64(stepMs) {
		out = append(out, interpClamp(times, positions, float64(t)))
	}
	return out
}

// interpClamp matches numpy.interp defaults: linear between knots, clamp
// outside the series' own time range (no extrapolation).
func interpClamp(xs, ys []float64, x float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	if x <= xs[0] {
		return ys[0]
	}
	if x >= xs[len(xs)-1] {
		return ys[len(ys)-1]
	}
	// Binary search for rightmost xs[i] <= x
	i := sort.Search(len(xs), func(i int) bool { return xs[i] > x })
	// xs[i-1] <= x < xs[i]
	x0, x1 := xs[i-1], xs[i]
	y0, y1 := ys[i-1], ys[i]
	if x1 == x0 {
		return y0
	}
	w := (x - x0) / (x1 - x0)
	return y0 + w*(y1-y0)
}

func invertSeries(s []float64) []float64 {
	out := make([]float64, len(s))
	for i, v := range s {
		out[i] = 100.0 - v
	}
	return out
}

// pearson returns Pearson r, or nil if either series has ~zero variance
// (matches fungen_compare.pearson — never 0.0 for undefined).
func pearson(a, b []float64) *float64 {
	if len(a) < 2 || len(a) != len(b) {
		return nil
	}
	meanA, meanB := mean(a), mean(b)
	var sumA, sumB, sumAB float64
	for i := range a {
		da := a[i] - meanA
		db := b[i] - meanB
		sumA += da * da
		sumB += db * db
		sumAB += da * db
	}
	// Population std (ddof=0), same as numpy.std / corrcoef default.
	n := float64(len(a))
	stdA := math.Sqrt(sumA / n)
	stdB := math.Sqrt(sumB / n)
	if stdA < 1e-9 || stdB < 1e-9 {
		return nil
	}
	r := sumAB / (n * stdA * stdB)
	return &r
}

func shapeNormalizedError(a, b []float64) *float64 {
	if len(a) < 2 || len(a) != len(b) {
		return nil
	}
	meanA, meanB := mean(a), mean(b)
	stdA, stdB := popStd(a, meanA), popStd(b, meanB)
	if stdA < 1e-9 || stdB < 1e-9 {
		return nil
	}
	var sum float64
	for i := range a {
		za := (a[i] - meanA) / stdA
		zb := (b[i] - meanB) / stdB
		sum += math.Abs(za - zb)
	}
	v := sum / float64(len(a))
	return &v
}

func mean(xs []float64) float64 {
	var s float64
	for _, v := range xs {
		s += v
	}
	return s / float64(len(xs))
}

func popStd(xs []float64, m float64) float64 {
	var s float64
	for _, v := range xs {
		d := v - m
		s += d * d
	}
	return math.Sqrt(s / float64(len(xs)))
}

func max64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func min64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}
