# native/tests

Parity tests for the C++ ports against the existing Go reference.

## beatbar

`test_beatbar_parity.py` mirrors the synthetic cases from
`generator/beatbar_test.go` (`TestFindBeatBarAndBeats` and the plain-noise
negative test).

## stroke filter

`test_stroke_filter_parity.py` mirrors key cases from
`funscript/stroke_filter_test.go` (smooth PCHIP/Makima, speed limit, etc.).

Build first (from `native/`):

```bash
mkdir -p build && cd build
cmake .. -Dpybind11_DIR=$(python3 -m pybind11 --cmakedir)
make -j
```

Then from the repo root:

```bash
PYTHONPATH=native/build python3 native/tests/test_beatbar_parity.py
PYTHONPATH=native/build python3 native/tests/test_stroke_filter_parity.py
```

The tests exit 0 (with a SKIP message) if the module is not built, so they can
be called unconditionally in CI once the build step is present.
