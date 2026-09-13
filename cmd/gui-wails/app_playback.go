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

// PlaybackOptions kommt als JSON vom Frontend-Formular.
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
	a.activeDevice = dev

	p := player.New(dev)
	p.LogEvery = time.Second
	p.SoftStartMs = opts.SoftStartMs
	p.OnLog = func(line string) {
		runtime.EventsEmit(a.ctx, "playback:log", line)
	}
	p.OnFrame = func(f funscript.Frame) {
		runtime.EventsEmit(a.ctx, "playback:frame", map[string]any{
			"atMs":      f.At,
			"vibration": f.Vibration,
			"suction":   f.Suction,
			"totalMs":   frames[len(frames)-1].At,
		})
	}
	if opts.UseVideoSync {
		p.PauseVideo = func() { runtime.EventsEmit(a.ctx, "video:pause") }
		p.ResumeVideo = func() { runtime.EventsEmit(a.ctx, "video:resume") }
	}
	a.activePlayer = p

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
			a.videoPositionCh = positions
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
