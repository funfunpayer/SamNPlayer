"""Test für generate_funscript.create_tracker().

Deckt die bekannten OpenCV-API-Varianten ab und den KCF-Fallback, wenn
CSRT fehlt (Issue #94/#95: Windows opencv 5.0.0 ohne CSRT — oft
opencv-python ohne contrib).

Ausführen: python3 generator/create_tracker_test.py
"""

from __future__ import annotations

import sys
from types import SimpleNamespace

import cv2

import generate_funscript as gf


class _FakeTracker:
    pass


def main():
    failures = []

    def check(name, cond, detail=""):
        print(("  OK   " if cond else "  FAIL ") + name
              + (f"  [{detail}]" if detail and not cond else ""))
        if not cond:
            failures.append(name)

    check("nutzt CSRT, wenn vorhanden", gf.create_tracker() is not None)
    check("opencv_has_usable_tracker True", gf.opencv_has_usable_tracker())

    # Snapshot real attrs to restore later
    saved = {
        "legacy": getattr(cv2, "legacy", None),
        "TrackerCSRT_create": getattr(cv2, "TrackerCSRT_create", None),
        "TrackerCSRT": getattr(cv2, "TrackerCSRT", None),
        "TrackerKCF_create": getattr(cv2, "TrackerKCF_create", None),
        "TrackerKCF": getattr(cv2, "TrackerKCF", None),
        "TrackerMIL_create": getattr(cv2, "TrackerMIL_create", None),
        "TrackerMIL": getattr(cv2, "TrackerMIL", None),
        "has_legacy": hasattr(cv2, "legacy"),
    }

    def strip_csrt():
        if hasattr(cv2, "legacy"):
            delattr(cv2, "legacy")
        for name in ("TrackerCSRT_create", "TrackerCSRT"):
            if hasattr(cv2, name):
                delattr(cv2, name)

    def restore():
        if saved["has_legacy"]:
            cv2.legacy = saved["legacy"]
        elif hasattr(cv2, "legacy"):
            delattr(cv2, "legacy")
        for name in ("TrackerCSRT_create", "TrackerCSRT",
                     "TrackerKCF_create", "TrackerKCF",
                     "TrackerMIL_create", "TrackerMIL"):
            val = saved[name]
            if val is not None:
                setattr(cv2, name, val)
            elif hasattr(cv2, name):
                delattr(cv2, name)

    # --- nur legacy.TrackerCSRT.create() ---
    strip_csrt()
    try:
        fake_cls = SimpleNamespace(create=lambda: _FakeTracker())
        cv2.legacy = SimpleNamespace(TrackerCSRT=fake_cls)
        # remove free/main CSRT leftovers already stripped
        t = gf.create_tracker()
        check("nutzt cv2.legacy.TrackerCSRT.create()", isinstance(t, _FakeTracker))
    finally:
        restore()

    # --- nur Hauptmodul TrackerCSRT.create() ---
    strip_csrt()
    try:
        cv2.TrackerCSRT = SimpleNamespace(create=lambda: _FakeTracker())
        t = gf.create_tracker()
        check("nutzt cv2.TrackerCSRT.create()", isinstance(t, _FakeTracker))
    finally:
        restore()

    # --- kein CSRT, aber KCF → Fallback ---
    strip_csrt()
    try:
        cv2.TrackerKCF_create = lambda: _FakeTracker()
        t = gf.create_tracker()
        check("fällt auf KCF zurück ohne CSRT", isinstance(t, _FakeTracker))
        check("opencv_has_usable_tracker mit nur KCF",
              gf.opencv_has_usable_tracker())
    finally:
        restore()

    # --- gar kein Tracker ---
    strip_csrt()
    for name in ("TrackerKCF_create", "TrackerKCF",
                 "TrackerMIL_create", "TrackerMIL"):
        if hasattr(cv2, name):
            delattr(cv2, name)
    if hasattr(cv2, "legacy"):
        delattr(cv2, "legacy")
    try:
        try:
            gf.create_tracker()
            check("wirft ohne jeden Tracker", False)
        except RuntimeError as exc:
            msg = str(exc)
            check("RuntimeError ohne Tracker", True)
            check("Fehlermeldung nennt pip uninstall opencv-python",
                  "pip uninstall opencv-python" in msg, msg[:200])
            check("Fehlermeldung nennt opencv-contrib-python",
                  "opencv-contrib-python" in msg, msg[:200])
        check("opencv_has_usable_tracker False ohne Tracker",
              not gf.opencv_has_usable_tracker())
    finally:
        restore()

    check("nach Restore funktioniert create_tracker wieder",
          gf.create_tracker() is not None)

    print(("FEHLGESCHLAGEN: " + ", ".join(failures)) if failures else
          "Alle Prüfungen bestanden.")
    return 1 if failures else 0


if __name__ == "__main__":
    sys.exit(main())
