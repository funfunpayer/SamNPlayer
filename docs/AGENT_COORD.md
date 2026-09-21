# Agent coordination (Cursor ↔ Claude ↔ ChatGPT)

Shared board **inside the repo** so agents split work without colliding.
Update this file in the same PR as the work you claim (or a tiny follow-up
commit on your branch). Product docs stay English; talk to the owner in
German.

**Not** a second roadmap. Checklists live in `docs/ROADMAP.md` /
`docs/PRODUCTION_ROADMAP.md`. Research journal: `docs/NEXT.md`.
This file is only: **who owns what right now**, and the handoff rules.

Agents: **Cursor** (cloud), **Claude** (`claude/keen-davinci-t2ysrs` and
follow-ons), **ChatGPT** (when active — claim a free lane below).

---

## Rules (read before starting)

1. **One theme per agent at a time.** Do not two/three-edit the same
   product surface (GUI Generate, mapper contact vib, `VERSION`, release
   workflow) in parallel.
2. **Claim before code.** Add yourself under Active with branch + PR;
   push this doc update early so the others see it.
3. **Prefer `main` as base.** Rebase/merge `main` before new work on
   long-lived branches.
4. **Docs-only vs product.** Bake-offs and golden corrections must not
   block a release bump unless they change shipped behavior.
5. **No silent default changes.** Tracker/backend/profile defaults change
   only after measured win + owner OK (`docs/SELF_BUILD.md`).
6. **Close the loop.** When done: move row to Done (short), free the lane,
   link the merged PR.
7. **Three agents:** if a lane is taken, pick another or wait — never
   force-push over someone else's claimed branch tip without asking.

---

## Lanes (sensible split)

| Lane | Typical owner | Examples | Do not mix with |
|------|---------------|----------|-----------------|
| **A — Product / release** | Cursor unless noted | Contact vib UX, version bump, portable smoke notes, GUI | Heavy bake-off compute |
| **B — Measurement / goldens** | Claude unless noted | FunGen compare, provenance, `SAM_ARCHITECTURE` bake-off tables | Release tag PRs |
| **C — Tf/Tj profile UX** | Claim explicitly (Cursor or ChatGPT) | Step 3 partner-mark, profile rename | SAM fusion implementation |
| **D — Perception / SAM** | After bake-off data | Fusion, Go ports of flow/grid_lk | Shipping without numbers |
| **E — Bugs / hygiene** | Free / ChatGPT good fit | Metadata stamp bug (#150 note), close stale PRs, issue triage | Competing with A on same files |

Owner (funfunpayer) always overrides. If two agents need the same lane,
**stop and write here** — do not race.

---

## Active (update me)

| Lane | Owner | Branch / PR | Goal | Status |
|------|-------|-------------|------|--------|
| B | **Claude** | local/background on goldens (branch TBD when PR opens) | Bake-off: CSRT vs **flow / grid_lk / region_fusion** on `clip_voll` (~280s) then `clip_ausschnitt` vs FunGen2; results → `docs/SAM_ARCHITECTURE.md` | **RUNNING** — flow on clip_voll in progress (21 Sep) |
| A | Cursor | #151 `docs/AGENT_COORD` then release prep | Keep board current; v0.5.17 after owner smoke | Board PR open; release **waiting on owner** |
| C | — | — | TFTJ step 3: partner mark only when contact vib on | Free — ChatGPT or Cursor after bake-off/release |
| E | — | — | Metadata provenance stamp bug; close #146 | Free — good ChatGPT lane |

---

## Queued (not started)

1. Metadata provenance bug (FunGen files stamped as SamNPlayer on
   import/save) — flagged in #150 / golden README.
2. Genuine same-tool internal-consistency test (two confirmed SamNPlayer
   exports of one tracking run).
3. TFTJ steps 4–7 — `docs/TFTJ_PROFILE_DIRECTION.md`.
4. Issues #145 (likely obsolete MIL trap on pre-0.5.16), #119 AI Train.
5. Close duplicate PR #146 (superseded by #140).

---

## Done recently (short)

| When | What | PR |
|------|------|----|
| 21 Sep | `clip_ausschnitt` provenance correction | #150 |
| 21 Sep | Contact vib on Normal/Autotune (default on) | #149 |
| 21 Sep | Tf/Tj-as-profile direction | #147 |
| 21 Sep | `clip_voll` goldens + windowed compare | #140 |
| 21 Sep | `phase --window-ms` | #143 |
| 21 Sep | v0.5.16 OpenCL auto + MSYS2 CI | #144 / #141 / #142 |

---

## Handoff template (paste into a commit/PR body)

```text
AGENT_COORD:
  agent: Claude | Cursor | ChatGPT
  lane: B
  claim: bake-off flow/grid_lk/region_fusion vs FunGen2
  branch: …
  based_on: main @ <sha>
  will_not_touch: VERSION, generator.js, release.yml
  needs_from_other: —
```

---

## Pointers

| Topic | Doc |
|-------|-----|
| Tf/Tj = profile, marking, vib | `docs/TFTJ_PROFILE_DIRECTION.md` |
| Multi-observer / bake-off vision | `docs/SAM_ARCHITECTURE.md` (Perception v1) |
| Signal Quality ≠ Motion Fidelity | `docs/SIGNAL_VS_FIDELITY.md` |
| Production / release checklist | `docs/PRODUCTION_ROADMAP.md` |
| Architecture overview | `HANDOFF.md` |
