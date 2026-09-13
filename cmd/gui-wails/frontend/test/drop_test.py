"""Regressionstest für Drag & Drop.

Geprüft wird die Weiche, nicht die Webview: Go liefert die echten Dateipfade
(die Webview bekäme aus einem Drop nur einen Blob ohne Pfad, mit dem der
Generator nichts anfangen kann), und das Frontend entscheidet daraus, welcher
Tab zuständig ist.

Der eigentliche Nutzen ist genau diese Weiche - ein Video soll im Generator
landen, ein Skript in der Wiedergabe, ohne dass man vorher den richtigen Tab
sucht. Geht sie kaputt, landet die Datei still im falschen Tab.

Ausführen:  python3 cmd/gui-wails/frontend/test/drop_test.py
"""

import sys

from playwright.sync_api import sync_playwright

from _harness import Checker, app_stub, serve

PAGE = """<!doctype html><html><body>
<div id="app">
  <nav id="tabbar">
    <button class="tab-btn active" data-tab="playback">Wiedergabe</button>
    <button class="tab-btn" data-tab="training">Training</button>
    <button class="tab-btn" data-tab="generator">Skript erzeugen</button>
    <button class="tab-btn" data-tab="device">Gerät</button>
    <button class="tab-btn" data-tab="settings">Einstellungen</button>
    <span id="version-label"></span>
  </nav>
  <section id="tab-playback" class="tab-panel active"></section>
  <section id="tab-training" class="tab-panel"></section>
  <section id="tab-generator" class="tab-panel"></section>
  <section id="tab-device" class="tab-panel"></section>
  <section id="tab-settings" class="tab-panel"></section>
</div>
<script type="module">
  window.__dropped = [];
  window.addEventListener('drop:video', e => window.__dropped.push(['video', e.detail]));
  window.addEventListener('drop:script', e => window.__dropped.push(['script', e.detail]));
  await import('/src/main.js');
  window.__ready = true;
</script></body></html>"""


def main():
    check = Checker()
    base, shutdown = serve(app_stub({
        "CurrentVersion": "async () => 'test'",
        "CheckForUpdate": "async () => ({ available: false })",
        "GetSettings": "async () => ({})",
        "GetDeviceStatus": "async () => ({ connected: false })",
        "GetCacheInfo": "async () => ({ files: 0, humanSize: '0 B', path: '/tmp' })",
        "LoadFunscript": "async () => ({ path: '', actionCount: 0, durationMs: 0 })",
        "GetHeatmap": "async () => []",
        "GetScriptCurve": "async () => []",
        "GetMarker": "async () => null",
    }))
    harness = None

    with sync_playwright() as p:
        browser = p.chromium.launch()
        page = browser.new_page()
        page.on("dialog", lambda d: d.dismiss())
        page.on("pageerror", lambda e: print("   [pageerror]", e))

        import pathlib
        harness = pathlib.Path(__file__).resolve().parent / "_drop_harness.html"
        harness.write_text(PAGE)
        page.goto(f"{base}/test/_drop_harness.html")
        page.wait_for_function("window.__ready === true")

        active = lambda: page.locator(".tab-btn.active").get_attribute("data-tab")
        check("Start im Wiedergabe-Tab", active() == "playback", str(active()))

        # --- Video: muss in den Generator führen --------------------------
        page.evaluate("""window.__triggerEvent('files:dropped',
            { videos: ['/videos/clip.mp4'], scripts: [], ignored: 0 })""")
        page.wait_for_function("window.__dropped.length > 0")
        check("Video wechselt in den Generator-Tab", active() == "generator", str(active()))
        dropped = page.evaluate("window.__dropped[0]")
        check("Videopfad kommt unverändert an",
              dropped == ["video", "/videos/clip.mp4"], str(dropped))

        # --- Skript: muss in die Wiedergabe führen ------------------------
        page.evaluate("""window.__triggerEvent('files:dropped',
            { videos: [], scripts: ['/skripte/a.funscript'], ignored: 0 })""")
        page.wait_for_function("window.__dropped.length > 1")
        check("Skript wechselt in den Wiedergabe-Tab", active() == "playback", str(active()))
        dropped = page.evaluate("window.__dropped[1]")
        check("Skriptpfad kommt unverändert an",
              dropped == ["script", "/skripte/a.funscript"], str(dropped))

        # --- Beides zugleich: Video hat Vorrang ---------------------------
        page.evaluate("""window.__triggerEvent('files:dropped',
            { videos: ['/v/b.mp4'], scripts: ['/s/b.funscript'], ignored: 0 })""")
        page.wait_for_function("window.__dropped.length > 2")
        check("bei Video und Skript gewinnt das Video",
              page.evaluate("window.__dropped[2][0]") == "video"
              and page.evaluate("window.__dropped.length") == 3,
              str(page.evaluate("window.__dropped")))

        # --- Ablagefläche erscheint und verschwindet ----------------------
        overlay_visible = "document.getElementById('drop-overlay').classList.contains('visible')"
        check("Ablagefläche zunächst unsichtbar", not page.evaluate(overlay_visible))
        page.evaluate("window.dispatchEvent(new Event('dragenter'))")
        check("Ablagefläche erscheint beim Ziehen", page.evaluate(overlay_visible))
        page.evaluate("window.dispatchEvent(new Event('dragleave'))")
        check("Ablagefläche verschwindet wieder", not page.evaluate(overlay_visible))

        browser.close()

    shutdown()
    if harness:
        harness.unlink(missing_ok=True)
    return check.report()


if __name__ == "__main__":
    sys.exit(main())
