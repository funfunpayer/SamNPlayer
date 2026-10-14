// V1 Training layout (Owner 14 Oct, target docs/visual-target/training.png):
//   left   — Training: programs (built-in + my programs), custom, history
//   centre — program + phase stepper, stage, session bar, live curves,
//            session timeline
//   right  — session settings, session mode, safety, feedback, recent sessions
// Existing elements are moved, not re-created (IDs, listeners and tests
// stay). New parts only show real data: the planned phases and total come
// from TrainingScriptPreview (tr:plan) or the Custom timings, the clock runs
// while the training runs, recent sessions are the training history.
import { ListTrainingScripts, PauseTraining, ResumeTraining } from '../wailsjs/go/main/App';

function make(tag, cls, text) {
  const e = document.createElement(tag);
  if (cls) e.className = cls;
  if (text != null) e.textContent = text;
  return e;
}

function card(cls, title) {
  const c = make('section', `card ${cls}`);
  if (title) c.append(make('p', 'trl-card-title', title));
  return c;
}

const PLAY_ICON = '<svg class="fp-play" viewBox="0 0 24 24" aria-hidden="true"><path d="M8.6 5.4c-.9-.55-2.1.1-2.1 1.15v10.9c0 1.05 1.2 1.7 2.1 1.15l8.9-5.45c.85-.52.85-1.78 0-2.3z"/></svg>';
const PAUSE_ICON = '<svg class="fp-pause" viewBox="0 0 24 24" aria-hidden="true"><rect x="6.5" y="5.5" width="4" height="13" rx="1.4"/><rect x="13.5" y="5.5" width="4" height="13" rx="1.4"/></svg>';
const STOP_ICON = '<svg viewBox="0 0 24 24" aria-hidden="true"><rect x="6.5" y="6.5" width="11" height="11" rx="2"/></svg>';
const SKIP_ICON = '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M6.6 6.2c-.7-.45-1.6.05-1.6.9v9.8c0 .85.9 1.35 1.6.9l7.4-4.9c.65-.42.65-1.38 0-1.8z"/><rect x="15.8" y="5.8" width="3.2" height="12.4" rx="1.2"/></svg>';

export function fmtClock(ms) {
  const s = Math.max(0, Math.round((ms || 0) / 1000));
  const h = Math.floor(s / 3600), m = Math.floor((s % 3600) / 60), sec = s % 60;
  const mm = String(m).padStart(h ? 2 : 1, '0');
  return (h ? h + ':' : '') + mm + ':' + String(sec).padStart(2, '0');
}

const num = (root, sel) => parseFloat(root.querySelector(sel)?.value) || 0;

// Planned curve points inside [from, to] (plus the edges, interpolated).
function slice(points, from, to) {
  const out = [];
  const at = ms => {
    for (let i = 1; i < points.length; i++) {
      const a = points[i - 1], b = points[i];
      if (ms <= b.atMs) {
        const f = b.atMs > a.atMs ? (ms - a.atMs) / (b.atMs - a.atMs) : 0;
        return a.level + (b.level - a.level) * Math.max(0, Math.min(1, f));
      }
    }
    return points.length ? points[points.length - 1].level : 0;
  };
  out.push({ atMs: from, level: at(from) });
  for (const p of points) if (p.atMs > from && p.atMs < to) out.push(p);
  out.push({ atMs: to, level: at(to) });
  return out;
}

// Colour of a phase from its planned strength (mean of the stronger
// channel): calm cyan → orange → red, like the target's timeline.
const HEAT = [[54, 200, 255], [255, 138, 35], [255, 59, 92]];
export function heatColor(level) {
  // Most plans run between ~25% and ~80%: spread that range over the colours.
  const v = Math.max(0, Math.min(1, (level - 0.25) / 0.55)) * (HEAT.length - 1);
  const i = Math.min(HEAT.length - 2, Math.floor(v)), f = v - i;
  const c = HEAT[i].map((a, k) => Math.round(a + (HEAT[i + 1][k] - a) * f));
  return `rgb(${c[0]}, ${c[1]}, ${c[2]})`;
}

// Mean planned level of the stronger channel inside a phase.
export function phaseStrength(plan, from, to) {
  let best = 0;
  for (const key of ['vibration', 'suction']) {
    const pts = plan[key] || [];
    if (pts.length < 2 || to <= from) continue;
    const s = slice(pts, from, to);
    let area = 0;
    for (let i = 1; i < s.length; i++) area += (s[i].atMs - s[i - 1].atMs) * (s[i].level + s[i - 1].level) / 2;
    best = Math.max(best, area / (to - from));
  }
  return best;
}

// A small curve of the planned levels inside one phase (SVG, both channels).
function miniCurve(plan, from, to) {
  const W = 100, H = 28;
  const paths = [['suction', '#ff8a23'], ['vibration', '#36c8ff']].map(([key, color]) => {
    const pts = plan[key] || [];
    if (pts.length < 2 || to <= from) return '';
    const d = slice(pts, from, to).map((p, i) =>
      `${i ? 'L' : 'M'}${(((p.atMs - from) / (to - from)) * W).toFixed(1)},${(H - 2 - p.level * (H - 4)).toFixed(1)}`).join('');
    return `<path d="${d}" fill="none" stroke="${color}" stroke-width="1.6" stroke-linejoin="round" vector-effect="non-scaling-stroke"/>`;
  }).join('');
  if (!paths) return null;
  const svg = document.createElementNS('http://www.w3.org/2000/svg', 'svg');
  svg.setAttribute('viewBox', `0 0 ${W} ${H}`);
  svg.setAttribute('preserveAspectRatio', 'none');
  svg.setAttribute('class', 'trl-tl-mini');
  svg.setAttribute('aria-hidden', 'true');
  svg.innerHTML = paths;
  return svg;
}

// Planned phases for the Custom form: one step per cycle (ramp + hold + rest).
export function customPlan(root) {
  const cycles = Math.max(1, Math.round(num(root, '#tr-cycles')));
  const per = num(root, '#tr-rampup') + num(root, '#tr-hold') + num(root, '#tr-rest');
  const phases = [];
  for (let i = 0; i < cycles; i++) phases.push({ name: `Cycle ${i + 1}`, atMs: i * per });
  return { phases, totalMs: cycles * per };
}

export function applyTrainingLayoutV1(root) {
  if (!root || root.dataset.trLayout === 'v1') return null;
  const q = sel => root.querySelector(sel);
  if (!q('.tr-layout')) return null;
  root.dataset.trLayout = 'v1';
  // Taken before anything moves: once moved into a not-yet-attached card,
  // root.querySelector no longer finds them.
  const techSel = q('#tr-technique');

  // ---- Left: Training navigation ------------------------------------------
  const nav = make('nav', 'trl-nav card');
  nav.setAttribute('aria-label', 'Training');
  nav.innerHTML = `
    <p class="trl-nav-title">Training</p>
    <p class="trl-nav-sub">Programs</p>
    <ul class="trl-nav-list" id="trl-programs"></ul>
    <p class="trl-nav-sub">My programs</p>
    <ul class="trl-nav-list" id="trl-my-programs"></ul>
    <button type="button" class="trl-nav-item" id="trl-new-program">＋ New program</button>
    <p class="trl-nav-sub">Library</p>
    <button type="button" class="trl-nav-item" id="trl-nav-history">History</button>`;

  const scriptSel = q('#tr-script');
  const pick = value => {
    if (!scriptSel) return;
    scriptSel.value = value;
    scriptSel.dispatchEvent(new Event('change'));
  };
  function renderPrograms(list) {
    const builtin = q('#trl-programs');
    const mine = q('#trl-my-programs');
    builtin.replaceChildren();
    mine.replaceChildren();
    const add = (ul, value, label) => {
      const li = make('li');
      const b = make('button', 'trl-nav-link', label);
      b.type = 'button';
      b.dataset.value = value;
      b.addEventListener('click', () => pick(value));
      li.append(b);
      ul.append(li);
    };
    add(builtin, '', 'Custom (Stop-Start / Plateau)');
    for (const s of list.filter(x => !x.custom)) {
      const opt = scriptSel?.querySelector(`option[value="${CSS.escape(s.name)}"]`);
      add(builtin, s.name, opt ? opt.textContent : s.name);
    }
    const custom = list.filter(x => x.custom);
    if (!custom.length) mine.append(make('li', 'trl-nav-empty', 'None saved yet'));
    for (const s of custom) add(mine, s.name, s.name);
    syncNav();
  }
  function syncNav() {
    const v = scriptSel ? scriptSel.value : '';
    nav.querySelectorAll('.trl-nav-link').forEach(b => b.classList.toggle('is-active', b.dataset.value === v));
  }
  async function loadPrograms() {
    try { renderPrograms(Array.isArray(await ListTrainingScripts()) ? await ListTrainingScripts() : []); }
    catch (_) { renderPrograms([]); }
  }

  // ---- Centre --------------------------------------------------------------
  const main = make('div', 'trl-main');

  // Program + phase stepper.
  const programCard = card('trl-program');
  const progCol = make('div', 'trl-program-col');
  progCol.append(make('p', 'trl-label', 'Training program'));
  if (scriptSel) progCol.append(scriptSel);
  const phaseCol = make('div', 'trl-phase-col');
  phaseCol.append(make('p', 'trl-label', 'Training phase'));
  const stepper = make('ol', 'trl-stepper');
  stepper.id = 'trl-stepper';
  phaseCol.append(stepper);
  programCard.append(progCol, phaseCol);

  // Stage: the stage switch, film, waves host, pixel clip and rings.
  const stageCard = card('trl-stage');
  for (const sel of ['.tr-stage-bar', '#tr-video-view', '#tr-pixel-stage', '#tr-rings-host']) {
    const n = q(sel);
    if (n) stageCard.append(n);
  }

  // Session bar: clock · controls · phase · time left.
  const bar = card('trl-session');
  bar.innerHTML = `
    <div class="trl-sess-time"><p class="trl-label">Session time</p>
      <p class="trl-big"><span id="trl-elapsed">0:00</span> / <span id="trl-total">–</span></p>
      <div class="trl-progress"><i id="trl-progress-fill"></i></div></div>
    <div class="trl-sess-ctl" id="trl-sess-ctl"></div>
    <div class="trl-sess-phase"><p class="trl-label">Phase</p>
      <p class="trl-big" id="trl-phase-name">–</p>
      <p class="trl-sub" id="trl-phase-step"></p></div>
    <div class="trl-sess-left"><p class="trl-label">Time left</p>
      <p class="trl-big" id="trl-left">–</p>
      <div class="trl-progress is-warm"><i id="trl-left-fill"></i></div></div>`;
  // Transport in the middle, like the Player: End · Play/Pause · Interrupt.
  // Idle, the big button is the real Start (#tr-start); while a session runs
  // it pauses and resumes it (PauseTraining / ResumeTraining: both channels
  // to 0, the session clock stands still).
  const startBtn = q('#tr-start');
  const interruptBtn = q('#tr-pause');
  const endBtn = q('#tr-stop');
  const startRow = startBtn?.parentElement;
  const playBtn = make('button', 'flow-play is-hidden');
  playBtn.type = 'button';
  playBtn.id = 'trl-play';
  playBtn.innerHTML = PLAY_ICON + PAUSE_ICON;
  if (startRow) {
    startRow.classList.add('trl-ctl-row', 'trl-transport');
    const roundBtn = (btn, icon, caption, title) => {
      btn.classList.remove('primary');
      btn.classList.add('flow-round');
      btn.title = title;
      btn.setAttribute('aria-label', title);
      btn.innerHTML = icon;
      const wrap = make('div', 'trl-tbtn');
      btn.replaceWith(wrap);
      wrap.append(btn, make('span', 'trl-cap', caption));
      return wrap;
    };
    const endWrap = endBtn ? roundBtn(endBtn, STOP_ICON, 'End', 'End session') : null;
    const intWrap = interruptBtn ? roundBtn(interruptBtn, SKIP_ICON, 'Interrupt',
      'Interrupt now (Esc): the cycle ends, the rest begins') : null;
    startBtn.classList.remove('primary');
    startBtn.classList.add('flow-play');
    startBtn.title = 'Start training';
    startBtn.setAttribute('aria-label', 'Start training');
    startBtn.innerHTML = PLAY_ICON;
    const centre = make('div', 'trl-tplay');
    centre.append(startBtn, playBtn);
    startRow.replaceChildren(...[endWrap, centre, intWrap].filter(Boolean));
    bar.querySelector('#trl-sess-ctl').append(startRow);
  }

  // Live training curves (the waves, always visible here, like the target).
  const curves = card('trl-curves');
  const waves = q('#tr-waves-host');
  if (waves) { waves.hidden = false; waves.dataset.pinned = '1'; curves.append(waves); }

  // Session timeline from the planned phases.
  const timeline = card('trl-timeline');
  timeline.innerHTML = `<div class="trl-tl-head"><p class="trl-card-title">Session timeline</p>
    <span class="trl-sub">Total <b id="trl-tl-total">–</b></span></div>
    <div class="trl-tl-track" id="trl-tl-track"><i class="trl-tl-head-mark" id="trl-tl-mark"></i></div>
    <div class="trl-tl-phases" id="trl-tl-phases"></div>`;

  main.append(programCard, stageCard, bar, curves, timeline);

  // ---- Right: settings, mode, safety, feedback, recent sessions -------------
  const side = make('aside', 'trl-side');
  const settings = card('trl-settings', 'Session settings');
  const settingsCol = q('.tr-col-settings');
  if (settingsCol) {
    // Everything that was in the settings column, minus the program select
    // (now in the program card).
    // The emptied "Preset script" row is dropped first.
    for (const row of settingsCol.querySelectorAll(':scope > .field-row')) {
      if (!row.querySelector('input, select, button, textarea') && /Preset script/.test(row.textContent)) row.remove();
    }
    while (settingsCol.firstChild) settings.append(settingsCol.firstChild);
  }
  // Session mode: technique cards for Custom (real select underneath).
  const mode = card('trl-mode', 'Session mode');
  const modes = make('div', 'trl-mode-cards');
  const modeHint = make('p', 'trl-sub trl-mode-hint');
  const MODE_TEXT = {
    stopstart: ['Stop-Start', 'Back to 0 between cycles'],
    plateau: ['Plateau', 'Stays high between peaks (edging)'],
  };
  if (techSel) {
    for (const o of techSel.options) {
      const [name] = MODE_TEXT[o.value] || [o.textContent];
      const b = make('button', 'trl-mode-card', name);
      b.type = 'button';
      b.dataset.value = o.value;
      b.addEventListener('click', () => { techSel.value = o.value; techSel.dispatchEvent(new Event('change')); syncMode(); });
      modes.append(b);
    }
  }
  mode.append(modes, modeHint);
  function syncMode() {
    const custom = !scriptSel || !scriptSel.value;
    mode.hidden = !custom || !techSel;
    if (!techSel) return;
    modes.querySelectorAll('.trl-mode-card').forEach(b => b.classList.toggle('is-active', b.dataset.value === techSel.value));
    modeHint.textContent = (MODE_TEXT[techSel.value] || ['', ''])[1];
  }
  techSel?.addEventListener('change', syncMode);

  // Safety: the real controls — interrupt (also Esc) and end the session.
  const safety = card('trl-safety', 'Safety controls');
  safety.innerHTML += `<p class="trl-sub">Esc interrupts the current cycle at any time.</p>
    <button type="button" class="trl-stop" id="trl-stop">⚠ Stop session</button>`;
  safety.querySelector('#trl-stop').addEventListener('click', () => {
    const stop = q('#tr-stop');
    if (stop && !stop.disabled) stop.click();
  });
  const feedback = card('trl-feedback');
  for (const sel of ['#tr-arousal', '.stat-row', '#tr-feedback-effect']) {
    const n = q(sel);
    if (n) feedback.append(n);
  }
  const recent = card('trl-recent', 'Recent sessions');
  const histHead = q('#tr-history-export')?.parentElement;
  if (histHead) { histHead.querySelector('h3')?.remove(); recent.append(histHead); }
  for (const sel of ['#tr-history-suggestion', '#tr-history']) {
    const n = q(sel);
    if (n) recent.append(n);
  }
  const logBox = make('details', 'card trl-log');
  logBox.append(make('summary', '', 'Session log'));
  const log = q('#tr-log');
  if (log) logBox.append(log);
  side.append(settings, mode, safety, feedback, recent, logBox);

  // Swap in the three columns; the old two-column wrapper goes.
  const layout = q('.tr-layout');
  layout.replaceWith(nav, main, side);
  // Leftover intro lines from the old live column are dropped with it.
  root.querySelector('.tr-layout')?.remove();

  // ---- Plan, clock, stepper, timeline ---------------------------------------
  let scriptPlan = window.__trPlan || null; // from tr:plan (script preview)
  let startedAt = 0;
  let timer = 0;
  let pausedAt = 0;    // performance.now() when paused, 0 = not paused
  let pausedTotal = 0; // ms spent paused this session (the clock stood still)
  const elapsedNow = () => {
    if (!startedAt) return 0;
    const now = pausedAt || performance.now();
    return now - startedAt - pausedTotal;
  };
  const plan = () => (scriptSel && scriptSel.value ? scriptPlan : customPlan(root)) || { phases: [], totalMs: 0 };

  function renderPlan() {
    const p = plan();
    stepper.replaceChildren();
    const tl = q('#trl-tl-phases');
    tl.replaceChildren();
    const track = q('#trl-tl-track');
    track.querySelectorAll('.trl-tl-seg').forEach(n => n.remove());
    q('#trl-total').textContent = p.totalMs ? fmtClock(p.totalMs) : '–';
    q('#trl-tl-total').textContent = p.totalMs ? fmtClock(p.totalMs) : '–';
    p.phases.forEach((ph, i) => {
      const end = p.phases[i + 1] ? p.phases[i + 1].atMs : p.totalMs;
      const li = make('li', 'trl-step');
      li.dataset.index = String(i);
      li.append(make('span', 'trl-step-num', String(i + 1)), make('span', 'trl-step-name', ph.name));
      stepper.append(li);
      const seg = make('i', 'trl-tl-seg');
      seg.style.left = `${p.totalMs ? (ph.atMs / p.totalMs) * 100 : 0}%`;
      seg.style.width = `${p.totalMs ? ((end - ph.atMs) / p.totalMs) * 100 : 0}%`;
      seg.dataset.index = String(i);
      track.prepend(seg);
      const tile = make('div', 'trl-tl-phase');
      tile.dataset.index = String(i);
      tile.append(make('span', 'trl-step-num', String(i + 1)), make('b', '', ph.name),
        make('small', '', `${fmtClock(ph.atMs)} – ${fmtClock(end)}`));
      // A script plan carries its curves: colour by planned strength and a
      // small curve of the phase. The Custom form has no curves, so no colour.
      if (p.vibration?.length || p.suction?.length) {
        const color = heatColor(phaseStrength(p, ph.atMs, end));
        seg.style.setProperty('--seg', color);
        tile.style.setProperty('--seg', color);
        tile.dataset.heat = '1';
        seg.dataset.heat = '1';
        const mini = miniCurve(p, ph.atMs, end);
        if (mini) tile.append(mini);
      }
      tl.append(tile);
    });
    tick();
  }

  function tick() {
    const p = plan();
    const running = startedAt > 0;
    const elapsed = elapsedNow();
    q('#trl-elapsed').textContent = fmtClock(elapsed);
    const frac = p.totalMs ? Math.min(1, elapsed / p.totalMs) : 0;
    q('#trl-progress-fill').style.width = `${frac * 100}%`;
    q('#trl-tl-mark').style.left = `${frac * 100}%`;
    q('#trl-tl-mark').hidden = !running;
    const left = p.totalMs ? Math.max(0, p.totalMs - elapsed) : 0;
    q('#trl-left').textContent = p.totalMs ? fmtClock(left) : '–';
    q('#trl-left-fill').style.width = `${p.totalMs ? (left / p.totalMs) * 100 : 0}%`;
    let cur = -1;
    if (running) p.phases.forEach((ph, i) => { if (elapsed >= ph.atMs) cur = i; });
    const name = cur >= 0 ? p.phases[cur].name : (running ? 'Starting' : 'Ready');
    q('#trl-phase-name').textContent = name;
    const step = cur >= 0 ? `Step ${cur + 1} of ${p.phases.length}` : (p.phases.length ? `${p.phases.length} steps` : '');
    q('#trl-phase-step').textContent = pausedAt ? `Paused${step ? ' · ' + step : ''}` : step;
    root.querySelectorAll('.trl-step, .trl-tl-phase, .trl-tl-seg').forEach(n => {
      const i = Number(n.dataset.index);
      n.classList.toggle('is-current', i === cur);
      n.classList.toggle('is-done', cur >= 0 && i < cur);
    });
  }

  window.addEventListener('tr:plan', e => { scriptPlan = e.detail || null; renderPlan(); });
  // Custom timings: re-plan on every change, also when saved settings are
  // applied later (the value labels change then).
  const replanCustom = () => { if (!scriptSel || !scriptSel.value) renderPlan(); };
  for (const sel of ['#tr-cycles', '#tr-rampup', '#tr-hold', '#tr-rest']) {
    q(sel)?.addEventListener('input', replanCustom);
    q(sel)?.addEventListener('change', replanCustom);
    const label = q(sel + '-val');
    if (label) new MutationObserver(replanCustom).observe(label, { childList: true, characterData: true, subtree: true });
  }
  scriptSel?.addEventListener('change', () => { syncNav(); syncMode(); renderPlan(); });

  // Running = the Start button is locked (training.js owns that state).
  function setPaused(paused) {
    if (paused && !pausedAt) pausedAt = performance.now();
    if (!paused && pausedAt) { pausedTotal += performance.now() - pausedAt; pausedAt = 0; }
    syncTransport();
    window.dispatchEvent(new CustomEvent('tr:paused', { detail: { paused: !!pausedAt } }));
    tick();
  }
  function syncTransport() {
    const running = startedAt > 0;
    startBtn?.classList.toggle('is-hidden', running);
    playBtn.classList.toggle('is-hidden', !running);
    playBtn.classList.toggle('is-playing', running && !pausedAt);
    const label = pausedAt ? 'Resume training' : 'Pause training';
    playBtn.title = label;
    playBtn.setAttribute('aria-label', label);
    root.dataset.trPaused = pausedAt ? '1' : '';
  }
  playBtn.addEventListener('click', async () => {
    if (!startedAt || playBtn.disabled) return;
    playBtn.disabled = true;
    try {
      if (pausedAt) { await ResumeTraining(); setPaused(false); }
      else { await PauseTraining(); setPaused(true); }
    } catch (_) {
      // The session ended meanwhile; the running state follows #tr-start.
    } finally {
      playBtn.disabled = false;
    }
  });
  const syncRunning = () => {
    const running = !!startBtn && startBtn.disabled;
    if (running && !startedAt) {
      startedAt = performance.now();
      pausedAt = 0;
      pausedTotal = 0;
      clearInterval(timer);
      timer = setInterval(tick, 250);
    } else if (!running && startedAt) {
      startedAt = 0;
      if (pausedAt) { pausedAt = 0; window.dispatchEvent(new CustomEvent('tr:paused', { detail: { paused: false } })); }
      clearInterval(timer);
    }
    root.dataset.trRunning = running ? '1' : '';
    syncTransport();
    tick();
  };
  if (startBtn) new MutationObserver(syncRunning).observe(startBtn, { attributes: true, attributeFilter: ['disabled'] });

  q('#trl-new-program').addEventListener('click', () => {
    const ed = q('#tr-editor');
    if (ed) { ed.open = true; ed.scrollIntoView({ block: 'nearest' }); }
    q('#tr-editor-new')?.click();
  });
  q('#trl-nav-history').addEventListener('click', () => recent.scrollIntoView({ block: 'nearest' }));

  syncMode();
  renderPlan();
  loadPrograms();
  // Saving / deleting a program changes the list.
  for (const sel of ['#tr-editor-save', '#tr-editor-delete']) {
    q(sel)?.addEventListener('click', () => setTimeout(loadPrograms, 400));
  }
  return { renderPlan, tick };
}
