package main

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/funfunpayer/SamNPlayer/device"
	"github.com/funfunpayer/SamNPlayer/logging"
	"github.com/funfunpayer/SamNPlayer/virtualperson"
)

// AppHost is the narrow virtualperson.Host implemented by the Wails App.
// Clock comes from the last playback frame / video position; device is the
// session device (may be nil when plugin runs animate-only).
type AppHost struct {
	app *App
}

func (h *AppHost) Device() virtualperson.DeviceBridge {
	h.app.stateMu.RLock()
	dev := h.app.activeDevice
	h.app.stateMu.RUnlock()
	return virtualperson.DeviceBridgeFrom(dev)
}

func (h *AppHost) NowMs() int64 {
	return h.app.vpClockMs.Load()
}

func (h *AppHost) EmitAnimation(pose virtualperson.PoseSample) {
	if h.app.ctx == nil {
		return
	}
	runtime.EventsEmit(h.app.ctx, "virtualperson:pose", poseToMap(pose))
}

func poseToMap(p virtualperson.PoseSample) map[string]any {
	props := make([]map[string]any, 0, len(p.Props))
	for _, pr := range p.Props {
		props = append(props, map[string]any{
			"propId":      string(pr.PropID),
			"socket":      string(pr.Socket),
			"phaseOffset": pr.PhaseOffset,
			"visible":     pr.Visible,
		})
	}
	ch := map[string]any{
		"atMs":       p.Channels.AtMs,
		"stroke":     p.Channels.Stroke,
		"vibration":  p.Channels.Vibration,
		"suction":    p.Channels.Suction,
		"intensity":  p.Channels.Intensity,
	}
	return map[string]any{
		"characterId": string(p.CharacterID),
		"atMs":        p.AtMs,
		"skin":        string(p.Skin),
		"channels":    ch,
		"props":       props,
	}
}

// Virtual Person plugin state lives on App (see ensureVP).
func (a *App) ensureVP() {
	a.vpOnce.Do(func() {
		host := &AppHost{app: a}
		a.vpPlugin = virtualperson.NewPlugin(host)
		a.vpHost = host
	})
}

// EnableVirtualPerson starts the in-process Virtual Person plugin.
// Safe when disabled later: Tick is a no-op if Running() is false.
func (a *App) EnableVirtualPerson() error {
	a.ensureVP()
	a.vpMu.Lock()
	defer a.vpMu.Unlock()
	if a.vpPlugin.Running() {
		return nil
	}
	if err := a.vpPlugin.Start(context.Background()); err != nil {
		return err
	}
	logging.Info("app: Virtual Person plugin started")
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "virtualperson:status", map[string]any{
			"enabled": true, "running": true,
		})
	}
	return nil
}

// DisableVirtualPerson stops the plugin and clears activity ownership.
func (a *App) DisableVirtualPerson() {
	a.ensureVP()
	a.vpMu.Lock()
	defer a.vpMu.Unlock()
	a.vpPlugin.Stop()
	logging.Info("app: Virtual Person plugin stopped")
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "virtualperson:status", map[string]any{
			"enabled": false, "running": false,
		})
	}
}

// VirtualPersonRunning reports whether the plugin tick loop is active.
func (a *App) VirtualPersonRunning() bool {
	a.ensureVP()
	return a.vpPlugin.Running()
}

// VirtualPersonGiveDildo is the MVP scene step: give_toy(dildo).
func (a *App) VirtualPersonGiveDildo() error {
	a.ensureVP()
	return a.vpPlugin.GiveDildo()
}

// VirtualPersonStartTitjob starts activity titjob_dildo (requires dildo).
func (a *App) VirtualPersonStartTitjob(intensity float64, durationS int) error {
	a.ensureVP()
	return a.vpPlugin.StartTitjob(intensity, durationS)
}

// VirtualPersonSetToySync enables/disables optional Neo 2 output from ToyHub.
// Default is off — virtual props remain the primary metaphor.
func (a *App) VirtualPersonSetToySync(on bool) {
	a.ensureVP()
	a.vpPlugin.Toys().SetSync(on)
	logging.Info("app: Virtual Person toy sync", "enabled", on)
}

// VirtualPersonToySync reports whether real-device sync is on.
func (a *App) VirtualPersonToySync() bool {
	a.ensureVP()
	return a.vpPlugin.Toys().SyncEnabled()
}

// tickVirtualPerson is called from the player OnFrame path. No-op when the
// plugin is disabled so playback is unaffected.
func (a *App) tickVirtualPerson(atMs int64, vib, suck float64) {
	a.ensureVP()
	if !a.vpPlugin.Running() {
		return
	}
	a.vpClockMs.Store(atMs)
	// scriptPos: approximate 0–1 from suction/vib peak for control loop stubs.
	scriptPos := vib
	if suck > scriptPos {
		scriptPos = suck
	}
	if err := a.vpPlugin.Tick(scriptPos, vib, suck); err != nil {
		logging.Debug("app: virtualperson Tick", "error", err)
	}
}

// --- fields embedded via methods that App gains (declared here for clarity) ---
// Actual fields must live on App; we add them in a companion patch to app.go.

var (
	_ = sync.Mutex{}
	_ = atomic.Int64{}
	_ = time.Time{}
	_ = device.Device(nil)
)
