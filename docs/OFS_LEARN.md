# Learning from OpenFunscripter (OFS) — in Go

OpenFunscripter is a desktop **script editor** (frame-accurate timeline,
heatmap, chapters, multi-axis, Lua extensions). SamNPlayer stays a
**generate → inspect → play** app for Sam Neo 2, but we deliberately
**learn from OFS and port the best ideas into Go** — not as an ImGui
clone, and not blocked by older “no OFS clone” wording.

Official project (archived): [OpenFunscripter/OFS](https://github.com/OpenFunscripter/OFS).

## Already aligned (shared community / already shipped)

| Idea | Where |
|---|---|
| Intensity `500 × \|Δpos\| / \|Δt\|` | `funscript/device_compat.go`, OFS/funscript-utils/XBVR |
| Curve edit (drag / add / delete) | Wiedergabe + `app_editor.go` |
| Heatmap, chapters, bookmarks | Player UI + `motionx` / `ScriptChapters` |
| Soft spline curve draw | `playback.js` |

## Adopt next — implement in Go, wire in GUI

Priority order (highest payoff first):

1. **Max-speed highlights on the curve** ✅ started  
   Community intensity over a threshold → red bands on the curve  
   (`funscript.SpeedHighlights`, `GetSpeedHighlights`).

2. **Chapters in funscript metadata + heatmap export**  
   Persist chapter markers in `.funscript` metadata the way the community
   expects; optional PNG heatmap with chapter ticks (Go writer).

3. **Project persistence (lightweight)**  
   Relative video + script + seek/offset/markers in one sidecar JSON
   (not `.ofsp` CBOR — keep it simple Go/JSON). Autobackup of edits.

4. **Frame / tempo editing aids**  
   Snap-to-frame when video FPS known; optional BPM grid for rhythmic
   sections — Go helpers + thin UI controls.

5. **Selection / repair tools**  
   Select time range → delete / scale / smooth / speed-cap (Script Doctor
   already repairs; expose range ops in Go).

6. **Optional WebSocket bridge**  
   Later: push playhead / actions to external tools (MultiFunPlayer-style),
   without embedding a full plugin host.

## Explicitly not copying

- Full ImGui OFS UI / multi-axis 3D studio as the main shell  
- Lua extension runtime as a hard dependency for core features  
- Bundling OFS binaries or reusing OFS source tree  

Ideas are reimplemented clean-room in Go under SamNPlayer’s own packages
(`funscript/`, `motionx/`, `cmd/gui-wails/`).

## Workflow reminder

AI/ONNX still only **proposes ROIs**. Classical Go/Python tracking still
**writes** the funscript. OFS-inspired editor features improve the
**inspect / repair** step after generation.
