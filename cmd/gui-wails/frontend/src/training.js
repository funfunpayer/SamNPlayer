import {
  StartTraining, StopTraining, StopTrainingCycle, ReportArousal, TrainingHistory,
  ListTrainingScripts, TrainingScriptPreview,
  SaveTrainingScript, DeleteTrainingScript, LoadTrainingScriptForEditing, PreviewTrainingScriptDraft,
} from '../wailsjs/go/main/App';
import { EventsOn } from '../wailsjs/runtime/runtime';
import { getSettingsCache, saveSetting } from './settings.js';
import { uiError } from './notify.js';

const TECHNIQUE_LABELS = { stopstart: 'Stop-Start', plateau: 'Plateau' };
const CHANNEL_LABELS = { vibration: 'Vibration', suction: 'Suction', both: 'Both' };

function clamp01(v) { return Math.max(0, Math.min(1, v || 0)); }

// Pixel-Figuren neben den Intensitätsbalken - eine Flamme (Vibration) und
// ein Tropfen (Suction), je ein 7x9-Raster fester Silhouette. Von unten
// nach oben werden Zeilen "beleuchtet", passend zur Intensität - dieselbe
// Zahl, die auch der Balken zeigt, nur als kleines Pixelbild statt als
// Füllstand. Als String-Zeilen statt verschachtelter Arrays, damit die
// Form beim Lesen als Bild erkennbar bleibt.
const PIXEL_FIGURES = {
  vibration: [
    '0001000',
    '0011100',
    '0111110',
    '0111110',
    '1111111',
    '1111111',
    '1111111',
    '0111110',
    '0011100',
  ],
  suction: [
    '0001000',
    '0001000',
    '0011100',
    '0111110',
    '1111111',
    '1111111',
    '1111111',
    '0111110',
    '0011100',
  ],
};

function renderPixelGrid(axisName) {
  const rows = PIXEL_FIGURES[axisName];
  const cells = rows.map((row, r) => [...row].map((bit, c) =>
    `<div class="tr-pixel-cell${bit === '1' ? ' tr-pixel-silhouette' : ''}" data-row="${r}" data-col="${c}"></div>`
  ).join('')).join('');
  return `<div class="tr-pixel-grid" id="tr-pixel-${axisName}">${cells}</div>`;
}

// Beleuchtet die Silhouette-Zellen von unten nach oben passend zu level
// (0-1) - dieselbe Zahl wie updateIntensityMeter's Balken bekommt.
function updatePixelGrid(axisName, level, pulsing) {
  const grid = document.getElementById(`tr-pixel-${axisName}`);
  if (!grid) return;
  const rows = PIXEL_FIGURES[axisName].length;
  const litRows = Math.round(clamp01(level) * rows);
  grid.querySelectorAll('.tr-pixel-cell.tr-pixel-silhouette').forEach(cell => {
    const fromBottom = rows - 1 - Number(cell.dataset.row);
    cell.classList.toggle('tr-pixel-lit', fromBottom < litRows);
    cell.classList.toggle(`tr-pixel-${axisName}`, fromBottom < litRows);
  });
  grid.classList.toggle('tr-pulsing', pulsing);
}

// Farben für die zwei Kanal-Linien in der Script-Vorschau/Live-Anzeige -
// dieselben Marken-Tokens wie der Intensitätsmesser unten (--accent/--teal),
// damit Vorschau, Live-Anzeige und Meter dieselbe Kanal-Farbe zeigen statt
// drei verschiedene Paare im selben Tab.
const VIBRATION_COLOR = 'var(--accent)';
const SUCTION_COLOR = 'var(--teal)';

// renderScriptCurveSvg zeichnet die geplante Kurve eines Scripts (siehe
// app_training.go's TrainingScriptPreview): zwei Linien (Vibration/Sog)
// über der Zeit, Phasengrenzen als gestrichelte Linien mit Namen. Reines
// SVG statt Canvas, damit es sich ohne Animationsschleife einfach neu
// zeichnen lässt (Vorschau vor dem Start, dann erneut mit angepassten
// Werten während der Session).
function renderScriptCurveSvg(preview) {
  const W = 600, H = 130, TOP = 14, BOTTOM = 116;
  const x = ms => preview.totalMs > 0 ? (ms / preview.totalMs) * W : 0;
  const y = level => BOTTOM - level * (BOTTOM - TOP);

  const toPath = points => (points || [])
    .map((p, i) => `${i === 0 ? 'M' : 'L'}${x(p.atMs).toFixed(1)},${y(p.level).toFixed(1)}`)
    .join(' ');

  const phaseLines = (preview.phaseMarkers || []).map(m => `
    <line x1="${x(m.atMs).toFixed(1)}" y1="${TOP}" x2="${x(m.atMs).toFixed(1)}" y2="${BOTTOM}"
          stroke="var(--border)" stroke-dasharray="3,3" />
    <text x="${x(m.atMs).toFixed(1) + 3}" y="${TOP + 9}" font-size="9" fill="var(--text-dim)">${m.name}</text>
  `).join('');

  return `
    <svg viewBox="0 0 ${W} ${H}" style="width:100%; height:110px; display:block;">
      <rect id="tr-phase-highlight" x="0" y="${TOP}" width="0" height="${BOTTOM - TOP}"
            fill="var(--accent, #7c9cff)" opacity="0.08" />
      ${phaseLines}
      <path d="${toPath(preview.vibration)}" fill="none" stroke="${VIBRATION_COLOR}" stroke-width="2" />
      <path d="${toPath(preview.suction)}" fill="none" stroke="${SUCTION_COLOR}" stroke-width="2" />
    </svg>
    <div class="hint" style="display:flex; gap:14px; margin-top:2px;">
      <span><span style="display:inline-block;width:10px;height:10px;background:${VIBRATION_COLOR};border-radius:2px;"></span> Vibration</span>
      <span><span style="display:inline-block;width:10px;height:10px;background:${SUCTION_COLOR};border-radius:2px;"></span> Suction</span>
    </div>
  `;
}

function formatHistoryDate(iso) {
  const d = new Date(iso);
  if (isNaN(d.getTime())) return iso || '?';
  return d.toLocaleString();
}

// Trainings-Modus: eigenständige Auf/Ab-Cycles unabhängig von einem Skript,
// angelehnt to die klinisch beschriebene Stop-Start-Methode (Semans) bzw.
// deren Plateau/Edging-Variante. Steuert ausschließlich Vibration (siehe
// player/training.go für den Hintergrund) - kein funscript nötig.
export function initTraining(root) {
  root.innerHTML = `
    <h2>Training</h2>
    <p class="hint">
      Standalone up/down cycles for stamina/control training — no
      video or script required. "Stop-start": fully to 0 between cycles.
      "Plateau": stays at a high level instead of dropping all the way
      ("edging"). Vibration and suction are both stepless, so either
      channel is free to choose. Each cycle is logged
      (Settings → log folder) so drive can be tuned later.
    </p>

    <div class="field-row"><label>Device</label>
      <span class="checkbox-row" style="margin:0"><input type="checkbox" id="tr-mock" /> <label for="tr-mock" style="width:auto">Mock (test without device)</label></span>
    </div>

    <div class="field-row"><label>Preset script</label>
      <select id="tr-script">
        <option value="">Custom (settings below)</option>
      </select>
    </div>
    <p class="hint" id="tr-script-description" style="margin-top:0; display:none;"></p>
    <div id="tr-script-preview" style="display:none; margin-bottom:10px;"></div>

    <details id="tr-editor" style="margin-bottom:12px; border:1px solid var(--border); border-radius:4px; padding:8px 10px;">
      <summary style="cursor:pointer;">Script editor — build your own multi-axis pattern</summary>
      <p class="hint">A script is a list of phases. Each phase has two independent
        <b>axes</b> — Vibration and Suction — that can ramp up/hold/down on their
        own timing. Leave an axis unchecked to keep that channel untouched
        (it stays at whatever level the previous phase left it).</p>

      <div class="field-row"><label>Load existing</label>
        <select id="tr-editor-load"><option value="">-- new script --</option></select>
      </div>
      <div class="field-row"><label>Name</label><input type="text" id="tr-editor-name" /></div>
      <div class="field-row"><label>Description</label><input type="text" id="tr-editor-description" /></div>

      <div id="tr-editor-phases"></div>
      <div class="row" style="margin-top:6px;">
        <button id="tr-editor-add-phase">+ Add phase</button>
      </div>

      <div id="tr-editor-preview" style="margin-top:10px;"></div>
      <div id="tr-editor-status" class="hint"></div>
      <div class="row" style="margin-top:6px;">
        <button id="tr-editor-save" class="primary">Save script</button>
        <button id="tr-editor-delete" disabled>Delete</button>
        <button id="tr-editor-new">New</button>
      </div>
    </details>

    <div id="tr-manual-fields">
      <div class="field-row"><label>Technique</label>
        <select id="tr-technique">
          <option value="stopstart">Stop-start (to 0 between cycles)</option>
          <option value="plateau">Plateau (stays high, "edging")</option>
        </select>
      </div>
      <div class="field-row"><label>Channel</label>
        <select id="tr-channel">
          <option value="vibration">Vibration only</option>
          <option value="suction">Suction only</option>
          <option value="both">Both</option>
        </select>
      </div>
      <div class="field-row"><label>Cycles</label><input type="number" min="1" id="tr-cycles" value="5" /></div>
      <div class="field-row"><label>Ramp up (ms)</label><input type="number" min="0" step="500" id="tr-rampup" value="8000" /></div>
      <div class="field-row"><label>Hold (ms)</label><input type="number" min="0" step="500" id="tr-hold" value="3000" /></div>
      <div class="field-row"><label>Rest (ms)</label><input type="number" min="0" step="500" id="tr-rest" value="10000" /></div>
      <div class="field-row"><label>Peak intensity</label><input type="number" min="0" max="1" step="0.05" id="tr-peak" value="0.8" /></div>
      <div class="field-row" id="tr-plateau-row"><label>Plateau fraction</label><input type="number" min="0" max="1" step="0.05" id="tr-plateaufrac" value="0.7" /></div>
      <div class="field-row"><label>Progression per cycle</label><input type="number" min="0" max="1" step="0.05" id="tr-progression" value="0.15" /></div>
    </div>

    <div class="row">
      <button id="tr-start" class="primary">▶ Start training</button>
      <button id="tr-pause" class="primary" disabled>Interrupt now</button>
      <button id="tr-stop" disabled>■ End session</button>
    </div>

    <fieldset id="tr-arousal" disabled style="margin-top:12px; border:1px solid var(--border);
              border-radius:4px; padding:10px;">
      <legend style="padding:0 6px;">Feedback</legend>
      <p class="hint" style="margin-top:0;">How close are you right now? Your rating affects the
        <em>next</em> cycle: high values lead to shorter, gentler cycles with longer
        rest. Target is 7 — close, but with margin.</p>
      <div class="row" id="tr-arousal-buttons" style="flex-wrap:wrap; gap:4px;"></div>
      <div id="tr-arousal-status" class="hint" style="margin-top:6px;"></div>
    </div>

    <div class="stat-row">
      <span>Cycle: <b id="tr-cycle-label">-</b></span>
      <span>Current peak: <b id="tr-peak-label">-</b></span>
    </div>
    <div class="tr-meter">
      <div class="tr-meter-row">
        ${renderPixelGrid('vibration')}
        <span class="tr-meter-label">Vibration</span>
        <div class="tr-meter-track"><div class="tr-meter-fill tr-meter-vibration" id="tr-meter-vibration"></div></div>
      </div>
      <div class="tr-meter-row">
        ${renderPixelGrid('suction')}
        <span class="tr-meter-label">Suction</span>
        <div class="tr-meter-track"><div class="tr-meter-fill tr-meter-suction" id="tr-meter-suction"></div></div>
      </div>
    </div>
    <div id="tr-feedback-effect" class="hint" style="min-height:1.2em;"></div>
    <div id="tr-log" style="background:var(--bg-alt); border:1px solid var(--border); border-radius:4px; padding:8px; height:100px; overflow-y:auto; font-family:monospace; font-size:11px; color:var(--text-dim); white-space:pre-wrap;"></div>

    <h3 style="margin-top:16px;">History</h3>
    <p class="hint" style="margin-top:0;">Jede Session wird mitgeschrieben (siehe oben) -
      hier eine Zeile je vergangener Session, neueste zuerst.</p>
    <div id="tr-history" class="hint">Loading…</div>
  `;

  const el = id => root.querySelector(id);
  let running = false;

  function log(line) {
    const box = el('#tr-log');
    box.textContent += (box.textContent ? '\n' : '') + line;
    box.scrollTop = box.scrollHeight;
  }

  function setRunningState(isRunning) {
    running = isRunning;
    el('#tr-start').disabled = isRunning;
    el('#tr-stop').disabled = !isRunning;
    el('#tr-pause').disabled = !isRunning;
    el('#tr-arousal').disabled = !isRunning;
    if (!isRunning) {
      el('#tr-arousal-status').textContent = '';
      updateIntensityMeter({ vibration: 0, suction: 0 });
    }
  }

  // Zwei pulsierende Balken statt nur Text im stat-row - "etwas zum
  // Nachvollziehen" für die laufende Intensität. running (Closure-Variable
  // oben) entscheidet, ob die Balken pulsieren oder nur ihre Füllhöhe
  // zeigen (z.B. beim Zurücksetzen nach Sessionende).
  function updateIntensityMeter({ vibration = 0, suction = 0 }) {
    const setBar = (id, level) => {
      const bar = el(id);
      const pct = Math.round(clamp01(level) * 100);
      bar.style.width = pct + '%';
      bar.classList.toggle('tr-pulsing', running && pct > 0);
    };
    setBar('#tr-meter-vibration', vibration);
    setBar('#tr-meter-suction', suction);
    updatePixelGrid('vibration', vibration, running && vibration > 0);
    updatePixelGrid('suction', suction, running && suction > 0);
  }


  function updateTechniqueVisibility() {
    el('#tr-plateau-row').style.display = el('#tr-technique').value === 'plateau' ? 'flex' : 'none';
  }

  // Script-Vorschau: Presets aus player.BuiltinTrainingScripts() (siehe
  // ListTrainingScripts) statt der einfachen Technik/Kanal-Form. Bei
  // "Custom" bleibt das alte Formular unverändert die einzige Quelle.
  let scriptPreviews = {}; // name -> TrainingScriptPreviewResult, lazily geladen
  let currentPhaseMarkers = [];
  let currentTotalMs = 0;

  async function loadScripts() {
    try {
      const scripts = await ListTrainingScripts();
      if (!Array.isArray(scripts)) return;
      const select = el('#tr-script');
      // Alles außer "Custom" (erste Option) neu aufbauen - sonst verdoppeln
      // sich die Einträge, wenn loadScripts() nach dem Speichern eines
      // eigenen Scripts erneut läuft.
      while (select.options.length > 1) select.remove(1);
      for (const s of scripts) {
        const opt = document.createElement('option');
        opt.value = s.name;
        opt.textContent = (s.custom ? '★ ' : '') + s.name.replace(/-/g, ' ');
        opt.title = s.description;
        select.appendChild(opt);
      }
      renderEditorLoadOptions(scripts.filter(s => s.custom));
    } catch (err) {
      log('Could not load preset scripts: ' + err);
    }
  }

  async function updateScriptVisibility() {
    const name = el('#tr-script').value;
    el('#tr-manual-fields').style.display = name ? 'none' : '';
    const descBox = el('#tr-script-description');
    const previewBox = el('#tr-script-preview');
    if (!name) {
      descBox.style.display = 'none';
      previewBox.style.display = 'none';
      return;
    }
    const scripts = await ListTrainingScripts().catch(() => []);
    const info = scripts.find(s => s.name === name);
    descBox.textContent = info ? info.description : '';
    descBox.style.display = info ? '' : 'none';

    if (!scriptPreviews[name]) {
      try {
        scriptPreviews[name] = await TrainingScriptPreview(name);
      } catch (err) {
        previewBox.style.display = 'none';
        log('Could not load preview: ' + err);
        return;
      }
    }
    const preview = scriptPreviews[name];
    currentPhaseMarkers = preview.phaseMarkers || [];
    currentTotalMs = preview.totalMs || 0;
    previewBox.innerHTML = renderScriptCurveSvg(preview);
    previewBox.style.display = '';
  }

  // Hebt die gerade laufende Phase in der Vorschau hervor (siehe
  // renderScriptCurveSvg's #tr-phase-highlight-Rechteck) - ein grobes,
  // aber einfaches "wo bin ich gerade", ohne die Wiederholungen selbst
  // auf die Millisekunde zu synchronisieren.
  function highlightPhase(phaseIndex) {
    const rect = root.querySelector('#tr-phase-highlight');
    if (!rect || currentTotalMs <= 0 || !currentPhaseMarkers[phaseIndex]) return;
    const startMs = currentPhaseMarkers[phaseIndex].atMs;
    const endMs = currentPhaseMarkers[phaseIndex + 1] ? currentPhaseMarkers[phaseIndex + 1].atMs : currentTotalMs;
    const W = 600;
    const x1 = (startMs / currentTotalMs) * W;
    const x2 = (endMs / currentTotalMs) * W;
    rect.setAttribute('x', x1.toFixed(1));
    rect.setAttribute('width', Math.max(0, x2 - x1).toFixed(1));
  }

  // ---- Script editor: build a custom multi-axis (Vibration/Suction)
  // script instead of only picking a built-in preset. "Axis" is the
  // owner's own term for it - each phase has up to two independent
  // axes, each with its own ramp/hold/ramp-down shape (see
  // player.ChannelCurve / docs/TRAINING_MODE_RESEARCH.md).

  function defaultAxis() {
    return { enabled: false, startLevel: 0.2, peakLevel: 0.6, endLevel: 0.2, rampUpMs: 3000, holdMs: 2000, rampDownMs: 3000, randomJitterFraction: 0 };
  }
  function defaultPhase() {
    return { name: 'Phase', repeatCycles: 3, restMs: 3000, vibration: { ...defaultAxis(), enabled: true }, suction: defaultAxis() };
  }
  function blankEditorScript() {
    return { name: '', description: '', progressionPerCycle: 0, phases: [defaultPhase()] };
  }

  let editorScript = blankEditorScript();
  let editingCustomName = null; // null = neues Script, sonst der Name des geladenen (zum Löschen/Überschreiben)
  let previewDebounce = null;

  function renderEditorLoadOptions(customScripts) {
    const select = el('#tr-editor-load');
    const current = select.value;
    select.innerHTML = '<option value="">-- new script --</option>';
    for (const s of customScripts) {
      const opt = document.createElement('option');
      opt.value = s.name;
      opt.textContent = s.name;
      select.appendChild(opt);
    }
    select.value = customScripts.some(s => s.name === current) ? current : '';
  }

  // axisFieldsHtml baut die Zahlenfelder EINER Achse - identisch für
  // Vibration und Suction, nur data-axis unterscheidet sie.
  function axisFieldsHtml(axisName, axis, label) {
    const dis = axis.enabled ? '' : 'disabled';
    const field = (f, v, step, min, max) =>
      `<input type="number" data-axis="${axisName}" data-field="${f}" value="${v}" step="${step}" min="${min}" ${max !== undefined ? `max="${max}"` : ''} ${dis} style="width:70px;" />`;
    return `
      <div data-axis-block="${axisName}" style="border:1px solid var(--border); border-radius:4px; padding:6px; flex:1; min-width:260px;">
        <label style="display:flex; align-items:center; gap:6px; font-weight:600;">
          <input type="checkbox" data-axis="${axisName}" data-field="enabled" ${axis.enabled ? 'checked' : ''} /> ${label} axis
        </label>
        <div style="display:${axis.enabled ? 'grid' : 'none'}; grid-template-columns:repeat(3,auto); gap:4px 10px; margin-top:6px; font-size:11px;" data-axis-fields="${axisName}">
          <span>Start ${field('startLevel', axis.startLevel, 0.05, 0, 1)}</span>
          <span>Peak ${field('peakLevel', axis.peakLevel, 0.05, 0, 1)}</span>
          <span>End ${field('endLevel', axis.endLevel, 0.05, 0, 1)}</span>
          <span>Ramp up ms ${field('rampUpMs', axis.rampUpMs, 100, 0)}</span>
          <span>Hold ms ${field('holdMs', axis.holdMs, 100, 0)}</span>
          <span>Ramp down ms ${field('rampDownMs', axis.rampDownMs, 100, 0)}</span>
          <span>Jitter ${field('randomJitterFraction', axis.randomJitterFraction, 0.05, 0, 1)}</span>
        </div>
      </div>`;
  }

  function renderEditorPhases() {
    el('#tr-editor-phases').innerHTML = editorScript.phases.map((phase, i) => `
      <div class="tr-editor-phase" data-phase-index="${i}" style="border:1px solid var(--border); border-radius:4px; padding:8px; margin-top:8px;">
        <div class="row" style="align-items:center; flex-wrap:wrap; gap:8px;">
          <input type="text" data-field="name" value="${phase.name.replace(/"/g, '&quot;')}" placeholder="Phase name" style="flex:1; min-width:120px;" />
          <label>Repeats <input type="number" data-field="repeatCycles" value="${phase.repeatCycles}" min="1" style="width:55px;" /></label>
          <label>Rest ms <input type="number" data-field="restMs" value="${phase.restMs}" min="0" step="100" style="width:75px;" /></label>
          <button data-action="remove-phase" ${editorScript.phases.length <= 1 ? 'disabled' : ''}>✕</button>
        </div>
        <div class="row" style="margin-top:6px; flex-wrap:wrap;">
          ${axisFieldsHtml('vibration', phase.vibration, 'Vibration')}
          ${axisFieldsHtml('suction', phase.suction, 'Suction')}
        </div>
      </div>
    `).join('');
  }

  function collectDraftScript() {
    return {
      name: editorScript.name,
      description: editorScript.description,
      progressionPerCycle: editorScript.progressionPerCycle,
      phases: editorScript.phases.map(p => ({
        name: p.name,
        repeatCycles: p.repeatCycles,
        restMs: p.restMs,
        vibration: p.vibration.enabled ? { ...p.vibration, channel: 'vibration' } : null,
        suction: p.suction.enabled ? { ...p.suction, channel: 'suction' } : null,
      })),
    };
  }

  function scheduleEditorPreview() {
    clearTimeout(previewDebounce);
    previewDebounce = setTimeout(refreshEditorPreview, 250);
  }

  async function refreshEditorPreview() {
    const box = el('#tr-editor-preview');
    try {
      const preview = await PreviewTrainingScriptDraft(collectDraftScript());
      box.innerHTML = renderScriptCurveSvg(preview);
      el('#tr-editor-status').textContent = '';
    } catch (err) {
      box.innerHTML = '';
      el('#tr-editor-status').textContent = 'Not valid yet: ' + err;
    }
  }

  function loadEditorScript(script, customName) {
    editorScript = {
      name: script.name || '',
      description: script.description || '',
      progressionPerCycle: script.progressionPerCycle || 0,
      phases: (script.phases || []).map(p => ({
        name: p.name || 'Phase',
        repeatCycles: p.repeatCycles || 1,
        restMs: p.restMs || 0,
        vibration: p.vibration ? { ...defaultAxis(), ...p.vibration, enabled: true } : defaultAxis(),
        suction: p.suction ? { ...defaultAxis(), ...p.suction, enabled: true } : defaultAxis(),
      })),
    };
    if (editorScript.phases.length === 0) editorScript.phases.push(defaultPhase());
    editingCustomName = customName;
    el('#tr-editor-name').value = editorScript.name;
    el('#tr-editor-description').value = editorScript.description;
    el('#tr-editor-delete').disabled = !customName;
    renderEditorPhases();
    scheduleEditorPreview();
  }

  el('#tr-editor-phases').addEventListener('input', e => {
    const t = e.target;
    const phaseEl = t.closest('.tr-editor-phase');
    if (!phaseEl || !t.dataset.field) return;
    const phase = editorScript.phases[Number(phaseEl.dataset.phaseIndex)];
    const axisName = t.dataset.axis;
    const target = axisName ? phase[axisName] : phase;
    const value = t.type === 'checkbox' ? t.checked
      : t.type === 'number' ? (parseFloat(t.value) || 0)
      : t.value;
    target[t.dataset.field] = value;
    if (axisName && t.dataset.field === 'enabled') {
      const fields = phaseEl.querySelector(`[data-axis-fields="${axisName}"]`);
      fields.style.display = value ? 'grid' : 'none';
      fields.querySelectorAll('input').forEach(inp => inp.disabled = !value);
    }
    scheduleEditorPreview();
  });

  el('#tr-editor-phases').addEventListener('click', e => {
    if (e.target.dataset.action !== 'remove-phase') return;
    const phaseEl = e.target.closest('.tr-editor-phase');
    editorScript.phases.splice(Number(phaseEl.dataset.phaseIndex), 1);
    renderEditorPhases();
    scheduleEditorPreview();
  });

  el('#tr-editor-add-phase').addEventListener('click', () => {
    editorScript.phases.push(defaultPhase());
    renderEditorPhases();
    scheduleEditorPreview();
  });

  el('#tr-editor-name').addEventListener('input', e => { editorScript.name = e.target.value; });
  el('#tr-editor-description').addEventListener('input', e => { editorScript.description = e.target.value; });

  el('#tr-editor-new').addEventListener('click', () => loadEditorScript(blankEditorScript(), null));

  el('#tr-editor-load').addEventListener('change', async e => {
    const name = e.target.value;
    if (!name) { loadEditorScript(blankEditorScript(), null); return; }
    try {
      const script = await LoadTrainingScriptForEditing(name);
      loadEditorScript(script, name);
    } catch (err) {
      el('#tr-editor-status').textContent = 'Could not load: ' + err;
    }
  });

  el('#tr-editor-save').addEventListener('click', async () => {
    try {
      const info = await SaveTrainingScript(collectDraftScript());
      editingCustomName = info.name;
      el('#tr-editor-delete').disabled = false;
      el('#tr-editor-status').textContent = `Saved as "${info.name}".`;
      await loadScripts();
      renderEditorLoadOptions((await ListTrainingScripts()).filter(s => s.custom));
      el('#tr-editor-load').value = info.name;
    } catch (err) {
      el('#tr-editor-status').textContent = 'Could not save: ' + err;
    }
  });

  el('#tr-editor-delete').addEventListener('click', async () => {
    if (!editingCustomName) return;
    try {
      await DeleteTrainingScript(editingCustomName);
      el('#tr-editor-status').textContent = `Deleted "${editingCustomName}".`;
      loadEditorScript(blankEditorScript(), null);
      await loadScripts();
    } catch (err) {
      el('#tr-editor-status').textContent = 'Could not delete: ' + err;
    }
  });

  async function refreshHistory() {
    const box = el('#tr-history');
    try {
      const history = await TrainingHistory();
      if (!Array.isArray(history) || history.length === 0) {
        box.textContent = 'No completed session yet.';
        return;
      }
      box.innerHTML = history.map(s => {
        const technique = TECHNIQUE_LABELS[s.technique] || s.technique;
        const channel = CHANNEL_LABELS[s.channel] || s.channel;
        const stopped = s.cyclesStoppedEarly > 0
          ? `, ${s.cyclesStoppedEarly}× interrupted` : '';
        const arousal = s.arousalReportsCount > 0
          ? `, avg feedback ${s.meanArousalReported.toFixed(1)}` : '';
        return `<div>${formatHistoryDate(s.startedAt)} — ${technique}/${channel}: `
          + `${s.cyclesCompleted} cycles, avg peak ${Math.round(s.meanPeakIntensity * 100)}%`
          + `${stopped}${arousal}</div>`;
      }).join('');
    } catch (err) {
      box.textContent = 'Could not load history: ' + err;
    }
  }

  async function start() {
    el('#tr-log').textContent = '';
    el('#tr-feedback-effect').textContent = '';
    updateIntensityMeter({ vibration: 0, suction: 0 });
    const scriptName = el('#tr-script').value;
    const req = scriptName
      ? { mock: el('#tr-mock').checked, scriptName }
      : {
          mock: el('#tr-mock').checked,
          technique: el('#tr-technique').value,
          channel: el('#tr-channel').value,
          cycles: parseInt(el('#tr-cycles').value, 10) || 1,
          rampUpMs: parseInt(el('#tr-rampup').value, 10) || 0,
          holdMs: parseInt(el('#tr-hold').value, 10) || 0,
          restMs: parseInt(el('#tr-rest').value, 10) || 0,
          peakIntensity: parseFloat(el('#tr-peak').value) || 0,
          plateauFraction: parseFloat(el('#tr-plateaufrac').value) || 0,
          progressionPerCycle: parseFloat(el('#tr-progression').value) || 0,
        };
    try {
      await StartTraining(req);
    } catch (err) {
      uiError('Start training: ' + err, el('#tr-log'));
      return;
    }
    setRunningState(true);
  }

  async function stop() {
    await StopTraining();
    setRunningState(false);
  }

  EventsOn('training:log', log);
  EventsOn('training:error', msg => log('ERROR: ' + msg));
  EventsOn('training:done', () => { setRunningState(false); log('Training finished.'); refreshHistory(); });
  EventsOn('training:cycle', c => {
    el('#tr-cycle-label').textContent = `${c.cycleIndex + 1} / ${c.cyclesTotal}`;
    el('#tr-peak-label').textContent = Math.round(c.peakIntensity * 100) + '%';
    const reached = c.reachedPeakAfterMs ? `, erreicht nach ${(c.reachedPeakAfterMs / 1000).toFixed(1)}s` : '';
    const stopped = c.stoppedByUser ? ' — auf Wunsch unterbrochen' : '';
    const fb = c.arousalBefore ? `, angepasst nach Feedback ${c.arousalBefore}` : '';
    log(`Zyklus ${c.cycleIndex + 1}/${c.cyclesTotal}: Spitze ${Math.round(c.peakIntensity * 100)}%, Halten ${c.holdMs}ms${fb}${reached}${stopped}`);

    // Die einfache Technik/Kanal-Form kennt den Kanal nicht im Event
    // selbst (nur EIN peakIntensity), sondern über das eigene Formularfeld.
    const channel = el('#tr-channel').value;
    updateIntensityMeter({
      vibration: (channel === 'vibration' || channel === 'both') ? c.peakIntensity : 0,
      suction: (channel === 'suction' || channel === 'both') ? c.peakIntensity : 0,
    });
  });

  // Script-Sessions melden pro Kanal statt einer einzelnen Kurve - siehe
  // app_training.go's emitTrainingScriptCycle. Die Feedback-Effekt-Zeile
  // beantwortet direkt die Sorge, ob die Rückmeldung wirklich beide
  // Kanäle schwächt und die Pause verlängert, statt es nur zu behaupten.
  EventsOn('training:scriptCycle', c => {
    el('#tr-cycle-label').textContent = `${c.phaseName} (${c.repeatIndex + 1}/${c.repeatsTotal}), phase ${c.phaseIndex + 1}/${c.phasesTotal}`;
    const peaks = [];
    if (c.vibrationPeak > 0) peaks.push(`Vibration ${Math.round(c.vibrationPeak * 100)}%`);
    if (c.suctionPeak > 0) peaks.push(`Suction ${Math.round(c.suctionPeak * 100)}%`);
    el('#tr-peak-label').textContent = peaks.join(', ') || '-';
    highlightPhase(c.phaseIndex);
    updateIntensityMeter({ vibration: c.vibrationPeak, suction: c.suctionPeak });

    const stopped = c.stoppedByUser ? ' — interrupted on request' : '';
    log(`${c.phaseName} ${c.repeatIndex + 1}/${c.repeatsTotal}: ${peaks.join(', ') || '(no channel)'}, rest ${c.restMs}ms${stopped}`);

    if (c.arousalBefore) {
      const peakPct = Math.round((c.peakFactorApplied - 1) * 100);
      const restPct = Math.round((c.restFactorApplied - 1) * 100);
      el('#tr-feedback-effect').textContent =
        `Feedback ${c.arousalBefore} → peak ${peakPct >= 0 ? '+' : ''}${peakPct}%, `
        + `rest ${restPct >= 0 ? '+' : ''}${restPct}% — applied to both channels.`;
    } else {
      el('#tr-feedback-effect').textContent = '';
    }
  });

  // Skala 1-10 aufbauen.
  const scale = el('#tr-arousal-buttons');
  for (let i = 1; i <= 10; i++) {
    const btn = document.createElement('button');
    btn.textContent = i;
    btn.style.minWidth = '38px';
    if (i === 7) btn.classList.add('primary');   // Zielwert hervorheben
    btn.addEventListener('click', async () => {
      try {
        await ReportArousal(i);
        el('#tr-arousal-status').textContent =
          `${i} reported — affects the next cycle.`;
      } catch (err) {
        el('#tr-arousal-status').textContent = 'Not applied: ' + err;
      }
    });
    scale.appendChild(btn);
  }

  el('#tr-start').addEventListener('click', start);
  el('#tr-stop').addEventListener('click', stop);

  // Unterbricht nur den laufenden Zyklus - die Session geht danach weiter.
  // Das ist der Kern der Stop-Start-Methode: nicht eine Stoppuhr entscheidet,
  // wann unterbrochen wird, sondern du.
  el('#tr-pause').addEventListener('click', async () => {
    try {
      await StopTrainingCycle();
      log('Cycle interrupted — pause running, then continues.');
    } catch (err) {
      log('Could not interrupt: ' + err);
    }
  });
  el('#tr-technique').addEventListener('change', updateTechniqueVisibility);
  updateTechniqueVisibility();
  refreshHistory();

  el('#tr-script').addEventListener('change', updateScriptVisibility);
  loadScripts().then(updateScriptVisibility);

  renderEditorPhases();
  scheduleEditorPreview();

  getSettingsCache().then(s => {
    el('#tr-mock').checked = s.trainingMock;
    el('#tr-technique').value = s.trainingTechnique;
    el('#tr-channel').value = s.trainingChannel;
    el('#tr-cycles').value = s.trainingCycles;
    el('#tr-rampup').value = s.trainingRampUpMs;
    el('#tr-hold').value = s.trainingHoldMs;
    el('#tr-rest').value = s.trainingRestMs;
    el('#tr-peak').value = s.trainingPeakIntensity;
    el('#tr-plateaufrac').value = s.trainingPlateauFraction;
    el('#tr-progression').value = s.trainingProgressionPerCycle;
    updateTechniqueVisibility();
  });
  el('#tr-mock').addEventListener('change', e => saveSetting('training.mock', e.target.checked));
  el('#tr-technique').addEventListener('change', e => saveSetting('training.technique', e.target.value));
  el('#tr-channel').addEventListener('change', e => saveSetting('training.channel', e.target.value));
  el('#tr-cycles').addEventListener('change', e => saveSetting('training.cycles', parseFloat(e.target.value)));
  el('#tr-rampup').addEventListener('change', e => saveSetting('training.ramp_up_ms', parseFloat(e.target.value)));
  el('#tr-hold').addEventListener('change', e => saveSetting('training.hold_ms', parseFloat(e.target.value)));
  el('#tr-rest').addEventListener('change', e => saveSetting('training.rest_ms', parseFloat(e.target.value)));
  el('#tr-peak').addEventListener('change', e => saveSetting('training.peak_intensity', parseFloat(e.target.value)));
  el('#tr-plateaufrac').addEventListener('change', e => saveSetting('training.plateau_fraction', parseFloat(e.target.value)));
  el('#tr-progression').addEventListener('change', e => saveSetting('training.progression_per_cycle', parseFloat(e.target.value)));
}
