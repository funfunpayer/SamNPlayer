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
**Bake-off done 21 Sep (#154):** no Go port for flow/grid_lk/region_fusion.

**F-003 correction (21 Sep):** the timing-drift *signature* (weak
whole-clip r, higher windowed r, swinging lag) is real, but its claimed
cause (VFR/frame-index drift) is refuted — `clip_ausschnitt.mp4` is
genuine CFR, frame-index-vs-real-PTS error is a constant 41ms, not
growing. A constant offset can't produce a ±900ms swinging lag. Practical
guidance ("don't trust whole-clip r alone") is unaffected — only the
mechanism explanation changes, so `TFTJ_PROFILE_DIRECTION.md`'s citation
of that guidance needs no revert.

**F-003 periodicity-aliasing CONFIRMED (21 Sep, this PR):** synthetic
ground-truth test (`funscript.TestPeriodicityAliasingCharacterization`,
runs in CI) proves a lag search over periodic motion can alias onto a
wrong multiple of the stroke period and produce swinging windowed lag +
an orientation flip — from a *provably constant* injected offset, no
real drift involved. Non-periodic motion under the identical offset
recovers the true lag in every window. `BestLagCorrelation` /
`WindowedBestLagCorrelation` are unchanged (characterization test, not a
fix). Full writeup: `docs/FINDINGS_TIMING_TF.md` § F-003. Any mitigation
(e.g. constrain lag search to detected period) is a new idea, not
decided or implemented — needs its own sign-off.

**Rule:** piece by piece. One improvement ships and is measured before the
next big theme. Prefer cleanup + focus over parallel feature sprawl.

---

## Sprint order (21 Sep — all agents)

| # | What | Who | Status |
|---|------|-----|--------|
| **0** | Cleanup | Claude + ChatGPT E | Done (#150/#151/#146/#152/#154) |
| **1** | Bake-off | Claude B | **DONE** #154 — no Go port |
| **2** | **v0.5.17** | Cursor A | **DONE** #153 + tag `v0.5.17` |
| **3** | TFTJ step 3 partner-mark | Cursor C | **DONE** #160 |
| **4** | #145/#119 + metadata + **flow hang** | ChatGPT E / Cursor A | #145 tip-only **DONE** #164; #119 + flow media remain |
| **5** | **v0.5.18** | Cursor A | **DONE** #166 + tag `v0.5.18` |
| **6** | TFTJ milestone: auto feel + motion candidates | Cursor A | 4b **DONE** #167; bump **v0.5.19** |
| **7** | **v0.5.19** | Cursor A | **DONE** #168 + tag `v0.5.19` |
| **8** | No-mark 4-zone in GUI + Contact-first | Cursor A | **DONE** #169 — step-4 measure: 4-zone < tip CSRT → keep opt-in |
| **9** | **v0.5.20** | Cursor A | **DONE** #171 + tag `v0.5.20` — owner Flow / 4-zone smoke |
| **10** | Prep **v0.5.21** | Cursor A | **DONE** #172 |
| **11** | **v0.5.21** bump + tag | Cursor A | **DONE** #175 + tag `v0.5.21` |
| **12** | **v0.5.22** bump + tag | Cursor A | **DONE** #181 + tag `v0.5.22` |
| **13** | **v0.5.23** bump + tag | Cursor A | **DONE** #198 + tag `v0.5.23` — Training clip frames in portable + MT wave |

---

## Cleanup checklist

- [x] #150 provenance fix
- [x] #151 AGENT_COORD on `main`
- [x] #146 closed
- [x] #152 ChatGPT lane E handoff merged
- [x] #156 Flow CLI scaling fix merged; GitHub Tests passed
- [x] Bake-off results in `SAM_ARCHITECTURE.md` + NEXT (#154)
- [x] No Go ports of flow/grid_lk from this bake-off

---

## Active

| Lane | Owner | Branch / PR | Goal | Status |
|------|-------|-------------|------|--------|
| Rel | Cursor | [#208](https://github.com/funfunpayer/SamNPlayer/pull/208) + tag `v0.5.25` | Contact marks UI + bump **v0.5.25** + Release | **DONE** — portable live |
| Z4→1 | Cursor | [#209](https://github.com/funfunpayer/SamNPlayer/pull/209) merged | **4-Zone out of Generate GUI** (Advanced too); CSRT 1-Zone only; backend/CLI stay | **DONE** (`3421b97`) |
| Marks | Cursor | [#211](https://github.com/funfunpayer/SamNPlayer/pull/211) merged | Persist `metadata.contact_marks` + Play overlay | **DONE** (`7f81db3`) |
| TipBox | Cursor | [#212](https://github.com/funfunpayer/SamNPlayer/pull/212) merged | Stamp tip ROI into `contact_marks.tip` + blue overlay | **DONE** (`d207ece`) |
| Feel | Cursor | [#214](https://github.com/funfunpayer/SamNPlayer/pull/214) merged | Feel-decouple Stage A: spatial vib near marks when trajectory present | **DONE** (`f1d0517`) |
| Traj | Cursor | [#217](https://github.com/funfunpayer/SamNPlayer/pull/217) merged | Soft-on tip-path capture with Contact vib (Stage A needs it) | **DONE** (`3ca37d1`) |
| Rel27 | Cursor | [#218](https://github.com/funfunpayer/SamNPlayer/pull/218) + tag `v0.5.27` | bump **v0.5.27** + Release (Feel + Traj + Training mosaic/suction) | **DONE** — https://github.com/funfunpayer/SamNPlayer/releases/tag/v0.5.27 |
| Brand | Cursor | [#220](https://github.com/funfunpayer/SamNPlayer/pull/220) merged | **Emotion Script** product name; one format (`.samn`); less technical GUI | **DONE** (`8009331`) |
| Drift | Claude + Cursor | [#226](https://github.com/funfunpayer/SamNPlayer/pull/226) + [#230](https://github.com/funfunpayer/SamNPlayer/pull/230) + [#233](https://github.com/funfunpayer/SamNPlayer/pull/233) + [#236](https://github.com/funfunpayer/SamNPlayer/pull/236) | **CSRT long-clip drift** — #226 guards; #230 adaptive detrend; #233 rhythm grid (opt-in signal source + orientation fix); #236 Advanced GUI toggle | **DONE** — engine free; default-on = Owner after ≥4–5 clips |
| BugE | ChatGPT | free | Owner: bugfix / copy review — docs + GUI Emotion Script strings; steward | **NEXT** — ChatGPT |
| Branding | Claude | #469 | **Announced 11 Oct, before work (Owner: say first so ChatGPT can help):** SN logo B redrawn as vector in lilac + teal edge → `build/appicon.png` 1024, `build/windows/icon.ico` 16–256, `frontend/src/assets/images/logo-universal.png`, `website/media/logo.png`; calmer palette in `style.css` (one accent lilac, flat surfaces instead of most of the 61 gradients, single-colour wordmark, teal/feel only for data); new Settings **About SamNPlayer** (logo, version, edition, EULA/notices, donate, partner graphic SN 🤝 figures of the "Sex for him" site, link once Owner sends the URL). Files touched: `style.css`, `index.html`, `settings.js`, `legal.js`, icon files, website logo. **ChatGPT:** welcome to review copy/visuals on the PR; StudioEmbed is yours on your own branch (see #467 comment) | **DONE** (merged #469) |
| PlayLayout | Claude | `claude/keen-davinci-t2ysrs` | Big video like FunGen2: info column hidden on Play (`body[data-tab]`), tools column 340 px sticky + scrollable, collapsible via ⟩ button or **T** (remembered), video up to 70–78 vh; all IDs unchanged. Test `pb_tools_toggle_test.py` | **DONE** (merged #470) |
| MultiAxisCurves | Claude | `claude/keen-davinci-t2ysrs` | **Announced 11 Oct, before work:** Play gets colour-coded axis lanes under the main curve (Vibration amber, Suction lilac; filled area + playhead, zoom shared), a legend row with clickable axis chips (= `#pb-axis`), main curve takes the colour of the edited axis with the stroke curve as faint ghost behind it. Create has no curve view (hands off to Play), so Play only. Files: `playback.js`, `style.css`, new test `pb_axis_lanes_test.py`. **ChatGPT:** visual review welcome on the PR | **DONE** (merged #472) |
| GUIAudit | Claude | `claude/keen-davinci-t2ysrs` | Owner 11 Oct: final check of GUI wiring, modularity and AI anchoring → `docs/GUI_AUDIT_2026-10.md` (docs only). Follow-ups proposed there (binding cleanup, splitting playback.js / generator.js) are **not** claimed — free for anyone after Owner OK | **DONE** (merged #473) |
| RepoRefresh | Claude | `claude/keen-davinci-t2ysrs` | **Announced 11 Oct on #468 before work:** new GUI screenshots (`docs/media` + `website/media`, same names + `gui-multiaxis`, `gui-device`, `gui-about`), lilac logo/wordmark, README tagline/screenshots/Playback bullets, website lilac palette + screenshot section, CHANGELOG lines for #470/#472. Not touched: README release/download sections (ChatGPT #468). Training screenshot deliberately **not** published (shows the pixel clip) | **DONE** (merged #474) |
| PlayCore (#475) | Claude | `claude/keen-davinci-t2ysrs` | Owner 11 Oct: Claude builds the Play **core**, ChatGPT the **layout/look** around it. Core PR: `data-pb-region` hooks, `data-pb-state/tools/playing/video`, DOM order video→transport→timeline, CSS theme tokens read by JS, ResizeObserver redraw, keyboard on axis chips/lanes, `docs/PLAY_UI_CONTRACT.md` (incl. Owner mockup → real feature map). Follow-ups listed in the contract §7 | **DONE** (merged #476) |
| TabNames (#475) | Claude | `claude/keen-davinci-t2ysrs` (#478) | Owner decision: tabs Play · Generator · Training · Tools (Tools = Device/Log/Settings via `#tools-nav`), tagline Generate · Play · Train; copy updated; QA freeze `docs/QA_CHECK_2026-10-11.md` | **DONE** (merged #478) |
| Brand v3 (#477) | Claude integrates, ChatGPT+Owner supply files | V1 step S9 | Flow icon/banner: exact Owner-approved files, no redraw. Theme already live (#480) | **WAITING for files** |
| **V1.0** | Claude + ChatGPT | `docs/V1_PLAN.md` | Owner 11 Oct: final mockup = new look, release 1.0. Steps S2–S11. S2 app shell done (this branch). **Next: S3 Player 3-column shell (Claude)** | **S2 DONE, CLAIMED S3** |
| PlayLayout v2 (#475) | ChatGPT | own branch, after PlayCore merges | Play section of `style.css` only, built on `docs/PLAY_UI_CONTRACT.md`: grid areas, 1366×768 fit, tool grouping, empty state, token colours; no JS/ID changes | **NEXT** |
| AudioVib | Claude | [#462](https://github.com/funfunpayer/SamNPlayer/pull/462) merged (`a7c6e1d`) | **Owner 9 Oct:** Play vibration from video audio only, switchable — Settings → Play + Play panel `playback.audio_vibration` (default **off**); suction stays on script; no file changes. Also fixed stroke-preview rate (2–3× too high at turning points) | **DONE** |
| CursorLane | Claude | standing (Owner 10 Oct) | Cursor's open items now Claude's: GUI follow-ups, board hygiene, merges, releases (release/tag still on Owner ask). V1 `.vlm.json`→marks superseded by contact-points + candidates (see `VLM_TEACHER_PLAN.md`) | **ACTIVE** |
| AudioCancel | ChatGPT | [#420](https://github.com/funfunpayer/SamNPlayer/pull/420) merged (`82062c7`) | Propagate Generate cancellation into post-hoc ffmpeg audio check; regression test; no defaults | **DONE** |
| AtomicWrite | ChatGPT | [#428](https://github.com/funfunpayer/SamNPlayer/pull/428) merged (`6538296`) | Write native `.funscript` through same-directory temp + sync + rename; preserve destination on failure | **DONE** |
| CLICancel | ChatGPT | [#432](https://github.com/funfunpayer/SamNPlayer/pull/432) merged (`3d8da4a`) | Propagate CLI `generate` signals into tracker/ffmpeg/Python cancellation; regression test; no defaults | **DONE** |
| StrokePreviewCancel | ChatGPT | [#435](https://github.com/funfunpayer/SamNPlayer/pull/435) merged (`fedef03`) | Propagate CLI `stroke-preview` signals into analyzer/ffmpeg cancellation; regression test; no defaults | **DONE** |
| Bugfix | Cursor | [#227](https://github.com/funfunpayer/SamNPlayer/pull/227) merged | **Create→Play `.samn` feel** (recipe/marks/trajectory) + Create busy/overwrite | **DONE** |
| Look | Cursor | [#225](https://github.com/funfunpayer/SamNPlayer/pull/225) merged | Emotion GUI look (Sora/Figtree) | **DONE** |
| Rel28 | Cursor | [#228](https://github.com/funfunpayer/SamNPlayer/pull/228) + tag `v0.5.28` | bump **v0.5.28** + Release (look + feel + CSRT guards) | **DONE** — https://github.com/funfunpayer/SamNPlayer/releases/tag/v0.5.28 |
| Rel29 | Cursor | [#232](https://github.com/funfunpayer/SamNPlayer/pull/232) + tag `v0.5.29` | bump **v0.5.29** + Release (#230 stroke detrend on by default) | **DONE** — https://github.com/funfunpayer/SamNPlayer/releases/tag/v0.5.29 |
| Rel30 | Cursor | [#238](https://github.com/funfunpayer/SamNPlayer/pull/238) + tag `v0.5.30` | bump **v0.5.30** + Release (#233 rhythm grid + #236 Advanced toggle) | **DONE** — https://github.com/funfunpayer/SamNPlayer/releases/tag/v0.5.30 |
| GuiRG | Cursor | [#236](https://github.com/funfunpayer/SamNPlayer/pull/236) merged | Advanced opt-in *Rhythm-robust signal* checkbox (landed #235 onto main) | **DONE** |
| BF3 | Cursor | [#239](https://github.com/funfunpayer/SamNPlayer/pull/239) merged | **Stroke-preview Stage B finish** — cut-rate → PerSceneROI; pan → camera compensation (this run) + GUI tip sync | **DONE** |
| G1next | Cursor | [#241](https://github.com/funfunpayer/SamNPlayer/pull/241) merged | **G1 heuristics inventory** — knob→code→default map in `GENERATE_HEURISTICS.md` (docs only; no behavior change) | **DONE** |
| G1safe | Cursor | [#257](https://github.com/funfunpayer/SamNPlayer/pull/257) merged | **G1 without Owner** — G1.2 QD×audio English hint + Everyday blurb; **no** default changes | **DONE** (`1a4929e`) |
| G3safe | Cursor | [#258](https://github.com/funfunpayer/SamNPlayer/pull/258) merged | **G3 suggest-only harden** — English profile-suggest copy + G3 status sync; **no** CSRT/ROI default changes | **DONE** |
| Look2 | Cursor | [#259](https://github.com/funfunpayer/SamNPlayer/pull/259) merged | Emotion Look v2 — thinner rail, honey/teal, organic radii (CSS only) | **DONE** |
| GuiLoad | Cursor | [#260](https://github.com/funfunpayer/SamNPlayer/pull/260) merged | **GUI load rule** — Load project + Scale range | **DONE** |
| AIWrite | Cursor | [#261](https://github.com/funfunpayer/SamNPlayer/pull/261) merged | **AI script writer path S0** — opt-in local draft plan + package + GUI stub; Everyday CSRT unchanged | **DONE** |
| GapHeal | Cursor | [#262](https://github.com/funfunpayer/SamNPlayer/pull/262) merged | **Heal tracking gaps** — strip+bridge `tracking_gaps`; clear metadata | **DONE** |
| Refine | Cursor | [#263](https://github.com/funfunpayer/SamNPlayer/pull/263) merged | Look v3 lilac polish + motion; AIWrite S1 export; USER_HANDBOOK + **in-GUI handbook** | **DONE** |
| BookmarksUI | Cursor | [#266](https://github.com/funfunpayer/SamNPlayer/pull/266) merged | **Play bookmarks** — wire Get/SaveScriptBookmarks (add/seek/remove) | **DONE** |
| ChaptersUI | Cursor | [#269](https://github.com/funfunpayer/SamNPlayer/pull/269) merged | **Play chapters** — wire Get/SaveScriptChapterMarks (add from selection/seek/remove) | **DONE** |
| SceneMap | Claude (plan) → Cursor (build) | [#243](https://github.com/funfunpayer/SamNPlayer/pull/243) merged | **Scene map plan** — `docs/SCENE_MAP_PLAN.md`; Owner § 6 + P1 approved | **DONE** (plan on main) |
| SceneP1 | Cursor | [#246](https://github.com/funfunpayer/SamNPlayer/pull/246) merged | **SceneMap P1** — score/choose, ScanSceneMap, Advanced Show scene map | **DONE** |
| SceneP2 | Cursor | [#247](https://github.com/funfunpayer/SamNPlayer/pull/247) → [#249](https://github.com/funfunpayer/SamNPlayer/pull/249) | **SceneMap P2** — heatmap overlay + exclude/source/region marks | **DONE** (`v0.5.31`) |
| SceneP3 | Cursor | [#251](https://github.com/funfunpayer/SamNPlayer/pull/251) merged | **SceneMap P3** — engine honours marks (candidate filter, Go masks+rhythm grid, camera multi-exclude) | **DONE** (`9d20e05`) |
| SceneP4 | Cursor | [#253](https://github.com/funfunpayer/SamNPlayer/pull/253) merged | **SceneMap P4** — `.samn` `sceneMap` persist (writer+reader+round-trip; no funscript export) | **DONE** |
| SceneP4b | Cursor | [#254](https://github.com/funfunpayer/SamNPlayer/pull/254) merged | **SceneMap P4 GUI** — restore marks/map from companion `.samn` on Create load | **DONE** |
| SceneP5a | Cursor | [#255](https://github.com/funfunpayer/SamNPlayer/pull/255) merged | **SceneMap P5a** — CLI `export-learning` L0 collect (opt-in; no YOLO train write) | **DONE** (`c57b910`) |
| SceneP5b | Cursor | [#256](https://github.com/funfunpayer/SamNPlayer/pull/256) merged | **SceneMap P5b** — Settings Collect/Delete + Create Export for learning | **DONE** (`524d23b`) |
| AIScript | ChatGPT → Cursor | [#244](https://github.com/funfunpayer/SamNPlayer/pull/244) → [#249](https://github.com/funfunpayer/SamNPlayer/pull/249) | **Local script-model foundation** — Go profilemodel + AI Training | **DONE** (`v0.5.31`) |
| TargetLock | ChatGPT → Cursor | [#248](https://github.com/funfunpayer/SamNPlayer/pull/248) → [#249](https://github.com/funfunpayer/SamNPlayer/pull/249) | **Semantic target lock** — strict body class + rhythm seed lock | **DONE** (`v0.5.31`) |
| Rel31 | Cursor | [#249](https://github.com/funfunpayer/SamNPlayer/pull/249) + tag `v0.5.31` | bump **v0.5.31** + Release (profilemodel + target lock + SceneMap P2 + Stage B notes) | **DONE** — https://github.com/funfunpayer/SamNPlayer/releases/tag/v0.5.31 |
| Rel32 | Cursor | [#271](https://github.com/funfunpayer/SamNPlayer/pull/271) | bump **v0.5.32** bookkeeping (SceneMap P3–P5, Look2/3, AIWrite, GapHeal, GuiLoad, Bookmarks/Chapters, …) | **DONE** — never tagged; portable → Rel33 |
| Rel33 | Cursor | [#278](https://github.com/funfunpayer/SamNPlayer/pull/278) + tag `v0.5.33` | bump **v0.5.33** + Release (Rel32 stack + Plugin H0/H1 + Follow/Ignore marks S0/S1) | **DONE** — https://github.com/funfunpayer/SamNPlayer/releases/tag/v0.5.33 |
| Rel34 | Cursor | [#291](https://github.com/funfunpayer/SamNPlayer/pull/291) + tag `v0.5.34` | bump **v0.5.34** + Release (post-33: vib/AIWrite/Intiface/Marks/IdLock #279–#288+#287) | **DONE** — https://github.com/funfunpayer/SamNPlayer/releases/tag/v0.5.34 |
| Rel35 | Cursor | [#302](https://github.com/funfunpayer/SamNPlayer/pull/302) + tag `v0.5.35` | bump **v0.5.35** + Release (VLM0 + contact points + candidates import + RF-DETR tooling) | **DONE** — https://github.com/funfunpayer/SamNPlayer/releases/tag/v0.5.35 |
| Rel36 | Cursor | [#310](https://github.com/funfunpayer/SamNPlayer/pull/310) + tag `v0.5.36` | bump **v0.5.36** + Release (VLM1 engine + Advanced Use contact points) | **DONE** — https://github.com/funfunpayer/SamNPlayer/releases/tag/v0.5.36 |
| Rel37 | Cursor | [#319](https://github.com/funfunpayer/SamNPlayer/pull/319) + tag `v0.5.37` | bump **v0.5.37** + Release (handbook + VLM1 GUI) | **DONE** — https://github.com/funfunpayer/SamNPlayer/releases/tag/v0.5.37 |
| Rel38 | Cursor | [#326](https://github.com/funfunpayer/SamNPlayer/pull/326) + tag `v0.5.38` | bump **v0.5.38** + Release (VLM2/3 tooling + AI setup) | **DONE** — https://github.com/funfunpayer/SamNPlayer/releases/tag/v0.5.38 |
| Rel39 | Cursor | [#333](https://github.com/funfunpayer/SamNPlayer/pull/333) + tag `v0.5.39` | bump **v0.5.39** + Release (contact points GUI + handbook) | **DONE** — https://github.com/funfunpayer/SamNPlayer/releases/tag/v0.5.39 |
| Rel40 | Cursor | [#340](https://github.com/funfunpayer/SamNPlayer/pull/340) + tag `v0.5.40` | bump **v0.5.40** + Release (VLM teacher plan update) | **DONE** — https://github.com/funfunpayer/SamNPlayer/releases/tag/v0.5.40 |
| Rel41 | Cursor | [#347](https://github.com/funfunpayer/SamNPlayer/pull/347) + tag `v0.5.41` | bump **v0.5.41** + Release (contact detector + VLM) | **DONE** — https://github.com/funfunpayer/SamNPlayer/releases/tag/v0.5.41 |
| Rel42 | Cursor | [#354](https://github.com/funfunpayer/SamNPlayer/pull/354) + tag `v0.5.42` | bump **v0.5.42** + Release (VLM probe + golden) | **DONE** — https://github.com/funfunpayer/SamNPlayer/releases/tag/v0.5.42 |
| Rel43 | Cursor | [#361](https://github.com/funfunpayer/SamNPlayer/pull/361) + tag `v0.5.43` | bump **v0.5.43** + Release (VLM1 + contact) | **DONE** — https://github.com/funfunpayer/SamNPlayer/releases/tag/v0.5.43 |
| Rel44 | Cursor | [#368](https://github.com/funfunpayer/SamNPlayer/pull/368) + tag `v0.5.44` | bump **v0.5.44** + Release (VLM2/3) | **DONE** — https://github.com/funfunpayer/SamNPlayer/releases/tag/v0.5.44 |
| Rel45 | Cursor | [#375](https://github.com/funfunpayer/SamNPlayer/pull/375) + tag `v0.5.45` | bump **v0.5.45** + Release (AI setup) | **DONE** — https://github.com/funfunpayer/SamNPlayer/releases/tag/v0.5.45 |
| Rel46 | Cursor | [#382](https://github.com/funfunpayer/SamNPlayer/pull/382) + tag `v0.5.46` | bump **v0.5.46** + Release (contact points) | **DONE** — https://github.com/funfunpayer/SamNPlayer/releases/tag/v0.5.46 |
| Rel47 | Cursor | [#389](https://github.com/funfunpayer/SamNPlayer/pull/389) + tag `v0.5.47` | bump **v0.5.47** + Release (VLM) | **DONE** — https://github.com/funfunpayer/SamNPlayer/releases/tag/v0.5.47 |
| Rel48 | Cursor | [#396](https://github.com/funfunpayer/SamNPlayer/pull/396) + tag `v0.5.48` | bump **v0.5.48** + Release (handbook) | **DONE** — https://github.com/funfunpayer/SamNPlayer/releases/tag/v0.5.48 |
| Rel49 | Cursor | [#403](https://github.com/funfunpayer/SamNPlayer/pull/403) + tag `v0.5.49` | bump **v0.5.49** + Release (VLM1 GUI) | **DONE** — https://github.com/funfunpayer/SamNPlayer/releases/tag/v0.5.49 |
| Rel50 | Cursor | [#410](https://github.com/funfunpayer/SamNPlayer/pull/410) + tag `v0.5.50` | bump **v0.5.50** + Release (VLM2/3 tooling) | **DONE** — https://github.com/funfunpayer/SamNPlayer/releases/tag/v0.5.50 |
| Rel51 | Cursor | [#417](https://github.com/funfunpayer/SamNPlayer/pull/417) + tag `v0.5.51` | bump **v0.5.51** + Release (AI setup) | **DONE** — https://github.com/funfunpayer/SamNPlayer/releases/tag/v0.5.51 |
| Rel52 | Cursor | [#424](https://github.com/funfunpayer/SamNPlayer/pull/424) + tag `v0.5.52` | bump **v0.5.52** + Release (contact) | **DONE** — https://github.com/funfunpayer/SamNPlayer/releases/tag/v0.5.52 |
| Rel53 | Cursor | [#431](https://github.com/funfunpayer/SamNPlayer/pull/431) + tag `v0.5.53` | bump **v0.5.53** + Release (VLM) | **DONE** — https://github.com/funfunpayer/SamNPlayer/releases/tag/v0.5.53 |
| Rel54 | Cursor | [#438](https://github.com/funfunpayer/SamNPlayer/pull/438) + tag `v0.5.54` | bump **v0.5.54** + Release (handbook) | **DONE** — https://github.com/funfunpayer/SamNPlayer/releases/tag/v0.5.54 |
| Rel55 | Cursor | [#445](https://github.com/funfunpayer/SamNPlayer/pull/445) + tag `v0.5.55` | bump **v0.5.55** + Release (VLM1) | **DONE** — https://github.com/funfunpayer/SamNPlayer/releases/tag/v0.5.55 |
| Rel56 | Cursor | [#452](https://github.com/funfunpayer/SamNPlayer/pull/452) + tag `v0.5.56` | bump **v0.5.56** + Release (VLM2/3) | **DONE** — https://github.com/funfunpayer/SamNPlayer/releases/tag/v0.5.56 |
| Rel57 | Cursor | [#459](https://github.com/funfunpayer/SamNPlayer/pull/459) + tag `v0.5.57` | bump **v0.5.57** + Release (AI) | **DONE** — https://github.com/funfunpayer/SamNPlayer/releases/tag/v0.5.57 |
| Rel58 | Cursor | [#466](https://github.com/funfunpayer/SamNPlayer/pull/466) + tag `v0.5.58` | bump **v0.5.58** + Release (contact) | **DONE** — https://github.com/funfunpayer/SamNPlayer/releases/tag/v0.5.58 |
| Rel59 | Cursor | [#473](https://github.com/funfunpayer/SamNPlayer/pull/473) + tag `v0.5.59` | bump **v0.5.59** + Release (handbook) | **DONE** — https://github.com/funfunpayer/SamNPlayer/releases/tag/v0.5.59 |
| Rel60 | Cursor | [#480](https://github.com/funfunpayer/SamNPlayer/pull/480) + tag `v0.5.60` | bump **v0.5.60** + Release (Flow theme) | **DONE** — https://github.com/funfunpayer/SamNPlayer/releases/tag/v0.5.60 |

---

## Next free

- **V1.0 S3** Player 3-column shell (Library left, Inspector right with tabs Devices/Audio/Playback/View, timeline bottom). Claude owns. Collapsible panels, state remembered. Existing controls moved into tabs, IDs unchanged.
- Brand v3 files (S9) waiting for Owner/ChatGPT.
- PlayLayout v2 CSS (ChatGPT).

---

## Handoff template

```text
AGENT_COORD:
  agent: Claude | Cursor | ChatGPT
  lane: E
  claim: …
  branch: …
  based_on: main @ <sha>
  will_not_touch: generator.js, VERSION, release.yml
  needs_from_other: —
```

---

## Pointers

| Topic | Doc |
|-------|-----|
| Tf/Tj direction | `docs/TFTJ_PROFILE_DIRECTION.md` |
| SAM / bake-off | `docs/SAM_ARCHITECTURE.md` |
| VLM teacher → own detector | `docs/VLM_TEACHER_PLAN.md` |
| Release spine | `docs/PRODUCTION_ROADMAP.md` |
| Signal ≠ Fidelity | `docs/SIGNAL_VS_FIDELITY.md` |
| Architecture | `HANDOFF.md` |
