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

        # force=True siehe Kommentar beim zweiten #tr-start-Klick weiter
        # unten - der Ring animiert (Kippen/Glanz) schon ab dem ersten
        # Render, nicht erst nach einer Session.
        page.click("#tr-start", force=True)
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

        # Echte, aber SANFTE Bewegung statt nur Farbe: beide Hälften pulsen
        # per Skalierung (ease-in-out, kein hartes Zittern) - Vibration
        # sanft nach außen, Suction sanft nach innen ("gesaugt"). Amplitude
        # kommt aus einer CSS-Variable, die updateIntensityRing proportional
        # zur Intensität setzt.
        vib_amp = page.locator("#tr-ring-anim-vibration").evaluate(
            "e => getComputedStyle(e).getPropertyValue('--vib-amp')").strip()
        suc_amp = page.locator("#tr-ring-anim-suction").evaluate(
            "e => getComputedStyle(e).getPropertyValue('--suc-amp')").strip()
        check("Vibration-Amplitude proportional zur Intensität (70% -> ~0.063)",
              vib_amp == "0.063", vib_amp)
        check("Suction-Amplitude proportional zur Intensität (70% -> ~0.098)",
              suc_amp == "0.098", suc_amp)

        def sample_scales(selector, n, wait_ms):
            values = set()
            for _ in range(n):
                values.add(page.locator(selector).evaluate("e => getComputedStyle(e).transform"))
                page.wait_for_timeout(wait_ms)
            return values

        vib_scales = sample_scales("#tr-ring-anim-vibration", 6, 100)
        check("Vibrationshälfte pulsiert tatsächlich (mehrere Skalierungswerte über Zeit)",
              len(vib_scales) > 1, str(vib_scales))
        check("Vibrationshälfte pulst zeitweise über volle Größe hinaus (nach außen)",
              any("1, 0, 0, 1" not in m for m in vib_scales), str(vib_scales))

        suc_scales = sample_scales("#tr-ring-anim-suction", 6, 160)
        check("Suction-Hälfte pulsiert tatsächlich (mehrere Skalierungswerte über Zeit)",
              len(suc_scales) > 1, str(suc_scales))
        check("Suction-Hälfte zieht sich zeitweise unter volle Größe zusammen (nach innen)",
              any("1, 0, 0, 1" not in m for m in suc_scales), str(suc_scales))
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
        # force=True: der Ring kippt/glänzt jetzt DAUERHAFT (tr-ring-tilt3d/
        # tr-ring-gloss-sweep, siehe style.css) - ein rein kosmetisches,
        # sub-Pixel-kleines 3D-transform+filter auf dem Ring, das einen
        # echten Mausklick nicht im Geringsten stört. Playwrights
        # Stabilitätsprüfung vergleicht #tr-starts getBoundingClientRect()
        # aber zwischen zwei Frames auf exakte Gleichheit - bei einer
        # dauerhaft animierten Seite (egal wie klein die Bewegung) trifft
        # das praktisch nie zu (mit allen Animationen/Transitions
        # abgeschaltet verschwindet die "Instabilität" komplett - es ist
        # kein echter Layout-Sprung des Buttons). force überspringt genau
        # diese hier irreführende Prüfung.
        page.click("#tr-start", force=True)
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
