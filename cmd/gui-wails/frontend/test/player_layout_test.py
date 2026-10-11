"""V1 Player shell (S3, docs/V1_PLAN.md): Library · main · Inspector with
tabs Devices / Audio / Playback / View, bottom tabs Timeline / Chapters /
Bookmarks / Scene Map / Audio Analysis. Existing controls are moved, not
re-created — ids stay unique and keep working.

Run: python3 cmd/gui-wails/frontend/test/player_layout_test.py
"""

import json
import pathlib
import sys

from playwright.sync_api import sync_playwright

from _harness import Checker, app_stub, serve

HERE = pathlib.Path(__file__).resolve().parent


def main():
    check = Checker()
    acts = [{"at": i * 500, "pos": 10 if i % 2 else 90} for i in range(120)]
    html = (HERE.parent / "index.html").read_text().replace('href="/src/style.css"', 'href="/src/style.css?real"')
    harness = HERE / "_player_layout_harness.html"
    harness.write_text(html)
    base, shutdown = serve(app_stub({
        "GetEULA": "async () => ({version:'x', text:'', accepted:true})",
        "IsStudioBuild": "async () => false",
        "CurrentVersion": "async () => 'test'",
        "GetSettings": "async () => ({updateCheckOnStartup:false, playbackMock:true})",
        "PickFunscriptFile": "async () => '/s/demo.samn'",
        "LoadFunscript": "async (p) => { window.__loaded = (window.__loaded || []).concat([p]); return ({path:p, actionCount:%d, durationMs:60000, videoPath:'', hasVideo:false}); }" % len(acts),
        "GetScriptCurve": "async () => %s" % json.dumps([{"atMs": a["at"], "pos": a["pos"]} for a in acts]),
        "GetScriptAxisActions": "async (axis) => (!axis || axis === 'general') ? %s : null" % json.dumps(acts),
        "GetHeatmap": "async n => Array.from({ length: n }, (_, i) => ({ atMs: i * 200, intensity: 0.5 }))",
        "GetMarker": "async () => null",
        "VideoFileURL": "async () => ''",
    }))
    try:
        with sync_playwright() as p:
            b = p.chromium.launch()
            page = b.new_page(viewport={"width": 1600, "height": 900})
            errors = []
            page.on("pageerror", lambda e: errors.append(str(e)))
            page.add_init_script("try{localStorage.clear()}catch(e){}")
            page.goto(f"{base}/test/_player_layout_harness.html")
            page.wait_for_timeout(800)
            root = "document.querySelector('#tab-playback')"
            check("layout v1 applied", page.evaluate(f"{root}.dataset.pbLayout") == "v1")
            for r in ["library", "main", "inspector-tabs", "bottom", "bottom-tabs"]:
                check(f"region '{r}' present", page.evaluate(f"!!document.querySelector('[data-pb-region=\"{r}\"]')"))
            dup = page.evaluate("""() => { const seen = {}; let d = [];
                document.querySelectorAll('#tab-playback [id]').forEach(e => { if (seen[e.id]) d.push(e.id); seen[e.id] = 1; }); return d; }""")
            check("no duplicate ids after moving controls", dup == [], str(dup[:5]))

            check("Library visible", page.is_visible(".pb-library"))
            check("Recent starts empty", "Nothing yet" in page.text_content("#pb-lib-recent"))
            page.click("#pb-lib-local")
            page.wait_for_function(f"{root}.dataset.pbState === 'loaded'", timeout=5000)
            check("Local Files opens the script picker and loads", page.evaluate("window.__loaded") == ["/s/demo.samn"])
            page.wait_for_function("document.querySelector('#pb-lib-recent').textContent.includes('demo.samn')", timeout=3000)
            check("Recent lists the loaded script", True)
            page.click(".pb-lib-recent-item")
            page.wait_for_function("window.__loaded.length === 2", timeout=3000)
            check("Recent entry reloads it", page.evaluate("window.__loaded[1]") == "/s/demo.samn")

            tabs = page.evaluate("[...document.querySelectorAll('.pb-insp-tab')].map(b => b.textContent.trim())")
            check("Inspector tabs Devices · Audio · Playback · View", tabs == ["Devices", "Audio", "Playback", "View"], str(tabs))
            check("Devices: channel rows Stroke / Vibration / Suction",
                  page.evaluate("[...document.querySelectorAll('.pb-chan-name')].map(e => e.textContent)") == ["Stroke", "Vibration", "Suction"])
            check("Devices shows the axis row", page.is_visible("#pb-axis-row"))
            page.click('.pb-insp-tab[data-insp="audio"]')
            check("Audio tab: BPM and audio vibration", page.is_visible("#pb-bpm-auto") and page.is_visible("#pb-audio-vibration"))
            check("Devices content hidden on Audio tab", page.is_hidden("#pb-axis-row"))
            page.click('.pb-insp-tab[data-insp="playback"]')
            check("Playback tab: Script Tools (offset)", page.is_visible("#pb-script-tools-card"))
            page.click('.pb-insp-tab[data-insp="view"]')
            check("View tab: 0–100 overlay control", page.is_visible("#pb-pos-overlay-toggle") or page.evaluate("!!document.querySelector('#pb-insp-view #pb-pos-overlay-row')"))
            page.keyboard.press("ArrowLeft")
            check("← on inspector tabs", page.evaluate(f"{root}.dataset.pbInspector") in ("view", "playback"))

            btabs = page.evaluate("[...document.querySelectorAll('.pb-bottom-tab')].map(b => b.textContent.trim())")
            check("bottom tabs as mockup", btabs == ["Timeline", "Chapters", "Bookmarks", "Scene Map", "Audio Analysis"], str(btabs))
            check("Timeline tab shows the curve", page.is_visible("#pb-curve"))
            page.click('.pb-bottom-tab[data-btab="chapters"]')
            check("Chapters tab shows the chapter section", page.is_visible("#pb-chapter-details"))
            check("curve hidden while Chapters shown", page.is_hidden("#pb-curve"))
            page.click('.pb-bottom-tab[data-btab="timeline"]')
            page.click('.pb-bottom-tab[data-btab="scenemap"]')
            check("Scene Map bottom tab opens the Scene Map view",
                  page.get_attribute("#tab-generator", "data-view") == "scenemap")
            page.click('#rail .tab-btn[data-tab="playback"]')
            page.click("#pb-lib-bpm")
            check("Library → BPM Detection opens Inspector Audio", page.evaluate(f"{root}.dataset.pbInspector") == "audio")
            check("no page errors", errors == [], str(errors[:3]))
            b.close()
    finally:
        shutdown()
        harness.unlink(missing_ok=True)
    return check.report()


if __name__ == "__main__":
    sys.exit(main())
