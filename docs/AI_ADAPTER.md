# Lokale KI als Adapter auf dem bestehenden System

Diese Datei hält fest, wie eine lokale KI in den Generator einzieht, ohne
das messbare, nicht-KI-basierte System zu ersetzen - Auftrag aus dem Chat
vom 14. September 2026: "der Generator soll mit lokaler KI laufen und dafür
die Ergebnisse/Werkzeuge des Systems ohne KI nutzen, das sollte
zusammenarbeiten."

## Grundsatz

Die KI schlägt vor, das bestehende System misst. Das ist keine neue Regel,
sondern die bereits in `docs/TEAM_STAND.md` (Abschnitt 8) und
`quality_model.py` gelebte Linie: ein gelerntes Modell wird nur benutzt,
wenn es in Kreuzvalidierung besser abschneidet als die festen Regeln, und
der Mensch bestätigt am Ende. Für die KI-Erweiterung gilt dasselbe:

- Die KI liefert einen **Vorschlag** (Region, Profil, Urteil).
- Der Vorschlag läuft durch **dieselbe** klassische Pipeline (CSRT/Flow-
  Tracking, Quality Doctor, Mapper) wie ein von Hand markierter Vorschlag -
  es gibt keinen zweiten, KI-eigenen Ausgabepfad, der am Rest vorbei ein
  `.funscript` erzeugt.
- Schlägt die KI fehl (kein Modell, keine Erkennung, Server nicht
  erreichbar), fällt das System auf das nicht-KI-Verfahren zurück, so wie
  `track_by_scenes()` heute schon jeden `roi_finder`-Fehler auffängt und die
  vorherige Region weiterverwendet.
- Beide Engines sind **optional** und laufen **lokal** - kein Modell im
  Repository, kein automatischer Download, keine Telemetrie. Wer sie nicht
  installiert/startet, merkt nichts von ihrer Existenz.

## Zwei Engines, zwei Aufgaben

Auf Wunsch ausdrücklich **nicht** über eine Gateway-Infrastruktur (Bifrost/
LM Studio), sondern zwei eigenständige, leichtgewichtige lokale Engines -
jede für die Aufgabe, für die sie gebaut ist:

| Engine | Aufgabe | Warum diese |
|---|---|---|
| **ONNX Runtime** | Bildverarbeitung: Region vorschlagen (später: Profil aus Bewegungssignatur) | Klein (wenige MB Modell + Laufzeit), CPU/GPU, kein Python-Zwang zur Laufzeit - passt zum Ziel "Python irgendwann aus der .exe werfen" (HANDOFF.md, Abschnitt "Später") |
| **Colibri** ([JustVugg/colibri](https://github.com/JustVugg/colibri)) | Große lokale Sprachmodelle für Urteile mit Begründung (Qualität, Profilwahl in Textform) | Reines C, kein CUDA/PyTorch nötig, läuft auf normaler Hardware, `coli serve` spricht die OpenAI-`/v1/chat/completions`-API - ein simpler HTTP-Client genügt als Anbindung, kein SDK |

Beide sind austauschbar: das Backend-Register (`backends.py`) und der
`roi_finder`-Injektionspunkt in `track_by_scenes()` sind genau für diesen
Zweck da - eine Funktion mit festem Vertrag anmelden, an der Pipeline ändert
sich nichts.

## Woher die Idee "ONNX wie FunGen 2" kommt - und was NICHT übernommen wurde

FunGen 2 ist geschlossenes Binary (PolyForm Strict für FunGen 1), siehe
`docs/HANDOFF.md`, Abschnitt "Fremder Code". Es wurde kein Code gelesen.
Öffentlich beschrieben (fungen.app, GitHub-README) ist: FunGen erkennt
Objekte per YOLO-Modell und trackt darauf aufbauend. Übernommen ist davon
ausschließlich die **Idee** "Erkennung schlägt Region vor, Tracking misst
weiter" - der Modellvertrag in `ai_roi.py` (Eingabe `1x3xHxW`, Ausgabe
`(N,6)` normalisiert) ist eine eigene, unabhängig entworfene Schnittstelle,
kein Nachbau eines bestimmten Exportformats.

## Reihenfolge (mit dem Nutzer abgestimmt: ROI → Profil → Qualität)

### 1. Region vorschlagen - **umgesetzt**

- `generator/ai_roi.py`: `find_roi(video, start_frame, end_frame)` mit
  demselben Vertrag wie `auto_roi.find_roi` - beide sind gegeneinander
  austauschbare `roi_finder`.
- CLI: `--per-scene-roi --roi-finder ai --ai-model-path modell.onnx`
  (Standard bleibt `auto`, also unverändertes Verhalten ohne die neue
  Option).
- `decode_detections()` und `select_best_box()` sind reine Funktionen und
  ohne Modell/onnxruntime testbar (`generator/ai_roi_test.py`).
- `onnxruntime` steht in `generator/requirements-ai.txt`, NICHT in
  `requirements.txt`.
- **Noch offen:** kein mitgeliefertes/empfohlenes ONNX-Modell, keine
  GUI-Anbindung (Go-seitig wäre das eine Kopie von `FindROIWithProgress` in
  `generator/generator.go` mit `ai_roi.py` statt `auto_roi.py` als
  Zielskript - der Rückgabevertrag `ROI x y w h` auf stdout ist bereits
  identisch gehalten).

### 2. Profil vorschlagen (standard/tf/tj/…) - **offen**

Setzt auf der bereits vorhandenen, aber unverdrahteten
Bewegungssignatur auf (`HANDOFF.md`: "Noch nicht verdrahtet: Benennen in der
Oberfläche und die Übernahme der Parameter"). Zwei mögliche Wege, noch nicht
entschieden:

- Rein aus den acht Signatur-Kennzahlen per kleinem lokalem Modell
  (bliebe bei ONNX, gleiche Engine wie Schritt 1).
- Oder als Textbegründung über Colibri ("diese Szene ähnelt der als 'tj'
  benannten Szene X, weil …") - das würde die Erklärbarkeit, die
  `quality_model.py` durch den Verzicht auf neuronale Netze bewusst
  erkauft, auf andere Weise zurückbringen: nicht lesbare Gewichte, aber ein
  Modell, das seine Entscheidung in Prosa begründen kann.

### 3. Qualitätsurteil - **offen**

Ergänzt, nicht ersetzt `quality_model.py`. Gleiche Sicherung wie dort
bereits vorhanden: ein KI-Urteil wird nur übernommen, wenn es die
Leave-one-out-Kreuzvalidierung gegen die festen Regeln UND das bestehende
gelernte Modell schlägt (`MIN_SAMPLES`/`MIN_PER_CLASS` als Vorbild). Über
Colibris `/v1/chat/completions` ließe sich zusätzlich eine
Klartext-Begründung einholen, die im Messbericht neben der Zahl steht.

## Nicht Teil dieser Änderung

- Kein Akt-Detektor - unverändert Grundsatz aus `docs/TEAM_STAND.md`.
- Kein Zwang, KI zu installieren - beide Engines bleiben Zusatzangebote wie
  der Intiface-Weg beim Gerät (`device/intiface.go`): eine zweite
  Möglichkeit, kein Ersatz für den bestehenden Weg.
- Keine Cloud-Anbindung, kein Gateway - beide Engines laufen ausschließlich
  lokal auf `127.0.0.1`.
