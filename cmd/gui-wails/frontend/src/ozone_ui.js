import { ApplySuggestedOZone, SuggestPolarity, InvertLoadedScript, SaveOMarkers, GetOMarkers, ApplyRingDown } from '../wailsjs/go/main/App';

export function enhancePlaybackOZone(root) {
  const oRow = root.querySelector('#pb-omarker-add-row');
  if (!oRow || root.querySelector('#pb-ozone-suggest')) return;

  const suggestBtn = document.createElement('button');
  suggestBtn.id = 'pb-ozone-suggest';
  suggestBtn.type = 'button';
  suggestBtn.textContent = 'O-Zone vorschlagen';
  oRow.appendChild(suggestBtn);

  const invertBtn = document.createElement('button');
  invertBtn.id = 'pb-polarity-invert';
  invertBtn.type = 'button';
  invertBtn.textContent = 'Richtung prüfen / umkehren';
  oRow.appendChild(invertBtn);

  const ringBtn = document.createElement('button');
  ringBtn.id = 'pb-ringdown';
  ringBtn.type = 'button';
  ringBtn.textContent = 'Ring-down anwenden';
  ringBtn.title = 'Gedämpfte Halbzyklen nach aktueller Position anhängen (Skript speichern)';
  oRow.appendChild(ringBtn);

  const status = document.createElement('span');
  status.id = 'pb-ozone-status';
  status.className = 'hint';
  status.style.margin = '0';
  oRow.appendChild(status);

  suggestBtn.addEventListener('click', async () => {
    try {
      const zone = await ApplySuggestedOZone();
      if (!zone || !zone.ok) {
        status.textContent = zone && zone.reason ? zone.reason : 'kein Vorschlag';
        return;
      }
      status.textContent = zone.reason;
      window.dispatchEvent(new CustomEvent('ozone:suggested', { detail: zone }));
    } catch (err) {
      status.textContent = String(err);
    }
  });

  invertBtn.addEventListener('click', async () => {
    try {
      const hint = await SuggestPolarity();
      const msg = (hint && hint.reason) || '';
      if (hint && hint.suggestInvert) {
        if (confirm(msg + '\n\nPositionen jetzt spiegeln (100 − pos)?')) {
          await InvertLoadedScript();
          status.textContent = 'Richtung umgekehrt und gespeichert.';
          window.dispatchEvent(new CustomEvent('polarity:inverted'));
        } else {
          status.textContent = 'unveraendert. ' + msg;
        }
      } else {
        status.textContent = msg;
      }
    } catch (err) {
      status.textContent = String(err);
    }
  });

  ringBtn.addEventListener('click', async () => {
    try {
      const video = root.querySelector('#pb-video');
      let atMs = 0;
      if (video && Number.isFinite(video.currentTime)) {
        atMs = Math.round(video.currentTime * 1000);
      }
      if (!confirm('Ring-down nach t=' + atMs + ' ms anhängen und Skript speichern? (2 Zyklen, endet bei 0)')) {
        status.textContent = 'Ring-down abgebrochen.';
        return;
      }
      await ApplyRingDown(atMs, 2);
      status.textContent = 'Ring-down gespeichert (nach ' + atMs + ' ms).';
      window.dispatchEvent(new CustomEvent('script:ringdown', { detail: { atMs } }));
    } catch (err) {
      status.textContent = String(err);
    }
  });

  document.addEventListener('keydown', (e) => {
    const tag = (e.target.tagName || '').toLowerCase();
    if (tag === 'input' || tag === 'select' || tag === 'textarea') return;
    if (e.key !== 'o' && e.key !== 'O') return;
    e.preventDefault();
    const video = root.querySelector('#pb-video');
    const nowMs = video && !video.paused ? Math.round(video.currentTime * 1000) : 0;
    window.dispatchEvent(new CustomEvent('ozone:hotkey', { detail: { nowMs } }));
  });
}

export async function applyHotkeyOMarker(scriptPath, nowMs, existing) {
  if (!scriptPath || nowMs < 0) return existing || [];
  const markers = Array.isArray(existing) ? existing.slice() : await GetOMarkers(scriptPath);
  const start = Math.max(0, nowMs);
  const end = start + 4000;
  markers.push({ startMs: start, endMs: end, kind: 'primary', intensity: 1 });
  await SaveOMarkers(scriptPath, markers);
  return markers;
}
