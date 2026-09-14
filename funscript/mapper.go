package funscript

import (
	"fmt"
	"math"
)

type SyncMode int

const (
	SyncIndependent SyncMode = iota
	SyncSynchronized
	SyncAlternating
	SyncVibrationOnly
	SyncSuctionOnly
	SyncSuctionPosition
)

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
	case "suction_position":
		return SyncSuctionPosition, nil
	default:
		return SyncIndependent, fmt.Errorf("funscript: unbekannter sync-mode %q", s)
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
	case SyncSuctionPosition:
		return "suction_position"
	default:
		return "independent"
	}
}

type Frame struct {
	At        int64
	Vibration float64
	Suction   float64
}

type MapOptions struct {
	TickMs       int64
	MaxSpeed     float64
	MinVibration float64
	MinSuction   float64
	Smoothing    float64
	Sync         SyncMode

	// ContactVibration: nur bei Sync == SyncSuctionPosition (tf/tj) wirksam.
	// Statt Vibration fest auf 0 zu halten, folgt sie dem Positionssignal
	// selbst, sobald es nahe sein eigenes, über das ganze Skript beobachtetes
	// Maximum steigt (ROI1 berührt/streift ROI2 - je nach ROI-Wahl z.B.
	// Eichel an Brustwarze oder Zunge). Dauer und Stärke der Vibration
	// ergeben sich so direkt aus dem gemessenen Abstandsverlauf dieses
	// Videos, statt aus einem festen Impuls - siehe docs/NEXT.md, Priorität
	// "Contact-triggered vibration for Tf/Tj". Reine Abstandsmessung, kein
	// Akt-Detektor.
	ContactVibration bool
}

// contactVibrationSpan (0-1) legt fest, welcher Anteil des in diesem Skript
// beobachteten Positions-Spektrums als "Kontakt" zählt: nur das oberste
// Viertel. Bewusst hoch gewählt statt eines festen Pixel-/Positionswerts,
// damit die Erkennung pro Video adaptiv bleibt (siehe ToIntensityCurve).
const contactVibrationSpan = 0.75

// contactVibrationMinSpan: liegt das gesamte Positions-Spektrum des Skripts
// darunter, gibt es zu wenig Variation, um "Kontakt" von normaler Bewegung
// zu unterscheiden - Vibration bleibt dann komplett aus, statt durchgehend
// zu brummen.
const contactVibrationMinSpan = 5.0

func DefaultMapOptions() MapOptions {
	return MapOptions{
		TickMs: 50, MaxSpeed: 0.6, MinVibration: 0.15,
		MinSuction: 0, Smoothing: 0.3, Sync: SyncIndependent,
	}
}

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

	// Kontakt-Schwelle einmal über das ganze Skript bestimmen (nicht pro
	// Frame neu), aus den rohen Pos-Werten (0-100, bei tf/tj auf 20-90
	// geklemmt) - siehe contactVibrationSpan.
	var contactMin, contactMax float64
	contactEnabled := opts.ContactVibration && opts.Sync == SyncSuctionPosition
	if contactEnabled {
		posMin, posMax := float64(s.Actions[0].Pos), float64(s.Actions[0].Pos)
		for _, a := range s.Actions[1:] {
			p := float64(a.Pos)
			if p < posMin {
				posMin = p
			}
			if p > posMax {
				posMax = p
			}
		}
		if posMax-posMin < contactVibrationMinSpan {
			contactEnabled = false
		} else {
			contactMin = posMin + contactVibrationSpan*(posMax-posMin)
			contactMax = posMax
		}
	}
	for t := int64(0); t <= duration; t += opts.TickMs {
		for segIdx < len(s.Actions)-2 && s.Actions[segIdx+1].At <= t {
			segIdx++
		}
		a, b := s.Actions[segIdx], s.Actions[segIdx+1]
		dt := b.At - a.At
		if dt <= 0 {
			dt = 1
		}
		speed := math.Abs(float64(b.Pos-a.Pos)) / float64(dt)
		intensity := clamp01(speed / opts.MaxSpeed)
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
		case SyncSuctionPosition:
			vib, suc = 0, posSignal
			if contactEnabled && pos >= contactMin {
				vib = clamp01((pos - contactMin) / (contactMax - contactMin))
				if vib > 0 && opts.MinVibration > 0 {
					vib = liftFloor(vib, opts.MinVibration)
				}
			}
		default:
			vib, suc = intensity, posSignal
		}
		if opts.Sync != SyncSuctionOnly && opts.Sync != SyncSuctionPosition {
			vib = liftFloor(vib, opts.MinVibration)
		}
		if opts.Sync != SyncVibrationOnly {
			suc = liftFloor(suc, opts.MinSuction)
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

func liftFloor(value, floor float64) float64 {
	if floor <= 0 {
		return value
	}
	if floor >= 1 {
		return 1
	}
	return floor + value*(1-floor)
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
