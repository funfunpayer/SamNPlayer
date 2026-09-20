package simpletrack

import (
	"context"
	"fmt"

	"github.com/funfunpayer/SamNPlayer/videox"
)

// Partner is one Tf/Tj contact anchor (ROI2 or extra --target).
type Partner struct {
	ROI   Rect
	Fixed bool
}

// TrackMultiPoints follows tip + N partners; Positions = min tipPartnerDistance.
func TrackMultiPoints(ctx context.Context, videoPath string, tip Rect, partners []Partner, opts Options) (Result, error) {
	if len(partners) == 0 {
		return Result{}, fmt.Errorf("simpletrack: TrackMultiPoints requires at least one partner")
	}
	if len(partners) == 1 {
		o := opts
		o.FixedB = partners[0].Fixed
		return TrackTwoPoints(ctx, videoPath, tip, partners[0].ROI, o)
	}
	return trackMultiNCC(ctx, videoPath, tip, partners, opts)
}

func trackMultiNCC(ctx context.Context, videoPath string, tip Rect, partners []Partner, opts Options) (Result, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	if tip.W < 4 || tip.H < 4 {
		return Result{}, fmt.Errorf("simpletrack: tip ROI too small")
	}
	for _, p := range partners {
		if p.ROI.W < 4 || p.ROI.H < 4 {
			return Result{}, fmt.Errorf("simpletrack: partner ROI too small")
		}
	}
	margin := opts.SearchMargin
	if margin <= 0 {
		margin = 32
	}
	step := opts.Step
	if step <= 0 {
		step = 2
	}

	info, err := videox.Probe(ctx, videoPath)
	if err != nil {
		return Result{}, fmt.Errorf("simpletrack: probe: %w", err)
	}
	fps := info.FPS
	if fps <= 0 {
		fps = 25
	}

	r, err := videox.NewGrayReader(ctx, videoPath, info, videox.GrayReaderOptions{
		FPS:        fps,
		MaxWidth:   640,
		AutoRotate: true,
		StartSec:   opts.StartTimeSec,
	})
	if err != nil {
		return Result{}, fmt.Errorf("simpletrack: open: %w", err)
	}
	defer r.Close()

	frame, err := r.Next(0)
	if err != nil {
		return Result{}, fmt.Errorf("simpletrack: first frame: %w", err)
	}
	w, h := frame.Width, frame.Height
	srcW, srcH := info.Rotated()
	sx, sy := 1.0, 1.0
	if srcW > 0 && srcH > 0 && (srcW != w || srcH != h) {
		sx = float64(w) / float64(srcW)
		sy = float64(h) / float64(srcH)
		tip = scaleRect(tip, sx, sy)
		for i := range partners {
			partners[i].ROI = scaleRect(partners[i].ROI, sx, sy)
		}
	}

	tipBox := clampRect(tip, w, h)
	tmplTip := extract(frame.Pixels, w, h, tipBox)
	if len(tmplTip) == 0 {
		return Result{}, fmt.Errorf("simpletrack: empty tip template")
	}
	boxes := make([]Rect, len(partners))
	tmpls := make([][]byte, len(partners))
	for i, p := range partners {
		boxes[i] = clampRect(p.ROI, w, h)
		if !p.Fixed {
			tmpls[i] = extract(frame.Pixels, w, h, boxes[i])
			if len(tmpls[i]) == 0 {
				return Result{}, fmt.Errorf("simpletrack: empty partner template")
			}
		}
	}

	minDist := func(tipB Rect, ps []Rect) float64 {
		best := tipPartnerDistance(tipB, ps[0])
		for i := 1; i < len(ps); i++ {
			d := tipPartnerDistance(tipB, ps[i])
			if d < best {
				best = d
			}
		}
		return best
	}

	timestamps := []int{0}
	distances := []float64{minDist(tipBox, boxes)}
	lostFlags := []bool{false}
	lost, valid := 0, 1
	frameIdx := 1

	totalFrames := 0
	if info.Duration > 0 && fps > 0 {
		totalFrames = int(info.Duration.Seconds()*fps + 0.5)
	}
	if opts.MaxFrames > 0 && (totalFrames == 0 || opts.MaxFrames < totalFrames) {
		totalFrames = opts.MaxFrames
	}
	prog := newProgressReporter(opts.OnProgress, totalFrames)
	prog.report(0)

	for {
		if opts.Cancel != nil && opts.Cancel() {
			return Result{Canceled: true}, ErrCanceled
		}
		if err := ctx.Err(); err != nil {
			return Result{Canceled: true}, err
		}
		frame, err = r.Next(frameIdx)
		if err != nil {
			break
		}
		bestTip, scoreTip, okTip := searchNCC(frame.Pixels, w, h, tmplTip, tipBox, margin, step)
		okTip = okTip && scoreTip >= 0.35
		if okTip {
			tipBox = bestTip
			if frameIdx%15 == 0 {
				tmplTip = extract(frame.Pixels, w, h, tipBox)
			}
		}
		frameOK := okTip
		for i, p := range partners {
			if p.Fixed {
				continue
			}
			best, score, found := searchNCC(frame.Pixels, w, h, tmpls[i], boxes[i], margin, step)
			ok := found && score >= 0.35
			if ok {
				boxes[i] = best
				if frameIdx%15 == 0 {
					tmpls[i] = extract(frame.Pixels, w, h, boxes[i])
				}
			} else {
				frameOK = false
			}
		}
		frameLost := !frameOK
		if frameLost {
			lost++
		} else {
			valid++
		}
		distances = append(distances, minDist(tipBox, boxes))
		timestamps = append(timestamps, int(float64(frameIdx)*1000.0/fps))
		lostFlags = append(lostFlags, frameLost)
		prog.report(frameIdx)
		frameIdx++
		if opts.MaxFrames > 0 && frameIdx >= opts.MaxFrames {
			break
		}
	}
	prog.report(frameIdx)

	if opts.StartTimeSec > 0 {
		off := int(opts.StartTimeSec*1000 + 0.5)
		for i := range timestamps {
			timestamps[i] += off
		}
	}

	conf := 0.0
	if frameIdx > 0 {
		conf = float64(valid) / float64(frameIdx)
	}
	reason := ""
	if frameIdx > 0 && float64(lost)/float64(frameIdx) > 0.5 {
		reason = "tracker_lost_heavy"
	} else if frameIdx > 0 && float64(lost)/float64(frameIdx) > 0.15 {
		reason = "tracker_lost_elevated"
	}

	return Result{
		TimestampsMs: timestamps,
		Positions:    distances,
		Width:        w,
		Height:       h,
		LostFlags:    lostFlags,
		Stats: Stats{
			TrackerLostFrames: lost,
			TotalFrames:       frameIdx,
			VerticalRange:     ptp(distances),
			ValidFrames:       valid,
			Confidence:        conf,
			Reason:            reason,
		},
	}, nil
}
