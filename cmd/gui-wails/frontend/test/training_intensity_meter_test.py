"""Regressionstest für den Intensitätsmesser (zwei pulsierende Balken,
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

    def bar_width(page, which):
        return page.locator(f"#tr-meter-{which}").evaluate("e => e.style.width")

    def bar_pulsing(page, which):
        return page.locator(f"#tr-meter-{which}").evaluate("e => e.classList.contains('tr-pulsing')")

    def lit_pixel_count(page, which):
        return page.locator(f"#tr-pixel-{which} .tr-pixel-lit").count()

    with sync_playwright() as p:
        browser = p.chromium.launch()
        page = browser.new_page()
        page.on("pageerror", lambda e: print("   [pageerror]", e))
        page.goto(f"{base}/test/_training_meter_harness.html")
        page.wait_for_function("window.__ready === true")

        check("vor dem Start: Vibrationsbalken auf 0",
              bar_width(page, "vibration") in ("0%", ""))
        check("vor dem Start: Sog-Balken auf 0",
              bar_width(page, "suction") in ("0%", ""))
        check("Pixel-Raster (Vibration) ist im DOM vorhanden",
              page.locator("#tr-pixel-vibration .tr-pixel-cell").count() > 0)
        check("vor dem Start: keine Pixel beleuchtet", lit_pixel_count(page, "vibration") == 0)

        page.click("#tr-start")
        page.select_option("#tr-channel", "both")
        page.evaluate("window.__triggerEvent('training:cycle', "
                       "{ cycleIndex: 0, cyclesTotal: 3, peakIntensity: 0.7, holdMs: 1000 })")
        page.wait_for_function("document.querySelector('#tr-meter-vibration').style.width === '70%'")
        check("Kanal 'both': Vibrationsbalken zeigt die Intensität",
              bar_width(page, "vibration") == "70%", bar_width(page, "vibration"))
        check("Kanal 'both': Sog-Balken zeigt dieselbe Intensität",
              bar_width(page, "suction") == "70%", bar_width(page, "suction"))
        check("laufende Session: Vibrationsbalken pulsiert", bar_pulsing(page, "vibration"))
        check("laufende Session: Sog-Balken pulsiert", bar_pulsing(page, "suction"))
        lit_at_70 = lit_pixel_count(page, "vibration")
        check("bei 70% sind Pixel beleuchtet", lit_at_70 > 0, str(lit_at_70))
        check("Pixel-Raster pulsiert mit", page.locator("#tr-pixel-vibration").evaluate(
            "e => e.classList.contains('tr-pulsing')"))

        page.select_option("#tr-channel", "vibration")
        page.evaluate("window.__triggerEvent('training:cycle', "
                       "{ cycleIndex: 1, cyclesTotal: 3, peakIntensity: 0.5, holdMs: 1000 })")
        page.wait_for_function("document.querySelector('#tr-meter-vibration').style.width === '50%'")
        check("Kanal 'vibration': nur der Vibrationsbalken bewegt sich",
              bar_width(page, "vibration") == "50%", bar_width(page, "vibration"))
        check("Kanal 'vibration': Sog-Balken geht auf 0 zurück",
              bar_width(page, "suction") == "0%", bar_width(page, "suction"))
        check("Kanal 'vibration': Sog-Balken pulsiert nicht mehr",
              not bar_pulsing(page, "suction"))
        check("weniger Intensität -> weniger beleuchtete Pixel",
              lit_pixel_count(page, "vibration") < lit_at_70,
              f"{lit_pixel_count(page, 'vibration')} >= {lit_at_70}")
        check("Kanal 'vibration': Sog-Pixel sind wieder dunkel",
              lit_pixel_count(page, "suction") == 0)

        page.evaluate("window.__triggerEvent('training:done')")
        page.wait_for_function("document.querySelector('#tr-meter-vibration').style.width === '0%'")
        check("nach Sessionende: Vibrationsbalken zurückgesetzt",
              bar_width(page, "vibration") == "0%")
        check("nach Sessionende: Sog-Balken zurückgesetzt",
              bar_width(page, "suction") == "0%")
        check("nach Sessionende: kein Pulsieren mehr", not bar_pulsing(page, "vibration"))
        check("nach Sessionende: keine Pixel mehr beleuchtet",
              lit_pixel_count(page, "vibration") == 0)

        # Script-Pfad: beide Werte kommen getrennt im Event, kein
        # Formularfeld-Umweg wie bei der einfachen Form oben.
        page.click("#tr-start")
        page.evaluate("window.__triggerEvent('training:scriptCycle', {"
                       "phaseIndex: 0, phaseName: 'Wave', phasesTotal: 1,"
                       "repeatIndex: 0, repeatsTotal: 1, vibrationPeak: 0.3, suctionPeak: 0.9,"
                       "restMs: 1000, stoppedByUser: false, arousalBefore: 0 })")
        page.wait_for_function("document.querySelector('#tr-meter-suction').style.width === '90%'")
        check("Script-Event: Vibrationsbalken übernimmt vibrationPeak",
              bar_width(page, "vibration") == "30%", bar_width(page, "vibration"))
        check("Script-Event: Sog-Balken übernimmt suctionPeak",
              bar_width(page, "suction") == "90%", bar_width(page, "suction"))
        check("Script-Event: beide pulsieren, wenn beide > 0",
              bar_pulsing(page, "vibration") and bar_pulsing(page, "suction"))
        check("Script-Event: Sog hat mehr beleuchtete Pixel als Vibration (0.9 > 0.3)",
              lit_pixel_count(page, "suction") > lit_pixel_count(page, "vibration"),
              f"suction={lit_pixel_count(page, 'suction')} vibration={lit_pixel_count(page, 'vibration')}")

        browser.close()

    shutdown()
    harness.unlink(missing_ok=True)
    return check.report()


if __name__ == "__main__":
    sys.exit(main())
