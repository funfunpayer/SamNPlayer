"""Main tabs (Owner 11 Oct): Player · Training · Tools. The Player is the
workspace with the view switch Play · Create · Edit (Create = Generator,
Scene Map inside Create); data-tab "generator" / "scenemap" stay as routes.
Tools bundles Home / Device / Log / Settings behind a sub-navigation; old entry
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
        "IsSuiteBuild": "async () => false",
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
            check("top bar: Mode pill + Tools", rail == ["Tools"] and page.is_visible("#mode-pill"), str(rail))
            check("Mode pill names the mode", page.text_content("#mode-pill-name") == "Player")
            check("Play · Edit toggle gone (Editor is a mode)", page.is_hidden("#ws-view"))
            check("deck closed at start", page.is_hidden("#mode-deck"))
            page.click("#mode-pill")
            check("pill opens the deck", page.is_visible("#mode-deck") and page.get_attribute("#mode-pill", "aria-expanded") == "true")
            cards = page.evaluate("[...document.querySelectorAll('#mode-deck .mode-card')].filter(b => b.offsetParent).map(b => b.dataset.mode)")
            check("deck: Player · Editor · Generator · Training · Scene Map",
                  cards == ["player", "editor", "generator", "training", "scenemap"], str(cards))
            check("current mode marked", page.get_attribute('#mode-deck [data-mode="player"]', "aria-current") == "true")
            page.keyboard.press("Escape")
            check("Esc closes the deck", page.is_hidden("#mode-deck") and page.get_attribute("#mode-pill", "aria-expanded") == "false")
            page.keyboard.press("m")
            check("M opens the deck", page.is_visible("#mode-deck"))
            page.keyboard.press("4")
            check("key 4 = Training", page.is_visible("#tab-training") and page.is_hidden("#mode-deck")
                  and page.text_content("#mode-pill-name") == "Training")
            page.click("#mode-pill"); page.click(".mode-deck-backdrop", position={"x": 5, "y": 700})
            check("click outside closes the deck", page.is_hidden("#mode-deck"))
            page.click('#mode-pill'); page.click('#mode-deck [data-mode="player"]')
            check("mode hook: player", page.evaluate("document.body.dataset.mode") == "player")
            check("Play view", page.evaluate("document.body.dataset.wsView") == "play")
            check("tabs live in the top bar", page.evaluate("!!document.querySelector('#topbar #rail')"))
            check("tagline is Generate · Play · Train",
                  page.get_attribute("#brand-tagline", "aria-label") == "Generate · Play · Train"
                  and " ".join(page.text_content("#brand-tagline").split()) == "Generate Play Train")
            check("tools sub-nav hidden on Play", page.is_hidden("#tools-nav"))
            check("no Scene Map sub-button (top tab now)", page.is_hidden("#ws-scenemap"))

            page.click('#rail .tab-btn[data-tab="tools"]')
            check("Tools opens Home first", page.is_visible("#tab-tools-home") and page.is_hidden("#tab-device"))
            check("view switch hidden outside the Player", page.is_hidden("#ws-view"))
            check("Player panel hidden on Tools", page.is_hidden("#tab-playback"))
            check("sub-nav visible in Tools", page.is_visible("#tools-nav"))
            check("rail Tools marked active",
                  page.get_attribute('#rail .tab-btn[data-tab="tools"]', "aria-selected") == "true")
            sub = page.evaluate("[...document.querySelectorAll('#tools-nav .tab-btn')].filter(b => b.offsetParent).map(b => b.textContent.trim())")
            check("Free sub-nav: Home · Device · Log · Settings", sub == ["Home", "Device", "Log", "Settings"], str(sub))

            page.click('#tools-nav .tab-btn[data-tab="settings"]')
            check("Settings panel shown", page.is_visible("#tab-settings") and page.is_hidden("#tab-device"))
            page.click('#mode-pill'); page.click('#mode-deck [data-tab="playback"]')
            check("back on Play hides sub-nav", page.is_hidden("#tools-nav") and page.is_visible("#tab-playback"))
            page.click('#rail .tab-btn[data-tab="tools"]')
            check("Tools remembers the last section (Settings)", page.is_visible("#tab-settings"))

            # Old entry points keep working.
            page.click('#mode-pill'); page.click('#mode-deck [data-tab="playback"]')
            page.evaluate("""document.querySelector('.tab-btn[data-tab="log"]').click()""")
            check(".tab-btn[data-tab=log].click() opens Log inside Tools",
                  page.is_visible("#tab-log") and page.is_visible("#tools-nav"))
            page.click('#mode-pill'); page.click('#mode-deck [data-tab="playback"]')
            page.click("#topbar-device")
            check("top-bar device button opens Tools → Device", page.is_visible("#tab-device"))

            # Generator mode: its own top tab, the Generator panel.
            page.click('#mode-pill'); page.click('#mode-deck [data-tab="generator"]')
            check("Generator shows the Generator panel", page.is_visible("#tab-generator") and page.is_visible("#gen-steps"))
            check("Player panel hidden in Generator (no grid override)", page.is_hidden("#tab-playback"))
            check("Generator mode current",
                  page.get_attribute('#mode-deck .tab-btn[data-tab="generator"]', "aria-selected") == "true"
                  and page.text_content("#mode-pill-name") == "Generator")
            check("mode hook: generator", page.evaluate("document.body.dataset.mode") == "generator")
            check("Play · Edit toggle hidden in Generator", page.is_hidden("#ws-view"))
            check("old route switchTab('generator') = Create view", page.evaluate("document.body.dataset.wsView") == "create")
            # Scene Map mode (Owner: A): the Player's scene view.
            page.click('#mode-pill'); page.click('#mode-deck [data-tab="scenemap"]')
            check("Scene Map shows the Player panel", page.is_visible("#tab-playback") and page.is_hidden("#tab-generator"))
            check("Scene Map opens the scene grid (shown once a script is loaded)",
                  page.get_attribute("#tab-playback", "data-pb-bottom") == "scenemap", str(page.get_attribute("#tab-playback", "data-pb-bottom")))
            check("Scene Map mode current", page.get_attribute('#mode-deck .tab-btn[data-tab="scenemap"]', "aria-selected") == "true"
                  and page.text_content("#mode-pill-name") == "Scene Map")
            check("mode hook: scenemap", page.evaluate("document.body.dataset.mode") == "scenemap")
            page.evaluate("document.querySelector('#pb-scenes-heatmap').click()")
            check("rhythm heatmap link opens the Generator's scene map",
                  page.is_visible("#tab-generator") and page.get_attribute("#tab-generator", "data-view") == "scenemap"
                  and page.is_visible("#gen-scene-map") and page.is_hidden("#gen-steps"))
            check("rhythm heatmap counts as Generator", page.evaluate("document.body.dataset.mode") == "generator")
            page.click('#mode-pill'); page.click('#mode-deck [data-tab="generator"]')
            check("Generator leaves the rhythm view",
                  page.get_attribute("#tab-generator", "data-view") is None and page.is_visible("#gen-steps"))
            page.click('#mode-pill'); page.click('#mode-deck [data-mode="player"]')
            check("Player leaves the scene grid", page.get_attribute("#tab-playback", "data-pb-bottom") != "scenemap")
            # Edit: a toggle inside the Player, remembered.
            page.click('#mode-pill'); page.click('#mode-deck [data-tab="playback"]')
            page.click('#mode-pill'); page.click('#mode-deck [data-mode="editor"]')
            check("mode hook: editor", page.evaluate("document.body.dataset.mode") == "editor")
            check("Edit shows the Player panel", page.is_visible("#tab-playback")
                  and page.get_attribute("#tab-playback", "data-ws-view") == "edit")
            check("Edit remembered", page.evaluate("localStorage.getItem('ws.view')") == "edit")
            check("pill shows Editor", page.text_content("#mode-pill-name") == "Editor")
            page.click('#mode-pill'); page.click('#mode-deck [data-tab="training"]')
            page.click('#mode-pill'); page.click('#mode-deck [data-tab="playback"]')
            check("Player card opens the Play view", page.evaluate("document.body.dataset.wsView") == "play"
                  and page.text_content("#mode-pill-name") == "Player")
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
            check("suite sections stay hidden in the Free build",
                  page.is_hidden('#tools-nav .tab-btn[data-tab="benchmark"]'))
            b.close()
    finally:
        shutdown()
        harness.unlink(missing_ok=True)
    return check.report()


if __name__ == "__main__":
    sys.exit(main())
