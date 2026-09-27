package aiscript

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadImitationSamplesSkipsBad(t *testing.T) {
	dir := t.TempDir()
	_, err := ExportImitationSample(dir, ImitationSample{
		Actions: []Action{{At: 0, Pos: 10}, {At: 1000, Pos: 90}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "junk.json"), []byte(`{not json`), 0o644); err != nil {
		t.Fatal(err)
	}
	samples, err := LoadImitationSamples(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(samples) != 1 {
		t.Fatalf("got %d samples", len(samples))
	}
	if CountValidSamples(dir) != 1 {
		t.Fatal("count mismatch")
	}
}

func TestPickBestSamplePrefersDurationAndQD(t *testing.T) {
	pass := true
	fail := false
	samples := []LoadedSample{
		{Sample: ImitationSample{DurationMs: 5000, QDPassed: &fail, Actions: []Action{{At: 0, Pos: 0}, {At: 5000, Pos: 100}}}},
		{Sample: ImitationSample{DurationMs: 9800, QDPassed: &pass, Actions: []Action{{At: 0, Pos: 0}, {At: 9800, Pos: 100}}}},
		{Sample: ImitationSample{DurationMs: 10000, QDPassed: &fail, Actions: []Action{{At: 0, Pos: 0}, {At: 10000, Pos: 50}}}},
	}
	got, err := PickBestSample(samples, 10000)
	if err != nil {
		t.Fatal(err)
	}
	if got.Sample.DurationMs != 9800 {
		t.Fatalf("want QD-passed near 10s, got %d", got.Sample.DurationMs)
	}
}

func TestStretchActions(t *testing.T) {
	in := []Action{{At: 100, Pos: 10}, {At: 600, Pos: 90}, {At: 1100, Pos: 20}}
	out := StretchActions(in, 2000)
	if len(out) != 3 {
		t.Fatalf("len=%d", len(out))
	}
	if out[0].At != 0 {
		t.Fatalf("start=%d", out[0].At)
	}
	if out[2].At != 2000 {
		t.Fatalf("end=%d", out[2].At)
	}
	if out[1].Pos != 90 {
		t.Fatalf("pos mutated: %d", out[1].Pos)
	}
}
