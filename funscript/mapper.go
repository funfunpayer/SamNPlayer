package funscript

import (
	"fmt"
	"math"
	"strings"
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

// TrackingGap is a time window where at least one of the two Tf/Tj trackers
// lost its target. Contact vibration must stay off there — holding the last
// known distance would otherwise keep buzzing as if contact continued.
type TrackingGap struct {
	StartMs int64 `json:"start_ms"`
	EndMs   int64 `json:"end_ms"`
}

// Contact-curve names persisted in device_recipe.contact_vibration_curve.
const (
	ContactCurveLinear = "linear"
	ContactCurveSoft   = "soft" // weicher Einstieg: t²
	ContactCurvePeak   = "peak" // stärkerer Peak: √t
)

// DefaultContactEnvelopeSmooth: kurze Extra-Glättung nur für die
// Kontakt-Vibrationshüllkurve (Tracker-Jitter), unabhängig vom Sog-
// Smoothing der Recipe. Höher = träger.
const DefaultContactEnvelopeSmooth = 0.45

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

	// ContactVibrationSpan (0-1): welcher Anteil des Positions-Spektrums
	// als "Kontakt" zählt. 0 / außerhalb → DefaultContactVibrationSpan.
	// Niedriger = früher an; höher = nur tief.
	ContactVibrationSpan float64

	// ContactVibrationCurve: "linear" (default), "soft", "peak".
	ContactVibrationCurve string

	// ContactVibrationEnvelope: 0 → DefaultContactEnvelopeSmooth. Nur für
	// die Vibrationsspur bei Kontakt (Sog behält Smoothing).
	ContactVibrationEnvelope float64

	// TrackingGaps: Tracker-Verlustfenster — Vibration aus, Sog unverändert.
	TrackingGaps []TrackingGap
}

// DefaultContactVibrationSpan: oberstes Viertel des beobachteten
// Positions-Spektrums. Bewusst hoch gewählt statt eines festen
// Pixel-/Positionswerts, damit die Erkennung pro Video adaptiv bleibt.
const DefaultContactVibrationSpan = 0.75

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

// NormalizeContactCurve liefert einen kanonischen Kurvennamen.
func NormalizeContactCurve(name string) string {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case ContactCurveSoft, "soft_entry", "weicher":
		return ContactCurveSoft
	case ContactCurvePeak, "strong_peak", "peaky":
		return ContactCurvePeak
	default:
		return ContactCurveLinear
	}
}

// EffectiveContactSpan klammert die Empfindlichkeit in einen sinnvollen
// Bereich; 0 oder ungültig → Default.
func EffectiveContactSpan(span float64) float64 {
	if span <= 0 || span >= 1 {
		return DefaultContactVibrationSpan
	}
	if span < 0.4 {
		return 0.4
	}
	if span > 0.95 {
		return 0.95
	}
	return span
}

// applyContactCurve formt den linearen Nähe-Anteil t∈[0,1] um.
func applyContactCurve(t float64, curve string) float64 {
	t = clamp01(t)
	switch NormalizeContactCurve(curve) {
	case ContactCurveSoft:
		return t * t
	case ContactCurvePeak:
		return math.Sqrt(t)
	default:
		return t
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
	var prevContactVib float64
	envelope := opts.ContactVibrationEnvelope
	envelopeOn := true
	if envelope < 0 {
		envelopeOn = false
	} else if envelope == 0 || envelope >= 1 {
		envelope = DefaultContactEnvelopeSmooth
	}

	// Kontakt-Schwelle einmal über das ganze Skript bestimmen (nicht pro
	// Frame neu), aus den rohen Pos-Werten (0-100, bei tf/tj auf 20-90
	// geklemmt) - siehe ContactVibrationSpan.
	var contactMin, contactMax float64
	contactEnabled := opts.ContactVibration && opts.Sync == SyncSuctionPosition
	contactSpan := EffectiveContactSpan(opts.ContactVibrationSpan)
	contactCurve := NormalizeContactCurve(opts.ContactVibrationCurve)
	gaps := opts.TrackingGaps
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
			contactMin = posMin + contactSpan*(posMax-posMin)
			contactMax = posMax
		}
	}
	inGap := func(t int64) bool {
		for _, g := range gaps {
			if t >= g.StartMs && t <= g.EndMs {
				return true
			}
		}
		return false
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
			if contactEnabled && pos >= contactMin && !inGap(t) {
				linear := clamp01((pos - contactMin) / (contactMax - contactMin))
				vib = applyContactCurve(linear, contactCurve)
				if vib > 0 && opts.MinVibration > 0 {
					vib = liftFloor(vib, opts.MinVibration)
				}
			}
			// Kurze Envelope-Glättung nur für Kontakt-Vib (Tracker-Jitter),
			// bevor die allgemeine Recipe-Glättung auf Sog+Vib wirkt.
			if contactEnabled && envelopeOn {
				if len(frames) > 0 {
					vib = envelope*prevContactVib + (1-envelope)*vib
				}
				prevContactVib = vib
			}
		default:
			vib, suc = intensity, posSignal
		}
		if opts.Sync != SyncSuctionOnly && opts.Sync != SyncSuctionPosition {
			vib = liftFloor(vib, opts.MinVibration)
		}
		// SyncSuctionPosition: do NOT apply MinSuction via liftFloor.
		// Tf/Tj scripts already clamp actions to 20–90 at generation time, so
		// posSignal is already in [0.20, 0.90]. liftFloor(0.20, 0.20) would
		// remap that to 0.36 (and 0.90→0.92), stacking two floors and raising
		// resting suction for no gain — see docs/TF_TJ.md and the finding in
		// docs/FINDINGS_TIMING_TF.md. MinSuction remains in the recipe metadata
		// as the intended script-space floor (the clamp), not a second runtime remap.
		if opts.Sync != SyncVibrationOnly && opts.Sync != SyncSuctionPosition {
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
