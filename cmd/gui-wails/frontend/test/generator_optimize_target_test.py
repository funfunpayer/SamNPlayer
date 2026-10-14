"""Generator: "Optimize for" (any device / Sam Neo 2) runs on the new script
after Create and opens the optimized file; Off keeps the script as created.

Run: python3 cmd/gui-wails/frontend/test/generator_optimize_target_test.py
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
        "LoadFirstFrame": f"async () => ({{ width: 640, height: 360, pngBase64: '{TINY_PNG_B64}' }})",
        "LoadFrameAt": f"async () => ({{ width: 640, height: 360, pngBase64: '{TINY_PNG_B64}' }})",
        "SuggestPipeline": "async () => ({ Backend: 'csrt', Profile: 'standard', Reason: 'test', GoPath: true })",
        "SuggestProfile": "async () => ({ found: false })",
        "CheckAIRoiAvailable": "async () => false",
        "CheckAudioCheckAvailable": "async () => false",
        "ScriptExistsForVideo": "async () => false",
        "GenerateScript": (
            "async (opts) => { setTimeout(() => window.__triggerEvent && window.__triggerEvent("
            "'generate:done', { path: '/tmp/clip.funscript', qualityScore: 0.9, qualityPassed: true, "
            "pipeline: 'go', tracking: 'csrt', backend: 'csrt' }), 0); return {}; }"
        ),
        "ImproveGeneratedScript": "async (req) => ({ path: req.path, gapsFilled: 0, pointsAdded: 0, message: 'ok' })",
        "OptimizeLoadedFor": (
            "async (target) => { window.__opt = (window.__opt || []).concat([target]); "
            "return { path: '/tmp/clip.samn', message: 'Neo 2 ready · axes baked' }; }"
        ),
        "GetScriptChapterMarks": "async () => []",
    }))
    harness = FRONTEND / "test" / "_generator_optimize_target_harness.html"
    harness.write_text(PAGE)
    try:
        with sync_playwright() as p:
            browser = p.chromium.launch()
            page = browser.new_page()
            page.on("dialog", lambda d: d.dismiss())
            errors = []
            page.on("pageerror", lambda e: errors.append(str(e)))
            page.goto(f"{base}/test/{harness.name}")
            page.wait_for_function("window.__ready === true")
            opts = page.evaluate("[...document.querySelectorAll('#gen-optimize-target option')].map(o => o.value)")
            check("Optimize for: Off · Any device · Sam Neo 2", opts == ["", "general", "neo2"], str(opts))
            check("Off by default", page.input_value("#gen-optimize-target") == "")

            page.click("#gen-choose")
            page.wait_for_function("document.querySelector('#gen-autoroi').disabled === false", timeout=5000)

            # Off: no optimization.
            page.click("#gen-generate")
            page.wait_for_function("window.__playLoads.length >= 1", timeout=5000)
            page.wait_for_timeout(300)
            check("Off: nothing optimized", page.evaluate("window.__opt") is None)

            # Sam Neo 2: optimized after Create, the optimized file opens.
            page.select_option("#gen-optimize-target", "neo2")
            check("choice remembered", page.evaluate("localStorage.getItem('gen.optimizeTarget')") == "neo2")
            n = page.evaluate("window.__playLoads.length")
            page.wait_for_function("!document.querySelector('#gen-generate').disabled", timeout=5000)
            page.click("#gen-generate")
            page.wait_for_function("(window.__opt || []).length === 1", timeout=5000)
            check("Sam Neo 2: OptimizeLoadedFor('neo2')", page.evaluate("window.__opt") == ["neo2"])
            page.wait_for_function(f"window.__playLoads.length >= {n + 2}", timeout=5000)
            check("optimized file opens in the Editor",
                  page.evaluate("window.__playLoads.some(l => l[0] === '/tmp/clip.samn' && l[1].review)"))
            check("no page errors", not errors, "; ".join(errors))
            browser.close()
    finally:
        shutdown()
        harness.unlink(missing_ok=True)
    return check.report()


if __name__ == "__main__":
    sys.exit(main())
