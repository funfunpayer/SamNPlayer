# Agent state: what is done, what is open

**Read this first** when you continue work on SamNPlayer (a new session, a
new agent, or after a context loss). Update it in **every PR**, finished or
not. Detailed history lives in `CHANGELOG.md` and `docs/NEXT.md`; this file
is the short, current state.

**Last update:** 15 Oct 2026, Grok, branch `claude/keen-davinci-t2ysrs`.

## Who works how

* **Claude works alone** on this repo; ChatGPT's and Cursor's access was
removed by the Owner. Old lanes in AGENT_COORD.md are history.
* **Talk to the Owner in German**; the product UI and the docs are in
English.
* **Merging:** Claude squash-merges its own PRs once CI is green and the PR
is mergeable. Never merge on red CI. After a squash merge, merge
origin/main back into the branch; never force-push.
* **Releases:** the Owner pushes release tags; agents only prepare them.
* **Never:**
  * adult media in the repo or in screenshots; screenshots use test patterns only;
  * model names in commits, PRs or files;
  * copied GPL, BSL or EULA code (MIT only with a notice);
  * public links to Suite files.
* **Only real functions**, no placeholders that pretend to work.

## Direction (Owner, 15 Oct)

* **Go is replaced completely by Python and C++**, step by step: see
[`MIGRATION_CPP_PYTHON.md`](MIGRATION_CPP_PYTHON.md).
* **Measure first, then C++.**
* Product plan: [`ROADMAP_2026-10.md`](ROADMAP_2026-10.md), which covers
v0.8, the full editor and the language split.

## Done recently (merged to `main`)

| PR | What |
| --- | --- |
| #541 | Video first: Open/drop in the empty Player; a video finds its script; "Create script" |
| #542 | Corpus sweep (758 scripts), soft seek, line count in the README, v0.7.0 prepared |
| #543 | Roadmap 2026-10 (v0.8, full editor, languages) |
| #545 | Forum post draft (`docs/owner/forum-post.md`) |
| #546 | v0.8 item 1: script folders, tolerant names, variants |
| #547, #549 | v0.8 item 2: stroker motion filter (PCHIP/MAKIMA, speed limit, idle fill, home on pause), plus a fix |
| #552 | Cock Hero / Fap Hero beat bar → one stroke per beat (`generator/beatbar*.go`, `funscript.BeatStrokes`, "From the beat bar") |
| #553 | Roadmap: more Python, C++ instead of Rust; this file; the migration plan |

## In progress

* **M1:** native/ C++ library with CMake + pybind11. Ports: beat-bar (FindBeatBar, BeatTimes, peaks, target, tracker), stroke filter (PCHIP/Makima, speed limit, idle fill), funscript parse/write. Parity tests in `native/tests/` mirror Go cases. **Linux and Windows CI jobs** build the modules and run the parity tests.
* **M0, generator profile:** measure where Generate spends its time
(CSRT tracking, frame decode, post-processing), with OpenCV built locally
(`-tags opencv`, libopencv-dev). Result goes to
`docs/owner/generator-profile-2026-10.md`.

## Open, in order

1. **Before tagging v0.7.0:** Suite files must not go public (D14), per
`docs/next/HANDOFF_2026-10-15.md` task 1. Then the **Owner** pushes the
tag v0.7.0 and runs `docs/RELEASE_0_7_0.md`.
2. Migration M0 → M1 → M2 (`MIGRATION_CPP_PYTHON.md`). Next for M1: more ports if needed, then measure (M0) before claiming speed; M2 generator orchestration.
3. v0.8 items 3–6: audio files with a script, library tiles, skip pauses
plus a shortcut sheet, device wizard. New code for these goes in Python
where the migration has reached that part.
4. Beat bar: an audio onset fallback for Hero rounds without a visible bar;
a beat-bar overlay in the Editor.
5. Editor E1–E5 (`ROADMAP_2026-10.md` §4).
6. Sentry for **development builds only**: needs a project and DSN from the
Owner (org funfun-0q, region de.sentry.io). It must never be in the
shipped app.

## Waiting on the Owner

* Tag v0.7.0 (after item 1 above).
* Hardware tests with the Sam Neo 2 (stroker filter, home on pause, beat-bar
scripts on real Cock Hero clips).
* Migration host: Python + pywebview (recommended) or a C++ window
(needed before M6).
* Delete old branches (`docs/owner/old-branches.txt`); put a public
download link into the forum post.
