// Package pluginhost is the narrow SamNPlayer host surface for drop-folder
// plugin packs (docs/PLUGIN_SYSTEM.md).
//
// Stage H1 (infra): license-aware Slot + drop-folder discovery
// (samn-plugin.json) + optional OnFrame Tick wiring. Product plugins
// (Virtual Person / #264) are parked — this package keeps the generic
// handshake only. No marketplace, no dynamic Go .so, no Everyday CSRT
// changes.
package pluginhost

import (
	"fmt"
	"sync"
	"sync/atomic"
)

// PluginIDVirtualPerson is the reserved pack id for the parked Virtual Person
// product. Drop-folder discovery accepts any valid manifest id; this constant
// stays for pack compatibility and tests.
const PluginIDVirtualPerson = "virtual_person"

// Stage is the host maturity marker (H1 = drop-folder + optional OnFrame tick).
const Stage = "H1"

// Host is the narrow surface a future in-process plugin may use from SamNPlayer.
// Names match docs/PLUGIN_SYSTEM.md.
type Host interface {
	NowMs() int64
	EmitAnimation(pose PoseSample)
}

// PoseSample is a minimal animation snapshot for a future frontend event bridge.
type PoseSample struct {
	AtMs   int64   `json:"atMs"`
	Stroke float64 `json:"stroke"`
	Vibe   float64 `json:"vibe"`
	Suck   float64 `json:"suck"`
}

// Status reports a plugin-host slot for Settings / API.
type Status struct {
	FeatureID   string `json:"featureId,omitempty"`
	Allowed     bool   `json:"allowed"`
	Enabled     bool   `json:"enabled"`
	Running     bool   `json:"running"`
	Stage       string `json:"stage"`
	Ticks       uint64 `json:"ticks"`
	Message     string `json:"message"`
	PackID      string `json:"packId,omitempty"`
	PackName    string `json:"packName,omitempty"`
	PackVersion string `json:"packVersion,omitempty"`
	PackRoot    string `json:"packRoot,omitempty"`
	PluginsDir  string `json:"pluginsDir,omitempty"`
	PackFound   bool   `json:"packFound"`
}

// Slot is a single in-process plugin-host slot (infra).
// Tick is a no-op when not running so playback stays unaffected.
// Rel35 ships discovery/install only — product Enable UI is parked.
type Slot struct {
	mu         sync.Mutex
	enabled    bool
	running    bool
	clockMs    atomic.Int64
	ticks      atomic.Uint64
	lastEmit   PoseSample
	pack       *Pack
	pluginsDir string
	featureID  string
}

// NewSlot returns an idle plugin-host slot.
func NewSlot() *Slot {
	return &Slot{}
}

// SetPluginsDir records where drop-in packs live (for Status / UI).
func (s *Slot) SetPluginsDir(dir string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pluginsDir = dir
}

// SetFeatureID overrides the license feature id shown in Status (optional).
func (s *Slot) SetFeatureID(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.featureID = id
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
	featureID := s.featureID
	if featureID == "" && s.pack != nil && s.pack.Manifest.RequiresFeature != "" {
		featureID = s.pack.Manifest.RequiresFeature
	}
	st := Status{
		FeatureID:  featureID,
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
		st.Message = "License required for this plugin pack. Everyday Create stays free."
	case s.running && s.pack != nil:
		st.Message = fmt.Sprintf("Plugin host running — pack %s v%s.", s.pack.Manifest.Name, s.pack.Manifest.Version)
	case s.running:
		st.Message = "Plugin host running (H1 — OnFrame tick; no pack bound)."
	case s.enabled:
		st.Message = "Plugin host enabled but not running."
	case !st.PackFound:
		st.Message = "No plugin pack found. Drop a pack folder into Plugins, or use Install pack…"
	default:
		st.Message = fmt.Sprintf("Pack ready: %s v%s.", s.pack.Manifest.Name, s.pack.Manifest.Version)
	}
	return st
}

// Enable arms the host slot. Fails closed when allowed is false.
// Idempotent when already running. Product Enable UI is parked for Rel35.
func (s *Slot) Enable(allowed bool) error {
	if !allowed {
		id := s.featureID
		if id == "" {
			id = "plugin"
		}
		return fmt.Errorf("pluginhost: license feature %q required", id)
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

// EmitAnimation records the last pose (Host surface).
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
