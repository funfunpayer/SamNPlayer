package generator

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/funfunpayer/SamNPlayer/generator/bodyparts"
	"github.com/funfunpayer/SamNPlayer/logging"
)

// RoiTrainingAvailable prüft, ob das Trainieren eines eigenen Modells
// grundsätzlich möglich ist - ultralytics installiert (siehe
// requirements-ai-train.txt). Bootstrap (Datensammeln) braucht OpenCV und
// wird separat geprüft; hier nur das eigentliche Trainieren, damit ein
// fehlendes OpenCV den Train-Knopf nicht fälschlich sperrt, wenn ultralytics
// bereits da ist (und umgekehrt eine klare Meldung möglich bleibt).
func RoiTrainingAvailable() bool {
	st := GetRoiTrainingStatus()
	return st.Ultralytics
}

// RoiTrainingStatus beschreibt die Voraussetzungen für Bootstrap und Training.
type RoiTrainingStatus struct {
	Python      bool   `json:"python"`
	OpenCV      bool   `json:"opencv"`
	Ultralytics bool   `json:"ultralytics"`
	Detail      string `json:"detail"`
}

// GetRoiTrainingStatus prüft Python, OpenCV-Tracker und ultralytics getrennt.
func GetRoiTrainingStatus() RoiTrainingStatus {
	st := RoiTrainingStatus{}
	py, err := FindPython()
	if err != nil {
		st.Detail = "Kein Python gefunden"
		return st
	}
	st.Python = true
	if err := CheckDependencies(); err == nil {
		st.OpenCV = true
	}
	mainScript, err := writeScriptToTemp()
	if err != nil {
		st.Detail = "Eingebettete Skripte nicht schreibbar"
		return st
	}
	defer cleanupScriptTemp(mainScript)
	scriptPath := filepath.Join(filepath.Dir(mainScript), "train_yolo_model.py")
	out, err := command(py, scriptPath, "--check").Output()
	if err == nil && strings.TrimSpace(string(out)) == "AVAILABLE" {
		st.Ultralytics = true
	}
	switch {
	case st.Ultralytics && st.OpenCV:
		st.Detail = "Ready: bootstrap and training available"
	case st.Ultralytics && !st.OpenCV:
		st.Detail = "Training OK; bootstrap needs opencv-contrib-python (Install AI train deps restores CSRT after ultralytics)"
	case !st.Ultralytics && st.OpenCV:
		st.Detail = "Bootstrap OK; training: Install AI train deps (or pip install ultralytics onnx)"
	default:
		st.Detail = "Neither ultralytics nor OpenCV trackers found — use Install AI train deps"
	}
	return st
}

// InstallRoiTrainingDeps installs ultralytics+onnx via pip for the detected
// Python, then restores opencv-contrib-python. ultralytics depends on
// opencv-python, which removes CSRT trackers needed for video bootstrap
// (“Use for training”) — Issues #94/#95/#119.
func InstallRoiTrainingDeps(onProgress func(line string)) error {
	py, err := FindPython()
	if err != nil {
		return err
	}
	req, err := pythonFiles.ReadFile("requirements-ai-train.txt")
	if err != nil {
		return fmt.Errorf("generator: requirements-ai-train.txt fehlt im Binary: %w", err)
	}
	tmp, err := os.CreateTemp("", "samn-ai-train-req-*.txt")
	if err != nil {
		return err
	}
	reqPath := tmp.Name()
	defer os.Remove(reqPath)
	if _, err := tmp.Write(req); err != nil {
		tmp.Close()
		return err
	}
	tmp.Close()
	if onProgress != nil {
		onProgress("pip install -r requirements-ai-train.txt …")
	}
	cmd := command(py, "-m", "pip", "install", "-r", reqPath)
	out, err := cmd.CombinedOutput()
	emitPipLines(string(out), onProgress)
	if err != nil {
		return fmt.Errorf("generator: pip install fehlgeschlagen: %w", err)
	}
	if onProgress != nil {
		onProgress("Restoring opencv-contrib-python (CSRT for video bootstrap)…")
	}
	// Best-effort uninstall of the non-contrib wheels ultralytics may have pulled.
	un := command(py, "-m", "pip", "uninstall", "-y", "opencv-python", "opencv-python-headless")
	unOut, _ := un.CombinedOutput()
	emitPipLines(string(unOut), onProgress)
	// Both OpenCV distributions own cv2 files. Uninstalling the plain wheel
	// can remove those files while contrib's installed-version metadata stays
	// intact. --upgrade alone then reports "already satisfied" and cannot
	// repair cv2. Reinstall only contrib; do not force-reinstall the full
	// numerical stack (or the much larger training dependencies).
	restore := command(py, "-m", "pip", "install", "--upgrade", "--force-reinstall", "--no-deps", "opencv-contrib-python>=4.8")
	restoreOut, restoreErr := restore.CombinedOutput()
	emitPipLines(string(restoreOut), onProgress)
	if restoreErr != nil {
		return fmt.Errorf("generator: opencv-contrib-python restore failed: %w", restoreErr)
	}
	fix := command(py, "-m", "pip", "install", "--upgrade", "opencv-contrib-python>=4.8", "scipy>=1.10", "numpy>=1.24")
	fixOut, fixErr := fix.CombinedOutput()
	emitPipLines(string(fixOut), onProgress)
	if fixErr != nil {
		return fmt.Errorf("generator: opencv-contrib-python restore failed: %w", fixErr)
	}
	if err := CheckDependencies(); err != nil {
		return fmt.Errorf("generator: OpenCV still missing CSRT after restore — %w", err)
	}
	if !RoiTrainingAvailable() {
		return fmt.Errorf("generator: ultralytics still unavailable after pip install")
	}
	return nil
}

func emitPipLines(out string, onProgress func(line string)) {
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		logging.Info("roitraining: pip", "zeile", line)
		if onProgress != nil {
			onProgress(line)
		}
	}
}

// WriteAIRequirementFiles writes embedded AI requirement lists into dir
// (for users who prefer manual pip). Creates dir if needed.
func WriteAIRequirementFiles(dir string) error {
	if strings.TrimSpace(dir) == "" {
		return fmt.Errorf("generator: Zielordner leer")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	for _, name := range []string{"requirements-ai.txt", "requirements-ai-train.txt", "requirements.txt"} {
		data, err := pythonFiles.ReadFile(name)
		if err != nil {
			return fmt.Errorf("generator: %s: %w", name, err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), data, 0o644); err != nil {
			return err
		}
	}
	return nil
}

// RoiTrainingDevice beschreibt eine wählbare Trainingsbeschleunigung
// (CUDA / DirectML / MPS / CPU) - siehe train_yolo_model.list_devices.
type RoiTrainingDevice struct {
	ID        string `json:"id"`
	Label     string `json:"label"`
	Available bool   `json:"available"`
}

// ListRoiTrainingDevices fragt train_yolo_model.py --list-devices ab.
// Bei Fehlern (kein Python / ultralytics egal - list_devices braucht kein
// ultralytics) kommt eine statische Fallback-Liste, damit die GUI trotzdem
// Auto/CUDA/DirectML/CPU anbieten kann; die echte Verfügbarkeit prüft dann
// resolve_device beim Start des Trainings.
func ListRoiTrainingDevices() []RoiTrainingDevice {
	fallback := []RoiTrainingDevice{
		{ID: "auto", Label: "Automatisch (bestes verfügbares)", Available: true},
		{ID: "cuda", Label: "NVIDIA CUDA", Available: false},
		{ID: "directml", Label: "DirectML (Windows, AMD/Intel/NVIDIA)", Available: false},
		{ID: "mps", Label: "Apple MPS", Available: false},
		{ID: "cpu", Label: "CPU (sehr langsam)", Available: true},
	}
	py, err := FindPython()
	if err != nil {
		return fallback
	}
	mainScript, err := writeScriptToTemp()
	if err != nil {
		return fallback
	}
	defer cleanupScriptTemp(mainScript)
	scriptPath := filepath.Join(filepath.Dir(mainScript), "train_yolo_model.py")
	out, err := command(py, scriptPath, "--list-devices").Output()
	if err != nil {
		return fallback
	}
	var devices []RoiTrainingDevice
	if err := json.Unmarshal(out, &devices); err != nil || len(devices) == 0 {
		return fallback
	}
	return devices
}

// RoiTrainingRegion is a marked box + class for the YOLO bootstrap dataset
// (bootstrap_yolo_dataset.py). ROI1 is required; ROI2–ROI9 optional (up to
// bodyparts.MaxRegionsPerImage classes on one frame).
type RoiTrainingRegion struct {
	ROI       ROI    `json:"ROI"`
	ClassName string `json:"ClassName"`
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
	return BootstrapRoiTrainingSampleOpts(videoPath, regions, outputDir, samplePrefix, 12, true, 0, 1.0, onProgress, nil)
}

// BootstrapRoiTrainingSampleOpts is BootstrapRoiTrainingSample with sampling
// stride, optional audio extraction, startSeconds seek, and boxScale pad.
// onPercent receives 0–100 (or -1) from Python PROGRESS lines during tracking.
func BootstrapRoiTrainingSampleOpts(videoPath string, regions []RoiTrainingRegion, outputDir, samplePrefix string,
	sampleEvery int, extractAudio bool, startSeconds, boxScale float64, onProgress func(line string), onPercent func(pct int)) error {
	if len(regions) == 0 {
		return fmt.Errorf("generator: at least one region required")
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
	if boxScale <= 0 {
		boxScale = 1.0
	}

	if err := runPythonScript(py, buildBootstrapArgsOpts(scriptPath, videoPath, regions, outputDir, samplePrefix, sampleEvery, startSeconds, boxScale),
		"roi_training", onProgress, onPercent); err != nil {
		return err
	}
	if extractAudio {
		path, err := ExtractTrainingAudio(videoPath, outputDir, samplePrefix, startSeconds)
		if err != nil {
			if onProgress != nil {
				onProgress("Hinweis: Audio nicht extrahiert — " + err.Error())
			}
		} else if onProgress != nil {
			onProgress("Audio gespeichert: " + path)
		}
	}
	return nil
}

// RunRoiModelTraining trainiert ein YOLO-Modell auf dem gesammelten
// Datensatz und exportiert es nach outputModelPath (siehe
// train_yolo_model.py) - kann bei echten Datensätzen lange laufen (Minuten
// bis Stunden, abhängig von epochs/Datensatzgröße/GPU), darum ohne eigenes
// Timeout hier; die GUI ruft das asynchron auf (siehe app_roi_training.go).
func RunRoiModelTraining(datasetDir, outputModelPath string, epochs int, device string,
	onProgress func(line string)) error {
	return RunRoiModelTrainingWithProgress(datasetDir, outputModelPath, epochs, device, onProgress, nil)
}

// RunRoiModelTrainingWithProgress is RunRoiModelTraining with a percent callback
// (ultralytics rarely emits PROGRESS lines; kept for API symmetry / future hooks).
func RunRoiModelTrainingWithProgress(datasetDir, outputModelPath string, epochs int, device string,
	onProgress func(line string), onPercent func(pct int)) error {
	dataYAML := filepath.Join(datasetDir, "data.yaml")
	if _, err := os.Stat(dataYAML); err != nil {
		return fmt.Errorf("no training dataset yet (missing %s). In AI Train: mark region(s), click “Use for training”, then start training", dataYAML)
	}
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
		"roi_training", onProgress, onPercent)
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
	return buildBootstrapArgsOpts(scriptPath, videoPath, regions, outputDir, samplePrefix, 12, 0, 1.0)
}

func buildBootstrapArgsOpts(scriptPath, videoPath string, regions []RoiTrainingRegion, outputDir, samplePrefix string, sampleEvery int, startSeconds, boxScale float64) []string {
	if len(regions) > bodyparts.MaxRegionsPerImage {
		regions = regions[:bodyparts.MaxRegionsPerImage]
	}
	for i := range regions {
		regions[i].ClassName = bodyparts.Normalize(regions[i].ClassName)
		if regions[i].ClassName == "" {
			regions[i].ClassName = "motion_region"
		}
	}
	args := []string{scriptPath,
		"--video", videoPath,
		"--roi", roiArg(regions[0].ROI),
		"--class-name", regions[0].ClassName,
		"--output-dir", outputDir,
	}
	if samplePrefix != "" {
		args = append(args, "--sample-prefix", samplePrefix)
	}
	if sampleEvery > 0 && sampleEvery != 12 {
		args = append(args, "--sample-every", itoa(sampleEvery))
	}
	if startSeconds > 0 {
		args = append(args, "--start-seconds", fmt.Sprintf("%.3f", startSeconds))
	}
	if boxScale > 0 && (boxScale < 0.999 || boxScale > 1.001) {
		args = append(args, "--box-scale", fmt.Sprintf("%.3f", boxScale))
	}
	for i := 1; i < len(regions) && i < 9; i++ {
		n := i + 1
		args = append(args,
			"--roi"+itoa(n), roiArg(regions[i].ROI),
			"--class-name"+itoa(n), regions[i].ClassName,
		)
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
