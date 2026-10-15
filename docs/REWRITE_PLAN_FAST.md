# Fastest path: SamNPlayer in C++ and Python

**Owner, 15 Oct 2026:** write everything in C/C++ and Python, as fast as
possible, and keep the plan in the repo. This replaces the step order of
`MIGRATION_CPP_PYTHON.md`, which aimed at a slow replacement inside this
repo; its rules stay in force (parity tests, measure before C++, no copied
code, file formats unchanged).

## Decision 1: a new repository

**Owner, 15 Oct: a new repo** (working name `SamNPlayer-2`). This repo (Go)
gets only fixes until the new app ships.

| | New repo | Inside this repo |
|---|---|---|
| Speed | Fast: no Go build, Wails or OpenCV-cgo in the way; its own CI in minutes | Slow: each step must keep the Go app running |
| Risk for users | None: v0.7.x stays as it is | Every migration PR touches the shipped app |
| Two agents | Each gets its own area; no fights over one branch | Collisions (seen on 15 Oct) |
| Reuse | Copy the frontend, `cv.cpp` and the Python generator code; Go tests become the specification | the same |

## Decision 2: the GUI — Qt / PySide6 (Owner, 15 Oct)

The Owner chose a **new native interface in Qt (PySide6, LGPL)** instead of
keeping the 26 000-line JS interface in a web view.

- **Video:** QtMultimedia (FFmpeg backend since Qt 6.5), so no browser
  codec limits. The "Make playable" proxy stays as a fallback.
- **Curve, heatmap, editor timeline:** custom `QWidget`/`QPainter` drawing
  (fast enough for 100 000 points with level-of-detail).
- **Tests:** pytest-qt (MIT) with `QT_QPA_PLATFORM=offscreen` in CI.
- **The 148 JS test suites** become the feature checklist: each one names
  behaviour that the Qt interface must show again, with its own test.
- **Licences:** PySide6 is LGPL; ship it as shared libraries (PyInstaller
  does), name it in the third-party notices, and do not change Qt itself.
- **Effort:** the largest phase (about 3–6 months for everything). The
  phases below bring the Player first, so a usable app comes early.

## Target layout (new repo)

```text
ui/              Qt / PySide6: windows, Player, Editor, Generator, settings
app/             Python: app logic (the 224 Go functions as a Python API), settings, library,
                 player and sync, devices (bleak, Sam Neo 2), updater, logging
generator/       Python orchestration + the existing Python generator code
native/          C++ (CMake, pybind11): tracking and frame pipeline (from
                 trackcv/cv.cpp, which is C++ already), beat bar, stroke filter,
                 funscript hot paths
tests/           pytest (ported Go tests), pytest-qt (interface), native parity
```

## Phases, fastest order

Each phase ends with a usable, tested state. The two agents work in
parallel:
- **Claude:** the Python app, API and review.
- **Grok:** C++ ports in `native/`, each through a PR with parity tests.

| Phase | Content | Done when |
|---|---|---|
| P0 | New repo, CI (Linux + Windows), `native/` moved over, Qt main window with the mode switch (Play / Edit / Generate) | The window opens in CI (offscreen); native parity and pytest-qt are green |
| P1 | **Player in Qt:** open a video or script (video first), QtMultimedia video, curve, heatmap, markers, offset, loop, script variants; funscript/samn in Python plus `native/` | Play a script with its video like v0.7; ported funscript tests and pytest-qt pass |
| P2 | **Devices:** Sam Neo 2 over BLE (bleak), mock device, sync, stroker filter | Mock tests pass; **Owner hardware test** |
| P3 | **Generator:** CSRT tracking and camera estimate in C++ (moved from `cv.cpp`; camera estimate on a second thread), post-processing in Python, beat bar | Synthetic and Owner clips: r equal or better than Go, time equal or better |
| P4 | **Editor, library, settings, AI helpers** in Qt (the rest of the 224 functions) | Every behaviour on the checklist (from the 148 JS suites) has a pytest-qt test that passes |
| P5 | **Packaging:** PyInstaller/Nuitka for Windows and Linux with ffmpeg + OpenCV, updater, release workflow | First release of the new app; Go app archived |

## What makes it fast

- The Player comes first in Qt, so there is a usable app early; the Editor follows.
- `cv.cpp` and the Python generator code already exist, so the generator
  moves instead of being rewritten.
- The Go tests are the specification: port them and they say when a part
  is done.
- Two agents, one area each, so nobody waits on the other.
