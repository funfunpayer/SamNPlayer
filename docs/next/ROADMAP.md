# 2.0 Next — gates, not dates

A gate is passed when its checks are green and recorded in the PR. Each work
package is its own PR with tests. Release order and the editor steps are in
`docs/ROADMAP_2026-10.md`; this list covers the engine work for 2.0 Next.

## P — Preparation ✅

| ID | Deliverable | Status |
|---|---|---|
| P1 | Plan, roadmap, open decisions, archived #510 docs | ✅ |
| P3 | Owner answers [`OPEN_DECISIONS.md`](OPEN_DECISIONS.md) | partly answered |
| P4 | [Spec: golden-clip manifest + benchmark report + gate](specs/P4_GOLDEN_BENCHMARK.md) | ✅ draft |
| P5 | [Spec: `.samn` v2 axis map + companion files](specs/P5_SAMN_V2_AXES.md) | ✅ draft — refresh against the merged companion-axis work first |
| P6 | [Spec: batch job states + checkpoints](specs/P6_BATCH_JOBS.md) | ✅ draft |
| P7 | [Licence pass](specs/P7_ENTITLEMENT_THREAT_MODEL.md) | ⛔ not needed — no paid version |
| P8 | [Spec: release safety and updater source](specs/N1_RELEASES.md) | ✅ draft |
| P9 | [Inventory EDIT-00](inventory/EDIT-00_EDITOR.md) | superseded by `ROADMAP_2026-10.md` §4 |
| P10 | [Inventory ENG-01 part 1: where Python still runs](inventory/ENG-01_PYTHON_RUNTIME.md) | ✅ static part; timings at N2 |

## N1 — Safe releases (before the next tag)

- **D14:** `release.yml` attaches Suite files to the public release. Fix before
  v0.7.0 is tagged ([N1 spec](specs/N1_RELEASES.md)).
- **MEDIA-01:** `videox/mp4probe.go` reports an unknown codec as `h264` /
  `yuv420p`, so the "Make playable" proxy may not be offered. Report unknown as
  unknown; test with a stub MP4.

## N2 — Measure the baseline

| ID | Deliverable |
|---|---|
| ENG-01 part 2 | Benchmark runner (P4), golden report on the Owner's local clips |
| ENG-02 | Go backend/track contract, fast scan — **output unchanged** vs baseline |
| — | Split the Python embed: training scripts only in Suite (ENG-01 inventory §4) |

The profile that `ROADMAP_2026-10.md` §6 step 4 requires before the first Rust
step uses this runner.

## N3 — Smart Batch

Project Recovery (CAP-12 in the [capability register](PRIORITIZED_CAPABILITY_REGISTER.md)) is the prerequisite: checkpoints and restart before batch claims resume.

STU-01 Go job queue + batch UI, pause/resume/cancel, error report (P6) ·
STUDIO-A6 crash resume, atomic results · STU-02 20-clip acceptance (VFR, no
audio, broken file, Unicode name, read-only target).

## N4 — AutoDirector

ENG-03 per-scene candidates + independent verifier · ENG-04 selective retry, no
quality regression · STUDIO-A3 compare/accept/reject (shares E6 Compare).

## N5 — Devices + AI

D2–D5 device adapters with mocks · E8 device preview · ENG-05 / AUTO-02 optional
local AI with classical fallback · Windows 11 hardware smoke.

## N6 — 2.0 Next release

Version bump, changelog, handbook, screenshots from the real build, QA at
1366×768 and 1920×1080, site update. Fair FunGen comparison on identical clips
before any comparative claim.
