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

func TestGetVibrationCurveContactRecipe(t *testing.T) {
	a := NewApp()
	script := &funscript.Script{Actions: []funscript.Action{
		{At: 0, Pos: 20}, {At: 500, Pos: 20},
		{At: 600, Pos: 90}, {At: 900, Pos: 90},
		{At: 1000, Pos: 20}, {At: 1500, Pos: 20},
	}}
	script.Metadata.Profile = "tj"
	script.Metadata.DeviceRecipe = &funscript.DeviceRecipe{
		Sync: "suction_position", ContactVibration: true,
		ContactVibrationCurve: "peak", ContactVibrationSpan: 0.5,
	}
	a.currentScript = script

	pts, err := a.GetVibrationCurve(100)
	if err != nil {
		t.Fatalf("GetVibrationCurve: %v", err)
	}
	if len(pts) < 2 {
		t.Fatal("erwartete Vibrationsspur")
	}
	var max float64
	for _, p := range pts {
		if p.Vibration > max {
			max = p.Vibration
		}
	}
	if max < 0.5 {
		t.Errorf("Kontaktfenster sollte spürbare Vibration liefern, max=%.3f", max)
	}
}

func TestGetVibrationCurveAbsentWithoutFlag(t *testing.T) {
	a := NewApp()
	script := &funscript.Script{Actions: []funscript.Action{
		{At: 0, Pos: 20}, {At: 500, Pos: 90}, {At: 1000, Pos: 20},
	}}
	script.Metadata.Profile = "tj"
	script.Metadata.DeviceRecipe = &funscript.DeviceRecipe{Sync: "suction_position"}
	a.currentScript = script
	pts, err := a.GetVibrationCurve(100)
	if err != nil {
		t.Fatalf("GetVibrationCurve: %v", err)
	}
	if pts != nil {
		t.Errorf("ohne contact_vibration sollte keine Spur kommen, bekam %d Punkte", len(pts))
	}
}
