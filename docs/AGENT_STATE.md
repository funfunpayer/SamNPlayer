# Agent state: what is done, what is open

**Read this first** when you continue work on SamNPlayer (a new session, a
new agent, or after a context loss). Update it in **every PR**, finished or
not. Detailed history lives in `CHANGELOG.md` and `docs/NEXT.md`; this file
is the short, current state.

**Last update:** 15 Oct 2026, Claude (after merging Grok's M1 work), branch `claude/keen-davinci-t2ysrs`.

## Who works how

- **Two agents (Owner, 15 Oct):** Claude on `claude/keen-davinci-t2ysrs`;
  **Grok only on its own branch** (e.g. `grok/native`), never on Claude's
  branch. Grok's work reaches `main` through a PR that Claude reviews
  (build, tests, YAML) before the merge. ChatGPT's and Cursor's access was
  removed; old lanes in `AGENT_COORD.md` are history.
- **Write files with real content.** Grok's first pushes replaced
  `tests.yml` and two docs with the literal text `$(cat /tmp/...)` and later
  dropped four CI jobs; check `git diff --stat` before every push and never
  regenerate a workflow from scratch. Native jobs live in `tests.yml` only.
- **Talk to the Owner in German**; the product UI and the docs are in
  English.
- **Merging:** Claude squash-merges its own PRs once CI is green and the PR
  is mergeable. Never merge on red CI. After a squash merge, merge
  `origin/main` back into the branch; never force-push.
- **Releases:** the Owner pushes release tags; agents only prepare them.
- **Never:**
  - adult media in the repo or in screenshots; screenshots use test
    patterns only;
  - model names in commits, PRs or files;
  - copied GPL, BSL or EULA code (MIT only with a notice);
  - public links to Suite files.
- **Only real functions**, no placeholders that pretend to work.

## Direction (Owner, 15 Oct)

- **Go is replaced completely by Python and C++**, step by step: see
  [`MIGRATION_CPP_PYTHON.md`](MIGRATION_CPP_PYTHON.md).
- **Measure first, then C++.**
- Product plan: [`ROADMAP_2026-10.md`](ROADMAP_2026-10.md), which covers
  v0.8, the full editor and the language split.

## Done recently (merged to `main`)

| PR | What |
|---|---|
| #541 | Video first: Open/drop in the empty Player; a video finds its script; "Create script" |
| #542 | Corpus sweep (758 scripts), soft seek, line count in the README, v0.7.0 prepared |
| #543 | Roadmap 2026-10 (v0.8, full editor, languages) |
| #545 | Forum post draft (`docs/owner/forum-post.md`) |
| #546 | v0.8 item 1: script folders, tolerant names, variants |
| #547, #549 | v0.8 item 2: stroker motion filter (PCHIP/MAKIMA, speed limit, idle fill, home on pause), plus a fix |
| #552 | Cock Hero / Fap Hero beat bar → one stroke per beat (`generator/beatbar*.go`, `funscript.BeatStrokes`, "From the beat bar") |
| #553 | Roadmap: more Python, C++ instead of Rust; this file; the migration plan |

## In progress

- **M1:** `native/` C++ library with CMake + pybind11.
  - Beat-bar pure functions fully ported (FindBeatBar, BeatTimes, peaks, target, tracker). Parity tests (`native/tests/test_beatbar_parity.py`) mirror the Go synthetic cases.
  - Stroke filter fully ported (PCHIP/Makima smooth, max speed, idle fill). Parity tests (`native/tests/test_stroke_filter_parity.py`) mirror key Go cases.
  - BeatStrokes fully ported (one stroke per beat, speed limit, hold on long gaps). Parity tests (`native/tests/test_beat_strokes_parity.py`) mirror the Go cases.
  - Funscript core parse/write ported (a duration like "10:31" was read as 10 ms because `std::stod` stops at the colon; fixed by Claude) (actions with float rounding/clamp/corrupt drop, inverted, duration OFS rules). Parity tests (`native/tests/test_funscript_parity.py`). nlohmann/json fetched via CMake.
  - Linux and Windows CI jobs (in `tests.yml`) build all modules and run all parity tests. In CI `SAMN_NATIVE_REQUIRED=1` makes a missing module fail instead of "SKIP" with exit 0.
  - `ShiftActions` ported too (Grok), with parity tests.
  - Next: more pure functions if needed; Profile (M0) still required before claiming speed.
- **M2 (started by Grok):** `app/generator_orch.py` is a skeleton only — not wired into the app yet.
- **M0, generator profile:** first result in
  [`owner/generator-profile-2026-10.md`](owner/generator-profile-2026-10.md)
  (tool: `scripts/profile_generate`). On a synthetic 1080p clip, 99 % of the
  time is already in OpenCV/FFmpeg (CSRT 46 %, camera estimate 41 %, scene
  cuts 10 %); Go is about 0 %. A half-size camera estimate was 39 % faster
  but r 0.94 → 0.71, so it was rejected. Next: the camera estimate on a
  second thread (same result, less wall time); the same profile on the
  Owner's clips.

## Open, in order

1. **Before tagging v0.7.0:** Suite files must not go public (D14) — **done in this branch** (`release.yml` builds Suite as compile check only, excludes from artifacts/checksums/release, guard step fails on any `suite` name). Then the **Owner** pushes the tag `v0.7.0` and runs `docs/RELEASE_0_7_0.md`.
2. **MEDIA-01 (done):** lean MP4 probe leaves unknown/empty codec empty (no h264 default); empty or unknown is not playable so the proxy is offered. Covered by `TestProbeISOBMFFUnknownCodec`.
3. Migration M0 → M1 → M2 (`MIGRATION_CPP_PYTHON.md`).
4. v0.8 items 3–6: audio files with a script, library tiles, skip pauses
   plus a shortcut sheet, device wizard. New code for these goes in Python
   where the migration has reached that part.
5. Beat bar: an audio onset fallback for Hero rounds without a visible bar;
   a beat-bar overlay in the Editor.
6. Editor E1–E5 (`ROADMAP_2026-10.md` §4).
7. Sentry for **development builds only**: needs a project and DSN from the
   Owner (org `funfun-0q`, region de.sentry.io). It must never be in the
   shipped app.

## Waiting on the Owner

- Tag `v0.7.0` (after item 1 above).
- Hardware tests with the Sam Neo 2 (stroker filter, home on pause, beat-bar
  scripts on real Cock Hero clips).
- Migration host: Python + pywebview (recommended) or a C++ window
  (needed before M6).
- Delete old branches (`docs/owner/old-branches.txt`); put a public
  download link into the forum post.
