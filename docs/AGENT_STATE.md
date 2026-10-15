# Agent state (15 Oct 2026, branch claude/keen-davinci-t2ysrs)

Updated in every PR. For a new session: read this, then the open items.

## Done (merged or in this branch)

- M1 start: native/ C++ skeleton + beat-bar full port + Linux CI + parity tests.
- Stroke filter ported to native/ (C++ + pybind11 + parity tests). Next: Windows CI, funscript parse/write.

## In progress

- **M1:** native/ library. Beat bar done; stroke filter done in this update. Remaining: funscript parse & write, Windows CI job, more pure functions.
- **M0:** generator profile (Owner / local OpenCV measurement) still open.

## Open (in order)

1. Before tagging v0.7.0: Suite files must not go public (D14). Then Owner tags.
2. Finish M1 (Windows CI + funscript hot paths), then M2.
3. v0.8 items 3–6 (audio files, library tiles, skip pauses, device wizard) — new code in Python where migration has reached.
4. Beat-bar audio onset fallback + Editor overlay.
5. Editor E1–E5.
6. Sentry for development builds only (needs Owner DSN).

## Waiting on Owner

- Tag v0.7.0 (after D14).
- Hardware tests (Sam Neo 2: stroker filter, home on pause, beat-bar scripts).
- Migration host decision (Python + pywebview recommended) before M6.
- Delete old branches; public download link for forum post.

Next coding: continue M1 — Windows CI for native/, then funscript parse/write port.
