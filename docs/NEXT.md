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
signals). **Not yet done:** running it against the real dataset — it
needs `--dataset` pointed at the user's `funscript-tests` folder, which
is local to their machine; the persisted manifest (hashes, generator
versions, ROIs) from the brief's "Reproduce before changing the
algorithm" section is also still open.

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
