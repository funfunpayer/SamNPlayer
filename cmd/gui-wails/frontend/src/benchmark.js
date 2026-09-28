import { RunGoldenClipBenchmark, GetBenchmarkHistory, PickBenchmarkManifest, PickFunscriptFile, PickVideoFile, ScoreScriptPair, AppendBenchmarkPairLabel, SuggestBenchmarkPairBesideVideo } from '../wailsjs/go/main/App';
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
    return `<tr><td>${clip.name}</td><td colspan="3" style="color:var(--danger)">FAILED: ${clip.error}</td></tr>`;
  }
  const score = typeof clip.quality_score === 'number' ? `${Math.round(clip.quality_score * 100)}%` : 'n/a';
  const passed = clip.quality_passed ? 'OK' : 'REVIEW';
  const corr = clip.correlation ? `r=${clip.correlation.r.toFixed(3)}${clip.correlation.low_confidence ? ' (low confidence)' : ''}` : '-';
  return `<tr><td>${clip.name}</td><td>${score} (${passed})</td><td>${corr}</td>
          <td>${(clip.quality_warnings || []).length}</td></tr>`;
}

function renderResult(result) {
  const s = result.summary;
  const scoreLine = typeof s.mean_quality_score === 'number'
    ? `Avg quality score: ${Math.round(s.mean_quality_score * 100)}%` : '';
  const corrLine = typeof s.mean_correlation === 'number'
    ? ` · avg reference correlation: ${s.mean_correlation.toFixed(3)} (${s.clips_with_reference} clip(s) with reference)` : '';
  return `
    <p><b>${s.ok}/${s.total} clips succeeded</b>, ${s.quality_passed}/${s.ok || 1} passed Quality Doctor
      ${result.git_commit ? ` · Commit ${result.git_commit}` : ''}</p>
    <p class="hint" style="margin:0 0 8px 0;">${scoreLine}${corrLine}</p>
    <table class="bench-table"><thead><tr><th>Clip</th><th>Quality</th><th>Reference</th><th>Warnings</th></tr></thead>
      <tbody>${result.clips.map(renderClipRow).join('')}</tbody></table>
  `;
}

function renderHistoryRow(r) {
  const s = r.summary;
  const score = typeof s.mean_quality_score === 'number' ? `${Math.round(s.mean_quality_score * 100)}%` : 'n/a';
  const corr = typeof s.mean_correlation === 'number' ? `, r=${s.mean_correlation.toFixed(3)}` : '';
  return `<div>${formatDate(r.timestamp)}${r.git_commit ? ` (${r.git_commit})` : ''} — `
    + `${s.ok}/${s.total} ok, avg score ${score}${corr}</div>`;
}

function labelDe(label) {
  if (label === 'good') return 'GUT';
  if (label === 'review') return 'PRÜFEN';
  return 'NICHT GUT';
}

/** Wails may surface json tags or Go field names — normalize for UI. */
function pairFields(score) {
  const s = score || {};
  const fidelity = s.fidelity || s.Fidelity || {};
  const diagnosis = fidelity.diagnosis || fidelity.Diagnosis || {};
  const quality = s.quality || s.Quality || {};
  const r = fidelity.r ?? fidelity.R;
  return {
    label: s.label || s.Label || '',
    passed: !!(s.passed ?? s.Passed),
    detail: s.detail || s.Detail || '',
    r: r != null ? Number(r) : null,
    verdict: diagnosis.verdict || diagnosis.Verdict || '?',
    lowConfidence: !!(fidelity.low_confidence ?? fidelity.LowConfidence),
    qScore: quality.score ?? quality.Score,
    qPassed: !!(quality.passed ?? quality.Passed),
  };
}

function renderPairScore(score) {
  const p = pairFields(score);
  const r = p.r != null ? p.r.toFixed(3) : 'n/a';
  const qScore = p.qScore != null ? Math.round(Number(p.qScore) * 100) + '%' : 'n/a';
  const qPass = p.qPassed ? 'OK' : 'FAIL';
  const color = p.passed ? 'var(--teal)' : (p.label === 'review' ? 'var(--accent)' : 'var(--danger)');
  return `
    <p style="margin:8px 0 4px 0;"><b style="color:${color}">${labelDe(p.label)}</b>
      <span class="hint"> · label=${p.label} · passed=${p.passed}</span></p>
    <p class="hint" style="margin:0 0 8px 0;">${p.detail || ''}</p>
    <table class="bench-table"><thead><tr><th>Motion Fidelity (vs Ref)</th><th>Quality Doctor</th></tr></thead>
      <tbody><tr>
        <td>r=${r} · verdict=${p.verdict}${p.lowConfidence ? ' · low confidence' : ''}</td>
        <td>${qScore} (${qPass})</td>
      </tr></tbody></table>
  `;
}

// Golden-Clip-Benchmark + Pair-Compare. Everyday Go CSRT remains the
// production basis; KI never replaces Everyday. Pair-compare scores any
// candidate (typically Everyday output, or hybrid-assisted) against a
// FunGen/reference funscript. Manifest path still drives the full
// generate+measure loop (docs/GOLDEN_CLIPS.md).
export function initBenchmark(root) {
  root.innerHTML = `
    <h2>Benchmark</h2>
    <p class="hint">
      Everyday recognition (Go CSRT) is always the basis — also when KI assists
      in hybrid mode. Score a candidate script against a FunGen reference, or
      run the golden-clip manifest through the real Everyday pipeline.
    </p>
    <p class="hint" style="margin-top:0;">
      Optional prep (owner): cut short downscaled clips before Compare or a golden
      manifest — Everyday Go CSRT stays the recognition basis.
    </p>

    <details id="bm-clip-prep" class="bm-clip-prep" open>
      <summary style="cursor:pointer;">Clip-Prep (scripts — light UI)</summary>
      <p class="hint" style="margin:8px 0 6px 0;">
        Not Everyday Create. Cut 20–60&nbsp;s clips at ~720p (1280-wide) with ffmpeg.
        Full guide: repo <code>docs/owner/benchmark-clip-prep.md</code> ·
        <code>scripts/benchmark-prep/README.md</code>.
      </p>
      <pre id="bm-clip-prep-cmd" class="hint" style="white-space:pre-wrap;margin:0 0 8px 0;padding:8px;border:1px solid rgba(255,255,255,0.08);">./scripts/benchmark-prep/cut_clip.sh \
  -i /path/long.mp4 \
  --start 01:20 --end 02:05 \
  -o ~/clips/hub_easy.mp4</pre>
      <div class="row" style="flex-wrap:wrap;gap:8px;">
        <button type="button" id="bm-clip-prep-copy" class="secondary">Copy command</button>
        <button type="button" id="bm-clip-prep-batch" class="secondary">Show batch (marks.json)</button>
        <span class="hint" id="bm-clip-prep-status" style="margin:0;"></span>
      </div>
    </details>

    <h3>Compare scripts</h3>
    <p class="hint" style="margin-top:0;">
      Short clip + FunGen/reference + Everyday candidate → gut / prüfen / nicht gut.
      After Clip-Prep, pick the clip then <b>Suggest beside video</b> (names like
      <code>clip.funscript</code> + <code>clip__hub.funscript</code>).
    </p>
    <div class="field-row"><label>Video</label>
      <input type="text" id="bm-video" placeholder="Short clip path (for labels + suggest)" style="flex:1" />
      <button id="bm-pick-video" type="button">Browse…</button>
    </div>
    <div class="row" style="margin:4px 0 8px 0;flex-wrap:wrap;gap:8px;">
      <button id="bm-suggest-pair" type="button" class="secondary" disabled
        data-help="After Clip-Prep: fills Reference from stem.funscript (FunGen) and Candidate from stem__hub.funscript (Everyday), searching the clip folder and one level of subfolders (e.g. mit_yolo/). Only stem.funscript found → used as Candidate; pick FunGen Ref with Browse. Needs Video set. Does not run Create.">Suggest beside video</button>
      <span class="hint" id="bm-suggest-status" style="margin:0;"></span>
    </div>
    <div class="field-row"><label>Reference</label>
      <input type="text" id="bm-ref" placeholder="FunGen / reference .funscript" style="flex:1" />
      <button id="bm-pick-ref" type="button">Browse…</button>
    </div>
    <div class="field-row"><label>Candidate</label>
      <input type="text" id="bm-cand" placeholder="Everyday / candidate .funscript" style="flex:1" />
      <button id="bm-pick-cand" type="button">Browse…</button>
    </div>
    <div class="row">
      <button id="bm-score" class="primary" disabled>Score vs reference</button>
      <button id="bm-save-label" type="button" disabled>Save label for KI</button>
    </div>
    <div class="path-label" id="bm-pair-status"></div>
    <div id="bm-pair-result" style="margin-top:8px;"></div>

    <h3 style="margin-top:20px;">Golden-clip manifest</h3>
    <p class="hint" style="margin-top:0;">
      Runs a fixed set of your comparison clips through the real Everyday
      pipeline and measures Quality Doctor + FunGen agreement. Manifest points
      at local videos; nothing is uploaded.
    </p>

    <div class="field-row"><label>Manifest</label>
      <input type="text" id="bm-manifest" placeholder="Path to manifest.json" style="flex:1" />
      <button id="bm-pick">Browse…</button>
    </div>

    <div class="row"><button id="bm-run" class="primary" disabled>Run benchmark</button></div>
    <div id="bm-progress-wrap" style="display:none; margin-top:8px;">
      <div style="height:10px; border-radius:5px; background:rgba(255,255,255,0.10); overflow:hidden;">
        <div id="bm-progress-bar" style="height:100%; width:0%; background:linear-gradient(90deg,var(--accent),var(--teal));
             transition:width .2s linear;"></div>
      </div>
      <div id="bm-progress-text" class="hint" style="margin-top:4px;"></div>
    </div>
    <div class="path-label" id="bm-status"></div>

    <div id="bm-result" style="margin-top:12px;"></div>

    <h3 style="margin-top:16px;">History</h3>
    <p class="hint" style="margin-top:0;">Each manifest run is appended to the history file
      (Settings tab) — one line per past run, newest first.</p>
    <div id="bm-history" class="hint">Loading…</div>
  `;

  const el = id => root.querySelector(id);
  let manifestPath = '';
  let videoPath = '';
  let refPath = '';
  let candPath = '';
  let lastPairScore = null;

  function updateRunEnabled() {
    el('#bm-run').disabled = !manifestPath;
  }

  function updateScoreEnabled() {
    el('#bm-score').disabled = !(refPath && candPath);
    el('#bm-save-label').disabled = !lastPairScore;
    if (el('#bm-suggest-pair')) el('#bm-suggest-pair').disabled = !videoPath;
  }

  function applyPairSuggestion(sug) {
    const s = sug || {};
    const ref = s.reference || s.Reference || '';
    const cand = s.candidate || s.Candidate || '';
    const note = s.note || s.Note || '';
    if (ref) {
      refPath = ref;
      el('#bm-ref').value = ref;
    }
    if (cand) {
      candPath = cand;
      el('#bm-cand').value = cand;
    }
    if (el('#bm-suggest-status')) el('#bm-suggest-status').textContent = note;
    lastPairScore = null;
    el('#bm-pair-result').innerHTML = '';
    updateScoreEnabled();
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
        box.textContent = 'No run recorded yet.';
        return;
      }
      box.innerHTML = history.map(renderHistoryRow).join('');
    } catch (err) {
      box.textContent = 'Could not load history: ' + err;
    }
  }

  el('#bm-pick-video').addEventListener('click', async () => {
    try {
      const path = await PickVideoFile();
      if (!path) return;
      videoPath = path;
      el('#bm-video').value = path;
      updateScoreEnabled();
    } catch (err) {
      uiError('Choose video: ' + err, el('#bm-pair-status'));
    }
  });
  el('#bm-suggest-pair')?.addEventListener('click', async () => {
    if (!videoPath) return;
    el('#bm-suggest-pair').disabled = true;
    if (el('#bm-suggest-status')) el('#bm-suggest-status').textContent = 'Looking beside clip…';
    try {
      const sug = await SuggestBenchmarkPairBesideVideo(videoPath);
      applyPairSuggestion(sug);
      if (!sug?.reference && !sug?.Reference && !sug?.candidate && !sug?.Candidate) {
        el('#bm-pair-status').textContent = sug?.note || sug?.Note || 'No pair found.';
      } else {
        el('#bm-pair-status').textContent = 'Suggested — review paths, then Score vs reference.';
      }
    } catch (err) {
      uiError('Suggest pair: ' + err, el('#bm-pair-status'));
      if (el('#bm-suggest-status')) el('#bm-suggest-status').textContent = '';
    }
    updateScoreEnabled();
  });
  el('#bm-pick-ref').addEventListener('click', async () => {
    try {
      const path = await PickFunscriptFile();
      if (!path) return;
      refPath = path;
      el('#bm-ref').value = path;
      updateScoreEnabled();
    } catch (err) {
      uiError('Choose reference: ' + err, el('#bm-pair-status'));
    }
  });
  el('#bm-pick-cand').addEventListener('click', async () => {
    try {
      const path = await PickFunscriptFile();
      if (!path) return;
      candPath = path;
      el('#bm-cand').value = path;
      updateScoreEnabled();
    } catch (err) {
      uiError('Choose candidate: ' + err, el('#bm-pair-status'));
    }
  });
  el('#bm-video').addEventListener('change', e => {
    videoPath = e.target.value.trim();
    updateScoreEnabled();
  });
  el('#bm-ref').addEventListener('change', e => {
    refPath = e.target.value.trim();
    updateScoreEnabled();
  });
  el('#bm-cand').addEventListener('change', e => {
    candPath = e.target.value.trim();
    updateScoreEnabled();
  });

  el('#bm-score').addEventListener('click', async () => {
    if (!refPath || !candPath) return;
    el('#bm-score').disabled = true;
    el('#bm-pair-status').textContent = 'Scoring…';
    el('#bm-pair-result').innerHTML = '';
    lastPairScore = null;
    updateScoreEnabled();
    try {
      const score = await ScoreScriptPair(refPath, candPath, videoPath || '');
      lastPairScore = score;
      el('#bm-pair-status').textContent = 'Done · ' + labelDe(pairFields(score).label);
      el('#bm-pair-result').innerHTML = renderPairScore(score);
    } catch (err) {
      uiError('Score failed: ' + err, el('#bm-pair-status'));
    }
    updateScoreEnabled();
  });

  el('#bm-save-label').addEventListener('click', async () => {
    if (!lastPairScore) return;
    try {
      const path = await AppendBenchmarkPairLabel(lastPairScore);
      el('#bm-pair-status').textContent = 'Label saved: ' + path;
    } catch (err) {
      uiError('Save label: ' + err, el('#bm-pair-status'));
    }
  });

  el('#bm-pick').addEventListener('click', async () => {
    try {
      const path = await PickBenchmarkManifest();
      if (!path) return;
      manifestPath = path;
      el('#bm-manifest').value = path;
      saveSetting('generator.benchmarkManifestPath', path);
      updateRunEnabled();
    } catch (err) {
      uiError('Choose manifest: ' + err, el('#bm-status'));
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
    el('#bm-status').textContent = 'Running…';
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
      uiError('Benchmark failed: ' + payload.error, el('#bm-status'));
      return;
    }
    el('#bm-status').textContent = 'Done: ' + formatDate(payload.result.timestamp);
    el('#bm-result').innerHTML = renderResult(payload.result);
    refreshHistory();
  });

  getSettingsCache().then(s => {
    manifestPath = s.benchmarkManifestPath || '';
    el('#bm-manifest').value = manifestPath;
    updateRunEnabled();
  });
  refreshHistory();

  const SINGLE_CMD = `./scripts/benchmark-prep/cut_clip.sh \\
  -i /path/long.mp4 \\
  --start 01:20 --end 02:05 \\
  -o ~/clips/hub_easy.mp4`;
  const BATCH_CMD = `./scripts/benchmark-prep/cut_clip.sh \\
  --marks ~/clips/marks.json \\
  --out-dir ~/clips/out`;

  el('#bm-clip-prep-copy')?.addEventListener('click', async () => {
    const text = el('#bm-clip-prep-cmd')?.textContent || SINGLE_CMD;
    try {
      if (navigator.clipboard?.writeText) {
        await navigator.clipboard.writeText(text);
      } else {
        const ta = document.createElement('textarea');
        ta.value = text;
        document.body.appendChild(ta);
        ta.select();
        document.execCommand('copy');
        ta.remove();
      }
      if (el('#bm-clip-prep-status')) el('#bm-clip-prep-status').textContent = 'Copied.';
    } catch (err) {
      if (el('#bm-clip-prep-status')) el('#bm-clip-prep-status').textContent = 'Copy failed — select the command manually.';
    }
  });
  el('#bm-clip-prep-batch')?.addEventListener('click', () => {
    if (el('#bm-clip-prep-cmd')) el('#bm-clip-prep-cmd').textContent = BATCH_CMD;
    if (el('#bm-clip-prep-status')) {
      el('#bm-clip-prep-status').textContent = 'Batch mode — fill example_marks.json fields first.';
    }
  });
}
