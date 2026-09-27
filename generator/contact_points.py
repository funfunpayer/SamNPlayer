#!/usr/bin/env python3
"""contact_points.py - asks one or more LOCAL teachers where the stroke
contact is and writes `<clip>.contact.json` for the rhythm grid
(`generate --rhythm-grid --contact-points <file>`, Options.ContactPointsFile).

VLM1 of docs/VLM_TEACHER_PLAN.md. The engine uses a point only where the
CSRT box is further from it than the grid's search radius (3 cells), so a
wrong point can not pull a working track away; measured on the multi-person
clip: windowed r 0.304 -> 0.406 with NudeNet alone (real Go TrackROI);
goldens unchanged (clip_ausschnitt bit-identical, clip_voll same r).

Teachers (use any combination - the Owner's "ask several per video"):
- NudeNet (`--nudenet`): body-part detector, runs every --step-s seconds.
  Contact rule: exposed penis centre, else the cleavage point between a
  breast pair, else nothing. (A face/mouth rule measured 0 % correct in
  titjob scenes and is deliberately absent.) Optional dependency: the
  package is MIT and bundles its ONNX model; install it next to the
  generator's OpenCV with `pip install nudenet --no-deps onnxruntime`.
- VLM probe results (`--vlm clip.vlm.json`, repeatable): the contact box
  centre of each keyframe, e.g. Qwen2.5-VL and Qwen3-VL runs of vlm_probe.py.

Every point records how many distinct teachers put a point within
--agree-cells grid cells and --agree-ms of it (`agree`). `--contact-min-agree
2` on the Generate side uses only consensus points; those are also the
training-data candidates (reviewed:false) for our own detector later.

Local only: frames never leave the machine, no model in the repo, nothing
downloads by itself.
"""

import argparse
import json
import math
import os
import sys

import cv2

GRID_COLS, GRID_ROWS = 16, 9
MIN_SCORE = 0.3


def nudenet_detector():
    """NudeNet's detect(frame) or a clear error if the package is missing."""
    try:
        from nudenet import NudeDetector
    except ImportError as e:  # pragma: no cover - depends on the machine
        raise RuntimeError(
            "NudeNet is not installed - `pip install nudenet --no-deps onnxruntime` "
            "(see docs/VLM_MODELS.md)") from e
    return NudeDetector().detect


def contact_from_parts(dets, w, h, min_score=MIN_SCORE):
    """NudeNet detections -> (x, y normalized, rule, score, box) or None.

    dets: [{"class": str, "score": float, "box": [x, y, w, h] px}]."""
    d = [x for x in dets if x.get("score", 0) >= min_score]
    g = [x for x in d if x["class"] == "MALE_GENITALIA_EXPOSED"]
    if g:
        best = max(g, key=lambda x: x["score"])
        bx, by, bw, bh = best["box"]
        return ((bx + bw / 2) / w, (by + bh / 2) / h, "penis", best["score"],
                [bx / w, by / h, (bx + bw) / w, (by + bh) / h])
    br = [x for x in d if x["class"] == "FEMALE_BREAST_EXPOSED"]
    pair = None
    for i in range(len(br)):
        for j in range(i + 1, len(br)):
            a, b = br[i]["box"], br[j]["box"]
            dx = abs((a[0] + a[2] / 2) - (b[0] + b[2] / 2))
            # One person's pair: side by side, not further apart than
            # ~2.5 breast widths (two people's breasts are further apart).
            if dx < 2.5 * max(a[2], b[2]) and (pair is None or dx < pair[0]):
                pair = (dx, br[i], br[j])
    if pair:
        _, p, q = pair
        a, b = p["box"], q["box"]
        x = ((a[0] + a[2] / 2) + (b[0] + b[2] / 2)) / 2 / w
        y = max(a[1] + a[3] * 0.6, b[1] + b[3] * 0.6) / h
        return x, y, "cleavage", min(p["score"], q["score"]), None
    return None


def nudenet_points(video, detect, step_s=0.5, log=print):
    cap = cv2.VideoCapture(video)
    if not cap.isOpened():
        raise RuntimeError(f"Could not open video: {video}")
    fps = cap.get(cv2.CAP_PROP_FPS) or 30.0
    n = int(cap.get(cv2.CAP_PROP_FRAME_COUNT) or 0)
    w, h = cap.get(cv2.CAP_PROP_FRAME_WIDTH), cap.get(cv2.CAP_PROP_FRAME_HEIGHT)
    step = max(1, int(round(step_s * fps)))
    out = []
    for i in range(0, n, step):
        cap.set(cv2.CAP_PROP_POS_FRAMES, i)
        ok, frame = cap.read()
        if not ok:
            break
        c = contact_from_parts(detect(frame), w, h)
        if c:
            x, y, rule, score, box = c
            p = {"t_ms": int(round(i * 1000 / fps)), "x": round(x, 4), "y": round(y, 4),
                 "source": "nudenet", "rule": rule, "score": round(float(score), 3)}
            if box:
                p["box"] = [round(v, 4) for v in box]
            out.append(p)
    cap.release()
    log(f"nudenet: {len(out)} points from {len(range(0, n, step))} frames")
    return out


def vlm_points(probe):
    """Contact box centres of a vlm_probe.py result."""
    src = "vlm:" + str(probe.get("model", "?"))
    out = []
    for f in probe.get("frames", []):
        boxes = [b for b in f.get("boxes", []) if b.get("label") == "contact"]
        if not boxes:
            continue
        b = max(boxes, key=lambda b: (b["x1"] - b["x0"]) * (b["y1"] - b["y0"]))
        out.append({"t_ms": int(f["t_ms"]), "x": round((b["x0"] + b["x1"]) / 2, 4),
                    "y": round((b["y0"] + b["y1"]) / 2, 4), "source": src,
                    "box": [b["x0"], b["y0"], b["x1"], b["y1"]]})
    return out


def count_agreement(points, agree_cells=1.5, agree_ms=1500):
    """Sets p["agree"] = number of distinct teachers (incl. p's own) with a
    point within agree_ms and agree_cells grid cells of p."""
    for p in points:
        teachers = {p["source"]}
        for q in points:
            if q["source"] in teachers or abs(q["t_ms"] - p["t_ms"]) > agree_ms:
                continue
            if math.hypot((q["x"] - p["x"]) * GRID_COLS, (q["y"] - p["y"]) * GRID_ROWS) <= agree_cells:
                teachers.add(q["source"])
        p["agree"] = len(teachers)
    return points


def build(video, use_nudenet=False, vlm_files=(), step_s=0.5, agree_cells=1.5,
          agree_ms=1500, detect=None, log=print):
    points, teachers = [], []
    if use_nudenet:
        points += nudenet_points(video, detect or nudenet_detector(), step_s, log)
        teachers.append("nudenet")
    for path in vlm_files:
        with open(path, encoding="utf-8") as f:
            vp = vlm_points(json.load(f))
        log(f"{path}: {len(vp)} contact points")
        points += vp
        teachers += sorted({p["source"] for p in vp}) or [f"vlm:{os.path.basename(path)}"]
    if not teachers:
        raise ValueError("no teacher selected - use --nudenet and/or --vlm")
    points.sort(key=lambda p: (p["t_ms"], p["source"]))
    count_agreement(points, agree_cells, agree_ms)
    agreed = sum(1 for p in points if p["agree"] >= 2)
    return {
        "version": 1, "tool": "contact_points", "video": os.path.basename(video),
        "teachers": teachers, "step_s": step_s,
        "agree_cells": agree_cells, "agree_ms": agree_ms,
        "points": points,
        "summary": {"points": len(points), "agree_2plus": agreed,
                    "by_source": {s: sum(1 for p in points if p["source"] == s)
                                  for s in sorted({p["source"] for p in points})}},
    }


def main(argv=None):
    ap = argparse.ArgumentParser(description=__doc__.split("\n\n")[0])
    ap.add_argument("--video", required=True)
    ap.add_argument("--nudenet", action="store_true", help="use the NudeNet teacher")
    ap.add_argument("--vlm", action="append", default=[],
                    help="vlm_probe.py result (.vlm.json), repeatable")
    ap.add_argument("--step-s", type=float, default=0.5)
    ap.add_argument("--agree-cells", type=float, default=1.5)
    ap.add_argument("--agree-ms", type=int, default=1500)
    ap.add_argument("--out")
    args = ap.parse_args(argv)
    try:
        res = build(args.video, args.nudenet, args.vlm, args.step_s,
                    args.agree_cells, args.agree_ms)
    except (RuntimeError, ValueError) as e:
        print(str(e), file=sys.stderr)
        return 2
    out = args.out or os.path.splitext(args.video)[0] + ".contact.json"
    with open(out, "w", encoding="utf-8") as f:
        json.dump(res, f, indent=1)
    print(json.dumps(res["summary"], indent=2))
    print(f"wrote {out}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
