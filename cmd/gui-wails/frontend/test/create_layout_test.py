"""V1 Create layout (create_layout.js, target visual-target/create.png):
Project column · centre (project header, step bar with sub-labels, preview
+ source clips) · settings column with the Create button. Elements are
moved, not copied; the Scene Map view keeps its own stacked look.

Run: python3 cmd/gui-wails/frontend/test/create_layout_test.py
"""

import pathlib
import sys

from playwright.sync_api import sync_playwright

from _harness import Checker, app_stub, serve

HERE = pathlib.Path(__file__).resolve().parent
PNG = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg=="


def main():
    check = Checker()
    html = (HERE.parent / "index.html").read_text().replace('href="/src/style.css"', 'href="/src/style.css?real"')
    harness = HERE / "_create_layout_harness.html"
    harness.write_text(html)
    base, shutdown = serve("window.__picked = [];\n" + app_stub({
        "GetEULA": "async () => ({version:'x', text:'', accepted:true})",
        "IsSuiteBuild": "async () => false",
        "CurrentVersion": "async () => 'test'",
        "GetSettings": "async () => ({updateCheckOnStartup:false})",
        "PickVideoFile": "async () => '/v/first clip.mp4'",
        "LoadFirstFrame": f"async (p) => {{ window.__picked.push(p); return ({{width: 64, height: 36, pngBase64: '{PNG}'}}); }}",
        "LoadFrameAt": f"async () => ({{width: 64, height: 36, pngBase64: '{PNG}'}})",
        "ScriptExistsForVideo": "async () => false",
        "ProbeCreateVideo": f"async (p) => ({{path: p, name: p.split('/').pop(), width: 1920, height: 1080, fps: 30, vfr: false, durationSec: 95, codec: 'h264', posterUrl: 'data:image/png;base64,{PNG}'}})",
    }))
    try:
        with sync_playwright() as p:
            b = p.chromium.launch()
            page = b.new_page(viewport={"width": 1600, "height": 960})
            errors = []
            page.on("pageerror", lambda e: errors.append(str(e)))
            page.add_init_script("try{localStorage.clear(); localStorage.setItem('pb.recentVideos', JSON.stringify(['/v/older clip.mp4']))}catch(e){}")
            page.goto(f"{base}/test/_create_layout_harness.html")
            page.wait_for_timeout(800)
            page.click('#ws-view [data-ws-view="create"]')
            page.wait_for_timeout(300)
            gen = "#tab-generator"
            check("layout v1 applied", page.get_attribute(gen, "data-gen-layout") == "v1")
            check("three columns", page.evaluate(f"getComputedStyle(document.querySelector('{gen}')).gridTemplateColumns.split(' ').length") == 3)
            check("page title gives way to the project header", page.is_hidden(".create-head"))
            check("handbook in the Project column", page.evaluate("!!document.querySelector('.gl-nav #gen-open-handbook')"))
            subs = page.evaluate("[...document.querySelectorAll('.gen-step-item small')].map(e => e.textContent)")
            check("step bar has sub-labels", subs == ["Add a clip", "Find the motion", "Shape the curves", "Build the script", "Refine and play"], str(subs))
            check("preview empty state before a video", page.is_visible(".gl-preview-empty") and page.is_hidden("#roi-canvas-wrap"))
            check("chooser lives in the Source clips card", page.evaluate("!!document.querySelector('.gl-clips #gen-choose')"))
            check("Create button in the settings column, disabled", page.evaluate("!!document.querySelector('.gl-side #gen-generate')")
                  and page.is_disabled("#gen-generate"))
            check("settings placeholder before a video", page.is_visible(".gl-side-empty"))
            check("generated scripts: honest empty text", "Nothing created yet" in page.text_content("#gl-nav-generated"))

            page.click("#gen-choose")
            page.wait_for_function("!document.querySelector('#gen-project-head').hidden", timeout=5000)
            page.wait_for_timeout(300)
            check("canvas in the preview card", page.evaluate("!!document.querySelector('.gl-preview #roi-canvas')") and page.is_visible("#roi-canvas-wrap"))
            check("single clip card mirrors the header", page.is_visible(".gl-single-clip")
                  and page.text_content(".gl-single-clip .gen-clip-name") == "first clip.mp4")
            check("Source clips (1)", page.text_content(".gl-clips .gl-card-title") == "Source clips (1)")
            check("Feel settings shown, placeholder gone", page.is_visible("#gen-step-motion") and page.is_hidden(".gl-side-empty"))
            check("step tools live in the right settings column", page.evaluate("!!document.querySelector('.gl-side .gl-tools #gen-step-region')"))
            check("workspace stays in view (no jump to step 2)", page.evaluate("document.querySelector('#gen-steps').getBoundingClientRect().top") >= 0)

            page.click("#gl-nav-recent .gl-nav-link >> text=older clip.mp4")
            page.wait_for_function("window.__picked.includes('/v/older clip.mp4')", timeout=5000)
            check("Recent videos loads the clip into Create", True)

            page.click("#gl-nav-scenemap")
            page.wait_for_timeout(300)
            check("Scene Map view: Project column and Feel settings hidden", page.is_hidden(".gl-nav") and page.is_hidden("#gen-step-motion") and page.is_hidden(".gl-run"))
            check("Scene Map view: video chooser still there", page.is_visible("#gen-choose"))
            check("no page errors", errors == [], str(errors[:3]))
            b.close()
    finally:
        shutdown()
        harness.unlink(missing_ok=True)
    return check.report()


if __name__ == "__main__":
    sys.exit(main())
