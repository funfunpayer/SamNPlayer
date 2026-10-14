package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGeneratorVideoServedSeparately(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	a := NewApp()
	if a.GeneratorVideoURL() != "" {
		t.Fatal("no generator video at start")
	}
	if _, err := a.SetGeneratorVideo(filepath.Join(t.TempDir(), "missing.mp4")); err == nil {
		t.Fatal("missing file must be refused")
	}
	clip := filepath.Join(t.TempDir(), "clip.mp4")
	if err := os.WriteFile(clip, []byte("CLIPDATA"), 0o644); err != nil {
		t.Fatal(err)
	}
	url, err := a.SetGeneratorVideo(clip)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(url, "/generator-video?v=") {
		t.Fatalf("unexpected URL %q", url)
	}
	if a.VideoFileURL() != "" {
		t.Fatal("Generator clip must not become the Player video")
	}
	again, _ := a.SetGeneratorVideo(clip)
	if again != url {
		t.Fatalf("same clip keeps its URL: %q vs %q", again, url)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/generator-video?v=1", nil)
	req.Header.Set("Range", "bytes=4-7")
	a.serveGeneratorVideo(rec, req)
	if rec.Code != http.StatusPartialContent || rec.Body.String() != "DATA" {
		t.Fatalf("range: %d %q", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "video/mp4" {
		t.Fatalf("content type %q", ct)
	}
}
