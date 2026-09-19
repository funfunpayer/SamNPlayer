package videox

import (
	"context"
	"os/exec"
)

// CommandContext returns an *exec.Cmd for the resolved ffmpeg binary.
func CommandContext(ctx context.Context, args ...string) (*exec.Cmd, error) {
	bin, err := FFmpeg()
	if err != nil {
		return nil, err
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return exec.CommandContext(ctx, bin, args...), nil
}

// ProbeCommandContext returns an *exec.Cmd for the resolved ffprobe binary.
func ProbeCommandContext(ctx context.Context, args ...string) (*exec.Cmd, error) {
	bin, err := FFprobe()
	if err != nil {
		return nil, err
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return exec.CommandContext(ctx, bin, args...), nil
}
