package videox

import (
	"context"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestInfoLikelyPlayable(t *testing.T) {
	cases := []struct {
		codec string
		want  bool
	}{
		{"h264", true},
		{"H264", true},
		{"avc1", true},
		{"vp9", true},
		{"av1", true},
		{"mpeg4", false},
		{"hevc", false},
		{"h265", false},
		{"", false},
	}
	for _, c := range cases {
		got := Info{Codec: c.codec}.LikelyPlayable()
		if got != c.want {
			t.Errorf("codec %q: got %v want %v", c.codec, got, c.want)
		}
	}
}

func TestProxyPath(t *testing.T) {
	p := ProxyPath(`/videos/clip.mkv`)
	if p == `/videos/clip.mkv` {
		t.Fatal("proxy path must differ from source")
	}
	if !contains(p, "samnplayer-h264") {
		t.Fatalf("unexpected proxy path %q", p)
	}
}

func TestIsH264Family(t *testing.T) {
	if !isH264Family("h264") || !isH264Family("avc1") {
		t.Fatal("expected h264 family")
	}
	if isH264Family("hevc") {
		t.Fatal("hevc is not h264")
	}
}

func TestEnsurePlayableProxyRemuxMKV(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}
	dir := t.TempDir()
	mp4 := filepath.Join(dir, "src.mp4")
	mkv := filepath.Join(dir, "src.mkv")
	// Create H.264 MP4 then remux to MKV (awkward container for webview).
	cmd := exec.Command("ffmpeg", "-v", "error", "-y",
		"-f", "lavfi", "-i", "testsrc=size=160x120:rate=10:duration=1",
		"-c:v", "libx264", "-pix_fmt", "yuv420p", mp4)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("make mp4: %v: %s", err, out)
	}
	cmd = exec.Command("ffmpeg", "-v", "error", "-y", "-i", mp4, "-c", "copy", mkv)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("make mkv: %v: %s", err, out)
	}
	out, converted, err := EnsurePlayableProxy(context.Background(), mkv, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !converted {
		t.Fatal("expected conversion/remux for mkv")
	}
	if out == mkv {
		t.Fatal("proxy path should differ")
	}
	info, err := Probe(context.Background(), out)
	if err != nil {
		t.Fatal(err)
	}
	if !info.LikelyPlayable() {
		t.Fatalf("proxy not playable: %+v", info)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		(func() bool {
			for i := 0; i+len(sub) <= len(s); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
			return false
		})())
}
