// Package funscript implementiert einen Parser für das .funscript-Dateiformat.
package funscript

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
)

// Action ist ein einzelner Punkt im Skript.
type Action struct {
	At  int64 `json:"at"`
	Pos int   `json:"pos"`
}

// Script ist das geparste .funscript-Dokument.
type Script struct {
	Actions  []Action `json:"actions"`
	Metadata struct {
		Duration        int64         `json:"duration"`
		Creator         string        `json:"creator"`
		QualityScore    *float64      `json:"quality_score,omitempty"`
		QualityPassed   *bool         `json:"quality_passed,omitempty"`
		QualityWarnings []string      `json:"quality_warnings,omitempty"`
		Profile         string        `json:"profile,omitempty"`
		DeviceRecipe    *DeviceRecipe `json:"device_recipe,omitempty"`
		SamnAxes        *SamnAxes     `json:"samn_axes,omitempty"`
		TrackingGaps    []TrackingGap `json:"tracking_gaps,omitempty"`
		AIOpinion       *AIOpinion    `json:"ai_opinion,omitempty"`
		AudioCheck      *AudioCheck   `json:"audio_check,omitempty"`
		// Trajectory: optional per-frame tip/partner track positions
		// (MT-Debug Review/Play overlay). Only present when generated
		// with the opt-in "capture trajectory" flag - off by default.
		Trajectory *TrajectoryData `json:"trajectory,omitempty"`
		// ContactMarks: optional tip/contact boxes + classes from Generate
		// when Contact vibration is on. Everyday stroke curve stays tip-CSRT
		// (drive_stroke=false); marks are for feel / later feel-decouple.
		ContactMarks *ContactMarks `json:"contact_marks,omitempty"`
	} `json:"metadata,omitempty"`
}

// TrajectoryPoint is one sampled tip/partner position in video-pixel space
// (0,0 = top-left), timestamped against the same clock as Actions.
type TrajectoryPoint struct {
	AtMs int64   `json:"atMs"`
	X    float64 `json:"x"`
	Y    float64 `json:"y"`
}

// TrajectoryData is the optional MT-Debug trajectory payload: Width/Height
// are the video's pixel dimensions at generation time (the coordinate
// space Tip/Partner points are in). Partner is empty when the script has
// no second tracked point (single-ROI Stroke) or when generated against
// 2+ contact partners (ambiguous "the" partner).
type TrajectoryData struct {
	Width   int               `json:"width"`
	Height  int               `json:"height"`
	Tip     []TrajectoryPoint `json:"tip,omitempty"`
	Partner []TrajectoryPoint `json:"partner,omitempty"`
}

// AIOpinion ist die optionale KI-Zweitmeinung zur Qualität (--ai-quality-
// opinion, generator/ai_quality.py) - rein informativ, verändert
// QualityScore/QualityPassed nicht.
type AIOpinion struct {
	Verdict string `json:"verdict"`
	Reason  string `json:"reason"`
}

// AudioCheck ist das optionale Ergebnis der Audio-Tempo-Plausibilitätsprüfung
// (--audio-check; Go: generator.CheckAudioTempo, Python: audio_check.py) -
// wie AIOpinion rein informativ, verändert QualityScore/QualityPassed nicht.
// Nur vorhanden, wenn die Prüfung tatsächlich lief (ffmpeg + lesbare Audiospur).
type AudioCheck struct {
	ScriptHz *float64 `json:"script_hz"`
	AudioHz  *float64 `json:"audio_hz"`
	Warnings []string `json:"warnings"`
}

func Load(path string) (*Script, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("funscript: Datei konnte nicht gelesen werden: %w", err)
	}
	return Parse(data)
}

func Parse(data []byte) (*Script, error) {
	var s Script
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("funscript: ungültiges JSON: %w", err)
	}
	if len(s.Actions) == 0 {
		return nil, fmt.Errorf("funscript: keine actions im Skript gefunden")
	}
	for i := range s.Actions {
		if s.Actions[i].Pos < 0 {
			s.Actions[i].Pos = 0
		} else if s.Actions[i].Pos > 100 {
			s.Actions[i].Pos = 100
		}
	}
	sort.Slice(s.Actions, func(i, j int) bool {
		return s.Actions[i].At < s.Actions[j].At
	})
	return &s, nil
}

func (s *Script) Duration() int64 {
	if len(s.Actions) == 0 {
		return 0
	}
	return s.Actions[len(s.Actions)-1].At
}
