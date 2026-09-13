# WIEDERAUFNAHME

Diese Datei ist für den Fall gedacht, dass die Arbeitsumgebung verloren geht
– das ist während der Entwicklung bereits einmal passiert: das gesamte
Arbeitsverzeichnis war weg, einschließlich der Go-Installation. Sie steht
bewusst getrennt von `HANDOFF.md`: dort steht, *was* das Projekt ist und
warum es so gebaut ist, hier steht, *wie man weitermacht*.

**Fassung: 0.2.0** (siehe Datei `VERSION` und `update.BaseVersion`)

---

## Sofort nach einem Verlust

Die letzte ausgelieferte ZIP ist die Sicherung. Entpacken, dann:

```bash
# Go ist in einer frischen Umgebung meist nicht vorhanden
curl -sL https://go.dev/dl/go1.23.4.linux-amd64.tar.gz -o /tmp/go.tar.gz
tar -C /usr/local -xzf /tmp/go.tar.gz
export PATH=$PATH:/usr/local/go/bin
export GOTOOLCHAIN=auto

pip install opencv-contrib-python scipy numpy
go install github.com/wailsapp/wails/v2/cmd/wails@v2.10.2
export PATH=$PATH:$(go env GOPATH)/bin
```

Danach **erst prüfen, was tatsächlich da ist**, bevor irgendetwas gebaut
wird:

```bash
go vet ./... && go test ./...
ls generator/*.py | wc -l      # sollte über 20 sein
```

Ein häufiger Irrtum an dieser Stelle: der Zustand *sieht* vollständig aus,
weil die Verzeichnisse existieren. Der Test ist der Beweis, nicht die
Verzeichnisliste.

---

## Vollständiger Testdurchlauf

Es gibt keinen einzelnen Befehl dafür – bewusst, weil die drei Gruppen
unterschiedlich lange brauchen:

```bash
# Go: Sekunden
go vet ./... && go test ./...

# Python: die Videotests brauchen je 1-3 Minuten
cd generator && for t in *_test.py; do echo "== $t"; python3 "$t"; done

# Frontend: braucht Headless-Chromium
for t in cmd/gui-wails/frontend/test/*_test.py; do python3 "$t"; done
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

## Woran zuletzt gearbeitet wurde

Reihenfolge aus einer externen Code-Durchsicht, die sich als zutreffend
erwiesen hat:

1. ~~BLE: Zustand beider Kanäle, Keepalive für beide, weniger Writes~~ **erledigt**
2. ~~Mapper-Tests und Grundintensität als Skalierung statt Abschneiden~~ **erledigt**
3. ~~Alle Python-Module einbetten~~ **erledigt** – erklärte zwei gemeldete Fehler
4. Rezeptfelder für Bewegungsarten im Mapper, dann Oberfläche

---

## Was aussteht und nur der Anwender liefern kann

Diese Punkte blockieren echten Fortschritt und lassen sich nicht durch mehr
Code ersetzen:

- **Hardwaretest.** Verbindung, Wiedergabe und Training sind ausschließlich
  gegen `device.Mock` und einen nachgebauten Buttplug-Server geprüft.
- **Rohwert-Auflösung des Geräts.** Die Bereiche 0–10 und 0–5 stammen aus
  der Buttplug-Konfiguration, nicht aus einer Untersuchung der Firmware.
  Der Rohwert-Test im Geräte-Tab klärt das in fünf Minuten.
- **Echte Videos mit Urteilen.** Sämtliche Qualitätsschwellen und das
  lernende Modell beruhen auf synthetischen Sinusvideos. Der Messbericht
  sammelt bereits alles Nötige; es fehlen die Urteile.
- **Ein 20-Sekunden-Ausschnitt** des Films, zu dem beide Skripte vorliegen.
  Damit ließe sich der verbleibende Intensitätsunterschied zu FunGen in
  einem Durchgang klären.
