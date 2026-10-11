# Anleitung (Deutsch) — Index

Owner-Guides liegen unter **[`docs/owner/`](owner/)**. Stand Produkt: **v0.5.40**.

| Start | Link |
|-------|------|
| Index | [`owner/README.md`](owner/README.md) |
| Funktionen | [`owner/anleitung-funktionen.md`](owner/anleitung-funktionen.md) |
| Zusammenhang & Testen | [`owner/zusammenhang-und-testen.md`](owner/zusammenhang-und-testen.md) |
| Roadmap + Meilensteine | [`owner/roadmap.md`](owner/roadmap.md) |
| Markieren | [`owner/generator-markieren.md`](owner/generator-markieren.md) |
| Win11 AI-Setup | [`owner/setup-windows11/README.md`](owner/setup-windows11/README.md) · Skripte [`scripts/setup-windows11/`](../scripts/setup-windows11/) |
| Benchmark | [`owner/benchmark-system.md`](owner/benchmark-system.md) · Clip-Prep [`owner/benchmark-clip-prep.md`](owner/benchmark-clip-prep.md) |
| Plugins | [`PLUGIN_INSTALL_DE.md`](PLUGIN_INSTALL_DE.md) · Contract [`PLUGIN_SYSTEM.md`](PLUGIN_SYSTEM.md) |

### Play audio and timing controls

The source changelog lists these controls under **v0.5.46**. This is a
feature guide, not confirmation of a published private or public download.

| Control | Where / requirements | Effect |
|---------|----------------------|--------|
| **Vibration from video audio only** | Play → **Playback & Extended-O**, or Settings → **Play**; playable script + video audio + ffmpeg | Default off; remembered; takes effect on the next Play start. Replaces vibration only; suction follows the script (0 after the script ends). Decode failure falls back to script vibration with a Play-log message. No file edits. |
| **From audio** | Play → **Curve edit (Cap · Scale · Speed HL · BPM)** → beside BPM; loaded script + video audio + ffmpeg | Measures the first 10 minutes, fills 20–240 BPM and enables the visual grid. No channel/file changes; failure keeps the current BPM/grid state. Manual BPM remains available. |
| **Write into script** | Play → **Offset** → **Write into script**; loaded writable script, nonzero offset, confirmation | Saves the shift on all three axes, bookmarks and chapters; removes negative-time points and resets offset to 0 on success. Keep an original copy; partial saves are possible on error. |

Detailed steps, silent-audio behavior, chapter boundaries and metadata
limitations: [User handbook — Play](USER_HANDBOOK.md#4-play--quick-map).

### Neu ab v0.5.40 (kurz)

| Feature | Wo in der GUI | Anleitung |
|---------|---------------|-----------|
| **Contact Verify** | Create → Advanced → **Verify with the engine (hybrid, K=1.5)** neben Use contact points | [`anleitung-funktionen.md`](owner/anleitung-funktionen.md) §6 |
| **Bench Suggest** | Tab **Bench** → Video → **Suggest beside video** (FunGen-Ref + Everyday `__hub`) | [`anleitung-funktionen.md`](owner/anleitung-funktionen.md) §13 · [`benchmark-system.md`](owner/benchmark-system.md) |

| Release | Link |
|---------|------|
| **v0.5.40** | Contact Verify + Bench Suggest — Tag [`v0.5.40`](https://github.com/funfunpayer/SamNPlayer/releases/tag/v0.5.40) |
| v0.5.38 | [`owner/v0.5.38-sammel.md`](owner/v0.5.38-sammel.md) |

**Abgrenzung:** Virtual Person (#264) ist **cancelled / out of scope** — nicht geparkt. Everyday = Go CSRT.

In-App handbook (English product copy): [`USER_HANDBOOK.md`](USER_HANDBOOK.md).
