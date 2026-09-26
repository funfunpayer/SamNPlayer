package virtualperson

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Host is the narrow surface Virtual Person code may use from SamNPlayer.
type Host interface {
	Device() DeviceBridge
	NowMs() int64
	EmitAnimation(pose PoseSample)
}

// Plugin is the Virtual Person entry point loaded by the host app.
type Plugin struct {
	mu        sync.RWMutex
	host      Host
	registry  *Registry
	inventory *PropInventory
	bus       *MotionBus
	catalog   *ActivityCatalog
	mapper    *ChannelMapper
	driver    *AnimationDriver
	toys      *ToyHub
	chat      *ChatSession
	ai        ControlLoop
	running   bool
	lastTick  time.Time
}

// NewPlugin constructs an idle plugin with character-01 wiring stubs.
func NewPlugin(host Host) *Plugin {
	reg := NewRegistry()
	_ = reg.EnsureCharacter01()
	inv := NewPropInventory(Character01.ID)
	bus := NewMotionBus()
	cat := NewActivityCatalog(inv, bus)
	hub := NewToyHub()
	if host != nil {
		if d := host.Device(); d != nil {
			hub.AddRoute(ToyRoute{
				Target:          ToySamNeo2,
				Bridge:          d,
				Enabled:         true,
				MapStrokeToSuck: true,
			})
		}
	}
	p := &Plugin{
		host:      host,
		registry:  reg,
		inventory: inv,
		bus:       bus,
		catalog:   cat,
		mapper:    NewChannelMapper(),
		driver:    NewAnimationDriver(reg),
		toys:      hub,
		ai:        NewPassthroughControl(),
		chat:      NewChatSession(DefaultPersona01, bus, cat, nil),
	}
	return p
}

func (p *Plugin) Registry() *Registry           { return p.registry }
func (p *Plugin) Inventory() *PropInventory     { return p.inventory }
func (p *Plugin) Catalog() *ActivityCatalog     { return p.catalog }
func (p *Plugin) Bus() *MotionBus               { return p.bus }
func (p *Plugin) Toys() *ToyHub                 { return p.toys }
func (p *Plugin) Chat() *ChatSession            { return p.chat }
func (p *Plugin) Mapper() *ChannelMapper        { return p.mapper }
func (p *Plugin) Driver() *AnimationDriver      { return p.driver }

// GiveDildo is the MVP scene step: give_toy(dildo).
func (p *Plugin) GiveDildo() error {
	return p.catalog.GiveProp(PropDildo)
}

// StartTitjob starts activity titjob_dildo (requires dildo in inventory).
func (p *Plugin) StartTitjob(intensity float64, durationS int) error {
	return p.catalog.Start(ActivityIntent{
		ID:        ActivityTitjobDildo,
		Intensity: intensity,
		DurationS: durationS,
		Source:    SourceActivity,
	})
}

// SetControlLoop swaps the AI/control strategy (MVP: passthrough).
func (p *Plugin) SetControlLoop(loop ControlLoop) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if loop == nil {
		loop = NewPassthroughControl()
	}
	p.ai = loop
}

// Start begins the control tick. Idempotent.
func (p *Plugin) Start(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.running {
		return nil
	}
	if p.registry.Active() == nil {
		return fmt.Errorf("virtualperson: no active character (register character-01 first)")
	}
	p.running = true
	p.lastTick = time.Now()
	_ = ctx
	return nil
}

// Stop ends the control tick and stops real toys.
func (p *Plugin) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.running = false
	_ = p.catalog.Stop()
	if p.toys != nil {
		p.toys.EmergencyStop()
	}
	if p.host != nil {
		if d := p.host.Device(); d != nil {
			_ = d.Stop()
		}
	}
}

// Running reports whether Start succeeded and Stop has not been called.
func (p *Plugin) Running() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.running
}

// Tick applies one control step. Funscript values publish to the bus;
// active activities advance; animation + optional ToyHub follow the winner.
func (p *Plugin) Tick(scriptPos float64, vib, suck float64) error {
	p.mu.RLock()
	running := p.running
	host := p.host
	ai := p.ai
	mapper := p.mapper
	driver := p.driver
	bus := p.bus
	catalog := p.catalog
	toys := p.toys
	inv := p.inventory
	last := p.lastTick
	p.mu.RUnlock()

	if !running {
		return nil
	}

	now := time.Now()
	dt := now.Sub(last).Seconds()
	if dt <= 0 || dt > 0.5 {
		dt = 1.0 / 30.0
	}
	p.mu.Lock()
	p.lastTick = now
	p.mu.Unlock()

	clock := int64(0)
	if host != nil {
		clock = host.NowMs()
	}

	// Funscript contribution — inactive while a scene activity is running so
	// virtual prop activities (e.g. titjob_dildo) own the bus; video sync
	// can re-enable by stopping the activity or a future pin flag.
	activityRunning := catalog.State() == StateTitjobActive
	intent := ai.Propose(ControlInput{
		AtMs:      clock,
		ScriptPos: scriptPos,
		Vibration: vib,
		Suction:   suck,
	})
	ch := mapper.Map(intent)
	ch.AtMs = clock
	bus.Publish(MotionContribution{
		Source:   SourceFunscript,
		Active:   !activityRunning,
		Channels: ch,
	})

	catalog.Tick(dt)

	out, src, persona := bus.Sample()
	out.AtMs = clock
	_ = src
	_ = persona

	pose, err := driver.Apply(out)
	if err != nil {
		return err
	}
	pose.Props = propSnapshots(inv, catalog.PropPhase())
	if host != nil {
		host.EmitAnimation(pose)
	}
	if toys != nil {
		if err := toys.Apply(out); err != nil {
			return err
		}
	}
	// Legacy: only seize device if ControlIntent says so AND sync is on.
	if host != nil && intent.DriveDevice && toys != nil && toys.SyncEnabled() {
		if d := host.Device(); d != nil {
			if err := d.SetVibration(intent.Vibration); err != nil {
				return err
			}
			if err := d.SetSuction(intent.Suction); err != nil {
				return err
			}
		}
	}
	return nil
}

func propSnapshots(inv *PropInventory, phase float64) []PropSnapshot {
	if inv == nil {
		return nil
	}
	eq := inv.Equipped()
	out := make([]PropSnapshot, 0, len(eq))
	for _, inst := range eq {
		out = append(out, PropSnapshot{
			PropID:      inst.Def.ID,
			Socket:      inst.Socket,
			PhaseOffset: phase,
			Visible:     true,
		})
	}
	return out
}
