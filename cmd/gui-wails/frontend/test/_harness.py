"""Gemeinsame Hilfsfunktionen für die Frontend-Tests.

Der Grund für diese Datei: die Tests laden das echte JavaScript und ersetzen
nur die wailsjs-Bindings durch Stubs. Diese Stub-Listen wurden bisher in
jedem Test von Hand gepflegt - und veralteten damit bei JEDER neuen
Go-Methode. Der Test schlug dann mit "does not provide an export named ..."
fehl, also aus einem Grund, der mit dem Prüfgegenstand nichts zu tun hat.
Zweimal ist genau das passiert.

Hier wird die Stub-Liste stattdessen aus der tatsächlich erzeugten
App.js gelesen. Neue Bindings sind damit automatisch enthalten.
"""

import functools
import http.server
import pathlib
import re
import threading

FRONTEND = pathlib.Path(__file__).resolve().parent

RUNTIME_STUB = """
const handlers = {};
export function EventsOn(name, cb) { (handlers[name] ||= []).push(cb); }
export function EventsEmit() {}
window.__triggerEvent = (name, data) => (handlers[name] || []).forEach(cb => cb(data));
"""


def app_stub(overrides=None):
    """Erzeugt einen Stub für ALLE echten Bindings.

    overrides bildet Bindingnamen auf JavaScript-Ausdrücke ab - für die
    Aufrufe, deren Verhalten der Test tatsächlich braucht. Alles andere wird
    zu einer harmlosen Funktion, die ein leeres Objekt liefert.
    """
    bindings = FRONTEND.parent / "wailsjs" / "go" / "main" / "App.js"
    names = sorted(set(re.findall(r"export function (\w+)", bindings.read_text())))
    if not names:
        raise RuntimeError(
            f"Keine Bindings in {bindings} gefunden - wurde 'wails build' schon ausgeführt?")

    overrides = overrides or {}
    lines = ["window.__calls = window.__calls || [];"]
    for name in names:
        lines.append(f"export const {name} = {overrides.get(name, 'async () => ({})')};")
    # Overrides für Namen, die (noch) nicht in den Bindings stehen, trotzdem
    # aufnehmen - sonst schlägt ein Test still fehl, weil sein Stub fehlt.
    for name, body in overrides.items():
        if name not in names:
            lines.append(f"export const {name} = {body};")
    return "\n".join(lines)


def make_handler(app_stub_code):
    class Handler(http.server.SimpleHTTPRequestHandler):
        STUBS = {
            "/wailsjs/go/main/App": app_stub_code,
            "/wailsjs/runtime/runtime": RUNTIME_STUB,
            # main.js macht "import './style.css'" - das ist eine Vite-
            # Eigenheit, die ein Browser nicht kennt. Im Test als leeres
            # Modul ausliefern, sonst scheitert der gesamte Import und der
            # Test schlägt aus einem Grund fehl, der nichts mit dem
            # Prüfgegenstand zu tun hat.
            "/src/style.css": "export default '';",
        }

        def do_GET(self):
            stub = self.STUBS.get(self.path)
            if stub is not None:
                body = stub.encode()
                self.send_response(200)
                self.send_header("Content-Type", "text/javascript")
                self.send_header("Content-Length", str(len(body)))
                self.end_headers()
                self.wfile.write(body)
                return
            super().do_GET()

        def log_message(self, *args):
            pass

    return Handler


def serve(app_stub_code):
    """Startet den Testserver und liefert (basis_url, herunterfahren)."""
    server = http.server.ThreadingHTTPServer(
        ("127.0.0.1", 0), functools.partial(make_handler(app_stub_code),
                                            directory=str(FRONTEND.parent))
    )
    threading.Thread(target=server.serve_forever, daemon=True).start()
    return f"http://127.0.0.1:{server.server_address[1]}", server.shutdown


class Checker:
    """Sammelt Prüfergebnisse und gibt sie einheitlich aus."""

    def __init__(self):
        self.failures = []

    def __call__(self, name, condition, detail=""):
        print(("  OK   " if condition else "  FAIL ") + name
              + (f"  [{detail}]" if not condition else ""))
        if not condition:
            self.failures.append(name)

    def report(self):
        print(("FEHLGESCHLAGEN: " + ", ".join(self.failures)) if self.failures
              else "Alle Prüfungen bestanden.")
        return 1 if self.failures else 0
