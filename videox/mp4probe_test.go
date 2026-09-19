package videox

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestProbeISOBMFFLean(t *testing.T) {
	path := makeTestVideo(t)
	info, err := probeISOBMFF(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Width != 320 || info.Height != 240 {
		t.Fatalf("geometry %dx%d", info.Width, info.Height)
	}
	if info.Duration < 1500*time.Millisecond || info.Duration > 2500*time.Millisecond {
		t.Fatalf("duration %v", info.Duration)
	}
	if info.Codec != "h264" {
		t.Fatalf("codec %q", info.Codec)
	}
}

func TestProbeFallsBackWithoutFFprobe(t *testing.T) {
	path := makeTestVideo(t)
	ResetToolCache()
	t.Setenv("SAMNPLAYER_FFPROBE", filepath.Join(t.TempDir(), "missing-ffprobe"))
	t.Setenv("SAMNPLAYER_FFMPEG", os.Getenv("SAMNPLAYER_FFMPEG"))
	// Force ffprobe miss; ffmpeg may still exist on PATH for other tests.
	info, err := Probe(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Width != 320 || info.Height != 240 {
		t.Fatalf("fallback geometry %dx%d", info.Width, info.Height)
	}
	ResetToolCache()
}

func TestProbeISOBMFFRejectsWebM(t *testing.T) {
	if _, err := probeISOBMFF("clip.webm"); err == nil {
		t.Fatal("expected unsupported extension")
	}
}
