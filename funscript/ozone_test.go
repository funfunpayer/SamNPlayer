package funscript

import "testing"

func TestSuggestOZonePicksLatePeak(t *testing.T) {
	var actions []Action
	for i := 0; i <= 100; i++ {
		pos := 20
		if i >= 90 {
			pos = 90
		}
		actions = append(actions, Action{At: int64(i * 100), Pos: pos})
	}
	s := SuggestOZone(actions)
	if !s.OK {
		t.Fatalf("erwartete Zone, got %+v", s)
	}
	if s.StartMs < 8000 {
		t.Fatalf("Zone sollte im letzten Achtel liegen, start=%d", s.StartMs)
	}
	if s.EndMs <= s.StartMs {
		t.Fatalf("Ende muss nach Start liegen: %+v", s)
	}
}

func TestSuggestOZoneRejectsFlatTail(t *testing.T) {
	var actions []Action
	for i := 0; i <= 50; i++ {
		actions = append(actions, Action{At: int64(i * 100), Pos: 20})
	}
	s := SuggestOZone(actions)
	if s.OK {
		t.Fatalf("flaches Ende darf keine Zone liefern: %+v", s)
	}
}

func TestSuggestOZoneShortScript(t *testing.T) {
	s := SuggestOZone([]Action{{At: 0, Pos: 0}, {At: 100, Pos: 100}, {At: 200, Pos: 0}, {At: 300, Pos: 50}})
	if s.OK {
		t.Fatalf("kurzes Skript: %+v", s)
	}
}
