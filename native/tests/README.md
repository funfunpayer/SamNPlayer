# native/tests

Parity tests for the C++ ports against the existing Go reference.

## beatbar

`test_beatbar_parity.py` mirrors the synthetic cases from
`generator/beatbar_test.go` (`TestFindBeatBarAndBeats` and the plain-noise
negative test).

Build first (from `native/`):

```bash
mkdir -p build && cd build
cmake .. -Dpybind11_DIR=$(python3 -m pybind11 --cmakedir)
make -j
```

Then from the repo root:

```bash
PYTHONPATH=native/build python3 native/tests/test_beatbar_parity.py
```

The test exits 0 (with a SKIP message) if the module is not built, so it can
be called unconditionally in CI once the build step is present.
