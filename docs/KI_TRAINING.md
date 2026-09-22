# AI training system — guide (what / where / how)

As of September 2026 · SamNPlayer **0.5.5+**

This document explains the **regions model** (YOLO → ONNX), not device **Training** in the Training tab (stop-start/plateau).

## Quick: what AI may do — and what not

| May | May not |
|-----|---------|
| Suggest region(s) | Write Funscripts |
| Set box in the GUI | Replace tracking |
| Learn locally on your machine | Cloud / telemetry / bundled model |

Classic tracking (CSRT / flow / …) writes `.samn` (native) plus a
community `.funscript` export. AI still never writes either file by itself.
See also `docs/SAMN_FORMAT.md`, `docs/AI_ADAPTER.md` and `docs/FUNSCRIPT_ALGOS.md`.

## Requirements

1. **Python 3** (same interpreter SamNPlayer finds — Windows: `py`/`python`)
2. **Base CV:** `opencv-contrib-python` (not `opencv-python` alone) — for bootstrap (video sampling). OpenCV 5: CSRT often missing → automatic KCF fallback (#95).
3. **Training:** `ultralytics` + `onnx` (pulls PyTorch)
4. **Inference (Generate tab):** `onnxruntime`
5. Optional: **ffmpeg** for still conversion / audio sidecar / player proxy

### Installation (GUI, recommended)

In the **AI training** tab → **Install AI train deps**.

This writes the embedded `requirements-ai-train.txt`, runs
`python -m pip install -r …`, then **restores `opencv-contrib-python`**
(ultralytics depends on plain `opencv-python`, which removes CSRT needed
for **Use for training** — Issues #94/#119). Works without a source tree
(release `.exe`).

On Windows, prefer a real install from [python.org](https://python.org)
(e.g. `%LOCALAPPDATA%\Programs\Python\…`). The Microsoft Store / WindowsApps
stub on PATH is demoted so pip hints target the real interpreter.

### Installation (manual)

```bash
# Inference
pip install -r generator/requirements-ai.txt

# Training (+ PyTorch) — then restore contrib (ultralytics may replace it)
pip install -r generator/requirements-ai-train.txt
pip uninstall -y opencv-python opencv-python-headless
pip install opencv-contrib-python

# Windows without NVIDIA (optional):
pip install torch-directml
```

**0.5.9 / 0.5.10 workaround** (before the restore landed in 0.5.11): after
any ultralytics install, run the uninstall + `opencv-contrib-python` lines
above in the **same** Python SamNPlayer reports in the bootstrap error.

From the binary: Settings / or later “Export requirements” —
`WriteAIRequirementFiles(target folder)`.

## Step-by-step (GUI)

1. **AI training** tab
2. **Choose video** (or **image** for still annotation)
3. For black intro: seek time → load frame
4. Draw up to **9 boxes**, name classes with the English taxonomy
   (`face`, `mouth`, `breasts`, `nipples`, `hand_1`, `hand_2`, `penis`,
   `glans`, `vagina` — see `docs/BODY_REGIONS.md`). Legacy German labels
   (`brust`, `eichel`, …) still normalize.
5. **Use for training** → CSRT/KCF tracks and writes YOLO samples
6. **Review:** discard bad samples, **correct** boxes (important — bootstrap box size is fixed; tracker may drift)
7. **Start training** (epochs, device Auto/CUDA/DirectML/CPU)
8. Result: `roi_detector.onnx` under model path (Settings)
9. **Generate** tab → enable **AI detection (ONNX)** → find region → verify box → generate Funscript

## Folders

| What | Typical path |
|------|----------------|
| Dataset | `%AppData%/SamNPlayer/roi_training_dataset` or `~/.config/SamNPlayer/…` |
| Model | `%LOCALAPPDATA%/SamNPlayer/models/roi_detector.onnx` |
| Checkpoints | `<dataset>/runs/samnplayer_roi/weights/best.pt` |
| Audio (optional) | `<dataset>/audio/*.wav` — not used in training yet |

## CLI (source tree)

```bash
python generator/bootstrap_yolo_dataset.py \
  --video clip.mp4 --roi x,y,w,h --class-name hand \
  --output-dir ~/.config/SamNPlayer/roi_training_dataset

python generator/train_yolo_model.py \
  --dataset-dir ~/.config/SamNPlayer/roi_training_dataset \
  --output ~/.config/SamNPlayer/models/roi_detector.onnx \
  --epochs 100 --device auto
```

## Vision: what we use / what not

| Technique | Status |
|-----------|--------|
| YOLOv8n → ONNX ROI suggestion | **Product path** |
| Classic CSRT/KCF/flow | **writes Funscript** |
| Two-ROI AI (`find_two_rois`) | Code present, rarely measured in practice |
| Audio sidecar | stored only |
| 3D / depth / pose | **experimental** — classical depth proxy + optional ONNX stub in `support_signals.py`; opt-in ROI soft-rank only (`SAMNPLAYER_DEPTH_RANK=1`); no Funscript writer; bake-off required before default |

## Tips for good models

- Many short clips instead of one long one
- Correct boxes in review (bootstrap keeps fixed w/h)
- Optional **box scale** 1.1–1.2 for padding around the mark
- At least one val example (still path switches automatically)
- GPU strongly recommended; CPU only for smoke tests (few epochs)
- After training: Generate tab → AI detection; in Settings set **Preferred classes** for multi-class models (e.g. `hand,breast`)
- 3D / depth / pose: experimental scaffold only — see `docs/DEPTH_POSE.md`; golden-clip win required before default

## Already transfer learning — how to go faster (22 Sep)

Training is **not** from scratch. `train_yolo_model.py` defaults to
`--base-model yolov8n.pt` (COCO-pretrained Ultralytics weights), then
fine-tunes on **your** body-part boxes → ONNX for Generate proposals.

| Speed / quality lever | Status | Notes |
|-----------------------|--------|-------|
| COCO `yolov8n.pt` fine-tune | **Already default** | Small + fast; keep unless data is huge |
| GPU (`--device auto`) | Wired | CUDA → MPS → DirectML → CPU |
| More corrected samples | Highest leverage | Review/correct beats bigger YOLO |
| `yolov8s` / YOLO26 later | Opt-in `--base-model` | Only after n plateaus on your classes |
| External body/hand ONNX as **proposal helper** | Idea (extra) | Suggest boxes only; never write Stroke; measure vs CSRT auto-ROI first |
| ByteTrack/BoT-SORT ID layer | Fahrplan MT-Speed/ID | After classical MT-Seed; not Stroke writer |
| Freeze backbone / fewer epochs | Easy CLI knobs later | Good once sample count is stable |

**Product rule stays:** AI may **suggest** Tip/body-part regions; CSRT
(Everyday, no AI required) still writes the curve. Do not swap Everyday
to a third-party model just to “train faster.”

## Troubleshooting

| Symptom | Cause / fix |
|---------|-------------|
| “ultralytics missing” | Install AI train deps |
| Bootstrap fails: OpenCV without tracker / WindowsApps pip hint | Prefer real Python; Install AI train deps (restores contrib) or manual `pip uninstall opencv-python … && pip install opencv-contrib-python` (#119) |
| Bootstrap fails on OpenCV 5 | `opencv-contrib-python`; check KCF fallback |
| Training: empty val | at least 2 stills or video bootstrap |
| AI checkbox gray | no `.onnx` at model path |
| Poor suggestions | more/corrected samples; preferred classes; not “bigger YOLO” |
| Preferred class ignored | `classes.json` beside `.onnx` missing (retrain or copy) |

## Related files

- `cmd/gui-wails/frontend/src/roi_training.js` — GUI
- `cmd/gui-wails/app_roi_training.go` — Wails API
- `generator/roi_training.go`, `bootstrap_yolo_dataset.py`, `train_yolo_model.py`, `ai_roi.py`, `support_signals.py`
- `docs/AI_ADAPTER.md` — architecture principle
