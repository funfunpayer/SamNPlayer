package main

import (
	"testing"

	"github.com/funfunpayer/SamNPlayer/funscript"
)

// Die Ausdünnung darf die Amplitude nicht verlieren: ein Funscript besteht
// fast nur aus Hoch- und Tiefpunkten, jeden n-ten Punkt zu nehmen würde die
// Kurve flacher aussehen lassen, als sie ist.
func TestGetScriptCurveKeepsExtremes(t *testing.T) {
	a := NewApp()
	var actions []funscript.Action
	for i := 0; i < 5000; i++ {
		pos := 0
		if i%2 == 0 {
			pos = 100
		}
		actions = append(actions, funscript.Action{At: int64(i * 100), Pos: pos})
	}
	a.currentScript = &funscript.Script{Actions: actions}

	pts, err := a.GetScriptCurve(200)
	if err != nil {
		t.Fatalf("GetScriptCurve: %v", err)
	}
	if len(pts) > 200 {
		t.Errorf("zu viele Punkte: %d", len(pts))
	}
	if len(pts) < 50 {
		t.Fatalf("zu wenige Punkte: %d", len(pts))
	}

	var sawMin, sawMax bool
	for _, p := range pts {
		if p.Pos == 0 {
			sawMin = true
		}
		if p.Pos == 100 {
			sawMax = true
		}
	}
	if !sawMin || !sawMax {
		t.Error("Ausdünnung hat die Extremwerte verloren - die Kurve würde flacher wirken")
	}

	// Zeitlich aufsteigend, sonst zickzackt die Linie rückwärts.
	for i := 1; i < len(pts); i++ {
		if pts[i].AtMs < pts[i-1].AtMs {
			t.Fatalf("Punkte nicht zeitlich sortiert bei %d: %d < %d", i, pts[i].AtMs, pts[i-1].AtMs)
		}
	}
}

func TestGetScriptCurveShortScriptUnchanged(t *testing.T) {
	a := NewApp()
	a.currentScript = &funscript.Script{Actions: []funscript.Action{
		{At: 0, Pos: 10}, {At: 500, Pos: 90}, {At: 1000, Pos: 20},
	}}
	pts, err := a.GetScriptCurve(100)
	if err != nil {
		t.Fatalf("GetScriptCurve: %v", err)
	}
	if len(pts) != 3 {
		t.Errorf("kurzes Skript muss unverändert durchgereicht werden, bekam %d Punkte", len(pts))
	}
}

func TestGetScriptCurveWithoutScript(t *testing.T) {
	a := NewApp()
	if _, err := a.GetScriptCurve(100); err == nil {
		t.Error("ohne geladenes Skript muss ein Fehler kommen")
	}
}
