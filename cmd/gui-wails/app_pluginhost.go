package main

import (
	"context"
	"fmt"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/funfunpayer/SamNPlayer/license"
	"github.com/funfunpayer/SamNPlayer/logging"
	"github.com/funfunpayer/SamNPlayer/pluginhost"
	"github.com/funfunpayer/SamNPlayer/virtualperson"
)

var errVPHostNotRunning = fmt.Errorf("virtual person host is not running — enable it in Settings first")

// vpAppHost adapts App to virtualperson.Host. Clock comes from the last
// playback OnFrame; device bridge is available but ToyHub sync stays off
// (player remains the sole device writer — ownership flip is a follow-up).
type vpAppHost struct {
	app *App
}

func (h *vpAppHost) Device() virtualperson.DeviceBridge {
	h.app.stateMu.RLock()
	dev := h.app.activeDevice
	h.app.stateMu.RUnlock()
	return virtualperson.DeviceBridgeFrom(dev)
}

func (h *vpAppHost) NowMs() int64 {
	return h.app.vpClockMs.Load()
}

func (h *vpAppHost) EmitAnimation(pose virtualperson.PoseSample) {
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
		"surge":      p.Channels.Surge,
		"sway":       p.Channels.Sway,
		"twist":      p.Channels.Twist,
		"vibe":       p.Channels.Vibe,
		"suck":       p.Channels.Suck,
		"expression": p.Channels.Expression,
	}
	return map[string]any{
		"characterId": string(p.CharacterID),
		"atMs":        p.AtMs,
		"skin":        string(p.Skin),
		"channels":    ch,
		"props":       props,
	}
}

func (a *App) ensureVPHost() *pluginhost.Slot {
	a.vpOnce.Do(func() {
		host := &vpAppHost{app: a}
		a.vpHostImpl = host
		a.vpHost = pluginhost.NewSlot()
		a.vpPlugin = virtualperson.NewPlugin(host)
		// ToyHub sync stays off — virtual props are the primary metaphor.
		if a.vpPlugin.Toys() != nil {
			a.vpPlugin.Toys().SetSync(false)
		}
	})
	return a.vpHost
}

func (a *App) ensureVPPlugin() *virtualperson.Plugin {
	a.ensureVPHost()
	return a.vpPlugin
}

// LicenseAllowsVirtualPerson reports whether the Virtual Person host may be
// enabled. While license.Enforcement is false, always true (dev / Everyday).
// When sharp: requires a valid key with feature virtual_person (standard €40).
func (a *App) LicenseAllowsVirtualPerson() bool {
	return license.EffectiveHasFeature(a.GetLicenseStatus(), license.FeatureVirtualPerson)
}

// VirtualPersonHostStatus is the Settings / API view of the host slot.
func (a *App) VirtualPersonHostStatus() pluginhost.Status {
	return a.ensureVPHost().Status(a.LicenseAllowsVirtualPerson())
}

// EnableVirtualPersonHost starts the host slot + in-process virtualperson
// plugin. Fails when the license feature is required and missing.
// Does not change Everyday CSRT or seize the device (ToyHub sync off).
func (a *App) EnableVirtualPersonHost() (pluginhost.Status, error) {
	allowed := a.LicenseAllowsVirtualPerson()
	slot := a.ensureVPHost()
	plugin := a.ensureVPPlugin()
	a.vpMu.Lock()
	defer a.vpMu.Unlock()
	if err := slot.Enable(allowed); err != nil {
		return slot.Status(allowed), err
	}
	if err := plugin.Start(context.Background()); err != nil {
		slot.Disable()
		return slot.Status(allowed), err
	}
	logging.Info("app: Virtual Person host enabled", "stage", pluginhost.Stage)
	return slot.Status(allowed), nil
}

// DisableVirtualPersonHost stops the plugin and host slot.
func (a *App) DisableVirtualPersonHost() pluginhost.Status {
	a.vpMu.Lock()
	plugin := a.vpPlugin
	slot := a.vpHost
	a.vpMu.Unlock()
	if plugin != nil {
		plugin.Stop()
	}
	if slot != nil {
		slot.Disable()
	} else {
		a.ensureVPHost().Disable()
	}
	logging.Info("app: Virtual Person host disabled")
	return a.ensureVPHost().Status(a.LicenseAllowsVirtualPerson())
}

// VirtualPersonGiveDildo is the MVP scene step: give_toy(dildo).
// Requires the host to be enabled (GUI load rule — Settings button).
func (a *App) VirtualPersonGiveDildo() error {
	plugin := a.ensureVPPlugin()
	if !plugin.Running() {
		return errVPHostNotRunning
	}
	return plugin.GiveDildo()
}

// VirtualPersonStartTitjob starts activity titjob_dildo (requires dildo).
func (a *App) VirtualPersonStartTitjob(intensity float64, durationS int) error {
	plugin := a.ensureVPPlugin()
	if !plugin.Running() {
		return errVPHostNotRunning
	}
	return plugin.StartTitjob(intensity, durationS)
}

// tickVirtualPersonHost is called from the player OnFrame path.
// No-op unless the host was explicitly enabled — playback-safe.
func (a *App) tickVirtualPersonHost(atMs int64, vibe, suck float64) {
	if a.vpHost == nil || !a.vpHost.Running() {
		return
	}
	a.vpClockMs.Store(atMs)
	scriptPos := vibe * 100
	if suck > vibe {
		scriptPos = suck * 100
	}
	a.vpHost.Tick(atMs, scriptPos, vibe, suck)
	if a.vpPlugin != nil && a.vpPlugin.Running() {
		if err := a.vpPlugin.Tick(scriptPos, vibe, suck); err != nil {
			logging.Debug("app: virtualperson Tick", "error", err)
		}
	}
}
