import './style.css';
import { CurrentVersion, CheckForUpdate, ApplyUpdate, GetSettings } from '../wailsjs/go/main/App';
import { initPlayback } from './playback.js';
import { initTraining } from './training.js';
import { initGenerator } from './generator.js';
import { EventsOn } from '../wailsjs/runtime/runtime';
import { initDevice } from './device.js';
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
initDevice(document.getElementById('tab-device'));
initSettings(document.getElementById('tab-settings'));
initSidebar(document.getElementById('sidebar'));
enhanceGeneratorPreview(document.getElementById('tab-generator'));
enhancePlaybackOZone(document.getElementById('tab-playback'));
initPostGenerateReview();

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
      console.warn('Update-Prüfung beim Start fehlgeschlagen:', res.error);
      return;
    }
    if (!res.available) return;
    const tag = res.release ? res.release.tag_name : '?';
    if (confirm(`Version ${tag} ist verfügbar (aktuell: ${currentVersion}).\n\nJetzt herunterladen und neu starten?`)) {
      ApplyUpdate().catch(err => alert('Update fehlgeschlagen: ' + err));
    }
  }).catch(err => console.warn('Update-Prüfung beim Start fehlgeschlagen:', err));
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
    window.dispatchEvent(new CustomEvent('drop:video', { detail: data.videos[0] }));
    if (data.videos.length > 1) {
      window.dispatchEvent(new CustomEvent('drop:videos', { detail: data.videos }));
    }
    return;
  }
  if (data.scripts && data.scripts.length) {
    switchTab('playback');
    window.dispatchEvent(new CustomEvent('drop:script', { detail: data.scripts[0] }));
    return;
  }
  if (data.ignored) {
    alert('Damit kann ich nichts anfangen. Zieh eine Videodatei oder eine '
        + '.funscript-Datei ins Fenster.');
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
