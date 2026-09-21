# Golden clip: `clip_voll` (Tf/Tj)

First real (non-synthetic) Tf/Tj comparison dataset, supplied by the user
21 Sep 2026. Used for `docs/NEXT.md` "First real Tf/Tj golden-clip
measurement" and `docs/FINDINGS_TIMING_TF.md`. Per `docs/GOLDEN_CLIPS.md`,
the source video itself is **not** committed (license + size); only the
`.funscript`/`.samn` outputs, which are small text/JSON.

## Contents

Two sub-directories, one per FunGen reference, each laid out for
`fungen_compare.py --dataset DIR` (reference `.funscript` + SamNPlayer
`*__tj.funscript` batch-output naming convention):

- `mit_yolo/` — FunGen 2.6.3 reference generated with FunGen's own YOLO
  detector, plus SamNPlayer's `tj`-profile output for the same clip.
- `ohne_yolo/` — FunGen 2.6.3 reference generated with YOLO disabled
  (classical detector only — the fairer comparison against SamNPlayer's
  own classical CV tracking), plus the same SamNPlayer `tj` output.
- `clip_voll.samn` — SamNPlayer's native `.samn` sidecar for the same run
  (provenance only; comparisons use the `.funscript`).

The two `*__tj.funscript` files are byte-identical copies of the same
SamNPlayer run (one SamNPlayer output, compared against two different
FunGen references). They are kept in **separate** sub-directories rather
than one shared dataset directory: `fungen_compare.py`'s
`dedupe_by_content()` collapses identical-content variant files to guard
against counting one copied file twice (its bug-1 fix), which is correct
in general but would silently drop one of these two legitimate
comparisons if both lived in one directory. Run each sub-directory
separately, not run `--dataset clip_voll_tftj` on the parent.

Source clip: `clip_voll.mp4`, ~280s (stays local — see `docs/GOLDEN_CLIPS.md`).
Durations: 277.1s (`ohne_yolo` ref) / 279.999s (`mit_yolo` ref) / 280.1s
(SamNPlayer) — within a few seconds of each other, not a confound here.

## How to reproduce the numbers in `docs/NEXT.md`

```bash
# Whole-clip (existing tool)
python3 generator/fungen_compare.py --dataset generator/testdata/golden_clips/clip_voll_tftj/mit_yolo --max-lag-ms 3000
python3 generator/fungen_compare.py --dataset generator/testdata/golden_clips/clip_voll_tftj/ohne_yolo --max-lag-ms 3000

# Windowed (new tool, generator/fungen_compare_windowed.py)
python3 generator/fungen_compare_windowed.py \
  "generator/testdata/golden_clips/clip_voll_tftj/mit_yolo/clip_voll_2 mit yolo.funscript" \
  "generator/testdata/golden_clips/clip_voll_tftj/mit_yolo/clip_voll_2 mit yolo__tj.funscript" \
  --window-ms 30000 --max-lag-ms 3000
```

Expect: whole-clip r≈0.06 (both references), windowed mean r≈0.27–0.35 with
lag drifting roughly -2600..+2800ms across the clip and one orientation
flip partway through (~150-210s). Exact per-window numbers can differ
slightly run-to-run of the windowed script depending on window boundary
alignment; the qualitative finding (drift, not noise) is what to check.
