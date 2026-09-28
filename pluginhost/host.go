// Package pluginhost is the narrow SamNPlayer host surface for Virtual Person
// plugins (docs/PLUGIN_SYSTEM.md).
//
// Stage H1: license gate + drop-folder discovery (samn-plugin.json) + playback
// OnFrame Tick wiring. Package virtualperson holds props/activities/bus
// (cherry-picked from #264). End users install by copying a pack into the
// Plugins folder and clicking Enable. No marketplace, no dynamic Go .so, no
// Everyday CSRT changes. The first plugin id is virtual_person; it requires
// license.FeatureVirtualPerson when license.Enforcement is on (included in
// the standard €40 key).
package pluginhost

import (
	"fmt"
	"sync"
	"sync/atomic"
)

// PluginIDVirtualPerson is the in-process Virtual Person slot.
const PluginIDVirtualPerson = "virtual_person"

// Stage is the host maturity marker (H1 = drop-folder + OnFrame tick + VP core).
const Stage = "H1"

// Host is the narrow surface a Virtual Person plugin may use from SamNPlayer.
// Names match docs/PLUGIN_SYSTEM.md — implement on the Wails App later.
type Host interface {
	NowMs() int64
	EmitAnimation(pose PoseSample)
}

// PoseSample is a minimal animation snapshot for the frontend event bridge.
// Expanded fields land with a later virtualperson package; H0 keeps the shape.
type PoseSample struct {
	AtMs   int64   `json:"atMs"`
	Stroke float64 `json:"stroke"`
	Vibe   float64 `json:"vibe"`
	Suck   float64 `json:"suck"`
}

// Status reports the Virtual Person host slot for Settings / API.
type Status struct {
	FeatureID   string `json:"featureId"` // license.FeatureVirtualPerson
	Allowed     bool   `json:"allowed"`   // EffectiveHasFeature result
	Enabled     bool   `json:"enabled"`
	Running     bool   `json:"running"`
	Stage       string `json:"stage"`
	Ticks       uint64 `json:"ticks"` // OnFrame Tick count while running
	Message     string `json:"message"`
	PackID      string `json:"packId,omitempty"`
	PackName    string `json:"packName,omitempty"`
	PackVersion string `json:"packVersion,omitempty"`
	PackRoot    string `json:"packRoot,omitempty"`
	PluginsDir  string `json:"pluginsDir,omitempty"`
	PackFound   bool   `json:"packFound"`
}

// Slot is the single in-process Virtual Person host slot (H1).
// Tick is a no-op when not running so playback stays unaffected.
type Slot struct {
	mu         sync.Mutex
	enabled    bool
	running    bool
	clockMs    atomic.Int64
	ticks      atomic.Uint64
	lastEmit   PoseSample
	pack       *Pack
	pluginsDir string
}

// NewSlot returns an idle Virtual Person host slot.
func NewSlot() *Slot {
	return &Slot{}
}

// SetPluginsDir records where drop-in packs live (for Status / UI).
func (s *Slot) SetPluginsDir(dir string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pluginsDir = dir
}

// BindPack attaches a discovered pack (may be nil to clear).
func (s *Slot) BindPack(p *Pack) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pack = p
}

// BoundPack returns a copy of the bound pack, if any.
func (s *Slot) BoundPack() (Pack, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.pack == nil {
		return Pack{}, false
	}
	return *s.pack, true
}

// Status builds the UI/API view. allowed comes from the license gate.
func (s *Slot) Status(allowed bool) Status {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := Status{
		FeatureID:  PluginIDVirtualPerson,
		Allowed:    allowed,
		Enabled:    s.enabled,
		Running:    s.running,
		Stage:      Stage,
		Ticks:      s.ticks.Load(),
		PluginsDir: s.pluginsDir,
	}
	if s.pack != nil {
		st.PackFound = true
		st.PackID = s.pack.Manifest.ID
		st.PackName = s.pack.Manifest.Name
		st.PackVersion = s.pack.Manifest.Version
		st.PackRoot = s.pack.Root
	}
	switch {
	case !allowed:
		st.Message = "License required for Virtual Person (feature virtual_person). Everyday Create stays free."
	case s.running && s.pack != nil:
		st.Message = fmt.Sprintf("Virtual Person running — pack %s v%s.", s.pack.Manifest.Name, s.pack.Manifest.Version)
	case s.running:
		st.Message = "Virtual Person host running (H1 — OnFrame tick + scene bus; no pack bound)."
	case s.enabled:
		st.Message = "Virtual Person host enabled but not running."
	case !st.PackFound:
		st.Message = "No Virtual Person pack found. Drop the pack folder into Plugins, or use Install pack…"
	default:
		st.Message = fmt.Sprintf("Pack ready: %s v%s. Click Enable.", s.pack.Manifest.Name, s.pack.Manifest.Version)
	}
	return st
}

// Enable arms the host slot. Fails closed when allowed is false (Enforcement on
// without virtual_person). Idempotent when already running.
func (s *Slot) Enable(allowed bool) error {
	if !allowed {
		return fmt.Errorf("pluginhost: license feature %q required", PluginIDVirtualPerson)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.enabled = true
	s.running = true
	return nil
}

// Disable stops the host slot. Safe when already idle.
func (s *Slot) Disable() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.enabled = false
	s.running = false
}

// Running reports whether the stub tick path is active.
func (s *Slot) Running() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

// NowMs returns the last clock sample from Tick (Host surface).
func (s *Slot) NowMs() int64 {
	return s.clockMs.Load()
}

// EmitAnimation records the last pose (Host surface; no frontend yet in H0).
func (s *Slot) EmitAnimation(pose PoseSample) {
	s.mu.Lock()
	s.lastEmit = pose
	s.mu.Unlock()
}

// LastEmit returns the last pose passed to EmitAnimation (tests / diagnostics).
func (s *Slot) LastEmit() PoseSample {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastEmit
}

// TickCount returns how many Ticks ran while the slot was running.
func (s *Slot) TickCount() uint64 {
	return s.ticks.Load()
}

// Tick advances the stub clock. No-op when not running — playback-safe.
func (s *Slot) Tick(atMs int64, scriptPos, vibe, suck float64) {
	if !s.Running() {
		return
	}
	s.clockMs.Store(atMs)
	s.ticks.Add(1)
	_ = scriptPos
	s.EmitAnimation(PoseSample{AtMs: atMs, Stroke: vibe * 100, Vibe: vibe, Suck: suck})
}
