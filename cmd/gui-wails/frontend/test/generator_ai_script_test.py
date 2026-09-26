"""Create Advanced: AI draft script control loads and stays disabled in S0.

Owner path docs/AI_SCRIPT_WRITER.md — Everyday CSRT unchanged; button present
so the lane is visible in the GUI (Owner: functions must load in the GUI).

Run: python3 cmd/gui-wails/frontend/test/generator_ai_script_test.py
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
        "PickVideoFile": "async () => '/tmp/clip.mp4'",
        "LoadFirstFrame": f"async () => ({{ width: 1280, height: 720, "
                          f"pngBase64: '{TINY_PNG_B64}' }})",
        "LoadFrameAt": f"async () => ({{ width: 1280, height: 720, "
                       f"pngBase64: '{TINY_PNG_B64}' }})",
        "SuggestPipeline": "async () => ({ Backend: 'csrt', Profile: 'standard', "
                           "Reason: 'everyday', GoPath: true })",
        "SuggestProfile": "async () => ({ found: false })",
        "CheckAIRoiAvailable": "async () => false",
        "CheckAudioCheckAvailable": "async () => true",
        "ScriptExistsForVideo": "async () => false",
        "SceneMapAvailable": "async () => false",
        "AIScriptWriterStatus": (
            "async () => ({ available: false, reason: "
            "'No AI draft model configured yet (experimental path; Everyday Create still uses CSRT).', "
            "stage: 'S0' })"
        ),
        "DraftAIScript": (
            "async () => { throw new Error('aiscript: not available'); }"
        ),
    }))
    harness = FRONTEND / "test" / "_generator_ai_script_harness.html"
    harness.write_text(PAGE)

    with sync_playwright() as p:
        browser = p.chromium.launch()
        page = browser.new_page()
        page.on("dialog", lambda d: d.dismiss())
        page.on("pageerror", lambda e: print("   [pageerror]", e))
        page.goto(f"{base}/test/_generator_ai_script_harness.html")
        page.wait_for_function("window.__ready === true")

        check("AI draft button in DOM", page.locator("#gen-ai-script-draft").count() == 1)

        page.click("#gen-choose")
        page.wait_for_function(
            "!document.querySelector('#gen-step-run').hidden",
            timeout=5000)

        page.wait_for_function(
            "!document.querySelector('#gen-step-run').hidden",
            timeout=5000)
        page.locator("#gen-advanced").evaluate("e => { e.open = true; }")
        page.wait_for_selector("#gen-ai-script-draft", state="visible", timeout=5000)
        check("disabled in S0", page.is_disabled("#gen-ai-script-draft"))
        status = page.locator("#gen-ai-script-status").inner_text()
        check("status explains S0 / CSRT",
              "S0" in status or "CSRT" in status or "draft" in status.lower()
              or "model" in status.lower(),
              status)

        browser.close()

    shutdown()
    harness.unlink(missing_ok=True)
    return check.report()


if __name__ == "__main__":
    sys.exit(main())
