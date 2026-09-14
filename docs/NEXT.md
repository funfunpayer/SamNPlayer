# Weiter

Branch `feat/tf-tj-suction`, PR #2. Chat: **weiter**.

## Status
- Rezept `tf`/`tj` + `SyncSuctionPosition` + Playback-Metadata: da
- `generator.go`: `--profile tf|tj` + `--roi2` da; ROI2 muss Wert-Typ sein (wie app_generator)
- `generate_funscript.py`: auf dem Branch nur Placeholder — von **main** holen, dann Tf/Tj-Diff
- `generator.js`: 0 Byte — von **main** holen, dann Profil + 2. Region
- `playback.js`: `suction_position` fehlt noch

## Restore (einmal)
```
git checkout feat/tf-tj-suction
git checkout origin/main -- generator/generate_funscript.py cmd/gui-wails/frontend/src/generator.js
git commit -m "restore py+js von main" && git push
```
Danach wieder **weiter**.

## Fest
- tf = tj, Sog=Position, Vib=0, MinSuction 0.20
- keine ausgeschriebenen Akt-Wörter
- keine Placeholder in Kern-Dateien
