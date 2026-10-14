# Scene feel — how the Sam Neo 2's suction follows a stroke

**Owner, 14 Oct 2026:** the Sam Neo 2 must render oral, intercourse and breast
scenes well. Suction is the important channel. Vibration may follow the same
pattern, and a mix moves the weight between the two.

Code: [`funscript/feel_scene.go`](../funscript/feel_scene.go) (algorithms),
[`funscript/mapper.go`](../funscript/mapper.go) (where they apply),
[`cmd/gui-wails/app_feel.go`](../cmd/gui-wails/app_feel.go) (save into a script),
GUI: Generator → *How it feels* (`gen_feel.js`) and Editor / Player →
Inspector → Devices → *Sam Neo 2 feel* (`feel_card.js`).

## The classic mapping stays

Without a scene, with the mix in the centre and "Vibration follows suction"
off, nothing changes:

- **Suction** = stroke depth (position).
- **Vibration** = stroke speed, or the contact vibration near the deep point.

Tests prove it frame for frame (`TestSceneFeelOffKeepsOldMapping`).

## Scene feels

The mapper gives each scene its own suction shape. Inputs:

- **depth** = position / 100;
- **intensity** = speed / MaxSpeed, so it rises with every stroke.

```
suction = Floor + Range × (DepthW × depth + SpeedW × intensity)
```

| Scene | Floor | Range | Depth | Speed | Feel |
|---|---|---|---|---|---|
| **Oral** | 0.30 | 0.70 | 0.65 | 0.35 | A steady seal that never lets go; deeper strokes pull harder, and each stroke adds a pulse |
| **Intercourse** | 0.15 | 0.85 | 0.45 | 0.55 | Follows the rhythm first, depth second |
| **Breast** | 0.20 | 0.55 | 0.70 | 0.30 | Softer; the squeeze follows depth |
| **Hand** | 0.10 | 0.70 | 0.25 | 0.75 | The grip follows speed |

The AI scene types map onto these:

| AI scene type | Scene feel |
|---|---|
| blowjob | Oral |
| penetration | Intercourse |
| titjob | Breast |
| handjob | Hand |

The Generator's **Auto** picks the scene in this order:

1. The scene detection, when a scene proposal was found.
2. The contact mark's class: *mouth* → Oral, *nipple / breast* → Breast.
3. Otherwise Classic.

## Vibration follows suction · Mix

- **Vibration follows suction:** vibration takes the suction pattern, so both
  move together.
- **Mix** (−100 … 100 %): the favoured channel stays full, and the other is
  reduced, down to 0 at the ends. At *Suction 40 %*, vibration runs at 60 %.

## Where it is stored

The settings live in `metadata.device_recipe` (`.funscript`) or in the
`.samn` recipe:

- `scene_feel`
- `vibration_follows_suction`
- `feel_balance`

An Emotion Script that plays from baked axes is re-baked on save, so the
device feels the change.

## The device decides what is shown (Generator)

| For device | Shown |
|---|---|
| **Sam Neo 2** (also when no device is connected) | Contact vibration + Sam Neo 2 feel |
| **Vibrator / other toy** | Contact vibration (the video's sound can drive it too: Player → Audio) |
| **Stroker** (The Handy, OSR2 …) | Neither — the stroke is all it needs |

*Show all* brings the hidden settings back. The feel is written only for a
Sam Neo 2, and only when it differs from Classic.

## Vibration from the video's sound (vibrators)

For a vibrator or another toy without suction, the vibration lane can come
from the video's loudness. It is written into the script, so it plays the
same everywhere and can be edited like any other lane.

**Sources:**

- *Contact (as created)*: the script's own vibration on deep strokes; nothing
  is written.
- *Sound + contact*: whichever is stronger at each moment, the video's
  loudness or the script's own vibration.
- *Sound only*: the video's loudness.

**Where to set it:**

- Generator → *For device: Vibrator* → *Vibration*. It is written right after
  Create.
- Inspector → Devices → *Vibration from sound* → *Write into script*.

Code: `cmd/gui-wails/app_sound_vibration.go` (`BakeSoundVibration`).

**How the loudness becomes a curve:** the decoder is the same one the
Player's live *Audio vibration* uses (`generator.AudioVibrationEnvelope`). It
measures loudness in 50 ms steps and normalises it in dB. Silence is gated
out, and beats are kept punchy without dropping to zero between them.

**Suction is kept:** when the script has no explicit suction lane yet, its
mapped suction is written first.

## Next

- **Scene per chapter:** Auto picks the scene for each chapter from motion
  (classic) and from the scene detection (AI). These are suggestions; you
  confirm them first.
