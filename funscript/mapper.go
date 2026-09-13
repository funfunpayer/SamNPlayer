package funscript

import (
	"fmt"
	"math"
)

// Der Sam Neo 2 hat keinen linearen Aktor wie ein The Handy oder OSR2 -
// er kennt nur Vibrationsintensität und Sog-Intensität. Ein funscript
// beschreibt aber eine lineare Position (0-100) über die Zeit. Wir müssen
// die Positionskurve also in etwas übersetzen, das sich "richtig" anfühlt.
//
// Der in der Community übliche Ansatz für Nicht-Stroker-Geräte (Vibratoren,
// Sauggeräte) ist eine geschwindigkeitsbasierte Zuordnung: schnelle
// Positionswechsel im Skript -> hohe Intensität, langsame/ruhige Passagen
// -> niedrige Intensität. Das approximiert das Rhythmusgefühl des Skripts,
// auch wenn keine physische Auf-Ab-Bewegung stattfindet.
//
// Wie dieses eine Geschwindigkeitssignal auf die zwei unabhängigen Kanäle
// (Vibration, Sog) des Neo 2 verteilt wird, steuert SyncMode - siehe unten.

// SyncMode legt fest, wie Vibrations- und Sog-Kanal zueinander stehen.
// Die Bezeichnungen und Semantik orientieren sich am syncMode-Parameter
// des Referenzprojekts Kyure-A/mcp-svakom-samneo ("Combo"-Tool).
type SyncMode int

const (
	// SyncIndependent (Standard): Vibration folgt der Geschwindigkeit,
	// Sog folgt der aktuellen Position. Zwei unabhängige Signale.
	SyncIndependent SyncMode = iota
	// SyncSynchronized: beide Kanäle folgen demselben Geschwindigkeitssignal.
	SyncSynchronized
	// SyncAlternating: Kanäle laufen gegenläufig - ist Vibration hoch, ist
	// Sog niedrig und umgekehrt.
	SyncAlternating
	// SyncVibrationOnly: nur der Vibrationskanal wird angesteuert, Sog
	// bleibt immer 0. Nützlich für fremde/importierte Skripte, die man
	// gezielt nur auf einen Kanal legen will.
	SyncVibrationOnly
	// SyncSuctionOnly: Umkehrung von SyncVibrationOnly - nur Sog wird
	// angesteuert, Vibration bleibt immer 0.
	SyncSuctionOnly
)

// ParseSyncMode wandelt einen CLI-String in einen SyncMode um.
func ParseSyncMode(s string) (SyncMode, error) {
	switch s {
	case "", "independent":
		return SyncIndependent, nil
	case "synchronized":
		return SyncSynchronized, nil
	case "alternating":
		return SyncAlternating, nil
	case "vibration_only":
		return SyncVibrationOnly, nil
	case "suction_only":
		return SyncSuctionOnly, nil
	default:
		return SyncIndependent, fmt.Errorf("funscript: unbekannter sync-mode %q (erlaubt: independent, synchronized, alternating, vibration_only, suction_only)", s)
	}
}

func (m SyncMode) String() string {
	switch m {
	case SyncSynchronized:
		return "synchronized"
	case SyncAlternating:
		return "alternating"
	case SyncVibrationOnly:
		return "vibration_only"
	case SyncSuctionOnly:
		return "suction_only"
	default:
		return "independent"
	}
}

// Frame ist ein resampelter Steuerbefehl zu einem festen Zeitpunkt.
type Frame struct {
	At        int64   // Millisekunden ab Skriptstart
	Vibration float64 // 0.0 - 1.0
	Suction   float64 // 0.0 - 1.0
}

// MapOptions steuert die Umrechnung.
type MapOptions struct {
	// TickMs ist der Abstand zwischen zwei resampelten Frames.
	// Kleiner = feiner, aber mehr BLE-Schreibvorgänge. 50ms (20Hz) ist ein
	// vernünftiger Startwert; die meisten BLE-Sexspielzeuge vertragen keine
	// deutlich höhere Update-Rate zuverlässig.
	TickMs int64
	// MaxSpeed ist die Positionsänderung (in pos-Einheiten pro Millisekunde),
	// die als "volle Intensität" (1.0) gilt. Ein typischer schneller Stroke
	// (0->100 in ~150ms) liegt bei ca. 0.67. Alles darüber wird gekappt.
	MaxSpeed float64
	// MinVibration ist eine Grundintensität, die auch in ruhigen Passagen
	// nicht unterschritten wird (0 = Gerät pausiert komplett zwischen
	// Bewegungen, was manche Nutzer als abrupt empfinden).
	MinVibration float64
	// Smoothing glättet Sprünge zwischen Frames exponentiell (0 = kein
	// Smoothing, 0.8 = starkes Nachziehen). Reduziert "Knattern" bei sehr
	// dichten Skripten.
	Smoothing float64
	// Sync legt fest, wie das Geschwindigkeitssignal auf Vibration/Sog
	// verteilt wird (siehe SyncMode).
	Sync SyncMode
}

// DefaultMapOptions liefert plausible Startwerte.
func DefaultMapOptions() MapOptions {
	return MapOptions{
		TickMs:       50,
		MaxSpeed:     0.6,
		MinVibration: 0.15,
		Smoothing:    0.3,
		Sync:         SyncIndependent,
	}
}

// ToIntensityCurve wandelt die Actions in eine mit fester Rate
// resamplete Frame-Liste um, geeignet zum direkten Abspielen über einen
// device.Device.
func (s *Script) ToIntensityCurve(opts MapOptions) []Frame {
	if len(s.Actions) < 2 {
		return nil
	}
	if opts.TickMs <= 0 {
		opts.TickMs = 50
	}
	if opts.MaxSpeed <= 0 {
		opts.MaxSpeed = 0.6
	}

	duration := s.Duration()
	frames := make([]Frame, 0, duration/opts.TickMs+1)

	segIdx := 0
	var prevVib, prevSuc float64

	for t := int64(0); t <= duration; t += opts.TickMs {
		// Zum aktuellen Segment vorspulen (Actions sind sortiert).
		for segIdx < len(s.Actions)-2 && s.Actions[segIdx+1].At <= t {
			segIdx++
		}
		a, b := s.Actions[segIdx], s.Actions[segIdx+1]

		dt := b.At - a.At
		if dt <= 0 {
			dt = 1 // Div/0 bei doppelten Zeitstempeln vermeiden
		}
		speed := math.Abs(float64(b.Pos-a.Pos)) / float64(dt)
		intensity := clamp01(speed / opts.MaxSpeed)
		if intensity < opts.MinVibration {
			intensity = opts.MinVibration
		}

		// Interpolierte Position im Segment (für SyncIndependent).
		frac := float64(t-a.At) / float64(dt)
		pos := float64(a.Pos) + frac*float64(b.Pos-a.Pos)
		posSignal := clamp01(pos / 100.0)

		var vib, suc float64
		switch opts.Sync {
		case SyncSynchronized:
			vib, suc = intensity, intensity
		case SyncAlternating:
			vib, suc = intensity, clamp01(1-intensity)
		case SyncVibrationOnly:
			vib, suc = intensity, 0
		case SyncSuctionOnly:
			vib, suc = 0, intensity
		default: // SyncIndependent
			vib, suc = intensity, posSignal
		}

		if opts.Smoothing > 0 && len(frames) > 0 {
			vib = opts.Smoothing*prevVib + (1-opts.Smoothing)*vib
			suc = opts.Smoothing*prevSuc + (1-opts.Smoothing)*suc
		}
		prevVib, prevSuc = vib, suc

		frames = append(frames, Frame{At: t, Vibration: vib, Suction: suc})
	}

	return frames
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
