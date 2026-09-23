# Emotion Script (product format)

**Product name:** Emotion Script  
**File on disk:** `.samn` (SamNPlayer native)  
**App framing:** SamNPlayer is an **Emotion Generator** (and can play community files).

## One format for users

| What the user sees | File | Role |
|--------------------|------|------|
| **Emotion Script** | `.samn` | **The** product script — create, play, save |
| Share / other apps | `.funscript` | Import + optional export only |
| *(never shown)* | `.sam` | Internal contact-intensity sidecar — not a second script |

Generate writes / opens Emotion Script (`.samn`). Companion `.funscript` may still
exist for tooling sync, but the GUI talks about **one** Emotion Script.

Overwrite confirmations must disclose both the Emotion Script and its
`.funscript` companion when either can be replaced. The companion may be the
only existing file. One product format must not hide a destructive side effect.

Do not present `.sam` as a script type in the GUI.

## Why not “Funscript” in the product UI

Funscript remains the community interchange format. In the app we say
**Emotion Script** so the brand and the native Neo-2 axes story stay clear.
Export for other apps still offers `.funscript` under a share-style label.

## Related

- Wire format: [SAMN format](SAMN_FORMAT.md)
- Everyday Create: [Everyday Create](EVERYDAY_GENERATE.md)
