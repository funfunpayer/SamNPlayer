# Findings: Timing, Tf/Tj, Go-native (research triage)

Operator research (16–17 Sep 2026) triaged into this inventory. Direction
summary: `docs/ENGINE.md`. This file records what was **acted on**, what
stays **open pending measurement**, and what is **deferred** — per project
rules: no speculative modules, every quality claim needs a measurement.

## Acted on now

### Tf/Tj suction double-floor (finding §2)

**Claim:** with actions clamped to 20–90 and `MinSuction=0.20`,
`liftFloor(pos/100, 0.20)` remaps resting suction from 0.20 → **0.36**
and peak from 0.90 → **0.92**.

**Verified in code** (`funscript/mapper.go` + `RecipeFor("tj")`): yes.
`TestRecipeTJSuctionNoDoubleFloor` locks the intended mapping:
pos 20 → suction ≈ 0.20, pos 90 → ≈ 0.90.

**Fix:** for `SyncSuctionPosition`, skip the runtime `liftFloor` on
suction. The generation-time 20–90 clamp *is* the floor. Recipe metadata
still reports `min_suction: 0.20` as that script-space floor.

Still needs a **feel check on real hardware** (priority 1) — software
can only assert the command values, not perceived intensity.

### Phase Analyzer core in Go (finding F-001)

**Shipped:** `funscript.BestLagCorrelation` + `DiagnosePhase` — pure-Go
port of `fungen_compare.best_lag_correlation` (shared absolute timeline,
±lag search, orientation, shape error, low-confidence). CLI:
`SamNPlayer phase` / `compare`. Goldens mirror `fungen_compare_test.py`.

### Observation contract (F-004)

**Shipped on trackcv:** `ValidFrames` / `Confidence` / `Reason` (+ cancel)
written into native funscript metadata.

### Dense Quality Doctor + FunGen-compare CLI

**Shipped:** `EvaluateDenseQuality` on native CSRT; `SamNPlayer compare`.

## Signal Quality ≠ Motion Fidelity (17 Sep 2026)

Script Doctor / Quality Doctor = **Signal Quality**. FunGen /
`best_lag_ms` / Phase CLI = **Motion Fidelity**. Canonical:
`docs/SIGNAL_VS_FIDELITY.md`. Next architecture milestone: Perception v1
on real goldens — see `docs/ROADMAP.md` / `docs/ENGINE.md`.

## F-003: FrameIndex/FPS timing drift (open, now measured twice)

**Claim (source research material, 16 Sep 2026):** funscript keyframe
timestamps derived from frame *index* × nominal FPS drift from the
video's real presentation timestamps under VFR/frame-rate imprecision,
producing a growing (or otherwise non-constant) offset over a clip's
length rather than one fixed lag.

**Measured twice now, independently:**

1. `clip_voll` (21 Sep 2026, `docs/NEXT.md` "First real Tf/Tj golden-clip
   measurement"): SamNPlayer `tj`/`hub` vs. FunGen references — whole-clip
   r≈0.06, windowed (30s) mean r≈0.22–0.35, per-window lag range up to
   -2600..+2800ms, orientation flip mid-clip.
2. `clip_ausschnitt` (21 Sep 2026, `docs/NEXT.md` "Native Go pipeline,
   `clip_ausschnitt`"): same pattern between **two SamNPlayer exports of
   the identical CSRT tracking run** (native Go pipeline, no FunGen
   involved at all) — whole-clip r=0.27–0.47, windowed (10s) mean
   r=0.53–0.59, per-window lag swings up to ±900ms, orientation flips.

The second measurement narrows the mechanism: since both signals in that
comparison come from the *same* tracked positions (`frames`/`lost`/
`range_px` identical to 13 decimals), the drift cannot be a cross-tool
ROI/marking-convention difference — it has to be in how `generator/
posttrack` assigns absolute timestamps to keyframes derived from frame
indices. **Next step: instrument `posttrack`'s keyframe timestamp
assignment against the video's real per-frame PTS (not `frame_index /
nominal_fps`) and re-run this same `clip_ausschnitt` internal-consistency
check** — this is now a concrete, scoped repro case, not just an external
claim.

## Open inventory

### Already in Go

| Piece | Where |
|---|---|
| CSRT tracker (experimental) | `generator/trackcv` |
| Post-tracking signal path | `generator/posttrack` |
| Opt-in native CSRT generate | `generator/native*.go` |
| Script Doctor + dense doctor | `funscript.EvaluateScriptQuality` / `EvaluateDenseQuality` |
| Device-compat | `funscript.EvaluateDeviceCompat` |
| Phase / FunGen-compare | `BestLagCorrelation`, `WindowedBestLagCorrelation`, CLI `phase`/`compare` (`--window-ms`) |

### Still Python / blocked

| Piece | Notes |
|---|---|
| Default `generate_funscript.py` | Native covers CSRT+single ROI only |
| flow / grid_lk / fusion / Tf/Tj two-point | Need goldens before Go ports |
| Golden-clip real data | First real Tf/Tj comparison landed 21 Sep 2026, funscripts committed at `generator/testdata/golden_clips/clip_voll_tftj/` and `clip_ausschnitt_native/` (see `docs/NEXT.md`, F-003 above) — a same-day `hub`-profile run on `clip_voll` measured similarly low Motion Fidelity despite much higher Signal Quality (not Tf/Tj-specific), and the `clip_ausschnitt` native-pipeline set shows the same drift between two SamNPlayer exports with no FunGen involved at all. `golden_clip_benchmark.py` manifest still not wired up to either, but no longer zero evidence |
| Sam Neo 2 feel | Hardware |

### Next Go / product slices (quality-first)

1. **`phase` CLI: windowed mode.** **Shipped** (`funscript.WindowedBestLagCorrelation`
   + `FormatWindowedReport`, CLI `SamNPlayer phase A B --window-ms 30000`).
   21 Sep 2026 measurement: whole-clip r≈0.06 on a real Tf/Tj clip, but
   r≈0.27–0.35 averaged over 30s windows with the lag drifting
   -2600..+2800ms across the clip and one orientation flip partway
   through. Whole-clip `BestLagCorrelation` cannot represent a drifting
   offset; windowed mode can. Python twin remains at
   `generator/fungen_compare_windowed.py` for offline dataset runs.
2. Tf/Tj two-point in Go — only with goldens  
3. Native CSRT default — after real-clip measurement  
4. Full PTS→device phase chain — media work  
5. Windows OpenCV / accelerator abstraction — later  

### Not now

- 4-zone relative graph as default — measure first  
- Forced 63 BPM / Spatial 3D / Prediction — after measured latency  
- Rust rewrite — profiler gate only  

## Language reminder

Python-vs-Go is not the Tf/Tj correlation problem. Go removes install
burden; perception/timing/ROI semantics still decide quality.
