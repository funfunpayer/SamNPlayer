import { StartTraining, StopTraining, StopTrainingCycle, ReportArousal, TrainingHistory } from '../wailsjs/go/main/App';

const TECHNIQUE_LABELS = { stopstart: 'Stop-Start', plateau: 'Plateau' };
const CHANNEL_LABELS = { vibration: 'Vibration', suction: 'Sog', both: 'Beide' };

function formatHistoryDate(iso) {
  const d = new Date(iso);
  if (isNaN(d.getTime())) return iso || '?';
  return d.toLocaleString();
}
import { EventsOn } from '../wailsjs/runtime/runtime';
import { getSettingsCache, saveSetting } from './settings.js';

// Trainings-Modus: eigenständige Auf/Ab-Zyklen unabhängig von einem Skript,
// angelehnt an die klinisch beschriebene Stop-Start-Methode (Semans) bzw.
// deren Plateau/Edging-Variante. Steuert ausschließlich Vibration (siehe
// player/training.go für den Hintergrund) - kein funscript nötig.
export function initTraining(root) {
  root.innerHTML = `
    <h2>Training</h2>
    <p class="hint">
      Eigenständige Auf/Ab-Zyklen zum Ausdauer-/Kontrolltraining - kein
      Video/Skript nötig. "Stop-Start": komplett auf 0 zwischen den Zyklen.
      "Plateau": bleibt auf einem hohen Niveau statt ganz abzufallen
      ("Edging"). Vibration und Sog sind beide stufenlos steuerbar, darum
      frei als Kanal wählbar. Jeder Zyklus wird mitgeschrieben
      (Einstellungen-Tab → Log-Ordner), um die Ansteuerung später
      feinabstimmen zu können.
    </p>

    <div class="field-row"><label>Gerät</label>
      <span class="checkbox-row" style="margin:0"><input type="checkbox" id="tr-mock" /> <label for="tr-mock" style="width:auto">Mock (ohne Gerät testen)</label></span>
    </div>
    <div class="field-row"><label>Technik</label>
      <select id="tr-technique">
        <option value="stopstart">Stop-Start (auf 0 zwischen Zyklen)</option>
        <option value="plateau">Plateau (bleibt oben, "Edging")</option>
      </select>
    </div>
    <div class="field-row"><label>Kanal</label>
      <select id="tr-channel">
        <option value="vibration">Nur Vibration</option>
        <option value="suction">Nur Sog</option>
        <option value="both">Beide</option>
      </select>
    </div>
    <div class="field-row"><label>Zyklen</label><input type="number" min="1" id="tr-cycles" value="5" /></div>
    <div class="field-row"><label>Hochfahren (ms)</label><input type="number" min="0" step="500" id="tr-rampup" value="8000" /></div>
    <div class="field-row"><label>Halten (ms)</label><input type="number" min="0" step="500" id="tr-hold" value="3000" /></div>
    <div class="field-row"><label>Pause/Absenken (ms)</label><input type="number" min="0" step="500" id="tr-rest" value="10000" /></div>
    <div class="field-row"><label>Höchst-Intensität</label><input type="number" min="0" max="1" step="0.05" id="tr-peak" value="0.8" /></div>
    <div class="field-row" id="tr-plateau-row"><label>Plateau-Anteil</label><input type="number" min="0" max="1" step="0.05" id="tr-plateaufrac" value="0.7" /></div>
    <div class="field-row"><label>Steigerung/Zyklus</label><input type="number" min="0" max="1" step="0.05" id="tr-progression" value="0.15" /></div>

    <div class="row">
      <button id="tr-start" class="primary">▶ Training starten</button>
      <button id="tr-pause" class="primary" disabled>Jetzt unterbrechen</button>
      <button id="tr-stop" disabled>■ Session beenden</button>
    </div>

    <fieldset id="tr-arousal" disabled style="margin-top:12px; border:1px solid var(--border);
              border-radius:4px; padding:10px;">
      <legend style="padding:0 6px;">Rückmeldung</legend>
      <p class="hint" style="margin-top:0;">Wie nah bist du gerade? Die Angabe wirkt auf den
        <em>nächsten</em> Zyklus: hohe Werte führen zu kürzeren, sanfteren Zyklen mit längerer
        Pause. Ziel ist 7 — nah dran, aber mit Abstand.</p>
      <div class="row" id="tr-arousal-buttons" style="flex-wrap:wrap; gap:4px;"></div>
      <div id="tr-arousal-status" class="hint" style="margin-top:6px;"></div>
    </div>

    <div class="stat-row">
      <span>Zyklus: <b id="tr-cycle-label">-</b></span>
      <span>Aktuelle Spitze: <b id="tr-peak-label">-</b></span>
    </div>
    <div id="tr-log" style="background:var(--bg-alt); border:1px solid var(--border); border-radius:4px; padding:8px; height:100px; overflow-y:auto; font-family:monospace; font-size:11px; color:var(--text-dim); white-space:pre-wrap;"></div>

    <h3 style="margin-top:16px;">Verlauf</h3>
    <p class="hint" style="margin-top:0;">Jede Session wird mitgeschrieben (siehe oben) -
      hier eine Zeile je vergangener Session, neueste zuerst.</p>
    <div id="tr-history" class="hint">Lädt...</div>
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
    if (!isRunning) el('#tr-arousal-status').textContent = '';
  }

  function updateTechniqueVisibility() {
    el('#tr-plateau-row').style.display = el('#tr-technique').value === 'plateau' ? 'flex' : 'none';
  }

  async function refreshHistory() {
    const box = el('#tr-history');
    try {
      const history = await TrainingHistory();
      if (!Array.isArray(history) || history.length === 0) {
        box.textContent = 'Noch keine abgeschlossene Session.';
        return;
      }
      box.innerHTML = history.map(s => {
        const technique = TECHNIQUE_LABELS[s.technique] || s.technique;
        const channel = CHANNEL_LABELS[s.channel] || s.channel;
        const stopped = s.cyclesStoppedEarly > 0
          ? `, ${s.cyclesStoppedEarly}x unterbrochen` : '';
        const arousal = s.arousalReportsCount > 0
          ? `, Ø-Rückmeldung ${s.meanArousalReported.toFixed(1)}` : '';
        return `<div>${formatHistoryDate(s.startedAt)} — ${technique}/${channel}: `
          + `${s.cyclesCompleted} Zyklen, Ø-Spitze ${Math.round(s.meanPeakIntensity * 100)}%`
          + `${stopped}${arousal}</div>`;
      }).join('');
    } catch (err) {
      box.textContent = 'Verlauf konnte nicht geladen werden: ' + err;
    }
  }

  async function start() {
    el('#tr-log').textContent = '';
    const req = {
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
      alert('Fehler: ' + err);
      return;
    }
    setRunningState(true);
  }

  async function stop() {
    await StopTraining();
    setRunningState(false);
  }

  EventsOn('training:log', log);
  EventsOn('training:error', msg => log('FEHLER: ' + msg));
  EventsOn('training:done', () => { setRunningState(false); log('Training beendet.'); refreshHistory(); });
  EventsOn('training:cycle', c => {
    el('#tr-cycle-label').textContent = `${c.cycleIndex + 1} / ${c.cyclesTotal}`;
    el('#tr-peak-label').textContent = Math.round(c.peakIntensity * 100) + '%';
    const reached = c.reachedPeakAfterMs ? `, erreicht nach ${(c.reachedPeakAfterMs / 1000).toFixed(1)}s` : '';
    const stopped = c.stoppedByUser ? ' — auf Wunsch unterbrochen' : '';
    const fb = c.arousalBefore ? `, angepasst nach Rückmeldung ${c.arousalBefore}` : '';
    log(`Zyklus ${c.cycleIndex + 1}/${c.cyclesTotal}: Spitze ${Math.round(c.peakIntensity * 100)}%, Halten ${c.holdMs}ms${fb}${reached}${stopped}`);
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
          `${i} gemeldet — wirkt auf den nächsten Zyklus.`;
      } catch (err) {
        el('#tr-arousal-status').textContent = 'Nicht übernommen: ' + err;
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
      log('Zyklus unterbrochen - Pause läuft, danach geht es weiter.');
    } catch (err) {
      log('Konnte nicht unterbrechen: ' + err);
    }
  });
  el('#tr-technique').addEventListener('change', updateTechniqueVisibility);
  updateTechniqueVisibility();
  refreshHistory();

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
