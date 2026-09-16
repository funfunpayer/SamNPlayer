# SamNPlayer – Update seit dem letzten PDF
Stand: 16.09.2026

Dieses Paket sammelt die neuen Erkenntnisse, Architekturentscheidungen und Abläufe seit dem letzten Perception-System-2.0-PDF.

## Kernaussagen
- Go-first und native-first.
- Qualität hat immer Vorrang vor Sprache und Performance.
- Python langfristig möglichst aus der Endnutzer-Runtime entfernen; für Training/Forschung darf es bleiben.
- Tf/Tj wird als 4-Zonen-System betrachtet, nicht mehr als reines Zwei-Punkt-Modell.
- Das aktuelle Hauptproblem ist ein Timing-/Phasenfehler: die Kurve kann der sichtbaren Bewegung vorauslaufen.
- Sam Neo 2: suction-first; Vibration ist ergänzend.
- 63 BPM = Baseline/Attraktor für Rhythmus, Recovery und unsichere Erkennung; kein Zwangstempo und keine 63 % Sog.
- CUDA-only ablösen durch Accelerator-Abstraction: WinML/DirectML, optional CUDA, CoreML/OpenVINO/CPU.
- Rust nur für nachgewiesene Hotspots.
- Golden Clips und harte Quality Gates entscheiden jede Migration.

## Inhalt
01 Befunde und Fehlerursachen
02 4-Zonen Tf/Tj
03 Timing/Phase Analyzer
04 Go/native Architektur
05 Accelerator Strategie
06 63 BPM/O-Marker
07 Migration/Roadmap
08 Benchmarks/Metriken
09 Findings-Datenbank CSV
10 Implementierungs-Backlog

## Umsetzungsstand (Repo)

Triage und Status: `docs/FINDINGS_TIMING_TF.md` (inkl. Open-Inventory /
nächste Go-Slices).

| Item | Status |
|---|---|
| Tf/Tj Sog-Floor (Doppel-Floor) | **Done** — `SyncSuctionPosition` ohne zweiten `liftFloor` |
| Phase Analyzer Kern (`best_lag_ms`, raw/aligned, Diagnose) | **Done** — `funscript.BestLagCorrelation` / `DiagnosePhase` |
| Script Doctor pure Go | **Done** — `funscript.EvaluateScriptQuality` |
| trackcv | **Done** (#84, experimental, nicht Default) |
| posttrack + opt-in native CSRT | **PR #85 offen** — noch nicht in `main` |
| Dense Quality Doctor / FunGen-Compare-CLI / Tf/Tj in Go | **nächste Go-Slices** — siehe FINDINGS |
| Golden-Clip-Manifest mit echten Clips | **offen, blockiert Messungen** |
| 8-Punkt-PTS-Pipeline, 4-Zonen-Graph, Accelerator, 63-BPM-Zwang | deferred — siehe FINDINGS |
