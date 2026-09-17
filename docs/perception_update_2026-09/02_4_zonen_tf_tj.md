# 4-Zonen Tf/Tj

## Pipeline
Video → 4 Zonen → Tracking/Flow pro Zone → Relative Motion Graph → Global Motion Separation → relevante lokale Bewegung → Cycle/Phase → Contact Confidence → Suction Envelope → Device Response.

Vier Zonen erzeugen sechs Beziehungen:
R1-R2, R1-R3, R1-R4, R2-R3, R2-R4, R3-R4.

Pro Zone: Position, Velocity, Acceleration, Flow, Stability, Occlusion, Lost, Confidence, History.

Pro Paar: dx/dy, Distanz, Closing Velocity, relative Acceleration, Winkel, Korrelation, Phase Agreement, Confidence.

Gemeinsame Bewegung aller Zonen ist eher globale Kamera-/Körperbewegung. Differenzielle Bewegung ist für Tf/Tj interessanter.

Eine schlechte Zone darf das Gesamtsignal nicht zerstören. Confidence-weighted Consensus statt Gleichgewichtung.

## Tf/Tj v2
4 Zones → Relative Motion → robust distance/vector → Noise Floor → Dynamic Deadband → Turning Points → Cycle → Phase → Contact Confidence → Motion Envelope → Suction Mapping → Sam Neo 2 Response.
