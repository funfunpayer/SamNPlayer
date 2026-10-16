"""Video first (Owner 14 Oct): Open… / Ctrl+O take a video or a script.

A video with a script beside it opens that script; without one the empty
Player shows the video with "Create script" (Generator, same video),
"Choose script…" (that script with this video) and "Always create right
away". Over a loaded script the question offers "Back to the script".

Ausführen:  python3 cmd/gui-wails/frontend/test/open_media_test.py
"""

import pathlib
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
  window.__pick = '';
  window.__pickScript = '';
  window.addEventListener('drop:video', e => window.__dropped.push(e.detail.path));
  await import('/src/main.js');
  window.__ready = true;
</script></body></html>"""

CURVE = "[{atMs:0,pos:10},{atMs:500,pos:90},{atMs:1000,pos:10},{atMs:1500,pos:90}]"


def main():
    check = Checker()
    base, shutdown = serve(app_stub({
        "CurrentVersion": "async () => 'test'",
        "CheckForUpdate": "async () => ({ available: false })",
        "GetSettings": "async () => ({})",
        "GetDeviceStatus": "async () => ({ connected: false })",
        "GetCacheInfo": "async () => ({ files: 0, humanSize: '0 B', path: '/tmp' })",
        "PickMediaFile": "async () => window.__pick",
        "PickFunscriptFile": "async () => window.__pickScript",
        "FindScriptForVideo": "async v => (window.__calls.push(['find', v]), v === '/v/has.mp4'"
                              " ? { videoPath: v, scriptPath: '/v/has.samn', found: true, variants: ["
                              "{ path: '/v/has.samn', label: '', beside: true },"
                              "{ path: '/lib/has (Soft).funscript', label: 'soft', beside: false }] }"
                              " : v === '/v/folder.mp4' && window.__folderAdded"
                              " ? { videoPath: v, scriptPath: '/lib/folder.funscript', found: true,"
                              " variants: [{ path: '/lib/folder.funscript', label: '', beside: false }] }"
                              " : { videoPath: v, scriptPath: '', found: false, variants: [] })",
        "CreateBeatBarScript": "async v => (window.__calls.push(['beatbar', v]), v === '/v/hero.mp4'"
                               " ? { found: true, scriptPath: '/v/hero (Beat bar).funscript', beats: 120, bpm: 110,"
                               " message: 'Beat bar read: 120 beats, about 110 BPM.' }"
                               " : { found: false, scriptPath: '', message: 'No beat bar found in this video.' })",
        "AddScriptFolder": "async () => (window.__folderAdded = true, ['/lib'])",
        "LoadFrameAt": "async () => ({ width: 2, height: 2, pngBase64: 'iVBORw0KGgo=' })",
        "LoadFunscript": "async p => (window.__calls.push(['load', p]), { path: p, actionCount: 4, durationMs: 1500, "
                         "videoPath: '', hasVideo: false })",
        "SetPlaybackVideo": "async v => (window.__calls.push(['video', v]), '')",
        "GetScriptCurve": f"async () => {CURVE}",
        "GetHeatmap": "async n => Array.from({ length: n }, (_, i) => ({ atMs: i * 10, intensity: 0.5 }))",
        "GetMarker": "async () => null",
        "VideoFileURL": "async () => ''",
    }))
    harness = pathlib.Path(__file__).resolve().parent / "_open_media_harness.html"
    harness.write_text(PAGE)
    errors = []
    try:
        with sync_playwright() as p:
            b = p.chromium.launch()
            page = b.new_page()
            page.on("pageerror", lambda e: errors.append(str(e)))
            page.goto(f"{base}/test/_open_media_harness.html")
            page.wait_for_function("window.__ready === true")
            calls = lambda kind: page.evaluate(f"window.__calls.filter(c => c[0] === '{kind}').map(c => c[1])")
            mode = lambda: page.evaluate("document.body.dataset.mode")

            check("empty Player: Open… in the middle", page.text_content("#pb-choose-empty").strip() == "Open…")
            check("head button opens too", page.text_content("#pb-choose").strip() == "Open…")

            # Open… → a video without a script → the question.
            page.evaluate("window.__pick = '/v/new clip.mp4'")
            page.click("#pb-choose-empty")
            page.wait_for_function("!document.querySelector('#pb-noscript').hidden")
            check("video without script: name shown", page.text_content("#pb-noscript-name") == "new clip.mp4")
            page.wait_for_function("!document.querySelector('#pb-noscript-img').hidden")
            check("video without script: a still of it", True)
            check("no Back without a loaded script", page.evaluate("document.querySelector('#pb-noscript-back').hidden"))

            # Choose script… → that script, with this video.
            page.evaluate("window.__pickScript = '/s/mine.funscript'")
            page.click("#pb-noscript-choose")
            page.wait_for_function("document.querySelector('#tab-playback').dataset.pbState === 'loaded'")
            page.wait_for_function("window.__calls.some(c => c[0] === 'video')")
            check("Choose script…: script loaded", "/s/mine.funscript" in calls("load"), str(calls("load")))
            check("Choose script…: the video is linked", calls("video")[-1] == "/v/new clip.mp4", str(calls("video")))
            check("question gone", page.evaluate("document.querySelector('#pb-noscript').hidden"))
            check("Player visible again", page.evaluate("!document.querySelector('#pb-loaded').hidden"))

            # Ctrl+O → a video with a script beside it → that script.
            page.evaluate("window.__pick = '/v/has.mp4'")
            page.keyboard.press("Control+o")
            page.wait_for_function("window.__calls.some(c => c[0] === 'load' && c[1] === '/v/has.samn')")
            check("Ctrl+O: video finds its script", True)
            check("Ctrl+O: no question", page.evaluate("document.querySelector('#pb-noscript').hidden"))

            # Several scripts for this video: the variant choice appears and switches.
            page.wait_for_function("!document.querySelector('#pb-variant').hidden", timeout=4000)
            check("variants: two choices", page.evaluate("document.querySelectorAll('#pb-variant option').length") == 2)
            check("variants: the loaded one is selected",
                  page.evaluate("document.querySelector('#pb-variant').selectedOptions[0].textContent").startswith("Main script"))
            check("variants: folder ones say so",
                  "script folder" in page.evaluate("document.querySelectorAll('#pb-variant option')[1].textContent"))
            page.select_option("#pb-variant", "1")
            page.wait_for_function("window.__calls.some(c => c[0] === 'load' && c[1] === '/lib/has (Soft).funscript')")
            check("variants: choosing one loads it", True)
            page.wait_for_timeout(300)
            check("variants: the video stays", calls("video")[-1] == "/v/has.mp4", str(calls("video")[-3:]))

            # No script anywhere yet: "Add a script folder…" searches it right away.
            page.evaluate("window.__pick = '/v/folder.mp4'")
            page.click("#pb-choose")
            page.wait_for_function("!document.querySelector('#pb-noscript').hidden")
            page.click("#pb-noscript-folder")
            page.wait_for_function("window.__calls.some(c => c[0] === 'load' && c[1] === '/lib/folder.funscript')")
            check("folder button: the script found there opens", page.evaluate("document.querySelector('#pb-noscript').hidden"))

            # Cock Hero / Fap Hero: "From the beat bar" writes a script and opens it.
            page.evaluate("window.__pick = '/v/plain.mp4'")
            page.click("#pb-choose")
            page.wait_for_function("!document.querySelector('#pb-noscript').hidden")
            page.click("#pb-noscript-beatbar")
            page.wait_for_function("document.querySelector('#pb-noscript-status').textContent.includes('No beat bar')")
            check("beat bar: none found is said, the question stays",
                  page.evaluate("!document.querySelector('#pb-noscript').hidden"))
            check("beat bar: button usable again", page.evaluate("!document.querySelector('#pb-noscript-beatbar').disabled"))
            page.evaluate("window.__pick = '/v/hero.mp4'")
            page.click("#pb-noscript-other")
            page.wait_for_function("document.querySelector('#pb-noscript-name').textContent === 'hero.mp4'")
            page.click("#pb-noscript-beatbar")
            page.wait_for_function("window.__calls.some(c => c[0] === 'load' && c[1] === '/v/hero (Beat bar).funscript')")
            check("beat bar: asked for this video", calls("beatbar")[-1] == "/v/hero.mp4", str(calls("beatbar")))
            check("beat bar: the new script opens", page.evaluate("document.querySelector('#pb-noscript').hidden"))

            # Over a loaded script: question with Back, the script stays.
            page.evaluate("window.__pick = '/v/other.mp4'")
            page.click("#pb-choose")
            page.wait_for_function("!document.querySelector('#pb-noscript').hidden")
            check("over a loaded script: Back offered", page.evaluate("!document.querySelector('#pb-noscript-back').hidden"))
            check("over a loaded script: Player hidden meanwhile", page.evaluate("document.querySelector('#pb-loaded').hidden"))
            page.click("#pb-noscript-back")
            check("Back: the script is shown again",
                  page.evaluate("!document.querySelector('#pb-loaded').hidden && document.querySelector('#pb-empty').hidden"))

            # Editor stays Editor while asking.
            page.evaluate("window.dispatchEvent(new CustomEvent('ws:set', { detail: { view: 'edit' } }))")
            page.evaluate("window.__pick = '/v/edit.mp4'")
            page.click("#pb-choose")
            page.wait_for_function("!document.querySelector('#pb-noscript').hidden")
            check("Editor: asks in the Editor", mode() == "editor", mode())

            # Create script → Generator with the same video.
            page.click("#pb-noscript-create")
            page.wait_for_function("window.__dropped.length > 0")
            check("Create script: Generator", mode() == "generator", mode())
            check("Create script: same video", page.evaluate("window.__dropped[0]") == "/v/edit.mp4")

            # In the Generator, Open… with a video goes straight into it.
            page.evaluate("window.__pick = '/v/gen.mp4'")
            page.click("#topbar-open") if page.locator("#topbar-open").count() else page.keyboard.press("Control+o")
            page.wait_for_function("window.__dropped.length > 1")
            check("Generator: Open… keeps the Generator", mode() == "generator" and page.evaluate("window.__dropped[1]") == "/v/gen.mp4")

            # Always create right away: no question, Create starts by itself.
            page.evaluate("localStorage.setItem('open.autoCreate', '1')")
            page.evaluate("window.dispatchEvent(new CustomEvent('ws:set', { detail: { view: 'play' } }))")
            page.evaluate("""(() => { const b = document.querySelector('#gen-generate');
              window.__created = 0; b.addEventListener('click', () => window.__created++);
              b.disabled = true; })()""")
            page.evaluate("window.__pick = '/v/auto.mp4'")
            page.click("#pb-choose")
            page.wait_for_function("window.__dropped.length > 2")
            check("auto: Generator with the video", mode() == "generator" and page.evaluate("window.__dropped[2]") == "/v/auto.mp4")
            check("auto: no question", page.evaluate("document.querySelector('#pb-noscript').hidden"))
            page.evaluate("document.querySelector('#gen-generate').disabled = false")
            page.wait_for_function("window.__created === 1", timeout=4000)
            check("auto: Create starts once the Generator is ready", True)

            check("no page errors", not errors, "; ".join(errors[:5]))
            b.close()
    finally:
        shutdown()
        harness.unlink(missing_ok=True)
    return check.report()


if __name__ == "__main__":
    sys.exit(main())
