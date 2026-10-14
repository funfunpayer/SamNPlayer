"""Training stage (Owner, 12/14 Oct): the stage switches between the
pixel clip, the live rings, live waves and a film; the film plays while the session runs and pauses when
it ends; Loop / Muted and the choice are remembered; recent videos are shared
with the Player; the Player video is never touched.

Run: python3 cmd/gui-wails/frontend/test/training_video_test.py
"""

import json
import pathlib
import subprocess
import sys

from playwright.sync_api import sync_playwright

from _harness import Checker, app_stub, serve

HERE = pathlib.Path(__file__).resolve().parent
CLIP = HERE / "_training_video_clip.webm"

PAGE = """<!doctype html><html><head><link rel="stylesheet" href="/src/style.css?real"></head>
<body><div id="root"></div>
<script type="module">
  import { initTraining } from '/src/training.js';
  initTraining(document.querySelector('#root'));
  window.__ready = true;
</script></body></html>"""


def make_clip():
    subprocess.run(["ffmpeg", "-f", "lavfi", "-i", "testsrc=duration=6:size=320x180:rate=10",
                    "-c:v", "libvpx-vp9", "-crf", "35", "-b:v", "0", "-y", str(CLIP)], check=True, capture_output=True)


def main():
    check = Checker()
    make_clip()
    base, shutdown = serve("window.__set = []; window.__cleared = 0;\n" + app_stub({
        "StartTraining": "async () => {}",
        "StopTraining": "async () => { window.__triggerEvent('training:done'); }",
        "StopTrainingCycle": "async () => {}",
        "TrainingHistory": "async () => []",
        "ListTrainingScripts": "async () => []",
        "PickVideoFile": "async () => '/films/My film.webm'",
        "SetTrainingVideo": "async (p) => { window.__set.push(p); return '/test/_training_video_clip.webm?v=' + window.__set.length; }",
        "ClearTrainingVideo": "async () => { window.__cleared++; }",
        "SetPlaybackVideo": "async () => { window.__playerTouched = true; return ''; }",
    }))
    harness = HERE / "_training_video_harness.html"
    harness.write_text(PAGE)
    try:
        with sync_playwright() as p:
            b = p.chromium.launch()
            ctx = b.new_context()
            page = ctx.new_page()
            errors = []
            page.on("pageerror", lambda e: errors.append(str(e)))
            page.add_init_script("""if (!sessionStorage.getItem('seeded')) { sessionStorage.setItem('seeded', '1');
                localStorage.clear(); localStorage.setItem('pb.recentVideos', JSON.stringify(['/films/Older film.mp4'])); }""")
            page.goto(f"{base}/test/_training_video_harness.html")
            page.wait_for_function("window.__ready === true")
            check("starts on the pixel clip", page.is_visible("#tr-pixel-stage") and not page.is_visible("#tr-video-view")
                  and page.get_attribute("#tr-stage-pixel", "aria-checked") == "true")
            check("video controls hidden on the pixel clip", not page.is_visible("#tr-video-choose"))

            page.click("#tr-stage-video")
            check("Video mode: film area shown, pixel clip hidden",
                  page.is_visible("#tr-video-view") and not page.is_visible("#tr-pixel-stage"))
            check("empty hint until a film is chosen", page.is_visible("#tr-video-empty"))
            check("Loop and Muted on by default", page.is_checked("#tr-video-loop") and page.is_checked("#tr-video-muted"))
            recent = page.evaluate("[...document.querySelectorAll('#tr-video-recent option')].map(o => o.textContent)")
            check("recent videos shared with the Player", recent == ["Recent videos", "Older film.mp4"], str(recent))

            page.click("#tr-video-choose")
            page.wait_for_function("window.__set.length === 1", timeout=3000)
            page.wait_for_function("document.querySelector('#tr-video').readyState >= 1", timeout=5000)
            check("chosen film loaded on its own URL", page.evaluate("window.__set") == ["/films/My film.webm"]
                  and "_training_video_clip.webm" in page.evaluate("document.querySelector('#tr-video').src"))
            check("film name shown, hint gone", page.text_content("#tr-video-name") == "My film.webm" and not page.is_visible("#tr-video-empty"))
            check("film added to recent videos", page.evaluate("JSON.parse(localStorage.getItem('pb.recentVideos'))[0]") == "/films/My film.webm")
            check("film loops and is muted", page.evaluate("[document.querySelector('#tr-video').loop, document.querySelector('#tr-video').muted]") == [True, True])
            check("not playing before the session", page.evaluate("document.querySelector('#tr-video').paused"))

            page.click("#tr-start")
            page.wait_for_function("!document.querySelector('#tr-video').paused", timeout=5000)
            check("film plays when the training starts", True)
            page.click("#tr-stop")
            page.wait_for_function("document.querySelector('#tr-video').paused", timeout=5000)
            check("film pauses when the training ends", True)

            page.uncheck("#tr-video-muted")
            check("Muted off reaches the video", page.evaluate("document.querySelector('#tr-video').muted") is False)
            check("Player video untouched", page.evaluate("window.__playerTouched") is None)

            # Reload: mode, film and options are remembered; the film is re-registered.
            page.reload()
            page.wait_for_function("window.__ready === true")
            page.wait_for_function("window.__set.length === 1", timeout=3000)  # fresh page, fresh stub counter
            check("after reload: Video mode and film back", page.is_visible("#tr-video-view")
                  and page.evaluate("window.__set") == ["/films/My film.webm"])
            check("after reload: Muted stays off", not page.is_checked("#tr-video-muted"))

            page.click("#tr-stage-pixel")
            check("back to pixel clip clears the training film", page.evaluate("window.__cleared") == 1
                  and page.is_visible("#tr-pixel-stage") and not page.is_visible("#tr-video-view"))
            # Rings: the live rings become the stage; the clip and the film hide.
            page.click("#tr-stage-rings")
            check("Rings mode: rings are the stage", page.evaluate(
                "document.querySelector('#tr-rings-host').classList.contains('tr-rings-stage')")
                and not page.is_visible("#tr-pixel-stage") and not page.is_visible("#tr-video-view"))
            # Waves: real levels from training:levels, last minute.
            page.click("#tr-stage-waves")
            check("Waves mode: waves shown, rings back to compact", page.is_visible("#tr-waves-canvas")
                  and not page.evaluate("document.querySelector('#tr-rings-host').classList.contains('tr-rings-stage')"))
            check("Waves: hint before the training runs", page.is_visible("#tr-waves-empty"))
            page.click("#tr-start")
            page.wait_for_timeout(200)
            for i in range(6):
                page.evaluate(f"window.__triggerEvent('training:levels', {{vibration: {0.2 + i * 0.1}, suction: {0.5}}})")
                page.wait_for_timeout(60)
            page.wait_for_function("document.querySelector('#tr-waves-empty').hidden", timeout=3000)
            lit = page.evaluate("""(() => { const c = document.querySelector('#tr-waves-canvas');
                const d = c.getContext('2d').getImageData(0, 0, c.width, c.height).data; let n = 0;
                for (let i = 3; i < d.length; i += 4) if (d[i] > 40) n++; return n; })()""")
            check("Waves: real levels drawn", lit > 200, str(lit))
            page.click("#tr-stop")
            page.reload()
            page.wait_for_function("window.__ready === true")
            check("after reload: Waves mode remembered", page.get_attribute("#tr-stage-waves", "aria-checked") == "true")
            check("no page errors", not errors, "; ".join(errors))
            b.close()
    finally:
        shutdown()
        harness.unlink(missing_ok=True)
        CLIP.unlink(missing_ok=True)
    return check.report()


if __name__ == "__main__":
    sys.exit(main())
