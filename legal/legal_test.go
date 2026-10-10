package legal

import (
	"strings"
	"testing"
)

func TestEULAEmbedded(t *testing.T) {
	e := EULA()
	for _, want := range []string{"freeware", "not open", "private, non-commercial", EULAVersion, "Donations", "own risk"} {
		if !strings.Contains(e, want) {
			t.Errorf("EULA missing %q", want)
		}
	}
	n := ThirdPartyNotices()
	for _, want := range []string{"FFmpeg", "GPL", "OpenCV", "Wails", "Open Font License"} {
		if !strings.Contains(n, want) {
			t.Errorf("notices missing %q", want)
		}
	}
}

func TestDonateLinksOnlyHTTPS(t *testing.T) {
	for _, l := range DonateLinks() {
		if !strings.HasPrefix(l.URL, "https://") || l.Name == "" {
			t.Fatalf("bad donate link %+v", l)
		}
	}
}
