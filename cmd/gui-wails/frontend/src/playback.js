// TEMPORARY STUB — full module was repeatedly reduced to a placeholder on this branch.
// Restore from main (cmd/gui-wails/frontend/src/playback.js @ main) and re-apply:
//   1. import { applyHotkeyOMarker } from './ozone_ui.js';
//   2. window listener 'ozone:hotkey' → applyHotkeyOMarker(scriptPath, nowMs, oMarkers)
//   3. refresh listeners for ozone:suggested / polarity:inverted / ringdown:applied
//   4. keyboard hints: Taste O
// Local fixed copy is ready in the agent workspace; push failed due to payload size.
// See PR #50 / next operator commit.

import { getSettingsCache, saveSetting } from './settings.js';

export function initPlayback(root) {
  root.innerHTML = `
    <h2>Wiedergabe</h2>
    <p class="hint" style="color:#f66">
      playback.js auf diesem Branch ist ein Stub (Platzhalter-Push).
      Bitte die vollständige Datei von main wiederherstellen und die O-Zone-Hooks
      aus ozone_ui.js verdrahten (siehe Kommentar oben in der Quelldatei).
    </p>
    <div id="pb-log"></div>
  `;
  return {
    loadScriptPath: async () => {
      alert('playback.js Stub — bitte aus main wiederherstellen.');
    },
  };
}
