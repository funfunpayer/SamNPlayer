"""M2 start: Python generator orchestration using native C++ modules.

This is the thin control layer that will call the C++ tracking / frame
pipeline (still to be ported) and the already-ported pure functions
(beat bar, stroke filter, funscript parse/write).

Until the C++ tracking lands, the Go path remains the one used by the app.
"""

from __future__ import annotations

from typing import Any

try:
    import funscript_native as fs
    import stroke_filter_native as sf
    import beatbar_native as bb
    import beat_strokes_native as bs
    NATIVE_OK = True
except ImportError:
    NATIVE_OK = False


def apply_stroke_filter(actions: list[tuple[int, int]], **opts: Any) -> list[tuple[int, int]]:
    """Apply the native stroke filter. Raises when the native module is missing
    (an unfiltered result must never pass for a filtered one)."""
    if not NATIVE_OK:
        raise RuntimeError("native stroke filter not built")
    # Convert to Action objects expected by the binding
    acts = [sf.Action(at, pos) for at, pos in actions]
    f = sf.StrokeFilter()
    if "smooth" in opts:
        f.smooth = opts["smooth"]
    if "max_speed" in opts:
        f.max_speed = float(opts["max_speed"])
    if "idle_fill_ms" in opts:
        f.idle_fill_ms = int(opts["idle_fill_ms"])
    out = sf.apply_stroke_filter(acts, f)
    return [(a.at, a.pos) for a in out]


def parse_funscript(json_str: str) -> dict[str, Any]:
    if not NATIVE_OK:
        raise RuntimeError("native funscript not built")
    s = fs.parse(json_str)
    return {
        "actions": [(a.at, a.pos) for a in s.actions],
        "inverted": s.inverted,
        "dropped": s.dropped,
        "duration_meta": s.duration_meta,
        "metadata_json": s.metadata_json,
    }


# More orchestration (tracking loop, frame pipeline) will be added once the
# C++ CSRT / OpenCV path is ported and the M0 profile confirms the need.
