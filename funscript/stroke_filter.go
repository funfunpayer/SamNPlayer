package funscript

import (
	"math"
	"sort"
)

// Stroke filter (v0.8, docs/ROADMAP_2026-10.md §3 item 2; seen in
// MultiFunPlayer, ScriptPlayer and UwU Player): applied to the stroke
// positions a linear device gets while playing. The script file is never
// changed.
//
//   - Smooth: a stroker moves in straight lines between the points it is
//     sent. "pchip" and "makima" add points along a smooth curve through
//     the script's own points (shape-preserving: no overshoot past
//     neighbouring extremes for pchip; makima keeps flat parts flat and
//     rounds corners a little more).
//   - MaxSpeed: no move faster than this many units (0–100) per second;
//     a too fast move stops short instead of slamming.
//   - IdleFill: script pauses longer than IdleFillMs get a gentle wave
//     around the level the pause holds, so the device does not sit still.

const (
	SmoothOff    = ""
	SmoothPCHIP  = "pchip"
	SmoothMakima = "makima"
)

// StrokeFilter configures ApplyStrokeFilter. The zero value changes nothing.
type StrokeFilter struct {
	Smooth     string  `json:"smooth"`     // "", "pchip" or "makima"
	StepMs     int64   `json:"stepMs"`     // sample spacing for Smooth (default 100)
	MaxSpeed   float64 `json:"maxSpeed"`   // units per second, 0 = no limit
	IdleFillMs int64   `json:"idleFillMs"` // fill pauses at least this long, 0 = off
}

// Active reports whether the filter changes anything.
func (f StrokeFilter) Active() bool {
	return f.Smooth != SmoothOff || f.MaxSpeed > 0 || f.IdleFillMs > 0
}

// NormalizeSmooth maps a name to a known smoothing mode ("" for unknown).
func NormalizeSmooth(s string) string {
	switch s {
	case SmoothPCHIP, SmoothMakima:
		return s
	}
	return SmoothOff
}

// ApplyStrokeFilter returns the filtered actions sorted by time.
// The input slice is not modified. Duplicate timestamps keep the last Pos.
func ApplyStrokeFilter(actions []Action, f StrokeFilter) []Action {
	out := make([]Action, len(actions))
	copy(out, actions)
	if !f.Active() || len(out) < 2 {
		return out
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].At < out[j].At })
	out = dedupeActionTimes(out)
	if len(out) < 2 {
		return out
	}
	if f.IdleFillMs > 0 {
		out = idleFill(out, f.IdleFillMs)
	}
	if s := NormalizeSmooth(f.Smooth); s != SmoothOff {
		step := f.StepMs
		if step <= 0 {
			step = 100
		}
		out = smoothResample(out, s, step)
	}
	if f.MaxSpeed > 0 {
		out = limitSpeed(out, f.MaxSpeed)
	}
	return out
}

// dedupeActionTimes keeps the last action for each identical At (caller sorted).
func dedupeActionTimes(a []Action) []Action {
	if len(a) < 2 {
		return a
	}
	out := make([]Action, 0, len(a))
	out = append(out, a[0])
	for i := 1; i < len(a); i++ {
		if a[i].At == out[len(out)-1].At {
			out[len(out)-1] = a[i]
			continue
		}
		out = append(out, a[i])
	}
	return out
}

// idleFill puts a slow wave (period idleWaveMs, ±idleWaveAmp around the
// pause's level) into every pause of at least minGap.
const (
	idleWaveMs  = 1600
	idleWaveAmp = 12
	idleMarginM = 400
)

func idleFill(a []Action, minGap int64) []Action {
	if minGap < 2*idleMarginM+idleWaveMs {
		minGap = 2*idleMarginM + idleWaveMs
	}
	out := make([]Action, 0, len(a))
	for i := 0; i < len(a); i++ {
		out = append(out, a[i])
		if i+1 >= len(a) {
			break
		}
		start, end := a[i].At, a[i+1].At
		if end-start < minGap {
			continue
		}
		level := (a[i].Pos + a[i+1].Pos) / 2
		lo, hi := clampPos(level-idleWaveAmp), clampPos(level+idleWaveAmp)
		up := true
		for t := start + idleMarginM; t+idleWaveMs/2 <= end-idleMarginM; t += idleWaveMs / 2 {
			p := lo
			if up {
				p = hi
			}
			up = !up
			out = append(out, Action{At: t, Pos: p})
		}
	}
	return out
}

// smoothResample keeps every original point and adds points every step ms
// along a pchip / makima curve through them.
func smoothResample(a []Action, mode string, step int64) []Action {
	n := len(a)
	x := make([]float64, n)
	y := make([]float64, n)
	for i, p := range a {
		x[i], y[i] = float64(p.At), float64(p.Pos)
	}
	var d []float64
	if mode == SmoothMakima {
		d = makimaSlopes(x, y)
	} else {
		d = pchipSlopes(x, y)
	}
	out := make([]Action, 0, n*2)
	for i := 0; i < n-1; i++ {
		out = append(out, a[i])
		h := x[i+1] - x[i]
		if h <= 0 {
			continue
		}
		for t := a[i].At + step; t < a[i+1].At; t += step {
			s := (float64(t) - x[i]) / h
			v := hermite(y[i], y[i+1], d[i]*h, d[i+1]*h, s)
			out = append(out, Action{At: t, Pos: clampPos(int(math.Round(v)))})
		}
	}
	return append(out, a[n-1])
}

func hermite(y0, y1, m0, m1, s float64) float64 {
	s2, s3 := s*s, s*s*s
	return (2*s3-3*s2+1)*y0 + (s3-2*s2+s)*m0 + (-2*s3+3*s2)*y1 + (s3-s2)*m1
}

// pchipSlopes: Fritsch–Carlson monotone slopes (no overshoot).
func pchipSlopes(x, y []float64) []float64 {
	n := len(x)
	d := make([]float64, n)
	delta := secants(x, y)
	if n == 2 {
		d[0], d[1] = delta[0], delta[0]
		return d
	}
	for i := 1; i < n-1; i++ {
		a, b := delta[i-1], delta[i]
		if a == 0 || b == 0 || (a > 0) != (b > 0) {
			continue // local extreme or flat: slope 0
		}
		h0, h1 := x[i]-x[i-1], x[i+1]-x[i]
		w1, w2 := 2*h1+h0, h1+2*h0
		d[i] = (w1 + w2) / (w1/a + w2/b)
	}
	d[0] = pchipEnd(x[1]-x[0], x[2]-x[1], delta[0], delta[1])
	d[n-1] = pchipEnd(x[n-1]-x[n-2], x[n-2]-x[n-3], delta[n-2], delta[n-3])
	return d
}

func pchipEnd(h0, h1, d0, d1 float64) float64 {
	s := ((2*h0+h1)*d0 - h0*d1) / (h0 + h1)
	if (s > 0) != (d0 > 0) || d0 == 0 {
		return 0
	}
	if (d0 > 0) != (d1 > 0) && math.Abs(s) > math.Abs(3*d0) {
		return 3 * d0
	}
	return s
}

// makimaSlopes: modified Akima slopes (flat runs stay flat, fewer wiggles).
func makimaSlopes(x, y []float64) []float64 {
	n := len(x)
	delta := secants(x, y)
	// Pad two secants on each side (linear extrapolation of the secants).
	m := make([]float64, n+3)
	for i := 0; i < n-1; i++ {
		m[i+2] = delta[i]
	}
	if n-1 >= 2 {
		m[1] = 2*m[2] - m[3]
		m[0] = 2*m[1] - m[2]
		m[n+1] = 2*m[n] - m[n-1]
		m[n+2] = 2*m[n+1] - m[n]
	} else {
		m[0], m[1], m[n+1], m[n+2] = m[2], m[2], m[2], m[2]
	}
	d := make([]float64, n)
	for i := 0; i < n; i++ {
		m0, m1, m2, m3 := m[i], m[i+1], m[i+2], m[i+3]
		w1 := math.Abs(m3-m2) + math.Abs(m3+m2)/2
		w2 := math.Abs(m1-m0) + math.Abs(m1+m0)/2
		if w1+w2 == 0 {
			d[i] = 0
			continue
		}
		d[i] = (w1*m1 + w2*m2) / (w1 + w2)
	}
	return d
}

func secants(x, y []float64) []float64 {
	n := len(x)
	delta := make([]float64, n-1)
	for i := 0; i < n-1; i++ {
		if h := x[i+1] - x[i]; h > 0 {
			delta[i] = (y[i+1] - y[i]) / h
		}
	}
	return delta
}

// limitSpeed walks forward; each point may move at most maxSpeed units per
// second from the previous (already limited) one.
func limitSpeed(a []Action, maxSpeed float64) []Action {
	out := make([]Action, len(a))
	copy(out, a)
	for i := 1; i < len(out); i++ {
		dt := float64(out[i].At-out[i-1].At) / 1000
		if dt <= 0 {
			out[i].Pos = out[i-1].Pos
			continue
		}
		maxStep := maxSpeed * dt
		diff := float64(out[i].Pos - out[i-1].Pos)
		if math.Abs(diff) > maxStep {
			if diff > 0 {
				out[i].Pos = out[i-1].Pos + int(math.Floor(maxStep))
			} else {
				out[i].Pos = out[i-1].Pos - int(math.Floor(maxStep))
			}
		}
	}
	return out
}
