//go:build cgo && opencv

// profile_generate runs the Everyday CSRT generator on one video with a CPU
// profile, for the migration's measurement step (docs/MIGRATION_CPP_PYTHON.md
// M0). Each OpenCV call shows up in the profile as its own _Cfunc_… entry,
// so the profile splits decode, tracking and camera compensation.
//
//	go run -tags opencv ./scripts/profile_generate -video clip.mp4 -roi 400,300,120,120 -cpuprofile cpu.out
//	go tool pprof -top -nodecount=40 cpu.out
//
// With -synth it first writes a synthetic HD test clip (moving bright box on
// a noisy test pattern; no real footage needed) and profiles that.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/pprof"
	"strconv"
	"strings"
	"time"

	"github.com/funfunpayer/SamNPlayer/generator"
)

func main() {
	video := flag.String("video", "", "video file")
	roiFlag := flag.String("roi", "", "tip box x,y,w,h in video pixels")
	prof := flag.String("cpuprofile", "cpu.out", "CPU profile output")
	synth := flag.String("synth", "", "write a synthetic clip here first (WxH@fps:seconds, e.g. 1920x1080@30:60)")
	rhythm := flag.Bool("rhythm", false, "turn on the rhythm grid (optical flow cells)")
	flag.Parse()

	if *synth != "" {
		path, roi, err := writeSynth(*synth)
		if err != nil {
			fail(err)
		}
		*video, *roiFlag = path, roi
		fmt.Println("synthetic clip:", path, "roi", roi)
	}
	if *video == "" || *roiFlag == "" {
		fail(fmt.Errorf("need -video and -roi (or -synth)"))
	}
	roi, err := parseROI(*roiFlag)
	if err != nil {
		fail(err)
	}
	f, err := os.Create(*prof)
	if err != nil {
		fail(err)
	}
	defer f.Close()
	out := filepath.Join(os.TempDir(), "profile_generate.funscript")
	opts := generator.Options{
		Backend:             "csrt",
		NativePipeline:      true,
		SmoothWindow:        11,
		MinPeakDistanceMs:   150,
		MinActionIntervalMs: 100,
		NormPercentile:      2,
		RhythmGrid:          *rhythm,
	}
	if err := pprof.StartCPUProfile(f); err != nil {
		fail(err)
	}
	start := time.Now()
	err = generator.GenerateNativeCSRT(context.Background(), *video, roi, out, opts,
		func(line string) { fmt.Printf("[%6.1fs] %s\n", time.Since(start).Seconds(), line) }, nil)
	pprof.StopCPUProfile()
	if err != nil {
		fail(err)
	}
	fmt.Printf("total %.2fs, profile in %s\n", time.Since(start).Seconds(), *prof)
}

func parseROI(s string) (generator.ROI, error) {
	p := strings.Split(s, ",")
	if len(p) != 4 {
		return generator.ROI{}, fmt.Errorf("roi must be x,y,w,h")
	}
	var v [4]int
	for i := range p {
		n, err := strconv.Atoi(strings.TrimSpace(p[i]))
		if err != nil {
			return generator.ROI{}, err
		}
		v[i] = n
	}
	return generator.ROI{X: v[0], Y: v[1], W: v[2], H: v[3]}, nil
}

// writeSynth makes a test clip: testsrc2 pattern with noise and a bright box
// moving up and down at 1.5 Hz in the middle.
func writeSynth(spec string) (string, string, error) {
	var w, h, fps, sec int
	if _, err := fmt.Sscanf(spec, "%dx%d@%d:%d", &w, &h, &fps, &sec); err != nil {
		return "", "", fmt.Errorf("synth spec %q: %w", spec, err)
	}
	path := filepath.Join(os.TempDir(), fmt.Sprintf("synth_%dx%d_%d_%d.mp4", w, h, fps, sec))
	box := h / 8
	x, y0, amp := w/2-box/2, h/2-box/2, h/5
	// overlay (not drawbox) evaluates t per frame.
	fc := fmt.Sprintf("[0:v]noise=alls=12:allf=t[bg];[bg][1:v]overlay=x=%d:y='%d+%d*sin(2*PI*t*1.5)':eval=frame:shortest=1", x, y0, amp)
	cmd := exec.Command("ffmpeg", "-y", "-loglevel", "error",
		"-f", "lavfi", "-i", fmt.Sprintf("testsrc2=size=%dx%d:rate=%d:duration=%d", w, h, fps, sec),
		"-f", "lavfi", "-i", fmt.Sprintf("color=c=white:size=%dx%d:rate=%d:duration=%d", box, box, fps, sec),
		"-filter_complex", fc, "-c:v", "libx264", "-preset", "veryfast", "-pix_fmt", "yuv420p", path)
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", "", fmt.Errorf("ffmpeg: %v %s", err, out)
	}
	// The box at t=0 (sin 0 = 0), a little larger so CSRT has context.
	pad := box / 4
	return path, fmt.Sprintf("%d,%d,%d,%d", x-pad, y0-pad, box+2*pad, box+2*pad), nil
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "profile_generate:", err)
	os.Exit(1)
}
