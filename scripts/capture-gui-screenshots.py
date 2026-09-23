#!/usr/bin/env python3
"""Capture English Emotion Generator GUI screenshots for docs/media/.

Serves the Vite-built frontend with a window.go stub (no Wails binary),
opens Play + Create tabs, writes PNGs into docs/media/.
"""

from __future__ import annotations

import http.server
import pathlib
import socketserver
import threading

from playwright.sync_api import sync_playwright

ROOT = pathlib.Path(__file__).resolve().parents[1]
DIST = ROOT / "cmd" / "gui-wails" / "frontend" / "dist"
OUT = ROOT / "docs" / "media"

GO_STUB = """
(() => {
  const noop = async () => ({});
  const status = {
    connected: true,
    mock: true,
    name: 'Sam Neo 2',
    address: '',
    rssi: 0,
    batteryOk: true,
    batteryPct: 78,
    transport: 'mock',
    capBattery: true,
    capRaw: true,
  };
  const app = new Proxy({}, {
    get(_t, prop) {
      if (prop === 'CurrentVersion') return async () => 'v0.5.27';
      if (prop === 'GetSettings') return async () => ({
        playbackMock: true,
        playbackVideoPlayAutostart: true,
      });
      if (prop === 'GetDeviceStatus') return async () => status;
      if (prop === 'CheckForUpdate') return async () => ({ available: false });
      if (prop === 'CheckGeneratorDependencies') return async () => ({
        ok: true, python: true, opencv: true, ffmpeg: true,
      });
      if (prop === 'CheckAIRoiAvailable') return async () => ({ available: false });
      if (prop === 'CheckAudioCheckAvailable') return async () => ({ available: false });
      if (prop === 'GetLicenseStatus') return async () => ({ active: false });
      if (prop === 'GetCacheInfo') return async () => ({});
      if (prop === 'GetLogEntries') return async () => [];
      if (prop === 'GetScriptAxisActions') return async () => ({ actions: [] });
      if (prop === 'GetPlaybackSource') return async () => 'script';
      return noop;
    }
  });
  window.go = { main: { App: app } };
  const handlers = {};
  window.runtime = new Proxy({
    EventsOn(name, cb) {
      (handlers[name] ||= []).push(cb);
      return () => {};
    },
    EventsOnMultiple(name, cb) {
      (handlers[name] ||= []).push(cb);
      return () => {};
    },
    EventsOnce(name, cb) {
      (handlers[name] ||= []).push(cb);
      return () => {};
    },
    EventsEmit() {},
    EventsOff() {},
    LogPrint() {}, LogTrace() {}, LogDebug() {}, LogInfo() {},
    LogWarning() {}, LogError() {}, LogFatal() {},
    Quit() {}, WindowMinimise() {}, WindowMaximise() {},
    WindowUnmaximise() {}, WindowFullscreen() {}, WindowUnfullscreen() {},
    WindowSetTitle() {}, WindowIsMaximised() { return false; },
    WindowIsMinimised() { return false; },
    WindowIsFullscreen() { return false; },
    WindowIsNormal() { return true; },
    BrowserOpenURL() {}, ClipboardGetText() { return Promise.resolve(''); },
    ClipboardSetText() { return Promise.resolve(); },
    Environment() { return Promise.resolve({ platform: 'linux', arch: 'amd64' }); },
  }, {
    get(t, prop) {
      if (prop in t) return t[prop];
      return (..._args) => undefined;
    }
  });
})();
"""


def serve_dist():
    handler = http.server.SimpleHTTPRequestHandler
    httpd = socketserver.ThreadingTCPServer(
        ("127.0.0.1", 0),
        lambda *a, **k: handler(*a, directory=str(DIST), **k),
    )
    httpd.allow_reuse_address = True
    threading.Thread(target=httpd.serve_forever, daemon=True).start()
    return f"http://127.0.0.1:{httpd.server_address[1]}", httpd.shutdown


def main():
    if not (DIST / "index.html").exists():
        raise SystemExit(f"missing {DIST}/index.html — run npm run build first")
    OUT.mkdir(parents=True, exist_ok=True)
    base, shutdown = serve_dist()

    with sync_playwright() as p:
        browser = p.chromium.launch(
            headless=True,
            args=["--disable-dev-shm-usage"],
        )
        page = browser.new_page(viewport={"width": 1440, "height": 900})
        page.add_init_script(GO_STUB)
        page.goto(base + "/", wait_until="networkidle")
        page.wait_for_selector("#brand-tagline", timeout=15000)
        page.wait_for_timeout(800)

        # Play tab (default) — empty Emotion Script state
        page.locator('.tab-btn[data-tab="playback"]').click()
        page.wait_for_timeout(400)
        play_path = OUT / "gui-playback.png"
        page.screenshot(path=str(play_path), full_page=False)
        print(f"wrote {play_path}")

        # Rail with live-looking meters (DOM paint only — public face)
        page.evaluate(
            """() => {
              const vib = document.getElementById('sb-vib-pct');
              const suc = document.getElementById('sb-suc-pct');
              const vibBar = document.getElementById('sb-vib-bar');
              const sucBar = document.getElementById('sb-suc-bar');
              const chip = document.getElementById('sb-recipe-chip');
              if (vib) vib.textContent = '42%';
              if (suc) suc.textContent = '28%';
              if (vibBar) vibBar.style.width = '42%';
              if (sucBar) sucBar.style.width = '28%';
              if (chip) chip.textContent = 'Emotion Script · demo.samn';
            }"""
        )
        page.wait_for_timeout(200)
        rail_path = OUT / "gui-player-rail.png"
        page.screenshot(path=str(rail_path), full_page=False)
        print(f"wrote {rail_path}")

        # Create Emotion Script tab
        page.locator('.tab-btn[data-tab="generator"]').click()
        page.wait_for_selector("#tab-generator h2", timeout=10000)
        page.wait_for_timeout(600)
        gen_path = OUT / "gui-generator.png"
        page.screenshot(path=str(gen_path), full_page=False)
        print(f"wrote {gen_path}")

        browser.close()
    shutdown()


if __name__ == "__main__":
    main()
