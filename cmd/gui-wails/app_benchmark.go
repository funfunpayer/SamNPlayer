package main

import (
	"bufio"
	"encoding/json"
	"os"
	"sort"

	"github.com/wailsapp/wails/v2/pkg/runtime"

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
