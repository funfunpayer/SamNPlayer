# Team-Stand und nächste Änderungen

Stand: 13. September 2026.
Geschrieben nach Durchsicht des Repos `funfunpayer/SamNPlayer` (Branch `main`),
der vorhandenen HANDOFF/README und öffentlicher FunGen-2-Beschreibungen
(Website, Changelog, README). **Kein FunGen-Quellcode gelesen, nichts übernommen.**

Diese Datei ist die gemeinsame Arbeitsspur. Code-Änderungen gehören in Branches
und Pull Requests, nicht nur in den Chat.

---

## Ziel

Besser als FunGen 2 **auf dem SVAKOM Sam Neo 2**, nicht Feature-Parität.

Zuerst klassisch (gemessen). KI später als Vorschlag auf denselben
Zwischenständen (Tracks, Signaturen, Urteile, Rezepte).

Kein Akt-Detektor in der ersten Stufe. Der Nutzer benennt
(`titjob` / `blowjob`); das Programm überträgt Parameter auf ähnliche
Bewegungssignaturen.

FunGen 1 steht unter PolyForm Strict. Dessen Code nicht lesen, nicht
strukturgleich nachbauen. Öffentlich beschriebenes *Verhalten* darf Vorbild
sein.

---

## Was schon gut ist (nicht umbauen)

- Schichten: `device` / `funscript` / `player` / `generator` / GUI.
- BLE-Protokoll gegen Buttplug-Rust festgehalten (`device/protocol.go`).
- `Sync()` an der Videouhr, Skript-Offset an einer Stelle.
- Generator-Optionen mit Messung begründet (HANDOFF).
- Zwei-Punkt-Abstand (`--roi2`), Bewegungssignatur, Quality Doctor.
- MIT-Lizenz. `update.RepoOwner` / `RepoName` stehen auf `funfunpayer/SamNPlayer`.

---

## Was sich ändern soll – Reihenfolge

Nicht UI und nicht Titjob zuerst. Der Go-Kern ist der wunde Punkt fürs
Gerätegefühl.

### 1. BLE-Schreibweg (zuerst)

Dateien: `device/samneo2.go`, Tests dazu.

Heute: jedes Tick `SetVibration` + `SetSuction` = zwei GATT-Writes mit
Response (bis ~40/s bei 50 ms). Keepalive speichert nur `lastPacket` – der
andere Kanal wird in Pausen nicht gehalten.

Soll:

- Gerätezustand `{vibration, suction}` halten.
- Keepalive **beide** Kanäle wiederholen.
- Unveränderte Stufe nicht nochmal senden.
- Writes zusammenfassen, wo das Protokoll das hergibt.

Ohne das bleiben Rezepte und Profile Theorie.

### 2. Mapper testbar + Rezeptfelder

Dateien: `funscript/mapper.go`, neue `funscript/mapper_test.go`,
`funscript/funscript.go` (Metadata).

Heute: `ToIntensityCurve` ohne eigene Tests. `MinVibration` wirkt auf das
Tempo-Signal. Es gibt keinen `suction_floor` / Peak-Gate.

Soll: Tests für Independent, Stillstand, schnellen Hub. Danach optionale
Felder für Geräterezepte (siehe unten). Metadata `profile` und
`device_recipe` durchreichen, Player liest sie.

### 3. Generator-Embed vollständig

Datei: `generator/generator.go` (`writeScriptToTemp`).

Heute landen in der Temp-Kopie nur `generate_funscript.py` und
`quality_doctor.py`. `flow_backend`, `quality_model` und weitere Imports
fehlen. Im Repo beim Entwickeln oft unsichtbar, in der **fertigen .exe**
bricht `--backend flow` und das gelernte Modell.

Soll: alle Module einbetten, die der Generator zur Laufzeit importiert.

### 4. Profile `titjob` / `blowjob`

Kein Detektor. Parameterpaket + GUI-Karten + zweite ROI.

Generator-Startwerte (nach Hardware nachziehen):

| | standard | titjob (Peak+Grund) | titjob_peak | blowjob |
|---|---|---|---|---|
| Signal | 1 ROI | Abstand ROI1–ROI2 Pflicht | wie titjob | Achse Mund/Kopf, optional Abstand zur Basis |
| pos_floor / pos_ceil | 5–95 | 20–90 | 15–92 | 8–95 |
| hold_contact | aus | an | an | aus |
| norm_window_s | 6 | 4 | 4 | 5 |
| Sog-Quelle | Position | invertierter Abstand + Boden 0.20 | nur pos > 70 | Tiefe, Tal darf 0 |

`MapOptions`-Startwerte:

| | standard | titjob | titjob_peak | blowjob |
|---|---|---|---|---|
| Sync | independent | independent | independent | independent |
| TickMs | 50 | 50 | 50 | 40 |
| MaxSpeed | 0.60 | 0.50 | 0.55 | 0.70 |
| MinVibration | 0.15 | 0.20 | 0.12 | 0.10 |
| Smoothing | 0.30 | 0.22 | 0.18 | 0.20 |
| suction_floor | 0 | 0.20 | 0 | 0 |
| suction_peak_gate | aus | aus | an | aus |

Default Titjob: **Peak plus Grunddruck**. Variante nur Peak als Checkbox.

Profilname intern `titjob` / `blowjob`. Keine stillen Akt-Klassen.

### 5. Doctor darf ändern

Quality Doctor prüft. FunGen-öffentlich: Doctor **repariert** (Tempo, Lücken).
Bei uns zuerst: Mindestabstand und Speed-Cap schon erzwingen (teilweise da),
dann ein expliziter Fix-Schritt mit Diff „vorher/nachher“.

### 6. Mini-Editor, nicht OFS-Klon

Nach dem Generieren: Kurve sehen, Peaks ziehen, Abschnitt löschen/wiederholen,
Offset (habt ihr). Geist der letzten Erzeugung über der Kurve reicht.
Kein Multi-Axis-Studio für OSR.

### 7. UI (danach)

Dunkles Layout, Icon-Leiste, Video groß, Kurve+Heatmap unten, Gerät rechts
immer sichtbar. Vibration Amber, Sog Violett. Generator: drei Karten
Standard / Busen / Oral.

### 8. Später

- Kapitel aus vorhandenen Scene-Cuts, Abschnitt neu erzeugen.
- Audio-Takt als Fallback, wenn das Bild lügt (Idee aus FunGen-Changelog,
  eigene Implementierung).
- Python aus der .exe werfen (`videox` verdrahten, wenn der Vorteil gemessen
  ist).
- KI nur als Adapter: schlägt ROI/Profil vor, Mensch bestätigt. Muss die
  festen Regeln in Kreuzvalidierung schlagen.

---

## Von FunGen 2 öffentlich gelernt (Verhalten, kein Code)

Quellen: fungen.app, GitHub-README/Changelog von ack00gar/FunGen (Binary),
nicht das archivierte FunGen-1-Python-Repo.

Übernehmen als *Idee*:

- Erzeugen → sehen → flicken → auf **dieses** Gerät spielen.
- Latenz / Interpolation pro Gerät.
- Provenienz in der Datei (`creator`, Herkunft, bei uns zusätzlich Rezept).
- Tracking-Cache nach außen: Parameter ändern, Video nicht nochmal.
- Kapitel/Marker als Schnitt.

Nicht übernehmen:

- YOLO-/Körperteil-Pipeline, VR-Pro-Modelle.
- Sechs Achsen für OSR.
- Stash/XBVR als erstes Produktziel.
- Strukturgleiche Nachbildung ihrer Tracker.

---

## Testclips

Synthetische Clips (kein Adult-Content, bekannte Wahrheit):

- Hub still / mit Schwenk
- Zwei-Punkt-Abstand (Titjob-Physik)
- Takt mit Pausen (Blowjob-Physik)
- Szenenwechsel, Verdeckung, wechselnde Amplitude

Echte Titjob-/Blowjob-Takes dreht das Team selbst (15–40 s, H.264, erst
Stativ). Urteil in den Messbericht, Vergleich gegen ein Fremdskript desselben
Films wenn vorhanden.

---

## Zusammenarbeit auf GitHub

Repo: https://github.com/funfunpayer/SamNPlayer

- `main` bleibt lauffähig.
- Arbeit an `feat/…` oder `fix/…`.
- Jede Code-Änderung als Pull Request, auch kleine.
- In den PR: was gemessen wurde, nicht nur was sich anfühlt.
- HANDOFF.md nur ändern, wenn der *Code* sich ändert.
- Diese Datei aktualisieren, wenn die Reihenfolge oder ein Rezept kippt.

Wenn hier (Grok) Code geschrieben wird, kommt er in einen Branch und einen PR
in genau diesem Repo – nicht nur als Chat-Snippet.
