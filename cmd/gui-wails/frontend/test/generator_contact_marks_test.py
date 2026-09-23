"""Contact marks UI: only when Contact vibration is on; clearer labels.

Run: python3 cmd/gui-wails/frontend/test/generator_contact_marks_test.py
"""

import pathlib

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
        "PickVideoFile": "async () => '/tmp/clip.mp4'",
        "LoadFirstFrame": f"async () => ({{ width: 1280, height: 720, "
                          f"pngBase64: '{TINY_PNG_B64}' }})",
        "LoadFrameAt": f"async () => ({{ width: 1280, height: 720, "
                       f"pngBase64: '{TINY_PNG_B64}' }})",
        "SuggestPipeline": "async () => ({ Backend: 'csrt', Profile: 'standard', "
                           "Reason: 'everyday', GoPath: true })",
        "SuggestProfile": "async () => ({ found: false })",
        "CheckAIRoiAvailable": "async () => true",
        "CheckAudioCheckAvailable": "async () => true",
        "ScriptExistsForVideo": "async () => false",
        "GenerateScript": "async () => ({ path: '/tmp/clip.funscript' })",
    }))
    harness = FRONTEND / "test" / "_generator_contact_marks_harness.html"
    harness.write_text(PAGE)

    with sync_playwright() as p:
        browser = p.chromium.launch()
        page = browser.new_page()
        page.on("dialog", lambda d: d.dismiss())
        page.goto(f"{base}/test/_generator_contact_marks_harness.html")
        page.wait_for_function("window.__ready === true")

        page.click("#gen-choose")
        page.wait_for_function(
            "document.querySelector('#gen-autoroi').disabled === false",
            timeout=5000)

        check("Contact vib on by default",
              page.locator("#gen-contact-vibration").is_checked())
        check("Contact marks wrap visible when vib on",
              page.locator("#gen-contact-marks-wrap").is_visible())
        check("Tip class inside contact wrap when vib on",
              page.locator("#gen-contact-marks-wrap #gen-region-class").is_visible())
        check("Default tip class glans",
              page.locator("#gen-region-class").input_value() == "glans")
        check("Mark contact area button present",
              page.locator("#gen-roi2-toggle").inner_text().lower().find("contact") >= 0)
        check("Default contact type nipples",
              page.locator("#gen-region-class2").input_value() == "nipples")
        check("4-zone everyday button hidden",
              page.locator("#gen-nomark").is_hidden())
        check("4-zone not in Tracking method dropdown",
              "region_fusion_auto" not in page.eval_on_selector_all(
                  "#gen-backend option", "els => els.map(e => e.value)"))

        page.uncheck("#gen-contact-vibration")
        page.wait_for_timeout(50)
        check("Contact marks wrap hidden when vib off",
              page.locator("#gen-contact-marks-wrap").is_hidden())
        check("Tip class hidden when vib off",
              page.locator("#gen-region-class").is_hidden())

        page.check("#gen-contact-vibration")
        page.wait_for_timeout(50)
        check("Contact marks wrap returns when vib on again",
              page.locator("#gen-contact-marks-wrap").is_visible())
        check("Tip class returns when vib on again",
              page.locator("#gen-region-class").is_visible())

        browser.close()

    shutdown()
    harness.unlink(missing_ok=True)
    return check.report()


if __name__ == "__main__":
    raise SystemExit(main())
