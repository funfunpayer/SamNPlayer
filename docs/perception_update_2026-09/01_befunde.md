# Neue Befunde und Fehlerursachen

## 1. Tf/Tj: Phase statt nur Tempo
Die Kurve läuft der sichtbaren Bewegung teilweise voraus. Deshalb getrennt messen:
Video Event → Observation → Filter → Turning Point → Funscript Action → Runtime Command → physischer Device-Effekt.

Der wichtigste neue Messwert ist `best_lag_ms`. Steigt die Korrelation nach zeitlicher Ausrichtung stark, ist die Kurvenform brauchbar und primär die Phase falsch.

## 2. Sog-Mapping
Aus der bisherigen Codeanalyse: Tf/Tj nutzt einen Positionsbereich 20–90 und `min_suction=0.20`. Wenn der Positionswert bereits 0.20 beträgt und anschließend nochmals ein 20-%-Floor angewendet wird, ergibt sich effektiv 0.36. Bei 0.90 entstehen 0.92. Das kann einen unnötig hohen Dauer-Sog erzeugen und muss A/B-getestet werden.

## 3. tick_ms ist kein Tempo
50 ms = 20 Hz Command-Update. 63 BPM = ca. 952 ms pro Zyklus. Diese Größen dürfen nicht gekoppelt werden.

## 4. Smoothing
Ein kausaler Filter kann Peaks verschieben. Offline-Generation darf phasenneutrale Verfahren nutzen; Live-Runtime braucht kausale Filter. Filterqualität muss zusätzlich mit Peak-/Turning-Point-Timing bewertet werden.

## 5. Timeline
FrameIndex/FPS kann bei VFR und exakter Synchronität ungenau sein. Echte Decoder-PTS sollten bevorzugt durch die Pipeline getragen werden.

## 6. Tracking Loss
Das Wiederholen der letzten Position ist kein echter Messwert. Observation benötigt `valid`, `confidence`, `reason` und Provenance.

## 7. Camera Compensation
Fehlerzustand und echte Nullbewegung dürfen nicht beide als 0.0 dargestellt werden. Affine Information wie X, Y, Rotation und Scale sollte nicht unnötig auf nur Y reduziert werden.

## 8. Normalisierung
Min/Max auf kurzen/verrauschten Szenen kann Rauschen aufblasen. P05/P95, MAD-Noise-Floor und scene-stabile Referenzen benchmarken.

## 9. Fusion
Kein blindes Mitteln. Bei starkem Quellen-Widerspruch Reanalyse/Fallback. Agreement selbst ist ein Qualitätssignal.

## 10. Sprache
Das aktuelle Qualitätsproblem ist kein Python-vs-Go-Problem. Go verbessert Runtime, Packaging und deterministische Verarbeitung; Messmodell und Timing müssen trotzdem korrekt sein.
