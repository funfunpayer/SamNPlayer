// Generator "How it feels" follows the device (Owner 14 Oct):
//   For device — the connected device (auto), Sam Neo 2, a vibrator / other
//   toy, or a stroker. Only what the device can use is shown; "Show all
//   settings" brings the rest back.
//   Sam Neo 2 feel — Scene (auto from the scene detection or the contact
//   mark, Classic = the old mapping, Oral / Intercourse / Breast / Hand),
//   Mix (vibration ↔ suction) and "Vibration follows suction". Applied to the
//   new script right after Create (SaveFeelSettings); Classic + centre Mix +
//   follow off leaves the script exactly as before.
import { GetDeviceStatus } from '../wailsjs/go/main/App';

const KEY = 'gen.feel';
const SCENES = { oral: 'Oral', intercourse: 'Intercourse', breast: 'Breast', hand: 'Hand' };

// Device kind from a device status: neo2 (vibration + suction), stroker
// (linear stroke), vibrator (vibration and/or suction, no stroke); null = none.
export function deviceKind(status) {
  const feats = status && status.connected && status.capabilities && Array.isArray(status.capabilities.features)
    ? status.capabilities.features : null;
  if (!feats || !feats.length) return null;
  const has = re => feats.some(f => re.test(String(f.channel || '')));
  if (has(/^stroke$/) && !has(/^suction$/)) return 'stroker';
  if (has(/^vibration\d*$/) && has(/^suction$/)) return 'neo2';
  return 'vibrator';
}

// Scene feel from an AI scene type or the contact mark's class.
export function sceneFromHints(sceneType, partnerClass) {
  const t = String(sceneType || '').toLowerCase();
  if (/blowjob|oral/.test(t)) return 'oral';
  if (/titjob|breast/.test(t)) return 'breast';
  if (/handjob|hand/.test(t)) return 'hand';
  if (/penetration|intercourse|sex/.test(t)) return 'intercourse';
  const c = String(partnerClass || '').toLowerCase();
  if (/mouth|lips|oral/.test(c)) return 'oral';
  if (/nipple|breast/.test(c)) return 'breast';
  if (/hand/.test(c)) return 'hand';
  return '';
}

export function initGenFeel(root, { sceneType = () => '' } = {}) {
  const q = sel => root.querySelector(sel);
  const contactWrap = q('#gen-contact-vibration-wrap');
  if (!contactWrap || q('#gen-device-target')) return null;
  const box = document.createElement('div');
  box.className = 'gen-feel-device';
  box.innerHTML = `
    <div class="field-row">
      <label for="gen-device-target" data-help="Only the settings this device can use are shown. Auto follows the connected device.">For device</label>
      <select id="gen-device-target">
        <option value="auto">Connected device (auto)</option>
        <option value="neo2">Sam Neo 2 — vibration + suction</option>
        <option value="vibrator">Vibrator / other toy</option>
        <option value="stroker">Stroker — The Handy, OSR2 …</option>
      </select>
      <span class="checkbox-row gen-show-all">
        <input type="checkbox" id="gen-show-all-feel" />
        <label for="gen-show-all-feel" data-help="Also show the settings this device cannot use.">Show all</label>
      </span>
    </div>
    <p class="hint gen-device-hint" id="gen-device-hint"></p>
    <details class="gen-neo-feel" id="gen-neo-feel">
      <summary class="gen-neo-feel-title">Sam Neo 2 feel · <span id="gen-neo-feel-sum"></span></summary>
      <div class="field-row">
        <label for="gen-scene-feel" data-help="How suction follows the stroke. Oral: a steady seal that pulls deeper with each stroke. Intercourse: follows the rhythm. Breast: softer, follows depth. Hand: grip follows speed. Classic: suction follows depth, as before.">Scene</label>
        <select id="gen-scene-feel">
          <option value="auto">Auto (from the scene)</option>
          <option value="">Classic — suction follows depth</option>
          <option value="oral">Oral</option>
          <option value="intercourse">Intercourse</option>
          <option value="breast">Breast</option>
          <option value="hand">Hand</option>
        </select>
      </div>
      <p class="hint gen-scene-auto" id="gen-scene-auto"></p>
      <div class="field-row gen-mix-row">
        <label for="gen-feel-balance" data-help="Move the weight between vibration and suction. Centre = both full.">Mix</label>
        <span class="gen-mix-end">Vibration</span>
        <input type="range" id="gen-feel-balance" min="-100" max="100" step="10" value="0" />
        <span class="gen-mix-end">Suction</span>
      </div>
      <div class="checkbox-row">
        <input type="checkbox" id="gen-vib-follow" />
        <label for="gen-vib-follow" data-help="Vibration takes the same pattern as suction — both move together; use Mix to make one weaker.">Vibration follows suction</label>
      </div>
    </details>
    `;
  contactWrap.before(box);

  const deviceSel = q('#gen-device-target');
  const sceneSel = q('#gen-scene-feel');
  const balance = q('#gen-feel-balance');
  const follow = q('#gen-vib-follow');
  const showAll = q('#gen-show-all-feel');
  const hint = q('#gen-device-hint');
  const autoNote = q('#gen-scene-auto');
  let connected = null; // device kind of the connected device

  try {
    const s = JSON.parse(localStorage.getItem(KEY) || '{}');
    if (s.device && [...deviceSel.options].some(o => o.value === s.device)) deviceSel.value = s.device;
    if (typeof s.scene === 'string' && [...sceneSel.options].some(o => o.value === s.scene)) sceneSel.value = s.scene;
    if (Number.isFinite(s.balance)) balance.value = String(s.balance);
    follow.checked = !!s.follow;
    showAll.checked = !!s.showAll;
  } catch (_) { /* ignore */ }
  const store = () => {
    try {
      localStorage.setItem(KEY, JSON.stringify({ device: deviceSel.value, scene: sceneSel.value,
        balance: Number(balance.value), follow: follow.checked, showAll: showAll.checked }));
    } catch (_) { /* ignore */ }
  };

  const kind = () => (deviceSel.value === 'auto' ? (connected || 'neo2') : deviceSel.value);
  const resolvedScene = () => (sceneSel.value === 'auto'
    ? sceneFromHints(sceneType(), q('#gen-region-class2')?.value) : sceneSel.value);

  function sync() {
    const k = kind();
    root.dataset.genDevice = k;
    root.dataset.genShowAll = showAll.checked ? '1' : '';
    // A short note only where settings are hidden; the Sam Neo 2 shows all.
    hint.textContent = {
      vibrator: 'Vibrator: contact vibration (the video\'s sound can drive it too: Player → Audio). Suction feel hidden.',
      stroker: 'Stroker: the stroke is all it needs — vibration and suction settings hidden.',
    }[k] || '';
    hint.hidden = !hint.textContent || showAll.checked;
    deviceSel.title = deviceSel.value === 'auto' && !connected ? 'No device connected — assuming Sam Neo 2' : '';
    const auto = resolvedScene();
    autoNote.hidden = sceneSel.value !== 'auto';
    autoNote.textContent = auto ? `Auto: ${SCENES[auto]}` : 'Auto: no scene detected yet — Classic until one is.';
    const b = Number(balance.value);
    q('#gen-neo-feel-sum').textContent = [
      auto ? SCENES[auto] : 'Classic',
      b ? `mix ${b > 0 ? 'suction' : 'vibration'} ${Math.abs(b)}%` : '',
      follow.checked ? 'vibration follows suction' : '',
    ].filter(Boolean).join(' · ');
  }
  for (const n of [deviceSel, sceneSel, balance, follow, showAll]) {
    n.addEventListener('change', () => { store(); sync(); });
  }
  q('#gen-region-class2')?.addEventListener('change', sync);
  window.addEventListener('device:status', e => { connected = deviceKind(e.detail); sync(); });
  GetDeviceStatus().then(st => { if (connected == null) { connected = deviceKind(st); sync(); } }).catch(() => {});
  sync();

  return {
    sync,
    // The feel to write after Create, or null when nothing changes the old
    // mapping (or the device has no suction / vibration feel).
    settings() {
      if (kind() !== 'neo2') return null;
      const sceneFeel = resolvedScene();
      const b = Math.max(-1, Math.min(1, Number(balance.value) / 100 || 0));
      if (!sceneFeel && !follow.checked && b === 0) return null;
      return { sceneFeel, follow: follow.checked, balance: b };
    },
  };
}
