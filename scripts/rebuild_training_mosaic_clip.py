#!/usr/bin/env python3
"""Rebuild Training mosaic clip frames (f00–f15) from a video window.

Keeps the existing Curriculum: f00 = retracted / low curve, f15 = peak / high
curve. Pixelation matches frames.json (cell=9, size=320).

Example (Owner clip 2:14–2:40):

  python3 scripts/rebuild_training_mosaic_clip.py \\
    --video /path/to/clip_voll.mp4 \\
    --start-sec 134 --end-sec 160 \\
    --out cmd/gui-wails/frontend/src/assets/images/training-mosaic-clip
"""

from __future__ import annotations

import argparse
import json
import os
from pathlib import Path

import cv2
import numpy as np


def pixelate(bgr: np.ndarray, cell: int, size: int) -> np.ndarray:
    h, w = bgr.shape[:2]
    small = cv2.resize(bgr, (max(1, w // cell), max(1, h // cell)), interpolation=cv2.INTER_LINEAR)
    big = cv2.resize(small, (size, size), interpolation=cv2.INTER_NEAREST)
    return big


def crop_box(frame: np.ndarray, box) -> np.ndarray:
    h, w = frame.shape[:2]
    x0, y0, x1, y1 = [int(v) for v in box]
    x0 = max(0, min(w - 1, x0))
    x1 = max(x0 + 1, min(w, x1))
    y0 = max(0, min(h - 1, y0))
    y1 = max(y0 + 1, min(h, y1))
    return frame[y0:y1, x0:x1]


def sample_indices(n_frames: int, start_f: int, end_f: int, count: int) -> list[int]:
    end_f = min(n_frames - 1, max(start_f + 1, end_f))
    if count == 1:
        return [start_f]
    return [int(round(start_f + i * (end_f - start_f) / (count - 1))) for i in range(count)]


def tip_proxy_score(gray: np.ndarray) -> float:
    """Cheap vertical motion proxy: brighter mass near top of central column → peak-ish."""
    h, w = gray.shape
    x0, x1 = int(w * 0.35), int(w * 0.65)
    y0, y1 = int(h * 0.05), int(h * 0.75)
    roi = gray[y0:y1, x0:x1].astype(np.float32)
    if roi.size == 0:
        return 0.0
    # Row-weighted mean brightness (upper rows weigh more for "tip toward face").
    weights = np.linspace(1.0, 0.15, roi.shape[0], dtype=np.float32)[:, None]
    return float((roi * weights).mean())


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("--video", required=True)
    ap.add_argument("--start-sec", type=float, default=134.0)
    ap.add_argument("--end-sec", type=float, default=160.0)
    ap.add_argument("--count", type=int, default=16)
    ap.add_argument("--cell", type=int, default=9)
    ap.add_argument("--size", type=int, default=320)
    ap.add_argument(
        "--crop",
        default="280,0,1000,700",
        help="x0,y0,x1,y1 in source pixels (default matches prior frames.json)",
    )
    ap.add_argument(
        "--out",
        default="cmd/gui-wails/frontend/src/assets/images/training-mosaic-clip",
    )
    ap.add_argument(
        "--order",
        choices=("time", "score-asc"),
        default="score-asc",
        help="score-asc = low tip proxy first (f00 retracted → f15 peak)",
    )
    args = ap.parse_args()

    cap = cv2.VideoCapture(args.video)
    if not cap.isOpened():
        raise SystemExit(f"cannot open {args.video}")
    fps = float(cap.get(cv2.CAP_PROP_FPS) or 30.0)
    n = int(cap.get(cv2.CAP_PROP_FRAME_COUNT) or 0)
    start_f = int(max(0, round(args.start_sec * fps)))
    end_f = int(max(start_f + 1, round(args.end_sec * fps)))
    if n > 0:
        end_f = min(end_f, n - 1)

    # Dense sample then pick count ordered by tip proxy (or evenly in time).
    dense = sample_indices(max(n, end_f + 1), start_f, end_f, max(args.count * 8, args.count))
    scored = []
    crop = [int(x) for x in args.crop.split(",")]
    for idx in dense:
        cap.set(cv2.CAP_PROP_POS_FRAMES, idx)
        ok, frame = cap.read()
        if not ok or frame is None:
            continue
        crop_f = crop_box(frame, crop)
        gray = cv2.cvtColor(crop_f, cv2.COLOR_BGR2GRAY)
        scored.append((tip_proxy_score(gray), idx, crop_f))
    cap.release()
    if len(scored) < args.count:
        raise SystemExit(f"only got {len(scored)} frames in window")

    if args.order == "time":
        scored.sort(key=lambda t: t[1])
        picked = [
            scored[int(round(i * (len(scored) - 1) / (args.count - 1)))]
            for i in range(args.count)
        ]
    else:
        scored.sort(key=lambda t: t[0])  # low → high tip proxy
        # Unique-ish indices across the depth range
        picks = []
        for i in range(args.count):
            j = int(round(i * (len(scored) - 1) / (args.count - 1)))
            picks.append(scored[j])
        picked = picks

    out = Path(args.out)
    out.mkdir(parents=True, exist_ok=True)
    source_idxs = []
    depths = []
    lo, hi = picked[0][0], picked[-1][0]
    span = (hi - lo) or 1.0
    for i, (score, idx, crop_f) in enumerate(picked):
        pix = pixelate(crop_f, args.cell, args.size)
        path = out / f"f{i:02d}.png"
        cv2.imwrite(str(path), pix)
        source_idxs.append(int(idx))
        depths.append(round((score - lo) / span, 4))
        print(f"wrote {path.name} src_frame={idx} score={score:.3f}")

    meta = {
        "frames": [f"f{i:02d}.png" for i in range(args.count)],
        "source_idxs": source_idxs,
        "depths": depths,
        "crop_display": crop,
        "cell": args.cell,
        "size": args.size,
        "count": args.count,
        "source_video": os.path.basename(args.video),
        "window_sec": [args.start_sec, args.end_sec],
        "fps": fps,
        "note": "CANONICAL Vorlage: pixelated clip stroke. Soft assets derived from clip frames. Curve→frame drives penis between breasts.",
        "vorlage": "clip-locked",
        "order": args.order,
    }
    (out / "frames.json").write_text(json.dumps(meta, indent=2) + "\n", encoding="utf-8")
    print(f"wrote {out / 'frames.json'}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
