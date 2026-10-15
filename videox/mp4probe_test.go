package videox

import (
	"context"
	"encoding/binary"
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
	path := filepath.Join(t.TempDir(), "unknown.mp4")
	if err := writeStubUnknownCodecMP4(path); err != nil {
		t.Fatal(err)
	}
	info, err := probeISOBMFF(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Codec == "h264" || info.PixFmt == "yuv420p" {
		t.Fatalf("unknown codec defaulted to h264/yuv420p: %+v", info)
	}
	if info.LikelyPlayable() {
		t.Fatalf("unknown codec treated as playable: %+v", info)
	}
	if info.Width != 320 || info.Height != 240 {
		t.Fatalf("geometry %dx%d, want 320x240", info.Width, info.Height)
	}
	if info.Codec == "" {
		t.Log("codec empty (acceptable for unknown)")
	} else {
		t.Logf("codec reported as %q (not h264, good)", info.Codec)
	}
}

func writeStubUnknownCodecMP4(path string) error {
	var file []byte

	writeBox := func(dst *[]byte, typ string, payload []byte) {
		size := uint32(8 + len(payload))
		hdr := make([]byte, 4)
		binary.BigEndian.PutUint32(hdr, size)
		*dst = append(*dst, hdr...)
		*dst = append(*dst, typ...)
		*dst = append(*dst, payload...)
	}

	// ftyp
	writeBox(&file, "ftyp", []byte("isom\x00\x00\x02\x00isomiso2mp41"))

	// moov payload
	var moov []byte
	mvhd := make([]byte, 100)
	binary.BigEndian.PutUint32(mvhd[12:16], 1000)
	binary.BigEndian.PutUint32(mvhd[16:20], 2000)
	writeBox(&moov, "mvhd", mvhd)

	// trak
	var trak []byte
	tkhd := make([]byte, 84)
	binary.BigEndian.PutUint32(tkhd[76:80], 320<<16)
	binary.BigEndian.PutUint32(tkhd[80:84], 240<<16)
	writeBox(&trak, "tkhd", tkhd)

	// mdia
	var mdia []byte
	mdhd := make([]byte, 24)
	binary.BigEndian.PutUint32(mdhd[12:16], 1000)
	binary.BigEndian.PutUint32(mdhd[16:20], 2000)
	writeBox(&mdia, "mdhd", mdhd)

	// stsd with unknown "xxxx"
	var stsd []byte
	stsd = append(stsd, 0, 0, 0, 0) // version + flags
	stsd = append(stsd, 0, 0, 0, 1) // entry_count = 1
	entry := make([]byte, 40)
	binary.BigEndian.PutUint32(entry[0:4], uint32(len(entry)))
	copy(entry[4:8], "xxxx")
	binary.BigEndian.PutUint16(entry[32:34], 320)
	binary.BigEndian.PutUint16(entry[34:36], 240)
	stsd = append(stsd, entry...)

	var stbl []byte
	writeBox(&stbl, "stsd", stsd)

	var minf []byte
	writeBox(&minf, "stbl", stbl)
	writeBox(&mdia, "minf", minf)
	writeBox(&trak, "mdia", mdia)
	writeBox(&moov, "trak", trak)
	writeBox(&file, "moov", moov)

	return os.WriteFile(path, file, 0644)
}
