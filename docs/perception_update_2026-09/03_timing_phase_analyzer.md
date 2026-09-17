# Timing/Phase Analyzer

## Messpunkte
1 Decoder PTS
2 Observation Timestamp
3 Filter Output Timestamp
4 Turning Point Timestamp
5 Funscript Action Timestamp
6 Runtime Scheduled Timestamp
7 Device Command Timestamp
8 optional physische Device-Reaktion

## Metriken
best_lag_ms; peak timing error; valley timing error; turning-point MAE; phase error; cycle-duration error; correlation raw/aligned; false micro-cycle rate; command jitter; end-to-end latency.

## Diagnoseablauf
A. Rohsignal gegen Video/Referenz.
B. Bestes zeitliches Lag bestimmen.
C. Wenn aligned correlation deutlich besser wird: Timing reparieren.
D. Wenn auch aligned schlecht: Perception/Normalisierung/ROI-Semantik reparieren.
E. Erst danach Device-Latenz kompensieren.
F. Prediction niemals verwenden, um einen Generatorfehler zu kaschieren.

## Repo-Umsetzung
- Kern B+C/D: `funscript.BestLagCorrelation` + `DiagnosePhase`
  (Port von `generator/fungen_compare.best_lag_correlation`).
- Tests: `funscript/phase_test.go` (Shift / Invert / Konstante /
  Low-Confidence / Diagnose).
- Noch offen: Messpunkte 1–8 (PTS → Device) als durchgängige Pipeline.
