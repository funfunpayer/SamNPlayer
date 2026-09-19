# Lean self-build

When an open-source tool does a job we need, the default is **not** to
vendor the whole stack. Prefer a **leaner self-built** piece that covers
our measured use case — inspired by how OSS solves it, not a copy of the
dependency tree.

## Rule

1. Can we cover ≥ the cases we ship with a small Go module we own? → build it.  
2. Would a full reimplementation be worse (codecs, browsers, BLE stacks)? → keep the thin kernel (ffmpeg, OS webview, platform BLE) behind our interface.  
3. Never add a heavy library “because FunGen/OFS has it” without a golden-clip win.

## Done (self-built / lean)

| Area | Instead of | Ours |
|------|------------|------|
| Post-track signal | Python scipy path | `generator/posttrack` |
| NCC tracking (no OpenCV) | Python OpenCV | `generator/simpletrack` |
| Audio tempo check | `audio_check.py` only | `generator/audiocheck.go` |
| Funscript / `.samn` | third-party script libs | `funscript/`, `samn/` |
| Device playback clock | Electron players | `player/` + Wails webview |
| ISO BMFF probe (common MP4) | always need ffprobe | `videox` lean probe fallback |
| ffmpeg discovery | “install system-wide” | portable beside app + tools dir |

## Keep as thin kernels (do not reimplement)

| Kernel | Why |
|--------|-----|
| **ffmpeg** decode/encode | Full AV codec zoo; we ship portable binaries |
| **OpenCV CSRT** (when linked) | Quality-critical tracker; no lean Go CSRT yet |
| **OS webview** (Wails) | Chromium-in-Electron is the heavy alternative we avoid |
| **Platform BLE / Intiface** | OS and vendor stacks |

## Next lean candidates (only with measurement)

| Candidate | Gate |
|-----------|------|
| Go `auto_roi` (rhythm heuristic) | Match Python ROI on golden clips |
| Optical-flow / `grid_lk` in Go | Beat or match Python backend scores |
| More container probes (WebM/MKV) | Only if portable users lack ffprobe often |

See also: `docs/ENGINE.md`, `docs/PLATFORMS.md`, `docs/FFMPEG_TOOLS.md`.
