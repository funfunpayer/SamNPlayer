package main

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/funfunpayer/SamNPlayer/funscript"
	"github.com/funfunpayer/SamNPlayer/generator"
	"github.com/funfunpayer/SamNPlayer/samn"
)

// WriteVibrationFromSound writes the video's loudness as the Vibration
// channel of the loaded script. mode is "sound" (replace) or "sound+contact"
// (pointwise max with the existing vibration). Used by the Generator for
// vibrators after Create. Requires a linked video with an audio track.
func (a *App) WriteVibrationFromSound(mode string) error {
	if mode != "sound" && mode != "sound+contact" {
		return fmt.Errorf("unknown vibration mode %q", mode)
	}
	path := a.loadedScriptPath()
	if path == "" || a.loadedScript() == nil {
		return fmt.Errorf("no script loaded")
	}
	video := a.currentVideoPath()
	if video == "" {
		return fmt.Errorf("no video linked — sound vibration needs the video's audio track")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	env, err := a.audioVibrationEnvelope(ctx, video, 50)
	if err != nil {
		return fmt.Errorf("sound envelope: %w", err)
	}
	if len(env) < 2 {
		return fmt.Errorf("sound too short for a vibration curve")
	}

	acts := envelopeToVibrationActions(env, 50)
	if len(acts) < 2 {
		return fmt.Errorf("sound too short for a vibration curve")
	}

	if mode == "sound+contact" {
		existing, err := a.GetScriptAxisActions("vibration")
		if err == nil && len(existing) > 0 {
			acts = maxVibrationActions(acts, existing)
		}
	}

	if err := a.SaveScriptAxisActions("vibration", acts); err != nil {
		return err
	}
	// Ensure axes mode for .samn so the written curve is used.
	if samn.IsSamnPath(path) {
		if err := a.SetPlaybackSource(funscript.PlaybackSourceAxes); err != nil {
			// non-fatal if already set by SaveScriptAxisActions
			_ = err
		}
	}
	return a.reloadLoadedScript()
}

// envelopeToVibrationActions thins a 0..1 loudness envelope (stepMs) to
// ~4 points/sec, matching the Player's Sound Curve writer.
func envelopeToVibrationActions(env []float64, stepMs int64) []funscript.Action {
	if stepMs <= 0 {
		stepMs = 50
	}
	win := int(math.Max(1, math.Round(250.0/float64(stepMs))))
	var raw []funscript.Action
	for i := 0; i < len(env); i += win {
		sum, n := 0.0, 0
		end := i + win
		if end > len(env) {
			end = len(env)
		}
		for k := i; k < end; k++ {
			sum += env[k]
			n++
		}
		at := int64(i) * stepMs
		pos := int(math.Round((sum / float64(n)) * 100))
		if pos < 0 {
			pos = 0
		}
		if pos > 100 {
			pos = 100
		}
		raw = append(raw, funscript.Action{At: at, Pos: pos})
	}
	if len(raw) == 0 {
		return nil
	}
	out := []funscript.Action{raw[0]}
	for _, p := range raw[1:] {
		last := out[len(out)-1]
		if math.Abs(float64(p.Pos-last.Pos)) >= 4 || p.At-last.At >= 2000 {
			out = append(out, p)
		}
	}
	if out[len(out)-1] != raw[len(raw)-1] {
		out = append(out, raw[len(raw)-1])
	}
	return out
}

// maxVibrationActions returns the pointwise maximum of two vibration curves
// (sound and contact), sampled at the union of their timestamps.
func maxVibrationActions(a, b []funscript.Action) []funscript.Action {
	if len(a) == 0 {
		return b
	}
	if len(b) == 0 {
		return a
	}
	times := map[int64]int{}
	for _, p := range a {
		times[p.At] = p.Pos
	}
	for _, p := range b {
		if cur, ok := times[p.At]; !ok || p.Pos > cur {
			times[p.At] = p.Pos
		}
	}
	out := make([]funscript.Action, 0, len(times))
	for at, pos := range times {
		out = append(out, funscript.Action{At: at, Pos: pos})
	}
	for i := 1; i < len(out); i++ {
		j := i
		for j > 0 && out[j].At < out[j-1].At {
			out[j], out[j-1] = out[j-1], out[j]
			j--
		}
	}
	return out
}
