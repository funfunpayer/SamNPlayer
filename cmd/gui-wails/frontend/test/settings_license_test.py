"""Settings License section: English status + import/clear wiring.

Run: python3 cmd/gui-wails/frontend/test/settings_license_test.py
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
        "GetSettings": "async () => ({updateCheckOnStartup:false,deviceConnectTest:false,logLevel:'info',logPath:'',reportPath:'',defaultReportPath:'',clearCacheOnExit:false,aiRoiModelPath:'',aiPreferredClasses:'',aiBaseUrl:''})",
        "GetLicenseStatus": """async () => (window.__lic || {
          present:false, licensed:false, effective:true, enforcement:false,
          state:'none',
          message:'No license imported. Enforcement is off — full features available.'
        })""",
        "ImportLicenseText": """async (t) => {
          window.__imported = t;
          window.__lic = {
            present:true, licensed:true, effective:true, enforcement:false,
            state:'internal', sub:'owner', tier:'internal', validUntil:'never',
            message:'Internal license for owner (no expiry). Enforcement is off.'
          };
          return window.__lic;
        }""",
        "ImportLicenseFile": """async () => {
          window.__lic = {
            present:true, licensed:true, effective:true, enforcement:false,
            state:'invite', sub:'friend', tier:'invite', validUntil:'never',
            message:'Invite license for friend (no expiry). Enforcement is off.'
          };
          return window.__lic;
        }""",
        "ClearLicense": """async () => {
          window.__lic = {
            present:false, licensed:false, effective:true, enforcement:false,
            state:'none',
            message:'No license imported. Enforcement is off — full features available.'
          };
          return window.__lic;
        }""",
        "InstallPluginPack": """async () => {
          window.__packs = [{
            root:'/tmp/plugins/sample_pack',
            manifest:{id:'sample_pack', name:'Sample Pack', version:'0.1.0'}
          }];
          return window.__packs[0];
        }""",
        "OpenPluginsFolder": "async () => {}",
        "PluginsDir": "async () => '/tmp/plugins'",
        "ListInstalledPlugins": """async () => (window.__packs || [{
          root:'/tmp/plugins/sample_pack',
          manifest:{id:'sample_pack', name:'Sample Pack', version:'0.1.0'}
        }])""",
        "SetSetting": "async () => {}",
        "GetRuntimeHealth": "async () => ({ok:true,deps:[],dirsCreated:[],resources:{}})",
        "EnsureVideoTools": "async () => {}",
        "CurrentVersion": "async () => '0.5.9-dev'",
        "CheckForUpdate": "async () => ({available:false})",
        "GetHardwareInfo": "async () => 'ok'",
        "GetCacheInfo": "async () => ({path:'',entries:0,bytes:0})",
        "ReportExists": "async () => false",
        "ReportSummary": "async () => ''",
        "QualityModelInfo": "async () => ({trained:false})",
        "CheckAIRoiAvailable": "async () => false",
        "CheckAIServerAvailable": "async () => ({ok:false,baseUrl:'',message:''})",
        "OpenLogFolder": "async () => {}",
        "PickReportPath": "async () => ''",
        "TrainQualityModel": "async () => {}",
        "ClearCache": "async () => ({path:'',entries:0,bytes:0})",
        "ApplyUpdate": "async () => {}",
    }))

    harness = pathlib.Path(__file__).resolve().parent / "_settings_license_harness.html"
    harness.write_text(PAGE)

    with sync_playwright() as p:
        browser = p.chromium.launch()
        page = browser.new_page()
        page.on("pageerror", lambda e: print("   [pageerror]", e))
        page.goto(f"{base}/test/_settings_license_harness.html")
        page.wait_for_function("window.__ready === true")

        check("License heading visible", page.locator("h3", has_text="License").count() == 1)
        # Status fills async via GetLicenseStatus
        page.wait_for_function(
            "document.querySelector('#st-license-status').textContent.includes('enforcement')",
            timeout=5000)
        text = page.locator("#st-license-status").inner_text()
        check("status mentions enforcement off",
              "enforcement: off" in text or "Enforcement is off" in text)

        page.fill("#st-license-paste", "SNP1.test.token")
        page.click("#st-license-import-text")
        page.wait_for_function(
            "document.querySelector('#st-license-status').textContent.toLowerCase().includes('internal') || document.querySelector('#st-license-status').textContent.includes('owner')",
            timeout=5000)
        text2 = page.locator("#st-license-status").inner_text()
        check("after import shows internal/owner",
              "internal" in text2.lower() or "owner" in text2)

        page.click("#st-license-clear")
        page.wait_for_function(
            "document.querySelector('#st-license-status').textContent.toLowerCase().includes('enforcement') && "
            "(document.querySelector('#st-license-status').textContent.toLowerCase().includes('off') || "
            "document.querySelector('#st-license-status').textContent.includes('No license'))",
            timeout=5000)
        text3 = page.locator("#st-license-status").inner_text()
        check("after clear shows none/off",
              "enforcement" in text3.lower() and ("off" in text3.lower() or "no license" in text3.lower()))

        check("Plugins details present",
              page.locator("#st-plugins-details").count() == 1)
        page.evaluate(
            "document.querySelector('#st-plugins-details').open = true")
        page.wait_for_function(
            "document.querySelector('#st-plugins-status') && document.querySelector('#st-plugins-status').textContent.includes('Sample Pack')",
            timeout=5000)
        page.click("#st-plugins-install")
        page.wait_for_function(
            "document.querySelector('#st-plugins-status').textContent.includes('Sample Pack')",
            timeout=5000)

        browser.close()

    shutdown()
    harness.unlink(missing_ok=True)
    return check.report()


if __name__ == "__main__":
    sys.exit(main())
