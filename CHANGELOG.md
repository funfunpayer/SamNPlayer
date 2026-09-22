# Changelog

Format loosely follows [Keep a Changelog](https://keepachangelog.com/).
Curated and grouped by theme, not a raw commit dump — see `git log` or
GitHub for the exact PR-by-PR history. `docs/NEXT.md` carries the detailed
measurement history behind each entry; this file is the short version for
"what changed", not "why" or "how it was measured".

## Unreleased

### Added

- **Stroke preview in Generate:** Stage A pre-pass before track
  (`STROKE_PREVIEW` progress lines) — audio gate when weak/unstable, optional
  peak-distance bias, `metadata.stroke_preview` stamp. CLI `stroke-preview`
  unchanged. First measure: `clip_ausschnitt` ~1.1s, peak-overlap 0.78 vs hub.
- **Everyday Generate (FunGen-like):** after choosing a video, auto-find tip
  ROI → CSRT Stroke + Contact (measured first choice). Generate without a
  painted box triggers the same find. 4-zone demoted to advanced. See
  `docs/EVERYDAY_GENERATE.md`.
- **Review → Improve:** trim start/end, fill gaps (optional audio-tempo
  spacing), audio check on/off — FunGen-like polish after CSRT. Dead Advanced
  knobs removed (AI second opinion, audio row moved to Review).
- **Play curve = FunGen-like dots + soft stroke:** keyframe dots always drawn
  (incl. during play); live playhead height marker; Edit curve still for
  drag/add/delete. Improve reloads Play in review mode.
- **Bugfix / Go-first / license:** AutoDetectROI stamps `videoPath`+`seq`
  (stale finds dropped); post-generate auto fill-gaps; stroke-preview no
  longer mutates peak distance; license gates wired (`MaxOutputMs` trial,
  `.samn` Play block) — Enforcement still off. Review opens Play with dots,
  Edit opt-in.
- **Optimize for Neo 2 (Play):** one-click import path — fill gaps → Contact
  → bake vibe/suction → `.samn`. Multi-axis editor then tunes channels.
  Ballast trimmed (AI opinion / Flow downscale GUI gone; scene memory collapsed).
  See `docs/CONTENT_SOURCES.md`.

### Fixed / observability

- **Flow duration traced:** Farneback ~36 ms/frame at downscale 0.5 → ~56 s
  for 50 s 720p (not a hang). Always-on `FLOW_SUMMARY` stderr line.
  Unset/`0` flow downscale now defaults to **0.5** (CLI). Flow stays
  CLI-only — Everyday GUI no longer shows a downscale control.

## [0.5.21] — September 21, 2026

### Added

- **Flow soft observability:** `FLOW_WALL_WARN` / `FLOW_STALL_WARN` on stderr
  (defaults 120s / 60s; set `FLOW_BACKEND_WALL_WARN_S=0` to silence). Registry
  Flow path now forwards `on_progress`. No hard timeout / no GUI Flow yet.

### Changed

- **GUI profile rename (TFTJ step 5):** Stroke (Normal) / Soft (rings) /
  Autotune — wire values `standard`/`weich`/`autotune` unchanged.
- **Feel-decouple (TFTJ step 6 partial):** Play can save/preview Contact vib
  on Stroke scripts; `SuggestPipeline` two-ROI → `standard` (not `tf`).
  Zone2 distance on stroke still scaffold-only.
- Flow CLI help: retired false “4× faster / ~18ms” claims; prefer
  `--flow-downscale 0.5` on 720p.

### Measured

- **TFTJ step 4 (Claude `clip_ausschnitt`):** no-mark 4-zone windowed mean
  FunGen r=0.363 vs tip CSRT 0.468 vs committed hub 0.590 (ohne_yolo).
  Contact vib does not change stroke actions. **Do not default 4-zone.**
  Details: `docs/TFTJ_PROFILE_DIRECTION.md` § Step 4 measure.
- **Method matrix (same clip):** tip-matched `region_fusion` 0.562 /
  `grid_lk` 0.494 / flow×0.5 0.231 windowed vs ohne_yolo. Full-res flow
  incomplete at ~180s. Soft wall warn works. See TFTJ § Method matrix.

## [0.5.20] — September 21, 2026

### Added

- **Whole-frame 4-zone motion (GUI):** Generate → “Track whole-frame motion
  (4 zones)” / Advanced backend `region_fusion_auto` — no hand mark; splits
  the frame into 2×2 and tracks motion (Python). Pair with Contact vibration.

### Changed

- **Contact-first (no Tf/Tj required):** product profile dropdown is Stroke /
  Soft / Autotune only. Contact vibration (default on) is the feel layer.
  Zone 2 is optional and no longer switches to Tf/Tj or blocks Generate.
  Legacy tip↔partner distance stays CLI `--profile tf|tj`.

## [0.5.19] — September 21, 2026

### Added

- **Motion candidates (TFTJ 4b):** Generate → “Show motion candidates” lists
  ranked classical motion regions as dashed overlays; click one to set Zone 1
  only. Zone 2 is never auto-filled. CLI: `auto_roi.py --list` (#167).

## [0.5.18] — September 21, 2026

### Fixed

- **#145 Autotune + Zone 2:** stroke profiles (`autotune` / `standard` / `weich`)
  ignore Zone 2 and track tip ROI only; distance partners require Tf/Tj.
  Avoids two-point+bandpass “no discernible motion” crashes from mixed UI state.
  Clearer posttrack error when the tip curve is flat (#164).

### Added

- **F-003 option 1:** `LagCorrelation.AliasingRisk` (+ `DominantPeriodMs`,
  `AlternateLagsMs`) — flags near-tied lags at ±k×stroke period without
  changing reported lag/r. CLI `phase` / compare report surface it.
  (`docs/FINDINGS_TIMING_TF.md`, #163). Note: flag does not yet fire on
  committed goldens — period detector floors ~100–140ms (#165).

### Changed

- **TFTJ step 3:** Tf/Tj always needs **two markers** (tip + Zone 2). Contact
  vib on → Zone 2 partner is **tracked by default** (GUI no longer
  auto-checks “Fix Zone 2”). Regression: simpletrack fixed vs tracked
  partner on synthetic motion (`docs/TFTJ_PROFILE_DIRECTION.md`, #160).

## [0.5.17] — September 21, 2026

### Added

- **Contact vibration on Normal / Autotune / Soft:** same depth-based
  envelope as Tf/Tj (high pos → vibe). GUI shows the toggle for every
  profile, **on by default**, user can turn off. CLI `--contact-vibration`
  now applies to stroke profiles too (writes `device_recipe`). See
  `docs/TFTJ_PROFILE_DIRECTION.md` step 2 (#149).
- **Docs:** `docs/AGENT_COORD.md` — shared Cursor ↔ Claude ↔ ChatGPT board
  (lanes, sprint order, handoff template) (#151).
- **Docs/data:** Tf/Tj golden set `clip_voll` + windowed compare tool;
  `clip_ausschnitt` FunGen refs (provenance corrected) (#140, #148, #150).
- **CLI:** `phase --window-ms` — per-window BestLagCorrelation (#143).

### Changed

- **Docs:** `docs/TFTJ_PROFILE_DIRECTION.md` — Tf/Tj as device profile
  (not dual-ROI recognition); contact vib on Normal/Auto; partner mark
  only for Tf/Blow when vib on; later AI profile suggest (#147).

## [0.5.16] — September 21, 2026

### Changed

- **Generate OpenCL:** no GUI checkbox — when the Python path runs, OpenCL
  is tried automatically and only logged (`OpenCL active` / not available).
  Checking the old toggle used to force Python and skip Go CSRT; that trap
  is gone. CLI `--opencl` is a deprecated no-op; `--no-opencl` disables (#141).

### Fixed

- **CI Windows OpenCV:** install `libgfortran` before `opencv` so pacman
  does not silently skip OpenCV after MSYS2 renamed `gcc-libs` → `cc-libs`
  (#142).

## [0.5.15] — September 20, 2026

### Fixed

- **Tf/Tj multi-partner fusion:** lost tracked partners no longer feed
  stale boxes into `min()` distance; tip loss / no usable partner holds
  the last good distance (F-006 / star-topology interim) (#137).

## [0.5.14] — September 20, 2026

Tf/Tj contact zones: tip-end distance, named Zone 1/2/3+, Go multi-partner.

### Changed

- **Tf/Tj tip→contact distance:** measure from the tip-box point nearest
  the partner (not tip center), so a whole-penis mark still registers
  glans-end contact; GUI names Zone 1/2/3+ for tip + nipple contacts (#134).
- **Tf/Tj Zone 3+:** optional body-part class on extras (`--target x,y,w,h,class`
  / GUI Zone 3+ class); Go CSRT `TrackMultiPoints` runs N contact partners
  without Python (soft masks still Python) (#135).

## [0.5.13] — September 20, 2026

Windows Generate on **Go CSRT** (no Python): MSYS2 OpenCV 5 in CI/release,
portable zip ships OpenCV DLLs beside the GUI.

### Changed

- **Windows OpenCV CSRT (G0.1):** `trackcv` on Windows with `-tags opencv`;
  required CI `Go (OpenCV Windows)`; release builds Windows on
  `windows-latest` and collects MinGW OpenCV DLLs under MSYS2
  (`docs/WINDOWS_OPENCV.md`, `scripts/collect-mingw-opencv-dlls.sh`).

## [0.5.12] — September 20, 2026

Sequential Generate UI + production spine docs. Clip gate: full Go CSRT
on test clip (1199 frames, Quality Doctor passed). Windows in-binary
OpenCV CSRT remains **next** (G0.1 → 0.5.13).

### Added

- **Generate sequential workflow (GUI):** steps appear one after another —
  Video → Region → Motion → Generate → Review. Next panel unlocks only
  when the previous action is done (Tf/Tj waits for ROI2). Classic CSRT
  first; AI stays optional on the region step.

### Docs

- **Sequential production spine (G0→G4):** stabilize CSRT → classical
  heuristics + audio → Neo 2 raw-value mapping → AI helpers → license/
  platforms. License stays developer mode through G2.
  Spec: `docs/PRODUCTION_ROADMAP.md`, `docs/GENERATE_HEURISTICS.md`.

## [0.5.11] — September 20, 2026

Quality + AI Train recovery after 0.5.9/0.5.10 Windows pain (#119/#120).
Website/presentation deferred.

### Fixed

- **AI Train bootstrap OpenCV (#119):** after **Install AI train deps**,
  ultralytics’ `opencv-python` dependency no longer leaves CSRT dead —
  GUI/pip path uninstalls non-contrib wheels and reinstalls
  `opencv-contrib-python`, then re-checks trackers.
- **WindowsApps Python preference:** Store stubs under
  `%LOCALAPPDATA%\Microsoft\WindowsApps` are demoted when a real
  install exists, so bootstrap errors and pip hints point at
  `Programs\Python\…` instead of the stub (#94/#119).
- **Generate product path (#120):** one strong tracker — **CSRT**.
  Go CSRT when OpenCV is linked; otherwise **Python CSRT** is the
  Generate path (Windows today), with a hard error if missing — not a
  soft degrade to NCC. `simpletrack` is lab/CLI (`PreferSimpletrack`)
  only. Next: Windows in-binary OpenCV CSRT so Generate needs no Python.
- **Progress display (Generate + AI Train):** Go tracking reports percent
  during the run; AI Train wires `PROGRESS` to a bar; scrolling run logs
  in both tabs.
- **Status copy:** ROI training readiness strings in English; install
  success reports whether CSRT bootstrap is ready.

### Docs

- `docs/KI_TRAINING.md`: restore-contrib note + 0.5.9/0.5.10 manual
  workaround.
- `docs/ENGINE.md` / production roadmap: Windows OpenCV CSRT is next
  (true Go parity); website polish deferred.

## [0.5.10] — September 20, 2026

Careful bugfix + lean GUI motion + multi body-part regions for Tf/Tj /
Generate / AI training. Tag when GitHub Actions billing is restored and
Checks are green on `main`.

### Fixed

- **AI Train empty dataset:** starting train without samples no longer
  ends in a raw German `Keine data.yaml` traceback. Go/Python fail fast;
  GUI warns to mark region(s) → **Use for training** first (#117).
- **ffmpeg console flash (Windows):** `videox` now hides the console
  window when spawning ffmpeg/ffprobe (same as Python helper processes),
  so Generate/Play no longer blink a black popup.
- **Update check noise:** missing/empty GitHub release is “no update”,
  not a startup warning. No browser `confirm()` popup — in-app banner /
  Settings button instead.
- **Weak generator backends off the product GUI:** Flow / grid_lk /
  region_fusion* removed from Advanced dropdown (CLI `--backend` only).
  SuggestPipeline always picks CSRT → Go path (fewer dependencies).
  Tf/Tj, masks, multi-target, AI ROI, Quality Doctor stay.
- **Session guard:** rejected second StartPlayback/StartTraining no longer
  overwrites the live session’s `activePlayer`/`activeDevice`.
- **Generate overwrite:** also refuse when a companion `.samn` exists
  (matched `ScriptExistsForVideo`).
- **Contact intensity 0:** slider value `0` no longer coerced to `1`
  (`finiteOr` instead of `x || 1`).
- **English operator UI:** analysis summary, playback/generator/device
  status strings, videox convert progress, native pipeline progress —
  leftover German user strings → English.
- **license-tool:** unknown `--tier` rejected; internal/invite issue path
  simplified.
- **bootstrap_yolo_dataset.py:** Tab/space mix that broke import (CI Python
  job would fail once runners start again).

### Added

- **Multi body-part regions:** English taxonomy
  (`face`, `mouth`, `breasts`, `nipples`, `hand_1`, `hand_2`, `penis`,
  `glans`, `vagina`) shared across Go/Python/GUI (`docs/BODY_REGIONS.md`).
  AI training marks up to **9** classes/frame; Generate Tf/Tj gets ROI
  class picks + **Fix ROI2** (static contact target); preferred-classes
  default lists all nine IDs. Legacy DE labels (`brust`, `eichel`, …)
  normalize.

### Changed

- **GUI lean motion polish:** CSS-only brand/rail/tab/progress animations,
  primary gradient hover, empty-Play drift, device LED pulse; respects
  `prefers-reduced-motion`. No layout redesign.

## [0.5.9] — September 19, 2026

Production train after v0.5.8: native `.samn`, leaner Go generation path,
portable ffmpeg, English GUI lock, production roadmap for owner testing.

### Added

- **Native script `.samn`:** source of truth for Sam Neo 2 (general +
  vibration + suction curves, recipe vs axes drive, chapters/bookmarks,
  **O-markers**, contact-vibration recipe, strength presets soft/normal/strong).
  Community `.funscript` remains import/export. Docs: `docs/SAMN_FORMAT.md`.
  Playback UI: curve channel, drive mode, bake axes, export funscript / save `.samn`.
- **Go audio-tempo check** (`generator/audiocheck.go`): post-hoc on the
  native pipeline; `--audio-check` no longer forces Python.
- **ROI second-pass** (`VerifyROI`): after auto-detect, warn when motion
  concentration in the box looks weak — never auto-rewrites the region.
- **Player proxy:** remux-first for H.264 in awkward containers; soft
  Lanczos downscale when re-encoding above 1920px; more pick/MIME formats
  (ts/m2ts/flv/mpg/3gp/ogv). Analysis GrayReader uses Lanczos (was bilinear).
- **ffmpeg without system install:** `videox` resolves a copy next to the
  app / user tools dir; Settings → **Install video tools** (opt-in);
  release ships **portable** archives with ffmpeg beside the GUI
  (`docs/FFMPEG_TOOLS.md`).
- **Platforms stake:** desktop Win/Linux now; macOS when a Mac builder
  exists; later **iOS/Android player-only** (no Generate) —
  `docs/PLATFORMS.md`, `docs/COMPETITIVE.md`. Runtime health shows
  CPU / GOMAXPROCS (honest budget, not fan-max).
- **Lean self-build principle** (`docs/SELF_BUILD.md`): self-build only
  when clip tests show **equal or better** quality — never worse; fewer
  deps after that gate. ISO-BMFF probe is **fallback only** (ffprobe
  preferred); geometry matched against ffprobe on synthetic clip.
- **Production roadmap** (`docs/PRODUCTION_ROADMAP.md`): open workstreams,
  owner test loop, targets and prerequisites.

### Changed

- **`NativePipelineEligible`:** `AudioCheck` no longer blocks the Go path
  (AI quality opinion still does).
- **GUI English lock:** leftover German user strings (playback convert
  banners, ROI labels, ROI training hints, Thanks/feedback) → English.

### Docs

- **License system concept** (not built / not sharp): yearly key, trial =
  1‑minute generate + funscript-only play; Ed25519 signed files;
  `docs/LICENSE_SYSTEM.md`.
- **Audio workflow / player verdicts:** soft CSS/WebGL upscale still
  rejected (measured worse); proxy remux + Lanczos downscale shipped
  instead (`docs/AUDIO_WORKFLOW.md`).

## [0.5.8] — September 19, 2026

Playback UX release after #102. Green CI + clip7776 / full frontend suite
before tag (`CONTRIBUTING.md` release gate).

### Added

- **Playback:** hover tooltips on the funscript curve and intensity heatmap
  (time + position / intensity). Playlist **Shuffle** (reshuffles queue
  when enabled; reshuffles again on repeat wrap) and **Repeat playlist**
  (auto-advance loops the list). Toggle state kept in session storage for
  the app run.
- **Docs:** release gate in `CONTRIBUTING.md` — bugfix + clip7776 before
  every tag.

## [0.5.7] — September 19, 2026

Follow-up after v0.5.6: finish English operator-facing logs, ship
experimental depth/pose supporting signals (opt-in only), and a small
bugfix pass (clip7776 + bindings).

### Changed

- **Operator-facing logs / errors / file dialogs** in English
  (`docs/LANGUAGE.md`). Structured log keys aligned where touched
  (`fehler`→`error`, etc.). Historical code comments may stay German
  until touched.

### Added

- **Experimental depth / pose supporting signals** (`docs/DEPTH_POSE.md`):
  classical relative-depth proxy + optional ONNX stubs; soft-rank AI ROI
  candidates only when `SAMNPLAYER_DEPTH_RANK=1`. Never writes a
  Funscript alone; no bundled model. Golden-clip win still required
  before any default.

### Fixed

- Export `SupportSignalsAvailable` in wailsjs bindings.
- Leftover German UI strings found in clip7776 / playback bugfix
  (heatmap/project toasts, range-mark warnings, ROI-training status,
  generator workflow tip, benchmark hint).

## [0.5.6] — September 19, 2026

Polish release after English UI/docs alignment and a selective GUI review.
CI green on tip before tag.

### Changed

- **Product language:** user-facing UI and docs aligned to **English**
  (`docs/LANGUAGE.md`). Prefer English for new strings and documentation;
  keep Go/Python/JS where each fits (no speculative rewrites).

### Added

- **KI-Training:** one-click `InstallRoiTrainingDeps` (embedded
  `requirements-ai-train.txt` in release binaries); split status
  (Python / OpenCV / ultralytics); still samples get a val split;
  review **correct box**; guides `docs/KI_TRAINING.md` +
  `docs/PLAYER_CODECS.md`.
- **Class-aware AI ROI:** `preferred_class_ids` /
  `--ai-preferred-classes` / Settings preferred classes; `classes.json`
  copied next to `.onnx` on train; two-ROI pick prefers distinct classes.
- **Bootstrap box scale:** `--box-scale` / GUI field for slight YOLO pad.
- **Playback codecs:** ffprobe playability check, “Make playable”
  H.264/AAC proxy via ffmpeg, `<video>` error banner, correct MIME on
  local video serve; player key hints + transport polish.
- **GUI a11y / desktop basics:** window `MinWidth`/`MinHeight`, tablist
  ARIA + arrow-key nav, non-blocking ERROR toast with “Open Log”,
  spacing/focus tokens. Deliberately **not** taken: light mode,
  frameless chrome, glassmorphism, skeleton screens, disconnect confirm
  (see review notes in PR #100).

### Fixed

- Train button no longer requires OpenCV when only ultralytics is missing
  (and vice versa — clear status text).

## [0.5.5] — September 18, 2026

Release after stacking GUI/startup, OFS editor tools, OpenCV 5 tracker
fallback (#95), and funscript autotune workflow. Green CI on tip before tag.

### Fixed

- **OpenCV 5.0 CSRT** (#94/#95): `create_tracker` / package gate try CSRT then
  KCF/MIL under `cv2` and `cv2.legacy` — generate no longer dies on Windows
  opencv-contrib 5.0.0.
- **CapSpeedRange editor stale state:** after speed-cap / range-delete, reload
  duration + `rawActions` so edit mode cannot overwrite the capped file.
- **ROI redraw vs backend:** manual Flow/grid_lk/etc. survive ROI re-draw
  (`userTouched` on backend/profile).
- **bandpass NameError:** parse `--bandpass-hz` inside `process_one`.
- **CapSpeedRange overlap:** later segments use original timestamps.
- **Playlist:** remove-active reloads highlight; video end advances via
  `stop({user:false})`; connect-fail `playback:done {failed:true}` does not
  auto-advance.
- **OFS bookmarks/chapters:** accept float seconds.
- **CI Go (OpenCV):** `ptpFloat` redeclaration in `trackcv`.

### Added

- **Profil `autotune`:** detrend + bandpass 0.5–4 Hz + speed 400; GUI Max Speed /
  Flow-Downscale; workflow tip Flow→CSRT→Autotune→Audio-Check
  (`docs/FUNSCRIPT_ALGOS.md`).
- **OFS-inspired Go tools:** max-speed highlights, chapters/bookmarks metadata,
  heatmap PNG, `.snp.json` projects, FPS snap, range delete/speed-cap.
- **Clean SamNPlayer wordmark:** Space Grotesk 700, amber + teal N.

### Changed

- **Extended-O:** scales curve amplitude only (vib/suc × factor); rhythm/shape
  unchanged — no flat freeze and no video/timeline pause during hold.
- **Startup:** creates missing SamNPlayer folders (logs/sessions/models/cache/…)
  and checks dependencies (ffmpeg required, python/ffprobe optional).
- **Connect smoke test:** optional short vib/suc pulse after connect
  (`device.connect_test`, default off) — Settings checkbox.
- **Wiedergabe layout:** media-first (video/curve/heatmap + transport), tools
  beside; empty-state CTA; script-alone without video supported.
- **Kontakt-Vibration:** Tf/Tj defaults to on (soft curve ≈ touch); after
  generate auto-opens Playback for review/edit; save contact settings into
  script metadata; fewer post-generate popups.
- **GUI visual refresh:** Space Grotesk + logo palette (gold `#f2b03d` /
  teal `#3dccc0` on deep navy); sharper geometry, dual-tone rail/brand
  offset, SamN wordmark in topbar; violet accents removed (ROI2/Sog → gold/teal).
- **Topbar device:** connected device name always visible top-right (icon +
  LED); **Verbinden / Trennen** there (uses saved transport from settings);
  click name opens Gerät tab for transport/tests/diagnostics.
- **Go-native Tf/Tj:** two-point distance tracking runs in Go (`trackcv` /
  `simpletrack.TrackTwoPoints`) — no Python soft-fallback when the Go path
  is eligible. `tracking_gaps` written into funscript metadata for Kontakt mute.
- **Auto pipeline:** marking ROI(s) preselects backend + profile
  (`SuggestPipeline`); Tf/Tj + CSRT → Go path by default.
- **KI-Training:** seek past black intro, up to 4 marks/classes, still-image
  samples, sample stride, audio WAV sidecar; class presets + chips from
  collected dataset. Seek offset now reaches generate + bootstrap pipelines;
  WebP/HEIC stills convert via ffmpeg.
- **GUI brand:** Wails logo replaced with SamNPlayer SN mark; seek controls
  on Generator + KI-Training tabs.
- **Generator seek:** `StartTimeSec` / `--start-seconds` wired through Go
  trackers (trackcv/simpletrack) and Python generate path; funscript
  timestamps and `tracking_gaps` are offset to absolute video time; bootstrap
  audio respects the same seek.

## [0.5.4] — September 18, 2026

Test release for Kontakt-Vibration + SAM runtime. `.funscript` stays the
file format; SAM is the internal motion model (optional `.sam` sidecar).

### Added

- **Kontakt-Vibration product controls:** sensitivity slider („früher an“ /
  „nur tief“), curve shape (linear / soft / peak), playback checkbox
  „Kontakt-Vibration ab“ (no regenerate), orange second track on the
  position curve.
- **Kontakt-Vibration signal quality:** mute vibration during tracker-loss
  windows (`metadata.tracking_gaps`), short contact-envelope smoothing,
  opt-in two-ROI auto-suggest on Tf/Tj.
- **SAM enrich + sidecar:** `sam.FromFunscriptEnriched` + CLI
  `SamNPlayer sam FILE.funscript`; Tf/Tj generate writes `.sam`; playback
  prefers sidecar when it has contact Intensity.
- **SAM Densify:** tick-grid Intensity from interpolated Position (classic
  parity for contact mapping).
- **SAM RuntimeAdjust (Milestone 2 start):** live Kontakt-Stärke /
  Empfindlichkeit / Kurve without rewriting files; GUI preview via
  `GetVibrationCurvePreview`; CLI `--contact-intensity`, `--mute-contact`,
  `--contact-span`, `--contact-curve`, `--contact-extra-smooth`.

### Fixed

- SAM CLI `sam FILE --output OUT` honors flags after the path (`splitCLIArgs`).
- `opts.TrackingGaps` respected without metadata; Confidence≈0 mutes vib.
- Envelope/Smoothing no longer bleed vibration into tracking gaps.
- Thin (position-only) `.sam` sidecars are not preferred over Enrich.

## [0.5.3] — September 17, 2026

### Added

- **Log tab:** copyable in-app protocol (select text / „Alles kopieren“),
  filter warnings/errors, clear ring, open log folder. Live `log:line`
  events from the logging ring.
- **CLI `generate`:** `SamNPlayer generate --video FILE --roi x,y,w,h`
  (same Go-auto / Python-fallback path as the GUI) for headless smoke tests.
- **Docs:** `FUNGEN_FEATURE_COMPARE.md`, `RELEASE_0_5_3.md` checklist.

### Changed

- **Go generate path is automatic:** single-ROI CSRT settings (including
  default Auto-Retry) use `trackcv` or `simpletrack` without a GUI checkbox.
  Auto-Retry varies signal params in Go (same idea as Python). Special cases
  (other backends, Tf/Tj, per-scene, AI/audio, OpenCL) still use Python.
  `PreferPython` forces the old path for tests/CLI.
- **Native failure soft-falls back to Python** when a Python install is
  available (cancel/deadline still abort). GUI cancel detection uses
  `errors.Is`.
- **Generator UI** shows `Pfad: Go (…) / Python` and includes `pipeline` /
  `tracking` / `backend` on `generate:done`.

## [0.5.2] — September 17, 2026

### Added

- **Windows-ready Go generate without OpenCV/Python:** `generator/simpletrack`
  (NCC/SAD over `videox`/ffmpeg) backs `GenerateNativeSimple` when CSRT is
  not linked. Opt-in `NativePipeline` no longer requires OpenCV for the
  single-ROI case — Windows release builds can generate without a Python
  install (ffmpeg still required). Weaker than CSRT; GUI help updated.
- **`GenerateWithContext` cancel tests:** Python path (fake interpreter) and
  native simple/CSRT path with synthetic ffmpeg clips must return
  `context.Canceled`.

### Changed

- **`NativePipelineEligible`:** options/ROI only — OpenCV availability no
  longer gates eligibility; routing picks CSRT or simpletrack at generate
  time.

## [0.5.1] — September 17, 2026

### Fixed

- **CLI `phase` negative flag values:** `--max-lag-ms -500` no longer
  drops the value (custom split treated leading `-` as another option).
  Shared `splitCLIArgs` + tests; paths may still precede or follow flags.

### Changed

- **Doc cleanup (review feedback):** removed dated review dumps
  (`REVIEW_*.md`) and the multi-file `perception_update_2026-09/` pack.
  Direction lives in lean `docs/ENGINE.md`; open work in `ROADMAP.md` /
  `NEXT.md`; shipped history in `CHANGELOG.md` + code comments.

## [0.5.0] — September 17, 2026

### Added

- **GUI polish (no hardware):** Nunito + brand logo in hero chrome, soft
  page gradients, tab fade/slide transitions, device silhouette (shell +
  fill states including searching), help chips on Device/Playback, shorter
  Flow copy (~2×), soft `SuggestProfile` on generator load.
- **Native Go pipeline cancel:** `GenerateWithContext` aborts CSRT via
  `trackcv.Options.Cancel` (same Abbrechen path as Python `CommandContext`).
- **Trackcv observation contract (F-004):** `Stats.ValidFrames` /
  `Confidence` / `Reason` (+ `Result.Canceled`); written into native
  funscript metadata.
- **Dense Quality Doctor in pure Go** (`funscript.EvaluateDenseQuality`):
  rhythm / active-time / reconstruction / tracker-lost / motion-amplitude
  on the posttrack dense curve — wired into `GenerateNativeCSRT` metadata.
- **FunGen-compare CLI in Go:** `SamNPlayer compare --dataset DIR`
  (`fungen-compare` alias) — thin report on `CompareDataset` /
  `BestLagCorrelation` (Python compare half no longer required).
- **GUI help chips („?“):** Generator- and KI-Training options show a short
  popover explaining what each toggle/setting does (`help.js` + `data-help`).
  Long wall-of-text hints in advanced settings were shortened; details live
  behind the chip. Advanced options grouped (Tracking / Signal / Keyframes).
- **KI-Training device switch (auto / CUDA / DirectML / MPS / CPU):**
  `train_yolo_model.py` defaults to `--device auto` (CUDA → MPS → DirectML →
  CPU). DirectML needs optional `torch-directml` on Windows. GUI dropdown +
  `ListRoiTrainingDevices` / `--list-devices`. Full WinML inference stack
  still deferred (`docs/ENGINE.md`); this
  is the practical training-side switch.

### Fixed

- **`videox.FrameReader.Close` after normal EOF:** cancelling the
  CommandContext made `Wait` return `context.Canceled` even when ffmpeg
  finished cleanly — `TestGrayReaderScalesAndCounts` failed. Close now
  treats cancel/deadline as success unless stderr reports a real error.
- **App script-state race:** `currentScript` / `scriptPath` /
  `currentFrames` go through `stateMu` accessors (`app_script_state.go`);
  locked by `TestLoadedScriptStateNoRace` under `-race`.
- **Generation not abortable:** `GenerateWithContext` + GUI
  `CancelGenerate` / Abbrechen button kill the Python subprocess via
  `CommandContext`; native CSRT also checks `ctx` each frame.
- **Script Doctor unsorted-timestamp check:** `EvaluateScriptQuality`
  now inspects the original action order before sorting (was dead code
  after `sort.SliceStable`). `generator.ScriptQuality` reads file-order
  actions instead of `Load`/`Parse` so spliced scripts are still
  flagged. Locked by `TestEvaluateScriptQualityFlagsUnsorted` and
  Python goldens in `funscript/testdata/script_quality_goldens.json`.
- **Tf/Tj suction double-floor:** with actions already clamped to 20–90,
  applying `liftFloor(pos/100, MinSuction=0.20)` remapped resting suction
  from 0.20→0.36 and peaks from 0.90→0.92. `SyncSuctionPosition` no longer
  applies that second floor; the clamp *is* the floor. Recipe metadata
  still reports `min_suction: 0.20`. Locked by
  `TestRecipeTJSuctionNoDoubleFloor`. See `docs/FINDINGS_TIMING_TF.md`.

### Added

- **CI installs ffmpeg in the Go jobs** so `videox` tests actually run
  instead of `t.Skip("ffmpeg not available")` — that skip made the Go
  job permanently green while the same tests caught `FrameReader.Close`
  locally. Both `go` and `go-opencv` install ffmpeg now.
- **`update` package unit tests** (`IsNewer`, `AssetForThisPlatform`,
  `Describe`, URL validation) — coverage was previously ~0%.
- **`generator/posttrack` + optional Go generation path (experimental)**:
  second step of the "more Go, less Python" move. Pure-Go port of
  `positions_to_funscript` (Savitzky-Golay, percentile normalisation,
  dynamic-range lift, peak/valley + prominence, adaptive keyframes, min
  action interval, RDP via `motionx`, speed limit). GEMESSEN against
  committed Python goldens (`posttrack/testdata/positions_goldens.json`):
  identical action lists and dense-curve max abs diff < 0.05 on every
  fixture. Opt-in via `Options.NativePipeline` / GUI "Go-Pipeline
  (experimentell)" — runs `trackcv` + `posttrack` without a Python
  subprocess for CSRT + single ROI only; otherwise falls back to Python.
  Quality Doctor, AI opinion, audio check, Tf/Tj, other backends stay on
  Python. Native OpenCV linking is opt-in via `-tags opencv` (not merely
  `CGO_ENABLED=1`): default builds and Windows cross-compiles use the
  stub (`NativeTrackingAvailable() == false`) so a fresh clone without
  `libopencv-dev` still builds. Linux release binaries pass `-tags opencv`.
  Also fixed a latent Python bug: `process_one`'s
  `build()` never forwarded `--peak-prominence` / `--dynamic-range-ms` /
  `--min-action-interval-ms` into `positions_to_funscript` (so
  `--profile weich` was a no-op for those knobs) — guarded by
  `process_one_kwargs_test.py`.
- **Signal Quality ≠ Motion Fidelity** (`docs/SIGNAL_VS_FIDELITY.md`):
  Script Doctor / Quality Doctor labeled as Signal Quality (`kind:
  signal_quality`); Phase CLI / `EvaluateMotionFidelity` as Motion
  Fidelity. GUI copy updated so a high doctor score is never read as
  “matches the video”. SAM Perception v1 ordered in `docs/ROADMAP.md`.
- **CLI `phase` subcommand** (`SamNPlayer-cli phase A.funscript B.funscript`):
  runs `BestLagCorrelation` / `DiagnosePhase` without Python.
- **Phase Analyzer core in pure Go** (`funscript.BestLagCorrelation` +
  `DiagnosePhase`): port of `fungen_compare.best_lag_correlation`
  (lag search, orientation, shape-normalized error, low-confidence)
  plus the research-doc timing-vs-shape verdict. Tests mirror
  `fungen_compare_test.py`. Full 8-point PTS→device pipeline still
  deferred.
- **Script Doctor ↔ Python goldens:** `TestEvaluateScriptQualityMatchesPythonGoldens`
  locks actions-only scores against `quality_doctor.evaluate()`.

- **Engine direction** summarized in `docs/ENGINE.md` (replaces the
  dated multi-file research pack); full open/Go-migration inventory in
  `docs/FINDINGS_TIMING_TF.md`.
- **Script Doctor in pure Go** (`funscript.EvaluateScriptQuality` +
  `EvaluateDeviceCompat`): playback-tab “Skript prüfen” no longer needs
  a Python install. Same actions-only checks as Quality Doctor without
  dense tracking data; `EstimatedFromScriptOnly` remains set.
- **Research triage** of timing / 4-zone / accelerator / 63-BPM notes:
  `docs/FINDINGS_TIMING_TF.md` — what was fixed, what waits on
  measurement, what stays deferred, what the next Go slices are.
- **Playback tab: pressing the native video Play control now also starts
  funscript/device playback**, instead of these being two separate actions
  (the dedicated "Abspielen" button and the video's own play control).
  Guarded against re-triggering itself when video-sync makes the app call
  `videoEl.play()`. Doesn't reset the video to the start, unlike the
  dedicated button - the video is already running at its current position
  when this fires.
- **`generator/trackcv`, a Go-native CSRT tracking package (experimental)**:
  first step of writing more of the generator in Go. Ports `track_roi()`
  (CSRT + scene-cut + camera compensation + appearance memory) via a small
  custom cgo OpenCV wrapper (not `gocv`). GEMESSEN: r=0.9996 vs Python,
  ~15% less wall time. Now reachable through the native pipeline above
  when opted in; still not the default.
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
