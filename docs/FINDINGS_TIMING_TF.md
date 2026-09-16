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

## Research pack location

Canonical copy: `docs/perception_update_2026-09/` (README + 01–10 +
findings CSV + ADR JSON). Status of each item is tracked here and in
`docs/ROADMAP.md` / backlog triage — do not re-implement deferred rows
without new measurements.
