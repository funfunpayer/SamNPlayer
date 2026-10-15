package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/funfunpayer/SamNPlayer/funscript"
	"github.com/funfunpayer/SamNPlayer/generator"
	"github.com/funfunpayer/SamNPlayer/videox"
)

// Cock Hero / Fap Hero beat bar (Owner 15 Oct): read the strip of scrolling
// dots the video shows and write one stroke per beat. The result is a new
// file "<video> (Beat bar).funscript" beside the video, so it shows up as a
// script variant and never overwrites an existing script.

// Swapped in tests (no ffmpeg needed).
var (
	detectBeatBarFn = generator.DetectBeatBarBeats
	probeVideoFn    = videox.Probe
)

const beatBarTimeout = 20 * time.Minute

// BeatBarScriptResult is what CreateBeatBarScript did.
type BeatBarScriptResult struct {
	Found      bool    `json:"found"`
	ScriptPath string  `json:"scriptPath"`
	Beats      int     `json:"beats"`
	BPM        float64 `json:"bpm"`
	Message    string  `json:"message"`
}

// CreateBeatBarScript looks for a beat bar in videoPath ("" = the loaded
// video) and writes a stroke script from its beats. Found is false (no
// error) when the video has no beat bar.
func (a *App) CreateBeatBarScript(videoPath string) (BeatBarScriptResult, error) {
	var res BeatBarScriptResult
	if videoPath == "" {
		videoPath = a.currentVideoPath()
	}
	if videoPath == "" {
		return res, fmt.Errorf("no video loaded")
	}
	if _, err := os.Stat(videoPath); err != nil {
		return res, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), beatBarTimeout)
	defer cancel()
	var durSec float64
	if info, err := probeVideoFn(ctx, videoPath); err == nil {
		durSec = info.Duration.Seconds()
	}
	bb, err := detectBeatBarFn(ctx, videoPath, durSec)
	if err != nil {
		return res, err
	}
	if !bb.Found || len(bb.BeatsMs) < 2 {
		res.Message = "No beat bar found in this video (a strip of dots scrolling to a target)."
		return res, nil
	}
	actions := funscript.BeatStrokes(bb.BeatsMs, funscript.BeatStrokeOptions{})
	path := beatBarScriptPath(videoPath)
	doc := map[string]any{
		"version":  "1.0",
		"inverted": false,
		"range":    100,
		"actions":  actions,
		"metadata": map[string]any{
			"creator": "SamNPlayer",
			"notes":   fmt.Sprintf("Beat bar: %d beats, about %.0f BPM", len(bb.BeatsMs), bb.BPM),
		},
	}
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return res, err
	}
	if err := funscript.WriteFileAtomic(path, data, 0o644); err != nil {
		return res, err
	}
	res.Found, res.ScriptPath, res.Beats, res.BPM = true, path, len(bb.BeatsMs), bb.BPM
	res.Message = fmt.Sprintf("Beat bar read: %d beats, about %.0f BPM.", res.Beats, res.BPM)
	return res, nil
}

// beatBarScriptPath: "<stem> (Beat bar).funscript", then "(Beat bar 2)" …
// so an earlier result or the author's script is never overwritten.
func beatBarScriptPath(videoPath string) string {
	stem := strings.TrimSuffix(videoPath, filepath.Ext(videoPath))
	for n := 1; ; n++ {
		label := "Beat bar"
		if n > 1 {
			label = fmt.Sprintf("Beat bar %d", n)
		}
		p := stem + " (" + label + ").funscript"
		if _, err := os.Stat(p); os.IsNotExist(err) {
			return p
		}
	}
}
