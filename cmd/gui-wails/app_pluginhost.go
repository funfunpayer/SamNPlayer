package main

import (
	"github.com/funfunpayer/SamNPlayer/license"
	"github.com/funfunpayer/SamNPlayer/logging"
	"github.com/funfunpayer/SamNPlayer/pluginhost"
)

func (a *App) ensureVPHost() *pluginhost.Slot {
	if a.vpHost == nil {
		a.vpHost = pluginhost.NewSlot()
	}
	return a.vpHost
}

// LicenseAllowsVirtualPerson reports whether the Virtual Person host may be
// enabled. While license.Enforcement is false, always true (dev / Everyday).
// When sharp: requires a valid key with feature virtual_person (standard €40).
func (a *App) LicenseAllowsVirtualPerson() bool {
	return license.EffectiveHasFeature(a.GetLicenseStatus(), license.FeatureVirtualPerson)
}

// VirtualPersonHostStatus is the Settings / API view of the H0 host slot.
func (a *App) VirtualPersonHostStatus() pluginhost.Status {
	return a.ensureVPHost().Status(a.LicenseAllowsVirtualPerson())
}

// EnableVirtualPersonHost starts the H0 stub host. Fails when the license
// feature is required and missing. Does not change Everyday CSRT.
func (a *App) EnableVirtualPersonHost() (pluginhost.Status, error) {
	allowed := a.LicenseAllowsVirtualPerson()
	slot := a.ensureVPHost()
	if err := slot.Enable(allowed); err != nil {
		return slot.Status(allowed), err
	}
	logging.Info("app: Virtual Person host enabled", "stage", pluginhost.Stage)
	return slot.Status(allowed), nil
}

// DisableVirtualPersonHost stops the H0 stub host.
func (a *App) DisableVirtualPersonHost() pluginhost.Status {
	slot := a.ensureVPHost()
	slot.Disable()
	logging.Info("app: Virtual Person host disabled")
	return slot.Status(a.LicenseAllowsVirtualPerson())
}

// tickVirtualPersonHost is called from the player OnFrame path when wired.
// H0: safe no-op unless the host was explicitly enabled.
func (a *App) tickVirtualPersonHost(atMs int64, vibe, suck float64) {
	if a.vpHost == nil || !a.vpHost.Running() {
		return
	}
	scriptPos := vibe * 100
	if suck > vibe {
		scriptPos = suck * 100
	}
	a.vpHost.Tick(atMs, scriptPos, vibe, suck)
}
