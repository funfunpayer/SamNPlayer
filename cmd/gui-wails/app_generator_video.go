package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"
)

// Generator mode (Owner 14 Oct, #533): the Generator clip plays in a real
// <video> with transport and seeking, like the Player. It has its own path
// and URL on the local video server, so choosing a clip to create from never
// replaces the Player's linked video (same rule as the Training film).

// SetGeneratorVideo selects the clip for the Generator video and returns the
// URL the <video> element loads.
func (a *App) SetGeneratorVideo(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("no video path")
	}
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return "", fmt.Errorf("video file not found: %s", path)
	}
	a.stateMu.Lock()
	if a.generatorVideoPath != path {
		a.generatorVideoPath = path
		a.generatorVideoEpoch++
	}
	a.stateMu.Unlock()
	return a.GeneratorVideoURL(), nil
}

// GeneratorVideoURL is the local URL of the Generator clip ("" = none).
func (a *App) GeneratorVideoURL() string {
	a.stateMu.RLock()
	path := a.generatorVideoPath
	epoch := a.generatorVideoEpoch
	a.stateMu.RUnlock()
	if path == "" {
		return ""
	}
	port, err := a.ensureVideoServer()
	if err != nil {
		return ""
	}
	return fmt.Sprintf("http://127.0.0.1:%d/generator-video?v=%d", port, epoch)
}

// serveGeneratorVideo serves only the one selected file (same rules as the
// Player video: localhost server, no path from the request).
func (a *App) serveGeneratorVideo(w http.ResponseWriter, r *http.Request) {
	a.stateMu.RLock()
	path := a.generatorVideoPath
	a.stateMu.RUnlock()
	if path == "" {
		http.NotFound(w, r)
		return
	}
	if _, err := os.Stat(path); err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", videoContentType(path))
	http.ServeFile(w, r, path)
}
