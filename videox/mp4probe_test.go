package videox

import (
	"bytes"
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

func TestProbeISOBMFFMatchesFFprobeOnClip(t *testing.T) {
	// Quality gate: lean probe must not disagree with ffprobe on geometry /
	// duration for a normal H.264 MP4 (docs/SELF_BUILD.md — equal or reject).
	ResetToolCache()
	if _, err := FFprobe(); err != nil {
		t.Skip("ffprobe not available")
	}
	path := makeTestVideo(t)
	ref, err := probeWithFF(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	lean, err := probeISOBMFF(path)
	if err != nil {
		t.Fatal(err)
	}
	if lean.Width != ref.Width || lean.Height != ref.Height {
		t.Fatalf("geometry lean=%dx%d ffprobe=%dx%d", lean.Width, lean.Height, ref.Width, ref.Height)
	}
	if lean.Codec != "h264" && lean.Codec != ref.Codec {
		t.Fatalf("codec lean=%q ffprobe=%q", lean.Codec, ref.Codec)
	}
	delta := lean.Duration - ref.Duration
	if delta < 0 {
		delta = -delta
	}
	if delta > 250*time.Millisecond {
		t.Fatalf("duration lean=%v ffprobe=%v", lean.Duration, ref.Duration)
	}
}

func TestProbePrefersFFprobeWhenPresent(t *testing.T) {
	// Lean must not silently replace ffprobe: when ffprobe works, Probe uses it
	// (pix_fmt / fps filled — lean leaves fps empty on this fixture path).
	ResetToolCache()
	if _, err := FFprobe(); err != nil {
		t.Skip("ffprobe not available")
	}
	path := makeTestVideo(t)
	info, err := Probe(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	if info.PixFmt == "" || info.FPS < 24 {
		t.Fatalf("expected ffprobe-quality fields, got %+v", info)
	}
}

func TestProbeISOBMFFUnknownCodec(t *testing.T) {
	path := makeTestVideo(t)
	// Patch the last avc1/avc3 (sample-entry type inside stsd, not the ftyp brand)
	// so the lean probe cannot map it to a known codec (MEDIA-01: must not default to h264).
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	patched := false
	for _, old := range [][]byte{[]byte("avc1"), []byte("avc3")} {
		i := bytes.LastIndex(b, old)
		if i >= 0 {
			copy(b[i:], []byte("unkn"))
			patched = true
			break
		}
	}
	if !patched {
		t.Fatal("could not find avc1/avc3 sample entry to patch")
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		t.Fatal(err)
	}

	info, err := probeISOBMFF(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Codec == "h264" || info.Codec == "avc1" {
		t.Fatalf("unknown sample entry defaulted to known codec: %q", info.Codec)
	}
	if info.LikelyPlayable() {
		t.Fatalf("unknown codec %q treated as playable", info.Codec)
	}
	if info.Width != 320 || info.Height != 240 {
		t.Fatalf("geometry lost after patch: %dx%d", info.Width, info.Height)
	}
}
