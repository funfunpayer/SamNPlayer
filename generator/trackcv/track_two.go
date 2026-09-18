//go:build cgo && opencv && !windows

package trackcv

import "math"

// TrackTwoPoints follows two ROIs and returns 2D center distance as Positions —
// Go port of generate_funscript.track_two_points (CSRT, no appearance memory;
// shared pan cancels in the distance by construction).
func TrackTwoPoints(videoPath string, roiA, roiB Rect, opts Options) (Result, error) {
	cap := OpenVideo(videoPath)
	defer cap.Close()
	if !cap.IsOpened() {
		return Result{}, &trackError{"Video konnte nicht geöffnet werden: " + videoPath}
	}

	fps := cap.Get(CapPropFPS)
	if fps <= 0 {
		fps = 30.0
	}
	width := int(cap.Get(CapPropFrameWidth))
	height := int(cap.Get(CapPropFrameHeight))

	if !cap.Read() {
		return Result{}, &trackError{"Erster Frame konnte nicht gelesen werden"}
	}

	trackerA := NewTracker()
	trackerB := NewTracker()
	defer func() {
		trackerA.Close()
		trackerB.Close()
	}()
	trackerA.Init(cap, roiA)
	trackerB.Init(cap, roiB)
	boxA, boxB := roiA, roiB

	dist := func(a, b Rect) float64 {
		ax := float64(a.X) + float64(a.W)/2
		ay := float64(a.Y) + float64(a.H)/2
		bx := float64(b.X) + float64(b.W)/2
		by := float64(b.Y) + float64(b.H)/2
		return math.Hypot(bx-ax, by-ay)
	}

	timestamps := []int{0}
	distances := []float64{dist(boxA, boxB)}
	lostFlags := []bool{false}
	lost, valid := 0, 1
	idx := 1

	total := int(cap.Get(CapPropFrameCount))
	if opts.MaxFrames > 0 && (total == 0 || opts.MaxFrames < total) {
		total = opts.MaxFrames
	}

	for {
		if opts.Cancel != nil && opts.Cancel() {
			return Result{Canceled: true}, ErrCanceled
		}
		if opts.MaxFrames > 0 && idx >= opts.MaxFrames {
			break
		}
		if !cap.Read() {
			break
		}
		newA, okA := trackerA.Update(cap)
		newB, okB := trackerB.Update(cap)
		if okA {
			boxA = newA
		}
		if okB {
			boxB = newB
		}
		frameLost := !(okA && okB)
		if frameLost {
			lost++
		} else {
			valid++
		}
		distances = append(distances, dist(boxA, boxB))
		timestamps = append(timestamps, int(float64(idx)*1000.0/fps))
		lostFlags = append(lostFlags, frameLost)
		idx++
		_ = total
	}

	conf := 0.0
	if idx > 0 {
		conf = float64(valid) / float64(idx)
	}
	reason := ""
	if idx > 0 && float64(lost)/float64(idx) > 0.5 {
		reason = "tracker_lost_heavy"
	} else if idx > 0 && float64(lost)/float64(idx) > 0.15 {
		reason = "tracker_lost_elevated"
	}

	return Result{
		TimestampsMs: timestamps,
		Positions:    distances,
		Width:        width,
		Height:       height,
		LostFlags:    lostFlags,
		Stats: Stats{
			TrackerLostFrames: lost,
			TotalFrames:       idx,
			VerticalRange:     ptp(distances),
			ValidFrames:       valid,
			Confidence:        conf,
			Reason:            reason,
		},
	}, nil
}
