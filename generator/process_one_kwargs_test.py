#!/usr/bin/env python3
"""Regression: process_one must pass profile signal kwargs into positions_to_funscript.

Until September 2026, build() inside process_one called positions_to_funscript
without peak_prominence / dynamic_range_ms / min_action_interval_ms — so
`--profile weich` set those args on the namespace but they never reached the
signal path. This test guards the wiring, not the algorithm itself.
"""

import ast
import pathlib
import sys

ROOT = pathlib.Path(__file__).resolve().parent
src = (ROOT / "generate_funscript.py").read_text(encoding="utf-8")
tree = ast.parse(src)


def main() -> int:
    # Find the positions_to_funscript call inside process_one / build.
    hits = []
    for node in ast.walk(tree):
        if not isinstance(node, ast.Call):
            continue
        func = node.func
        name = None
        if isinstance(func, ast.Name):
            name = func.id
        elif isinstance(func, ast.Attribute):
            name = func.attr
        if name != "positions_to_funscript":
            continue
        kw = {k.arg for k in node.keywords if k.arg}
        hits.append(kw)

    if not hits:
        print("FAIL: no positions_to_funscript call found")
        return 1

    required = {
        "peak_prominence",
        "dynamic_range_ms",
        "min_interval_ms",
    }
    # At least one call site (the process_one build path) must pass them.
    if not any(required <= kw for kw in hits):
        print("FAIL: positions_to_funscript call missing", sorted(required))
        for i, kw in enumerate(hits):
            print(f"  call[{i}] kwargs={sorted(kw)}")
        return 1
    print("OK: process_one passes peak_prominence/dynamic_range_ms/min_interval_ms")
    return 0


if __name__ == "__main__":
    sys.exit(main())
