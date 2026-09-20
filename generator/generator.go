package generator

import (
	"bufio"
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/funfunpayer/SamNPlayer/funscript"
	"github.com/funfunpayer/SamNPlayer/logging"
	"github.com/funfunpayer/SamNPlayer/videox"
)

//go:embed *.py requirements.txt requirements-ai.txt requirements-ai-train.txt
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

// commandContext is command() with a cancellable context — used by long
// generate runs so the GUI can abort without leaving an orphan Python.
func commandContext(ctx context.Context, name string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, name, args...)
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
	// ROI2Fixed keeps the second Tf/Tj box at its marked position (static
	// anchor). ROI1 is still tracked. Distance then reflects tip motion only.
	ROI2Fixed bool
	// RegionClass / RegionClass2 are optional canonical body-part IDs
	// (docs/BODY_REGIONS.md) for ROI / ROI2 — used for AI preference + UI.
	RegionClass           string
	RegionClass2          string
	AIQualityOpinion      bool
	AIBaseURL             string
	ContactVibration      bool
	ContactVibrationSpan  float64
	ContactVibrationCurve string
	AudioCheck            bool
	// StartTimeSec skips the first N seconds before tracking (GUI seek past
	// black intro). 0 = start at the beginning.
	StartTimeSec float64
	// PreferPython skips the automatic Go pipeline (CLI/tests/advanced).
	// Default false: GenerateWithContext uses trackcv or simpletrack when
	// NativePipelineEligible — no GUI checkbox required.
	PreferPython bool
	// NativePipeline is retained for JSON/API compat and ignored for routing.
	// Go is chosen automatically when eligible unless PreferPython is set.
	NativePipeline bool
	// DetrendWindowMs / Bandpass* — FunGen/Flow-inspired post filters (0 = off).
	DetrendWindowMs float64
	BandpassLowHz   float64
	BandpassHighHz  float64
	// FlowDownscale shrinks frames for the optical-flow backend (e.g. 0.5).
	// 0 or 1 = full resolution. Ignored by CSRT/native.
	FlowDownscale float64
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
	// CSRT/KCF must exist — plain opencv-python (no contrib) imports cv2
	// but leaves create_tracker() dead (Issues #94/#95).
	script := "import cv2, scipy, numpy\n" +
		"ok=False\n" +
		"leg=getattr(cv2,'legacy',None)\n" +
		"for free,cls in (('TrackerCSRT_create','TrackerCSRT'),('TrackerKCF_create','TrackerKCF'),('TrackerMIL_create','TrackerMIL')):\n" +
		"  if callable(getattr(cv2,free,None)) or (getattr(cv2,cls,None) is not None and callable(getattr(getattr(cv2,cls,None),'create',None))):\n" +
		"    ok=True; break\n" +
		"  if leg is not None and (callable(getattr(leg,free,None)) or (getattr(leg,cls,None) is not None and callable(getattr(getattr(leg,cls,None),'create',None)))):\n" +
		"    ok=True; break\n" +
		"assert ok, 'OpenCV ohne Tracker (CSRT/KCF) — oft opencv-python statt opencv-contrib-python. pip uninstall opencv-python opencv-python-headless && pip install opencv-contrib-python'\n"
	out, err := command(py, "-c", script).CombinedOutput()
	return err == nil, string(out)
}

func FindPython() (string, error) {
	candidates := pythonCandidates()
	if len(candidates) == 0 {
		return "", fmt.Errorf("generator: no Python found (python3/python/py on PATH) — install Python 3.9+: https://python.org")
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
	return "", fmt.Errorf("generator: no working Python interpreter found (checked: %v)", candidates)
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
	return fmt.Errorf("generator: missing packages. Install with: \"%s\" -m pip install opencv-contrib-python scipy numpy\n(Important: do not install alongside opencv-python — it removes CSRT.)\nChecked:\n%s", target, strings.Join(details, "\n"))
}

func firstLine(s string) string {
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "Traceback") && !strings.HasPrefix(line, "File \"") {
			return line
		}
	}
	return "failed to start"
}

func writeScriptToTemp() (string, error) {
	dir, err := os.MkdirTemp("", "funscript-generator-*")
	if err != nil {
		return "", fmt.Errorf("generator: Temp-Verzeichnis: %w", err)
	}
	entries, err := pythonFiles.ReadDir(".")
	if err != nil {
		os.RemoveAll(dir)
		return "", fmt.Errorf("generator: embedded files not readable: %w", err)
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
			return "", fmt.Errorf("generator: %s not readable: %w", name, err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), content, 0644); err != nil {
			os.RemoveAll(dir)
			return "", fmt.Errorf("generator: %s could not be written: %w", name, err)
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
		return "", fmt.Errorf("generator: temp file for script: %w", err)
	}
	defer f.Close()
	if _, err := f.Write(content); err != nil {
		os.Remove(f.Name())
		return "", fmt.Errorf("generator: script could not be written: %w", err)
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

// FindTwoROIsWithProgress schlägt ROI1+ROI2 vor (auto_roi --two). Nur
// Vorschlag — GUI muss bestätigen/korrigieren (docs/NEXT.md Priorität 3).
func FindTwoROIsWithProgress(videoPath string, onProgress func(line string), onPercent func(pct int)) (ROI, ROI, error) {
	return findTwoROIsViaScript("auto_roi.py", []string{"--two"}, videoPath, "auto_roi", onProgress, onPercent)
}

// FindROIAIWithProgress is the AI variant of FindROIWithProgress: same
// stdout/stderr contract, but ai_roi.py (local ONNX) instead of auto_roi.py.
// modelPath == "" uses ai_roi.default_model_path(). preferredClasses is a
// comma-separated list of names or ids (empty = no class filter).
func FindROIAIWithProgress(videoPath, modelPath, preferredClasses string, onProgress func(line string), onPercent func(pct int)) (ROI, error) {
	var extraArgs []string
	if modelPath != "" {
		extraArgs = append(extraArgs, "--model", modelPath)
	}
	if preferredClasses != "" {
		extraArgs = append(extraArgs, "--preferred-classes", preferredClasses)
		if modelPath != "" {
			extraArgs = append(extraArgs, "--classes-json", filepath.Dir(modelPath))
		}
	}
	return findROIViaScript("ai_roi.py", extraArgs, videoPath, "ai_roi", onProgress, onPercent)
}

// FindTwoROIsAIWithProgress is the AI variant of FindTwoROIsWithProgress
// (ai_roi.py --two). roi2 may be empty when no second object was found.
func FindTwoROIsAIWithProgress(videoPath, modelPath, preferredClasses string, onProgress func(line string), onPercent func(pct int)) (ROI, ROI, error) {
	extraArgs := []string{"--two"}
	if modelPath != "" {
		extraArgs = append(extraArgs, "--model", modelPath)
	}
	if preferredClasses != "" {
		extraArgs = append(extraArgs, "--preferred-classes", preferredClasses)
		if modelPath != "" {
			extraArgs = append(extraArgs, "--classes-json", filepath.Dir(modelPath))
		}
	}
	return findTwoROIsViaScript("ai_roi.py", extraArgs, videoPath, "ai_roi", onProgress, onPercent)
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

// SupportSignalsAvailable reports experimental depth/pose helper availability
// (classical proxy always on; ONNX flags only when optional models exist).
// Returns nil on any probe failure so callers can treat it as unavailable.
func SupportSignalsAvailable(depthOnnxPath, poseOnnxPath string) map[string]bool {
	py, err := FindPython()
	if err != nil {
		return nil
	}
	if err := CheckDependencies(); err != nil {
		return nil
	}
	mainScript, err := writeScriptToTemp()
	if err != nil {
		return nil
	}
	defer cleanupScriptTemp(mainScript)
	scriptPath := filepath.Join(filepath.Dir(mainScript), "support_signals.py")
	args := []string{scriptPath, "--check"}
	if depthOnnxPath != "" {
		args = append(args, "--depth-onnx", depthOnnxPath)
	}
	if poseOnnxPath != "" {
		args = append(args, "--pose-onnx", poseOnnxPath)
	}
	out, err := command(py, args...).Output()
	if err != nil {
		return nil
	}
	var flags map[string]bool
	if err := json.Unmarshal(out, &flags); err != nil {
		return nil
	}
	return flags
}

// AudioCheckAvailable prüft, ob --audio-check grundsätzlich nutzbar ist -
// nur ffmpeg auf dem PATH nötig (Go: CheckAudioTempo / Python: audio_check.py).
// Kein separates Python-Paket. Für die GUI, um die Checkbox zu aktivieren/
// auszublenden statt sie anzubieten und dann bei jedem Versuch mit
// "nicht möglich" scheitern zu lassen.
func AudioCheckAvailable() bool {
	return videox.Available()
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
		return ROI{}, fmt.Errorf("generator: start failed: %w", err)
	}
	// cmd.Wait() schließt die Pipes, sobald der Prozess beendet ist - laut
	// os/exec-Doku "incorrect to call Wait before all reads from the pipe
	// have completed". stdout enthält hier nur eine kurze "ROI ..."-Zeile
	// und ist meist sofort fertig gelesen, während stderr (Fortschritt/Log,
	// oft viel mehr Text) noch in der Goroutine läuft - ohne WaitGroup
	// konnte Wait() die stderr-Pipe schließen, bevor die Goroutine
	// überhaupt zu lesen begonnen hatte, und praktisch der gesamte
	// Fortschritt/Log ging verloren (reproduziert: 0 von 20000 Testzeilen
	// empfangen).
	var stderrDone sync.WaitGroup
	stderrDone.Add(1)
	go func() {
		defer stderrDone.Done()
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
	stderrDone.Wait()
	if err := cmd.Wait(); err != nil {
		return ROI{}, fmt.Errorf("generator: automatic region search failed: %w", err)
	}
	if !found {
		return ROI{}, fmt.Errorf("generator: no region found — mark manually")
	}
	logging.Info("generator: region found automatically", "roi", fmt.Sprintf("%+v", roi))
	return roi, nil
}

// findTwoROIsViaScript wie findROIViaScript, liest zusätzlich optional
// "ROI2 x y w h". Fehlt ROI2, ist der zweite Rückgabewert leer (W=0).
func findTwoROIsViaScript(scriptName string, extraArgs []string, videoPath, logPrefix string,
	onProgress func(line string), onPercent func(pct int)) (ROI, ROI, error) {
	py, err := FindPython()
	if err != nil {
		return ROI{}, ROI{}, err
	}
	if err := CheckDependencies(); err != nil {
		return ROI{}, ROI{}, err
	}
	mainScript, err := writeScriptToTemp()
	if err != nil {
		return ROI{}, ROI{}, err
	}
	defer cleanupScriptTemp(mainScript)
	scriptPath := filepath.Join(filepath.Dir(mainScript), scriptName)
	args := append([]string{scriptPath, "--video", videoPath}, extraArgs...)
	cmd := command(py, args...)
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return ROI{}, ROI{}, fmt.Errorf("generator: stderr-Pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return ROI{}, ROI{}, fmt.Errorf("generator: stdout-Pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return ROI{}, ROI{}, fmt.Errorf("generator: start failed: %w", err)
	}
	var stderrDone sync.WaitGroup
	stderrDone.Add(1)
	go func() {
		defer stderrDone.Done()
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
	var roi, roi2 ROI
	found := false
	sc := bufio.NewScanner(stdout)
	for sc.Scan() {
		line := sc.Text()
		var x, y, w, h int
		if n, _ := fmt.Sscanf(line, "ROI2 %d %d %d %d", &x, &y, &w, &h); n == 4 {
			roi2 = ROI{X: x, Y: y, W: w, H: h}
			continue
		}
		if n, _ := fmt.Sscanf(line, "ROI %d %d %d %d", &x, &y, &w, &h); n == 4 {
			roi = ROI{X: x, Y: y, W: w, H: h}
			found = true
		}
	}
	stderrDone.Wait()
	if err := cmd.Wait(); err != nil {
		return ROI{}, ROI{}, fmt.Errorf("generator: automatic two-region search failed: %w", err)
	}
	if !found {
		return ROI{}, ROI{}, fmt.Errorf("generator: no region found — mark manually")
	}
	logging.Info("generator: regions found automatically",
		"roi", fmt.Sprintf("%+v", roi), "roi2", fmt.Sprintf("%+v", roi2))
	return roi, roi2, nil
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
		return 0, 0, fmt.Errorf("generator: frame extraction failed: %w\n%s", err, string(out))
	}
	for _, line := range splitLines(string(out)) {
		var w, h int
		if n, _ := fmt.Sscanf(line, "FRAME_SIZE %d %d", &w, &h); n == 2 {
			return w, h, nil
		}
	}
	return 0, 0, fmt.Errorf("generator: frame size not readable from script output: %s", string(out))
}

// ProfileSuggestion ist das Ergebnis von SuggestProfile. Found=false ist ein
// normales Ergebnis (keine gespeicherte Szene nah genug, kein KI-Server
// erreichbar), kein Fehler - der Aufrufer entscheidet, was er anzeigt.
type ProfileSuggestion struct {
	Found bool
	Label string
	Kind  string // "measured" (motion_signature, kein KI) oder "ai" (Colibri)
	// Confidence: bei Kind=="ai" eine 0..1-Konfidenz (höher = sicherer).
	// Bei Kind=="measured" stattdessen der Signaturabstand zur nächsten
	// gespeicherten Szene (niedriger = ähnlicher) - andere Skala, gleiches
	// Feld, weil beide Fälle nie gleichzeitig auftreten.
	Confidence float64
}

// SuggestProfile fragt --suggest-profile ab (generate_funscript.py, siehe
// motion_signature.py/ai_profile.py): vergleicht die Bewegungssignatur des
// Videos zuerst gegen mit LabelScene benannte Szenen (gemessen, keine KI),
// und nur wenn keine nah genug ist, gegen einen optionalen Colibri-Server.
// baseURL == "" nutzt colibri_client.DEFAULT_BASE_URL.
func SuggestProfile(videoPath, baseURL string) (ProfileSuggestion, error) {
	py, err := FindPython()
	if err != nil {
		return ProfileSuggestion{}, err
	}
	if err := CheckDependencies(); err != nil {
		return ProfileSuggestion{}, err
	}
	scriptPath, err := writeScriptToTemp()
	if err != nil {
		return ProfileSuggestion{}, err
	}
	defer cleanupScriptTemp(scriptPath)
	args := []string{scriptPath, "--video", videoPath, "--suggest-profile"}
	if baseURL != "" {
		args = append(args, "--ai-base-url", baseURL)
	}
	out, err := command(py, args...).Output()
	if err != nil {
		return ProfileSuggestion{}, fmt.Errorf("generator: profile suggestion failed: %w", err)
	}
	// "PROFILE_SUGGESTION <label> <measured|ai> <confidence>" - Label kann
	// selbst Leerzeichen enthalten (freier Szenenname), darum von den festen
	// letzten beiden Feldern her parsen statt von vorne zu zählen.
	for _, line := range splitLines(string(out)) {
		fields := strings.Fields(line)
		if len(fields) < 4 || fields[0] != "PROFILE_SUGGESTION" {
			continue
		}
		confidence, _ := strconv.ParseFloat(fields[len(fields)-1], 64)
		return ProfileSuggestion{
			Found:      true,
			Label:      strings.Join(fields[1:len(fields)-2], " "),
			Kind:       fields[len(fields)-2],
			Confidence: confidence,
		}, nil
	}
	return ProfileSuggestion{}, nil
}

// LabelScene speichert die Bewegungssignatur des Videos unter label (siehe
// --label-scene, motion_signature.py) - die Grundlage, gegen die
// SuggestProfile spätere, ähnliche Szenen misst.
func LabelScene(videoPath, label string) error {
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
	out, err := command(py, scriptPath, "--video", videoPath, "--label-scene", label).CombinedOutput()
	if err != nil {
		return fmt.Errorf("generator: scene could not be saved: %w\n%s", err, string(out))
	}
	return nil
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

func GenerateWithProgress(videoPath string, roi ROI, outputPath string, opts Options, onProgress func(line string), onPercent func(pct int)) error {
	return GenerateWithContext(context.Background(), videoPath, roi, outputPath, opts, onProgress, onPercent)
}

// GenerateWithContext runs generation under ctx. Cancel ctx to kill the
// Python subprocess (review: generation must be abortable) or abort native
// tracking mid-loop. Returns context.Canceled when aborted.
//
// When PreferPython is false and opts+roi are NativePipelineEligible, uses
// the Go path automatically (CSRT via trackcv when OpenCV is linked, else
// simpletrack over videox) — no opt-in flag. Eligible failures are NOT
// soft-failed to Python (overhead / hides Go bugs); PreferPython forces
// the Python path explicitly.
func GenerateWithContext(ctx context.Context, videoPath string, roi ROI, outputPath string, opts Options, onProgress func(line string), onPercent func(pct int)) error {
	if ctx == nil {
		ctx = context.Background()
	}
	logging.Info("generator: starting generation", "video", videoPath, "roi", fmt.Sprintf("%+v", roi), "output", outputPath)

	if !opts.PreferPython && NativePipelineEligible(opts, roi) {
		var err error
		if NativeTrackingAvailable() {
			err = GenerateNativeCSRT(ctx, videoPath, roi, outputPath, opts, onProgress, onPercent)
		} else {
			err = GenerateNativeSimple(ctx, videoPath, roi, outputPath, opts, onProgress, onPercent)
		}
		if err == nil {
			logging.Info("generator: native generation finished", "output", outputPath)
			return nil
		}
		return err
	}
	if !opts.PreferPython && opts.NativePipeline {
		logging.Warn("generator: Go pipeline not usable for these settings — using Python",
			"csrt", NativeTrackingAvailable(),
			"simple", SimpleTrackingAvailable(),
			"backend", opts.Backend,
			"roi2", opts.ROI2.W > 0,
			"auto_retry", opts.AutoRetry)
		if onProgress != nil {
			onProgress("Go pipeline not usable for these settings — using Python")
		}
	}

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
	cmd := commandContext(ctx, py, args...)
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("generator: stderr-Pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("generator: start failed: %w", err)
	}
	scanner := bufio.NewScanner(stderr)
	var lastLines []string
	for scanner.Scan() {
		if err := ctx.Err(); err != nil {
			_ = cmd.Wait()
			return err
		}
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
		if errors.Is(ctx.Err(), context.Canceled) || errors.Is(err, context.Canceled) {
			logging.Info("generator: generation cancelled")
			return context.Canceled
		}
		if errors.Is(ctx.Err(), context.DeadlineExceeded) || errors.Is(err, context.DeadlineExceeded) {
			return context.DeadlineExceeded
		}
		logging.Error("generator: generation failed", "error", err)
		return fmt.Errorf("generator: generation failed: %w\nLast output:\n%s", err, joinLines(lastLines))
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	logging.Info("generator: generation finished", "output", outputPath)
	return nil
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
	if opts.Backend != "" && opts.Backend != "csrt" {
		args = append(args, "--backend", opts.Backend)
	}
	if opts.Profile == "weich" || opts.Profile == "tf" || opts.Profile == "tj" || opts.Profile == "autotune" {
		args = append(args, "--profile", opts.Profile)
	}
	if opts.ROI2.W > 0 && opts.ROI2.H > 0 {
		args = append(args, "--roi2", fmt.Sprintf("%d,%d,%d,%d", opts.ROI2.X, opts.ROI2.Y, opts.ROI2.W, opts.ROI2.H))
		if opts.ROI2Fixed {
			args = append(args, "--roi2-fixed")
		}
	}
	if opts.RegionClass != "" {
		args = append(args, "--region-class", opts.RegionClass)
	}
	if opts.RegionClass2 != "" {
		args = append(args, "--region-class2", opts.RegionClass2)
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
	if opts.DetrendWindowMs > 0 {
		args = append(args, "--detrend-ms", strconv.FormatFloat(opts.DetrendWindowMs, 'f', -1, 64))
	}
	if opts.BandpassLowHz > 0 || opts.BandpassHighHz > 0 {
		lo, hi := opts.BandpassLowHz, opts.BandpassHighHz
		if lo <= 0 {
			lo = 0.5
		}
		if hi <= 0 {
			hi = 4
		}
		args = append(args, "--bandpass-hz",
			strconv.FormatFloat(lo, 'f', -1, 64)+","+strconv.FormatFloat(hi, 'f', -1, 64))
	}
	if opts.FlowDownscale > 0 && opts.FlowDownscale != 1 {
		args = append(args, "--flow-downscale", strconv.FormatFloat(opts.FlowDownscale, 'f', -1, 64))
	}
	if opts.Axis == "x" || opts.Axis == "y" {
		// Leer bleibt "auto" (Pythons Standard seit der automatischen
		// Achsenwahl) - nur eine explizite Erzwingung wird weitergereicht.
		args = append(args, "--axis", opts.Axis)
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
	if opts.AIQualityOpinion {
		args = append(args, "--ai-quality-opinion")
	}
	if opts.AIBaseURL != "" {
		args = append(args, "--ai-base-url", opts.AIBaseURL)
	}
	if opts.ContactVibration {
		args = append(args, "--contact-vibration")
		if opts.ContactVibrationSpan > 0 {
			args = append(args, "--contact-vibration-span",
				strconv.FormatFloat(opts.ContactVibrationSpan, 'f', -1, 64))
		}
		if opts.ContactVibrationCurve != "" && opts.ContactVibrationCurve != "linear" {
			args = append(args, "--contact-vibration-curve", opts.ContactVibrationCurve)
		}
	}
	if opts.AudioCheck {
		args = append(args, "--audio-check")
	}
	if opts.StartTimeSec > 0 {
		args = append(args, "--start-seconds", strconv.FormatFloat(opts.StartTimeSec, 'f', 3, 64))
	}
	return args
}

func BuildArgsForTest(opts Options) []string {
	return buildArgs("script.py", "v.mp4", "o.funscript", ROI{}, opts)
}

func joinLines(lines []string) string {
	out := ""
	for _, l := range lines {
		out += l + "\n"
	}
	return out
}

// ScriptQualityResult ist die Antwort von --script-quality (siehe
// generate_funscript.py) - derselbe Vertrag wie quality_doctor.evaluate()
// selbst zurückgibt, plus EstimatedFromScriptOnly als Warnhinweis: ohne
// Video fehlen die trackingbasierten Prüfungen (aktiver Zeitanteil,
// Rekonstruktionsfehler, Tracker-Verlust, Bewegungsspielraum), das Ergebnis
// ist darum vorsichtiger zu lesen als nach einer echten Generierung.
//
// Kind is always "signal_quality" (docs/SIGNAL_VS_FIDELITY.md) — not Motion
// Fidelity against video/reference.
type ScriptQualityResult struct {
	Score                   float64  `json:"score"`
	Passed                  bool     `json:"passed"`
	Warnings                []string `json:"warnings"`
	EstimatedFromScriptOnly bool     `json:"estimatedFromScriptOnly"`
	Kind                    string   `json:"kind"`
}

// ScriptQuality wendet Quality Doctor auf eine bereits vorhandene
// .funscript-Datei an, ohne Video - z.B. eine aus einem anderen Werkzeug
// importierte Datei ("Script Doctor"). Pure Go (funscript.EvaluateScriptQuality);
// no Python install required for this check.
//
// Actions are read in file order (not via Load/Parse, which sorts) so that
// unsorted timestamps are still flagged — same as quality_doctor on the
// Python --script-quality path.
func ScriptQuality(funscriptPath string) (ScriptQualityResult, error) {
	data, err := os.ReadFile(funscriptPath)
	if err != nil {
		return ScriptQualityResult{}, fmt.Errorf("generator: script could not be read: %w", err)
	}
	var doc struct {
		Actions []funscript.Action `json:"actions"`
		General []funscript.Action `json:"general"` // .samn native
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return ScriptQualityResult{}, fmt.Errorf("generator: invalid JSON: %w", err)
	}
	actions := doc.Actions
	if len(actions) == 0 {
		actions = doc.General
	}
	if len(actions) == 0 {
		return ScriptQualityResult{}, fmt.Errorf("generator: no actions found in script")
	}
	got := funscript.EvaluateScriptQuality(actions)
	return ScriptQualityResult{
		Score:                   got.Score,
		Passed:                  got.Passed,
		Warnings:                got.Warnings,
		EstimatedFromScriptOnly: true,
		Kind:                    "signal_quality",
	}, nil
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
		return fmt.Errorf("generator: verdict could not be saved: %w\n%s", err, string(out))
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
		return "", fmt.Errorf("generator: report could not be evaluated: %w", err)
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
		return "", fmt.Errorf("generator: hardware query failed: %w", err)
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
		return string(out), fmt.Errorf("generator: training failed: %w", err)
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
		return "", fmt.Errorf("generator: model query failed: %w", err)
	}
	return string(out), nil
}

// BenchmarkCorrelation ist das Ergebnis von fungen_compare.best_lag_correlation()
// für einen Golden-Clip-Eintrag mit hinterlegter FunGen-Referenz - nur
// vorhanden, wenn der Manifest-Eintrag ein reference_funscript trägt.
type BenchmarkCorrelation struct {
	R             float64  `json:"r"`
	LagMs         int      `json:"lag_ms"`
	Orientation   string   `json:"orientation"`
	NSamples      int      `json:"n_samples"`
	ShapeError    *float64 `json:"shape_error"`
	LowConfidence bool     `json:"low_confidence"`
}

// BenchmarkClipResult ist ein einzelner Clip aus golden_clip_benchmark.py's
// run_clip(). OK=false bedeutet: dieser Clip ist fehlgeschlagen (siehe
// Error), hat den restlichen Lauf aber nicht abgebrochen.
type BenchmarkClipResult struct {
	Name            string                `json:"name"`
	OK              bool                  `json:"ok"`
	Error           string                `json:"error,omitempty"`
	QualityScore    *float64              `json:"quality_score"`
	QualityPassed   *bool                 `json:"quality_passed"`
	QualityWarnings []string              `json:"quality_warnings"`
	Correlation     *BenchmarkCorrelation `json:"correlation"`
}

// BenchmarkSummary fasst einen ganzen Lauf zusammen - siehe
// golden_clip_benchmark.summarize().
type BenchmarkSummary struct {
	Total              int      `json:"total"`
	OK                 int      `json:"ok"`
	Failed             int      `json:"failed"`
	QualityPassed      int      `json:"quality_passed"`
	MeanQualityScore   *float64 `json:"mean_quality_score"`
	MeanCorrelation    *float64 `json:"mean_correlation"`
	ClipsWithReference int      `json:"clips_with_reference"`
}

// BenchmarkResult ist ein vollständiger Golden-Clip-Benchmark-Lauf - jede
// Zeile der Verlaufsdatei (--history) ist genau eine BenchmarkResult als
// JSON, siehe golden_clip_benchmark.run_benchmark()/append_history().
type BenchmarkResult struct {
	Timestamp string                `json:"timestamp"`
	GitCommit string                `json:"git_commit"`
	Manifest  string                `json:"manifest"`
	Clips     []BenchmarkClipResult `json:"clips"`
	Summary   BenchmarkSummary      `json:"summary"`
}

// RunGoldenClipBenchmark führt generator/golden_clip_benchmark.py gegen ein
// Manifest fester Vergleichs-Clips aus (docs/NEXT.md Priorität 2: "Reproduce
// before changing the algorithm" - eine feste, wiederholbare Vergleichsbasis
// statt Einzelmessungen). historyPath="" bedeutet: kein Verlaufseintrag,
// nur der aktuelle Lauf.
func RunGoldenClipBenchmark(manifestPath, historyPath string, onProgress func(line string), onPercent func(pct int)) (BenchmarkResult, error) {
	var result BenchmarkResult
	py, err := FindPython()
	if err != nil {
		return result, err
	}
	if err := CheckDependencies(); err != nil {
		return result, err
	}
	mainScript, err := writeScriptToTemp()
	if err != nil {
		return result, err
	}
	defer cleanupScriptTemp(mainScript)
	scriptPath := filepath.Join(filepath.Dir(mainScript), "golden_clip_benchmark.py")

	jsonOut, err := os.CreateTemp("", "golden-clip-result-*.json")
	if err != nil {
		return result, fmt.Errorf("generator: temp file for benchmark result: %w", err)
	}
	jsonOutPath := jsonOut.Name()
	jsonOut.Close()
	defer os.Remove(jsonOutPath)

	args := []string{scriptPath, "--manifest", manifestPath, "--json-output", jsonOutPath}
	if historyPath != "" {
		args = append(args, "--history", historyPath)
	}
	cmd := command(py, args...)
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return result, fmt.Errorf("generator: stderr-Pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return result, fmt.Errorf("generator: start failed: %w", err)
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
		return result, fmt.Errorf("generator: golden-clip benchmark failed: %w\nLast output:\n%s", err, joinLines(lastLines))
	}
	data, err := os.ReadFile(jsonOutPath)
	if err != nil {
		return result, fmt.Errorf("generator: benchmark result not readable: %w", err)
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return result, fmt.Errorf("generator: benchmark result invalid: %w", err)
	}
	return result, nil
}
