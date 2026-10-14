"""Generator "How it feels" follows the device (gen_feel.js): Sam Neo 2
shows contact vibration and the suction feel; a vibrator hides the suction
feel; a stroker hides both; "Show all" brings everything back. The Sam Neo 2
feel (scene / mix / follow) is written after Create — and nothing is written
for Classic + centre mix + follow off (the old mapping stays).

Run: python3 cmd/gui-wails/frontend/test/generator_feel_device_test.py
"""

import pathlib
import sys

from playwright.sync_api import sync_playwright

from _harness import Checker, app_stub, serve

FRONTEND = pathlib.Path(__file__).resolve().parents[1]

PAGE = """<!doctype html><html><head><link rel="stylesheet" href="/src/style.css?real" /></head>
<body><div id="root"></div>
<script type="module">
  import { initGenerator } from '/src/generator.js';
  window.__playLoads = [];
  localStorage.clear();
  initGenerator(document.querySelector('#root'), {
    loadScriptPath: (path, opts) => { window.__playLoads.push([path, opts || {}]); },
  });
  window.__ready = true;
</script></body></html>"""

TINY_PNG_B64 = ("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk"
                "+A8AAQUBAScY42YAAAAASUVORK5CYII=")

STROKER = "{connected:true, capabilities:{device:'OSR2', features:[{channel:'stroke',kind:'linear'}]}}"
VIBRATOR = "{connected:true, capabilities:{device:'Lush', features:[{channel:'vibration',kind:'scalar'}]}}"
NEO2 = "{connected:true, capabilities:{device:'Sam Neo 2', features:[{channel:'vibration'},{channel:'suction'}]}}"


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
        "GetDeviceStatus": "async () => ({connected:false})",
        "GenerateScript": (
            "async () => { setTimeout(() => window.__triggerEvent('generate:done', { path: '/tmp/clip.funscript', "
            "qualityScore: 0.9, qualityPassed: true, pipeline: 'go', tracking: 'csrt', backend: 'csrt' }), 0); return {}; }"
        ),
        "ImproveGeneratedScript": "async (req) => ({ path: req.path, gapsFilled: 0, pointsAdded: 0, message: 'ok' })",
        "SaveFeelSettings": "async (scene, follow, balance) => { window.__feel = (window.__feel || []).concat([[scene, follow, balance]]); }",
        "GetScriptChapterMarks": "async () => []",
    }))
    harness = FRONTEND / "test" / "_generator_feel_device_harness.html"
    harness.write_text(PAGE)
    try:
        with sync_playwright() as p:
            b = p.chromium.launch()
            page = b.new_page(viewport={"width": 1600, "height": 960})
            errors = []
            page.on("pageerror", lambda e: errors.append(str(e)))
            page.on("dialog", lambda d: d.dismiss())
            page.goto(f"{base}/test/{harness.name}")
            page.wait_for_function("window.__ready === true")
            status = lambda js: page.evaluate(f"window.dispatchEvent(new CustomEvent('device:status', {{detail: {js}}}))")
            shown = lambda sel: page.evaluate(f"(() => {{ const e = document.querySelector('{sel}'); return !!e && getComputedStyle(e).display !== 'none'; }})()")

            page.click("#gen-choose")
            page.wait_for_function("document.querySelector('#gen-autoroi').disabled === false", timeout=5000)

            check("For device: auto · Sam Neo 2 · vibrator · stroker",
                  page.evaluate("[...document.querySelectorAll('#gen-device-target option')].map(o => o.value)") == ["auto", "neo2", "vibrator", "stroker"])
            check("no device: Sam Neo 2 assumed, contact + suction feel shown",
                  page.get_attribute("#root", "data-gen-device") == "neo2" and shown("#gen-contact-vibration-wrap") and shown("#gen-neo-feel"))

            status(STROKER)
            check("stroker: contact and suction feel hidden",
                  page.get_attribute("#root", "data-gen-device") == "stroker"
                  and not shown("#gen-contact-vibration-wrap") and not shown("#gen-neo-feel"))
            check("stroker: says why", "Stroker" in page.text_content("#gen-device-hint"))
            page.check("#gen-show-all-feel")
            check("Show all brings everything back", shown("#gen-contact-vibration-wrap") and shown("#gen-neo-feel"))
            page.uncheck("#gen-show-all-feel")

            status(VIBRATOR)
            check("vibrator: contact shown, suction feel hidden",
                  shown("#gen-contact-vibration-wrap") and not shown("#gen-neo-feel"))

            status(NEO2)
            check("Sam Neo 2 connected: all shown", shown("#gen-contact-vibration-wrap") and shown("#gen-neo-feel"))
            check("scene choices", page.evaluate("[...document.querySelectorAll('#gen-scene-feel option')].map(o => o.value)")
                  == ["auto", "", "oral", "intercourse", "breast", "hand"])

            # Auto scene from the contact mark (mouth → Oral).
            page.evaluate("const s = document.querySelector('#gen-region-class2'); const o = new Option('mouth', 'mouth'); s.add(o); s.value = 'mouth'; s.dispatchEvent(new Event('change'))")
            check("auto scene from the contact mark", page.text_content("#gen-scene-auto") == "Auto: Oral")

            # Classic + centre + follow off → nothing written (old mapping).
            page.evaluate("document.querySelector('#gen-neo-feel').open = true")
            page.select_option("#gen-scene-feel", "")
            page.click("#gen-generate")
            page.wait_for_function("window.__playLoads.length >= 1", timeout=5000)
            page.wait_for_timeout(300)
            check("Classic: feel not written (old mapping)", page.evaluate("window.__feel") is None)

            # Oral, mix toward suction, vibration follows → written after Create.
            page.evaluate("document.querySelector('#gen-neo-feel').open = true")
            page.select_option("#gen-scene-feel", "oral")
            page.eval_on_selector("#gen-feel-balance", "e => { e.value = '40'; e.dispatchEvent(new Event('change')); }")
            page.check("#gen-vib-follow")
            check("summary shows the feel", "Oral" in page.text_content("#gen-neo-feel-sum")
                  and "suction 40%" in page.text_content("#gen-neo-feel-sum"))
            page.wait_for_function("!document.querySelector('#gen-generate').disabled", timeout=5000)
            page.click("#gen-generate")
            page.wait_for_function("(window.__feel || []).length === 1", timeout=5000)
            check("Oral feel written after Create", page.evaluate("window.__feel[0]") == ["oral", True, 0.4], str(page.evaluate("window.__feel")))
            check("settings remembered", '"scene":"oral"' in (page.evaluate("localStorage.getItem('gen.feel')") or ""))

            # A stroker never gets a suction feel.
            status(STROKER)
            page.wait_for_function("!document.querySelector('#gen-generate').disabled", timeout=5000)
            page.click("#gen-generate")
            page.wait_for_timeout(600)
            check("stroker: no feel written", len(page.evaluate("window.__feel")) == 1)
            check("no page errors", not errors, "; ".join(errors))
            b.close()
    finally:
        shutdown()
        harness.unlink(missing_ok=True)
    return check.report()


if __name__ == "__main__":
    sys.exit(main())
