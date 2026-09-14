# Mitarbeiten

Diese Datei beschreibt, wie mehrere Leute an SamNPlayer arbeiten können,
ohne sich gegenseitig zu blockieren. Sie ist kurz gehalten – alles, was
darüber hinausgeht, steht in `HANDOFF.md` (was das Projekt ist und warum es
so gebaut ist) und `WIEDERAUFNAHME.md` (wie man die Umgebung aufsetzt).

## Wo lässt sich parallel arbeiten

Die Schnitte sind so gewählt, dass mehrere Leute sich nicht ins Gehege
kommen. Wer an einem dieser Bereiche arbeitet, fasst die anderen nicht an:

| Bereich | Was dort passiert | Berührt |
|---|---|---|
| `device/` | BLE-Protokoll, Intiface, Geräteansteuerung | nichts anderes |
| `generator/*.py` | Analyse, Signalverarbeitung, Qualitätsbewertung | nur Python |
| `cmd/gui-wails/frontend/` | Oberfläche | nur JS/CSS |
| `player/` | Wiedergabe, Synchronisation, Training | Go-Kern |
| `motionx/`, `videox/` | eigenständige Hilfspakete | nichts anderes |

**Neue Analyseverfahren brauchen keinen Eingriff in die Pipeline.** Dafür
gibt es das Backend-Register (`generator/backends.py`): eine Funktion
schreiben, mit `register()` anmelden, fertig. Zwei Leute können so
gleichzeitig an verschiedenen Verfahren arbeiten, ohne dieselbe Datei zu
berühren. Der Vertrag steht im Kopf von `backends.py`.

## Nach dem Klonen

`cmd/gui-wails` bindet `frontend/dist` per `go:embed` ein. Das Verzeichnis
ist ein Bauartefakt und liegt nicht im Repository – ein frischer Klon
scheitert deshalb zunächst an

```
pattern all:frontend/dist: no matching files found
```

Das ist kein Fehler, sondern ein fehlender erster Schritt:

```bash
cd cmd/gui-wails/frontend && npm install && npm run build && cd ../../..
go build ./...
```

Danach läuft alles wie gewohnt. `wails build` erledigt diesen Schritt
ohnehin mit.

## Ablauf

1. Branch von `main`, benannt nach dem Bereich: `device/keepalive-fix`,
   `generator/zwei-punkt-auto`.
2. Ändern, **Tests dazuschreiben**, lokal laufen lassen.
3. Pull Request. Die Prüfung in `.github/workflows/tests.yml` läuft
   automatisch – Go mit Race-Detector, Python, Oberfläche.
4. Jemand anderes schaut drüber. Bei Änderungen an `device/` möglichst
   jemand mit echter Hardware.

Direkt auf `main` zu schieben ist technisch möglich, aber in einem Team der
schnellste Weg zu einem Stand, den niemand mehr nachvollziehen kann.

## Was ein Beitrag mitbringen muss

**Einen Test, der ohne die Änderung fehlschlägt.** Das ist die einzige
Regel, die hier wirklich zählt. Zweimal wurde in diesem Projekt eine
Funktion gebaut, die nachweislich gar nichts tat – einmal landete eine
Einfügung in der falschen Datei, einmal wirkte ein Parameter nirgends.
Beides fiel nur auf, weil gemessen wurde. Also: Test schreiben, grün sehen,
Änderung testweise zurücknehmen, **rot sehen**, Änderung wiederherstellen.

**Zahlen statt Einschätzungen.** Wer behauptet, etwas sei besser, belegt es
mit einer Messung im Kommentar oder im Test. Mehrfach hat sich eine
plausible Erklärung als falsch erwiesen – die Kamerakompensation war nicht
kaputt, sondern das Testmaterial; die Glättung fraß keine Amplitude; ein
naheliegender zusätzlicher Schätzer machte das Ergebnis schlechter.

**Auch negative Ergebnisse dokumentieren.** `HANDOFF.md` hat den Abschnitt
"Geprüft und verworfen" mit den jeweiligen Messwerten. Wer eine Idee
verwirft, trägt sie dort ein – sonst probiert sie in drei Monaten jemand
erneut.

**Nichts einbauen, was nicht gebraucht wird.** Zwei Module liegen derzeit
ungenutzt im Baum (`fusion.py`, `videox/`), beide mit begründetem Vermerk,
warum sie noch nicht verdrahtet sind. Das ist die Ausnahme, nicht das
Muster.

## Fallstricke, die Zeit gekostet haben

- **Testvideos mit Rauschhintergrund** taugen nicht für Kamerakompensation:
  `goodFeaturesToTrack` findet dort keine stabilen Merkmale, und die
  Schätzung wird zum Zufallsprozess. Immer texturierte Hintergründe.
- **Bewegte Objekte im Testvideo** müssen in Weltkoordinaten liegen, sonst
  machen sie einen Kameraschwenk nicht mit – und die Kompensation sieht
  fälschlich kaputt aus.
- **`TRACK_CACHE_VERSION` erhöhen**, wenn sich `track_roi` oder die
  Kamerakompensation ändert. Sonst liefert der Cache still Ergebnisse der
  alten Fassung.
- **`wails build` nach jeder neuen Go-Methode**, sonst fehlen die Bindings
  und die Frontend-Tests scheitern mit einer irreführenden Meldung.
- **`-trimpath` beim Bauen.** Ohne das landet der Pfad des Build-Rechners –
  und damit der Benutzername – in der fertigen Datei.

## Fremder Code

Vor jeder Übernahme aus einem anderen Projekt die Lizenz prüfen. Die Tabelle
in `HANDOFF.md` hält den Stand fest. Besonders: **FunGen steht unter
PolyForm Strict** – keine kommerzielle Nutzung und keine abgeleiteten Werke.
Den Quelltext zu lesen und Funktionen strukturgleich zu übertragen wäre ein
abgeleitetes Werk. Ideen und beschriebenes Verhalten sind dagegen frei.

## Versionen

`VERSION` und `update.BaseVersion` müssen übereinstimmen. Ein Release
entsteht durch einen Tag `vX.Y.Z`; der Workflow baut dann die Binaries und
hängt sie an den Release, gegen den die Selbstaktualisierung prüft.
