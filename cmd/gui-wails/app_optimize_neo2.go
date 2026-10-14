package main

import (
	"fmt"
	"strings"

	"github.com/funfunpayer/SamNPlayer/samn"
)

// OptimizeNeo2Result is the one-click import → Neo 2 path.
type OptimizeNeo2Result struct {
	Path        string `json:"path"`
	Message     string `json:"message"`
	PointsAdded int    `json:"pointsAdded"`
	GapsFilled  int    `json:"gapsFilled"`
	ContactOn   bool   `json:"contactOn"`
	Baked       bool   `json:"baked"`
	HasNeoAxes  bool   `json:"hasNeoAxes"`
}

// OptimizeLoadedForNeo2 polishes an imported community .funscript (or .samn)
// for Sam Neo 2: optional fill-gaps → Contact vib on → bake vibe/suction axes
// → save .samn. Multi-axis editor can then tune general/vibration/suction.
//
// Does not invent a stroke from audio — fill is linear only. Safe for
// Everyday + foreign scripts from other tools/platforms.
func (a *App) OptimizeLoadedForNeo2(fillGaps bool) (OptimizeNeo2Result, error) {
	out := OptimizeNeo2Result{}
	path := a.loadedScriptPath()
	if path == "" {
		return out, fmt.Errorf("no script loaded")
	}

	video := a.loadedVideoPath()
	if fillGaps {
		imp, err := a.ImproveGeneratedScript(ImproveScriptRequest{
			Path:             path,
			VideoPath:        video,
			FillGaps:         true,
			HealTrackingGaps: true,
			AudioCheck:       false,
			UseAudioForFill:  false,
		})
		if err != nil {
			return out, fmt.Errorf("fill gaps: %w", err)
		}
		out.PointsAdded = imp.PointsAdded
		out.GapsFilled = imp.GapsFilled
		if imp.Path != "" {
			path = imp.Path
		}
		// Improve may have switched to companion .samn — reload path.
		if a.loadedScriptPath() != "" {
			path = a.loadedScriptPath()
		}
	}

	// Contact vib soft defaults — Neo 2 feel layer on stroke scripts.
	if err := a.SaveContactSettings(true, 0.75, "soft"); err != nil {
		return out, fmt.Errorf("contact recipe: %w", err)
	}
	out.ContactOn = true

	baked, err := a.BakeNeoAxesOnLoaded()
	if err != nil {
		return out, fmt.Errorf("bake Neo2 axes: %w", err)
	}
	out.Baked = true
	out.Path = baked
	out.HasNeoAxes = true

	parts := []string{"Neo 2 ready"}
	if out.GapsFilled > 0 {
		parts = append(parts, fmt.Sprintf("filled %d gap(s) (+%d pts)", out.GapsFilled, out.PointsAdded))
	}
	parts = append(parts, "contact on", "axes baked")
	if samn.IsSamnPath(baked) {
		parts = append(parts, "saved "+baked)
	}
	out.Message = strings.Join(parts, " · ")
	return out, nil
}

func (a *App) loadedVideoPath() string {
	a.stateMu.RLock()
	defer a.stateMu.RUnlock()
	return a.videoPath
}

// Optimize targets (Owner 14 Oct): a script is optimized for any device
// (general) or for one device. "general" only polishes the stroke (fill /
// heal gaps) and keeps the file usable everywhere; "neo2" adds the Sam Neo 2
// feel layer (contact vibration + baked vibration / suction). Device
// profiles for strokers etc. join this list when they exist — no target is
// offered that does not do something real.
const (
	OptimizeTargetGeneral = "general"
	OptimizeTargetNeo2    = "neo2"
)

// OptimizeLoadedFor optimizes the loaded script for target ("general" or
// "neo2").
func (a *App) OptimizeLoadedFor(target string) (OptimizeNeo2Result, error) {
	switch target {
	case OptimizeTargetNeo2:
		return a.OptimizeLoadedForNeo2(true)
	case OptimizeTargetGeneral, "":
		out := OptimizeNeo2Result{}
		path := a.loadedScriptPath()
		if path == "" {
			return out, fmt.Errorf("no script loaded")
		}
		imp, err := a.ImproveGeneratedScript(ImproveScriptRequest{
			Path:             path,
			VideoPath:        a.loadedVideoPath(),
			FillGaps:         true,
			HealTrackingGaps: true,
		})
		if err != nil {
			return out, fmt.Errorf("fill gaps: %w", err)
		}
		out.PointsAdded = imp.PointsAdded
		out.GapsFilled = imp.GapsFilled
		out.Path = imp.Path
		if out.Path == "" {
			out.Path = a.loadedScriptPath()
		}
		parts := []string{"Ready for any device"}
		if out.GapsFilled > 0 {
			parts = append(parts, fmt.Sprintf("filled %d gap(s) (+%d pts)", out.GapsFilled, out.PointsAdded))
		} else {
			parts = append(parts, "no gaps to fill")
		}
		out.Message = strings.Join(parts, " · ")
		return out, nil
	default:
		return OptimizeNeo2Result{}, fmt.Errorf("unknown optimize target %q", target)
	}
}
