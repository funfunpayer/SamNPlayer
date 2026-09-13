package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/funfunpayer/SamNPlayer/generator"
	"github.com/funfunpayer/SamNPlayer/logging"
)

// Hintergrund: sämtliche Schwellen im Quality Doctor sind an synthetischen
// Testvideos bestimmt worden - saubere Sinusbewegungen, deren Wahrheit per
// Konstruktion bekannt war. Echtes Material ist unregelmäßiger und liegt
// systematisch niedriger. Zum Nachziehen der Schwellen braucht es beides:
// die Kennzahlen echter Läufe UND ein menschliches Urteil dazu. Ohne das
// Urteil sind die Zahlen wertlos, weil niemand weiß, welche davon zu einem
// brauchbaren Ergebnis gehören.

// ReportFeedback ist ein Urteil zu einem erzeugten Skript.
type ReportFeedback struct {
	OutputPath string `json:"outputPath"`
	Verdict    string `json:"verdict"`
	Comment    string `json:"comment"`
}

// activeReportPath liefert den eingestellten Berichtspfad, oder "" wenn
// keiner gesetzt ist (dann wird kein Bericht geschrieben).
func (a *App) activeReportPath() string {
	return a.settings.GetString(prefReportPath, "")
}

// SubmitFeedback trägt ein Urteil zum zuletzt erzeugten Skript nach.
func (a *App) SubmitFeedback(fb ReportFeedback) error {
	path := a.activeReportPath()
	if path == "" {
		return fmt.Errorf("kein Berichtspfad eingestellt - im Einstellungen-Tab festlegen, " +
			"damit Messwerte und Urteile gespeichert werden können")
	}
	if err := generator.AddFeedback(path, fb.OutputPath, fb.Verdict, fb.Comment); err != nil {
		logging.Warn("bericht: Urteil konnte nicht gespeichert werden", "fehler", err)
		return err
	}
	logging.Info("bericht: Urteil gespeichert", "urteil", fb.Verdict,
		"skript", fb.OutputPath, "kommentar", fb.Comment)
	return nil
}

// ReportSummary gibt die nach Urteil gruppierte Auswertung zurück.
func (a *App) ReportSummary() (string, error) {
	path := a.activeReportPath()
	if path == "" {
		return "", fmt.Errorf("kein Berichtspfad eingestellt")
	}
	return generator.ReportSummary(path)
}

// ReportExists sagt, ob schon Messwerte vorliegen - damit die Oberfläche die
// Auswertung nur anbietet, wenn es etwas auszuwerten gibt.
func (a *App) ReportExists() bool {
	path := a.activeReportPath()
	if path == "" {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && info.Size() > 0
}

// PickReportPath lässt den Nutzer eine Zieldatei wählen.
func (a *App) PickReportPath() (string, error) {
	current := a.activeReportPath()
	if current == "" {
		current = defaultReportPath()
	}
	return runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:                "Messwerte speichern unter",
		DefaultFilename:      filepath.Base(current),
		DefaultDirectory:     filepath.Dir(current),
		CanCreateDirectories: true,
		Filters: []runtime.FileFilter{
			{DisplayName: "Messwerte (*.jsonl)", Pattern: "*.jsonl"},
		},
	})
}

// TrainQualityModel lernt aus den gesammelten Urteilen.
func (a *App) TrainQualityModel() (string, error) {
	path := a.activeReportPath()
	if path == "" {
		return "", fmt.Errorf("kein Berichtspfad eingestellt - ohne aufgezeichnete " +
			"Messwerte und Urteile gibt es nichts zu lernen")
	}
	report, err := generator.TrainQualityModel(path)
	if err != nil {
		logging.Warn("modell: Lernen fehlgeschlagen", "fehler", err)
		return report, err
	}
	logging.Info("modell: Lernvorgang abgeschlossen")
	return report, nil
}

// QualityModelInfo beschreibt das aktuell verwendete Qualitätsmodell.
func (a *App) QualityModelInfo() (string, error) {
	return generator.QualityModelInfo()
}
