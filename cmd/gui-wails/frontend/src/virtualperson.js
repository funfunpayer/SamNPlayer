/**
 * Virtual Person overlay — consumes host EmitAnimation + status events.
 * MVP: control card in sidebar + lightweight stage readout (sprite assets later).
 * Product metaphor: virtual props/activities; real-device sync opt-in.
 */
import { EventsOn } from '../wailsjs/runtime/runtime';
import { uiError, uiInfo } from './notify.js';
import {
  EnableVirtualPerson,
  DisableVirtualPerson,
  VirtualPersonRunning,
  VirtualPersonGiveDildo,
  VirtualPersonStartTitjob,
  VirtualPersonSetToySync,
  VirtualPersonToySync,
} from '../wailsjs/go/main/App';

const VP_CSS = `
.vp-card .vp-row { display: flex; gap: 8px; flex-wrap: wrap; }
.vp-card .vp-row button { flex: 1 1 auto; min-width: 0; font-size: 12px; padding: 7px 10px; }
.vp-status { margin-top: 8px; font-size: 12px; color: var(--text-dim); font-weight: 500; }
.vp-status.is-on { color: var(--ok); }
.vp-sync-label { display: flex; align-items: center; gap: 8px; margin-top: 12px; font-size: 12px; color: var(--text-dim); cursor: pointer; user-select: none; }
.vp-sync-label input { accent-color: var(--teal); }
.vp-pose { margin-top: 12px; border-top: 1px solid rgba(50, 54, 74, 0.55); padding-top: 10px; }
.vp-pose-stage { position: relative; height: 88px; border-radius: var(--radius-sm); background: rgba(12, 14, 20, 0.85); border: 1px solid rgba(50, 54, 74, 0.6); display: grid; place-items: center; overflow: hidden; }
.vp-idle { font-size: 11px; color: var(--text-dim); opacity: 0.75; }
.vp-puppet { position: relative; width: 48px; height: 72px; transform: translateY(calc((50 - var(--stroke, 50)) * 0.18px)); transition: transform 0.08s linear; }
.vp-body { position: absolute; inset: 0; border-radius: 18px 18px 10px 10px; background: linear-gradient(180deg, rgba(232, 176, 110, 0.35), rgba(90, 212, 196, 0.18)); border: 1px solid rgba(232, 176, 110, 0.35); }
.vp-prop { position: absolute; left: 50%; top: 42%; width: 10px; height: 28px; margin-left: -5px; border-radius: 4px; background: rgba(90, 212, 196, 0.55); border: 1px solid rgba(90, 212, 196, 0.7); opacity: 0; transform: scaleY(0.6); transition: opacity 0.2s ease, transform 0.2s ease; }
.vp-prop.is-visible { opacity: 1; transform: scaleY(1); }
.vp-phase { position: absolute; left: 8px; bottom: 6px; font-size: 10px; color: var(--text-dim); letter-spacing: 0.02em; }
.vp-channels { display: grid; grid-template-columns: 1fr 1fr; gap: 4px 10px; margin-top: 8px; font-size: 11px; color: var(--text-dim); }
.vp-ch { display: flex; justify-content: space-between; gap: 6px; }
.vp-ch b { color: var(--text); font-weight: 600; font-variant-numeric: tabular-nums; }
`;

function ensureVpStyles() {
  if (document.getElementById('vp-styles')) return;
  const s = document.createElement('style');
  s.id = 'vp-styles';
  s.textContent = VP_CSS;
  document.head.appendChild(s);
}

/** @param {HTMLElement} [root] optional sidebar host (main.js may pass it) */
export function initVirtualPerson(root) {
  ensureVpStyles();
  const sidebar = root || document.getElementById('sidebar');
  if (!sidebar) return;

  const card = document.createElement('div');
  card.className = 'card vp-card';
  card.innerHTML = `
    <h2>Virtual Person</h2>
    <p class="hint" style="margin:0 0 10px">
      Props in-scene first. Neo&nbsp;2 sync optional (off by default).
    </p>
    <div class="vp-row">
      <button type="button" id="vp-enable" class="primary">Enable</button>
      <button type="button" id="vp-disable" disabled>Disable</button>
    </div>
    <div class="vp-status" id="vp-status">Off</div>
    <div class="vp-row" style="margin-top:10px">
      <button type="button" id="vp-give" disabled>Give dildo</button>
      <button type="button" id="vp-titjob" disabled>Titjob</button>
    </div>
    <label class="vp-sync-label">
      <input type="checkbox" id="vp-sync" />
      Sync real device (ToyHub)
    </label>
    <div class="vp-pose" id="vp-pose" aria-live="polite">
      <div class="vp-pose-stage" id="vp-stage">
        <span class="vp-idle">No pose yet</span>
      </div>
      <div class="vp-channels" id="vp-channels"></div>
    </div>
  `;
  sidebar.appendChild(card);

  const btnEnable = card.querySelector('#vp-enable');
  const btnDisable = card.querySelector('#vp-disable');
  const btnGive = card.querySelector('#vp-give');
  const btnTitjob = card.querySelector('#vp-titjob');
  const chkSync = card.querySelector('#vp-sync');
  const statusEl = card.querySelector('#vp-status');
  const stageEl = card.querySelector('#vp-stage');
  const chEl = card.querySelector('#vp-channels');

  let running = false;

  function setRunning(on) {
    running = !!on;
    btnEnable.disabled = running;
    btnDisable.disabled = !running;
    btnGive.disabled = !running;
    btnTitjob.disabled = !running;
    statusEl.textContent = running ? 'Running' : 'Off';
    statusEl.classList.toggle('is-on', running);
  }

  function paintPose(pose) {
    if (!pose) return;
    const ch = pose.channels || {};
    const stroke = Math.round(Number(ch.stroke) || 0);
    const vibe = Math.round((Number(ch.vibe) || 0) * 100);
    const suck = Math.round((Number(ch.suck) || 0) * 100);
    const props = Array.isArray(pose.props) ? pose.props : [];
    const propLabel = props.length
      ? props.map(p => `${p.propId}@${p.socket}`).join(', ')
      : 'none';

    stageEl.innerHTML = `
      <div class="vp-puppet" style="--stroke:${stroke}">
        <div class="vp-body"></div>
        <div class="vp-prop ${props.length ? 'is-visible' : ''}" title="${propLabel}"></div>
      </div>
      <span class="vp-phase">${pose.characterId || 'character-01'} · stroke ${stroke}</span>
    `;
    chEl.innerHTML = `
      <div class="vp-ch"><span>Stroke</span><b>${stroke}</b></div>
      <div class="vp-ch"><span>Vibe</span><b>${vibe}%</b></div>
      <div class="vp-ch"><span>Suck</span><b>${suck}%</b></div>
      <div class="vp-ch"><span>Props</span><b>${propLabel}</b></div>
    `;
  }

  btnEnable.addEventListener('click', async () => {
    try {
      await EnableVirtualPerson();
      setRunning(true);
      uiInfo('Virtual Person enabled');
    } catch (err) {
      uiError('Enable Virtual Person: ' + err);
    }
  });

  btnDisable.addEventListener('click', () => {
    try {
      DisableVirtualPerson();
      setRunning(false);
      stageEl.innerHTML = '<span class="vp-idle">No pose yet</span>';
      chEl.innerHTML = '';
      uiInfo('Virtual Person disabled');
    } catch (err) {
      uiError('Disable: ' + err);
    }
  });

  btnGive.addEventListener('click', async () => {
    try {
      await VirtualPersonGiveDildo();
      uiInfo('Gave dildo prop');
    } catch (err) {
      uiError('Give dildo: ' + err);
    }
  });

  btnTitjob.addEventListener('click', async () => {
    try {
      await VirtualPersonStartTitjob(0.6, 0);
      uiInfo('Started titjob_dildo');
    } catch (err) {
      uiError('Start titjob: ' + err);
    }
  });

  chkSync.addEventListener('change', () => {
    try {
      VirtualPersonSetToySync(!!chkSync.checked);
    } catch (err) {
      uiError('Toy sync: ' + err);
      chkSync.checked = false;
    }
  });

  EventsOn('virtualperson:pose', (pose) => {
    if (running) paintPose(pose);
  });

  EventsOn('virtualperson:status', (st) => {
    if (st && typeof st.running === 'boolean') setRunning(st.running);
  });

  VirtualPersonRunning()
    .then((on) => setRunning(!!on))
    .catch(() => setRunning(false));
  VirtualPersonToySync()
    .then((on) => { chkSync.checked = !!on; })
    .catch(() => { chkSync.checked = false; });
}
