package main

import (
	"fmt"

	"github.com/funfunpayer/SamNPlayer/funscript"
	"github.com/funfunpayer/SamNPlayer/motionx"
)

func (a *App) SuggestBackend(w, h int) string {
	return funscript.SuggestBackend(w, h)
}

func (a *App) ScriptChapters() ([]motionx.Chapter, error) {
	if a.currentScript == nil || len(a.currentScript.Actions) < 4 {
		return nil, fmt.Errorf("kein Skript geladen")
	}
	points := make([]motionx.Point, 0, len(a.currentScript.Actions))
	for _, act := range a.currentScript.Actions {
		points = append(points, motionx.Point{TMs: float64(act.At), Pos: float64(act.Pos)})
	}
	segs := motionx.Classify(points, motionx.ClassifyOptions{})
	return motionx.Chapters(segs), nil
}
