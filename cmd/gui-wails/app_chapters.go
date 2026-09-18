package main

import (
	"fmt"

	"github.com/funfunpayer/SamNPlayer/funscript"
	"github.com/funfunpayer/SamNPlayer/motionx"
)

func (a *App) SuggestBackend(w, h int) string {
	return funscript.SuggestBackend(w, h)
}

func (a *App) SuggestPipeline(w, h, w2, h2 int) funscript.PipelineSuggestion {
	return funscript.SuggestPipeline(w, h, w2, h2)
}

func (a *App) ScriptChapters() ([]motionx.Chapter, error) {
	script := a.loadedScript()
	if script == nil || len(script.Actions) < 4 {
		return nil, fmt.Errorf("kein Skript geladen")
	}
	points := make([]motionx.Point, 0, len(script.Actions))
	for _, act := range script.Actions {
		points = append(points, motionx.Point{TMs: float64(act.At), Pos: float64(act.Pos)})
	}
	segs := motionx.Classify(points, motionx.ClassifyOptions{})
	return motionx.Chapters(segs), nil
}
