# Weiter ohne Neu-Erklärung

Quelle der Wahrheit: Branch `feat/tf-tj-suction`, PR #2.
Befehl an Grok in diesem Projekt-Chat: **weiter**.

## Offen (Reihenfolge)
1. `generator/generator.go`: Options.ROI2 + `--profile tf|tj` + `--roi2` in buildArgs
2. `generator/generate_funscript.py`: choices tf/tj, Pos 20-90, metadata.device_recipe
3. `frontend/src/generator.js`: Profil Tf/Tj, 2. Region (Shift/Knopf)
4. `frontend/src/playback.js`: Option suction_position
5. `go test ./funscript/ ./generator/` — erst dann mergen

## Fest
- tf = tj, nur Sog, Vibration 0, MinSuction 0.20
- Keine ausgeschriebenen Akt-Wörter in UI
- GitHub connected, nicht main direkt anfassen
