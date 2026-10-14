"""Generator video (#533 step 2): the clip to create from plays in a real
<video> with transport and seeking. Pausing loads exactly that frame into
the tip canvas over the video; playing hides the canvas. The clip has its
own URL (SetGeneratorVideo) — the Player's video is not touched.

Run: python3 cmd/gui-wails/frontend/test/gen_video_test.py
"""

import pathlib
import subprocess

from playwright.sync_api import sync_playwright

from _harness import Checker, app_stub, serve

HERE = pathlib.Path(__file__).resolve().parent
PNG = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg=="
CLIP = HERE / "_gen_video_clip.webm"


def make_clip():
    subprocess.run([
        "ffmpeg", "-f", "lavfi", "-i", "testsrc=duration=8:size=320x180:rate=10",
        "-c:v", "libvpx-vp9", "-crf", "35", "-b:v", "0", "-y", str(CLIP),
    ], check=True, capture_output=True)


def main():
    check = Checker()
    make_clip()
    links = "".join(f'<link rel="stylesheet" href="/src/{css}?real" />'
                    for css in ("style.css", "theme_flow.css", "flow_fx.css", "training_rings.css"))
    html = (HERE.parent / "index.html").read_text().replace('<link rel="stylesheet" href="/src/style.css" />', links)
    harness = HERE / "_gen_video_harness.html"
    harness.write_text(html)
    base, shutdown = serve("window.__frames = []; window.__genVideo = []; window.__playerVideo = 0;\n" + app_stub({
        "GetEULA": "async () => ({version:'x', text:'', accepted:true})",
        "IsSuiteBuild": "async () => false",
        "CurrentVersion": "async () => 'test'",
        "GetSettings": "async () => ({updateCheckOnStartup:false})",
        "PickVideoFile": "async () => '/v/clip.webm'",
        "SetGeneratorVideo": "async (p) => { window.__genVideo.push(p); return '/test/_gen_video_clip.webm'; }",
        "SetPlaybackVideo": "async () => { window.__playerVideo++; return ''; }",
        "LoadFirstFrame": f"async () => {{ window.__frames.push(0); return ({{width: 320, height: 180, pngBase64: '{PNG}'}}); }}",
        "LoadFrameAt": f"async (p, t) => {{ window.__frames.push(t); return ({{width: 320, height: 180, pngBase64: '{PNG}'}}); }}",
        "ScriptExistsForVideo": "async () => false",
        "ProbeCreateVideo": f"async (p) => ({{path: p, name: 'clip.webm', width: 320, height: 180, fps: 10, vfr: false, durationSec: 8, codec: 'vp9', posterUrl: 'data:image/png;base64,{PNG}'}})",
    }))
    try:
        with sync_playwright() as p:
            b = p.chromium.launch()
            page = b.new_page(viewport={"width": 1600, "height": 960})
            errors = []
            page.on("pageerror", lambda e: errors.append(str(e)))
            page.add_init_script("try{localStorage.clear()}catch(e){}")
            page.goto(f"{base}/test/_gen_video_harness.html")
            page.wait_for_timeout(800)
            page.click('#mode-pill'); page.click('#mode-deck [data-tab="generator"]')
            check("no video yet: video and transport hidden",
                  page.is_hidden(".gl-video") and page.is_hidden(".gl-vt"))
            page.click("#gen-choose")
            page.wait_for_function("document.querySelector('#gen-video').readyState >= 1", timeout=8000)
            check("clip gets its own URL", page.evaluate("window.__genVideo") == ["/v/clip.webm"])
            check("Player video untouched", page.evaluate("window.__playerVideo") == 0)
            check("transport enabled", not page.is_disabled("#gen-video-play") and not page.is_disabled("#gen-video-seek"))
            check("duration shown", "0:08" in page.text_content("#gen-video-time"))
            check("paused: tip canvas over the video", page.evaluate(
                "getComputedStyle(document.querySelector('#roi-canvas-wrap')).position") == "absolute"
                and page.is_visible("#roi-canvas"))

            page.click("#gen-video-play")
            page.wait_for_function("!document.querySelector('#gen-video').paused && document.querySelector('#gen-video').currentTime > 0.3", timeout=8000)
            check("playing: canvas hidden", page.evaluate(
                "getComputedStyle(document.querySelector('#roi-canvas-wrap')).visibility") == "hidden")
            check("play button turns into pause", page.get_attribute("#gen-video-play", "aria-label") == "Pause")

            page.click("#gen-video-play")
            page.wait_for_function("document.querySelector('#gen-video').paused", timeout=4000)
            t = page.evaluate("Math.round(document.querySelector('#gen-video').currentTime * 10) / 10")
            page.wait_for_function(f"window.__frames.some(f => Math.abs(f - {t}) < 0.05)", timeout=4000)
            check("pause loads exactly that frame into the tip canvas", True)
            check("Time field follows the video", abs(float(page.input_value("#gen-seek")) - t) < 0.05)
            check("paused again: canvas visible", page.evaluate(
                "getComputedStyle(document.querySelector('#roi-canvas-wrap')).visibility") == "visible")

            page.click("#gen-video-fwd")
            page.wait_for_function(f"Math.abs(document.querySelector('#gen-video').currentTime - {min(8, t + 5)}) < 0.3", timeout=4000)
            page.wait_for_function(f"window.__frames.some(f => Math.abs(f - {min(8, t + 5)}) < 0.35)", timeout=4000)
            check("+5s seeks and loads that frame", True)
            page.eval_on_selector("#gen-video-seek", "e => { e.value = '2'; e.dispatchEvent(new Event('input')); }")
            page.wait_for_function("Math.abs(document.querySelector('#gen-video').currentTime - 2) < 0.2", timeout=4000)
            page.wait_for_function("window.__frames.some(f => Math.abs(f - 2) < 0.15)", timeout=4000)
            check("seek bar scrubs and loads the frame", True)
            page.fill("#gen-seek", "5")
            page.click("#gen-seek-btn")
            page.wait_for_function("Math.abs(document.querySelector('#gen-video').currentTime - 5) < 0.2", timeout=4000)
            check("Time / Frame control moves the video", True)
            check("no page errors", not errors, "; ".join(errors))
            b.close()
    finally:
        shutdown()
        harness.unlink(missing_ok=True)
        CLIP.unlink(missing_ok=True)
    return check.report()


if __name__ == "__main__":
    raise SystemExit(main())
