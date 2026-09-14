# Nächste Arbeiten

## Verifizierter Ausgangsstand · 14. September 2026

- `main`: `5988203`, Release [v0.2.1](https://github.com/funfunpayer/SamNPlayer/releases/tag/v0.2.1).
- PRs #2, #3 und #4 sind integriert: Tf/Tj-Rezept, Generator-Anbindung,
  zweite Region in der GUI und `suction_position` im Playback.
- Tests und Release-Workflow für diesen Commit sind erfolgreich.
- Zum Zeitpunkt der Prüfung keine offenen Issues oder Pull Requests.
- Dieser Arbeitsstand korrigiert Versionen und Anleitungen und ergänzt die
  Versionsprüfung. Merge und CI des zugehörigen PR stehen noch aus.

Diese Datei führt die kurzfristigen Aufgaben. Architektur und Messergebnisse
stehen in `HANDOFF.md`, Arbeitsregeln in `CONTRIBUTING.md` und Setup in
`WIEDERAUFNAHME.md`. GitHub-Status vor jeder Fortsetzung erneut prüfen.

## Prioritäten und Abnahme

### 1. Echtes Gerät validieren · benötigt Sam Neo 2 und Anwender

BLE und Intiface getrennt prüfen: Verbinden, Wiedergabe, Pause/Stop,
Wiederverbinden und Training. Rohwert-Auflösung beider Kanäle protokollieren.
Für Tf/Tj prüfen, dass Vibration aus bleibt und die Sog-Ausgabe zum Signal passt.
Abnahme: Geräte-/Adapterdaten, App-Version, Schritte und beobachtetes Ergebnis
festhalten; Abweichungen als reproduzierbare Fehler erfassen.

### 2. Generatorqualität messen · benötigt bewertetes Testmaterial

Reproduzierbare Clips mit Sollsignal plus repräsentative reale Testclips
verwenden. Standard und Tf/Tj mit dokumentierten ROIs/Parametern vergleichen.
Rohsignal, Ergebnis, Qualitätsbericht und menschliches Urteil zusammenhalten.
Abnahme: begründete Verbesserungen an Rhythmus, Amplitude und Tracking;
synthetische Testergebnisse und reale Bewertungen getrennt ausweisen.

### 3. Automatische Zwei-ROI-Vorschläge verbessern

Vor Änderungen `find_two_rois` und vorhandene Tests prüfen. Automatische
Vorschläge gegen manuell gesetzte Regionen und bekanntes Sollsignal messen.
Abnahme: nachgewiesener Nutzen und erkennbare Unsicherheit; manuelle
Korrektur bleibt möglich. Kein ungeprüftes automatisches Profil-Umschalten.

### 4. Bewegungssignaturen und Profile in der GUI vervollständigen

Zuerst vorhandene Benennungs-/Speicherpfade gegen den Code prüfen.
Abnahme: Benennung und Parameter über Neustarts erhalten; Wiederverwendung
bei ähnlichen Szenen nachvollziehbar anbieten und ablehnen können.

### Später

- Script Doctor für importierte `.funscript`-Dateien.
- Trainingsverlauf über mehrere Sessions auswerten.
- KI als austauschbares Analyse-Backend: vorhandene Rohdaten, Parameter,
  Qualitätsberichte und bestätigte Urteile weiterverwenden.

## Produktvorgaben

Allgemeine Generatorqualität und das Ergebnis am Sam Neo 2 stehen im Zentrum.
Die Profile heißen `tf`/`tj`; ihr Rezept nutzt `suction_position` mit
Vibration aus. Weitere Parameter erst nach Messung kalibrieren. Bestehende
klassische Analyse bleibt als Grundlage für spätere KI erhalten.
