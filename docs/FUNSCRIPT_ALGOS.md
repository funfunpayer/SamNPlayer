# Funscript-Algorithmen in SamNPlayer

Research-Überblick (Optical Flow, YOLO, Audio, Deep Learning, Filter-Pipelines)
übersetzt in **das, was SamNPlayer tatsächlich macht** — und was bewusst
nicht.

Stand: September 2026. Kein FunGen-Quellcode; nur öffentlich beschriebene
Ideen und eigene Messungen.

## Architektur-Regel

1. **Klassisches Tracking schreibt das Funscript** (CSRT / Flow / Grid-LK / Tf-Tj).
2. **KI schlägt nur ROI / Meinung vor** — nie die Positions-Kurve.
3. **Post-Processing** (Savgol, Detrend, Bandpass, Peaks, RDP, Speed-Cap)
   sitzt in `generator/posttrack` (Go) und spiegelt sich in
   `generate_funscript.py`.

## Empfohlener Multi-Modal-Workflow

Der „Tipp am Ende“ aus der Research: **mehrere Stufen kombinieren**, nicht
eine Methode allein.

| Stufe | Was | Warum |
|-------|-----|--------|
| 1. Scout | Backend `flow` mit `--flow-downscale 0.5` | Schneller Überblick, keine ROI nötig |
| 2. ROI | Manuell oder KI-Vorschlag | Fokus auf die relevante Bewegung |
| 3. Track | `csrt` (Standard) oder Tf/Tj | Robuste Kurve, Go-native wo möglich |
| 4. Autotune | Profil `autotune` | Detrend 3s + Bandpass 0.5–4 Hz + Speed 400 |
| 5. Check | Quality Doctor + optional Audio | Drift, Tempo, Geräte-Sicherheit |

CLI-Beispiel:

```bash
# Schneller Flow-Entwurf
python3 generator/generate_funscript.py VIDEO --backend flow --flow-downscale 0.5 -o draft.funscript

# Feinschliff mit CSRT + Autotune-Nachbearbeitung
python3 generator/generate_funscript.py VIDEO --roi x,y,w,h --profile autotune -o out.funscript
```

GUI: Bewegungsart **Autotune**, optional **Max. Speed** und **Flow-Downscale**
unter Erweitert.

## Was wir aus der Research übernehmen

| Idee | Umsetzung in SamNPlayer |
|------|-------------------------|
| Optical-Flow + Downsample | `flow` + `--flow-downscale` |
| Savitzky–Golay | `--smooth-window` / `posttrack` |
| Detrend (Drift) | `--detrend-ms` / `DetrendWindowMs` |
| Bandpass ~0.5–4 Hz | `--bandpass-hz LOW,HIGH` |
| RDP Keyframe-Reduktion | `--rdp-tolerance` |
| Speed Limiter | `--max-speed` / GUI Max. Speed |
| Ultimate-Autotune-Rezept | Profil `autotune` |
| Multi-Modal | Workflow oben (Flow → CSRT → Autotune → Check) |

## Was wir bewusst nicht kopieren

| Idee | Warum nicht (jetzt) |
|------|---------------------|
| YOLO-Körperteil-Tracking als Script-Writer | Lizenz/Scope; KI bleibt ROI-Hilfe |
| End-to-End DeepFunGen | Braucht große Datensätze; Qualität unkontrolliert |
| NVOFA-Hardware-Flow | Optional später; Farneback+Downscale deckt den Win |
| Funscript-Flow „Divergence Center“ | Bei uns schlechter gemessen (siehe HANDOFF) |

## Post-Processing-Reihenfolge

```
roh (px)
  → Savitzky–Golay
  → Rolling Detrend          (optional / autotune)
  → Bandpass 0.5–4 Hz        (optional / autotune)
  → Dynamic-Range-Lift       (optional)
  → Perzentil-Normalisierung 0–100
  → Peaks/Valleys + Prominence
  → Adaptive Keyframes + Min-Interval
  → RDP
  → Speed-Cap                (optional / autotune 400)
  → .funscript
```

## Mess-Hinweis

Jede neue Filterstufe braucht einen Vorher/Nachher-Lauf auf demselben Clip
(`Quality Doctor`-Score + Sichtprüfung der Heatmap). Autotune ist ein
**Rezept**, kein Garant — bei weichem Gewebe weiter Profil `weich` nutzen.
