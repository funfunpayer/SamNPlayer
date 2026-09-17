package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/funfunpayer/SamNPlayer/funscript"
	"github.com/funfunpayer/SamNPlayer/generator"
	"github.com/funfunpayer/SamNPlayer/logging"
)

func (a *App) CheckGeneratorDependencies() error {
	return generator.CheckDependencies()
}

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
	X2                        int     `json:"x2"`
	Y2                        int     `json:"y2"`
	W2                        int     `json:"w2"`
	H2                        int     `json:"h2"`
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
	Overwrite                 bool    `json:"overwrite"`
	AIQualityOpinion          bool    `json:"aiQualityOpinion"`
	ContactVibration          bool    `json:"contactVibration"`
	AutoOZoneMarker           bool    `json:"autoOZoneMarker"`
	AudioCheck                bool    `json:"audioCheck"`
	NativePipeline            bool    `json:"nativePipeline"`
}

// AutoDetectROI sucht die Region automatisch. engine "ai" nutzt den lokalen
// ONNX-Objekterkenner (ai_roi.py, siehe docs/AI_ADAPTER.md); jeder andere
// Wert (leer, "auto", ...) bleibt bei der klassischen Rhythmus-Heuristik
// (auto_roi.py) - so bricht ein alter Frontend-Aufruf ohne engine-Argument
// nicht, er bekommt nur weiterhin das klassische Verhalten.
func (a *App) AutoDetectROI(videoPath string, engine string) {
	go func() {
		var roi generator.ROI
		var err error
		if engine == "ai" {
			roi, err = generator.FindROIAIWithProgress(videoPath, a.settings.GetString(prefAIRoiModelPath, ""),
				func(line string) { runtime.EventsEmit(a.ctx, "generate:progress", line) },
				func(pct int) { runtime.EventsEmit(a.ctx, "generate:percent", pct) })
		} else {
			roi, err = generator.FindROIWithProgress(videoPath,
				func(line string) { runtime.EventsEmit(a.ctx, "generate:progress", line) },
				func(pct int) { runtime.EventsEmit(a.ctx, "generate:percent", pct) })
		}
		if err != nil {
			runtime.EventsEmit(a.ctx, "generate:autoroi", map[string]any{"error": err.Error()})
			return
		}
		runtime.EventsEmit(a.ctx, "generate:autoroi", map[string]any{
			"x": roi.X, "y": roi.Y, "w": roi.W, "h": roi.H, "engine": engine,
		})
	}()
}

// CheckAIRoiAvailable meldet, ob die KI-Regionssuche grundsätzlich nutzbar
// ist (onnxruntime installiert + Modell vorhanden) - ohne selbst ein Video zu
// öffnen. Die GUI nutzt das, um den KI-Knopf zu aktivieren/auszublenden statt
// ihn anzubieten und dann bei jedem Versuch scheitern zu lassen.
func (a *App) CheckAIRoiAvailable() bool {
	return generator.AIRoiAvailable(a.settings.GetString(prefAIRoiModelPath, ""))
}

// CheckAudioCheckAvailable meldet, ob --audio-check grundsätzlich nutzbar
// ist (ffmpeg auf dem PATH) - die GUI nutzt das, um die Checkbox zu
// aktivieren/auszublenden statt sie anzubieten und dann scheitern zu lassen.
func (a *App) CheckAudioCheckAvailable() bool {
	return generator.AudioCheckAvailable()
}

// SuggestProfile vergleicht die Bewegungssignatur des Videos gegen zuvor mit
// LabelScene benannte Szenen und, falls keine nah genug ist, gegen einen
// optionalen Colibri-Server. Found=false ist ein normales Ergebnis (kein
// Fehler) - keine passende Szene, kein KI-Server erreichbar.
func (a *App) SuggestProfile(videoPath string) (generator.ProfileSuggestion, error) {
	return generator.SuggestProfile(videoPath, a.settings.GetString(prefAIBaseURL, ""))
}

// LabelScene merkt sich die Bewegungssignatur des Videos unter label, damit
// spätere ähnliche Szenen darüber ein Profil vorgeschlagen bekommen
// (SuggestProfile).
func (a *App) LabelScene(videoPath, label string) error {
	return generator.LabelScene(videoPath, label)
}

func (a *App) ScriptExistsForVideo(videoPath string) bool {
	_, err := os.Stat(scriptPathForVideo(videoPath))
	return err == nil
}

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
			PerSceneROI:               opts.PerSceneROI,
			AdaptiveKeyframeError:     opts.AdaptiveKeyframeError,
			MaxSpeed:                  opts.MaxSpeed,
			Axis:                      opts.Axis,
			AutoRetry:                 opts.AutoRetry,
			Backend:                   opts.Backend,
			ReportPath:                a.activeReportPath(),
			UseOpenCL:                 opts.UseOpenCL,
			DynamicRangeMs:            opts.DynamicRangeMs,
			Profile:                   opts.Profile,
			Invert:                    opts.Invert,
			SmoothWindow:              opts.SmoothWindow,
			MinPeakDistanceMs:         opts.MinPeakDistanceMs,
			DisableCameraCompensation: opts.DisableCameraCompensation,
			DisableSceneCutDetection:  opts.DisableSceneCutDetection,
			RDPTolerance:              opts.RDPTolerance,
			AIQualityOpinion:          opts.AIQualityOpinion,
			AIBaseURL:                 a.settings.GetString(prefAIBaseURL, ""),
			ContactVibration:          opts.ContactVibration,
			AudioCheck:                opts.AudioCheck,
			NativePipeline:            opts.NativePipeline,
		}
		if opts.W2 > 0 && opts.H2 > 0 {
			genOpts.ROI2 = generator.ROI{X: opts.X2, Y: opts.Y2, W: opts.W2, H: opts.H2}
		}
		ctx, cancel := context.WithCancel(context.Background())
		a.stateMu.Lock()
		prev := a.genCancel
		a.genSeq++
		mySeq := a.genSeq
		a.genCancel = cancel
		a.stateMu.Unlock()
		if prev != nil {
			prev()
		}
		defer func() {
			a.stateMu.Lock()
			if a.genSeq == mySeq {
				a.genCancel = nil
			}
			a.stateMu.Unlock()
			cancel()
		}()
		err := generator.GenerateWithContext(ctx, opts.VideoPath, roi, outPath, genOpts,
			func(line string) { runtime.EventsEmit(a.ctx, "generate:progress", line) },
			func(pct int) { runtime.EventsEmit(a.ctx, "generate:percent", pct) })
		if err != nil {
			if err == context.Canceled {
				runtime.EventsEmit(a.ctx, "generate:done", map[string]any{"error": "Generierung abgebrochen", "cancelled": true})
				return
			}
			runtime.EventsEmit(a.ctx, "generate:done", map[string]any{"error": err.Error()})
			return
		}
		payload := map[string]any{"path": outPath}
		if script, loadErr := funscript.Load(outPath); loadErr == nil {
			if script.Metadata.QualityScore != nil {
				payload["qualityScore"] = *script.Metadata.QualityScore
				payload["qualityWarnings"] = script.Metadata.QualityWarnings
				if script.Metadata.QualityPassed != nil {
					payload["qualityPassed"] = *script.Metadata.QualityPassed
				}
				logging.Info("generator: Qualitätsbewertung", "output", outPath, "score", *script.Metadata.QualityScore)
				if script.Metadata.AIOpinion != nil {
					payload["aiOpinionVerdict"] = script.Metadata.AIOpinion.Verdict
					payload["aiOpinionReason"] = script.Metadata.AIOpinion.Reason
				}
				if script.Metadata.AudioCheck != nil {
					payload["audioCheckWarnings"] = script.Metadata.AudioCheck.Warnings
				}
			}
			if opts.AutoOZoneMarker {
				zone, err := applyAutoOZoneMarker(outPath, script.Actions)
				if err != nil {
					logging.Warn("generator: O-Marker konnte nicht gespeichert werden", "output", outPath, "fehler", err)
				} else if zone.OK {
					payload["oZoneMarkerStartMs"] = zone.StartMs
					payload["oZoneMarkerEndMs"] = zone.EndMs
				}
			}
		} else {
			logging.Warn("generator: erzeugtes Skript nicht lesbar", "output", outPath, "fehler", loadErr)
		}
		runtime.EventsEmit(a.ctx, "generate:done", payload)
	}()
}

// applyAutoOZoneMarker schreibt einen automatisch aus dem Positionssignal
// vorgeschlagenen O-Marker (funscript.SuggestOZone, letztes Achtel mit der
// höchsten mittleren Position) in die gerade erzeugte Datei, falls die
// Suggestion greift - klassisch aus dem Signal, kein eigenes KI-Modell
// (docs/NEXT.md Priorität 6/7). Eigene Funktion statt Inline-Code in
// GenerateScript, damit sie ohne echten Video-Generierungslauf testbar ist.
// Schreibt zusätzlich bis zu zwei schwächere, frühere Marker, falls das
// Signal welche hergibt (funscript.SuggestSecondaryOZones) - derselbe
// "primary + optionale sekundäre" Vorschlag wie beim manuellen
// "O-Zone vorschlagen"-Knopf (app_suggest.go), hier am Generierungspfad.
func applyAutoOZoneMarker(outPath string, actions []funscript.Action) (funscript.OZoneSuggestion, error) {
	zone := funscript.SuggestOZone(actions)
	if !zone.OK {
		return zone, nil
	}
	markers := []funscript.OMarker{{
		StartMs: zone.StartMs, EndMs: zone.EndMs,
		Kind: funscript.OMarkerPrimary, Intensity: 1,
	}}
	for _, sec := range funscript.SuggestSecondaryOZones(actions, zone, 2) {
		markers = append(markers, secondaryMarkerFromSuggestion(sec, zone))
	}
	return zone, funscript.SaveOMarkers(outPath, markers)
}

func scriptPathForVideo(videoPath string) string {
	return strings.TrimSuffix(videoPath, filepath.Ext(videoPath)) + ".funscript"
}

func (a *App) GetHardwareInfo() (string, error) {
	return generator.HardwareInfo()
}

// CancelGenerate bricht die laufende Generierung ab (GenerateWithContext).
// Idempotent, wenn gerade nichts läuft.
func (a *App) CancelGenerate() {
	a.stateMu.Lock()
	cancel := a.genCancel
	a.stateMu.Unlock()
	if cancel != nil {
		cancel()
	}
}
