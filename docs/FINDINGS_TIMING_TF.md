# Findings: Timing, Tf/Tj, Go-native (research triage)

Source: operator research notes uploaded 16 September 2026
(`docs/perception_update_2026-09/`, formerly `01_BEFUNDE…` through
`10_IMPLEMENTIERUNGS_BACKLOG…` plus ADR/CSV). This file records what was
**acted on**, what stays **open pending measurement**, and what is
**deferred** — per project rules: no speculative modules, every quality
claim needs a measurement.

## Acted on now

### Tf/Tj suction double-floor (finding §2)

**Claim:** with actions clamped to 20–90 and `MinSuction=0.20`,
`liftFloor(pos/100, 0.20)` remaps resting suction from 0.20 → **0.36**
and peak from 0.90 → **0.92**.

**Verified in code** (`funscript/mapper.go` + `RecipeFor("tj")`): yes.
`TestRecipeTJSuctionNoDoubleFloor` locks the intended mapping:
pos 20 → suction ≈ 0.20, pos 90 → ≈ 0.90.

**Fix:** for `SyncSuctionPosition`, skip the runtime `liftFloor` on
suction. The generation-time 20–90 clamp *is* the floor. Recipe metadata
still reports `min_suction: 0.20` as that script-space floor.

Still needs a **feel check on real hardware** (priority 1) — software
can only assert the command values, not perceived intensity.

### Phase Analyzer core in Go (finding F-001 / doc 03)

**Claim:** the dominant quality gap is often phase, not shape —
`best_lag_ms` + raw vs aligned correlation separates "fix timing"
from "fix perception".

**Shipped:** `funscript.BestLagCorrelation` + `DiagnosePhase` — pure-Go
port of `fungen_compare.best_lag_correlation` (shared absolute timeline,
lag search, normal/inverted, shape-normalized error, low-confidence
flag) plus the research-doc decision tree (`timing` / `shape` / `ok` /
`undefined`). Locked by `funscript/phase_test.go` against the same
fixtures as `fungen_compare_test.py`.

**Not shipped (still deferred):** the full 8-point timestamp pipeline
(decoder PTS → device reaction). That is media/runtime tooling, not
this correlation core.

## Documented, not built yet (needs measurement first)

| Idea | Why not now |
|---|---|
| 4-zone Tf/Tj relative-motion graph | `region_fusion_auto` exists; full pairwise graph needs golden-clip wins before replacing two-ROI distance |
| Full 8-point Timing pipeline | correlation core is done; PTS→device chain needs media-layer work and a concrete clip set |
| Decoder PTS through the pipeline | Real VFR issue; trackcv/OpenCV still use frame index today |
| Observation `valid`/`confidence`/`reason` | Right model for tracking loss; belongs with trackcv provenance, not a drive-by |
| Camera-comp affine (X/Y/rot/scale) + error≠0 | Valid; change only with amplitude A/B against current Y-only path |
| Accelerator abstraction (WinML/CUDA/…) | No end-user inference hotspot in the default path yet; ONNX stays optional |
| 63 BPM as forced SAM attractor | Useful as recovery prior later; must not override video tempo when confidence is high — needs SAM consumer first |
| Fusion on disagreement | `fusion.py` already waits for a third independent source |

## Go-native direction (aligned, already in flight)

Research doc `04_go_native.md` matches the standing decision:
more Go for runtime, Python for training/research. Shipped / in PR:

- `generator/trackcv` (#84) — CSRT tracking
- `generator/posttrack` (#85) — signal path, opt-in native pipeline
- Script Doctor / device-compat checks in pure Go
- Phase Analyzer core (`funscript.BestLagCorrelation`) — this change

Next Go slices when useful: Quality Doctor dense-signal checks, then
optional Tf/Tj two-point in Go — each with goldens, not rewrites for
their own sake.

## Language note (finding §10)

Agreed: Python-vs-Go is not the Tf/Tj correlation problem. Fix
perception/timing/ROI semantics first; Go removes install burden and
makes the pipeline deterministic to package.

## Signal Quality ≠ Motion Fidelity (17 Sep 2026 review)

Script Doctor / Quality Doctor measure **Signal Quality**. FunGen /
`best_lag_ms` / Phase CLI measure **Motion Fidelity**. Canonical doc:
`docs/SIGNAL_VS_FIDELITY.md`. Next architecture milestone is **SAM
Perception v1** (multi-observer measure → fuse → motion model), not
another backend — see `docs/ROADMAP.md` and
`docs/REVIEW_SCRIPTQUALITAET_2026-09-17.md`. Blocked on real golden clips.

## Open inventory (September 16, 2026)

What is still open, what belongs in Go next, and what is blocked.
Companion checklist: `docs/ROADMAP.md`. Research source:
`docs/perception_update_2026-09/`.

### Already in Go (shipped / in open PRs)

| Piece | Where | Notes |
|---|---|---|
| CSRT tracker (experimental) | `generator/trackcv` (#84, merged) | Not default generation path; Linux OpenCV/cgo; Windows not solved |
| Post-tracking signal path | `generator/posttrack` (#85, open) | Savgol → normalize → peaks → keyframes → RDP → speed limit; goldens vs Python |
| Opt-in native CSRT generate | `generator/native*.go` (#85) | GUI checkbox; still falls back to Python for Quality Doctor / AI / audio / other backends |
| Script Doctor (actions-only) | `funscript.EvaluateScriptQuality` (#86) | Playback “Skript prüfen”; no dense-signal checks |
| Dense Quality Doctor | `funscript.EvaluateDenseQuality` (#86) | Native CSRT path; rhythm/recon/lost/active/motion |
| Device-compat checks | `funscript.EvaluateDeviceCompat` (#86) | Same Go path |
| Phase Analyzer core | `funscript.BestLagCorrelation` / `DiagnosePhase` (#86) | Port of `best_lag_ms`; not the 8-point PTS chain |
| FunGen-compare CLI | `SamNPlayer compare` / `CompareDataset` (#86) | Thin Go report; generate half still Python for other backends |
| Funscript I/O, recipes, O-markers, polarity, ozone, mapper, player, device, BLE | existing packages | Already Go |

### Still Python (end-user or tooling that generation still shells to)

| Piece | Why it still matters | Go candidate? |
|---|---|---|
| `generate_funscript.py` orchestration | Default generate path | Partially replaced by #85 native CSRT only |
| Quality Doctor **dense** checks | Rhythm/noise, reconstruction error, tracker-lost, active-time | **Done in Go** (`EvaluateDenseQuality`); still Python on default generate path |
| Backends: flow, grid_lk, region_fusion(_auto), two-point Tf/Tj | Most users / Tf/Tj | Later; each needs goldens; Tf/Tj two-point is high value |
| auto_roi / scene-cut ROI / camera compensation (full path) | Classical auto region | Partial in trackcv; rest later |
| AI ROI / profile / quality opinion | Opt-in ONNX | Keep Python or ONNX-from-Go later; training stays Python |
| audio_check | ffmpeg + tempo | Possible in Go (ffmpeg CLI already); low urgency |
| quality_model train/infer | Learned Quality Doctor | Training-only → stay Python; inference could move later |
| golden_clip_benchmark + fungen_compare CLI | Benchmark / FunGen parity | **Compare half in Go** (`compare` CLI); generate half still needs Python until native path covers backends |
| YOLO bootstrap / export / train | Research tooling | Stay Python |

### P0 research findings — status

| ID | Topic | Status |
|---|---|---|
| F-001 | Phase / `best_lag_ms` | **Core done (Go)**; 8-point PTS pipeline open |
| F-002 | Suction double-floor | **Software done**; hardware feel 🔒 |
| F-003 | Real PTS | Open — media/VFR; trackcv still frame-index |
| F-004 | Missing-data / valid/confidence | **Done on trackcv** (`ValidFrames`/`Confidence`/`Reason` + native metadata) |
| F-005/6 | 4-zone relative graph + confidence | Deferred — needs golden-clip win vs two-ROI |
| F-007 | P05/P95 + MAD normalize | Open — A/B vs current percentile path (#85 normalize) |
| F-008 | Filter phase lag | Covered by Phase Analyzer diagnostics; change filter only with benchmark |
| F-012 | Device suction-first | Recipe/mapper done; feel 🔒 |
| F-018 | Golden-clip gate | Tool exists; **manifest empty** — biggest process blocker |
| F-020 | Deadband / refractory | Open — signal polish after goldens |

### Next Go slices (priority order, quality-first)

1. **Land / keep #85** (`posttrack` + opt-in native CSRT) — prerequisite for Python-light generate on CSRT.
2. **Dense Quality Doctor in Go** — **done** (`EvaluateDenseQuality` + native wire).
3. **FunGen compare CLI in Go** — **done** (`SamNPlayer compare`).
4. **Tf/Tj two-point distance path in Go** — only with goldens; highest product value after CSRT hub.
5. **Observation contract** (`valid` / `confidence` / `reason`) on trackcv — **done**.
6. **Other backends / Windows OpenCV** — after CSRT native is measured default-worthy.

### Not Go-now (blocked or wrong lever)

- Populate golden clips / real FunGen refs — 🔒 you (content), unblocks everything else
- Sam Neo 2 feel-check (suction floor, contact vibration) — 🔒 hardware
- Full 8-point PTS→device pipeline — media + runtime instrumentation
- 4-zone relative-motion as default — measurement first
- Accelerator abstraction / SAM 2 / depth / ByteTrack — Perception 2.0 later phases
- Forced 63 BPM — recovery prior only after SAM consumer exists
- Rust rewrite — profiler gate only

### Language reminder

Python-vs-Go is not the Tf/Tj correlation problem. Go removes install
burden and packages a deterministic runtime; perception/timing/ROI
semantics still decide quality.

## Research pack location

Canonical copy: `docs/perception_update_2026-09/` (README + 01–10 +
findings CSV + ADR JSON). Status of each item is tracked here and in
`docs/ROADMAP.md` / backlog triage — do not re-implement deferred rows
without new measurements.
