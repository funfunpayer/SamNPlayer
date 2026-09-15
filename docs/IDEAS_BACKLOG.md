# Ideas backlog (not on main)

Branch `feat/polarity-roi2-o-zone`, draft PR #50. Updated 2026-09-15.

## Done without operator files or hardware

- C polarity + ROI2 coach + backend-size hint
- D O-zone suggest/apply + post-generate review/invert confirm
- Golden-clip protocol (`docs/GOLDEN_CLIPS.md`)
- Chapter labels from motionx (`pause|build|steady|crescendo|winddown`) — pure + `ScriptChapters` App method + Wails binding
- Ring-down helper (`funscript.RingDown`) — wired via `ApplyRingDown` + playback button (confirmation required)
- `SuggestBackend` from ROI area (grid_lk if small, else CSRT) — Go + binding + ROI coach call
- ozone_ui.js: Taste O dispatches `ozone:hotkey`, buttons for suggest/invert/ring-down

## Open on this branch (code) — priority

- **playback.js is a placeholder again** (concurrent push regression). Must restore the full module from `main` and add:
  - `import { applyHotkeyOMarker } from './ozone_ui.js'`
  - `window` listener for `ozone:hotkey` → `applyHotkeyOMarker` (4s primary) + list/heatmap/curve refresh
  - listeners for `ozone:suggested` / `polarity:inverted` / `ringdown:applied`
  - keyboard hints mention Taste O
- Optional: show `ScriptChapters()` labels on the heatmap/curve (binding exists; no auto-apply to device — deliberate)
- Optional: regenerate full Wails `App.d.ts` so TypeScript matches App.js

## Blocked on you

- Golden clips (video + FunGen .funscript + fixed ROIs)
- Scene names for LabelScene
- Sam Neo 2 measurements

## Deliberately not built

Upscale, sharpen, backend fusion, auto-apply chapters to the device,
changing Extended-O hold times, FunGen source, pipeline GUI,
auto RingDown inside live Extended-O restore (explicit button only).
