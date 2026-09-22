"""Regressionstest für den KI-Trainingssystem-Tab.

Deckt den vollen Weg ab, den app_roi_training.go anbietet: Region(en)
markieren -> Bootstrap anstoßen -> Kontrollansicht zeigt die gesammelten
Beispiele mit Box-Overlay -> Verwerfen entfernt ein Beispiel -> Übersicht
zählt Klassen -> Training anstoßen und Log/Status aus den Events übernehmen.
Die eigentliche Python-/Go-Logik (Tracking, Label-Dateien, Training) läuft
hier nicht mit - nur die Verdrahtung im Frontend wird geprüft, mit Stubs für
alle Go-Bindings (siehe app_stub in _harness.py).

Ausführen:  python3 cmd/gui-wails/frontend/test/roi_training_test.py
"""

import json
import pathlib
import sys

from playwright.sync_api import sync_playwright

from _harness import Checker, app_stub, serve

FRONTEND = pathlib.Path(__file__).resolve().parents[1]

PAGE = """<!doctype html><html><body><div id="root"></div>
<script type="module">
  import { initRoiTraining } from '/src/roi_training.js';
  initRoiTraining(document.querySelector('#root'));
  window.__ready = true;
</script></body></html>"""

# 1x1-Pixel-PNG (transparent) - dieselbe Attrappe wie in generator_tftj_test.py.
TINY_PNG_B64 = ("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk"
                "+A8AAQUBAScY42YAAAAASUVORK5CYII=")

SAMPLES = [
    {
        "split": "train", "name": "clipA_000000.jpg", "imagePath": "/data/images/train/clipA_000000.jpg",
        "boxes": [{"classId": 0, "className": "brust", "xc": 0.5, "yc": 0.5, "w": 0.2, "h": 0.2}],
    },
    {
        "split": "train", "name": "clipA_000001.jpg", "imagePath": "/data/images/train/clipA_000001.jpg",
        "boxes": [{"classId": 0, "className": "brust", "xc": 0.4, "yc": 0.4, "w": 0.2, "h": 0.2}],
    },
]

SUMMARY = {
    "datasetDir": "/data",
    "hasDataYaml": True,
    "trainImages": 2,
    "readyToTrain": True,
    "classes": [{"className": "brust", "classId": 0, "trainCount": 2, "valCount": 0}],
}


def main():
    check = Checker()
    base, shutdown = serve(app_stub({
        "PickVideoFile": "async () => '/tmp/video.mp4'",
        "LoadFirstFrame": f"async () => ({{ width: 640, height: 360, "
                          f"pngBase64: '{TINY_PNG_B64}' }})",
        "LoadFrameAt": f"async () => ({{ width: 640, height: 360, "
                       f"pngBase64: '{TINY_PNG_B64}' }})",
        "BootstrapRoiTrainingRegions": "async () => { window.__calls.push('bootstrap'); return 'clipA_deadbeef'; }",
        "BootstrapRoiTrainingSampleEx": "async () => { window.__calls.push('bootstrap'); return 'clipA_deadbeef'; }",
        "ListRoiTrainingSamples": f"async () => {{ window.__calls.push('list'); return {json.dumps(SAMPLES)}; }}",
        "GetRoiTrainingSampleImage": f"async () => '{TINY_PNG_B64}'",
        "DiscardRoiTrainingSample": "async (dir, split, name) => { window.__calls.push(['discard', split, name]); }",
        "GetRoiDatasetSummary": f"async () => ({json.dumps(SUMMARY)})",
        "RunRoiModelTraining": "async (epochs, device) => { window.__calls.push(['train', epochs, device]); }",
        "GetSettings": "async () => ({ roiDatasetDir: '/data', defaultRoiDatasetDir: '/data' })",
        "CheckRoiTrainingAvailable": "async () => true",
        "CheckRoiTrainingStatus": "async () => ({ python: true, opencv: true, "
                                  "ultralytics: true, detail: 'Bereit' })",
        "InstallRoiTrainingDeps": "async () => {}",
        "UpdateRoiTrainingSample": "async () => {}",
        "ListRoiTrainingDevices": "async () => (["
            "{id:'auto',label:'Automatic (best available)',available:true},"
            "{id:'cuda',label:'NVIDIA CUDA',available:false},"
            "{id:'directml',label:'DirectML (Windows)',available:false},"
            "{id:'mps',label:'Apple MPS',available:false},"
            "{id:'cpu',label:'CPU (very slow)',available:true}])",
    }))
    harness = FRONTEND / "test" / "_roi_training_harness.html"
    harness.write_text(PAGE)

    with sync_playwright() as p:
        browser = p.chromium.launch()
        page = browser.new_page()
        page.on("dialog", lambda d: d.dismiss())
        page.on("pageerror", lambda e: print("   [pageerror]", e))
        page.goto(f"{base}/test/_roi_training_harness.html")
        page.wait_for_function("window.__ready === true")

        # getSettingsCache() löst asynchron auf - warten statt sofort zu prüfen,
        # sonst ist der Test von der Reihenfolge zweier Promise-Ticks abhängig.
        page.wait_for_function(
            "document.querySelector('#rt-dataset-dir').value === '/data'", timeout=5000)
        check("Datensatzordner wird aus den Einstellungen übernommen", True)

        check("Body map present", page.locator("#rt-body-figure .body-figure-svg").count() == 1)
        check("Nine legend body parts",
              page.locator(".body-figure-legend [data-class]").count() == 9)
        check("Pixel cue next to body map",
              page.locator("#rt-body-figure .body-figure-pixel-cue .pixel-figure-svg").count() == 1)

        page.click("#rt-pick-video")
        page.wait_for_function(
            "document.querySelector('#rt-video-path').textContent.includes('video.mp4')", timeout=5000)
        check("Video wird geladen", True)

        box = page.locator("#rt-canvas").bounding_box()

        # --- Region 1 markieren, ohne Klasse: Knopf bleibt gesperrt ----------
        page.mouse.move(box["x"] + 40, box["y"] + 40)
        page.mouse.down()
        page.mouse.move(box["x"] + 120, box["y"] + 120, steps=5)
        page.mouse.up()
        check("Ohne Klassennamen bleibt 'Use for training' gesperrt",
              page.locator("#rt-bootstrap").is_disabled())

        # Body map sets class without typing — then clear for the text-fill check.
        page.locator('.bf-zone[data-class="glans"]').first.click()
        page.wait_for_timeout(50)
        check("Body map click sets Class 1 to glans",
              page.locator("#rt-class1").input_value() == "glans")
        check("Body map unlocks Use for training",
              not page.locator("#rt-bootstrap").is_disabled())
        page.fill("#rt-class1", "")
        page.locator("#rt-class1").dispatch_event("input")
        check("Cleared class locks Use for training again",
              page.locator("#rt-bootstrap").is_disabled())

        page.fill("#rt-class1", "brust")
        page.locator("#rt-class1").dispatch_event("input")
        check("Mit Klassenname und 1 Region wird der Knopf frei",
              not page.locator("#rt-bootstrap").is_disabled())

        # --- 2. Region ohne deren Klassenname: wieder gesperrt ----------------
        page.click("#rt-mark-next")
        page.locator("#rt-canvas").scroll_into_view_if_needed()
        box = page.locator("#rt-canvas").bounding_box()
        page.mouse.move(box["x"] + 200, box["y"] + 40)
        page.mouse.down()
        page.mouse.move(box["x"] + 280, box["y"] + 120, steps=5)
        page.mouse.up()
        check("2. Region ohne deren Klasse sperrt den Knopf wieder",
              page.locator("#rt-bootstrap").is_disabled())

        page.fill("#rt-class2", "hand")
        page.locator("#rt-class2").dispatch_event("input")
        check("Mit beiden Klassennamen wieder frei",
              not page.locator("#rt-bootstrap").is_disabled())

        # --- Bootstrap anstoßen und per Event abschließen ---------------------
        page.click("#rt-bootstrap")
        page.wait_for_function(
            "window.__calls.includes('bootstrap')", timeout=5000)
        check("Knopf ist während des Laufs gesperrt", page.locator("#rt-bootstrap").is_disabled())

        page.evaluate("window.__triggerEvent('roitraining:bootstrap:progress', 'Frame 1/50')")
        check("Fortschrittszeile erscheint im Log",
              "Frame 1/50" in page.locator("#rt-bootstrap-log").inner_text())
        page.evaluate("window.__triggerEvent('roitraining:bootstrap:percent', 37)")
        check("Bootstrap-Balken sichtbar",
              page.locator("#rt-bootstrap-progress-wrap").evaluate("e => e.style.display") == "block")
        check("Bootstrap-Balken 37 %",
              page.locator("#rt-bootstrap-progress-bar").evaluate("e => e.style.width") == "37%",
              page.locator("#rt-bootstrap-progress-bar").evaluate("e => e.style.width"))

        page.evaluate("window.__triggerEvent('roitraining:bootstrap:done', { prefix: 'clipA_deadbeef' })")
        page.wait_for_function(
            "document.querySelector('#rt-bootstrap-progress-wrap').style.display === 'none'",
            timeout=5000)
        page.wait_for_function("window.__calls.includes('list')", timeout=5000)
        check("Knopf wird nach Abschluss wieder frei", not page.locator("#rt-bootstrap").is_disabled())

        # --- Kontrollansicht: Karten mit Bild + Box-Overlay --------------------
        page.wait_for_selector(".rt-card", timeout=5000)
        cards = page.locator(".rt-card")
        check("Kontrollansicht zeigt beide Beispiele", cards.count() == 2, str(cards.count()))
        page.wait_for_selector(".rt-box", timeout=5000)
        check("Box-Overlay wird gezeichnet", page.locator(".rt-box").count() >= 1)
        check("Klassenname steht auf dem Overlay",
              "brust" in page.locator(".rt-box-label").first.inner_text())

        # --- Verwerfen entfernt genau eine Karte --------------------------------
        page.locator(".rt-card button:text('Discard')").first.click()
        page.wait_for_function("window.__calls.some(c => Array.isArray(c) && c[0] === 'discard')", timeout=5000)
        page.wait_for_function("document.querySelectorAll('.rt-card').length === 1", timeout=5000)
        check("Verwerfen entfernt genau eine Karte", True)

        # --- Datensatz-Übersicht ------------------------------------------------
        page.click("#rt-summary-refresh")
        page.wait_for_selector("#rt-summary table", timeout=5000)
        check("Übersicht zeigt die Klasse 'brust'", "brust" in page.locator("#rt-summary").inner_text())
        check("Übersicht zeigt die Train-Anzahl", "2" in page.locator("#rt-summary").inner_text())

        # --- Training ------------------------------------------------------------
        # CheckRoiTrainingAvailable() löst asynchron auf - warten, bis der Knopf
        # freigegeben ist, statt gegen einen deaktivierten Knopf zu klicken.
        page.wait_for_function("!document.querySelector('#rt-train').disabled", timeout=5000)
        page.fill("#rt-epochs", "50")
        page.select_option("#rt-device", "cpu")
        page.click("#rt-train")
        page.wait_for_function(
            "window.__calls.some(c => Array.isArray(c) && c[0] === 'train')", timeout=5000)
        check("Training wird mit Epochen und Gerät aufgerufen",
              page.evaluate("window.__calls.find(c => Array.isArray(c) && c[0] === 'train')")
              == ["train", 50, "cpu"])
        check("Trainings-Knopf ist während des Laufs gesperrt", page.locator("#rt-train").is_disabled())

        page.evaluate("window.__triggerEvent('roitraining:train:progress', 'Epoch 1/50')")
        check("Trainings-Fortschrittszeile erscheint im Log",
              "Epoch 1/50" in page.locator("#rt-train-log").inner_text())

        page.evaluate("window.__triggerEvent('roitraining:train:done', { modelPath: '/models/roi.onnx' })")
        page.wait_for_function("!document.querySelector('#rt-train').disabled", timeout=5000)
        check("Status zeigt den Modellpfad nach Abschluss",
              "/models/roi.onnx" in page.locator("#rt-train-status").inner_text())

        browser.close()

    shutdown()

    # --- ohne ultralytics: Knopf bleibt gesperrt, Hinweis mit pip-Befehl -------
    base2, shutdown2 = serve(app_stub({
        "CheckRoiTrainingAvailable": "async () => false",
        "CheckRoiTrainingStatus": "async () => ({ python: true, opencv: true, "
                                  "ultralytics: false, detail: 'Bootstrap möglich; "
                                  "Training: pip install ultralytics onnx' })",
        "InstallRoiTrainingDeps": "async () => {}",
        "ListRoiTrainingDevices": "async () => []",
    }))
    with sync_playwright() as p:
        browser = p.chromium.launch()
        page = browser.new_page()
        page.on("dialog", lambda d: d.dismiss())
        page.goto(f"{base2}/test/_roi_training_harness.html")
        page.wait_for_function("window.__ready === true")
        page.wait_for_function(
            "document.querySelector('#rt-train-unavailable').style.display === 'block'", timeout=5000)
        check("Ohne ultralytics bleibt der Training-Knopf gesperrt",
              page.locator("#rt-train").is_disabled())
        hint = page.locator("#rt-train-unavailable").inner_text()
        check("Hinweis nennt pip/ultralytics",
              "ultralytics" in hint.lower() or "pip install" in hint.lower(), hint)
        browser.close()
    shutdown2()

    harness.unlink(missing_ok=True)

    return check.report()


if __name__ == "__main__":
    sys.exit(main())
