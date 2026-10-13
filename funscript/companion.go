package funscript

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Companion axis files (docs/DEVICE_SUPPORT.md, D4): multi-axis scripts
// keep each extra axis in its own funscript beside the main one, named
// "<name>.<axis>.funscript" (the convention of OpenFunscripter and
// MultiFunPlayer). The main script stays the stroke (L0).

// CompanionAxes lists the extra axes SamNPlayer reads, in display order.
var CompanionAxes = []AxisName{"surge", "sway", "twist", "roll", "pitch"}

// IsCompanionAxis reports whether name is one of CompanionAxes.
func IsCompanionAxis(name AxisName) bool {
	for _, a := range CompanionAxes {
		if a == name {
			return true
		}
	}
	return false
}

// CompanionPath returns the file of axis next to the main script path
// (".funscript" or ".samn"): /x/clip.funscript → /x/clip.twist.funscript.
func CompanionPath(mainPath string, axis AxisName) string {
	base := mainPath
	if ext := filepath.Ext(base); strings.EqualFold(ext, ".funscript") || strings.EqualFold(ext, ".samn") {
		base = strings.TrimSuffix(base, ext)
	}
	return base + "." + string(axis) + ".funscript"
}

// LoadCompanionAxis reads one companion axis. A missing file is no error:
// it returns nil, nil. Actions come sorted by time, inversion applied.
func LoadCompanionAxis(mainPath string, axis AxisName) ([]Action, error) {
	if mainPath == "" || !IsCompanionAxis(axis) {
		return nil, nil
	}
	s, err := Load(CompanionPath(mainPath, axis))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	acts := append([]Action(nil), s.PlaybackActions()...)
	sort.SliceStable(acts, func(i, j int) bool { return acts[i].At < acts[j].At })
	return acts, nil
}

// CompanionAxesFound lists the companion axes that exist beside mainPath.
func CompanionAxesFound(mainPath string) []AxisName {
	var out []AxisName
	for _, a := range CompanionAxes {
		if acts, err := LoadCompanionAxis(mainPath, a); err == nil && len(acts) > 0 {
			out = append(out, a)
		}
	}
	return out
}

// SaveCompanionAxis writes actions to an existing companion file (Edit view,
// D4). Other fields of the file stay as they are; a file marked "inverted"
// gets 100−pos written back, so it plays exactly as edited.
func SaveCompanionAxis(mainPath string, axis AxisName, actions []Action) error {
	if !IsCompanionAxis(axis) {
		return fmt.Errorf("funscript: %q is not a companion axis", axis)
	}
	path := CompanionPath(mainPath, axis)
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("funscript: %s file not found beside the script: %w", axis, err)
	}
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(data, &doc); err != nil {
		return fmt.Errorf("funscript: ungültiges JSON: %w", err)
	}
	cleaned, err := SanitizeActions(actions)
	if err != nil {
		return err
	}
	var inverted bool
	if raw, ok := doc["inverted"]; ok {
		_ = json.Unmarshal(raw, &inverted)
	}
	if inverted {
		for i := range cleaned {
			cleaned[i].Pos = 100 - cleaned[i].Pos
		}
	}
	actionsJSON, err := json.Marshal(cleaned)
	if err != nil {
		return err
	}
	doc["actions"] = actionsJSON
	out, err := json.Marshal(doc)
	if err != nil {
		return err
	}
	return writeFileAtomic(path, out, 0o644)
}

// CreateCompanionAxis starts a new companion file for axis beside the main
// script (Edit → "Add axis"): two points at the centre (50), from 0 to
// endMs, so the axis can be shaped right away. An existing file is never
// overwritten.
func CreateCompanionAxis(mainPath string, axis AxisName, endMs int64) (string, error) {
	if !IsCompanionAxis(axis) {
		return "", fmt.Errorf("funscript: %q is not a companion axis", axis)
	}
	if mainPath == "" {
		return "", fmt.Errorf("funscript: no script loaded")
	}
	path := CompanionPath(mainPath, axis)
	if _, err := os.Stat(path); err == nil {
		return "", fmt.Errorf("funscript: %s already exists", filepath.Base(path))
	}
	if endMs < 1000 {
		endMs = 1000
	}
	doc := map[string]any{
		"version":  "1.0",
		"inverted": false,
		"range":    100,
		"actions":  []Action{{At: 0, Pos: 50}, {At: endMs, Pos: 50}},
	}
	data, err := json.Marshal(doc)
	if err != nil {
		return "", err
	}
	if err := writeFileAtomic(path, data, 0o644); err != nil {
		return "", err
	}
	return path, nil
}
