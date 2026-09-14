# WIEDERAUFNAHME

Diese Datei ist für den Fall gedacht, dass die Arbeitsumgebung verloren geht
– das ist während der Entwicklung bereits einmal passiert: das gesamte
Arbeitsverzeichnis war weg, einschließlich der Go-Installation. Sie steht
bewusst getrennt von `HANDOFF.md`: dort steht, *was* das Projekt ist und
warum es so gebaut ist, hier steht, *wie man weitermacht*.

Die Quellversion steht in `VERSION` und `update.BaseVersion`; den zuletzt
veröffentlichten Stand zeigen die GitHub-Releases. Offene Arbeit steht
nur in [docs/NEXT.md](docs/NEXT.md).

## Umgebung wiederherstellen

Quellcode aus `https://github.com/funfunpayer/SamNPlayer` klonen. Für einen
exakten veröffentlichten Stand den entsprechenden Tag auschecken. Eine ZIP
ist ein ergänzendes Backup, GitHub enthält die nachvollziehbare Historie.
Vor dem Weiterarbeiten `git status`, Branch, letzte Commits und `docs/NEXT.md`
prüfen. Die ausführbaren Programme gibt es unter den GitHub-Releases.

Voraussetzungen:

- Go gemäß `go.mod` (aktuell mindestens 1.25.0; automatischer Toolchain-Download
  kann für Abhängigkeiten eine neuere Fassung benötigen).
- Node.js ab 22.12 mit npm für das Vite-Frontend.
- Python mit Paketen aus `generator/requirements.txt`; CI nutzt Python 3.12.
- Windows: WebView2 für die GUI. Ubuntu 24.04: `libgtk-3-dev` und
  `libwebkit2gtk-4.1-dev`, beim Wails-Build `-tags webkit2_41` verwenden.

Im Repository die zum Modul passende Wails-CLI installieren (PowerShell):

```powershell
$wailsVersion = go list -m -f '{{.Version}}' github.com/wailsapp/wails/v2
go install "github.com/wailsapp/wails/v2/cmd/wails@$wailsVersion"
python -m pip install -r generator/requirements.txt
Push-Location cmd/gui-wails/frontend
npm install
npm run build
Pop-Location
go vet ./...
go test ./...
```

`wails` liegt danach im `bin`-Ordner von `go env GOPATH`; diesen dem PATH
hinzufügen. Im frischen Klon muss `frontend/dist` vor den Go-Prüfungen
gebaut werden, weil `go:embed` das Verzeichnis benötigt.

## Vollständiger Testdurchlauf

Es gibt keinen einzelnen Befehl dafür – bewusst, weil die drei Gruppen
unterschiedlich lange brauchen (folgende Befehle für Bash):

```bash
# Go: Sekunden
go vet ./... && go test ./...

# Python: die Videotests brauchen je 1-3 Minuten
(cd generator && for t in *_test.py; do echo "== $t"; python3 "$t" || exit 1; done)

# Frontend: braucht Headless-Chromium
for t in cmd/gui-wails/frontend/test/*_test.py; do python3 "$t" || exit 1; done
```

Die Frontend-Tests erzeugen ihre Attrappen automatisch aus
`frontend/wailsjs/go/main/App.js`. Fehlt dort eine neue Go-Methode, weil
`wails build` seit der Änderung nicht lief, scheitern sie mit *"does not
provide an export named …"* – das ist kein Testfehler, sondern der Hinweis,
dass die Bindings neu erzeugt werden müssen.

---

## Bauen und ausliefern

```bash
cd cmd/gui-wails && wails build -platform windows/amd64 -trimpath
cd ../..
cp cmd/gui-wails/build/bin/SamNPlayer.exe dist/SamNPlayer-gui-windows-amd64.exe
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath \
  -o dist/SamNPlayer-cli-windows-amd64.exe ./cmd/cli
```

`-trimpath` ist nicht optional: ohne das landet der Pfad des Build-Rechners
– und damit der Benutzername – in der fertigen Datei.

Packen für die Übergabe:

```bash
zip -qr SamNPlayer.zip SamNPlayer \
  -x "*/node_modules/*" "*/build/bin/*" "*/__pycache__/*"
```

---

## Arbeitsweise, die sich bewährt hat

Diese Punkte sind nicht Stilfragen – jeder davon steht für einen Fehler, der
tatsächlich passiert ist.

**Messen statt annehmen.** Jede Behauptung über Qualität oder Geschwindigkeit
wird belegt. Mehrfach hat sich dabei eine plausible Erklärung als falsch
erwiesen: die Kamerakompensation war nicht kaputt (das Testmaterial war es),
die Glättung fraß keine Amplitude, und die Divergenz als Schätzer war
schlechter statt besser.

**Gegenprobe bei jedem Test.** Nach dem Grünwerden die Änderung testweise
zurücknehmen und prüfen, dass der Test rot wird. Zweimal wäre sonst eine
Funktion ausgeliefert worden, die gar nichts tat – einmal landete eine
Einfügung sogar in der Testdatei statt in der Pipeline.

**Negative Ergebnisse dokumentieren.** `HANDOFF.md` hat den Abschnitt
"Geprüft und verworfen" mit den Messwerten. Ohne ihn wird dieselbe Sackgasse
erneut betreten.

**Testmaterial ist eine Fehlerquelle.** Videos mit Rauschhintergrund taugen
nicht für Kamerakompensation (keine verfolgbaren Merkmale). Bewegte Objekte
müssen in Weltkoordinaten liegen, sonst machen sie einen Schwenk nicht mit.
Und synthetische Sinusvideos haben die Rhythmusmessung jahrelang falsch
aussehen lassen, weil echtes Material ständig das Tempo wechselt.

**Zwei Zahlen im Kopf behalten**, die gegen Selbsttäuschung helfen: Der
Kalibrierungssatz besteht aus fünf gültigen und vier wertlosen Videos – nach
jeder Änderung an der Bewertung müssen die fünf bestehen und die vier
durchfallen. Und der Cache macht Wiederholungen 36-mal schneller; wer ohne
ihn misst, wartet unnötig.

---

## Weiterarbeiten

[docs/NEXT.md](docs/NEXT.md) enthält den verifizierten Ausgangsstand,
Prioritäten und Abnahmekriterien. Vorhandene Hardware- und Qualitätsgrenzen
stehen in `HANDOFF.md`. Neue Arbeitsstände nicht zusätzlich hier pflegen.
