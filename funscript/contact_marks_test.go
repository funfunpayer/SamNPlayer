package funscript

import "testing"

func TestContactMarksHasAreas(t *testing.T) {
	if (&ContactMarks{TipClass: "glans"}).HasAreas() {
		t.Fatal("tip class alone is not an area")
	}
	cm := &ContactMarks{Primary: &ContactMarkBox{W: 10, H: 10}}
	if !cm.HasAreas() {
		t.Fatal("primary box should count")
	}
}
