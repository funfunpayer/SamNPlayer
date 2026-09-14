package funscript

import "testing"

func TestNormalizeProfile(t *testing.T) {
	if NormalizeProfile("TF") != ProfileTJ {
		t.Fatalf("tf sollte auf tj fallen, ist %q", NormalizeProfile("TF"))
	}
	if NormalizeProfile("tj") != ProfileTJ {
		t.Fatalf("tj kanonisch, ist %q", NormalizeProfile("tj"))
	}
	if !IsDistanceProfile("tf") || !IsDistanceProfile("TJ") {
		t.Fatal("tf/tj müssen Distanzprofile sein")
	}
	if IsDistanceProfile("standard") {
		t.Fatal("standard ist kein Distanzprofil")
	}
}

func TestRecipeTJSuctionOnlyNoVibration(t *testing.T) {
	script := scriptFrom(
		Action{At: 0, Pos: 20},
		Action{At: 800, Pos: 90},
		Action{At: 1600, Pos: 20},
	)
	opts := RecipeFor("tj")
	opts.Smoothing = 0
	frames := script.ToIntensityCurve(opts)
	if len(frames) == 0 {
		t.Fatal("keine Frames")
	}
	var maxSuc, maxVib float64
	var minSuc = 1.0
	for _, f := range frames {
		if f.Vibration > maxVib {
			maxVib = f.Vibration
		}
		if f.Suction > maxSuc {
			maxSuc = f.Suction
		}
		if f.Suction < minSuc {
			minSuc = f.Suction
		}
	}
	if maxVib > 0.001 {
		t.Errorf("tf/tj darf nicht vibrieren, max %.3f", maxVib)
	}
	if maxSuc < 0.7 {
		t.Errorf("Sog muss der hohen Position folgen, max %.3f", maxSuc)
	}
	if minSuc < 0.15 {
		t.Errorf("Sog-Boden fehlt: min %.3f", minSuc)
	}
}

func TestRecipeTFEqualsTJ(t *testing.T) {
	a, b := RecipeFor("tf"), RecipeFor("tj")
	if a.Sync != b.Sync || a.MinSuction != b.MinSuction {
		t.Fatalf("tf und tj müssen dasselbe Rezept sein: %+v vs %+v", a, b)
	}
	if a.Sync != SyncSuctionPosition {
		t.Fatalf("erwartet suction_position, ist %s", a.Sync)
	}
}
