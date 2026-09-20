"""Sequential Generate workflow: steps appear one after another.

Only step 1 (Video) is visible at first. After choose → region panel,
after ROI1 → motion panel, after regions ready → generate panel.
Review stays hidden until a successful generate:done.

Run: python3 cmd/gui-wails/frontend/test/generator_workflow_test.py
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


def visible(page, sel):
    return page.locator(sel).evaluate("e => !e.hidden && e.getAttribute('hidden') === null")


def main():
    check = Checker()
    base, shutdown = serve(app_stub({
        "PickVideoFile": "async () => '/tmp/video.mp4'",
        "LoadFirstFrame": f"async () => ({{ width: 640, height: 360, "
                          f"pngBase64: '{TINY_PNG_B64}' }})",
        "LoadFrameAt": f"async () => ({{ width: 640, height: 360, "
                       f"pngBase64: '{TINY_PNG_B64}' }})",
        "SuggestPipeline": "async () => ({ Backend: 'csrt', Profile: 'standard', "
                           "Reason: 'test', GoPath: true })",
        "SuggestProfile": "async () => ({ found: false })",
        "CheckAIRoiAvailable": "async () => false",
        "CheckAudioCheckAvailable": "async () => false",
        "ScriptExistsForVideo": "async () => false",
    }))
    harness = FRONTEND / "test" / "_generator_workflow_harness.html"
    harness.write_text(PAGE)

    with sync_playwright() as p:
        browser = p.chromium.launch()
        page = browser.new_page()
        page.on("dialog", lambda d: d.dismiss())
        page.on("pageerror", lambda e: print("   [pageerror]", e))
        page.goto(f"{base}/test/_generator_workflow_harness.html")
        page.wait_for_function("window.__ready === true")

        check("Step strip present", page.locator("#gen-steps").count() == 1)
        check("Step 1 video visible at start", visible(page, "#gen-step-video"))
        check("Step 2 region hidden at start", not visible(page, "#gen-step-region"))
        check("Step 3 motion hidden at start", not visible(page, "#gen-step-motion"))
        check("Step 4 generate hidden at start", not visible(page, "#gen-step-run"))
        check("Step 5 review hidden at start", not visible(page, "#gen-step-result"))
        check("Current step is 1",
              page.locator('.gen-step-item[data-step="1"]').evaluate(
                  "e => e.classList.contains('is-current')"))

        page.click("#gen-choose")
        page.wait_for_function(
            "document.querySelector('#gen-step-region') && "
            "!document.querySelector('#gen-step-region').hidden",
            timeout=5000)
        check("After video: region step appears", visible(page, "#gen-step-region"))
        check("After video: motion still hidden", not visible(page, "#gen-step-motion"))
        check("After video: generate still hidden", not visible(page, "#gen-step-run"))

        box = page.locator("#roi-canvas").bounding_box()
        page.mouse.move(box["x"] + 40, box["y"] + 40)
        page.mouse.down()
        page.mouse.move(box["x"] + 120, box["y"] + 120, steps=5)
        page.mouse.up()
        page.wait_for_function(
            "!document.querySelector('#gen-step-motion').hidden", timeout=5000)
        check("After ROI1: motion step appears", visible(page, "#gen-step-motion"))
        check("After ROI1 (standard): generate appears", visible(page, "#gen-step-run"))
        check("Generate button enabled",
              page.locator("#gen-generate").is_enabled())

        # Switch to Tf/Tj → generate should hide until ROI2
        page.select_option("#gen-profile", "tf")
        page.wait_for_function(
            "document.querySelector('#gen-step-run').hidden === true", timeout=5000)
        check("Tf without ROI2: generate step hidden again",
              not visible(page, "#gen-step-run"))
        check("Prompt mentions 2nd region",
              "2nd" in page.locator("#gen-step-prompt").inner_text().lower()
              or "tf" in page.locator("#gen-step-prompt").inner_text().lower())

        box = page.locator("#roi-canvas").bounding_box()
        page.keyboard.down("Shift")
        page.mouse.move(box["x"] + 200, box["y"] + 40)
        page.mouse.down()
        page.mouse.move(box["x"] + 280, box["y"] + 120, steps=5)
        page.mouse.up()
        page.keyboard.up("Shift")
        page.wait_for_function(
            "() => { const l = document.querySelector('#gen-roi2-label');"
            " return l && !l.textContent.startsWith('No '); }",
            timeout=5000)
        page.wait_for_function(
            "!document.querySelector('#gen-step-run').hidden", timeout=5000)
        check("After ROI2: generate step back", visible(page, "#gen-step-run"))
        check("Review still hidden until done", not visible(page, "#gen-step-result"))

        browser.close()

    shutdown()
    harness.unlink(missing_ok=True)
    return check.report()


if __name__ == "__main__":
    sys.exit(main())
