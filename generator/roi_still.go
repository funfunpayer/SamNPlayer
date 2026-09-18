package generator

import (
	"encoding/json"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// AddStillTrainingSample writes one labeled still image into the YOLO
// dataset (train split). Used for learning from photos without video
// tracking — marks transfer later via clip bootstrap.
func AddStillTrainingSample(imagePath string, regions []RoiTrainingRegion, outputDir, samplePrefix string) (string, error) {
	if len(regions) == 0 {
		return "", fmt.Errorf("generator: mindestens eine Region nötig")
	}
	if strings.TrimSpace(outputDir) == "" {
		return "", fmt.Errorf("generator: outputDir leer")
	}
	f, err := os.Open(imagePath)
	if err != nil {
		return "", err
	}
	defer f.Close()
	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		return "", fmt.Errorf("generator: Bild lesen: %w", err)
	}
	w, h := cfg.Width, cfg.Height
	if w < 8 || h < 8 {
		return "", fmt.Errorf("generator: Bild zu klein (%dx%d)", w, h)
	}

	if err := os.MkdirAll(filepath.Join(outputDir, "images", "train"), 0o755); err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Join(outputDir, "labels", "train"), 0o755); err != nil {
		return "", err
	}

	registry, err := loadClassRegistry(outputDir)
	if err != nil {
		return "", err
	}
	var lines []string
	for _, r := range regions {
		name := strings.TrimSpace(r.ClassName)
		if name == "" {
			return "", fmt.Errorf("generator: Klassenname fehlt")
		}
		id := assignClassID(registry, name)
		xc := (float64(r.ROI.X) + float64(r.ROI.W)/2) / float64(w)
		yc := (float64(r.ROI.Y) + float64(r.ROI.H)/2) / float64(h)
		nw := float64(r.ROI.W) / float64(w)
		nh := float64(r.ROI.H) / float64(h)
		if nw <= 0 || nh <= 0 {
			continue
		}
		lines = append(lines, fmt.Sprintf("%d %.6f %.6f %.6f %.6f", id, xc, yc, nw, nh))
	}
	if len(lines) == 0 {
		return "", fmt.Errorf("generator: keine gültigen Boxen")
	}
	if err := saveClassRegistry(outputDir, registry); err != nil {
		return "", err
	}

	prefix := samplePrefix
	if prefix == "" {
		base := strings.TrimSuffix(filepath.Base(imagePath), filepath.Ext(imagePath))
		prefix = "still_" + sanitizeStem(base)
	}
	stem := fmt.Sprintf("%s_000000", prefix)
	dstImg := filepath.Join(outputDir, "images", "train", stem+filepath.Ext(imagePath))
	if ext := strings.ToLower(filepath.Ext(dstImg)); ext != ".jpg" && ext != ".jpeg" && ext != ".png" {
		dstImg = filepath.Join(outputDir, "images", "train", stem+".jpg")
	}
	if err := copyFile(imagePath, dstImg); err != nil {
		return "", err
	}
	lbl := filepath.Join(outputDir, "labels", "train", stem+".txt")
	if err := os.WriteFile(lbl, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		return "", err
	}
	_ = writeDataYAMLFromRegistry(outputDir, registry)
	return prefix, nil
}

// ExtractTrainingAudio dumps the clip audio next to the dataset so later
// multimodal training can use it. Best-effort: missing audio is not an error.
func ExtractTrainingAudio(videoPath, outputDir, samplePrefix string) (string, error) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return "", fmt.Errorf("ffmpeg nicht gefunden")
	}
	audioDir := filepath.Join(outputDir, "audio")
	if err := os.MkdirAll(audioDir, 0o755); err != nil {
		return "", err
	}
	prefix := samplePrefix
	if prefix == "" {
		prefix = sanitizeStem(filepath.Base(videoPath))
	}
	out := filepath.Join(audioDir, prefix+".wav")
	args := []string{"-v", "error", "-y", "-i", videoPath, "-vn", "-ac", "1", "-ar", "16000", out}
	if b, err := exec.Command("ffmpeg", args...).CombinedOutput(); err != nil {
		return "", fmt.Errorf("audio-extraktion: %w\n%s", err, string(b))
	}
	return out, nil
}

func sanitizeStem(s string) string {
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else {
			b.WriteRune('_')
		}
	}
	out := b.String()
	if len(out) > 40 {
		out = out[:40]
	}
	if out == "" {
		out = "sample"
	}
	return out
}

func loadClassRegistry(dir string) (map[string]int, error) {
	path := filepath.Join(dir, "classes.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]int{}, nil
		}
		return nil, err
	}
	var reg map[string]int
	if err := json.Unmarshal(data, &reg); err != nil {
		return nil, err
	}
	if reg == nil {
		reg = map[string]int{}
	}
	return reg, nil
}

func saveClassRegistry(dir string, reg map[string]int) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(reg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "classes.json"), data, 0o644)
}

func assignClassID(reg map[string]int, name string) int {
	if id, ok := reg[name]; ok {
		return id
	}
	next := 0
	for _, id := range reg {
		if id >= next {
			next = id + 1
		}
	}
	reg[name] = next
	return next
}

func writeDataYAMLFromRegistry(dir string, reg map[string]int) error {
	type pair struct {
		name string
		id   int
	}
	var names []pair
	for n, id := range reg {
		names = append(names, pair{n, id})
	}
	for i := 0; i < len(names); i++ {
		for j := i + 1; j < len(names); j++ {
			if names[j].id < names[i].id {
				names[i], names[j] = names[j], names[i]
			}
		}
	}
	var b strings.Builder
	b.WriteString("path: ")
	abs, _ := filepath.Abs(dir)
	b.WriteString(abs)
	b.WriteString("\ntrain: images/train\nval: images/val\nnames:\n")
	for _, p := range names {
		b.WriteString(fmt.Sprintf("  %d: %s\n", p.id, p.name))
	}
	return os.WriteFile(filepath.Join(dir, "data.yaml"), []byte(b.String()), 0o644)
}

func copyFile(src, dst string) error {
	in, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, in, 0o644)
}
