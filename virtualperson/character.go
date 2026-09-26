package virtualperson

import (
	"fmt"
	"sync"
)

// CharacterID is a stable registry key (e.g. "character-01").
type CharacterID string

// SkinKind describes how the character is rendered in the GUI.
// MVP targets a 2D puppet / clip-strip driven by animation channels.
// Live2D and glTF are reserved for later once real assets exist.
type SkinKind string

const (
	SkinSpritePuppet SkinKind = "sprite_puppet" // MVP: layered 2D / clip frames
	SkinLive2D       SkinKind = "live2d"        // future: Cubism model
	SkinGLTF         SkinKind = "gltf"          // future: 3D
)

// Character is one Virtual Person. First character is character-01; the
// registry supports multiple entries for later expansion.
type Character struct {
	ID          CharacterID
	DisplayName string
	Skin        SkinKind
	// RefImage is a path relative to the character pack (authoritative look).
	// Only the first reference image is the look; other promo images are
	// style/mood helpers and must not be treated as alternate primary assets.
	RefImage string
	// AnimationChannels lists channels this skin can consume.
	AnimationChannels []ChannelName
	Meta              map[string]string
}

// Character01 is the first Virtual Person stub. Ref image is expected at
// media/character-01/primary.jpg in the Project store (and optionally
// mirrored under cmd/gui-wails/frontend assets when the overlay ships).
var Character01 = Character{
	ID:          "character-01",
	DisplayName: "Character 01",
	Skin:        SkinSpritePuppet,
	RefImage:    "character-01/primary.jpg",
	AnimationChannels: []ChannelName{
		ChannelStroke,
		ChannelSurge,
		ChannelSway,
		ChannelTwist,
		ChannelVibe,
		ChannelSuck,
		ChannelExpression,
	},
	Meta: map[string]string{
		"status": "look_from_primary_ref",
		"note":   "authoritative look = first reference image only",
	},
}

// Registry holds Virtual Persons and tracks the active one.
type Registry struct {
	mu       sync.RWMutex
	byID     map[CharacterID]*Character
	activeID CharacterID
}

// NewRegistry returns an empty registry.
func NewRegistry() *Registry {
	return &Registry{byID: make(map[CharacterID]*Character)}
}

// Register adds or replaces a character. Does not change the active ID
// unless the registry was empty.
func (r *Registry) Register(c Character) error {
	if c.ID == "" {
		return fmt.Errorf("virtualperson: character id required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := c
	r.byID[c.ID] = &cp
	if r.activeID == "" {
		r.activeID = c.ID
	}
	return nil
}

// SetActive selects the active character for animation / AI control.
func (r *Registry) SetActive(id CharacterID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.byID[id]; !ok {
		return fmt.Errorf("virtualperson: unknown character %q", id)
	}
	r.activeID = id
	return nil
}

// Active returns a copy of the active character, or nil.
func (r *Registry) Active() *Character {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.byID[r.activeID]
	if !ok || c == nil {
		return nil
	}
	cp := *c
	return &cp
}

// List returns registered characters in insertion-unspecified order.
func (r *Registry) List() []Character {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Character, 0, len(r.byID))
	for _, c := range r.byID {
		out = append(out, *c)
	}
	return out
}

// EnsureCharacter01 registers the first character if missing.
func (r *Registry) EnsureCharacter01() error {
	r.mu.RLock()
	_, ok := r.byID[Character01.ID]
	r.mu.RUnlock()
	if ok {
		return nil
	}
	return r.Register(Character01)
}
