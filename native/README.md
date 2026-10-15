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

Then import `beatbar_native` from the build directory (or install).

First port: beat-bar detection (parity with `generator/beatbar.go`).
Stroke filter and funscript parse/write follow.

## Tests

Parity tests live in `tests/` (see `tests/README.md`).

```bash
# after the build above
PYTHONPATH=native/build python3 native/tests/test_beatbar_parity.py
```
