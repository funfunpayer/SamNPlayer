"""Training follows the connected device (training_device.js): rings and the
Channel select show only the channels the device has; a stroker without
vibration / suction cannot start; unknown device or Mock shows both.

Run: python3 cmd/gui-wails/frontend/test/training_device_test.py
"""

import pathlib
import sys

from playwright.sync_api import sync_playwright

from _harness import Checker, app_stub, serve

HERE = pathlib.Path(__file__).resolve().parent

NEO2 = "{connected:true, name:'Sam Neo 2', capabilities:{device:'Sam Neo 2', features:[{channel:'vibration',kind:'scalar',driven:true},{channel:'suction',kind:'scalar',driven:true}]}}"
VIB = "{connected:true, name:'Lush', capabilities:{device:'Lush', features:[{channel:'vibration',kind:'scalar',driven:true}]}}"
STROKER = "{connected:true, name:'OSR2', capabilities:{device:'OSR2', features:[{channel:'stroke',kind:'linear',driven:true}]}}"


def main():
    check = Checker()
    links = "".join(f'<link rel="stylesheet" href="/src/{css}?real" />'
                    for css in ("style.css", "theme_flow.css", "flow_fx.css", "training_rings.css"))
    html = (HERE.parent / "index.html").read_text().replace('<link rel="stylesheet" href="/src/style.css" />', links)
    harness = HERE / "_training_device_harness.html"
    harness.write_text(html)
    base, shutdown = serve("window.__started = 0;\n" + app_stub({
        "GetEULA": "async () => ({version:'x', text:'', accepted:true})",
        "IsSuiteBuild": "async () => false",
        "CurrentVersion": "async () => 'test'",
        "GetSettings": "async () => ({updateCheckOnStartup:false})",
        "GetDeviceStatus": "async () => (window.__dev || {connected:false})",
        "ListTrainingScripts": "async () => []",
        "TrainingHistory": "async () => []",
        "StartTraining": "async () => { window.__started++; }",
    }))
    try:
        with sync_playwright() as p:
            b = p.chromium.launch()
            page = b.new_page(viewport={"width": 1600, "height": 960})
            errors = []
            page.on("pageerror", lambda e: errors.append(str(e)))
            page.goto(f"{base}/test/{harness.name}")
            page.click('#mode-pill'); page.click('#mode-deck [data-tab="training"]')
            page.click("#tr-stage-rings")
            page.wait_for_timeout(300)
            ring = lambda axis: page.is_visible(f'.flow-tr__channel[data-axis="{axis}"]')
            # The app also polls GetDeviceStatus: the stub answers the same status.
            status = lambda js: page.evaluate(f"window.__dev = {js}; window.dispatchEvent(new CustomEvent('device:status', {{detail: window.__dev}}))")

            check("no device: both rings", ring("vibration") and ring("suction"))
            check("no device: no note", page.is_hidden("#tr-device-note"))

            status(NEO2)
            check("Sam Neo 2: both rings", ring("vibration") and ring("suction"))
            check("Sam Neo 2: all channel options", not page.evaluate("[...document.querySelectorAll('#tr-channel option')].some(o => o.disabled)"))

            status(VIB)
            check("vibrator: only the vibration ring", ring("vibration") and not ring("suction"))
            check("vibrator: channel set to vibration", page.input_value("#tr-channel") == "vibration")
            check("vibrator: suction / both not offered",
                  page.evaluate("[...document.querySelectorAll('#tr-channel option')].filter(o => o.disabled).map(o => o.value).sort().join(',')") == "both,suction")
            check("vibrator: note names the device", "Lush has no suction" in page.text_content("#tr-device-note"))

            status(STROKER)
            check("stroker: no ring", not ring("vibration") and not ring("suction"))
            check("stroker: explained", "OSR2 has no vibration or suction" in page.text_content("#tr-device-note"))
            page.click("#tr-start")
            page.wait_for_timeout(300)
            check("stroker: Start does not start", page.evaluate("window.__started") == 0)

            page.check("#tr-mock")
            check("Mock: both rings again", ring("vibration") and ring("suction"))
            check("Mock: note gone", page.is_hidden("#tr-device-note"))
            page.click("#tr-start")
            page.wait_for_function("window.__started === 1", timeout=3000)
            check("Mock: Start starts", True)
            check("no page errors", not errors, "; ".join(errors))
            b.close()
    finally:
        shutdown()
        harness.unlink(missing_ok=True)
    return check.report()


if __name__ == "__main__":
    sys.exit(main())
