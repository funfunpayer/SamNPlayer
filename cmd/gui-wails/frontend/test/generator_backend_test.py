"""Regressionstest für den Generator-Tab: Tracking-Verfahren-Dropdown.

Prüft, dass region_fusion (das neue 4-Teilregionen-Verfahren, siehe
generator/region_fusion_backend.py) als Option angeboten wird, und dass es -
wie schon "flow" - bei einer Zwei-Punkt-Messung (2. Region markiert)
gesperrt wird: generate_funscript.py hat für region_fusion keinen eigenen
Zwei-Punkt-Pfad und würde sonst still auf CSRT zurückfallen (siehe dessen
"Hinweis: --backend ... unterstützt keine Zwei-Punkt-Messung"). Die
Oberfläche soll das nicht erst anbieten und dann stillschweigend
überschreiben.

Ausführen:  python3 cmd/gui-wails/frontend/test/generator_backend_test.py
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
        "CheckAIRoiAvailable": "async () => false",
        "CheckAudioCheckAvailable": "async () => false",
    }))
    harness = FRONTEND / "test" / "_generator_backend_harness.html"
    harness.write_text(PAGE)

    with sync_playwright() as p:
        browser = p.chromium.launch()
        page = browser.new_page()
        page.on("dialog", lambda d: d.dismiss())
        page.on("pageerror", lambda e: print("   [pageerror]", e))
        page.goto(f"{base}/test/_generator_backend_harness.html")
        page.wait_for_function("window.__ready === true")

        options = page.eval_on_selector_all("#gen-backend option", "els => els.map(e => e.value)")
        check("Dropdown bietet region_fusion an",
              "region_fusion" in options, str(options))

        # #gen-backend steckt in <details id="gen-advanced"> - erst öffnen,
        # sonst gilt es Playwright als nicht sichtbar für select_option/
        # is_disabled (anders als eval_on_selector_all oben, das keine
        # Sichtbarkeit braucht).
        page.click("#gen-advanced summary")

        page.click("#gen-choose")
        page.wait_for_function(
            "document.querySelector('#gen-autoroi').disabled === false", timeout=5000)

        box = page.locator("#roi-canvas").bounding_box()

        # --- 1. Region markieren, region_fusion wählen -------------------------
        page.mouse.move(box["x"] + 40, box["y"] + 40)
        page.mouse.down()
        page.mouse.move(box["x"] + 120, box["y"] + 120, steps=5)
        page.mouse.up()
        page.select_option("#gen-backend", "region_fusion")
        check("region_fusion mit nur 1 Region wählbar (nicht gesperrt)",
              not page.locator('#gen-backend option[value="region_fusion"]').is_disabled())

        # --- 2. Region markieren: region_fusion wird gesperrt und auf csrt zurückgesetzt ---
        page.click("#gen-roi2-toggle")
        page.mouse.move(box["x"] + 200, box["y"] + 40)
        page.mouse.down()
        page.mouse.move(box["x"] + 280, box["y"] + 120, steps=5)
        page.mouse.up()

        page.wait_for_function(
            "document.querySelector('#gen-backend option[value=\"region_fusion\"]').disabled === true",
            timeout=5000)
        check("region_fusion wird bei Zwei-Punkt-Messung gesperrt", True)
        check("Backend wird bei Zwei-Punkt-Messung automatisch auf csrt zurückgesetzt",
              page.locator("#gen-backend").input_value() == "csrt",
              page.locator("#gen-backend").input_value())
        check("flow ist ebenfalls gesperrt (bestehendes Verhalten, unverändert)",
              page.locator('#gen-backend option[value="flow"]').is_disabled())

        browser.close()

    shutdown()
    harness.unlink(missing_ok=True)

    return check.report()


if __name__ == "__main__":
    sys.exit(main())
