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

export function initVirtualPerson() {
  const sidebar = document.getElementById('sidebar');
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
