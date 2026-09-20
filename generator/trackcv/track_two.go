//go:build cgo && opencv

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

	if opts.StartTimeSec > 0 {
		sf := int(opts.StartTimeSec*fps + 0.5)
		if sf > 0 {
			cap.Seek(sf)
		}
	}

	if !cap.Read() {
		return Result{}, &trackError{"Erster Frame konnte nicht gelesen werden"}
	}

	trackerA := NewTracker()
	trackerA.Init(cap, roiA)
	var trackerB *Tracker
	boxA, boxB := roiA, roiB
	if !opts.FixedB {
		trackerB = NewTracker()
		trackerB.Init(cap, roiB)
	}
	defer func() {
		trackerA.Close()
		if trackerB != nil {
			trackerB.Close()
		}
	}()

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
	prog := newProgressReporter(opts.OnProgress, total)
	prog.report(0)

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
		if okA {
			boxA = newA
		}
		okB := true
		if trackerB != nil {
			var newB Rect
			newB, okB = trackerB.Update(cap)
			if okB {
				boxB = newB
			}
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
		prog.report(idx)
		idx++
	}
	prog.report(idx)

	if opts.StartTimeSec > 0 {
		off := int(opts.StartTimeSec*1000 + 0.5)
		for i := range timestamps {
			timestamps[i] += off
		}
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
