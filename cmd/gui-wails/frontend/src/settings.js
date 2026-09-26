import { GetSettings, SetSetting, PickReportPath, ReportSummary, ReportExists, GetHardwareInfo, GetCacheInfo, ClearCache, TrainQualityModel, QualityModelInfo, OpenLogFolder, CheckAIRoiAvailable, CurrentVersion, CheckForUpdate, ApplyUpdate, GetRuntimeHealth, EnsureVideoTools, GetLicenseStatus, ImportLicenseText, ImportLicenseFile, ClearLicense, DeleteSceneMapLearningData } from '../wailsjs/go/main/App';
import { EventsOn } from '../wailsjs/runtime/runtime';
import { uiError, uiInfo } from './notify.js';
import { openHandbook } from './handbook.js';
import { wireDataHelp } from './help.js';

let cachedSettings = null;
let cachedPromise = null;

// getSettingsCache wird von anderen Modulen (playback.js) genutzt, um beim
// Start die gespeicherten Standardwerte zu übernehmen, ohne dass jedes Modul
// selbst GetSettings() aufrufen und auf die Reihenfolge achten muss.
export function getSettingsCache() {
  if (cachedSettings) return Promise.resolve(cachedSettings);
  if (!cachedPromise) cachedPromise = GetSettings().then(s => { cachedSettings = s; return s; });
  return cachedPromise;
}

export function saveSetting(key, value) {
  if (cachedSettings) {
    const map = {
      'update.check_on_startup': 'updateCheckOnStartup',
      'device.connect_test': 'deviceConnectTest',
      'log.level': 'logLevel',
      'playback.mock': 'playbackMock',
      'playback.sync_mode': 'playbackSync',
      'playback.tick_ms': 'playbackTickMs',
      'playback.max_speed': 'playbackMaxSpeed',
      'playback.smoothing': 'playbackSmoothing',
      'playback.soft_start_ms': 'playbackSoftStartMs',
      'playback.extended_o_enabled': 'playbackEOEnabled',
      'playback.extended_o_min': 'playbackEOMin',
      'playback.extended_o_hold_seconds': 'playbackEOHoldS',
      'playback.extended_o_restore_ms': 'playbackEORestoreMs',
      'playback.video_play_autostart': 'playbackVideoPlayAutostart',
      'playback.trajectory_overlay': 'playbackTrajectoryOverlay',
      'training.mock': 'trainingMock',
      'training.technique': 'trainingTechnique',
      'training.channel': 'trainingChannel',
      'training.cycles': 'trainingCycles',
      'training.ramp_up_ms': 'trainingRampUpMs',
      'training.hold_ms': 'trainingHoldMs',
      'training.rest_ms': 'trainingRestMs',
      'training.peak_intensity': 'trainingPeakIntensity',
      'training.plateau_fraction': 'trainingPlateauFraction',
      'training.progression_per_cycle': 'trainingProgressionPerCycle',
      'generator.aiRoiModelPath': 'aiRoiModelPath',
      'generator.aiPreferredClasses': 'aiPreferredClasses',
      'generator.aiBaseUrl': 'aiBaseUrl',
      'generator.roiTrainingDatasetDir': 'roiDatasetDir',
      'generator.collectLearningData': 'collectLearningData',
    };
    const field = map[key];
    if (field) cachedSettings[field] = value;
  }
  return SetSetting(key, value);
}

export function initSettings(root) {
  root.innerHTML = `
    <h2>Settings</h2>
    <div class="row" style="align-items:center; margin-bottom:14px;">
      <button type="button" id="st-open-handbook" class="primary"
        data-help="Opens the in-app user handbook: Create, Play, gaps, AI path, troubleshooting.">Open user handbook</button>
      <span class="hint" style="margin:0">FAQ + how to use every Everyday control. Also on Create.</span>
    </div>
    <div class="checkbox-row">
      <input type="checkbox" id="st-update-check" />
      <label for="st-update-check">Check for updates on startup</label>
    </div>
    <div class="checkbox-row">
      <input type="checkbox" id="st-connect-test" />
      <label for="st-connect-test">On connect, run a short connection test (brief vib/suction)</label>
    </div>
    <p class="hint" style="margin-top:0">Off by default. When on: after a successful connect, a short pulse confirms that commands arrive.</p>

    <h3>What you need (keep it lean)</h3>
    <ul class="hint" style="margin:0 0 12px; padding-left:1.2em; line-height:1.55;">
      <li><b>Play + Create (default)</b> — use the <b>portable</b> download: app + ffmpeg in one folder. No extra install.</li>
      <li><b>Video tools missing?</b> Settings → Install video tools (one click), or re-download portable.</li>
      <li><b>Python</b> — only for the classic Python generator path and <b>AI Train</b>. The Go Create path does not need it.</li>
      <li><b>AI Train</b> (optional) — Python + “Install dependencies” in the AI Train tab (downloads ultralytics/torch; large). Skip if you only Play/Create.</li>
    </ul>

    <h3>License</h3>
    <p class="hint">Personal yearly key (one person). Invite/internal keys have no expiry.
      Enforcement is <b>off</b> in this build — import works so we can test the path;
      Create/Play are not limited yet. See docs/LICENSE_SYSTEM.md.</p>
    <p class="hint" id="st-license-status" style="margin-top:0">…</p>
    <div class="row" style="align-items:flex-start;">
      <textarea id="st-license-paste" rows="3" placeholder="Paste license token (SNP1.…)" style="flex:1; font-family:ui-monospace,monospace; font-size:12px;"></textarea>
    </div>
    <div class="row" style="align-items:center; margin-top:6px;">
      <button id="st-license-import-text" type="button">Import pasted key</button>
      <button id="st-license-import-file" type="button">Import from file…</button>
      <button id="st-license-clear" type="button">Clear license</button>
      <button id="st-license-refresh" type="button">Refresh status</button>
    </div>

    <h3>Runtime &amp; updates</h3>
    <div class="row" style="align-items:center;">
      <button id="st-runtime-check" type="button">Check folders &amp; dependencies</button>
      <button id="st-install-ffmpeg" type="button"
        data-help="Downloads a static ffmpeg into your SamNPlayer tools folder when missing. Prefer the portable release (ffmpeg already next to the app).">Install video tools</button>
      <span class="hint" id="st-runtime-status" style="margin:0"></span>
    </div>
    <div class="row" style="align-items:center;">
      <button id="st-update-now" type="button">Check for updates now</button>
      <button id="st-update-apply" type="button" class="primary" style="display:none;">Download &amp; restart</button>
      <span class="hint" id="st-update-status" style="margin:0"></span>
    </div>
    <div class="field-row"><label>Log level</label>
      <select id="st-log-level">
        <option value="debug">debug</option>
        <option value="info">info</option>
        <option value="warn">warn</option>
        <option value="error">error</option>
      </select>
    </div>
    <div class="row">
      <span class="path-label" id="st-log-path"></span>
      <button id="st-open-log">Open log folder</button>
    </div>

    <h3>AI region detection (local, optional)</h3>
    <p class="hint">Local ONNX object detector as an alternative to the classical
      rhythm heuristic in the Create tab (“Find tip area”). No model
      ships with the app and none is downloaded — without your own <code>.onnx</code>
      file, classical detection stays in use. Leave empty to use the default folder
      (<code>%LOCALAPPDATA%\\SamNPlayer\\models\\roi_detector.onnx</code> on Windows).</p>
    <div class="row">
      <input type="text" id="st-ai-roi-path" placeholder="(default folder)" style="flex:1;" />
      <button id="st-ai-roi-check">Check availability</button>
    </div>
    <div class="field-row" style="margin-top:8px">
      <label for="st-ai-pref-classes"
        data-help="Comma-separated English body-part IDs (face,mouth,breasts,nipples,hand_1,hand_2,penis,glans,vagina). Empty = all classes from the model. Names need classes.json next to the .onnx.">Preferred classes</label>
      <input type="text" id="st-ai-pref-classes" placeholder="face,mouth,breasts,nipples,hand_1,hand_2,penis,glans,vagina" style="flex:1;" />
    </div>
    <p class="hint" id="st-ai-roi-status" style="margin-top:0"></p>

    <h3>Scene map learning (local, optional)</h3>
    <p class="hint">Collects JSON/JSONL next to your ROI dataset (<code>scene_map_learning/</code>)
      from Create’s companion <code>.samn</code> — engine trace, excludes, auto candidates
      (<code>reviewed: false</code>). Default <b>off</b>. Never uploads; never writes YOLO
      <code>images/train</code> / <code>labels/train</code>. Use Create → Advanced →
      Export for learning after Generate (or after Show scene map + companion save).</p>
    <div class="checkbox-row">
      <input type="checkbox" id="st-collect-learning" />
      <label for="st-collect-learning"
        data-help="Required before Export for learning. Off by default (Owner).">Collect learning data</label>
    </div>
    <div class="row" style="align-items:center; margin-top:6px;">
      <button id="st-delete-learning" type="button"
        data-help="Deletes only scene_map_learning under the ROI dataset folder. Hand YOLO samples stay.">Delete learning data</button>
      <span class="hint" id="st-learning-status" style="margin:0"></span>
    </div>

    <h3>AI server for profile suggestion &amp; quality second opinion (local, optional)</h3>
    <p class="hint">Address of a local Colibri server (<code>coli serve</code>,
      see docs/AI_ADAPTER.md) for the “Suggest profile” button and the AI quality
      opinion in the Create tab. Both work without this server — measured scene
      similarity (no AI) remains the only source for profile suggestions. Leave
      empty to use the default address.</p>
    <div class="row">
      <input type="text" id="st-ai-base-url" placeholder="(default address)" style="flex:1;" />
    </div>

    <h3>Hardware</h3>
    <p class="hint">Which acceleration the generator can actually use. Having an
      NVIDIA GPU does not mean it is used — typical OpenCV pip packages are built
      without CUDA.</p>
    <div class="row"><button id="st-hardware">Check hardware</button></div>
    <pre id="st-hardware-out" class="hint" style="white-space:pre-wrap; margin-top:6px;"></pre>

    <h3>Cache</h3>
    <p class="hint">The generator stores tracking results so a re-run with different
      settings does not re-decode the whole video — about 36× faster. The cache grows
      with each video.</p>
    <div class="row">
      <span class="path-label" id="st-cache-info">…</span>
      <button id="st-cache-clear">Clear now</button>
    </div>
    <div class="checkbox-row">
      <input type="checkbox" id="st-cache-exit" />
      <label for="st-cache-exit">Clear automatically on quit</label>
    </div>

    <h3>Generator metrics</h3>
    <p class="hint">Appends a line of metrics for every Create run (motion amplitude,
      spectral concentration, tracker loss, runtime). Together with your judgments in
      the Create tab, this is the basis for tuning quality scoring on real material —
      so far it relies on synthetic test videos. Leave empty to disable recording.</p>
    <div class="row">
      <input type="text" id="st-report-path" placeholder="(no recording)"
             style="flex:1;" />
      <button id="st-pick-report">Choose…</button>
      <button id="st-default-report">Default</button>
    </div>
    <p class="hint" id="st-report-status" style="margin-top:2px;"></p>
    <div class="row">
      <button id="st-report-summary">Show summary</button>
      <button id="st-model-info">Show model</button>
      <button id="st-model-train" class="primary">Learn from judgments</button>
    </div>
    <p class="hint">Quality scoring uses thresholds set on synthetic test videos.
      With enough of your own judgments, a model can be learned instead. It is only
      adopted if it beats the current rules in cross-validation — otherwise nothing
      changes.</p>
    <pre id="st-report-out" class="hint" style="white-space:pre-wrap; margin-top:6px;"></pre>
  `;

  const el = id => root.querySelector(id);

  wireDataHelp(root);
  el('#st-open-handbook')?.addEventListener('click', () => openHandbook());

  // Ohne das war unsichtbar, ob unter dem eingestellten Pfad schon Messwerte
  // stehen - "Show summary" beantwortete das zwar auch, aber erst
  // nach einem Klick und mit einer errorsmeldung statt eines einfachen
  // Hinweises, wenn (noch) nichts drin ist.
  async function updateReportStatus() {
    const status = el('#st-report-status');
    const path = el('#st-report-path').value.trim();
    if (!path) {
      status.textContent = '';
      return;
    }
    try {
      status.textContent = (await ReportExists())
        ? '✓ Already contains metrics.'
        : 'No metrics recorded yet (created on the next generator run).';
    } catch (err) {
      status.textContent = '';
    }
  }

  getSettingsCache().then(s => {
    el('#st-update-check').checked = s.updateCheckOnStartup;
    el('#st-connect-test').checked = !!s.deviceConnectTest;
    el('#st-log-level').value = s.logLevel;
    el('#st-log-path').textContent = s.logPath || '(no log file written yet)';
    el('#st-report-path').value = s.reportPath || '';
    el('#st-cache-exit').checked = !!s.clearCacheOnExit;
    el('#st-report-path').dataset.default = s.defaultReportPath || '';
    el('#st-ai-roi-path').value = s.aiRoiModelPath || '';
    el('#st-ai-pref-classes').value = s.aiPreferredClasses || '';
    el('#st-ai-base-url').value = s.aiBaseUrl || '';
    el('#st-collect-learning').checked = !!s.collectLearningData;
    updateReportStatus();
    refreshLicenseStatus();
  });

  async function refreshLicenseStatus() {
    const box = el('#st-license-status');
    if (!box) return;
    try {
      const st = await GetLicenseStatus();
      const bits = [
        `state: ${st.state || 'none'}`,
        st.sub ? `person: ${st.sub}` : null,
        st.tier ? `tier: ${st.tier}` : null,
        st.validUntil ? `until: ${st.validUntil}` : null,
        `enforcement: ${st.enforcement ? 'ON' : 'off'}`,
        `effective: ${st.effective ? 'full access' : 'trial'}`,
      ].filter(Boolean);
      box.textContent = (st.message || '') + ' · ' + bits.join(' · ');
    } catch (err) {
      box.textContent = 'License status failed: ' + err;
    }
  }

  el('#st-license-refresh')?.addEventListener('click', () => refreshLicenseStatus());
  el('#st-license-import-text')?.addEventListener('click', async () => {
    const box = el('#st-license-status');
    const raw = el('#st-license-paste')?.value || '';
    try {
      const st = await ImportLicenseText(raw);
      el('#st-license-paste').value = '';
      uiInfo(st.message || 'License imported.', box);
      await refreshLicenseStatus();
    } catch (err) {
      uiError('Import license: ' + err, box);
    }
  });
  el('#st-license-import-file')?.addEventListener('click', async () => {
    const box = el('#st-license-status');
    try {
      const st = await ImportLicenseFile();
      uiInfo(st.message || 'License imported.', box);
      await refreshLicenseStatus();
    } catch (err) {
      uiError('Import license: ' + err, box);
    }
  });
  el('#st-license-clear')?.addEventListener('click', async () => {
    const box = el('#st-license-status');
    try {
      await ClearLicense();
      uiInfo('License cleared.', box);
      await refreshLicenseStatus();
    } catch (err) {
      uiError('Clear license: ' + err, box);
    }
  });

  el('#st-update-check').addEventListener('change', e => saveSetting('update.check_on_startup', e.target.checked));
  el('#st-connect-test').addEventListener('change', e => saveSetting('device.connect_test', e.target.checked));
  el('#st-runtime-check').addEventListener('click', async () => {
    const status = el('#st-runtime-status');
    status.textContent = 'Checking…';
    try {
      const h = await GetRuntimeHealth();
      const missing = (h.deps || []).filter(d => !d.found).map(d => d.label);
      const created = (h.dirsCreated || []).length;
      const parts = [];
      if (created) parts.push(`${created} folder(s) created`);
      if (missing.length) parts.push('missing: ' + missing.join(', '));
      else parts.push('Dependencies OK');
      if (h.resources) {
        parts.push(`${h.resources.goos}/${h.resources.goarch} · ${h.resources.numCPU} CPU · GOMAXPROCS ${h.resources.goMaxProcs}`);
      }
      status.textContent = (h.ok ? '✓ ' : '⚠ ') + parts.join(' · ');
    } catch (err) {
      status.textContent = 'Check failed: ' + err;
    }
  });
  EventsOn('runtime:tools', line => {
    const status = el('#st-runtime-status');
    if (status) status.textContent = String(line);
  });
  el('#st-install-ffmpeg').addEventListener('click', async () => {
    const status = el('#st-runtime-status');
    const btn = el('#st-install-ffmpeg');
    btn.disabled = true;
    status.textContent = 'Installing video tools (ffmpeg)…';
    try {
      await EnsureVideoTools();
      const h = await GetRuntimeHealth();
      const ff = (h.deps || []).find(d => d.id === 'ffmpeg');
      if (ff && ff.found) {
        uiInfo('Video tools ready: ' + (ff.path || 'ffmpeg'), status);
        status.textContent = '✓ ffmpeg: ' + (ff.path || 'ok');
      } else {
        status.textContent = 'Install finished but ffmpeg still missing — use the portable release.';
      }
    } catch (err) {
      uiError('Install video tools: ' + err, status);
    } finally {
      btn.disabled = false;
    }
  });
  el('#st-update-now').addEventListener('click', async () => {
    const status = el('#st-update-status');
    const btn = el('#st-update-now');
    const apply = el('#st-update-apply');
    btn.disabled = true;
    if (apply) apply.style.display = 'none';
    status.textContent = 'Checking…';
    try {
      const version = await CurrentVersion();
      const res = await CheckForUpdate();
      if (res.error) {
        uiError('Update check: ' + res.error, status);
      } else if (!res.available) {
        status.textContent = `No update available (current: ${version}).`;
      } else {
        const tag = res.release ? res.release.tag_name : '?';
        status.textContent = `Version ${tag} available (current: ${version}).`;
        if (apply) {
          apply.style.display = '';
          apply.onclick = async () => {
            apply.disabled = true;
            try {
              await ApplyUpdate();
            } catch (err) {
              uiError('Update failed: ' + err, status);
              apply.disabled = false;
            }
          };
        }
      }
    } catch (err) {
      uiError('Update check: ' + err, status);
    } finally {
      btn.disabled = false;
    }
  });
  el('#st-log-level').addEventListener('change', e => saveSetting('log.level', e.target.value));
  el('#st-open-log').addEventListener('click', () => OpenLogFolder().catch(err => uiError('Log folder: ' + err)));

  el('#st-report-path').addEventListener('change', e =>
    saveSetting('generator.reportPath', e.target.value.trim()).then(updateReportStatus));

  el('#st-pick-report').addEventListener('click', async () => {
    try {
      const path = await PickReportPath();
      if (path) {
        el('#st-report-path').value = path;
        await saveSetting('generator.reportPath', path);
        updateReportStatus();
      }
    } catch (err) {
      uiError('Report path: ' + err);
    }
  });

  el('#st-default-report').addEventListener('click', async () => {
    const fallback = el('#st-report-path').dataset.default || '';
    el('#st-report-path').value = fallback;
    await saveSetting('generator.reportPath', fallback);
    updateReportStatus();
  });

  el('#st-ai-roi-path').addEventListener('change', e =>
    saveSetting('generator.aiRoiModelPath', e.target.value.trim()));

  el('#st-ai-pref-classes').addEventListener('change', e =>
    saveSetting('generator.aiPreferredClasses', e.target.value.trim()));

  el('#st-ai-base-url').addEventListener('change', e =>
    saveSetting('generator.aiBaseUrl', e.target.value.trim()));

  el('#st-collect-learning')?.addEventListener('change', e =>
    saveSetting('generator.collectLearningData', e.target.checked));

  el('#st-delete-learning')?.addEventListener('click', async () => {
    const status = el('#st-learning-status');
    if (status) status.textContent = 'Deleting…';
    try {
      await DeleteSceneMapLearningData();
      if (status) status.textContent = 'Deleted scene_map_learning (YOLO samples kept).';
      uiInfo('Scene map learning data deleted.', status);
    } catch (err) {
      if (status) status.textContent = '';
      uiError('Delete learning data: ' + err, status);
    }
  });

  el('#st-ai-roi-check').addEventListener('click', async () => {
    const status = el('#st-ai-roi-status');
    status.textContent = 'Checking…';
    try {
      const available = await CheckAIRoiAvailable();
      status.textContent = available
        ? 'Available — the Create tab now offers smarter tip find.'
        : 'Not available — onnxruntime missing or no .onnx at '
          + '(specified or default) path.';
    } catch (err) {
      status.textContent = 'Check failed: ' + err;
    }
  });

  async function refreshCache() {
    try {
      const c = await GetCacheInfo();
      el('#st-cache-info').textContent =
        `${c.files} file(s), ${c.humanSize || '0 B'} — ${c.path}`;
    } catch (err) {
      el('#st-cache-info').textContent = 'unreadable: ' + err;
    }
  }
  refreshCache();

  el('#st-cache-clear').addEventListener('click', async () => {
    try {
      await ClearCache();
      await refreshCache();
    } catch (err) {
      uiError('Clear cache: ' + err, el('#st-cache-info'));
    }
  });

  el('#st-cache-exit').addEventListener('change', e =>
    saveSetting('cache.clearOnExit', e.target.checked));

  el('#st-hardware').addEventListener('click', async () => {
    const out = el('#st-hardware-out');
    out.textContent = 'Checking…';
    try {
      out.textContent = await GetHardwareInfo();
    } catch (err) {
      out.textContent = 'Could not determine: ' + err;
    }
  });

  el('#st-model-info').addEventListener('click', async () => {
    const out = el('#st-report-out');
    out.textContent = 'Querying model…';
    try {
      out.textContent = await QualityModelInfo();
    } catch (err) {
      out.textContent = 'Unavailable: ' + err;
    }
  });

  el('#st-model-train').addEventListener('click', async () => {
    const out = el('#st-report-out');
    out.textContent = 'Learning from ratings…';
    try {
      out.textContent = await TrainQualityModel();
    } catch (err) {
      out.textContent = 'Learning failed: ' + err;
    }
  });

  el('#st-report-summary').addEventListener('click', async () => {
    const out = el('#st-report-out');
    out.textContent = 'Reading values…';
    try {
      out.textContent = await ReportSummary();
    } catch (err) {
      out.textContent = 'No summary available: ' + err;
    }
  });
}
