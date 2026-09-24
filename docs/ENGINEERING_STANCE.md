# Engineering stance (Bau / Planung lock — 24 Sep 2026)

Cursor coordinating doc. Product UI stays English; this file is the
**shared technical stance** so agents do not re-litigate stack choices or
steal each other’s lanes.

Related: `AGENT_COORD.md` (who owns what), `PRODUCTION_ROADMAP.md` (G0–G4),
`SCENE_MAP_PLAN.md` (Claude plan #243 — lands with that PR),
`AI_ADAPTER.md` / `KI_TRAINING.md` (AI suggest-only),
`GENERATE_HEURISTICS.md` (G1 knob map).

---

## Product locks (do not confuse)

| Topic | Lock |
|-------|------|
| **Everyday Generate** | Stroke tip-CSRT (1-Zone) + Contact vib. No 4-Zone in Generate GUI. |
| **Detrend** | On by default for stroke profiles (#230). **Not** the same as Speed Cap. |
| **Speed Cap** | Autotune forces 400 if unset; Normal/Soft stay uncapped (`MaxSpeed=0`) until Owner decides a Neo-2-safe default. G1.1 gate. |
| **Tf / Tj** | Profile / feel + CLI / Advanced legacy distance path — **not** Everyday Create. |
| **Rhythm grid** | Opt-in Advanced (#233/#236, v0.5.30). Default-on only after Owner ≥4–5 clips (r up + no orientation regression where FunGen refs agree). |
| **AI** | Proposes only. Classical CSRT + posttrack **write** the script. No LLM / vision model as a curve writer. |
| **F-003 tier 2** | Do not implement lag tie-breaking; tier-1 `AliasingRisk` is enough for now. |

---

## Stack stance (essay mapping — adopt ideas, not blog deps)

External essays (Go libs, Go+AI, FFmpeg+Go, RAG/FT, optimization) were
reviewed as **input**, not a shopping list.

| Keep / prefer | Why |
|---------------|-----|
| **`videox` + os/exec FFmpeg** | Already the decode/proxy/tools path; no FFmpeg-as-library binding. |
| **stdlib-first Go** | Prefer `slog`, stdlib HTTP/JSON, small pure-Go packages over framework sprawl. |
| **Py train / Go infer** | Heavy training (YOLO/ultralytics) stays Python; product inference prefers ONNX / pure-Go (`profilemodel`) when possible. |
| **Local-only AI** | No bundled weights, no auto-download, no telemetry, no cloud call without opt-in. |

| Reject / defer | Why |
|----------------|-----|
| LLM / chat model writing positions | Breaks per-frame spatial identity; violates AI Adapter principle. |
| RAG-first Generate | Wrong layer — Generate needs measured tracks, not retrieval. |
| PGO / micro-opt as a priority | Measure clip quality first; no speculative optimizer work. |
| Blog dependency stacks (extra ORMs, LLM SDKs, Diffusers, DeepSpeed, …) | Unite.ai filter already in Multi-track Fahrplan; same filter here. |
| Replacing `videox` with a third-party “Go FFmpeg toolkit” | Architecture notes OK; dependency swap needs a measured win. |

---

## Open lanes (24 Sep) — merge order

```text
  Parallel OK (different layers):
    • Claude #243  — SCENE_MAP_PLAN.md (docs only)     → merge when green
    • Manus #244   — profilemodel + GUI Suggest→Apply  → merge when green
        (CI green; no trackcv overlap; only AGENT_COORD may conflict)

  After both on main:
    Cursor builds SceneMap P1 → P5 per SCENE_MAP_PLAN.md
    (Claude reviews engine + goldens; Owner answers § 6)
```

**#244 may land on main now.** It is already GUI-wired (Remember scene +
style, Train Go profile model, Suggest→Apply). It does **not** write
curves and does not touch Everyday defaults.

**Do not “code-merge” Claude’s plan into Manus’s PR.** They stack later:

| Layer | Who | What |
|-------|-----|------|
| Style / profile suggestion | #244 `profilemodel` | eight motion features → Normal/Soft/Autotune guess |
| Where the stroke is | SceneMap P1–P3 | rhythm heatmap + exclude/source marks |
| Who / labels for YOLO | SceneMap P4–P5 → L2 | `.samn` marks + export into `KI_TRAINING` dataset |
| How / cell scorer | SceneMap L3 | engine-trace JSONL — after enough clips |

So: merge Manus for the profile helper; keep Claude’s plan as the map /
learning spine; Cursor implements SceneMap on top of both. No rewrite of
#244 into a heatmap, and no SceneMap code inside #244.

Do **not** start SceneMap **code** until #243 is on `main`. G1.1 tune
stays blocked on Owner (speed-cap rule + rhythm clips).

### SceneMap (Cursor after #243)

Canonical plan: `docs/SCENE_MAP_PLAN.md`. Short order:

| Phase | What | Gate |
|-------|------|------|
| **P1** | Split score/choose; `SceneMap` + quick scan; Wails bind | bit-identical curve |
| **P2** | Advanced map view + marks (exclude / source / region) | Owner can paint + persist |
| **P3** | Engine honours marks; masks on Go when RhythmGrid on | goldens, announced |
| **P4** | `metadata.scene_map` in `.samn` | round-trip; export unchanged |
| **P5** | Local training export + privacy off-by-default | delete works |
| **P6+** | L1–L4 learning | per-stage suggest-only gates |

### AIScript (#244 — ChatGPT / Manus)

First measurable G3 helper slice: pure-Go `generator/profilemodel`
(centroids + spread, reject unknown/ambiguous), Create stores scene+Style,
AI Training trains JSON model, Suggest → Apply only. Falls back to
nearest-scene / Colibri. Fits `AI_ADAPTER.md` order (region → profile →
quality). Field data still needed before threshold tuning.

---

## Owner gates (block defaults)

1. **Speed-cap product rule** — Normal uncapped vs Neo-2-safe default + opt-out.
2. **Rhythm-grid default** — ≥4–5 real clips; r and orientation gates.
3. **SceneMap § 6** — quick-scan trigger; learning-data default; auto-label
   review; mark time-scope default.
4. **Any learned stage (L2/L3) or fusion default** — golden win required.

---

## Agent roles (Wagenheber)

| Who | Owns now |
|-----|----------|
| **Cursor** | Coord / stance / board; after #243 → SceneMap P1+; releases |
| **Claude** | #243 plan; reviews SceneMap engine + measures goldens |
| **ChatGPT / Manus** | #244 AIScript foundation; steward / bugfix |
| **Owner** | Gates above; merge order; local clip smoke |

One theme per agent. Base on current `main`. No silent default changes
(`AGENT_COORD` rule 4).

---

## GoCV / MOSSE / YOLO+ByteTrack essay (24 Sep)

Mapped against what we already ship. **Do not adopt blindly.**

| Essay idea | Our status |
|------------|------------|
| GoCV (`gocv.io`) as OpenCV wrapper | **Rejected as dependency** — `trackcv` is a thin CGO wrapper (`cv.cpp`) on purpose; see package comment (contrib/patent/compile issues with full gocv). |
| TrackerMOSSE / KCF | **Measured and rejected** for hard tip ROIs (`NEXT.md`); product Stroke = **CSRT**. |
| YOLO + ByteTrack IDs | **Already planned** as opt-in proposal/ID layer (MT-Seed done; MT-ID Owner gate) — **not** Stroke writer. |
| V4L2 / go4vl webcam capture | N/A for Generate-from-file product path. |
| WebRTC / Pion streaming | Out of Generate spine; parked with platforms. |

Keep CSRT + rhythm grid + SceneMap path. Borrow MOT *ideas* only via the
Multi-track Fahrplan — not a second tracker stack.
