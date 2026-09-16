# Tf / Tj

Internally named `tf` and `tj`; both use the same recipe. The generator's
profile dropdown shows a single "Tf/Tj (Abstand + Sog)" entry (`value="tf"`)
rather than two identical-behaving options - the CLI's `--profile` still
accepts either name for backward compatibility.

## Signal

Track two regions and measure their distance. A smaller distance means a
higher position value and stronger suction.

## Neo 2

- Sync: `suction_position` (suction follows position; vibration = 0).
- MinSuction: 0.20.
- Tick: 50 ms; smoothing: 0.22.
- Position clamp: 20–90.

## Usage

Generator: mark the first and second regions using Shift or the
region-selection button - marking a second region automatically selects the
Tf/Tj profile (no other tracking method evaluates a second region, so
marking one already is the selection). Selecting the profile by hand still
works and pre-selects the second-region-drawing mode either way.
Playback: uses the recipe from metadata; the UI default `independent`
does not override it.

## CLI

```text
--profile tj --roi x,y,w,h --roi2 x,y,w,h
```
