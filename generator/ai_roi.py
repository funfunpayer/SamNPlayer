#!/usr/bin/env python3
"""ai_roi.py - schlägt die Bewegungsregion per lokalem ONNX-Objekterkenner
vor, als KI-Alternative zu `auto_roi.py` (Rhythmus-Heuristik ohne Modell).

Warum ein eigenes Modul statt eines Umbaus von auto_roi.py: beide erfüllen
denselben Vertrag (`find_roi(video, start_frame, end_frame) -> (x,y,w,h)`)
und lassen sich darum 1:1 austauschen - `track_by_scenes()` in
generate_funscript.py nimmt jeden `roi_finder` mit dieser Signatur, siehe
den Parameter dort. auto_roi.py bleibt unverändert die Vorgabe; dieses Modul
ist eine Ergänzung, keine Ablösung.

WICHTIG - was hier NICHT passiert:
- Kein Modell liegt im Repository oder wird automatisch heruntergeladen.
  `model_path` zeigt auf eine vom Nutzer bereitgestellte .onnx-Datei; ohne
  sie meldet `available()` False und `find_roi()` einen klaren Fehler, den
  `track_by_scenes()` bereits abfängt und auf die vorherige Region
  zurückfällt (siehe dortiger try/except).
- Kein Code oder Gewicht aus FunGen 1 (PolyForm Strict) oder FunGen 2
  (geschlossenes Binary) wurde gelesen oder übernommen. Nur öffentlich
  beschriebenes VERHALTEN (fungen.app, Changelog: YOLO-Erkennung +
  Tracking) diente als Vorbild, siehe docs/TEAM_STAND.md. Der Vertrag unten
  ist eine eigene, unabhängig entworfene Schnittstelle.
- `onnxruntime` ist NICHT in requirements.txt - es bleibt eine optionale
  Abhängigkeit (siehe requirements-ai.txt), damit der Generator ohne lokale
  KI weiterhin ohne zusätzliche Installation läuft ("was nicht gebraucht
  wird, kommt nicht rein", HANDOFF.md).

MODELL-VERTRAG
---------------
Erwartet wird ein ONNX-Modell mit einem Eingabetensor `1x3xHxW` (float32,
RGB, Werte 0..1) und einem Ausgabetensor, der sich verlustfrei zu
`(N, 6)` reshapen lässt: pro Zeile `[x0, y0, x1, y1, confidence, class_id]`
in NORMALISIERTEN Bildkoordinaten (0..1, nicht Pixel - das entkoppelt die
Nachverarbeitung von der Eingabegröße). Das ist ein bewusst einfacher,
eigener Vertrag - kein Nachbau eines bestimmten YOLO-Exportformats -, damit
sich ein beliebiges kompatibel exportiertes Detektionsmodell einstecken
lässt. `decode_detections()` und `select_best_box()` sind reine Funktionen
und darum ohne echtes Modell testbar (siehe ai_roi_test.py).

Nutzung (eigenständig oder aus generate_funscript.py importiert):
  python3 ai_roi.py --video input.mp4 --model roi_detector.onnx
    -> gibt "ROI x,y,w,h" wie auto_roi.py auf stdout aus
"""

import argparse
import os
import sys

import cv2
import numpy as np


class ModelUnavailable(RuntimeError):
    """onnxruntime fehlt oder das Modell wurde nicht gefunden."""


def default_model_path():
    """Ablageort analog zu backends.default_plugin_dir() - ein
    Nutzerkonfigurationsordner, kein Pfad im Repository/Binary."""
    local = os.environ.get("LOCALAPPDATA")
    if local:
        return os.path.join(local, "SamNPlayer", "models", "roi_detector.onnx")
    xdg = os.environ.get("XDG_CONFIG_HOME") or os.path.join(os.path.expanduser("~"), ".config")
    return os.path.join(xdg, "SamNPlayer", "models", "roi_detector.onnx")


def available(model_path=None):
    """True, wenn onnxruntime installiert ist UND ein Modell an model_path
    (oder dem Standardpfad) liegt. Reine Prüffunktion, löst nichts aus -
    die GUI kann sie nutzen, um den KI-Button zu aktivieren/auszublenden."""
    try:
        import onnxruntime  # noqa: F401
    except ImportError:
        return False
    path = model_path or default_model_path()
    return os.path.isfile(path)


def decode_detections(raw_output, confidence_threshold=0.35):
    """Reine Funktion: rohe (N,6)-Modellausgabe -> gefilterte Detektionen.

    Kein onnxruntime nötig, darum ohne Modell testbar. raw_output ist
    alles, was sich zu (N,6) reshapen lässt (Liste, ndarray, ...).
    """
    arr = np.asarray(raw_output, dtype=float).reshape(-1, 6)
    if arr.size == 0:
        return []
    keep = arr[:, 4] >= confidence_threshold
    detections = []
    for x0, y0, x1, y1, conf, cls in arr[keep]:
        if x1 <= x0 or y1 <= y0:
            continue  # entartete Box, z.B. aus Rauschen im Rohtensor
        detections.append({
            "x0": float(x0), "y0": float(y0), "x1": float(x1), "y1": float(y1),
            "confidence": float(conf), "class_id": int(cls),
        })
    return detections


def _filter_preferred(detections, preferred_class_ids=None):
    """If preferred_class_ids is set and at least one detection matches,
    keep only those. Otherwise return detections unchanged (honest fallback
    so a single-class model or an empty preference list still works)."""
    if not detections or not preferred_class_ids:
        return detections
    preferred = {int(c) for c in preferred_class_ids}
    filtered = [d for d in detections if int(d.get("class_id", -1)) in preferred]
    return filtered if filtered else detections


def _depth_rank_enabled(use_depth_rank=False):
    if use_depth_rank:
        return True
    return os.environ.get("SAMNPLAYER_DEPTH_RANK", "").strip().lower() in (
        "1", "true", "yes", "on")


def _optional_depth_map(frame_bgr, use_depth_rank=False):
    if frame_bgr is None or not _depth_rank_enabled(use_depth_rank):
        return None
    try:
        import support_signals
        gray = cv2.cvtColor(frame_bgr, cv2.COLOR_BGR2GRAY)
        return support_signals.relative_depth_map(gray)
    except Exception:
        return None


def _detection_rank_score(det, frame_w, frame_h, depth_map=None, use_depth_rank=False):
    """Confidence plus optional soft depth contrast bonus (opt-in only)."""
    score = float(det["confidence"])
    if not _depth_rank_enabled(use_depth_rank) or depth_map is None:
        return score
    try:
        import support_signals
    except ImportError:
        return score
    x0 = int(det["x0"] * frame_w)
    y0 = int(det["y0"] * frame_h)
    x1 = int(det["x1"] * frame_w)
    y1 = int(det["y1"] * frame_h)
    w = max(1, x1 - x0)
    h = max(1, y1 - y0)
    bonus = support_signals.depth_roi_confidence((x0, y0, w, h), depth_map)
    return score + 0.12 * bonus


def select_best_box(detections, frame_w, frame_h, min_size_px=16,
                    preferred_class_ids=None, use_depth_rank=False, depth_map=None):
    """Pure function: detection list (normalized coords) ->
    (x,y,w,h) in pixels for the highest-confidence box, or None.

    When preferred_class_ids is given and any detection matches, only those
    classes compete; otherwise all detections are considered (so a
    breast/hand-trained model can prefer the contact class without
    failing when that class is absent in the frame).

    When use_depth_rank is True or SAMNPLAYER_DEPTH_RANK=1, detections get a
    small bonus from support_signals.depth_roi_confidence (does not run when
    depth_map is None).
    """
    candidates = _filter_preferred(detections, preferred_class_ids)
    if not candidates:
        return None
    best = max(
        candidates,
        key=lambda d: _detection_rank_score(d, frame_w, frame_h, depth_map, use_depth_rank),
    )
    x0 = int(best["x0"] * frame_w)
    y0 = int(best["y0"] * frame_h)
    x1 = int(best["x1"] * frame_w)
    y1 = int(best["y1"] * frame_h)
    x0, x1 = sorted((max(0, x0), min(frame_w, x1)))
    y0, y1 = sorted((max(0, y0), min(frame_h, y1)))
    w = max(min_size_px, x1 - x0)
    h = max(min_size_px, y1 - y0)
    x0 = min(x0, max(0, frame_w - w))
    y0 = min(y0, max(0, frame_h - h))
    return (x0, y0, w, h)


def _iou(a, b):
    """Intersection over Union zweier Boxen in normalisierten (x0,y0,x1,y1)-
    Detektionen - reine Geometrie, kein Bezug zu Pixelkoordinaten nötig."""
    ix0, iy0 = max(a["x0"], b["x0"]), max(a["y0"], b["y0"])
    ix1, iy1 = min(a["x1"], b["x1"]), min(a["y1"], b["y1"])
    inter = max(0.0, ix1 - ix0) * max(0.0, iy1 - iy0)
    area_a = max(0.0, a["x1"] - a["x0"]) * max(0.0, a["y1"] - a["y0"])
    area_b = max(0.0, b["x1"] - b["x0"]) * max(0.0, b["y1"] - b["y0"])
    union = area_a + area_b - inter
    return inter / union if union > 0 else 0.0


def select_two_best_boxes(detections, frame_w, frame_h, min_size_px=16, max_iou=0.3,
                          preferred_class_ids=None, use_depth_rank=False, depth_map=None):
    """Like select_best_box, but picks up to TWO spatially separated objects
    for profiles that need two regions (tf/tj).

    When preferred_class_ids has more than one id, the second box prefers a
    *different* class from the first (e.g. hand + breast) before falling
    back to any low-IoU candidate. Never invents a second box: box2 is None
    when nothing sufficiently separated exists.
    """
    candidates = _filter_preferred(detections, preferred_class_ids)
    if not candidates:
        return None, None
    ranked = sorted(
        candidates,
        key=lambda d: _detection_rank_score(d, frame_w, frame_h, depth_map, use_depth_rank),
        reverse=True,
    )
    best = ranked[0]
    box1 = select_best_box([best], frame_w, frame_h, min_size_px)
    best_cls = int(best.get("class_id", -1))
    # Prefer a different class when multi-class preference is active.
    second = None
    if preferred_class_ids and len(set(int(c) for c in preferred_class_ids)) > 1:
        second = next(
            (d for d in ranked[1:]
             if int(d.get("class_id", -1)) != best_cls and _iou(d, best) <= max_iou),
            None)
    if second is None:
        second = next((d for d in ranked[1:] if _iou(d, best) <= max_iou), None)
    box2 = select_best_box([second], frame_w, frame_h, min_size_px) if second else None
    return box1, box2


def load_class_registry(path):
    """Load name->id map from classes.json (dataset dir or file path)."""
    import json
    if not path:
        return {}
    file_path = path
    if os.path.isdir(path):
        file_path = os.path.join(path, "classes.json")
    if not os.path.isfile(file_path):
        return {}
    with open(file_path, encoding="utf-8") as f:
        data = json.load(f)
    return {str(k): int(v) for k, v in data.items()} if isinstance(data, dict) else {}


def resolve_preferred_class_ids(names_or_ids, registry=None):
    """Parse 'hand,breast' / '0,1' / mixed into a list of int class ids.
    Names need registry (classes.json). Unknown names are skipped."""
    if not names_or_ids:
        return None
    if isinstance(names_or_ids, (list, tuple)):
        parts = [str(p).strip() for p in names_or_ids if str(p).strip()]
    else:
        parts = [p.strip() for p in str(names_or_ids).split(",") if p.strip()]
    if not parts:
        return None
    registry = registry or {}
    # case-insensitive name lookup
    by_lower = {str(k).lower(): int(v) for k, v in registry.items()}
    out = []
    for p in parts:
        if p.isdigit() or (p.startswith("-") and p[1:].isdigit()):
            out.append(int(p))
            continue
        cid = by_lower.get(p.lower())
        if cid is not None:
            out.append(cid)
    return out or None


def _preprocess_frame(frame_bgr, input_size):
    """BGR-Frame (OpenCV) -> (1,3,H,W) float32 RGB 0..1, wie im Modell-Vertrag."""
    resized = cv2.resize(frame_bgr, (input_size, input_size))
    rgb = cv2.cvtColor(resized, cv2.COLOR_BGR2RGB).astype(np.float32) / 255.0
    chw = np.transpose(rgb, (2, 0, 1))
    return np.expand_dims(chw, axis=0)


def _load_session(model_path):
    try:
        import onnxruntime
    except ImportError as exc:
        raise ModelUnavailable(
            "onnxruntime ist nicht installiert - siehe generator/requirements-ai.txt"
        ) from exc
    if not os.path.isfile(model_path):
        raise ModelUnavailable(f"Kein Modell unter {model_path!r} gefunden")
    return onnxruntime.InferenceSession(model_path, providers=["CPUExecutionProvider"])


def _run_model(session, frame_bgr, input_size=640):
    """Echte Inferenz - braucht onnxruntime + Modell. Nicht unit-getestet
    (kein Modell im Repo); ai_roi_test.py prüft stattdessen decode/select
    und find_roi() mit injiziertem _run_model_fn."""
    input_name = session.get_inputs()[0].name
    tensor = _preprocess_frame(frame_bgr, input_size)
    raw = session.run(None, {input_name: tensor})[0]
    return raw


def find_roi(video_path, start_frame=0, end_frame=None, report_progress=True,
             model_path=None, confidence_threshold=0.35, sample_frames=5,
             preferred_class_ids=None, use_depth_rank=False, _run_model_fn=None):
    """Same contract as auto_roi.find_roi: returns (x,y,w,h) or raises.

    preferred_class_ids: optional list of class ids (from classes.json) so a
    multi-class ROI model prefers e.g. hand/breast over background junk.
    """
    model_path = model_path or default_model_path()
    if _run_model_fn is None:
        session = _load_session(model_path)  # raises ModelUnavailable
        _run_model_fn = lambda frame: _run_model(session, frame)

    cap = cv2.VideoCapture(video_path)
    if not cap.isOpened():
        raise RuntimeError(f"Video konnte nicht geöffnet werden: {video_path}")
    try:
        total = int(cap.get(cv2.CAP_PROP_FRAME_COUNT))
        width = int(cap.get(cv2.CAP_PROP_FRAME_WIDTH))
        height = int(cap.get(cv2.CAP_PROP_FRAME_HEIGHT))
        end = min(end_frame, total) if end_frame is not None else total
        end = max(end, start_frame + 1)
        indices = sorted(set(
            int(start_frame + i * (end - start_frame - 1) / max(1, sample_frames - 1))
            for i in range(sample_frames)
        ))

        best_overall = None
        for n, idx in enumerate(indices):
            if report_progress:
                print(f"PROGRESS {n} {len(indices)}", file=sys.stderr, flush=True)
            cap.set(cv2.CAP_PROP_POS_FRAMES, idx)
            ok, frame = cap.read()
            if not ok:
                continue
            raw = _run_model_fn(frame)
            detections = decode_detections(raw, confidence_threshold)
            detections = _filter_preferred(detections, preferred_class_ids)
            depth_map = _optional_depth_map(frame, use_depth_rank)
            for det in detections:
                if best_overall is None or _detection_rank_score(
                        det, width, height, depth_map, use_depth_rank) > _detection_rank_score(
                        best_overall, width, height, depth_map, use_depth_rank):
                    best_overall = det
        if report_progress:
            print(f"PROGRESS {len(indices)} {len(indices)}", file=sys.stderr, flush=True)
    finally:
        cap.release()

    if best_overall is None:
        raise RuntimeError(
            "KI-Regionssuche: keine Erkennung über der Konfidenzschwelle - "
            "bitte Region von Hand markieren oder --roi-finder auto verwenden")
    return select_best_box([best_overall], width, height,
                           preferred_class_ids=preferred_class_ids,
                           use_depth_rank=use_depth_rank)


def find_two_rois(video_path, start_frame=0, end_frame=None, report_progress=True,
                   model_path=None, confidence_threshold=0.35, sample_frames=5,
                   preferred_class_ids=None, use_depth_rank=False, _run_model_fn=None):
    """Like find_roi(), but returns (roi1, roi2). roi2 is None when no second
    sufficiently separated object was found (see select_two_best_boxes).

    Still a human-confirmed proposal — never silently auto-committed (#8).
    """
    model_path = model_path or default_model_path()
    if _run_model_fn is None:
        session = _load_session(model_path)  # raises ModelUnavailable
        _run_model_fn = lambda frame: _run_model(session, frame)

    cap = cv2.VideoCapture(video_path)
    if not cap.isOpened():
        raise RuntimeError(f"Video konnte nicht geöffnet werden: {video_path}")
    try:
        total = int(cap.get(cv2.CAP_PROP_FRAME_COUNT))
        width = int(cap.get(cv2.CAP_PROP_FRAME_WIDTH))
        height = int(cap.get(cv2.CAP_PROP_FRAME_HEIGHT))
        end = min(end_frame, total) if end_frame is not None else total
        end = max(end, start_frame + 1)
        indices = sorted(set(
            int(start_frame + i * (end - start_frame - 1) / max(1, sample_frames - 1))
            for i in range(sample_frames)
        ))

        pooled = []
        last_frame = None
        for n, idx in enumerate(indices):
            if report_progress:
                print(f"PROGRESS {n} {len(indices)}", file=sys.stderr, flush=True)
            cap.set(cv2.CAP_PROP_POS_FRAMES, idx)
            ok, frame = cap.read()
            if not ok:
                continue
            last_frame = frame
            raw = _run_model_fn(frame)
            pooled.extend(decode_detections(raw, confidence_threshold))
        if report_progress:
            print(f"PROGRESS {len(indices)} {len(indices)}", file=sys.stderr, flush=True)
    finally:
        cap.release()

    if not pooled:
        raise RuntimeError(
            "KI-Regionssuche: keine Erkennung über der Konfidenzschwelle - "
            "bitte Regionen von Hand markieren oder --roi-finder auto verwenden")
    depth_map = _optional_depth_map(last_frame, use_depth_rank)
    return select_two_best_boxes(pooled, width, height,
                                 preferred_class_ids=preferred_class_ids,
                                 use_depth_rank=use_depth_rank, depth_map=depth_map)


def main():
    ap = argparse.ArgumentParser(description=__doc__,
                                  formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--video", help="Pfad zum Video (nicht nötig mit --check)")
    ap.add_argument("--model", default=None, help="Pfad zur .onnx-Datei (sonst Standardordner)")
    ap.add_argument("--confidence", type=float, default=0.35)
    ap.add_argument("--preferred-classes", default=None,
                     help="Comma-separated class names or ids to prefer "
                          "(e.g. hand,breast or 0,1). Names need --classes-json.")
    ap.add_argument("--classes-json", default=None,
                     help="Path to classes.json or its dataset directory "
                          "(default: sibling of --model, then dataset under config).")
    ap.add_argument("--check", action="store_true",
                     help="Nur prüfen, ob KI-Erkennung nutzbar ist (onnxruntime + Modell "
                          "vorhanden), ohne Video zu öffnen - für die GUI, um den KI-Knopf "
                          "zu aktivieren/auszublenden, ohne selbst eine Erkennung anzustoßen.")
    ap.add_argument("--two", action="store_true",
                     help="Zwei Regionen statt einer suchen (find_two_rois statt find_roi) - "
                          "für Profile mit zweiter Region (tf/tj). UNGETESTET gegen echtes "
                          "Material, siehe find_two_rois()-Docstring: bleibt ein Vorschlag, "
                          "den ein Mensch bestätigt/korrigiert, nie automatisch übernommen.")
    args = ap.parse_args()

    if args.check:
        ok = available(args.model)
        path = args.model or default_model_path()
        print("AVAILABLE" if ok else "UNAVAILABLE")
        print(f"model_path={path}", file=sys.stderr)
        return

    if not args.video:
        ap.error("--video ist erforderlich, außer bei --check")

    preferred = None
    if args.preferred_classes:
        registry_path = args.classes_json
        if not registry_path and args.model:
            registry_path = os.path.dirname(os.path.abspath(args.model))
        registry = load_class_registry(registry_path) if registry_path else {}
        preferred = resolve_preferred_class_ids(args.preferred_classes, registry)
        if preferred is None:
            print("Warning: --preferred-classes could not be resolved "
                  "(names require --classes-json)", file=sys.stderr)

    if args.two:
        try:
            roi1, roi2 = find_two_rois(args.video, model_path=args.model,
                                        confidence_threshold=args.confidence,
                                        preferred_class_ids=preferred)
        except (ModelUnavailable, RuntimeError) as exc:
            print(f"AI region search failed: {exc}", file=sys.stderr)
            sys.exit(1)
        if roi1 is None:
            print("AI region search failed: no region found", file=sys.stderr)
            sys.exit(1)
        x, y, w, h = roi1
        print(f"ROI {x} {y} {w} {h}")
        print(f"AI region 1 found: x={x} y={y} w={w} h={h}", file=sys.stderr)
        if roi2 is not None:
            x2, y2, w2, h2 = roi2
            print(f"ROI2 {x2} {y2} {w2} {h2}")
            print(f"AI region 2 found: x={x2} y={y2} w={w2} h={h2}", file=sys.stderr)
        else:
            print("AI region 2: no second distinct object found — mark manually", file=sys.stderr)
        return

    try:
        x, y, w, h = find_roi(args.video, model_path=args.model,
                               confidence_threshold=args.confidence,
                               preferred_class_ids=preferred)
    except (ModelUnavailable, RuntimeError) as exc:
        print(f"AI region search failed: {exc}", file=sys.stderr)
        sys.exit(1)

    print(f"ROI {x} {y} {w} {h}")
    print(f"AI region found: x={x} y={y} w={w} h={h}", file=sys.stderr)


if __name__ == "__main__":
    main()
