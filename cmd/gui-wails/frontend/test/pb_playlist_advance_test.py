"""Regression: Video-Ende darf Film-Liste weiter schalten (nicht wie Stop).

Der Stop-Knopf setzt userStopRequested und blockiert Auto-Next bewusst.
Natürliches video-'ended' muss stop({ user: false }) nutzen, sonst bleibt
die Liste beim ersten Film stehen.

Ausführen: python3 cmd/gui-wails/frontend/test/pb_playlist_advance_test.py
"""

import pathlib
import sys

from _harness import Checker


def main():
    check = Checker()
    src = (pathlib.Path(__file__).resolve().parent.parent / "src" / "playback.js").read_text()

    check(
        "stop() unterscheidet User-Stop von natürlichem Ende",
        "async function stop({ user = true } = {})" in src,
    )
    check(
        "Video-ended ruft stop({ user: false })",
        "stop({ user: false })" in src and "addEventListener('ended'" in src,
    )
    check(
        "onPlaybackFinished ehrt userStopRequested",
        "if (userStopRequested)" in src and "playNextInPlaylist" in src,
    )
    # Stop-Knopf bleibt Default (user=true) — addEventListener übergibt Event,
    # destructuring findet kein .user → Default true.
    check(
        "Stop-Knopf gebunden",
        "el('#pb-stop').addEventListener('click', stop)" in src,
    )
    check(
        "Connect-Fehler: failed-Payload blockiert Advance",
        "payload.failed" in src and "onPlaybackFinished(payload" in src,
    )
    check(
        "Aktiven Playlist-Eintrag entfernen lädt nach",
        "removingCurrent" in src and "loadScript(playlist[playlistIndex].path" in src,
    )

    return check.report()


if __name__ == "__main__":
    sys.exit(main())
