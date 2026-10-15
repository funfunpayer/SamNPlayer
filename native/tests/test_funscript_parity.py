"""Parity tests for native funscript clean/sort vs funscript.Parse rules."""

import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent.parent / "build"))

try:
    import funscript_native as fs
except ImportError as e:
    print("SKIP: funscript_native not built", file=sys.stderr)
    print(e, file=sys.stderr)
    sys.exit(0)


def test_drop_bad_times_and_clamp():
    raw = [
        fs.Action(-100, 50),
        fs.Action(0, -5),
        fs.Action(100, 150),
        fs.Action(200, 50),
        fs.Action(fs.max_action_ms() + 1, 10),
    ]
    out = fs.clean_actions(raw)
    assert len(out) == 3
    assert out[0].at == 0 and out[0].pos == 0
    assert out[1].at == 100 and out[1].pos == 100
    assert out[2].at == 200 and out[2].pos == 50


def test_sort():
    raw = [fs.Action(300, 10), fs.Action(100, 20), fs.Action(200, 30)]
    out = fs.clean_actions(raw)
    ats = [a.at for a in out]
    assert ats == [100, 200, 300]


def test_write_roundtrip():
    acts = [fs.Action(0, 0), fs.Action(500, 100)]
    js = fs.write_actions(acts)
    assert '"at": 0' in js and '"pos": 100' in js
    # basic structure
    assert js.strip().startswith("{") and "actions" in js


def main():
    test_drop_bad_times_and_clamp()
    test_sort()
    test_write_roundtrip()
    print("All funscript parity checks passed.")


if __name__ == "__main__":
    main()
