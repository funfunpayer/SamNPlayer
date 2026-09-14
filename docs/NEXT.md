# Next steps

## Verified baseline · September 14, 2026

- `main`: `3984ec7`, release [v0.2.1](https://github.com/funfunpayer/SamNPlayer/releases/tag/v0.2.1).
- PRs #2, #3, #4, and #5 are integrated: the Tf/Tj recipe, generator
  integration, second GUI region, `suction_position` playback mode, version
  validation, and the English translation of project documentation.
- Tests and the release workflow succeeded for that commit.
- [PR #6](https://github.com/funfunpayer/SamNPlayer/pull/6) adds an optional
  local-AI adapter for the generator (ONNX-based region proposal, see
  `docs/AI_ADAPTER.md`). Review and merge remain pending; check GitHub for
  current CI status.

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

### 2. Measure generator quality · requires rated test material

Use reproducible clips with known reference signals alongside representative
real clips. Compare standard and Tf/Tj profiles with documented ROIs and
parameters. Keep raw signals, output, quality reports, and human ratings together.
Acceptance: demonstrate improvements in rhythm, amplitude, and tracking;
report synthetic results separately from real-material ratings.

### 3. Improve automatic two-ROI suggestions

Inspect `find_two_rois` and existing tests before making changes. Compare
automatic suggestions against manually selected regions and known ground truth.
Acceptance: measurable benefit, visible uncertainty, and manual correction.
Do not switch profiles automatically without validation.

### 4. Complete motion-signature and profile integration in the GUI

The naming and persistence paths now exist at the CLI level
(`--label-scene`, `--suggest-profile` in `generate_funscript.py`, PR #6) —
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
