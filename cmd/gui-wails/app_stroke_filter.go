package main

import (
	"github.com/funfunpayer/SamNPlayer/funscript"
)

// Stroke filter for strokers (v0.8, docs/ROADMAP_2026-10.md §3 item 2):
// smooth curves, a speed limit, filled pauses and a slow return home on
// pause. It works on what the device gets while playing; the script file is
// never changed (funscript.ApplyStrokeFilter).

const (
	prefStrokeSmooth   = "playback.stroke_smooth"
	prefStrokeMaxSpeed = "playback.stroke_max_speed"
	prefStrokeIdleFill = "playback.stroke_idle_fill"
	prefStrokeHome     = "playback.stroke_home_on_pause"
)

// strokeIdleFillMs: script pauses at least this long get the gentle wave.
const strokeIdleFillMs = 4000

// StrokeFilterSettings is the GUI's view of the filter.
type StrokeFilterSettings struct {
	Smooth      string  `json:"smooth"`      // "", "pchip", "makima"
	MaxSpeed    float64 `json:"maxSpeed"`    // units/s, 0 = no limit
	IdleFill    bool    `json:"idleFill"`    // fill long pauses
	HomeOnPause bool    `json:"homeOnPause"` // move slowly to the bottom on pause
}

func (a *App) strokeFilterState() (funscript.StrokeFilter, bool) {
	a.stateMu.Lock()
	defer a.stateMu.Unlock()
	if !a.strokeFilterLoaded && a.settings != nil {
		s := a.settings
		a.strokeFilter = funscript.StrokeFilter{
			Smooth:   funscript.NormalizeSmooth(s.GetString(prefStrokeSmooth, "")),
			MaxSpeed: clampMaxSpeed(s.GetFloat(prefStrokeMaxSpeed, 0)),
		}
		if s.GetBool(prefStrokeIdleFill, false) {
			a.strokeFilter.IdleFillMs = strokeIdleFillMs
		}
		a.strokeHome = s.GetBool(prefStrokeHome, false)
		a.strokeFilterLoaded = true
	}
	return a.strokeFilter, a.strokeHome
}

// clampMaxSpeed: 0 (off) or 50–1000 units per second.
func clampMaxSpeed(v float64) float64 {
	if !(v > 0) {
		return 0
	}
	if v < 50 {
		return 50
	}
	if v > 1000 {
		return 1000
	}
	return v
}

// GetStrokeFilter returns the stroke filter settings.
func (a *App) GetStrokeFilter() StrokeFilterSettings {
	f, home := a.strokeFilterState()
	return StrokeFilterSettings{Smooth: f.Smooth, MaxSpeed: f.MaxSpeed, IdleFill: f.IdleFillMs > 0, HomeOnPause: home}
}

// SetStrokeFilter stores the filter and applies it at once to a running
// playback. It returns the stored (normalised) settings.
func (a *App) SetStrokeFilter(in StrokeFilterSettings) (StrokeFilterSettings, error) {
	a.strokeFilterState() // loaded before it is overwritten
	f := funscript.StrokeFilter{Smooth: funscript.NormalizeSmooth(in.Smooth), MaxSpeed: clampMaxSpeed(in.MaxSpeed)}
	if in.IdleFill {
		f.IdleFillMs = strokeIdleFillMs
	}
	if a.settings != nil {
		if err := a.settings.SetMany(map[string]any{
			prefStrokeSmooth: f.Smooth, prefStrokeMaxSpeed: f.MaxSpeed,
			prefStrokeIdleFill: in.IdleFill, prefStrokeHome: in.HomeOnPause,
		}); err != nil {
			return a.GetStrokeFilter(), err
		}
	}
	a.stateMu.Lock()
	a.strokeFilter, a.strokeHome = f, in.HomeOnPause
	p := a.activePlayer
	a.stateMu.Unlock()
	if p != nil {
		if script := a.loadedScript(); script != nil {
			p.SetStroke(funscript.ApplyStrokeFilter(script.PlaybackActions(), f))
		}
		p.SetStrokeHome(in.HomeOnPause)
	}
	return a.GetStrokeFilter(), nil
}

// filteredStroke is the stroke list a new playback gets.
func (a *App) filteredStroke(script *funscript.Script) ([]funscript.Action, bool) {
	f, home := a.strokeFilterState()
	return funscript.ApplyStrokeFilter(script.PlaybackActions(), f), home
}
