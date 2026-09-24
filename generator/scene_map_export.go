package generator

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/funfunpayer/SamNPlayer/funscript"
	"github.com/funfunpayer/SamNPlayer/samn"
)

// SceneMapLearningSubdir is the only tree cleared by DeleteSceneMapLearningData.
// It sits under the ROI dataset root so hand YOLO bootstrap samples stay safe.
const SceneMapLearningSubdir = "scene_map_learning"

// ErrLearningExportNotOptedIn is returned when export is refused without
// --opt-in / Collect learning data (Owner: default OFF).
var ErrLearningExportNotOptedIn = fmt.Errorf("generator: learning export requires explicit opt-in (Collect learning data / --opt-in)")

// LearningExportOptions controls SceneMap P5a local export (L0 collect only).
type LearningExportOptions struct {
	// OptIn must be true — Settings CollectLearningData or CLI --opt-in.
	OptIn bool
	// OutputDir defaults to DefaultRoiDatasetDir()/scene_map_learning.
	OutputDir string
}

// LearningExportResult is a short summary of what was written.
type LearningExportResult struct {
	OutDir          string
	Windows         int
	Negatives       int
	AutoCandidates  int
	UserRegionMarks int
	TracePath       string
	NegativesPath   string
	AutoPath        string
	UserRegionsPath string
}

// DefaultSceneMapLearningDir is …/roi_training_dataset/scene_map_learning.
func DefaultSceneMapLearningDir() string {
	root := DefaultRoiDatasetDir()
	if root == "" {
		return ""
	}
	return filepath.Join(root, SceneMapLearningSubdir)
}

// ExportSceneMapLearning walks one .samn with sceneMap and writes local
// JSON/JSONL artifacts. Never writes images/train or labels/train.
// Auto candidates are always reviewed:false.
func ExportSceneMapLearning(samnPath string, opts LearningExportOptions) (LearningExportResult, error) {
	var out LearningExportResult
	if !opts.OptIn {
		return out, ErrLearningExportNotOptedIn
	}
	doc, err := samn.Load(samnPath)
	if err != nil {
		return out, err
	}
	if doc.SceneMap == nil || !doc.SceneMap.HasWindows() {
		return out, fmt.Errorf("generator: %s has no sceneMap windows", samnPath)
	}
	outDir := opts.OutputDir
	if outDir == "" {
		outDir = DefaultSceneMapLearningDir()
	}
	if outDir == "" {
		return out, fmt.Errorf("generator: no learning output directory")
	}
	clipKey := learningClipKey(samnPath, doc.SceneMap)
	clipDir := filepath.Join(outDir, clipKey)
	if err := os.MkdirAll(clipDir, 0o755); err != nil {
		return out, err
	}

	tracePath := filepath.Join(clipDir, "engine_trace.jsonl")
	if err := writeEngineTrace(tracePath, doc.SceneMap); err != nil {
		return out, err
	}
	negPath := filepath.Join(clipDir, "negatives.json")
	nNeg, err := writeNegatives(negPath, doc.SceneMap.Marks)
	if err != nil {
		return out, err
	}
	autoPath := filepath.Join(clipDir, "auto_candidates.jsonl")
	nAuto, err := writeAutoCandidates(autoPath, doc.SceneMap)
	if err != nil {
		return out, err
	}
	userPath := filepath.Join(clipDir, "user_region_marks.json")
	nUser, err := writeUserRegionMarks(userPath, doc.SceneMap.Marks)
	if err != nil {
		return out, err
	}

	metaPath := filepath.Join(clipDir, "source.json")
	_ = writeJSON(metaPath, map[string]any{
		"samn":        filepath.Base(samnPath),
		"sha256_head": doc.SceneMap.Video.Sha256Head,
		"grid":        doc.SceneMap.Grid,
		"video":       doc.SceneMap.Video,
		"export":      "scene_map_learning_p5a",
		"note":        "auto_candidates are author=auto reviewed=false — not for YOLO train until reviewed",
	})

	out = LearningExportResult{
		OutDir:          clipDir,
		Windows:         len(doc.SceneMap.Windows),
		Negatives:       nNeg,
		AutoCandidates:  nAuto,
		UserRegionMarks: nUser,
		TracePath:       tracePath,
		NegativesPath:   negPath,
		AutoPath:        autoPath,
		UserRegionsPath: userPath,
	}
	return out, nil
}

// DeleteSceneMapLearningData removes only scene_map_learning under the
// dataset root (or OutputDir if it ends with that subdir name).
func DeleteSceneMapLearningData(datasetRoot string) error {
	if datasetRoot == "" {
		datasetRoot = DefaultRoiDatasetDir()
	}
	if datasetRoot == "" {
		return fmt.Errorf("generator: no dataset root")
	}
	target := filepath.Join(datasetRoot, SceneMapLearningSubdir)
	if err := os.RemoveAll(target); err != nil {
		return err
	}
	return nil
}

func learningClipKey(samnPath string, m *funscript.SceneMapData) string {
	base := strings.TrimSuffix(filepath.Base(samnPath), filepath.Ext(samnPath))
	if m != nil && m.Video.Sha256Head != "" && len(m.Video.Sha256Head) >= 12 {
		return base + "_" + m.Video.Sha256Head[:12]
	}
	return base
}

func writeEngineTrace(path string, m *funscript.SceneMapData) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	cols, rows := m.Grid.Cols, m.Grid.Rows
	if cols <= 0 {
		cols = 16
	}
	cellW, cellH := 0.0, 0.0
	if m.Video.Width > 0 && cols > 0 {
		cellW = float64(m.Video.Width) / float64(cols)
	}
	if m.Video.Height > 0 && rows > 0 {
		cellH = float64(m.Video.Height) / float64(rows)
	}
	enc := json.NewEncoder(f)
	for _, w := range m.Windows {
		line := map[string]any{
			"startMs":   w.StartMs,
			"endMs":     w.EndMs,
			"tempoHz":   w.TempoHz,
			"chosen":    w.Chosen,
			"signRule":  w.SignRule,
			"trackerR":  w.TrackerR,
			"box":       w.Box,
			"marks":     w.Marks,
			"score_b64": w.ScoreB64,
		}
		if len(w.Box) >= 2 && w.Chosen >= 0 && cellW > 0 && cellH > 0 {
			cx := (float64(w.Chosen%cols) + 0.5) * cellW
			cy := (float64(w.Chosen/cols) + 0.5) * cellH
			line["chosenCellDistPx"] = math.Hypot(cx-w.Box[0], cy-w.Box[1])
		}
		if err := enc.Encode(line); err != nil {
			return err
		}
	}
	return nil
}

func writeNegatives(path string, marks []funscript.SceneMapMark) (int, error) {
	var negs []map[string]any
	for _, m := range marks {
		if strings.ToLower(m.Kind) != "exclude" {
			continue
		}
		negs = append(negs, map[string]any{
			"id":     m.ID,
			"kind":   m.Kind,
			"rect":   m.Rect,
			"fromMs": m.FromMs,
			"toMs":   m.ToMs,
			"author": m.Author,
		})
	}
	if negs == nil {
		negs = []map[string]any{}
	}
	return len(negs), writeJSON(path, map[string]any{
		"version": 1,
		"role":    "not_stroke_source",
		"marks":   negs,
	})
}

func writeUserRegionMarks(path string, marks []funscript.SceneMapMark) (int, error) {
	var regs []map[string]any
	for _, m := range marks {
		if strings.ToLower(m.Kind) != "region" {
			continue
		}
		if strings.EqualFold(m.Author, "auto") {
			continue
		}
		regs = append(regs, map[string]any{
			"id":     m.ID,
			"kind":   m.Kind,
			"class":  m.Class,
			"role":   m.Role,
			"rect":   m.Rect,
			"fromMs": m.FromMs,
			"toMs":   m.ToMs,
			"author": m.Author,
		})
	}
	if regs == nil {
		regs = []map[string]any{}
	}
	return len(regs), writeJSON(path, map[string]any{"version": 1, "marks": regs})
}

// writeAutoCandidates applies the M5 agreement rule on persisted windows only:
// box centre within one cell of chosen, |trackerR|≥0.3, ≥3 consecutive windows.
// Always reviewed:false — never YOLO train set in P5a.
func writeAutoCandidates(path string, m *funscript.SceneMapData) (int, error) {
	cols, rows := m.Grid.Cols, m.Grid.Rows
	if cols <= 0 {
		cols = 16
	}
	cellW, cellH := 1.0, 1.0
	if m.Video.Width > 0 {
		cellW = float64(m.Video.Width) / float64(cols)
	}
	if m.Video.Height > 0 && rows > 0 {
		cellH = float64(m.Video.Height) / float64(rows)
	}

	agree := make([]bool, len(m.Windows))
	for i, w := range m.Windows {
		if w.Chosen < 0 || len(w.Box) < 2 || math.Abs(w.TrackerR) < 0.3 {
			continue
		}
		cx := (float64(w.Chosen%cols) + 0.5) * cellW
		cy := (float64(w.Chosen/cols) + 0.5) * cellH
		if math.Abs(cx-w.Box[0]) <= cellW && math.Abs(cy-w.Box[1]) <= cellH {
			agree[i] = true
		}
	}

	reviewed := false
	f, err := os.Create(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	n := 0
	run := 0
	for i := 0; i <= len(agree); i++ {
		ok := i < len(agree) && agree[i]
		if ok {
			run++
			continue
		}
		if run >= 3 {
			for j := i - run; j < i; j++ {
				w := m.Windows[j]
				rec := map[string]any{
					"author":     "auto",
					"reviewed":   reviewed,
					"confidence": math.Abs(w.TrackerR),
					"startMs":    w.StartMs,
					"endMs":      w.EndMs,
					"chosen":     w.Chosen,
					"box":        w.Box,
					"class":      "glans",
					"note":       "CSRT box agrees with chosen cell — review before training",
				}
				if err := enc.Encode(rec); err != nil {
					return n, err
				}
				n++
			}
		}
		run = 0
	}
	return n, nil
}

func writeJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}
