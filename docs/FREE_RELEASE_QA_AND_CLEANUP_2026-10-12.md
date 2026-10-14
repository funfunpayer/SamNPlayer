# SamNPlayer Free — Release-Abnahme und GitHub-Aufräumplan

**Aktualisiert: 14. Oktober 2026 (Release-PR v0.7.0). Prüfstand, keine Release-Freigabe.**
Dieses Dokument beschreibt den aktuellen Free-Abschluss. Die Entwicklung von
SamNPlayer 2.0 Next ist ein eigenes Projekt und blockiert die Free-Veröffentlichung nicht.

## Stand für v0.7.0 (14.10.2026, Release-PR)

Kandidat: `main` nach #542 und der Release-PR, `VERSION=0.7.0`. Das letzte
veröffentlichte Release ist weiterhin **v0.5.45**. 0.5.46 und 0.6.0 wurden
nie getaggt; v0.7.0 trägt beide mit.

| Gate | Stand | Beleg / was fehlt |
|---|---|---|
| G1 | erfüllt (automatisch) | CI grün auf #540–#542: Go, Go OpenCV Linux/Windows, Python, Frontend (148 Playwright-Suites). Zusätzlich 758 öffentliche Community-Scripts durch Laden, Kurve, Heatmap, Kapitel, Neo-2-Frames (alle Szenen-Gefühle) und *Optimize for* (Allgemein und Neo 2): 758/758 ohne Befund, nachdem der Fix für kaputte Zeitstempel (#542) drin war. |
| G2 | erledigt | #489 ist geschlossen (nicht gemergt); #532 ist gemergt. |
| G3 | **offen, Owner** | Nativer Windows-11-Lauf mit dem Build. In der Cloud gibt es nur Browser-Captures (`docs/media/`, Testbild statt echter Videos). |
| G4 | teilweise | Video→Generator→Edit→Play nur in Browser-Tests mit Stubs. Ein echter Durchlauf mit einem Video steht in der Checkliste `docs/RELEASE_0_7_0.md`. |
| G5 | **offen, Owner** | Mocktests für Neo 2, Intiface, TCode und Stroker-Soft-Seek sind grün. Hardware ist nicht getestet: Checkliste Punkte 7–8. |
| G6 | geprüft | `release.yml` baut Free und Suite; der Updater wählt die eigene Edition. Suite-Dateien nur im privaten Release, nicht öffentlich verlinken (steht in den Release-Notes). |
| G7 | erledigt | README (inkl. „In numbers“), Handbuch, CHANGELOG `[0.7.0]` und Roadmap beziehen sich auf denselben Kandidaten. |
| G8 | offen | Website ist ein eigener PR (SamNPlayer-site). |
| G9 | nach dem Tag | Tag, Artefakte/Hashes, Update-Smoketest. |

Nur ein Tag baut herunterladbare Dateien. Deshalb ist das private Release
v0.7.0 der Prüfkandidat für G3, G4 und G5. Öffentlich verlinkt (G8) wird erst,
wenn diese Prüfungen bestanden sind. Geht etwas schief, werden Tag und
Release zurückgenommen (`docs/owner/owner-aufgaben.md`, Teil 2).

## Bestätigter Stand (12.–14.10.2026, vor v0.7.0)

- Geprüftes `main`: `800b2337af889701be48f5e21a4c1eefc51e4a8d`; `VERSION=0.6.0`.
- Die GitHub-Release-Abfrage vom 14.10.2026 liefert **v0.5.45** (veröffentlicht 2.10.2026) als jüngstes privates Release, mit GUI-/CLI-/Portable-Artefakten und Prüfsummen. Die Release-Liste von `SamNPlayer-site` ist leer. Ein öffentlicher Download oder veröffentlichtes 0.6.0-Release ist damit nicht belegt.
- #509 sowie **#520** sind gemergt. #520 ist seit 12.10.2026 integriert; ein erneuter Merge ist kein offenes Gate.
- `main` enthält inzwischen #524–#529 und #531: TCode über WLAN/USB, Companion-Achsen, achsenspezifische Bearbeitung und neue Achsendateien sowie Pegel je Motor. Integration in den Quellstand ersetzt keine Hardware-Abnahme.
- **#489** ist weiterhin Draft/offen und aktuell konfliktbehaftet. Nur noch benötigte CSS-Änderungen gegen den aktuellen Stand abgleichen; keinen alten Gesamtstand übernehmen.
- **#532** ist offen, Head `6e2aa3a`. Claude meldet dort vollständige Frontend-Suite und Vite-Build lokal erfolgreich sowie aktualisierte Layout-/Theme-Tests und Handbuchtext. Dies ist ein berichtetes Testergebnis, keine in diesem Dokument unabhängig wiederholte native Abnahme. Keine interaktive Kurvenbearbeitung oder funktionslose Curve-Preview-Tabs in Create behaupten; Ergebnisse öffnen in Edit.
- **#510 wurde am 13.10.2026 ohne Merge geschlossen** und ist historisch/superseded. Die fünf Konzeptdokumente bleiben in seiner PR-Historie erhalten.
- #515 wurde ohne Merge geschlossen: keine Sieben-Tage-Preview. Free und SamNPlayer 2.0 Next bleiben kostenlose, durch freiwillige Spenden unterstützte Software; Suite ist die interne Edition. EULA und proprietäre Quellcode-Lizenz werden durch dieses Cleanup nicht geändert.

## QA-Gates vor Tag/Veröffentlichung

| Gate | Test/Prüfung | Verantwortlich | Beleg zur Freigabe |
|---|---|---|---|
| G1 | Bereits integrierte Geräte-/Tools-Änderungen am tatsächlichen Release-Kandidaten prüfen; #520 nicht erneut als offenen Merge führen | Claude | Kandidaten-SHA, vollständiges CI-Ergebnis, Geräte-Mocktests |
| G2 | #489-Konflikte und verbliebene CSS-Lücken gegen aktuelles main und #532 auflösen; Create-Layout/Test-/Handbuchstand von #532 bei Integration erhalten | ChatGPT CSS + Claude | geprüfter Diff, Theme-Tests, keine Logik- oder DOM-Regression |
| G3 | Native Wails Windows 11: Play/Create/Edit/Training/Tools, 1366×768 und 1920×1080; Fokus, Kontrast, Disabled, Reduced Motion, Empty, Play/Pause/Resume/Stop | Claude + Owner | reale Captures aus Build + Testprotokoll |
| G4 | Video→ROI→klassischer Generator→.samn/.funscript→Edit→Play→Export/Import; fehlendes Modell, Audio, Video, ffmpeg; Undo, Cancel, Resume; Create mit mehreren Clips/lange Dateinamen/Expert-Zuständen | Claude | reproduzierbare Testmatrix und Logs ohne private Videos |
| G5 | Neo2, Intiface, TCode WLAN/USB und Companion-Achsen am Kandidaten prüfen: No-Device/Disconnect/Pause/Stop/Emergency Stop/jeweilige Limits; Hardware-Abdeckung ausdrücklich ausweisen | Claude + Owner | Mocktests und beaufsichtigter Hardware-Smoke; ungetestete Geräte benennen |
| G6 | Free/Suite sauber getrennt: Build-Tags, nur Free im öffentlichen Publish, Edition-sicherer Updater, EULA, Notices, Donate-Link | Claude | Free-/Suite-Build-Report und geprüfte Release-Artefakte |
| G7 | Handbuch, Changelog, README und Editions-/Versionshinweise auf denselben Kandidaten beziehen; #521 enthält die Bereinigung aus #530 | ChatGPT Docs + Claude | konsistente Dokumentation; keine Behauptung, dass Quellversion oder Doku-Merge eine Veröffentlichung beweisen |
| G8 | Öffentliche Website: reale Screenshots, bestätigte Downloads, aktuelle Features, bekannte Einschränkungen und Donate | ChatGPT Website + Owner | separater SamNPlayer-site-PR, verifizierte öffentliche Links |
| G9 | Nach QA-/Owner-Freigabe Tag, Artefakte/Hashes, öffentliche Free-Downloads und Update-Smoketest prüfen | Claude + Owner | bestätigter Release-Tag, herunterladbare Free-Binaries, Website-Link |

## Priorisiertes Repository-Cleanup

1. **Integration und QA:** #489 nicht unverändert mergen; #532 und den tatsächlichen Release-Kandidaten prüfen. Mergeability allein ist keine Freigabe.
2. **Dokumentation zusammenführen:** #521 integriert die README-Korrekturen und den vollständigen historischen Supersession-Abschnitt aus #530. #530 wird danach als durch #521 ersetzt geschlossen; kein zweiter Editions-Patch erforderlich.
3. **#510 ist erledigt:** geschlossen ohne Merge. Historische Konzepte behalten; technische Vorschläge nur einzeln geprüft in eine Next-Roadmap übernehmen. Keine erneute Archivmigration oder Löschung voraussetzen.
4. **Weitere PRs/Branches:** nur mit nachgewiesenem Supersession-Link schließen. Keine unkontrollierte Löschung von History, Tests, Migrationsskripten, Drittanbieter-Nachweisen oder Nutzerfunktionen.
5. **Öffentliches Release:** ausschließlich verifizierte Free-Downloads; interne Suite-Dateien, Quellcode, Datensätze/Modelle und Benchmarks nicht versehentlich veröffentlichen.

## Definition of Done

**Free ist freigegeben**, wenn G1–G9 mit Belegen abgenommen sind.
`VERSION=0.7.0`, ein einzelner grüner Workflow, ein Dokumentationsmerge oder
Zielbilder alleine sind keine Release-Bestätigung.

Spenden bleiben freiwillig: keine Paywall, keine Lizenzschlüssel, kein
EUR-40-Entitlement und keine zeitliche Funktionssperre. Spätere Änderungen am
Free-Funktionsumfang sind kein Bestandteil dieser Bereinigung und benötigen
einen eigenen geprüften Owner-Auftrag einschließlich Migration und Regressionstests.
