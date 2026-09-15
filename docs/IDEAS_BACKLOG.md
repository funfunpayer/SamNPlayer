# Ideas backlog (not on main)

Branch `feat/polarity-roi2-o-zone`, draft PR #50. Updated 2026-09-15.

## Done without operator files or hardware

- C polarity + ROI2 coach + backend-size hint
- D O-zone suggest/apply + post-generate review/invert confirm
- Golden-clip protocol (`docs/GOLDEN_CLIPS.md`)
- Chapter labels from motionx (`pause|build|steady|crescendo|winddown`)
- Ring-down helper (`funscript.RingDown`) — wired via `ApplyRingDown` + playback button (confirmation required)
- `SuggestBackend` from ROI area (grid_lk if small, else CSRT)
- ozone_ui.js: Taste O dispatches `ozone:hotkey`, buttons for suggest/invert/ring-down

## Open on this branch (code)

- **Restore `cmd/gui-wails/frontend/src/playback.js` from main** and re-apply:
  1. `import { applyHotkeyOMarker } from './ozone_ui.js'`
  2. listener `ozone:hotkey` → `applyHotkeyOMarker(scriptPath, nowMs, oMarkers)` + UI refresh
  3. listeners for `ozone:suggested` / `polarity:inverted` / `ringdown:applied` → reload curve/heatmap/markers
  4. keyboard-hint texts mention Taste O
  Current file is an explicit stub so the breakage is visible. Full ~40k module
  could not be pushed through the agent path (payload truncation).

## Blocked on you

- Golden clips (video + FunGen .funscript + fixed ROIs)
- Scene names for LabelScene
- Sam Neo 2 measurements

## Deliberately not built

Upscale, sharpen, backend fusion, auto-apply chapters to the device,
changing Extended-O hold times, FunGen source, pipeline GUI,
auto RingDown inside live Extended-O restore (explicit button only).
