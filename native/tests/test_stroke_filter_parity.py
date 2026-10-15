"""Parity-style tests for native/ stroke filter vs funscript/stroke_filter.go.

Mirrors key cases from funscript/stroke_filter_test.go.
Requires the stroke_filter_native module to be built and importable.

Run from repo root after building:
  PYTHONPATH=native/build python3 native/tests/test_stroke_filter_parity.py
"""

import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent.parent / "build"))

try:
    import stroke_filter_native as sf
except ImportError as e:
    print("SKIP: stroke_filter_native not built or not on PYTHONPATH", file=sys.stderr)
    print(e, file=sys.stderr)
    sys.exit(0)


def A(at, pos):
    return sf.Action(at, pos)


zigzag = [A(0, 0), A(500, 100), A(1000, 0), A(1500, 100), A(2000, 0)]


def test_off_changes_nothing():
    out = sf.apply_stroke_filter(zigzag, sf.StrokeFilter())
    assert len(out) == len(zigzag)
    # input not modified (Python lists are copied by the binding)
    assert zigzag[0].pos == 0


def test_smooth_keeps_points_and_range():
    for mode in ("pchip", "makima"):
        f = sf.StrokeFilter()
        f.smooth = mode
        f.step_ms = 50
        out = sf.apply_stroke_filter(zigzag, f)
        assert len(out) > len(zigzag), f"{mode}: no points added"
        orig = {a.at: a.pos for a in zigzag}
        for i, a in enumerate(out):
            assert 0 <= a.pos <= 100, f"{mode}: out of range {a}"
            if i > 0:
                assert a.at > out[i - 1].at, f"{mode}: time not increasing"
            if a.at in orig:
                assert a.pos == orig[a.at], f"{mode}: original point moved"
        # Rounded after top
        q = next((a for a in out if a.at == 600), None)
        assert q is not None and q.pos > 80, f"{mode}: not rounded after the top: {q}"


def test_pchip_no_overshoot():
    a = [A(0, 0), A(300, 10), A(600, 90), A(900, 100)]
    f = sf.StrokeFilter()
    f.smooth = "pchip"
    f.step_ms = 25
    out = sf.apply_stroke_filter(a, f)
    for i in range(1, len(out)):
        # monotone increasing in this data
        assert out[i].pos >= out[i - 1].pos - 1  # allow small rounding


def test_max_speed_limits():
    a = [A(0, 0), A(100, 100)]  # 1000 units/s
    f = sf.StrokeFilter()
    f.max_speed = 200  # max 20 units in 100 ms
    out = sf.apply_stroke_filter(a, f)
    assert out[-1].pos <= 20


if __name__ == "__main__":
    test_off_changes_nothing()
    test_smooth_keeps_points_and_range()
    test_pchip_no_overshoot()
    test_max_speed_limits()
    print("All stroke filter parity checks passed.")
