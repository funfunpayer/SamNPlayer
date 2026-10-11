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
    }))
    try:
        with sync_playwright() as p:
            b = p.chromium.launch()
            page = b.new_page(viewport={"width": 1366, "height": 768})
            page.on("pageerror", lambda e: print("   [pageerror]", e))
            page.goto(f"{base}/test/_tools_nav_harness.html")
            page.wait_for_timeout(800)
            rail = page.evaluate("[...document.querySelectorAll('#rail .tab-btn')].filter(b => b.offsetParent).map(b => b.textContent.trim())")
            check("rail shows Play · Generator · Training · Tools", rail == ["Play", "Generator", "Training", "Tools"], str(rail))
            check("tagline is Generate · Play · Train",
                  page.text_content("#brand-tagline").strip() == "Generate · Play · Train")
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
