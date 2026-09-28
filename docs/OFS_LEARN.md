# OFS / Funscript — was wir für SamNPlayer nutzen

Kurzer Stand der OFS- und Funscript-Recherche (kein Release-Versprechen).
Details und Quellen: [FUNSCRIPT_RESEARCH.md](./FUNSCRIPT_RESEARCH.md), [HEATMAP.md](./HEATMAP.md).

## Bereits im Player

| Thema | Status |
|-------|--------|
| Multi-Achsen (`.vibration` / `.suction` / `.samn`) | Produkt — Play + Create |
| Bookmarks / Chapters (metadata) | Play — laden/speichern |
| Heatmap (PNG + Canvas) | Play — Export + Live-Leiste |
| Projekt-Sidecar `.snp.json` | Play — speichern/laden |
| Range-Edit (Cap / Scale / Delete) | Play — **axis-aware** (aktive Curve: general/vibration/suction); Scale mit Faktor-Slider ×0.5–×1.5 + optional Soft edges |
| Frame-Snap | Play — optional am Scrubber |

## Noch nicht (bewusst später)

- Volles OFS-`project.json` (Legacy) — unser Sidecar reicht für Emotion
- T-Code / Handy-Device-Protokoll — getrennt von Funscript-Format
- Community-"ScriptPlayer"-Features 1:1 — nur übernehmen, was UX hilft

## Nächste sinnvolle Scheiben (Board)

1. ~~Cap-Speed / Scale auf Vib+Suction-Achsen (nicht nur stroke)~~ → **DONE** (Play axis Cap/Scale/Delete)
2. Chapter-Leiste klickbar → Seek (teilweise vorhanden)
3. Heatmap-Farben an OFS-Referenz angleichen (Doku in HEATMAP.md)
