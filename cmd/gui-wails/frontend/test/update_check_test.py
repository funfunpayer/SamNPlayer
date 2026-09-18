"""Regressionstest für den manuellen "Jetzt nach Updates suchen"-Knopf im
Einstellungen-Tab.

Vorher gab es nur die stille Prüfung beim Programmstart, deren Ergebnis bei
einem Fehler (Netzwerk, GitHub-Ratenlimit, ...) komplett verschluckt wurde
(leeres .catch in main.js) - von außen war "kein Update gefunden, weil es
keins gibt" nicht von "die Prüfung ist stillschweigend gescheitert" zu
unterscheiden, und es gab keine Möglichkeit, auf Wunsch erneut zu prüfen.
Geprüft wird hier: kein Update -> Hinweis nennt die aktuelle Version;
Update verfügbar -> Hinweis nennt die neue Version; Fehler -> Hinweis zeigt
die Fehlermeldung, nicht Stille.

Ausführen:  python3 cmd/gui-wails/frontend/test/update_check_test.py
"""

import pathlib
import sys

from playwright.sync_api import sync_playwright

from _harness import Checker, app_stub, serve

PAGE = """<!doctype html><html><body><div id="root"></div>
<script type="module">
  import { initSettings } from '/src/settings.js';
  window.__settings = initSettings(document.querySelector('#root'));
  window.__ready = true;
</script></body></html>"""


def main():
    check = Checker()
    base, shutdown = serve(app_stub({
        "GetSettings": "async () => ({ reportPath: '', defaultReportPath: '' })",
        "CurrentVersion": "async () => '0.3.0'",
        "CheckForUpdate": "async () => window.__updateResult",
    }))

    harness = pathlib.Path(__file__).resolve().parent / "_update_check_harness.html"
    harness.write_text(PAGE)

    with sync_playwright() as p:
        browser = p.chromium.launch()
        page = browser.new_page()
        page.on("dialog", lambda d: d.dismiss())
        page.on("pageerror", lambda e: print("   [pageerror]", e))
        page.goto(f"{base}/test/_update_check_harness.html")
        page.wait_for_function("window.__ready === true")

        # Kein Update verfügbar.
        page.evaluate("window.__updateResult = { available: false }")
        page.click("#st-update-now")
        page.wait_for_function(
            "document.querySelector('#st-update-status').textContent.includes('No update')",
            timeout=5000)
        txt = page.locator("#st-update-status").inner_text()
        check("no update: status mentions 'No update'", "No update" in txt)
        check("kein Update: aktuelle Version wird genannt", "0.3.0" in txt, txt)

        # Update verfügbar (Dialog wird oben dismissed statt akzeptiert).
        page.evaluate(
            "window.__updateResult = { available: true, release: { tag_name: 'v0.4.0' } }")
        page.click("#st-update-now")
        page.wait_for_function(
            "document.querySelector('#st-update-status').textContent.includes('v0.4.0')",
            timeout=5000)
        check("Update verfügbar: neue Version im Hinweis",
              "v0.4.0" in page.locator("#st-update-status").inner_text())

        # Fehler bei der Prüfung darf NICHT stillschweigend verschwinden.
        page.evaluate("window.__updateResult = { error: 'kein Netz' }")
        page.click("#st-update-now")
        page.wait_for_function(
            "document.querySelector('#st-update-status').textContent.includes('kein Netz')",
            timeout=5000)
        check("Fehler bei der Prüfung wird angezeigt, nicht verschluckt",
              "kein Netz" in page.locator("#st-update-status").inner_text())

        browser.close()

    shutdown()
    harness.unlink(missing_ok=True)
    return check.report()


if __name__ == "__main__":
    sys.exit(main())
