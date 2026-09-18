import './style.css';
import './help.js';
import { uiError, uiInfo, uiWarn } from './notify.js';
import { CurrentVersion, CheckForUpdate, ApplyUpdate, GetSettings, ConnectDeviceVia, DisconnectDevice } from '../wailsjs/go/main/App';
import { initPlayback } from './playback.js';
import { initTraining } from './training.js';
import { initGenerator } from './generator.js';
import { initBenchmark } from './benchmark.js';
import { initRoiTraining } from './roi_training.js';
import { EventsOn } from '../wailsjs/runtime/runtime';
import { initDevice } from './device.js';
import { initLog } from './log.js';
import { initSettings } from './settings.js';
import { initSidebar } from './sidebar.js';
import { enhanceGeneratorPreview } from './roi_help.js';
import { enhancePlaybackOZone } from './ozone_ui.js';
import { initPostGenerateReview } from './postgen.js';

function switchTab(name) {
  document.querySelectorAll('.tab-btn').forEach(b => b.classList.toggle('active', b.dataset.tab === name));
  document.querySelectorAll('.tab-panel').forEach(p => p.classList.toggle('active', p.id === 'tab-' + name));
}

document.querySelectorAll('.tab-btn').forEach(btn => {
  btn.addEventListener('click', () => switchTab(btn.dataset.tab));
});

const playback = initPlayback(document.getElementById('tab-playback'));
initTraining(document.getElementById('tab-training'));
initGenerator(document.getElementById('tab-generator'), playback);
initBenchmark(document.getElementById('tab-benchmark'));
initRoiTraining(document.getElementById('tab-roi-training'));
initDevice(document.getElementById('tab-device'));
initLog(document.getElementById('tab-log'));
initSettings(document.getElementById('tab-settings'));
initSidebar(document.getElementById('sidebar'));
enhanceGeneratorPreview(document.getElementById('tab-generator'));
enhancePlaybackOZone(document.getElementById('tab-playback'));
initPostGenerateReview();

// Topbar: Geräte-Status + Verbinden/Trennen (Transport bleibt im Gerät-Tab).
(function initTopbarDevice() {
  const wrap = document.getElementById('topbar-device-wrap');
  const btn = document.getElementById('topbar-device');
  const nameEl = document.getElementById('topbar-device-name');
  const action = document.getElementById('topbar-device-action');
  if (!wrap || !btn || !nameEl || !action) return;

  let connected = false;
  let busy = false;

  function render(st) {
    connected = !!(st && st.connected);
    wrap.classList.remove('is-online', 'is-offline', 'is-searching');
    if (busy && !connected) {
      wrap.classList.add('is-searching');
      nameEl.textContent = 'Suche…';
      action.textContent = '…';
      action.disabled = true;
      btn.title = 'Verbindung läuft…';
      return;
    }
    if (connected) {
      wrap.classList.add('is-online');
      const label = st.mock
        ? 'Mock-Gerät'
        : (st.name && String(st.name).trim()) || 'Verbunden';
      nameEl.textContent = label;
      action.textContent = 'Trennen';
      action.disabled = busy;
      btn.title = st.address
        ? `${label} · ${st.address} — Details im Gerätetab`
        : `${label} — Details im Gerätetab`;
    } else {
      wrap.classList.add('is-offline');
      nameEl.textContent = 'Nicht verbunden';
      action.textContent = 'Verbinden';
      action.disabled = busy;
      btn.title = 'Details / Verbindungsart im Gerätetab';
    }
  }

  window.addEventListener('device:status', e => {
    // Während eigener Topbar-Verbindung kein Fremd-Status überschreiben,
    // außer das Ergebnis ist bereits connected (Erfolg vom Gerät-Tab).
    if (busy && !(e.detail && e.detail.connected)) return;
    if (e.detail && e.detail.connected) busy = false;
    render(e.detail || {});
  });

  btn.addEventListener('click', () => switchTab('device'));

  action.addEventListener('click', async () => {
    if (busy) return;
    if (connected) {
      busy = true;
      action.disabled = true;
      try {
        const st = await DisconnectDevice();
        busy = false;
        render(st);
        window.dispatchEvent(new CustomEvent('device:status', { detail: st }));
      } catch (err) {
        busy = false;
        uiError('Trennen fehlgeschlagen: ' + err);
        render({ connected: true });
      }
      return;
    }
    busy = true;
    render({ connected: false });
    try {
      const s = await GetSettings();
      const transport = (s && s.deviceTransport) || 'ble';
      const url = (s && s.intifaceUrl) || '';
      const st = await ConnectDeviceVia(transport, url);
      busy = false;
      render(st);
      window.dispatchEvent(new CustomEvent('device:status', { detail: st }));
    } catch (err) {
      busy = false;
      render({ connected: false });
      uiError('Verbindung fehlgeschlagen: ' + err + ' — Verbindungsart (BLE / Intiface / Mock) im Gerät-Tab wählen.');
    }
  });

  render({ connected: false });
})();

let currentVersion = 'dev';
CurrentVersion().then(v => {
  currentVersion = v;
  document.getElementById('version-label').textContent = v === 'dev' ? 'dev' : v;
});

// Stiller Update-Check beim Start - respektiert die Einstellung, die
// settings.js beim Laden aus GetSettings() liest; hier zusätzlich einmal
// direkt geprüft, damit main.js nicht auf settings.js warten muss.
//
// "Still" heißt: kein Dialog bei "kein Update"/Fehler, nicht "Fehler
// verschwinden lassen" - ein CheckForUpdate()-Fehlschlag wurde hier bisher
// komplett verschluckt (leeres .catch), nicht mal geloggt. Von außen war
// "kein Update gefunden, weil es keins gibt" nicht von "die Prüfung ist
// stillschweigend gescheitert" zu unterscheiden - dafür gibt es jetzt den
// Knopf "Jetzt nach Updates suchen" in den Einstellungen (settings.js), der
// das Ergebnis (auch einen Fehler) explizit anzeigt.
GetSettings().then(s => {
  if (!s.updateCheckOnStartup) return;
  CheckForUpdate().then(res => {
    if (res.error) {
      uiWarn('Update-Prüfung beim Start: ' + res.error);
      return;
    }
    if (!res.available) return;
    const tag = res.release ? res.release.tag_name : '?';
    // Update-Installation ist bewusst bestätigt (wie Überschreiben beim Erzeugen).
    if (confirm(`Version ${tag} ist verfügbar (aktuell: ${currentVersion}).\n\nJetzt herunterladen und neu starten?`)) {
      ApplyUpdate().catch(err => uiError('Update fehlgeschlagen: ' + err));
    }
  }).catch(err => uiWarn('Update-Prüfung beim Start: ' + err));
});


// --- Dateien per Drag & Drop -------------------------------------------
//
// Go liefert die echten Dateipfade (siehe registerFileDrop in app.go) - die
// Webview allein bekäme aus einem Drop nur einen Blob ohne Pfad, mit dem
// der Generator nichts anfangen kann.
//
// Die fallengelassene Datei bestimmt den Tab: ein Video gehört in den
// Generator, ein Skript in die Wiedergabe. Das erspart es, vorher den
// richtigen Tab zu suchen.
EventsOn('files:dropped', data => {
  const overlay = document.getElementById('drop-overlay');
  if (overlay) overlay.classList.remove('visible');

  if (data.videos && data.videos.length) {
    switchTab('generator');
    // Nur das erste Video wird geladen - Stapelverarbeitung mehrerer Videos
    // gibt es noch nicht. Der Generator zeigt das explizit an (extraCount),
    // statt die übrigen Dateien einfach stillschweigend zu verwerfen.
    window.dispatchEvent(new CustomEvent('drop:video', {
      detail: { path: data.videos[0], extraCount: data.videos.length - 1 },
    }));
    return;
  }
  if (data.scripts && data.scripts.length) {
    switchTab('playback');
    // Nur das erste Skript wird geladen - dieselbe Stapelverarbeitungslücke
    // wie beim Video-Drop oben, hier für Skripte. Zeigt es sichtbar an,
    // statt weitere abgelegte Dateien stillschweigend zu verwerfen.
    window.dispatchEvent(new CustomEvent('drop:script', {
      detail: { path: data.scripts[0], extraCount: data.scripts.length - 1 },
    }));
    return;
  }
  if (data.ignored) {
    uiWarn('Drop ignoriert — bitte eine Videodatei oder .funscript ablegen.');
  }
});

// Sichtbare Rückmeldung, solange etwas über dem Fenster schwebt.
const overlay = document.createElement('div');
overlay.id = 'drop-overlay';
overlay.innerHTML = '<div>Video oder .funscript hier ablegen</div>';
document.body.appendChild(overlay);

let dragDepth = 0;
window.addEventListener('dragenter', e => {
  e.preventDefault();
  dragDepth++;
  overlay.classList.add('visible');
});
window.addEventListener('dragover', e => e.preventDefault());
window.addEventListener('dragleave', () => {
  // dragleave feuert auch beim Wechsel zwischen Kindelementen - deshalb
  // zählen statt einfach auszublenden, sonst flackert die Anzeige.
  dragDepth = Math.max(0, dragDepth - 1);
  if (dragDepth === 0) overlay.classList.remove('visible');
});
window.addEventListener('drop', () => {
  dragDepth = 0;
  overlay.classList.remove('visible');
});
