package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/funfunpayer/SamNPlayer/funscript"
	"github.com/funfunpayer/SamNPlayer/sam"
	"github.com/funfunpayer/SamNPlayer/samn"
)

func TestImproveGeneratedScriptHealTrackingGaps(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "clip.funscript")
	body := map[string]any{
		"actions": []map[string]any{
			{"at": 0, "pos": 0},
			{"at": 100, "pos": 10},
			{"at": 900, "pos": 99},
			{"at": 1100, "pos": 98},
			{"at": 2000, "pos": 90},
			{"at": 2100, "pos": 100},
		},
		"metadata": map[string]any{
			"creator": "test",
			"tracking_gaps": []map[string]any{
				{"start_ms": 500, "end_ms": 1600},
			},
		},
	}
	raw, _ := json.Marshal(body)
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}

	a := &App{}
	res, err := a.ImproveGeneratedScript(ImproveScriptRequest{
		Path:             path,
		HealTrackingGaps: true,
		FillGaps:         false,
		AudioCheck:       false,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.WindowsHealed < 1 {
		t.Fatalf("expected heal, got %+v", res)
	}
	script, err := funscript.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(script.Metadata.TrackingGaps) != 0 {
		t.Fatalf("tracking_gaps not cleared: %+v", script.Metadata.TrackingGaps)
	}
	for _, act := range script.Actions {
		if act.At > 500 && act.At < 1600 && (act.Pos == 99 || act.Pos == 98) {
			t.Fatalf("junk survived: %+v", act)
		}
	}
}

func TestImproveGeneratedScriptFillGaps(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "clip.funscript")
	body := map[string]any{
		"actions": []map[string]any{
			{"at": 0, "pos": 0},
			{"at": 100, "pos": 10},
			{"at": 5000, "pos": 90},
			{"at": 5100, "pos": 100},
		},
		"metadata": map[string]any{"creator": "test"},
	}
	raw, _ := json.Marshal(body)
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}

	a := &App{}
	res, err := a.ImproveGeneratedScript(ImproveScriptRequest{
		Path:       path,
		FillGaps:   true,
		MaxGapMs:   800,
		AudioCheck: false,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.GapsFilled < 1 || res.PointsAdded < 1 {
		t.Fatalf("expected fill, got %+v", res)
	}
	script, err := funscript.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(script.Actions) <= 4 {
		t.Fatalf("actions not expanded: %d", len(script.Actions))
	}
}

// TestImproveGeneratedScriptRefreshesStaleQualityAndSidecar is the QC-B
// Finding 1 regression (PR #184): raw Generate writes quality_score /
// quality_warnings and (for Tf/Tj) a .sam sidecar once and never again, so
// an Improve edit (fill-gaps here) used to leave both describing the
// pre-edit curve forever. Improve must now re-derive them from the actions
// it just wrote.
func TestImproveGeneratedScriptRefreshesStaleQualityAndSidecar(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "clip.funscript")
	staleScore := 0.1
	stalePassed := false
	body := map[string]any{
		"actions": []map[string]any{
			{"at": 0, "pos": 0},
			{"at": 100, "pos": 10},
			{"at": 5000, "pos": 90},
			{"at": 5100, "pos": 100},
		},
		"metadata": map[string]any{
			"creator":          "test",
			"profile":          "tf",
			"quality_score":    staleScore,
			"quality_passed":   stalePassed,
			"quality_warnings": []string{"stale warning from raw generate"},
		},
	}
	raw, _ := json.Marshal(body)
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	// Stale .sam sidecar: fewer/different frames than the post-fill-gaps
	// funscript will end up with - proves it gets overwritten, not just
	// left alone because a file already exists there.
	sidecarPath := sam.SidecarPath(path)
	staleSidecar := map[string]any{
		"version": "0.1",
		"frames": []map[string]any{
			{"time": 0, "motion": map[string]any{"position": 42}},
		},
	}
	rawSidecar, _ := json.Marshal(staleSidecar)
	if err := os.WriteFile(sidecarPath, rawSidecar, 0o644); err != nil {
		t.Fatal(err)
	}

	a := &App{}
	res, err := a.ImproveGeneratedScript(ImproveScriptRequest{
		Path:     path,
		FillGaps: true,
		MaxGapMs: 800,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.GapsFilled < 1 || res.PointsAdded < 1 {
		t.Fatalf("expected fill, got %+v", res)
	}

	script, err := funscript.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if script.Metadata.QualityScore == nil || *script.Metadata.QualityScore == staleScore {
		t.Fatalf("quality_score not refreshed: %+v", script.Metadata.QualityScore)
	}
	if script.Metadata.QualityPassed == nil || *script.Metadata.QualityPassed != true {
		t.Fatalf("quality_passed not refreshed to a clean curve's true: %+v", script.Metadata.QualityPassed)
	}
	for _, w := range script.Metadata.QualityWarnings {
		if w == "stale warning from raw generate" {
			t.Fatalf("stale warning survived refresh: %+v", script.Metadata.QualityWarnings)
		}
	}

	sidecar, err := sam.Load(sidecarPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(sidecar.Frames) != len(script.Actions) {
		t.Fatalf(".sam sidecar not refreshed: %d frames vs %d actions", len(sidecar.Frames), len(script.Actions))
	}
}

// TestImproveGeneratedScriptSamnRefreshesQuality is the .samn-path variant:
// saveImprovedActions must update the Document's own QualityScore too (it's
// baked in at raw Generate, same staleness bug), not just the exported
// .funscript companion.
func TestImproveGeneratedScriptSamnRefreshesQuality(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "clip.samn")
	staleScore := 0.1
	doc := &samn.Document{
		Version:    samn.CurrentVersion,
		Kind:       samn.Kind,
		DurationMs: 5100,
		General: []samn.Point{
			{At: 0, Pos: 0},
			{At: 100, Pos: 10},
			{At: 5000, Pos: 90},
			{At: 5100, Pos: 100},
		},
		QualityScore: &staleScore,
	}
	if err := samn.Save(path, doc); err != nil {
		t.Fatal(err)
	}

	a := &App{}
	res, err := a.ImproveGeneratedScript(ImproveScriptRequest{
		Path:     path,
		FillGaps: true,
		MaxGapMs: 800,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.GapsFilled < 1 {
		t.Fatalf("expected fill, got %+v", res)
	}

	reloaded, err := samn.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.QualityScore == nil || *reloaded.QualityScore == staleScore {
		t.Fatalf(".samn QualityScore not refreshed: %+v", reloaded.QualityScore)
	}

	companion, err := funscript.Load(samn.CompanionFunscriptPath(path))
	if err != nil {
		t.Fatal(err)
	}
	if companion.Metadata.QualityScore == nil || *companion.Metadata.QualityScore != *reloaded.QualityScore {
		t.Fatalf("companion .funscript quality_score out of sync with .samn: %+v vs %+v",
			companion.Metadata.QualityScore, reloaded.QualityScore)
	}
}

func TestImproveGeneratedScriptTrim(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "clip.funscript")
	body := `{"actions":[{"at":0,"pos":0},{"at":1000,"pos":50},{"at":2000,"pos":100},{"at":3000,"pos":50}],"metadata":{}}`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	a := &App{}
	res, err := a.ImproveGeneratedScript(ImproveScriptRequest{
		Path:     path,
		StartSec: 0.5,
		EndSec:   2.5,
		FillGaps: false,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Trimmed {
		t.Fatalf("expected trim: %+v", res)
	}
	script, err := funscript.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if script.Actions[0].At != 500 || script.Actions[len(script.Actions)-1].At != 2500 {
		t.Fatalf("ends %+v", script.Actions)
	}
}

func TestImproveGeneratedScriptHealSamnClearsGaps(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "clip.samn")
	doc := &samn.Document{
		Version: samn.CurrentVersion,
		Kind:    samn.Kind,
		General: []funscript.Action{
			{At: 0, Pos: 0},
			{At: 100, Pos: 10},
			{At: 900, Pos: 99},
			{At: 1100, Pos: 98},
			{At: 2000, Pos: 90},
			{At: 2100, Pos: 100},
		},
		TrackingGaps: []funscript.TrackingGap{{StartMs: 500, EndMs: 1600}},
	}
	if err := samn.Save(path, doc); err != nil {
		t.Fatal(err)
	}
	a := &App{}
	res, err := a.ImproveGeneratedScript(ImproveScriptRequest{
		Path:             path,
		HealTrackingGaps: true,
		FillGaps:         false,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.WindowsHealed < 1 {
		t.Fatalf("expected heal: %+v", res)
	}
	got, err := samn.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.TrackingGaps) != 0 {
		t.Fatalf("samn gaps not cleared: %+v", got.TrackingGaps)
	}
	funPath := samn.CompanionFunscriptPath(path)
	fs, err := funscript.Load(funPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(fs.Metadata.TrackingGaps) != 0 {
		t.Fatalf("companion gaps not cleared: %+v", fs.Metadata.TrackingGaps)
	}
}
