# Tf / Tj

Internally named `tf` and `tj`; both use the same recipe.

## Signal

Track two regions and measure their distance. A smaller distance means a
higher position value and stronger suction.

## Neo 2

- Sync: `suction_position` (suction follows position; vibration = 0).
- MinSuction: 0.20.
- Tick: 50 ms; smoothing: 0.22.
- Position clamp: 20–90.

## Usage

Generator: select the Tf / Tj profile, then mark the first and second regions
using Shift or the region-selection button.
Playback: uses the recipe from metadata; the UI default `independent`
does not override it.

## CLI

```text
--profile tj --roi x,y,w,h --roi2 x,y,w,h
```
