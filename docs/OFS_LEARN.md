# Learning from OpenFunscripter (OFS) — in Go

OpenFunscripter is a desktop **script editor**. SamNPlayer stays
**generate → inspect → play**, but we **port the best OFS ideas into Go**.

Official project (archived): [OpenFunscripter/OFS](https://github.com/OpenFunscripter/OFS).

## Shipped

| Idea | Where |
|---|---|
| Intensity `500 × \|Δpos\| / \|Δt\|` | `funscript/device_compat.go` |
| Curve edit | Wiedergabe + `app_editor.go` |
| Heatmap / auto chapters / bookmarks UI | Player + `motionx` |
| Max-speed highlights on curve | `funscript.SpeedHighlights` |
| Chapters/bookmarks in `.funscript` metadata | `funscript/bookmarks.go` |
| Heatmap PNG export (+ chapter ticks) | `funscript.ExportHeatmapPNG` / `ExportScriptHeatmapPNG` |
| Project sidecar `.snp.json` | `funscript/project.go` / `SavePlaybackProject` |
| Frame snap (`SnapMs`) | `SnapTimeMs` + FPS-Snap in Wiedergabe |
| Range delete / speed-cap / scale | `EditDeleteRange`, `EditCapSpeedRange`, `EditScaleRange` |

## Still optional later

- BPM / tempo grid overlay in the UI
- WebSocket bridge to external players
- Richer bookmark editor UI (CRUD list)

## Not copying

- Full ImGui OFS shell / multi-axis 3D studio  
- Lua extension host as a core dependency  
- OFS source tree or binaries  

## Workflow reminder

AI/ONNX proposes ROIs only. Classical tracking writes the funscript.
OFS-inspired tools improve **inspect / repair** after generation.
