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
	Actions []Action `json:"actions"`
	// Inverted is the official funscript top-level flag: players flip
	// positions (100−pos) at playback. Actions on disk stay as written.
	// Review "Invert" bakes 100−pos into actions and clears this flag
	// so playback does not flip twice.
	Inverted bool `json:"inverted,omitempty"`
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
		Trajectory      *TrajectoryData `json:"trajectory,omitempty"`
		ContactMarks    *ContactMarks   `json:"contact_marks,omitempty"`
	} `json:"metadata,omitempty"`
}

type TrajectoryPoint struct {
	AtMs int64   `json:"atMs"`
	X    float64 `json:"x"`
	Y    float64 `json:"y"`
}

type TrajectoryData struct {
	Width   int               `json:"width"`
	Height  int               `json:"height"`
	Tip     []TrajectoryPoint `json:"tip,omitempty"`
	Partner []TrajectoryPoint `json:"partner,omitempty"`
}

type AIOpinion struct {
	Verdict string `json:"verdict"`
	Reason  string `json:"reason"`
}

type AudioCheck struct {
	ScriptHz     *float64           `json:"script_hz"`
	AudioHz      *float64           `json:"audio_hz"`
	Warnings     []string           `json:"warnings"`
	Segments     []AudioSegmentHint `json:"segments,omitempty"`
	SpeechHoldMs int64              `json:"speech_hold_ms,omitempty"`
}

type AudioSegmentHint struct {
	Label      string `json:"label"`
	StartMs    int64  `json:"start_ms"`
	EndMs      int64  `json:"end_ms"`
	SpeechHold bool   `json:"speech_hold,omitempty"`
	Reason     string `json:"reason,omitempty"`
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

// PlaybackPos is the position the device should use at action i.
// When Inverted is set, that is 100−pos; the stored action is unchanged.
func (s *Script) PlaybackPos(i int) int {
	if s == nil || i < 0 || i >= len(s.Actions) {
		return 0
	}
	pos := s.Actions[i].Pos
	if s.Inverted {
		pos = 100 - pos
	}
	if pos < 0 {
		return 0
	}
	if pos > 100 {
		return 100
	}
	return pos
}

// WriteInverted sets the top-level inverted flag without touching actions.
func WriteInverted(path string, inverted bool) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("funscript: Datei konnte nicht gelesen werden: %w", err)
	}
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(data, &doc); err != nil {
		return fmt.Errorf("funscript: ungültiges JSON: %w", err)
	}
	if !inverted {
		delete(doc, "inverted")
	} else {
		b, err := json.Marshal(true)
		if err != nil {
			return err
		}
		doc["inverted"] = b
	}
	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, out, 0o644)
}
