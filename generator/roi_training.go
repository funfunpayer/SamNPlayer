package generator

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/funfunpayer/SamNPlayer/logging"
)

// RoiTrainingAvailable prüft, ob das Trainieren eines eigenen Modells
// grundsätzlich möglich ist - ultralytics installiert (siehe
// requirements-ai-train.txt, bewusst NICHT Teil der Basis-Installation, da
// es u.a. PyTorch nachzieht). Für die GUI, um den "Training starten"-Knopf
// zu aktivieren/auszublenden, statt ihn anzubieten und dann mit einem
// rohen ModuleNotFoundError-Traceback scheitern zu lassen - dasselbe Muster
// wie AIRoiAvailable für die kleinere onnxruntime-Abhängigkeit. Bootstrap
// (Datensammeln, braucht nur die Basis-Installation) ist davon nicht
// betroffen, nur das eigentliche Trainieren.
func RoiTrainingAvailable() bool {
	py, err := FindPython()
	if err != nil {
		return false
	}
	if err := CheckDependencies(); err != nil {
		return false
	}
	mainScript, err := writeScriptToTemp()
	if err != nil {
		return false
	}
	defer cleanupScriptTemp(mainScript)
	scriptPath := filepath.Join(filepath.Dir(mainScript), "train_yolo_model.py")
	out, err := command(py, scriptPath, "--check").Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) == "AVAILABLE"
}

// RoiTrainingRegion ist eine markierte Region samt Klassenname für den
// Bootstrap-Datensatz (siehe bootstrap_yolo_dataset.py) - ROI1 ist immer
// nötig, ROI2 optional (z.B. Eichel + Brustwarze für Tf/Tj, beide als
// eigene Klasse im selben Bild, siehe dessen Moduldoc).
type RoiTrainingRegion struct {
	ROI       ROI
	ClassName string
}

// BootstrapRoiTrainingSample trackt eine oder zwei Regionen durchs Video und
// hängt die Ergebnisse an den Datensatz unter outputDir an (mehrere Aufrufe
// bauen denselben Datensatz über viele Clips auf - siehe
// bootstrap_yolo_dataset.py). Liefert KEINE fertigen Trainingsbeispiele im
// Sinne von "sofort vertrauenswürdig" - das Tracking dahinter kann auf
// echtem Material danebenliegen (gemessen, docs/NEXT.md, 16. September
// 2026); die Kontrollansicht (ListRoiTrainingSamples) ist deshalb kein
// optionaler Schritt, sondern Teil des vorgesehenen Ablaufs.
func BootstrapRoiTrainingSample(videoPath string, regions []RoiTrainingRegion, outputDir, samplePrefix string,
	onProgress func(line string)) error {
	if len(regions) == 0 {
		return fmt.Errorf("generator: mindestens eine Region nötig")
	}
	py, err := FindPython()
	if err != nil {
		return err
	}
	if err := CheckDependencies(); err != nil {
		return err
	}
	mainScript, err := writeScriptToTemp()
	if err != nil {
		return err
	}
	defer cleanupScriptTemp(mainScript)
	scriptPath := filepath.Join(filepath.Dir(mainScript), "bootstrap_yolo_dataset.py")

	return runPythonScript(py, buildBootstrapArgs(scriptPath, videoPath, regions, outputDir, samplePrefix),
		"roi_training", onProgress, nil)
}

// RunRoiModelTraining trainiert ein YOLO-Modell auf dem gesammelten
// Datensatz und exportiert es nach outputModelPath (siehe
// train_yolo_model.py) - kann bei echten Datensätzen lange laufen (Minuten
// bis Stunden, abhängig von epochs/Datensatzgröße/GPU), darum ohne eigenes
// Timeout hier; die GUI ruft das asynchron auf (siehe app_roi_training.go).
func RunRoiModelTraining(datasetDir, outputModelPath string, epochs int, device string,
	onProgress func(line string)) error {
	py, err := FindPython()
	if err != nil {
		return err
	}
	mainScript, err := writeScriptToTemp()
	if err != nil {
		return err
	}
	defer cleanupScriptTemp(mainScript)
	scriptPath := filepath.Join(filepath.Dir(mainScript), "train_yolo_model.py")

	return runPythonScript(py, buildTrainArgs(scriptPath, datasetDir, outputModelPath, epochs, device),
		"roi_training", onProgress, nil)
}

// runPythonScript führt ein Python-Skript aus, das seinen Fortschritt/Log
// nur über stderr ausgibt (kein strukturiertes stdout wie findROIViaScript) -
// gemeinsamer Kern für Bootstrap und Training, beide brauchen nur
// Log-Weiterleitung und Erfolg/Fehler, kein geparstes Ergebnis auf stdout.
func runPythonScript(py string, args []string, logPrefix string,
	onProgress func(line string), onPercent func(pct int)) error {
	cmd := command(py, args...)
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("generator: stderr-Pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("generator: Start fehlgeschlagen: %w", err)
	}
	scanner := bufio.NewScanner(stderr)
	// ultralytics' eigene Fortschrittszeilen können sehr lang werden
	// (Tabellen-Zeilen mit vielen Spalten) - der Standardpuffer (64KB)
	// reicht normalerweise, aber sicherheitshalber wie beim Benchmark-Report
	// großzügig bemessen statt mitten im Training an einer zu langen Zeile
	// zu scheitern.
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	var lastLines []string
	for scanner.Scan() {
		line := scanner.Text()
		if done, total, ok := parseProgress(line); ok {
			if onPercent != nil {
				onPercent(percentOf(done, total))
			}
			continue
		}
		lastLines = append(lastLines, line)
		if len(lastLines) > 20 {
			lastLines = lastLines[1:]
		}
		logging.Debug(logPrefix + ": " + line)
		if onProgress != nil {
			onProgress(line)
		}
	}
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("generator: %s fehlgeschlagen: %w\nLetzte Ausgabe:\n%s",
			logPrefix, err, joinLines(lastLines))
	}
	return nil
}

// buildBootstrapArgs und buildTrainArgs sind bewusst von der eigentlichen
// Prozessausführung getrennt (wie generator.go's buildArgs/BuildArgsForTest)
// - reine Funktionen, testbar ohne Python/Subprozess. Eine Option, die hier
// nicht ankommt, ist schlimmer als keine (vgl. args_test.go).
func buildBootstrapArgs(scriptPath, videoPath string, regions []RoiTrainingRegion, outputDir, samplePrefix string) []string {
	args := []string{scriptPath,
		"--video", videoPath,
		"--roi", roiArg(regions[0].ROI),
		"--class-name", regions[0].ClassName,
		"--output-dir", outputDir,
	}
	if samplePrefix != "" {
		args = append(args, "--sample-prefix", samplePrefix)
	}
	if len(regions) > 1 {
		args = append(args, "--roi2", roiArg(regions[1].ROI), "--class-name2", regions[1].ClassName)
	}
	return args
}

func buildTrainArgs(scriptPath, datasetDir, outputModelPath string, epochs int, device string) []string {
	args := []string{scriptPath,
		"--dataset-dir", datasetDir,
		"--output", outputModelPath,
	}
	if epochs > 0 {
		args = append(args, "--epochs", itoa(epochs))
	}
	if device != "" {
		args = append(args, "--device", device)
	}
	return args
}

func roiArg(r ROI) string {
	return fmt.Sprintf("%d,%d,%d,%d", r.X, r.Y, r.W, r.H)
}

func itoa(n int) string {
	return fmt.Sprintf("%d", n)
}

// DefaultRoiDatasetDir schlägt einen Ort neben den übrigen Nutzerdaten vor -
// analog zu ai_roi.default_model_path() (Python-Seite).
func DefaultRoiDatasetDir() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "SamNPlayer", "roi_training_dataset")
}
