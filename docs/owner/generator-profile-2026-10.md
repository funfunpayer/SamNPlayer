# Generator profile — 2026-10 (M0)

**Status:** template. Fill after running `scripts/profile-generator.sh` on a real clip with OpenCV.

## How to measure

1. `sudo apt install libopencv-dev pkg-config` (or MSYS2 equivalent).
2. `./scripts/profile-generator.sh`
3. Run the built CLI on a representative clip (Everyday CSRT path) with CPU profile.
4. Record median and p95 of total time, and the top 5 functions from pprof (CSRT, frame decode, post-processing, etc.).

## Results (to fill)

| Sink | % of wall time | Notes |
|---|---|---|
| CSRT tracking | | |
| Frame decode (videox) | | |
| Post-processing / funscript write | | |
| Other | | |

**Gate:** only move a path to C++ if Python is measurably slower here (3 warm-up + 5 measured runs, median and p95).

See [`MIGRATION_CPP_PYTHON.md`](../MIGRATION_CPP_PYTHON.md) M0.
