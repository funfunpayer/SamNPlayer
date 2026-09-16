# Accelerator Strategie

CUDA-only → hardwareunabhängige Accelerator-Abstraction.

Go Accelerator Interface
→ Windows: WinML/DirectML
→ NVIDIA: CUDA optional
→ macOS: CoreML
→ Intel: OpenVINO wo sinnvoll
→ CPU fallback

Provider-Auswahl:
detect → warmup → inference benchmark → RAM/VRAM → numerische Output-Toleranz → schnellsten qualitätsgleichen Provider wählen → Auswahl cachen.

CUDA wird nicht verboten. Auf NVIDIA darf CUDA gewinnen. AMD/Intel sollen ohne NVIDIA-Zwang performant laufen.
