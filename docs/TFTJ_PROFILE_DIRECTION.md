# Tf/Tj as profile — direction (not yet implementation)

Owner direction, 21 Sep 2026. Think first; implement only after this
shape is agreed. Related: `docs/ENGINE.md`, `docs/GENERATE_HEURISTICS.md`,
`docs/NEXT.md` (§3 auto two-ROI, §5 contact vibration, region_fusion_auto),
`docs/FINDINGS_TIMING_TF.md`, `docs/PRODUCTION_ROADMAP.md` (G1).

---

## What the owner wants (plain)

1. **Tf/Tj should exist as a profile** — a feel / device recipe — not as
   “the product mode where you must mark two points and hope recognition
   works.”
2. **Recognition for Tf/Tj is not the lever that feels relevant right
   now.** Learning (classes / YOLO) we still need later; it is not the
   everyday Generate path.
3. **Automatic recognition currently misbehaves** — see open issues
   (notably #145 on an old build: two-point + MIL + “no motion”; earlier
   #126 auto-region missing packages). Fix / verify on v0.5.16 before
   building more auto-ROI UI on top.
4. **Contact vibration must move into Auto and into Normal** — not stay
   a Tf/Tj-only checkbox. Users should not hunt a special profile just
   to get that feel.
5. **Do not force marking everything in the generator.** FunGen 2 does
   not require that either — including without YOLO. That is why we
   already sketched **4 zones + sampling (+ audio as check/hint)**.

---

## Diagnosis (what we already measured)

| Fact | Source | Consequence |
|------|--------|-------------|
| Whole-clip FunGen r≈0.06 on Tf/Tj goldens is mostly **lag drift**, not “Tf/Tj is noise” | windowed phase / #143, `clip_voll` | Do not redesign tracking from whole-clip r alone |
| Hub (single-ROI) on same clip also drifts; Signal Quality ≠ Motion Fidelity | goldens + NEXT | Timing drift is not Tf/Tj-specific |
| Two-point distance alone does not close FunGen gap on titjob POV | NEXT Sep 16 | Multi-mark Tf/Tj is not the FunGen parity path |
| `find_two_rois` was measured insufficient for default | NEXT §3, issue #8 | Never silent-auto-commit ROI2 |
| `region_fusion` (4 sub-boxes **inside** a marked ROI) is competitive vs CSRT | NEXT Sep 16 | Useful, but still needs a mark |
| `region_fusion_auto` (full-frame 4 zones, no mark) exists, synthetic only | NEXT Sep 16 | This is the classical “no mark” candidate — **not product-default yet** |
| Contact vibration works when the distance signal is good; inherits Tf/Tj unreliability | NEXT §5 | Decouple vib recipe from “must be two-ROI track” |
| OpenCL checkbox forced Python → MIL on old builds | #141 / #145 | v0.5.16 removes that trap; re-smoke before blaming “auto” |

**Bottom line:** we conflated three different products under “Tf/Tj”:

```text
  A. Tracking mode   → how we get a 0–100 curve from video
  B. Device profile  → how that curve maps to Neo 2 (suction / vib / recipe)
  C. Marking UX      → manual ROIs vs auto / no-mark classical vs AI propose
```

FunGen 2 (classical) mostly ships **A+C without forcing marks**, then a
feel. We shipped **B glued to A=two-point** and **C=manual-first**.

---

## Target product shape

### Profiles (B) — names the user picks for *feel*

| Profile | Meaning (target) | Tracking (A) may be |
|---------|------------------|---------------------|
| **Normal / hub / standard** | Stroke feel, usual suction mapping | No-mark 4-zone **or** one ROI |
| **Auto** | “Just generate” — classical default path | Prefer no-mark; fall back to propose+confirm |
| **Weich / autotune** | Soft / filtered variants of Normal | Same as Normal |
| **Tf / Tj** | **Suction-centric recipe** (tight = high suction); contact vib on by default once promoted | Same curve sources as Normal/Auto — *not* forced dual-ROI |

Tf and Tj stay **aliases for one profile** (already the direction in
NEXT priority 2). They describe **device intent**, not “you marked a
nipple.”

### Tracking modes (A) — mostly invisible unless Advanced

Priority order toward FunGen-like UX:

1. **No-mark classical** — full-frame 4-zone sampling
   (`region_fusion_auto` family) + CSRT/grid primitives we already own
2. **One-ROI** — manual or AI propose → confirm (today’s solid path)
3. **N-point / tip+partners** — optional Advanced when the user *wants*
   distance semantics or training labels; not the everyday entry
4. **AI classes** (penis / breast / hand / …) — G3; learn when classical
   plateaus; never invent the 0–100 curve (`AI_ADAPTER.md`)

### Contact vibration (recipe, not detection)

Today: opt-in, Tf/Tj-only, derived from high `pos` on a suction/distance
script.

Target:

- Available on **Normal** and **Auto** (and still on Tf/Tj).
- On a **single-curve** script, “contact” means the **deep / high-`pos`
  slice of that script’s own range** (same envelope math we already
  use) — honest naming in UI: it follows depth of stroke, not a second
  tracked object, unless an N-point / class path is active.
- When a true partner distance exists, keep today’s distance-based
  envelope (better semantic match for tip↔body).
- Default **on** for Tf/Tj profile; **opt-in or gentle default** for
  Normal/Auto after hardware feel check (ROADMAP still open).

### Marking UX (C)

| Everyday | Correction / Advanced |
|----------|------------------------|
| Generate without drawing boxes (4-zone / auto) | Drag ROI(s) when auto is wrong |
| Audio = tempo plausibility / optional peak hint | Never “loudness → positions” |

Same rule as G1: audio does not invent motion.

### Learning

Keep training (#119 and friends) as the path to **better proposals and
class-aware partners**, not as the gate for shipping no-mark Generate.
Classical 4-zone must stand on its own first (FunGen without YOLO is the
fair bar).

---

## Bugs / debt to clear before building the new UX

1. **#145** — confirm obsolete on portable **v0.5.16** (Go CSRT log, no
   MIL). If still broken with Auto/two-point, file a fresh bug with
   v0.5.16 log; else close as build trap.
2. **Auto-region path** — package/bootstrap errors (#126 closed) must not
   return; Auto must not silently land on Python MIL.
3. **Do not wire `find_two_rois` as default** until real-clip bar is met
   (unchanged; issue #8).
4. **Goldens** — land `clip_voll` set (#146) so no-mark / profile bake-offs
   are measurable (windowed Motion Fidelity, not only whole-clip r).

---

## Proposed work order (clean approach)

Do **not** start a 4-zone rewrite and a profile redesign in one PR.

| Step | Work | Exit gate |
|------|------|-----------|
| **0** | Agree this doc (owner) | This file accepted or amended |
| **1** | Smoke v0.5.16; close or rewrite #145 | Log shows Go CSRT / Path: Go |
| **2** | **Recipe:** expose contact vibration on Normal + Auto (shared envelope; Tf/Tj default on) | Unit tests + one golden script metadata check; no tracking change |
| **3** | **Measure** `region_fusion_auto` (and/or successor) vs FunGen no-YOLO on `clip_voll` + 1–2 more clips, windowed | Numbers in NEXT/CHANGELOG; promote only if ≥ CSRT single-ROI |
| **4** | **Product UX:** Auto Generate = no-mark path when gate passes; manual mark = override | GUI copy: profile = feel; Advanced = tracking |
| **5** | **Decouple** Tf/Tj profile from mandatory ROI2 in GUI/CLI | Tf/Tj can run on single-curve; dual-ROI remains Advanced |
| **6** | Learning / class partners | Only after 3–5; train data from optional marks |
| **7** | ENGINE P2 “4-zone relative graph” (common-mode, reliability) | After goldens show need beyond activity-weighted fusion |

Steps 2 and 3 can overlap once 0–1 are done; 4–5 depend on 3.

---

## Explicit non-goals (for now)

- Making `find_two_rois` the silent default
- AI writing the funscript curve
- Dropping manual ROI entirely before no-mark beats marked CSRT on goldens
- Treating whole-clip FunGen r as the only quality number
- Another tracker dropdown (CSRT product path stays)

---

## Open questions for the owner (short)

1. **Auto profile** = “no-mark Generate” only, or also “autotune filtering”?
2. Contact vib on Normal/Auto: default **off** until hardware feel, or
   default **on** like Tf/Tj?
3. Keep showing “Tf/Tj” as a profile name in the GUI, or rename to
   something like “Suction / contact” once dual-ROI is Advanced-only?

Amend this file when those are answered; then start at step 1–2.
