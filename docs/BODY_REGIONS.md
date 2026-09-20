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
| `tracked` | CSRT / two-point tracker follows the box |
| `fixed` | Box stays where marked (static anchor) |
| `mask` | Soft-exclude / ignore for stroke; still labeled for AI |

Tf/Tj still drives the script from **two** distance partners (tip ↔
target). Other marked classes are for AI suggestion, masking, and
dataset quality — not a third distance axis yet.

## Surfaces

| Surface | Capacity |
|---------|----------|
| AI training marks | up to **9** boxes/classes per frame |
| Generate / Tf/Tj stroke | ROI1 + ROI2 (+ optional fixed flag on ROI2) |
| AI suggest | prefers canonical class list (`preferred_classes`) |
| Segmentation masks | roadmap — box roles first |

## Owner test

1. AI training: mark Face + Mouth + Breasts + Nipples + Hand 1 + Hand 2 +
   Penis + Glans + Vagina on one still → “Use for training”.
2. Generate Tf/Tj: tip = Glans (tracked), target = Nipples (fixed) →
   distance still moves when only the tip moves.
3. Settings preferred classes defaults to the CSV of all nine IDs.

Code: `generator/bodyparts`, `docs/TF_TJ.md`, `docs/KI_TRAINING.md`.
