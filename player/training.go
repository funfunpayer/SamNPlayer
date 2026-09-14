// Package player: Trainingsmodus - unabhängig von einem funscript, erzeugt
// eigene Auf/Ab-Zyklen zur Ausdauer-/Kontrollübung, angelehnt an die
// klinisch beschriebene "Stop-Start"-Methode (Semans, 1950er, ursprünglich
// zur Behandlung vorzeitiger Ejakulation entwickelt, wird heute auch
// allgemein zum Ausdauertraining genutzt) sowie die "Edging/Plateau"-
// Variante (Intensität wird nicht auf 0 gefahren, sondern knapp unter dem
// Höhepunkt gehalten). Steuert Vibration und/oder Sog, je nach Channel -
// beide Kanäle sind laut Buttplug-Rust-Quellcode (nachgeprüft, siehe
// device/samneo2.go) stufenlose Intensitäten, kein Unterschied mehr in der
// Ansteuerbarkeit.
package player

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// TrainingTechnique wählt das Grundmuster eines Zyklus.
type TrainingTechnique string

const (
	// TechniqueStopStart: hochfahren, kurz halten, komplett auf ~0 fahren,
	// Pause, von vorn. Die klassische, klinisch beschriebene Methode.
	TechniqueStopStart TrainingTechnique = "stopstart"
	// TechniquePlateau: hochfahren, dann nicht auf 0 sondern auf einem
	// Plateau knapp unter dem Höhepunkt halten ("Edging"/"Surfing"),
	// erst danach etwas absenken - kein vollständiges Abbrechen.
	TechniquePlateau TrainingTechnique = "plateau"
)

// TrainingChannel wählt, welche(r) Kanal/Kanäle die Auf/Ab-Zyklen fahren.
type TrainingChannel string

const (
	ChannelVibration TrainingChannel = "vibration"
	ChannelSuction   TrainingChannel = "suction"
	ChannelBoth      TrainingChannel = "both"
)

// TrainingOptions konfiguriert eine Trainings-Session.
type TrainingOptions struct {
	Technique TrainingTechnique
	Channel   TrainingChannel

	Cycles int // Anzahl Zyklen in dieser Session

	RampUpMs int // Zeit zum Hochfahren auf PeakIntensity
	HoldMs   int // Haltezeit auf dem Höhepunkt (bzw. Plateau)
	RestMs   int // Pause/Absenkzeit zwischen den Zyklen

	PeakIntensity   float64 // Ziel-Intensität am Höhepunkt, 0-1
	PlateauFraction float64 // nur TechniquePlateau: Plateau-Level als Anteil von PeakIntensity, 0-1 (z.B. 0.7 = 70% des Höhepunkts)

	// ProgressionPerCycle: 0-1, erhöht PeakIntensity und HoldMs über die
	// Zyklen hinweg linear um diesen Anteil bis zum letzten Zyklus (0 = alle
	// Zyklen identisch, 0.3 = letzter Zyklus 30% intensiver/länger als der erste).
	ProgressionPerCycle float64
}

// TrainingControl erlaubt Eingriffe während einer laufenden Session.
//
// Das ist der Kern der Stop-Start-Methode und fehlte bisher: RunTraining
// fuhr eine vorher festgelegte Zeitkette ab, ohne jede Rückmeldung. Bei
// Semans entscheidet aber der Anwender, wann gestoppt wird - nämlich beim
// Herannahen des Punktes ohne Wiederkehr - und nicht eine Stoppuhr. Ohne
// diesen Eingriff ist es Intervalltraining, nicht Stop-Start.
type TrainingControl struct {
	stopCycle chan struct{}

	mu      sync.Mutex
	arousal int // zuletzt gemeldeter Wert, 0 = nichts gemeldet
}

// ArousalTarget ist der Zielbereich der Skala 1-10: nahe an der Grenze, aber
// mit Abstand. Meldungen darüber führen zu kürzeren, sanfteren Zyklen mit
// längerer Pause, Meldungen darunter zu längeren.
const ArousalTarget = 7

// ReportArousal nimmt eine Rückmeldung auf der Skala 1-10 entgegen. Sie wirkt
// auf den NÄCHSTEN Zyklus, nicht auf den laufenden - in den laufenden greift
// stattdessen StopCycle ein.
//
// Ohne diese Rückmeldung ist der Ablauf gesteuert, aber nicht geregelt: die
// Zyklen laufen nach Stoppuhr ab, unabhängig davon, wie es tatsächlich
// gerade ist. Werte außerhalb 1-10 werden ignoriert statt geklemmt - eine
// versehentliche 0 aus einem leeren Eingabefeld soll nicht als "sehr
// entspannt" durchgehen und die Intensität hochtreiben.
func (c *TrainingControl) ReportArousal(level int) {
	if c == nil || level < 1 || level > 10 {
		return
	}
	c.mu.Lock()
	c.arousal = level
	c.mu.Unlock()
}

// takeArousal liest die Meldung und setzt sie zurück: jede Rückmeldung wirkt
// genau auf einen Zyklus. Sonst würde eine einmalige hohe Meldung den Rest
// der Session dämpfen, auch wenn sich die Lage längst geändert hat.
func (c *TrainingControl) takeArousal() int {
	if c == nil {
		return 0
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	level := c.arousal
	c.arousal = 0
	return level
}

// adjustForArousal passt Höhepunkt, Haltezeit und Pause an die letzte
// Rückmeldung an.
//
// Bewusst asymmetrisch: nach oben (bei hoher Erregung) wird deutlich
// stärker gegengesteuert als nach unten. Zu früh zu weit zu gehen ist der
// Fehler, der eine Session beendet; zu vorsichtig zu sein kostet nur Zeit.
func adjustForArousal(arousal int, peak float64, holdMs, restMs int) (float64, int, int) {
	if arousal < 1 {
		return peak, holdMs, restMs
	}
	deviation := float64(arousal - ArousalTarget)

	// Richtungsabhängige Koeffizienten. Ein gemeinsamer Faktor mit
	// anschließender Klemmung sah zwar asymmetrisch aus, kehrte das
	// Verhältnis an den Rändern aber um: bei Erregung 10 wurde um 45%
	// gedämpft, bei Erregung 2 um 50% verlängert - also mehr Risiko statt
	// weniger. Zu früh zu weit zu gehen ist der Fehler, der eine Session
	// beendet; zu vorsichtig zu sein kostet nur Zeit.
	holdFactor, restFactor, peakFactor := 1.0, 1.0, 1.0
	if deviation > 0 {
		holdFactor = clampRange(1.0-0.15*deviation, 0.35, 1.0)
		restFactor = clampRange(1.0+0.20*deviation, 1.0, 2.5)
		peakFactor = clampRange(1.0-0.07*deviation, 0.5, 1.0)
	} else if deviation < 0 {
		// Nach unten deutlich zurückhaltender, und die Pause wird nie
		// unter die Einstellung gekürzt.
		holdFactor = clampRange(1.0-0.07*deviation, 1.0, 1.3)
	}

	return clamp01(peak * peakFactor),
		int(float64(holdMs) * holdFactor),
		int(float64(restMs) * restFactor)
}

func clampRange(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func NewTrainingControl() *TrainingControl {
	// Gepuffert: ein Druck darf nicht verloren gehen, nur weil der Ablauf
	// gerade nicht auf den Kanal hört.
	return &TrainingControl{stopCycle: make(chan struct{}, 1)}
}

// StopCycle bricht den laufenden Zyklus sofort ab und geht in die Pause
// über. Die Session läuft danach mit dem nächsten Zyklus weiter - im
// Unterschied zum Abbruch der gesamten Session.
func (c *TrainingControl) StopCycle() {
	if c == nil {
		return
	}
	select {
	case c.stopCycle <- struct{}{}:
	default: // schon ein Druck vorgemerkt, reicht
	}
}

func (c *TrainingControl) requested() bool {
	if c == nil {
		return false
	}
	select {
	case <-c.stopCycle:
		return true
	default:
		return false
	}
}

// drain verwirft einen vorgemerkten Druck - zu Beginn jedes Zyklus, damit
// ein Druck aus dem vorherigen Zyklus nicht den nächsten sofort abbricht.
func (c *TrainingControl) drain() {
	if c == nil {
		return
	}
	select {
	case <-c.stopCycle:
	default:
	}
}

// TrainingCycleResult wird nach jedem abgeschlossenen Zyklus gemeldet -
// Grundlage für Session-Logging und Live-Anzeige im Frontend.
type TrainingCycleResult struct {
	CycleIndex    int // 0-basiert
	PeakIntensity float64
	HoldMs        int
	StartedAt     time.Time
	EndedAt       time.Time

	// StoppedByUser: der Zyklus wurde vorzeitig per StopCycle beendet.
	// Für das Sessionprotokoll die wichtigste Angabe überhaupt - wie oft
	// und wie früh gestoppt werden musste, ist das eigentliche Maß des
	// Trainingsfortschritts.
	StoppedByUser bool
	// ReachedPeakAfterMs: Zeit vom Zyklusbeginn bis zum Erreichen des
	// Höhepunkts bzw. bis zum vorzeitigen Stopp.
	ReachedPeakAfterMs int
	// RestMs: die für diesen Zyklus tatsächlich verwendete Pause (kann von
	// der Einstellung abweichen, wenn eine Rückmeldung vorlag).
	RestMs int
	// ArousalBefore: die Rückmeldung, die in diesen Zyklus eingeflossen ist
	// (0 = keine). Fürs Protokoll wichtiger als der reine Zyklusindex.
	ArousalBefore int
}

// trainingDevice ist die minimale Schnittstelle, die RunTraining braucht -
// unabhängig vom vollen device.Device-Interface, damit sich das Paket
// leicht ohne echtes Gerät testen lässt.
type trainingDevice interface {
	SetVibration(float64) error
	SetSuction(float64) error
	Stop() error
}

// RunTraining führt die konfigurierten Zyklen aus, bis sie abgeschlossen
// sind oder ctx abgebrochen wird (z.B. Nutzer klickt Stop - "man muss die
// Funktion auch ausschlagen können"). onCycle wird nach jedem Zyklus mit
// dessen Ergebnis aufgerufen (fürs Session-Log/UI), darf nil sein.
func RunTraining(ctx context.Context, dev trainingDevice, opts TrainingOptions, onCycle func(TrainingCycleResult)) error {
	return RunTrainingWithControl(ctx, dev, opts, nil, onCycle)
}

// RunTrainingWithControl ist RunTraining mit der Möglichkeit, einzelne Zyklen
// vorzeitig zu beenden (siehe TrainingControl).
func RunTrainingWithControl(ctx context.Context, dev trainingDevice, opts TrainingOptions,
	control *TrainingControl, onCycle func(TrainingCycleResult)) error {
	if opts.Cycles <= 0 {
		return fmt.Errorf("player: Training braucht mindestens 1 Zyklus")
	}
	if opts.Channel == "" {
		opts.Channel = ChannelVibration
	}
	defer dev.Stop() //nolint:errcheck // best effort beim Beenden

	for i := 0; i < opts.Cycles; i++ {
		frac := 0.0
		if opts.Cycles > 1 {
			frac = float64(i) / float64(opts.Cycles-1)
		}
		progress := 1.0 + opts.ProgressionPerCycle*frac
		peak := clamp01(opts.PeakIntensity * progress)
		holdMs := int(float64(opts.HoldMs) * progress)
		restMs := opts.RestMs

		// Rückmeldung des vorherigen Zyklus einarbeiten. Sie überschreibt
		// die Progression bewusst NICHT, sondern wirkt darauf: die geplante
		// Steigerung bleibt, wird aber gedämpft, wenn es zu viel war.
		arousal := control.takeArousal()
		peak, holdMs, restMs = adjustForArousal(arousal, peak, holdMs, restMs)

		control.drain()
		result := TrainingCycleResult{CycleIndex: i, PeakIntensity: peak, HoldMs: holdMs,
			RestMs: restMs, ArousalBefore: arousal, StartedAt: time.Now()}

		// Hochfahren
		stopped, err := rampChannel(ctx, dev, opts.Channel, 0, peak, opts.RampUpMs, control)
		if err != nil {
			return err
		}
		// Halten - hier wird in kleinen Schritten gewartet, damit ein Druck
		// auf "Stopp" sofort wirkt statt erst am Ende der Haltezeit.
		if !stopped {
			var err error
			stopped, err = waitInterruptible(ctx, time.Duration(holdMs)*time.Millisecond, control)
			if err != nil {
				return err
			}
		}
		result.ReachedPeakAfterMs = int(time.Since(result.StartedAt).Milliseconds())
		result.StoppedByUser = stopped

		if stopped {
			// Auf Wunsch abgebrochen: sofort auf 0 und volle Pause, egal
			// welche Technik eingestellt ist. Ein Plateau zu halten, nachdem
			// jemand um Unterbrechung gebeten hat, wäre das Gegenteil des
			// Gewünschten.
			if err := setChannel(dev, opts.Channel, 0); err != nil {
				return err
			}
			if err := waitOrDone(ctx, time.Duration(restMs)*time.Millisecond); err != nil {
				return err
			}
		} else {
			switch opts.Technique {
			case TechniquePlateau:
				plateau := clamp01(peak * opts.PlateauFraction)
				if _, err := rampChannel(ctx, dev, opts.Channel, peak, plateau, restMs, control); err != nil {
					return err
				}
			default: // TechniqueStopStart
				if _, err := rampChannel(ctx, dev, opts.Channel, peak, 0, restMs, control); err != nil {
					return err
				}
				if err := waitOrDone(ctx, time.Duration(restMs)*time.Millisecond); err != nil {
					return err
				}
			}
		}

		result.EndedAt = time.Now()
		if onCycle != nil {
			onCycle(result)
		}
	}
	return nil
}

// rampChannel rampt den/die gewählten Kanal/Kanäle in kleinen Schritten von
// "from" nach "to". Bei ChannelBoth bekommen Vibration und Sog denselben
// Wert - eine per-Kanal-unterschiedliche Kurve wäre auch denkbar, ist aber
// für den Trainings-Anwendungsfall (ein gemeinsames Auf/Ab-Gefühl) nicht
// nötig.
// setChannel setzt den/die gewählten Kanal/Kanäle auf einen Wert.
func setChannel(dev trainingDevice, channel TrainingChannel, v float64) error {
	if channel == ChannelVibration || channel == ChannelBoth {
		if err := dev.SetVibration(v); err != nil {
			return err
		}
	}
	if channel == ChannelSuction || channel == ChannelBoth {
		if err := dev.SetSuction(v); err != nil {
			return err
		}
	}
	return nil
}

// rampChannel rampt den/die gewählten Kanal/Kanäle in kleinen Schritten von
// "from" nach "to". Bei ChannelBoth bekommen Vibration und Sog denselben
// Wert - eine per-Kanal-unterschiedliche Kurve wäre auch denkbar, ist aber
// für den Trainings-Anwendungsfall (ein gemeinsames Auf/Ab-Gefühl) nicht
// nötig.
//
// Gibt zurück, ob der Nutzer währenddessen um Unterbrechung gebeten hat.
// Die Rampe bricht dann sofort ab, statt erst zu Ende zu laufen.
//
// Die Schrittweite ist zeitbasiert statt fest: eine feste Anzahl Schritte
// bedeutete bei langen Rampen große zeitliche Lücken (bei 8 Sekunden und 20
// Schritten alle 400ms eine Änderung), und in genau diesen Lücken hätte ein
// Druck auf "Stopp" nicht gewirkt.
func rampChannel(ctx context.Context, dev trainingDevice, channel TrainingChannel,
	from, to float64, durationMs int, control *TrainingControl) (bool, error) {
	if durationMs <= 0 {
		return false, setChannel(dev, channel, to)
	}

	const stepMs = 50
	steps := durationMs / stepMs
	if steps < 1 {
		steps = 1
	}
	stepDur := time.Duration(durationMs) * time.Millisecond / time.Duration(steps)

	for i := 1; i <= steps; i++ {
		if control.requested() {
			return true, nil
		}
		v := from + (to-from)*float64(i)/float64(steps)
		if err := setChannel(dev, channel, v); err != nil {
			return false, err
		}
		if err := waitOrDone(ctx, stepDur); err != nil {
			return false, err
		}
	}
	return control.requested(), nil
}

// waitInterruptible wartet, prüft dabei aber regelmäßig auf einen
// Stopp-Wunsch. Ein einfaches time.After würde den Druck erst nach Ablauf
// der gesamten Haltezeit bemerken - bei mehreren Sekunden Haltezeit wäre
// der Knopf damit praktisch wirkungslos.
//
// Gibt den ctx-Fehler zurück statt ihn zu verschlucken: der Aufrufer hat
// sonst keine Möglichkeit zu erkennen, dass die Wartezeit wegen eines
// Context-Abbruchs (nicht wegen StopCycle) vorzeitig endete, und würde in
// den normalen Rampe-Pfad weiterlaufen - der dort noch mindestens einen
// weiteren Gerätebefehl absetzt, bevor der Fehler eine Ebene später doch
// noch bemerkt wird (gefunden über TestContextCancelDuringHoldStopsImmediately).
func waitInterruptible(ctx context.Context, d time.Duration, control *TrainingControl) (bool, error) {
	const tick = 50 * time.Millisecond
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if control.requested() {
			return true, nil
		}
		remaining := time.Until(deadline)
		if remaining > tick {
			remaining = tick
		}
		if err := waitOrDone(ctx, remaining); err != nil {
			return false, err
		}
	}
	return control.requested(), nil
}

func waitOrDone(ctx context.Context, d time.Duration) error {
	select {
	case <-time.After(d):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
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
