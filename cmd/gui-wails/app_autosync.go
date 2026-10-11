package main

import (
	"context"
	"fmt"
	"time"

	"github.com/funfunpayer/SamNPlayer/funscript"
)

// AudioSyncSuggestion is the result of estimating a script offset from the
// video's sound (V1 S8 Auto sync). Suggest-only: the user confirms before
// it is applied.
type AudioSyncSuggestion struct {
	OffsetMs     int64   `json:"offsetMs"`     // positive = script later
	Correlation  float64 `json:"correlation"`  // Pearson r of the best alignment
	Confidence   string  `json:"confidence"`   // "high" / "low" / "none"
	Note         string  `json:"note"`
}

// EstimateAudioSyncOffset estimates a timing offset that aligns the loaded
// script's stroke curve with the video's audio loudness envelope.
// It is a suggestion only — nothing is written until the user applies it
// (via the existing offset controls or BakeScriptOffset).
func (a *App) EstimateAudioSyncOffset() (AudioSyncSuggestion, error) {
	script := a.loadedScript()
	if script == nil {
		return AudioSyncSuggestion{}, fmt.Errorf("no script loaded")
	}
	video := a.currentVideoPath()
	if video == "" {
		return AudioSyncSuggestion{}, fmt.Errorf("no video linked")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	env, err := a.audioVibrationEnvelope(ctx, video, audioLevelStepMs)
	if err != nil {
		return AudioSyncSuggestion{}, err
	}
	if len(env) < 20 {
		return AudioSyncSuggestion{Confidence: "none", Note: "audio too short or quiet"}, nil
	}

	// Convert envelope to actions (pos = loudness 0–100).
	audioActs := make([]funscript.Action, 0, len(env))
	for i, v := range env {
		audioActs = append(audioActs, funscript.Action{
			At:  int64(i) * audioLevelStepMs,
			Pos: int(v * 100),
		})
	}

	scriptActs := script.PlaybackActions()
	if len(scriptActs) < 10 {
		return AudioSyncSuggestion{Confidence: "none", Note: "script too short"}, nil
	}

	// Search ±2 s at 50 ms steps (same resolution as the envelope).
	corr := funscript.BestLagCorrelation(audioActs, scriptActs, 2000, 50, 50)
	if corr == nil || corr.R < 0.15 {
		return AudioSyncSuggestion{
			Confidence: "none",
			Note:       "no clear alignment found — try adjusting offset by hand",
		}, nil
	}

	// LagMs from BestLagCorrelation is the shift applied to the second series
	// (script) so that it matches the first (audio). Positive lag means the
	// script must be delayed (offset positive).
	conf := "low"
	if corr.R >= 0.45 && !corr.LowConfidence {
		conf = "high"
	} else if corr.R >= 0.25 {
		conf = "medium"
	}

	return AudioSyncSuggestion{
		OffsetMs:    int64(corr.LagMs),
		Correlation: corr.R,
		Confidence:  conf,
		Note:        fmt.Sprintf("suggested offset %+d ms (r=%.2f)", corr.LagMs, corr.R),
	}, nil
}
