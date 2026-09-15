package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/funfunpayer/SamNPlayer/device"
	"github.com/funfunpayer/SamNPlayer/logging"
	"github.com/funfunpayer/SamNPlayer/player"
)

// TrainingRequest kommt als JSON vom Frontend-Formular.
type TrainingRequest struct {
	Mock                bool    `json:"mock"`
	Technique           string  `json:"technique"` // "stopstart" | "plateau"
	Channel             string  `json:"channel"`   // "vibration" | "suction" | "both"
	Cycles              int     `json:"cycles"`
	RampUpMs            int     `json:"rampUpMs"`
	HoldMs              int     `json:"holdMs"`
	RestMs              int     `json:"restMs"`
	PeakIntensity       float64 `json:"peakIntensity"`
	PlateauFraction     float64 `json:"plateauFraction"`
	ProgressionPerCycle float64 `json:"progressionPerCycle"`
}

// sessionLogEntry ist eine Zeile im Trainings-Sitzungsprotokoll (JSONL -
// ein JSON-Objekt pro Zeile, leicht später auszuwerten/zu importieren).
// Das ist die "alle Daten mitschreiben"-Grundlage, um die Ansteuerung des
// Sam Neo 2 später anhand echter Sitzungsverläufe weiter feinabstimmen zu
// können.
type sessionLogEntry struct {
	Timestamp     string  `json:"timestamp"`
	Technique     string  `json:"technique"`
	Channel       string  `json:"channel"`
	CycleIndex    int     `json:"cycleIndex"`
	CyclesTotal   int     `json:"cyclesTotal"`
	PeakIntensity float64 `json:"peakIntensity"`
	HoldMs        int     `json:"holdMs"`
	DurationMs    int64   `json:"durationMs"`

	// StoppedByUser und ReachedPeakAfterMs sind die eigentlich
	// aussagekräftigen Werte für den Trainingsfortschritt: wie oft und wie
	// früh musste unterbrochen werden. Ein Protokoll ohne diese Angaben
	// zeigt nur, dass die Zyklen abgelaufen sind - nicht, wie sie verliefen.
	StoppedByUser      bool `json:"stoppedByUser"`
	ReachedPeakAfterMs int  `json:"reachedPeakAfterMs"`

	// ArousalBefore und RestMs machen im Protokoll nachvollziehbar, WARUM
	// ein Zyklus so verlief - ohne sie sähe eine gedämpfte Haltezeit wie
	// eine falsche Einstellung aus statt wie eine Reaktion.
	ArousalBefore int `json:"arousalBefore"`
	RestMs        int `json:"restMs"`
}

// StartTraining startet eine Trainings-Session in einer eigenen Goroutine.
// Fortschritt kommt über Events ("training:cycle", "training:done",
// "training:error") - derselbe Aufbau wie StartPlayback.
func (a *App) StartTraining(req TrainingRequest) error {
	if req.Cycles <= 0 {
		return fmt.Errorf("mindestens 1 Zyklus nötig")
	}

	var dev device.Device
	if req.Mock {
		dev = device.NewMock(false)
	} else {
		dev = device.NewSamNeo2(device.SamNeo2Protocol{})
	}
	// Unter stateMu wie in StartPlayback (siehe dessen Kommentar) - Training
	// hat kein Gegenstück zu TriggerExtendedO, das a.activeDevice läse, aber
	// shutdown() liest es (app.go) und muss denselben Schutz sehen wie jeden
	// anderen Schreibzugriff auf dieses Feld.
	a.stateMu.Lock()
	a.activeDevice = dev
	a.stateMu.Unlock()

	sessionFile, sessionErr := a.openSessionLog(req.Technique)
	if sessionErr != nil {
		logging.Warn("app: Session-Log konnte nicht angelegt werden", "fehler", sessionErr)
	}

	opts := player.TrainingOptions{
		Technique:           player.TrainingTechnique(req.Technique),
		Channel:             player.TrainingChannel(req.Channel),
		Cycles:              req.Cycles,
		RampUpMs:            req.RampUpMs,
		HoldMs:              req.HoldMs,
		RestMs:              req.RestMs,
		PeakIntensity:       req.PeakIntensity,
		PlateauFraction:     req.PlateauFraction,
		ProgressionPerCycle: req.ProgressionPerCycle,
	}

	ctx, err := a.tryStartSession()
	if err != nil {
		return err
	}

	go func() {
		defer a.endSession()
		if sessionFile != nil {
			defer sessionFile.Close()
		}

		connectCtx, connectCancel := context.WithTimeout(ctx, 20*time.Second)
		defer connectCancel()
		runtime.EventsEmit(a.ctx, "training:log", "Verbinde...")
		if err := dev.Connect(connectCtx); err != nil {
			logging.Error("app: Verbindung fehlgeschlagen (Training)", "fehler", err)
			runtime.EventsEmit(a.ctx, "training:error", err.Error())
			runtime.EventsEmit(a.ctx, "training:done")
			return
		}
		defer dev.Disconnect()

		runtime.EventsEmit(a.ctx, "training:log", "Training startet...")

		control := player.NewTrainingControl()
		a.stateMu.Lock()
		a.trainingControl = control
		a.stateMu.Unlock()
		defer func() {
			a.stateMu.Lock()
			a.trainingControl = nil
			a.stateMu.Unlock()
		}()

		err := player.RunTrainingWithControl(ctx, dev, opts, control, func(result player.TrainingCycleResult) {
			runtime.EventsEmit(a.ctx, "training:cycle", map[string]any{
				"cycleIndex":         result.CycleIndex,
				"cyclesTotal":        req.Cycles,
				"peakIntensity":      result.PeakIntensity,
				"holdMs":             result.HoldMs,
				"stoppedByUser":      result.StoppedByUser,
				"reachedPeakAfterMs": result.ReachedPeakAfterMs,
				"arousalBefore":      result.ArousalBefore,
				"restMs":             result.RestMs,
			})
			if sessionFile != nil {
				entry := sessionLogEntry{
					Timestamp:     result.EndedAt.Format(time.RFC3339),
					Technique:     req.Technique,
					Channel:       req.Channel,
					CycleIndex:    result.CycleIndex,
					CyclesTotal:   req.Cycles,
					PeakIntensity: result.PeakIntensity,
					HoldMs:        result.HoldMs,
					DurationMs:    result.EndedAt.Sub(result.StartedAt).Milliseconds(),

					StoppedByUser:      result.StoppedByUser,
					ReachedPeakAfterMs: result.ReachedPeakAfterMs,
					ArousalBefore:      result.ArousalBefore,
					RestMs:             result.RestMs,
				}
				if b, err := json.Marshal(entry); err == nil {
					if _, writeErr := sessionFile.Write(append(b, '\n')); writeErr != nil {
						logging.Warn("app: Trainings-Zyklus konnte nicht ins Session-Log geschrieben werden", "fehler", writeErr)
					}
				}
			}
		})

		if err != nil && err != context.Canceled {
			runtime.EventsEmit(a.ctx, "training:error", err.Error())
		}
		runtime.EventsEmit(a.ctx, "training:done")
	}()

	return nil
}

// StopTrainingCycle bricht den laufenden Zyklus ab und geht in die Pause -
// die Session läuft danach weiter. Das ist der eigentliche Kern der
// Stop-Start-Methode: nicht eine Stoppuhr entscheidet, wann unterbrochen
// wird, sondern der Anwender.
func (a *App) StopTrainingCycle() error {
	a.stateMu.RLock()
	control := a.trainingControl
	a.stateMu.RUnlock()
	if control == nil {
		return fmt.Errorf("es läuft gerade kein Training")
	}
	control.StopCycle()
	logging.Info("training: Zyklus auf Wunsch unterbrochen")
	return nil
}

// ReportArousal nimmt eine Rückmeldung auf der Skala 1-10 entgegen. Sie
// wirkt auf den nächsten Zyklus: hohe Werte führen zu kürzeren, sanfteren
// Zyklen mit längerer Pause. Damit wird aus dem gesteuerten Ablauf ein
// geregelter.
func (a *App) ReportArousal(level int) error {
	a.stateMu.RLock()
	control := a.trainingControl
	a.stateMu.RUnlock()
	if control == nil {
		return fmt.Errorf("es läuft gerade kein Training")
	}
	if level < 1 || level > 10 {
		return fmt.Errorf("Wert muss zwischen 1 und 10 liegen (ist %d)", level)
	}
	control.ReportArousal(level)
	logging.Info("training: Rückmeldung", "erregung", level)
	return nil
}

func (a *App) StopTraining() {
	a.stopSession()
}

// openSessionLog legt (falls nötig) den Sessions-Unterordner neben der
// normalen Logdatei an und öffnet eine neue JSONL-Datei für diese Session.
func (a *App) openSessionLog(technique string) (*os.File, error) {
	dir, err := logging.Dir()
	if err != nil {
		return nil, err
	}
	sessionsDir := filepath.Join(dir, "sessions")
	if err := os.MkdirAll(sessionsDir, 0755); err != nil {
		return nil, err
	}
	name := fmt.Sprintf("training-%s-%s.jsonl", technique, time.Now().Format("20060102-150405"))
	return os.Create(filepath.Join(sessionsDir, name))
}
