package sam

import "github.com/funfunpayer/SamNPlayer/funscript"

// FromFunscript wandelt ein geladenes .funscript in ein SAM Motion Script
// um - reine Übernahme von Zeit und Position, ohne die reicheren SAM-Felder
// zu raten: ein importiertes Funscript hat nur Position gesetzt (Type bleibt
// MotionUnknown), bis ein echter Producer
// (docs/SAM_ARCHITECTURE.md, "Deferred") sie befüllt.
func FromFunscript(fs *funscript.Script) *Script {
	frames := make([]Frame, len(fs.Actions))
	for i, a := range fs.Actions {
		frames[i] = Frame{
			Time:   a.At,
			Motion: Motion{Position: float64(a.Pos)},
		}
	}
	return &Script{
		Version: ScriptVersion,
		Frames:  frames,
		Metadata: Metadata{
			Creator: fs.Metadata.Creator,
			Source:  "funscript-import",
		},
	}
}

// ToFunscript wandelt ein SAM Motion Script zurück in ein .funscript - nimmt
// nur Time/Position, die reicheren SAM-Felder gehen dabei verloren. Das ist
// der Zweck der Kompatibilitätsschicht: bestehende Player und Skripte
// funktionieren weiter, ohne SAM zu kennen (docs/SAM_ARCHITECTURE.md).
func ToFunscript(s *Script) *funscript.Script {
	actions := make([]funscript.Action, len(s.Frames))
	for i, f := range s.Frames {
		actions[i] = funscript.Action{
			At:  f.Time,
			Pos: int(f.Motion.Position),
		}
	}
	fs := &funscript.Script{Actions: actions}
	fs.Metadata.Creator = s.Metadata.Creator
	if len(actions) > 0 {
		fs.Metadata.Duration = actions[len(actions)-1].At
	}
	return fs
}
