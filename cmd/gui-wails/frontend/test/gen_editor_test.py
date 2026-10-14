"""Generator with editor (gen_editor.js): in Generator mode the Player's
timeline sits under the Generator video and the edit tools join the settings
column; editing is on for the loaded script; the Generator video moves the
timeline's playhead; leaving the Generator gives everything back to the Player.

Uses the screenshot script's demo data (needs ffmpeg).
Run: python3 cmd/gui-wails/frontend/test/gen_editor_test.py
"""

import pathlib
import shutil
import sys

from playwright.sync_api import sync_playwright

HERE = pathlib.Path(__file__).resolve().parent
sys.path.insert(0, str(HERE.parents[3] / "scripts" / "screenshots"))
import capture_gui as cg  # noqa: E402
from _harness import Checker, app_stub, serve  # noqa: E402


def main():
    check = Checker()
    if not shutil.which("ffmpeg"):
        print("skipped: ffmpeg is needed for the test-pattern video")
        return 0
    media = HERE / "_gen_editor_media"
    media.mkdir(exist_ok=True)
    harness = HERE / "_gen_editor_harness.html"
    cg.make_video(media / "clip.webm")
    links = "".join(f'<link rel="stylesheet" href="/src/{css}?real" />'
                    for css in ("style.css", "theme_flow.css", "flow_fx.css", "training_rings.css"))
    harness.write_text((HERE.parent / "index.html").read_text()
                       .replace('<link rel="stylesheet" href="/src/style.css" />', links))
    stubs = dict(cg.STUBS)
    stubs["SetGeneratorVideo"] = "async () => '/test/_gen_editor_media/clip.webm'"
    stubs["SaveScriptAxisActions"] = "async (axis, acts) => { window.__saved = (window.__saved || []).concat([[axis, acts.length]]); }"
    base, shutdown = serve(cg.data_js("/test/_gen_editor_media/clip.webm") + app_stub(stubs))
    try:
        with sync_playwright() as p:
            b = p.chromium.launch()
            page = b.new_page(viewport={"width": 1600, "height": 960})
            errors = []
            page.on("pageerror", lambda e: errors.append(str(e)))
            page.add_init_script("try{localStorage.clear()}catch(e){}")
            page.goto(f"{base}/test/{harness.name}")
            page.wait_for_timeout(800)
            mode = lambda m: (page.click("#mode-pill"), page.click(f'#mode-deck [data-mode="{m}"]'), page.wait_for_timeout(300))

            # No script yet: the Generator's editor says how to fill it.
            mode("generator")
            check("Generator has an Editor", page.is_visible(".gl-editor"))
            check("empty: hint, no timeline", page.is_visible("#gl-editor-empty")
                  and page.is_hidden('.gl-editor [data-pb-region="bottom"]'))
            check("empty: no edit tools", page.is_hidden(".gl-edit-tools"))

            # A script in the Player, then the Generator borrows its timeline.
            mode("player")
            page.click("#pb-choose")
            page.wait_for_function("!document.querySelector('#pb-ruler').hidden", timeout=8000)
            check("Player owns the timeline", page.evaluate("!!document.querySelector('#tab-playback [data-pb-region=\"bottom\"]')"))
            mode("generator")
            check("Generator: timeline under the video", page.evaluate("!!document.querySelector('.gl-editor [data-pb-region=\"bottom\"]')")
                  and page.is_visible(".gl-editor #pb-curve"))
            check("Generator: edit tools in the settings column", page.is_visible(".gl-edit-tools #pb-insp-view")
                  and page.is_visible(".gl-edit-tools #pb-curve-edit"))
            page.wait_for_function("document.querySelector('#pb-curve-edit').checked", timeout=4000)
            check("Generator: editing on", True)
            check("Generator: script name shown", "Demo clip" in page.text_content("#gl-editor-name"))

            # Edit in the Generator: a click on an empty spot adds a point and saves.
            box = page.locator(".gl-editor #pb-curve").bounding_box()
            page.mouse.click(box["x"] + box["width"] * 0.5, box["y"] + 6)
            page.wait_for_function("(window.__saved || []).length >= 1", timeout=4000)
            check("Generator: a new point is saved", page.evaluate("window.__saved[0][1]") > 0, str(page.evaluate("window.__saved")))

            # Generator video → timeline playhead.
            page.click("#gen-choose")
            page.wait_for_function("document.querySelector('#gen-video') && document.querySelector('#gen-video').readyState >= 1", timeout=8000)
            page.wait_for_function("document.querySelector('#pb-video').readyState >= 1", timeout=8000)
            page.evaluate("document.querySelector('#gen-video').currentTime = 42")
            page.wait_for_function("Math.abs(document.querySelector('#pb-video').currentTime - 42) < 0.5", timeout=4000)
            check("Generator video moves the timeline", True)

            # Back to the Player: everything returns, editing off again.
            mode("player")
            check("Player gets its timeline back", page.evaluate("!!document.querySelector('#tab-playback [data-pb-region=\"bottom\"]')")
                  and page.evaluate("!document.querySelector('.gl-editor [data-pb-region=\"bottom\"]')"))
            check("Player gets its View tools back", page.evaluate("!!document.querySelector('#tab-playback #pb-insp-view')"))
            check("Player: editing off again", not page.is_checked("#pb-curve-edit"))
            check("no page errors", not errors, "; ".join(errors[:5]))
            b.close()
    finally:
        shutdown()
        harness.unlink(missing_ok=True)
        shutil.rmtree(media, ignore_errors=True)
    return check.report()


if __name__ == "__main__":
    sys.exit(main())
