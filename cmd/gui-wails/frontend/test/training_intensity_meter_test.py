"""Regressionstest für den Intensitätsmesser (ein atmender Doppelring -
außen Vibration, innen Suction) im Trainings-Tab.

Muss für BEIDE Ereignisquellen funktionieren: die einfache Technik/Kanal-
Form ("training:cycle", ein einzelner peakIntensity-Wert - der Kanal
kommt aus dem eigenen Formularfeld, nicht aus dem Event) und die
Multi-Phasen-Scripts ("training:scriptCycle", zwei getrennte Werte). Und
er muss beim Sessionende wieder auf 0 zurückgehen, sonst zeigt er nach
"Stop" weiter die letzte Intensität an.

Ausführen:  python3 cmd/gui-wails/frontend/test/training_intensity_meter_test.py
"""

import pathlib
import sys

from playwright.sync_api import sync_playwright

from _harness import Checker, app_stub, serve

FRONTEND = pathlib.Path(__file__).resolve().parents[1]

# style.css geladen (mit "?raw=1", weil der Harness den bloßen Pfad
# "/src/style.css" auf ein leeres JS-Modul umbiegt - für main.js's
# CSS-Import gedacht, siehe _harness.py). Andere Trainings-Tests brauchen
# das nicht (sie prüfen nur DOM-Klassen/-Attribute), aber die
# Zittern/Zusammenziehen-Animationen unten sind selbst CSS-Keyframes -
# ohne geladenes Stylesheet bliebe computed transform immer "none".
PAGE = """<!doctype html><html><head><link rel="stylesheet" href="/src/style.css?raw=1"></head>
<body><div id="root"></div>
<script type="module">
  import { initTraining } from '/src/training.js';
  initTraining(document.querySelector('#root'));
  window.__ready = true;
</script></body></html>"""


def main():
    check = Checker()
    base, shutdown = serve(app_stub({
        "StartTraining": "async () => {}",
        "StopTraining": "async () => {}",
        "TrainingHistory": "async () => []",
        "ListTrainingScripts": "async () => []",
    }))

    harness = FRONTEND / "test" / "_training_meter_harness.html"
    harness.write_text(PAGE)

    def ring_value(page, which):
        return page.locator(f"#tr-ring-value-{which}").inner_text()

    def ring_pulsing(page, which):
        """Das Glanz-Glühen einer einzelnen Bahn - eigene Klasse auf dem
        jeweiligen Kreis, unabhängig vom gemeinsamen Atmen des Widgets."""
        return page.locator(f"#tr-ring-{which}").evaluate("e => e.classList.contains('tr-pulsing')")

    def widget_breathing(page):
        return page.locator("#tr-ring-wrap").evaluate("e => e.classList.contains('tr-pulsing')")

    def ring_fill_fraction(page, which):
        """Wie weit der Fortschrittsbogen gefüllt ist, aus stroke-dashoffset
        errechnet (0 = leer, 1 = voll) - die eigentliche visuelle Aussage
        des Rings, nicht nur der Textwert daneben."""
        return page.locator(f"#tr-ring-{which}").evaluate("""e => {
            const circumference = parseFloat(e.getAttribute('stroke-dasharray'));
            const offset = parseFloat(e.style.strokeDashoffset || e.getAttribute('stroke-dashoffset'));
            return 1 - (offset / circumference);
        }""")

    with sync_playwright() as p:
        browser = p.chromium.launch()
        page = browser.new_page()
        page.on("pageerror", lambda e: print("   [pageerror]", e))
        page.goto(f"{base}/test/_training_meter_harness.html")
        page.wait_for_function("window.__ready === true")

        check("EIN gemeinsames Ring-Widget im DOM (kein zweites)",
              page.locator("#tr-ring-wrap").count() == 1)
        check("vor dem Start: Vibrationsring zeigt 0%", ring_value(page, "vibration") == "0%")
        check("vor dem Start: Sog-Ring zeigt 0%", ring_value(page, "suction") == "0%")
        check("vor dem Start: Vibrationsring ist leer", ring_fill_fraction(page, "vibration") < 0.01,
              str(ring_fill_fraction(page, "vibration")))
        check("Legende nennt beide Achsen",
              "Vibration" in page.locator(".tr-ring-legend").inner_text()
              and "Suction" in page.locator(".tr-ring-legend").inner_text())

        page.click("#tr-start")
        page.select_option("#tr-channel", "both")
        page.evaluate("window.__triggerEvent('training:cycle', "
                       "{ cycleIndex: 0, cyclesTotal: 3, peakIntensity: 0.7, holdMs: 1000 })")
        page.wait_for_function("document.querySelector('#tr-ring-value-vibration').textContent === '70%'")
        check("Kanal 'both': Vibrationsring (äußere Bahn) zeigt die Intensität",
              ring_value(page, "vibration") == "70%", ring_value(page, "vibration"))
        check("Kanal 'both': Sog-Ring (innere Bahn) zeigt dieselbe Intensität",
              ring_value(page, "suction") == "70%", ring_value(page, "suction"))
        check("Kanal 'both': Vibrationsring ist zu ~70% gefüllt",
              abs(ring_fill_fraction(page, "vibration") - 0.7) < 0.02,
              str(ring_fill_fraction(page, "vibration")))
        check("laufende Session: Vibrationsbahn glüht", ring_pulsing(page, "vibration"))
        check("laufende Session: Sog-Bahn glüht", ring_pulsing(page, "suction"))

        # Echte Bewegung, nicht nur Farbe: Vibration zittert (Position
        # ändert sich mehrfach über Zeit), Suction zieht sich zusammen
        # (Skalierung sinkt zeitweise unter 1). Amplitude kommt aus einer
        # CSS-Variable, die updateIntensityRing proportional zur
        # Intensität setzt.
        vib_amp = page.locator("#tr-ring-anim-vibration").evaluate(
            "e => getComputedStyle(e).getPropertyValue('--vib-amp')").strip()
        suc_amp = page.locator("#tr-ring-anim-suction").evaluate(
            "e => getComputedStyle(e).getPropertyValue('--suc-amp')").strip()
        check("Vibration-Amplitude proportional zur Intensität (70% -> ~1.54px)",
              vib_amp == "1.54px", vib_amp)
        check("Suction-Amplitude proportional zur Intensität (70% -> ~0.112)",
              suc_amp == "0.112", suc_amp)

        positions = set()
        for _ in range(5):
            box = page.locator("#tr-ring-anim-vibration").bounding_box()
            positions.add((round(box["x"], 1), round(box["y"], 1)))
            page.wait_for_timeout(60)
        check("Vibrationsbahn bewegt sich tatsächlich (mehrere Positionen über Zeit)",
              len(positions) > 1, str(positions))

        scales = set()
        for _ in range(6):
            m = page.locator("#tr-ring-anim-suction").evaluate("e => getComputedStyle(e).transform")
            scales.add(m)
            page.wait_for_timeout(160)
        check("Suction-Bahn skaliert sich tatsächlich (mehrere Werte über Zeit)",
              len(scales) > 1, str(scales))
        check("Suction-Bahn zieht sich zeitweise unter volle Größe zusammen",
              any("1, 0, 0, 1" not in m for m in scales), str(scales))
        check("laufende Session: das ganze Widget atmet", widget_breathing(page))

        page.select_option("#tr-channel", "vibration")
        page.evaluate("window.__triggerEvent('training:cycle', "
                       "{ cycleIndex: 1, cyclesTotal: 3, peakIntensity: 0.5, holdMs: 1000 })")
        page.wait_for_function("document.querySelector('#tr-ring-value-vibration').textContent === '50%'")
        check("Kanal 'vibration': nur der Vibrationsring bewegt sich",
              ring_value(page, "vibration") == "50%", ring_value(page, "vibration"))
        check("Kanal 'vibration': Sog-Ring geht auf 0% zurück",
              ring_value(page, "suction") == "0%", ring_value(page, "suction"))
        check("Kanal 'vibration': Sog-Bahn glüht nicht mehr",
              not ring_pulsing(page, "suction"))
        check("Kanal 'vibration': Widget atmet weiter (Vibration noch aktiv)",
              widget_breathing(page))

        page.evaluate("window.__triggerEvent('training:done')")
        page.wait_for_function("document.querySelector('#tr-ring-value-vibration').textContent === '0%'")
        check("nach Sessionende: Vibrationsring zurückgesetzt",
              ring_value(page, "vibration") == "0%")
        check("nach Sessionende: Sog-Ring zurückgesetzt",
              ring_value(page, "suction") == "0%")
        check("nach Sessionende: keine Bahn glüht mehr", not ring_pulsing(page, "vibration"))
        check("nach Sessionende: Widget atmet nicht mehr", not widget_breathing(page))

        # Script-Pfad: beide Werte kommen getrennt im Event, kein
        # Formularfeld-Umweg wie bei der einfachen Form oben.
        page.click("#tr-start")
        page.evaluate("window.__triggerEvent('training:scriptCycle', {"
                       "phaseIndex: 0, phaseName: 'Wave', phasesTotal: 1,"
                       "repeatIndex: 0, repeatsTotal: 1, vibrationPeak: 0.3, suctionPeak: 0.9,"
                       "restMs: 1000, stoppedByUser: false, arousalBefore: 0 })")
        page.wait_for_function("document.querySelector('#tr-ring-value-suction').textContent === '90%'")
        check("Script-Event: Vibrationsring übernimmt vibrationPeak",
              ring_value(page, "vibration") == "30%", ring_value(page, "vibration"))
        check("Script-Event: Sog-Ring übernimmt suctionPeak",
              ring_value(page, "suction") == "90%", ring_value(page, "suction"))
        check("Script-Event: beide Bahnen glühen, wenn beide > 0",
              ring_pulsing(page, "vibration") and ring_pulsing(page, "suction"))
        check("Script-Event: Sog-Ring ist voller gefüllt als Vibrationsring (0.9 > 0.3)",
              ring_fill_fraction(page, "suction") > ring_fill_fraction(page, "vibration"),
              f"suction={ring_fill_fraction(page, 'suction')} vibration={ring_fill_fraction(page, 'vibration')}")

        browser.close()

    shutdown()
    harness.unlink(missing_ok=True)
    return check.report()


if __name__ == "__main__":
    sys.exit(main())
