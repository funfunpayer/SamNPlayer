package main

import "github.com/funfunpayer/SamNPlayer/generator/aiscript"

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

// DraftAIScript is the opt-in AI stroke draft. S0 always fails closed with
// an English error; GUI must not call this unless Status.Available.
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
