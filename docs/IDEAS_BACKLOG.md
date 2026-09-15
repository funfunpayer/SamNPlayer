# Ideas backlog (not on main)

Recorded 2026-09-15 from the product discussion. **C and D are implemented
on `feat/polarity-roi2-o-zone` only.** Everything else stays a note until
measured.

## Implemented on this branch

### C — polarity + ROI2 help in the preview

- Visible invert checkbox outside Advanced settings, labelled as polarity
  (FunGen disagreement is a sign convention, not a tracking bug).
- Coach text when ROI1 is tiny, on the frame edge, overlapping ROI2, or
  only a horizontal copy of ROI1 (the dead-signal case from the Y-only
  distance bug).
- Connector line between ROI centres when native frame size is known.
- `SuggestPolarity` / `InvertLoadedScript` on a loaded script: first-half
  mean position; invert is confirmed by the user, never applied silently.

### D — propose an O-zone + editor marker

- `SuggestOZone`: last 8–15% of the script, sliding window with the
  highest mean position. Flat tails produce no suggestion.
- `ApplySuggestedOZone`: writes the sidecar marker *and* a primary
  `oMarkers` entry. User can still edit/delete.
- Playback UI: “O-Zone vorschlagen”, “Richtung prüfen / umkehren”.
- Hotkey `O` emits `ozone:hotkey` with current video time (4s primary
  marker helper `applyHotkeyOMarker`).

## Still only ideas (do not ship without measurement)

### GUI smoother

- Generate → curve → play as one pipeline.
- Live periodicity while dragging the ROI.
- Curve undo, re-track last N seconds, zoom/pan.
- Show device packet lag vs video time; drag offset on the heatmap.
- Decouple canvas redraw from the device tick.
- Honest generator progress: frame, scene, lost-lock, cache hit.

### Film / action recognition

- Wire motion signature into the generator UI (HANDOFF: exists, unconnected).
- Classify cuts as pause / build / steady / crescendo from the 8 features.
- Steer the ROI over time; hard reset on cuts; no guess below similarity.
- Auto two-ROI only after 2–3 real clips keep FunGen-r.
- Backend per scene, not per film.
- Keep Quality Doctor and FunGen-r as separate numbers.

Rejected without a new multi-clip sweep: upscale, unsharp, backend fusion,
mandatory neural detection.

### O-system (beyond this branch)

- Separate build-up vs hold / Extended-O (Tf/Tj: vibration off on hold).
- Contact vibration from peak proximity, not duration (already tested).
- Soft ring-down after O instead of a jump to 0.
- DeviceProfile hold length once real hardware numbers exist.

### Suggested order after this branch

A golden clips → B motion signature UI → E pipeline GUI → F hardware.
C+D are done here so they can be reviewed off main.
