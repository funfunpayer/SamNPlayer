# SamNPlayer 1.0 — build plan (Owner, 11 Oct 2026)

**Target:** the Owner's final mockup of 11 Oct, "this is the new look". The
release is **Version 1.0**, built together by Claude and ChatGPT. The mockup
image is the visual reference. It contains no unsafe media, but stays out of
the repo (it is a raster concept, not an asset).

## Owner decisions (11 Oct)

| Topic | Decision |
|---|---|
| Look | Flow theme (navy / cyan / warm orange), airy, not boxed. Done in #480 |
| Top tabs | **Player · Generator · Training · Scene Map · Tools** (exactly as the mockup) |
| Tagline | GENERATE · PLAY · TRAIN |
| Channels | **Only real channels:** Stroke · Vibration · Suction. No "Axis 3 Thrust / Axis 4 Rotation" (the Sam Neo 2 has no such motors) |
| Playback speed | **Build** 0.5× – 2×. Video and device stay in sync (script time scaled) |
| Auto sync (audio) | **Build**: estimate the script offset from the video audio. It is a suggestion; the user confirms |
| Functions | Every control in the mockup that maps to a real feature works. New features are built where decided. Nothing is shown that does nothing |

## Target layout (from the mockup)

- **Top bar:**
  - SN Flow icon + "SamNPlayer" (SamN orange, Player cyan) + tagline;
  - tabs Player / Generator / Training / Scene Map / Tools;
  - right: open file (folder), Settings (gear).
- **Left: Library**
  - Local Files, Recent, Favorites, Playlists;
  - Scripts: Imported / Generated / Edited;
  - Tools: Scene Map, Audio Analysis, BPM Detection;
  - Quick Access drop zone.
- **Centre: Player**
  - video file selector (with recent files);
  - icons: note/comment, bookmark, snapshot, fullscreen;
  - the video;
  - progress bar;
  - transport: loop, sleep timer, speed, previous, big circular play, next, volume, fullscreen.
- **Right: Inspector** with tabs **Devices · Audio · Playback · View**
  - Devices:
    - Enable device + profile (strength preset);
    - a row per real channel: toggle, level %, live mini wave, settings;
    - Mode (Recipe / Axes), Sync (video sync).
  - Effects / Curve / Limits / Advanced:
    - smoothing, *Sync with audio* (audio vibration), Invert, Reduce peaks (max speed);
    - knobs Intensity / Response.
  - Cards:
    - **Audio Analysis**: BPM with waveform, "From audio", Audio Vibration;
    - **Script Tools**: Write into script, Apply offset ±, Auto sync (audio).
- **Bottom: Timeline** with tabs **Timeline · Chapters · Bookmarks · Scene Map · Audio Analysis**
  - time ruler;
  - lane name column with colour dots;
  - stacked lanes (Stroke / Vibration / Suction) + Audio Level;
  - shared playhead;
  - scene markers (diamonds) + scene thumbnail strip, with "+" to add a marker.

## Steps — one PR each, merged on green CI

| # | Step | Owner | Notes |
|---|---|---|---|
| S1 | Flow theme + compact Play layout | Claude | ✅ #480 |
| S2 | **App shell**: top bar with tabs (Player, Generator, Training, Scene Map, Tools), folder + gear, logo/wordmark/tagline; rail removed; Tools sub-nav kept | Claude | IDs and `data-tab` values unchanged; Scene Map tab shows the Generator scene-map section for the current video |
| S3 | **Player 3-column shell**: Library left, Inspector right (tabs Devices / Audio / Playback / View), timeline tabs bottom; existing controls moved into their tab, IDs unchanged | Claude | Collapsible panels, state remembered |
| S4 | **Timeline core**: time ruler, lane names with colour dots, Audio Level lane (envelope), scene markers (diamonds) + "+" add, thumbnail strip | Claude | Go: thumbnails via ffmpeg (cached), audio envelope reused |
| S5 | **Transport**: big circular play, loop, previous / next, volume, fullscreen, **speed 0.5–2×** (Go: playback time scale), sleep timer | Claude | Speed is new core; device sync tested |
| S6 | **Library**: Recent, Favorites, Playlists, Scripts Imported / Generated / Edited (Go library index), Tools shortcuts, Quick Access drop zone, video selector dropdown | Claude | Local only |
| S7 | **Inspector, Devices + Effects**: per-channel enable + level + live wave, profile, Mode, Sync; Effects / Curve / Limits / Advanced; knobs | Claude | Per-channel level is new core (device-safe) |
| S8 | **Audio Analysis + Script Tools cards**, incl. **Auto sync (audio)** (Go: offset estimate audio↔script, suggest-only) | Claude | Needs a measurement on real clips |
| S9 | **Brand assets**: Flow icon (EXE ICO 16–256, 1024 PNG), banner, website/README | Claude integrates; ChatGPT + Owner supply the final files | Exact files, no redraw |
| S10 | **Other tabs** in the same style: Generator, Training, Scene Map, Tools (Device / Log / Settings) | Claude | Shared cards, buttons, spacing |
| S11 | **Release 1.0**: `VERSION` 1.0.0, CHANGELOG, handbook, screenshots, QA at 1366×768 and 1920×1080; the Owner pushes the tag | Claude + ChatGPT | Full test matrix (`docs/QA_CHECK_*`) |

**ChatGPT, per step:**
- acceptance checklist and visual review on each PR;
- copy and labels;
- handbook and README text for the new names;
- brand asset handover (S9);
- public-site alignment after S9.

ChatGPT doesn't edit `playback.js`, `main.js`, `index.html` or `style.css` while a Claude step PR is open. Hooks are listed in `docs/PLAY_UI_CONTRACT.md`.

## Not in 1.0 (unless the Owner asks)
- Thrust / Rotation axes.
- The "Frequency" knob: Neo 2 vibration has intensity only, so the mockup knob row becomes Intensity · Response.
- Picture-in-picture.
- Frameless window controls: the min / max / close buttons in the mockup are the OS frame.

## Rules
- Every step:
  - keeps all IDs and bindings;
  - has tests for new behaviour;
  - runs the full frontend suite, Go for both editions, and Python.
- Before/after screenshots at 1366×768 and 1920×1080.
- Board row before each step (`docs/AGENT_COORD.md`).
