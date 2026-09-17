package sam

import (
	"github.com/funfunpayer/SamNPlayer/funscript"
)

// ToDeviceFrames baut Geräte-Frames aus einem (typischerweise enriched)
// SAM-Skript. Sog folgt Position/Range; Kontakt-Vibration kommt aus
// Motion.Intensity (nicht erneut aus Pos geraten). TrackingGaps und
// Confidence=0 halten Vib bei 0.
//
// Das ist der erste Playback-Consumer des SAM-Modells: .funscript bleibt
// die Datei, SAM die Bewegungsbeschreibung dazwischen.
func ToDeviceFrames(s *Script, opts funscript.MapOptions) []funscript.Frame {
	if s == nil || len(s.Frames) < 2 {
		return nil
	}
	if opts.TickMs <= 0 {
		opts.TickMs = 50
	}
	duration := s.Duration()
	frames := make([]funscript.Frame, 0, duration/opts.TickMs+1)
	segIdx := 0
	var prevVib, prevSuc float64
	var prevContactVib float64
	envelope := opts.ContactVibrationEnvelope
	envelopeOn := true
	if envelope < 0 {
		envelopeOn = false
	} else if envelope == 0 || envelope >= 1 {
		envelope = funscript.DefaultContactEnvelopeSmooth
	}
	contactOn := opts.ContactVibration && opts.Sync == funscript.SyncSuctionPosition
	gaps := toFunscriptGaps(s.Metadata.TrackingGaps)

	inGap := func(t int64) bool {
		for _, g := range gaps {
			if t >= g.StartMs && t <= g.EndMs {
				return true
			}
		}
		return false
	}

	for t := int64(0); t <= duration; t += opts.TickMs {
		for segIdx < len(s.Frames)-2 && s.Frames[segIdx+1].Time <= t {
			segIdx++
		}
		a, b := s.Frames[segIdx], s.Frames[segIdx+1]
		dt := b.Time - a.Time
		if dt <= 0 {
			dt = 1
		}
		frac := float64(t-a.Time) / float64(dt)
		pos := a.Motion.Position + frac*(b.Motion.Position-a.Motion.Position)
		posSignal := clamp01(pos / 100.0)
		intens := a.Motion.Intensity + frac*(b.Motion.Intensity-a.Motion.Intensity)

		var vib, suc float64
		switch opts.Sync {
		case funscript.SyncSuctionPosition:
			suc = posSignal
			if a.Motion.Range > 0 || b.Motion.Range > 0 {
				ra, rb := a.Motion.Range, b.Motion.Range
				if ra == 0 {
					ra = clamp01(a.Motion.Position / 100)
				}
				if rb == 0 {
					rb = clamp01(b.Motion.Position / 100)
				}
				suc = ra + frac*(rb-ra)
			}
			if contactOn && !inGap(t) {
				vib = clamp01(intens)
				if vib > 0 && opts.MinVibration > 0 {
					vib = opts.MinVibration + vib*(1-opts.MinVibration)
				}
			}
			if contactOn && envelopeOn {
				if len(frames) > 0 {
					vib = envelope*prevContactVib + (1-envelope)*vib
				}
				prevContactVib = vib
			}
		default:
			// SAM-Consumer v1: nur Tf/Tj/Contact-Pfad; sonst Fallback leer.
			return nil
		}
		if opts.Smoothing > 0 && len(frames) > 0 {
			vib = opts.Smoothing*prevVib + (1-opts.Smoothing)*vib
			suc = opts.Smoothing*prevSuc + (1-opts.Smoothing)*suc
		}
		prevVib, prevSuc = vib, suc
		frames = append(frames, funscript.Frame{At: t, Vibration: vib, Suction: suc})
	}
	return frames
}

// PlaybackFramesFromFunscript: Enrich in-memory, dann ToDeviceFrames —
// .funscript-Datei unverändert, SAM dazwischen.
func PlaybackFramesFromFunscript(fs *funscript.Script, opts funscript.MapOptions) []funscript.Frame {
	if fs == nil {
		return nil
	}
	if !(opts.ContactVibration && opts.Sync == funscript.SyncSuctionPosition) {
		return fs.ToIntensityCurve(opts)
	}
	enriched := FromFunscriptEnriched(fs)
	out := ToDeviceFrames(enriched, opts)
	if out == nil {
		return fs.ToIntensityCurve(opts)
	}
	return out
}
