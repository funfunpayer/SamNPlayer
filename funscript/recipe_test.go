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

// Ohne ContactVibration bleibt tf/tj wie bisher stumm, auch bei tiefem
// Kontakt (Pos nahe Maximum) - reine Opt-in-Erweiterung, kein geändertes
// Standardverhalten.
func TestRecipeTJContactVibrationOffByDefault(t *testing.T) {
	script := scriptFrom(
		Action{At: 0, Pos: 20},
		Action{At: 500, Pos: 90},
		Action{At: 1000, Pos: 90},
		Action{At: 1500, Pos: 20},
	)
	opts := RecipeFor("tj")
	opts.Smoothing = 0
	frames := script.ToIntensityCurve(opts)
	for _, f := range frames {
		if f.Vibration > 0.001 {
			t.Fatalf("ContactVibration nicht gesetzt, aber Vibration %.3f bei %dms", f.Vibration, f.At)
		}
	}
}

// Mit ContactVibration steigt die Vibration nur, wenn die Position nahe ihr
// eigenes, im Skript beobachtetes Maximum kommt (simulierter Kontakt), und
// bleibt davor/danach bei 0 - Dauer/Stärke kommen aus dem Positionssignal
// selbst, nicht aus einem festen Impuls.
func TestRecipeTJContactVibrationTracksPosition(t *testing.T) {
	script := scriptFrom(
		Action{At: 0, Pos: 20}, // fern
		Action{At: 500, Pos: 20},
		Action{At: 600, Pos: 90},  // Kontakt: nahe am beobachteten Maximum
		Action{At: 900, Pos: 90},  // Kontakt hält an - lange Berührung
		Action{At: 1000, Pos: 20}, // wieder fern
		Action{At: 1500, Pos: 20},
	)
	opts := RecipeFor("tj")
	opts.ContactVibration = true
	opts.Smoothing = 0
	frames := script.ToIntensityCurve(opts)
	if len(frames) == 0 {
		t.Fatal("keine Frames")
	}
	var vibDuringFar, vibDuringContact, maxVib float64
	var contactFrames int
	for _, f := range frames {
		if f.At < 400 || (f.At > 1100 && f.At < 1500) {
			if f.Vibration > vibDuringFar {
				vibDuringFar = f.Vibration
			}
		}
		if f.At >= 700 && f.At <= 850 {
			contactFrames++
			if f.Vibration > vibDuringContact {
				vibDuringContact = f.Vibration
			}
		}
		if f.Vibration > maxVib {
			maxVib = f.Vibration
		}
	}
	if vibDuringFar > 0.001 {
		t.Errorf("fern von Kontakt sollte Vibration 0 sein, ist %.3f", vibDuringFar)
	}
	if contactFrames == 0 {
		t.Fatal("Testfenster für Kontakt leer - Test defekt")
	}
	if vibDuringContact < 0.5 {
		t.Errorf("während anhaltendem Kontakt sollte Vibration deutlich steigen, max %.3f", vibDuringContact)
	}
	if maxVib > 1.0001 {
		t.Errorf("Vibration über 1.0: %.4f", maxVib)
	}
}

// Ein Skript ohne nennenswerte Positions-Spannweite (z.B. nur ein kurzer,
// flacher Ausschlag) darf nicht dauerhaft vibrieren - sonst würde jede
// normale Bewegung als "Kontakt" fehlinterpretiert.
func TestRecipeTJContactVibrationNeedsSpan(t *testing.T) {
	script := scriptFrom(
		Action{At: 0, Pos: 50},
		Action{At: 500, Pos: 52},
		Action{At: 1000, Pos: 50},
	)
	opts := RecipeFor("tj")
	opts.ContactVibration = true
	opts.Smoothing = 0
	frames := script.ToIntensityCurve(opts)
	for _, f := range frames {
		if f.Vibration > 0.001 {
			t.Fatalf("zu geringe Spannweite sollte keinen Kontakt auslösen, Vibration %.3f bei %dms", f.Vibration, f.At)
		}
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
