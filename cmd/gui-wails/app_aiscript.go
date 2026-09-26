package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/funfunpayer/SamNPlayer/funscript"
	"github.com/funfunpayer/SamNPlayer/generator"
	"github.com/funfunpayer/SamNPlayer/generator/aiscript"
)

// AIScriptWriterStatus reports the experimental AI draft-script path
// (docs/AI_SCRIPT_WRITER.md). Everyday Create stays CSRT until a later
// stage ships a local model and the user opts in.
func (a *App) AIScriptWriterStatus() aiscript.Status {
	path := ""
	if a.settings != nil {
		path = a.settings.GetString("generator.aiScriptModelPath", "")
	}
	return aiscript.StatusFor(path)
}

// DraftAIScript is the opt-in AI stroke draft. Always fails closed until S2+
// ships inference; GUI must not call this unless Status.Available.
func (a *App) DraftAIScript(videoPath string) (aiscript.DraftResult, error) {
	path := ""
	if a.settings != nil {
		path = a.settings.GetString("generator.aiScriptModelPath", "")
	}
	return aiscript.Draft(aiscript.DraftRequest{
		VideoPath: videoPath,
		ModelPath: path,
	})
}

// ExportAIScriptImitationResult is returned after writing one S1 sample.
type ExportAIScriptImitationResult struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

// ExportAIScriptImitation writes one classical good-run sample for offline
// AI draft training (S1). Opt-in only — never called from Everyday Generate.
func (a *App) ExportAIScriptImitation(scriptPath, videoPath string) (ExportAIScriptImitationResult, error) {
	out := ExportAIScriptImitationResult{}
	scriptPath = strings.TrimSpace(scriptPath)
	if scriptPath == "" {
		return out, fmt.Errorf("no script path")
	}
	script, err := a.loadScriptDocument(preferSamnCompanion(scriptPath))
	if err != nil {
		return out, err
	}
	if len(script.Actions) < 2 {
		return out, fmt.Errorf("script needs at least 2 points")
	}
	dir := aiscript.ImitationDirUnder(generator.DefaultRoiDatasetDir())
	sample := aiscript.ImitationSample{
		VideoPath:  strings.TrimSpace(videoPath),
		ScriptPath: scriptPath,
		Engine:     "csrt",
		Actions:    toAIScriptActions(script.Actions),
		QDPassed:   script.Metadata.QualityPassed,
		QDScore:    script.Metadata.QualityScore,
		Notes:      "S1 classical export — opt-in; Everyday CSRT unchanged",
	}
	if len(script.Actions) >= 2 {
		sample.DurationMs = script.Actions[len(script.Actions)-1].At - script.Actions[0].At
	}
	path, err := aiscript.ExportImitationSample(dir, sample)
	if err != nil {
		return out, err
	}
	out.Path = path
	out.Message = fmt.Sprintf("Exported training sample → %s", filepath.Base(path))
	return out, nil
}

func toAIScriptActions(in []funscript.Action) []aiscript.Action {
	out := make([]aiscript.Action, len(in))
	for i, a := range in {
		out[i] = aiscript.Action{At: a.At, Pos: a.Pos}
	}
	return out
}
