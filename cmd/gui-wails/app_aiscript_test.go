package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/funfunpayer/SamNPlayer/funscript"
	"github.com/funfunpayer/SamNPlayer/generator"
)

func TestExportAIScriptImitation(t *testing.T) {
	dir := t.TempDir()
	// Point ROI dataset root at temp via env if supported — otherwise write
	// under DefaultRoiDatasetDir; force dir by writing script only and
	// accepting export next to default. Prefer isolating via chdir of output:
	scriptPath := filepath.Join(dir, "clip.funscript")
	body := []byte(`{"actions":[{"at":0,"pos":10},{"at":500,"pos":90},{"at":1000,"pos":20}],"metadata":{"quality_passed":true,"quality_score":0.8}}`)
	if err := os.WriteFile(scriptPath, body, 0o644); err != nil {
		t.Fatal(err)
	}
	// Use a dedicated settings-less App; export uses DefaultRoiDatasetDir.
	_ = generator.DefaultRoiDatasetDir()
	a := &App{}
	res, err := a.ExportAIScriptImitation(scriptPath, filepath.Join(dir, "clip.mp4"), 10, 20, 80, 90)
	if err != nil {
		t.Fatal(err)
	}
	if res.Path == "" {
		t.Fatal("empty path")
	}
	if _, err := os.Stat(res.Path); err != nil {
		t.Fatal(err)
	}
	// Sanity: loadable JSON actions count + tip ROI seed
	raw, err := os.ReadFile(res.Path)
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) < 20 {
		t.Fatalf("short file: %s", raw)
	}
	if !bytes.Contains(raw, []byte(`"tipX": 10`)) && !bytes.Contains(raw, []byte(`"tipX":10`)) {
		t.Fatalf("expected tipX in sample: %s", raw)
	}
	_ = funscript.Action{}
}
