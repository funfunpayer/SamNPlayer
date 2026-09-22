# Classical Generate heuristics (G1 package)

Design for the **classical-only** heuristics package on the production
roadmap (`docs/PRODUCTION_ROADMAP.md` phase **G1**). Not AI. Goal: one
coherent set of rules that turn CSRT observations into better funscripts
for Sam Neo 2 — measurable, tunable, and safe to teach AI later.

Related: `docs/FUNSCRIPT_ALGOS.md`, `docs/AUDIO_WORKFLOW.md`,
`docs/SIGNAL_VS_FIDELITY.md`, `docs/ENGINE.md`, `docs/SELF_BUILD.md`.

---

## Why this exists

0.5.11 locked **one CSRT path**. Next quality wins are not “another
tracker in the dropdown,” but **stable post-track decisions**: axis,
peaks, smooth, RDP, speed, weak-ROI hints, audio plausibility.

We learn for AI by first producing **good classical scripts**. AI must
not write the 0–100 curve (`docs/AI_ADAPTER.md`).

---

## Workflow (ship this order)

```text
  1. ROI
        manual mark (product default)
        optional: classic/AI find-region → VerifyROI warn-only
  2. Track
        CSRT only (Go trackcv when linked; else Python CSRT)
  3. Heuristics package (this doc) — posttrack + peak/RDP/speed/axis
  4. Quality Doctor          → Signal Quality (label it)
  5. Audio tempo check       → warn only (never invent positions)
  6. Write .funscript + .samn (+ optional .samn quality sidecar fields)
```

Do **not** put audio or AI before a tracked curve exists (except an
optional tempo *hint* that only biases peak distance — G1.3, gated).

---

## Package contents (v1)

| Knob | Role | Product rule |
|------|------|--------------|
| Axis (`x`/`y`/auto) | Pick dominant motion | Auto OK; surface choice in log; allow force |
| Smooth / Savgol | Kill tracker jitter | Profile-driven; no silent over-smooth |
| Peak distance | Stroke tempo | Device-safe min gap (~100 ms); audio may *hint* later |
| RDP | Keyframe density | Keep feel; avoid “stair” from over-reduce |
| Speed cap | Device compat | Always on for Neo 2 limits |
| Detrend / bandpass | Autotune profile | Opt-in profile, not silent default for Hub |
| Weak-ROI / tiny span | Hint, not rewrite | “Check ROI / axis” — never invent motion |
| Tracking gaps | Metadata + warning | Visible in Quality Doctor / report |
| Audio Hz vs script Hz | Plausibility | Post-hoc; fail open; see `AUDIO_WORKFLOW.md` |
| **Stroke preview (Stage A)** | Sparse extrema + cut/pan flags before/beside track | Timing/flags only — `generator/strokepreview`; never invent curve; audio gate when `suggest_audio_check` |

Implementation homes today: `generator/posttrack`,
`generate_funscript.py`, `generator/audiocheck.go`, Quality Doctor,
`generator/strokepreview` (Stage A probe).
G1 work is **unify + document + tune as one package**, not three
scattered knobs.

---

## Signal Quality vs Motion Fidelity

| Stage | Reports |
|-------|---------|
| Quality Doctor / Script Doctor | **Signal Quality** only |
| FunGen / phase / golden correlation | **Motion Fidelity** only |
| Heuristics changes | Must not mix the two into one marketed score |

A smoother script (higher Signal Quality) that loses phase is a
**regression**. Gate with goldens (`docs/GOLDEN_CLIPS.md`).

---

## Audio (classical)

| Mode | Status | Rule |
|------|--------|------|
| Post-check script Hz vs audio energy Hz | Shipped; keep default-on when ffmpeg present | Warn only |
| Pre-pass tempo hint → bias peak distance | G1.3 optional | Only after goldens show help > harm |
| “No motion → write from audio” | **Rejected** | Audio is not a position curve |

---

## Exit criteria for “heuristics v1 done”

1. One English GUI/docs explanation of the workflow above (Generate help).
2. Same fixed-ROI golden set: Motion Fidelity not worse; Signal Quality
   equal or better on average.
3. Audio check never changes actions; weak-track + clear audio → English
   hint only.
4. No new GUI tracker backends; no AI auto-commit of ROI2.

---

## Out of scope here (later phases)

| Topic | Phase |
|-------|-------|
| Windows in-binary CSRT | G0 |
| Neo 2 raw-value profile / Intent→device | G2 |
| AI ROI / train / fusion | G3 |
| License enforcement | G4 |

---

## Implementation note

Prefer extending existing posttrack + Quality Doctor + audiocheck over
new packages. Every behavior change needs a regression that fails without
it (`CONTRIBUTING.md`). Clip tests beat vibes (`docs/SELF_BUILD.md`).
