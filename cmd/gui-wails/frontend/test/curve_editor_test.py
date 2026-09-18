"""Regressionstest für den Kurven-Editor (docs/NEXT.md "Later" - manueller
Funscript-Editor).

Anders als die reine Anzeige (curve_display_test.py) prüft dieser Test das
eigentliche Bearbeiten: Editiermodus einschalten lädt die vollen Punkte
(nicht die resamplete Anzeigekurve), Ziehen eines Punkts verschiebt ihn
und speichert per SaveScriptActions, Klick auf freie Stelle legt einen
neuen Punkt an, Doppelklick löscht einen Punkt (mit Mindestanzahl-Schutz),
und während der Wiedergabe ist die Checkbox gesperrt.

Ausführen:  python3 cmd/gui-wails/frontend/test/curve_editor_test.py
"""

import pathlib
import sys

from playwright.sync_api import sync_playwright

from _harness import Checker, app_stub, serve

PAGE = """<!doctype html><html><body><div id="root"></div>
<script type="module">
  import { initPlayback } from '/src/playback.js';
  window.__pb = initPlayback(document.querySelector('#root'));
  window.__ready = true;
</script></body></html>"""


def main():
    check = Checker()
    base, shutdown = serve(app_stub({
        "PickFunscriptFile": "async () => '/tmp/test.funscript'",
        "LoadFunscript": "async () => ({ path: '/tmp/test.funscript', actionCount: 3, "
                         "durationMs: 100000, videoPath: '', hasVideo: false })",
        "GetScriptCurve": "async () => [{ atMs: 0, pos: 0 }, { atMs: 50000, pos: 50 }, "
                          "{ atMs: 100000, pos: 100 }]",
        "GetScriptActions": "async () => [{ at: 0, pos: 0 }, { at: 50000, pos: 50 }, "
                            "{ at: 100000, pos: 100 }]",
        "SaveScriptActions": "async (actions) => { window.__calls.push(['saveActions', actions]); }",
        "GetHeatmap": "async n => Array.from({ length: n }, (_, i) => "
                      "({ atMs: i * 1000, intensity: 0.5 }))",
        "GetMarker": "async () => null",
        "VideoFileURL": "async () => ''",
        "GetSpeedHighlights": "async () => []",
        "ExportScriptHeatmapPNG": "async () => '/tmp/h.png'",
        "SavePlaybackProject": "async () => '/tmp/p.snp.json'",
        "EditCapSpeedRange": "async () => {}",
        "EditDeleteRange": "async () => {}",
        "SnapTimeMs": "async (t, fps) => t",
        "GetScriptOffset": "async () => 0",
        "GetOMarkers": "async () => []",
        "SaveOMarkers": "async () => {}",
    }))

    harness = pathlib.Path(__file__).resolve().parent / "_curve_editor_harness.html"
    harness.write_text(PAGE)

    with sync_playwright() as p:
        browser = p.chromium.launch()
        page = browser.new_page()
        page.on("dialog", lambda d: d.dismiss())
        page.on("pageerror", lambda e: print("   [pageerror]", e))
        page.goto(f"{base}/test/_curve_editor_harness.html")
        page.wait_for_function("window.__ready === true")

        check("vor dem Laden: Editieren-Zeile verborgen",
              page.locator("#pb-curve-edit-row").evaluate("e => e.style.display") == "none")

        page.click("#pb-choose")
        page.wait_for_function(
            "document.querySelector('#pb-curve-edit-row').style.display === 'flex'", timeout=5000)
        check("nach dem Laden: Editieren-Zeile sichtbar", True)
        check("Hinweistext zunächst verborgen",
              page.locator("#pb-curve-edit-hint").evaluate("e => e.style.display") == "none")

        page.check("#pb-curve-edit")
        page.wait_for_function(
            "document.querySelector('#pb-curve-edit-hint').style.display === 'block'", timeout=5000)
        check("Editiermodus zeigt Hinweistext", True)

        box = page.locator("#pb-curve").bounding_box()

        # Mittlerer Punkt liegt bei (50000ms, pos=50) -> 50% Breite, mittige
        # Höhe (pad=6, usableH=98 bei height=110: y = 6 + (1-0.5)*98 = 55).
        mid_x = box["x"] + box["width"] * 0.5
        mid_y = box["y"] + 55

        # Diesen Punkt nach oben ziehen (Richtung pos=100, kleineres y).
        page.mouse.move(mid_x, mid_y)
        page.mouse.down()
        page.mouse.move(mid_x, box["y"] + 6, steps=5)
        page.mouse.up()

        page.wait_for_function("window.__calls.some(c => c[0] === 'saveActions')", timeout=5000)
        saved = page.evaluate("window.__calls.filter(c => c[0] === 'saveActions').pop()[1]")
        check("Ziehen speichert 3 Punkte (kein neuer, keiner verloren)",
              len(saved) == 3, str(saved))
        middle = next(a for a in saved if 40000 <= a["at"] <= 60000)
        check("gezogener Punkt liegt jetzt nahe pos=100 (nach oben gezogen)",
              middle["pos"] >= 90, str(saved))

        # Klick auf eine freie Stelle (25% Breite, tief unten = niedrige
        # Position) legt einen neuen Punkt an.
        free_x = box["x"] + box["width"] * 0.25
        free_y = box["y"] + 100
        page.mouse.move(free_x, free_y)
        page.mouse.down()
        page.mouse.up()
        page.wait_for_function(
            "window.__calls.filter(c => c[0] === 'saveActions').length === 2", timeout=5000)
        saved2 = page.evaluate("window.__calls.filter(c => c[0] === 'saveActions').pop()[1]")
        check("Klick auf freie Stelle legt einen vierten Punkt an",
              len(saved2) == 4, str(saved2))

        # Doppelklick auf den (jetzt) ersten Punkt (at=0, pos=0, ganz links
        # unten) löscht ihn wieder.
        first_x = box["x"] + 1
        first_y = box["y"] + box["height"] - 7
        page.mouse.move(first_x, first_y)
        page.dblclick("#pb-curve", position={"x": 1, "y": box["height"] - 7})
        page.wait_for_function(
            "window.__calls.filter(c => c[0] === 'saveActions').length === 3", timeout=5000)
        saved3 = page.evaluate("window.__calls.filter(c => c[0] === 'saveActions').pop()[1]")
        check("Doppelklick auf einen Punkt löscht ihn",
              len(saved3) == 3 and not any(a["at"] == 0 for a in saved3), str(saved3))

        # Während der Wiedergabe ist die Checkbox gesperrt und der
        # Editiermodus wird beendet.
        page.evaluate("window.__pb")  # no-op, hält Referenz für Debug-Zwecke
        page.evaluate("document.querySelector('#pb-play').click()")
        page.wait_for_function(
            "document.querySelector('#pb-curve-edit').disabled === true", timeout=5000)
        check("Wiedergabe sperrt die Editieren-Checkbox", True)
        check("Wiedergabe beendet einen aktiven Editiermodus",
              not page.locator("#pb-curve-edit").is_checked())

        browser.close()

    shutdown()
    harness.unlink(missing_ok=True)
    return check.report()


if __name__ == "__main__":
    sys.exit(main())
