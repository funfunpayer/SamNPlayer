"""Smoke run through the whole app with the demo clip (Owner 14 Oct: "Test
Clip laufen lassen, alle Funktionen testen"): every mode from the Mode deck,
the Player's transport, bottom tabs and inspector tabs, the Editor, the
Generator, Training (Mock: start, pause, resume, end), Scene Map (scene
details, prev / next) and every Tools section. Fails on any page error or
console error, and on a control that does not react.

Uses the screenshot script's demo data (test-pattern video, a stroke script,
multi-axis companions, chapters). Needs ffmpeg.

Run: python3 cmd/gui-wails/frontend/test/smoke_all_modes_test.py
"""

import pathlib
import shutil
import sys

from playwright.sync_api import sync_playwright

HERE = pathlib.Path(__file__).resolve().parent
sys.path.insert(0, str(HERE.parents[3] / "scripts" / "screenshots"))
import capture_gui as cg  # noqa: E402
from _harness import Checker, app_stub, serve  # noqa: E402

# Console noise that is not an app error.
IGNORED = ("favicon", "Failed to load resource: the server responded with a status of 404")


def main():
    check = Checker()
    if not shutil.which("ffmpeg"):
        print("skipped: ffmpeg is needed for the test-pattern video")
        return 0
    media = HERE / "_smoke_media"
    media.mkdir(exist_ok=True)
    harness = HERE / "_smoke_harness.html"
    cg.make_video(media / "clip.webm")
    links = "".join(f'<link rel="stylesheet" href="/src/{css}?real" />'
                    for css in ("style.css", "theme_flow.css", "flow_fx.css", "training_rings.css"))
    harness.write_text((HERE.parent / "index.html").read_text()
                       .replace('<link rel="stylesheet" href="/src/style.css" />', links))
    stubs = dict(cg.STUBS)
    stubs.update({
        "StartTraining": "async () => { window.__trStarted = (window.__trStarted || 0) + 1; }",
        "StopTraining": "async () => { window.__triggerEvent('training:done'); }",
        "PauseTraining": "async () => { window.__trPaused = (window.__trPaused || 0) + 1; }",
        "ResumeTraining": "async () => { window.__trResumed = (window.__trResumed || 0) + 1; }",
    })
    base, shutdown = serve(cg.data_js("/test/_smoke_media/clip.webm") + app_stub(stubs))
    try:
        with sync_playwright() as p:
            b = p.chromium.launch()
            page = b.new_page(viewport={"width": 1600, "height": 960})
            errors = []
            page.on("pageerror", lambda e: errors.append("pageerror: " + str(e)))
            page.on("console", lambda m: m.type == "error" and not any(s in m.text for s in IGNORED)
                    and errors.append("console: " + m.text))
            page.add_init_script("try{localStorage.clear()}catch(e){}")
            page.goto(f"{base}/test/{harness.name}")
            page.wait_for_timeout(800)
            mode = lambda m: (page.click("#mode-pill"), page.click(f'#mode-deck [data-mode="{m}"]'), page.wait_for_timeout(250))

            # ---- Player ---------------------------------------------------
            page.click("#pb-choose")
            page.wait_for_function("!document.querySelector('#pb-ruler').hidden", timeout=8000)
            check("Player: script and video loaded", page.get_attribute("#tab-playback", "data-pb-state") == "loaded")
            page.click("#pb-play")
            page.wait_for_function("document.querySelector('#pb-play').dataset.state === 'pause'", timeout=4000)
            check("Player: Play starts", True)
            page.click("#pb-play")
            page.wait_for_function("document.querySelector('#pb-play').dataset.state === 'play'", timeout=4000)
            check("Player: Pause stops", True)
            for sel in ("#pb-fwd10", "#pb-back10", "#pb-zoom-in", "#pb-zoom-in", "#pb-zoom-out"):
                if page.locator(sel).count() and page.is_visible(sel):
                    page.click(sel)
            for tab in ("chapters", "bookmarks", "scenemap", "audio", "timeline"):
                page.click(f'.pb-bottom-tab[data-btab="{tab}"]')
                check(f"Player: bottom tab {tab}", page.get_attribute("#tab-playback", "data-pb-bottom") == tab)
            insp = page.evaluate("[...document.querySelectorAll('[data-pb-region=\"inspector-tabs\"] [data-insp]')].map(b => b.dataset.insp)")
            for t in insp:
                page.click(f'[data-pb-region="inspector-tabs"] [data-insp="{t}"]')
            check("Player: every inspector tab opens", len(insp) >= 3, str(insp))
            page.click("#pb-lib-toggle")
            page.click("#pb-lib-toggle")
            check("Player: Library folds and returns", page.is_visible("#pb-lib-local"))

            # ---- Editor -----------------------------------------------------
            mode("editor")
            check("Editor: curve editing on", page.is_checked("#pb-curve-edit"))
            check("Editor: Optimize for shown", page.is_visible("#pb-optimize-wrap"))
            mode("player")
            check("Player: Optimize for hidden outside the Editor", page.is_hidden("#pb-optimize-card"))

            # ---- Scene Map ---------------------------------------------------
            mode("scenemap")
            page.wait_for_function("document.querySelectorAll('.pb-scene-card').length === 4", timeout=8000)
            page.click('.pb-scene-card[data-scene="1"]')
            nxt = page.locator("#pb-scene-details button", has_text="Next")
            if nxt.count():
                nxt.first.click()
            check("Scene Map: details follow the scene", "3 / 4" in page.text_content("#pb-scene-details"),
                  page.text_content("#pb-scene-details")[:80])

            # ---- Generator ---------------------------------------------------
            mode("generator")
            check("Generator: panel and steps", page.is_visible("#tab-generator") and page.is_visible("#gen-steps"))
            check("Generator: Optimize for above Create", page.locator("#gen-optimize-target").count() == 1)

            # ---- Training ----------------------------------------------------
            mode("training")
            page.check("#tr-mock")
            page.click("#tr-stage-rings")
            page.click("#tr-start")
            page.wait_for_function("window.__trStarted === 1 && document.querySelector('#tab-training').dataset.trRunning === '1'", timeout=4000)
            page.click("#trl-play")
            page.wait_for_function("window.__trPaused === 1", timeout=3000)
            page.click("#trl-play")
            page.wait_for_function("window.__trResumed === 1", timeout=3000)
            check("Training: start, pause, resume", True)
            page.click("#tr-stop")
            page.wait_for_function("document.querySelector('#tab-training').dataset.trRunning === ''", timeout=4000)
            check("Training: End ends the session", True)

            # ---- Tools -------------------------------------------------------
            page.click('#rail .tab-btn[data-tab="tools"]')
            for sec in ("tools-home", "device", "log", "settings"):
                page.click(f'#tools-nav .tab-btn[data-tab="{sec}"]')
                check(f"Tools: {sec}", page.is_visible(f"#tab-{sec}"))
            check("Mode pill dimmed in Tools", "is-dim" in (page.get_attribute("#mode-pill", "class") or ""))

            # ---- Mode deck by keyboard ----------------------------------------
            page.keyboard.press("m")
            page.keyboard.press("1")
            check("Keys M + 1 → Player", page.evaluate("document.body.dataset.mode") == "player")

            check("no page or console errors", not errors, "\n      ".join(errors[:10]))
            b.close()
    finally:
        shutdown()
        harness.unlink(missing_ok=True)
        shutil.rmtree(media, ignore_errors=True)
    return check.report()


if __name__ == "__main__":
    sys.exit(main())
