package main

import (
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
}

func (a *App) AutoDetectROI(videoPath string) {
	go func() {
		roi, err := generator.FindROIWithProgress(videoPath,
			func(line string) { runtime.EventsEmit(a.ctx, "generate:progress", line) },
			func(pct int) { runtime.EventsEmit(a.ctx, "generate:percent", pct) })
		if err != nil {
			runtime.EventsEmit(a.ctx, "generate:autoroi", map[string]any{"error": err.Error()})
			return
		}
		runtime.EventsEmit(a.ctx, "generate:autoroi", map[string]any{
			"x": roi.X, "y": roi.Y, "w": roi.W, "h": roi.H,
		})
	}()
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
		}
		if opts.W2 > 0 && opts.H2 > 0 {
			genOpts.ROI2 = generator.ROI{X: opts.X2, Y: opts.Y2, W: opts.W2, H: opts.H2}
		}
		err := generator.GenerateWithProgress(opts.VideoPath, roi, outPath, genOpts,
			func(line string) { runtime.EventsEmit(a.ctx, "generate:progress", line) },
			func(pct int) { runtime.EventsEmit(a.ctx, "generate:percent", pct) })
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
			logging.Info("generator: Qualitätsbewertung", "output", outPath, "score", *script.Metadata.QualityScore)
		} else if loadErr != nil {
			logging.Warn("generator: erzeugtes Skript nicht lesbar", "output", outPath, "fehler", loadErr)
		}
		runtime.EventsEmit(a.ctx, "generate:done", payload)
	}()
}

func scriptPathForVideo(videoPath string) string {
	return strings.TrimSuffix(videoPath, filepath.Ext(videoPath)) + ".funscript"
}

func (a *App) GetHardwareInfo() (string, error) {
	return generator.HardwareInfo()
}
