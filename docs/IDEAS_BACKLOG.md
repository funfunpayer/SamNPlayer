# Ideas backlog (not on main)

Branch `feat/polarity-roi2-o-zone`, draft PR #50. Updated 2026-09-15.

## Done without operator files or hardware

- C polarity + ROI2 coach + backend-size hint
- D O-zone suggest/apply + post-generate review/invert confirm
- Golden-clip protocol (`docs/GOLDEN_CLIPS.md`)
- Chapter labels from motionx (`pause|build|steady|crescendo|winddown`)
- Ring-down helper (`funscript.RingDown`) — wired via `ApplyRingDown` + playback button (confirmation required)
- `SuggestBackend` from ROI area (grid_lk if small, else CSRT)
- Taste O hotkey path: `ozone:hotkey` → `applyHotkeyOMarker` (4s primary) + UI refresh; polarity/ring-down/ozone listeners reload curve/heatmap

## Blocked on you

- Golden clips (video + FunGen .funscript + fixed ROIs)
- Scene names for LabelScene
- Sam Neo 2 measurements

## Deliberately not built

Upscale, sharpen, backend fusion, auto-apply chapters to the device,
changing Extended-O hold times, FunGen source, pipeline GUI,
auto RingDown inside live Extended-O restore (explicit button only).
