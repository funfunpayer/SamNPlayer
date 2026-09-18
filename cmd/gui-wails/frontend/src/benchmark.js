import { RunGoldenClipBenchmark, GetBenchmarkHistory, PickBenchmarkManifest } from '../wailsjs/go/main/App';
import { EventsOn } from '../wailsjs/runtime/runtime';
import { getSettingsCache, saveSetting } from './settings.js';
import { uiError } from './notify.js';

function formatDate(iso) {
  const d = new Date(iso);
  if (isNaN(d.getTime())) return iso || '?';
  return d.toLocaleString();
}

function renderClipRow(clip) {
  if (!clip.ok) {
    return `<tr><td>${clip.name}</td><td colspan="3" style="color:var(--danger)">FEHLGESCHLAGEN: ${clip.error}</td></tr>`;
  }
  const score = typeof clip.quality_score === 'number' ? `${Math.round(clip.quality_score * 100)}%` : 'n/a';
  const passed = clip.quality_passed ? 'OK' : 'PRÜFEN';
  const corr = clip.correlation ? `r=${clip.correlation.r.toFixed(3)}${clip.correlation.low_confidence ? ' (unsicher)' : ''}` : '-';
  return `<tr><td>${clip.name}</td><td>${score} (${passed})</td><td>${corr}</td>
          <td>${(clip.quality_warnings || []).length}</td></tr>`;
}

function renderResult(result) {
  const s = result.summary;
  const scoreLine = typeof s.mean_quality_score === 'number'
    ? `Ø Quality-Score: ${Math.round(s.mean_quality_score * 100)}%` : '';
  const corrLine = typeof s.mean_correlation === 'number'
    ? ` · Ø FunGen-Korrelation: ${s.mean_correlation.toFixed(3)} (${s.clips_with_reference} Clip(s) mit Referenz)` : '';
  return `
    <p><b>${s.ok}/${s.total} Clips erfolgreich</b>, ${s.quality_passed}/${s.ok || 1} Quality-Doctor bestanden
      ${result.git_commit ? ` · Commit ${result.git_commit}` : ''}</p>
    <p class="hint" style="margin:0 0 8px 0;">${scoreLine}${corrLine}</p>
    <table class="bench-table"><thead><tr><th>Clip</th><th>Quality</th><th>FunGen</th><th>Warnungen</th></tr></thead>
      <tbody>${result.clips.map(renderClipRow).join('')}</tbody></table>
  `;
}

function renderHistoryRow(r) {
  const s = r.summary;
  const score = typeof s.mean_quality_score === 'number' ? `${Math.round(s.mean_quality_score * 100)}%` : 'n/a';
  const corr = typeof s.mean_correlation === 'number' ? `, r=${s.mean_correlation.toFixed(3)}` : '';
  return `<div>${formatDate(r.timestamp)}${r.git_commit ? ` (${r.git_commit})` : ''} — `
    + `${s.ok}/${s.total} ok, Ø-Score ${score}${corr}</div>`;
}

// Golden-Clip-Benchmark: läuft ein festes, vom Nutzer gepflegtes Manifest
// aus echten Vergleichs-Clips (siehe generator/golden_clip_benchmark.py)
// durch die echte Pipeline und misst Quality-Doctor-Score sowie (wo eine
// FunGen-Referenz hinterlegt ist) die Übereinstimmung - eine feste,
// wiederholbare Vergleichsbasis statt Einzelmessungen (docs/NEXT.md
// Priorität 2, "Perception & Motion System 2.0"-Konzept Phase 1). Die
// Clips selbst (persönliches Videomaterial) liegen nicht im Repository,
// nur das Manifest verweist auf lokale Pfade.
export function initBenchmark(root) {
  root.innerHTML = `
    <h2>Golden-Clip-Benchmark</h2>
    <p class="hint">
      Führt ein festes Set eigener Vergleichs-Clips durch die echte Pipeline
      und misst Quality-Doctor-Score sowie (wo eine FunGen-Referenz
      hinterlegt ist) die Übereinstimmung - damit sich Verbesserungen (oder
      Regressionen) tatsächlich über die Zeit verfolgen lassen, statt nur
      als Einzelmessung im Gespräch zu stehen. Das Manifest verweist auf
      lokale Videodateien, es wird nichts hochgeladen.
    </p>

    <div class="field-row"><label>Manifest</label>
      <input type="text" id="bm-manifest" placeholder="Pfad zur manifest.json" style="flex:1" />
      <button id="bm-pick">Durchsuchen…</button>
    </div>

    <div class="row"><button id="bm-run" class="primary" disabled>Benchmark starten</button></div>
    <div id="bm-progress-wrap" style="display:none; margin-top:8px;">
      <div style="height:10px; border-radius:5px; background:rgba(255,255,255,0.10); overflow:hidden;">
        <div id="bm-progress-bar" style="height:100%; width:0%; background:linear-gradient(90deg,var(--accent),var(--teal));
             transition:width .2s linear;"></div>
      </div>
      <div id="bm-progress-text" class="hint" style="margin-top:4px;"></div>
    </div>
    <div class="path-label" id="bm-status"></div>

    <div id="bm-result" style="margin-top:12px;"></div>

    <h3 style="margin-top:16px;">Verlauf</h3>
    <p class="hint" style="margin-top:0;">Jeder Lauf wird an die Verlaufsdatei angehängt
      (Einstellungen-Tab) - hier eine Zeile je vergangenem Lauf, neuester zuerst.</p>
    <div id="bm-history" class="hint">Lädt...</div>
  `;

  const el = id => root.querySelector(id);
  let manifestPath = '';

  function updateRunEnabled() {
    el('#bm-run').disabled = !manifestPath;
  }

  function showProgress(show) {
    el('#bm-progress-wrap').style.display = show ? 'block' : 'none';
    if (!show) {
      el('#bm-progress-bar').style.width = '0%';
      el('#bm-progress-text').textContent = '';
    }
  }

  async function refreshHistory() {
    const box = el('#bm-history');
    try {
      const history = await GetBenchmarkHistory();
      if (!Array.isArray(history) || history.length === 0) {
        box.textContent = 'Noch kein Lauf aufgezeichnet.';
        return;
      }
      box.innerHTML = history.map(renderHistoryRow).join('');
    } catch (err) {
      box.textContent = 'Verlauf konnte nicht geladen werden: ' + err;
    }
  }

  el('#bm-pick').addEventListener('click', async () => {
    try {
      const path = await PickBenchmarkManifest();
      if (!path) return;
      manifestPath = path;
      el('#bm-manifest').value = path;
      saveSetting('generator.benchmarkManifestPath', path);
      updateRunEnabled();
    } catch (err) {
      uiError('Manifest wählen: ' + err, el('#bm-status'));
    }
  });

  el('#bm-manifest').addEventListener('change', e => {
    manifestPath = e.target.value.trim();
    saveSetting('generator.benchmarkManifestPath', manifestPath);
    updateRunEnabled();
  });

  el('#bm-run').addEventListener('click', () => {
    if (!manifestPath) return;
    el('#bm-run').disabled = true;
    el('#bm-status').textContent = 'Läuft…';
    el('#bm-result').innerHTML = '';
    showProgress(true);
    RunGoldenClipBenchmark(manifestPath);
  });

  EventsOn('benchmark:percent', pct => {
    if (pct < 0) return;
    el('#bm-progress-bar').style.width = pct + '%';
  });
  EventsOn('benchmark:progress', line => {
    el('#bm-progress-text').textContent = line;
  });
  EventsOn('benchmark:done', payload => {
    showProgress(false);
    updateRunEnabled();
    if (payload.error) {
      uiError('Benchmark fehlgeschlagen: ' + payload.error, el('#bm-status'));
      return;
    }
    el('#bm-status').textContent = 'Fertig: ' + formatDate(payload.result.timestamp);
    el('#bm-result').innerHTML = renderResult(payload.result);
    refreshHistory();
  });

  getSettingsCache().then(s => {
    manifestPath = s.benchmarkManifestPath || '';
    el('#bm-manifest').value = manifestPath;
    updateRunEnabled();
  });
  refreshHistory();
}
