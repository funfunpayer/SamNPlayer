package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

// settingsStore ist eine bewusst simple JSON-Datei-Persistenz - kein
// externes Key-Value-Store-Paket, das wir nicht brauchen (siehe "was wir
// nicht brauchen muss nicht rein"). Liegt im selben Nutzerkonfigurations-
// ordner wie die Logdatei (logging.Dir()'s Elternverzeichnis).
type settingsStore struct {
	mu   sync.Mutex
	path string
	data map[string]any
}

func newSettingsStore() (*settingsStore, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(base, "SamNPlayer")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	s := &settingsStore{path: filepath.Join(dir, "settings.json"), data: map[string]any{}}
	s.load()
	return s, nil
}

func (s *settingsStore) load() {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, err := os.ReadFile(s.path)
	if err != nil {
		return // Datei existiert noch nicht - leere Defaults sind ok
	}
	_ = json.Unmarshal(b, &s.data)
}

func (s *settingsStore) save() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, b, 0644)
}

func (s *settingsStore) GetString(key, fallback string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if v, ok := s.data[key].(string); ok {
		return v
	}
	return fallback
}

func (s *settingsStore) GetBool(key string, fallback bool) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if v, ok := s.data[key].(bool); ok {
		return v
	}
	return fallback
}

func (s *settingsStore) GetFloat(key string, fallback float64) float64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	if v, ok := s.data[key].(float64); ok {
		return v
	}
	return fallback
}

func (s *settingsStore) Set(key string, value any) error {
	s.mu.Lock()
	s.data[key] = value
	s.mu.Unlock()
	return s.save()
}
