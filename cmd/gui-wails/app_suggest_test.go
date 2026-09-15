package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/funfunpayer/SamNPlayer/funscript"
)

func TestSuggestPolarityAndInvertRoundtrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "p.funscript")
	body := `{"actions":[{"at":0,"pos":10},{"at":100,"pos":15},{"at":200,"pos":12},{"at":300,"pos":80},{"at":400,"pos":90}]}`
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
	a := NewApp()
	if _, err := a.LoadFunscript(path); err != nil {
		t.Fatal(err)
	}
	hint, err := a.SuggestPolarity()
	if err != nil {
		t.Fatal(err)
	}
	if !hint.SuggestInvert {
		t.Fatalf("erwartete Invert-Empfehlung: %+v", hint)
	}
	if err := a.InvertLoadedScript(); err != nil {
		t.Fatal(err)
	}
	if a.currentScript.Actions[0].Pos != 90 {
		t.Fatalf("erste Position nach Invert: %d", a.currentScript.Actions[0].Pos)
	}
}

func TestSuggestOZoneOnLoadedScript(t *testing.T) {
	path := filepath.Join(t.TempDir(), "o.funscript")
	actions := make([]funscript.Action, 0, 101)
	for i := 0; i <= 100; i++ {
		pos := 25
		if i >= 90 {
			pos = 88
		}
		actions = append(actions, funscript.Action{At: int64(i * 100), Pos: pos})
	}
	script := funscript.Script{Actions: actions}
	raw, err := json.Marshal(script)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0644); err != nil {
		t.Fatal(err)
	}
	a := NewApp()
	if _, err := a.LoadFunscript(path); err != nil {
		t.Fatal(err)
	}
	zone, err := a.SuggestOZone()
	if err != nil {
		t.Fatal(err)
	}
	if !zone.OK || zone.StartMs < 8000 {
		t.Fatalf("unerwartete Zone: %+v", zone)
	}
}

func TestApplyRingDownOnLoadedScript(t *testing.T) {
	path := filepath.Join(t.TempDir(), "r.funscript")
	body := `{"actions":[{"at":0,"pos":20},{"at":1000,"pos":85},{"at":2000,"pos":90}]}`
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
	a := NewApp()
	if _, err := a.LoadFunscript(path); err != nil {
		t.Fatal(err)
	}
	nBefore := len(a.currentScript.Actions)
	if err := a.ApplyRingDown(2000, 2); err != nil {
		t.Fatal(err)
	}
	if len(a.currentScript.Actions) <= nBefore {
		t.Fatalf("erwartete zusätzliche Punkte, vorher %d nachher %d", nBefore, len(a.currentScript.Actions))
	}
	last := a.currentScript.Actions[len(a.currentScript.Actions)-1]
	if last.Pos != 0 {
		t.Fatalf("Ring-down muss mit 0 enden: %+v", last)
	}
	if last.At <= 2000 {
		t.Fatalf("Ring-down muss nach atMs liegen: %+v", last)
	}
}
