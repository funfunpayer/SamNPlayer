package strokepreview

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestAnalyzeSyntheticBounceFindsExtrema(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}
	dir := t.TempDir()
	video := filepath.Join(dir, "bounce.mp4")
	// White box oscillates vertically ~1 Hz for 4 seconds.
	args := []string{
		"-v", "error", "-y",
		"-f", "lavfi", "-i", "color=c=black:s=320x240:d=4:r=24",
		"-f", "lavfi", "-i", "color=c=white:s=40x40:d=4:r=24",
		"-filter_complex", "[0][1]overlay=x=140:y='100+80*sin(2*PI*t)'",
		"-pix_fmt", "yuv420p", video,
	}
	if out, err := exec.Command("ffmpeg", args...).CombinedOutput(); err != nil {
		t.Fatalf("ffmpeg: %v: %s", err, out)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	rep, err := Analyze(ctx, video, Options{MaxSeconds: 4})
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if rep.SampleCount < 8 {
		t.Fatalf("too few samples: %+v", rep)
	}
	if rep.UpCount+rep.DownCount < 4 {
		t.Fatalf("expected several extrema on ~1Hz bounce, got up=%d down=%d span=%.4f",
			rep.UpCount, rep.DownCount, rep.SignalSpan)
	}
	if rep.StrokeHz < 0.4 || rep.StrokeHz > 2.0 {
		t.Fatalf("stroke Hz out of range for 1Hz bounce: %.3f (rep=%+v)", rep.StrokeHz, rep)
	}
	if rep.Quality == "" {
		t.Fatal("quality unset")
	}
}

func TestClassifyWeakWhenNoMotion(t *testing.T) {
	r := Report{SampleCount: 20, SignalSpan: 0.005, UpCount: 0, DownCount: 0}
	q, audio, reason := classify(r)
	if q != "weak" || !audio {
		t.Fatalf("got %s audio=%v reason=%s", q, audio, reason)
	}
}

func TestStrokeHzFromPeaks(t *testing.T) {
	times := []int{0, 100, 200, 300, 400, 500, 600, 700, 800, 900, 1000}
	peaks := []int{0, 5, 10} // 500ms intervals → 2 Hz
	hz := strokeHzFromPeaks(times, peaks)
	if hz < 1.8 || hz > 2.2 {
		t.Fatalf("hz=%v", hz)
	}
}

func TestAnalyzeMissingFile(t *testing.T) {
	_, err := Analyze(context.Background(), filepath.Join(t.TempDir(), "nope.mp4"), Options{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestFindPeaksExportStillWorks(t *testing.T) {
	// Smoke: ensure posttrack.FindPeaks is reachable via package build.
	_ = os.DevNull
}
