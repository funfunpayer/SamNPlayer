# SamNPlayer roadmap after v0.7.0 (15 Oct 2026)

One plan instead of several. It merges three sources:

- **ChatGPT's plans** from 12 Oct, kept in Git history:
  - `FREE_NEXT_LEVEL_SCOPE_2026-10-12.md` (commit `7720316`): Free stays stable; Next Level is the place for heavy innovation.
  - `NEXT_LEVEL_REPOSITORY_BOOTSTRAP_2026-10-12.md` (commit `c93b359`): repository, gates N0–N6, test levels L0–L7.
  - `STUDIO_EDITOR_COMPETITIVE_BLUEPRINT_2026-10-12.md` (commit `b074155`): what a full editor means, ED-01…ED-10, EDIT-00…08.
- **The comparison with other players** on 14 Oct: ScriptPlayer, ScriptPlayer+, BetterPlayer and MultiFunPlayer. FredTungsten's collection of 758 scripts served as a test corpus.
- **The Owner's wishes** of 14 and 15 Oct:
  - the editor becomes a full editor;
  - Go for the core, Rust where AI and video need speed.

Since 14 Oct Claude works on this alone. Earlier ChatGPT roles in these plans now belong to Claude. Nothing here is built yet unless it says ✅.

---

## 1. Languages: what goes where

| Part | Language | Why |
|---|---|---|
| App, devices, player, editor operations, formats, generator orchestration | **Go**, as now | One codebase, tested (about 60 000 Go lines incl. tests), fast enough, Wails. |
| Interface | JavaScript in Wails, as now | Already built. No toolkit change without proof. |
| **Hot paths in video and AI:** tracking, optical flow, frame decode loop, ONNX inference wrapper | **Rust**, as a separate library or helper binary called from Go | Speed and memory safety. A Rust part is built only when a profile shows Go/OpenCV is the bottleneck, and it must beat the measured baseline on the golden clips (ChatGPT's L6 gate: 3 warm + 5 measured runs, median and p95, no quality loss). |
| AI training, teacher models, experiments | Python, as now | The ecosystem lives there. It moves to Rust/Go only piece by piece, behind the same benchmark gate. |

**First Rust candidate:** the CSRT/flow tracking loop in `generator/trackcv` and the frame pipeline. Long clips are where the time goes, and drift fixes need many re-runs. Before that comes a profile on the Owner's clips (see "Motion recognition — next steps", `docs/owner/motion-recognition-next.md`, priority 1).

---

## 2. Two lines: Free and Next Level

This keeps ChatGPT's decision.

- **SamNPlayer Free** (this repo) gets stability, the comparison features below, and the full editor in steps. Every step must stay compatible with `.funscript` and `.samn`.
- **SamNPlayer Next Level** (separate private repo `SamNPlayer-Next-Level`) gets:
  - adaptive automatic generation per scene;
  - local AI as draft or repair, always checked by the classical verifier;
  - Rust prototypes;
  - batch processing;
  - Studio.

Bug fixes are ported between the two lines deliberately and with tests. There is no automatic merging.

**Update 15 Oct (Owner):** the 2.0 Next plan, specs and inventories now live in this repo under [`docs/next/`](next/MASTER_PLAN.md). Next is free like today's app (`docs/EDITIONS.md`); whether its code also moves here is open decision D12 there.

---

## 3. v0.8 (Free): what the other players do better

Each item here was seen in at least one other player. We build our own version and do not copy code: ScriptPlayer is GPL, BetterPlayer is BSL/GPL, and ScriptPlayer+ is closed. Only MultiFunPlayer (MIT) could be reused, and only with a notice.

| # | Feature | Seen in | Size |
|---|---|---|---|
| 1 | **Find scripts:** script folders besides the video's own folder; variants of one video (Soft/Hard), switchable; tolerant names, e.g. "Film (2019)". | all four | S–M |
| 2 | **Smooth stroker output, as a live playback filter** (the file stays untouched): PCHIP/MAKIMA interpolation instead of point to point; speed limit per axis; home position on pause; idle fill. Soft start ✅ v0.7.0. | MultiFunPlayer, ScriptPlayer, UwU Player | M |
| 3 | **Audio files with a script** (MP3, FLAC, …). | ScriptPlayer+ | S |
| 4 | **Library as tiles:** thumbnails, hover preview, sort and rating. | BetterPlayer, ScriptPlayer+ | M |
| 5 | **Skip long pauses** (with a delay); **keyboard shortcut sheet**, shortcuts customisable. | ScriptPlayer, ScriptPlayer+, MultiFunPlayer | S |
| 6 | **Device wizard:** connect step by step. | BetterPlayer | S |
| Large | **Native Handy connection:** upload the script, then server-time sync (HSSP). | ScriptPlayer, ScriptPlayer+ | L |
| Large | **Follow an external player:** VLC, MPV, VR players such as DeoVR and HereSphere. | MultiFunPlayer | L |

### 3b. From the EroScripts posts (read on 15 Oct)

| Project | What stands out | For us |
|---|---|---|
| **PeakSync** (browser, cloud AI) | Its own AI model makes scripts; videos are **uploaded**. Editor with smooth, clamp / normalize and **Speed Guard**; a player with a projected-output preview. Users praise the accuracy on hard clips. Complaint: uploading takes 30 min for 110 MB and is impractical for VR files. | Our generator runs **locally**: no upload, no credits. Say so clearly (forum post). Speed Guard = editor E7; projected output = E8. |
| **UwU Player** (Windows / Android, free + Patreon) | **Post-processing live during playback** without touching the file (invert, smoothing, speed limit, idle fill); library with watched / NEW badges and tags; **session recap** (time, max / average speed, distance, activity heatmap); native Handy with drift compensation; native Vorze. | Post-processing live = item 2 (do it as a playback filter, file untouched). Library badges / tags → item 4. Session recap: new idea, small (we log sessions already in Training). Vorze CSV import: small. |
| **ScriptPlayer-CN** (GPL fork of ScriptPlayer) | libmpv playback, library with NFO and tags, AB loop, hover preview, switch between scripts of **different authors** for one video. | Author switch = script variants (item 1). AB loop: we have loop selection ✅. |
| **Funscript Heatmap Generator** | Folder in → a heatmap per script, one **combined multi-axis heatmap**, speed statistics, chapter timeline. | We export a heatmap PNG for one script. Small add: multi-axis and a folder at once, with statistics. |
| **StrokeGPT** (chat-driven Handy) | Users found v2 broken, rolled back to v1.5; antivirus warnings. | Confirms: not now. |
| **Lovense software bridge** (question thread) | Games that only speak Lovense (Game Mode REST API) — users want them to drive an OSR2 / Handy. Idea: a local server that speaks the Lovense Game Mode API and forwards to Intiface. | Interesting, but it imitates a vendor's API; legal check first. Not now. |
| **BuzzyBody**, **Fap Hero Journey** | Android player for cheap LoveSpouse toys; a game-style sync player. | Far from our core. Not now. |

---

## 4. The full editor (Owner: "a real editor")

The target follows ChatGPT's blueprint §1. Below is what exists now and what is missing.

### What the editor can do today ✅

- Drag, add and delete points on every lane: stroke, vibration, suction, and the companion axes surge, sway, twist, roll and pitch.
- Start a new axis ("+ Axis…").
- Undo / Redo.
- Range tools on the heatmap selection: speed cap, scale, ramp scale, delete range, invert.
- Chapters, bookmarks, O-markers and BPM grid.
- *Optimize for* (general / Sam Neo 2) and the Sam Neo 2 feel per scene and chapter.
- Editing right in the Generator (Generator with editor).

### What is missing, in order

| Step | Content | Test (from the blueprint) |
|---|---|---|
| **E1 Selection** | Lasso / box select of points; select all / range / chapter / peaks / valleys; multi-select across one lane; numeric point edit (time, position). | ED-03 |
| **E2 Clipboard and keys** | Copy / paste at the playhead (relative time); nudge with arrows (frame / ms / position); delete with a gap policy; frame step (`,` `.` exist, made frame-exact). | ED-07 |
| **E3 Transform library** | Shift, stretch / compress time, mirror, smooth (with preview), equalize, normalize to a range, repeat a pattern, ripple insert / delete, retime to BPM. | ED-03, numeric reference tests |
| **E4 Patterns** | Pattern cards (wave, tease, ramp, …), inserted into a range or a gap; the same engine fills gaps in the player. | ED-03 |
| **E5 History** | Undo as commands with transactions (a drag is one step; a multi-axis edit is all or nothing); a history list with before/after; history survives reopening (journal). | ED-04 (1000 mixed edits, crash recovery) |
| **E6 Compare** | Ghost lane of the original or another version; A/B with one shared playhead; an AI or *Optimize* result shown as a diff, with accept / revert. | ED-09 |
| **E7 Quality on the timeline** | Warnings on the lane: duplicate or unordered times, spikes, too fast for the device, tracking gaps. Fix preview next to them. | — |
| **E8 Device preview** | Shows what the device really gets after limits. It only moves the device after a deliberate *Arm*. | ED-08 |
| **E9 Export pack** | One file, one axis, or the whole companion pack, with a round-trip check. | ED-01, ED-02, ED-10 |
| **E10 Scale** | A 1-hour video with 100 000+ points stays smooth at 1366×768 and 1920×1080. | ED-06, ED-07 |

E1–E5 belong in Free, because they are basic editor work. E6–E10 can follow in Free or Next Level, whichever comes first.

---

## 5. Things we deliberately do not build (for now)

- Chat control in the style of StrokeGPT, "Together" sessions, or a game mode. They are far from the core, and some need a server.
- An in-app browser with an EroScripts download. It raises legal and technical questions.

---

## 6. Order

1. **v0.7.0:** the Owner tags and checks; Claude fixes what comes up.
2. **v0.8:** Section 3, items 1–3 and 5, plus editor steps E1–E2.
3. **v0.9:** editor steps E3–E5, items 4 and 6.
4. **Profile, then the first Rust step** for tracking, under the L6 gate.
5. **Next Level:** gates N0–N6 from ChatGPT's bootstrap plan, in its own repo.
