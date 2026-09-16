# Changelog

Format loosely follows [Keep a Changelog](https://keepachangelog.com/).
Curated and grouped by theme, not a raw commit dump — see `git log` or
GitHub for the exact PR-by-PR history. `docs/NEXT.md` carries the detailed
measurement history behind each entry; this file is the short version for
"what changed", not "why" or "how it was measured".

## Unreleased

### Added

- **Playback tab: pressing the native video Play control now also starts
  funscript/device playback**, instead of these being two separate actions
  (the dedicated "Abspielen" button and the video's own play control).
  Guarded against re-triggering itself when video-sync makes the app call
  `videoEl.play()`. Doesn't reset the video to the start, unlike the
  dedicated button - the video is already running at its current position
  when this fires.
- **`generator/trackcv`, a Go-native CSRT tracking package (experimental,
  not yet wired into the app)**: the first step of an evaluation, per the
  user's request, of writing more of the generator in Go instead of
  Python where it measurably helps. Ports `track_roi()` (CSRT + scene-cut
  detection + camera compensation + appearance memory) to Go via a small
  custom cgo wrapper around the system OpenCV C++ library - not the
  `gocv` package, whose bundled `contrib` bindings don't compile against
  Ubuntu's OpenCV build (it excludes `xfeatures2d` for patent reasons,
  and `gocv` couples that into the same compilation unit as the tracker
  it also needs). GEMESSEN before writing a line of the real port: a
  minimal feasibility test (CSRT alone, same synthetic clip, same ROI,
  Python's `create_tracker()` vs. this Go wrapper) gave r=0.9996
  correlation, max. 3px difference, ~15% less wall time (same C++ tracking
  work either way, just without a Python interpreter per frame) - see
  docs/NEXT.md for the numbers. Own Go test suite (no Python/ffmpeg
  dependency) with the same "synthetic video, known ground truth" pattern
  already used by the Python backend tests. Two real memory bugs were
  caught and fixed while getting that suite green: a missing `channels`
  array in a `cv::calcHist` call (undefined behavior, not a "works on my
  machine" issue) and a tracker double-free from Go's `defer` capturing a
  pointer value at the `defer` statement rather than at the deferred
  call. Deliberately NOT the generation path yet - `generator.go` still
  calls `generate_funscript.py` for everything (smoothing, keyframes,
  Quality Doctor, funscript writing stay Python); this package covers
  only the single most expensive sub-step. CI's Go job now installs
  `libopencv-dev` to build/test it.
- **`region_fusion_auto` tracking backend**: like `region_fusion`, but
  without a marked region - automatically divides the whole frame into a
  2x2 grid (4 zones), the way `flow` needs no marked region either.
  Normalizes each zone's tracked position to that zone's own box before
  fusing (not the raw pixel blend `region_fusion` uses, which only makes
  sense for a small, already-localized marked region) - see docs/NEXT.md
  for why. Not available for two-point (Tf/Tj) measurement, same as
  `flow`/`region_fusion`. NOT yet measured against a reference - a
  candidate, not a result.

## [0.5.0] — September 16, 2026

### Added

- **`region_fusion` tracking backend**: splits the marked region into a
  2x2 grid (4 sub-regions), tracks each independently, and fuses them
  per-frame into one signal weighted by each sub-region's current motion
  strength — instead of a single box (`csrt`) or one point grid spanning
  the whole region (`grid_lk`) that implicitly averages in whatever part
  of the region is currently still. Selectable via "Tracking-Verfahren"
  in the Generator tab, or `--backend region_fusion` on the CLI. Not
  available for two-point (Tf/Tj) measurement — falls back to CSRT there,
  same as `flow`. GEMESSEN on real material against a FunGen2 reference:
  at least on par with `csrt`, clearly ahead of both `csrt` and `grid_lk`
  in one of two tested segments — see docs/NEXT.md for the numbers and an
  important caveat about clip/reference timing alignment.

### Fixed

- **CSRT tracker crash, second variant.** The earlier fix only covered two
  of the three known OpenCV API shapes for creating a CSRT tracker
  (`cv2.legacy.TrackerCSRT_create()` and `cv2.TrackerCSRT_create()`). A
  real `opencv-contrib-python` install had neither, only the newer
  class-based `cv2.TrackerCSRT.create()` — `create_tracker()` now tries
  all three in order and raises a clear error naming the installed
  OpenCV version (instead of a raw `AttributeError`) only if none of them
  exist.

## [0.4.2] — September 16, 2026

### Fixed

- **YOLO training used a fixed batch size (16)** regardless of the GPU's
  actual VRAM. On a lower-VRAM card (e.g. a 4GB GTX 1650) this could hit
  "CUDA out of memory" even though the default model (`yolov8n`, the
  smallest) would fit fine with a smaller batch. Now passes `batch=-1` to
  ultralytics, which auto-sizes the batch to target ~60% of the free GPU
  memory instead of a hardcoded value. No effect on CPU training (falls
  back to the same fixed default there).

## [0.4.1] — September 16, 2026

### Fixed

- **"Training starten" crashed with a raw `ModuleNotFoundError` traceback**
  when `ultralytics` wasn't installed (it's a separate, heavy optional
  install on top of the base requirements — `requirements-ai-train.txt`,
  pulls in PyTorch). The button now checks availability up front (same
  `--check` pattern the KI-Erkennung region finder already uses for
  `onnxruntime`) and stays disabled with a clear `pip install` hint instead
  of offering something that then fails.
- **Bootstrapping training samples with two regions crashed on Windows**
  with `AttributeError: module 'cv2.legacy' has no attribute
  'TrackerCSRT_create'`. `track_roi()`'s initial tracker creation called
  `cv2.legacy.TrackerCSRT_create()` directly instead of going through the
  existing `create_tracker()` helper (which already handles this exact
  OpenCV version difference and is used everywhere else a tracker gets
  re-created, e.g. after a scene cut) — the one remaining unguarded call
  site.
- **Starting playback/training while a device was connected via the Device
  tab required manually disconnecting there first**, even though both use
  the same physical device and BLE only allows one connection anyway.
  Playback/training now reuse that exact connection instead of demanding a
  redundant disconnect-and-reconnect cycle; disconnecting from the Device
  tab while a session is actively using that connection is now itself
  rejected (rather than silently pulling the connection out from under a
  running session), symmetric to the existing rule that a session can't
  start a competing test connection.

## [0.4.0] — September 16, 2026

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
