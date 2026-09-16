# Migration Roadmap

P0 Qualität:
- 4-Zonen-Datenfluss instrumentieren
- echte PTS prüfen
- ~~Phase Analyzer~~ **Done (Kern in Go)** — 8-Punkt-Pipeline offen
- ~~Sog-Floor prüfen~~ **Done (Doppel-Floor entfernt)** — Hardware-Feel offen
- Normalisierung/Missing Data
- Turning Points/Cycles
- Device Mapping separat messen

P1 Benchmark:
Golden Clips, Referenzen, feste Config, Commit Hash, Hardware, reproduzierbare Scores.

P2 Go Motion Core:
relative, velocity, acceleration, jerk, turningpoint, cycle, phase, tempo, hysteresis, confidence, prediction; signal: adaptive filter, percentile, noise, envelope, resample.

P3 Observation Contract:
timestamp, source, zone/track, position, velocity, confidence, valid, quality_flags, provenance.

P4 Python-Monolith modularisieren ohne Verhaltensänderung; dann Modul für Modul benchmarken und portieren.

P5 Python-freie normale Runtime, sobald Quality Gates bestehen.

P6 Accelerator Layer: ONNX Runtime + WinML/DirectML + CUDA optional + CoreML/OpenVINO/CPU.

P7 Advanced Perception erst danach: moderner Tracker, SAM 2/2.1 selektiv, Pose, Depth, ReID, Prediction.

P8 Rust nur für gemessene Hotspots.
