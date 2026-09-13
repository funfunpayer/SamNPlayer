# Tf / Tj

Intern `tf` und `tj` – dasselbe Rezept.

## Signal
Zwei Regionen, Abstand. Enge Distanz = hohe pos = starker Sog.

## Neo 2
- Sync: `suction_position` (Sog = Position, Vibration = 0)
- MinSuction 0.20
- Tick 50 ms, Smoothing 0.22
- Pos-Klammer 20–90

## Bedienung
Generator: Profil „Tf / Tj“, 1. und 2. Region (Shift oder Knopf).
Playback: Rezept aus Metadata, UI-Default `independent` ändert das nicht.

## CLI
`--profile tj --roi x,y,w,h --roi2 x,y,w,h`
