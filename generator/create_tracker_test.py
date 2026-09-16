"""Test für generate_funscript.create_tracker().

Reales Problem (Nutzerbericht, 16. September 2026): eine echte
opencv-contrib-python-Installation hatte WEDER cv2.legacy.TrackerCSRT_create
NOCH cv2.TrackerCSRT_create - nur die dritte, klassenbasierte API
cv2.TrackerCSRT.create(). create_tracker() kannte bis dahin nur die ersten
zwei und stürzte mit einem rohen AttributeError ab.

Da sich die drei API-Varianten nicht zuverlässig lokal nachstellen lassen
(diese Testumgebung hat immer alle drei), wird hier direkt am echten
cv2-Modul manipuliert: die jeweils "höherwertigen" Attribute vorübergehend
entfernt, damit create_tracker() auf den nächstniedrigeren Pfad zurückfällt
- und am Ende alles wiederhergestellt, damit andere Tests im selben Prozess
nicht betroffen sind.

Ausführen: python3 generator/create_tracker_test.py
"""

import sys

import cv2

import generate_funscript as gf


def main():
    failures = []

    def check(name, cond, detail=""):
        print(("  OK   " if cond else "  FAIL ") + name + (f"  [{detail}]" if not cond else ""))
        if not cond:
            failures.append(name)

    # --- Pfad 1: cv2.legacy.TrackerCSRT_create() bevorzugt, falls vorhanden ---
    check("nutzt cv2.legacy.TrackerCSRT_create(), wenn vorhanden",
          gf.create_tracker() is not None)

    # --- Pfad 2: ohne cv2.legacy fällt es auf cv2.TrackerCSRT_create() zurück ---
    has_legacy = hasattr(cv2, "legacy")
    legacy_mod = getattr(cv2, "legacy", None)
    if has_legacy:
        delattr(cv2, "legacy")
    try:
        check("fällt ohne cv2.legacy auf cv2.TrackerCSRT_create() zurück",
              gf.create_tracker() is not None)

        # --- Pfad 3: ohne cv2.TrackerCSRT_create() auf cv2.TrackerCSRT.create() ---
        has_free_fn = hasattr(cv2, "TrackerCSRT_create")
        free_fn = getattr(cv2, "TrackerCSRT_create", None)
        if has_free_fn:
            delattr(cv2, "TrackerCSRT_create")
        try:
            check("fällt ohne cv2.TrackerCSRT_create() auf cv2.TrackerCSRT.create() zurück "
                  "(der reale Fall aus dem Nutzerbericht)",
                  gf.create_tracker() is not None)

            # --- Keine der drei APIs vorhanden: klarer Fehler statt Traceback ---
            has_cls = hasattr(cv2, "TrackerCSRT")
            cls = getattr(cv2, "TrackerCSRT", None)
            if has_cls:
                delattr(cv2, "TrackerCSRT")
            try:
                try:
                    gf.create_tracker()
                    check("wirft, wenn keine der drei APIs existiert", False)
                except RuntimeError as exc:
                    check("wirft RuntimeError statt AttributeError, wenn nichts passt",
                          True, str(exc))
                    check("Fehlermeldung nennt den Aktualisierungs-Hinweis",
                          "opencv-contrib-python" in str(exc), str(exc))
                except AttributeError as exc:
                    check("wirft RuntimeError statt eines rohen AttributeError", False, str(exc))
            finally:
                if has_cls:
                    cv2.TrackerCSRT = cls
        finally:
            if has_free_fn:
                cv2.TrackerCSRT_create = free_fn
    finally:
        if has_legacy:
            cv2.legacy = legacy_mod

    # --- Aufräumen erfolgreich: create_tracker() funktioniert wieder normal ---
    check("nach dem Wiederherstellen funktioniert create_tracker() wie zuvor",
          gf.create_tracker() is not None)

    print(("FEHLGESCHLAGEN: " + ", ".join(failures)) if failures else "Alle Prüfungen bestanden.")
    return 1 if failures else 0


if __name__ == "__main__":
    sys.exit(main())
