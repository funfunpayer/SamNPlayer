"""Topbar-Geräteindikator: Name des verbundenen Geräts oben rechts."""

import pathlib
import sys

from playwright.sync_api import sync_playwright

from _harness import app_stub, serve

FRONTEND = pathlib.Path(__file__).resolve().parents[1]

PAGE = """<!doctype html><html><head><link rel="stylesheet" href="/src/style.css"/></head>
<body>
<div id="app">
  <div class="app-main">
    <header id="topbar">
      <h1 class="topbar-brand"><span class="mark">SamN</span><span class="rest">Player</span></h1>
      <div class="pills">
        <button type="button" id="topbar-device" class="topbar-device is-offline">
          <svg class="topbar-device-icon" viewBox="0 0 24 24"><rect x="7" y="4" width="10" height="16" rx="3"/><path d="M12 17h.01"/></svg>
          <span id="topbar-device-name">Nicht verbunden</span>
          <i id="topbar-device-led" class="topbar-device-led"></i>
        </button>
        <span id="version-label" class="pill">dev</span>
      </div>
    </header>
    <div class="stage">
      <div class="stage-main">
        <section id="tab-playback" class="tab-panel active"></section>
        <section id="tab-device" class="tab-panel"></section>
      </div>
    </div>
  </div>
</div>
<nav id="rail" style="display:none">
  <button class="tab-btn active" data-tab="playback"></button>
  <button class="tab-btn" data-tab="device"></button>
</nav>
<script type="module">
  function switchTab(name) {
    document.querySelectorAll('.tab-btn').forEach(b => b.classList.toggle('active', b.dataset.tab === name));
    document.querySelectorAll('.tab-panel').forEach(p => p.classList.toggle('active', p.id === 'tab-' + name));
    window.__lastTab = name;
  }
  const btn = document.getElementById('topbar-device');
  const nameEl = document.getElementById('topbar-device-name');
  function render(st) {
    btn.classList.remove('is-online', 'is-offline', 'is-searching');
    if (st && st.connected) {
      btn.classList.add('is-online');
      nameEl.textContent = st.mock ? 'Mock-Gerät' : (st.name && String(st.name).trim()) || 'Verbunden';
    } else {
      btn.classList.add('is-offline');
      nameEl.textContent = 'Nicht verbunden';
    }
  }
  window.addEventListener('device:status', e => render(e.detail || {}));
  btn.addEventListener('click', () => switchTab('device'));
  render({ connected: false });
  window.__ready = true;
  window.__setDevice = (st) => window.dispatchEvent(new CustomEvent('device:status', { detail: st }));
</script>
</body></html>"""


def main():
    base, shutdown = serve(app_stub({
        "GetDeviceStatus": "async () => ({ connected: false })",
    }))
    (FRONTEND / "test" / "_topbar_device_harness.html").write_text(PAGE)
    failures = []

    def check(name, cond, detail=""):
        print(("  OK   " if cond else "  FAIL ") + name + (f"  [{detail}]" if not cond else ""))
        if not cond:
            failures.append(name)

    with sync_playwright() as p:
        browser = p.chromium.launch()
        page = browser.new_page()
        page.goto(f"{base}/test/_topbar_device_harness.html")
        page.wait_for_function("window.__ready === true")

        label = lambda: page.locator("#topbar-device-name").inner_text()
        check("Start offline", label() == "Nicht verbunden", label())
        check("Start Klasse offline",
              "is-offline" in page.locator("#topbar-device").get_attribute("class"))

        page.evaluate("window.__setDevice({ connected: true, name: 'Sam Neo 2 Pro', mock: false })")
        check("Verbunden zeigt Gerätename", label() == "Sam Neo 2 Pro", label())
        check("Online-Klasse", "is-online" in page.locator("#topbar-device").get_attribute("class"))

        page.evaluate("window.__setDevice({ connected: true, mock: true, name: 'Mock' })")
        check("Mock-Label", label().startswith("Mock"), label())

        page.evaluate("window.__setDevice({ connected: false })")
        check("Nach Trennen offline", label() == "Nicht verbunden", label())

        page.click("#topbar-device")
        check("Klick öffnet Gerätetab", page.evaluate("window.__lastTab") == "device")

        browser.close()
    shutdown()
    (FRONTEND / "test" / "_topbar_device_harness.html").unlink(missing_ok=True)

    if failures:
        print("FEHLGESCHLAGEN:", ", ".join(failures))
        return 1
    print("Alle Prüfungen bestanden.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
