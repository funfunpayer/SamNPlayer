package generator

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"math/rand"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/funfunpayer/SamNPlayer/videox"
)

// synthBeatVideo draws w×h frames: random "film" noise everywhere, and a
// strip (rows y0..y1) with a static target line at targetX and dots that
// scroll at speed px/frame (negative = leftward), spawned every period
// frames, vanishing once they pass the target. It returns the true beat
// times in frames.
func synthBeatVideo(n, w, h, y0, y1, targetX int, speed float64, period int, seed int64) ([]GrayFrame, []float64) {
	rng := rand.New(rand.NewSource(seed))
	frames := make([]GrayFrame, n)
	type dot struct{ x float64 }
	var dots []dot
	var beats []float64
	start := float64(w - 4)
	if speed > 0 {
		start = 3
	}
	for i := 0; i < n; i++ {
		f := GrayFrame{W: w, H: h, Pix: make([]byte, w*h)}
		for k := range f.Pix {
			f.Pix[k] = byte(rng.Intn(120))
		}
		if i%period == 0 {
			dots = append(dots, dot{start})
			// It reaches the target after |start-target|/|speed| frames.
			beats = append(beats, float64(i)+math.Abs(start-float64(targetX))/math.Abs(speed))
		}
		// Strip: dark background, target line, dots.
		for y := y0; y < y1; y++ {
			for x := 0; x < w; x++ {
				f.Pix[y*w+x] = 10
			}
			f.Pix[y*w+targetX] = 200
		}
		keep := dots[:0]
		for _, d := range dots {
			passed := (speed < 0 && d.x < float64(targetX)) || (speed > 0 && d.x > float64(targetX))
			if passed {
				continue
			}
			cx := int(math.Round(d.x))
			for y := y0; y < y1; y++ {
				for x := cx - 2; x <= cx+2; x++ {
					if x >= 0 && x < w {
						f.Pix[y*w+x] = 250
					}
				}
			}
			d.x += speed
			keep = append(keep, d)
		}
		dots = keep
		frames[i] = f
	}
	var real []float64
	for _, b := range beats {
		if b < float64(n-1) {
			real = append(real, b)
		}
	}
	return frames, real
}

func TestFindBeatBarAndBeats(t *testing.T) {
	for _, tc := range []struct {
		name    string
		speed   float64
		targetX int
	}{
		{"leftward to the left third", -8, 60},
		{"rightward to the middle", 6, 160},
	} {
		t.Run(tc.name, func(t *testing.T) {
			const w, h = 320, 192 // bands of 8 rows
			frames, truth := synthBeatVideo(300, w, h, 168, 176, tc.targetX, tc.speed, 17, 1)
			bar, ok := FindBeatBar(frames)
			if !ok {
				t.Fatalf("strip not found: %+v", bar)
			}
			if bar.Y0 != 168 || bar.Y1 != 176 {
				t.Fatalf("wrong band %d–%d", bar.Y0, bar.Y1)
			}
			if math.Abs(bar.Velocity-tc.speed) > 0.6 {
				t.Fatalf("velocity %v, want %v", bar.Velocity, tc.speed)
			}
			if math.Abs(bar.TargetX-float64(tc.targetX)) > 4 {
				t.Fatalf("target %v, want %v", bar.TargetX, tc.targetX)
			}
			beats := BeatTimes(frames, bar, 30)
			frameMs := 1000.0 / 30
			// Every true beat found within 1.5 frames, and no extra beats.
			if len(beats) < len(truth)-1 || len(beats) > len(truth)+1 {
				t.Fatalf("%d beats, want %d: %v", len(beats), len(truth), beats)
			}
			for _, tb := range truth {
				best := math.Inf(1)
				for _, b := range beats {
					best = math.Min(best, math.Abs(b-tb*frameMs))
				}
				if best > 1.5*frameMs {
					t.Fatalf("beat at %.0f ms missed (closest %.0f ms off)", tb*frameMs, best)
				}
			}
		})
	}
}

func TestFindBeatBarNoneInPlainFilm(t *testing.T) {
	rng := rand.New(rand.NewSource(2))
	frames := make([]GrayFrame, 120)
	for i := range frames {
		f := GrayFrame{W: 160, H: 96, Pix: make([]byte, 160*96)}
		for k := range f.Pix {
			f.Pix[k] = byte(rng.Intn(200))
		}
		frames[i] = f
	}
	if bar, ok := FindBeatBar(frames); ok {
		t.Fatalf("noise taken for a beat bar: %+v", bar)
	}
}

// End to end through ffmpeg: encode a synthetic beat-bar clip at twice the
// analysis size, then detect it from the file.
func TestDetectBeatBarBeatsFromVideo(t *testing.T) {
	for _, vertical := range []bool{false, true} {
		t.Run(fmt.Sprintf("vertical=%v", vertical), func(t *testing.T) { testDetectBeatBarFromVideo(t, vertical) })
	}
}

// rotateCCW turns a frame a quarter counter-clockwise (a bottom strip ends
// up on the right edge, running up and down).
func rotateCCW(f GrayFrame) GrayFrame {
	r := GrayFrame{W: f.H, H: f.W, Pix: make([]byte, len(f.Pix))}
	for y := 0; y < r.H; y++ {
		for x := 0; x < r.W; x++ {
			r.Pix[y*r.W+x] = f.Pix[x*f.W+(f.W-1-y)]
		}
	}
	return r
}

func testDetectBeatBarFromVideo(t *testing.T, vertical bool) {
	bin, err := videox.FFmpeg()
	if err != nil {
		t.Skip("ffmpeg not available")
	}
	const n = 300
	w, h := 640, 384
	frames, truth := synthBeatVideo(n, w, h, 336, 352, 120, -16, 17, 3)
	if vertical {
		for i := range frames {
			frames[i] = rotateCCW(frames[i])
		}
		w, h = h, w
	}
	path := filepath.Join(t.TempDir(), "bar.mp4")
	cmd := exec.Command(bin, "-loglevel", "error", "-f", "rawvideo", "-pix_fmt", "gray",
		"-s", fmt.Sprintf("%dx%d", w, h), "-r", "30", "-i", "-",
		"-c:v", "mpeg4", "-q:v", "3", "-pix_fmt", "yuv420p", path)
	var raw bytes.Buffer
	for _, f := range frames {
		raw.Write(f.Pix)
	}
	cmd.Stdin = &raw
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("encode: %v %s", err, out)
	}
	res, err := DetectBeatBarBeats(context.Background(), path, 10)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Found || res.Vertical != vertical {
		t.Fatalf("beat bar not found as vertical=%v: %+v", vertical, res)
	}
	frameMs := 1000.0 / 30
	if len(res.BeatsMs) < len(truth)-1 || len(res.BeatsMs) > len(truth)+1 {
		t.Fatalf("%d beats, want %d: %v", len(res.BeatsMs), len(truth), res.BeatsMs)
	}
	for _, tb := range truth {
		best := math.Inf(1)
		for _, b := range res.BeatsMs {
			best = math.Min(best, math.Abs(b-tb*frameMs))
		}
		if best > 2*frameMs {
			t.Fatalf("beat at %.0f ms missed (closest %.0f ms off)", tb*frameMs, best)
		}
	}
	// 17 frames at 30 fps ≈ 106 BPM.
	if res.BPM < 100 || res.BPM > 112 {
		t.Fatalf("bpm %v", res.BPM)
	}
}
