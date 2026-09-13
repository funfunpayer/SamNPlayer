// Package generator erzeugt .funscript-Dateien aus Video per klassischem
// CV-Motion-Tracking (kein Deep Learning, kein trainiertes Modell - der
// Nutzer markiert eine Bildregion, ein OpenCV-Tracker verfolgt sie).
//
// Für deutlich höhere Qualität (Deep-Learning-Objekterkennung, VR-Support,
// automatische Szenenerkennung) siehe FunGen 2 (https://fungen.app) - dessen
// Ausgabe (.funscript) diesen Player ohnehin schon anstandslos abspielt,
// eine Integration hier ist dafür nicht nötig. Der hier eingebaute Generator
// ist bewusst der einfache, komplett selbst enthaltene Ansatz ohne externe
// Abhängigkeit von einem separaten Tool.
//
// Das eigentliche Tracking passiert in generate_funscript.py (Python +
// OpenCV/scipy) - eigenständig geschrieben, nicht von FunGen übernommen
// (dessen Quellcode steht unter der PolyForm Strict License, die genau das
// verbietet: https://polyformproject.org/licenses/strict/1.0.0). Das Skript
// ist per go:embed in dieses Binary eingebettet, damit die fertige .exe eine
// einzelne Datei bleibt - zur Laufzeit wird es in eine Temp-Datei geschrieben
// und dort per "python3"/"python"/"py" ausgeführt.
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

// Sämtliche Python-Dateien werden als Verzeichnis eingebettet, nicht
// einzeln aufgezählt.
//
// Der Grund ist ein echter Fehler, der genau so passiert ist: eingebettet
// waren vier Dateien, ins Temp-Verzeichnis geschrieben wurden zwei. Im
// Entwicklungsbaum fiel das nie auf, weil dort alle Module nebeneinander
// liegen. In der fertigen .exe scheiterte dagegen "--backend flow" am
// fehlenden flow_backend, das gelernte Qualitätsmodell wurde nie gefunden,
// und die Geräteprüfung lief still gar nicht - sie steht in einem
// try/except und verschwand damit lautlos.
//
// Mit einem Verzeichnismuster kann ein neu hinzugefügtes Modul nicht mehr
// vergessen werden. Der Test in generator_embed_test.go prüft zusätzlich,
// dass jedes von generate_funscript.py importierte lokale Modul auch
// wirklich dabei ist.
//
//go:embed *.py requirements.txt
var pythonFiles embed.FS

// requirementsSource bleibt einzeln, weil die Fehlermeldung bei fehlenden
// Paketen den Inhalt anzeigt.
//
//go:embed requirements.txt
var requirementsSource []byte

// ROI ist die vom Nutzer im ersten Frame markierte Bildregion (Pixelkoordinaten).
type ROI struct {
	X, Y, W, H int
}

// Options steuert die Feinheiten der Kurvenerzeugung - siehe generate_funscript.py
// für die Bedeutung im Detail.
type Options struct {
	Invert            bool
	SmoothWindow      int // 0 = Skript-Standard verwenden
	MinPeakDistanceMs int // 0 = Skript-Standard verwenden
	MaxFrames         int // 0 = ganzes Video (zum Testen auf wenige Frames begrenzbar)
	// DisableCameraCompensation schaltet die Kamerabewegungs-Kompensation ab
	// (siehe generate_funscript.py). Default false = Kompensation aktiv,
	// wie von der Projektdokumentation empfohlen; das Skript fällt selbst
	// sicher auf "keine Korrektur" zurück, wenn zu wenige verlässliche
	// Hintergrund-Features gefunden werden.
	DisableCameraCompensation bool
	// UseOpenCL verlagert geeignete Rechenschritte auf die Grafikkarte.
	// Wichtig zu wissen: die pip-Pakete opencv-python/opencv-contrib-python
	// enthalten KEIN CUDA. OpenCL ist damit der einzige Weg, eine
	// NVIDIA-Karte ohne selbst kompiliertes OpenCV überhaupt zu nutzen.
	// Ob es schneller ist, hängt von Karte und Treiber ab - deshalb ein
	// Schalter und keine stille Voreinstellung.
	UseOpenCL bool
	// Threads: OpenCV-Threads je Prozess. 0 = automatisch.
	Threads int
	// ReportPath: Zieldatei für die Messwerte dieses Laufs. Leer = kein
	// Bericht.
	ReportPath string
	// Backend wählt das Analyseverfahren: "" bzw. "csrt" verfolgt eine
	// markierte Region mit einem Tracker, "flow" bestimmt das
	// Bewegungszentrum je Frame aus dichtem Optical Flow. Der Flow-Weg
	// braucht keine markierte Region und ist rund 4x schneller (gemessen:
	// 11.5s gegen 51s für dasselbe 20-Sekunden-Video).
	Backend string
	// Profile wählt Voreinstellungen für eine Bewegungsart: "" bzw.
	// "standard" für Hubbewegung, "weich" für weiches Gewebe, das nach
	// einem Anstoß gedämpft ausschwingt.
	Profile string
	// PeakProminence unterdrückt Nachschwingungen: Mindest-Prominenz eines
	// Extremwerts als Anteil der Gesamtauslenkung. 0 = aus.
	PeakProminence float64
	// DynamicRangeMs aktiviert die gleitende Normalisierung über ein Fenster
	// dieser Länge. 0 = aus, Empfehlung 3000.
	//
	// Grund: eine globale Normalisierung legt EINE Skala über das ganze
	// Video, Abschnitte mit schwächerer Bewegung bleiben dauerhaft schwach.
	// An einem echten Skript gemessen wurden in der Hälfte aller Fenster nur
	// 30 von 100 Punkten genutzt; die mittlere Bewegungsstärke stieg mit
	// gleitender Dynamik von 10.1 auf 26.2.
	DynamicRangeMs float64
	// MinActionIntervalMs erzwingt einen Mindestabstand zwischen Actions.
	// 0 = Skript-Standard (100ms). Negativ = ausdrücklich abschalten.
	//
	// Grund: Actions, die dichter aufeinanderfolgen, kann das Gerät nicht
	// mehr einzeln ausführen, der Abschnitt gerät aus dem Takt. Sie
	// entstehen systematisch, weil Hoch- und Tiefpunkte getrennt gesucht
	// werden - bei verrauschten Signalen lagen dadurch bis zu 45% der
	// Actions darunter.
	MinActionIntervalMs float64
	// MaxSpeed begrenzt die Positionsänderung auf Einheiten pro Sekunde
	// (0-100-Skala). 0 = aus. Sprünge, die schneller sind als das Gerät
	// fahren kann, werden vom Gerät nicht schneller ausgeführt, sondern
	// abgeschnitten - das Ergebnis fühlt sich schwächer an als ein Skript,
	// das die Grenze einhält.
	MaxSpeed float64
	// Axis wählt die ausgewertete Bewegungsachse ("y" senkrecht, "x"
	// waagerecht). Leer = senkrecht.
	Axis string
	// AutoRetry lässt bei nicht bestandener Qualitätsprüfung alternative
	// Signalparameter durchprobieren und behält das beste Ergebnis.
	AutoRetry bool
	// AdaptiveKeyframeError setzt zusätzliche Keyframes, bis die Kurve auf
	// diesen Fehler (in Punkten 0-100) genau wiedergegeben wird. 0 = aus.
	// Reine Peak/Valley-Punkte beschreiben asymmetrische Bewegungen falsch,
	// weil dazwischen linear interpoliert wird: an einem Hub mit schnellem
	// Anstieg und langsamem Absinken lag die Rekonstruktion um 28 von 100
	// Punkten daneben, mit Toleranz 4 nur noch um 2.8.
	AdaptiveKeyframeError float64
	// PerSceneROI sucht nach jedem Szenenschnitt eine neue Bewegungsregion,
	// statt die der ersten Szene über Schnittgrenzen hinweg weiterzuverwenden.
	// Ohne das klebt der Tracker nach einem Schnitt auf Hintergrund, wenn das
	// Objekt in der neuen Szene woanders liegt - und meldet dabei KEINEN
	// Verlust, weil er den falschen Ausschnitt zuverlässig verfolgt.
	// Kostet einen zusätzlichen Durchlauf über das Video.
	PerSceneROI bool
	// CacheDir legt fest, wo Trackingergebnisse zwischengespeichert werden.
	// Leer = plattformüblicher Ort (siehe DefaultCacheDir). Das Verfolgen der
	// ROI ist der mit Abstand teuerste Schritt; alles danach rechnet in
	// Sekundenbruchteilen. Ohne Cache müsste jede Parameteränderung das
	// komplette Video erneut dekodieren.
	CacheDir string
	// DisableCache schaltet die Zwischenspeicherung ganz ab.
	DisableCache bool
	// NormPercentile steuert die Normalisierung der Kurve auf 0-100.
	// Default (0 = nicht gesetzt) bedeutet: Skript-Standard verwenden, also
	// robuste Perzentil-Normalisierung. Ein einzelner Tracker-Ausreißer legt
	// bei reiner Min/Max-Normalisierung die Skala des gesamten Skripts fest
	// und drückt die echte Bewegung in einen schmalen Mittelbereich.
	// Negativ = ausdrücklich altes Min/Max-Verhalten erzwingen.
	NormPercentile float64
	// RDPTolerance aktiviert eine zusätzliche Ramer-Douglas-Peucker-
	// Simplifizierung der erkannten Keyframes. 0 = abgeschaltet (Default).
	// Einheit: Positionswerte (0-100), nicht Pixel oder Millisekunden.
	RDPTolerance float64
	// DisableSceneCutDetection schaltet die Szenenschnitt-Erkennung ab
	// (siehe generate_funscript.py). Default false = Erkennung aktiv - bei
	// einem harten Schnitt wird der Tracker an der zuletzt bekannten
	// Position neu verankert, statt blind über den Schnitt hinweg zu
	// tracken.
	DisableSceneCutDetection bool
}

// FindPython sucht einen funktionierenden Python-3-Interpreter. Windows hat
// üblicherweise "python" oder den "py"-Launcher statt "python3".
// pythonCandidates sammelt alle plausiblen Interpreter, in der Reihenfolge,
// in der sie probiert werden. Neben dem PATH werden die Standard-
// Installationspfade durchsucht: unter Windows landet Python dort auch dann,
// wenn beim Installieren das PATH-Häkchen vergessen wurde.
//
// Wichtig für Windows: %LOCALAPPDATA%\Microsoft\WindowsApps enthält
// Alias-Stubs für python.exe/python3.exe, die standardmäßig im PATH stehen.
// Diese Stubs sind entweder Weiterleitungen zum Microsoft Store oder zeigen
// auf eine ANDERE Python-Installation als die, in der der Nutzer seine
// Pakete installiert hat. Genau deshalb reicht "der erste Treffer im PATH"
// als Auswahlkriterium nicht aus - es muss geprüft werden, welcher
// Interpreter die benötigten Pakete tatsächlich hat.
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

	// Übliche Installationsorte, falls das PATH-Häkchen fehlt.
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
		// Rückwärts: höhere Versionsnummern zuerst.
		for i := len(matches) - 1; i >= 0; i-- {
			add(matches[i])
		}
	}
	return out
}

// hasPackages prüft, ob dieser Interpreter die benötigten Pakete importieren
// kann. Liefert die Ausgabe des Versuchs für die Fehlermeldung mit zurück.
func hasPackages(py string) (bool, string) {
	out, err := exec.Command(py, "-c", "import cv2, scipy, numpy").CombinedOutput()
	return err == nil, string(out)
}

// FindPython sucht einen startbaren Python-3-Interpreter. Bevorzugt wird
// einer, der die benötigten Pakete bereits hat - siehe pythonCandidates zu
// den Windows-Alias-Stubs. Gibt es keinen solchen, wird der erste startbare
// zurückgegeben (dann meldet CheckDependencies die fehlenden Pakete).
func FindPython() (string, error) {
	candidates := pythonCandidates()
	if len(candidates) == 0 {
		return "", fmt.Errorf("generator: kein Python gefunden (python3/python/py im PATH) - " +
			"bitte Python 3.9+ installieren: https://python.org (bei der Installation " +
			`"Add python.exe to PATH" anhaken)`)
	}
	var firstRunnable string
	for _, py := range candidates {
		ok, _ := hasPackages(py)
		if ok {
			return py, nil
		}
		if firstRunnable == "" {
			// Läuft der Interpreter überhaupt? Ein Store-Alias-Stub scheitert
			// hier bereits, ein echtes Python ohne Pakete nicht.
			if err := exec.Command(py, "-c", "pass").Run(); err == nil {
				firstRunnable = py
			}
		}
	}
	if firstRunnable != "" {
		return firstRunnable, nil
	}
	return "", fmt.Errorf("generator: kein funktionierender Python-Interpreter gefunden "+
		"(geprüft: %v) - bitte Python 3.9+ von https://python.org installieren und dabei "+
		`"Add python.exe to PATH" anhaken`, candidates)
}

// CheckDependencies prüft, ob Python UND die benötigten Pakete (opencv,
// scipy, numpy) verfügbar sind. Liefert bei Erfolg nil, sonst eine
// verständliche Fehlermeldung inkl. des passenden pip-Befehls - kein
// stilles Scheitern erst mitten in einer langen Videoverarbeitung.
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
	// Für den pip-Befehl den Interpreter vorschlagen, den die App auch
	// verwenden würde - sonst installiert der Nutzer erneut ins falsche
	// Python, was genau der Fehler ist, der hierher geführt hat.
	target, err := FindPython()
	if err != nil {
		return err
	}
	return fmt.Errorf(
		"generator: in keinem gefundenen Python sind die benötigten Pakete installiert.\n\n"+
			"Installieren mit GENAU diesem Befehl (der Pfad ist wichtig - auf diesem Rechner "+
			"gibt es mehrere Python-Installationen):\n  \"%s\" -m pip install %s\n\n"+
			"Geprüfte Interpreter:\n%s",
		target, "opencv-contrib-python scipy numpy", strings.Join(details, "\n"))
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

// writeScriptToTemp schreibt das eingebettete Hauptskript UND die Module,
// die es per "import" lädt (quality_doctor.py), in ein gemeinsames
// Temp-Verzeichnis unter ihrem echten Dateinamen. Ein einzelner
// os.CreateTemp()-Aufruf mit Zufallsnamen würde hier nicht reichen -
// Pythons "import quality_doctor" sucht die Datei im selben Verzeichnis
// wie das ausgeführte Skript, nicht irgendwo im System-Temp-Ordner.
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
		// Testdateien sind im Betrieb überflüssig und blähen nur das
		// Temp-Verzeichnis auf.
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

// cleanupScriptTemp entfernt das ganze Temp-Verzeichnis wieder (nicht nur
// die eine Datei) - siehe writeScriptToTemp.
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

// parseProgress erkennt die maschinenlesbaren Fortschrittszeilen der
// Python-Skripte ("PROGRESS <erledigt> <gesamt>"). Sie sollen NICHT als
// Statustext in der Oberfläche landen - sonst scrollt das Log mit hunderten
// Zeilen voll. total kann 0 sein, wenn die Frame-Anzahl nicht bekannt ist.
func parseProgress(line string) (done, total int, ok bool) {
	if !strings.HasPrefix(line, "PROGRESS ") {
		return 0, 0, false
	}
	if n, _ := fmt.Sscanf(line, "PROGRESS %d %d", &done, &total); n != 2 {
		return 0, 0, false
	}
	return done, total, true
}

// DefaultCacheDir liefert den Ort für zwischengespeicherte
// Trackingergebnisse. Muss mit default_cache_dir() in generate_funscript.py
// übereinstimmen, damit CLI und GUI denselben Cache nutzen.
func DefaultCacheDir() string {
	dir, err := os.UserCacheDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "SamNPlayer")
}

// percentOf rechnet den Fortschritt in Prozent um. -1 bedeutet "Gesamtlänge
// unbekannt" - die Oberfläche zeigt dann einen unbestimmten Balken statt
// eines erfundenen Werts.
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

// FindROI analysiert das Video automatisch und liefert die Region mit der
// stärksten rhythmischen Bewegung - siehe auto_roi.py für das Verfahren
// (Optical Flow + Kamerabewegungs-Kompensation + Periodizitäts-Bewertung).
// Der Nutzer muss dann nichts mehr von Hand markieren, kann das Ergebnis
// im Vorschaubild aber weiterhin korrigieren.
//
// onProgress bekommt Statuszeilen, onPercent den Fortschritt 0-100 (bzw. -1,
// wenn die Gesamtlänge unbekannt ist). Beide dürfen nil sein.
func FindROI(videoPath string, onProgress func(line string)) (ROI, error) {
	return FindROIWithProgress(videoPath, onProgress, nil)
}

// FindROIWithProgress ist FindROI mit zusätzlicher Fortschrittsmeldung.
func FindROIWithProgress(videoPath string, onProgress func(line string), onPercent func(pct int)) (ROI, error) {
	py, err := FindPython()
	if err != nil {
		return ROI{}, err
	}
	if err := CheckDependencies(); err != nil {
		return ROI{}, err
	}

	// Auch die Regionssuche läuft aus dem gemeinsamen Temp-Verzeichnis:
	// einzeln abgelegt fände sie ihre Nachbarmodule nicht. Das war derselbe
	// Fehler wie beim Hauptskript, nur an zweiter Stelle.
	mainScript, err := writeScriptToTemp()
	if err != nil {
		return ROI{}, err
	}
	defer cleanupScriptTemp(mainScript)
	scriptPath := filepath.Join(filepath.Dir(mainScript), "auto_roi.py")

	cmd := exec.Command(py, scriptPath, "--video", videoPath)
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

	// Fortschritts-/Fehlerzeilen kommen auf stderr, das Ergebnis auf stdout.
	go func() {
		sc := bufio.NewScanner(stderr)
		for sc.Scan() {
			line := sc.Text()
			if done, total, ok := parseProgress(line); ok {
				if onPercent != nil {
					onPercent(percentOf(done, total))
				}
				continue // nicht als Statustext weiterreichen
			}
			logging.Debug("auto_roi: " + line)
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

// DumpFirstFrame speichert den ersten Videoframe als PNG (für die
// ROI-Auswahl in der GUI) und gibt dessen Pixelgröße zurück.
func DumpFirstFrame(videoPath, outputPNG string) (width, height int, err error) {
	py, err := FindPython()
	if err != nil {
		return 0, 0, err
	}
	// Die Frame-Extraktion ist der erste Schritt nach der Videoauswahl und
	// damit der Pfad, den ein frisch eingerichteter Rechner zuerst trifft.
	// Ohne diese Prüfung bekäme der Nutzer hier einen rohen Python-Traceback
	// ("ModuleNotFoundError: No module named 'cv2'") statt der Meldung, die
	// den passenden pip-Befehl nennt - obwohl FindROI und Generate genau
	// diese Prüfung längst machen.
	if err := CheckDependencies(); err != nil {
		return 0, 0, err
	}
	scriptPath, err := writeScriptToTemp()
	if err != nil {
		return 0, 0, err
	}
	defer cleanupScriptTemp(scriptPath)

	cmd := exec.Command(py, scriptPath, "--video", videoPath, "--dump-first-frame", outputPNG)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return 0, 0, fmt.Errorf("generator: Frame-Extraktion fehlgeschlagen: %w\n%s", err, string(out))
	}
	// "FRAME_SIZE <w> <h>" aus stderr parsen (siehe dump_first_frame() im Skript).
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

// Generate erzeugt eine .funscript-Datei aus videoPath, verfolgt ab roi.
// onProgress wird pro Statuszeile aufgerufen (das Skript schreibt seinen
// Fortschritt nach stderr), darf nil sein. Blockiert bis das Skript fertig
// ist oder fehlschlägt.
func Generate(videoPath string, roi ROI, outputPath string, opts Options, onProgress func(line string)) error {
	return GenerateWithProgress(videoPath, roi, outputPath, opts, onProgress, nil)
}

// buildArgs setzt die Kommandozeile für generate_funscript.py zusammen.
// Bewusst als eigene Funktion, damit testbar ist, dass jede Option auch
// tatsächlich beim Skript ankommt - eine GUI-Checkbox, die nirgends landet,
// ist schlimmer als gar keine.
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
	if opts.Profile == "weich" {
		args = append(args, "--profile", "weich")
	}
	if opts.PeakProminence > 0 {
		args = append(args, "--peak-prominence",
			strconv.FormatFloat(opts.PeakProminence, 'f', -1, 64))
	}
	if opts.DynamicRangeMs > 0 {
		args = append(args, "--dynamic-range-ms",
			strconv.FormatFloat(opts.DynamicRangeMs, 'f', -1, 64))
	}
	if opts.MinActionIntervalMs > 0 {
		args = append(args, "--min-action-interval-ms",
			strconv.FormatFloat(opts.MinActionIntervalMs, 'f', -1, 64))
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
		args = append(args, "--adaptive-keyframes",
			strconv.FormatFloat(opts.AdaptiveKeyframeError, 'f', -1, 64))
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

// BuildArgsForTest macht buildArgs für Tests zugänglich.
func BuildArgsForTest(opts Options) []string {
	return buildArgs("script.py", "v.mp4", "o.funscript", ROI{}, opts)
}

// GenerateWithProgress ist Generate mit zusätzlicher Fortschrittsmeldung
// (0-100, oder -1 wenn die Frame-Anzahl des Videos nicht bekannt ist).
func GenerateWithProgress(videoPath string, roi ROI, outputPath string, opts Options,
	onProgress func(line string), onPercent func(pct int)) error {
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

	cmd := exec.Command(py, args...)
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
			continue // nicht als Statustext weiterreichen, sonst scrollt das Log voll
		}
		lastLines = append(lastLines, line)
		if len(lastLines) > 20 {
			lastLines = lastLines[1:]
		}
		// Die Statuszeilen des Skripts (Frame-Anzahl, Szenenschnitte,
		// Kamerakompensation, Quality-Doctor-Bericht) gingen bisher nur live
		// an die GUI und waren nach dem Schließen des Fensters weg. Bei einem
		// FEHLGESCHLAGENEN Lauf landeten die letzten Zeilen wenigstens in der
		// Fehlermeldung - ausgerechnet beim erfolgreichen Lauf mit
		// mittelmäßigem Score blieb nichts übrig. Als Debug, damit der
		// Normalbetrieb das Log nicht vollschreibt; Level ist im
		// Einstellungen-Tab umschaltbar.
		logging.Debug("generator: " + line)
		if onProgress != nil {
			onProgress(line)
		}
	}

	if err := cmd.Wait(); err != nil {
		logging.Error("generator: Generierung fehlgeschlagen", "fehler", err)
		return fmt.Errorf("generator: Generierung fehlgeschlagen: %w\nLetzte Ausgabe:\n%s",
			err, joinLines(lastLines))
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

// AddFeedback trägt ein Urteil zu einem erzeugten Skript in den Messbericht
// ein. Die Logik liegt im Python-Skript, damit Bericht und Auswertung nur an
// EINER Stelle definiert sind - eine zweite Implementierung in Go würde
// zwangsläufig auseinanderlaufen.
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

	args := []string{scriptPath, "--report", reportPath, "--output", outputPath,
		"--feedback", verdict}
	if comment != "" {
		args = append(args, comment)
	}
	out, err := exec.Command(py, args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("generator: Urteil konnte nicht gespeichert werden: %w\n%s",
			err, string(out))
	}
	return nil
}

// ReportSummary liefert die nach Urteil gruppierte Auswertung des Berichts.
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

	out, err := exec.Command(py, scriptPath, "--report", reportPath,
		"--report-summary").Output()
	if err != nil {
		return "", fmt.Errorf("generator: Bericht konnte nicht ausgewertet werden: %w", err)
	}
	return string(out), nil
}

// HardwareInfo meldet, welche Beschleunigung auf diesem Rechner verfügbar
// ist. Die Antwort ist nicht offensichtlich: eine vorhandene NVIDIA-Karte
// bedeutet NICHT, dass OpenCV sie nutzen kann - die üblichen pip-Pakete
// sind ohne CUDA gebaut.
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

	out, err := exec.Command(py, scriptPath, "--hardware-info").Output()
	if err != nil {
		return "", fmt.Errorf("generator: Hardware-Abfrage fehlgeschlagen: %w", err)
	}
	return string(out), nil
}

// TrainQualityModel lernt aus den beurteilten Läufen im Messbericht ein
// Qualitätsmodell.
//
// Es wird nur übernommen, wenn es die bestehenden festen Regeln in einer
// Kreuzvalidierung schlägt. Ein Modell, das schlechter ist als das, was
// schon da war, wäre ein Rückschritt mit dem Anschein von Fortschritt -
// und schwerer zu bemerken als eine schlechte Regel, weil es fundiert
// aussieht.
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

	cmd := exec.Command(py, scriptPath, "--report", reportPath, "--train-model")
	out, err := cmd.CombinedOutput()
	// Rückgabewert 2 bedeutet "nicht übernommen" - das ist kein Fehler,
	// sondern eine der vorgesehenen Antworten. Der Bericht erklärt warum.
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 2 {
			return string(out), nil
		}
		return string(out), fmt.Errorf("generator: Lernen fehlgeschlagen: %w", err)
	}
	return string(out), nil
}

// QualityModelInfo beschreibt das aktuell gültige Modell.
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
	out, err := exec.Command(py, scriptPath, "--model-info").Output()
	if err != nil {
		return "", fmt.Errorf("generator: Modellabfrage fehlgeschlagen: %w", err)
	}
	return string(out), nil
}
