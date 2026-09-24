# GUI makeup — professional polish ideas

Visual language stays **warm ink + honey amber + soft teal** (`style.css`
tokens). Emotion fonts: **Sora + Figtree**. No purple redesign, no card-grid
shell, no light mode (`docs/COMPETITIVE.md`, `docs/ROADMAP.md`).

## Shipped

### Look v1 (#225)
- Calmer rail / active tab (less halo, clearer amber edge)
- Primary CTA: crisp hover/press (no endless float)
- Content pane: soft vignette + steadier type rhythm
- Topbar: slightly clearer glass edge
- Play empty: quieter presence (inset wash, no bobbing)

### Look v2 (this branch — Owner brief 24 Sep)
- **Thinner chrome:** narrower rail (`76px`), 1px rail edge, lighter borders
- **Friendlier palette:** warmer honey accent, softer teal / feel, gentler ink
- **Organic radii:** `--radius` 22px / `--radius-sm` 14px; brand strokes pill-soft
- **Subtle graphics:** extra soft wash gradients; thinner angled brand strokes
- **Unchanged:** Sora/Figtree; Generate / CSRT behaviour; no second theme fork

## Ideas backlog (next slices)

| Idea | Why | Risk |
|------|-----|------|
| **Play first viewport only** — brand moment + one CTA + video plane | Competitors judge players first | Don’t add stats/chips to hero |
| **Generate step chrome** — numbered rail already there; tighten spacing + one progress tone | Everyday feel | Keep CSRT path untouched |
| **Device silhouette fill** — sync colors with Contact amber/teal | Hardware trust | Mock-testable only |
| **Log as “studio tape”** — monospace strip, severity tint | Support/debug | Don’t make Log the home |
| **Settings density** — group with hairline rules, not more cards | Pro tools look | English copy only |
| **Screenshot pack** for README | Public face | Match UI after Look v2 |
| **Motion budget** — tab in, primary press, device LED only | Presence without noise | Honor `prefers-reduced-motion` |

## Explicit non-goals

- Frameless glass / acrylic everywhere  
- Rounded-full pill clusters / emoji decoration  
- Dashboard layout on Play  
- Redesigning Generate flow for “pretty”
- Purple / cream-terracotta / broadsheet AI-default looks
