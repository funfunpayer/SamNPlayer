#!/usr/bin/env bash
# M0 generator profile harness.
# Builds the CLI with OpenCV and runs a short Generate under pprof.
# Result (top sinks) goes into docs/owner/generator-profile-2026-10.md.
# Requires: libopencv-dev, pkg-config, a test video + ROI.
set -euo pipefail
cd "$(dirname "$0")/.."

echo "Building CLI with OpenCV..."
go build -tags opencv -o /tmp/samn-cli-profile ./cmd/cli

# Example invocation (Owner fills real clip + ROI):
# /tmp/samn-cli-profile generate -in clip.mp4 -roi 100,100,50,50 -out /tmp/out.funscript -profile cpu
echo "Built. Run with a real clip, e.g.:"
echo "  go tool pprof -http=:0 /tmp/samn-cli-profile cpu.pprof"
echo "See docs/owner/generator-profile-2026-10.md for the report format."
