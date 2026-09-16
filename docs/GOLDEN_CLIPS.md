# Golden-Clip protocol

Needed for FunGen parity. This file is the checklist; the clips stay on
the operator machine (license + size).

**Automated as of `generator/golden_clip_benchmark.py`** (GUI: the "Bench"
tab) — point its manifest at clips following this checklist and it runs
them through the real pipeline, computes Quality Doctor + FunGen
agreement per clip, and tracks results over time. This file stays the
checklist for *building* the clip set by hand; the tool is what runs it
repeatably. See `docs/ROADMAP.md` for how this fits the wider task list.

## What one clip must include

1. Source video (same cut FunGen used), note codec and resolution.
2. FunGen export as `.funscript` only — never `.fungen` / project files.
3. ROI1 and ROI2 written once in *video pixels* on frame 0. Reuse them.
4. SamNPlayer outputs: `--profile standard` (hub) and `--profile tj` with
   the same ROIs, CSRT and optionally `grid_lk`.
5. Keep invert *off* for the first pair; if `fungen_compare` reports
   inverted polarity, record that as a separate row, do not hide it in r.

## How to compare

```bash
python3 generator/fungen_compare.py --dataset DIR --output report.md --max-lag-ms 300
```

Report both Quality Doctor and FunGen r. They are not proxies.

## Minimum set

Three clips is enough: one easy hub, one two-body (Tf/Tj), one with a cut
or camera move. More than that without fixed ROIs repeats issue #8.

## Do not change yet

`smooth-window`, peaks, per-scene ROI, upscale, sharpen — only after the
matrix above exists and ROI quality is the remaining variable.
