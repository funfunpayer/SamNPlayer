#!/usr/bin/env python3
"""train_yolo_model.py - trainiert ein YOLO-Modell auf einem mit
bootstrap_yolo_dataset.py gesammelten (und idealerweise von Hand
kontrollierten) Datensatz, und exportiert es direkt im Vertrag, den
ai_roi.py erwartet.

Schließt die Lücke zwischen den beiden bisherigen Werkzeugen:
bootstrap_yolo_dataset.py baut den Datensatz, export_yolo_onnx.py
normalisiert ein FERTIG trainiertes .pt - das eigentliche `yolo detect
train` dazwischen musste bisher von Hand auf der Kommandozeile laufen.
Dieses Skript ruft beides hintereinander auf, damit "Video(s) rein, Modell
raus" ein einziger Schritt ist (Chat, 16. September 2026: "brauchen wir
Trainingssystem fest drin").

GPU vorausgesetzt für echte Datensätze (--device cuda) - CPU-Training ist
laut export_yolo_onnx.py/requirements-ai-train.txt für echte Datensätze sehr
langsam, bleibt hier aber als Option verfügbar (--device cpu), falls jemand
nur an einem winzigen Testdatensatz prüfen will, dass der Ablauf überhaupt
durchläuft.

ultralytics gibt während des Trainings selbst laufend Fortschritt aus
(Epoche für Epoche, Verlustwerte) - dieses Skript leitet das unverändert
weiter (kein eigenes PROGRESS-Format), die GUI zeigt es als Rohtext-Log an,
genau wie beim Bench-Tab.

Nutzung:
  python3 train_yolo_model.py --dataset-dir ./yolo_dataset \
      --output roi_detector.onnx --epochs 100 --device cuda
"""

import argparse
import os
import sys


def available():
    """True, wenn ultralytics installiert ist. Reine Prüffunktion, löst
    nichts aus - die GUI nutzt sie, um den Training-Knopf zu aktivieren/
    auszublenden, statt ihn anzubieten und dann mit einem rohen
    ModuleNotFoundError-Traceback (Temp-Skriptpfad, kein Hinweis auf den
    fehlenden pip install) scheitern zu lassen. Siehe ai_roi.available()
    für dasselbe Muster bei der kleineren onnxruntime-Abhängigkeit."""
    try:
        import ultralytics  # noqa: F401
    except ImportError:
        return False
    return True


def train_and_export(dataset_dir, output_path, epochs=100, device="cuda",
                      imgsz=640, base_model="yolov8n.pt", project_dir=None,
                      run_name="samnplayer_roi"):
    """Trainiert base_model auf dataset_dir/data.yaml und exportiert das
    Ergebnis als normalisiertes ONNX nach output_path. Wirft, statt ein
    kaputtes/fehlendes Modell stillschweigend als Erfolg zu melden - der
    Aufrufer (GUI) soll einen echten Fehler sehen, keinen leeren Erfolg."""
    try:
        from ultralytics import YOLO
    except ImportError as exc:
        # Die GUI prüft das vorher per available()/--check und sperrt den
        # Knopf entsprechend - dieser Zweig ist nur für den direkten
        # CLI-Aufruf ohne diese Prüfung (z.B. von Hand auf der
        # Kommandozeile), damit auch der einen klaren Hinweis statt eines
        # rohen ModuleNotFoundError-Tracebacks bekommt.
        raise RuntimeError(
            "ultralytics ist nicht installiert - für das Training nötig "
            "(nicht Teil der Basis-Installation, siehe requirements-ai-train.txt): "
            "pip install -r generator/requirements-ai-train.txt") from exc

    data_yaml = os.path.join(dataset_dir, "data.yaml")
    if not os.path.isfile(data_yaml):
        raise RuntimeError(
            f"Keine data.yaml unter {dataset_dir} - erst bootstrap_yolo_dataset.py "
            "laufen lassen, um den Datensatz anzulegen.")

    project_dir = project_dir or os.path.join(dataset_dir, "runs")
    print(f"Training: {base_model} auf {data_yaml}, {epochs} Epochen, device={device}",
          file=sys.stderr)
    model = YOLO(base_model)
    model.train(data=data_yaml, epochs=epochs, device=device, imgsz=imgsz,
                project=project_dir, name=run_name, exist_ok=True)

    weights_path = os.path.join(project_dir, run_name, "weights", "best.pt")
    if not os.path.isfile(weights_path):
        raise RuntimeError(
            f"Training abgeschlossen, aber keine best.pt unter {weights_path} gefunden - "
            "vermutlich ist das Training fehlgeschlagen, siehe Log oberhalb.")

    import export_yolo_onnx
    print(f"Exportiere {weights_path} -> {output_path}...", file=sys.stderr)
    export_yolo_onnx.export_and_normalize(weights_path, output_path, imgsz=imgsz)
    if not export_yolo_onnx.verify(output_path, imgsz):
        print("WARNUNG: Verifikation des exportierten Modells unerwartet - "
              "manuell prüfen, bevor es produktiv verwendet wird.", file=sys.stderr)
    print(f"Fertig: {output_path}", file=sys.stderr)
    return output_path


def main():
    ap = argparse.ArgumentParser(description=__doc__,
                                  formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--dataset-dir", help="Von bootstrap_yolo_dataset.py angelegter Ordner "
                                          "(enthält data.yaml) - nicht nötig mit --check")
    ap.add_argument("--output", help="Zielpfad für die fertige .onnx-Datei - nicht nötig mit --check")
    ap.add_argument("--epochs", type=int, default=100)
    ap.add_argument("--device", default="cuda", help="'cuda', 'cpu', oder eine GPU-Nummer wie '0'")
    ap.add_argument("--imgsz", type=int, default=640)
    ap.add_argument("--base-model", default="yolov8n.pt",
                     help="Ultralytics-Basismodell - 'n' (nano) ist der kleinste/schnellste, "
                          "reicht für ein einzelnes ROI-Erkennungsproblem üblicherweise aus")
    ap.add_argument("--check", action="store_true",
                     help="Nur prüfen, ob Training grundsätzlich möglich ist (ultralytics "
                          "installiert), ohne etwas zu trainieren - für die GUI, um den "
                          "Training-Knopf zu aktivieren/auszublenden.")
    args = ap.parse_args()

    if args.check:
        print("AVAILABLE" if available() else "UNAVAILABLE")
        return

    if not args.dataset_dir or not args.output:
        ap.error("--dataset-dir und --output sind erforderlich, außer bei --check")

    train_and_export(args.dataset_dir, args.output, epochs=args.epochs,
                      device=args.device, imgsz=args.imgsz, base_model=args.base_model)


if __name__ == "__main__":
    main()
