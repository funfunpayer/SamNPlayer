// Training follows the connected device (Owner 14 Oct): the stage shows a
// ring only for the channels the device really has (device capabilities),
// the Channel select offers only those, and a device with neither vibration
// nor suction (a stroker) cannot start a training that would do nothing.
// Without a connection (training connects on Start) or with Mock, both
// channels stay — nothing is known yet, nothing is hidden.
import { GetDeviceStatus } from '../wailsjs/go/main/App';

// Which training channels a device status offers: null = unknown (show all).
export function trainingChannels(status) {
  const feats = status && status.connected && status.capabilities && Array.isArray(status.capabilities.features)
    ? status.capabilities.features : null;
  if (!feats || !feats.length) return null;
  const has = re => feats.some(f => re.test(String(f.channel || '')) && f.driven !== false);
  return { vibration: has(/^vibration\d*$/), suction: has(/^suction$/), name: status.capabilities.device || status.name || 'This device' };
}

export function applyTrainingDevice(root) {
  if (!root) return null;
  const q = sel => root.querySelector(sel);
  const channelSel = q('#tr-channel');
  const mock = q('#tr-mock');
  const startBtn = q('#tr-start');
  const note = document.createElement('p');
  note.className = 'trl-device-note';
  note.id = 'tr-device-note';
  note.hidden = true;
  (q('.trl-session') || q('#tr-start')?.parentElement || root).append(note);
  let status = null;
  let blocked = false;

  function apply() {
    const ch = mock && mock.checked ? null : trainingChannels(status);
    const vib = !ch || ch.vibration;
    const suc = !ch || ch.suction;
    root.querySelectorAll('.flow-tr__channel[data-axis="vibration"]').forEach(c => { c.hidden = !vib; });
    root.querySelectorAll('.flow-tr__channel[data-axis="suction"]').forEach(c => { c.hidden = !suc; });
    root.dataset.trChannels = [vib && 'vibration', suc && 'suction'].filter(Boolean).join(',');
    if (channelSel) {
      for (const o of channelSel.options) {
        o.disabled = (o.value === 'vibration' && !vib) || (o.value === 'suction' && !suc)
          || (o.value === 'both' && !(vib && suc));
      }
      const pick = vib && !suc ? 'vibration' : suc && !vib ? 'suction' : null;
      if (pick && channelSel.value !== pick) {
        channelSel.value = pick;
        channelSel.dispatchEvent(new Event('change'));
      }
    }
    blocked = !!ch && !vib && !suc;
    root.dataset.trBlocked = blocked ? '1' : '';
    if (!ch) note.textContent = '';
    else if (blocked) note.textContent = `${ch.name} has no vibration or suction motor — Training needs one of them (e.g. Sam Neo 2). Mock tests without a device.`;
    else if (!suc) note.textContent = `${ch.name} has no suction — Training drives vibration.`;
    else if (!vib) note.textContent = `${ch.name} has no vibration — Training drives suction.`;
    else note.textContent = '';
    note.hidden = !note.textContent;
  }

  // A blocked start never reaches training.js (capture runs first at the target).
  // Saved settings may set the Channel select after the device was seen:
  // check again right before a start.
  startBtn?.addEventListener('click', e => {
    apply();
    if (!blocked) return;
    e.preventDefault();
    e.stopImmediatePropagation();
    note.classList.remove('is-flash');
    void note.offsetWidth;
    note.classList.add('is-flash');
  }, true);

  window.addEventListener('device:status', e => { status = e.detail || null; apply(); });
  mock?.addEventListener('change', apply);
  GetDeviceStatus().then(st => { if (status == null) { status = st; apply(); } }).catch(() => {});
  apply();
  return { apply };
}
