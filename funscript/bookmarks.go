package funscript

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
)

// Bookmark is a named time marker in funscript metadata (OFS/community style).
type Bookmark struct {
	Name string `json:"name"`
	Time int64  `json:"time"` // ms
}

// ChapterMark is a named span in metadata (optional end for open-ended).
type ChapterMark struct {
	Name      string `json:"name"`
	StartTime int64  `json:"startTime"`
	EndTime   int64  `json:"endTime,omitempty"`
}

// LoadBookmarks reads metadata.bookmarks without rewriting the file.
func LoadBookmarks(path string) ([]Bookmark, error) {
	meta, err := readMetadataMap(path)
	if err != nil {
		return nil, err
	}
	raw, ok := meta["bookmarks"]
	if !ok || len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var out []Bookmark
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("funscript: bookmarks ungültig: %w", err)
	}
	return out, nil
}

// SaveBookmarks replaces metadata.bookmarks, preserving other metadata keys.
func SaveBookmarks(path string, bookmarks []Bookmark) error {
	sorted := append([]Bookmark(nil), bookmarks...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Time < sorted[j].Time })
	return patchMetadataKey(path, "bookmarks", sorted)
}

// LoadChapters reads metadata.chapters.
func LoadChapters(path string) ([]ChapterMark, error) {
	meta, err := readMetadataMap(path)
	if err != nil {
		return nil, err
	}
	raw, ok := meta["chapters"]
	if !ok || len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var out []ChapterMark
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("funscript: chapters ungültig: %w", err)
	}
	return out, nil
}

// SaveChapters replaces metadata.chapters.
func SaveChapters(path string, chapters []ChapterMark) error {
	sorted := append([]ChapterMark(nil), chapters...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].StartTime < sorted[j].StartTime })
	return patchMetadataKey(path, "chapters", sorted)
}

func readMetadataMap(path string) (map[string]json.RawMessage, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("funscript: Datei konnte nicht gelesen werden: %w", err)
	}
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("funscript: ungültiges JSON: %w", err)
	}
	meta := map[string]json.RawMessage{}
	if raw, ok := doc["metadata"]; ok && len(raw) > 0 && string(raw) != "null" {
		if err := json.Unmarshal(raw, &meta); err != nil {
			return nil, fmt.Errorf("funscript: metadata ungültig: %w", err)
		}
	}
	return meta, nil
}

func patchMetadataKey(path, key string, value any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("funscript: Datei konnte nicht gelesen werden: %w", err)
	}
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(data, &doc); err != nil {
		return fmt.Errorf("funscript: ungültiges JSON: %w", err)
	}
	meta := map[string]json.RawMessage{}
	if raw, ok := doc["metadata"]; ok && len(raw) > 0 && string(raw) != "null" {
		if err := json.Unmarshal(raw, &meta); err != nil {
			return fmt.Errorf("funscript: metadata ungültig: %w", err)
		}
	}
	enc, err := json.Marshal(value)
	if err != nil {
		return err
	}
	meta[key] = enc
	metaJSON, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	doc["metadata"] = metaJSON
	out, err := json.Marshal(doc)
	if err != nil {
		return err
	}
	return os.WriteFile(path, out, 0o644)
}
