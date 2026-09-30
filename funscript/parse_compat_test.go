package funscript

import (
	"encoding/json"
	"strings"
	"testing"
)

// OFS schreibt metadata.duration in Sekunden als Kommazahl (631.8). Vorher
// scheiterte das ganze Skript am JSON-Typ (int64) - jede solche Datei war
// in Play/Improve nicht ladbar.
func TestParseOFSStyleMetadata(t *testing.T) {
	for name, tc := range map[string]struct {
		json string
		want int64
	}{
		"ofs seconds float":   {`{"actions":[{"at":0,"pos":0},{"at":631000,"pos":100}],"inverted":false,"metadata":{"creator":"","duration":631.8,"performers":[],"tags":[],"type":"basic"},"range":100,"version":"1.0"}`, 631800},
		"ofs seconds integer": {`{"actions":[{"at":0,"pos":0},{"at":631000,"pos":100}],"metadata":{"duration":631}}`, 631000},
		"ms stays ms":         {`{"actions":[{"at":0,"pos":0},{"at":631000,"pos":100}],"metadata":{"duration":631800}}`, 631800},
		"string ignored":      {`{"actions":[{"at":0,"pos":0},{"at":1000,"pos":100}],"metadata":{"duration":"10:31"}}`, 0},
		"null":                {`{"actions":[{"at":0,"pos":0},{"at":1000,"pos":100}],"metadata":{"duration":null}}`, 0},
	} {
		s, err := Parse([]byte(tc.json))
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		if int64(s.Metadata.Duration) != tc.want {
			t.Errorf("%s: duration %d, want %d", name, s.Metadata.Duration, tc.want)
		}
	}
}

// Manche Werkzeuge schreiben at/pos als Kommazahlen ("at": 500.4,
// "pos": 99.6) - vorher ebenfalls ein Ladefehler fürs ganze Skript.
func TestParseFloatActions(t *testing.T) {
	s, err := Parse([]byte(`{"actions":[{"at":500.4,"pos":99.6},{"at":0.0,"pos":0.5},{"at":1000,"pos":50}]}`))
	if err != nil {
		t.Fatal(err)
	}
	want := []Action{{0, 1}, {500, 100}, {1000, 50}}
	for i, a := range s.Actions {
		if a != want[i] {
			t.Fatalf("actions %v, want %v", s.Actions, want)
		}
	}
	// Geschrieben wird weiter mit ganzen Zahlen.
	b, _ := json.Marshal(s.Actions[1])
	if string(b) != `{"at":500,"pos":100}` {
		t.Errorf("marshal %s", b)
	}
	if _, err := Parse([]byte(`{"actions":[{"at":"x","pos":1}]}`)); err == nil || !strings.Contains(err.Error(), "ungültiges JSON") {
		t.Errorf("non-number at must still fail: %v", err)
	}
}
