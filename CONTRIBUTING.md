# Mitarbeiten

Diese Datei beschreibt, wie mehrere Leute an SamNPlayer arbeiten können,
ohne sich gegenseitig zu blockieren. Sie ist kurz gehalten – alles, was
darüber hinausgeht, steht in `HANDOFF.md` (was das Projekt ist und warum es
so gebaut ist) und `WIEDERAUFNAHME.md` (wie man die Umgebung aufsetzt).

## Welche Anleitung ist maßgeblich?

- `README.md`: Installation, Bedienung und Einstieg.
- `HANDOFF.md`: Architektur, belegte Ergebnisse, Grenzen und verworfene Ansätze.
- `WIEDERAUFNAHME.md`: Arbeitsumgebung wiederherstellen, bauen und prüfen.
- `docs/NEXT.md`: einzige operative Aufgabenliste mit Prioritäten und Abnahmekriterien.

Vor Arbeitsbeginn aktuellen Code, `git status`, `docs/NEXT.md` und den
zugehörigen PR prüfen. Alte Chatprotokolle sind Kontext; dort genannte
Fehler, Berechtigungen und erledigte Arbeiten am aktuellen Stand verifizieren.
Bei Widersprüchen zählen Code und reproduzierbare Tests. Messwerte ohne
Hardwarebeleg nicht als getestete Geräteeigenschaft darstellen.

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

1. Arbeitsziel und Abnahme festhalten: in `docs/NEXT.md` oder einem verlinkten
   Issue. Einen überschaubaren Punkt übernehmen; betroffene Bereiche nennen.
2. Aktuelles `main` holen und eigenen Branch erstellen, für Codex-Arbeit
   beispielsweise `codex/project-workflow-cleanup`. Vorher lokale Änderungen
   prüfen; fremde Änderungen weder überschreiben noch zurücksetzen.
3. Änderung umsetzen und passende Tests ausführen. Bei Verhaltensänderungen
   einen Regressionstest mit Gegenprobe ergänzen. Reine Dokumentationsänderungen
   brauchen keine künstlichen Tests; Angaben, Links und Befehle prüfen.
4. Pull Request mit Problem, Ergebnis, ausgeführten Prüfungen und offenen
   Einschränkungen erstellen. Keine leeren Dateien oder Platzhalter als
   Zwischenlösung veröffentlichen. Vor dem Commit den vollständigen Diff prüfen.
5. CI abwarten und Review einholen. Bei Änderungen an `device/` einen
   Hardwaretest vorsehen; Mock-Tests ausdrücklich als solche benennen.
6. Nach dem Merge `docs/NEXT.md` abgleichen. Erledigte Arbeiten mit PR oder
   Commit belegen; neue Erkenntnisse zur Architektur in `HANDOFF.md` eintragen.
   Alte Branches erst löschen, wenn ihre Änderungen nachweislich integriert sind.

## Was ein Beitrag mitbringen muss

**Bei Verhaltensänderungen einen Test, der ohne die Änderung fehlschlägt.**
Zweimal wurde in diesem Projekt eine
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

`VERSION` und `update.BaseVersion` müssen dieselbe Version ohne `v` enthalten.
`go test ./update` prüft diese Übereinstimmung. Entwicklungs-Binaries zeigen
`BaseVersion-dev`; Release-Binaries erhalten die Tag-Version über `-ldflags`.

Vor einem neuen Release:

1. Beide Versionsangaben im selben PR erhöhen, relevante Dokumentation und
   `docs/NEXT.md` aktualisieren und die CI erfolgreich abschließen.
2. Den geprüften Commit auf `main` mit `vX.Y.Z` taggen und genau diesen Tag pushen.
3. Der Release-Workflow prüft vor dem Build Tag, `VERSION` und `BaseVersion`.
   Bei Abweichungen bricht er ab. Go und die Wails-CLI kommen aus `go.mod`.
4. Workflow-Ergebnis, vier Binaries (GUI/CLI für Windows/Linux) und
   `checksums.txt` kontrollieren. Ein erfolgreicher Build ersetzt keinen
   Hardwaretest und keinen Test der Selbstaktualisierung am installierten Programm.

Veröffentlichte Tags nicht nachträglich verschieben. Die Korrektur des
Quellstands auf 0.2.1 verändert das bereits veröffentlichte Release nicht.
