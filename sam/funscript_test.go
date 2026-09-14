package sam

import (
	"testing"

	"github.com/funfunpayer/SamNPlayer/funscript"
)

func TestFromFunscriptPreservesTimingAndPosition(t *testing.T) {
	fs, err := funscript.Parse([]byte(`{"actions":[
		{"at":0,"pos":20},{"at":500,"pos":80},{"at":1000,"pos":10}
	],"metadata":{"creator":"test-creator"}}`))
	if err != nil {
		t.Fatalf("funscript.Parse fehlgeschlagen: %v", err)
	}
	s := FromFunscript(fs)
	if s.Version != ScriptVersion {
		t.Errorf("erwartet Version %q, ist %q", ScriptVersion, s.Version)
	}
	if len(s.Frames) != len(fs.Actions) {
		t.Fatalf("Frame-Anzahl weicht ab: %d vs %d", len(s.Frames), len(fs.Actions))
	}
	for i, a := range fs.Actions {
		if s.Frames[i].Time != a.At {
			t.Errorf("Frame %d: Time weicht ab: %d vs %d", i, s.Frames[i].Time, a.At)
		}
		if int(s.Frames[i].Motion.Position) != a.Pos {
			t.Errorf("Frame %d: Position weicht ab: %v vs %d", i, s.Frames[i].Motion.Position, a.Pos)
		}
		if s.Frames[i].Motion.Type != MotionUnknown {
			t.Errorf("Frame %d: Type sollte nicht geraten werden, ist %q", i, s.Frames[i].Motion.Type)
		}
	}
	if s.Metadata.Creator != "test-creator" {
		t.Errorf("Creator sollte übernommen werden, ist %q", s.Metadata.Creator)
	}
	if s.Metadata.Source != "funscript-import" {
		t.Errorf("Source sollte 'funscript-import' sein, ist %q", s.Metadata.Source)
	}
}

func TestToFunscriptPreservesTimingAndPosition(t *testing.T) {
	s := &Script{
		Version: ScriptVersion,
		Frames: []Frame{
			{Time: 0, Motion: Motion{Position: 20, Type: MotionRhythmic, Energy: 0.5}},
			{Time: 500, Motion: Motion{Position: 80}},
			{Time: 1000, Motion: Motion{Position: 10}},
		},
		Metadata: Metadata{Creator: "sam-creator"},
	}
	fs := ToFunscript(s)
	if len(fs.Actions) != len(s.Frames) {
		t.Fatalf("Action-Anzahl weicht ab: %d vs %d", len(fs.Actions), len(s.Frames))
	}
	for i, f := range s.Frames {
		if fs.Actions[i].At != f.Time {
			t.Errorf("Action %d: At weicht ab: %d vs %d", i, fs.Actions[i].At, f.Time)
		}
		if fs.Actions[i].Pos != int(f.Motion.Position) {
			t.Errorf("Action %d: Pos weicht ab: %d vs %v", i, fs.Actions[i].Pos, f.Motion.Position)
		}
	}
	if fs.Metadata.Creator != "sam-creator" {
		t.Errorf("Creator sollte übernommen werden, ist %q", fs.Metadata.Creator)
	}
	if fs.Metadata.Duration != 1000 {
		t.Errorf("Duration sollte 1000 sein, ist %d", fs.Metadata.Duration)
	}
}

// Roundtrip: .funscript -> SAM -> .funscript muss Timing und Positionen
// exakt erhalten (docs/SAM_ARCHITECTURE.md, Meilenstein-1-Akzeptanzkriterium
// "bestehende Player, Skripte funktionieren weiter, ohne SAM zu kennen").
func TestRoundtripFunscriptSAMFunscript(t *testing.T) {
	original, err := funscript.Parse([]byte(`{"actions":[
		{"at":0,"pos":50},{"at":133,"pos":20},{"at":316,"pos":95},
		{"at":650,"pos":33},{"at":883,"pos":48},{"at":1033,"pos":0},
		{"at":1150,"pos":100}
	],"metadata":{"creator":"roundtrip-test"}}`))
	if err != nil {
		t.Fatalf("funscript.Parse fehlgeschlagen: %v", err)
	}

	samScript := FromFunscript(original)
	roundtripped := ToFunscript(samScript)

	if len(roundtripped.Actions) != len(original.Actions) {
		t.Fatalf("Action-Anzahl weicht nach Roundtrip ab: %d vs %d",
			len(roundtripped.Actions), len(original.Actions))
	}
	for i, a := range original.Actions {
		if roundtripped.Actions[i].At != a.At {
			t.Errorf("Action %d: Timing nach Roundtrip verändert: %d vs %d",
				i, roundtripped.Actions[i].At, a.At)
		}
		if roundtripped.Actions[i].Pos != a.Pos {
			t.Errorf("Action %d: Position nach Roundtrip verändert: %d vs %d",
				i, roundtripped.Actions[i].Pos, a.Pos)
		}
	}
	if roundtripped.Metadata.Creator != original.Metadata.Creator {
		t.Errorf("Creator nach Roundtrip verändert: %q vs %q",
			roundtripped.Metadata.Creator, original.Metadata.Creator)
	}
}

// Großes Funscript (10000 Actions, ~ein langes Video bei dichter Abtastung)
// - eigenes Akzeptanzkriterium aus docs/SAM_ARCHITECTURE.md ("große Dateien
// testen"), nicht nur eine Handvoll Punkte.
func TestRoundtripLargeFunscript(t *testing.T) {
	const n = 10000
	actions := make([]funscript.Action, n)
	for i := 0; i < n; i++ {
		pos := i % 101 // deterministisches, aber abwechslungsreiches Muster
		actions[i] = funscript.Action{At: int64(i) * 20, Pos: pos}
	}
	original := &funscript.Script{Actions: actions}
	original.Metadata.Creator = "large-file-test"

	samScript := FromFunscript(original)
	if len(samScript.Frames) != n {
		t.Fatalf("erwartet %d Frames, sind %d", n, len(samScript.Frames))
	}
	roundtripped := ToFunscript(samScript)
	if len(roundtripped.Actions) != n {
		t.Fatalf("erwartet %d Actions nach Roundtrip, sind %d", n, len(roundtripped.Actions))
	}
	for i := 0; i < n; i += 137 { // stichprobenartig prüfen, nicht jeden einzelnen Punkt
		if roundtripped.Actions[i] != original.Actions[i] {
			t.Fatalf("Action %d weicht nach Roundtrip ab: %+v vs %+v",
				i, roundtripped.Actions[i], original.Actions[i])
		}
	}
	// Randpunkte (erster/letzter) immer explizit prüfen.
	if roundtripped.Actions[0] != original.Actions[0] {
		t.Fatalf("erste Action weicht ab: %+v vs %+v", roundtripped.Actions[0], original.Actions[0])
	}
	if roundtripped.Actions[n-1] != original.Actions[n-1] {
		t.Fatalf("letzte Action weicht ab: %+v vs %+v", roundtripped.Actions[n-1], original.Actions[n-1])
	}
}
