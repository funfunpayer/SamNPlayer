# Go-first / Native-first

Wails UI
→ Go Application Core
  → Media/Timeline
  → Perception Orchestration
  → 4-Zone Logic
  → Signal Processing
  → Relative Motion
  → Fusion/Confidence
  → SAM Motion Engine
  → Quality/Benchmarks
  → Runtime/Scheduler
  → Device Layer
→ native Libraries / Model Runtime
  → FFmpeg
  → OpenCV/native Kernel wo sinnvoll
  → ONNX Runtime / Plattform-Accelerator

## Nach Go
Timeline, Pipeline, Observation-Schema, Track Memory, Relative Motion Graph, Cycle/Phase/Tempo, Confidence, Fusion, Quality, Benchmark, Profile, SAM State, Runtime, Device Response, Logging/Config/Cache.

## Nicht blind in pure Go neu schreiben
FFmpeg, hochoptimierte CV-/GPU-Kernel und neuronale Inference-Kernel. Go kapselt sie sauber.

## Python
Langfristig möglichst nicht für die normale Endnutzer-Runtime. Weiter sinnvoll für Training, Dataset Tools, Forschung und schnelle ML-Experimente. Erfolgreiche Modelle exportieren.

## Rust
Nur für profiler-belegte Hotspots, etwa SIMD/eigene native CV-Kernel. Keine zweite Hauptsprache ohne messbaren Nutzen.
