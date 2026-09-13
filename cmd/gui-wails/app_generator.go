package main

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/funfunpayer/SamNPlayer/funscript"
	"github.com/funfunpayer/SamNPlayer/generator"
	"github.com/funfunpayer/SamNPlayer/logging"
)

func (a *App) CheckGeneratorDependencies() error {
	return generator.CheckDependencies()
}

// FramePreview wird ans Frontend zurückgegeben - base64-kodiertes PNG statt
// eines Dateipfads, damit das Frontend es direkt in ein <img src="data:...">
// packen kann, ohne sich um file://-Zugriffsrechte kümmern zu müssen.
type FramePreview struct {
	Width  int    `json:"width"`
	Height int    `json:"height"`
	PNGB64 string `json:"pngBase64"`
}

func (a *App) LoadFirstFrame(videoPath string) (FramePreview, error) {
	tmpPNG, err := tempFile("frame-preview-*.png")
	if err != nil {
		return FramePreview{}, err
	}
	defer removeFile(tmpPNG)

	w, h, err := generator.DumpFirstFrame(videoPath, tmpPNG)
	if err != nil {
		return FramePreview{}, err
	}
	b64, err := fileToBase64(tmpPNG)
	if err != nil {
		return FramePreview{}, err
	}
	return FramePreview{Width: w, Height: h, PNGB64: b64}, nil
}

type GenerateOptions struct {
	VideoPath                 string  `json:"videoPath"`
	X                         int     `json:"x"`
	Y                         int     `json:"y"`
	W                         int     `json:"w"`
	H                         int     `json:"h"`
	Invert                    bool    `json:"invert"`
	SmoothWindow              int     `json:"smoothWindow"`
	MinPeakDistanceMs         int     `json:"minPeakDistanceMs"`
	DisableCameraCompensation bool    `json:"disableCameraCompensation"`
	DisableSceneCutDetection  bool    `json:"disableSceneCutDetection"`
	RDPTolerance              float64 `json:"rdpTolerance"`
	PerSceneROI               bool    `json:"perSceneRoi"`
	AdaptiveKeyframeError     float64 `json:"adaptiveKeyframeError"`
	MaxSpeed                  float64 `json:"maxSpeed"`
	Axis                      string  `json:"axis"`
	AutoRetry                 bool    `json:"autoRetry"`
	Backend                   string  `json:"backend"`
	UseOpenCL                 bool    `json:"useOpenCl"`
	DynamicRangeMs            float64 `json:"dynamicRangeMs"`
	Profile                   string  `json:"profile"`
	// Overwrite muss explizit true sein, um eine vorhandene
	// .funscript-Datei zu ersetzen. Ohne das würde der Generator ein
	// von Hand erstelltes oder heruntergeladenes Skript kommentarlos
	// überschreiben - Datenverlust, den man nicht rückgängig machen kann.
	Overwrite bool `json:"overwrite"`
}

// AutoDetectROI sucht die Bewegungsregion automatisch (Optical Flow +
// Kamerabewegungs-Kompensation, siehe generator/auto_roi.py). Läuft in
// einer eigenen Goroutine, da die Analyse einige Sekunden dauert;
// Ergebnis/Fehler kommen über das Event "generate:autoroi".
func (a *App) AutoDetectROI(videoPath string) {
	go func() {
		roi, err := generator.FindROIWithProgress(videoPath,
			func(line string) {
				runtime.EventsEmit(a.ctx, "generate:progress", line)
			},
			func(pct int) {
				runtime.EventsEmit(a.ctx, "generate:percent", pct)
			})
		if err != nil {
			runtime.EventsEmit(a.ctx, "generate:autoroi", map[string]any{"error": err.Error()})
			return
		}
		runtime.EventsEmit(a.ctx, "generate:autoroi", map[string]any{
			"x": roi.X, "y": roi.Y, "w": roi.W, "h": roi.H,
		})
	}()
}

// ScriptExistsForVideo meldet, ob neben dem Video schon ein Skript liegt,
// das eine Generierung überschreiben würde. Das Frontend fragt damit vorher
// nach, statt ungefragt zu ersetzen.
func (a *App) ScriptExistsForVideo(videoPath string) bool {
	_, err := os.Stat(scriptPathForVideo(videoPath))
	return err == nil
}

// GenerateScript läuft in einer eigenen Goroutine, Fortschritt kommt über
// das Event "generate:progress", Abschluss über "generate:done" (mit
// Ergebnispfad oder Fehlermeldung).
func (a *App) GenerateScript(opts GenerateOptions) {
	go func() {
		outPath := scriptPathForVideo(opts.VideoPath)

		if !opts.Overwrite {
			if _, err := os.Stat(outPath); err == nil {
				runtime.EventsEmit(a.ctx, "generate:done", map[string]any{
					"error": "Es existiert bereits ein Skript: " + outPath + " - Generierung abgebrochen, um es nicht zu überschreiben.",
				})
				return
			}
		}

		roi := generator.ROI{X: opts.X, Y: opts.Y, W: opts.W, H: opts.H}
		genOpts := generator.Options{
			PerSceneROI:           opts.PerSceneROI,
			AdaptiveKeyframeError: opts.AdaptiveKeyframeError,
			MaxSpeed:              opts.MaxSpeed,
			Axis:                  opts.Axis,
			AutoRetry:             opts.AutoRetry,
			Backend:               opts.Backend,
			ReportPath:            a.activeReportPath(),
			UseOpenCL:             opts.UseOpenCL,
			DynamicRangeMs:        opts.DynamicRangeMs,
			Profile:               opts.Profile,
			Invert:                    opts.Invert,
			SmoothWindow:              opts.SmoothWindow,
			MinPeakDistanceMs:         opts.MinPeakDistanceMs,
			DisableCameraCompensation: opts.DisableCameraCompensation,
			DisableSceneCutDetection:  opts.DisableSceneCutDetection,
			RDPTolerance:              opts.RDPTolerance,
		}
		err := generator.GenerateWithProgress(opts.VideoPath, roi, outPath, genOpts,
			func(line string) {
				runtime.EventsEmit(a.ctx, "generate:progress", line)
			},
			func(pct int) {
				runtime.EventsEmit(a.ctx, "generate:percent", pct)
			})
		if err != nil {
			runtime.EventsEmit(a.ctx, "generate:done", map[string]any{"error": err.Error()})
			return
		}

		payload := map[string]any{"path": outPath}
		if script, loadErr := funscript.Load(outPath); loadErr == nil && script.Metadata.QualityScore != nil {
			payload["qualityScore"] = *script.Metadata.QualityScore
			payload["qualityWarnings"] = script.Metadata.QualityWarnings
			if script.Metadata.QualityPassed != nil {
				payload["qualityPassed"] = *script.Metadata.QualityPassed
			}
			// Eine dauerhafte Zeile pro erzeugtem Video auf Info-Level: beim
			// Durchtesten mehrerer Videos ist genau das die Information, die
			// man hinterher sucht. Der ausführliche Bericht steht auf
			// Debug-Level daneben (siehe generator.Generate).
			passed := "unbekannt"
			if script.Metadata.QualityPassed != nil {
				passed = strconv.FormatBool(*script.Metadata.QualityPassed)
			}
			logging.Info("generator: Qualitätsbewertung",
				"output", outPath,
				"score", *script.Metadata.QualityScore,
				"bestanden", passed,
				"warnungen", len(script.Metadata.QualityWarnings),
				"actions", len(script.Actions))
			for _, w := range script.Metadata.QualityWarnings {
				logging.Info("generator: Qualitätswarnung", "output", outPath, "warnung", w)
			}
		} else if loadErr != nil {
			logging.Warn("generator: erzeugtes Skript nicht lesbar", "output", outPath, "fehler", loadErr)
		}
		runtime.EventsEmit(a.ctx, "generate:done", payload)
	}()
}

func scriptPathForVideo(videoPath string) string {
	ext := filepath.Ext(videoPath)
	return strings.TrimSuffix(videoPath, ext) + ".funscript"
}

// GetHardwareInfo zeigt im Einstellungen-Tab, welche Beschleunigung
// tatsächlich zur Verfügung steht.
func (a *App) GetHardwareInfo() (string, error) {
	return generator.HardwareInfo()
}
