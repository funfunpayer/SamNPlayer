package generator

import (
	"context"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestApplyStrokePreviewSynthetic(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}
	dir := t.TempDir()
	video := filepath.Join(dir, "bounce.mp4")
	args := []string{
		"-v", "error", "-y",
		"-f", "lavfi", "-i", "color=c=black:s=320x240:d=3:r=24",
		"-f", "lavfi", "-i", "color=c=white:s=40x40:d=3:r=24",
		"-filter_complex", "[0][1]overlay=x=140:y='100+80*sin(2*PI*t)'",
		"-pix_fmt", "yuv420p", video,
	}
	if out, err := exec.Command("ffmpeg", args...).CombinedOutput(); err != nil {
		t.Fatalf("ffmpeg: %v: %s", err, out)
	}
	var lines []string
	opts := applyStrokePreview(context.Background(), video, Options{}, func(s string) {
		lines = append(lines, s)
	})
	if opts.StrokePreviewHint == nil {
		t.Fatalf("expected hint, lines=%v", lines)
	}
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "STROKE_PREVIEW") {
		t.Fatalf("no progress lines: %s", joined)
	}
	if opts.MinPeakDistanceMs == 0 && opts.StrokePreviewHint["quality"] == "ok" {
		// Peak bias is advisory-only now — still expect a tip in the hint map.
		if _, ok := opts.StrokePreviewHint["suggested_min_peak_distance_ms"]; !ok {
			t.Fatal("ok preview should suggest peak distance in hint metadata")
		}
	}
}

func TestApplyStrokePreviewSkip(t *testing.T) {
	opts := applyStrokePreview(context.Background(), "/nope.mp4", Options{SkipStrokePreview: true}, nil)
	if opts.StrokePreviewHint != nil {
		t.Fatal("skip must not run")
	}
}
