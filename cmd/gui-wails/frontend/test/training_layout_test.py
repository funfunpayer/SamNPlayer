"""V1 Training layout (training_layout.js, target visual-target/training.png):
Training nav · program + phases, stage, session bar, live curves, timeline ·
settings, mode, safety, feedback, recent sessions. Elements are moved, not
copied; plan, clock and steps come from real data only.

Run: python3 cmd/gui-wails/frontend/test/training_layout_test.py
"""

import pathlib
import sys

from playwright.sync_api import sync_playwright

from _harness import Checker, app_stub, serve

HERE = pathlib.Path(__file__).resolve().parent


def main():
    check = Checker()
    links = "".join(f'<link rel="stylesheet" href="/src/{css}?real" />'
                    for css in ("style.css", "theme_flow.css", "flow_fx.css", "training_rings.css"))
    html = (HERE.parent / "index.html").read_text().replace('<link rel="stylesheet" href="/src/style.css" />', links)
    harness = HERE / "_training_layout_harness.html"
    harness.write_text(html)
    base, shutdown = serve("window.__stopped = 0;\n" + app_stub({
        "GetEULA": "async () => ({version:'x', text:'', accepted:true})",
        "IsSuiteBuild": "async () => false",
        "CurrentVersion": "async () => 'test'",
        "GetSettings": "async () => ({updateCheckOnStartup:false})",
        "ListTrainingScripts": "async () => [{name:'plateau', description:'Plateau'}, {name:'my-own', description:'Mine', custom:true}]",
        "TrainingScriptPreview": "async () => ({totalMs: 600000, phaseMarkers: [{name:'Warm up', atMs:0}, {name:'Build', atMs:120000}, {name:'Peak', atMs:360000}], vibration: [{atMs:0, level:0.2}, {atMs:300000, level:0.8}, {atMs:600000, level:0.3}], suction: [{atMs:0, level:0.1}, {atMs:600000, level:0.5}]})",
        "TrainingHistory": "async () => []",
        "StartTraining": "async () => {}",
        "StopTraining": "async () => { window.__stopped++; window.__triggerEvent('training:done'); }",
        "PauseTraining": "async () => { window.__paused = (window.__paused || 0) + 1; }",
        "ResumeTraining": "async () => { window.__resumed = (window.__resumed || 0) + 1; }",
    }))
    try:
        with sync_playwright() as p:
            b = p.chromium.launch()
            page = b.new_page(viewport={"width": 1600, "height": 960})
            errors = []
            page.on("pageerror", lambda e: errors.append(str(e)))
            page.add_init_script("try{localStorage.clear(); localStorage.setItem('tr.stage', JSON.stringify({mode:'rings'}))}catch(e){}")
            page.goto(f"{base}/test/_training_layout_harness.html")
            page.wait_for_timeout(800)
            page.click('#rail .tab-btn[data-tab="training"]')
            page.wait_for_timeout(300)
            t = "#tab-training"
            check("layout v1 applied", page.get_attribute(t, "data-tr-layout") == "v1")
            check("three columns", page.evaluate(f"getComputedStyle(document.querySelector('{t}')).gridTemplateColumns.split(' ').length") == 3)
            check("program select in the program card", page.evaluate("!!document.querySelector('.trl-program #tr-script')"))
            check("start / interrupt / end in the session bar", page.evaluate("!!document.querySelector('.trl-session #tr-start') && !!document.querySelector('.trl-session #tr-stop')"))
            check("waves pinned under the session bar", page.evaluate("!!document.querySelector('.trl-curves #tr-waves-canvas')") and page.is_visible("#tr-waves-canvas"))
            check("history in Recent sessions", page.evaluate("!!document.querySelector('.trl-recent #tr-history')"))
            check("old two-column wrapper gone", page.query_selector(".tr-layout") is None)
            page.wait_for_function("document.querySelectorAll('#trl-programs .trl-nav-link').length >= 2", timeout=3000)
            check("built-in programs listed", "Custom" in page.text_content("#trl-programs"))
            check("my programs listed", "my-own" in page.text_content("#trl-my-programs"))
            check("Custom: no target, no phase colours", page.is_hidden(".trl-curves .tr-waves-target")
                  and page.evaluate("document.querySelectorAll('#trl-tl-phases [data-heat]').length") == 0)
            check("Custom active at start", page.evaluate("document.querySelector('#trl-programs .trl-nav-link.is-active')?.dataset.value") == "")

            steps = page.evaluate("document.querySelectorAll('#trl-stepper .trl-step').length")
            cycles = int(page.input_value("#tr-cycles"))
            check("Custom: one step per cycle", steps == cycles, f"{steps} vs {cycles}")
            page.eval_on_selector("#tr-cycles", "e => { e.value = '3'; e.dispatchEvent(new Event('input')); }")
            check("changing cycles re-plans", page.evaluate("document.querySelectorAll('#trl-stepper .trl-step').length") == 3)
            check("session mode shown for Custom", page.is_visible(".trl-mode"))
            page.click('.trl-mode-card[data-value="plateau"]')
            check("mode card sets the technique", page.input_value("#tr-technique") == "plateau")

            page.click('#trl-programs .trl-nav-link[data-value="plateau"]')
            page.wait_for_function("document.querySelectorAll('#trl-stepper .trl-step').length === 3 && document.querySelector('#trl-total').textContent === '10:00'", timeout=3000)
            check("script: phases and total from the preview", page.text_content("#trl-stepper").count("Warm up") == 1)
            check("timeline tiles with times", "2:00 – 6:00" in page.text_content("#trl-tl-phases"))
            check("session mode hidden for a script", page.is_hidden(".trl-mode"))
            check("program select follows the nav", page.input_value("#tr-script") == "plateau")
            check("script plan: phases coloured by planned strength",
                  page.evaluate("[...document.querySelectorAll('#trl-tl-phases .trl-tl-phase')].every(t => t.dataset.heat === '1' && t.style.getPropertyValue('--seg'))"))
            check("script plan: a small planned curve per phase",
                  page.evaluate("document.querySelectorAll('#trl-tl-phases svg.trl-tl-mini path').length") == 6)
            check("live curves show the planned target", page.is_visible(".trl-curves .tr-waves-target"))

            check("transport: End · Start · Interrupt, Start in the middle",
                  page.evaluate("[...document.querySelectorAll('.trl-transport button')].filter(b => b.offsetParent).map(b => b.id).join(',')")
                  == "tr-stop,tr-start,tr-pause")
            check("Start is the app-symbol play button", page.evaluate("document.querySelector('#tr-start').classList.contains('flow-play')"))
            page.click("#tr-start")
            page.wait_for_function("document.querySelector('#tab-training').dataset.trRunning === '1'", timeout=3000)
            check("running: the big button pauses", page.is_visible("#trl-play") and page.is_hidden("#tr-start")
                  and page.get_attribute("#trl-play", "aria-label") == "Pause training")
            page.wait_for_function("document.querySelector('#trl-elapsed').textContent !== '0:00'", timeout=4000)
            check("clock runs while training", True)
            check("first phase is current", page.text_content("#trl-phase-name") == "Warm up"
                  and page.evaluate("document.querySelector('#trl-stepper .trl-step.is-current')?.dataset.index") == "0")
            # Read both clocks in one step: under CI load the elapsed time can be several seconds.
            el, left = page.evaluate("[document.querySelector('#trl-elapsed').textContent, document.querySelector('#trl-left').textContent]")
            secs = lambda t: int(t.split(":")[0]) * 60 + int(t.split(":")[1])
            check("time left counts down", 0 < secs(el) and abs(secs(left) - (600 - secs(el))) <= 1, f"{el} / {left}")
            page.click("#trl-play")
            page.wait_for_function("document.querySelector('#tab-training').dataset.trPaused === '1'", timeout=3000)
            frozen = page.text_content("#trl-elapsed")
            page.wait_for_timeout(1300)
            check("pause: PauseTraining called, clock stands still",
                  page.evaluate("window.__paused") == 1 and page.text_content("#trl-elapsed") == frozen, frozen)
            check("pause: shown in the phase line", page.text_content("#trl-phase-step").startswith("Paused"))
            check("pause: the big button resumes", page.get_attribute("#trl-play", "aria-label") == "Resume training")
            page.click("#trl-play")
            page.wait_for_function("document.querySelector('#tab-training').dataset.trPaused === ''", timeout=3000)
            page.wait_for_function(f"document.querySelector('#trl-elapsed').textContent !== '{frozen}'", timeout=4000)
            check("resume: ResumeTraining called, clock runs on", page.evaluate("window.__resumed") == 1)
            page.click("#trl-stop")
            page.wait_for_function("window.__stopped === 1", timeout=3000)
            check("Stop session ends the training", True)
            check("no page errors", not errors, "; ".join(errors))
            b.close()
    finally:
        shutdown()
        harness.unlink(missing_ok=True)
    return check.report()


if __name__ == "__main__":
    sys.exit(main())
