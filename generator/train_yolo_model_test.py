"""Test für train_yolo_model.py.

Läuft OHNE echtes Training (kein GPU-Zeitbudget, kein Netzzugriff für
Gewichte-Downloads) - geprüft werden die beiden Teile, die ohne ein
laufendes Training testbar UND es wert sind: der Fail-Fast-Schutz bei
fehlendem Datensatz (feuert VOR dem teuren YOLO(base_model)-Aufruf, siehe
train_and_export) und dass main() die CLI-Argumente mit den richtigen
Standardwerten an train_and_export durchreicht - eine GUI/CLI-Option, die
dort nicht ankommt, ist schlimmer als keine (vgl. args_test.go).

Ausführen: python3 generator/train_yolo_model_test.py
"""

import os
import sys
import tempfile

import train_yolo_model


def main():
    failures = []

    def check(name, cond, detail=""):
        print(("  OK   " if cond else "  FAIL ") + name + (f"  [{detail}]" if not cond else ""))
        if not cond:
            failures.append(name)

    # --- train_and_export: klare Fehlermeldung statt stillem Ultralytics-Rauschen ---
    with tempfile.TemporaryDirectory() as tmp:
        try:
            train_yolo_model.train_and_export(tmp, os.path.join(tmp, "out.onnx"))
            check("wirft bei fehlender data.yaml", False)
        except RuntimeError as exc:
            check("wirft RuntimeError bei fehlender data.yaml", "data.yaml" in str(exc), str(exc))
            check("Fehlermeldung verweist auf bootstrap_yolo_dataset.py",
                  "bootstrap_yolo_dataset.py" in str(exc), str(exc))

    # --- main(): CLI-Argumente erreichen train_and_export mit den erwarteten Werten ---
    captured = {}

    def fake_train_and_export(dataset_dir, output_path, epochs=100, device="cuda",
                               imgsz=640, base_model="yolov8n.pt", project_dir=None,
                               run_name="samnplayer_roi"):
        captured.clear()
        captured.update(dataset_dir=dataset_dir, output_path=output_path, epochs=epochs,
                         device=device, imgsz=imgsz, base_model=base_model)

    orig = train_yolo_model.train_and_export
    train_yolo_model.train_and_export = fake_train_and_export
    try:
        sys.argv = ["train_yolo_model.py", "--dataset-dir", "/tmp/ds", "--output", "/tmp/out.onnx"]
        train_yolo_model.main()
        check("Pfade kommen unverändert an",
              captured.get("dataset_dir") == "/tmp/ds" and captured.get("output_path") == "/tmp/out.onnx",
              str(captured))
        check("Standard-Epochen 100", captured.get("epochs") == 100, str(captured))
        check("Standard-Gerät cuda (Nvidia-GPU vorausgesetzt)", captured.get("device") == "cuda", str(captured))
        check("Standard-Bildgröße 640", captured.get("imgsz") == 640, str(captured))
        check("Standard-Basismodell yolov8n.pt (kleinstes/schnellstes)",
              captured.get("base_model") == "yolov8n.pt", str(captured))

        sys.argv = ["train_yolo_model.py", "--dataset-dir", "/tmp/ds", "--output", "/tmp/out.onnx",
                    "--epochs", "5", "--device", "cpu", "--imgsz", "320", "--base-model", "yolov8s.pt"]
        train_yolo_model.main()
        check("--epochs überschreibt den Standardwert", captured.get("epochs") == 5, str(captured))
        check("--device überschreibt den Standardwert", captured.get("device") == "cpu", str(captured))
        check("--imgsz überschreibt den Standardwert", captured.get("imgsz") == 320, str(captured))
        check("--base-model überschreibt den Standardwert",
              captured.get("base_model") == "yolov8s.pt", str(captured))
    finally:
        train_yolo_model.train_and_export = orig

    print(("FEHLGESCHLAGEN: " + ", ".join(failures)) if failures else "Alle Prüfungen bestanden.")
    return 1 if failures else 0


if __name__ == "__main__":
    sys.exit(main())
