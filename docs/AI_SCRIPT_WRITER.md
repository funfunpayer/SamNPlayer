# AI script writer — Owner path (opt-in)

**Opened:** 26 Sep 2026 (Owner: *create a way that AI can also write Funscripts*).  
**Default unchanged:** Everyday Create stays **CSRT tip → classical posttrack → `.samn`**.

This document is the product + engineering contract for an **explicit second
path**. It does **not** replace `docs/AI_ADAPTER.md` region/profile/quality
helpers; those stay propose-only.

Related: `AI_ADAPTER.md`, `KI_TRAINING.md`, `EVERYDAY_GENERATE.md`,
`PRODUCTION_ROADMAP.md` § G3, `ENGINEERING_STANCE.md`.

---

## Product rule

| Path | Who writes 0–100 | When |
|------|------------------|------|
| **Everyday (default)** | Classical CSRT + posttrack | Always, unless user opts into AI draft |
| **AI draft (opt-in)** | Local learned / vision model → draft actions | Explicit Advanced control + model present |
| **Never** | Cloud LLM inventing per-frame positions from chat tokens | Out of product |

AI draft must:

1. Be **off by default** (no silent Everyday change).
2. Require an **installed local model** (no bundled weights, no auto-download).
3. Run the draft through **Quality Doctor** (and optional audio warn) before
   the user can keep it.
4. Require an explicit **Keep draft / Apply** in Create (same spirit as
   Suggest profile → Apply and Apply target).
5. Still write **`.samn` first**; community `.funscript` remains export.

---

## Why not “ChatGPT writes the curve”?

Per-frame stroke needs **spatial identity** (tip box → tracker → positions).
A free-form LLM token stream does not. Local LLMs (Colibri) stay useful for
**profile / quality prose** (`AI_ADAPTER.md` § 2–3), not as the stroke writer.

The AI script path is therefore:

```text
  video (+ optional tip ROI / scene marks)
       → local draft model (ONNX or pure-Go, TBD per stage)
       → actions[] draft
       → classical Quality Doctor (+ audio warn)
       → user Keep / Discard
       → .samn (+ .funscript export)
```

---

## Stages

| Stage | Deliverable | Exit gate |
|-------|-------------|-----------|
| **S0** | Plan + Go `aiscript` + GUI stub | **Done** (#261) — Everyday bit-identical |
| **S1** | Training export: classical good runs → imitation samples | **Partial** — `ExportImitationSample` + Create **Export classical run** |
| **S2** | First local draft model (train offline; infer in-app) | Blind QD ≥ classical floor *or* Owner “experimental” |
| **S3** | Create Advanced: **AI draft script** → preview → Keep | Apply required; English copy only |
| **S4** | Optional: draft seeded from tip ROI + scene-map marks | Goldens announced; no Everyday default |

**Out until Owner re-opens:** cloud APIs, Diffusers, replacing CSRT default,
audio inventing positions, auto-Keep without QD.

---

## Package contract (`generator/aiscript`)

```text
Status() → { available bool, reason string, modelPath string }
Draft(req) → { actions []Action, meta, warnings } | error
```

- `available=false` when no model / deps → GUI control stays disabled with
  English reason (same pattern as `CheckAIRoiAvailable`).
- `Draft` never overwrites Everyday Generate; Create calls it only from the
  AI-draft control.
- Errors are English, closed (no silent classical fallback that pretends to
  be AI).

---

## GUI (Create)

Under **Advanced settings** (not Everyday step chrome):

- Checkbox / button: **AI draft script (experimental)**  
- Enabled only when `AIScriptWriterAvailable()` is true.  
- Help: points here; states CSRT remains the normal Create path.  
- After draft: show Quality Doctor box + **Keep draft** / **Discard**.

Settings may later add model path (like AI ROI) — S3+.

---

## Relationship to existing AI pieces

| Piece | Role vs AI script writer |
|-------|--------------------------|
| ONNX ROI / semantic target | Still **proposal** for tip box; can **seed** S4 draft |
| Go `profilemodel` | Style suggest only; may condition S2 features |
| Colibri quality/profile | Opinion / prose only — not stroke writer |
| SceneMap learning export | Fuel for S1 imitation labels (opt-in collect) |

---

## Agent notes

- One lane: **AIScriptWriter** — do not mix with Look CSS or SceneMap P-next.
- No default-on without Owner gate after S2 metrics.
- English GUI/docs; Owner chat German OK.
