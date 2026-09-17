# Scriptqualität — Code-/Architektur-Zusammenfassung (17.09.2026)

Quelle: hochgeladenes Review-PDF. Unten Rohtext; Umsetzungshinweise am Ende.

---

SamNPlayer - technische Projektzusammenfassung | 17.09.2026
Seite 1
 SamNPlayer - Scriptqualitaet, aktuelle
Befunde und naechste Entwicklungsschritte
 Code- und Architektur-Zusammenfassung auf Basis des aktuellen GitHub-Stands vom 17. September
 2026
Kurzfazit
SamNPlayer hat seit der ersten FunGen2-Vergleichsanalyse mehrere echte Ursachen gefunden und
korrigiert. Besonders wichtig sind die korrigierte Zwei-Punkt-Distanz, ein belastbarerer Benchmark,
bessere ROI-Erkenntnisse, Region-Fusion und der experimentelle Go-native CSRT-Tracker. Trotzdem
ist die allgemeine Luecke auf schwierigem Realmaterial noch nicht geschlossen. Der groesste
verbleibende Engpass liegt heute nicht beim Schreiben der .funscript-Datei, sondern bei der
verlaesslichen Wahrnehmung und Interpretation der Bewegung im Video.
1. Was inzwischen korrigiert wurde
 Zwei-Punkt-Messung: Die alte Logik betrachtete im problematischen Pfad nur den vertikalen Abstand. Bei zwei
 Regionen auf aehnlicher Hoehe konnte dadurch trotz realer Bewegung fast kein Signal entstehen. Die
Berechnung nutzt inzwischen die volle 2D-Mittelpunktsdistanz via hypot. Eine Gegenprobe dokumentiert etwa 1
px alte Amplitude gegen rund 80 px tatsaechliche Bewegung.
Benchmark / fungen_compare.py: Der fruehere Vergleich hatte mehrere methodische Fehler: Kandidaten
 konnten falsch dedupliziert werden, zeitlich verschobene Signale wurden unguenstig verglichen, konstante
Referenzen wurden falsch behandelt und invertierte Polaritaet war nicht sauber erkennbar. Diese Punkte wurden
mit Regressionstests korrigiert.
ROI-Qualitaet: Ein realer 42-s-Test zeigte, wie stark die Regionenauswahl wirkt: bei unguenstiger ROI-Platzierung
 wurden sehr viele Frames als verloren bewertet. Mit engeren, gezielteren ROIs sank der dokumentierte Verlust
auf etwa 2 %, waehrend der Quality-Doctor-Score von 0,60 auf 0,95 stieg.
Region Fusion: region_fusion teilt eine markierte Region in vier Teilbereiche und gewichtet deren Bewegung
 adaptiv. region_fusion_auto geht weiter: das Gesamtbild wird automatisch in vier Zonen aufgeteilt, ohne vorherige
manuelle ROI.
Go-native CSRT: generator/trackcv portiert den teuersten Tracking-Schritt experimentell nach Go ueber eine
 schmale OpenCV-cgo-Schicht. Der dokumentierte Machbarkeitstest ergab r=0,9996 gegen den Python-CSRT,
maximal etwa 3 px Abweichung und ungefaehr 15 % weniger Laufzeit. Der Pfad ist noch nicht produktiv in den
Generator verdrahtet.
2. Warum die Skripte noch nicht durchgehend an die Referenz
heranreichen
Die aktuellen Messungen zeigen zwei sehr unterschiedliche Welten. Auf sauberem synthetischem Material wurde
nach den Korrekturen eine deutlich hoehere Uebereinstimmung erreicht; dokumentiert sind Mittelwerte um r=0,52
fuer Hub und r=0,58 fuer TF auf den gueltigen Clips eines groesseren synthetischen Datensatzes. Auf echtem
Material war die Uebereinstimmung in frueheren Versuchen wesentlich schwaecher. Ein einzelner spaeterer
Realclip kam auf etwa r=0,34 bis 0,36, zeigte aber weiterhin deutliche Form-/Amplitudenabweichungen.
Das spricht dafuer, dass die verbliebene Differenz vor allem in der Video-Wahrnehmung entsteht: Verdeckungen,
mehrere gleichzeitig bewegte Strukturen, nicht-starre Bewegung, Bewegungsunschaerfe, Kamerabewegung,

SamNPlayer - technische Projektzusammenfassung | 17.09.2026
Seite 2
Schnitte, Identitaetswechsel und zeitweiser Trackingverlust. Ein technisch sauberes Funscript kann daher
trotzdem die falsche Bewegung abbilden.
3. Signal Quality ist nicht Motion Fidelity
Diese Trennung sollte im Projekt fest verankert werden. Der Quality Doctor kann beurteilen, ob ein Signal
technisch plausibel ist - etwa wenig Jitter, keine unmoeglichen Spruenge, sinnvolle Kontinuitaet und erkennbare
Trackingfehler. Er beweist aber nicht, dass das Signal die Bewegung des Videos korrekt wiedergibt.
Qualitaetsart
Beispiele fuer Messgroessen
Signal Quality
Jitter, Spruenge, Kontinuitaet, Trackingverlust, Glattung, unplausible Werte
Motion Fidelity
Rhythmus, Phase, Richtungswechsel, Amplitude, Geschwindigkeit, Ereignis-Timing, Referenznaehe
4. Bewertung der neuesten Architektur-Aenderungen
region_fusion und region_fusion_auto: Beide sind strategisch sinnvoll, weil sie die Abhaengigkeit von einem
einzelnen Trackingpunkt reduzieren. Der Auto-Pfad ist besonders wichtig fuer das Ziel eines weitgehend
automatischen Generators. Er ist aber noch als Kandidat zu behandeln, bis er auf einem festen Satz echter
Golden Clips reproduzierbar besser abschneidet.
Go-native CSRT: Der Port ist ein guter Engineering-Schritt, weil vor der eigentlichen Portierung gemessen wurde.
Er reduziert potenziell Laufzeit und spaeter Python-Abhaengigkeiten, verbessert aber nicht automatisch die
Wahrnehmungsintelligenz von CSRT. Die eigentliche Frage bleibt, welche Bewegung verfolgt werden soll und wie
widerspruechliche Beobachtungen bewertet werden.
Tests: Die neue Go-Testsuite hat bereits zwei reale Speicherfehler aufgedeckt - einen fehlerhaften calcHist-Aufruf
und einen Double-Free im Tracker-Lebenszyklus. Das bestaetigt den Projektansatz: messen, testen, Fehler
reproduzieren, korrigieren und erst dann integrieren.
5. Der naechste grosse Hebel: SAM Perception v1
Die vorhandenen Verfahren sollten kuenftig nicht nur als alternative Dropdown-Backends nebeneinander
existieren. SamNPlayer hat inzwischen genug unterschiedliche Beobachter, um einen Fusion-Layer aufzubauen.
CSRT, grid_lk, region_fusion, region_fusion_auto und Optical Flow koennen pro Segment unabhaengige Evidenz
liefern. Spaeter koennen Pose, Depth und weitere Modelle hinzukommen.
Stufe
Aufgabe
1. Beobachtung
Mehrere Tracker/Estimatoren erzeugen Rohsignale und Confidence.
2. Segmentbewertung
Trackingverlust, Bewegungskonsistenz, Kameraeinfluss und Signalqualitaet bestimmen.
3. Konsens/Fusion
Uebereinstimmende Beobachter staerken Confidence; Widersprueche werden markiert oder anders gewichtet.
4. Motion Model
Rhythmus, Richtung, Geschwindigkeit, Beschleunigung, Range, Phase und Bewegungsklasse ableiten.
5. Script/Runtime
Erst aus der semantischen Bewegung ein Funscript bzw. geraetespezifisches Laufzeitsignal erzeugen.
6. Empfohlene Reihenfolge der naechsten Entwicklung
 1. Golden-Clip-Datensatz mit mehreren echten Videos, Referenzskripten, festen Segmenten, Hashes, Parametern
 und dokumentierten ROIs vervollstaendigen.

SamNPlayer - technische Projektzusammenfassung | 17.09.2026
Seite 3
2. Alle vorhandenen Backends auf exakt denselben Segmenten messen - nicht nur Korrelation, sondern auch
 Turning-Point-/Event-Timing, Amplitude, Phase, Trackingverlust und Confidence.
3. Raw Tracking und nachgelagerte Verarbeitung getrennt bewerten, damit klar wird, ob Fehler im Tracker,
 Smoothing, Keyframe-Extraktion oder Mapping entstehen.
4. Auto-Evaluator bauen, der pro Segment den besten Beobachter bzw. eine gewichtete Fusion waehlt.
 5. Confidence kalibrieren: hohe Confidence muss statistisch tatsaechlich mit niedrigerem Fehler
 zusammenhaengen.
6. Unsichere oder widerspruechliche Segmente automatisch in eine Training Queue geben, statt den Benutzer das
 ganze Video pruefen zu lassen.
7. Erst danach entscheiden, welcher weitere Python-Teil nach Go portiert wird. Migration nur bei gemessenem
 Gewinn und ohne Regression.
8. SAM Motion Model als stabile Zwischenschicht zwischen Wahrnehmung und Funscript/Device Runtime
 etablieren.
9. Signal Quality und Motion Fidelity getrennt im GUI und in Reports anzeigen.
 10. Erst wenn der Benchmark eine reproduzierbare Verbesserung auf echten Clips zeigt, einen neuen
 Backend-/Fusion-Pfad zum Standard machen.
7. Was noch offen ist
Der aktuelle Projektstatus fuehrt den Abgleich mit FunGen2-Referenzen weiterhin als offene, laufende
Messaufgabe. Automatische Zwei-ROI-Vorschlaege sind noch nicht ausreichend. Der neue Go-Tracker ist noch
nicht in den produktiven Generator verdrahtet. Fuer die allgemeine Aussage, dass SamNPlayer auf Realmaterial
gleichwertig oder besser ist, fehlen noch mehrere sauber reproduzierte echte Clips. Ebenso steht die reale
Sam-Neo-2-Hardwarevalidierung weiterhin separat aus.
8. Architekturentscheidung
Der sinnvollste naechste Meilenstein ist daher nicht "noch ein Tracker", sondern SAM Perception v1:
mehrere vorhandene Wahrnehmungsmethoden messen, segmentweise bewerten, Confidence
erzeugen, automatisch fusionieren und daraus ein geraeteunabhaengiges Motion Model ableiten.
Dieses Modell wird anschliessend in Funscript oder die spaetere adaptive Device Runtime
uebersetzt.
Quellenbasis
Diese Zusammenfassung basiert auf dem am 17.09.2026 eingesehenen aktuellen GitHub-Stand von funfunpayer/SamNPlayer,
insbesondere docs/NEXT.md, docs/FUNGEN_PARITY_PLAN.md und dem neuesten Commit zum experimentellen
generator/trackcv-Paket. Zahlen und Statusangaben wurden aus diesen Projektunterlagen uebernommen; nicht dokumentierte
Gleichwertigkeit mit FunGen2 wird ausdruecklich nicht behauptet.


---

## Umsetzungsstand (Agent)

- Signal Quality ≠ Motion Fidelity: **verankert** in `docs/SIGNAL_VS_FIDELITY.md`, API `kind`, GUI-Labels, Phase-CLI
- SAM Perception v1 als nächster Meilenstein: **in** `docs/ROADMAP.md` + `docs/SAM_ARCHITECTURE.md`
- Golden Clips / Backend-Bake-off / Auto-Evaluator / Fusion: **offen** (Messung/Clips nötig)
- Kein spekulativer Fusion-Layer ohne Golden-Clip-Gewinn
