# Flow theme proposals for Generator and Training

Lightweight, non-imported CSS proposals that reuse the locked Flow tokens
(`docs/design/flow-theme-tokens.css`) and the airy layout principles from
`docs/BRAND_FLOW_UI_DIRECTION_2026-10.md`.

**Status:** design notes only. Claude implements app-wide token mapping;
these files show how Generator (`#tab-generator`) and Training (`#tab-training`)
can adopt the same surfaces, spacing, radii and accents without inventing
controls or changing IDs/JS.

- Generator: step panels, cards and primary buttons map to Flow tokens.
  Scene Map remains inside Generator (Advanced) until a separate Owner decision.
- Training: cards, progress indicators and visual areas follow the same tokens.
  No new fictional features.

See also `docs/design/generator-layout-proposal.css` and
`docs/design/training-layout-proposal.css`.

**Coordination:** same branch ownership as the rest of the design-spec PR.
Assets still pending Owner paste to #477.
