package edition

import "testing"

func TestNameMatchesStudio(t *testing.T) {
	if (Name() == "studio") != Studio() {
		t.Fatalf("Name %q vs Studio %v", Name(), Studio())
	}
}
