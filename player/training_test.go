package player

import (
	"context"
	"sync"
	"testing"
	"time"
)

// recordingDevice hält fest, welche Intensitäten gesetzt wurden - damit
// prüfbar ist, dass nach einem Stopp-Wunsch tatsächlich auf 0 gefahren wird
// und nicht nur die Buchführung stimmt.
type recordingDevice struct {
	mu         sync.Mutex
	vibrations []float64
	stops      int
}

func (d *recordingDevice) SetVibration(v float64) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.vibrations = append(d.vibrations, v)
	return nil
}

func (d *recordingDevice) SetSuction(float64) error { return nil }

func (d *recordingDevice) Stop() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.stops++
	return nil
}

func (d *recordingDevice) last() float64 {
	d.mu.Lock()
	defer d.mu.Unlock()
	if len(d.vibrations) == 0 {
		return -1
	}
	return d.vibrations[len(d.vibrations)-1]
}

func (d *recordingDevice) max() float64 {
	d.mu.Lock()
	defer d.mu.Unlock()
	m := 0.0
	for _, v := range d.vibrations {
		if v > m {
			m = v
		}
	}
	return m
}

func fastOptions(cycles int) TrainingOptions {
	return TrainingOptions{
		Technique:     TechniqueStopStart,
		Channel:       ChannelVibration,
		Cycles:        cycles,
		RampUpMs:      300,
		HoldMs:        2000,
		RestMs:        100,
		PeakIntensity: 1.0,
	}
}

// Der Kern der Stop-Start-Methode: der Anwender bricht ab, nicht die
// Stoppuhr. Vorher lief RunTraining eine feste Zeitkette ohne jede
// Eingriffsmöglichkeit - die einzige Steuerung war der Abbruch der ganzen
// Session.
func TestStopCycleInterruptsHoldAndContinues(t *testing.T) {
	dev := &recordingDevice{}
	control := NewTrainingControl()
	opts := fastOptions(2)

	var results []TrainingCycleResult
	var mu sync.Mutex
	done := make(chan error, 1)

	go func() {
		done <- RunTrainingWithControl(context.Background(), dev, opts, control,
			func(r TrainingCycleResult) {
				mu.Lock()
				results = append(results, r)
				mu.Unlock()
			})
	}()

	// Nach der Rampe (300ms) mitten in die Haltezeit (2000ms) hinein stoppen.
	time.Sleep(500 * time.Millisecond)
	started := time.Now()
	control.StopCycle()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("RunTraining: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Training ist nicht beendet worden - Stopp hat den Ablauf blockiert")
	}

	mu.Lock()
	defer mu.Unlock()
	if len(results) != 2 {
		t.Fatalf("erwartet 2 Zyklen, bekam %d", len(results))
	}
	if !results[0].StoppedByUser {
		t.Error("erster Zyklus muss als vom Nutzer gestoppt markiert sein")
	}
	if results[1].StoppedByUser {
		t.Error("zweiter Zyklus darf nicht als gestoppt gelten - ein Druck aus dem " +
			"vorherigen Zyklus darf nicht nachwirken")
	}
	// Der Stopp muss schnell wirken. Ohne unterbrechbares Warten würde erst
	// nach der vollen Haltezeit von 2000ms reagiert.
	if elapsed := results[0].EndedAt.Sub(started); elapsed > 1200*time.Millisecond {
		t.Errorf("Stopp wirkte erst nach %v - die Haltezeit wird nicht unterbrochen", elapsed)
	}
	if results[0].ReachedPeakAfterMs <= 0 {
		t.Error("ReachedPeakAfterMs muss gesetzt sein")
	}
	if dev.last() != 0 {
		t.Errorf("nach dem Training muss die Intensität 0 sein, ist %v", dev.last())
	}
}

// Ein Druck während der Hochfahrrampe muss diese sofort abbrechen, statt sie
// noch zu Ende laufen zu lassen.
func TestStopCycleInterruptsRamp(t *testing.T) {
	dev := &recordingDevice{}
	control := NewTrainingControl()
	opts := fastOptions(1)
	opts.RampUpMs = 2000

	done := make(chan error, 1)
	go func() {
		done <- RunTrainingWithControl(context.Background(), dev, opts, control, nil)
	}()

	time.Sleep(200 * time.Millisecond)
	control.StopCycle()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("RunTraining: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Rampe wurde nicht abgebrochen")
	}

	// Bei 2s Rampe und Abbruch nach 200ms darf der Höhepunkt nicht erreicht
	// worden sein.
	if m := dev.max(); m > 0.6 {
		t.Errorf("Rampe lief trotz Stopp bis %v weiter", m)
	}
}

// Ohne Stopp-Wunsch muss sich alles verhalten wie bisher.
func TestTrainingWithoutStopUnchanged(t *testing.T) {
	dev := &recordingDevice{}
	opts := fastOptions(2)
	opts.HoldMs = 50

	var results []TrainingCycleResult
	err := RunTraining(context.Background(), dev, opts, func(r TrainingCycleResult) {
		results = append(results, r)
	})
	if err != nil {
		t.Fatalf("RunTraining: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("erwartet 2 Zyklen, bekam %d", len(results))
	}
	for i, r := range results {
		if r.StoppedByUser {
			t.Errorf("Zyklus %d fälschlich als gestoppt markiert", i)
		}
	}
	if m := dev.max(); m < 0.9 {
		t.Errorf("Höhepunkt wurde nicht erreicht (max %v)", m)
	}
}

// Ein nil-Control darf nicht abstürzen - RunTraining reicht genau das durch.
func TestNilControlIsSafe(t *testing.T) {
	var control *TrainingControl
	control.StopCycle()
	if control.requested() {
		t.Error("nil-Control darf keinen Stopp melden")
	}
	control.drain()
}

// Die Rückmeldung soll den nächsten Zyklus regeln, nicht nur protokolliert
// werden. Ohne sie ist der Ablauf gesteuert, aber nicht geregelt.
func TestArousalAdjustsNextCycle(t *testing.T) {
	base := 0.8
	baseHold, baseRest := 4000, 1000

	// Am Ziel: nichts ändert sich.
	peak, hold, rest := adjustForArousal(ArousalTarget, base, baseHold, baseRest)
	if peak != base || hold != baseHold || rest != baseRest {
		t.Errorf("am Zielwert darf nichts angepasst werden: %v %d %d", peak, hold, rest)
	}

	// Hohe Erregung: kürzer, sanfter, längere Pause.
	peakHigh, holdHigh, restHigh := adjustForArousal(10, base, baseHold, baseRest)
	if holdHigh >= baseHold {
		t.Errorf("hohe Erregung muss die Haltezeit verkürzen (%d >= %d)", holdHigh, baseHold)
	}
	if restHigh <= baseRest {
		t.Errorf("hohe Erregung muss die Pause verlängern (%d <= %d)", restHigh, baseRest)
	}
	if peakHigh >= base {
		t.Errorf("hohe Erregung muss den Höhepunkt senken (%v >= %v)", peakHigh, base)
	}

	// Niedrige Erregung: länger, aber der Höhepunkt wird NICHT über die
	// Einstellung hinaus angehoben.
	peakLow, holdLow, restLow := adjustForArousal(2, base, baseHold, baseRest)
	if holdLow <= baseHold {
		t.Errorf("niedrige Erregung sollte die Haltezeit verlängern (%d <= %d)", holdLow, baseHold)
	}
	if restLow != baseRest {
		t.Errorf("Pause darf bei niedriger Erregung unverändert bleiben (%d > %d)", restLow, baseRest)
	}
	if peakLow > base {
		t.Errorf("Höhepunkt darf die Einstellung nie überschreiten (%v > %v)", peakLow, base)
	}

	// Gegensteuern nach oben muss stärker ausfallen als nach unten:
	// zu früh zu weit zu gehen ist der teurere Fehler.
	downShift := float64(baseHold-holdHigh) / float64(baseHold)
	upShift := float64(holdLow-baseHold) / float64(baseHold)
	if downShift <= upShift {
		t.Errorf("Dämpfung bei hoher Erregung (%.2f) muss stärker sein als "+
			"Verlängerung bei niedriger (%.2f)", downShift, upShift)
	}
}

// Ungültige Werte dürfen nicht als Rückmeldung durchgehen - eine
// versehentliche 0 aus einem leeren Eingabefeld wäre sonst "sehr entspannt"
// und würde die Intensität hochtreiben.
func TestArousalIgnoresInvalidValues(t *testing.T) {
	control := NewTrainingControl()
	for _, invalid := range []int{0, -3, 11, 999} {
		control.ReportArousal(invalid)
		if got := control.takeArousal(); got != 0 {
			t.Errorf("ungültiger Wert %d wurde übernommen (%d)", invalid, got)
		}
	}
	control.ReportArousal(8)
	if got := control.takeArousal(); got != 8 {
		t.Errorf("gültiger Wert ging verloren: %d", got)
	}
}

// Jede Rückmeldung wirkt auf genau einen Zyklus - sonst würde eine einmalige
// hohe Meldung den Rest der Session dämpfen.
func TestArousalAppliesToOneCycleOnly(t *testing.T) {
	control := NewTrainingControl()
	control.ReportArousal(9)
	if got := control.takeArousal(); got != 9 {
		t.Fatalf("erste Abfrage: %d", got)
	}
	if got := control.takeArousal(); got != 0 {
		t.Errorf("Rückmeldung wirkte ein zweites Mal: %d", got)
	}
}

// Ende-zu-Ende: eine hohe Meldung muss den folgenden Zyklus messbar dämpfen.
func TestArousalReachesRunningSession(t *testing.T) {
	dev := &recordingDevice{}
	control := NewTrainingControl()
	opts := fastOptions(2)
	opts.HoldMs = 600
	opts.RampUpMs = 100

	var results []TrainingCycleResult
	var mu sync.Mutex
	done := make(chan error, 1)
	go func() {
		done <- RunTrainingWithControl(context.Background(), dev, opts, control,
			func(r TrainingCycleResult) {
				mu.Lock()
				results = append(results, r)
				mu.Unlock()
			})
	}()

	control.ReportArousal(10)

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("RunTraining: %v", err)
		}
	case <-time.After(15 * time.Second):
		t.Fatal("Training nicht beendet")
	}

	mu.Lock()
	defer mu.Unlock()
	if len(results) != 2 {
		t.Fatalf("erwartet 2 Zyklen, bekam %d", len(results))
	}
	if results[0].ArousalBefore != 10 {
		t.Errorf("Rückmeldung kam nicht im ersten Zyklus an: %d", results[0].ArousalBefore)
	}
	if results[0].HoldMs >= opts.HoldMs {
		t.Errorf("Haltezeit wurde nicht gedämpft: %d >= %d", results[0].HoldMs, opts.HoldMs)
	}
	if results[1].ArousalBefore != 0 {
		t.Errorf("Rückmeldung wirkte in den zweiten Zyklus nach: %d", results[1].ArousalBefore)
	}
}
