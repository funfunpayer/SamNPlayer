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
    <button class="tab-btn" data-tab="roi-training">KI-Train.</button>
    <span id="version-label"></span>
  </nav>
  <section id="tab-playback" class="tab-panel active"></section>
  <section id="tab-training" class="tab-panel"></section>
  <section id="tab-generator" class="tab-panel"></section>
  <section id="tab-benchmark" class="tab-panel"></section>
  <section id="tab-device" class="tab-panel"></section>
  <section id="tab-settings" class="tab-panel"></section>
  <section id="tab-roi-training" class="tab-panel"></section>
  <aside id="sidebar"></aside>
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
              dropped == ["video", {"path": "/videos/clip.mp4", "extraCount": 0}], str(dropped))

        # --- Skript: muss in die Wiedergabe führen ------------------------
        page.evaluate("""window.__triggerEvent('files:dropped',
            { videos: [], scripts: ['/skripte/a.funscript'], ignored: 0 })""")
        page.wait_for_function("window.__dropped.length > 1")
        check("Skript wechselt in den Wiedergabe-Tab", active() == "playback", str(active()))
        dropped = page.evaluate("window.__dropped[1]")
        check("Skriptpfad kommt unverändert an",
              dropped == ["script", {"path": "/skripte/a.funscript", "extraCount": 0}], str(dropped))

        # --- Beides zugleich: Video hat Vorrang ---------------------------
        page.evaluate("""window.__triggerEvent('files:dropped',
            { videos: ['/v/b.mp4'], scripts: ['/s/b.funscript'], ignored: 0 })""")
        page.wait_for_function("window.__dropped.length > 2")
        check("bei Video und Skript gewinnt das Video",
              page.evaluate("window.__dropped[2][0]") == "video"
              and page.evaluate("window.__dropped.length") == 3,
              str(page.evaluate("window.__dropped")))

        # --- Mehrere Videos: nur das erste wird geladen, aber sichtbar ----
        # (vorher gab es dafür ein 'drop:videos'-Event, auf das nichts
        # gehört hat - die übrigen Dateien verschwanden spurlos).
        page.evaluate("""window.__triggerEvent('files:dropped',
            { videos: ['/v/first.mp4', '/v/second.mp4', '/v/third.mp4'], scripts: [], ignored: 0 })""")
        page.wait_for_function("window.__dropped.length > 3")
        dropped = page.evaluate("window.__dropped[3]")
        check("mehrere Videos: nur der erste Pfad wird geladen",
              dropped[1]["path"] == "/v/first.mp4", str(dropped))
        check("mehrere Videos: Anzahl der übrigen wird mitgeschickt",
              dropped[1]["extraCount"] == 2, str(dropped))
        page.wait_for_function(
            "document.querySelector('#gen-status').textContent.includes('ignoriert')",
            timeout=5000)
        check("mehrere Videos: Hinweis auf die ignorierten Dateien in der Oberfläche sichtbar",
              "2" in page.locator("#gen-status").inner_text())

        # --- Mehrere Skripte: nur das erste wird geladen, aber sichtbar ---
        # (dieselbe Lücke wie bei Videos oben, hier für Skripte - vorher
        # verschwanden weitere abgelegte Skripte spurlos.)
        page.evaluate("""window.__triggerEvent('files:dropped',
            { videos: [], scripts: ['/s/first.funscript', '/s/second.funscript'], ignored: 0 })""")
        page.wait_for_function("window.__dropped.length > 4")
        dropped = page.evaluate("window.__dropped[4]")
        check("mehrere Skripte: nur der erste Pfad wird geladen",
              dropped[1]["path"] == "/s/first.funscript", str(dropped))
        check("mehrere Skripte: Anzahl der übrigen wird mitgeschickt",
              dropped[1]["extraCount"] == 1, str(dropped))
        page.wait_for_function(
            "document.querySelector('#pb-script-path').textContent.includes('ignoriert')",
            timeout=5000)
        check("mehrere Skripte: Hinweis auf die ignorierten Dateien in der Oberfläche sichtbar",
              "1" in page.locator("#pb-script-path").inner_text())

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
