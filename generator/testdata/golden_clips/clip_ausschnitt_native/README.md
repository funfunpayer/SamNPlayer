# Golden clip: `clip_ausschnitt` (native Go pipeline vs. Python path)

Supplied by the user 21 Sep 2026, showing the new native Go pipeline's
("Go Erkennung", `generator/trackcv` + `generator/posttrack`, no Python)
output for `clip_ausschnitt` (~50s). No FunGen reference exists for this
clip, so this dataset is a **parity/internal-consistency check**, not a
Motion Fidelity score against ground truth. See `docs/NEXT.md` ("Native Go
pipeline, `clip_ausschnitt`") and `docs/FINDINGS_TIMING_TF.md` (F-003) for
the measurement and interpretation. Source video (`clip_ausschnitt.mp4`)
is **not** committed (license + size, per `docs/GOLDEN_CLIPS.md`).

## Contents

| File | Actions | Source | Notes |
|---|---|---|---|
| `clip_ausschnitt_python_OLD.funscript` | 175 | Python path (`generate_funscript.py`, classical CV), earlier session | Baseline, pos clamped 20-90 |
| `clip_ausschnitt_native_v1.funscript` | 71 | Native Go, sparse export, has chapters/tags metadata | |
| `clip_ausschnitt_native_v2.funscript` | 738 | Native Go, dense export, has chapters/tags metadata | User named this upload "ohne_yolo" — misleading here: the native Go pipeline is classical CSRT only, no YOLO/detector involved at any point. Keep the label only as the user's original filename, not as a meaningful YOLO/no-YOLO distinction like the `clip_voll_tftj` FunGen dataset. |
| `clip_ausschnitt_native_v3.funscript` | 145 | Native Go, minimal metadata (no chapters/tags) | User uploaded this exact file twice (byte-identical) |

`v1`, `v2`, and `v3` all report the **identical** underlying CSRT tracking
run in `metadata.native_pipeline`: `frames=1199, lost=0,
range_px=105.0539100525142` (matching to 13 decimal places across all
three — not a coincidence). Only their `posttrack`/export settings
differ (keyframe density, chapters).

## Measurement (reproduce with the Go `phase` CLI)

```bash
go run ./cmd/cli phase clip_ausschnitt_native_v1.funscript clip_ausschnitt_native_v3.funscript -max-lag-ms 1000
go run ./cmd/cli phase clip_ausschnitt_native_v1.funscript clip_ausschnitt_native_v3.funscript -max-lag-ms 1000 -window-ms 10000
```

Whole-clip r between the three NEW exports (same tracking run): 0.27-0.47
— not the >0.9 same-signal exports should read. Windowed (10s): mean r
rises to 0.53-0.59, per-window lag swings up to ±700-900ms, orientation
flips window to window. Old Python vs. any new Go export: similarly weak
(whole-clip r=0.14-0.24, windowed mean r≈0.53-0.55) — the Go-native path
is not obviously worse than Python on this clip, but neither is clean.

**Reading:** because the low-correlating pairs include two exports of the
exact same tracked positions, this cannot be a cross-tool ROI-marking
difference — it points at `posttrack`'s keyframe timestamp assignment
(frame-index-based, not real per-frame PTS) as the mechanism. See F-003
in `docs/FINDINGS_TIMING_TF.md`.
