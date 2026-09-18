# KI-Trainingssystem — Anleitung (was / wo / wie)

Stand: September 2026 · SamNPlayer **0.5.5+**

Dieses Dokument erklärt das **Regions-Modell** (YOLO → ONNX), nicht das
Geräte-„Training“ im Tab Training (Stop-Start/Plateau).

## Kurz: Was die KI darf — und was nicht

| Darf | Darf nicht |
|------|------------|
| Region(en) vorschlagen | Funscript schreiben |
| Box in der GUI setzen | Tracking ersetzen |
| Lokal auf deinem Rechner lernen | Cloud / Telemetrie / mitgeliefertes Modell |

Klassisches Tracking (CSRT / Flow / …) schreibt immer das `.funscript`.
Siehe auch `docs/AI_ADAPTER.md` und `docs/FUNSCRIPT_ALGOS.md`.

## Voraussetzungen

1. **Python 3** (dasselbe, das SamNPlayer findet — Windows: `py`/`python`)
2. **Basis-CV:** `opencv-contrib-python` (nicht nur `opencv-python`) — für
   Bootstrap (Video abtasten). OpenCV 5: CSRT oft weg → automatischer
   KCF-Fallback (#95).
3. **Training:** `ultralytics` + `onnx` (zieht PyTorch nach)
4. **Inferenz (Erzeugen-Tab):** `onnxruntime`
5. Optional: **ffmpeg** für Still-Konvertierung / Audio-Sidecar / Player-Proxy

### Installation (GUI, empfohlen)

Im Tab **KI-Trainingssystem** → **Abhängigkeiten installieren**.

Das schreibt die im Binary eingebettete `requirements-ai-train.txt` und ruft
`python -m pip install -r …` auf — funktioniert auch ohne Quellbaum
(Release-.exe).

### Installation (manuell)

```bash
# Inferenz
pip install -r generator/requirements-ai.txt

# Training (+ PyTorch)
pip install -r generator/requirements-ai-train.txt

# Windows ohne NVIDIA (optional):
pip install torch-directml
```

Aus dem Binary: Einstellungen / oder später „Requirements exportieren“ —
`WriteAIRequirementFiles(zielordner)`.

## Schritt-für-Schritt (GUI)

1. Tab **KI-Trainingssystem**
2. **Video wählen** (oder **Bild** für Still-Annotation)
3. Bei schwarzem Intro: Zeit vorstellen → Frame laden
4. Bis zu **4 Boxen** ziehen, Klassen benennen (`hand`, `brust`, …)
5. **Für Training verwenden** → CSRT/KCF trackt und schreibt YOLO-Samples
6. **Kontrollansicht:** schlechte Samples **verwerfen**, Boxen
   **korrigieren** (wichtig — Bootstrap-Boxgröße ist fest, Tracker kann
   danebenliegen)
7. **Training starten** (Epochen, Gerät Auto/CUDA/DirectML/CPU)
8. Ergebnis: `roi_detector.onnx` unter dem Modellpfad (Einstellungen)
9. Tab **Erzeugen** → Häkchen **KI-Erkennung (ONNX)** → Region finden lassen
   → Box prüfen → Funscript generieren

## Ordner

| Was | Typischer Pfad |
|-----|----------------|
| Datensatz | `%AppData%/SamNPlayer/roi_training_dataset` bzw. `~/.config/SamNPlayer/…` |
| Modell | `%LOCALAPPDATA%/SamNPlayer/models/roi_detector.onnx` |
| Zwischengewichte | `<dataset>/runs/samnplayer_roi/weights/best.pt` |
| Audio (optional) | `<dataset>/audio/*.wav` — noch nicht im Training genutzt |

## CLI (Quellbaum)

```bash
python generator/bootstrap_yolo_dataset.py \
  --video clip.mp4 --roi x,y,w,h --class-name hand \
  --output-dir ~/.config/SamNPlayer/roi_training_dataset

python generator/train_yolo_model.py \
  --dataset-dir ~/.config/SamNPlayer/roi_training_dataset \
  --output ~/.config/SamNPlayer/models/roi_detector.onnx \
  --epochs 100 --device auto
```

## Bild-Erkennung: Was wir nutzen / was nicht

| Technik | Status |
|---------|--------|
| YOLOv8n → ONNX ROI-Vorschlag | **Produktpfad** |
| Klassisches CSRT/KCF/Flow | **schreibt Funscript** |
| Zwei-ROI-KI (`find_two_rois`) | Code da, real kaum gemessen |
| Audio-Sidecar | nur abgelegt |
| 3D / Depth / Pose | **nicht** — erst wenn Golden-Clips zeigen, dass es die Funscript-Qualität hebt |

## Tipps für gute Modelle

- Viele kurze Clips statt eines langen
- Boxen in der Kontrollansicht korrigieren (Bootstrap hält feste w/h)
- Optional **Box scale** 1.1–1.2 für etwas Polster um die Markierung
- Mindestens ein Val-Beispiel (Still-Pfad wechselt automatisch)
- GPU stark empfohlen; CPU nur für Smoke-Tests (wenige Epochen)
- Nach Training: Erzeugen-Tab → KI-Erkennung; in Settings **Preferred classes**
  setzen bei Mehrklassen-Modellen (z.B. `hand,breast`)
- 3D / Depth / Pose: **nicht** — erst wenn Golden-Clips Funscript-Gewinn zeigen

## Fehlerbilder

| Symptom | Ursache / Fix |
|---------|----------------|
| „ultralytics fehlt“ | Abhängigkeiten installieren |
| Bootstrap scheitert OpenCV 5 | `opencv-contrib-python`; KCF-Fallback prüfen |
| Training: leeres val | mind. 2 Stills oder Video-Bootstrap |
| KI-Häkchen grau | kein `.onnx` unter Modellpfad |
| Schlechte Vorschläge | mehr/korrigierte Samples; Preferred classes; nicht „größeres YOLO“ |
| Preferred class ignoriert | `classes.json` neben `.onnx` fehlt (neu trainieren oder kopieren) |

## Verwandte Dateien

- `cmd/gui-wails/frontend/src/roi_training.js` — GUI
- `cmd/gui-wails/app_roi_training.go` — Wails-API
- `generator/roi_training.go`, `bootstrap_yolo_dataset.py`, `train_yolo_model.py`, `ai_roi.py`
- `docs/AI_ADAPTER.md` — Architekturprinzip
