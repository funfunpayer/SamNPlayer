// Package player führt eine resamplete funscript-Kurve auf einem
// device.Device ab - zeitgenau, unterbrechbar per Context, mit optional
// live triggerbarem Extended-O-Klimax-Modus.
package player

import (
	"context"
	"fmt"
	"time"

	"github.com/funfunpayer/SamNPlayer/device"
	"github.com/funfunpayer/SamNPlayer/funscript"
	"github.com/funfunpayer/SamNPlayer/logging"
)

// Player spielt eine Frame-Liste in Echtzeit ab.
type Player struct {
	Device device.Device
	// LogEvery gibt an, wie oft OnFrame-Statuszeilen erzeugt werden
	// (0 = aus, sonst z.B. alle 1s). Gilt unabhängig für die
	// Printf-Konsolenausgabe und den OnLog-Callback.
	LogEvery time.Duration

	// OnFrame wird nach jedem gesendeten Frame aufgerufen (z.B. für eine
	// Fortschrittsanzeige in einer GUI). Optional, darf nil sein. Läuft im
	// selben Goroutine wie Play() - bei GUI-Nutzung also selbst per
	// fyne.Do() o.ä. auf den UI-Thread wechseln.
	OnFrame func(f funscript.Frame)

	// OnLog wird zusätzlich zur Konsolenausgabe für Statuszeilen aufgerufen
	// (Start/Ende von Extended-O, periodische Fortschrittszeilen). Optional.
	OnLog func(line string)

	// PauseVideo/ResumeVideo sind optionale Hooks, die Sync() während eines
	// Extended-O-Haltens aufruft, um z.B. ein HTML5-<video>-Element im
	// Frontend zu pausieren/fortzusetzen - rein kosmetisch fürs
	// Zusammenspiel, Sync() funktioniert auch ohne (dann läuft das Video
	// während des Haltens einfach weiter).
	PauseVideo  func()
	ResumeVideo func()

	// SoftStartMs: >0 aktiviert sanftes Hochfahren beim Start von Play()
	// bzw. beim ersten Frame in Sync() - statt direkt auf den vollen
	// Skriptwert zu springen, wird über diese Dauer in kleinen Schritten
	// von 0 dorthin gerampt. Vermeidet einen spürbaren Ruck beim Loslegen.
	// 0 = aus (Standardverhalten wie bisher).
	SoftStartMs int

	lastSyncPosMs int64

	extendedOCh chan ExtendedOOptions
}

func New(dev device.Device) *Player {
	return &Player{
		Device:      dev,
		extendedOCh: make(chan ExtendedOOptions, 1),
	}
}

func (p *Player) logf(format string, args ...interface{}) {
	line := fmt.Sprintf(format, args...)
	fmt.Println(line)
	logging.Debug("player: " + line)
	if p.OnLog != nil {
		p.OnLog(line)
	}
}

// Play spielt die Frames ab. Blockiert bis zum Ende oder bis ctx
// abgebrochen wird. Am Ende (auch bei Abbruch/Fehler) wird Stop() auf dem
// Device aufgerufen, damit nichts "hängen bleibt".
//
// Wird währenddessen TriggerExtendedO() aufgerufen (typischerweise aus
// einer anderen Goroutine, z.B. einem stdin-Listener oder einem GUI-Button),
// pausiert Play die Skript-Timeline für die Dauer des Extended-O-Zyklus und
// setzt sie danach exakt dort fort, wo sie unterbrochen wurde - das Skript
// "verliert" also keine Sekunden, sondern die Gesamtwiedergabe verlängert
// sich um die Haltezeit.
func (p *Player) Play(ctx context.Context, frames []funscript.Frame) error {
	if len(frames) == 0 {
		return fmt.Errorf("player: keine Frames zum Abspielen")
	}
	logging.Info("player: Wiedergabe startet", "frames", len(frames), "dauer_ms", frames[len(frames)-1].At)
	defer func() {
		logging.Info("player: Wiedergabe beendet")
		p.Device.Stop() //nolint:errcheck // best effort beim Beenden
	}()

	start := time.Now()
	lastLog := start
	var curVib, curSuc float64

	if p.SoftStartMs > 0 && len(frames) > 0 {
		t0 := time.Now()
		if err := p.softStart(ctx, frames[0]); err != nil {
			return err
		}
		curVib, curSuc = frames[0].Vibration, frames[0].Suction
		start = start.Add(time.Since(t0)) // Zeitbasis um die Ramp-Dauer verschieben, sonst "hinkt" der Rest der Wiedergabe hinterher und holt hektisch auf
	}

	for _, f := range frames {
		// Anhängigen Extended-O-Trigger abarbeiten, bevor der nächste
		// Frame gewartet/gesendet wird.
		select {
		case opts := <-p.extendedOCh:
			elapsed, err := p.runExtendedO(ctx, opts, curVib, curSuc, f.At)
			start = start.Add(elapsed) // Timeline um Haltezeit verschieben
			if err != nil {
				return err
			}
		default:
		}

		target := start.Add(time.Duration(f.At) * time.Millisecond)
		if d := time.Until(target); d > 0 {
			select {
			case <-time.After(d):
			case opts := <-p.extendedOCh:
				// Trigger kam während des Wartens rein - sofort reagieren
				// statt bis zum nächsten Frame zu warten.
				elapsed, err := p.runExtendedO(ctx, opts, curVib, curSuc, f.At)
				start = start.Add(elapsed)
				if err != nil {
					return err
				}
				if d := time.Until(start.Add(time.Duration(f.At) * time.Millisecond)); d > 0 {
					select {
					case <-time.After(d):
					case <-ctx.Done():
						return ctx.Err()
					}
				}
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		if err := p.Device.SetVibration(f.Vibration); err != nil {
			return fmt.Errorf("player: SetVibration bei t=%dms: %w", f.At, err)
		}
		if err := p.Device.SetSuction(f.Suction); err != nil {
			return fmt.Errorf("player: SetSuction bei t=%dms: %w", f.At, err)
		}
		curVib, curSuc = f.Vibration, f.Suction

		if p.OnFrame != nil {
			p.OnFrame(f)
		}

		if p.LogEvery > 0 && time.Since(lastLog) >= p.LogEvery {
			p.logf("t=%6dms  vib=%.2f  suc=%.2f", f.At, f.Vibration, f.Suction)
			lastLog = time.Now()
		}

		if ctx.Err() != nil {
			return ctx.Err()
		}
	}
	return nil
}

// runExtendedO fährt Vibration/Sog auf opts.MinLevel, hält, und rampt
// zurück auf (curVib, curSuc). Gibt die insgesamt verstrichene Zeit zurück,
// damit der Aufrufer die Skript-Timeline entsprechend verschieben kann.
// atMs ist die "eingefrorene" Skript-Position (für OnFrame/Fortschrittsanzeigen -
// die Timeline pausiert ja während des gesamten Zyklus).
func (p *Player) runExtendedO(ctx context.Context, opts ExtendedOOptions, curVib, curSuc float64, atMs int64) (time.Duration, error) {
	t0 := time.Now()
	p.logf("[extended-o] halte bei %.0f%% für %s...", opts.MinLevel*100, opts.HoldDuration)

	if err := p.Device.SetVibration(opts.MinLevel); err != nil {
		return time.Since(t0), fmt.Errorf("player: extended-o SetVibration: %w", err)
	}
	if err := p.Device.SetSuction(opts.MinLevel); err != nil {
		return time.Since(t0), fmt.Errorf("player: extended-o SetSuction: %w", err)
	}
	if p.OnFrame != nil {
		p.OnFrame(funscript.Frame{At: atMs, Vibration: opts.MinLevel, Suction: opts.MinLevel})
	}

	select {
	case <-time.After(opts.HoldDuration):
	case <-ctx.Done():
		return time.Since(t0), ctx.Err()
	}

	if opts.RestoreDuration <= 0 {
		if err := p.Device.SetVibration(curVib); err != nil {
			return time.Since(t0), err
		}
		if err := p.Device.SetSuction(curSuc); err != nil {
			return time.Since(t0), err
		}
		if p.OnFrame != nil {
			p.OnFrame(funscript.Frame{At: atMs, Vibration: curVib, Suction: curSuc})
		}
	} else {
		const steps = 10
		stepDur := opts.RestoreDuration / steps
		for i := 1; i <= steps; i++ {
			frac := float64(i) / float64(steps)
			v := opts.MinLevel + frac*(curVib-opts.MinLevel)
			s := opts.MinLevel + frac*(curSuc-opts.MinLevel)
			if err := p.Device.SetVibration(v); err != nil {
				return time.Since(t0), err
			}
			if err := p.Device.SetSuction(s); err != nil {
				return time.Since(t0), err
			}
			if p.OnFrame != nil {
				p.OnFrame(funscript.Frame{At: atMs, Vibration: v, Suction: s})
			}
			select {
			case <-time.After(stepDur):
			case <-ctx.Done():
				return time.Since(t0), ctx.Err()
			}
		}
	}

	p.logf("[extended-o] wiederhergestellt, Wiedergabe läuft weiter")
	return time.Since(t0), nil
}
