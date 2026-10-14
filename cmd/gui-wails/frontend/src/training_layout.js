// V1 Training layout (Owner 14 Oct, target docs/visual-target/training.png):
//   left   — Training: programs (built-in + my programs), custom, history
//   centre — program + phase stepper, stage, session bar, live curves,
//            session timeline
//   right  — session settings, session mode, safety, feedback, recent sessions
// Existing elements are moved, not re-created (IDs, listeners and tests
// stay). New parts only show real data: the planned phases and total come
// from TrainingScriptPreview (tr:plan) or the Custom timings, the clock runs
// while the training runs, recent sessions are the training history.
import { ListTrainingScripts } from '../wailsjs/go/main/App';

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

export function fmtClock(ms) {
  const s = Math.max(0, Math.round((ms || 0) / 1000));
  const h = Math.floor(s / 3600), m = Math.floor((s % 3600) / 60), sec = s % 60;
  const mm = String(m).padStart(h ? 2 : 1, '0');
  return (h ? h + ':' : '') + mm + ':' + String(sec).padStart(2, '0');
}

const num = (root, sel) => parseFloat(root.querySelector(sel)?.value) || 0;

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
  const startRow = q('#tr-start')?.parentElement;
  if (startRow) { startRow.classList.add('trl-ctl-row'); bar.querySelector('#trl-sess-ctl').append(startRow); }

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
      tl.append(tile);
    });
    tick();
  }

  function tick() {
    const p = plan();
    const running = startedAt > 0;
    const elapsed = running ? performance.now() - startedAt : 0;
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
    q('#trl-phase-step').textContent = cur >= 0 ? `Step ${cur + 1} of ${p.phases.length}` : (p.phases.length ? `${p.phases.length} steps` : '');
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
  const startBtn = q('#tr-start');
  const syncRunning = () => {
    const running = !!startBtn && startBtn.disabled;
    if (running && !startedAt) {
      startedAt = performance.now();
      clearInterval(timer);
      timer = setInterval(tick, 250);
    } else if (!running && startedAt) {
      startedAt = 0;
      clearInterval(timer);
    }
    root.dataset.trRunning = running ? '1' : '';
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
