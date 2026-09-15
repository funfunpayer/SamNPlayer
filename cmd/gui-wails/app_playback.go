package main

import (
	"context"
	"fmt"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/funfunpayer/SamNPlayer/device"
	"github.com/funfunpayer/SamNPlayer/funscript"
	"github.com/funfunpayer/SamNPlayer/logging"
	"github.com/funfunpayer/SamNPlayer/player"
)

type PlaybackOptions struct {
	Mock               bool    `json:"mock"`
	SyncMode           string  `json:"syncMode"`
	TickMs             int64   `json:"tickMs"`
	MaxSpeed           float64 `json:"maxSpeed"`
	Smoothing          float64 `json:"smoothing"`
	SoftStartMs        int     `json:"softStartMs"`
	UseVideoSync       bool    `json:"useVideoSync"`
	ExtendedOEnabled   bool    `json:"extendedOEnabled"`
	ExtendedOMin       float64 `json:"extendedOMin"`
	ExtendedOHoldS     float64 `json:"extendedOHoldS"`
	ExtendedORestoreMs float64 `json:"extendedORestoreMs"`
}

func (a *App) StartPlayback(opts PlaybackOptions) error {
	if a.currentScript == nil {
		return fmt.Errorf("kein Skript geladen")
	}
	mapOpts := funscript.DefaultMapOptions()
	profile := a.currentScript.Metadata.Profile
	if funscript.IsDistanceProfile(profile) {
		mapOpts = funscript.RecipeFor(profile)
		if dr := a.currentScript.Metadata.DeviceRecipe; dr != nil {
			mapOpts.ContactVibration = dr.ContactVibration
		}
	}
	if opts.TickMs > 0 {
		mapOpts.TickMs = opts.TickMs
	}
	if opts.MaxSpeed > 0 {
		mapOpts.MaxSpeed = opts.MaxSpeed
	}
	mapOpts.Smoothing = opts.Smoothing
	syncMode, err := funscript.ParseSyncMode(opts.SyncMode)
	if err != nil {
		return err
	}
	if !funscript.IsDistanceProfile(profile) || (opts.SyncMode != "" && opts.SyncMode != "independent") {
		mapOpts.Sync = syncMode
	}
	frames := a.currentScript.ToIntensityCurve(mapOpts)
	if len(frames) == 0 {
		return fmt.Errorf("das Skript enthält keine abspielbaren Actions")
	}
	a.currentFrames = frames
	var dev device.Device
	if opts.Mock {
		dev = device.NewMock(false)
	} else {
		dev = device.NewSamNeo2(device.SamNeo2Protocol{})
	}
	p := player.New(dev)
	p.LogEvery = time.Second
	p.SoftStartMs = opts.SoftStartMs
	p.OnLog = func(line string) { runtime.EventsEmit(a.ctx, "playback:log", line) }
	p.OnFrame = func(f funscript.Frame) {
		runtime.EventsEmit(a.ctx, "playback:frame", map[string]any{
			"atMs": f.At, "vibration": f.Vibration, "suction": f.Suction, "totalMs": frames[len(frames)-1].At,
		})
	}
	if opts.UseVideoSync {
		p.PauseVideo = func() { runtime.EventsEmit(a.ctx, "video:pause") }
		p.ResumeVideo = func() { runtime.EventsEmit(a.ctx, "video:resume") }
	}
	// Unter stateMu wie jedes andere geteilte Feld auf App (siehe dessen
	// eigene Deklaration) - vorher unguarded gesetzt, während
	// TriggerExtendedO a.activePlayer ebenso unguarded liest: eine echte
	// Datenwettlauf zwischen einem Wiedergabe-Start und einem Klick auf
	// "Extended-O auslösen" kurz danach.
	a.stateMu.Lock()
	a.activeDevice = dev
	a.activePlayer = p
	a.stateMu.Unlock()
	ctx, err := a.tryStartSession()
	if err != nil {
		return err
	}
	go func() {
		defer a.endSession()
		connectCtx, connectCancel := context.WithTimeout(ctx, 20*time.Second)
		defer connectCancel()
		runtime.EventsEmit(a.ctx, "playback:log", "Verbinde...")
		if err := dev.Connect(connectCtx); err != nil {
			logging.Error("app: Verbindung fehlgeschlagen", "fehler", err)
			runtime.EventsEmit(a.ctx, "playback:error", err.Error())
			runtime.EventsEmit(a.ctx, "playback:done")
			return
		}
		defer dev.Disconnect()
		runtime.EventsEmit(a.ctx, "playback:log", "Wiedergabe startet...")
		var playErr error
		if opts.UseVideoSync {
			positions := make(chan int64, 4)
			a.stateMu.Lock()
			a.videoPositionCh = positions
			a.stateMu.Unlock()
			playErr = p.Sync(ctx, frames, positions)
		} else {
			playErr = p.Play(ctx, frames)
		}
		if playErr != nil && playErr != context.Canceled {
			runtime.EventsEmit(a.ctx, "playback:error", playErr.Error())
		}
		runtime.EventsEmit(a.ctx, "playback:done")
	}()
	return nil
}

func (a *App) StopPlayback() {
	a.stopSession()
	// Lock() statt RLock(), und zwar über den ganzen close() hinweg: eine
	// laufende ReportVideoPosition() hält für ihren send bereits RLock() -
	// Lock() blockiert also so lange, bis dieser send fertig ist, und kann
	// den Kanal nie schließen, während gleichzeitig darauf gesendet wird
	// ("panic: send on closed channel", reproduziert mit go test -race und
	// ohne -race unter Last).
	a.stateMu.Lock()
	ch := a.videoPositionCh
	a.videoPositionCh = nil
	a.stateMu.Unlock()
	if ch != nil {
		close(ch)
	}
}

func (a *App) ReportVideoPosition(ms int64) {
	// RLock() muss den send mit abdecken, nicht nur das Lesen des Kanals -
	// sonst kann StopPlayback() den Kanal zwischen Prüfung und send
	// schließen (siehe Kommentar dort).
	a.stateMu.RLock()
	defer a.stateMu.RUnlock()
	if a.videoPositionCh == nil {
		return
	}
	ms -= a.scriptOffsetMs
	if ms < 0 {
		ms = 0
	}
	select {
	case a.videoPositionCh <- ms:
	default:
	}
}

func (a *App) TriggerExtendedO(minLevel, holdSeconds, restoreMs float64) {
	a.stateMu.RLock()
	p := a.activePlayer
	a.stateMu.RUnlock()
	if p == nil {
		return
	}
	p.TriggerExtendedO(player.ExtendedOOptions{
		MinLevel:        minLevel,
		HoldDuration:    time.Duration(holdSeconds * float64(time.Second)),
		RestoreDuration: time.Duration(restoreMs * float64(time.Millisecond)),
	})
}

type HeatmapPoint struct {
	AtMs      int64   `json:"atMs"`
	Intensity float64 `json:"intensity"`
}

type CurvePoint struct {
	AtMs int64 `json:"atMs"`
	Pos  int   `json:"pos"`
}

func (a *App) GetScriptCurve(maxPoints int) ([]CurvePoint, error) {
	if a.currentScript == nil {
		return nil, fmt.Errorf("kein Skript geladen")
	}
	actions := a.currentScript.Actions
	if len(actions) == 0 {
		return nil, fmt.Errorf("skript enthält keine Actions")
	}
	if maxPoints < 100 {
		maxPoints = 100
	}
	if len(actions) <= maxPoints {
		out := make([]CurvePoint, len(actions))
		for i, act := range actions {
			out[i] = CurvePoint{AtMs: act.At, Pos: act.Pos}
		}
		return out, nil
	}
	windows := maxPoints / 2
	span := actions[len(actions)-1].At - actions[0].At
	if span <= 0 {
		span = 1
	}
	out := make([]CurvePoint, 0, maxPoints)
	start := 0
	for w := 0; w < windows && start < len(actions); w++ {
		endAt := actions[0].At + span*int64(w+1)/int64(windows)
		minIdx, maxIdx := start, start
		i := start
		for ; i < len(actions) && (actions[i].At <= endAt || i == start); i++ {
			if actions[i].Pos < actions[minIdx].Pos {
				minIdx = i
			}
			if actions[i].Pos > actions[maxIdx].Pos {
				maxIdx = i
			}
		}
		lo, hi := minIdx, maxIdx
		if lo > hi {
			lo, hi = hi, lo
		}
		out = append(out, CurvePoint{AtMs: actions[lo].At, Pos: actions[lo].Pos})
		if hi != lo {
			out = append(out, CurvePoint{AtMs: actions[hi].At, Pos: actions[hi].Pos})
		}
		start = i
	}
	return out, nil
}

func (a *App) GetHeatmap(buckets int) ([]HeatmapPoint, error) {
	if a.currentScript == nil {
		return nil, fmt.Errorf("kein Skript geladen")
	}
	if buckets < 10 {
		buckets = 10
	}
	duration := a.currentScript.Duration()
	if duration <= 0 {
		return nil, fmt.Errorf("skript hat keine gültige Dauer")
	}
	opts := funscript.DefaultMapOptions()
	opts.TickMs = duration / int64(buckets)
	if opts.TickMs < 10 {
		opts.TickMs = 10
	}
	frames := a.currentScript.ToIntensityCurve(opts)
	points := make([]HeatmapPoint, len(frames))
	for i, f := range frames {
		intensity := f.Vibration
		if f.Suction > intensity {
			intensity = f.Suction
		}
		points[i] = HeatmapPoint{AtMs: f.At, Intensity: intensity}
	}
	return points, nil
}

func (a *App) SetScriptOffset(ms int64) {
	if ms < -10000 {
		ms = -10000
	}
	if ms > 10000 {
		ms = 10000
	}
	a.stateMu.Lock()
	a.scriptOffsetMs = ms
	path := a.currentScriptPath
	a.stateMu.Unlock()
	if path != "" {
		_ = a.settings.Set(offsetKeyFor(path), ms)
	}
	logging.Info("wiedergabe: Skript-Offset gesetzt", "ms", ms, "skript", path)
}

func (a *App) GetScriptOffset() int64 {
	a.stateMu.RLock()
	defer a.stateMu.RUnlock()
	return a.scriptOffsetMs
}

func offsetKeyFor(scriptPath string) string {
	return "playback.offset." + scriptPath
}
