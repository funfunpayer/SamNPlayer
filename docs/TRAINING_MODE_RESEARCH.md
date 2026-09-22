# Training mode — research and improvement proposals

Research pass, not yet a decided direction (see `TFTJ_PROFILE_DIRECTION.md`
for what an "owner-decided direction" doc looks like once this is one).
Scope: `player/training.go`, `cmd/gui-wails/app_training.go`,
`cmd/gui-wails/app_training_history.go`,
`cmd/gui-wails/frontend/src/training.js`. Related: `HANDOFF.md`'s
"Training" and "Training history" sections, `docs/NEXT.md`.

---

## What exists today (verified by reading the current code, not guessed)

- **Two techniques:** Stop-Start (ramp to peak, hold, ramp to **0**, rest)
  and Plateau/"edging" (ramp to peak, hold, ramp down to a **fraction** of
  peak instead of 0). Selectable per session.
- **Channel choice:** vibration only, suction only, or both. In "both"
  mode the two channels always receive the **same** curve — there is no
  per-channel shaping (`rampChannel`'s own comment already names this as
  a deliberate simplification, not a limitation nobody noticed).
- **One feedback signal:** a 1–10 "arousal" button row. A report affects
  only the **next** cycle (asymmetric adjustment — dampens harder on the
  way up than it lengthens on the way down, by design,
  `adjustForArousal`). The **current** cycle is unaffected by a report;
  interrupting it requires the separate "Interrupt now" button
  (`StopCycle`).
- **Fixed within-session progression:** `ProgressionPerCycle` linearly
  raises peak intensity and hold time across the cycles of **one**
  session. Nothing carries over to the **next** session — each session
  starts from the same manually-set defaults (or last-used values via
  `saveSetting`/`getSettingsCache`), not from how previous sessions went.
- **Per-cycle session logging** (JSONL, one file per session) capturing
  peak, hold, rest, whether the cycle was stopped early, time-to-peak, and
  the arousal report that shaped it.
- **Cross-session history** (`TrainingHistory`, shipped per
  `docs/NEXT.md`'s "Done, PR #55"): reads those logs back and shows one
  summary line per past session (cycles, mean peak, early-stop count, mean
  arousal). **Display only** — nothing reads this data back into new
  session defaults yet.
- **Mock device toggle**, log ticker, and two live stats (current cycle,
  current peak). No live intensity graph — the app's playback tab already
  renders a Funscript curve with a moving playhead, but training has no
  equivalent view of the ramp it's currently running.

Two things checked directly rather than assumed:
- No global "panic stop" keybinding exists for the training tab (grepped
  `frontend/src` for `Escape`/`emergency`/`panic`/global stop handlers —
  only the playback tab binds `Escape`, and only to exit fullscreen).
  Stopping mid-cycle requires clicking a button.
- `HANDOFF.md`'s "Known limitations" already states device resolution is
  unmeasured (Buttplug's 0–10/0–5 quantization, not confirmed firmware
  steps) and that training has **only run against `device.Mock`**, never
  real hardware — both apply directly to any proposal below that assumes
  a certain number of usable intensity steps or ramp granularity.

---

## Proposals, roughly by expected value vs. effort

### A. Feed history back into next-session defaults (small, high value)

The data to do this already exists and is already summarized
(`TrainingSessionSummary`) — nothing new needs measuring or logging. What
is missing is using it. Concretely: when opening the training tab, if the
last N sessions at the currently-selected technique/channel completed with
**zero** early stops, suggest (not silently apply) a slightly higher
starting peak/cycle count for the next session; if early-stop rate is
high, suggest backing off. This is the same "let the data set the next
parameter, but don't auto-adopt without evidence" pattern the project
already uses for `quality_model.py` (accepted only if cross-validation
beats the fixed rule). Effort is low because `TrainingHistory` already
returns everything needed — this is a frontend read plus a suggestion
banner, not new plumbing.

### B. Let a high arousal report affect the *current* cycle, not just the next one

Right now, two independent mechanisms exist for "this is too much":
`ReportArousal` (soft, affects only the next cycle) and `StopCycle`
(hard, ends the current cycle immediately). A report of 9 or 10 sits in
between — clearly urgent, but the user still has to reach for a second,
separate control to act on it *now*. A small, low-risk change: treat a
report at or above some threshold (e.g. 9) as an implicit `StopCycle` for
the cycle in progress, in addition to shaping the next one. This does not
remove the explicit button — it just closes the gap between "I reported
it's too much" and the ramp actually responding, for the case where the
two clearly agree.

### C. Session-length safety ceiling

Nothing currently bounds how long a session can run beyond the
user-chosen cycle count and per-cycle timings — there's no maximum
elapsed-time cap independent of those settings. A generous, silent
disable-by-default ceiling (e.g. warn or auto-end past some elapsed
minutes) would guard against a mis-typed setting (an extra zero on
`restMs`, `cycles` in the hundreds) running far longer than intended
without requiring the user to notice and click Stop. This is a pure
safety net, not a training-technique change, so it should never change
outcomes for anyone within normal ranges.

### D. Global stop reachable without precise clicking

The confirmed gap above (no keybinding, only buttons) is worth closing
independently of anything else here: a single key (e.g. Space, mirroring
playback's existing use of Space, or a dedicated key so it doesn't
collide when a video isn't in view) bound to `StopTrainingCycle` while
the training tab is active and a session is running. Low effort, direct
safety value, and consistent with the existing "Interrupt now" button
that already exists — this only adds a second way to trigger the same
call.

### E. Live intensity graph during a running session

The training tab currently shows only two text stats (cycle number,
current peak) and a scrolling text log. The playback tab already has a
reusable pattern for drawing a value-over-time curve with a live
position. A small version of that for training — plotting the actual
ramp/hold/rest curve as it plays, not just logging cycle summaries after
the fact — would make it easier to see *where in the cycle* you are
without reading timestamps in the log. This is presentation only; no
change to `player/training.go`'s behavior.

### F. Session data quality for real personalization (needs hardware first)

Item A above works with fixed nominal intensities (0–1 floats sent to the
device). Any deeper personalization — e.g. learning *this specific
person's* effective hold/rest ratio the way `quality_model.py` learns
quality weights from ratings — depends on knowing what the device
actually does with those floats, which `HANDOFF.md` already flags as
unmeasured and blocked on priority 1 ("Validate real hardware") in
`docs/NEXT.md`. Listed here so it isn't proposed again independently:
**do not build a learned intensity model before that measurement
exists** — it would be tuning against Buttplug's quantization curve
instead of the device's actual response, silently.

### G. Per-channel curves for "both"

`rampChannel`'s own comment already flags this as skipped deliberately
("a per-channel curve is conceivable but not needed for training's
shared up/down feel"). Revisiting it (e.g. suction sustained while
vibration pulses within the same hold phase) is plausible but should stay
untouched until there's a concrete reason to believe today's shared curve
is actually worse — the project's own convention (see `HANDOFF.md`'s
"Tested and rejected" table) is to only ship a change once it's measured
against the simpler baseline, not because it's imaginable.

---

## Explicitly not proposed, and why

- **Randomizing cycle timing to prevent habituation.** Tempting on
  paper, but Semans/edging protocols as clinically described are
  deliberately regular, and the existing arousal-feedback loop already
  adapts each cycle to the user's own state — adding random jitter on
  top would make cycle timing in the session log harder to attribute to
  either the feedback loop or the randomizer, which would undermine the
  history-based proposal in (A). Not ruled out permanently, just not
  worth the log-signal cost without a stated reason to want it.
- **Combining training mode with funscript/video playback.** Training's
  entire premise in this codebase (see `player/training.go`'s own header
  comment) is a technique that runs independent of content. Merging it
  with playback would turn a focused, well-tested control loop into a
  second video-sync feature, duplicating what the playback tab already
  does, for a use case the clinical technique doesn't call for.
- **Biofeedback hardware (heart rate, etc.) as an automatic arousal
  input.** Would remove the one active thing the user currently reports
  by hand, but there is no such integration anywhere in this codebase
  today, no evidence it would track arousal better than self-report, and
  it would add a second Bluetooth peripheral and pairing flow to a
  product whose device story (`docs/PLATFORMS.md`, the BLE/Intiface
  section of `HANDOFF.md`) is already carrying real complexity. Worth
  naming as a possible future direction, not proposing now.

---

## Suggested order

(A) and (D) are both small and self-contained — either can ship without
the other. (B) and (C) touch `player/training.go`'s control logic
directly and should get the same "behavior change needs a regression
test that fails without it" treatment `CONTRIBUTING.md` requires
elsewhere in the project (a test asserting a >=9 report interrupts the
running cycle; a test asserting the ceiling actually ends a session).
(E) is presentation-only and can happen independently of the others. (F)
stays blocked until priority 1 in `docs/NEXT.md` closes; (G) stays
un-started until someone has a concrete reason to measure against.
