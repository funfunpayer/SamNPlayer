# Everyday Generate — FunGen2-like (clip → done)

Product target (owner, 22 Sep 2026): pick a clip; barely choose anything
else — at most **AI region on/off** and **where vibration should feel**
(Contact on/off + optional Zone 2). Everything else automatic.
**Best measured method is the first choice.**

Related: `TFTJ_PROFILE_DIRECTION.md`, `GENERATE_HEURISTICS.md`,
`AUDIO_WORKFLOW.md`, `NEXT.md` § Stroke preview / clip_ausschnitt matrix.

---

## Measured first choice (`clip_ausschnitt`, windowed FunGen r)

| Rank | Method | Windowed mean r vs ohne_yolo |
|-----:|--------|-----------------------------:|
| **1** | **CSRT hub / tip (Go)** | **0.59 / 0.47** |
| 2 | region_fusion tip | 0.56 |
| 3 | grid_lk tip | 0.49 |
| … | 4-zone `region_fusion_auto` | **0.36** — opt-in only |
| … | Flow | slow + weaker — not everyday |

**Rule:** Everyday = **auto-find tip region → CSRT → Stroke + Contact vib**.
Do **not** default 4-zone or Flow.

---

## User-visible path

```text
  1. Choose video
  2. Auto: find tip region (classic motion, or AI if checkbox on)
       — user may correct the box or pick a motion candidate
  3. Optional: Contact vibration on (default) · optional Zone 2 “where it vibrates”
  4. Generate → .samn + .funscript
  5. Review & improve: trim start/end · fill gaps · audio check on/off
       — then **Play**: soft curve + keyframe dots (FunGen-like); Edit curve to adjust
```

Advanced (collapsed): knobs that **change CSRT output** (invert, cam-comp,
scene-cut, dynamics, retry, axis, smooth, peak spacing, RDP, max speed).
Removed from Everyday Advanced: AI second opinion (no curve effect), audio
check row (moved to Review), Flow downscale (Flow is CLI-only). Scene memory
collapsed under Power-user. Imported scripts → Play **Optimize for Neo 2**
(`docs/CONTENT_SOURCES.md`).

AI checkbox = **region proposal only** (ONNX). Never writes the 0–100 curve
(`AI_ADAPTER.md`).

---

## What stays automatic

| Piece | Behavior |
|-------|----------|
| Backend | CSRT (Go when OpenCV linked) |
| Profile | Stroke (`standard`) |
| Contact vib | On by default |
| Stroke preview | Stage A pre-pass (timing/flags; audio if weak) |
| Audio check | On when ffmpeg present (generate + Review toggle) |
| Fill gaps | **Auto after Generate** + Review step — linear bridge; optional audio-tempo spacing |
| Trim start/end | Review step (start also via seek before Generate) |
| Flow downscale | Hidden unless Flow (CLI) |
| Tf/Tj distance | CLI / Advanced only — not everyday GUI |
| License | Gates wired (`MaxOutputMs` trial / `.samn` Play); Enforcement still **off** |

---

## Acceptance

1. New user: video → Generate without painting a box (auto-ROI ran).
2. Result path is CSRT Stroke, not 4-zone, unless they opted in.
3. AI off by default; on only proposes ROI.
4. Clip gate: everyday path ≤ few % behind measured tip/hub CSRT on goldens.
5. After Generate: gaps filled once (Go); Play shows dots; Edit is opt-in.
6. Stale auto-ROI results from a previous video are ignored.

---

## Remaining Python (Everyday)

| Step | Still Python? | Notes |
|------|---------------|-------|
| Auto-find tip | Yes (`auto_roi.py`) | Highest-value Go port next |
| CSRT track + posttrack + QD + audio | **Go** when OpenCV linked | Windows → Python CSRT until #120 |
| Improve trim/fill | **Go** (`funscript.ImproveScript`) | |
| AI region | ONNX (`ai_roi.py`) | Opt-in only |
