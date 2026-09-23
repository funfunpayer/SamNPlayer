package trackcv

import "math"

// Rhythm grid: an alternative source for the stroke signal that tolerates
// CSRT drift.
//
// CSRT on long clips walks off the target gradually (clip_voll: by ~140s the
// box sits ~110px beside the shaft), and the stroke curve is the box's own
// motion - so the curve degrades with it. The rhythm grid keeps CSRT only as
// a rough anchor: per frame the dense optical flow is averaged onto a coarse
// cell grid (FlowCells), and per 8s window the cell near the box whose
// motion is most concentrated at the stroke tempo supplies the signal. The
// box only has to stay within ~3 cells of the real action, not on it.
//
// Measured against both FunGen references through the production post
// pipeline (docs/AGENT_COORD.md, 23 Sep, "rhythm grid"): clip_voll windowed
// r 0.386/0.552 -> 0.411/0.767, clip_ausschnitt 0.449/0.712 -> 0.466/0.877.
const (
	rhythmGridCols    = 16
	rhythmWindowSec   = 8.0
	rhythmStepSec     = 2.0
	rhythmSearchCells = 3.0 // search radius around the box, in cell widths
	rhythmTempoLoHz   = 0.6
	rhythmTempoHiHz   = 2.5
	rhythmBandHz      = 0.15
	rhythmTotalLoHz   = 0.2
	rhythmTotalHiHz   = 6.0
	rhythmTopCells    = 10
	rhythmAnchorSec   = 2.0 // detrend window for the CSRT sign reference
	// rhythmSignMinR: below this |r| between the cell and the tracker the
	// tracker's sign is a coin toss, so orientation comes from continuity
	// with the curve already written instead. clip_voll: 4 of 140 chunks
	// (all |r| <= 0.04), windowed r vs the YOLO reference 0.658 -> 0.766
	// and 10/10 windows consistently oriented. Raising it to >= 0.25 also
	// overrides still-informative tracker signs and measured worse.
	rhythmSignMinR = 0.1
)

// rhythmGridRows keeps cells roughly square for the frame's aspect ratio.
func rhythmGridRows(width, height int) int {
	if width <= 0 || height <= 0 {
		return 9
	}
	return max(1, int(math.Round(float64(rhythmGridCols)*float64(height)/float64(width))))
}

// rhythmGridPositions builds a position curve from per-frame cell flow.
// cellV[i] is the flow (px/frame, one axis) of every cell between frame i-1
// and i (all zero for frame 0 and scene cuts). cx/cy are the CSRT box
// centers per frame, anchor the CSRT position on the same axis. Windows
// where no cell carries a rhythm fall back to the CSRT motion, so the result
// is never worse-defined than the tracker alone.
func rhythmGridPositions(cellV [][]float32, gw, gh int, width, height int,
	cx, cy, anchor []float64, fps float64) []float64 {
	n := len(cellV)
	if n == 0 || len(cx) != n || len(cy) != n || len(anchor) != n || fps <= 0 {
		return append([]float64(nil), anchor...)
	}
	cells := gw * gh
	cellW, cellH := float64(width)/float64(gw), float64(height)/float64(gh)
	radius := rhythmSearchCells * cellW

	signRef := subtractRollingMean(anchor, int(math.Round(rhythmAnchorSec*fps)))

	// Default: the tracker's own motion.
	v := make([]float64, n)
	for i := 1; i < n; i++ {
		v[i] = anchor[i] - anchor[i-1]
	}

	win := int(math.Round(rhythmWindowSec * fps))
	step := int(math.Round(rhythmStepSec * fps))
	if win < 8 || step < 1 {
		return append([]float64(nil), anchor...)
	}
	x := make([][]float64, cells) // per-cell window, reused
	written := 0                  // v[:written] already comes from grid cells
	for s := -win/2 + step/2; s < n; s += step {
		a, b := max(0, s), min(n, s+win)
		m := b - a
		if m < win/2 {
			continue
		}
		for c := 0; c < cells; c++ {
			if cap(x[c]) < m {
				x[c] = make([]float64, m)
			}
			x[c] = x[c][:m]
			// Integrated motion = the cell's "position" within the window
			// (the offset from before the window drops out in detrending,
			// so no whole-clip position matrix is needed).
			acc := 0.0
			for i := 0; i < m; i++ {
				acc += float64(cellV[a+i][c])
				x[c][i] = acc
			}
			detrendLinear(x[c])
		}
		score := rhythmScores(x, fps)
		if score == nil {
			continue
		}
		mid := min(n-1, s+win/2)
		best, bestS := -1, 0.0
		for c := 0; c < cells; c++ {
			px := (float64(c%gw) + 0.5) * cellW
			py := (float64(c/gw) + 0.5) * cellH
			if math.Hypot(px-cx[mid], py-cy[mid]) <= radius && score[c] > bestS {
				best, bestS = c, score[c]
			}
		}
		if best < 0 {
			continue
		}
		// Cell flow has no inherent orientation relative to the stroke
		// (a cell may sit on a part moving opposite to the tip): take the
		// sign from the tracker, which is locally right even when drifting.
		r := pearson(x[best], signRef[a:b])
		if math.Abs(r) < rhythmSignMinR && written-a > step {
			r = continuityR(cellV, best, v, a, written)
		}
		sgn := 1.0
		if r < 0 {
			sgn = -1
		}
		c0, c1 := max(1, mid-step/2), min(n, mid+step/2)
		for i := c0; i < c1; i++ {
			v[i] = sgn * float64(cellV[i][best])
		}
		written = c1
	}

	out := make([]float64, n)
	out[0] = anchor[0]
	for i := 1; i < n; i++ {
		out[i] = out[i-1] + v[i]
	}
	return out
}

// continuityR correlates cell's integrated motion over [a, end) with the
// curve already written there (v), so a new chunk keeps the orientation of
// the chunks before it.
func continuityR(cellV [][]float32, cell int, v []float64, a, end int) float64 {
	m := end - a
	cellPos, curve := make([]float64, m), make([]float64, m)
	var accC, accV float64
	for i := 0; i < m; i++ {
		accC += float64(cellV[a+i][cell])
		accV += v[a+i]
		cellPos[i], curve[i] = accC, accV
	}
	return pearson(cellPos, curve)
}

// rhythmScores returns, per cell, how strongly its motion in this window
// sits at the dominant stroke tempo: E^2/total, where E is the Hann-windowed
// power at f0 and 2*f0 (+-0.15Hz) and total the power in 0.2-6Hz. Squaring E
// favours cells that are both strong and clean. f0 is the peak of the summed
// spectrum of the most active cells in the 0.6-2.5Hz stroke range. nil when
// the window is too short to resolve the band.
func rhythmScores(x [][]float64, fps float64) []float64 {
	m := len(x[0])
	df := fps / float64(m)
	kLo, kHi := int(math.Ceil(rhythmTotalLoHz/df)), int(math.Floor(rhythmTotalHiHz/df))
	if kHi > m/2 {
		kHi = m / 2
	}
	if kHi <= kLo {
		return nil
	}
	hann := make([]float64, m)
	for i := range hann {
		hann[i] = 0.5 - 0.5*math.Cos(2*math.Pi*float64(i)/float64(m-1))
	}
	nb := kHi - kLo + 1
	cosT, sinT := make([][]float64, nb), make([][]float64, nb)
	for j := 0; j < nb; j++ {
		k := float64(kLo + j)
		cosT[j], sinT[j] = make([]float64, m), make([]float64, m)
		for i := 0; i < m; i++ {
			ph := 2 * math.Pi * k * float64(i) / float64(m)
			cosT[j][i], sinT[j][i] = hann[i]*math.Cos(ph), hann[i]*math.Sin(ph)
		}
	}
	cells := len(x)
	power := make([][]float64, cells)
	for c := 0; c < cells; c++ {
		power[c] = make([]float64, nb)
		for j := 0; j < nb; j++ {
			var re, im float64
			for i, xv := range x[c] {
				re += xv * cosT[j][i]
				im += xv * sinT[j][i]
			}
			power[c][j] = re*re + im*im
		}
	}
	freq := func(j int) float64 { return float64(kLo+j) * df }
	inTempo := func(j int) bool { f := freq(j); return f >= rhythmTempoLoHz && f <= rhythmTempoHiHz }

	// Most active cells in the stroke range decide the local tempo.
	act := make([]float64, cells)
	for c := range act {
		for j := 0; j < nb; j++ {
			if inTempo(j) {
				act[c] += power[c][j]
			}
		}
	}
	top := topIndices(act, rhythmTopCells)
	f0, bestSum := 0.0, -1.0
	for j := 0; j < nb; j++ {
		if !inTempo(j) {
			continue
		}
		sum := 0.0
		for _, c := range top {
			sum += power[c][j]
		}
		if sum > bestSum {
			f0, bestSum = freq(j), sum
		}
	}
	if f0 == 0 {
		return nil
	}

	score := make([]float64, cells)
	for c := 0; c < cells; c++ {
		var e, tot float64
		for j := 0; j < nb; j++ {
			f := freq(j)
			tot += power[c][j]
			if math.Abs(f-f0) <= rhythmBandHz || math.Abs(f-2*f0) <= rhythmBandHz {
				e += power[c][j]
			}
		}
		score[c] = e * e / (tot + 1e-9)
	}
	return score
}

// detrendLinear removes mean and least-squares slope in place.
func detrendLinear(x []float64) {
	m := len(x)
	if m == 0 {
		return
	}
	mean := 0.0
	for _, v := range x {
		mean += v
	}
	mean /= float64(m)
	var num, den float64
	for i, v := range x {
		t := float64(i) - float64(m-1)/2
		num += t * (v - mean)
		den += t * t
	}
	slope := 0.0
	if den > 0 {
		slope = num / den
	}
	for i := range x {
		x[i] -= mean + slope*(float64(i)-float64(m-1)/2)
	}
}

// subtractRollingMean returns x minus its centered moving average (window k).
func subtractRollingMean(x []float64, k int) []float64 {
	out := make([]float64, len(x))
	if k < 2 {
		copy(out, x)
		return out
	}
	prefix := make([]float64, len(x)+1)
	for i, v := range x {
		prefix[i+1] = prefix[i] + v
	}
	for i := range x {
		lo, hi := max(0, i-k/2), min(len(x), i-k/2+k)
		out[i] = x[i] - (prefix[hi]-prefix[lo])/float64(hi-lo)
	}
	return out
}

func pearson(a, b []float64) float64 {
	n := float64(len(a))
	if len(a) != len(b) || n < 2 {
		return 0
	}
	var ma, mb float64
	for i := range a {
		ma += a[i]
		mb += b[i]
	}
	ma, mb = ma/n, mb/n
	var sab, saa, sbb float64
	for i := range a {
		da, db := a[i]-ma, b[i]-mb
		sab += da * db
		saa += da * da
		sbb += db * db
	}
	if saa == 0 || sbb == 0 {
		return 0
	}
	return sab / math.Sqrt(saa*sbb)
}

// topIndices returns the indices of the k largest values (unordered).
func topIndices(v []float64, k int) []int {
	idx := make([]int, 0, k)
	for i := range v {
		if len(idx) < k {
			idx = append(idx, i)
			continue
		}
		minJ := 0
		for j := range idx {
			if v[idx[j]] < v[idx[minJ]] {
				minJ = j
			}
		}
		if v[i] > v[idx[minJ]] {
			idx[minJ] = i
		}
	}
	return idx
}
