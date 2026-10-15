# SamNPlayer 2.0 Next — engine plan

**Status:** plan · moved here from `SamNPlayer-Next-Level` on 15 Oct 2026 (Owner: "all
content into the SamNPlayer repo").
Open Owner decisions: [`OPEN_DECISIONS.md`](OPEN_DECISIONS.md) · order of work:
[`ROADMAP.md`](ROADMAP.md) · specs: [`specs/`](specs/) · what the code does today:
[`inventory/`](inventory/) · the five planning documents from PR #510, unchanged:
[`sources/`](sources/).

Also imported from the Next-Level repo (ChatGPT, 13 Oct): the
[capability register](PRIORITIZED_CAPABILITY_REGISTER.md) (12 features, P0–P2),
the [five-engine architecture and GUI](FIVE_ENGINE_ARCHITECTURE_AND_GUI.md), and
the [engine / learning / GUI review](reviews/ENGINE_LEARNING_GUI_2026-10-13.md).
They add detail; where they disagree with this file, this file wins.

## 1. One program, free, financed by donations

There is only one SamNPlayer. It is free freeware with an EULA and voluntary
donations; there is no paid version, licence key, price or supporter tier
(Owner, 12 Oct — see `docs/EDITIONS.md`). **SamNPlayer 2.0 Next** is the next
major version of this same program. The internal Suite build (`-tags suite`)
stays as it is.

Everything in `sources/` about a paid "Studio" — prices, the >40 € supporter
right, licence pass, checkout, expiring previews, cutting AI or batch from Free
— is history only.

## 2. How this plan fits `docs/ROADMAP_2026-10.md`

| Topic | Owned by |
|---|---|
| Release order v0.7 → v0.8 → v0.9, player comparison features, **full editor E1–E10**, Rust-after-profile rule | `docs/ROADMAP_2026-10.md` |
| **Engine for 2.0 Next:** golden benchmark, AutoDirector, Smart Batch, device optimizer, optional local AI; specs and inventories | this folder |

Where both touch (editor schema P5, benchmark gate P4), the specs here give the
details and `ROADMAP_2026-10.md` gives the order.

## 3. Rules

1. **Everything is free.** Every feature ships to everyone, unless the Owner
   keeps it in the internal Suite build. Donations give no extra rights.
2. **Safety first.** Emergency stop and device caps work in every build, always.
3. **Classical first, AI optional.** Every AI result is checked by an
   independent verifier against video evidence, timebase and device limits.
   Without a model, the classical pipeline still works.
4. **Formats:** `.funscript` stays the universal export; `.samn` is extended
   backwards-compatibly only. No invented device axes.
5. **Go is the core;** Rust only for measured hot paths with golden-clip parity
   (same rule as `ROADMAP_2026-10.md` §1 and `docs/SELF_BUILD.md`).
6. **Local and private.** No telemetry, no hidden uploads, learning is opt-in
   with export and delete.
7. **No empty buttons, no unproven claims.**
8. **Users keep their data.** Updates never break existing `.samn`,
   `.funscript`, projects or settings.

## 4. What 2.0 Next adds

| Rank | Feature | IDs (from `sources/`) | Why this order |
|---|---|---|---|
| 1 | **Golden benchmark** — manifest v2, report, no-regression gate | ENG-01, P4 | Every later claim needs it; also the gate for the first Rust step |
| 2 | **Smart Batch** — Go job queue, crash resume, per-clip retry, reports | STUDIO-A2, A6, STU-01/02, P6, CAP-02 | Clear value; today's queue lives only in the UI and is lost on restart. Project Recovery (CAP-12) comes first |
| 3 | **AutoDirector** — fast scan, per-scene method choice, selective retry, verifier | STUDIO-A1, A3, A5, ENG-02..04 | Main quality lever; needs 1 |
| 4 | **Device optimizer** — capability-aware projection, preview behind *Arm* | STUDIO-A4, D2–D5, E8 | Builds on the device work already merged |
| 5 | Local AI assist (LM Studio / ONNX), learning in shadow mode | ENG-05, AUTO-02, STUDIO-B3 | Only after verifier and goldens exist |
| — | Full editor | E1–E10 | Planned in `ROADMAP_2026-10.md` §4 |

## 5. Architecture (target)

```text
Wails UI -> Go Job API
  -> VideoProbe / FrameCache (FFmpeg, PTS-aware)
  -> SceneScanner (classical)
  -> BackendRegistry [legacy Python | native Go | native Rust]
  -> CandidateScorer -> SelectiveRetry -> CurveComposer
  -> optional LocalAIAdapter -> IndependentVerifier
  -> .samn + .funscript -> Device Adapter -> Safety Governor
```

Contracts (`TrackResult`, `DecisionResult`, `CurveSegment`, `JobState`,
`EditCommand`, `AxisDescriptor`, `TimelinePosition` in integer µs PTS):
`sources/FREE_TO_STUDIO_PRODUCT_PLAN_2026-10-12.md` §4 and
`sources/STUDIO_EDITOR_COMPETITIVE_BLUEPRINT_2026-10-12.md` §1.3. Proposals until
their PR lands with tests. Batch is orchestration over the same single-clip
pipeline, never a second generator.

## 6. Test levels

L0 static · L1 deterministic fixtures · L2 golden clips (local, manifests only in
git) · L3 local AI with a mocked OpenAI-compatible server · L4 device mocks ·
L5 native Windows 11 at 1366×768 and 1920×1080 · L6 performance bake-off (3 warm
+ 5 measured runs, median/p95, peak RAM, no quality regression). Details:
`sources/NEXT_LEVEL_REPOSITORY_BOOTSTRAP_2026-10-12.md`.
