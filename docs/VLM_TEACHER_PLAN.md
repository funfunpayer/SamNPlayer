# Local VLM teacher → our own detector (plan)

Status: **V0 probe tool + scorer + golden reference labels in this PR** (Claude). V1+ not started.
Owner go, 27 Sep 2026: *"The system has to get better. Start if you can,
check whether it holds up. Best later: our own model that uses the data we
generated with Qwen & co. Everything is approved — coordinate with Cursor."*
Owner hardware: one NVIDIA card with **16 GB**.

This doc plans the mechanisms and their order. It changes no default. It
extends `docs/SCENE_MAP_PLAN.md` (marks, learning export, L2 detector) and
follows `docs/AI_ADAPTER.md` (the AI proposes, the existing system measures).

---

## 1. Why

Rhythm-grid numbers (windowed 30 s best-lag r vs FunGen ohne / mit YOLO):

| State | `clip_voll` | `clip_ausschnitt` |
|---|---|---|
| CSRT only | 0.386 / 0.552 | 0.449 / 0.712 |
| Grid, best 23 Sep (no lock) | 0.411 / 0.767 | 0.466 / 0.877 |
| Grid + identity lock (#248) | 0.427 / 0.626 | 0.383 / 0.666 |
| Grid + lock release (#287, main) | 0.455 / 0.752 | 0.486 / 0.887 |

Every fix so far tunes *where the motion is*. None of them knows *what* is
moving:
- CSRT follows a pixel patch.
- The rhythm grid follows the cell that moves at the stroke tempo.

That is why the known failures happen:
- The ROI starts on the breast or a hand next to the stroke (IdLock root cause).
- The thigh rocks at the same tempo.
- CSRT drifts off the shaft.

A human (or a vision-language model looking at the frame) sees at once
where the contact point is. The engine needs that knowledge, but it needs it
per frame, fast, and local. A 7B VLM is too slow for that and too big to
ship. So:

> **The VLM is the teacher, not the engine.** It labels keyframes offline.
> A person confirms the labels. Our own small detector (ONNX, the existing
> L2 path) learns from them and runs per frame in Generate.

---

## 2. Pipeline

```
video ─► keyframes (per shot + every N s)
          │
          ▼
   local VLM (Qwen-VL via Ollama / LM Studio, OpenAI-compatible API)
          │  boxes: contact / penis / glans / hand / breasts / thigh / face …
          ▼
   V0 probe JSON  ──► measurement on goldens (does it hold up?)
          │
          ▼
   V1 SceneMap marks  author:"vlm", reviewed:false  ──► user accepts / rejects
          │
          ▼
   V2 dataset (SceneMap M5 export; reviewed only) + CSRT propagation
          │
          ▼
   V3 own detector (YOLO-type → ONNX, ai_roi.py contract)
          │
          ▼
   anchor for rhythm grid / CSRT seed / exclude, per frame, in Generate
```

These rules hold throughout:
- **Local only.** Frames never leave the machine. There is no cloud VLM and
  no telemetry. No model ships in the repo and nothing downloads
  automatically; the same rule already applies to `ai_roi.py` and Colibri.
- **Review before training.** This is SceneMap § 6 decision 3. VLM boxes
  start as `reviewed:false` and never enter training unreviewed.
- **The AI never writes the script.** VLM boxes only choose *where* the
  engine looks. The curve still comes from flow and tracking.

---

## 3. Runtime and models (16 GB NVIDIA)

| Model | VRAM (approx.) | Licence (check at download) | Role |
|---|---|---|---|
| **Qwen2.5-VL-7B-Instruct**, Q4_K_M / Q8 | ~6 / ~9 GB | Apache-2.0 | **default teacher** |
| Qwen3-VL ~8B (if the local runtime has it) | ~6–10 GB | check | second opinion / later swap |
| Qwen2.5-VL-3B-Instruct | ~3–4 GB | Qwen *research* licence (non-commercial) | plumbing tests only, **not** for product data |

- **Runtime:** Ollama (`ollama pull qwen2.5vl:7b`, port 11434) or LM Studio
  (port 1234). Both speak `/v1/chat/completions` with `image_url`
  base64 parts. `virtualperson/chat.go` already uses both, and the probe
  reuses `generator/colibri_client.py` for the HTTP call.
- **Coordinates differ per model family.** Qwen2.5-VL returns absolute
  pixels of the image it was sent. Qwen3-VL is documented as 0–1000
  relative. Others return 0–1. The probe therefore sends frames already at
  a model-friendly size (multiples of 28, long side ≤ 896), so the model
  does not resize them. It also has a **calibration mode**: a synthetic image
  with known coloured rectangles, whose answer shows which convention the
  model uses. The probe never guesses the convention.
- **Explicit content is the main unknown.** General VLMs are often weak on,
  or evasive about, explicit anatomy. V0 measures the refusal rate and box
  quality before anything is built on top.

---

## 4. Phases

### V0 — probe: does it hold up? (Claude, this PR)

Tool: `generator/vlm_probe.py`. It is opt-in and has no dependencies beyond
the generator's own.

The tool does four things:
- Samples keyframes: the middle of each shot (`detect_scene_boundaries`)
  plus one frame every `--every-s` seconds.
- Sends each keyframe with one fixed, versioned prompt. The prompt asks for
  JSON boxes with labels from a small vocabulary: the canonical body regions
  plus `thigh` and `contact`.
- Writes `*.vlm.json`. Per keyframe it holds: time, size sent, latency, raw
  answer, `refused`, and normalised boxes 0..1.
- Optionally writes overlay JPEGs, so a person can check the boxes by eye.

**Oracle ceiling on the two goldens (measured 27 Sep, before any Qwen run).**
Claude labelled the keyframes itself: 28 on `clip_voll` (every 10 s) and
10 on `clip_ausschnitt` (every 5 s). Each label is a `contact` box plus
thigh / hand `exclude` boxes. The labels are committed as
`testdata/golden_clips/*/vlm_oracle.json`. They were fed to the engine as
SceneMap marks in the offline harness, which reproduces #287:
0.456 / 0.752 and 0.482 / 0.888.

| Variant (r ohne / mit) | clip_voll | clip_ausschnitt |
|---|---|---|
| #287 baseline (no marks) | 0.456 / 0.752 | 0.482 / 0.888 |
| contact box as `source` mark | 0.393 / 0.662 | 0.454 / 0.867 |
| thighs / hands as `exclude` | = baseline (never chosen anyway) | = baseline |
| ROI seeded on the first contact box | 0.393 / 0.662 | 0.501 / 0.848 |
| re-seed whenever the locked cell leaves the contact box | 0.444 / 0.770 | 0.454 / 0.867 |

What this shows:
- **"Where" is already solved on these two clips.** Since #287 the chosen
  cell lies inside the hand-labelled contact box in 118 / 140 windows
  (`clip_voll`) and 24 / 25 (`clip_ausschnitt`). The misses are
  neighbouring cells around shot changes.
- **A `source` mark only acts at a lock (re-)seed.** The identity lock
  overrides it afterwards (the M3 caveat in `rhythm_grid.go`). Seeding on a
  box edge cell (120 / 136) instead of 119 costs about 0.09.
- **The references cap what we can measure.** The two FunGen references
  agree with each other at only r ≈ 0.45, windowed:
  - `clip_voll`: 0.45, with windows ranging from 0.85 down to 0.15.
  - `clip_ausschnitt`: 0.45.

  Our r against ohne (0.46 / 0.48) is already at that level.

So a VLM cannot show a gain on these two goldens, even a perfect one. Its
value is **robustness**: clips where today's ROI or lock lands on the wrong
body part, shots without a user ROI, and multi-person scenes. The gate
therefore changes to:

1. **Box quality vs the reference labels:**
   `vlm_score.py <clip>.vlm.json <golden>/vlm_oracle.json`.
   - Contact hit rate ≥ 80 % (the probe's contact centre lies in the
     labelled box; refusals and missing boxes count as misses).
   - Contact-on-exclude ≤ 5 %.
   - Refusal rate < 20 %.
2. **End to end on clips where the baseline fails.**
   - These are the Owner's new clips. At least one has to show the ROI or
     lock on the wrong body part, which the labels identify.
   - The requirement is a gain there, with no regression on the two goldens.
   - This needs references for the new clips. A hand-corrected script is
     best, because FunGen output is only a ~0.45 proxy.

**Where V0 runs:** on the Owner's PC. The cloud session cannot download
weights, because its network policy blocks huggingface.co. The Owner runs
one command and sends back the `.vlm.json` (text only, no frames).

### V1 — VLM proposals as marks (Cursor, after V0 passes)

- Import `.vlm.json` as SceneMap marks:
  - `contact` becomes `source`.
  - `thigh` and `hand_*` become `exclude`.
  - The body regions become `region`.
  - All of them get `author:"vlm"` and `reviewed:false`.
  - The time scope is the current shot (§ 6 decision 4).
- The map view shows them in a distinct style, with accept / reject per mark
  and per shot.
- Generate honours only accepted marks. Unreviewed VLM marks count only if
  the user ticks "use VLM suggestions" (off by default).
- Claude measures the accepted-marks run on both goldens.

### V2 — dataset (ChatGPT P5c + Cursor)

- SceneMap M5 export writes the reviewed boxes as YOLO labels, using the
  existing `bootstrap_yolo_dataset.py` layout.
- **Propagation:** a reviewed keyframe box is carried along its shot with
  CSRT, which gives labels on the in-between frames. Those labels get
  `author:"vlm+track"` and are kept only while CSRT confidence is high and
  the box stays within one grid cell of the chosen rhythm cell. This is the
  M5 agreement rule.
- Negatives: exclude boxes go to hard-negative mining.

### V3 — our own detector (Owner GPU)

- `train_yolo_model.py` → `export_yolo_onnx.py` → `ai_roi.py` contract. A
  nano or small detector trains fine on 16 GB.
- **Gate (SceneMap L2):**
  - Box accuracy on held-out, reviewed frames.
  - End-to-end windowed r on **≥ 4–5 clips**, no clip regressing, and
    orientation judged against the YOLO reference.
  - It stays opt-in until the Owner flips it.
- Then per frame:
  - A detector box seeds and anchors the rhythm-grid search, instead of the
    ROI centre.
  - It excludes hand and thigh cells.
  - It re-seeds CSRT after drift.

### V4 — later

- The learned cell scorer (L3) uses detector labels as features.
- Perception v1 fusion (L4).

---

## 5. Ownership

| Who | Does |
|---|---|
| **Claude** | V0 tool + all measurements (V0 gate, V1 run, V3 gate); reviews the propagation rule |
| **Cursor** | V1 import + map-view review UI + "use VLM suggestions" switch; V2 propagation in the export |
| **ChatGPT** | proposed: P5c reviewed-label export (the V2 sink) — answers the E-ask on the board |
| **Owner** | runs V0/V3 on the 16 GB GPU; **4–5 more clips with references** (still the biggest lever); licence decision below |

---

## 6. Open Owner decisions

1. **Detector framework licence.**
   - `ultralytics` (used by `train_yolo_model.py`) is AGPL-3.0, and
     Ultralytics applies it to trained weights too. For a paid product that
     means either an Ultralytics licence or an Apache-2.0 detector (for
     example YOLOX or D-FINE / RT-DETR in a permissive implementation).
   - This only has to be decided before V3 ships, not for V0–V2.
2. **Runtime default:** Ollama or LM Studio. Proposal: document both, and
   default the probe to Ollama.
3. **Optional:** allow `huggingface.co` in the cloud environment's network
   policy, so that Claude can run the 3B model on CPU for plumbing tests.
   This is not needed for V0 itself.

---

## 7. Risks

- **Refusal or poor boxes on explicit frames.** V0 measures this first.
- **Hallucinated boxes.** Mandatory review, plus the agreement rule for
  propagated labels.
- **Over-fitting to two goldens.** The ≥ 4–5 clip gate applies to every
  default flip.
- **Runtime cost.** A keyframe every 5 s on a 280 s clip is about 60 VLM
  calls. At roughly 1–3 s each on a 16 GB card, that is minutes, once per
  clip, offline.
