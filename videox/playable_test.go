package videox

import "testing"

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
