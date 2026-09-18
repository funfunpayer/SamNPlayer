package main

import (
	"log/slog"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/funfunpayer/SamNPlayer/generator"
	"github.com/funfunpayer/SamNPlayer/logging"
)

const (
	prefUpdateCheckOnStartup = "update.check_on_startup"
	prefLogLevel             = "log.level"
	prefReportPath           = "generator.reportPath"
	prefClearCacheOnExit     = "cache.clearOnExit"
	prefDeviceTransport      = "device.transport"
	prefIntifaceURL          = "device.intifaceUrl"
	prefDeviceConnectTest    = "device.connect_test"
	prefAIRoiModelPath       = "generator.aiRoiModelPath"
	prefAIBaseURL            = "generator.aiBaseUrl"
	prefBenchmarkManifest    = "generator.benchmarkManifestPath"
	prefBenchmarkHistoryPath = "generator.benchmarkHistoryPath"
	prefDiagnosticsHistory   = "device.diagnosticsHistoryPath"
	prefRoiDatasetDir        = "generator.roiTrainingDatasetDir"

	prefPlaybackMock        = "playback.mock"
	prefPlaybackSync        = "playback.sync_mode"
	prefPlaybackTickMs      = "playback.tick_ms"
	prefPlaybackMaxSpeed    = "playback.max_speed"
	prefPlaybackSmoothing   = "playback.smoothing"
	prefPlaybackSoftStartMs = "playback.soft_start_ms"
	prefPlaybackEOEnabled   = "playback.extended_o_enabled"
	prefPlaybackEOMin       = "playback.extended_o_min"
	prefPlaybackEOHoldS     = "playback.extended_o_hold_seconds"
	prefPlaybackEORestoreMs = "playback.extended_o_restore_ms"

	prefTrainingMock                = "training.mock"
	prefTrainingTechnique           = "training.technique"
	prefTrainingChannel             = "training.channel"
	prefTrainingCycles              = "training.cycles"
	prefTrainingRampUpMs            = "training.ramp_up_ms"
	prefTrainingHoldMs              = "training.hold_ms"
	prefTrainingRestMs              = "training.rest_ms"
	prefTrainingPeakIntensity       = "training.peak_intensity"
	prefTrainingPlateauFraction     = "training.plateau_fraction"
	prefTrainingProgressionPerCycle = "training.progression_per_cycle"
)

// Settings wird komplett auf einmal ans Frontend gegeben und zurückgenommen -
// einfacher als viele einzelne Get/Set-Aufrufe über die Wails-Bindung.
type Settings struct {
	UpdateCheckOnStartup bool    `json:"updateCheckOnStartup"`
	LogLevel             string  `json:"logLevel"`
	PlaybackMock         bool    `json:"playbackMock"`
	PlaybackSync         string  `json:"playbackSync"`
	PlaybackTickMs       float64 `json:"playbackTickMs"`
	PlaybackMaxSpeed     float64 `json:"playbackMaxSpeed"`
	PlaybackSmoothing    float64 `json:"playbackSmoothing"`
	PlaybackSoftStartMs  float64 `json:"playbackSoftStartMs"`
	PlaybackEOEnabled    bool    `json:"playbackEOEnabled"`
	PlaybackEOMin        float64 `json:"playbackEOMin"`
	PlaybackEOHoldS      float64 `json:"playbackEOHoldS"`
	PlaybackEORestoreMs  float64 `json:"playbackEORestoreMs"`

	TrainingMock                bool    `json:"trainingMock"`
	TrainingTechnique           string  `json:"trainingTechnique"`
	TrainingChannel             string  `json:"trainingChannel"`
	TrainingCycles              float64 `json:"trainingCycles"`
	TrainingRampUpMs            float64 `json:"trainingRampUpMs"`
	TrainingHoldMs              float64 `json:"trainingHoldMs"`
	TrainingRestMs              float64 `json:"trainingRestMs"`
	TrainingPeakIntensity       float64 `json:"trainingPeakIntensity"`
	TrainingPlateauFraction     float64 `json:"trainingPlateauFraction"`
	TrainingProgressionPerCycle float64 `json:"trainingProgressionPerCycle"`

	LogPath string `json:"logPath"`

	// ReportPath: Zieldatei für die Messwerte jedes Generatorlaufs.
	// Leer = kein Bericht. Frei wählbar, damit der Bericht dort landet, wo
	// er gebraucht wird - etwa neben der Videosammlung.
	ReportPath        string `json:"reportPath"`
	DefaultReportPath string `json:"defaultReportPath"`
	ClearCacheOnExit  bool   `json:"clearCacheOnExit"`

	// Verbindungsart und Serveradresse merken: wer Intiface auf dem Handy
	// laufen lässt, tippt sonst bei jedem Start dieselbe IP neu ein.
	DeviceTransport string `json:"deviceTransport"`
	IntifaceURL     string `json:"intifaceUrl"`
	// DeviceConnectTest: nach erfolgreichem Verbinden einmal kurz Vib/Sog
	// testen. Standard aus - stört sonst bei jedem Connect.
	DeviceConnectTest bool `json:"deviceConnectTest"`

	// AIRoiModelPath: Pfad zur .onnx-Modelldatei für die KI-Regionssuche
	// (generator/ai_roi.py). Leer = Standardordner (siehe
	// ai_roi.default_model_path()) - die meisten Nutzer legen die Datei
	// einfach dort ab, statt hier einen Pfad einzutragen. Ob die KI-Suche
	// nutzbar ist, prüft CheckAIRoiAvailable() bei Bedarf separat (kostet
	// einen Python-Start, gehört darum nicht in dieses Massen-Get).
	AIRoiModelPath string `json:"aiRoiModelPath"`

	// AIBaseURL: Adresse eines lokalen Colibri-Servers (coli serve) für
	// Profil-Vorschlag und KI-Zweitmeinung zur Qualität (docs/AI_ADAPTER.md).
	// Leer = colibri_client.DEFAULT_BASE_URL (Standard-Localhost-Port).
	AIBaseURL string `json:"aiBaseUrl"`

	// BenchmarkManifestPath: Pfad zur Manifest-Datei des Golden-Clip-
	// Benchmarks (siehe generator/golden_clip_benchmark.py) - zeigt auf
	// lokale Videos/Referenzen des Nutzers, liegt darum nicht im Repository
	// und hat keinen sinnvollen Standardwert.
	BenchmarkManifestPath string `json:"benchmarkManifestPath"`

	// BenchmarkHistoryPath: JSONL-Datei, an die jeder Benchmark-Lauf
	// angehängt wird - Standardort analog zu DefaultReportPath.
	BenchmarkHistoryPath        string `json:"benchmarkHistoryPath"`
	DefaultBenchmarkHistoryPath string `json:"defaultBenchmarkHistoryPath"`

	// DiagnosticsHistoryPath: JSONL-Datei, an die jeder Geräte-Diagnoselauf
	// angehängt wird (siehe app_diagnostics.go) - Standardort analog zu
	// DefaultBenchmarkHistoryPath.
	DiagnosticsHistoryPath        string `json:"diagnosticsHistoryPath"`
	DefaultDiagnosticsHistoryPath string `json:"defaultDiagnosticsHistoryPath"`

	// RoiDatasetDir: Ordner, in dem der Bootstrap-Trainingsdatensatz für die
	// KI-Regionserkennung wächst (siehe app_roi_training.go) - Standardort
	// analog zu DefaultBenchmarkHistoryPath.
	RoiDatasetDir        string `json:"roiDatasetDir"`
	DefaultRoiDatasetDir string `json:"defaultRoiDatasetDir"`
}

func (a *App) GetSettings() Settings {
	s := a.settings
	return Settings{
		UpdateCheckOnStartup: s.GetBool(prefUpdateCheckOnStartup, true),
		LogLevel:             s.GetString(prefLogLevel, "info"),
		PlaybackMock:         s.GetBool(prefPlaybackMock, true),
		PlaybackSync:         s.GetString(prefPlaybackSync, "independent"),
		PlaybackTickMs:       s.GetFloat(prefPlaybackTickMs, 50),
		PlaybackMaxSpeed:     s.GetFloat(prefPlaybackMaxSpeed, 0.6),
		PlaybackSmoothing:    s.GetFloat(prefPlaybackSmoothing, 0.3),
		PlaybackSoftStartMs:  s.GetFloat(prefPlaybackSoftStartMs, 500),
		PlaybackEOEnabled:    s.GetBool(prefPlaybackEOEnabled, true),
		PlaybackEOMin:        s.GetFloat(prefPlaybackEOMin, 0.1),
		PlaybackEOHoldS:      s.GetFloat(prefPlaybackEOHoldS, 10),
		PlaybackEORestoreMs:  s.GetFloat(prefPlaybackEORestoreMs, 500),

		TrainingMock:                s.GetBool(prefTrainingMock, true),
		TrainingTechnique:           s.GetString(prefTrainingTechnique, "stopstart"),
		TrainingChannel:             s.GetString(prefTrainingChannel, "vibration"),
		TrainingCycles:              s.GetFloat(prefTrainingCycles, 5),
		TrainingRampUpMs:            s.GetFloat(prefTrainingRampUpMs, 8000),
		TrainingHoldMs:              s.GetFloat(prefTrainingHoldMs, 3000),
		TrainingRestMs:              s.GetFloat(prefTrainingRestMs, 10000),
		TrainingPeakIntensity:       s.GetFloat(prefTrainingPeakIntensity, 0.8),
		TrainingPlateauFraction:     s.GetFloat(prefTrainingPlateauFraction, 0.7),
		TrainingProgressionPerCycle: s.GetFloat(prefTrainingProgressionPerCycle, 0.15),

		LogPath:           logging.Path(),
		ReportPath:        s.GetString(prefReportPath, ""),
		DefaultReportPath: defaultReportPath(),
		ClearCacheOnExit:  s.GetBool(prefClearCacheOnExit, false),
		DeviceTransport:   s.GetString(prefDeviceTransport, "ble"),
		IntifaceURL:       s.GetString(prefIntifaceURL, ""),
		DeviceConnectTest: s.GetBool(prefDeviceConnectTest, false),
		AIRoiModelPath:    s.GetString(prefAIRoiModelPath, ""),
		AIBaseURL:         s.GetString(prefAIBaseURL, ""),

		BenchmarkManifestPath:       s.GetString(prefBenchmarkManifest, ""),
		BenchmarkHistoryPath:        s.GetString(prefBenchmarkHistoryPath, ""),
		DefaultBenchmarkHistoryPath: defaultBenchmarkHistoryPath(),

		DiagnosticsHistoryPath:        s.GetString(prefDiagnosticsHistory, ""),
		DefaultDiagnosticsHistoryPath: defaultDiagnosticsHistoryPath(),

		RoiDatasetDir:        s.GetString(prefRoiDatasetDir, ""),
		DefaultRoiDatasetDir: generator.DefaultRoiDatasetDir(),
	}
}

// SetSetting speichert einen einzelnen Wert - wird bei jeder Änderung eines
// Formularfelds im Frontend aufgerufen (statt alles auf einmal zu speichern,
// damit nichts verloren geht, falls die App zwischendurch beendet wird).
func (a *App) SetSetting(key string, value any) error {
	if key == prefLogLevel {
		if s, ok := value.(string); ok {
			logging.SetLevel(parseLogLevel(s))
		}
	}
	return a.settings.Set(key, value)
}

func (a *App) OpenLogFolder() error {
	dir, err := logging.Dir()
	if err != nil {
		return err
	}
	if _, err := os.Stat(dir); err != nil {
		return err
	}
	runtime.BrowserOpenURL(a.ctx, fileURL(dir))
	return nil
}

func parseLogLevel(s string) slog.Level {
	switch s {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// defaultReportPath schlägt einen Ort neben der Logdatei vor.
func defaultReportPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "SamNPlayer", "messwerte.jsonl")
}

// defaultBenchmarkHistoryPath schlägt einen Ort neben den übrigen
// Nutzerdaten vor - analog zu defaultReportPath.
func defaultBenchmarkHistoryPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "SamNPlayer", "golden_clip_history.jsonl")
}

// defaultAIRoiModelPath bildet ai_roi.py's default_model_path() 1:1 in Go
// nach - GLEICHE Umgebungsvariablen, gleiche Fallback-Reihenfolge (NICHT
// os.UserConfigDir(), das unter Windows %AppData% statt %LOCALAPPDATA%
// liefert). CheckAIRoiAvailable() prüft am Ende genau den Pfad, den die
// Python-Funktion berechnet - ein frisch trainiertes Modell muss exakt
// dort landen, sonst findet die App ihr eigenes Trainingsergebnis nicht.
func defaultAIRoiModelPath() string {
	if local := os.Getenv("LOCALAPPDATA"); local != "" {
		return filepath.Join(local, "SamNPlayer", "models", "roi_detector.onnx")
	}
	xdg := os.Getenv("XDG_CONFIG_HOME")
	if xdg == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		xdg = filepath.Join(home, ".config")
	}
	return filepath.Join(xdg, "SamNPlayer", "models", "roi_detector.onnx")
}

// defaultDiagnosticsHistoryPath schlägt einen Ort neben den übrigen
// Nutzerdaten vor - analog zu defaultBenchmarkHistoryPath.
func defaultDiagnosticsHistoryPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "SamNPlayer", "device_diagnostics_history.jsonl")
}
