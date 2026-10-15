"""Settings → Play → Script folders (v0.8): list, add, remove.

Run: python3 cmd/gui-wails/frontend/test/settings_script_folders_test.py
"""

import pathlib
import sys

from playwright.sync_api import sync_playwright

from _harness import Checker, app_stub, serve

PAGE = """<!doctype html><html><body><div id="root"></div>
<script type="module">
  import { initSettings } from '/src/settings.js';
  initSettings(document.querySelector('#root'));
  window.__ready = true;
</script></body></html>"""


def main():
    check = Checker()
    base, shutdown = serve(app_stub({
        "GetSettings": "async () => ({updateCheckOnStartup:false,deviceConnectTest:false,logLevel:'info',logPath:'',reportPath:'',defaultReportPath:'',clearCacheOnExit:false})",
        "CurrentVersion": "async () => '0.0.0-test'",
        "GetCacheInfo": "async () => ({files:0, humanSize:'0 B', path:'/tmp', entries:0, bytes:0})",
        "GetRuntimeHealth": "async () => ({ok:true,deps:[],dirsCreated:[],resources:{}})",
        "CheckForUpdate": "async () => ({available:false})",
        "PluginsDir": "async () => '/tmp/plugins'",
        "ListInstalledPlugins": "async () => ([])",
        "QualityModelInfo": "async () => ({trained:false})",
        "GetScriptFolders": "async () => (window.__dirs = window.__dirs || ['/scripts/a']).slice()",
        "AddScriptFolder": "async () => (window.__dirs.push('/scripts/b <new>'), window.__dirs.slice())",
        "RemoveScriptFolder": "async d => (window.__dirs = window.__dirs.filter(x => x !== d), window.__dirs.slice())",
    }))
    harness = pathlib.Path(__file__).resolve().parent / "_settings_script_folders_harness.html"
    harness.write_text(PAGE)
    errors = []
    try:
        with sync_playwright() as p:
            b = p.chromium.launch()
            page = b.new_page()
            page.on("pageerror", lambda e: errors.append(str(e)))
            page.goto(f"{base}/test/_settings_script_folders_harness.html")
            page.wait_for_function("window.__ready === true")
            page.wait_for_function("document.querySelectorAll('#st-folder-list .st-folder-path').length === 1")
            check("the card sits in the Play section, before 'What you need'",
                  page.evaluate("""(() => { const c = document.querySelector('#st-script-folders');
                    return c && c.nextElementSibling && c.nextElementSibling.id === 'st-need-details'; })()"""))
            check("saved folder listed", page.text_content("#st-folder-list .st-folder-path") == "/scripts/a")
            page.click("#st-folder-add")
            page.wait_for_function("document.querySelectorAll('#st-folder-list .st-folder-path').length === 2")
            check("added folder listed (as text, not HTML)",
                  page.text_content("#st-folder-list li:nth-child(2) .st-folder-path") == "/scripts/b <new>")
            page.click("#st-folder-list li:nth-child(1) .st-folder-remove")
            page.wait_for_function("document.querySelectorAll('#st-folder-list .st-folder-path').length === 1")
            check("removed folder gone", page.text_content("#st-folder-list .st-folder-path") == "/scripts/b <new>")
            page.click("#st-folder-list .st-folder-remove")
            page.wait_for_function("!!document.querySelector('#st-folder-list .st-folder-empty')")
            check("empty list explains itself", "only the video" in page.text_content("#st-folder-list"))
            check("no page errors", not errors, "; ".join(errors[:5]))
            b.close()
    finally:
        shutdown()
        harness.unlink(missing_ok=True)
    return check.report()


if __name__ == "__main__":
    sys.exit(main())
