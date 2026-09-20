# Tf / Tj

Internally named `tf` and `tj`; both use the same recipe. The generator's
profile dropdown shows a single "Tf/Tj (distance + suction)" entry
(`value="tf"`) rather than two identical-behaving options — the CLI's
`--profile` still accepts either name for backward compatibility.

## Signal

Track the tip region plus one or more contact targets and measure
**min** tip→partner distance. Distance uses the tip-box point **nearest
the partner center** (not tip center), so a whole-penis mark still
behaves like glans contact. A smaller distance means a higher position
value, stronger suction, and (with contact vibration) more vibration.

## Body parts (multi-region)

Anonymous ROI1/ROI2 still work. Prefer canonical English classes
(`docs/BODY_REGIONS.md`):

| Role | Typical class | Behaviour |
|------|---------------|-----------|
| Tip (ROI1) | `glans` / `penis` / `hand_1` | **Tracked** |
| Target (ROI2) | `nipples` / `mouth` / `vagina` / `breasts` | Often **fixed** (`--roi2-fixed`) |
| Extra targets | further contact classes | **Fixed** by default (`--target`, repeatable); distance = min over all |
| Soft masks | `face` / busy background | Feature punch-out only (`--mask`); not a stroke axis |

## Neo 2

- Sync: `suction_position` (suction follows position; vibration = 0).
- Script position clamp: 20–90 (generation time). That clamp *is* the
  suction floor in command space (`pos/100` → 0.20–0.90).
- Recipe metadata still records `min_suction: 0.20` for documentation;
  playback must **not** apply a second `liftFloor` on top (that remapped
  0.20→0.36 and 0.90→0.92 — see `docs/FINDINGS_TIMING_TF.md`).
- Tick: 50 ms (= 20 Hz command rate, **not** tempo); smoothing: 0.22.
  63 BPM ≈ 952 ms/cycle is unrelated to `tick_ms`.

## Usage

Generator: mark the first and second regions using Shift or the
region-selection button — marking a second region automatically selects the
Tf/Tj profile. Optionally set ROI1/ROI2 class and **Fix ROI2** for a static
contact target. Use **+ Target** for further min-distance partners and
**+ Mask** for soft feature excludes. Selecting the profile by hand still
works and pre-selects the second-region-drawing mode either way.
Playback: uses the recipe from metadata; the UI default `independent`
does not override it.

## CLI

```text
--profile tj --roi x,y,w,h --roi2 x,y,w,h --roi2-fixed \
  --target x,y,w,h --mask x,y,w,h \
  --region-class glans --region-class2 nipples
```
