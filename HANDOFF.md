# SamNPlayer – Projektübersicht

Diese Datei liegt bewusst **im Repository** und nicht als PDF daneben: sie
wandert mit dem Code mit und veraltet nicht getrennt von ihm. Wer hier etwas
ändert, ändert es zusammen mit der Änderung am Code.

Stand: September 2026

---

## Was das Projekt ist

Native Desktop-Anwendung (Go + Wails) zur Steuerung eines SVAKOM Sam Neo 2
per Bluetooth, synchron zu einer `.funscript`-Datei. Dazu ein Generator, der
`.funscript`-Dateien mit klassischer Bildverarbeitung aus Videos erzeugt –
ohne neuronale Netze.

Durchgehendes Prinzip: **Was nicht gebraucht wird, kommt nicht rein.** Und:
Jede Aussage über Qualität oder Geschwindigkeit wird gemessen, nicht
geschätzt.

**Stack:** Go ab 1.25.0 (siehe `go.mod`), Wails v2.15, Vanilla JS (kein Framework),
Python 3.9+ mit OpenCV/scipy/numpy (per `go:embed` eingebettet, zur Laufzeit
als Unterprozess gestartet).

---

## Aufbau

```
cmd/gui-wails/     Wails-GUI (Hauptprodukt), fünf Tabs
cmd/cli/           CLI-Variante (ohne Generator)
device/            BLE-Protokoll und -Transport, Mock-Gerät
player/            Wiedergabe, Synchronisation, Trainingsmodus
funscript/         Parser, Metadata, Mapping
generator/         Python-Pipeline + Go-Wrapper
motionx/           RDP-Reduktion und Bewegungszustands-Klassifikation (Salvage)
videox/            ffprobe/ffmpeg-Wrapper (Salvage, noch nicht verdrahtet)
logging/ update/   Protokollierung, Auto-Update über GitHub-Releases
```

---

## Was funktioniert

**Wiedergabe:** Skript laden, Video synchron abspielen, Funscript-Kurve mit
mitlaufendem Positionszeiger unter dem Video, Heatmap-Leiste, Markierungen.
Dazu **Skript-Offset** (je Skript gespeichert, wirkt sofort auch während der
Wiedergabe – fremde Skripte passen fast nie exakt zum eigenen Videoschnitt),
**Abschnittswiederholung** und Tastaturbedienung: Leertaste, ←/→ 5 s bzw.
1 s mit Shift, `,`/`.` Feinschritt, 1–9 springen, `+`/`−` Offset, `L`
Wiederholung, `E` Extended-O.

Der Offset wird an genau **einer** Stelle angewendet – dort, wo Videozeit auf
Skriptzeit trifft (`ReportVideoPosition`). Würde er zusätzlich im Frontend
auf die Anzeige gerechnet, liefen Kurve und Gerät auseinander.

**Zwei Verbindungswege zum Gerät:** direkt per Bluetooth (eigener Adapter,
keine Zusatzsoftware) oder über einen laufenden **Buttplug-Server**
(Intiface Central) per WebSocket. Der zweite Weg braucht keinen eigenen
Bluetooth-Adapter und funktioniert mit jedem von Buttplug unterstützten
Gerät; Verbindung, Pairing und Wiederverbindung übernimmt Intiface, und das
sind erfahrungsgemäß die fehleranfälligsten Teile. Umgesetzt nach der
offenen Protokollbeschreibung, nicht durch Codeübernahme; `gorilla/websocket`
war über Wails ohnehin im Baum, es kam also keine neue Abhängigkeit dazu.
Der Server darf auch **auf einem anderen Gerät** laufen – etwa Intiface
Central auf dem Handy, der Player auf dem Rechner. Dafür genügt die IP im
Adressfeld; `192.168.1.50`, `192.168.1.50:12345` und die vollständige
ws://-Adresse werden gleichermaßen angenommen. Verbindungsart und Adresse
werden nach erfolgreicher Verbindung gemerkt. Fehlermeldungen unterscheiden
lokal und Netzwerk, weil die Ursachen andere sind (Server nicht gestartet
gegen falsches WLAN, falsche IP oder Firewall).

Getestet gegen einen nachgebauten Buttplug-Server: sieben Prüfungen,
inklusive Gerät ohne Sog-Kanal, fehlendem Server, Adressumformung und der
Unterscheidung lokaler und netzwerkbezogener Fehlerursachen.

**Der direkte Bluetooth-Weg bleibt unverändert und ist die Voreinstellung.**
Intiface ist eine zusätzliche Möglichkeit, kein Ersatz.

**Gerät:** Eigener Tab mit Verbinden/Trennen, Gerätename, BLE-Adresse und
Signalstärke, Funktionstest für Vibration und Sog, Rohwert-Test.

**Training:** Stop-Start nach Semans und Plateau-Variante, „Jetzt
unterbrechen" (unterbricht den Zyklus, nicht die Session), Erregungsskala
1–10 mit Regelwirkung auf den nächsten Zyklus, Sessionprotokoll.

**Generator:** Zwei Backends – CSRT-Tracker mit markierter Region, oder
Optical Flow ohne Region. Automatische Regionssuche, Szenenschnitt-Erkennung
mit Regionssuche je Szene, Kamerakompensation, adaptive Keyframes, RDP,
Geschwindigkeitsbegrenzung, Achsenwahl, Auto-Retry, Stapelverarbeitung mit
Parallelbetrieb, Zwischenspeicher, Messbericht mit Rückmeldefunktion.

**Erscheinungsgedächtnis:** Der Tracker merkt sich das Aussehen der Region
und sucht sie nach einem Verlust oder Szenenschnitt im ganzen Bild wieder,
statt blind an der letzten Position neu zu verankern. Das bildet die eine
Fähigkeit klassisch nach, die ein Objekterkenner praktisch liefert. Am
Drei-Szenen-Testvideo gemessen (echte Bewegung 90/100/80 px): ohne
Gedächtnis 90,5 / **1,5** / **5,0** px, mit Gedächtnis 90,5 / **101,0** /
**80,0** px. Am Verdeckungsvideo fiel der Trackerverlust von 323 auf 148 von
500 Frames – die verbleibenden 148 sind die Frames, in denen das Objekt
tatsächlich fehlt, und werden als erfolglose Suche gemeldet statt versteckt.
Unterhalb einer Mindestübereinstimmung wird bewusst NICHT neu verankert:
eine geratene Position sieht aus wie eine Messung, ist aber keine.

**Zwei-Punkt-Messung:** Statt einer Region werden zwei verfolgt; das Signal
ist ihr **Abstand**. Ein Abstand zwischen zwei Punkten im selben Bild ist von
Kamerabewegung mathematisch unabhängig – schwenkt oder zoomt die Kamera,
verschieben sich beide gemeinsam. Das Problem entsteht gar nicht erst,
statt nachträglich herausgerechnet zu werden. Ebenso fällt gemeinsame
Bewegung beider Objekte heraus. An einem Testvideo mit Schwenk, gemeinsamer
Bewegung und schwingendem Abstand gemessen: ein Tracker −0,01, zwei Tracker
+0,62 Korrelation zum echten Abstand.

**Bewegungsart-Profile:** `--profile standard` (Hubbewegung) und
`--profile weich` (weiches Gewebe). Der Unterschied ist physikalisch, nicht
kosmetisch: ein starrer Hub ist eine einzelne Bewegung, weiches Gewebe
schwingt nach dem Anstoß gedämpft aus – diese Nachschwingung ist die *Folge*
des Anstoßes, kein eigener Hub. Ohne Prominenzbedingung wird jede davon ein
Keyframe.

Gemessen an einem Testvideo mit Anstoß alle 800 ms und Ausschwingen bei
4 Hz: 81 Keyframes ohne, 42 mit Prominenz 0,35 – das entspricht den rund 40
Anstößen. Der Rekonstruktionsfehler steigt dabei von 0,094 auf 0,176, weil
die Nachschwingungen bewusst nicht mehr abgebildet werden. Deshalb
Profilwahl und keine Voreinstellung. **Saubere Hubsignale bleiben exakt
unverändert** (42 bzw. 80 Keyframes, identischer Fehler).

**Bewegungssignatur (Szenen-Wiedererkennung):** Acht messbare Größen je
Szene – Hauptrichtung, Lage und Streuung des Bewegungsschwerpunkts, Zahl
getrennter Regionen, Kameraunruhe, Hubsymmetrie, Rhythmusstärke. Damit lässt
sich *wiedererkennen, dass eine Szene derselben Art ist wie eine früher
benannte*, und die dort bewährten Parameter übernehmen.

**Wichtige Abgrenzung:** Es wird NICHT erkannt, *was* zu sehen ist. Eine
Stellung als solche zu benennen wäre Bedeutungserkennung und bräuchte ein
trainiertes Modell mit beschrifteten Daten. Benannt wird vom Anwender; das
Programm überträgt die Benennung nur auf ähnliche Signaturen.

Gemessen: zwei senkrechte Szenen mit unterschiedlichem Tempo liegen 0,075
auseinander, senkrecht gegen waagerecht 0,26–0,32, gegen Kameraschwenk
0,28–0,29. Die Schwelle von 0,15 trennt beides sauber. Oberhalb wird
bewusst *nichts* zugeordnet – falsch übertragene Parameter sind schlechter
als gar keine und fallen später schwerer auf.

Noch nicht verdrahtet: Benennen in der Oberfläche und die Übernahme der
Parameter in den Erzeugungslauf. Die Regionenzählung liefert derzeit auf
allen Testvideos denselben Wert und trägt damit nichts zur Unterscheidung
bei.

**Lernende Qualitätsbewertung:** Aus den Urteilen im Messbericht
(brauchbar / grenzwertig / unbrauchbar) lässt sich eine logistische
Regression über sieben Kennzahlen lernen – bewusst kein neuronales Netz,
sondern reines numpy, damit die Gewichte lesbar bleiben und man nachsehen
kann, *warum* das Modell so entscheidet. Das Modell wird nur übernommen,
wenn es die festen Regeln in einer Leave-one-out-Kreuzvalidierung schlägt;
mindestens 12 beurteilte Läufe, mindestens 4 je Klasse. Gespeichert unter
`qualitaetsmodell.json` im Konfigurationsordner. „grenzwertig" zählt als
nicht bestanden – lieber eine Rückfrage zu viel.

**Skriptanalyse:** Zerlegt ein geladenes Skript in Bewegungszustände
(Stillstand, Anfahren, beschleunigend, gleichmäßig, abbremsend, Auslaufen)
und schreibt eine Zeile darunter, woraus es besteht. Das schließt eine
Lücke: viel Stillstand fällt den übrigen Prüfungen **nicht** auf, weil eine
flache Strecke weder verrauscht noch unrhythmisch ist.

**Mindestabstand zwischen Actions:** Wird beim Erzeugen durchgesetzt, nicht
nur gemeldet. Zu dichte Actions entstehen systematisch, weil Hoch- und
Tiefpunkte getrennt gesucht werden – der Mindestabstand gilt damit nicht
zwischen einem Hochpunkt und dem folgenden Tiefpunkt. Bei verrauschten
Signalen lagen dadurch bis zu 45 % der Actions unter 100 ms. Entfernt wird
jeweils der Punkt mit der kleineren Abweichung zur Verbindungslinie seiner
Nachbarn, damit die Scheitel erhalten bleiben; der Rekonstruktionsfehler
steigt dabei von 0,084 auf 0,086, saubere Signale bleiben unberührt.

**Geräteverträglichkeit:** Prüft die erzeugte Datei gegen die in der
Funscript-Gemeinschaft etablierten Grenzwerte – damit sie auch auf fremder
Hardware und in fremden Playern brauchbar ist, nicht nur im eigenen Player.
Alle Zahlen sind übernommen, nicht selbst gesetzt: Mindestabstand 100 ms
zwischen Actions (Launchcontrol-Sendeschwelle), langsamster sinnvoller
Vollhub 900 ms, nutzbarer Positionsbereich 5–95, und die Intensität
`500 × |Δpos| / |Δt|` als Tempo-Kennzahl (Definition aus funscript-utils,
dieselbe Größe wie in OpenFunscripter, Funscript.io und XBVR).

**Quality Doctor:** Bewertet Zeitstempel, Wertebereich, Lücken,
Geschwindigkeitsausreißer, Rhythmus (spektrale Konzentration am dichten,
**detrendeten** Signal), Tracker-Objektverlust, tatsächliche Bewegungs-
amplitude, den **aktiven Zeitanteil** und den **Rekonstruktionsfehler** – wie gut die exportierten
Actions den gemessenen Verlauf noch wiedergeben.

---

## Gemessene Kennzahlen

Alle Zahlen stammen aus tatsächlichen Läufen, nicht aus Schätzungen.

| Was | Wert |
|---|---|
| CSRT-Tracker | ~100 ms/Frame (97,5 % der Gesamtzeit) |
| Dichter Optical Flow (Farneback) | ~18 ms/Frame |
| Videodekodierung | 0,7 % der Gesamtzeit |
| Zwischenspeicher-Treffer | 31,7 s → 0,88 s (Faktor 36) |
| Flow-Backend gegen CSRT | 12 s statt 51 s je Testvideo |

**Amplitudentreue** (echte Objektbewegung 110 px, realistisch texturierter
Hintergrund):

| | stehende Kamera | Schwenk |
|---|---|---|
| CSRT + Merkmalskompensation | 112,2 px | 112,8 px |
| Flow-Backend (Vektorkorrektur, alt) | 137,8 px | 145,2 px |
| Flow-Backend (Positionskorrektur) | 137,8 px | **100,2 px** |

Die Kamerakorrektur im Flow-Backend arbeitet inzwischen merkmalsbasiert auf
der Position statt auf den Flow-Vektoren; beim Schwenk stieg die Korrelation
dadurch von 0,744 auf 0,805.

---

## Fremder Code: was benutzt werden darf

Für ein Projekt, das verkauft werden soll, ist die Lizenz der Vorlage
entscheidend – nicht nur die Frage, ob kopiert wurde.

| Projekt | Lizenz | Nutzbar |
|---|---|---|
| Funscript Flow | Apache-2.0 | ja, auch Code (mit Attribution) |
| funscript-utils, launchcontrol | permissiv | ja, Kennzahlen und Grenzwerte übernommen |
| Buttplug (Protokollwissen) | BSD-3 | Protokollfakten ja; bei Codeübernahme Vermerk nötig |
| **FunGen 1** | **PolyForm Strict 1.0.0** | **nein** – nichtkommerziell UND keine abgeleiteten Werke |
| FunGen 2 | geschlossenes Binary | nichts zu lesen |

PolyForm Strict verbietet abgeleitete Werke. Den Quellcode zu lesen und
seine Funktionen strukturgleich zu übertragen wäre ein abgeleitetes Werk –
in einem MIT-lizenzierten, verkäuflichen Projekt ein Risiko, das sich später
am Code nachweisen ließe. Ideen und beschriebenes Verhalten sind dagegen
nicht schutzfähig: aus öffentlichen Beschreibungen darf gelernt werden, was
ein Programm leistet.

## Übernommen aus fse-generator (Salvage)

Zwei Pakete aus einem Vorgängerprojekt, einzeln geprüft statt als Merge:

| Paket | Inhalt | Zustand |
|---|---|---|
| `motionx` | RDP (iterativ, gibt Indizes zurück, plus `Dedup`), Zustandsklassifikation | **verdrahtet** – Klassifikation in der Skriptanalyse |
| `videox` | `ffprobe`-Wrapper, ffmpeg-Graustufen-Reader | vorhanden, **nicht verdrahtet** |

`motionx` ist abhängigkeitsfrei (nur Standardbibliothek). `videox` dagegen
setzt **ffmpeg und ffprobe im PATH** voraus – eine Anforderung, die das
Projekt bisher nicht hat, weil die Videodekodierung über OpenCV in Python
läuft. Deshalb liegt es bei, ist aber an nichts angeschlossen: es einzubauen
hieße, allen Nutzern eine zusätzliche Installation aufzuerlegen, ohne dass
heute ein Vorteil gegenübersteht. Sinnvoll würde es erst, wenn die Python-
Abhängigkeit insgesamt entfallen soll – das ist eine Architekturentscheidung,
keine Dateiübernahme.

Ausdrücklich **nicht** übernommen (Begründungen aus der Salvage-Analyse):
`EstimateTranslation` integriert das Kamerasignal statt des Subjektsignals
bei Suchraster 4 und läuft binnen weniger Frames an den Anschlag;
`pattern.Periodicity` misst Glattheit statt Periodizität (linearer Drift
ergibt 1,000); `candidate.Build` bevorzugt dadurch systematisch die
überglättete Variante – dieselbe Fehlerklasse wie beim Zickzack-Problem.

## Erweiterbarkeit

Analyseverfahren sind austauschbare Bausteine (`generator/backends.py`).
Vorher war die Wahl eine fest verdrahtete Fallunterscheidung mitten in der
Pipeline – jedes neue Verfahren hätte dort einen Eingriff bedeutet.

Ein Backend ist eine Funktion mit festem Vertrag:

```
analyze(video_path, roi, options)
  -> (timestamps_ms, positions, frame_size, scene_cuts, stats)
```

Zwei Bedingungen sind nicht Formsache: Positionen kommen in
**Bildkoordinaten**, nicht normalisiert (sonst würden Dynamik und
Normalisierung stillschweigend ausgehebelt), und `stats["vertical_range"]`
ist die Amplitude **in Pixeln** (nach der Normalisierung ist sie
unwiederbringlich weg, und der Quality Doctor braucht sie, um echte Bewegung
von hochskaliertem Zittern zu unterscheiden).

Eigene Verfahren kommen als Python-Datei ins Plugin-Verzeichnis und melden
sich mit `register(name, func, beschreibung)` an. `--list-backends` zeigt
alle verfügbaren mit Herkunft. Vertragsverstöße scheitern sofort mit einer
Meldung, die sagt, *was* fehlt – ein Plugin, das eine Zeile zu wenig
liefert, fiele sonst erst als unerklärlich schlechtes Skript auf.

Absichtlich **keine Sandbox**: ein Plugin läuft mit denselben Rechten wie das
Programm. Das ehrlich zu sagen ist besser als eine Scheinsicherheit.

## Behobene Fehler mit Außenwirkung

**Unvollständige Einbettung der Python-Module.** Eingebettet waren vier
Dateien, ins Temp-Verzeichnis geschrieben wurden zwei. Im Entwicklungsbaum
unsichtbar, weil dort alle Module nebeneinander liegen. In der fertigen
`.exe` zeigte sich das als **drei scheinbar verschiedene Fehler**:
`--backend flow` scheiterte, das gelernte Qualitätsmodell wurde nie
gefunden, und die Geräteprüfung lief still gar nicht – ihr Import steht in
einem `try/except` und fiel lautlos durch. Jetzt wird das ganze Verzeichnis
per `go:embed *.py` eingebettet; `generator_embed_test.go` liest die
tatsächlichen Importe aus dem Quelltext und prüft, dass jedes davon im
Temp-Verzeichnis landet. Eine gepflegte Liste wäre genau das, was hier schon
einmal vergessen wurde.

**Keepalive hielt nur einen Kanal.** Es wiederholte das zuletzt gesendete
Paket. War das Sog, wurde die Vibration nicht gehalten – und umgekehrt.
Ausgerechnet in Pausen und beim Extended-O, also dort, wo das Keepalive
überhaupt greifen soll. Jetzt wird der Zustand **beider** Kanäle geführt und
wiederholt. Zusätzlich werden unveränderte Pakete nicht erneut gesendet: das
Gerät kennt nur ganzzahlige Stufen, 100 Rampenschritte ergeben höchstens 11
verschiedene Pakete – der Rest waren Roundtrips ohne Wirkung, bei zwei
Kanälen alle 50 ms bis zu 40 pro Sekunde.

## Bekannte Grenzen

**Das Flow-Backend überschätzt die Amplitude auch bei stehender Kamera**
(137,8 px statt 110). Das ist kein Kamerafehler – die Korrektur greift dort
gar nicht – sondern eine Eigenschaft des Zentrumsschätzers. Nach der
Normalisierung auf 0–100 fällt es weniger ins Gewicht als es aussieht; die
Form stimmt (Korrelation 0,941).

**Alle Qualitätsschwellen sind an synthetischen Videos kalibriert.** Saubere
Sinusbewegungen, deren Wahrheit per Konstruktion bekannt war. Echtes Material
ist unregelmäßiger und liegt systematisch niedriger. Deshalb gibt es den
Messbericht: erst mit echten Läufen **und** menschlichem Urteil lassen sich
die Schwellen belastbar nachziehen.

**Der Generator liefert schwächere Bewegung als FunGen – teilweise geklärt.**
Am selben Film gemessen: Sprunghöhe im Median 6,5 gegen 44,5, Sprünge über
50 Punkte 0,0 % gegen 34,7 %, Zeit im Mittelband 40–60 bei 33,3 % gegen
14,2 %. Unser Skript zappelte in der Mitte, statt zwischen den Extremen zu
wechseln.

Eine Ursache ist gefunden und behoben: In der Hälfte aller 6-Sekunden-Fenster
wurden nur 30 von 100 Punkten genutzt, weil global normalisiert wurde. Die
gleitende Dynamik hebt die mittlere Bewegungsstärke von 10,1 auf 26,2.

Der Rest ist **nicht** die Nachverarbeitung – das ist inzwischen gemessen.
An einem synthetischen Video mit realistisch wechselndem Tempo (1,2–2,2 Hz)
erreicht die Pipeline Intensität 162,1 gegen ideal 161,4, also praktisch
verlustfrei. Auch die Größe der markierten Region ändert nichts (112, 111,
110 px bei 40×40, 70×70 und 120×120). Die Glättung dämpft bei allen
realistischen Hubfrequenzen nur 0–8 %.

Bleibt als Erklärung die Messgröße selbst: eine einzelne verfolgte Region
misst, wie weit sich dieser Bildbereich verschiebt. Bei echtem Material ist
aber meist die **relative** Bewegung zweier Körper das Signal, und die kann
deutlich größer sein als die absolute Verschiebung eines der beiden. Dafür
gibt es die Zwei-Punkt-Messung (`--roi2`); die automatische Erkennung beider
Regionen ist gemessen noch nicht gut genug (siehe `find_two_rois`).

Zur endgültigen Klärung wird das Originalvideo gebraucht; aus dem
exportierten Skript allein lässt sich das ursprüngliche Signal nicht
zurückgewinnen.

**Nie an echter Hardware getestet.** Verbindung, Wiedergabe und Training
sind ausschließlich gegen `device.Mock` verifiziert.

**Auflösung des Geräts unbekannt.** Die Bereiche 0–10 (Vibration) und 0–5
(Sog) stammen aus der Buttplug-Gerätekonfiguration – das ist die Stufenzahl,
auf die *Buttplug* quantisiert, nicht notwendigerweise eine Grenze der
Firmware. Der Rohwert-Test im Geräte-Tab existiert, um das zu klären.

**Das gelernte Modell hat noch keine echten Daten gesehen.** Die Mechanik ist
mit synthetischen Beispielen geprüft (Trefferquote 50 % → 100 % in einem
konstruierten Fall), aber ob sie an realem Material trägt, ist offen. Bis
genügend Urteile vorliegen, gelten unverändert die festen Regeln.

**Auto-Retry ist unbelegt.** Die Funktion ist gebaut und getestet, aber im
gesamten Kalibrierungssatz gibt es keinen Fall, den sie verbessert – alle
Fehlschläge dort sind Tracking-Probleme, die sie korrekt überspringt.

---

## Geprüft und verworfen

Damit niemand dieselbe Arbeit zweimal macht:

| Idee | Ergebnis |
|---|---|
| Divergenz als Motion-Center-Schätzer (Funscript Flow) | 41 px statt 110, Korrelation 0,829 gegen 0,904/0,917. Verschlechtert die Kombination. |
| Symmetrische Projektionsgewichtung (Funscript Flow) | Kein messbarer Unterschied (110,5 gegen 110,1). |
| Optical Flow als Fusionspartner (integriert) | Korrelation 0,27–0,78, Amplitude bis Faktor 2,4 daneben. Integration summiert Schätzfehler. |
| Kamerakorrektur über Median des Flow-Felds | Korrigiert Vektoren, nicht Positionen – wirkungslos für dieses Backend. Ersetzt durch merkmalsbasierte Positionskorrektur. |
| Fusion von CSRT und Flow-Backend | Landet immer **zwischen** den Quellen, schlägt nie die beste (clean 1,000/0,926 → 0,985; occluded 0,293/0,747 → 0,620). Bei zwei Quellen ist die gegenseitige Übereinstimmung symmetrisch: sie sagt, DASS sie uneinig sind, nicht WER recht hat. `fusion.py` liegt getestet bei, ist aber nicht verdrahtet. |
| Mindestzyklenzahl als Periodizitätskriterium | Bestraft langsame, aber gültige Bewegung: zwei saubere Zyklen wären „unbelastbar". Der **aktive Zeitanteil** trennt besser – Einzelausschlag 0,27, zwei langsame Zyklen 0,86, durchgehend 1,00. |
| Rhythmus global über das ganze Video messen | Setzt EINEN durchgehenden Rhythmus voraus. An zwei echten Skripten desselben Films (77 s): global 0,203 und 0,039 – **beide** als verrauscht eingestuft, eines davon aus einem etablierten Fremdprogramm. Fensterweise (8 s, Median): 0,396 und 0,324. |
| Rhythmusmaß ohne Detrending | Linearer Drift bekam 0,289 – über der Ausschlussschwelle 0,20, wäre also durchgegangen. Umgekehrt fiel ein gültiger Sinus **mit** Drift von 1,000 auf 0,425. Detrending behebt beides. |
| Absolute Zahl der RANSAC-Übereinstimmungen als Gütemaß | Auf Rauschhintergrund reichlich Zufallstreffer; Amplitude stieg auf 233 px. Der **Anteil** trennt sauber (0,24 gegen 0,84–0,90). |
| Feinere Trainingsrampen | Zuerst mit falscher Begründung verworfen; die Auflösungsfrage ist offen, siehe Rohwert-Test. |
| CUDA über pip-OpenCV | `opencv-python`/`opencv-contrib-python` werden **ohne CUDA** gebaut. Nur über Eigenbau oder OpenCL. |

---

## Wo Fallstricke lauern

Diese Punkte haben bereits Zeit gekostet:

- **Testvideos mit Rauschhintergrund taugen nicht für Kamerakompensation.**
  `goodFeaturesToTrack` findet dort keine stabilen Merkmale, die Schätzung
  wird zum Random Walk, und die Kompensation sieht fälschlich kaputt aus.
  Immer texturierte Hintergründe verwenden.
- **Bewegte Objekte in Testvideos müssen in Weltkoordinaten liegen.** An
  fester Bildschirmposition gezeichnet machen sie einen Kameraschwenk nicht
  mit, und die Kompensation sieht fälschlich falsch aus.
- **`TRACK_CACHE_VERSION` erhöhen**, wenn sich `track_roi` oder die
  Kamerakompensation ändert. Sonst liefert der Cache still Ergebnisse der
  alten Implementierung.
- **Wails-Bindings neu erzeugen** (`wails build`) nach jeder neuen
  Go-Methode. Die Frontend-Tests erzeugen ihre Attrappen inzwischen aus
  `App.js`, fallen also sofort auf.
- **Rhythmusprüfung nur am dichten Signal.** Die Peak/Valley-Reduktion macht
  aus jedem Signal einen Zickzack, der rhythmisch aussieht.
- **`-trimpath` beim Bauen.** Ohne das landet der Pfad des Build-Rechners –
  und damit der Benutzername – in der EXE.

---

## Zusammenarbeit

`CONTRIBUTING.md` beschreibt Ablauf, Zuständigkeitsschnitte und die Regel,
auf die es ankommt: **Verhaltensänderungen brauchen einen Regressionstest
mit Gegenprobe.** Für reine Dokumentation gilt die Prüfung der Angaben.
Zwei Workflows laufen auf GitHub – `tests.yml` bei
jedem Push und Pull Request (Go mit Race-Detector, Python, Oberfläche),
`release.yml` nur bei einem Versions-Tag.

Vorher liefen die Tests **ausschließlich** beim Release. Bei mehreren
Beteiligten wäre ein Fehler erst beim Ausliefern aufgefallen, in fremdem
Code, dessen Zusammenhang längst vergessen ist.

## Tests

Vollständiger Durchlauf nach dem Setup aus `WIEDERAUFNAHME.md` (Bash):

```
go vet ./... && go test ./...
(cd generator && for t in *_test.py; do python3 "$t" || exit 1; done)
for t in cmd/gui-wails/frontend/test/*_test.py; do python3 "$t" || exit 1; done
```

Die Frontend-Tests laden das echte JavaScript in Headless-Chromium und
ersetzen nur die Wails-Bindings; die Attrappen entstehen automatisch aus
`App.js` (siehe `test/_harness.py`).

---

## Warum `fusion.py` und `videox` nicht verdrahtet sind

Beide erfüllen die harte Regel aus Abschnitt 16 des Zielkonzepts derzeit
nicht („kein Modul ohne verbesserten Pfad"). Sie liegen mit Tests im Baum,
weil die Voraussetzung für ihren Nutzen absehbar ist – eine dritte Messquelle
beziehungsweise der Verzicht auf die Python-Abhängigkeit –, nicht weil sie
schon gebraucht würden.

## Projektstatus und Weiterarbeit

Die operative Aufgabenliste mit Prioritäten steht ausschließlich in
[docs/NEXT.md](docs/NEXT.md). Die oben beschriebenen Hardware- und
Qualitätsgrenzen bleiben bestehen, bis Messberichte sie nachweislich klären.

Der Release-Workflow wurde am 14. September 2026 mit `v0.2.1` erfolgreich
ausgeführt: GUI und CLI für Windows/Linux sowie `checksums.txt` wurden
veröffentlicht. Tf/Tj, zweite GUI-Region und `suction_position` sind auf
`main` integriert (PRs #2–#4). Das belegt Build und Integration, nicht die
Wirkung an echter Hardware.

## Umgebung einrichten

Die Anleitung für Klonen, Abhängigkeiten und Builds steht in
[WIEDERAUFNAHME.md](WIEDERAUFNAHME.md). Go und Wails richten sich nach
`go.mod`; CI und Release verwenden dieselbe Quelle für ihre Tool-Versionen.
