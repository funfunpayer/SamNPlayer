# Findings: Timing, Tf/Tj, Go-native (research triage)

Source: operator research notes uploaded 16 September 2026
(`01_BEFUNDE…` through `06_63_BPM…`). This file records what was
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

## Documented, not built yet (needs measurement first)

| Idea | Why not now |
|---|---|
| 4-zone Tf/Tj relative-motion graph | `region_fusion_auto` exists; full pairwise graph needs golden-clip wins before replacing two-ROI distance |
| Timing/Phase Analyzer product | `fungen_compare.best_lag_ms` already exists; a full 8-point pipeline is tooling, not a generator fix — build when diagnosing a concrete clip set |
| Decoder PTS through the pipeline | Real VFR issue; requires media-layer work; trackcv/OpenCV still use frame index today |
| Observation `valid`/`confidence`/`reason` | Right model for tracking loss; belongs with trackcv provenance, not a drive-by |
| Camera-comp affine (X/Y/rot/scale) + error≠0 | Valid; change only with amplitude A/B against current Y-only path |
| Accelerator abstraction (WinML/CUDA/…) | No end-user inference hotspot in the default path yet; ONNX stays optional |
| 63 BPM as forced SAM attractor | Useful as recovery prior later; must not override video tempo when confidence is high — needs SAM consumer first |
| Fusion on disagreement | `fusion.py` already waits for a third independent source |

## Go-native direction (aligned, already in flight)

Research doc `04_GO_NATIVE_ARCHITEKTUR` matches the standing decision:
more Go for runtime, Python for training/research. Shipped / in PR:

- `generator/trackcv` (#84) — CSRT tracking
- `generator/posttrack` (#85) — signal path, opt-in native pipeline
- This change — Script Doctor / device-compat checks in pure Go (no
  Python for the playback-tab “Skript prüfen” path)

Next Go slices when useful: Quality Doctor dense-signal checks, then
optional Tf/Tj two-point in Go — each with goldens, not rewrites for
their own sake.

## Language note (finding §10)

Agreed: Python-vs-Go is not the Tf/Tj correlation problem. Fix
perception/timing/ROI semantics first; Go removes install burden and
makes the pipeline deterministic to package.
