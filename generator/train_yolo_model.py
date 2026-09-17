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

Gerätewahl (Accelerator-Strategie, docs/perception_update_2026-09/05):
  --device auto|cuda|directml|mps|cpu  (Standard: auto)
  auto wählt der Reihe nach CUDA → MPS → DirectML → CPU.
  DirectML braucht unter Windows zusätzlich torch-directml
  (optional, nicht in requirements-ai-train.txt, weil Linux/macOS
  es nicht brauchen). CUDA bleibt auf NVIDIA die schnellste Option;
  DirectML deckt AMD/Intel unter Windows ohne CUDA-Zwang ab.

ultralytics gibt während des Trainings selbst laufend Fortschritt aus
(Epoche für Epoche, Verlustwerte) - dieses Skript leitet das unverändert
weiter (kein eigenes PROGRESS-Format), die GUI zeigt es als Rohtext-Log an,
genau wie beim Bench-Tab.

Nutzung:
  python3 train_yolo_model.py --dataset-dir ./yolo_dataset \
      --output roi_detector.onnx --epochs 100 --device auto
  python3 train_yolo_model.py --list-devices
"""

import argparse
import json
import os
import sys


DEVICE_CHOICES = ("auto", "cuda", "directml", "mps", "cpu")


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


def _cuda_available():
    try:
        import torch
        return bool(torch.cuda.is_available())
    except Exception:
        return False


def _mps_available():
    try:
        import torch
        return bool(getattr(torch.backends, "mps", None)
                    and torch.backends.mps.is_available())
    except Exception:
        return False


def _directml_device():
    """torch.device für DirectML, oder None wenn nicht nutzbar."""
    try:
        import torch_directml
        if torch_directml.device_count() <= 0:
            return None
        return torch_directml.device()
    except Exception:
        return None


def list_devices():
    """Liste der wählbaren Geräte inkl. Verfügbarkeit - für GUI/CLI."""
    cuda_ok = _cuda_available()
    mps_ok = _mps_available()
    dml_ok = _directml_device() is not None
    return [
        {"id": "auto", "label": "Automatisch (bestes verfügbares)", "available": True},
        {"id": "cuda", "label": "NVIDIA CUDA", "available": cuda_ok},
        {"id": "directml", "label": "DirectML (Windows, AMD/Intel/NVIDIA)", "available": dml_ok},
        {"id": "mps", "label": "Apple MPS", "available": mps_ok},
        {"id": "cpu", "label": "CPU (sehr langsam)", "available": True},
    ]


def resolve_device(requested="auto"):
    """Mappt GUI/CLI-Wahl auf das, was ultralytics model.train(device=…)
    erwartet. Wirft RuntimeError mit klarer Meldung, wenn eine explizite
    Wahl nicht verfügbar ist (statt still auf CPU zu fallen - sonst denkt
    man, DirectML/CUDA laufe, obwohl es die CPU ist)."""
    req = (requested or "auto").strip().lower()
    if req in ("", "auto"):
        if _cuda_available():
            print("Gerät auto → cuda", file=sys.stderr)
            return "0"
        if _mps_available():
            print("Gerät auto → mps", file=sys.stderr)
            return "mps"
        dml = _directml_device()
        if dml is not None:
            print("Gerät auto → directml", file=sys.stderr)
            return dml
        print("Gerät auto → cpu (kein GPU-Backend gefunden)", file=sys.stderr)
        return "cpu"

    if req in ("cuda", "gpu"):
        if not _cuda_available():
            raise RuntimeError(
                "CUDA wurde gewählt, ist aber nicht verfügbar "
                "(kein NVIDIA-Treiber / kein torch mit CUDA). "
                "Unter Windows ohne NVIDIA: --device directml oder auto.")
        return "0"

    if req == "mps":
        if not _mps_available():
            raise RuntimeError(
                "MPS wurde gewählt, ist aber nicht verfügbar "
                "(nur Apple Silicon mit passendem PyTorch).")
        return "mps"

    if req in ("directml", "dml"):
        dml = _directml_device()
        if dml is None:
            raise RuntimeError(
                "DirectML wurde gewählt, ist aber nicht verfügbar. "
                "Unter Windows: pip install torch-directml "
                "(zusätzlich zu requirements-ai-train.txt).")
        return dml

    if req == "cpu":
        return "cpu"

    # Numerische GPU-Indexe und andere ultralytics-kompatible Werte
    # unverändert durchreichen (z.B. "0", "1", "0,1").
    return requested


def train_and_export(dataset_dir, output_path, epochs=100, device="auto",
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

    resolved = resolve_device(device)
    project_dir = project_dir or os.path.join(dataset_dir, "runs")
    print(f"Training: {base_model} auf {data_yaml}, {epochs} Epochen, "
          f"device={device!r} → {resolved!r}",
          file=sys.stderr)
    model = YOLO(base_model)
    # batch=-1 lässt ultralytics die Batchgröße automatisch an die freie
    # GPU-Speichergröße anpassen (Ziel ~60% Auslastung), statt den festen
    # Standardwert (16) zu erzwingen. Ohne das läuft eine Karte mit wenig
    # VRAM (z.B. eine GTX 1650 mit 4GB) beim Training leicht in ein "CUDA
    # out of memory", obwohl yolov8n (der Standard hier) mit kleinerer
    # Batchgröße problemlos passen würde. Auf der CPU wirkungslos (fällt
    # dort automatisch auf den festen Standardwert zurück, siehe
    # ultralytics.utils.autobatch.autobatch).
    model.train(data=data_yaml, epochs=epochs, device=resolved, imgsz=imgsz,
                batch=-1, project=project_dir, name=run_name, exist_ok=True)

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
                                          "(enthält data.yaml) - nicht nötig mit --check/--list-devices")
    ap.add_argument("--output", help="Zielpfad für die fertige .onnx-Datei - nicht nötig mit --check/--list-devices")
    ap.add_argument("--epochs", type=int, default=100)
    ap.add_argument("--device", default="auto",
                     help="auto|cuda|directml|mps|cpu (oder GPU-Index wie '0') - Standard: auto")
    ap.add_argument("--imgsz", type=int, default=640)
    ap.add_argument("--base-model", default="yolov8n.pt",
                     help="Ultralytics-Basismodell - 'n' (nano) ist der kleinste/schnellste, "
                          "reicht für ein einzelnes ROI-Erkennungsproblem üblicherweise aus")
    ap.add_argument("--check", action="store_true",
                     help="Nur prüfen, ob Training grundsätzlich möglich ist (ultralytics "
                          "installiert), ohne etwas zu trainieren - für die GUI, um den "
                          "Training-Knopf zu aktivieren/auszublenden.")
    ap.add_argument("--list-devices", action="store_true",
                     help="Verfügbare Trainingsgeräte als JSON auf stdout "
                          "(für die GUI-Geräteauswahl).")
    args = ap.parse_args()

    if args.check:
        print("AVAILABLE" if available() else "UNAVAILABLE")
        return

    if args.list_devices:
        print(json.dumps(list_devices(), ensure_ascii=False))
        return

    if not args.dataset_dir or not args.output:
        ap.error("--dataset-dir und --output sind erforderlich, außer bei --check/--list-devices")

    train_and_export(args.dataset_dir, args.output, epochs=args.epochs,
                      device=args.device, imgsz=args.imgsz, base_model=args.base_model)


if __name__ == "__main__":
    main()
