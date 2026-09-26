package virtualperson

import "sync"

// SourceID identifies who last wrote the motion bus.
type SourceID string

const (
	SourceIdle       SourceID = "idle"
	SourceFunscript  SourceID = "funscript"
	SourceActivity   SourceID = "activity"
	SourceChatPulse  SourceID = "chat"
)

// sourcePriority: higher wins while that source is active.
var sourcePriority = map[SourceID]int{
	SourceFunscript: 100,
	SourceActivity:  150,
	SourceChatPulse: 20,
	SourceIdle:      0,
}

// PersonaState is soft character state for expression / chat memory hooks.
// Adults only — no minor personas.
type PersonaState struct {
	Mood         string  // e.g. calm, teasing, intense
	Arousal      float64 // 0–1
	LastActivity ActivityID
}

// MotionContribution is one source's proposed channels for this tick.
type MotionContribution struct {
	Source   SourceID
	Active   bool
	Channels ChannelValues
	Persona  *PersonaState // optional patch
}

// MotionBus merges funscript playback, activity intents, and chat pulses
// into one coherent ChannelValues sample. LLM work must never block Tick:
// chat enqueues intents; the bus only reads the latest contribution.
type MotionBus struct {
	mu      sync.RWMutex
	contrib map[SourceID]MotionContribution
	persona PersonaState
}

// NewMotionBus returns an idle bus.
func NewMotionBus() *MotionBus {
	return &MotionBus{
		contrib: make(map[SourceID]MotionContribution),
		persona: PersonaState{Mood: "calm"},
	}
}

// Publish stores a contribution (latest wins per source).
func (b *MotionBus) Publish(c MotionContribution) {
	if c.Source == "" {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.contrib[c.Source] = c
	if c.Persona != nil {
		if c.Persona.Mood != "" {
			b.persona.Mood = c.Persona.Mood
		}
		b.persona.Arousal = clamp01(c.Persona.Arousal)
		if c.Persona.LastActivity != "" {
			b.persona.LastActivity = c.Persona.LastActivity
		}
	}
}

// Clear deactivates a source (e.g. activity finished).
func (b *MotionBus) Clear(src SourceID) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if c, ok := b.contrib[src]; ok {
		c.Active = false
		b.contrib[src] = c
	}
}

// Sample picks the highest-priority active contribution.
func (b *MotionBus) Sample() (ChannelValues, SourceID, PersonaState) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	bestPri := -1
	best := MotionContribution{Source: SourceIdle, Active: true}
	for _, c := range b.contrib {
		if !c.Active {
			continue
		}
		p := sourcePriority[c.Source]
		if p > bestPri {
			bestPri = p
			best = c
		}
	}
	// Sample is called under RLock; record winner via separate write path.
	// Callers that need WinningSource should use the returned SourceID.
	return best.Channels, best.Source, b.persona
}

// Persona returns a copy of soft state.
func (b *MotionBus) Persona() PersonaState {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.persona
}

// WinningSource is retained for API compatibility; prefer Sample's SourceID.
func (b *MotionBus) WinningSource() SourceID {
	_, src, _ := b.Sample()
	return src
}
