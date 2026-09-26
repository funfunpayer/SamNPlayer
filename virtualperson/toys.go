package virtualperson

import (
	"sync"
)

// ToyTarget is a real-world device sink (parallel to virtual props).
type ToyTarget string

const (
	ToySamNeo2   ToyTarget = "sam_neo_2"
	ToyIntiface  ToyTarget = "intiface" // device-agnostic buttplug path
	ToyMock      ToyTarget = "mock"
)

// ToyRoute maps animation channels to a real device bridge.
type ToyRoute struct {
	Target   ToyTarget
	Bridge   DeviceBridge
	Enabled  bool
	// MapStrokeToSuck: Neo 2 often maps depth-like stroke → suction.
	MapStrokeToSuck bool
}

// ToyHub fans MotionBus channels out to real toys when sync is enabled.
// Virtual props/activities remain authoritative for the scene; this is optional.
type ToyHub struct {
	mu     sync.Mutex
	routes []ToyRoute
	syncOn bool
}

// NewToyHub starts with sync off.
func NewToyHub() *ToyHub {
	return &ToyHub{}
}

// SetSync enables/disables all real-device output.
func (h *ToyHub) SetSync(on bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.syncOn = on
	if !on {
		for _, r := range h.routes {
			if r.Bridge != nil {
				_ = r.Bridge.Stop()
			}
		}
	}
}

// SyncEnabled reports whether real toys follow the bus.
func (h *ToyHub) SyncEnabled() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.syncOn
}

// AddRoute registers a device (e.g. primary Sam Neo 2).
func (h *ToyHub) AddRoute(r ToyRoute) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.routes = append(h.routes, r)
}

// Apply pushes channels to enabled routes. No-op if sync off.
func (h *ToyHub) Apply(ch ChannelValues) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if !h.syncOn {
		return nil
	}
	for _, r := range h.routes {
		if !r.Enabled || r.Bridge == nil {
			continue
		}
		vibe := ch.Vibe
		suck := ch.Suck
		if r.MapStrokeToSuck {
			suck = clamp01(ch.Stroke / 100)
		}
		if err := r.Bridge.SetVibration(vibe); err != nil {
			return err
		}
		if err := r.Bridge.SetSuction(suck); err != nil {
			return err
		}
	}
	return nil
}

// EmergencyStop cuts all real devices.
func (h *ToyHub) EmergencyStop() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.syncOn = false
	for _, r := range h.routes {
		if r.Bridge != nil {
			_ = r.Bridge.Stop()
		}
	}
}
