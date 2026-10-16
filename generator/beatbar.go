package generator

import (
	"math"
	"sort"
)

// Beat bar (Cock Hero / Fap Hero, Owner 15 Oct): these videos show a strip
// where dots or icons scroll at a steady speed toward a fixed target; every
// hit is one stroke. The strip is found in the picture itself, so the
// script follows exactly what the video shows, not the guessed motion of
// bodies.
//
// Pure functions on small grey frames (tests use synthetic frames); the
// ffmpeg decoding lives in beatbar_video.go.
//
//  1. Find the strip: split the frame into horizontal bands; per band take
//     the column profile of every frame and look for a steady sideways
//     shift between frames (the scrolling dots). The band with the most
//     consistent non-zero shift wins.
//  2. Remove what stands still (the target marker, a frame): subtract a
//     running per-column median.
//  3. Speed = the most common shift; target = where the moving peaks end.
//  4. Each crossing of the target is a beat, timed to a fraction of a frame.

// GrayFrame is one decoded frame, 8-bit luma, row-major.
type GrayFrame struct {
	W, H int
	Pix  []byte
}

// BeatBar describes a found strip.
type BeatBar struct {
	Y0, Y1   int     // rows of the band (Y1 exclusive)
	Velocity float64 // px per frame, sign = direction (negative = leftward)
	TargetX  float64 // column where the dots end
	Score    float64 // share of frame pairs with the steady shift (0–1)
}

const (
	beatBarBands    = 24
	beatBarMaxShift = 40
	beatBarMinCorr  = 0.6
	beatBarMinScore = 0.35
	beatBarBgFrames = 15
)

// bandProfile returns the mean of rows [y0,y1) for every column.
func bandProfile(f GrayFrame, y0, y1 int) []float64 {
	p := make([]float64, f.W)
	if y1 <= y0 {
		return p
	}
	for y := y0; y < y1; y++ {
		row := f.Pix[y*f.W : (y+1)*f.W]
		for x, v := range row {
			p[x] += float64(v)
		}
	}
	n := float64(y1 - y0)
	for x := range p {
		p[x] /= n
	}
	return p
}

// bestShift finds s in [-maxShift, maxShift] that maximises the normalised
// correlation of b against a shifted by s (b[x] ≈ a[x-s]).
func bestShift(a, b []float64, maxShift int) (shift int, corr float64) {
	corr = -2
	for s := -maxShift; s <= maxShift; s++ {
		var sa, sb, saa, sbb, sab float64
		n := 0
		for x := 0; x < len(b); x++ {
			xa := x - s
			if xa < 0 || xa >= len(a) {
				continue
			}
			va, vb := a[xa], b[x]
			sa += va
			sb += vb
			saa += va * va
			sbb += vb * vb
			sab += va * vb
			n++
		}
		if n < len(b)/2 {
			continue
		}
		fn := float64(n)
		cov := sab - sa*sb/fn
		den := math.Sqrt((saa - sa*sa/fn) * (sbb - sb*sb/fn))
		if den <= 1e-9 {
			continue
		}
		if c := cov / den; c > corr {
			corr, shift = c, s
		}
	}
	return shift, corr
}

// FindBeatBar looks for the scrolling strip in frames. ok is false when no
// band shows a steady sideways motion.
func FindBeatBar(frames []GrayFrame) (bar BeatBar, ok bool) {
	if len(frames) < 10 {
		return bar, false
	}
	w, h := frames[0].W, frames[0].H
	bandH := h / beatBarBands
	if bandH < 2 || w < 32 {
		return bar, false
	}
	maxShift := beatBarMaxShift
	if maxShift > w/4 {
		maxShift = w / 4
	}
	best := BeatBar{Score: -1}
	for b := 0; b < beatBarBands; b++ {
		y0, y1 := b*bandH, (b+1)*bandH
		var bg bgRemover
		profiles := make([][]float64, len(frames))
		for i, f := range frames {
			profiles[i] = bg.clean(bandProfile(f, y0, y1))
		}
		shifts := map[int]int{}
		steady := 0
		for i := 1; i < len(profiles); i++ {
			if flatProfile(profiles[i]) || flatProfile(profiles[i-1]) {
				continue
			}
			s, c := bestShift(profiles[i-1], profiles[i], maxShift)
			if c >= beatBarMinCorr && s != 0 {
				shifts[s]++
				steady++
			}
		}
		if steady == 0 {
			continue
		}
		mode, modeN := 0, 0
		for s, n := range shifts {
			if n > modeN || (n == modeN && abs(s) < abs(mode)) {
				mode, modeN = s, n
			}
		}
		// Neighbouring shifts (±1) are the same speed with rounding.
		near := modeN + shifts[mode-1] + shifts[mode+1]
		score := float64(near) / float64(len(frames)-1)
		if score > best.Score {
			v := weightedShift(shifts, mode)
			best = BeatBar{Y0: y0, Y1: y1, Velocity: v, Score: score}
		}
	}
	if best.Score < beatBarMinScore {
		return best, false
	}
	best.TargetX = findTarget(frames, best)
	return best, true
}

// bgRemover subtracts the per-column median of the last beatBarBgFrames
// raw profiles (what stands still: the target marker, a frame), keeping
// only what is brighter than that still background.
type bgRemover struct{ window [][]float64 }

func (r *bgRemover) clean(raw []float64) []float64 {
	r.window = append(r.window, raw)
	if len(r.window) > beatBarBgFrames {
		r.window = r.window[1:]
	}
	out := make([]float64, len(raw))
	col := make([]float64, len(r.window))
	for x := range raw {
		for k, p := range r.window {
			col[k] = p[x]
		}
		sort.Float64s(col)
		if d := raw[x] - col[len(col)/2]; d > 0 {
			out[x] = d
		}
	}
	return out
}

func flatProfile(p []float64) bool {
	for _, v := range p {
		if v > 8 {
			return false
		}
	}
	return true
}

func weightedShift(shifts map[int]int, mode int) float64 {
	var sum, n float64
	for _, s := range []int{mode - 1, mode, mode + 1} {
		sum += float64(s * shifts[s])
		n += float64(shifts[s])
	}
	if n == 0 {
		return float64(mode)
	}
	return sum / n
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// peaksOf returns the x of local maxima clearly above the profile's floor.
func peaksOf(p []float64) []float64 {
	var out []float64
	for x := 1; x < len(p)-1; x++ {
		v := p[x]
		if v < 20 || v < p[x-1] || v < p[x+1] {
			continue
		}
		// Plateau: report its centre once.
		end := x
		for end+1 < len(p) && p[end+1] == v {
			end++
		}
		out = append(out, float64(x+end)/2)
		x = end
	}
	return out
}

// findTarget: the moving peaks end at the target; take a low (or high)
// percentile of their positions along the motion.
func findTarget(frames []GrayFrame, bar BeatBar) float64 {
	var bg bgRemover
	var xs []float64
	for _, f := range frames {
		p := bg.clean(bandProfile(f, bar.Y0, bar.Y1))
		xs = append(xs, peaksOf(p)...)
	}
	if len(xs) == 0 {
		return 0
	}
	sort.Float64s(xs)
	q := 0.02
	if bar.Velocity > 0 {
		q = 0.98
	}
	return xs[int(q*float64(len(xs)-1))]
}

// beatTracker times beats frame by frame (BeatTimes and the streaming
// pass share it).
type beatTracker struct {
	bar      BeatBar
	frameMs  float64
	minGapMs float64
	bg       bgRemover
	frame    int
	beats    []float64
}

func newBeatTracker(bar BeatBar, fps float64) *beatTracker {
	frameMs := 1000 / fps
	return &beatTracker{bar: bar, frameMs: frameMs, minGapMs: math.Max(2*frameMs, 80)}
}

// add takes the next frame's raw band profile.
func (t *beatTracker) add(raw []float64) {
	p := t.bg.clean(raw)
	v := t.bar.Velocity
	speed := math.Abs(v)
	for _, x := range peaksOf(p) {
		// Distance still to go toward the target along the motion.
		d := x - t.bar.TargetX
		if v > 0 {
			d = t.bar.TargetX - x
		}
		if d < 0 || d >= speed {
			continue
		}
		at := (float64(t.frame) + d/speed) * t.frameMs
		if n := len(t.beats); n > 0 && at-t.beats[n-1] < t.minGapMs {
			continue
		}
		t.beats = append(t.beats, at)
	}
	t.frame++
}

// BeatTimes returns the beat times (ms) for frames at fps: every moment a
// moving peak crosses the target. Crossings closer than 2 frames (or
// 80 ms) merge.
func BeatTimes(frames []GrayFrame, bar BeatBar, fps float64) []float64 {
	if fps <= 0 || bar.Velocity == 0 {
		return nil
	}
	tr := newBeatTracker(bar, fps)
	for _, f := range frames {
		tr.add(bandProfile(f, bar.Y0, bar.Y1))
	}
	return tr.beats
}
