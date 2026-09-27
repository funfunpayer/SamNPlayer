package virtualperson

import (
	"fmt"
	"sync"
	"time"
)

// ActivityID names a catalog activity / scene.
type ActivityID string

const (
	ActivityIdle         ActivityID = "idle"
	ActivityStop         ActivityID = "stop"
	ActivityTitjobDildo  ActivityID = "titjob_dildo"
	ActivityKiss         ActivityID = "kiss"
	ActivityStrokeSlow   ActivityID = "stroke_slow"
	ActivityStrokeFast   ActivityID = "stroke_fast"
	ActivityOralSuction  ActivityID = "oral_suction"
	ActivityTease        ActivityID = "tease"
	ActivityClimaxWindow ActivityID = "climax_window"
)

// ActivityIntent is a request to run (or stop) an activity.
type ActivityIntent struct {
	ID        ActivityID
	Intensity float64 // 0–1
	DurationS int     // 0 = until stop
	Source    SourceID
}

// ActivityState is the scene-graph state for the active activity.
type ActivityState string

const (
	StateIdle          ActivityState = "idle"
	StateReceivingProp ActivityState = "receiving_prop"
	StateEquipped      ActivityState = "equipped"
	StateActive        ActivityState = "active" // generic activity running
	StateTitjobActive  ActivityState = "titjob_active"
	StateStopping      ActivityState = "stopping"
)

// ActivityDef describes requirements and animation binding.
type ActivityDef struct {
	ID           ActivityID
	DisplayName  string
	RequiredProp PropID // empty = none
	AttachSocket Socket // where prop moves when activity starts
	// StrokeMin/Max: channel range while looping.
	StrokeMin float64
	StrokeMax float64
}

// TitjobDildo is the MVP scene: character performs a titjob with a virtual dildo.
var TitjobDildo = ActivityDef{
	ID:           ActivityTitjobDildo,
	DisplayName:  "Titjob (dildo)",
	RequiredProp: PropDildo,
	AttachSocket: SocketChest,
	StrokeMin:    20,
	StrokeMax:    90,
}

// ActivityCatalog holds defs + runtime scene state.
type ActivityCatalog struct {
	mu        sync.Mutex
	defs      map[ActivityID]ActivityDef
	inv       *PropInventory
	bus       *MotionBus
	state     ActivityState
	active    ActivityID
	intensity float64
	phase     float64 // 0–1 loop
	deadline  time.Time
}

// NewActivityCatalog wires inventory + bus.
func NewActivityCatalog(inv *PropInventory, bus *MotionBus) *ActivityCatalog {
	defs := map[ActivityID]ActivityDef{
		ActivityIdle:        {ID: ActivityIdle, DisplayName: "Idle"},
		ActivityStop:        {ID: ActivityStop, DisplayName: "Stop"},
		ActivityTitjobDildo: TitjobDildo,
		ActivityKiss: {
			ID: ActivityKiss, DisplayName: "Kiss",
			StrokeMin: 10, StrokeMax: 40,
		},
		ActivityStrokeSlow: {
			ID: ActivityStrokeSlow, DisplayName: "Stroke slow",
			RequiredProp: PropDildo, AttachSocket: SocketHandR,
			StrokeMin: 15, StrokeMax: 55,
		},
		ActivityStrokeFast: {
			ID: ActivityStrokeFast, DisplayName: "Stroke fast",
			RequiredProp: PropDildo, AttachSocket: SocketHandR,
			StrokeMin: 25, StrokeMax: 95,
		},
		ActivityOralSuction: {
			ID: ActivityOralSuction, DisplayName: "Oral / suction",
			RequiredProp: PropDildo, AttachSocket: SocketMouth,
			StrokeMin: 30, StrokeMax: 80,
		},
		ActivityTease: {
			ID: ActivityTease, DisplayName: "Tease",
			StrokeMin: 5, StrokeMax: 35,
		},
		ActivityClimaxWindow: {
			ID: ActivityClimaxWindow, DisplayName: "Climax window",
			StrokeMin: 60, StrokeMax: 100,
		},
	}
	return &ActivityCatalog{
		defs:  defs,
		inv:   inv,
		bus:   bus,
		state: StateIdle,
	}
}

// State returns the current scene-graph state.
func (c *ActivityCatalog) State() ActivityState {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.state
}

// ActiveID returns the current activity or idle.
func (c *ActivityCatalog) ActiveID() ActivityID {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.active == "" {
		return ActivityIdle
	}
	return c.active
}

// IsActivityRunning reports whether any non-idle activity owns the scene.
func (c *ActivityCatalog) IsActivityRunning() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.state == StateActive || c.state == StateTitjobActive
}

// GiveProp is the scene-graph step give_toy(id).
func (c *ActivityCatalog) GiveProp(id PropID) error {
	if c.inv == nil {
		return fmt.Errorf("virtualperson: no inventory")
	}
	if err := c.inv.Give(id); err != nil {
		return err
	}
	c.mu.Lock()
	c.state = StateEquipped
	c.mu.Unlock()
	return nil
}

// Start validates requirements and begins an activity (e.g. titjob_dildo).
func (c *ActivityCatalog) Start(intent ActivityIntent) error {
	if intent.ID == ActivityStop || intent.ID == ActivityIdle {
		return c.Stop()
	}
	def, ok := c.defs[intent.ID]
	if !ok {
		return fmt.Errorf("virtualperson: unknown activity %q", intent.ID)
	}
	if def.RequiredProp != "" {
		if c.inv == nil || !c.inv.Has(def.RequiredProp) {
			return fmt.Errorf("virtualperson: activity %q requires prop %q — call GiveProp first", intent.ID, def.RequiredProp)
		}
		if def.AttachSocket != "" {
			_ = c.inv.Attach(def.RequiredProp, def.AttachSocket)
		}
	}
	c.mu.Lock()
	c.active = intent.ID
	c.intensity = clamp01(intent.Intensity)
	if c.intensity == 0 {
		c.intensity = 0.5
	}
	c.phase = 0
	if intent.ID == ActivityTitjobDildo {
		c.state = StateTitjobActive
	} else {
		c.state = StateActive
	}
	if intent.DurationS > 0 {
		c.deadline = time.Now().Add(time.Duration(intent.DurationS) * time.Second)
	} else {
		c.deadline = time.Time{}
	}
	c.mu.Unlock()

	if contrib, ok := c.Contribution(intent); ok && c.bus != nil {
		c.bus.Publish(contrib)
	}
	return nil
}

// Stop ends the activity; props stay equipped.
func (c *ActivityCatalog) Stop() error {
	c.mu.Lock()
	c.active = ActivityIdle
	c.state = StateEquipped
	if c.inv == nil || len(c.inv.Equipped()) == 0 {
		c.state = StateIdle
	}
	c.mu.Unlock()
	if c.bus != nil {
		c.bus.Clear(SourceActivity)
	}
	return nil
}

// Tick advances the activity phase and republishes motion (call from Plugin.Tick).
func (c *ActivityCatalog) Tick(dtSec float64) {
	c.mu.Lock()
	if c.active == "" || c.active == ActivityIdle {
		c.mu.Unlock()
		return
	}
	if !c.deadline.IsZero() && time.Now().After(c.deadline) {
		c.mu.Unlock()
		_ = c.Stop()
		return
	}
	speed := 0.4 + c.intensity*1.2 // cycles per second-ish
	c.phase += dtSec * speed
	for c.phase >= 1 {
		c.phase -= 1
	}
	def := c.defs[c.active]
	intensity := c.intensity
	phase := c.phase
	id := c.active
	c.mu.Unlock()

	// Triangle-ish stroke from phase.
	t := phase
	if t > 0.5 {
		t = 1 - t
	}
	t *= 2 // 0–1–0
	stroke := def.StrokeMin + t*(def.StrokeMax-def.StrokeMin)
	ch := ChannelValues{
		Stroke:     stroke,
		Vibe:       intensity * 0.6,
		Suck:       intensity * (0.3 + 0.7*t),
		Expression: intensity,
	}
	if c.bus != nil {
		c.bus.Publish(MotionContribution{
			Source:   SourceActivity,
			Active:   true,
			Channels: ch,
			Persona: &PersonaState{
				Mood:         "intimate",
				Arousal:      intensity,
				LastActivity: id,
			},
		})
	}
}

// Contribution builds a one-shot bus publish for chat/start (static intensity).
func (c *ActivityCatalog) Contribution(intent ActivityIntent) (MotionContribution, bool) {
	def, ok := c.defs[intent.ID]
	if !ok || intent.ID == ActivityStop || intent.ID == ActivityIdle {
		return MotionContribution{}, false
	}
	if def.RequiredProp != "" && (c.inv == nil || !c.inv.Has(def.RequiredProp)) {
		return MotionContribution{}, false
	}
	inten := clamp01(intent.Intensity)
	if inten == 0 {
		inten = 0.5
	}
	mid := (def.StrokeMin + def.StrokeMax) / 2
	if def.StrokeMax == 0 {
		mid = 50
	}
	return MotionContribution{
		Source: SourceActivity,
		Active: true,
		Channels: ChannelValues{
			Stroke:     mid,
			Vibe:       inten * 0.6,
			Suck:       inten * 0.5,
			Expression: inten,
		},
		Persona: &PersonaState{
			Mood:         "intimate",
			Arousal:      inten,
			LastActivity: intent.ID,
		},
	}, true
}

// PropPhase for animation: dildo slide offset during titjob.
func (c *ActivityCatalog) PropPhase() float64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.phase
}
