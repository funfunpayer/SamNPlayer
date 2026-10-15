# Migration: from Go to C++ and Python

**Owner decision, 15 Oct 2026:** "Go soll voll abgelöst werden." The app
moves to **Python** (app logic, AI, devices) and **C++** (fast video and
detection paths). Go is replaced step by step, not in one big rewrite.

**Status:** M0 (plan and first measurement) in progress. Agents keep the
step table below current in every migration PR.

## Rules for every step

- **The app stays releasable.** Each step replaces one part. Until the new
  part passes its parity tests, the Go part keeps running in the shipped
  app.
- **Parity first.** A new part is checked against the Go part on the same
  inputs. Funscript output must be equal or within a stated tolerance.
  The Go tests (about 23 000 lines) are the specification: each one is
  ported, or replaced by an equal or stronger test.
- **Measure, then C++.** A path moves to C++ only when a measurement shows
  that Python is too slow there. The gate is the same as in
  `ROADMAP_2026-10.md` §1: 3 warm-up runs and 5 measured runs, comparing the
  median and p95, with no loss in quality.
- **No copied code** from GPL, BSL or EULA projects. The existing rules in
  `ROADMAP_2026-10.md` §3 still apply.
- **The file formats stay** (`.funscript`, `.samn`, settings). Users notice
  nothing except that the app is faster.

## Target layout

```text
  frontend/ (JavaScript, as today, unchanged as far as possible)
      │  same function names as the Wails bindings today
      ▼
  app/  Python: window (pywebview), app API, settings, library,
        player and sync, devices (BLE via bleak), generator orchestration,
        AI (onnxruntime / PyTorch), updater
      │  pybind11
      ▼
  native/  C++ (CMake): frame pipeline, tracking (OpenCV CSRT and flow),
           beat-bar detection, funscript hot paths (filters, resampling)
```

**Why Python as the host and not C++:**
- The window, settings, devices and the generator's control flow mostly
  wait (on a timer, Bluetooth or a file). Python is fast enough there and
  much quicker to write.
- Python has the AI ecosystem: PyTorch, onnxruntime, YOLO and FunGen.
- Bluetooth via `bleak` (MIT) runs on Windows, Linux and macOS. The C++
  options are either licence-restricted (SimpleBLE is BUSL) or per
  platform.
- C++ does the pixel and detection work, where speed counts. OpenCV and
  ONNX Runtime are C++ themselves.

**Why pywebview:** it hosts the existing JavaScript interface without a
rewrite. The Wails bindings (`window.go.main.App.X(...)`) are swapped for a
small shim that calls `window.pywebview.api.X(...)` with the same names. So
`wailsjs/go/main/App.js` keeps its exports, and only the transport changes.

## Steps

| # | Step | Replaces (Go, code lines) | Done when | Status |
|---|---|---|---|---|
| M0 | Plan; measure the generator on real clips (profile) | — | Profile in `docs/owner/` names the top time sinks | synthetic clip done (`owner/generator-profile-2026-10.md`: Go ≈ 0 %, CSRT 46 %, camera estimate 41 %); the Owner's clips still open |
| M1 | `native/` C++ library with CMake and pybind11, plus CI for Linux and Windows. First ports: beat bar, stroke filter, funscript parse and write | `generator/beatbar*`, `funscript` hot paths | Parity tests against the Go output pass in CI | in progress (beat bar + stroke filter + core parse/write done) |
| M2 | Generator in Python plus C++: tracking loop and frame pipeline in C++ (OpenCV), orchestration in Python, reusing the existing Python generator code | `generator` (8 000), `trackcv` (2 900), `posttrack`, `simpletrack`, `strokepreview`, `videox` (1 500) | Golden clips equal or better, and faster per the gate | open |
| M3 | Formats and logic in Python: `funscript`, `samn`, `sam`, `motionx` | `funscript` (6 000), `samn`, `sam`, `motionx` | All ported tests pass, and the corpus sweep (758 scripts) matches the Go output | open |
| M4 | Devices in Python: Sam Neo 2 protocol and BLE (bleak), plus a mock device | `device` (2 300) | Mock tests pass; **Owner hardware test** with the Sam Neo 2 | open |
| M5 | Player and sync in Python (C++ timer only if a measurement needs it) | `player` (2 300) | Ported sync tests pass, and the latency is measured to be no worse | open |
| M6 | App shell: pywebview with the JS frontend, app API in Python, settings, updater, logging, plugin host | `cmd/gui-wails` (10 600), `update`, `logging`, `pluginhost`, `cmd/cli` | All 148 frontend suites pass against the Python API | open |
| M7 | Packaging and release: PyInstaller (or Nuitka) builds for Windows and Linux, with ffmpeg and OpenCV bundled, then the updater switch | `release.yml` | A release built from Python; the old Go release still works as a fallback | open |
| M8 | Remove Go | everything left | No `.go` files; the docs are updated | open |

The order follows benefit. The generator (M1–M2) gains most from C++ and
Python, because of detection and AI. The shell (M6) gains least, so it goes
last.

## Open questions for the Owner

- Is the host **Python + pywebview** (recommended) or a C++ window? This
  needs an answer before M6; until then, M1 to M5 work the same either way.
- macOS: still not a target (see `PLATFORMS.md`)?
