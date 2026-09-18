"""ROI neu ziehen darf manuell gewähltes Tracking-Verfahren nicht überschreiben.

Bugbot/Sept 2026: autoApplyPipeline setzte bei jedem mouseup Backend+Profil
aus SuggestPipeline — Flow/grid_lk gingen verloren.
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
        "SuggestPipeline": "async (w,h,w2,h2) => ({ Backend: 'csrt', "
                           "Profile: 'standard', Reason: 'test', GoPath: true })",
        "CheckAIRoiAvailable": "async () => false",
        "CheckAudioCheckAvailable": "async () => false",
    }))
    harness = FRONTEND / "test" / "_generator_backend_preserve_harness.html"
    harness.write_text(PAGE)

    with sync_playwright() as p:
        browser = p.chromium.launch()
        page = browser.new_page()
        page.on("dialog", lambda d: d.dismiss())
        page.goto(f"{base}/test/_generator_backend_preserve_harness.html")
        page.wait_for_function("window.__ready === true")

        page.click("#gen-advanced summary")
        page.click("#gen-choose")
        page.wait_for_function(
            "document.querySelector('#gen-autoroi').disabled === false", timeout=5000)

        box = page.locator("#roi-canvas").bounding_box()
        page.mouse.move(box["x"] + 40, box["y"] + 40)
        page.mouse.down()
        page.mouse.move(box["x"] + 120, box["y"] + 120, steps=5)
        page.mouse.up()
        page.wait_for_timeout(200)

        page.select_option("#gen-backend", "flow")
        check("Flow manuell gewählt", page.locator("#gen-backend").input_value() == "flow")

        page.mouse.move(box["x"] + 50, box["y"] + 50)
        page.mouse.down()
        page.mouse.move(box["x"] + 140, box["y"] + 130, steps=5)
        page.mouse.up()
        page.wait_for_timeout(200)

        check("Flow bleibt nach ROI-Neuzeichnen erhalten",
              page.locator("#gen-backend").input_value() == "flow",
              page.locator("#gen-backend").input_value())

        browser.close()
    shutdown()
    harness.unlink(missing_ok=True)
    return check.report()


if __name__ == "__main__":
    sys.exit(main())
