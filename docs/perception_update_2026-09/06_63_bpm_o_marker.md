# 63 BPM / O-Marker / Recovery

63 BPM = SAM-Design-Baseline.
Periode ≈ 952.38 ms; Halbperiode ≈ 476.19 ms.

Nicht verwechseln:
63 BPM ≠ 63 % Sog
63 BPM ≠ tick_ms
63 BPM ≠ erzwungenes Videotempo

Verwendung:
- Startprior der Tempoerkennung
- Stabilitätsanker bei niedriger Confidence
- Recovery-Ziel nach O-Marker
- Schutz gegen hektische Fehlmessungen

State:
BASELINE → BUILDUP → HIGH → O_MARKER → RECOVERY → BASELINE

63 ist ein Attraktor. Bei hoher Video-Confidence folgt SAM dem Video; bei unsicherer Erkennung wächst das Baseline-Gewicht.

Tf/Tj bleibt suction-first.

Cycle State:
IDLE, APPROACHING, CONTACT, HOLD, RETREATING, TURNING, UNKNOWN.
Dazu TempoEstimate, TempoConfidence, Phase, CycleDuration, ApproachDuration, ContactDuration, RetreatDuration, Amplitude.
