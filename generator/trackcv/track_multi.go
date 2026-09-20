//go:build cgo && opencv

package trackcv

// Partner is one Tf/Tj contact anchor (ROI2 or extra --target).
type Partner struct {
	ROI   Rect
	Fixed bool
}

// TrackMultiPoints follows the tip ROI plus N contact partners and returns
// min tip→partner distance (TipPartnerDistance) as Positions.
// Lost tracked partners are excluded from min() for that frame (F-006);
// tip loss or no usable partner keeps the previous distance.
func TrackMultiPoints(videoPath string, tip Rect, partners []Partner, opts Options) (Result, error) {
	if len(partners) == 0 {
		return Result{}, &trackError{"TrackMultiPoints requires at least one partner"}
	}
	if len(partners) == 1 {
		o := opts
		o.FixedB = partners[0].Fixed
		return TrackTwoPoints(videoPath, tip, partners[0].ROI, o)
	}

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

	tipTracker := NewTracker()
	tipTracker.Init(cap, tip)
	boxes := make([]Rect, len(partners))
	trackers := make([]*Tracker, len(partners))
	include0 := make([]bool, len(partners))
	for i, p := range partners {
		boxes[i] = p.ROI
		include0[i] = true // first frame: all partners at marked boxes
		if !p.Fixed {
			tr := NewTracker()
			tr.Init(cap, p.ROI)
			trackers[i] = tr
		}
	}
	defer func() {
		tipTracker.Close()
		for _, tr := range trackers {
			if tr != nil {
				tr.Close()
			}
		}
	}()

	tipBox := tip
	d0, _ := FuseTipPartners(tipBox, true, boxes, include0)
	timestamps := []int{0}
	distances := []float64{d0}
	lostFlags := []bool{false}
	lost, valid := 0, 1
	idx := 1
	lastDist := d0

	total := int(cap.Get(CapPropFrameCount))
	if opts.MaxFrames > 0 && (total == 0 || opts.MaxFrames < total) {
		total = opts.MaxFrames
	}
	prog := newProgressReporter(opts.OnProgress, total)
	prog.report(0)

	include := make([]bool, len(partners))
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
		newTip, okTip := tipTracker.Update(cap)
		if okTip {
			tipBox = newTip
		}
		for i := range partners {
			if trackers[i] == nil {
				include[i] = true // fixed
				continue
			}
			newB, okB := trackers[i].Update(cap)
			if okB {
				boxes[i] = newB
				include[i] = true
			} else {
				include[i] = false // exclude stale box from min()
			}
		}
		dist, fused := FuseTipPartners(tipBox, okTip, boxes, include)
		frameLost := !fused
		if frameLost {
			lost++
			dist = lastDist
		} else {
			valid++
			lastDist = dist
		}
		distances = append(distances, dist)
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
