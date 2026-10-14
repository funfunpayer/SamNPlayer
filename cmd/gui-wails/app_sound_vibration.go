package main

import (
	"context"
	"fmt"
	"math"

	"github.com/funfunpayer/SamNPlayer/funscript"
	"github.com/funfunpayer/SamNPlayer/sam"
)

// Vibration from the video's sound, written into the script (Owner 14 Oct,
// step 3 of the device-aware Generator): for vibrators and other toys whose
// feel comes from vibration. Until now the sound drove vibration only at
// play time (Player → Audio); this writes it as the script's vibration axis,
// so it plays the same everywhere and can be edited like any lane.

const (
	SoundVibSoundOnly    = "sound"         // vibration = the video's loudness
	SoundVibSoundContact = "sound+contact" // the stronger of loudness and the script's own vibration
)

// soundVibStepMs: envelope resolution; soundVibPointMs: written point spacing.
const (
	soundVibStepMs  = 50
	soundVibPointMs = 100
)

// SoundVibrationResult reports what was written.
type SoundVibrationResult struct {
	Points  int    `json:"points"`
	Message string `json:"message"`
}

// BakeSoundVibration writes vibration from the loaded video's sound into the
// loaded script ("sound" or "sound+contact"). Suction is kept: when the
// script has no explicit suction lane yet, its mapped suction is written
// first so playing from lanes feels the same.
func (a *App) BakeSoundVibration(mode string) (SoundVibrationResult, error) {
	out := SoundVibrationResult{}
	if mode != SoundVibSoundOnly && mode != SoundVibSoundContact {
		return out, fmt.Errorf("unknown vibration source %q", mode)
	}
	script := a.loadedScript()
	if script == nil || a.loadedScriptPath() == "" {
		return out, fmt.Errorf("no script loaded")
	}
	video := a.currentVideoPath()
	if video == "" {
		return out, fmt.Errorf("no video — the sound comes from the script's video")
	}
	ctx, cancel := context.WithTimeout(context.Background(), audioVibTimeout)
	defer cancel()
	env, err := a.audioVibrationEnvelope(ctx, video, soundVibStepMs)
	if err != nil {
		return out, fmt.Errorf("sound: %w", err)
	}
	opts := funscript.MapOptionsFromScript(script)
	frames := script.ToIntensityCurve(opts)
	if len(frames) == 0 {
		return out, fmt.Errorf("the script has no points")
	}
	a.stateMu.RLock()
	offset := a.scriptOffsetMs
	a.stateMu.RUnlock()
	sound := sam.ApplyAudioVibration(frames, env, soundVibStepMs, offset, 1)

	vib := make([]funscript.Frame, len(sound))
	for i, f := range sound {
		v := f.Vibration
		if mode == SoundVibSoundContact && i < len(frames) && frames[i].Vibration > v {
			v = frames[i].Vibration
		}
		vib[i] = funscript.Frame{At: f.At, Vibration: v}
	}

	if script.Metadata.SamnAxes == nil || len(script.Metadata.SamnAxes.Suction) == 0 {
		suc := framesToLane(frames, false)
		if err := a.SaveScriptAxisActions(string(funscript.AxisSuction), suc); err != nil {
			return out, fmt.Errorf("keep suction: %w", err)
		}
	}
	lane := framesToLane(vib, true)
	if err := a.SaveScriptAxisActions(string(funscript.AxisVibration), lane); err != nil {
		return out, err
	}
	out.Points = len(lane)
	what := "the video's sound"
	if mode == SoundVibSoundContact {
		what = "the video's sound and contact"
	}
	out.Message = fmt.Sprintf("Vibration from %s written (%d points).", what, len(lane))
	return out, nil
}

// framesToLane samples frames every soundVibPointMs into a 0–100 lane and
// drops points that repeat the previous level (keeps the last of a run).
func framesToLane(frames []funscript.Frame, vibration bool) []funscript.Action {
	var out []funscript.Action
	var next int64 = math.MinInt64
	for i, f := range frames {
		if f.At < next && i != len(frames)-1 {
			continue
		}
		next = f.At + soundVibPointMs
		v := f.Suction
		if vibration {
			v = f.Vibration
		}
		pos := int(math.Round(math.Max(0, math.Min(1, v)) * 100))
		if n := len(out); n >= 2 && out[n-1].Pos == pos && out[n-2].Pos == pos {
			out[n-1].At = f.At
			continue
		}
		out = append(out, funscript.Action{At: f.At, Pos: pos})
	}
	return out
}
