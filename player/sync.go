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
// Extended-O funktioniert weiterhin über TriggerExtendedO, hält aber -
// anders als bei Play() - die tatsächliche Videowiedergabe nicht an
// (die läuft ja im Frontend weiter). PauseVideo/ResumeVideo-Hooks lassen
// sich setzen, um das Frontend während des Haltens zu pausieren, für ein
// stimmigeres Verhalten - siehe Player.PauseVideo/ResumeVideo.
func (p *Player) Sync(ctx context.Context, frames []funscript.Frame, positions <-chan int64) error {
	if len(frames) == 0 {
		return errNoFrames
	}
	logging.Info("player: Sync-Wiedergabe startet (externe Positionsquelle)", "frames", len(frames))
	defer func() {
		logging.Info("player: Sync-Wiedergabe beendet")
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
			if p.PauseVideo != nil {
				p.PauseVideo()
			}
			// curVib/curSuc: letzten bekannten Zustand nehmen - bei Sync
			// haben wir keine fortlaufende "curVib"-Variable wie bei Play(),
			// darum den zur letzten gemeldeten Position passenden Frame
			// erneut nachschlagen.
			f := frameAt(frames, p.lastSyncPosMs)
			if _, err := p.runExtendedO(ctx, opts, f.Vibration, f.Suction, p.lastSyncPosMs); err != nil {
				return err
			}
			if p.ResumeVideo != nil {
				p.ResumeVideo()
			}

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
			if p.OnFrame != nil {
				p.OnFrame(f)
			}
		}
	}
}

// setOutput schickt einen Frame direkt ans Gerät, ohne Soft-Start-Rampe -
// der Normalfall für jeden Frame nach dem allerersten.
func (p *Player) setOutput(f funscript.Frame) error {
	if err := p.Device.SetVibration(f.Vibration); err != nil {
		return err
	}
	return p.Device.SetSuction(f.Suction)
}

// softStart rampt in kleinen Schritten von 0 auf den Zielframe hoch, statt
// direkt zu springen - siehe Player.SoftStartMs. Genutzt beim allerersten
// Frame von Play() und Sync().
func (p *Player) softStart(ctx context.Context, target funscript.Frame) error {
	const steps = 10
	stepDur := time.Duration(p.SoftStartMs) * time.Millisecond / steps
	for i := 1; i <= steps; i++ {
		frac := float64(i) / float64(steps)
		if err := p.Device.SetVibration(target.Vibration * frac); err != nil {
			return err
		}
		if err := p.Device.SetSuction(target.Suction * frac); err != nil {
			return err
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
