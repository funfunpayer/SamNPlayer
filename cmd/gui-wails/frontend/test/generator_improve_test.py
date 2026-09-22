"""Review improve panel: trim / fill gaps / audio toggles after generate.

Run: python3 cmd/gui-wails/frontend/test/generator_improve_test.py
"""

import pathlib
import sys

from playwright.sync_api import sync_playwright

from _harness import Checker, app_stub, serve

FRONTEND = pathlib.Path(__file__).resolve().parents[1]

PAGE = """<!doctype html><html><body><div id="root"></div>
<script type="module">
  import { initGenerator } from '/src/generator.js';
  window.__playLoads = [];
  initGenerator(document.querySelector('#root'), {
    loadScriptPath: (path, opts) => { window.__playLoads.push([path, opts || {}]); },
  });
  window.__ready = true;
</script></body></html>"""

TINY_PNG_B64 = ("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk"
                "+A8AAQUBAScY42YAAAAASUVORK5CYII=")


def main():
    check = Checker()
    base, shutdown = serve(app_stub({
        "PickVideoFile": "async () => '/tmp/clip.mp4'",
        "LoadFirstFrame": f"async () => ({{ width: 640, height: 360, "
                          f"pngBase64: '{TINY_PNG_B64}' }})",
        "LoadFrameAt": f"async () => ({{ width: 640, height: 360, "
                       f"pngBase64: '{TINY_PNG_B64}' }})",
        "SuggestPipeline": "async () => ({ Backend: 'csrt', Profile: 'standard', "
                           "Reason: 'test', GoPath: true })",
        "SuggestProfile": "async () => ({ found: false })",
        "CheckAIRoiAvailable": "async () => false",
        "CheckAudioCheckAvailable": "async () => true",
        "ScriptExistsForVideo": "async () => false",
        "GenerateScript": (
            "async (opts) => { window.__calls.push(['GenerateScript', opts]); "
            "setTimeout(() => window.__triggerEvent && window.__triggerEvent("
            "'generate:done', { path: '/tmp/clip.funscript', qualityScore: 0.9, "
            "qualityPassed: true, pipeline: 'go', tracking: 'csrt', backend: 'csrt' }), 0); "
            "return {}; }"
        ),
        "ImproveGeneratedScript": (
            "async (req) => { window.__calls.push(['ImproveGeneratedScript', req]); "
            "return { path: req.path, beforeCount: 10, afterCount: 14, trimmed: true, "
            "gapsFilled: 1, pointsAdded: 4, message: 'trimmed start/end · filled 1 gap(s)' }; }"
        ),
    }))
    harness = FRONTEND / "test" / "_generator_improve_harness.html"
    harness.write_text(PAGE)

    with sync_playwright() as p:
        browser = p.chromium.launch()
        page = browser.new_page()
        page.on("dialog", lambda d: d.dismiss())
        page.on("pageerror", lambda e: print("   [pageerror]", e))
        page.goto(f"{base}/test/_generator_improve_harness.html")
        page.wait_for_function("window.__ready === true")

        # Improve panel hidden until generate done.
        check("Improve panel hidden at start",
              page.locator("#gen-improve").evaluate("e => e.style.display === 'none'"))

        # AI second opinion / Advanced audio row removed from visible Advanced.
        check("AI quality not a visible Advanced checkbox",
              page.locator("label[for='gen-ai-quality']").count() == 0)
        check("Audio check not a visible Advanced checkbox",
              page.locator("label[for='gen-audio-check']").count() == 0)

        page.click("#gen-choose")
        page.wait_for_function(
            "document.querySelector('#gen-autoroi').disabled === false && "
            "!document.querySelector('#gen-roi-label').textContent.includes('No ')",
            timeout=5000)
        page.click("#gen-generate")
        page.wait_for_function(
            "document.querySelector('#gen-improve') && "
            "document.querySelector('#gen-improve').style.display !== 'none'",
            timeout=5000)

        # Auto fill-gaps runs on generate:done before Play load.
        page.wait_for_function(
            "() => (window.__calls || []).some(c => c[0] === 'ImproveGeneratedScript')",
            timeout=5000)
        auto = page.evaluate(
            "() => (window.__calls || []).filter(c => c[0] === 'ImproveGeneratedScript')")
        check("Auto fill-gaps after Generate",
              auto and auto[0][1].get("fillGaps") is True, str(auto))

        check("Improve panel shown after generate",
              page.locator("#gen-improve").evaluate("e => e.style.display !== 'none'"))
        check("Fill gaps on by default",
              page.locator("#gen-improve-fill").is_checked())
        check("Audio check on by default when ffmpeg ok",
              page.locator("#gen-improve-audio").is_checked())

        page.fill("#gen-improve-start", "1")
        page.fill("#gen-improve-end", "40")
        page.evaluate("window.__calls = (window.__calls || []).filter(c => c[0] !== 'ImproveGeneratedScript')")
        page.click("#gen-improve-apply")
        page.wait_for_function(
            "() => (window.__calls || []).some(c => c[0] === 'ImproveGeneratedScript')",
            timeout=5000)
        calls = page.evaluate(
            "() => (window.__calls || []).filter(c => c[0] === 'ImproveGeneratedScript')")
        check("ImproveGeneratedScript called from button", len(calls) == 1, str(calls))
        req = calls[0][1] if calls else {}
        check("Trim start passed", req.get("startSec") == 1, str(req))
        check("Trim end passed", req.get("endSec") == 40, str(req))
        check("Fill gaps passed", req.get("fillGaps") is True, str(req))
        check("Audio check passed", req.get("audioCheck") is True, str(req))

        page.wait_for_function("window.__playLoads && window.__playLoads.length > 0", timeout=3000)
        loads = page.evaluate("window.__playLoads")
        check("Improve/Generate reloads Play for editor",
              loads and any(l[1].get("review") is True for l in loads),
              str(loads))

        browser.close()

    shutdown()
    harness.unlink(missing_ok=True)
    return check.report()


if __name__ == "__main__":
    sys.exit(main())
