# Local AI as an adapter on top of the existing system

This file records how a local AI enters the generator without replacing the
measurable, non-AI system — a request from the chat on September 14, 2026:
"the generator should run with local AI, and for that should be able to use
the results/tools of the non-AI system; they should work together."

## Principle

The AI proposes, the existing system measures. This is not a new rule but
the line already followed in `docs/TEAM_STAND.md` (section 8) and
`quality_model.py`: a learned model is only used once it outperforms the
fixed rules in cross-validation, and a human confirms the result. The same
applies to the AI extension:

- The AI produces a **proposal** (region, profile, judgment).
- The proposal runs through **the same** classical pipeline (CSRT/flow
  tracking, Quality Doctor, mapper) as a manually marked one — there is no
  second, AI-only output path that produces a `.funscript` on its own.
- If the AI fails (no model, no detection, server unreachable), the system
  falls back to the non-AI method, the same way `track_by_scenes()` already
  catches any `roi_finder` failure and reuses the previous region.
- Both engines are **optional** and run **locally** — no model in the
  repository, no automatic download, no telemetry. Anyone who does not
  install or start them notices nothing of their existence.

## Two engines, two jobs

Deliberately **not** routed through a gateway (Bifrost/LM Studio), but two
independent, lightweight local engines — each for the job it is built for:

| Engine | Job | Why this one |
|---|---|---|
| **ONNX Runtime** | Vision: propose a region (later: propose a profile from the motion signature) | Small (a few MB of model plus runtime), CPU/GPU, no Python requirement at inference time — fits the stated goal of eventually removing Python from the .exe (`HANDOFF.md`, "Project status and next steps") |
| **Colibri** ([JustVugg/colibri](https://github.com/JustVugg/colibri)) | Large local language models for judgments with a reason attached (quality, profile choice in prose) | Pure C, no CUDA/PyTorch needed, runs on ordinary hardware, `coli serve` speaks the OpenAI `/v1/chat/completions` API — a plain HTTP client is enough to connect it, no SDK required |

Both are swappable: the backend register (`backends.py`) and the
`roi_finder` injection point in `track_by_scenes()` exist for exactly this
purpose — register a function with a fixed contract, nothing in the
pipeline changes.

## Where "ONNX like FunGen 2" comes from — and what was NOT taken from it

FunGen 2 is a closed binary (FunGen 1 is PolyForm Strict), see
`HANDOFF.md`, "Third-party code policy". No source code was read. Publicly
described (fungen.app, GitHub README): FunGen detects objects with a YOLO
model and tracks from there. Only the **idea** — "detection proposes a
region, tracking measures from it" — was carried over. The model contract
in `ai_roi.py` (input `1x3xHxW`, output `(N,6)` normalized) is an
independently designed interface, not a reimplementation of a specific
export format.

## Order (agreed with the user: region → profile → quality)

### 1. Propose a region — **implemented**

- `generator/ai_roi.py`: `find_roi(video, start_frame, end_frame)` with the
  same contract as `auto_roi.find_roi` — both are interchangeable
  `roi_finder` implementations.
- CLI: `--per-scene-roi --roi-finder ai --ai-model-path model.onnx`
  (default stays `auto`, so behavior is unchanged without the new option).
- `decode_detections()` and `select_best_box()` are pure functions and
  testable without a model or onnxruntime (`generator/ai_roi_test.py`).
- `onnxruntime` lives in `generator/requirements-ai.txt`, NOT in
  `requirements.txt`.
- **Still open:** no bundled/recommended ONNX model, no GUI wiring (on the
  Go side this would be a copy of `FindROIWithProgress` in
  `generator/generator.go` targeting `ai_roi.py` instead of `auto_roi.py` —
  the `ROI x y w h` stdout contract is already kept identical).

### 2. Propose a profile (standard/tf/tj/…) — **open**

Builds on the motion signature, which exists but is not yet wired up
(`HANDOFF.md`: naming in the UI and reusing saved parameters is not
connected). Two possible directions, not yet decided:

- Purely from the eight signature metrics via a small local model (would
  stay on ONNX, the same engine as step 1).
- Or as a text justification via Colibri ("this scene resembles the scene
  named 'tj', because …") — this would restore, in a different form, the
  explainability that `quality_model.py` deliberately buys by avoiding
  neural networks: not readable weights, but a model that can justify its
  decision in prose.

### 3. Quality judgment — **open**

Extends, does not replace, `quality_model.py`. Same safeguard already in
place there: an AI judgment is only adopted if it beats leave-one-out
cross-validation against both the fixed rules AND the existing learned
model (`MIN_SAMPLES`/`MIN_PER_CLASS` as the template). Colibri's
`/v1/chat/completions` could additionally supply a plain-text justification
to sit next to the number in the measurement report.

## Not part of this change

- No action detector — unchanged principle from `docs/TEAM_STAND.md`.
- No requirement to install AI — both engines remain optional additions,
  like the Intiface path for the device (`device/intiface.go`): a second
  option, not a replacement for the existing one.
- No cloud connection, no gateway — both engines run exclusively on
  `127.0.0.1`.
