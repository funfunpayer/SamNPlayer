# SamNPlayer

Steuert einen SVAKOM Sam Neo 2 / Sam Neo 2 Pro per BLE anhand einer
`.funscript`-Datei. Läuft als CLI (`cmd/cli`) oder native Desktop-GUI
(`cmd/gui-wails`, gebaut mit [Wails](https://wails.io) - Go-Backend +
schlankes Vanilla-JS-Frontend über die im Betriebssystem bereits vorhandene
Web-Engine: WebView2 unter Windows 11 bereits eingebaut, WebKitGTK unter
Linux, WKWebView unter macOS. Kein Electron, keine gebündelte
Browser-Engine).

## Fertige Windows-.exe

Fertige Binaries für Windows und Linux hängen an jedem
[Release](https://github.com/funfunpayer/SamNPlayer/releases) - inklusive
`checksums.txt`. Die GUI-.exe ist rund 13 MB, deutlich schlanker als
klassische Electron-Apps, da keine eigene Browser-Engine mitgeliefert werden
muss. Herunterladen und doppelklicken, keine Installation nötig.

Binaries werden bewusst **nicht** im Repository mitgeführt (siehe
`.gitignore`): sie ändern sich bei jedem Build vollständig und werden vom
Release-Workflow ohnehin automatisch gebaut.

**Bluetooth-Adapter unter Windows 11:** laut den Buttplug/Intiface-Entwicklern
(dieselbe Community, aus der das Geräteprotokoll stammt) empfohlen: **TP-Link
UB500** oder **Asus USB-BT500**. Eingebaute Mainboard-/Laptop-Bluetooth-Radios
ohne externe Antenne werden für Windows/Linux explizit *nicht* empfohlen.
Quelle: https://intiface.com/docs/intiface-central/hardware/bluetooth/

## Video-Wiedergabe mit echtem Sync

Der Wiedergabe-Tab bettet ein echtes `<video>`-Element ein (kein separates
Fenster, kein VLC-Prozess) und sucht automatisch nach einer gleichnamigen
Videodatei neben dem gewählten Funscript (`szene.mp4` + `szene.funscript`,
dieselbe Konvention wie bei MultiFunPlayer/ScriptPlayer). Mit aktiviertem
"Gerät folgt der echten Videoposition" treibt die tatsächliche
Video-Abspielposition (nicht eine eigene Uhr) die Geräte-Ausgabe an
(`player.Sync()`, siehe `player/sync.go`) - Pausieren, Spulen und die
Video-eigene Geschwindigkeit wirken sich direkt und korrekt aus. Extended-O
pausiert dabei optional auch das Video für die Haltezeit.

Technischer Hinweis, falls du selbst daran weiterbaust: eine direkte
`file://`-URL im `<video>`-Tag wird von WebView2/WebKitGTK aus
Cross-Origin-Gründen abgelehnt (getestet, nicht angenommen). Die App startet
darum beim ersten Laden eines Videos einen winzigen lokalen HTTP-Server
(nur `127.0.0.1`, liefert ausschließlich die aktuell gewählte Datei mit
Range-Request-Unterstützung fürs Spulen) und bindet darüber ein.

## Selbst bauen

Braucht Node.js/npm (fürs Frontend-Bundling) zusätzlich zu Go.

```
go build ./cmd/cli                              # CLI
cd cmd/gui-wails && wails build -tags webkit2_41 # GUI (Linux/macOS)
```

Das `webkit2_41`-Tag ist unter Ubuntu 24.04+ nötig, weil dort nur noch
WebKitGTK 4.1 (nicht mehr 4.0) verfügbar ist - auf älteren Systemen ohne
diesen Tag bauen.

Cross-Build nach Windows von Linux/Mac aus (so wurde die mitgelieferte .exe
gebaut) - für den Windows-Zielcode selbst wird kein C-Compiler gebraucht
(reines Go), nur für lokale Linux-GUI-Abhängigkeiten:

```
cd cmd/gui-wails
GOOS=windows GOARCH=amd64 wails build -platform windows/amd64 \
  -ldflags "-X github.com/funfunpayer/SamNPlayer/update.Version=v0.1.0"
```

## Releases & Auto-Update

`.github/workflows/release.yml` baut bei jedem Tag `vX.Y.Z` automatisch
Windows- und Linux-Binaries (GUI + CLI) und veröffentlicht sie als GitHub
Release inkl. `checksums.txt`. Die App selbst (`update/update.go`) prüft im
Einstellungen-Tab konfigurierbar beim Start dagegen und kann sich bei
Zustimmung selbst herunterladen (mit SHA256-Verifikation gegen die
Prüfsummen-Datei und einer Herkunfts-Prüfung, dass die Download-URL
tatsächlich von github.com kommt) und neu starten.

Das Release-Repository ist bereits auf `funfunpayer/SamNPlayer` eingestellt.
[v0.2.1](https://github.com/funfunpayer/SamNPlayer/releases/tag/v0.2.1)
wurde am 14. September 2026 erfolgreich für Windows und Linux gebaut.
Für kommende Releases gilt der Ablauf in [CONTRIBUTING.md](CONTRIBUTING.md#versionen).

## Skript aus Video erzeugen (eigener Generator)

Tab "Skript erzeugen": Video wählen, im ersten Frame (per `<canvas>`, Maus
ziehen) eine Region über das zu verfolgende Motiv markieren, generieren.
Nutzt klassisches CV-Tracking (OpenCV CSRT-Tracker + Savitzky-Golay-Glättung
+ Peak-Erkennung, `generator/generate_funscript.py`) - **kein** trainiertes
KI-Modell, dafür komplett selbst enthalten (Skript ist per `go:embed` im
Go-Binary, keine externe Installation nötig außer Python selbst).

**Braucht Python 3.9+** mit den Paketen aus `generator/requirements.txt`
(`pip install -r generator/requirements.txt`, oder der Button
"Abhängigkeiten prüfen" sagt genau, was fehlt).

**Für deutlich bessere Ergebnisse** (KI-Objekterkennung statt manueller
ROI, VR-Support, automatische Szenenerkennung, ganze Ordner auf einmal):
[FunGen 2](https://fungen.app) - kostenlos für Privatgebrauch, läuft lokal.
FunGen ist **nicht Open Source** (PolyForm Strict License), darum keine
Integration hier - aber dessen Ausgabe ist eine ganz normale
`.funscript`-Datei, die dieser Player direkt abspielt.

## Logging

Strukturiertes Logging (`logging/`, `log/slog`) schreibt in
`<Nutzerkonfigurationsordner>/SamNPlayer/logs/SamNPlayer.log`
(Windows: `%AppData%\SamNPlayer\logs\`), Rotation bei 5MB. Level im
Einstellungen-Tab umschaltbar (debug/info/warn/error), wirkt sofort und
bleibt über Neustarts erhalten.

## Einstellungen (persistiert)

Eigene schlanke JSON-Persistenz (`settings.json` im selben Ordner wie die
Logdatei - Wails hat kein eingebautes Preferences-API wie manche andere
GUI-Toolkits). Bleibt über Programmstarts erhalten:
- Automatischer Update-Check beim Start (an/aus)
- Log-Level
- Alle Wiedergabe-Standardwerte (Mock/Sync-Modus/Tick/Max-Speed/Extended-O)

## Neu: Community-inspirierte Verbesserungen

Nach Durchsicht des größten Community-Players (MultiFunPlayer) ergänzt:

- **Skript-Heatmap**: farbcodierte Intensitätsleiste (blau=ruhig, rot=
  intensiv) über die ganze Skriptlänge, direkt unter der Videovorschau.
- **Soft-Start**: beim Loslegen wird nicht sofort auf den vollen
  Skriptwert gesprungen, sondern in konfigurierbarer Zeit (Standard 500ms)
  sanft hochgefahren.
- **Glättung im Wiedergabe-Tab einstellbar** - vorher nur im Generator
  vorhanden, jetzt auch beim Abspielen selbst tunbar (0 = aus).
- **Tastenkürzel**: Leertaste = Abspielen/Stop, E = Extended-O auslösen.
- **Sync-Modi "nur Vibration" / "nur Sog"** (`funscript/mapper.go`): für
  fremde/importierte Skripte, die gezielt nur auf einen Kanal gelegt
  werden sollen, statt immer beide zu nutzen.

Sowohl Vibration als auch Sog sind beim Sam Neo 2 stufenlos steuerbar -
nachgeprüft im Buttplug-Rust-Quellcode (`OutputCommand::Constrict` nutzt
denselben generischen Skalar-Werttyp wie `OutputCommand::Vibrate`, nur mit
6 statt 11 Stufen). Eine frühere Annahme hier, der Sog sei auf 5 feste
Rhythmus-Muster begrenzt, war falsch (beruhte auf einem Forenbericht zur
*offiziellen SVAKOM-App* mit eigener Preset-UI, nicht auf dem rohen
Protokoll) und wurde wieder entfernt - keine künstliche Verzögerung mehr
beim Sog, er verhält sich jetzt genau wie die Vibration.

## Trainings-Modus

Eigener Tab, unabhängig von Video/Skript - eigenständige Auf/Ab-Zyklen zum
Ausdauer-/Kontrolltraining mit dem Gerät (`player/training.go`). Zwei
Techniken, beide klinisch/community-beschrieben, nicht selbst erfunden:
- **Stop-Start**: hochfahren, kurz halten, komplett auf ~0, Pause, von
  vorn. Die klassische Methode (Semans, 1950er).
- **Plateau/Edging**: bleibt nach dem Hochfahren auf einem hohen Niveau
  statt ganz abzufallen.

Beide mit konfigurierbarer Zyklenzahl, Zeiten, optionaler Steigerung von
Zyklus zu Zyklus, und frei wählbarem Kanal (Vibration, Sog, oder beide -
beide sind stufenlos steuerbar, siehe oben). Jede Session wird komplett
mitgeschrieben: `<Log-Ordner>/sessions/training-<technik>-<zeitstempel>.jsonl`
(ein JSON-Objekt pro Zyklus: Zeitstempel, Kanal, Spitzenwert, Haltezeit,
Dauer) - Grundlage für spätere Feinabstimmung.

## Markierter Bereich + automatisches Extended-O

Auf der Heatmap-Leiste im Wiedergabe-Tab lässt sich per Maus-Ziehen ein
Zeitbereich markieren (z.B. der Höhepunkt gegen Ende). Mit aktivierter
Checkbox "Extended-O automatisch im markierten Bereich auslösen" springt
Extended-O von selbst an, sobald die Wiedergabe (Video-Sync oder eigene
Uhr) diesen Bereich erreicht - einmal pro Wiedergabe. Die Markierung liegt
als kleine JSON-Datei neben dem Skript (`szene.funscript.marker.json`,
nicht im funscript selbst, um andere Player nicht zu stören) und lässt
sich jederzeit über "Markierung löschen" wieder entfernen.

## Stabilität (Fixes für den produktiven Einsatz)

- **Kein stilles Überschreiben im Generator**: liegt neben dem Video
  bereits ein `.funscript` (von Hand erstellt, heruntergeladen oder aus
  einem früheren Lauf), fragt der Generator jetzt nach, statt es
  kommentarlos zu ersetzen. Beim Testen war genau das passiert - ein
  Datenverlust, der sich nicht rückgängig machen lässt.
- **Tastaturfokus nach Datei-Dialogen**: nach dem Schließen des nativen
  Auswahl-Dialogs blieb der Fokus am Button hängen, wodurch die
  Tastenkürzel (Leertaste/E) scheinbar wirkungslos waren. Wird jetzt
  aktiv ins Fenster zurückgeholt.
- **Wiedergabe/Training schließen sich gegenseitig aus**: beide steuern
  dasselbe physische Gerät - ohne Absicherung hätte ein gleichzeitiger
  Start auf beiden Tabs dazu führen können, dass die zuerst gestartete
  Sitzung nicht mehr stoppbar war (überschriebene Abbruchfunktion). Jetzt
  meldet die zweite Sitzung einen klaren Fehler, statt die erste zu
  verwaisen. Live getestet: Training mit langer Haltezeit gestartet,
  während es lief Wiedergabe versucht - korrekt abgelehnt; nach Stop lief
  eine neue Sitzung sofort wieder an.
- **Update-Erkennung GUI vs. CLI**: `AssetForThisPlatform()` unterschied
  nur nach Betriebssystem/Architektur, nicht nach Programmvariante - unter
  Windows enden aber sowohl die GUI- als auch die CLI-Datei auf
  "-windows-amd64.exe". Die GUI hätte sich beim Update abhängig von der
  Reihenfolge der Server-Antwort versehentlich durch die CLI-Datei
  ersetzen können. Jetzt wird explizit nach "gui"/"cli" unterschieden.

## Automatische Regionssuche (Generator)

Statt die Bewegungsregion von Hand zu markieren, kann der Generator sie
selbst finden (`generator/auto_roi.py`, Button "Region automatisch
finden"). Das Verfahren entspricht Abschnitt 5-6 der Projekt-Dokumentation:

1. Dichter Optical Flow (Farneback) über ein Raster von Bildzellen.
2. **Kamerabewegungs-Kompensation**: die globale Bewegung wird per
   Feature-Tracking + affinem Modell (RANSAC) geschätzt und vom Flussfeld
   abgezogen - ohne das würde ein Kameraschwenk als "Bewegung überall"
   erscheinen und die Regionswahl wäre Zufall.
3. Bewertung nicht nach *stärkster*, sondern nach **periodischster**
   Bewegung: per FFT wird gemessen, wie stark eine einzelne Frequenz im
   Band 0.1-4 Hz dominiert. Rhythmisches gewinnt gegen zufälliges Zappeln.
4. Die besten Zellen werden zu einer zusammenhängenden Region vereinigt.

Bewusst **ohne trainiertes Erkennungsmodell**: der Ansatz sucht Rhythmus,
nicht bestimmte Objekte. Dadurch ist er inhaltsunabhängig, braucht keine
Modellgewichte im Programm und funktioniert auch bei Material, für das
kein passendes Modell existiert. Getestet gegen Videos mit gezielten
Störern (statisches Objekt, zufällig zappelndes Objekt, Kameraschwenk) -
in allen Fällen wurde die rhythmisch bewegte Region gefunden.

Das Ergebnis ist ein Vorschlag: die gefundene Region lässt sich im
Vorschaubild weiterhin von Hand korrigieren.

## Bekannte Grenzen

- **Sam-Neo-2-Protokoll** ist gegen den offiziellen Buttplug-Rust-Quellcode
  verifiziert (siehe Kommentare in `device/protocol.go`), aber nicht an
  echter Hardware getestet.
- **Video-Codec:** `<video>` spielt nur ab, was die jeweilige Webview-Engine
  nativ kann (H.264/MP4, WebM/VP9, AV1 - deckt praktisch alles ab, was
  gängige Kamera-/Konvertierungs-Software erzeugt). Exotische alte Codecs
  (z.B. MPEG-4 Part 2 / "mp4v", Xvid) werden nicht abgespielt - mit
  `ffmpeg -c:v libx264` neu kodieren, falls das mal vorkommt.
