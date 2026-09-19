"""Regressionstest für den Generator-Tab: Tf/Tj-Profil.

Tf und Tj waren zwei separate Dropdown-Einträge, obwohl sie intern seit
jeher dasselbe Rezept sind (funscript/recipe.go NormalizeProfile bildet
beide auf denselben Wert ab) - zu einem Eintrag zusammengelegt. Zusätzlich
wird das Profil jetzt automatisch ausgewählt, sobald eine 2. Region
markiert wird: keine andere Kombination aus Verfahren/Profil wertet eine
2. Region überhaupt aus (siehe generate_funscript.py), das Markieren IST
also bereits die eindeutige Auswahl - die Dropdown-Wahl von Hand ist dann
nur noch ein zusätzlicher, überflüssiger Schritt.

Ausführen:  python3 cmd/gui-wails/frontend/test/generator_tftj_test.py
"""

import pathlib
import sys

from playwright.sync_api import sync_playwright

from _harness import Checker, app_stub, serve

FRONTEND = pathlib.Path(__file__).resolve().parents[1]

PAGE = """<!doctype html><html><body><div id="root"></div>
<script type="module">
  import { initGenerator } from '/src/generator.js';
  initGenerator(document.querySelector('#root'), { loadScriptPath: () => {} });
  window.__ready = true;
</script></body></html>"""

# 1x1-Pixel-PNG (transparent) - die tatsächlichen Bildmaße sind für den Test
# egal, nativeW/nativeH kommen aus dem LoadFirstFrame-Stub, nicht aus dem
# dekodierten Bild.
TINY_PNG_B64 = ("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk"
                "+A8AAQUBAScY42YAAAAASUVORK5CYII=")


def main():
    check = Checker()
    base, shutdown = serve(app_stub({
        "PickVideoFile": "async () => '/tmp/video.mp4'",
        "LoadFirstFrame": f"async () => ({{ width: 640, height: 360, "
                          f"pngBase64: '{TINY_PNG_B64}' }})",
        "LoadFrameAt": f"async () => ({{ width: 640, height: 360, "
                       f"pngBase64: '{TINY_PNG_B64}' }})",
        "SuggestPipeline": "async (w,h,w2,h2) => ({ Backend: w2>0?'csrt':'csrt', Profile: w2>0?'tf':'standard', Reason: 'test', GoPath: true })",
        "CheckAIRoiAvailable": "async () => false",
        "CheckAudioCheckAvailable": "async () => false",
    }))
    harness = FRONTEND / "test" / "_generator_tftj_harness.html"
    harness.write_text(PAGE)

    with sync_playwright() as p:
        browser = p.chromium.launch()
        page = browser.new_page()
        page.on("dialog", lambda d: d.dismiss())
        page.on("pageerror", lambda e: print("   [pageerror]", e))
        page.goto(f"{base}/test/_generator_tftj_harness.html")
        page.wait_for_function("window.__ready === true")

        options = page.eval_on_selector_all("#gen-profile option", "els => els.map(e => e.value)")
        check("Dropdown hat nur einen Tf/Tj-Eintrag (kein separates 'tj' mehr)",
              options == ["standard", "weich", "autotune", "tf"] and "tj" not in options,
              str(options))

        page.click("#gen-choose")
        page.wait_for_function(
            "document.querySelector('#gen-autoroi').disabled === false", timeout=5000)

        box = page.locator("#roi-canvas").bounding_box()

        # --- 1. Region: ändert das Profil NICHT ------------------------------
        page.mouse.move(box["x"] + 40, box["y"] + 40)
        page.mouse.down()
        page.mouse.move(box["x"] + 120, box["y"] + 120, steps=5)
        page.mouse.up()
        page.wait_for_function(
            "!document.querySelector('#gen-roi-label').textContent.includes('No ')", timeout=5000)
        check("1. Region gesetzt", True)
        check("Profil bleibt 'standard' nach nur einer Region",
              page.locator("#gen-profile").input_value() == "standard",
              page.locator("#gen-profile").input_value())

        # --- 2. Region: schaltet automatisch auf Tf/Tj um ---------------------
        page.click("#gen-roi2-toggle")
        page.mouse.move(box["x"] + 200, box["y"] + 40)
        page.mouse.down()
        page.mouse.move(box["x"] + 280, box["y"] + 120, steps=5)
        page.mouse.up()

        page.wait_for_function(
            "document.querySelector('#gen-profile').value === 'tf'", timeout=5000)
        check("2. Region markiert -> Profil automatisch auf Tf/Tj umgeschaltet", True)
        check("2. Region ist gesetzt",
              not page.locator("#gen-roi2-label").inner_text().startswith("Keine"),
              page.locator("#gen-roi2-label").inner_text())
        check("Tf/Tj-Hinweistext sichtbar",
              page.locator("#gen-tftj-hint").evaluate("e => e.style.display") == "block")
        check("Kontakt-Vibration-Zeile sichtbar",
              page.locator("#gen-contact-vibration-row").evaluate("e => e.style.display") == "flex")
        # Standard an bei Tf/Tj — Optionen sofort sichtbar
        check("Kontakt-Vibration standardmäßig aktiv",
              page.locator("#gen-contact-vibration").is_checked())
        check("Kontakt-Vibration-Optionen sichtbar",
              page.locator("#gen-contact-vibration-opts").evaluate("e => e.style.display") == "block")
        check("Default-Kurve soft (wie Berührung)",
              page.locator("#gen-contact-curve").input_value() == "soft")
        check("Empfindlichkeits-Slider vorhanden",
              page.locator("#gen-contact-span").count() == 1)
        check("Kurvenwahl vorhanden",
              page.locator("#gen-contact-curve").count() == 1)

        browser.close()

    shutdown()
    harness.unlink(missing_ok=True)

    return check.report()


if __name__ == "__main__":
    sys.exit(main())
