# Tf/Tj as profile — direction (not yet implementation)

Owner direction, 21 Sep 2026 (answers amended same day). Think first;
implement only after this shape is agreed. Related: `docs/ENGINE.md`,
`docs/GENERATE_HEURISTICS.md`, `docs/NEXT.md`, `docs/FINDINGS_TIMING_TF.md`,
`docs/PRODUCTION_ROADMAP.md` (G1 / G3).

---

## What the owner wants (plain)

1. **Scene profiles are feel recipes**, not “recognition products.”
   Names will be renamed for clarity; later the **right profile should
   be suggested automatically from the motion/scene pattern** (likely AI).
2. **Everyday Generate does not force marking everything.** FunGen 2
   does not either (including without YOLO). Classical path: 4-zone
   sampling (+ audio as check/hint).
3. **Marking is optional and situational:** only for profiles that need
   a **contact partner** (Tf / blow / similar) **and only when the user
   wants contact vibration**. Then they mark the contact target. When
   that mark moves with the scene, we must **compensate** (track the
   partner, not assume a static box).
4. **Contact vibration on Normal and Auto** — default available / on,
   **user can turn it off**.
5. **Learning / AI** is for (a) later **auto profile pick** from pattern
   and (b) better partner proposals — not for inventing the 0–100 curve.

---

## Owner answers (21 Sep 2026) — locked

| # | Question | Answer |
|---|----------|--------|
| 1 | Auto / marking | **Mark only when needed:** Tf + blow (and similar contact scenes). Mark **only if contact vibration is wanted**. Not “mark everything.” When the marked partner **moves**, compensate (track it) — a dead static box will drift. |
| 2 | Contact vib on Normal / Auto | **Yes — on by default; user can disable.** |
| 3 | GUI names / profiles | **Rename** for clarity. Profiles stay **profiles**. Later: **auto-detect / suggest** profile from scene pattern (mix of current set + blow + more). That suggestion layer is **AI/pattern** work — not classical G1. |

---

## Diagnosis (what we already measured)

| Fact | Source | Consequence |
|------|--------|-------------|
| Whole-clip FunGen r≈0.06 on Tf/Tj goldens is mostly **lag drift**, not “Tf/Tj is noise” | windowed phase / #143, `clip_voll` | Do not redesign tracking from whole-clip r alone |
| Hub (single-ROI) on same clip also drifts; Signal Quality ≠ Motion Fidelity | goldens + NEXT | Timing drift is not Tf/Tj-specific |
| Two-point distance alone does not close FunGen gap on titjob POV | NEXT Sep 16 | Multi-mark is not the FunGen parity path |
| `find_two_rois` was measured insufficient for default | NEXT §3, issue #8 | Never silent-auto-commit ROI2 |
| `region_fusion` / `region_fusion_auto` exist; auto is synthetic-only | NEXT Sep 16 | No-mark classical candidate; measure before default |
| Contact vib works when distance signal is good | NEXT §5 | Decouple vib recipe from “must be two-ROI”; when partner is marked, **track** it |
| OpenCL checkbox forced Python → MIL on old builds | #141 / #145 | v0.5.16 removes trap; re-smoke before blaming “auto” |

**Bottom line:** three layers were conflated:

```text
  A. Tracking mode   → how we get a 0–100 curve from video
  B. Device profile  → how that curve maps to Neo 2 (suction / vib / recipe)
  C. Marking UX      → none | one ROI | contact partner (only if vib on)
  D. Profile suggest → later: pattern/AI picks B from the clip
```

---

## Target product shape

### Profiles (B) — feel recipes (rename in GUI)

Working set (names TBD in a small rename pass; keep CLI aliases):

| Profile (concept) | Feel | Marking (C) |
|-------------------|------|-------------|
| **Normal / hub** | Stroke / usual suction | None by default |
| **Auto** | “Just generate” classical path | None by default |
| **Weich / autotune** | Soft / filtered Normal | Same as Normal |
| **Tf / Tj** (rename e.g. suction/contact family) | Suction-centric | Partner mark **only if contact vib enabled** |
| **Blow** (and similar oral/contact) | Contact-centric recipe | Same rule as Tf |
| **Mix / others** | Combinations we already have + more later | Per-recipe rules |

Tf and Tj remain **aliases** for one suction-family profile until rename
ships. **Blow** is a sibling contact-family profile, not a second tracker.

### Contact vibration

| Context | Behavior |
|---------|----------|
| Normal / Auto | **On by default; user can turn off** |
| Tf / Blow (contact family) | On by default when that profile is selected; off disables partner-mark requirement |
| Signal without partner mark | Envelope from **deep / high-`pos` slice** of the single curve (honest: stroke depth, not tip↔body) |
| Signal with partner mark | Distance (or relative) envelope; **partner ROI is tracked** (motion compensation) — not a frozen box |

### Marking UX (C) — situational

```text
  Default Generate     → no marks (4-zone / single-curve classical)
  Contact vib OFF      → no partner mark required (any profile)
  Contact vib ON
    + Normal/Auto      → optional partner mark; without it use depth envelope
    + Tf / Blow / …    → partner mark expected; track partner over time
```

**Compensation rule (owner):** if the user marks a contact target and the
scene moves (camera, body, hands), the marked region must be **followed**.
Static “paint once and forget” will desync vibration from real contact.
Reuse existing multi-point / CSRT partner track; do not invent a new
tracker family for this.

### Tracking modes (A)

1. No-mark classical (4-zone / hub) — everyday
2. One primary ROI — override / Advanced
3. Primary + **tracked contact partner** — only when contact vib needs it
4. AI class proposals — G3; never write the curve

### Profile auto-suggest (D) — later, AI/pattern

Not G1. After classical Generate + vib recipes are solid:

- Input: motion signature, optional audio tempo, optional weak class scores
- Output: **suggested profile** (Normal / Blow / Tf-family / Mix / …)
- User can override; suggestion never silently locks a bad recipe
- Needs labeled examples over time (same training loop as #119 family)

---

## Bugs / debt before new UX

1. **#145** — smoke on portable **v0.5.16**; close or fresh bug with Go log.
2. Auto path must not fall back to Python MIL.
3. Do not default-wire `find_two_rois` (issue #8).
4. Land goldens (`clip_voll` / #146 content now on main via #140) for
   windowed Motion Fidelity bake-offs.

---

## Proposed work order

| Step | Work | Exit gate |
|------|------|-----------|
| **0** | This doc + owner answers (done) | Accepted |
| **1** | Smoke v0.5.16; resolve #145 | Go CSRT / Path: Go in log |
| **2** | Contact vib on **Normal + Auto**, default **on**, user toggle **off** | Shipped in this change set — tests + GUI always shows toggle |
| **3** | Contact-family: Zone 2 always two markers for Tf/Tj; vib on → **tracked** partner (not Fix default) | **In PR** — GUI Fix Zone 2 default off when vib on; always require Zone 2; simpletrack fixed-vs-tracked test |
| **4** | Measure no-mark 4-zone vs FunGen no-YOLO (windowed) | Promote only if ≥ single-ROI CSRT |
| **5** | GUI rename of profiles; keep aliases | Copy review |
| **6** | Decouple profile pick from forced dual-ROI when vib off | CLI/GUI |
| **7** | AI/pattern **profile suggest** (Blow vs Normal vs Tf-family vs Mix) | After G1 goldens; G3; override always |

---

## Explicit non-goals (for now)

- Silent auto-commit of two ROIs
- AI writing the funscript curve
- Mandatory marking for Normal/Auto when contact vib is off
- Treating whole-clip FunGen r as the only quality number
- Shipping profile auto-detect before vib-on-Normal/Auto + partner track

---

## Next concrete implementation (after this PR)

Start at **step 2** (contact vib → Normal/Auto, default on, can disable),
then **step 3** (tracked partner only when vib on for Tf/Blow — **done**).
Profile rename and AI suggest wait until those feel right on device.
