// Sam Neo 2 feel of the loaded script (Inspector → Devices): Scene (how
// suction follows the stroke), Mix (vibration ↔ suction) and "Vibration
// follows suction" — the same algorithms as the Generator (feel_scene.go),
// written into the script with SaveFeelSettings. Shown for the Sam Neo 2 or
// when no device is connected; a stroker or a vibrator without suction does
// not get it.
import { GetFeelSettings, SaveFeelSettings, BakeSoundVibration } from '../wailsjs/go/main/App';
import { deviceKind } from './gen_feel.js';

export function mountFeelCard(host, root, { onSaved } = {}) {
  if (!host) return null;
  const card = document.createElement('div');
  card.className = 'pb-card pb-feel-card';
  card.id = 'pb-feel-card';
  card.innerHTML = `
    <p class="pb-card-title">Sam Neo 2 feel</p>
    <div class="field-row">
      <label for="pb-scene-feel" data-help="How suction follows the stroke. Oral: steady seal, deeper pulls harder. Intercourse: follows the rhythm. Breast: softer, follows depth. Hand: grip follows speed. Classic: suction follows depth, as before.">Scene</label>
      <select id="pb-scene-feel">
        <option value="">Classic — suction follows depth</option>
        <option value="oral">Oral</option>
        <option value="intercourse">Intercourse</option>
        <option value="breast">Breast</option>
        <option value="hand">Hand</option>
      </select>
    </div>
    <div class="field-row pb-mix-row">
      <label for="pb-feel-balance">Mix</label>
      <span class="pb-mix-end">Vibration</span>
      <input type="range" id="pb-feel-balance" min="-100" max="100" step="10" value="0" />
      <span class="pb-mix-end">Suction</span>
    </div>
    <div class="checkbox-row">
      <input type="checkbox" id="pb-vib-follow" />
      <label for="pb-vib-follow">Vibration follows suction</label>
    </div>
    <p class="hint" id="pb-feel-status" aria-live="polite"></p>`;
  host.append(card);
  const scene = card.querySelector('#pb-scene-feel');
  const balance = card.querySelector('#pb-feel-balance');
  const follow = card.querySelector('#pb-vib-follow');
  const status = card.querySelector('#pb-feel-status');
  let kind = null;

  async function load() {
    try {
      const f = await GetFeelSettings();
      scene.value = f.sceneFeel || '';
      balance.value = String(Math.round((f.balance || 0) * 100));
      follow.checked = !!f.vibrationFollowsSuction;
      card.dataset.loaded = '1';
    } catch (_) {
      card.dataset.loaded = '';
    }
    sync();
  }
  function sync() {
    const loaded = root.dataset.pbState === 'loaded';
    card.hidden = !loaded || kind === 'stroker' || kind === 'vibrator';
    for (const n of [scene, balance, follow]) n.disabled = !loaded;
  }
  let saving = Promise.resolve();
  const save = () => {
    saving = saving.then(async () => {
      status.textContent = 'Saving…';
      try {
        await SaveFeelSettings(scene.value, follow.checked, Number(balance.value) / 100);
        status.textContent = 'Saved — the device feels it on the next play.';
        onSaved?.();
      } catch (err) {
        status.textContent = 'Could not save: ' + err;
      }
    });
  };
  for (const n of [scene, balance, follow]) n.addEventListener('change', save);
  new MutationObserver(() => { if (root.dataset.pbState === 'loaded') load(); else sync(); })
    .observe(root, { attributes: true, attributeFilter: ['data-pb-state', 'data-pb-script'] });
  window.addEventListener('device:status', e => { kind = deviceKind(e.detail); sync(); });
  sync();
  return { load };
}

// Vibration from the video's sound for a vibrator / other toy: writes the
// loaded script's vibration lane from the sound (or sound + contact).
// Shown when a vibrator (no stroke, not a Sam Neo 2) is connected.
export function mountSoundVibCard(host, root, { onSaved } = {}) {
  if (!host) return null;
  const card = document.createElement('div');
  card.className = 'pb-card pb-sound-vib-card';
  card.id = 'pb-sound-vib-card';
  card.innerHTML = `
    <p class="pb-card-title">Vibration from sound</p>
    <div class="field-row">
      <select id="pb-vib-source" aria-label="Vibration source">
        <option value="sound+contact">Sound + contact</option>
        <option value="sound">Sound only</option>
      </select>
      <button type="button" id="pb-vib-source-apply" data-help="Writes the vibration lane of this script from the video's loudness (and, with contact, the script's own vibration where it is stronger). It replaces the current vibration lane.">Write into script</button>
    </div>
    <p class="hint" id="pb-vib-source-status" aria-live="polite"></p>`;
  host.append(card);
  const status = card.querySelector('#pb-vib-source-status');
  const btn = card.querySelector('#pb-vib-source-apply');
  let kind = null;
  const sync = () => { card.hidden = root.dataset.pbState !== 'loaded' || kind !== 'vibrator'; };
  btn.addEventListener('click', async () => {
    btn.disabled = true;
    status.textContent = 'Reading the sound…';
    try {
      const r = await BakeSoundVibration(card.querySelector('#pb-vib-source').value);
      status.textContent = (r && r.message) || 'Written.';
      onSaved?.();
    } catch (err) {
      status.textContent = 'Could not write: ' + err;
    } finally {
      btn.disabled = false;
    }
  });
  new MutationObserver(sync).observe(root, { attributes: true, attributeFilter: ['data-pb-state'] });
  window.addEventListener('device:status', e => { kind = deviceKind(e.detail); sync(); });
  sync();
  return { sync };
}
