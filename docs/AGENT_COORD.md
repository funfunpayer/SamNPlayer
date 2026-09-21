# Agent coordination (Cursor ↔ Claude)

Shared board **inside the repo** so agents split work without colliding.
Update this file in the same PR as the work you claim (or a tiny follow-up
commit on your branch). Product docs stay English; talk to the owner in
German.

**Not** a second roadmap. Checklists live in `docs/ROADMAP.md` /
`docs/PRODUCTION_ROADMAP.md`. Research journal: `docs/NEXT.md`.
This file is only: **who owns what right now**, and the handoff rules.

---

## Rules (read before starting)

1. **One theme per agent at a time.** Do not both edit the same product
   surface (GUI Generate tab, mapper contact vib, release bump) in parallel.
2. **Claim before code.** Add yourself under Active below with branch +
   PR; push this doc update early so the other agent sees it.
3. **Prefer `main` as base.** Long-lived branches (`claude/keen-davinci-t2ysrs`,
   `cursor/*-d7cb`) rebase/merge `main` before new work.
4. **Docs-only vs product.** Bake-offs and golden corrections must not
   block a release bump unless they change shipped behavior.
5. **No silent default changes.** Tracker/backend/profile defaults change
   only after measured win + owner OK (`docs/SELF_BUILD.md`).
6. **Close the loop.** When done: move row to Done (short), free the lane,
   link the merged PR.

---

## Lanes (sensible split)

| Lane | Typical owner | Examples | Do not mix with |
|------|---------------|----------|-----------------|
| **A — Product / release** | Cursor (this cloud agent) unless noted | Contact vib UX, version bump, portable smoke notes, GUI | Heavy multi-backend bake-off runs |
| **B — Measurement / goldens** | Claude unless noted | FunGen compare, provenance fixes, `SAM_ARCHITECTURE` bake-off tables | Release tag PRs |
| **C — Tf/Tj profile UX** | Claim explicitly | Step 3 partner-mark, profile rename | SAM fusion implementation |
| **D — Perception / SAM** | After bake-off data | Fusion, Go ports of flow/grid_lk | Shipping without numbers |

Owner (funfunpayer) always overrides. If two agents need the same lane,
**stop and write here** — do not race.

---

## Active (update me)

| Lane | Owner | Branch / PR | Goal | Status |
|------|-------|-------------|------|--------|
| B | Claude | was `claude/keen-davinci-t2ysrs` → #150 merged | Provenance fix `clip_ausschnitt` | **Done on main** |
| B | Claude (proposed) | *claim branch when starting* | Python bake-off: CSRT vs flow/grid_lk/region_fusion on `clip_voll` + `clip_ausschnitt` vs FunGen2; write results into `docs/SAM_ARCHITECTURE.md` | **Ready to claim** — base = current `main` |
| A | Cursor | — | v0.5.17 bump after owner smoke (contact vib Normal + Go CSRT log) | **Waiting on owner** |
| C | — | — | TFTJ step 3: partner mark only when contact vib on (Tf/Blow) | **Queued after 0.5.17 or parallel if Cursor free** |
| — | — | close #146 | Duplicate goldens PR (superseded by #140) | **Owner: close** |

---

## Queued (not started)

1. Metadata provenance bug (FunGen files stamped as SamNPlayer on
   import/save) — flagged in #150 / golden README; needs root-cause + fix.
2. Genuine same-tool internal-consistency test (two confirmed SamNPlayer
   exports of one tracking run) — still open after #150 correction.
3. TFTJ steps 4–7 — see `docs/TFTJ_PROFILE_DIRECTION.md`.
4. Issues #145 (likely obsolete MIL/OpenCL trap — reopen only if v0.5.16+
   still fails), #119 AI Train.

---

## Done recently (short)

| When | What | PR |
|------|------|----|
| 21 Sep | Contact vib on Normal/Autotune (default on) | #149 |
| 21 Sep | Tf/Tj-as-profile direction (owner answers locked) | #147 |
| 21 Sep | `clip_ausschnitt` provenance correction | #150 |
| 21 Sep | `clip_voll` goldens + windowed compare | #140 |
| 21 Sep | `phase --window-ms` | #143 |
| 21 Sep | v0.5.16 OpenCL auto + MSYS2 CI | #144 / #141 / #142 |

---

## Handoff template (paste into a commit/PR body)

```text
AGENT_COORD:
  lane: B
  claim: bake-off flow/grid_lk/region_fusion vs FunGen2
  branch: …
  based_on: main @ <sha>
  will_not_touch: VERSION, generator.js contact vib, release.yml
  needs_from_other: —
```

When releasing:

```text
AGENT_COORD:
  lane: A
  claim: v0.5.17 bump + tag prep
  will_not_touch: golden_clips/**, SAM bake-off tables
  blocked_on: owner Win portable smoke (Go CSRT, vib on/off)
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
