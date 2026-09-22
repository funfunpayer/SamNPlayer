# GUI makeup — professional polish ideas

Visual language stays **dark ink + amber + teal** (`style.css` tokens).
No purple redesign, no card-grid shell, no light mode
(`docs/COMPETITIVE.md`, `docs/ROADMAP.md`).

## Shipped (this pass)

- Calmer rail / active tab (less halo, clearer amber edge)
- Primary CTA: crisp hover/press (no endless float)
- Content pane: soft vignette + steadier type rhythm
- Topbar: slightly clearer glass edge
- Play empty: quieter presence (inset wash, no bobbing)

## Ideas backlog (next slices)

| Idea | Why | Risk |
|------|-----|------|
| **Play first viewport only** — brand moment + one CTA + video plane | Competitors judge players first | Don’t add stats/chips to hero |
| **Generate step chrome** — numbered rail already there; tighten spacing + one progress tone | Everyday feel | Keep CSRT path untouched |
| **Device silhouette fill** — already Claude’s idea; sync colors with Contact amber/teal | Hardware trust | Mock-testable only |
| **Log as “studio tape”** — monospace strip, severity tint | Support/debug | Don’t make Log the home |
| **Settings density** — group with hairline rules, not more cards | Pro tools look | English copy only |
| **Screenshot pack** for README | Public face | Match current UI after polish |
| **Motion budget** — tab in, primary press, device LED only | Presence without noise | Honor `prefers-reduced-motion` |

## Explicit non-goals

- Frameless glass / acrylic everywhere  
- Rounded-full pill clusters / emoji decoration  
- Dashboard layout on Play  
- Redesigning Generate flow for “pretty”
