"""Bench tab Clip-Prep panel (discoverability).

Run: python3 cmd/gui-wails/frontend/test/benchmark_clip_prep_test.py
"""

import pathlib
import sys

from playwright.sync_api import sync_playwright

from _harness import Checker, app_stub, serve

PAGE = """<!doctype html><html><body><div id="root"></div>
<script type="module">
  import { initBenchmark } from '/src/benchmark.js';
  initBenchmark(document.querySelector('#root'));
  window.__ready = true;
</script></body></html>"""


def main():
    check = Checker()
    base, shutdown = serve(app_stub({
        "GetSettings": "async () => ({ benchmarkManifestPath: '' })",
        "GetBenchmarkHistory": "async () => []",
        "PickBenchmarkManifest": "async () => ''",
        "PickFunscriptFile": "async () => ''",
        "PickVideoFile": "async () => ''",
        "ScoreScriptPair": "async () => ({})",
        "AppendBenchmarkPairLabel": "async () => ''",
        "RunGoldenClipBenchmark": "async () => {}",
        "SetSetting": "async () => {}",
    }))
    harness = pathlib.Path(__file__).resolve().parent / "_benchmark_clip_prep_harness.html"
    harness.write_text(PAGE)

    with sync_playwright() as p:
        browser = p.chromium.launch()
        page = browser.new_page()
        page.goto(f"{base}/test/_benchmark_clip_prep_harness.html")
        page.wait_for_function("window.__ready === true")

        check("Clip-Prep details present",
              page.locator("#bm-clip-prep").count() == 1)
        check("Copy command button present",
              page.locator("#bm-clip-prep-copy").count() == 1)
        check("Single-clip command mentions cut_clip",
              "cut_clip" in (page.locator("#bm-clip-prep-cmd").inner_text() or ""))
        check("Compare scripts heading still present",
              page.locator("text=Compare scripts").count() >= 1)
        check("Score vs reference button present",
              page.locator("#bm-score").count() == 1)

        page.click("#bm-clip-prep-batch")
        page.wait_for_timeout(30)
        check("Batch command shows marks.json",
              "marks.json" in (page.locator("#bm-clip-prep-cmd").inner_text() or ""))

        browser.close()
    shutdown()
    harness.unlink(missing_ok=True)
    return check.report()


if __name__ == "__main__":
    raise SystemExit(main())
