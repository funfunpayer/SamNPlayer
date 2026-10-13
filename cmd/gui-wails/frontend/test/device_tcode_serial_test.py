"""D5: Tools → Device offers "TCode over USB (OSR2 / SR6)". Choosing it lists
the computer's real serial ports (ListSerialPorts), preselects the saved
one, and Connect passes the chosen port. No ports → a clear entry.

Run: python3 cmd/gui-wails/frontend/test/device_tcode_serial_test.py
"""

import pathlib
import sys

from playwright.sync_api import sync_playwright

from _harness import Checker, app_stub, serve

HERE = pathlib.Path(__file__).resolve().parent

PAGE = """<!doctype html><html><body><div id="root"></div>
<script type="module">
  import { initDevice } from '/src/device.js';
  initDevice(document.querySelector('#root'));
  window.__ready = true;
</script></body></html>"""


def main():
    check = Checker()
    base, shutdown = serve("window.__calls = []; window.__ports = ['COM3', 'COM7']; let state = { connected: false, sessionActive: false };\n" + app_stub({
        "GetDeviceStatus": "async () => ({ ...state })",
        "GetSettings": "async () => ({ deviceTransport: 'tcode-serial', tcodeSerialPort: 'COM7' })",
        "ListSerialPorts": "async () => window.__ports.slice()",
        "ConnectDeviceVia": ("async (t, a) => { window.__calls.push([t, a]); state = { connected: true, transport: t, "
                             "name: 'TCode device (L0 + extra axes)', address: 'serial://' + a }; return { ...state }; }"),
        "GetDiagnosticsHistory": "async () => []",
    }))
    harness = HERE / "_device_tcode_serial_harness.html"
    harness.write_text(PAGE)
    try:
        with sync_playwright() as p:
            b = p.chromium.launch()
            page = b.new_page()
            errors = []
            page.on("pageerror", lambda e: errors.append(str(e)))
            page.goto(f"{base}/test/_device_tcode_serial_harness.html")
            page.wait_for_function("window.__ready === true")
            page.wait_for_function("document.querySelectorAll('#dev-serial-port option').length === 2", timeout=3000)
            check("USB option offered and restored", page.input_value("#dev-transport") == "tcode-serial")
            check("port picker shown, network fields hidden", page.is_visible("#dev-serial-port")
                  and page.is_hidden("#dev-tcode-addr") and page.is_hidden("#dev-intiface-url"))
            check("real ports listed", page.evaluate("[...document.querySelectorAll('#dev-serial-port option')].map(o => o.value)") == ["COM3", "COM7"])
            check("saved port preselected", page.input_value("#dev-serial-port") == "COM7")
            check("hint names the baud rate", "115200" in page.text_content("#dev-transport-hint"))
            page.evaluate("window.__ports = []")
            page.click("#dev-serial-refresh")
            page.wait_for_function("document.querySelector('#dev-serial-port option').value === ''", timeout=3000)
            check("no ports → clear entry", page.text_content("#dev-serial-port") == "No serial port found")
            page.evaluate("window.__ports = ['COM3', 'COM7']")
            page.click("#dev-serial-refresh")
            page.wait_for_function("document.querySelectorAll('#dev-serial-port option').length === 2", timeout=3000)
            page.select_option("#dev-serial-port", "COM3")
            page.click("#dev-connect")
            page.wait_for_function("window.__calls.length === 1", timeout=3000)
            check("Connect passes the chosen port", page.evaluate("window.__calls[0]") == ["tcode-serial", "COM3"])
            page.wait_for_timeout(200)
            check("port picker locked while connected", page.is_disabled("#dev-serial-port"))
            check("no page errors", errors == [], str(errors[:3]))
            b.close()
    finally:
        shutdown()
        harness.unlink(missing_ok=True)
    return check.report()


if __name__ == "__main__":
    sys.exit(main())
