# Weiter

Branch `feat/tf-tj-suction`, PR #2. Im Chat: **weiter**.

## STOP — Dateien sind leer
Auf dem Branch sind aktuell 0 Byte:
- `generator/generator.go`
- `generator/generate_funscript.py`
- `cmd/gui-wails/frontend/src/generator.js`

Ohne Restore baut nichts. Einmal lokal:

```
git fetch origin
git checkout feat/tf-tj-suction
git checkout origin/main -- generator/generator.go generator/generate_funscript.py cmd/gui-wails/frontend/src/generator.js
git add generator/generator.go generator/generate_funscript.py cmd/gui-wails/frontend/src/generator.js
git commit -m "restore: Kern-Dateien von main"
git push origin feat/tf-tj-suction
```

Danach wieder **weiter**. Grok spielt dann nur die kleinen Tf/Tj-Diffs drauf.

## Danach offen
1. Options.ROI2 + `--profile tf|tj` + `--roi2`
2. Python: choices tf/tj, Pos 20-90, device_recipe
3. GUI: Profil Tf/Tj + 2. Region
4. playback.js: suction_position
5. `go test ./funscript/ ./generator/` — erst dann mergen

## Fest
- tf = tj, Sog aus Position, Vibration 0, MinSuction 0.20
- keine ausgeschriebenen Akt-Wörter
- nicht main direkt anfassen
