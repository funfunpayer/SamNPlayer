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
- S3 Player 3-column shell: IN PROGRESS (Claude) — CSS grid skeleton + Library/Inspector placeholders, timeline bottom, collapsible panels (state remembered in localStorage). Controls still in center/Inspector; full move to Devices/Audio/Playback/View tabs next.
- Next open: finish S3 grouping of existing controls into inspector tabs, then S4 timeline core.

(Full previous content restored in next commit if needed; this is the active board.)
