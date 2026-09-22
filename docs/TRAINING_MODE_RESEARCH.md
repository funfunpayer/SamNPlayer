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

## Follow-up design: multi-phase, per-channel scripts (owner ask, 22 Sep 2026)

Owner wants something more specific than proposal (G) above: not just "let
vibration and suction differ within one cycle", but a **script** — an
ordered sequence of phases, each with its own shape, where vibration and
suction can do genuinely different things over time. Concrete example the
owner gave: **light vibration, ramping up stronger, then back down again
— followed by a phase of suction alone with light vibration alongside
it.** Also: the arousal scale (1–10) must be confirmed to actually widen
rest and weaken **both** channels when it fires, and a display is wanted
for all of this. This section plans that out; it supersedes (G)'s
"leave alone" stance now that there's a concrete reason to change it.

### Why today's model can't express that example

`TrainingOptions` has exactly one `Channel` field and one intensity
curve (`peak`/`hold`/`rest`, shaped by `Technique`). "Both" sends the
**same** value to vibration and suction every step (`setChannel`). There
is no way, today, to say "vibration ramps 0.2→0.8→0.2 while suction
stays off" and then, later in the same session, "suction climbs while
vibration holds at a light constant level" — that needs two things the
current model doesn't have: **more than one shaped segment per session**,
and **independent curves per channel within a segment**.

### Proposed model: phases, each with per-channel curves

```go
// One channel's shape within a phase. Generalizes today's single
// Technique+PeakIntensity+PlateauFraction into a reusable shape that
// either channel can carry independently.
type ChannelCurve struct {
    Channel    TrainingChannel // ChannelVibration or ChannelSuction — never Both here
    StartLevel float64         // 0-1, level at phase start (0 = channel off)
    PeakLevel  float64         // 0-1, level reached at the shape's high point
    EndLevel   float64         // 0-1, level at phase end (e.g. 0 for stop-start, >0 to carry into the next phase)
    RampUpMs   int
    HoldMs     int
    RampDownMs int
}

// One named segment of a script. Vibration and suction each get their
// own optional curve (nil = channel stays at whatever level the
// previous phase left it at, e.g. "light constant" through a suction
// phase without re-specifying vibration every phase).
type TrainingPhase struct {
    Name        string
    Vibration   *ChannelCurve
    Suction     *ChannelCurve
    RepeatCycles int // this phase's own shape runs this many times before advancing
    RestMs       int // pause between repeats, and before the next phase
}

// A script is just an ordered list of phases. Today's Stop-Start and
// Plateau techniques become the two trivial single-phase scripts below
// (see "Backward compatibility") — nothing existing breaks.
type TrainingScript struct {
    Phases              []TrainingPhase
    ProgressionPerCycle float64 // still applies once, across the whole script's cycles
}
```

The owner's example becomes two phases:

```text
Phase 1 "Vibration wave"      Phase 2 "Suction focus"
  Vibration: 0.2 → 0.8 → 0.2    Vibration: (unset → stays at 0.2, carried from phase 1's EndLevel)
  Suction:   (unset, off)       Suction:   0.2 → 0.75 → 0.3
  RepeatCycles: 3                RepeatCycles: 3
```

This is a genuine generalization, not a rename: `RunTraining`'s current
loop already ramps one curve up/hold/down per cycle — running that same
inner loop once per `ChannelCurve` inside a phase, and once per phase in
sequence, reuses `rampChannel`/`waitInterruptible` unchanged. The new
work is the outer phase loop and the request/log schema, not the ramp
mechanics themselves.

### Arousal scaling has to move from "one peak" to "one factor, several targets"

`adjustForArousal` today returns a single adjusted `(peak, holdMs,
restMs)` triple, applied to whichever channel(s) `setChannel` happens to
touch — which only produces "both channels weaker" today because in
"both" mode they already shared one number. Under the phase model above,
that stops being true automatically, so the fix has to be explicit: split
`adjustForArousal` into **factor computation** (`holdFactor, restFactor,
peakFactor := arousalFactors(arousal)`, pure, unchanged math) applied
**once per cycle**, then multiply *every active channel's* `PeakLevel`
(and `EndLevel` where it's the plateau-style non-zero end) by the same
`peakFactor`, and the phase's shared `RestMs` by `restFactor` — vibration
and suction both get quieter, and the pause both channels share gets
longer, from one arousal report, exactly the "high arousal ⇒ longer
pauses AND weaker on both channels" behavior asked for. This is a
targeted refactor of existing, tested logic (`adjustForArousal` already
has `player/training_test.go` coverage) rather than new behavior
invented from scratch — the asymmetric up/down shape (dampen hard on the
way up, ease off gently on the way down) carries over unchanged.

### Display: show the planned curve, and show the adjustment happening

Two things, both building on proposal (E) above rather than replacing
it:

1. **Plan preview**, before starting: render each active channel's curve
   (vibration/suction as two distinct lines) across the phases about to
   run, the same way the playback tab already draws a Funscript curve —
   so "light vibration ramping up then down, then suction alone with
   light vibration" is something you look at and confirm, not something
   you infer from four number fields per phase.
2. **Live view during a session**: the same chart with a moving playhead
   (mirrors the playback tab's curve + playhead pattern), plus a small
   readout when an arousal report changes the next cycle — e.g. "feedback
   9 → peak −30%, rest +65%, both channels" — so the reaction described
   above is visibly confirmed happening, not just trusted to be running
   correctly in the background. This directly answers the "sollte drauf
   ja wirklich dann reagieren" concern: the adjustment becomes something
   you can see per cycle, not just an internal number.

### Backward compatibility

Today's two techniques become the two built-in single-phase scripts:

- **Stop-Start** → one phase, one curve per selected channel,
  `EndLevel: 0`, `RestMs` as configured.
- **Plateau** → one phase, one curve per selected channel, `EndLevel:
  PeakLevel * PlateauFraction`.

Existing `TrainingRequest`/session-log JSON keeps working unchanged for
these two cases; the phase/script fields are additive. Old JSONL session
logs remain readable by `TrainingHistory` (it never assumed multi-phase).

### What this needs before it's built, not after

Consistent with `CONTRIBUTING.md`'s rule (behavior changes need a
regression test that fails without the change) and the finer-ramp entry
in `HANDOFF.md`'s "Tested and rejected" table (rejected once already for
the wrong reason, actual device resolution still unmeasured): the
multi-channel arousal-factor refactor needs a test asserting a high
report weakens **both** configured channels and lengthens the shared
rest, not just one; and any specific numeric levels in a shipped preset
(like "light" = 0.2) should be treated as a starting guess to revise once
priority 1 in `docs/NEXT.md` (real hardware) shows what these floats
actually do physically, the same caveat proposal (F) already raises.

### Suggested order

(A) and (D) are both small and self-contained — either can ship without
the other. (B) and (C) touch `player/training.go`'s control logic
directly and should get the same "behavior change needs a regression
test that fails without it" treatment `CONTRIBUTING.md` requires
elsewhere in the project (a test asserting a >=9 report interrupts the
running cycle; a test asserting the ceiling actually ends a session).
(E) is presentation-only and can happen independently of the others. (F)
stays blocked until priority 1 in `docs/NEXT.md` closes; (G) is
superseded by the phase/script design above, which now has a concrete
reason (the owner's example) rather than being speculative.

For the phase/script design: a reasonable first slice is the
arousal-factor refactor (applies to both channels + rest, has direct
test coverage to write) plus the two built-in scripts (today's
Stop-Start/Plateau, unchanged in behavior) running through the new phase
loop — that alone proves the phase mechanism without yet building a
script editor UI. The plan-preview/live-adjustment display and the
owner's two-phase vibration/suction example as a shippable preset are a
natural second slice once the first is running and tested.
