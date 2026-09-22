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
  4. Generate → .samn + .funscript → Play
```

Advanced (collapsed): Soft/Autotune, 4-zone, Flow downscale, peak/RDP, …

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
| Audio check | On when ffmpeg present; forced if preview weak |
| Flow downscale | 0.5 if Flow ever selected |
| Tf/Tj distance | CLI / Advanced only — not everyday GUI |

---

## Acceptance

1. New user: video → Generate without painting a box (auto-ROI ran).
2. Result path is CSRT Stroke, not 4-zone, unless they opted in.
3. AI off by default; on only proposes ROI.
4. Clip gate: everyday path ≤ few % behind measured tip/hub CSRT on goldens.
