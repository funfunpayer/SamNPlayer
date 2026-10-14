"""Regressionstest für Drag & Drop.

Geprüft wird die Weiche, nicht die Webview: Go liefert die echten Dateipfade
(die Webview bekäme aus einem Drop nur einen Blob ohne Pfad, mit dem der
Generator nichts anfangen kann), und das Frontend entscheidet daraus, welcher
Tab zuständig ist.

Der eigentliche Nutzen ist genau diese Weiche (Video first, Owner 14 Oct):
ein Video sucht im Player sein Skript; ohne Skript bietet der leere Player
"Create script" an, das den Generator mit demselben Video öffnet. Im
Generator und bei mehreren Videos (Create-Queue) landet ein Video direkt im
Generator, ein Skript in der Wiedergabe.

Ausführen:  python3 cmd/gui-wails/frontend/test/drop_test.py
"""

import sys

from playwright.sync_api import sync_playwright

from _harness import Checker, app_stub, serve

PAGE = """<!doctype html><html><body>
<div id="app">
  <nav id="tabbar">
    <button class="tab-btn active" data-tab="playback">Playback</button>
    <button class="tab-btn" data-tab="training">Training</button>
    <button class="tab-btn" data-tab="generator">Generate</button>
    <button class="tab-btn" data-tab="device">Device</button>
    <button class="tab-btn" data-tab="log">Log</button>
    <button class="tab-btn" data-tab="settings">Settings</button>
    <button class="tab-btn" data-tab="roi-training">AI Train</button>
    <span id="version-label"></span>
  </nav>
  <section id="tab-playback" class="tab-panel active"></section>
  <section id="tab-training" class="tab-panel"></section>
  <section id="tab-generator" class="tab-panel"></section>
  <section id="tab-benchmark" class="tab-panel"></section>
  <section id="tab-device" class="tab-panel"></section>
  <section id="tab-log" class="tab-panel"></section>
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
        "FindScriptForVideo": "async v => (window.__calls.push(['find', v]), v === '/v/has.mp4'"
                              " ? { videoPath: v, scriptPath: '/v/has.samn', found: true }"
                              " : { videoPath: v, scriptPath: '', found: false })",
        "LoadFrameAt": "async () => ({ width: 2, height: 2, pngBase64: '' })",
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

        # The open tab (body[data-tab]); in the Player workspace the Player
        # button stays highlighted while Create (= generator) is shown.
        active = lambda: page.evaluate("document.body.dataset.tab || document.querySelector('.tab-btn.active').dataset.tab")
        check("Start im Wiedergabe-Tab", active() == "playback", str(active()))

        # --- Video ohne Skript: der Player fragt ---------------------------
        page.evaluate("""window.__triggerEvent('files:dropped',
            { videos: ['/videos/clip.mp4'], scripts: [], ignored: 0 })""")
        page.wait_for_function("!document.querySelector('#pb-noscript').hidden", timeout=5000)
        check("Video ohne Skript bleibt im Player", active() == "playback", str(active()))
        check("Skript wird neben dem Video gesucht",
              page.evaluate("window.__calls.some(c => c[0] === 'find' && c[1] === '/videos/clip.mp4')"))
        check("leerer Player nennt das Video",
              page.text_content("#pb-noscript-name") == "clip.mp4", page.text_content("#pb-noscript-name"))
        check("noch nichts an den Generator geschickt", page.evaluate("window.__dropped.length") == 0)
        page.click("#pb-noscript-create")
        page.wait_for_function("window.__dropped.length > 0")
        check("Create script öffnet den Generator", active() == "generator", str(active()))
        dropped = page.evaluate("window.__dropped[0]")
        check("Videopfad kommt unverändert an",
              dropped[0] == "video"
              and dropped[1]["path"] == "/videos/clip.mp4"
              and dropped[1].get("extraCount") == 0
              and dropped[1].get("paths") == ["/videos/clip.mp4"],
              str(dropped))
        check("Frage ist danach weg", page.evaluate("document.querySelector('#pb-noscript').hidden"))

        # --- Im Generator: ein Video bleibt dort --------------------------
        page.evaluate("""window.__triggerEvent('files:dropped',
            { videos: ['/videos/other.mp4'], scripts: [], ignored: 0 })""")
        page.wait_for_function("window.__dropped.length > 1")
        check("im Generator geht ein Video direkt in den Generator",
              active() == "generator" and page.evaluate("window.__dropped[1][1].path") == "/videos/other.mp4",
              str(page.evaluate("window.__dropped")))

        # --- Skript: muss in die Wiedergabe führen ------------------------
        page.evaluate("""window.__triggerEvent('files:dropped',
            { videos: [], scripts: ['/skripte/a.funscript'], ignored: 0 })""")
        page.wait_for_function("window.__dropped.length > 2")
        check("Skript wechselt in den Wiedergabe-Tab", active() == "playback", str(active()))
        dropped = page.evaluate("window.__dropped[2]")
        check("Skriptpfad kommt unverändert an",
              dropped[0] == "script"
              and dropped[1]["path"] == "/skripte/a.funscript"
              and dropped[1].get("paths") == ["/skripte/a.funscript"],
              str(dropped))

        # --- Video mit Skript daneben: das Skript wird geöffnet -------------
        page.evaluate("window.__calls.length = 0")
        page.evaluate("""window.__triggerEvent('files:dropped',
            { videos: ['/v/has.mp4'], scripts: [], ignored: 0 })""")
        page.wait_for_function("window.__calls.some(c => c[0] === 'find' && c[1] === '/v/has.mp4')")
        page.wait_for_timeout(300)
        check("Video mit Skript: keine Frage, kein Generator",
              page.evaluate("document.querySelector('#pb-noscript').hidden")
              and page.evaluate("window.__dropped.length") == 3 and active() == "playback",
              str(page.evaluate("window.__dropped")))

        # --- Beides zugleich: das Skript mit diesem Video ------------------
        page.evaluate("""window.__triggerEvent('files:dropped',
            { videos: ['/v/b.mp4'], scripts: ['/s/b.funscript'], ignored: 0 })""")
        page.wait_for_timeout(300)
        check("Video und Skript zusammen: kein Generator, keine Frage",
              page.evaluate("window.__dropped.length") == 3
              and page.evaluate("document.querySelector('#pb-noscript').hidden")
              and active() == "playback",
              str(page.evaluate("window.__dropped")))

        # --- Mehrere Videos: Queue (DeepFunGen UX), nicht still ignorieren ----
        page.evaluate("""window.__triggerEvent('files:dropped',
            { videos: ['/v/first.mp4', '/v/second.mp4', '/v/third.mp4'], scripts: [], ignored: 0 })""")
        page.wait_for_function("window.__dropped.length > 3")
        dropped = page.evaluate("window.__dropped[3]")
        check("mehrere Videos: nur der erste Pfad wird geladen",
              dropped[1]["path"] == "/v/first.mp4", str(dropped))
        check("mehrere Videos: komplette Liste wird mitgeschickt",
              dropped[1].get("paths") == ["/v/first.mp4", "/v/second.mp4", "/v/third.mp4"],
              str(dropped))
        check("mehrere Videos: Anzahl der übrigen wird mitgeschickt",
              dropped[1]["extraCount"] == 2, str(dropped))
        page.wait_for_function(
            "document.querySelector('#gen-queue') && !document.querySelector('#gen-queue').hidden",
            timeout=5000)
        check("mehrere Videos: Queue-UI sichtbar",
              page.locator("#gen-queue-list li").count() == 3,
              str(page.locator("#gen-queue-list li").count()))
        check("mehrere Videos: Queue-Hinweis in der Oberfläche",
              "queue" in page.locator("#gen-status").inner_text().lower()
              or "2" in page.locator("#gen-queue-summary").inner_text(),
              page.locator("#gen-status").inner_text() + " | " + page.locator("#gen-queue-summary").inner_text())

        # --- Mehrere Skripte: Film-Liste statt „ignoriert“ -----------------
        page.evaluate("window.__dropped.length = 0")
        page.evaluate("""window.__triggerEvent('files:dropped',
            { videos: [], scripts: ['/s/first.funscript', '/s/second.funscript'], ignored: 0 })""")
        page.wait_for_function("window.__dropped.length > 0")
        dropped = page.evaluate("window.__dropped[0]")
        check("mehrere Skripte: erster Pfad kommt an",
              dropped[1]["path"] == "/s/first.funscript", str(dropped))
        check("mehrere Skripte: komplette Liste wird mitgeschickt",
              dropped[1].get("paths") == ["/s/first.funscript", "/s/second.funscript"], str(dropped))
        page.wait_for_function(
            "document.querySelector('#pb-playlist') && !document.querySelector('#pb-playlist').hidden",
            timeout=5000)
        check("mehrere Skripte: Film-Liste sichtbar",
              page.locator("#pb-playlist-list li").count() == 2)

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
