package funscript

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestParseInvertedFlag(t *testing.T) {
	s, err := Parse([]byte(`{"inverted":true,"actions":[{"at":0,"pos":10},{"at":1000,"pos":90}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if !s.Inverted {
		t.Fatal("inverted flag dropped")
	}
	if s.Actions[0].Pos != 10 || s.Actions[1].Pos != 90 {
		t.Fatalf("stored actions must stay authored: %v", s.Actions)
	}
	if s.PlaybackPos(0) != 90 || s.PlaybackPos(1) != 10 {
		t.Fatalf("playback pos: %d %d", s.PlaybackPos(0), s.PlaybackPos(1))
	}
}

func TestToIntensityCurveHonorsInverted(t *testing.T) {
	plain := &Script{Actions: []Action{{0, 0}, {100, 100}}}
	inv := &Script{Actions: []Action{{0, 0}, {100, 100}}, Inverted: true}
	opts := MapOptions{TickMs: 50, MaxSpeed: 0.6, Sync: SyncSuctionPosition}
	a := plain.ToIntensityCurve(opts)
	b := inv.ToIntensityCurve(opts)
	if len(a) < 3 || len(b) != len(a) {
		t.Fatalf("len a=%d b=%d", len(a), len(b))
	}
	if a[0].Suction != 0 {
		t.Fatalf("plain start suc=%v", a[0].Suction)
	}
	if b[0].Suction != 1 {
		t.Fatalf("inverted start suc=%v want 1", b[0].Suction)
	}
	last := len(a) - 1
	if a[last].Suction != 1 || b[last].Suction != 0 {
		t.Fatalf("end plain=%v inv=%v", a[last].Suction, b[last].Suction)
	}
}

func TestWriteInvertedClearsFlag(t *testing.T) {
	p := filepath.Join(t.TempDir(), "x.funscript")
	if err := os.WriteFile(p, []byte(`{"inverted":true,"actions":[{"at":0,"pos":10},{"at":100,"pos":90}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := WriteInverted(p, false); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	var top map[string]json.RawMessage
	if err := json.Unmarshal(b, &top); err != nil {
		t.Fatal(err)
	}
	if _, ok := top["inverted"]; ok {
		t.Fatalf("flag still present: %s", b)
	}
	s, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if s.Inverted || s.Actions[0].Pos != 10 {
		t.Fatalf("actions/flag %+v %v", s.Actions, s.Inverted)
	}
}
