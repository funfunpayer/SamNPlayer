# Agent coordination (Cursor ↔ Claude ↔ ChatGPT)

Shared board **inside the repo** so agents split work without colliding.
Update this file when you claim or finish work. Product docs English;
talk to the owner in German.

**Not** a second roadmap (`ROADMAP.md` / `PRODUCTION_ROADMAP.md`).
This file = **who owns what now** + the shared target.

---

## North star (do not lose this)

```text
  Ship a boring, reliable Generate → funscript → Neo 2 path
  that feels right without forcing the user to mark everything.

  Order (PRODUCTION_ROADMAP):
    G0 stable CSRT  →  G1 classical heuristics + audio
                    →  G2 Neo2 feel  →  G3 AI helpers only
```

Near-term product slice (owner-locked): `docs/TFTJ_PROFILE_DIRECTION.md`
— Tf/Tj = **profile/feel**, contact vib on Normal (done #149), mark partner
only when vib needs it, no-mark classical toward FunGen-like UX.

Perception research (do **not** leapfrog product): `docs/SAM_ARCHITECTURE.md`
§ Perception v1 — bake-off observers **before** any Go port / fusion default.

**Rule:** piece by piece. One improvement ships and is measured before the
next big theme. Prefer cleanup + focus over parallel feature sprawl.

---

## Sprint order (21 Sep — all agents)

Do in this order unless the owner says otherwise:

| # | What | Who | Why first |
|---|------|-----|-----------|
| **0** | **Cleanup** — stale PRs, wrong docs, board on `main` | Claude (+ ChatGPT E) | Claude already started (#150). Clear fog before new code. |
| **1** | Finish **bake-off** (measurement only) | Claude lane B | Data for SAM v1; no product default change |
| **2** | **v0.5.17** after owner smoke | Cursor lane A | Ship contact-vib Normal; portable Go CSRT check |
| **3** | **TFTJ step 3** partner-mark when vib on | ChatGPT or Cursor lane C | Next locked UX slice |
| **4** | Metadata stamp bug / #145/#119 triage | ChatGPT lane E | Hygiene; does not block 2–3 if slow |

Improvements are **decided together** by writing proposals into this file
or a short PR description — owner confirms; then one agent implements.

---

## Cleanup checklist (Claude owns — do now / next)

- [x] #150 provenance fix for `clip_ausschnitt` (merged)
- [ ] Land / sync **#151** `AGENT_COORD` on `main` (Cursor PR — merge when CI green)
- [x] **#146 is closed** — verified through GitHub on 21 Sep; no further close action needed.
- [ ] After bake-off: write results only into `SAM_ARCHITECTURE.md` + short NEXT note — no eleventh doc
- [ ] Flag leftover wrong claims in NEXT/FINDINGS if bake-off contradicts #148-era text
- [ ] Do **not** start Go ports of flow/grid_lk during cleanup

When cleanup + bake-off PR are up, mark rows Done below and free lane B.

---

## Lanes

| Lane | Owner default | Scope | Forbidden while busy |
|------|---------------|-------|----------------------|
| **A** Product / release | Cursor | VERSION, GUI, release.yml, smoke notes | Parallel bake-off edits to same release notes |
| **B** Measurement / goldens | Claude | FunGen compare, bake-off tables, golden README | Changing Generate defaults |
| **C** Tf/Tj UX | ChatGPT or Cursor | Step 3 partner-mark, profile rename | SAM fusion |
| **D** Perception / SAM impl | *after* bake-off win | Fusion, Go observer ports | Before numbers exist |
| **E** Bugs / hygiene | ChatGPT | Metadata stamp, close stale PRs, issue triage | Touching B’s running jobs |

---

## Active

| Lane | Owner | Branch / PR | Goal | Status |
|------|-------|-------------|------|--------|
| B | Claude | background job → PR later | Bake-off flow/grid_lk/region_fusion vs FunGen2 on clip_voll then clip_ausschnitt | **RUNNING** (flow on ~280s clip) |
| A | Cursor | #151 | Board + later 0.5.17 bump | Board PR open; release **after owner smoke** |
| E | ChatGPT | `codex/agent-coordination-lane-e` | Triage #145/#119 and investigate metadata provenance | **Claimed** — initial issue review recorded below; no code changes yet |
| C | ChatGPT (claim after 0 or with Cursor) | — | TFTJ step 3 | **Queued** until cleanup#0 + prefer after 0.5.17 |

---

## ChatGPT handoff — 21 Sep

Claim: lane E, based on main `0fbb258b8c444016676ef4e433e0cc4d24f976e1`.

- #146 is already closed. This check does not establish that every change was integrated.
- #145 remains open. Its report shows OpenCL, Python two-point tracking, a CSRT-to-MIL fallback, then a no-discernible-motion error. This is reported evidence, not a reproduced root cause. The portable Go CSRT smoke requested in TFTJ_PROFILE_DIRECTION is still needed before closure.
- #119 remains open. Its report shows Python package/tracker capability checks failing during AI-training bootstrap on v0.5.9. Current-version reproduction and inspection of the bootstrap checks are needed before calling this fixed.
- Metadata stamping remains a hypothesis from #150/NEXT; inspect import/export paths and reproduce before changing provenance handling.

**To Cursor / Claude:** please record any ongoing lane-E fixes or newer smoke results here or in this branch's PR before overlapping implementation. ChatGPT's next investigation is metadata preservation on import/re-save, followed by current-code triage of #145/#119. Lane A release/GUI work and lane B bake-off jobs remain with their current owners. No new feature or default change is proposed in this handoff.

---

## Decision log (joint — append short lines)

| Date | Decision | By |
|------|----------|-----|
| 21 Sep | Contact vib on Normal/Auto default on | Owner → #149 |
| 21 Sep | Bake-off before any observer Go port | Owner + Claude + Cursor agree |
| 21 Sep | Three agents use this board; cleanup before new themes | Owner |

---

## Rules

1. One theme per agent. Claim in **Active** before coding.
2. Base on current `main`.
3. Docs/bake-off do not block release tags unless behavior changes.
4. No silent tracker/profile default changes.
5. Finish → Done row + free lane + PR link.
6. Never force-push another agent’s claimed tip.

### ChatGPT onboarding

1. Read **Active** — skip RUNNING lanes.  
2. Read `TFTJ_PROFILE_DIRECTION.md` + `SAM_ARCHITECTURE.md` § Perception v1.  
3. Claim **E** (cleanup) first if free; else **C** after sprint #0–2.  
4. PR body must include `AGENT_COORD:` block; update Active in same PR.

---

## Handoff template

```text
AGENT_COORD:
  agent: Claude | Cursor | ChatGPT
  lane: E
  claim: close #146 + triage #145
  branch: …
  based_on: main @ <sha>
  will_not_touch: generator.js, VERSION, golden bake-off scripts
  needs_from_other: —
```

---

## Pointers

| Topic | Doc |
|-------|-----|
| Tf/Tj direction | `docs/TFTJ_PROFILE_DIRECTION.md` |
| SAM / bake-off | `docs/SAM_ARCHITECTURE.md` |
| Release spine | `docs/PRODUCTION_ROADMAP.md` |
| Signal ≠ Fidelity | `docs/SIGNAL_VS_FIDELITY.md` |
| Architecture | `HANDOFF.md` |
