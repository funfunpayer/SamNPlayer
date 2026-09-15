package funscript

import "testing"

func TestSuggestBackendSmallBoxUsesGrid(t *testing.T) {
	if SuggestBackend(18, 16) != "grid_lk" {
		t.Fatal("kleine Box muss grid_lk sein")
	}
}

func TestSuggestBackendLargeBoxUsesCSRT(t *testing.T) {
	if SuggestBackend(80, 90) != "csrt" {
		t.Fatal("große Box muss csrt sein")
	}
}

func TestSuggestBackendInvalidDefaultsCSRT(t *testing.T) {
	if SuggestBackend(0, 10) != "csrt" {
		t.Fatal("ungueltige Box: csrt")
	}
}
