# Changelog

Format loosely follows [Keep a Changelog](https://keepachangelog.com/).
Curated and grouped by theme, not a raw commit dump — see `git log` or
GitHub for the exact PR-by-PR history. `docs/NEXT.md` carries the detailed
measurement history behind each entry; this file is the short version for
"what changed", not "why" or "how it was measured".

## [Unreleased] — since v0.3.0 (September 16, 2026)

### Added

- **Device-Diagnostics tool** (`device/diagnostics.go`, "Geräte-Diagnose"
  section in the Device tab): an automated test sequence against the
  connected device — raw-value acceptance sweep per channel, maximum
  stable update rate, channel interaction (alone/simultaneous/offset/
  rapid-switching) — logging every command's time, latency, and errors to
  a history. Only measures what's objectively obtainable without a sensor
  at the device (write-round-trip latency, value acceptance); the report
  says explicitly what it did not measure (felt intensity, true physical
  rise/fall time) instead of faking a number nobody collected.

- **KI-Trainingssystem tab**: collects training data for a custom YOLO
  region-detection model directly from videos run through the app, instead
  of only via the standalone `bootstrap_yolo_dataset.py`/`train_yolo_model.py`
  CLI scripts. Mark one or two regions in a video (with a class name each,
  e.g. "brust"/"hand") to bootstrap labeled samples via classical tracking;
  a review grid (image + box overlay) lets you discard wrongly tracked
  samples before they reach training — `bootstrap_yolo_dataset.py` itself
  documents that a detector trained on unreviewed tracking output never
  gets more reliable than the tracker that produced it, so this review step
  is not optional. A dataset-summary table shows sample counts per class
  and split; "Training starten" runs `train_yolo_model.py` (needs a
  CUDA-capable GPU for realistic training times) and drops the exported
  model directly at the AI model path, where it's usable in the Generator
  tab's "KI-Erkennung" without any further step. Purely additive — the
  existing manual region-marking and Tf/Tj workflow in the Generator tab is
  unchanged.

### Changed

- **Motion-axis selection is automatic by default** (`--axis auto`, GUI:
  "Bewegungsachse" now defaults to "Automatisch"). Horizontal and vertical
  position were already tracked in parallel in every backend (CSRT,
  grid_lk, and now flow too — see below); only which one made it into the
  funscript was a fixed, manually-chosen flag, which made no sense once
  both are measured for free. `auto` now picks whichever axis has the
  clearly larger range of motion; forcing `x`/`y` by hand remains available
  for the rare case where the automatic choice is wrong. The `flow`
  backend's camera-shift correction was extended to track both axes of the
  RANSAC-estimated shift (previously only the vertical translation was
  read from the affine matrix) — needed so a horizontal auto-selection
  gets the matching camera correction instead of none.

- **Tf and Tj merged into a single profile** in the generator's dropdown
  (`Tf/Tj (Abstand + Sog)`) — both were already the identical recipe
  internally (`funscript/recipe.go`'s `NormalizeProfile` always collapsed
  them), so offering two identical-behaving options was pure friction.
  Marking a second region in the generator now auto-selects this profile
  (no other tracking method evaluates a second region, so drawing one
  already *is* the selection) instead of requiring a separate manual
  dropdown pick. The CLI's `--profile` still accepts both `tf` and `tj`
  for backward compatibility.

## [0.3.0] — September 16, 2026

### Added

- **Golden-Clip Benchmark**, with a dedicated GUI tab (Bench): runs a
  fixed, user-maintained set of real clips through the actual pipeline and
  tracks Quality-Doctor score and (where a FunGen reference exists)
  agreement over time — a repeatable measurement basis instead of one-off
  numbers in chat.
- **Local ROI-detection training tooling** (`bootstrap_yolo_dataset.py`,
  `export_yolo_onnx.py`): bootstraps a YOLO training set from existing CSRT
  tracking and exports a compatible ONNX model for `ai_roi.py` — no model
  bundled, training stays something each user runs against their own
  material. See `docs/AI_ADAPTER.md`.
- **AI detector can propose two separated regions**, not just one
  (`ai_roi.select_two_best_boxes`/`find_two_rois`) — standalone CLI only so
  far, not wired into generation; never measured against real material.
- **Audio-tempo plausibility check** (`--audio-check`), classical
  (no AI), wired into the generator tab.
- **Secondary O-markers**: suggested automatically alongside the primary
  one, both from the manual "O-Zone vorschlagen" button and the
  generation-time auto-apply checkbox.
- **`grid_lk` tracking backend** (grid of independently tracked
  Lucas-Kanade points, median as position) — ~15x faster than CSRT,
  measurably more robust on small/hard regions; wired into the GUI
  (`#gen-backend` dropdown).
- **Contact-triggered vibration** for Tf/Tj (opt-in): vibration follows
  the ROI1↔ROI2 distance's own top slice.
- **Manual funscript curve editor**: drag to move, click to add,
  double-click to delete points; video follows the point being dragged.
- **Manual and automatic O-Zone/O-marker placement**, ring-down, and
  polarity suggestion (`SuggestPolarity`/`SuggestOZone`/`RingDown`).
- **Script Doctor** for imported `.funscript` files that never went
  through generation (no tracking data), marked
  `estimatedFromScriptOnly`.
- **Training session history** view (past sessions summarized: cycles,
  mean peak intensity, interruptions, mean feedback).
- **SAM Motion Model, first milestone** (`sam/` package: `Script`/
  `Frame`/`Motion` types, funscript roundtrip converter) — not yet wired
  into any GUI/CLI flow; see `docs/SAM_ARCHITECTURE.md`.
- **Local AI adapter**, all three planned steps: region proposal, profile
  suggestion, quality second opinion — each is a proposal only, the
  classical pipeline still measures/decides in every case. See
  `docs/AI_ADAPTER.md`.
- Right-hand sidebar (device/script/quality), dark UI redesign,
  "Advanced settings" collapse for rarely-changed generator options.
- `docs/SAM_NEO_2_RESEARCH.md`: source-tiered hardware research.

### Changed

- `auto_roi`'s two-ROI candidate search reworked (`_peak_regions`,
  region-bounds capping) — real improvement on synthetic scenes, still
  not good enough on real material to become the GUI default (see
  `docs/NEXT.md` priority 3).
- Flow backend's camera correction switched from correcting flow vectors
  to correcting positions (panning correlation 0.744 → 0.805).
- Two-point distance measurement now uses the full 2D center-to-center
  distance, not just the Y coordinate.

### Fixed

- `--backend` was silently ignored whenever `--roi2` was set (always fell
  through to CSRT regardless of the flag).
- Update-check failures were silently swallowed (empty `.catch`); now
  logged, with a manual "check now" button showing the real result.
- Dropping multiple video files silently discarded all but the first.
- Device keepalive only maintained the last-sent channel — vibration or
  suction could stop being refreshed during a pause, depending on which
  was sent last.
- `shutdown()` didn't stop/disconnect the *active* session device, only
  the Geräte-tab test connection; `Stop()` `lastPacket` bookkeeping and an
  `activeDevice` data race were also fixed in the same pass.
- SAM funscript roundtrip silently dropped `Profile`/`DeviceRecipe`
  (a Tf/Tj script would fall back to Hub-style mapping on real hardware).
- Several race conditions: the playback video-sync channel, a TOCTOU gap
  in the device test-connect guard, context cancellation during a
  training hold, and the generator's stderr-reading goroutine vs.
  `cmd.Wait()`.

### Investigated and explicitly not shipped

(kept here so the same unsuccessful approaches aren't retried without new
evidence — full measurements in `HANDOFF.md`'s "Tested and rejected" table
and `docs/NEXT.md`)

- KCF/MOSSE trackers as a faster CSRT replacement — real speedup, but
  near-total tracking collapse on small/hard regions.
- Two-tracker CSRT parallelization for Tf/Tj — no net gain in the real
  pipeline despite a positive isolated microbenchmark.
- Gentle upscaling (1.15x–1.3x) before tracking a hard ROI — unstable,
  not a clean dose-response.
- WebGL/Canvas sharpening of the player's `<video>` element — measurably
  *less* sharp than doing nothing; the loss happens at frame extraction,
  before any shader runs.
- A C#/.NET/Avalonia rewrite — stays Go + Wails.

## [0.2.2] — September 14, 2026

Last tagged release before the entries above. See GitHub Releases for the
published binaries and `git log v0.2.1..v0.2.2` for the exact commit list.
