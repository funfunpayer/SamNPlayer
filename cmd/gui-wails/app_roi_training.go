package main

import (
	"bufio"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/funfunpayer/SamNPlayer/generator"
	"github.com/funfunpayer/SamNPlayer/logging"
)

// Eigener Namensraum "Roi..." statt "Training...": "Training" ist im Rest
// der App schon für Geräte-Trainingssitzungen (Stop-Start/Plateau, siehe
// app_training.go) belegt - hier geht es um das Trainieren eines
// KI-Regionserkennungsmodells, etwas völlig anderes.

// roiTrainingRunning verhindert einen zweiten gleichzeitigen Bootstrap-
// oder Trainingslauf - beide sind für Sekunden bis (beim Training) Stunden
// blockierend und schreiben in denselben Datensatzordner.
var roiTrainingRunning bool

func claimRoiTrainingRun() error {
	if roiTrainingRunning {
		return fmt.Errorf("es läuft bereits ein Bootstrap- oder Trainingslauf")
	}
	roiTrainingRunning = true
	return nil
}

func releaseRoiTrainingRun() {
	roiTrainingRunning = false
}

// roiTrainingSamplePrefix erzeugt einen stabilen, dateinamentauglichen
// Präfix je AUFRUF (nicht nur je Video) - anders als Pythons interner
// _video_stem() (rein vom Videopfad abgeleitet, kollidiert bei einem
// zweiten Bootstrap-Lauf auf demselben Video mit sample_idx wieder bei 0).
// Die Zeitkomponente macht jeden Lauf eindeutig UND lässt die
// Kontrollansicht (ListRoiTrainingSamples) exakt auf "gerade eben
// hinzugekommen" eingrenzen, nicht auf "irgendwann für dieses Video".
func roiTrainingSamplePrefix(videoPath string) string {
	base := strings.TrimSuffix(filepath.Base(videoPath), filepath.Ext(videoPath))
	var safe strings.Builder
	for _, r := range base {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			safe.WriteRune(r)
		} else {
			safe.WriteRune('_')
		}
	}
	safeName := safe.String()
	if len(safeName) > 40 {
		safeName = safeName[:40]
	}
	h := sha1.Sum([]byte(videoPath + time.Now().Format(time.RFC3339Nano)))
	return fmt.Sprintf("%s_%x", safeName, h[:5])
}

// CheckRoiTrainingAvailable meldet, ob das eigentliche Trainieren (nicht das
// Sammeln von Trainingsdaten - das braucht nur die Basis-Installation)
// grundsätzlich nutzbar ist, also ob ultralytics installiert ist. Die GUI
// nutzt das, um den "Training starten"-Knopf zu aktivieren/auszublenden
// statt ihn anzubieten und dann mit einem rohen Python-Traceback scheitern
// zu lassen (siehe CheckAIRoiAvailable für dasselbe Muster).
func (a *App) CheckRoiTrainingAvailable() bool {
	return generator.RoiTrainingAvailable()
}

// BootstrapRoiTrainingSample trackt roi (und optional roi2) durchs Video und
// hängt die Ergebnisse an den Datensatz an - siehe
// generator.BootstrapRoiTrainingSample/bootstrap_yolo_dataset.py. Läuft
// asynchron (echtes Tracking, kann bei --backend csrt auf einem langen
// Video Minuten dauern), Events "roitraining:bootstrap:progress"/"...:done".
// Liefert den Sample-Präfix sofort zurück (schon vor Abschluss bekannt,
// siehe roiTrainingSamplePrefix), damit die GUI ihn für die anschließende
// Kontrollansicht bereithält, ohne auf das Event warten zu müssen.
func (a *App) BootstrapRoiTrainingSample(videoPath string, roi, roi2 *generator.ROI, className, className2 string) (string, error) {
	if err := claimRoiTrainingRun(); err != nil {
		return "", err
	}
	if roi == nil {
		releaseRoiTrainingRun()
		return "", fmt.Errorf("mindestens eine Region (ROI1) ist nötig")
	}
	if roi2 != nil && strings.TrimSpace(className2) == "" {
		releaseRoiTrainingRun()
		return "", fmt.Errorf("2. Region braucht einen Klassennamen")
	}

	datasetDir := a.settings.GetString(prefRoiDatasetDir, generator.DefaultRoiDatasetDir())
	prefix := roiTrainingSamplePrefix(videoPath)

	regions := []generator.RoiTrainingRegion{{ROI: *roi, ClassName: className}}
	if roi2 != nil {
		regions = append(regions, generator.RoiTrainingRegion{ROI: *roi2, ClassName: className2})
	}

	go func() {
		defer releaseRoiTrainingRun()
		err := generator.BootstrapRoiTrainingSample(videoPath, regions, datasetDir, prefix,
			func(line string) { runtime.EventsEmit(a.ctx, "roitraining:bootstrap:progress", line) })
		if err != nil {
			logging.Error("roitraining: Bootstrap fehlgeschlagen", "video", videoPath, "fehler", err)
			runtime.EventsEmit(a.ctx, "roitraining:bootstrap:done", map[string]any{"error": err.Error()})
			return
		}
		logging.Info("roitraining: Bootstrap abgeschlossen", "video", videoPath, "prefix", prefix)
		runtime.EventsEmit(a.ctx, "roitraining:bootstrap:done", map[string]any{"prefix": prefix})
	}()

	return prefix, nil
}

// RunRoiModelTraining trainiert ein Modell auf dem gesammelten Datensatz und
// legt es unter dem KI-Modellpfad der Einstellungen ab (derselbe Pfad, den
// ai_roi.default_model_path()/CheckAIRoiAvailable() prüfen) - nach Erfolg
// ist die KI-Erkennung im Generator-Tab also ohne weiteren Schritt nutzbar.
// Läuft asynchron, kann bei einem echten Datensatz lange dauern.
func (a *App) RunRoiModelTraining(epochs int, device string) error {
	if err := claimRoiTrainingRun(); err != nil {
		return err
	}

	datasetDir := a.settings.GetString(prefRoiDatasetDir, generator.DefaultRoiDatasetDir())
	modelPath := a.settings.GetString(prefAIRoiModelPath, "")
	if modelPath == "" {
		modelPath = defaultAIRoiModelPath()
	}

	go func() {
		defer releaseRoiTrainingRun()
		if err := os.MkdirAll(filepath.Dir(modelPath), 0o755); err != nil {
			runtime.EventsEmit(a.ctx, "roitraining:train:done", map[string]any{"error": err.Error()})
			return
		}
		err := generator.RunRoiModelTraining(datasetDir, modelPath, epochs, device,
			func(line string) { runtime.EventsEmit(a.ctx, "roitraining:train:progress", line) })
		if err != nil {
			logging.Error("roitraining: Training fehlgeschlagen", "fehler", err)
			runtime.EventsEmit(a.ctx, "roitraining:train:done", map[string]any{"error": err.Error()})
			return
		}
		logging.Info("roitraining: Training abgeschlossen", "modell", modelPath)
		runtime.EventsEmit(a.ctx, "roitraining:train:done", map[string]any{"modelPath": modelPath})
	}()

	return nil
}

// RoiTrainingBox ist eine einzelne markierte Region in einem Trainingsbild -
// normalisierte Koordinaten (0..1), genau das YOLO-Labelformat.
type RoiTrainingBox struct {
	ClassID   int     `json:"classId"`
	ClassName string  `json:"className"`
	XC        float64 `json:"xc"`
	YC        float64 `json:"yc"`
	W         float64 `json:"w"`
	H         float64 `json:"h"`
}

// RoiTrainingSample ist ein Bootstrap-Beispiel für die Kontrollansicht.
type RoiTrainingSample struct {
	Split     string           `json:"split"`
	Name      string           `json:"name"`
	ImagePath string           `json:"imagePath"`
	Boxes     []RoiTrainingBox `json:"boxes"`
}

// roiTrainingClassNames liest classes.json (Name -> ID, siehe
// bootstrap_yolo_dataset.py register_class) und dreht die Zuordnung um, für
// die Anzeige in der Kontrollansicht. Eine fehlende Datei ist kein Fehler -
// vor dem ersten Bootstrap-Lauf existiert sie schlicht noch nicht.
func roiTrainingClassNames(datasetDir string) (map[int]string, error) {
	path := filepath.Join(datasetDir, "classes.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[int]string{}, nil
		}
		return nil, err
	}
	var registry map[string]int
	if err := json.Unmarshal(data, &registry); err != nil {
		return nil, fmt.Errorf("classes.json ungültig: %w", err)
	}
	names := make(map[int]string, len(registry))
	for name, id := range registry {
		names[id] = name
	}
	return names, nil
}

// readRoiLabelFile parst eine YOLO-Labeldatei ("class xc yc w h" je Zeile) -
// genau dasselbe simple Format wie bootstrap_yolo_dataset.py's
// read_label_file/write_label_file, hier direkt in Go statt über einen
// Python-Unterprozess (reine Textdatei, kein cv2/numpy nötig - analog zu
// GetBenchmarkHistory's JSONL-Lesen ohne Python-Umweg).
func readRoiLabelFile(path string, classNames map[int]string) ([]RoiTrainingBox, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var boxes []RoiTrainingBox
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) != 5 {
			continue
		}
		classID, err := strconv.Atoi(fields[0])
		if err != nil {
			continue
		}
		vals := make([]float64, 4)
		ok := true
		for i, s := range fields[1:] {
			v, err := strconv.ParseFloat(s, 64)
			if err != nil {
				ok = false
				break
			}
			vals[i] = v
		}
		if !ok {
			continue
		}
		boxes = append(boxes, RoiTrainingBox{
			ClassID: classID, ClassName: classNames[classID],
			XC: vals[0], YC: vals[1], W: vals[2], H: vals[3],
		})
	}
	return boxes, scanner.Err()
}

func writeRoiLabelFile(path string, boxes []RoiTrainingBox) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	var sb strings.Builder
	for _, b := range boxes {
		fmt.Fprintf(&sb, "%d %.6f %.6f %.6f %.6f\n", b.ClassID, b.XC, b.YC, b.W, b.H)
	}
	return os.WriteFile(path, []byte(sb.String()), 0o644)
}

func roiLabelPathForImage(imagePath string) (string, error) {
	parts := strings.Split(filepath.ToSlash(imagePath), "/")
	idx := -1
	for i, p := range parts {
		if p == "images" {
			idx = i
		}
	}
	if idx == -1 {
		return "", fmt.Errorf("kein 'images'-Ordner im Pfad: %s", imagePath)
	}
	parts[idx] = "labels"
	joined := filepath.Join(parts...)
	ext := filepath.Ext(joined)
	return strings.TrimSuffix(joined, ext) + ".txt", nil
}

// ListRoiTrainingSamples listet Trainingsbeispiele aus datasetDir/images/*
// mit prefix im Dateinamen (siehe roiTrainingSamplePrefix) - die
// Kontrollansicht nach einem Bootstrap-Lauf. Läuft synchron: reines
// Verzeichnislisten + Textdateien lesen, kein Unterprozess nötig.
func (a *App) ListRoiTrainingSamples(datasetDir, prefix string) ([]RoiTrainingSample, error) {
	classNames, err := roiTrainingClassNames(datasetDir)
	if err != nil {
		return nil, err
	}
	var samples []RoiTrainingSample
	for _, split := range []string{"train", "val"} {
		imgDir := filepath.Join(datasetDir, "images", split)
		entries, err := os.ReadDir(imgDir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			name := e.Name()
			lower := strings.ToLower(name)
			if !strings.HasSuffix(lower, ".jpg") && !strings.HasSuffix(lower, ".jpeg") && !strings.HasSuffix(lower, ".png") {
				continue
			}
			if prefix != "" && !strings.HasPrefix(name, prefix) {
				continue
			}
			imagePath := filepath.Join(imgDir, name)
			labelPath, err := roiLabelPathForImage(imagePath)
			if err != nil {
				continue
			}
			boxes, err := readRoiLabelFile(labelPath, classNames)
			if err != nil {
				return nil, err
			}
			samples = append(samples, RoiTrainingSample{
				Split: split, Name: name, ImagePath: imagePath, Boxes: boxes,
			})
		}
	}
	sort.Slice(samples, func(i, j int) bool { return samples[i].Name < samples[j].Name })
	return samples, nil
}

// GetRoiTrainingSampleImage liefert ein Trainingsbild base64-kodiert - roh
// als file:// eingebunden lehnt die Wails-Webview <img src> ab (siehe
// LoadFirstFrame in app_generator.go, dieselbe Einschränkung), deshalb geht
// jedes Bild für die Kontrollansicht denselben Weg über die RPC-Brücke.
func (a *App) GetRoiTrainingSampleImage(imagePath string) (string, error) {
	return fileToBase64(imagePath)
}

// UpdateRoiTrainingSample überschreibt die Boxen eines Beispiels - für
// Korrekturen aus der Kontrollansicht (falsch/ungenau getrackte Box von
// Hand richten, statt sie unkontrolliert ins Training gehen zu lassen).
func (a *App) UpdateRoiTrainingSample(datasetDir, split, name string, boxes []RoiTrainingBox) error {
	imagePath := filepath.Join(datasetDir, "images", split, name)
	labelPath, err := roiLabelPathForImage(imagePath)
	if err != nil {
		return err
	}
	return writeRoiLabelFile(labelPath, boxes)
}

// DiscardRoiTrainingSample entfernt ein Beispiel vollständig (Bild +
// Labeldatei) - für Frames, die die Kontrollansicht als falsch verwirft.
func (a *App) DiscardRoiTrainingSample(datasetDir, split, name string) error {
	imagePath := filepath.Join(datasetDir, "images", split, name)
	labelPath, err := roiLabelPathForImage(imagePath)
	if err != nil {
		return err
	}
	for _, p := range []string{imagePath, labelPath} {
		if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

// RoiClassCount ist die Anzahl Beispiele je Klasse und Split - für die
// Datensatz-Übersicht.
type RoiClassCount struct {
	ClassName  string `json:"className"`
	ClassID    int    `json:"classId"`
	TrainCount int    `json:"trainCount"`
	ValCount   int    `json:"valCount"`
}

// RoiDatasetSummary fasst den aktuellen Trainingsdatensatz zusammen.
type RoiDatasetSummary struct {
	DatasetDir string          `json:"datasetDir"`
	Classes    []RoiClassCount `json:"classes"`
}

// GetRoiDatasetSummary liest classes.json und zählt, wie oft jede Klasse in
// den Labeldateien je Split vorkommt - Grundlage der "Datensatz-Übersicht"
// in der Trainings-Oberfläche (siehe docs für den vollen Ablauf).
func (a *App) GetRoiDatasetSummary(datasetDir string) (RoiDatasetSummary, error) {
	summary := RoiDatasetSummary{DatasetDir: datasetDir}
	classNames, err := roiTrainingClassNames(datasetDir)
	if err != nil {
		return summary, err
	}
	counts := make(map[int]*RoiClassCount, len(classNames))
	for id, name := range classNames {
		counts[id] = &RoiClassCount{ClassName: name, ClassID: id}
	}
	for _, split := range []string{"train", "val"} {
		lblDir := filepath.Join(datasetDir, "labels", split)
		entries, err := os.ReadDir(lblDir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return summary, err
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".txt") {
				continue
			}
			boxes, err := readRoiLabelFile(filepath.Join(lblDir, e.Name()), classNames)
			if err != nil {
				return summary, err
			}
			for _, b := range boxes {
				c, ok := counts[b.ClassID]
				if !ok {
					c = &RoiClassCount{ClassName: b.ClassName, ClassID: b.ClassID}
					counts[b.ClassID] = c
				}
				if split == "train" {
					c.TrainCount++
				} else {
					c.ValCount++
				}
			}
		}
	}
	for _, c := range counts {
		summary.Classes = append(summary.Classes, *c)
	}
	sort.Slice(summary.Classes, func(i, j int) bool { return summary.Classes[i].ClassID < summary.Classes[j].ClassID })
	return summary, nil
}
