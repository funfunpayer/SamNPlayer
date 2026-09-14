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


def select_best_box(detections, frame_w, frame_h, min_size_px=16):
    """Reine Funktion: Detektionsliste (normalisierte Koordinaten) ->
    (x,y,w,h) in Pixeln der bestbewerteten Box, oder None.

    Wählt schlicht die höchste Konfidenz - anders als auto_roi.py (das
    mangels Objektbegriff mehrere Rasterzellen zusammenfassen muss) liefert
    ein Detektor bereits eine zusammenhängende Objektbox.
    """
    if not detections:
        return None
    best = max(detections, key=lambda d: d["confidence"])
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
             _run_model_fn=None):
    """Wie auto_roi.find_roi: liefert (x,y,w,h) oder wirft RuntimeError.

    Gleicher Vertrag wie auto_roi.find_roi (start_frame/end_frame für
    szenenweise Suche, siehe track_by_scenes in generate_funscript.py),
    damit sich beide als roi_finder gegeneinander austauschen lassen.

    _run_model_fn(frame_bgr) -> raw_output ist der Testhaken: ai_roi_test.py
    injiziert hier eine Attrappe statt echter Inferenz, genau wie
    track_by_scenes() für roi_finder selbst einen Injektionspunkt hat.
    """
    model_path = model_path or default_model_path()
    if _run_model_fn is None:
        session = _load_session(model_path)  # wirft ModelUnavailable
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
            for det in detections:
                if best_overall is None or det["confidence"] > best_overall["confidence"]:
                    best_overall = det
        if report_progress:
            print(f"PROGRESS {len(indices)} {len(indices)}", file=sys.stderr, flush=True)
    finally:
        cap.release()

    if best_overall is None:
        raise RuntimeError(
            "KI-Regionssuche: keine Erkennung über der Konfidenzschwelle - "
            "bitte Region von Hand markieren oder --roi-finder auto verwenden")
    return select_best_box([best_overall], width, height)


def main():
    ap = argparse.ArgumentParser(description=__doc__,
                                  formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--video", required=True)
    ap.add_argument("--model", default=None, help="Pfad zur .onnx-Datei (sonst Standardordner)")
    ap.add_argument("--confidence", type=float, default=0.35)
    args = ap.parse_args()

    try:
        x, y, w, h = find_roi(args.video, model_path=args.model,
                               confidence_threshold=args.confidence)
    except (ModelUnavailable, RuntimeError) as exc:
        print(f"KI-Regionssuche fehlgeschlagen: {exc}", file=sys.stderr)
        sys.exit(1)

    print(f"ROI {x} {y} {w} {h}")
    print(f"KI-Region gefunden: x={x} y={y} w={w} h={h}", file=sys.stderr)


if __name__ == "__main__":
    main()
