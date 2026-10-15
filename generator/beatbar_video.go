package generator

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"math"
	"time"

	"github.com/funfunpayer/SamNPlayer/videox"
)

// Beat bar from a video file (beatbar.go has the detection): ffmpeg decodes
// small grey frames. A first pass over a window finds the strip; a second
// pass over the whole video reads only that strip and times every beat.

const (
	beatBarFPS     = 30
	beatBarW       = 320
	beatBarH       = 192 // 24 bands of 8 rows; the aspect does not matter here
	beatBarFindSec = 90
)

// BeatBarResult is what DetectBeatBarBeats found.
type BeatBarResult struct {
	Found    bool      `json:"found"`
	Bar      BeatBar   `json:"bar"`
	Vertical bool      `json:"vertical"` // the strip runs up/down (at a side edge)
	BeatsMs  []float64 `json:"beatsMs"`
	BPM      float64   `json:"bpm"` // median beat rate, display only
}

// DetectBeatBarBeats looks for a beat bar in videoPath and returns its beat
// times. Found is false (no error) when the video has no beat bar.
func DetectBeatBarBeats(ctx context.Context, videoPath string, durationSec float64) (BeatBarResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	var res BeatBarResult
	// The bar may start after an intro: try the start, then the middle.
	starts := []float64{0}
	if durationSec > 3*beatBarFindSec {
		starts = append(starts, durationSec*0.3, durationSec*0.6)
	}
	// Most bars run across the bottom or top; some run down a side edge —
	// those are read from the picture turned a quarter.
search:
	for _, vertical := range []bool{false, true} {
		for _, ss := range starts {
			frames, err := decodeGrayFrames(ctx, videoPath, vertical, ss, beatBarFindSec, 0, beatBarH)
			if err != nil {
				return res, err
			}
			if bar, ok := FindBeatBar(frames); ok {
				res.Found, res.Bar, res.Vertical = true, bar, vertical
				break search
			}
		}
	}
	if !res.Found {
		return res, nil
	}
	beats, err := streamBeatTimes(ctx, videoPath, res.Vertical, res.Bar)
	if err != nil {
		return res, err
	}
	res.BeatsMs = beats
	res.BPM = medianBPM(beats)
	return res, nil
}

// decodeGrayFrames reads frames [ss, ss+dur) (dur 0 = to the end) scaled to
// beatBarW × beatBarH, rows [y0, y1) of them; vertical turns the picture a
// quarter first (clockwise: a side strip becomes a bottom or top one).
func decodeGrayFrames(ctx context.Context, videoPath string, vertical bool, ss, dur float64, y0, y1 int) ([]GrayFrame, error) {
	var frames []GrayFrame
	err := streamGrayFrames(ctx, videoPath, vertical, ss, dur, y0, y1, func(f GrayFrame) { frames = append(frames, f) })
	return frames, err
}

func streamGrayFrames(ctx context.Context, videoPath string, vertical bool, ss, dur float64, y0, y1 int, fn func(GrayFrame)) error {
	h := y1 - y0
	turn := ""
	if vertical {
		turn = "transpose=clock,"
	}
	vf := fmt.Sprintf("fps=%d,%sscale=%d:%d,crop=%d:%d:0:%d,format=gray", beatBarFPS, turn, beatBarW, beatBarH, beatBarW, h, y0)
	args := []string{}
	if ss > 0 {
		args = append(args, "-ss", fmt.Sprintf("%.3f", ss))
	}
	args = append(args, "-i", videoPath)
	if dur > 0 {
		args = append(args, "-t", fmt.Sprintf("%.3f", dur))
	}
	args = append(args, "-an", "-vf", vf, "-f", "rawvideo", "-pix_fmt", "gray", "-loglevel", "error", "-")
	cmd, err := videox.CommandContext(ctx, args...)
	if err != nil {
		return fmt.Errorf("ffmpeg ist nicht installiert")
	}
	cmd.WaitDelay = 5 * time.Second
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("ffmpeg start: %w", err)
	}
	r := bufio.NewReaderSize(stdout, 1<<20)
	size := beatBarW * h
	var readErr error
	for {
		buf := make([]byte, size)
		if _, err := io.ReadFull(r, buf); err != nil {
			if err != io.EOF && err != io.ErrUnexpectedEOF {
				readErr = err
			}
			break
		}
		fn(GrayFrame{W: beatBarW, H: h, Pix: buf})
	}
	waitErr := cmd.Wait()
	if ctxErr := ctx.Err(); ctxErr != nil {
		return ctxErr
	}
	if readErr != nil {
		return readErr
	}
	if waitErr != nil {
		return fmt.Errorf("ffmpeg: %w", waitErr)
	}
	return nil
}

// streamBeatTimes times every beat over the whole video, reading only the
// strip's rows (memory stays small for long videos).
func streamBeatTimes(ctx context.Context, videoPath string, vertical bool, bar BeatBar) ([]float64, error) {
	tr := newBeatTracker(bar, beatBarFPS)
	err := streamGrayFrames(ctx, videoPath, vertical, 0, 0, bar.Y0, bar.Y1, func(f GrayFrame) {
		tr.add(bandProfile(f, 0, f.H))
	})
	return tr.beats, err
}

// medianBPM: 60 000 / median beat interval; 0 with fewer than 3 beats.
func medianBPM(beats []float64) float64 {
	if len(beats) < 3 {
		return 0
	}
	iv := make([]float64, 0, len(beats)-1)
	for i := 1; i < len(beats); i++ {
		iv = append(iv, beats[i]-beats[i-1])
	}
	m := medianFloat(iv)
	if m <= 0 {
		return 0
	}
	return math.Round(60000 / m)
}
