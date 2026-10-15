# native/

C++ library for SamNPlayer hot paths (M1 of the migration).

Built with CMake and exposed to Python via pybind11.

## Build

```bash
cd native
mkdir build && cd build
cmake .. -Dpybind11_DIR=$(python3 -m pybind11 --cmakedir)
make
```

Then import `beatbar_native`, `stroke_filter_native` and `funscript_native` from the build directory (or install).

## Modules

- **beatbar_native**: beat-bar detection (parity with `generator/beatbar.go`)
- **stroke_filter_native**: stroke filter (parity with `funscript/stroke_filter.go`) — PCHIP/Makima smooth, max speed, idle fill
- **funscript_native**: core parse/write (parity with `funscript/funscript.go` actions, inverted, duration OFS rules, corrupt-time drop, float rounding). Full metadata is not yet preserved.

## Tests

Parity tests live in `tests/` (see `tests/README.md`).

```bash
# after the build above
PYTHONPATH=native/build python3 native/tests/test_beatbar_parity.py
PYTHONPATH=native/build python3 native/tests/test_stroke_filter_parity.py
PYTHONPATH=native/build python3 native/tests/test_funscript_parity.py
```
