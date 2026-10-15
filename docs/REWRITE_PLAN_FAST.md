# Fastest path: SamNPlayer in C++ and Python

**Owner, 15 Oct 2026:** write everything in C/C++ and Python, as fast as
possible, and keep the plan in the repo. This replaces the step order of
`MIGRATION_CPP_PYTHON.md`, which aimed at a slow replacement inside this
repo; its rules stay in force (parity tests, measure before C++, no copied
code, file formats unchanged).

## Decision 1: a new repository

**Recommended:** a new repo (working name `SamNPlayer-2`). This repo (Go)
gets only fixes until the new app ships.

| | New repo | Inside this repo |
|---|---|---|
| Speed | Fast: no Go build, Wails or OpenCV-cgo in the way; its own CI in minutes | Slow: each step must keep the Go app running |
| Risk for users | None: v0.7.x stays as it is | Every migration PR touches the shipped app |
| Two agents | Each gets its own area; no fights over one branch | Collisions (seen on 15 Oct) |
| Reuse | Copy the frontend, `cv.cpp` and the Python generator code; Go tests become the specification | the same |

## Decision 2: the GUI

The interface is 26 000 lines of JavaScript/CSS with 148 browser tests. It
talks to Go through 224 functions and 42 events.

| Option | Effort | Verdict |
|---|---|---|
| **Keep the JS interface in a web view** (pywebview: WebView2 on Windows, WebKitGTK on Linux) | Small: only the 224 function calls get a new transport (a shim keeps `window.go.main.App.X` working) | **Recommended:** fastest; the 148 tests keep working |
| Qt / PySide6 (LGPL), new native interface | Very large: everything rewritten, about 3–6 months | Only if the web view causes problems (for example video codecs) |
| Dear ImGui (C++) | Large; looks like a tool, not a consumer app | No |

Video playback stays in the web view (HTML5 video), as now; the
"Make playable" proxy via ffmpeg stays.

## Target layout (new repo)

```text
frontend/        JS/CSS from SamNPlayer, plus the bridge shim (Wails → pywebview)
app/             Python: window, API (224 functions), events, settings, library,
                 player and sync, devices (bleak, Sam Neo 2), updater, logging
generator/       Python orchestration + the existing Python generator code
native/          C++ (CMake, pybind11): tracking and frame pipeline (from
                 trackcv/cv.cpp, which is C++ already), beat bar, stroke filter,
                 funscript hot paths
tests/           pytest (ported Go tests), native parity, frontend suites
```

## Phases, fastest order

Each phase ends with a usable, tested state. The two agents work in
parallel:
- **Claude:** the Python app, API and review.
- **Grok:** C++ ports in `native/`, each through a PR with parity tests.

| Phase | Content | Done when |
|---|---|---|
| P0 | New repo, CI (Linux + Windows), frontend copied, bridge shim, empty window shows the interface | The window opens; frontend suites run against stub responses |
| P1 | **Player:** load a script, play a video, curve, heatmap, markers, offset, loop; funscript/samn in Python (using `native/` where it already exists) | Play a script with its video like v0.7; ported funscript tests pass |
| P2 | **Devices:** Sam Neo 2 over BLE (bleak), mock device, sync, stroker filter | Mock tests pass; **Owner hardware test** |
| P3 | **Generator:** CSRT tracking and camera estimate in C++ (moved from `cv.cpp`; camera estimate on a second thread), post-processing in Python, beat bar | Synthetic and Owner clips: r equal or better than Go, time equal or better |
| P4 | **Editor, library, settings, AI helpers** (the rest of the 224 functions) | All 148 frontend suites pass against the Python API |
| P5 | **Packaging:** PyInstaller/Nuitka for Windows and Linux with ffmpeg + OpenCV, updater, release workflow | First release of the new app; Go app archived |

## What makes it fast

- Do not rewrite the interface: it is the largest part, and it already works.
- `cv.cpp` and the Python generator code already exist, so the generator
  moves instead of being rewritten.
- The Go tests are the specification: port them and they say when a part
  is done.
- Two agents, one area each, so nobody waits on the other.
