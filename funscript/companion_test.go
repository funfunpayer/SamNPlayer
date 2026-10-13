package funscript

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCompanionPath(t *testing.T) {
	cases := map[string]string{
		"/x/clip.funscript": "/x/clip.twist.funscript",
		"/x/clip.samn":      "/x/clip.twist.funscript",
		"/x/clip.FUNSCRIPT": "/x/clip.twist.funscript",
		"/x/clip":           "/x/clip.twist.funscript",
	}
	for in, want := range cases {
		if got := CompanionPath(in, "twist"); got != want {
			t.Errorf("CompanionPath(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestLoadCompanionAxis(t *testing.T) {
	dir := t.TempDir()
	main := filepath.Join(dir, "clip.funscript")
	write := func(name, body string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("clip.funscript", `{"actions":[{"at":0,"pos":0},{"at":500,"pos":100}]}`)
	write("clip.twist.funscript", `{"actions":[{"at":400,"pos":80},{"at":0,"pos":20}]}`)
	write("clip.roll.funscript", `{"inverted":true,"actions":[{"at":0,"pos":10},{"at":100,"pos":90}]}`)
	write("clip.pitch.funscript", `not json`)

	twist, err := LoadCompanionAxis(main, "twist")
	if err != nil || len(twist) != 2 || twist[0].At != 0 || twist[0].Pos != 20 {
		t.Fatalf("twist sorted by time: %+v %v", twist, err)
	}
	roll, _ := LoadCompanionAxis(main, "roll")
	if len(roll) != 2 || roll[0].Pos != 90 {
		t.Fatalf("roll inversion applied: %+v", roll)
	}
	if acts, err := LoadCompanionAxis(main, "surge"); acts != nil || err != nil {
		t.Fatalf("missing file is no error: %v %v", acts, err)
	}
	if _, err := LoadCompanionAxis(main, "pitch"); err == nil {
		t.Fatal("a broken companion file reports its error")
	}
	if acts, _ := LoadCompanionAxis(main, "general"); acts != nil {
		t.Fatal("only companion axes are read")
	}
	found := CompanionAxesFound(main)
	if len(found) != 2 || found[0] != "twist" || found[1] != "roll" {
		t.Fatalf("found %v, want [twist roll] in display order", found)
	}
}

func TestSaveCompanionAxisKeepsFieldsAndInversion(t *testing.T) {
	dir := t.TempDir()
	main := filepath.Join(dir, "clip.funscript")
	roll := filepath.Join(dir, "clip.roll.funscript")
	if err := os.WriteFile(roll, []byte(`{"version":"1.0","inverted":true,"range":90,"actions":[{"at":0,"pos":10},{"at":100,"pos":90}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := SaveCompanionAxis(main, "roll", []Action{{At: 0, Pos: 30}, {At: 200, Pos: 70}}); err != nil {
		t.Fatal(err)
	}
	got, err := LoadCompanionAxis(main, "roll")
	if err != nil || len(got) != 2 || got[0].Pos != 30 || got[1].At != 200 || got[1].Pos != 70 {
		t.Fatalf("round trip as edited: %+v %v", got, err)
	}
	raw, _ := os.ReadFile(roll)
	for _, want := range []string{`"range":90`, `"version":"1.0"`, `"inverted":true`, `"pos":70`} {
		if !strings.Contains(string(raw), want) {
			t.Fatalf("file lost %s: %s", want, raw)
		}
	}
	if err := SaveCompanionAxis(main, "pitch", []Action{{At: 0, Pos: 1}, {At: 9, Pos: 2}}); err == nil {
		t.Fatal("missing companion file must fail")
	}
	if err := SaveCompanionAxis(main, "general", nil); err == nil {
		t.Fatal("only companion axes")
	}
}
