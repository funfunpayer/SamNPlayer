package main

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/funfunpayer/SamNPlayer/funscript"
	"github.com/funfunpayer/SamNPlayer/generator"
	"github.com/funfunpayer/SamNPlayer/logging"
)

const maxBenchmarkHistoryEntries = 30

// RunGoldenClipBenchmark führt generator/golden_clip_benchmark.py gegen das
// eingestellte Manifest aus - docs/NEXT.md Priorität 2's "Reproduce before
// changing the algorithm": eine feste, wiederholbare Vergleichsbasis statt
// Einzelmessungen in Chat-Notizen. Läuft asynchron wie GenerateScript, mit
// eigenem Event-Namensraum ("benchmark:...") statt "generate:...", damit
// beide unabhängig laufen könnten (auch wenn die GUI aktuell nur eins nach
// dem anderen anbietet).
func (a *App) RunGoldenClipBenchmark(manifestPath string) {
	go func() {
		historyPath := a.settings.GetString(prefBenchmarkHistoryPath, defaultBenchmarkHistoryPath())
		result, err := generator.RunGoldenClipBenchmark(manifestPath, historyPath,
			func(line string) { runtime.EventsEmit(a.ctx, "benchmark:progress", line) },
			func(pct int) { runtime.EventsEmit(a.ctx, "benchmark:percent", pct) })
		if err != nil {
			logging.Error("benchmark: run failed", "manifest", manifestPath, "error", err)
			runtime.EventsEmit(a.ctx, "benchmark:done", map[string]any{"error": err.Error()})
			return
		}
		logging.Info("benchmark: run finished", "manifest", manifestPath,
			"clips", result.Summary.Total, "ok", result.Summary.OK)
		runtime.EventsEmit(a.ctx, "benchmark:done", map[string]any{"result": result})
	}()
}

// GetBenchmarkHistory liest die Verlaufsdatei (siehe
// generator.RunGoldenClipBenchmark/golden_clip_benchmark.append_history) -
// jede Zeile ist ein vollständiges generator.BenchmarkResult als JSON.
// Neueste zuerst, auf die letzten maxBenchmarkHistoryEntries begrenzt,
// dasselbe Muster wie TrainingHistory (app_training_history.go). Eine
// kaputte Zeile wird übersprungen, nicht die ganze Datei verworfen.
func (a *App) GetBenchmarkHistory() ([]generator.BenchmarkResult, error) {
	path := a.settings.GetString(prefBenchmarkHistoryPath, defaultBenchmarkHistoryPath())
	if path == "" {
		return []generator.BenchmarkResult{}, nil
	}
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []generator.BenchmarkResult{}, nil
		}
		return nil, err
	}
	defer f.Close()

	var results []generator.BenchmarkResult
	scanner := bufio.NewScanner(f)
	// Verlaufszeilen können sehr lang werden (viele Clips, jeder mit
	// Warnungen) - der Standardpuffer (64KB) reicht dafür nicht immer.
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		var r generator.BenchmarkResult
		if err := json.Unmarshal(scanner.Bytes(), &r); err != nil {
			logging.Warn("benchmark: history line skipped", "error", err)
			continue
		}
		results = append(results, r)
	}

	sort.Slice(results, func(i, j int) bool { return results[i].Timestamp > results[j].Timestamp })
	if len(results) > maxBenchmarkHistoryEntries {
		results = results[:maxBenchmarkHistoryEntries]
	}
	return results, nil
}

// BenchmarkPairScore is the GUI-facing PairScore (scripts vs FunGen/ref).
// Video is optional metadata for KI labels; scoring is Everyday-candidate
// (or any loaded script) against the reference — Everyday Go CSRT remains
// the production basis; KI never becomes the Everyday writer here.
type BenchmarkPairScore = funscript.PairScore

// ScoreScriptPair compares a candidate .funscript against a FunGen/reference
// .funscript (Owner Bench: pick video optional + pick scripts → gut/nicht gut).
// Synchronous — no generate subprocess; uses pure Go ScorePair.
func (a *App) ScoreScriptPair(referencePath, candidatePath, videoPath string) (BenchmarkPairScore, error) {
	var empty BenchmarkPairScore
	ref, err := funscript.Load(referencePath)
	if err != nil {
		return empty, err
	}
	cand, err := funscript.Load(candidatePath)
	if err != nil {
		return empty, err
	}
	score := funscript.ScorePair(ref.Actions, cand.Actions, 0, 0, 0)
	score.Reference = filepath.Clean(referencePath)
	score.Candidate = filepath.Clean(candidatePath)
	if videoPath != "" {
		score.Video = filepath.Clean(videoPath)
	}
	logging.Info("benchmark: pair score",
		"label", score.Label, "passed", score.Passed,
		"reference", score.Reference, "candidate", score.Candidate)
	return score, nil
}

// AppendBenchmarkPairLabel writes one KI-ready JSONL label for a scored pair
// into the configured benchmark history directory (labels.jsonl beside history).
func (a *App) AppendBenchmarkPairLabel(score BenchmarkPairScore) (string, error) {
	historyPath := a.settings.GetString(prefBenchmarkHistoryPath, defaultBenchmarkHistoryPath())
	dir := filepath.Dir(historyPath)
	if dir == "" || dir == "." {
		dir = filepath.Dir(defaultBenchmarkHistoryPath())
	}
	out := filepath.Join(dir, "benchmark_pair_labels.jsonl")
	rec := map[string]any{
		"kind":      "benchmark_pair_label",
		"label":     score.Label,
		"passed":    score.Passed,
		"video":     score.Video,
		"reference": score.Reference,
		"candidate": score.Candidate,
		"score":     score,
	}
	data, err := json.Marshal(rec)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return "", err
	}
	f, err := os.OpenFile(out, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err := f.Write(append(data, '\n')); err != nil {
		return "", err
	}
	return out, nil
}
