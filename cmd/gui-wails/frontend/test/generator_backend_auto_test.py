"""Regressionstest für den Generator-Tab: region_fusion_auto (das neue
"4 Zonen automatisch, ganzes Bild" Verfahren, siehe
generator/region_fusion_auto_backend.py).

Anders als region_fusion/csrt/grid_lk braucht region_fusion_auto KEINE
manuell markierte Region - wie "flow" bestimmt es seine Zonen selbst aus dem
ganzen Bild. Vorher verlangte der "Funscript generieren"-Knopf IMMER eine
markierte Region, unabhängig vom gewählten Verfahren (generator.js's
updateGenerateEnabled() prüfte nur `!roi`) - das hätte "keine Region nötig"
für dieses neue Verfahren (und eigentlich auch für "flow") an der Oberfläche
ausgehebelt. Geprüft wird hier: der Knopf bleibt bei csrt (braucht eine
Region) ohne Markierung gesperrt, wird aber bei region_fusion_auto (und
flow) ohne jede Markierung nutzbar, sobald ein Video geladen ist - und bleibt
weiterhin gesperrt, solange gar kein Video geladen ist.

Ausführen:  python3 cmd/gui-wails/frontend/test/generator_backend_auto_test.py
"""

import pathlib
import sys

from playwright.sync_api import sync_playwright

from _harness import Checker, app_stub, serve

FRONTEND = pathlib.Path(__file__).resolve().parents[1]

PAGE = """<!doctype html><html><body><div id="root"></div>
<script type="module">
  import { initGenerator } from '/src/generator.js';
  initGenerator(document.querySelector('#root'), { loadScriptPath: () => {} });
  window.__ready = true;
</script></body></html>"""

TINY_PNG_B64 = ("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk"
                "+A8AAQUBAScY42YAAAAASUVORK5CYII=")


def main():
    check = Checker()
    base, shutdown = serve(app_stub({
        "PickVideoFile": "async () => '/tmp/video.mp4'",
        "LoadFirstFrame": f"async () => ({{ width: 640, height: 360, "
                          f"pngBase64: '{TINY_PNG_B64}' }})",
        "LoadFrameAt": f"async () => ({{ width: 640, height: 360, "
                       f"pngBase64: '{TINY_PNG_B64}' }})",
        "SuggestPipeline": "async () => ({ Backend: 'csrt', Profile: 'standard', Reason: 'test', GoPath: true })",
        "CheckAIRoiAvailable": "async () => false",
        "CheckAudioCheckAvailable": "async () => false",
    }))
    harness = FRONTEND / "test" / "_generator_backend_auto_harness.html"
    harness.write_text(PAGE)

    with sync_playwright() as p:
        browser = p.chromium.launch()
        page = browser.new_page()
        page.on("dialog", lambda d: d.dismiss())
        page.on("pageerror", lambda e: print("   [pageerror]", e))
        page.goto(f"{base}/test/_generator_backend_auto_harness.html")
        page.wait_for_function("window.__ready === true")

        options = page.eval_on_selector_all("#gen-backend option", "els => els.map(e => e.value)")
        check("Dropdown bietet region_fusion_auto an",
              "region_fusion_auto" in options, str(options))

        check("Generieren-Knopf ist ohne geladenes Video gesperrt (auch bei "
              "region_fusion_auto gewählt)",
              page.eval_on_selector("#gen-generate", "e => e.disabled") is True)

        page.click("#gen-advanced summary")
        page.select_option("#gen-backend", "region_fusion_auto")
        check("Generieren-Knopf bleibt ohne Video gesperrt, auch nach Auswahl von "
              "region_fusion_auto",
              page.eval_on_selector("#gen-generate", "e => e.disabled") is True)

        page.click("#gen-choose")
        page.wait_for_function(
            "document.querySelector('#gen-autoroi').disabled === false", timeout=5000)

        # --- Video geladen, KEINE Region markiert, csrt gewählt: gesperrt ----
        page.select_option("#gen-backend", "csrt")
        check("csrt (braucht eine Region) bleibt ohne Markierung gesperrt",
              page.eval_on_selector("#gen-generate", "e => e.disabled") is True)

        # --- Auf region_fusion_auto wechseln, immer noch keine Region --------
        page.select_option("#gen-backend", "region_fusion_auto")
        check("region_fusion_auto wird ohne jede Regionsmarkierung nutzbar, sobald ein "
              "Video geladen ist",
              page.eval_on_selector("#gen-generate", "e => e.disabled") is False)

        # --- flow zeigt dasselbe Verhalten (bestehendes Backend, jetzt ---------
        # ebenfalls vom backendNeedsRoi()-Gate erfasst) -------------------------
        page.select_option("#gen-backend", "flow")
        check("flow wird ebenfalls ohne Regionsmarkierung nutzbar (bestehendes Backend, "
              "jetzt korrekt erfasst)",
              page.eval_on_selector("#gen-generate", "e => e.disabled") is False)

        # --- Zurück auf csrt: wieder gesperrt, weil weiterhin keine Region ----
        page.select_option("#gen-backend", "csrt")
        check("Zurück auf csrt sperrt den Knopf wieder (weiterhin keine Region markiert)",
              page.eval_on_selector("#gen-generate", "e => e.disabled") is True)

        # --- Zwei-Punkt-Messung: region_fusion_auto wird gesperrt --------------
        page.select_option("#gen-backend", "region_fusion_auto")
        box = page.locator("#roi-canvas").bounding_box()
        page.click("#gen-roi2-toggle")
        page.mouse.move(box["x"] + 40, box["y"] + 40)
        page.mouse.down()
        page.mouse.move(box["x"] + 120, box["y"] + 120, steps=5)
        page.mouse.up()
        page.wait_for_function(
            "document.querySelector('#gen-backend option[value=\"region_fusion_auto\"]')"
            ".disabled === true", timeout=5000)
        check("region_fusion_auto wird bei Zwei-Punkt-Messung gesperrt", True)
        check("Backend wird bei Zwei-Punkt-Messung automatisch auf csrt zurückgesetzt",
              page.locator("#gen-backend").input_value() == "csrt",
              page.locator("#gen-backend").input_value())

        browser.close()

    shutdown()
    harness.unlink(missing_ok=True)

    return check.report()


if __name__ == "__main__":
    sys.exit(main())
