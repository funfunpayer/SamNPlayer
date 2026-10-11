# SamNPlayer Flow Brand — owner-approved direction (2026-10-11)

**Status:** visual direction approved; implementation not yet merged. Coordinate with Claude (#477) before touching existing frontend branding (#469), Play layout (#470), multi-axis curves (#472) or RepoRefresh.

## Locked visual reference
Use the owner-approved **final banner** from 2026-10-11 11:39, not earlier concepts. The icon is a cyan glowing triangular Play core, wrapped by cyan and warm orange flowing triangular bands; no circular outer border, stars, particle dots, or ornamental separators. Banner lettering: **SamN** warm orange-red gradient, **Player** cyan-blue gradient; below: spaced **GENERATE PLAY TRAIN** (no dots). Background: midnight navy with 2–3 broad, restrained flowing ribbons. Keep this exact design; avoid generative redraw drift.

### Deliverables / assets
- App icon master PNG 1024×1024, Windows `.ico` 16–256, PNG 16/32/64/128/256/512, matching website banner.
- Generated concept image is a raster; do **not** call it a vector or replace existing assets until reviewed. Small-size readability must be checked at 16×16, 32×32 and Windows taskbar.
- Asset placement after Claude coordination: `build/appicon.png`, `build/windows/icon.ico`, `frontend/src/assets/images/logo-universal.png`, `website/media/logo.png` and public site media as appropriate. Check build references first.
- Avoid modifying `SamNPlayer-site` existing draft assets until the new source of truth is approved; previous public-site SVG does **not** match final concept.

## Application layout: airy, not boxed
- Deep navy surfaces; use implied divisions through spacing and low-contrast separators, not thick rectangular borders.
- Primary cyan #00D9F4, warm orange #FF8A23, dark navy #06121F; reserve purple/teal for distinguishable data series and accessible contrast.
- Top nav: **Play / Generator / Training / Tools** (matches Owner decision in #475 / #478 and `docs/PLAY_UI_CONTRACT.md` §2b). Scene Map lives inside Generator (Advanced) today; promoting it to a main tab needs a separate Owner decision.
- In-app tagline keeps **Generate · Play · Train** (with dots). Banner lettering may omit dots for visual style.
- Left library, right device/effects inspector, bottom timeline/scene thumbnails: each independently collapsible; persist layout when feasible; compact / studio / advanced can be proposed as layout presets, not assumed shipped.
- Keep video area visually dominant; when no media is loaded show branded abstract waves/logo **without implying video content**.
- Retain functional multi-axis lanes from #472 with shared playhead, legible scale, axis visibility, colored legends, and scene thumbnails only when supported by real data.
- Generator, Training, Scene Map, Settings should reuse the same spacing, typography, tokens and interaction affordances, without fictional controls.
- Motion: subtle, respect reduced-motion; keyboard access, focus states, screen reader labels and adequate contrast.

## Engineering and acceptance
1. Claude confirms files/branch ownership on #477 before implementation.
2. Use small, reviewable PRs: branding asset integration → shared tokens → Play polish → other tabs.
3. No broken selectors/JS bindings; run relevant frontend tests including `pb_tools_toggle_test.py` and `pb_axis_lanes_test.py`, screenshot QA at 1366×768 and 1920×1080.
4. Do not overwrite private `website/` through `scripts/sync-public-site.sh` until public landing sync conflicts are reconciled.
5. The approved generated banner/icon and ICO/PNG exports are attached in the ChatGPT handoff, **not yet uploaded to GitHub**. Copy the actual reviewed binary assets into the implementation PR, do not substitute a new generated image.

**Owner decision:** lock the final orange/cyan SamNPlayer brand; no further spontaneous icon or lettering changes.
