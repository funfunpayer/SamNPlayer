"""Main tabs (Owner 11 Oct, #475): Play · Generator · Training · Tools.
Tools bundles Device / Log / Settings behind a sub-navigation; old entry
points (switchTab('device'), .tab-btn[data-tab="log"].click()) still work.

Run: python3 cmd/gui-wails/frontend/test/tools_nav_test.py
"""

import pathlib
import sys

from playwright.sync_api import sync_playwright

from _harness import Checker, app_stub, serve

HERE = pathlib.Path(__file__).resolve().parent


def main():
    check = Checker()
    html = (HERE.parent / "index.html").read_text().replace('href="/src/style.css"', 'href="/src/style.css?real"')
    harness = HERE / "_tools_nav_harness.html"
    harness.write_text(html)
    base, shutdown = serve(app_stub({
        "GetEULA": "async () => ({version:'x', text:'', accepted:true})",
        "IsStudioBuild": "async () => false",
        "CurrentVersion": "async () => 'test'",
        "GetSettings": "async () => ({updateCheckOnStartup:false})",
        "PickFunscriptFile": "async () => { window.__picked = (window.__picked || 0) + 1; return ''; }",
    }))
    try:
        with sync_playwright() as p:
            b = p.chromium.launch()
            page = b.new_page(viewport={"width": 1366, "height": 768})
            page.on("pageerror", lambda e: print("   [pageerror]", e))
            page.goto(f"{base}/test/_tools_nav_harness.html")
            page.wait_for_timeout(800)
            rail = page.evaluate("[...document.querySelectorAll('#rail .tab-btn')].filter(b => b.offsetParent).map(b => b.textContent.trim())")
            check("top tabs: Player · Generator · Training · Scene Map · Tools",
                  rail == ["Player", "Generator", "Training", "Scene Map", "Tools"], str(rail))
            check("tabs live in the top bar", page.evaluate("!!document.querySelector('#topbar #rail')"))
            check("tagline is Generate · Play · Train",
                  page.get_attribute("#brand-tagline", "aria-label") == "Generate · Play · Train"
                  and " ".join(page.text_content("#brand-tagline").split()) == "Generate Play Train")
            check("tools sub-nav hidden on Play", page.is_hidden("#tools-nav"))

            page.click('#rail .tab-btn[data-tab="tools"]')
            check("Tools opens Device first", page.is_visible("#tab-device"))
            check("sub-nav visible in Tools", page.is_visible("#tools-nav"))
            check("rail Tools marked active",
                  page.get_attribute('#rail .tab-btn[data-tab="tools"]', "aria-selected") == "true")
            sub = page.evaluate("[...document.querySelectorAll('#tools-nav .tab-btn')].filter(b => b.offsetParent).map(b => b.textContent.trim())")
            check("customer sub-nav: Device · Log · Settings", sub == ["Device", "Log", "Settings"], str(sub))

            page.click('#tools-nav .tab-btn[data-tab="settings"]')
            check("Settings panel shown", page.is_visible("#tab-settings") and page.is_hidden("#tab-device"))
            page.click('#rail .tab-btn[data-tab="playback"]')
            check("back on Play hides sub-nav", page.is_hidden("#tools-nav") and page.is_visible("#tab-playback"))
            page.click('#rail .tab-btn[data-tab="tools"]')
            check("Tools remembers the last section (Settings)", page.is_visible("#tab-settings"))

            # Old entry points keep working.
            page.click('#rail .tab-btn[data-tab="playback"]')
            page.evaluate("""document.querySelector('.tab-btn[data-tab="log"]').click()""")
            check(".tab-btn[data-tab=log].click() opens Log inside Tools",
                  page.is_visible("#tab-log") and page.is_visible("#tools-nav"))
            page.click('#rail .tab-btn[data-tab="playback"]')
            page.click("#topbar-device")
            check("top-bar device button opens Tools → Device", page.is_visible("#tab-device"))

            # Scene Map: focused view on the Generator's scene map (same IDs).
            page.click('#rail .tab-btn[data-tab="scenemap"]')
            check("Scene Map shows the Generator panel in scene-map view",
                  page.is_visible("#tab-generator")
                  and page.get_attribute("#tab-generator", "data-view") == "scenemap")
            check("scene map controls visible", page.is_visible("#gen-scene-map"))
            check("video chooser visible in Scene Map", page.is_visible("#gen-choose"))
            check("other Generator steps hidden in Scene Map", page.is_hidden("#gen-steps"))
            page.click('#rail .tab-btn[data-tab="generator"]')
            check("Generator tab leaves scene-map view",
                  page.get_attribute("#tab-generator", "data-view") is None and page.is_visible("#gen-steps"))
            # Top-bar actions.
            page.click("#topbar-settings")
            check("gear opens Tools → Settings", page.is_visible("#tab-settings"))
            page.click("#topbar-open")
            check("folder opens the Player and the file picker",
                  page.is_visible("#tab-playback") and page.evaluate("window.__picked") == 1)
            page.click('#rail .tab-btn[data-tab="tools"]')

            # Keyboard inside the sub-nav.
            page.focus('#tools-nav .tab-btn[data-tab="device"]')
            page.keyboard.press("ArrowRight")
            check("→ in sub-nav moves to Log", page.is_visible("#tab-log"))
            check("studio sections stay hidden in the customer build",
                  page.is_hidden('#tools-nav .tab-btn[data-tab="benchmark"]'))
            b.close()
    finally:
        shutdown()
        harness.unlink(missing_ok=True)
    return check.report()


if __name__ == "__main__":
    sys.exit(main())
