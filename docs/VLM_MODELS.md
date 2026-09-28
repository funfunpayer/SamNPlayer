# Local teachers and models — what we can use, how to plug it in

Companion to `docs/VLM_TEACHER_PLAN.md`. Everything here runs **locally**:
- No frame leaves the machine.
- No model ships in the repo.
- Nothing downloads by itself.

The Owner installs what they want to try. Researched 27 Sep 2026; check
licences again at download time.

## How a teacher reaches the engine

```
video ─► teacher(s) ─► contact_points.py ─► <clip>.contact.json
                                              │
                    ┌─────────────────────────┴─────────────────────────┐
                    │ CLI: generate --rhythm-grid --contact-points …    │
                    │ GUI: Create → Advanced → Rhythm-robust →          │
                    │      Use contact points → Choose JSON             │
                    └─────────────────────────┬─────────────────────────┘
                                              │
        rhythm grid searches around the contact point only where the CSRT
        box is > 3 cells away (its own search radius) — elsewhere unchanged
```

Build the JSON via CLI; the GUI only picks the path (handbook FAQ).

- `contact_points.py --nudenet --vlm a.vlm.json --vlm b.vlm.json` asks
  several teachers per video (the Owner's idea).
- Every point carries `agree`: how many distinct teachers put a point within
  1.5 cells and 1.5 s of it.
- `--contact-min-agree 2` makes the engine use consensus points only.
- The same consensus points are the training-data candidates for our own
  detector (reviewed first, SceneMap § 6 decision 3).

## Teachers

| Teacher | What it gives | Runtime / install | Licence | Status here |
|---|---|---|---|---|
| **NudeNet** 320n | Boxes per frame: exposed/covered breasts, genitalia, buttocks, face (male/female) … (18 classes). Fast (320×320). | `pip install nudenet --no-deps onnxruntime`. Model bundled in the package. | Package MIT. Weights are YOLOv8-trained: re-check before we *bundle* anything; a user install is fine. | **Measured.** Penis found in only 7–37 % of keyframes (often occluded), but precise. The penis → cleavage rule hits 64 % / 80 % / 44 %, correct 18/18, 8/8, 12/16. As contact points (gated): multi-person r 0.304 → **0.406**; goldens unchanged (`clip_ausschnitt` bit-identical, `clip_voll` same r). |
| **Qwen2.5-VL 7B** | Boxes via prompt, pixel coordinates. | Ollama `qwen2.5vl:7b`, LM Studio. About 6 GB at Q4. | Apache-2.0 (3B: research-only). | `vlm_probe.py`. Waiting for the Owner's GPU run. |
| **Qwen3-VL 4B / 8B** | Boxes via prompt, **0–1000** coordinates. Also points. | Ollama `qwen3-vl:4b`, `qwen3-vl:8b`. | Check at download. | Probe calibration detects 0–1000 automatically. |
| Qwen2.5-VL abliterated | Same model with refusals removed. Box quality on explicit frames unknown. | Ollama `huihui_ai/qwen2.5-vl-abliterated`. | Derivative of Qwen2.5-VL. | Try if the plain model refuses (the V0 refusal-rate gate). |
| **InternVL3.5 8B** | Strong grounding (RefCOCO ~90 %). Boxes as `<box>[[…]]</box>`, 0–1000. | LM Studio / lmdeploy / vLLM (OpenAI-compatible). | Check at download. | Needs a `<box>` parser in the probe before use. |
| **Moondream 3** | Purpose-built `detect` (boxes) / `point`. 2B active parameters. | Moondream Station (own local API, not OpenAI chat). | Check at download. | Needs a small probe backend for its `/detect` API. |
| Florence-2 / OWLv2 / Grounding DINO | Open-vocabulary boxes from text (MIT / Apache). | transformers / ONNX. | MIT / Apache-2.0. | Probably weak on explicit content. Useful for non-explicit classes (hands, thighs, faces). |
| Engine itself | CSRT box + rhythm-grid cell. | built in | — | Already a teacher: the M5 auto-label agreement rule. |

Every new teacher gets the same treatment:
1. Run it on the keyframes that have labels.
2. Score it with `vlm_score.py`, against `testdata/golden_clips/*/vlm_oracle.json`
   and `testdata/vlm_labels/multi_person_642s.json`.
3. Run it end to end through `contact_points.py` and Generate.
4. Only then does it join the consensus.

## Our own model (student)

- **RF-DETR** (Roboflow) or **D-FINE**. Code *and* pretrained weights are
  Apache-2.0.
  - Fine-tunes on COCO/YOLO-format data.
  - Exports a single ONNX file (`model.export(format="onnx")`) for ONNX
    Runtime, like `ai_roi.py`.
- `ultralytics` (today's `train_yolo_model.py`) is AGPL-3.0, including trained
  weights by Ultralytics' reading. It is fine for local experiments, but not
  what we ship. This answers VLM plan § 6.1.
- Data: consensus points and boxes from the teachers above, reviewed by the
  user, plus the SceneMap marks the user draws, through the P5c export.

Gate (SceneMap L2):
- Box quality on held-out reviewed frames.
- End-to-end r on ≥ 4–5 clips.

## The training loop, as built (V3)

1. **Teachers → points:**
   `contact_points.py --video V --nudenet [--vlm a.vlm.json …] [--onnx contact_detector.onnx]`
   writes `V.contact.json`.
2. **Points → candidates:**
   `samnplayer-cli import-contact-candidates V.samn --contact V.contact.json [--min-agree 2]`
   - Consensus points become `contact` region marks with `author:"auto"`,
     `reviewed:false`, at most one per 2 s.
   - A re-import replaces unconfirmed candidates and keeps confirmed ones.
   - Region marks have no effect on Generate.
3. **Candidates → confirmed:** the user confirms (`reviewed:true`) or deletes
   them in the scene map, and may draw their own `contact` region marks,
   which count as confirmed.
4. **Confirmed → export:** SceneMap learning export (P5c, opt-in) writes only
   confirmed marks to `<learning>/<clip>/reviewed_yolo/`.
5. **Export → dataset:**
   `contact_detector.py dataset --learning-dir <learning> --out ds [--classes contact]`
   - Merges clips and remaps the per-clip class ids.
   - Splits by clip once there are 3 or more clips, so validation never sees
     training clips.
6. **Dataset → model:**
   `contact_detector.py train --dataset ds --out contact_detector.onnx --size nano --device cuda`
   - Needs `pip install "rfdetr[train]"`.
   - Writes `contact_detector.onnx` plus the model card `contact_detector.json`.
7. **Model → teacher:** `contact_points.py --onnx contact_detector.onnx`.
   - Runs through onnxruntime on CPU or GPU with no PyTorch.
   - Uses the same decode as RF-DETR's own ONNX helper.
   - Once it wins on the gate below, it can be the only teacher: fast, local,
     no big models needed.

## Adding a model to the probe

1. It has to speak OpenAI `/v1/chat/completions` with `image_url` parts.
   Ollama, LM Studio, vLLM and lmdeploy all do.
2. Run `vlm_probe.py --calibrate --model <id>`. The calibration shows its
   coordinate convention: pixels, 0–1000 or 0–1.
3. Run `vlm_probe.py --video … --model <id> [--exemplar-json … --follow]`.
4. Run `vlm_score.py <clip>.vlm.json <labels>`.
5. Feed it to `contact_points.py --vlm <clip>.vlm.json` together with the
   other teachers.
