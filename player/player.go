// Package player runs a resampled funscript/samn curve on a device.Device —
// time-accurate, cancelable, with optional Extended-O.
//
// Mobile share surface: this package (with device/, funscript/, samn/) is the
// intended core for a later iOS/Android *player-only* app. Do not import
// generator/ here. See docs/PLATFORMS.md.
package player

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
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

	// PauseVideo/ResumeVideo sind optionale Hooks (historisch für Extended-O
	// mit Video-Pause). Extended-O pausiert das Video nicht mehr - die Kurve
	// läuft weiter, nur die Amplitude sinkt - die Hooks bleiben für
	// mögliche andere Nutzung erhalten und werden vom Player nicht mehr
	// selbst aufgerufen.
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

	intensityMu    sync.Mutex
	intensityScale float64 // 1.0 = volle Kurvenhöhe; Extended-O senkt nur das
	eoBusy         atomic.Bool
}

func New(dev device.Device) *Player {
	return &Player{
		Device:         dev,
		extendedOCh:    make(chan ExtendedOOptions, 1),
		intensityScale: 1,
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
// läuft die Skript-Timeline weiter - Extended-O skaliert nur die Amplitude
// von Vibration/Sog (Kurve/Rhythmus bleiben), ohne die Wiedergabe zu
// pausieren oder zu verlängern.
func (p *Player) Play(ctx context.Context, frames []funscript.Frame) error {
	if len(frames) == 0 {
		return fmt.Errorf("player: keine Frames zum Abspielen")
	}
	logging.Info("player: Wiedergabe startet", "frames", len(frames), "dauer_ms", frames[len(frames)-1].At)
	defer func() {
		logging.Info("player: Wiedergabe beendet")
		p.setIntensityScale(1)
		p.Device.Stop() //nolint:errcheck // best effort beim Beenden
	}()

	start := time.Now()
	lastLog := start

	if p.SoftStartMs > 0 && len(frames) > 0 {
		t0 := time.Now()
		if err := p.softStart(ctx, frames[0]); err != nil {
			return err
		}
		start = start.Add(time.Since(t0)) // Zeitbasis um die Ramp-Dauer verschieben, sonst "hinkt" der Rest der Wiedergabe hinterher und holt hektisch auf
	}

	for _, f := range frames {
		p.drainExtendedO(ctx)

		target := start.Add(time.Duration(f.At) * time.Millisecond)
		if d := time.Until(target); d > 0 {
			select {
			case <-time.After(d):
			case opts := <-p.extendedOCh:
				// Trigger kam während des Wartens - Amplitude skalieren,
				// Timeline nicht anhalten.
				p.startExtendedO(ctx, opts)
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

		if err := p.setOutput(f); err != nil {
			return fmt.Errorf("player: Ausgabe bei t=%dms: %w", f.At, err)
		}

		if p.LogEvery > 0 && time.Since(lastLog) >= p.LogEvery {
			scale := p.getIntensityScale()
			p.logf("t=%6dms  vib=%.2f  suc=%.2f  scale=%.2f", f.At, f.Vibration*scale, f.Suction*scale, scale)
			lastLog = time.Now()
		}

		if ctx.Err() != nil {
			return ctx.Err()
		}
	}
	return nil
}
