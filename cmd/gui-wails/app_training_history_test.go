package main

import (
	"os"
	"path/filepath"
	"testing"
)

func writeSessionLog(t *testing.T, path string, lines []string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(joinLines(lines)), 0644); err != nil {
		t.Fatal(err)
	}
}

func joinLines(lines []string) string {
	out := ""
	for _, l := range lines {
		out += l + "\n"
	}
	return out
}

func TestSummarizeSessionLogAggregatesCycles(t *testing.T) {
	path := filepath.Join(t.TempDir(), "s.jsonl")
	writeSessionLog(t, path, []string{
		`{"timestamp":"2026-09-10T10:00:00Z","technique":"stopstart","channel":"vibration","cycleIndex":0,"cyclesTotal":3,"peakIntensity":0.6,"stoppedByUser":false,"reachedPeakAfterMs":4000,"arousalBefore":0}`,
		`{"timestamp":"2026-09-10T10:01:00Z","technique":"stopstart","channel":"vibration","cycleIndex":1,"cyclesTotal":3,"peakIntensity":0.8,"stoppedByUser":true,"reachedPeakAfterMs":2000,"arousalBefore":8}`,
		`{"timestamp":"2026-09-10T10:02:00Z","technique":"stopstart","channel":"vibration","cycleIndex":2,"cyclesTotal":3,"peakIntensity":1.0,"stoppedByUser":false,"reachedPeakAfterMs":0,"arousalBefore":0}`,
	})

	summary, err := summarizeSessionLog(path)
	if err != nil {
		t.Fatal(err)
	}
	if summary == nil {
		t.Fatal("erwartete eine Zusammenfassung, got nil")
	}
	if summary.CyclesCompleted != 3 {
		t.Fatalf("CyclesCompleted: %d", summary.CyclesCompleted)
	}
	if summary.CyclesStoppedEarly != 1 {
		t.Fatalf("CyclesStoppedEarly: %d", summary.CyclesStoppedEarly)
	}
	wantPeak := (0.6 + 0.8 + 1.0) / 3
	if diff := summary.MeanPeakIntensity - wantPeak; diff > 1e-9 || diff < -1e-9 {
		t.Fatalf("MeanPeakIntensity: %v, want %v", summary.MeanPeakIntensity, wantPeak)
	}
	// Nur die zwei Zyklen mit reachedPeakAfterMs > 0 fließen ein - der dritte
	// (0, "kein Peak erreicht") darf den Schnitt nicht künstlich nach unten ziehen.
	if summary.MeanReachedPeakAfterMs != 3000 {
		t.Fatalf("MeanReachedPeakAfterMs: %v", summary.MeanReachedPeakAfterMs)
	}
	if summary.ArousalReportsCount != 1 || summary.MeanArousalReported != 8 {
		t.Fatalf("Arousal: count=%d mean=%v", summary.ArousalReportsCount, summary.MeanArousalReported)
	}
	if summary.Technique != "stopstart" || summary.Channel != "vibration" {
		t.Fatalf("Technique/Channel: %+v", summary)
	}
	if summary.StartedAt != "2026-09-10T10:00:00Z" {
		t.Fatalf("StartedAt sollte vom ERSTEN Eintrag kommen: %s", summary.StartedAt)
	}
}

func TestSummarizeSessionLogSkipsBrokenLinesKeepsRest(t *testing.T) {
	path := filepath.Join(t.TempDir(), "s.jsonl")
	writeSessionLog(t, path, []string{
		`{"timestamp":"2026-09-10T10:00:00Z","technique":"plateau","channel":"suction","peakIntensity":0.5}`,
		`nicht einmal JSON`,
		`{"timestamp":"2026-09-10T10:01:00Z","technique":"plateau","channel":"suction","peakIntensity":0.7}`,
	})

	summary, err := summarizeSessionLog(path)
	if err != nil {
		t.Fatal(err)
	}
	if summary.CyclesCompleted != 2 {
		t.Fatalf("kaputte Zeile hätte übersprungen werden müssen, CyclesCompleted=%d", summary.CyclesCompleted)
	}
}

func TestSummarizeSessionLogEmptyFileReturnsNil(t *testing.T) {
	path := filepath.Join(t.TempDir(), "empty.jsonl")
	writeSessionLog(t, path, nil)

	summary, err := summarizeSessionLog(path)
	if err != nil {
		t.Fatal(err)
	}
	if summary != nil {
		t.Fatalf("leere Datei sollte nil liefern, got %+v", summary)
	}
}

func TestTrainingHistorySortsNewestFirstAndSkipsNonJsonl(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)
	sessionsDir := filepath.Join(tmp, "SamNPlayer", "logs", "sessions")
	if err := os.MkdirAll(sessionsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Dateiname trägt die TECHNIK vor dem Zeitstempel ("training-plateau-..."
	// vs "training-stopstart-...") - eine reine Dateinamen-Sortierung würde
	// hier die falsche Reihenfolge liefern. TrainingHistory muss nach dem
	// Zeitstempel IM Protokoll sortieren, nicht nach dem Dateinamen.
	writeSessionLog(t, filepath.Join(sessionsDir, "training-stopstart-20260901-100000.jsonl"), []string{
		`{"timestamp":"2026-09-01T10:00:00Z","technique":"stopstart","channel":"vibration","peakIntensity":0.5}`,
	})
	writeSessionLog(t, filepath.Join(sessionsDir, "training-plateau-20260910-100000.jsonl"), []string{
		`{"timestamp":"2026-09-10T10:00:00Z","technique":"plateau","channel":"suction","peakIntensity":0.9}`,
	})
	// Nicht-JSONL-Datei im selben Ordner darf nicht mitgezählt werden.
	if err := os.WriteFile(filepath.Join(sessionsDir, "readme.txt"), []byte("hi"), 0644); err != nil {
		t.Fatal(err)
	}

	a := NewApp()
	history, err := a.TrainingHistory()
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 2 {
		t.Fatalf("erwartete 2 Sessions, got %d: %+v", len(history), history)
	}
	if history[0].Technique != "plateau" {
		t.Fatalf("neueste Session (Datum, nicht Dateiname) sollte zuerst kommen: %+v", history)
	}
	if history[1].Technique != "stopstart" {
		t.Fatalf("älteste Session sollte zuletzt kommen: %+v", history)
	}
}

func TestTrainingHistoryNoSessionsDirIsNotAnError(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	a := NewApp()
	history, err := a.TrainingHistory()
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 0 {
		t.Fatalf("erwartete leere Liste ohne sessions-Ordner, got %+v", history)
	}
}
