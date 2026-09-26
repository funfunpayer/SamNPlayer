# GUI makeup — professional polish ideas

Visual language: **dark ink + soft lilac accent + teal secondary** (`style.css`
tokens — Look v3, Owner 26 Sep: lila preferred over honey orange). Emotion fonts:
**Sora + Figtree**. No card-grid shell, no light mode.

## Shipped

### Look v1 (#225)
- Calmer rail / active tab; primary CTA; content vignette; quieter Play empty

### Look v2 (#259)
- Thinner chrome, organic radii, warmer honey accent (superseded by v3)

### Look v3 (this branch)
- **Lilac accent** (`#b896e8`) replacing honey chrome — help chips / CTAs / washes
- Teal kept as secondary; warn stays warm amber for alerts
- **Motion:** tab in, Create step panel enter, help popover fade, primary press
- `prefers-reduced-motion` honored
- Figure heat ramp: teal → lilac → coral (Device / Training shared tokens)

## Ideas backlog (next slices)

| Idea | Why | Risk |
|------|-----|------|
| Play first viewport brand moment | Competitors judge players first | Don’t add stats/chips |
| Settings density | Pro tools look | English copy only |
| Bookmarks UI (API exists) | OFS parity | Keep Play simple |
| Screenshot pack for README | Public face | Match Look v3 |

## Explicit non-goals

- Frameless glass everywhere  
- Rounded-full pill clusters / emoji decoration  
- Dashboard layout on Play  
- Redesigning Generate flow for “pretty” only  
- Cream-terracotta / broadsheet AI-default looks  
- Defaulting AI draft on (Everyday CSRT stays)
