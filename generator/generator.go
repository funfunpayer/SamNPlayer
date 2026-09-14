package generator

import (
	"bufio"
	"embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/funfunpayer/SamNPlayer/logging"
)

//go:embed *.py requirements.txt
var pythonFiles embed.FS

//go:embed requirements.txt
var requirementsSource []byte

// command ersetzt exec.Command in diesem Paket: gleiches Verhalten, aber mit
// hiddenSysProcAttr() (siehe exec_windows.go/exec_unix.go) - ohne das blitzt
// unter Windows für jeden der vielen Python-Unterprozesse (Vorschau laden,
// Abhängigkeiten prüfen, Region suchen, Skript erzeugen, ...) kurz ein
// Konsolenfenster auf.
func command(name string, args ...string) *exec.Cmd {
	cmd := exec.Command(name, args...)
	cmd.SysProcAttr = hiddenSysProcAttr()
	return cmd
}

type ROI struct {
	X, Y, W, H int
}

type Options struct {
	Invert                    bool
	SmoothWindow              int
	MinPeakDistanceMs         int
	MaxFrames                 int
	DisableCameraCompensation bool
	UseOpenCL                 bool
	Threads                   int
	ReportPath                string
	Backend                   string
	Profile                   string
	PeakProminence            float64
	DynamicRangeMs            float64
	MinActionIntervalMs       float64
	MaxSpeed                  float64
	Axis                      string
	AutoRetry                 bool
	AdaptiveKeyframeError     float64
	PerSceneROI               bool
	CacheDir                  string
	DisableCache              bool
	NormPercentile            float64
	RDPTolerance              float64
	DisableSceneCutDetection  bool
	ROI2                      ROI
}

func pythonCandidates() []string {
	var out []string
	seen := map[string]bool{}
	add := func(p string) {
		if p == "" || seen[p] {
			return
		}
		seen[p] = true
		out = append(out, p)
	}
	for _, name := range []string{"python3", "python", "py"} {
		if path, err := exec.LookPath(name); err == nil {
			add(path)
		}
	}
	var patterns []string
	if local := os.Getenv("LOCALAPPDATA"); local != "" {
		patterns = append(patterns, filepath.Join(local, "Programs", "Python", "Python3*", "python.exe"))
	}
	if pf := os.Getenv("ProgramFiles"); pf != "" {
		patterns = append(patterns, filepath.Join(pf, "Python3*", "python.exe"))
	}
	patterns = append(patterns, `C:\Python3*\python.exe`)
	for _, pattern := range patterns {
		matches, _ := filepath.Glob(pattern)
		for i := len(matches) - 1; i >= 0; i-- {
			add(matches[i])
		}
	}
	return out
}

func hasPackages(py string) (bool, string) {
	out, err := command(py, "-c", "import cv2, scipy, numpy").CombinedOutput()
	return err == nil, string(out)
}

func FindPython() (string, error) {
	candidates := pythonCandidates()
	if len(candidates) == 0 {
		return "", fmt.Errorf("generator: kein Python gefunden (python3/python/py im PATH) - bitte Python 3.9+ installieren: https://python.org")
	}
	var firstRunnable string
	for _, py := range candidates {
		ok, _ := hasPackages(py)
		if ok {
			return py, nil
		}
		if firstRunnable == "" {
			if err := command(py, "-c", "pass").Run(); err == nil {
				firstRunnable = py
			}
		}
	}
	if firstRunnable != "" {
		return firstRunnable, nil
	}
	return "", fmt.Errorf("generator: kein funktionierender Python-Interpreter gefunden (geprüft: %v)", candidates)
}

func CheckDependencies() error {
	candidates := pythonCandidates()
	if len(candidates) == 0 {
		_, err := FindPython()
		return err
	}
	var details []string
	for _, py := range candidates {
		ok, out := hasPackages(py)
		if ok {
			return nil
		}
		details = append(details, fmt.Sprintf("  %s: %s", py, firstLine(out)))
	}
	target, err := FindPython()
	if err != nil {
		return err
	}
	return fmt.Errorf("generator: Pakete fehlen. Installieren mit: \"%s\" -m pip install opencv-contrib-python scipy numpy\nGeprüft:\n%s", target, strings.Join(details, "\n"))
}

func firstLine(s string) string {
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "Traceback") && !strings.HasPrefix(line, "File \"") {
			return line
		}
	}
	return "nicht startbar"
}

func writeScriptToTemp() (string, error) {
	dir, err := os.MkdirTemp("", "funscript-generator-*")
	if err != nil {
		return "", fmt.Errorf("generator: Temp-Verzeichnis: %w", err)
	}
	entries, err := pythonFiles.ReadDir(".")
	if err != nil {
		os.RemoveAll(dir)
		return "", fmt.Errorf("generator: eingebettete Dateien nicht lesbar: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, "_test.py") {
			continue
		}
		content, err := pythonFiles.ReadFile(name)
		if err != nil {
			os.RemoveAll(dir)
			return "", fmt.Errorf("generator: %s nicht lesbar: %w", name, err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), content, 0644); err != nil {
			os.RemoveAll(dir)
			return "", fmt.Errorf("generator: %s konnte nicht geschrieben werden: %w", name, err)
		}
	}
	return filepath.Join(dir, "generate_funscript.py"), nil
}

func cleanupScriptTemp(scriptPath string) {
	os.RemoveAll(filepath.Dir(scriptPath))
}

func writeEmbeddedToTemp(pattern string, content []byte) (string, error) {
	f, err := os.CreateTemp("", pattern)
	if err != nil {
		return "", fmt.Errorf("generator: Temp-Datei für Skript: %w", err)
	}
	defer f.Close()
	if _, err := f.Write(content); err != nil {
		os.Remove(f.Name())
		return "", fmt.Errorf("generator: Skript konnte nicht geschrieben werden: %w", err)
	}
	return f.Name(), nil
}

func parseProgress(line string) (done, total int, ok bool) {
	if !strings.HasPrefix(line, "PROGRESS ") {
		return 0, 0, false
	}
	if n, _ := fmt.Sscanf(line, "PROGRESS %d %d", &done, &total); n != 2 {
		return 0, 0, false
	}
	return done, total, true
}

func DefaultCacheDir() string {
	dir, err := os.UserCacheDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "SamNPlayer")
}

func percentOf(done, total int) int {
	if total <= 0 {
		return -1
	}
	pct := done * 100 / total
	if pct > 100 {
		return 100
	}
	if pct < 0 {
		return 0
	}
	return pct
}

func FindROI(videoPath string, onProgress func(line string)) (ROI, error) {
	return FindROIWithProgress(videoPath, onProgress, nil)
}

func FindROIWithProgress(videoPath string, onProgress func(line string), onPercent func(pct int)) (ROI, error) {
	return findROIViaScript("auto_roi.py", nil, videoPath, "auto_roi", onProgress, onPercent)
}

// FindROIAIWithProgress ist die KI-Variante von FindROIWithProgress: gleicher
// Vertrag (stdout "ROI x y w h", stderr Fortschritt/Log), aber ai_roi.py
// (lokales ONNX-Modell) statt auto_roi.py (Rhythmus-Heuristik ohne Modell).
// modelPath == "" nutzt ai_roi.defaultModelPath() (siehe dort).
func FindROIAIWithProgress(videoPath, modelPath string, onProgress func(line string), onPercent func(pct int)) (ROI, error) {
	var extraArgs []string
	if modelPath != "" {
		extraArgs = append(extraArgs, "--model", modelPath)
	}
	return findROIViaScript("ai_roi.py", extraArgs, videoPath, "ai_roi", onProgress, onPercent)
}

// AIRoiAvailable prüft (ohne ein Video zu öffnen), ob die KI-Regionssuche
// grundsätzlich nutzbar ist - onnxruntime installiert UND ein Modell unter
// modelPath (oder ai_roi.py's Standardordner) vorhanden. Für die GUI, um den
// KI-Knopf zu aktivieren/auszublenden. Liefert false bei jedem Fehler
// (fehlendes Python, fehlende Pakete, ...) statt den Fehler durchzureichen -
// die Verfügbarkeitsprüfung soll nie selbst scheitern können.
func AIRoiAvailable(modelPath string) bool {
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
	scriptPath := filepath.Join(filepath.Dir(mainScript), "ai_roi.py")
	args := []string{scriptPath, "--check"}
	if modelPath != "" {
		args = append(args, "--model", modelPath)
	}
	out, err := command(py, args...).Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) == "AVAILABLE"
}

// findROIViaScript führt eines der beiden austauschbaren ROI-Finder-Skripte
// (auto_roi.py, ai_roi.py) aus - beide erfüllen denselben Vertrag: stdout
// "ROI x y w h", stderr optional "PROGRESS done total"-Zeilen plus Logtext.
// logPrefix kennzeichnet nur die Logzeilen, ändert das Protokoll nicht.
func findROIViaScript(scriptName string, extraArgs []string, videoPath, logPrefix string,
	onProgress func(line string), onPercent func(pct int)) (ROI, error) {
	py, err := FindPython()
	if err != nil {
		return ROI{}, err
	}
	if err := CheckDependencies(); err != nil {
		return ROI{}, err
	}
	mainScript, err := writeScriptToTemp()
	if err != nil {
		return ROI{}, err
	}
	defer cleanupScriptTemp(mainScript)
	scriptPath := filepath.Join(filepath.Dir(mainScript), scriptName)
	args := append([]string{scriptPath, "--video", videoPath}, extraArgs...)
	cmd := command(py, args...)
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return ROI{}, fmt.Errorf("generator: stderr-Pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return ROI{}, fmt.Errorf("generator: stdout-Pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return ROI{}, fmt.Errorf("generator: Start fehlgeschlagen: %w", err)
	}
	go func() {
		sc := bufio.NewScanner(stderr)
		for sc.Scan() {
			line := sc.Text()
			if done, total, ok := parseProgress(line); ok {
				if onPercent != nil {
					onPercent(percentOf(done, total))
				}
				continue
			}
			logging.Debug(logPrefix + ": " + line)
			if onProgress != nil {
				onProgress(line)
			}
		}
	}()
	var roi ROI
	found := false
	sc := bufio.NewScanner(stdout)
	for sc.Scan() {
		var x, y, w, h int
		if n, _ := fmt.Sscanf(sc.Text(), "ROI %d %d %d %d", &x, &y, &w, &h); n == 4 {
			roi = ROI{X: x, Y: y, W: w, H: h}
			found = true
		}
	}
	if err := cmd.Wait(); err != nil {
		return ROI{}, fmt.Errorf("generator: automatische Regionssuche fehlgeschlagen: %w", err)
	}
	if !found {
		return ROI{}, fmt.Errorf("generator: keine Region gefunden - bitte von Hand markieren")
	}
	logging.Info("generator: Region automatisch gefunden", "roi", fmt.Sprintf("%+v", roi))
	return roi, nil
}

func DumpFirstFrame(videoPath, outputPNG string) (width, height int, err error) {
	py, err := FindPython()
	if err != nil {
		return 0, 0, err
	}
	if err := CheckDependencies(); err != nil {
		return 0, 0, err
	}
	scriptPath, err := writeScriptToTemp()
	if err != nil {
		return 0, 0, err
	}
	defer cleanupScriptTemp(scriptPath)
	cmd := command(py, scriptPath, "--video", videoPath, "--dump-first-frame", outputPNG)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return 0, 0, fmt.Errorf("generator: Frame-Extraktion fehlgeschlagen: %w\n%s", err, string(out))
	}
	for _, line := range splitLines(string(out)) {
		var w, h int
		if n, _ := fmt.Sscanf(line, "FRAME_SIZE %d %d", &w, &h); n == 2 {
			return w, h, nil
		}
	}
	return 0, 0, fmt.Errorf("generator: Framegröße nicht aus Skript-Ausgabe lesbar: %s", string(out))
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i, r := range s {
		if r == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func Generate(videoPath string, roi ROI, outputPath string, opts Options, onProgress func(line string)) error {
	return GenerateWithProgress(videoPath, roi, outputPath, opts, onProgress, nil)
}

func buildArgs(scriptPath, videoPath, outputPath string, roi ROI, opts Options) []string {
	args := []string{
		scriptPath,
		"--video", videoPath,
		"--output", outputPath,
		"--roi", fmt.Sprintf("%d,%d,%d,%d", roi.X, roi.Y, roi.W, roi.H),
	}
	if opts.Invert {
		args = append(args, "--invert")
	}
	if opts.SmoothWindow > 0 {
		args = append(args, "--smooth-window", strconv.Itoa(opts.SmoothWindow))
	}
	if opts.MinPeakDistanceMs > 0 {
		args = append(args, "--min-peak-distance-ms", strconv.Itoa(opts.MinPeakDistanceMs))
	}
	if opts.MaxFrames > 0 {
		args = append(args, "--max-frames", strconv.Itoa(opts.MaxFrames))
	}
	if opts.DisableCameraCompensation {
		args = append(args, "--no-camera-compensation")
	}
	if opts.RDPTolerance > 0 {
		args = append(args, "--rdp-tolerance", strconv.FormatFloat(opts.RDPTolerance, 'f', -1, 64))
	}
	if opts.DisableSceneCutDetection {
		args = append(args, "--no-scene-cut-detection")
	}
	if opts.UseOpenCL {
		args = append(args, "--opencl")
	}
	if opts.Threads > 0 {
		args = append(args, "--threads", strconv.Itoa(opts.Threads))
	}
	if opts.ReportPath != "" {
		args = append(args, "--report", opts.ReportPath)
	}
	if opts.Backend == "flow" {
		args = append(args, "--backend", "flow")
	}
	if opts.Profile == "weich" || opts.Profile == "tf" || opts.Profile == "tj" {
		args = append(args, "--profile", opts.Profile)
	}
	if opts.ROI2.W > 0 && opts.ROI2.H > 0 {
		args = append(args, "--roi2", fmt.Sprintf("%d,%d,%d,%d", opts.ROI2.X, opts.ROI2.Y, opts.ROI2.W, opts.ROI2.H))
	}
	if opts.PeakProminence > 0 {
		args = append(args, "--peak-prominence", strconv.FormatFloat(opts.PeakProminence, 'f', -1, 64))
	}
	if opts.DynamicRangeMs > 0 {
		args = append(args, "--dynamic-range-ms", strconv.FormatFloat(opts.DynamicRangeMs, 'f', -1, 64))
	}
	if opts.MinActionIntervalMs > 0 {
		args = append(args, "--min-action-interval-ms", strconv.FormatFloat(opts.MinActionIntervalMs, 'f', -1, 64))
	} else if opts.MinActionIntervalMs < 0 {
		args = append(args, "--min-action-interval-ms", "0")
	}
	if opts.MaxSpeed > 0 {
		args = append(args, "--max-speed", strconv.FormatFloat(opts.MaxSpeed, 'f', -1, 64))
	}
	if opts.Axis == "x" {
		args = append(args, "--axis", "x")
	}
	if opts.AutoRetry {
		args = append(args, "--auto-retry")
	}
	if opts.AdaptiveKeyframeError > 0 {
		args = append(args, "--adaptive-keyframes", strconv.FormatFloat(opts.AdaptiveKeyframeError, 'f', -1, 64))
	}
	if opts.PerSceneROI {
		args = append(args, "--per-scene-roi")
	}
	if opts.DisableCache {
		args = append(args, "--no-cache")
	} else {
		dir := opts.CacheDir
		if dir == "" {
			dir = DefaultCacheDir()
		}
		if dir != "" {
			args = append(args, "--cache-dir", dir)
		}
	}
	if opts.NormPercentile > 0 {
		args = append(args, "--norm-percentile", strconv.FormatFloat(opts.NormPercentile, 'f', -1, 64))
	} else if opts.NormPercentile < 0 {
		args = append(args, "--norm-percentile", "0")
	}
	return args
}

func BuildArgsForTest(opts Options) []string {
	return buildArgs("script.py", "v.mp4", "o.funscript", ROI{}, opts)
}

func GenerateWithProgress(videoPath string, roi ROI, outputPath string, opts Options, onProgress func(line string), onPercent func(pct int)) error {
	logging.Info("generator: starte Generierung", "video", videoPath, "roi", fmt.Sprintf("%+v", roi), "output", outputPath)
	py, err := FindPython()
	if err != nil {
		return err
	}
	if err := CheckDependencies(); err != nil {
		return err
	}
	scriptPath, err := writeScriptToTemp()
	if err != nil {
		return err
	}
	defer cleanupScriptTemp(scriptPath)
	args := buildArgs(scriptPath, videoPath, outputPath, roi, opts)
	cmd := command(py, args...)
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("generator: stderr-Pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("generator: Start fehlgeschlagen: %w", err)
	}
	scanner := bufio.NewScanner(stderr)
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
		logging.Debug("generator: " + line)
		if onProgress != nil {
			onProgress(line)
		}
	}
	if err := cmd.Wait(); err != nil {
		logging.Error("generator: Generierung fehlgeschlagen", "fehler", err)
		return fmt.Errorf("generator: Generierung fehlgeschlagen: %w\nLetzte Ausgabe:\n%s", err, joinLines(lastLines))
	}
	logging.Info("generator: Generierung abgeschlossen", "output", outputPath)
	return nil
}

func joinLines(lines []string) string {
	out := ""
	for _, l := range lines {
		out += l + "\n"
	}
	return out
}

func AddFeedback(reportPath, outputPath, verdict, comment string) error {
	py, err := FindPython()
	if err != nil {
		return err
	}
	scriptPath, err := writeScriptToTemp()
	if err != nil {
		return err
	}
	defer cleanupScriptTemp(scriptPath)
	args := []string{scriptPath, "--report", reportPath, "--output", outputPath, "--feedback", verdict}
	if comment != "" {
		args = append(args, comment)
	}
	out, err := command(py, args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("generator: Urteil konnte nicht gespeichert werden: %w\n%s", err, string(out))
	}
	return nil
}

func ReportSummary(reportPath string) (string, error) {
	py, err := FindPython()
	if err != nil {
		return "", err
	}
	scriptPath, err := writeScriptToTemp()
	if err != nil {
		return "", err
	}
	defer cleanupScriptTemp(scriptPath)
	out, err := command(py, scriptPath, "--report", reportPath, "--report-summary").Output()
	if err != nil {
		return "", fmt.Errorf("generator: Bericht konnte nicht ausgewertet werden: %w", err)
	}
	return string(out), nil
}

func HardwareInfo() (string, error) {
	py, err := FindPython()
	if err != nil {
		return "", err
	}
	scriptPath, err := writeScriptToTemp()
	if err != nil {
		return "", err
	}
	defer cleanupScriptTemp(scriptPath)
	out, err := command(py, scriptPath, "--hardware-info").Output()
	if err != nil {
		return "", fmt.Errorf("generator: Hardware-Abfrage fehlgeschlagen: %w", err)
	}
	return string(out), nil
}

func TrainQualityModel(reportPath string) (string, error) {
	py, err := FindPython()
	if err != nil {
		return "", err
	}
	scriptPath, err := writeScriptToTemp()
	if err != nil {
		return "", err
	}
	defer cleanupScriptTemp(scriptPath)
	cmd := command(py, scriptPath, "--report", reportPath, "--train-model")
	out, err := cmd.CombinedOutput()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 2 {
			return string(out), nil
		}
		return string(out), fmt.Errorf("generator: Lernen fehlgeschlagen: %w", err)
	}
	return string(out), nil
}

func QualityModelInfo() (string, error) {
	py, err := FindPython()
	if err != nil {
		return "", err
	}
	scriptPath, err := writeScriptToTemp()
	if err != nil {
		return "", err
	}
	defer cleanupScriptTemp(scriptPath)
	out, err := command(py, scriptPath, "--model-info").Output()
	if err != nil {
		return "", fmt.Errorf("generator: Modellabfrage fehlgeschlagen: %w", err)
	}
	return string(out), nil
}
