"""Regressionstest: ScriptChapters() wird tatsächlich in der Wiedergabe
angezeigt.

Die Bindung (motionx.Chapter, ScriptChapters) existierte bereits, wurde
aber von keiner Stelle im Frontend aufgerufen - eine Funktion mit Backend
und Test, aber ohne angeschlossene Oberfläche. Dieser Test hält fest, dass
das Laden eines Skripts die Kapitelliste in #pb-analysis anzeigt (neben der
schon bestehenden AnalyzeScript-Zusammenfassung, nicht anstelle davon).

Ausführen:  python3 cmd/gui-wails/frontend/test/chapters_display_test.py
"""

import pathlib
import sys

from playwright.sync_api import sync_playwright

from _harness import Checker, app_stub, serve

PAGE = """<!doctype html><html><body><div id="root"></div>
<script type="module">
  import { initPlayback } from '/src/playback.js';
  window.__pb = initPlayback(document.querySelector('#root'));
  window.__ready = true;
</script></body></html>"""


def main():
    check = Checker()
    base, shutdown = serve(app_stub({
        "PickFunscriptFile": "async () => '/tmp/test.funscript'",
        "LoadFunscript": "async () => ({ path: '/tmp/test.funscript', actionCount: 5, "
                         "durationMs: 10000, videoPath: '', hasVideo: false })",
        "GetScriptCurve": "async () => [{ atMs: 0, pos: 0 }, { atMs: 10000, pos: 100 }]",
        "GetHeatmap": "async n => Array.from({ length: n }, (_, i) => ({ atMs: i * 100, intensity: 0.5 }))",
        "GetMarker": "async () => null",
        "VideoFileURL": "async () => ''",
        "GetSpeedHighlights": "async () => []",
        "ExportScriptHeatmapPNG": "async () => '/tmp/h.png'",
        "SavePlaybackProject": "async () => '/tmp/p.snp.json'",
        "EditCapSpeedRange": "async () => {}",
        "EditDeleteRange": "async () => {}",
        "SnapTimeMs": "async (t, fps) => t",
        "GetScriptOffset": "async () => 0",
        "GetOMarkers": "async () => []",
        "SaveOMarkers": "async () => {}",
        "AnalyzeScript": "async () => ({ summary: '10 Sekunden: gleichmäßig 80%' })",
        "ScriptChapters": "async () => [{ kind: 'pause', startMs: 0, endMs: 1000 }, "
                          "{ kind: 'crescendo', startMs: 1000, endMs: 10000 }]",
    }))

    harness = pathlib.Path(__file__).resolve().parent / "_chapters_harness.html"
    harness.write_text(PAGE)

    with sync_playwright() as p:
        browser = p.chromium.launch()
        page = browser.new_page()
        page.on("pageerror", lambda e: print("   [pageerror]", e))
        page.goto(f"{base}/test/_chapters_harness.html")
        page.wait_for_function("window.__ready === true")

        page.click("#pb-choose")
        page.wait_for_function(
            "document.querySelector('#pb-analysis').textContent.includes('Kapitel')", timeout=5000)
        text = page.locator("#pb-analysis").inner_text()
        check("Analyse-Zusammenfassung bleibt erhalten", "gleichmäßig" in text, text)
        check("Kapitelliste wird angehängt", "Kapitel:" in text, text)
        check("Kapitelnamen werden übersetzt (Pause/Steigerung)",
              "Pause" in text and "Steigerung" in text, text)
        check("Zeiten werden als m:ss formatiert", "0:00" in text and "0:10" in text, text)

        browser.close()

    shutdown()
    harness.unlink(missing_ok=True)
    return check.report()


if __name__ == "__main__":
    sys.exit(main())
