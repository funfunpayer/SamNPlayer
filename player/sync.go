package player

import (
	"context"
	"errors"
	"sort"
	"time"

	"github.com/funfunpayer/SamNPlayer/funscript"
	"github.com/funfunpayer/SamNPlayer/logging"
)

var errNoFrames = errors.New("player: keine Frames zum Abspielen")

// Sync treibt das Gerät anhand einer EXTERNEN Positionsquelle statt der
// eigenen Uhr von Play() - gedacht für echte Videowiedergabe (z.B. ein
// HTML5-<video>-Element im Wails-Frontend), wo Pausieren, Spulen und die
// tatsächliche Abspielgeschwindigkeit außerhalb dieses Pakets passieren.
//
// positions liefert die aktuelle Videoposition in Millisekundem, typischerweise
// alle 100-250ms vom Aufrufer geschickt (z.B. per JS-Event aus dem Video-Tag).
// Bei jedem Wert wird der Frame gesucht, der zu dieser Position gehört, und
// sofort ausgegeben - kein Timing-Loop, kein "warten bis Zielzeit erreicht",
// da die Zeitbasis extern (vom Video) kommt statt von uns.
//
// Extended-O funktioniert über TriggerExtendedO und skaliert nur die
// Amplitude (Vib/Sog × Faktor) - Video und Skript-Rhythmus laufen weiter.
func (p *Player) Sync(ctx context.Context, frames []funscript.Frame, positions <-chan int64) error {
	if len(frames) == 0 {
		return errNoFrames
	}
	logging.Info("player: Sync-Wiedergabe startet (externe Positionsquelle)", "frames", len(frames))
	defer func() {
		logging.Info("player: Sync-Wiedergabe beendet")
		p.setIntensityScale(1)
		p.Device.Stop() //nolint:errcheck // best effort beim Beenden
	}()

	firstFrame := true

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case opts, ok := <-p.extendedOCh:
			if !ok {
				continue
			}
			p.startExtendedO(ctx, opts)

		case posMs, ok := <-positions:
			if !ok {
				return nil // Kanal geschlossen = Wiedergabe im Frontend beendet
			}
			p.lastSyncPosMs = posMs
			f := frameAt(frames, posMs)
			if firstFrame && p.SoftStartMs > 0 {
				if err := p.softStart(ctx, f); err != nil {
					return err
				}
			} else if err := p.setOutput(f); err != nil {
				return err
			}
			firstFrame = false
		}
	}
}

// setOutput schickt einen Frame skaliert ans Gerät (Extended-O senkt nur
// die Amplitude) und meldet den ausgegebenen Wert per OnFrame.
func (p *Player) setOutput(f funscript.Frame) error {
	scale := p.getIntensityScale()
	out := funscript.Frame{
		At:        f.At,
		Vibration: f.Vibration * scale,
		Suction:   f.Suction * scale,
	}
	if err := p.Device.SetVibration(out.Vibration); err != nil {
		return err
	}
	if err := p.Device.SetSuction(out.Suction); err != nil {
		return err
	}
	if p.OnFrame != nil {
		p.OnFrame(out)
	}
	return nil
}

// softStart rampt in kleinen Schritten von 0 auf den Zielframe hoch, statt
// direkt zu springen - siehe Player.SoftStartMs. Genutzt beim allerersten
// Frame von Play() und Sync(). Berücksichtigt den aktuellen Amplitudenfaktor.
func (p *Player) softStart(ctx context.Context, target funscript.Frame) error {
	const steps = 10
	stepDur := time.Duration(p.SoftStartMs) * time.Millisecond / steps
	scale := p.getIntensityScale()
	for i := 1; i <= steps; i++ {
		frac := float64(i) / float64(steps)
		out := funscript.Frame{
			At:        target.At,
			Vibration: target.Vibration * frac * scale,
			Suction:   target.Suction * frac * scale,
		}
		if err := p.Device.SetVibration(out.Vibration); err != nil {
			return err
		}
		if err := p.Device.SetSuction(out.Suction); err != nil {
			return err
		}
		if p.OnFrame != nil {
			p.OnFrame(out)
		}
		select {
		case <-time.After(stepDur):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

// frameAt sucht per Binärsuche den letzten Frame mit At <= posMs (bzw. den
// ersten, falls posMs davor liegt). Frames sind laut
// funscript.ToIntensityCurve() bereits zeitlich aufsteigend sortiert.
func frameAt(frames []funscript.Frame, posMs int64) funscript.Frame {
	i := sort.Search(len(frames), func(i int) bool { return frames[i].At > posMs })
	if i == 0 {
		return frames[0]
	}
	return frames[i-1]
}
