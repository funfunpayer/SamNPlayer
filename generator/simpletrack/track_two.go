package simpletrack

import (
	"context"
	"fmt"
	"math"

	"github.com/funfunpayer/SamNPlayer/videox"
)

// TrackTwoPoints follows two ROIs and returns their 2D center distance as
// the position signal — Go port of generate_funscript.track_two_points.
// Shared pan cancels in the distance; lost flags go into Stats for
// tracking_gaps (contact vibration mute).
func TrackTwoPoints(ctx context.Context, videoPath string, roiA, roiB Rect, opts Options) (Result, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	if roiA.W < 4 || roiA.H < 4 || roiB.W < 4 || roiB.H < 4 {
		return Result{}, fmt.Errorf("simpletrack: ROI too small for two-point")
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
	if srcW > 0 && srcH > 0 && (srcW != w || srcH != h) {
		sx := float64(w) / float64(srcW)
		sy := float64(h) / float64(srcH)
		roiA = scaleRect(roiA, sx, sy)
		roiB = scaleRect(roiB, sx, sy)
	}
	boxA := clampRect(roiA, w, h)
	boxB := clampRect(roiB, w, h)
	tmplA := extract(frame.Pixels, w, h, boxA)
	tmplB := extract(frame.Pixels, w, h, boxB)
	if len(tmplA) == 0 || len(tmplB) == 0 {
		return Result{}, fmt.Errorf("simpletrack: empty two-point template")
	}

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
	frameIdx := 1

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
		bestA, scoreA, okA := searchNCC(frame.Pixels, w, h, tmplA, boxA, margin, step)
		bestB, scoreB, okB := searchNCC(frame.Pixels, w, h, tmplB, boxB, margin, step)
		okA = okA && scoreA >= 0.35
		okB = okB && scoreB >= 0.35
		if okA {
			boxA = bestA
			if frameIdx%15 == 0 {
				tmplA = extract(frame.Pixels, w, h, boxA)
			}
		}
		if okB {
			boxB = bestB
			if frameIdx%15 == 0 {
				tmplB = extract(frame.Pixels, w, h, boxB)
			}
		}
		frameLost := !(okA && okB)
		if frameLost {
			lost++
		} else {
			valid++
		}
		distances = append(distances, dist(boxA, boxB))
		timestamps = append(timestamps, int(float64(frameIdx)*1000.0/fps))
		lostFlags = append(lostFlags, frameLost)
		frameIdx++
		if opts.MaxFrames > 0 && frameIdx >= opts.MaxFrames {
			break
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

func scaleRect(r Rect, sx, sy float64) Rect {
	out := Rect{
		X: int(float64(r.X) * sx),
		Y: int(float64(r.Y) * sy),
		W: int(float64(r.W) * sx),
		H: int(float64(r.H) * sy),
	}
	if out.W < 4 {
		out.W = 4
	}
	if out.H < 4 {
		out.H = 4
	}
	return out
}
