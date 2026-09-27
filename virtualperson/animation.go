package virtualperson

import "fmt"

// PoseSample is what the GUI overlay consumes each tick.
// Values are already mapped; the renderer decides Live2D params vs sprite
// offsets. MVP renderer: 2D sprite puppet / clip-strip (see pixel_figure.js)
// plus virtual prop sprites (e.g. dildo on chest during titjob).
type PoseSample struct {
	CharacterID CharacterID
	AtMs        int64
	Channels    ChannelValues
	Skin        SkinKind
	Props       []PropSnapshot
}

// AnimationDriver turns channel samples into pose samples for the active
// character. It does not draw — that stays in the Wails frontend.
type AnimationDriver struct {
	registry *Registry
}

// NewAnimationDriver binds a driver to a registry.
func NewAnimationDriver(reg *Registry) *AnimationDriver {
	return &AnimationDriver{registry: reg}
}

// Apply builds a PoseSample for the active character.
func (d *AnimationDriver) Apply(ch ChannelValues) (PoseSample, error) {
	if d == nil || d.registry == nil {
		return PoseSample{}, fmt.Errorf("virtualperson: animation driver not configured")
	}
	c := d.registry.Active()
	if c == nil {
		return PoseSample{}, fmt.Errorf("virtualperson: no active character")
	}
	return PoseSample{
		CharacterID: c.ID,
		AtMs:        ch.AtMs,
		Channels:    ch,
		Skin:        c.Skin,
	}, nil
}
