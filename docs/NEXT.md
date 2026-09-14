# Weiter

## Stand
- `generate_funscript.py` ist wieder vollständig (kein Placeholder).
- `generator.js` ist wieder da (nicht leer).
- Backend: Tf/Tj (Abstand + Sog), ROI2, `suction_position`.

## Dieses PR (`feat/tf-tj-gui`)
- GUI: Profil Tf/Tj, zweite Region (roi2, Shift+Ziehen oder „2. Region“), Payload `x2/y2/w2/h2`
- Playback: Sync-Option `suction_position` („Sog aus Position“)

## Noch offen
- CI: PR #3 (ungenutztes `strconv` + `per_scene_roi_test` OpenCV-Baseline)
- Optional: Branch-Aufräumen (`feat/tf-tj-suction` und alte Placeholder-Reste)

## Fest
- tf = tj, nur Abstand + Sog (`suction_position`), Vib konzeptionell 0
- kein Akt-Detektor, keine ausgeschriebenen Akt-Wörter
