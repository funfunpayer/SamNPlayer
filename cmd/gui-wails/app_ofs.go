package main

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/funfunpayer/SamNPlayer/funscript"
)

// GetScriptBookmarks liest metadata.bookmarks (OFS-/Community-Stil).
func (a *App) GetScriptBookmarks() ([]funscript.Bookmark, error) {
	path := a.loadedScriptPath()
	if path == "" {
		return nil, fmt.Errorf("no script loaded")
	}
	return funscript.LoadBookmarks(path)
}

// SaveScriptBookmarks schreibt metadata.bookmarks.
func (a *App) SaveScriptBookmarks(bookmarks []funscript.Bookmark) error {
	path := a.loadedScriptPath()
	if path == "" {
		return fmt.Errorf("no script loaded")
	}
	return funscript.SaveBookmarks(path, bookmarks)
}

// GetScriptChapterMarks liest metadata.chapters.
func (a *App) GetScriptChapterMarks() ([]funscript.ChapterMark, error) {
	path := a.loadedScriptPath()
	if path == "" {
		return nil, fmt.Errorf("no script loaded")
	}
	return funscript.LoadChapters(path)
}

// SaveScriptChapterMarks schreibt metadata.chapters.
func (a *App) SaveScriptChapterMarks(chapters []funscript.ChapterMark) error {
	path := a.loadedScriptPath()
	if path == "" {
		return fmt.Errorf("no script loaded")
	}
	return funscript.SaveChapters(path, chapters)
}

// ExportScriptHeatmapPNG schreibt eine Heatmap-PNG neben das Skript.
func (a *App) ExportScriptHeatmapPNG() (string, error) {
	script := a.loadedScript()
	path := a.loadedScriptPath()
	if script == nil || path == "" {
		return "", fmt.Errorf("no script loaded")
	}
	chapters, _ := funscript.LoadChapters(path)
	out := path[:len(path)-len(filepath.Ext(path))] + ".heatmap.png"
	err := funscript.ExportHeatmapPNG(out, script.Actions, funscript.HeatmapPNGOptions{
		Width: 960, Height: 56, Chapters: chapters,
	})
	return out, err
}

// SavePlaybackProject schreibt ein .snp.json neben Video oder Skript.
func (a *App) SavePlaybackProject(p funscript.Project) (string, error) {
	media := p.VideoPath
	if media == "" {
		media = p.ScriptPath
	}
	if media == "" {
		media = a.loadedScriptPath()
		p.ScriptPath = media
	}
	if media == "" {
		return "", fmt.Errorf("no video/script for project")
	}
	path := funscript.ProjectPathFor(media)
	p.LastOpenedMs = time.Now().UnixMilli()
	if err := funscript.SaveProject(path, p); err != nil {
		return "", err
	}
	return path, nil
}

// LoadPlaybackProject lädt ein .snp.json.
func (a *App) LoadPlaybackProject(path string) (funscript.Project, error) {
	return funscript.LoadProject(path)
}

// SnapTimeMs rastet auf Frame-Raster (fps).
func (a *App) SnapTimeMs(tMs int64, fps float64) int64 {
	return funscript.SnapMs(tMs, fps)
}

// EditDeleteRange löscht Punkte im Bereich und speichert.
func (a *App) EditDeleteRange(startMs, endMs int64) error {
	script := a.loadedScript()
	path := a.loadedScriptPath()
	if script == nil || path == "" {
		return fmt.Errorf("no script loaded")
	}
	out, err := funscript.DeleteRange(script.Actions, startMs, endMs)
	if err != nil {
		return err
	}
	return a.SaveScriptActions(out)
}

// EditCapSpeedRange begrenzt Intensität im Bereich und speichert.
func (a *App) EditCapSpeedRange(startMs, endMs int64, maxIntensity float64) error {
	script := a.loadedScript()
	path := a.loadedScriptPath()
	if script == nil || path == "" {
		return fmt.Errorf("no script loaded")
	}
	out, err := funscript.CapSpeedRange(script.Actions, startMs, endMs, maxIntensity)
	if err != nil {
		return err
	}
	return a.SaveScriptActions(out)
}

// EditScaleRange skaliert Positionen im Bereich um 50 und speichert.
func (a *App) EditScaleRange(startMs, endMs int64, factor float64) error {
	script := a.loadedScript()
	path := a.loadedScriptPath()
	if script == nil || path == "" {
		return fmt.Errorf("no script loaded")
	}
	out := funscript.ScaleRangePos(script.Actions, startMs, endMs, factor)
	return a.SaveScriptActions(out)
}
