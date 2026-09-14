"""Test für ai_roi.py - den optionalen ONNX-ROI-Vorschlag.

Läuft komplett OHNE onnxruntime und OHNE ein echtes Modell: decode/select
sind reine Funktionen, und find_roi() nimmt über _run_model_fn eine
Attrappe entgegen (derselbe Injektionstrick, den track_by_scenes() für
roi_finder selbst schon nutzt). Damit ist die Nachverarbeitung geprüft,
ohne Modellgewichte ins Repository legen oder herunterladen zu müssen.

Ausführen: python3 generator/ai_roi_test.py
"""

import sys
import tempfile
from pathlib import Path

import cv2
import numpy as np

import ai_roi

W, H, FPS, FRAMES = 320, 240, 25, 50
# Objekt bei 70% Bildbreite, 40% Bildhöhe - so lässt sich prüfen, dass die
# normalisierten Modellkoordinaten korrekt in Pixel des ECHTEN Frames
# umgerechnet werden, nicht nur der quadratischen Modelleingabe.
OBJECT_NORM = (0.60, 0.30, 0.80, 0.50)  # x0,y0,x1,y1


def write_video(path):
    vw = cv2.VideoWriter(str(path), cv2.VideoWriter_fourcc(*"mp4v"), FPS, (W, H))
    rng = np.random.default_rng(1)
    bg = np.full((H, W, 3), 40, np.uint8)
    for _ in range(60):
        x, y = int(rng.integers(0, W)), int(rng.integers(0, H))
        cv2.rectangle(bg, (x, y), (x + 12, y + 12), (90, 90, 90), -1)
    for _ in range(FRAMES):
        vw.write(bg)
    vw.release()


def main():
    failures = []

    def check(name, cond, detail=""):
        print(("  OK   " if cond else "  FAIL ") + name + (f"  [{detail}]" if not cond else ""))
        if not cond:
            failures.append(name)

    # --- decode_detections: Schwelle und entartete Boxen --------------------
    raw = np.array([
        [0.1, 0.1, 0.2, 0.2, 0.90, 0],   # klar über der Schwelle
        [0.3, 0.3, 0.4, 0.4, 0.10, 0],   # unter der Schwelle -> raus
        [0.5, 0.5, 0.4, 0.4, 0.99, 0],   # x1<x0 und y1<y0 -> entartet, raus
    ])
    detections = ai_roi.decode_detections(raw, confidence_threshold=0.35)
    check("Schwelle filtert niedrige Konfidenz", len(detections) == 1, str(detections))
    check("entartete Box wird verworfen",
          all(d["confidence"] != 0.99 for d in detections), str(detections))

    # --- decode_detections: Reshape akzeptiert flache Eingabe ---------------
    flat = ai_roi.decode_detections([0.1, 0.1, 0.2, 0.2, 0.9, 0], confidence_threshold=0.5)
    check("akzeptiert eine einzelne flache Box (reshape (-1,6))", len(flat) == 1, str(flat))

    # --- select_best_box: höchste Konfidenz gewinnt, korrekte Pixelumrechnung
    dets = [
        {"x0": 0.1, "y0": 0.1, "x1": 0.2, "y1": 0.2, "confidence": 0.5, "class_id": 0},
        {"x0": 0.60, "y0": 0.30, "x1": 0.80, "y1": 0.50, "confidence": 0.9, "class_id": 0},
    ]
    box = ai_roi.select_best_box(dets, frame_w=1000, frame_h=1000)
    check("wählt die Box mit höherer Konfidenz", box == (600, 300, 200, 200), str(box))

    # --- select_best_box: Mindestgröße und Bildgrenzen -----------------------
    tiny = [{"x0": 0.50, "y0": 0.50, "x1": 0.501, "y1": 0.501, "confidence": 0.9, "class_id": 0}]
    box_tiny = ai_roi.select_best_box(tiny, frame_w=100, frame_h=100, min_size_px=16)
    check("erzwingt Mindestgröße", box_tiny[2] >= 16 and box_tiny[3] >= 16, str(box_tiny))

    edge = [{"x0": 0.95, "y0": 0.95, "x1": 1.0, "y1": 1.0, "confidence": 0.9, "class_id": 0}]
    box_edge = ai_roi.select_best_box(edge, frame_w=100, frame_h=100, min_size_px=16)
    check("bleibt am Bildrand innerhalb der Grenzen",
          box_edge[0] + box_edge[2] <= 100 and box_edge[1] + box_edge[3] <= 100, str(box_edge))

    check("keine Detektion -> None statt Absturz",
          ai_roi.select_best_box([], 100, 100) is None, "")

    # --- available(): ehrliche Antwort ohne onnxruntime/Modell ---------------
    check("available() meldet False ohne Modell/Laufzeit unter erfundenem Pfad",
          ai_roi.available(model_path="/nicht/vorhanden.onnx") is False, "")

    # --- find_roi() Ende-zu-Ende mit injizierter Modell-Attrappe --------------
    with tempfile.TemporaryDirectory() as tmp:
        video = Path(tmp) / "object.mp4"
        write_video(video)

        def fake_model(frame_bgr):
            x0, y0, x1, y1 = OBJECT_NORM
            return np.array([[x0, y0, x1, y1, 0.95, 0.0]])

        x, y, w, h = ai_roi.find_roi(str(video), report_progress=False,
                                      _run_model_fn=fake_model)
        expected = ai_roi.select_best_box(
            ai_roi.decode_detections([[*OBJECT_NORM, 0.95, 0.0]]), W, H)
        check("find_roi liefert die vom Modell vorgeschlagene Box in Pixelkoordinaten",
              (x, y, w, h) == expected, f"{(x, y, w, h)} != {expected}")

        def no_detection_model(frame_bgr):
            return np.zeros((0, 6))

        raised = False
        try:
            ai_roi.find_roi(str(video), report_progress=False,
                             _run_model_fn=no_detection_model)
        except RuntimeError:
            raised = True
        check("find_roi wirft einen klaren Fehler ohne Detektion (kein stiller Fallback)",
              raised, "")

        # start_frame/end_frame: derselbe Vertrag wie auto_roi.find_roi, damit
        # track_by_scenes() beide Finder austauschbar per Szene aufrufen kann.
        x2, y2, w2, h2 = ai_roi.find_roi(str(video), start_frame=5, end_frame=30,
                                          report_progress=False, _run_model_fn=fake_model)
        check("find_roi nimmt start_frame/end_frame wie auto_roi.find_roi entgegen",
              (x2, y2, w2, h2) == expected, f"{(x2, y2, w2, h2)} != {expected}")

    # --- ohne echtes Modell: klarer Fehler statt Absturz ----------------------
    raised = False
    try:
        ai_roi.find_roi("irrelevant.mp4", model_path="/nicht/vorhanden.onnx")
    except ai_roi.ModelUnavailable:
        raised = True
    check("find_roi ohne Modell wirft ModelUnavailable statt zu crashen", raised, "")

    print(("FEHLGESCHLAGEN: " + ", ".join(failures)) if failures else "Alle Prüfungen bestanden.")
    return 1 if failures else 0


if __name__ == "__main__":
    sys.exit(main())
