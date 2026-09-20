# Body regions — multi-part detect + fix + mask

Product taxonomy for **AI training**, **Generate**, and **Tf/Tj**.
UI and class IDs stay **English**. See `docs/LANGUAGE.md`.

## Why

Tf/Tj needs more than two anonymous boxes: several body parts should be
**detected**, then **locked** (fixed) or **tracked**, and extras should be
**masked** so motion does not leak into the wrong tissue. The same classes
feed normal Generate (single stroke box) and the local YOLO trainer.

## Canonical classes (English IDs)

| ID | Label | Typical generate role |
|----|-------|------------------------|
| `face` | Face | mask |
| `mouth` | Mouth | fixed (contact target) |
| `breasts` | Breasts | fixed |
| `nipples` | Nipples | fixed (common Tf/Tj target) |
| `hand_1` | Hand 1 | tracked |
| `hand_2` | Hand 2 | tracked |
| `penis` | Penis | tracked |
| `glans` | Glans | tracked (common Tf/Tj tip) |
| `vagina` | Vagina | fixed |

Legacy German labels (`brust`, `eichel`, `mund`, …) **normalize** to these
IDs in Go (`generator/bodyparts`), Python (`bodyparts.py`), and the GUI
(`bodyparts.js`).

## Roles

| Role | Meaning |
|------|---------|
| `tracked` | CSRT / multi-point tracker follows the box |
| `fixed` | Box stays where marked (static anchor) |
| `mask` | Soft-exclude punch-out for camera / grid_lk features; not a distance driver |

## Tf/Tj distance partners

- **Tip (ROI1):** always tracked.
- **Primary target (ROI2):** required for Tf/Tj; optional `--roi2-fixed`.
- **Extra targets (`--target`, repeatable):** additional contact anchors
  (GUI: **+ Target**). Default **fixed**. Stroke signal =
  **min** 2D distance from tip to ROI2 and all extra targets.
- **Soft masks (`--mask`, repeatable):** GUI **+ Mask**. Punched out of
  camera-motion and grid_lk reseed feature masks only — they do **not**
  drive the stroke. Not pixel-perfect SAM segmentation; box soft-exclude.

Native Go pipeline stays on the Python path when ExtraTargets or MaskROIs
are set.

## Surfaces

| Surface | Capacity |
|---------|----------|
| AI training marks | up to **9** boxes/classes per frame |
| Generate / Tf/Tj stroke | ROI1 + ROI2 + N extra targets (min distance) + soft masks |
| AI suggest | prefers canonical class list (`preferred_classes`) |

## Owner test

1. AI training: mark Face + Mouth + Breasts + Nipples + Hand 1 + Hand 2 +
   Penis + Glans + Vagina on one still → “Use for training”.
2. Generate Tf/Tj: tip = Glans (tracked), target = Nipples (fixed) →
   distance still moves when only the tip moves.
3. Extra target: add a second contact (+ Target, e.g. mouth) → stroke
   follows the **nearest** partner.
4. Soft mask: + Mask over a busy background patch → tracking/camera
   features ignore that box.
5. Settings preferred classes defaults to the CSV of all nine IDs.

Code: `generator/bodyparts`, `docs/TF_TJ.md`, `docs/KI_TRAINING.md`.
