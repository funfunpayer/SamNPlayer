"""Regressionstest für den Intensitätsmesser (zwei atmende Ringe,
Vibration/Suction) im Trainings-Tab.

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

PAGE = """<!doctype html><html><body><div id="root"></div>
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
        return page.locator(f"#tr-ring-wrap-{which}").evaluate("e => e.classList.contains('tr-pulsing')")

    def ring_fill_fraction(page, which):
        """Wie weit der Fortschrittsbogen gefüllt ist, aus stroke-dashoffset
        errechnet (0 = leer, 1 = voll) - die eigentliche visuelle Aussage
        des Rings, nicht nur der Textwert daneben."""
        return page.locator(f"#tr-ring-{which}").evaluate("""e => {
            const circumference = parseFloat(e.getAttribute('stroke-dasharray'));
            const offset = parseFloat(e.style.strokeDashoffset || e.getAttribute('stroke-dashoffset'));
            return 1 - (offset / circumference);
        }""")

    def lit_pixel_count(page, which):
        return page.locator(f"#tr-pixel-{which} .tr-pixel-lit").count()

    with sync_playwright() as p:
        browser = p.chromium.launch()
        page = browser.new_page()
        page.on("pageerror", lambda e: print("   [pageerror]", e))
        page.goto(f"{base}/test/_training_meter_harness.html")
        page.wait_for_function("window.__ready === true")

        check("vor dem Start: Vibrationsring zeigt 0%", ring_value(page, "vibration") == "0%")
        check("vor dem Start: Sog-Ring zeigt 0%", ring_value(page, "suction") == "0%")
        check("vor dem Start: Vibrationsring ist leer", ring_fill_fraction(page, "vibration") < 0.01,
              str(ring_fill_fraction(page, "vibration")))
        check("Pixel-Raster (Vibration) ist im DOM vorhanden",
              page.locator("#tr-pixel-vibration .tr-pixel-cell").count() > 0)
        check("vor dem Start: keine Pixel beleuchtet", lit_pixel_count(page, "vibration") == 0)

        page.click("#tr-start")
        page.select_option("#tr-channel", "both")
        page.evaluate("window.__triggerEvent('training:cycle', "
                       "{ cycleIndex: 0, cyclesTotal: 3, peakIntensity: 0.7, holdMs: 1000 })")
        page.wait_for_function("document.querySelector('#tr-ring-value-vibration').textContent === '70%'")
        check("Kanal 'both': Vibrationsring zeigt die Intensität",
              ring_value(page, "vibration") == "70%", ring_value(page, "vibration"))
        check("Kanal 'both': Sog-Ring zeigt dieselbe Intensität",
              ring_value(page, "suction") == "70%", ring_value(page, "suction"))
        check("Kanal 'both': Vibrationsring ist zu ~70% gefüllt",
              abs(ring_fill_fraction(page, "vibration") - 0.7) < 0.02,
              str(ring_fill_fraction(page, "vibration")))
        check("laufende Session: Vibrationsring pulsiert/atmet", ring_pulsing(page, "vibration"))
        check("laufende Session: Sog-Ring pulsiert/atmet", ring_pulsing(page, "suction"))
        lit_at_70 = lit_pixel_count(page, "vibration")
        check("bei 70% sind Pixel beleuchtet", lit_at_70 > 0, str(lit_at_70))
        check("Pixel-Raster pulsiert mit", page.locator("#tr-pixel-vibration").evaluate(
            "e => e.classList.contains('tr-pulsing')"))

        page.select_option("#tr-channel", "vibration")
        page.evaluate("window.__triggerEvent('training:cycle', "
                       "{ cycleIndex: 1, cyclesTotal: 3, peakIntensity: 0.5, holdMs: 1000 })")
        page.wait_for_function("document.querySelector('#tr-ring-value-vibration').textContent === '50%'")
        check("Kanal 'vibration': nur der Vibrationsring bewegt sich",
              ring_value(page, "vibration") == "50%", ring_value(page, "vibration"))
        check("Kanal 'vibration': Sog-Ring geht auf 0% zurück",
              ring_value(page, "suction") == "0%", ring_value(page, "suction"))
        check("Kanal 'vibration': Sog-Ring pulsiert nicht mehr",
              not ring_pulsing(page, "suction"))
        check("weniger Intensität -> weniger beleuchtete Pixel",
              lit_pixel_count(page, "vibration") < lit_at_70,
              f"{lit_pixel_count(page, 'vibration')} >= {lit_at_70}")
        check("Kanal 'vibration': Sog-Pixel sind wieder dunkel",
              lit_pixel_count(page, "suction") == 0)

        page.evaluate("window.__triggerEvent('training:done')")
        page.wait_for_function("document.querySelector('#tr-ring-value-vibration').textContent === '0%'")
        check("nach Sessionende: Vibrationsring zurückgesetzt",
              ring_value(page, "vibration") == "0%")
        check("nach Sessionende: Sog-Ring zurückgesetzt",
              ring_value(page, "suction") == "0%")
        check("nach Sessionende: kein Pulsieren/Atmen mehr", not ring_pulsing(page, "vibration"))
        check("nach Sessionende: keine Pixel mehr beleuchtet",
              lit_pixel_count(page, "vibration") == 0)

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
        check("Script-Event: beide Ringe pulsieren, wenn beide > 0",
              ring_pulsing(page, "vibration") and ring_pulsing(page, "suction"))
        check("Script-Event: Sog-Ring ist voller gefüllt als Vibrationsring (0.9 > 0.3)",
              ring_fill_fraction(page, "suction") > ring_fill_fraction(page, "vibration"),
              f"suction={ring_fill_fraction(page, 'suction')} vibration={ring_fill_fraction(page, 'vibration')}")
        check("Script-Event: Sog hat mehr beleuchtete Pixel als Vibration (0.9 > 0.3)",
              lit_pixel_count(page, "suction") > lit_pixel_count(page, "vibration"),
              f"suction={lit_pixel_count(page, 'suction')} vibration={lit_pixel_count(page, 'vibration')}")

        browser.close()

    shutdown()
    harness.unlink(missing_ok=True)
    return check.report()


if __name__ == "__main__":
    sys.exit(main())
