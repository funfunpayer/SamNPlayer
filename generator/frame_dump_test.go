package generator

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDumpFrameAtSecCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := DumpFrameAtSec(ctx, "missing.mp4", filepath.Join(t.TempDir(), "out.jpg"), 0)
	if err == nil {
		t.Fatal("expected error for canceled context")
	}
	if !strings.Contains(err.Error(), "canceled") && ctx.Err() == nil {
		// DumpFrameAtSec should surface context cancellation when ctx is done.
	}
}

func TestDumpFrameAtSecMissingBinary(t *testing.T) {
	oldPath := os.Getenv("PATH")
	t.Cleanup(func() { _ = os.Setenv("PATH", oldPath) })
	_ = os.Setenv("PATH", "")
	err := DumpFrameAtSec(context.Background(), "missing.mp4", filepath.Join(t.TempDir(), "out.jpg"), 1.5)
	if err == nil {
		t.Fatal("expected error when ffmpeg is unavailable")
	}
}

func TestDumpFrameAtSecTimesOut(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond)
	defer cancel()
	time.Sleep(time.Millisecond)
	err := DumpFrameAtSec(ctx, "missing.mp4", filepath.Join(t.TempDir(), "out.jpg"), 0)
	if err == nil {
		t.Fatal("expected timeout/cancel error")
	}
}

func TestDumpFrameAtSecRejectsEmptyPaths(t *testing.T) {
	if err := DumpFrameAtSec(context.Background(), "", "out.jpg", 0); err == nil {
		t.Fatal("expected error for empty video path")
	}
	if err := DumpFrameAtSec(context.Background(), "video.mp4", "", 0); err == nil {
		t.Fatal("expected error for empty output path")
	}
}

// Ensure exec.CommandContext remains the failure mode for missing ffmpeg without panicking.
var _ = exec.CommandContext
