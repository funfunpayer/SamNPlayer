package main

import (
	"context"
	"fmt"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/funfunpayer/SamNPlayer/funscript"
	"github.com/funfunpayer/SamNPlayer/logging"
	"github.com/funfunpayer/SamNPlayer/player"
	"github.com/funfunpayer/SamNPlayer/sam"
)

type PlaybackOptions struct {
	Mock                    bool    `json:"mock"`
	SyncMode                string  `json:"syncMode"`
	TickMs                  int64   `json:"tickMs"`
	MaxSpeed                float64 `json:"maxSpeed"`
	Smoothing               float64 `json:"smoothing"`
	SoftStartMs             int     `json:"softStartMs"`
	UseVideoSync            bool    `json:"useVideoSync"`
	ExtendedOEnabled        bool    `json:"extendedOEnabled"`
	ExtendedOMin            float64 `json:"extendedOMin"`
	ExtendedOHoldS          float64 `json:"extendedOHoldS"`
	ExtendedORestoreMs      float64 `json:"extendedORestoreMs"`
	DisableContactVibration bool    `json:"disableContactVibration"`
	// ContactIntensityScale: live SAM-Korrektur 0–2 (1=Rezept), Datei unverändert.
	ContactIntensityScale float64 `json:"contactIntensityScale"`
	// ContactExtraSmooth: zusätzliche EMA nach dem Mapping (0=aus).
	ContactExtraSmooth float64 `json:"contactExtraSmooth"`
	// ContactVibrationSpan / Curve: Live-Override der Empfindlichkeit/Kurve
	// (0 / "" = aus DeviceRecipe). Datei bleibt unverändert.
	ContactVibrationSpan  float64 `json:"contactVibrationSpan"`
	ContactVibrationCurve string  `json:"contactVibrationCurve"`
}

// ContactPreviewOptions steuert die zweite Kurvenspur inkl. Live-Overrides.
type ContactPreviewOptions struct {
	MaxPoints             int     `json:"maxPoints"`
	ContactVibrationSpan  float64 `json:"contactVibrationSpan"`
	ContactVibrationCurve string  `json:"contactVibrationCurve"`
	ContactIntensityScale float64 `json:"contactIntensityScale"`
	MuteContact           bool    `json:"muteContact"`
}

// SaveContactSettings schreibt Kontakt-Vibration in die Skript-Metadata
// (device_recipe) und lädt das Skript neu — damit Preview und nächste
// Wiedergabe denselben „wie die Berührung“-Stand nutzen.
func (a *App) SaveContactSettings(enabled bool, span float64, curve string) error {
	path := a.loadedScriptPath()
	if path == "" {
		return fmt.Errorf("kein Skript geladen")
	}
	script := a.loadedScript()
	if script == nil {
		return fmt.Errorf("kein Skript geladen")
	}
	if !funscript.IsDistanceProfile(script.Metadata.Profile) {
		return fmt.Errorf("Kontakt-Vibration nur bei Tf/Tj-Skripten")
	}
	if err := funscript.SaveContactRecipe(path, enabled, span, curve); err != nil {
		return err
	}
	reloaded, err := funscript.Load(path)
	if err != nil {
		return err
	}
	a.setLoadedScript(path, reloaded)
	return nil
}

func (a *App) StartPlayback(opts PlaybackOptions) error {
	script := a.loadedScript()
	if script == nil {
		return fmt.Errorf("kein Skript geladen")
	}
	mapOpts := funscript.DefaultMapOptions()
	profile := script.Metadata.Profile
	contactOn := false
	if funscript.IsDistanceProfile(profile) {
		mapOpts = funscript.RecipeFor(profile)
		if dr := script.Metadata.DeviceRecipe; dr != nil {
			mapOpts.ContactVibration = dr.ContactVibration
			mapOpts.ContactVibrationSpan = dr.ContactVibrationSpan
			mapOpts.ContactVibrationCurve = dr.ContactVibrationCurve
		}
		if opts.ContactVibrationSpan > 0 {
			mapOpts.ContactVibrationSpan = opts.ContactVibrationSpan
		}
		if opts.ContactVibrationCurve != "" {
			mapOpts.ContactVibrationCurve = opts.ContactVibrationCurve
		}
		if len(script.Metadata.TrackingGaps) > 0 {
			mapOpts.TrackingGaps = append([]funscript.TrackingGap(nil), script.Metadata.TrackingGaps...)
		}
		if opts.DisableContactVibration {
			mapOpts.ContactVibration = false
		}
		contactOn = mapOpts.ContactVibration
	}
	if opts.TickMs > 0 {
		mapOpts.TickMs = opts.TickMs
	}
	if opts.MaxSpeed > 0 {
		mapOpts.MaxSpeed = opts.MaxSpeed
	}
	mapOpts.Smoothing = opts.Smoothing
	syncMode, err := funscript.ParseSyncMode(opts.SyncMode)
	if err != nil {
		return err
	}
	if !funscript.IsDistanceProfile(profile) || (opts.SyncMode != "" && opts.SyncMode != "independent") {
		mapOpts.Sync = syncMode
	}
	// Tf/Tj + Kontakt: SAM-Modell dazwischen (Intensity/Gaps), Datei bleibt .funscript.
	var frames []funscript.Frame
	if contactOn {
		frames = a.contactFrames(script, mapOpts)
	} else {
		frames = script.ToIntensityCurve(mapOpts)
	}
	frames = sam.AdjustDeviceFrames(frames, sam.RuntimeAdjust{
		IntensityScale: opts.ContactIntensityScale,
		ExtraSmooth:    opts.ContactExtraSmooth,
		MuteContact:    opts.DisableContactVibration || opts.ContactIntensityScale <= 0,
	})
	if len(frames) == 0 {
		return fmt.Errorf("das Skript enthält keine abspielbaren Actions")
	}
	a.setCurrentFrames(frames)
	dev, reusedDevice := a.claimSessionDevice(opts.Mock)
	p := player.New(dev)
	p.LogEvery = time.Second
	p.SoftStartMs = opts.SoftStartMs
	p.OnLog = func(line string) { runtime.EventsEmit(a.ctx, "playback:log", line) }
	p.OnFrame = func(f funscript.Frame) {
		runtime.EventsEmit(a.ctx, "playback:frame", map[string]any{
			"atMs": f.At, "vibration": f.Vibration, "suction": f.Suction, "totalMs": frames[len(frames)-1].At,
		})
	}
	if opts.UseVideoSync {
		p.PauseVideo = func() { runtime.EventsEmit(a.ctx, "video:pause") }
		p.ResumeVideo = func() { runtime.EventsEmit(a.ctx, "video:resume") }
	}
	// Unter stateMu wie jedes andere geteilte Feld auf App (siehe dessen
	// eigene Deklaration) - vorher unguarded gesetzt, während
	// TriggerExtendedO a.activePlayer ebenso unguarded liest: eine echte
	// Datenwettlauf zwischen einem Wiedergabe-Start und einem Klick auf
	// "Extended-O auslösen" kurz danach.
	a.stateMu.Lock()
	a.activeDevice = dev
	a.activePlayer = p
	a.stateMu.Unlock()
	ctx, err := a.tryStartSession()
	if err != nil {
		return err
	}
	go func() {
		defer a.endSession()
		if reusedDevice {
			// Bereits über den Geräte-Tab verbunden - nicht erneut
			// verbinden und am Ende NICHT trennen, das bleibt dessen
			// Sache (siehe claimSessionDevice).
			runtime.EventsEmit(a.ctx, "playback:log", "Nutze die bestehende Verbindung aus dem Geräte-Tab.")
		} else {
			connectCtx, connectCancel := context.WithTimeout(ctx, 20*time.Second)
			defer connectCancel()
			runtime.EventsEmit(a.ctx, "playback:log", "Verbinde...")
			if err := dev.Connect(connectCtx); err != nil {
				logging.Error("app: Verbindung fehlgeschlagen", "fehler", err)
				runtime.EventsEmit(a.ctx, "playback:error", err.Error())
				// failed:true — frontend must NOT auto-advance the playlist
				runtime.EventsEmit(a.ctx, "playback:done", map[string]any{"failed": true})
				return
			}
			defer dev.Disconnect()
		}
		runtime.EventsEmit(a.ctx, "playback:log", "Wiedergabe startet...")
		if contactOn {
			runtime.EventsEmit(a.ctx, "playback:log", "Kontakt-Vibration aktiv (SAM)")
		}
		var playErr error
		if opts.UseVideoSync {
			positions := make(chan int64, 4)
			a.stateMu.Lock()
			a.videoPositionCh = positions
			a.stateMu.Unlock()
			playErr = p.Sync(ctx, frames, positions)
		} else {
			playErr = p.Play(ctx, frames)
		}
		if playErr != nil && playErr != context.Canceled {
			runtime.EventsEmit(a.ctx, "playback:error", playErr.Error())
			runtime.EventsEmit(a.ctx, "playback:done", map[string]any{"failed": true})
			return
		}
		runtime.EventsEmit(a.ctx, "playback:done", map[string]any{"failed": false})
	}()
	return nil
}

func (a *App) StopPlayback() {
	a.stopSession()
	// Lock() statt RLock(), und zwar über den ganzen close() hinweg: eine
	// laufende ReportVideoPosition() hält für ihren send bereits RLock() -
	// Lock() blockiert also so lange, bis dieser send fertig ist, und kann
	// den Kanal nie schließen, während gleichzeitig darauf gesendet wird
	// ("panic: send on closed channel", reproduziert mit go test -race und
	// ohne -race unter Last).
	a.stateMu.Lock()
	ch := a.videoPositionCh
	a.videoPositionCh = nil
	a.stateMu.Unlock()
	if ch != nil {
		close(ch)
	}
	// Wait until the playback goroutine's endSession runs — otherwise
	// playNextInPlaylist → play() races tryStartSession (sessionActive).
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		a.stateMu.RLock()
		active := a.sessionActive
		a.stateMu.RUnlock()
		if !active {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func (a *App) ReportVideoPosition(ms int64) {
	// RLock() muss den send mit abdecken, nicht nur das Lesen des Kanals -
	// sonst kann StopPlayback() den Kanal zwischen Prüfung und send
	// schließen (siehe Kommentar dort).
	a.stateMu.RLock()
	defer a.stateMu.RUnlock()
	if a.videoPositionCh == nil {
		return
	}
	ms -= a.scriptOffsetMs
	if ms < 0 {
		ms = 0
	}
	select {
	case a.videoPositionCh <- ms:
	default:
	}
}

func (a *App) TriggerExtendedO(minLevel, holdSeconds, restoreMs float64) {
	a.stateMu.RLock()
	p := a.activePlayer
	a.stateMu.RUnlock()
	if p == nil {
		return
	}
	p.TriggerExtendedO(player.ExtendedOOptions{
		MinLevel:        minLevel,
		HoldDuration:    time.Duration(holdSeconds * float64(time.Second)),
		RestoreDuration: time.Duration(restoreMs * float64(time.Millisecond)),
	})
}

type HeatmapPoint struct {
	AtMs      int64   `json:"atMs"`
	Intensity float64 `json:"intensity"`
}

type CurvePoint struct {
	AtMs int64 `json:"atMs"`
	Pos  int   `json:"pos"`
}

// VibrationCurvePoint is the contact-vibration intensity (0-1) over time —
// second track under the position curve when device_recipe.contact_vibration
// is set. Derived from the same distance/pos signal as playback, so it lines
// up with the video timeline.
type VibrationCurvePoint struct {
	AtMs      int64   `json:"atMs"`
	Vibration float64 `json:"vibration"`
}

func (a *App) GetScriptCurve(maxPoints int) ([]CurvePoint, error) {
	script := a.loadedScript()
	if script == nil {
		return nil, fmt.Errorf("kein Skript geladen")
	}
	actions := script.Actions
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

// GetVibrationCurve returns the contact-vibration envelope for the loaded
// script (empty if none). Uses the baked-in device_recipe (no live override).
func (a *App) GetVibrationCurve(maxPoints int) ([]VibrationCurvePoint, error) {
	return a.GetVibrationCurvePreview(ContactPreviewOptions{
		MaxPoints: maxPoints, ContactIntensityScale: 1,
	})
}

// GetVibrationCurvePreview applies live Span/Kurve/Stärke-Overrides — gleiche
// Semantik wie StartPlayback, ohne die Datei anzufassen.
func (a *App) GetVibrationCurvePreview(preview ContactPreviewOptions) ([]VibrationCurvePoint, error) {
	script := a.loadedScript()
	if script == nil {
		return nil, fmt.Errorf("kein Skript geladen")
	}
	dr := script.Metadata.DeviceRecipe
	if dr == nil || !dr.ContactVibration || !funscript.IsDistanceProfile(script.Metadata.Profile) {
		return nil, nil
	}
	maxPoints := preview.MaxPoints
	if maxPoints < 50 {
		maxPoints = 50
	}
	opts := funscript.RecipeFor(script.Metadata.Profile)
	opts.ContactVibration = true
	opts.ContactVibrationSpan = dr.ContactVibrationSpan
	opts.ContactVibrationCurve = dr.ContactVibrationCurve
	if preview.ContactVibrationSpan > 0 {
		opts.ContactVibrationSpan = preview.ContactVibrationSpan
	}
	if preview.ContactVibrationCurve != "" {
		opts.ContactVibrationCurve = preview.ContactVibrationCurve
	}
	opts.TrackingGaps = append([]funscript.TrackingGap(nil), script.Metadata.TrackingGaps...)
	opts.Smoothing = 0
	opts.ContactVibrationEnvelope = -1
	duration := script.Duration()
	if duration <= 0 {
		return nil, nil
	}
	opts.TickMs = duration / int64(maxPoints)
	if opts.TickMs < 10 {
		opts.TickMs = 10
	}
	frames := a.contactFrames(script, opts)
	scale := preview.ContactIntensityScale
	mute := preview.MuteContact
	if scale <= 0 {
		mute = true
		scale = 1
	}
	frames = sam.AdjustDeviceFrames(frames, sam.RuntimeAdjust{
		IntensityScale: scale,
		MuteContact:    mute,
	})
	out := make([]VibrationCurvePoint, 0, len(frames))
	any := false
	for _, f := range frames {
		if f.Vibration > 0.001 {
			any = true
		}
		out = append(out, VibrationCurvePoint{AtMs: f.At, Vibration: f.Vibration})
	}
	if !any {
		return nil, nil
	}
	return out, nil
}

func (a *App) GetHeatmap(buckets int) ([]HeatmapPoint, error) {
	script := a.loadedScript()
	if script == nil {
		return nil, fmt.Errorf("kein Skript geladen")
	}
	if buckets < 10 {
		buckets = 10
	}
	duration := script.Duration()
	if duration <= 0 {
		return nil, fmt.Errorf("skript hat keine gültige Dauer")
	}
	opts := funscript.DefaultMapOptions()
	opts.TickMs = duration / int64(buckets)
	if opts.TickMs < 10 {
		opts.TickMs = 10
	}
	frames := script.ToIntensityCurve(opts)
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

// contactFrames: bevorzugt vorhandenes .sam-Sidecar mit Kontakt-Intensity,
// sonst Enrich aus dem Funscript. Dünne Sidecars (nur Position) werden
// übersprungen — sonst bliebe Vibration still auf 0.
// Sidecar/Enrich werden vor dem Geräte-Mapping verdichtet (Densify), damit
// Intensity der Classic-Per-Tick-Auswertung entspricht.
func (a *App) contactFrames(script *funscript.Script, mapOpts funscript.MapOptions) []funscript.Frame {
	if path := a.loadedScriptPath(); path != "" {
		if s, err := sam.LoadSidecarIfPresent(path); err == nil && s != nil && sam.HasContactIntensity(s) {
			dense := sam.Densify(s, mapOpts.TickMs, mapOpts)
			if frames := sam.ToDeviceFrames(dense, mapOpts); len(frames) > 0 {
				return frames
			}
		}
	}
	frames := sam.PlaybackFramesFromFunscript(script, mapOpts)
	if len(frames) == 0 {
		return script.ToIntensityCurve(mapOpts)
	}
	return frames
}

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
	if path != "" {
		_ = a.settings.Set(offsetKeyFor(path), ms)
	}
	logging.Info("wiedergabe: Skript-Offset gesetzt", "ms", ms, "skript", path)
}

func (a *App) GetScriptOffset() int64 {
	a.stateMu.RLock()
	defer a.stateMu.RUnlock()
	return a.scriptOffsetMs
}

func offsetKeyFor(scriptPath string) string {
	return "playback.offset." + scriptPath
}
