# AI training system — guide (what / where / how)

As of September 2026 · SamNPlayer **0.5.5+**

This document explains the **regions model** (YOLO → ONNX), not device **Training** in the Training tab (stop-start/plateau).

## Quick: what AI may do — and what not

| May | May not |
|-----|---------|
| Suggest region(s) | Write Funscripts |
| Set box in the GUI | Replace tracking |
| Learn locally on your machine | Cloud / telemetry / bundled model |

Classic tracking (CSRT / flow / …) always writes the `.funscript`.
See also `docs/AI_ADAPTER.md` and `docs/FUNSCRIPT_ALGOS.md`.

## Requirements

1. **Python 3** (same interpreter SamNPlayer finds — Windows: `py`/`python`)
2. **Base CV:** `opencv-contrib-python` (not `opencv-python` alone) — for bootstrap (video sampling). OpenCV 5: CSRT often missing → automatic KCF fallback (#95).
3. **Training:** `ultralytics` + `onnx` (pulls PyTorch)
4. **Inference (Generate tab):** `onnxruntime`
5. Optional: **ffmpeg** for still conversion / audio sidecar / player proxy

### Installation (GUI, recommended)

In the **AI training** tab → **Install dependencies**.

This writes the embedded `requirements-ai-train.txt` and runs
`python -m pip install -r …` — works without a source tree
(release `.exe`).

### Installation (manual)

```bash
# Inference
pip install -r generator/requirements-ai.txt

# Training (+ PyTorch)
pip install -r generator/requirements-ai-train.txt

# Windows without NVIDIA (optional):
pip install torch-directml
```

From the binary: Settings / or later “Export requirements” —
`WriteAIRequirementFiles(target folder)`.

## Step-by-step (GUI)

1. **AI training** tab
2. **Choose video** (or **image** for still annotation)
3. For black intro: seek time → load frame
4. Draw up to **4 boxes**, name classes (`hand`, `breast`, …)
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
| 3D / depth / pose | **not** — until golden clips show Funscript quality gain |

## Tips for good models

- Many short clips instead of one long one
- Correct boxes in review (bootstrap keeps fixed w/h)
- Optional **box scale** 1.1–1.2 for padding around the mark
- At least one val example (still path switches automatically)
- GPU strongly recommended; CPU only for smoke tests (few epochs)
- After training: Generate tab → AI detection; in Settings set **Preferred classes** for multi-class models (e.g. `hand,breast`)
- 3D / depth / pose: **not** — until golden clips show Funscript gain

## Troubleshooting

| Symptom | Cause / fix |
|---------|-------------|
| “ultralytics missing” | Install dependencies |
| Bootstrap fails on OpenCV 5 | `opencv-contrib-python`; check KCF fallback |
| Training: empty val | at least 2 stills or video bootstrap |
| AI checkbox gray | no `.onnx` at model path |
| Poor suggestions | more/corrected samples; preferred classes; not “bigger YOLO” |
| Preferred class ignored | `classes.json` beside `.onnx` missing (retrain or copy) |

## Related files

- `cmd/gui-wails/frontend/src/roi_training.js` — GUI
- `cmd/gui-wails/app_roi_training.go` — Wails API
- `generator/roi_training.go`, `bootstrap_yolo_dataset.py`, `train_yolo_model.py`, `ai_roi.py`
- `docs/AI_ADAPTER.md` — architecture principle
