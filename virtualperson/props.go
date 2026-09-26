package virtualperson

import (
	"fmt"
	"sync"
)

// PropID identifies a virtual toy/item (not real hardware).
type PropID string

const (
	PropDildo PropID = "dildo"
	// Future: PropVibrator, PropWand, …
)

// Socket is an attachment point on the character body.
type Socket string

const (
	SocketHandR Socket = "hand_r"
	SocketHandL Socket = "hand_l"
	SocketChest Socket = "chest"
	SocketMouth Socket = "mouth"
	SocketHip   Socket = "hip"
	SocketNone  Socket = ""
)

// PropDef is the catalog entry for a virtual prop.
type PropDef struct {
	ID          PropID
	DisplayName string
	// DefaultSocket when first given.
	DefaultSocket Socket
	// SpriteRel path under character media pack, e.g. props/dildo.png
	SpriteRel string
	// CompatibleActivities lists activities that require this prop.
	CompatibleActivities []ActivityID
}

// DildoProp is the MVP virtual toy for titjob_dildo.
var DildoProp = PropDef{
	ID:            PropDildo,
	DisplayName:   "Dildo",
	DefaultSocket: SocketHandR,
	SpriteRel:     "props/dildo.png",
	CompatibleActivities: []ActivityID{
		ActivityTitjobDildo,
	},
}

// PropInstance is a prop owned by a character (inventory row).
type PropInstance struct {
	Def      PropDef
	Socket   Socket // empty = in inventory, not shown on body
	Equipped bool
}

// PropInventory is per-character virtual toy storage + attachments.
type PropInventory struct {
	mu       sync.RWMutex
	owner    CharacterID
	byID     map[PropID]*PropInstance
	catalog  map[PropID]PropDef
}

// NewPropInventory creates an empty inventory with the built-in prop catalog.
func NewPropInventory(owner CharacterID) *PropInventory {
	cat := map[PropID]PropDef{
		PropDildo: DildoProp,
	}
	return &PropInventory{
		owner:   owner,
		byID:    make(map[PropID]*PropInstance),
		catalog: cat,
	}
}

// Give adds a prop to inventory and equips it on its default socket.
func (inv *PropInventory) Give(id PropID) error {
	def, ok := inv.catalog[id]
	if !ok {
		return fmt.Errorf("virtualperson: unknown prop %q", id)
	}
	inv.mu.Lock()
	defer inv.mu.Unlock()
	inv.byID[id] = &PropInstance{
		Def:      def,
		Socket:   def.DefaultSocket,
		Equipped: true,
	}
	return nil
}

// Take removes a prop from the character.
func (inv *PropInventory) Take(id PropID) error {
	inv.mu.Lock()
	defer inv.mu.Unlock()
	if _, ok := inv.byID[id]; !ok {
		return fmt.Errorf("virtualperson: prop %q not in inventory", id)
	}
	delete(inv.byID, id)
	return nil
}

// Attach moves an owned prop to a socket (e.g. hand → chest for titjob).
func (inv *PropInventory) Attach(id PropID, socket Socket) error {
	inv.mu.Lock()
	defer inv.mu.Unlock()
	inst, ok := inv.byID[id]
	if !ok {
		return fmt.Errorf("virtualperson: prop %q not in inventory", id)
	}
	inst.Socket = socket
	inst.Equipped = socket != SocketNone
	return nil
}

// Has reports whether the prop is owned.
func (inv *PropInventory) Has(id PropID) bool {
	inv.mu.RLock()
	defer inv.mu.RUnlock()
	_, ok := inv.byID[id]
	return ok
}

// Equipped returns a copy of equipped props (for the animation driver).
func (inv *PropInventory) Equipped() []PropInstance {
	inv.mu.RLock()
	defer inv.mu.RUnlock()
	out := make([]PropInstance, 0, len(inv.byID))
	for _, inst := range inv.byID {
		if inst.Equipped {
			out = append(out, *inst)
		}
	}
	return out
}

// PropSnapshot is GUI-facing prop transform hint for one frame.
type PropSnapshot struct {
	PropID   PropID
	Socket   Socket
	// PhaseOffset 0–1 along the activity path (e.g. slide on chest).
	PhaseOffset float64
	Visible     bool
}
