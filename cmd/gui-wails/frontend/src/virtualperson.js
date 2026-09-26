/**
 * Virtual Person overlay — consumes host EmitAnimation events.
 *
 * Backend: cmd/gui-wails/app_virtualperson.go emits:
 *   virtualperson:pose   — PoseSample map each Tick
 *   virtualperson:status — { enabled, running }
 *
 * MVP renderer: canvas sprite-puppet stub driven by stroke/expression
 * channels + prop phase. Real character-01 / props/dildo.png assets
 * remain Owner/tracken (plan item #1).
 */

import {
  EnableVirtualPerson,
  DisableVirtualPerson,
  VirtualPersonRunning,
  VirtualPersonGiveDildo,
  VirtualPersonStartTitjob,
  VirtualPersonSetToySync,
  VirtualPersonToySync,
} from '../wailsjs/go/main/App';
import { EventsOn } from '../wailsjs/runtime/runtime';
import { uiError, uiInfo } from './notify.js';

/** @typedef {{ propId: string, socket: string, phaseOffset: number, visible: boolean }} PropSnap */
/** @typedef {{ atMs?: number, stroke?: number, surge?: number, sway?: number, twist?: number, vibe?: number, suck?: number, expression?: number }} Channels */
/** @typedef {{ characterId?: string, atMs?: number, skin?: string, channels?: Channels, props?: PropSnap[] }} PoseMap */

let lastPose = /** @type {PoseMap|null} */ (null);
let running = false;
let canvas = /** @type {HTMLCanvasElement|null} */ (null);
let ctx2d = /** @type {CanvasRenderingContext2D|null} */ (null);
let raf = 0;

function clamp01(n) {
  const x = Number(n);
  if (!Number.isFinite(x)) return 0;
  return Math.max(0, Math.min(1, x));
}

function stroke01(ch) {
  if (!ch) return 0;
  // ChannelValues.Stroke is 0–100 style in the activity loop.
  const s = Number(ch.stroke);
  if (!Number.isFinite(s)) return 0;
  return s > 1.5 ? clamp01(s / 100) : clamp01(s);
}

/** Draw a simple layered puppet + optional dildo prop (placeholder geometry). */
function drawPose(pose) {
  if (!canvas || !ctx2d) return;
  const w = canvas.width;
  const h = canvas.height;
  const c = ctx2d;
  c.clearRect(0, 0, w, h);

  // Stage background
  c.fillStyle = 'rgba(12, 14, 20, 0.92)';
  c.fillRect(0, 0, w, h);
  c.strokeStyle = 'rgba(232, 176, 110, 0.22)';
  c.lineWidth = 1;
  c.strokeRect(0.5, 0.5, w - 1, h - 1);

  const ch = pose?.channels || {};
  const level = stroke01(ch);
  const expr = clamp01(ch.expression);
  const vibe = clamp01(ch.vibe);

  // Body silhouette (sprite_puppet stub — replace with primary.jpg layers later)
  const cx = w * 0.5;
  const cy = h * 0.52;
  const bodyW = 52;
  const bodyH = 78;
  const bob = (level - 0.5) * 10;

  // Torso
  c.fillStyle = `rgba(232, 176, 110, ${0.35 + expr * 0.35})`;
  c.beginPath();
  c.ellipse(cx, cy + bob, bodyW * 0.55, bodyH * 0.42, 0, 0, Math.PI * 2);
  c.fill();

  // Head
  c.fillStyle = `rgba(244, 241, 234, ${0.55 + expr * 0.25})`;
  c.beginPath();
  c.arc(cx, cy - bodyH * 0.38 + bob * 0.4, 16, 0, Math.PI * 2);
  c.fill();

  // Chest highlight (activity focus)
  c.fillStyle = `rgba(212, 154, 134, ${0.25 + level * 0.4})`;
  c.beginPath();
  c.ellipse(cx, cy - 6 + bob, 22, 12, 0, 0, Math.PI * 2);
  c.fill();

  // Props: dildo on chest / hand with phase offset slide
  const props = Array.isArray(pose?.props) ? pose.props : [];
  for (const pr of props) {
    if (!pr || pr.visible === false) continue;
    const phase = clamp01(pr.phaseOffset);
    const slide = (phase - 0.5) * 28;
    let px = cx;
    let py = cy - 4 + bob;
    if (pr.socket === 'hand_r') {
      px = cx + 28;
      py = cy + 8 + bob;
    } else if (pr.socket === 'hand_l') {
      px = cx - 28;
      py = cy + 8 + bob;
    } else if (pr.socket === 'chest') {
      px = cx + slide * 0.15;
      py = cy - 8 + bob + slide * 0.35;
    } else if (pr.socket === 'mouth') {
      px = cx;
      py = cy - bodyH * 0.32 + bob * 0.4;
    }

    // Dildo placeholder (capsule) — real art: props/dildo.png
    c.save();
    c.translate(px, py);
    c.rotate(-0.35 + phase * 0.5);
    c.fillStyle = `rgba(90, 212, 196, ${0.55 + vibe * 0.35})`;
    c.beginPath();
    const rw = 6;
    const rh = 22;
    c.roundRect(-rw, -rh, rw * 2, rh * 2, rw);
    c.fill();
    c.restore();
  }

  // Readout strip
  c.fillStyle = 'rgba(176, 170, 160, 0.85)';
  c.font = '11px Figtree, system-ui, sans-serif';
  const id = pose?.characterId || '—';
  const strokePct = Math.round(level * 100);
  c.fillText(`${id} · stroke ${strokePct}%`, 8, h - 10);
}

function scheduleDraw() {
  if (raf) return;
  raf = requestAnimationFrame(() => {
    raf = 0;
    if (lastPose) drawPose(lastPose);
  });
}

function setStatusUI(root, on) {
  const led = root.querySelector('#vp-led');
  const label = root.querySelector('#vp-status-label');
  if (led) {
    led.classList.toggle('is-on', !!on);
  }
  if (label) {
    label.textContent = on ? 'Running' : 'Off';
  }
  const enableBtn = root.querySelector('#vp-enable');
  const disableBtn = root.querySelector('#vp-disable');
  if (enableBtn) enableBtn.disabled = !!on;
  if (disableBtn) disableBtn.disabled = !on;
}

function updateMeters(root, pose) {
  const ch = pose?.channels || {};
  const stroke = stroke01(ch);
  const vibe = clamp01(ch.vibe);
  const suck = clamp01(ch.suck);
  const setBar = (sel, v) => {
    const el = root.querySelector(sel);
    if (el) el.style.width = `${Math.round(v * 100)}%`;
  };
  setBar('#vp-bar-stroke span', stroke);
  setBar('#vp-bar-vibe span', vibe);
  setBar('#vp-bar-suck span', suck);
  const propsEl = root.querySelector('#vp-props');
  if (propsEl) {
    const props = Array.isArray(pose?.props) ? pose.props : [];
    propsEl.textContent = props.length
      ? props.map((p) => `${p.propId}@${p.socket || 'inv'}`).join(', ')
      : 'no props';
  }
}

/**
 * Mount Virtual Person card into the sidebar (or optional host).
 * @param {HTMLElement|null} host
 */
export function initVirtualPerson(host) {
  const root = host || document.getElementById('sidebar');
  if (!root) return;

  const card = document.createElement('div');
  card.className = 'card vp-card';
  card.id = 'vp-card';
  card.innerHTML = `
    <h2>Virtual Person</h2>
    <p class="hint vp-hint">Props first · optional Neo 2 sync</p>
    <div class="vp-status-row">
      <i id="vp-led" class="led" aria-hidden="true"></i>
      <span id="vp-status-label">Off</span>
      <span class="hint" id="vp-props">no props</span>
    </div>
    <canvas id="vp-canvas" width="200" height="160" aria-label="Character stage"></canvas>
    <div class="meter vp-meter">
      <div class="row"><span>Stroke</span></div>
      <div class="bar" id="vp-bar-stroke"><span style="width:0%;background:var(--accent)"></span></div>
      <div class="row"><span>Vibe</span></div>
      <div class="bar" id="vp-bar-vibe"><span style="width:0%;background:var(--teal)"></span></div>
      <div class="row"><span>Suck</span></div>
      <div class="bar" id="vp-bar-suck"><span style="width:0%;background:var(--feel)"></span></div>
    </div>
    <div class="vp-actions">
      <button type="button" id="vp-enable" class="primary">Enable</button>
      <button type="button" id="vp-disable" disabled>Disable</button>
    </div>
    <div class="vp-actions">
      <button type="button" id="vp-give">Give dildo</button>
      <button type="button" id="vp-titjob">Start titjob</button>
    </div>
    <div class="checkbox-row vp-sync-row">
      <input type="checkbox" id="vp-toy-sync" />
      <label for="vp-toy-sync">Sync real device (off by default)</label>
    </div>
  `;
  root.prepend(card);

  canvas = card.querySelector('#vp-canvas');
  ctx2d = canvas ? canvas.getContext('2d') : null;
  drawPose(null);

  EventsOn('virtualperson:pose', (pose) => {
    lastPose = pose || null;
    scheduleDraw();
    updateMeters(card, lastPose);
  });

  EventsOn('virtualperson:status', (st) => {
    running = !!(st && st.running);
    setStatusUI(card, running);
  });

  // Bootstrap status from Go
  VirtualPersonRunning()
    .then((on) => {
      running = !!on;
      setStatusUI(card, running);
    })
    .catch(() => {});

  VirtualPersonToySync()
    .then((on) => {
      const box = card.querySelector('#vp-toy-sync');
      if (box) box.checked = !!on;
    })
    .catch(() => {});

  card.querySelector('#vp-enable')?.addEventListener('click', async () => {
    try {
      await EnableVirtualPerson();
      running = true;
      setStatusUI(card, true);
      uiInfo('Virtual Person enabled');
    } catch (err) {
      uiError('Enable Virtual Person: ' + err);
    }
  });

  card.querySelector('#vp-disable')?.addEventListener('click', async () => {
    try {
      await DisableVirtualPerson();
      running = false;
      setStatusUI(card, false);
      lastPose = null;
      drawPose(null);
      updateMeters(card, null);
      uiInfo('Virtual Person disabled');
    } catch (err) {
      uiError('Disable Virtual Person: ' + err);
    }
  });

  card.querySelector('#vp-give')?.addEventListener('click', async () => {
    try {
      await VirtualPersonGiveDildo();
      uiInfo('Gave dildo prop');
    } catch (err) {
      uiError('Give dildo: ' + err);
    }
  });

  card.querySelector('#vp-titjob')?.addEventListener('click', async () => {
    try {
      if (!running) {
        await EnableVirtualPerson();
        running = true;
        setStatusUI(card, true);
      }
      await VirtualPersonGiveDildo();
      await VirtualPersonStartTitjob(0.6, 30);
      uiInfo('Titjob activity started (30s)');
    } catch (err) {
      uiError('Start titjob: ' + err);
    }
  });

  card.querySelector('#vp-toy-sync')?.addEventListener('change', async (e) => {
    const on = !!e.target.checked;
    try {
      await VirtualPersonSetToySync(on);
      uiInfo(on ? 'Device sync on' : 'Device sync off');
    } catch (err) {
      uiError('Toy sync: ' + err);
      e.target.checked = !on;
    }
  });
}
