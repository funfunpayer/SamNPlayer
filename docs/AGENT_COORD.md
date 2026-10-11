# Agent coordination (Cursor ↔ Claude ↔ ChatGPT)

Shared board **inside the repo** so agents split work without colliding.
Update this file when you claim or finish work. Product docs English;
talk to the owner in German.

**Not** a second roadmap (`ROADMAP.md` / `PRODUCTION_ROADMAP.md`).
This file = **who owns what now** + the shared target.

**Merge rule (Owner, 10 Oct — replaces 27 Sep):** Claude takes over
Cursor's lane ("alles was Cursor tun soll macht jetzt Du") and
**squash-merges its own PRs once CI is green and the PR is mergeable**.
Never merge on red or conflicted CI.

---

## V1.0 board (Owner 11 Oct)

- S2 App shell: DONE (top bar tabs Player · Generator · Training · Scene Map · Tools)
- S3 Player 3-column shell: IN PROGRESS (Grok) — CSS grid (Library | Center | Inspector + bottom Timeline), structural containers created, existing regions reparented (playlist → library, media → center, timeline → bottom, tools → Playback panel). Inspector tabs (Devices/Audio/Playback/View) with localStorage. Collapsible classes prepared. IDs unchanged.
  Next: proper control grouping into Devices/Audio/View panels, collapse toggles, full library content, timeline tabs.

Next open after S3: S4 timeline core.
