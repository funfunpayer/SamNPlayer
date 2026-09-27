package virtualperson

// ChannelName is an animation / device semantic axis.
// Naming follows community funscript multi-axis conventions where possible
// (L0 stroke, L1 surge, L2 sway, R0 twist) plus Neo 2 vibe/suction and an
// expression channel for the Virtual Person face/body language.
type ChannelName string

const (
	ChannelStroke     ChannelName = "stroke"     // L0 / primary pos 0–100
	ChannelSurge      ChannelName = "surge"      // L1
	ChannelSway       ChannelName = "sway"       // L2
	ChannelTwist      ChannelName = "twist"      // R0
	ChannelVibe       ChannelName = "vibe"       // device vibration 0–1
	ChannelSuck       ChannelName = "suck"       // device suction 0–1
	ChannelExpression ChannelName = "expression" // face / body language 0–1
)

// ChannelValues is one sample of mapped animation channels (normalized).
// Stroke is 0–100 to match funscript.Action.Pos; vibe/suck/expression are 0–1.
type ChannelValues struct {
	AtMs       int64
	Stroke     float64
	Surge      float64
	Sway       float64
	Twist      float64
	Vibe       float64
	Suck       float64
	Expression float64
}

// ControlInput is what the control loop sees each tick.
type ControlInput struct {
	AtMs        int64
	ScriptPos   float64 // primary funscript position 0–100
	Vibration   float64 // already-mapped device vibe 0–1 (from player)
	Suction     float64 // already-mapped device suck 0–1
	CharacterID CharacterID
}

// ControlIntent is the AI/control proposal for this tick.
// DriveDevice=false means animate only (device follows the normal Player).
type ControlIntent struct {
	AtMs        int64
	ScriptPos   float64
	Vibration   float64
	Suction     float64
	Expression  float64
	DriveDevice bool
}

// ControlLoop turns script + context into intent. MVP is passthrough;
// later phases may learn from funscripts / film motion patterns using
// existing local tooling (profilemodel, OpenFunML/FunGen offline, ONNX).
type ControlLoop interface {
	Propose(in ControlInput) ControlIntent
}

// PassthroughControl copies player-mapped values and does not seize the device.
type PassthroughControl struct{}

// NewPassthroughControl returns the MVP control loop.
func NewPassthroughControl() *PassthroughControl { return &PassthroughControl{} }

// Propose implements ControlLoop.
func (p *PassthroughControl) Propose(in ControlInput) ControlIntent {
	return ControlIntent{
		AtMs:        in.AtMs,
		ScriptPos:   in.ScriptPos,
		Vibration:   in.Vibration,
		Suction:     in.Suction,
		Expression:  clamp01(in.Vibration*0.5 + in.Suction*0.5),
		DriveDevice: false,
	}
}

// ChannelMapper maps control intent to animation channels.
type ChannelMapper struct{}

// NewChannelMapper returns the default mapper.
func NewChannelMapper() *ChannelMapper { return &ChannelMapper{} }

// Map converts intent into ChannelValues for the animation driver.
func (m *ChannelMapper) Map(intent ControlIntent) ChannelValues {
	return ChannelValues{
		AtMs:       intent.AtMs,
		Stroke:     clampRange(intent.ScriptPos, 0, 100),
		Vibe:       clamp01(intent.Vibration),
		Suck:       clamp01(intent.Suction),
		Expression: clamp01(intent.Expression),
	}
}

// FunscriptAction is a minimal funscript point (avoids importing heavy
// metadata types into the plugin boundary). Compatible with funscript.Action.
type FunscriptAction struct {
	At  int64
	Pos int
}

// ParseFunscriptActions is a stub for loading actions from an already-parsed
// list. Real I/O stays in package funscript (funscript.Parse); Virtual Person
// code should call that and pass Actions here — do not duplicate the parser.
func ParseFunscriptActions(actions []FunscriptAction) []FunscriptAction {
	out := make([]FunscriptAction, len(actions))
	copy(out, actions)
	return out
}

// SampleAt linearly interpolates position at time tMs (same idea as player
// resampling). Returns 0 if empty.
func SampleAt(actions []FunscriptAction, tMs int64) float64 {
	if len(actions) == 0 {
		return 0
	}
	if tMs <= actions[0].At {
		return float64(actions[0].Pos)
	}
	last := actions[len(actions)-1]
	if tMs >= last.At {
		return float64(last.Pos)
	}
	for i := 1; i < len(actions); i++ {
		a, b := actions[i-1], actions[i]
		if tMs > b.At {
			continue
		}
		if b.At == a.At {
			return float64(b.Pos)
		}
		u := float64(tMs-a.At) / float64(b.At-a.At)
		return float64(a.Pos) + u*(float64(b.Pos)-float64(a.Pos))
	}
	return float64(last.Pos)
}

func clamp01(v float64) float64 { return clampRange(v, 0, 1) }

func clampRange(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
