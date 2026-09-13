# Weiter ohne Neu-Erklärung

Quelle der Wahrheit: Branch `feat/tf-tj-suction`, PR #2.
Befehl an Grok in diesem Projekt-Chat: **weiter**.

## Offen (Reihenfolge)
1. `generator/generator.go`: volle Datei von main + Options.ROI2 + `--profile tf|tj` + `--roi2` in buildArgs (Branch hat immer wieder PLACEHOLDER)
2. `generator/generate_funscript.py`: volle Datei von main + choices tf/tj, Pos 20-90, metadata.profile/device_recipe (Branch hat PLACEHOLDER)
3. ~~`frontend/src/generator.js`: Profil Tf/Tj, 2. Region (Shift/Knopf)~~ **erledigt**
4. ~~`cmd/gui-wails/app_generator.go`: X2/Y2/W2/H2 → Options.ROI2~~ **erledigt**
5. `frontend/src/playback.js`: Option suction_position im Dropdown
6. `go test ./...` auf dem Branch — erst dann mergen

## Fest
- tf = tj, nur Sog, Vibration 0, MinSuction 0.20
- Keine ausgeschriebenen Akt-Wörter in UI
- GitHub connected, nicht main direkt anfassen
- Kern-Dateien nicht leeren: Inhalt von `main` behalten und nur die tf/tj-Stellen ergänzen
