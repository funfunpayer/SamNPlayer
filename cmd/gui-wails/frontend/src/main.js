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

CurrentVersion().then(v => {
  document.getElementById('version-label').textContent = v === 'dev' ? 'dev' : v;
});

GetSettings().then(s => {
  if (!s.updateCheckOnStartup) return;
  CheckForUpdate().then(res => {
    if (res.error || !res.available) return;
    const tag = res.release ? res.release.tag_name : '?';
    if (confirm(`Version ${tag} ist verfügbar (aktuell: ${'dev'}).\n\nJetzt herunterladen und neu starten?`)) {
      ApplyUpdate().catch(err => alert('Update fehlgeschlagen: ' + err));
    }
  }).catch(() => {});
});

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
  dragDepth = Math.max(0, dragDepth - 1);
  if (dragDepth === 0) overlay.classList.remove('visible');
});
window.addEventListener('drop', () => {
  dragDepth = 0;
  overlay.classList.remove('visible');
});
