# SamNPlayer Free — Release-Abnahme und GitHub-Aufräumplan

**Status: 12. Oktober 2026, gegen 18 Uhr UTC; Prüfstand, keine Release-Freigabe.** Dieses Dokument beschreibt ausschließlich den aktuellen Free-Abschluss; der Next Level 2.0-Fahrplan ist ein eigenes Projekt.

## Bestätigter Stand

- `main` bei Prüfung: `29de9a121710ca3c0001f4da30078d33d72c1ce3`; `VERSION` enthält **0.6.0**.
- Letztes über GitHub-API gefundenes Release **im privaten SamNPlayer-Repo**: **v0.5.45**, veröffentlicht 2.10.2026. Die API-Liste für Releases in `SamNPlayer-site` war bei Prüfung leer. Daraus lässt sich kein neuer öffentlicher 0.6.0-Download belegen.
- GUI-Integrations-PR #509 ist gemergt. CSS-Polish **#489** ist noch **Draft/offen** und war mergeable=clean, aber nicht visuell/native abgenommen.
- **#520 Tools Home + Devices D2** ist offen, CI am jüngsten Commit lief bei Prüfung noch.
- Historischer Roadmap-PR **#510** enthält mittlerweile sachlich überholte kostenpflichtige Studio-/Entitlement-Abschnitte; **nicht unverändert nach main mergen**.
- Sieben-Tage-Preview #515 ist auf Owner-Wunsch ohne Merge geschlossen. Free und das geplante Next Level 2.0 sollen **spendenfinanzierte Freeware** bleiben. Suite ist die interne Testedition.

## QA-Gates vor Tag/Veröffentlichung

| Gate | Test/Prüfung | Verantwortlich | Beleg zur Freigabe |
|---|---|---|---|
| G1 | #520 auf Geräte-/Tools-API und CI prüfen, erst nach grünem Build und Review mergen | Claude | commit SHA, vollständiges CI-Ergebnis, Geräte-Mocktests |
| G2 | #489 gegen tatsächlich gemergten #509-HTML/JS/States prüfen; CSS isoliert testen | ChatGPT CSS + Claude | diff, keine Logik- oder DOM-Regression |
| G3 | Native Wails Windows 11: Play/Create/Edit/Training/Tools, 1366×768 und 1920×1080; Fokus, Kontrast, Disabled, Reduced Motion, Empty, Play/Pause/Resume/Stop | Claude + Owner | reale Captures aus Build + Testprotokoll |
| G4 | Video→ROI→klassischer Generator→.samn/.funscript→Edit→Play→Export/Import; fehlendes Modell, Audio, Video, ffmpeg; Undo, Cancel, Resume | Claude | reproduzierbare Testmatrix und Logs ohne private Videos |
| G5 | Neo2, Intiface und D2-Stroker entsprechend *tatsächlich* vorhandenem Code testen; No-Device/Disconnect/Emergency Stop/Cap-Limits | Claude + Owner | Mocktests und beaufsichtigter Hardware-Smoke |
| G6 | Free/Suite sauber getrennt: Build-Tags, nur Free im öffentlichen Publish, Edition-sicherer Updater, EULA, Notices, Donate-Link | Claude | Free-/Suite-Build-Report und geprüfte Release-Artefakte |
| G7 | Free-Dokumente, Handbuch, Changelog, README, Versions- und Taghinweise abgleichen; alte Paid-Studio-Behauptungen entfernen | ChatGPT Docs + Claude | einheitliche sichtbare Editionen- und Releaseinformationen |
| G8 | Öffentliche Website aktualisieren: reale Screenshots, bestätigte Downloads, aktuelle Features, bekannte Einschränkungen und Donate | ChatGPT Website + Owner | separater SamNPlayer-site-PR, Links nach Tagprüfung |
| G9 | Tag, Artefakte/Hashes und Update-Smoketest; erst danach Veröffentlichung | Claude + Owner | bestätigter Release-Tag, herunterladbare Free-Binaries, Website-Link |

## Priorisiertes Repository-Cleanup

1. **Erst Integrationen und QA:** #520 und #489 nicht aus reiner Mergeability mergen. Funktionierende React/Wails/Go-Wege nicht für kosmetisches Cleanup unterbrechen.
2. **Dokumentation isoliert korrigieren:** `docs/EDITIONS.md` und README sind im aktuellen Branch auf Free + Suite + Next 2.0 als Spenden-Freeware korrigiert. Diese Korrektur umfasst noch **keine** neue Veröffentlichung.
3. **#510 behandeln:** historisches Konzept in Next-Level-Archiv überführen beziehungsweise in einer neuen bereinigten Fassung verlinken; ursprüngliche paid-Studio-PR nach Owner/Claude-Abgleich als ersetzt schließen, nicht in main mergen.
4. **Branch-/PR-Aufräumen:** #509 hatte C1–C4 bereits integriert. Verwaiste GUI-PRs nur mit nachgewiesenem Supersession-Link schließen. Kein unkontrolliertes Löschen von History, Migrationsskripten, Tests, Drittanbieter-Nachweisen oder vorhandener Nutzerfunktion.
5. **Öffentliches Release:** `SamNPlayer-site` zeigt nur verifizierte Free-Downloads. Interne Suite-Dateien, geheimen Code, Datensätze/Modelle und Benchmarks nicht versehentlich publizieren.

## Definition of Done

**Free ist fertig**, wenn alle G1–G9 abgenommen sind. `VERSION=0.6.0`, ein grüner einzelner Workflow oder fertige Zielbilder alleine sind keine Release-Bestätigung. Der Nutzer erhält eine verlässlich benutzbare Anwendung; neues Next-Level-Experimentieren blockiert die Free-Veröffentlichung nicht.

**Wichtiger Schnitt:** Spendenbutton bleibt in Free; keine 7-Tage-Sperren, keine versteckte Paywall, keine €40-Entitlement-Implementierung. Spätere Community-Vereinfachung (weniger KI, ohne Batch) ist eine *separate* Owner-freigegebene Änderung nach diesem Release, mit verlustfreier Datenmigration und ehrlichen Release-Notes.
