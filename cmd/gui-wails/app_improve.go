package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/funfunpayer/SamNPlayer/funscript"
	"github.com/funfunpayer/SamNPlayer/generator"
	"github.com/funfunpayer/SamNPlayer/samn"
)

// ImproveScriptRequest is the FunGen-like post-generate polish step:
// trim start/end, fill gaps, optional audio tempo check (warn-only).
type ImproveScriptRequest struct {
	Path      string  `json:"path"`
	VideoPath string  `json:"videoPath"`
	StartSec  float64 `json:"startSec"`
	EndSec    float64 `json:"endSec"`
	FillGaps  bool    `json:"fillGaps"`
	MaxGapMs  int64   `json:"maxGapMs"`
	// AudioCheck re-runs tempo check and stamps metadata (does not rewrite curve).
	AudioCheck bool `json:"audioCheck"`
	// UseAudioForFill uses audio Hz (when available) for fill-gap step spacing.
	UseAudioForFill bool `json:"useAudioForFill"`
}

// ImproveScriptResult is returned to the GUI Review step.
type ImproveScriptResult struct {
	Path          string   `json:"path"`
	BeforeCount   int      `json:"beforeCount"`
	AfterCount    int      `json:"afterCount"`
	Trimmed       bool     `json:"trimmed"`
	GapsFilled    int      `json:"gapsFilled"`
	PointsAdded   int      `json:"pointsAdded"`
	FillGapMs     int64    `json:"fillGapMs"`
	FillStepMs    int64    `json:"fillStepMs"`
	AudioHz       *float64 `json:"audioHz,omitempty"`
	ScriptHz      *float64 `json:"scriptHz,omitempty"`
	AudioWarnings []string `json:"audioWarnings,omitempty"`
	Message       string   `json:"message"`
}

// ImproveGeneratedScript polishes a just-generated script in place
// (.samn preferred, companion .funscript kept in sync).
func (a *App) ImproveGeneratedScript(req ImproveScriptRequest) (ImproveScriptResult, error) {
	out := ImproveScriptResult{}
	if req.Path == "" {
		return out, fmt.Errorf("no path")
	}
	path := preferSamnCompanion(req.Path)
	out.Path = path

	script, err := a.loadScriptDocument(path)
	if err != nil {
		return out, err
	}
	actions := append([]funscript.Action(nil), script.Actions...)

	var audioHz float64
	var audioMeta *funscript.AudioCheck
	video := strings.TrimSpace(req.VideoPath)
	if video == "" {
		video = guessVideoBesideScript(path)
	}
	if (req.AudioCheck || req.UseAudioForFill) && video != "" && generator.AudioCheckAvailable() {
		audioMeta = generator.CheckAudioTempo(video, actions)
		if audioMeta != nil && audioMeta.AudioHz != nil {
			audioHz = *audioMeta.AudioHz
			out.AudioHz = audioMeta.AudioHz
		}
		if audioMeta != nil && audioMeta.ScriptHz != nil {
			out.ScriptHz = audioMeta.ScriptHz
		}
		if audioMeta != nil {
			out.AudioWarnings = append([]string(nil), audioMeta.Warnings...)
		}
	}

	opts := funscript.ImproveOpts{
		FillGaps: req.FillGaps,
		MaxGapMs: req.MaxGapMs,
	}
	if req.StartSec > 0 {
		opts.StartMs = int64(req.StartSec * 1000)
	}
	if req.EndSec > 0 {
		opts.EndMs = int64(req.EndSec * 1000)
	}
	if req.UseAudioForFill && audioHz > 0 {
		opts.AudioHz = audioHz
	}

	improved, err := funscript.ImproveScript(actions, opts)
	if err != nil {
		return out, err
	}
	out.BeforeCount = improved.BeforeCount
	out.AfterCount = improved.AfterCount
	out.Trimmed = improved.Trimmed
	out.GapsFilled = improved.GapsFilled
	out.PointsAdded = improved.PointsAdded
	out.FillGapMs = improved.FillGapMs
	out.FillStepMs = improved.FillStepMs

	changed := improved.Trimmed || improved.PointsAdded > 0
	if changed {
		if err := saveImprovedActions(path, improved.Actions); err != nil {
			return out, err
		}
		funPath := path
		if samn.IsSamnPath(path) {
			funPath = samn.CompanionFunscriptPath(path)
		}
		if req.AudioCheck && audioMeta != nil {
			_ = funscript.StampAudioCheck(funPath, audioMeta)
		}
		loaded := a.loadedScriptPath()
		if loaded == path || loaded == funPath || loaded == req.Path {
			_ = a.reloadLoadedScript()
		}
	} else if req.AudioCheck && audioMeta != nil {
		funPath := path
		if samn.IsSamnPath(path) {
			funPath = samn.CompanionFunscriptPath(path)
		}
		if err := funscript.StampAudioCheck(funPath, audioMeta); err != nil && !samn.IsSamnPath(path) {
			return out, err
		}
	}

	parts := []string{}
	if improved.Trimmed {
		parts = append(parts, "trimmed start/end")
	}
	if improved.GapsFilled > 0 {
		parts = append(parts, fmt.Sprintf("filled %d gap(s) (+%d points)", improved.GapsFilled, improved.PointsAdded))
	}
	if req.AudioCheck && audioMeta != nil {
		if len(audioMeta.Warnings) > 0 {
			parts = append(parts, "audio check: warnings")
		} else {
			parts = append(parts, "audio check: ok")
		}
	}
	if len(parts) == 0 {
		out.Message = "Nothing to change"
	} else {
		out.Message = strings.Join(parts, " · ")
	}
	return out, nil
}

func saveImprovedActions(path string, actions []funscript.Action) error {
	if samn.IsSamnPath(path) {
		doc, err := samn.Load(path)
		if err != nil {
			return err
		}
		doc.General = actions
		if err := samn.Save(path, doc); err != nil {
			return err
		}
		return doc.ExportFunscript(samn.CompanionFunscriptPath(path))
	}
	return funscript.SaveActions(path, actions)
}

func guessVideoBesideScript(scriptPath string) string {
	base := scriptPath
	for _, ext := range []string{".samn", ".funscript", filepath.Ext(scriptPath)} {
		base = strings.TrimSuffix(base, ext)
	}
	for _, ext := range []string{".mp4", ".mkv", ".webm", ".mov", ".avi"} {
		cand := base + ext
		if st, err := os.Stat(cand); err == nil && !st.IsDir() {
			return cand
		}
	}
	return ""
}
