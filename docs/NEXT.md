# Next steps

## Verified baseline · September 14, 2026

- `main`: `a53bb32`, release [v0.2.1](https://github.com/funfunpayer/SamNPlayer/releases/tag/v0.2.1).
- PRs #2–#7 are integrated: the Tf/Tj recipe, generator integration, second
  GUI region, `suction_position` playback mode, version validation, the
  English translation of project documentation, and the full local-AI
  adapter (region/profile/quality, `docs/AI_ADAPTER.md`).
- Tests and the release workflow succeeded for that commit.
- [Issue #8](https://github.com/funfunpayer/SamNPlayer/issues/8) and
  [`docs/FUNGEN_PARITY_PLAN.md`](FUNGEN_PARITY_PLAN.md) (from
  `codex/fungen-reference-plan`, merged into this branch) independently
  found the same thing from real Hub/Tf/Tj batches: near-zero correlation
  against manual FunGen references. The parity plan has a measured
  baseline (mean r ≈ -0.07 hub / 0.02 tf·tj on 3 valid moving references)
  and points at ROI quality, not filter/parameter tuning — see priority 2.
  `compare-fungen-manual.md` (7 rows, 4 distinct clips after removing
  duplicate matches) is consistent: mean r ≈ -0.05 hub / 0.01 tf·tj.

This is the short-term task list. Architecture and measurements belong in
`HANDOFF.md`, contribution rules in `CONTRIBUTING.md`, and setup instructions
in `WIEDERAUFNAHME.md`. Recheck GitHub status before resuming work.

## Priorities and acceptance criteria

### 1. Validate real hardware · requires a Sam Neo 2 and an operator

Test BLE and Intiface separately: connection, playback, pause/stop,
reconnection, and training. Record raw-value resolution for both channels.
For Tf/Tj, verify that vibration stays off and suction output matches the signal.
Acceptance: record device/adapter details, app version, steps, and observed
results; report deviations as reproducible bugs.

### 2. Match user-rated FunGen2 references · active code assignment

Follow [the implementation brief](FUNGEN_PARITY_PLAN.md): establish the
benchmark, isolate tracking errors, improve the shared Tf/Tj path, and retain
both names as compatible aliases for one profile. The user rates FunGen2
closer to the source motion and wants SamNPlayer to approach it. Identical
Tf/Tj output is intentional.

Use reproducible clips with known reference signals alongside representative
real clips. Compare standard and Tf/Tj profiles with documented ROIs and
parameters. Keep raw signals, output, quality reports, and human ratings together.
Acceptance: demonstrate improvements in rhythm, amplitude, and tracking;
report synthetic results separately from real-material ratings.

Per [issue #8](https://github.com/funfunpayer/SamNPlayer/issues/8): prefer
a small, fixed-ROI matrix (2–3 clips with a documented FunGen `.funscript`
reference, ROI1/ROI2 marked once and reused, same parameters, Hub vs Tf vs
Tj) over another large batch — a 22-clip run with Auto-ROI and an invented
ROI2 produced near-zero correlation and did not isolate whether the cause
was ROI or parameters. Compare with best-lag correlation, not same-duration
alone: a correctly-shaped signal shifted in time scores as a total mismatch
otherwise. Only tune `smooth-window`/peaks/`per-scene-roi` after ROI quality
is confirmed. FunGen exports are reference signal only (`.funscript`, not
`.fungen`/`FGPROJ`) — same license boundary as everywhere else in this repo.

**Work Package 1 (benchmark correctness) is implemented:**
`generator/fungen_compare.py` (`python3 fungen_compare.py --dataset DIR
--output report.md`) replaces the local, not-in-repository comparison
script that produced the `compare-fungen-manual.md` baseline. Reviewing
that script's logic (its source was shared in chat, not committed) found
four bugs that independently push correlation toward zero regardless of
tracking quality: it de-duplicated candidate files by path instead of
content (inflating `compare-fungen-manual.md`'s sample count), it aligned
the two files by INDEX after resampling instead of on a shared absolute
timeline or searching for the best lag (so a correctly-shaped but
time-shifted match scored as a mismatch), it returned `0.0` instead of
"undefined" for a constant reference, and it never checked for a
polarity-inverted match (which reads as a strongly negative correlation
and looks like bad tracking rather than a sign-convention mismatch).
`generator/fungen_compare_test.py` has a regression test and gegenprobe
for each, using the synthetic fixtures the brief asked for (known phase,
irregular cycles, pauses, amplitude changes, offset timestamps, constant
signals).

**Run against real clips, shared in chat rather than on disk here (two of
the three valid moving references from `compare-fungen-manual.md`, plus
their SamNPlayer outputs):** correlation stayed weak (r≈0.05–0.13) even
after lag and orientation search, including on a single continuous
chapter with no timing gaps. The four benchmark bugs were real but do not
explain the whole gap. The persisted manifest (hashes, generator
versions, ROIs) from the brief's "Reproduce before changing the
algorithm" section is still open; a proper scan of `--dataset` against
the user's full `funscript-tests` folder has not happened (local to their
machine).

**Work Package 2 (isolate the mismatch) found a real, fixed bug:**
`track_two_points()` measured the two-region distance using only the Y
coordinate of each box's center, not a full 2D distance (the brief's
zoom-invariance concern, section "Investigate ... test that claim and
correct it", also flagged the underlying formula). This is harmless when
ROI1/ROI2 sit at different heights, but produces a near-zero, tracking
quality-independent signal whenever they sit at the same height — exactly
what issue #8 describes the batch's ROI2 heuristic doing ("shift ROI1 by
~1.2× width", i.e. no height change). Confirmed on the real BBW/Entladen
`tj` outputs shared in chat: both sit flat at one extreme for most of
their length, then move briefly - the signature of a distance metric
seeing no variation until tracker drift accidentally introduces some.
Fixed: `two_point_distance()` now computes the full 2D center-to-center
distance (`np.hypot`), a strict generalization that keeps the existing
vertical-signal test (`two_point_test.py`) passing and adds
`two_point_axis_test.py` for the horizontal case, with a gegenprobe
(reverting to the Y-only formula reproduces the near-dead signal:
amplitude 1.0px instead of the true 80px).

**Run against a large synthetic dataset (48 clips, both a FunGen 2 export
and a SamNPlayer hub+tf run per clip, shared in chat as `fungen2.7z` /
`samnplayer_test_funscripts.7z`, generated the same day with the 2D-distance
fix and `auto_roi.find_two_rois` for ROI2, not the invented-shift
heuristic):** far stronger than the real-clip result — mean r ≈ 0.52 (hub)
/ 0.58 (tf) across ~30 valid clips (walk cycles, sine/triangle/square
waves, camera pans/zooms, occlusion, irregular events; 28 of 60 rows were
excluded as undefined, mostly constant FunGen references for clips FunGen
itself found nothing scriptable in). This is a large jump from the
real-clip r≈0.05–0.13 - clean, single-subject synthetic motion is
evidently far easier for both tools than real material with occlusion,
non-rigid deformation and multiple bodies. Confirms the earlier read: the
comparison-methodology bugs and the two_point axis bug were real and
worth fixing, but real-material tracking quality is its own, still largely
open problem - synthetic-clip success does not by itself predict
real-clip success.

**A methodology risk found while checking this:** the first pass used the
tool's old 3000ms default lag search window, which pushed several
short-clip correlations toward the search boundary (lag=±2800-3000ms) -
the sign of a wide search "finding" a coincidental match on a small
overlap window rather than a real delay. Narrowing to 500ms dropped the
mean from 0.696 to 0.486. Fixed in `fungen_compare.py`: the default
`--max-lag-ms` is now 1000 (was 3000), every result carries a
`low_confidence` flag when fewer than `LOW_CONFIDENCE_SAMPLES` (30, i.e.
under 3s of overlap at the 100ms resample step) samples were used, and
`format_report()` prints a confidence-excluded mean alongside the plain
one so a few short, lucky clips cannot quietly carry the average. On this
dataset the two means come out close (hub 0.517 vs 0.518, tf 0.581 vs
0.564), which is itself a good sign - the result isn't an artifact of a
handful of short overlaps.

Also added: `fungen_compare.py --dataset` now scans subfolders
(`rglob`, not `glob`) - the real dataset's references and batch output
each came in their own subfolder tree, which the original top-level-only
scan would have silently found nothing in.

**Not yet done:** re-running the *original* real (non-synthetic) batch
clips with the 2D-distance fix - needs the source videos, not just the
`.funscript` outputs shared so far, to see how much of the real-clip gap
it closes there specifically.

### 3. Improve automatic two-ROI suggestions

Inspect `find_two_rois` and existing tests before making changes. Compare
automatic suggestions against manually selected regions and known ground truth.
Acceptance: measurable benefit, visible uncertainty, and manual correction.
Do not switch profiles automatically without validation.

Per issue #8: **never** batch a Tf/Tj (distance-profile) run with an
invented/heuristic ROI2 (e.g. "shift ROI1 by ~1.2×width") — that alone can
explain a weak correlation before any parameter is even considered. Until
`find_two_rois` is measurably good enough, Tf/Tj needs a real, manually
placed second mark, or the AI ROI adapter's suggestion (`--roi-finder ai`,
`docs/AI_ADAPTER.md`) with a human confirming or correcting it — never a
silently auto-committed guess for either ROI.

### 4. Complete motion-signature and profile integration in the GUI

The naming and persistence paths now exist at the CLI level
(`--label-scene`, `--suggest-profile` in `generate_funscript.py`, PR #7) —
this item is now specifically the GUI wiring: a "name this scene" control
and a suggestion display, not building the underlying mechanism from
scratch. Acceptance: names and parameters survive restarts; reuse for
similar scenes is offered with an explanation and can be declined.

### Later

- Script Doctor for imported `.funscript` files.
- Training history across multiple sessions.
- AI as a replaceable analysis backend, reusing raw data, parameters,
  quality reports, and confirmed ratings. Region proposal, profile
  proposal, and quality second-opinion are all implemented at the
  CLI/generator level. GUI wiring for all three remains open. See
  `docs/AI_ADAPTER.md` for the architecture and order.

## Product requirements

General generator quality and the result on the Sam Neo 2 are the priorities.
Profiles are named `tf`/`tj`; their recipe uses `suction_position` with
vibration off. Calibrate further parameters only after measurement.
Keep classical analysis as the foundation for later AI integration.
