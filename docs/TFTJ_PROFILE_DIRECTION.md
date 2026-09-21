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
| 4 | Where Tf/Tj “sits” | **Feel recipe on top of the best classical stroke path** — Normal/Auto recognition stays the curve writer; Tf/Tj (+ contact vib) is how Neo 2 *feels*, not a second tracker product. Owner (21 Sep pm): Normal recognition already works better; Tf/Tj should ride that, with Contact. |
| 5 | Milestone UX | **Show every usable motion candidate** the system can find; user mainly marks the **primary stroke**. Quality must not equal “how well you painted boxes.” Silent auto-commit of ROI2 remains forbidden. |
| 6 | YOLO while available | Optional class helpers (**penis / glans / nipple** and similar) as *proposals* for main + contact targets — never write the 0–100 curve. |

---

## Milestone (owner, 21 Sep pm) — “mark the main stroke”

Product bar we are aiming at:

```text
  Classical path finds / shows motion candidates (4-zone + overlays)
       ↓
  User confirms or paints ONE primary stroke region (optimal mark)
       ↓
  Optional: YOLO/class proposes penis / glans / nipple / partner
       ↓
  Profile = feel (Normal stroke vs Tf/Tj suction + Contact vib)
       ↓
  Funscript + device_recipe — curve from classical track, not from AI
```

**Why this is the right split**

- Measuring already showed: two-point distance alone does not close the
  FunGen gap; hub/single-ROI timing issues are shared. So “more Tf/Tj
  markers” is the wrong lever.
- Contact vib on Normal/Auto (#149) already decouples vib from “must be
  Tf profile.” Next is to make Tf/Tj *feel* the default when contact
  scenes are suggested — without forcing dual-ROI everyday Generate.
- Showing candidates (overlay / list) is honest UX; auto-committing
  `find_two_rois` was measured bad (issue #8) and stays out.

**Not this milestone:** AI invents positions; silent second ROI; park
classical until YOLO is “perfect.”

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
  Contact vib OFF      → Normal/Auto: no marks; Tf/Tj: still two markers (tip+partner)
  Contact vib ON
    + Normal/Auto      → optional partner mark; without it use depth envelope
    + Tf / Blow / …    → two markers required; track partner over time (not Fix default)
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
| **1** | Smoke v0.5.16; resolve #145 | Go CSRT / Path: Go in log — tip-only Autotune **DONE** #164; smoke **0.5.18** |
| **2** | Contact vib on **Normal + Auto**, default **on**, user toggle **off** | **DONE** #149 |
| **3** | Contact-family: Zone 2 always two markers for Tf/Tj; vib on → **tracked** partner (not Fix default) | **DONE** #160 |
| **4** | Measure no-mark 4-zone vs FunGen no-YOLO (windowed) | Promote only if ≥ single-ROI CSRT — **next measure** |
| **4b** | **Motion candidates UI** — show all usable motion/ROI proposals before Generate; user picks primary | **DONE** #167 — in **v0.5.19** |
| **5** | GUI rename of profiles; keep aliases | Copy review |
| **6** | Decouple profile pick from forced dual-ROI when vib off; Tf/Tj feel available on Normal stroke path | CLI/GUI — owner #4 |
| **6b** | YOLO class proposals (penis / glans / nipple) as opt-in helpers for primary + partner | Suggest ≠ commit; needs working AI Train (#119) |
| **7** | AI/pattern **profile suggest** (Blow vs Normal vs Tf-family vs Mix) | After G1 goldens; G3; override always |

### Next release train (after 0.5.18 smoke)

| Release | Ship | Why |
|---------|------|-----|
| **v0.5.19** | Step **4** measure + start **4b** candidate overlay (read-only) | Prove no-mark ≥ CSRT before UX bet; show motion without forcing marks |
| **v0.5.20** | Step **6** feel-decouple (Tf/Tj recipe on Normal path when vib on) + polish 4b pick-primary | Milestone “mark main stroke” usable |
| **later** | **6b** YOLO proposals + step **7** profile suggest | Only after classical candidates feel trustworthy |

Owner media still needed for step 4 FunGen windowed (~3 min clips).

---

## Explicit non-goals (for now)

- Silent auto-commit of two ROIs
- AI writing the funscript curve
- Mandatory marking for Normal/Auto when contact vib is off
- Treating whole-clip FunGen r as the only quality number
- Shipping profile auto-detect before vib-on-Normal/Auto + partner track

---

## Next concrete implementation (after this PR)

Steps **2–3** done. After **0.5.18** Autotune smoke (#145): start **step 4**
(windowed no-mark vs FunGen) in parallel with **4b** read-only motion
candidates. Do **not** leapfrog to YOLO auto-commit or profile AI suggest.
