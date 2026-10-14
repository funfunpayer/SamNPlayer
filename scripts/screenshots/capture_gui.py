#!/usr/bin/env python3
"""Reproducible GUI screenshots for the README (docs/media/gui-*.png).

Loads the real frontend in headless Chromium with stubbed Go bindings and
synthetic data only: a generated stroke curve, generated feel channels and
an ffmpeg test pattern as the "video". No third-party media is used.

Run:  python3 scripts/screenshots/capture_gui.py [--out docs/media]
Needs: playwright (Chromium), ffmpeg.
"""

import argparse
import json
import math
import pathlib
import shutil
import subprocess
import sys
import tempfile

from playwright.sync_api import sync_playwright

ROOT = pathlib.Path(__file__).resolve().parents[2]
FRONTEND = ROOT / "cmd" / "gui-wails" / "frontend"
sys.path.insert(0, str(FRONTEND / "test"))
from _harness import app_stub, serve  # noqa: E402

DURATION_MS = 120_000
VIEW = {"width": 1600, "height": 960}


def stroke_actions():
    """A plausible stroke script: rhythm that speeds up and calms down."""
    acts, t, i = [], 0, 0
    while t <= DURATION_MS:
        phase = t / DURATION_MS
        period = 900 - 500 * math.sin(math.pi * phase)  # faster in the middle
        depth = 35 + 30 * math.sin(math.pi * phase)
        acts.append({"at": int(t), "pos": int(50 + (depth if i % 2 else -depth))})
        t += period / 2
        i += 1
    return acts


def axis(fn, step=500):
    return [{"at": t, "pos": max(0, min(100, int(fn(t / DURATION_MS))))} for t in range(0, DURATION_MS + 1, step)]


def make_video(path):
    subprocess.run(
        ["ffmpeg", "-loglevel", "error", "-y", "-f", "lavfi", "-i",
         f"testsrc2=size=1280x720:rate=25:duration={DURATION_MS // 1000}",
         "-c:v", "libvpx-vp9", "-b:v", "0", "-crf", "45", "-deadline", "realtime", "-cpu-used", "8", str(path)],
        check=True)


def frame_png_b64():
    out = subprocess.run(
        ["ffmpeg", "-loglevel", "error", "-f", "lavfi", "-i", "testsrc2=size=1280x720:rate=1", "-frames:v", "1",
         "-f", "image2pipe", "-vcodec", "png", "-"], check=True, capture_output=True).stdout
    import base64
    return base64.b64encode(out).decode()


def data_js(media_url):
    stroke = stroke_actions()
    vib = axis(lambda p: 45 + 35 * math.sin(p * 9) * math.sin(p * math.pi))
    suc = axis(lambda p: 30 + 25 * (1 + math.cos(p * 5)) * p)
    twist = axis(lambda p: 50 + 30 * math.sin(p * 14), step=2500)
    roll = axis(lambda p: 50 + 20 * math.cos(p * 7), step=4000)
    chapters = [
        {"name": "Intro", "startTime": 0, "endTime": 0},
        {"name": "Build-up", "startTime": 22000, "endTime": 0},
        {"name": "Peak", "startTime": 61000, "endTime": 0},
        {"name": "Cool-down", "startTime": 95000, "endTime": 0},
    ]
    return (
        f"window.__stroke = {json.dumps(stroke)};\n"
        f"window.__axes = {{ general: window.__stroke, vibration: {json.dumps(vib)}, suction: {json.dumps(suc)},"
        f" twist: {json.dumps(twist)}, roll: {json.dumps(roll)} }};\n"
        f"window.__chapters = {json.dumps(chapters)};\n"
        f"window.__media = {json.dumps(media_url)};\n"
        f"window.__frame = {json.dumps(frame_png_b64())};\n"
        "window.__thumb = (i) => { const c = document.createElement('canvas'); c.width = 160; c.height = 90;"
        " const g = c.getContext('2d'); const h = (i * 37) % 360;"
        " const grad = g.createLinearGradient(0, 0, 160, 90); grad.addColorStop(0, `hsl(${h},45%,35%)`);"
        " grad.addColorStop(1, `hsl(${(h + 60) % 360},50%,20%)`); g.fillStyle = grad; g.fillRect(0, 0, 160, 90);"
        " return c.toDataURL('image/png'); };\n"
    )


STUBS = {
    "GetEULA": "async () => ({version:'1', text:'', accepted:true})",
    "IsSuiteBuild": "async () => false",
    "CurrentVersion": "async () => 'v0.7.0'",
    "PickFunscriptFile": "async () => '/demo/Demo clip.funscript'",
    "LoadFunscript": ("async (p) => ({path: p, actionCount: window.__stroke.length, durationMs: %d,"
                      " videoPath: '/demo/Demo clip.mp4', hasVideo: true})" % DURATION_MS),
    "VideoFileURL": "async () => window.__media",
    "SetGeneratorVideo": "async () => window.__media",
    "GetScriptCurve": "async () => window.__stroke.map(a => ({atMs: a.at, pos: a.pos}))",
    "GetScriptAxisActions": "async (axis) => (window.__axes[axis || 'general'] || null)",
    "GetHeatmap": ("async n => Array.from({length: n}, (_, i) => ({atMs: i * %d / n,"
                   " intensity: 0.25 + 0.7 * Math.sin(Math.PI * i / n) * (0.6 + 0.4 * Math.sin(i / 3))}))" % DURATION_MS),
    "GetMarker": "async () => null",
    "GetScriptChapterMarks": "async () => window.__chapters.slice()",
    "GetVideoThumbnails": ("async (n) => Array.from({length: n}, (_, i) => ({atMs: Math.round((i + 0.5) * %d / n),"
                           " dataUrl: window.__thumb(i)}))" % DURATION_MS),
    "ProbeCreateVideo": ("async (p) => ({path: p, name: 'Demo clip.mp4', width: 1920, height: 1080, fps: 29.97,"
                         " vfr: false, durationSec: %d, codec: 'h264', posterUrl: window.__thumb(3)})" % (DURATION_MS // 1000)),
    "GetDeviceStatus": ("async () => ({connected: true, mock: false, transport: 'tcode-serial', name: 'TCode device (L0 + extra axes)',"
                        " address: 'serial://COM3', capabilities: {device: 'TCode device', features: ["
                        "{channel:'stroke',kind:'linear',index:0,driven:true},{channel:'twist',kind:'linear',index:3,driven:true},"
                        "{channel:'roll',kind:'linear',index:4,driven:true}]}})"),
    "GetRuntimeHealth": ("async () => ({dirsCreated: [], dirsFailed: [], ok: true, deps: ["
                         "{id:'ffmpeg', label:'ffmpeg', required:true, found:true, path:'ffmpeg (beside the app)'},"
                         "{id:'ffprobe', label:'ffprobe', required:true, found:true, path:'ffprobe (beside the app)'},"
                         "{id:'python', label:'Python (Generator extras)', required:false, found:false}]})"),
    "GetDiagnosticsHistory": "async () => [{timestamp: '2026-10-12T19:40:00Z', mock: false, deviceName: 'Sam Neo 2', report: {interrupted: false}}]",
    "GetCacheInfo": "async () => ({path:'', files: 14, bytes: 52428800, humanSize: '50.0 MB', clearOnExit: false})",
    "ListSerialPorts": "async () => ['COM3', 'COM4']",
    "PickVideoFile": "async () => '/demo/Demo clip.mp4'",
    "LoadFirstFrame": "async () => ({width: 1280, height: 720, pngBase64: window.__frame})",
    "LoadFrameAt": "async () => ({width: 1280, height: 720, pngBase64: window.__frame})",
    "ScriptExistsForVideo": "async () => false",
    "GetSettings": "async () => ({updateCheckOnStartup:false, playbackMock:true, deviceTransport:'tcode-serial', tcodeSerialPort:'COM3',"
                    " trainingTechnique:'stopstart', trainingChannel:'both', trainingCycles:6, trainingRampUpMs:15000,"
                    " trainingHoldMs:7500, trainingRestMs:30000, trainingPeakIntensity:0.7})",
}


def shoot(page, out, name, selector=None):
    target = out / f"gui-{name}.png"
    if selector:
        page.locator(selector).screenshot(path=str(target))
    else:
        page.screenshot(path=str(target))
    print("wrote", target)


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--out", default=str(ROOT / "docs" / "media"))
    args = ap.parse_args()
    out = pathlib.Path(args.out)
    out.mkdir(parents=True, exist_ok=True)
    if not shutil.which("ffmpeg"):
        sys.exit("ffmpeg is needed for the test-pattern video")

    media_dir = FRONTEND / "test" / "_shots_media"
    media_dir.mkdir(exist_ok=True)
    harness = FRONTEND / "test" / "_shots_harness.html"
    try:
        make_video(media_dir / "clip.webm")
        # The app imports these stylesheets from JS (Vite bundles them); the
        # test server stubs CSS imports, so link them for real, in app order.
        links = "".join(f'<link rel="stylesheet" href="/src/{css}?real" />'
                        for css in ("style.css", "theme_flow.css", "flow_fx.css", "training_rings.css"))
        harness.write_text((FRONTEND / "index.html").read_text()
                           .replace('<link rel="stylesheet" href="/src/style.css" />', links))
        base, shutdown = serve(data_js("/test/_shots_media/clip.webm") + app_stub(STUBS))
        try:
            with sync_playwright() as p:
                b = p.chromium.launch()
                page = b.new_page(viewport=VIEW, device_scale_factor=1)
                page.add_init_script("try{localStorage.clear()}catch(e){}")
                page.goto(f"{base}/test/_shots_harness.html")
                page.wait_for_timeout(1000)

                # Player — Play view with a script, video, lanes.
                page.click("#pb-choose")
                page.wait_for_function("!document.querySelector('#pb-ruler').hidden", timeout=8000)
                page.wait_for_timeout(1500)
                page.evaluate("document.querySelector('video') && (document.querySelector('video').currentTime = 34)")
                for _ in range(3):
                    page.evaluate("document.querySelector('#pb-zoom-in')?.click()")
                    page.wait_for_timeout(200)
                page.wait_for_timeout(800)
                shoot(page, out, "player")

                # Edit — lanes incl. multi-axis companion files, twist selected.
                page.click('#ws-view [data-ws-view="edit"]')
                page.wait_for_timeout(1200)
                page.evaluate("const s = document.querySelector('#pb-axis'); if (s && s.querySelector('option[value=twist]')) { s.value = 'twist'; s.dispatchEvent(new Event('change')); }")
                page.wait_for_timeout(1000)
                shoot(page, out, "edit-multiaxis")

                # Scene Map bottom tab.
                page.click('#ws-view [data-ws-view="play"]')
                page.wait_for_timeout(600)
                page.click('.pb-bottom-tab[data-btab="scenemap"]')
                page.wait_for_function("document.querySelectorAll('.pb-scene-card').length === 4", timeout=8000)
                page.click('.pb-scene-card[data-scene="2"]')
                page.wait_for_timeout(800)
                shoot(page, out, "scene-map")
                page.click('.pb-bottom-tab[data-btab="timeline"]')

                # Inspector → Devices with a TCode stroker (stroke range).
                page.evaluate("window.dispatchEvent(new CustomEvent('device:status', {detail: {connected: true,"
                              " capabilities: {features: [{channel:'stroke',kind:'linear',index:0,driven:true}]}}}))")
                page.evaluate("document.querySelector('#pb-insp-tab-devices')?.click()")
                page.wait_for_timeout(600)

                # Create — with a clip chosen.
                page.click('#rail .tab-btn[data-tab="generator"]')
                page.wait_for_timeout(800)
                page.click('#gen-choose')
                page.wait_for_timeout(2500)
                page.evaluate("window.scrollTo(0, 0); document.querySelectorAll('main, .app-main, #content, .tab-panel').forEach(e => e.scrollTop = 0)")
                page.wait_for_timeout(300)
                shoot(page, out, "create")
                page.screenshot(path=str(out / "gui-create-full.png"), full_page=True)

                # Training — on the neutral "Video" stage. The bundled pixel
                # clip is adult content and must never appear in screenshots:
                # switch the stage, hide the pixel host and check it is gone.
                page.click('#rail .tab-btn[data-tab="training"]')
                page.wait_for_timeout(800)
                page.click('#tr-stage-video')
                page.wait_for_timeout(500)
                page.evaluate("document.querySelectorAll('#tr-pixel-stage > :not(.tr-stage-bar):not(.tr-video-view)')"
                              ".forEach(e => { if (!e.querySelector('#tr-video')) e.style.visibility = 'hidden'; })")
                visible_pixel = page.evaluate(
                    "[...document.querySelectorAll('#tr-pixel-stage img, #tr-pixel-stage canvas')]"
                    ".some(e => e.offsetParent !== null && getComputedStyle(e).visibility !== 'hidden')")
                if visible_pixel:
                    print("skipped gui-training.png: pixel clip still visible")
                else:
                    shoot(page, out, "training")

                # Tools → Home, Tools → Device.
                page.click('#rail .tab-btn[data-tab="tools"]')
                page.wait_for_function("document.querySelectorAll('#th-rows .th-row').length > 0", timeout=8000)
                page.wait_for_timeout(600)
                shoot(page, out, "tools-home")
                page.click('#tools-nav .tab-btn[data-tab="device"]')
                page.wait_for_timeout(1200)
                shoot(page, out, "device")
                b.close()
        finally:
            shutdown()
    finally:
        harness.unlink(missing_ok=True)
        shutil.rmtree(media_dir, ignore_errors=True)


if __name__ == "__main__":
    main()
