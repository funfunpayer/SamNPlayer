package main

import (
	"context"
	"fmt"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/funfunpayer/SamNPlayer/device"
	"github.com/funfunpayer/SamNPlayer/funscript"
	"github.com/funfunpayer/SamNPlayer/logging"
	"github.com/funfunpayer/SamNPlayer/player"
)

// PlaybackOptions kommt als JSON vom Frontend-Formular.
type PlaybackOptions struct {
	Mock               bool    `json:"mock"`
	SyncMode           string  `json:"syncMode"`
	TickMs             int64   `json:"tickMs"`
	MaxSpeed           float64 `json:"maxSpeed"`
	Smoothing          float64 `json:"smoothing"`
	SoftStartMs        int     `json:"softStartMs"`
	UseVideoSync       bool    `json:"useVideoSync"` // true = Sync() folgt Videoposition statt eigener Uhr
	ExtendedOEnabled   bool    `json:"extendedOEnabled"`
	ExtendedOMin       float64 `json:"extendedOMin"`
	ExtendedOHoldS     float64 `json:"extendedOHoldS"`
	ExtendedORestoreMs float64 `json:"extendedORestoreMs"`
}

// StartPlayback baut Gerät + Player auf und startet die Wiedergabe in einer
// eigenen Goroutine. Fortschritt/Log/Fehler kommen als Events zurück
// ("playback:frame", "playback:log", "playback:done", "playback:error"),
// da eine gebundene Methode nicht "nebenbei" laufend Werte pushen kann -
// das ist der Wails-Weg für sowas (siehe runtime.EventsEmit).
func (a *App) StartPlayback(opts PlaybackOptions) error {
	if a.currentScript == nil {
		return fmt.Errorf("kein Skript geladen")
	}

	mapOpts := funscript.DefaultMapOptions()
	if opts.TickMs > 0 {
		mapOpts.TickMs = opts.TickMs
	}
	if opts.MaxSpeed > 0 {
		mapOpts.MaxSpeed = opts.MaxSpeed
	}
	// Glättung wird immer 1:1 übernommen (anders als Tick/MaxSpeed oben) -
	// 0 ist hier ein gültiger, bewusster Wert ("keine Glättung"), kein
	// "nicht gesetzt". Das Frontend schickt immer einen expliziten Wert.
	mapOpts.Smoothing = opts.Smoothing
	syncMode, err := funscript.ParseSyncMode(opts.SyncMode)
	if err != nil {
		return err
	}
	mapOpts.Sync = syncMode

	frames := a.currentScript.ToIntensityCurve(mapOpts)
	if len(frames) == 0 {
		return fmt.Errorf("das Skript enthält keine abspielbaren Actions")
	}
	a.currentFrames = frames

	var dev device.Device
	if opts.Mock {
		dev = device.NewMock(false)
	} else {
		dev = device.NewSamNeo2(device.SamNeo2Protocol{})
	}
	a.activeDevice = dev

	p := player.New(dev)
	p.LogEvery = time.Second
	p.SoftStartMs = opts.SoftStartMs
	p.OnLog = func(line string) {
		runtime.EventsEmit(a.ctx, "playback:log", line)
	}
	p.OnFrame = func(f funscript.Frame) {
		runtime.EventsEmit(a.ctx, "playback:frame", map[string]any{
			"atMs":      f.At,
			"vibration": f.Vibration,
			"suction":   f.Suction,
			"totalMs":   frames[len(frames)-1].At,
		})
	}
	if opts.UseVideoSync {
		p.PauseVideo = func() { runtime.EventsEmit(a.ctx, "video:pause") }
		p.ResumeVideo = func() { runtime.EventsEmit(a.ctx, "video:resume") }
	}
	a.activePlayer = p

	ctx, err := a.tryStartSession()
	if err != nil {
		return err
	}

	go func() {
		defer a.endSession()

		connectCtx, connectCancel := context.WithTimeout(ctx, 20*time.Second)
		defer connectCancel()
		runtime.EventsEmit(a.ctx, "playback:log", "Verbinde...")
		if err := dev.Connect(connectCtx); err != nil {
			logging.Error("app: Verbindung fehlgeschlagen", "fehler", err)
			runtime.EventsEmit(a.ctx, "playback:error", err.Error())
			runtime.EventsEmit(a.ctx, "playback:done")
			return
		}
		defer dev.Disconnect()

		runtime.EventsEmit(a.ctx, "playback:log", "Wiedergabe startet...")

		var playErr error
		if opts.UseVideoSync {
			positions := make(chan int64, 4)
			a.videoPositionCh = positions
			playErr = p.Sync(ctx, frames, positions)
		} else {
			playErr = p.Play(ctx, frames)
		}

		if playErr != nil && playErr != context.Canceled {
			runtime.EventsEmit(a.ctx, "playback:error", playErr.Error())
		}
		runtime.EventsEmit(a.ctx, "playback:done")
	}()

	return nil
}

func (a *App) StopPlayback() {
	a.stopSession()
	if a.videoPositionCh != nil {
		close(a.videoPositionCh)
		a.videoPositionCh = nil
	}
}

// ReportVideoPosition wird vom Frontend periodisch aufgerufen (aus einem
// timeupdate-Listener am <video>-Element), solange UseVideoSync aktiv ist -
// treibt den Sync()-Modus im Player an.
func (a *App) ReportVideoPosition(ms int64) {
	if a.videoPositionCh == nil {
		return
	}
	// Skript-Offset hier anwenden und nirgends sonst: es gibt genau eine
	// Stelle, an der Videozeit auf Skriptzeit trifft. Würde der Offset im
	// Frontend auf die Anzeige und hier auf die Ansteuerung gerechnet,
	// liefen Kurve und Gerät auseinander.
	//
	// Positiver Offset bedeutet: das Skript läuft dem Video voraus und muss
	// später greifen - also wird die Skriptzeit zurückgesetzt.
	a.stateMu.RLock()
	offset := a.scriptOffsetMs
	a.stateMu.RUnlock()
	ms -= offset
	if ms < 0 {
		ms = 0
	}
	select {
	case a.videoPositionCh <- ms:
	default:
		// Kanal voll (Player kommt gerade nicht hinterher) - einen Tick
		// verwerfen ist unkritisch, der nächste kommt in ~100-250ms.
	}
}

func (a *App) TriggerExtendedO(minLevel, holdSeconds, restoreMs float64) {
	if a.activePlayer == nil {
		return
	}
	a.activePlayer.TriggerExtendedO(player.ExtendedOOptions{
		MinLevel:        minLevel,
		HoldDuration:    time.Duration(holdSeconds * float64(time.Second)),
		RestoreDuration: time.Duration(restoreMs * float64(time.Millisecond)),
	})
}

// HeatmapPoint ist ein Eimer der über die Skriptlänge grob gerasterten
// Intensitätskurve - fürs Heatmap-Band im Wiedergabe-Tab.
type HeatmapPoint struct {
	AtMs      int64   `json:"atMs"`
	Intensity float64 `json:"intensity"` // 0-1, max(Vibration, Suction) des Frames
}

// GetHeatmap liefert eine grob gerasterte Intensitätskurve des aktuell
// geladenen Skripts (unabhängig von den tatsächlichen Wiedergabe-Optionen -
// nutzt bewusst die Standardeinstellungen, damit die Heatmap unabhängig von
// z.B. Sync-Modus/Glättung immer dieselbe Form zeigt und nicht bei jeder
// Einstellungsänderung neu geladen werden muss). buckets bestimmt die
// gewünschte Anzahl Punkte (die Vorschau-Leiste ist typischerweise ein paar
// hundert Pixel breit, mehr Punkte bringen keinen sichtbaren Mehrwert).
// CurvePoint ist ein Punkt der Funscript-Kurve für die Darstellung.
type CurvePoint struct {
	AtMs int64 `json:"atMs"`
	Pos  int   `json:"pos"`
}

// GetScriptCurve liefert die Actions des geladenen Skripts für die
// Kurvendarstellung unter dem Video, bei Bedarf ausgedünnt.
//
// Die Ausdünnung behält bewusst pro Zeitfenster das Minimum UND das Maximum,
// statt einfach jeden n-ten Punkt zu nehmen: ein Funscript besteht fast nur
// aus Hoch- und Tiefpunkten, und einfaches Weglassen würde genau die
// Amplitude zerstören, die man sehen will - die Kurve sähe flacher aus, als
// sie ist.
func (a *App) GetScriptCurve(maxPoints int) ([]CurvePoint, error) {
	if a.currentScript == nil {
		return nil, fmt.Errorf("kein Skript geladen")
	}
	actions := a.currentScript.Actions
	if len(actions) == 0 {
		return nil, fmt.Errorf("skript enthält keine Actions")
	}
	if maxPoints < 100 {
		maxPoints = 100
	}
	if len(actions) <= maxPoints {
		out := make([]CurvePoint, len(actions))
		for i, act := range actions {
			out[i] = CurvePoint{AtMs: act.At, Pos: act.Pos}
		}
		return out, nil
	}

	// Zwei Punkte pro Fenster (Min und Max) - deshalb halb so viele Fenster
	// wie gewünschte Punkte.
	windows := maxPoints / 2
	span := actions[len(actions)-1].At - actions[0].At
	if span <= 0 {
		span = 1
	}
	out := make([]CurvePoint, 0, maxPoints)
	start := 0
	for w := 0; w < windows && start < len(actions); w++ {
		endAt := actions[0].At + span*int64(w+1)/int64(windows)
		minIdx, maxIdx := start, start
		i := start
		for ; i < len(actions) && (actions[i].At <= endAt || i == start); i++ {
			if actions[i].Pos < actions[minIdx].Pos {
				minIdx = i
			}
			if actions[i].Pos > actions[maxIdx].Pos {
				maxIdx = i
			}
		}
		// In zeitlicher Reihenfolge ausgeben, sonst zickzackt die Linie
		// rückwärts.
		lo, hi := minIdx, maxIdx
		if lo > hi {
			lo, hi = hi, lo
		}
		out = append(out, CurvePoint{AtMs: actions[lo].At, Pos: actions[lo].Pos})
		if hi != lo {
			out = append(out, CurvePoint{AtMs: actions[hi].At, Pos: actions[hi].Pos})
		}
		start = i
	}
	return out, nil
}

func (a *App) GetHeatmap(buckets int) ([]HeatmapPoint, error) {
	if a.currentScript == nil {
		return nil, fmt.Errorf("kein Skript geladen")
	}
	if buckets < 10 {
		buckets = 10
	}
	duration := a.currentScript.Duration()
	if duration <= 0 {
		return nil, fmt.Errorf("skript hat keine gültige Dauer")
	}

	opts := funscript.DefaultMapOptions()
	opts.TickMs = duration / int64(buckets)
	if opts.TickMs < 10 {
		opts.TickMs = 10
	}
	frames := a.currentScript.ToIntensityCurve(opts)

	points := make([]HeatmapPoint, len(frames))
	for i, f := range frames {
		intensity := f.Vibration
		if f.Suction > intensity {
			intensity = f.Suction
		}
		points[i] = HeatmapPoint{AtMs: f.At, Intensity: intensity}
	}
	return points, nil
}

// SetScriptOffset verschiebt das Skript gegen das Video, in Millisekunden.
//
// Fremde Skripte passen fast nie exakt zum eigenen Videoschnitt -
// Abweichungen von ein paar hundert Millisekunden sind der Normalfall, und
// ohne Korrektur fühlt sich die ganze Wiedergabe daneben an. Der Wert wirkt
// sofort, auch während die Wiedergabe läuft, damit man ihn im Hören
// nachjustieren kann statt blind vorher zu raten.
func (a *App) SetScriptOffset(ms int64) {
	if ms < -10000 {
		ms = -10000
	}
	if ms > 10000 {
		ms = 10000
	}
	a.stateMu.Lock()
	a.scriptOffsetMs = ms
	path := a.currentScriptPath
	a.stateMu.Unlock()

	// Pro Skript merken: der passende Offset hängt am Videoschnitt, nicht
	// an einer allgemeinen Vorliebe.
	if path != "" {
		_ = a.settings.Set(offsetKeyFor(path), ms)
	}
	logging.Info("wiedergabe: Skript-Offset gesetzt", "ms", ms, "skript", path)
}

// GetScriptOffset liefert den gespeicherten Offset des geladenen Skripts.
func (a *App) GetScriptOffset() int64 {
	a.stateMu.RLock()
	defer a.stateMu.RUnlock()
	return a.scriptOffsetMs
}

func offsetKeyFor(scriptPath string) string {
	return "playback.offset." + scriptPath
}
