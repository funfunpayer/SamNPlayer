import { GetSettings, SetSetting, PickReportPath, ReportSummary, ReportExists, GetHardwareInfo, GetCacheInfo, ClearCache, TrainQualityModel, QualityModelInfo, OpenLogFolder, CheckAIRoiAvailable, CurrentVersion, CheckForUpdate, ApplyUpdate, GetRuntimeHealth } from '../wailsjs/go/main/App';

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
      'generator.aiBaseUrl': 'aiBaseUrl',
      'generator.roiTrainingDatasetDir': 'roiDatasetDir',
    };
    const field = map[key];
    if (field) cachedSettings[field] = value;
  }
  return SetSetting(key, value);
}

export function initSettings(root) {
  root.innerHTML = `
    <h2>Einstellungen</h2>
    <div class="checkbox-row">
      <input type="checkbox" id="st-update-check" />
      <label for="st-update-check">Beim Start automatisch nach Updates suchen</label>
    </div>
    <div class="checkbox-row">
      <input type="checkbox" id="st-connect-test" />
      <label for="st-connect-test">Beim Verbinden einmal Verbindung testen (kurz Vib/Sog)</label>
    </div>
    <p class="hint" style="margin-top:0">Standard aus. Wenn an: nach erfolgreichem Connect ein kurzer Impuls, damit klar ist, dass Steuerbefehle ankommen.</p>
    <div class="row" style="align-items:center;">
      <button id="st-runtime-check" type="button">Ordner &amp; Abhängigkeiten prüfen</button>
      <span class="hint" id="st-runtime-status" style="margin:0"></span>
    </div>
    <div class="row" style="align-items:center;">
      <button id="st-update-now" type="button">Jetzt nach Updates suchen</button>
      <span class="hint" id="st-update-status" style="margin:0"></span>
    </div>
    <div class="field-row"><label>Log-Level</label>
      <select id="st-log-level">
        <option value="debug">debug</option>
        <option value="info">info</option>
        <option value="warn">warn</option>
        <option value="error">error</option>
      </select>
    </div>
    <div class="row">
      <span class="path-label" id="st-log-path"></span>
      <button id="st-open-log">Log-Ordner öffnen</button>
    </div>

    <h3>KI-Regionserkennung (lokal, optional)</h3>
    <p class="hint">Lokales ONNX-Objekterkennungsmodell als Alternative zur klassischen
      Rhythmus-Heuristik im Generator-Tab ("Region automatisch finden"). Kein Modell liegt
      diesem Programm bei und keins wird heruntergeladen - ohne eigene .onnx-Datei bleibt
      es bei der klassischen Erkennung. Leer lassen nutzt den Standardordner
      (<code>%LOCALAPPDATA%\\SamNPlayer\\models\\roi_detector.onnx</code> unter Windows).</p>
    <div class="row">
      <input type="text" id="st-ai-roi-path" placeholder="(Standardordner)" style="flex:1;" />
      <button id="st-ai-roi-check">Verfügbarkeit prüfen</button>
    </div>
    <p class="hint" id="st-ai-roi-status" style="margin-top:0"></p>

    <h3>KI-Server für Profil-Vorschlag &amp; Qualitäts-Zweitmeinung (lokal, optional)</h3>
    <p class="hint">Adresse eines lokal laufenden Colibri-Servers (<code>coli serve</code>,
      siehe docs/AI_ADAPTER.md) für den "Profil vorschlagen"-Knopf und die
      KI-Zweitmeinung im Generator-Tab. Beides funktioniert auch ohne diesen Server -
      die gemessene Szenen-Ähnlichkeit (ohne KI) bleibt dann die einzige Quelle für
      Profil-Vorschläge. Leer lassen nutzt die Standardadresse.</p>
    <div class="row">
      <input type="text" id="st-ai-base-url" placeholder="(Standardadresse)" style="flex:1;" />
    </div>

    <h3>Hardware</h3>
    <p class="hint">Welche Beschleunigung der Generator tatsächlich nutzen kann. Eine
      vorhandene NVIDIA-Karte bedeutet nicht automatisch, dass sie genutzt wird — die
      üblichen pip-Pakete von OpenCV sind ohne CUDA gebaut.</p>
    <div class="row"><button id="st-hardware">Hardware prüfen</button></div>
    <pre id="st-hardware-out" class="hint" style="white-space:pre-wrap; margin-top:6px;"></pre>

    <h3>Zwischenspeicher</h3>
    <p class="hint">Der Generator speichert Trackingergebnisse, damit ein erneuter Lauf mit
      anderen Einstellungen nicht das ganze Video neu dekodieren muss — das ist rund 36-mal
      schneller. Dafür wächst der Speicher mit jedem Video.</p>
    <div class="row">
      <span class="path-label" id="st-cache-info">…</span>
      <button id="st-cache-clear">Jetzt leeren</button>
    </div>
    <div class="checkbox-row">
      <input type="checkbox" id="st-cache-exit" />
      <label for="st-cache-exit">Beim Beenden automatisch leeren</label>
    </div>

    <h3>Messwerte des Generators</h3>
    <p class="hint">Schreibt zu jedem Generatorlauf eine Zeile mit allen Kennzahlen
      (Bewegungsamplitude, spektrale Konzentration, Tracker-Verlust, Laufzeit) in eine
      Datei. Zusammen mit deinem Urteil im Generator-Tab ist das die Grundlage, um die
      Qualitätsbewertung an echtem Material zu justieren - bisher beruht sie auf
      synthetischen Testvideos. Leer lassen schaltet die Aufzeichnung ab.</p>
    <div class="row">
      <input type="text" id="st-report-path" placeholder="(keine Aufzeichnung)"
             style="flex:1;" />
      <button id="st-pick-report">Wählen…</button>
      <button id="st-default-report">Standard</button>
    </div>
    <p class="hint" id="st-report-status" style="margin-top:2px;"></p>
    <div class="row">
      <button id="st-report-summary">Auswertung anzeigen</button>
      <button id="st-model-info">Modell anzeigen</button>
      <button id="st-model-train" class="primary">Aus Urteilen lernen</button>
    </div>
    <p class="hint">Die Qualitätsbewertung arbeitet mit Schwellen, die an synthetischen
      Testvideos festgelegt wurden. Aus genügend eigenen Urteilen lässt sich stattdessen
      ein Modell lernen. Es wird nur übernommen, wenn es die bisherigen Regeln in einer
      Kreuzvalidierung schlägt — sonst bleibt alles, wie es ist.</p>
    <pre id="st-report-out" class="hint" style="white-space:pre-wrap; margin-top:6px;"></pre>
  `;

  const el = id => root.querySelector(id);

  // Ohne das war unsichtbar, ob unter dem eingestellten Pfad schon Messwerte
  // stehen - "Auswertung anzeigen" beantwortete das zwar auch, aber erst
  // nach einem Klick und mit einer Fehlermeldung statt eines einfachen
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
        ? '✓ Enthält bereits Messwerte.'
        : 'Noch keine Messwerte aufgezeichnet (wird beim nächsten Generatorlauf angelegt).';
    } catch (err) {
      status.textContent = '';
    }
  }

  getSettingsCache().then(s => {
    el('#st-update-check').checked = s.updateCheckOnStartup;
    el('#st-connect-test').checked = !!s.deviceConnectTest;
    el('#st-log-level').value = s.logLevel;
    el('#st-log-path').textContent = s.logPath || '(noch keine Logdatei geschrieben)';
    el('#st-report-path').value = s.reportPath || '';
    el('#st-cache-exit').checked = !!s.clearCacheOnExit;
    el('#st-report-path').dataset.default = s.defaultReportPath || '';
    el('#st-ai-roi-path').value = s.aiRoiModelPath || '';
    el('#st-ai-base-url').value = s.aiBaseUrl || '';
    updateReportStatus();
  });

  el('#st-update-check').addEventListener('change', e => saveSetting('update.check_on_startup', e.target.checked));
  el('#st-connect-test').addEventListener('change', e => saveSetting('device.connect_test', e.target.checked));
  el('#st-runtime-check').addEventListener('click', async () => {
    const status = el('#st-runtime-status');
    status.textContent = 'Prüfe…';
    try {
      const h = await GetRuntimeHealth();
      const missing = (h.deps || []).filter(d => !d.found).map(d => d.label);
      const created = (h.dirsCreated || []).length;
      const parts = [];
      if (created) parts.push(`${created} Ordner angelegt`);
      if (missing.length) parts.push('fehlt: ' + missing.join(', '));
      else parts.push('Abhängigkeiten ok');
      status.textContent = (h.ok ? '✓ ' : '⚠ ') + parts.join(' · ');
    } catch (err) {
      status.textContent = 'Prüfung fehlgeschlagen: ' + err;
    }
  });
  el('#st-update-now').addEventListener('click', async () => {
    const status = el('#st-update-status');
    const btn = el('#st-update-now');
    btn.disabled = true;
    status.textContent = 'Prüfe...';
    try {
      const version = await CurrentVersion();
      const res = await CheckForUpdate();
      if (res.error) {
        status.textContent = 'Prüfung fehlgeschlagen: ' + res.error;
      } else if (!res.available) {
        status.textContent = `Kein Update verfügbar (aktuell: ${version}).`;
      } else {
        const tag = res.release ? res.release.tag_name : '?';
        status.textContent = `Version ${tag} verfügbar (aktuell: ${version}).`;
        if (confirm(`Version ${tag} ist verfügbar (aktuell: ${version}).\n\nJetzt herunterladen und neu starten?`)) {
          try {
            await ApplyUpdate();
          } catch (err) {
            status.textContent = 'Update fehlgeschlagen: ' + err;
          }
        }
      }
    } catch (err) {
      status.textContent = 'Prüfung fehlgeschlagen: ' + err;
    } finally {
      btn.disabled = false;
    }
  });
  el('#st-log-level').addEventListener('change', e => saveSetting('log.level', e.target.value));
  el('#st-open-log').addEventListener('click', () => OpenLogFolder().catch(err => alert('Fehler: ' + err)));

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
      alert('Fehler: ' + err);
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

  el('#st-ai-base-url').addEventListener('change', e =>
    saveSetting('generator.aiBaseUrl', e.target.value.trim()));

  el('#st-ai-roi-check').addEventListener('click', async () => {
    const status = el('#st-ai-roi-status');
    status.textContent = 'Prüfe...';
    try {
      const available = await CheckAIRoiAvailable();
      status.textContent = available
        ? 'Verfügbar - der Generator-Tab bietet die KI-Erkennung jetzt an.'
        : 'Nicht verfügbar - onnxruntime fehlt oder es liegt keine .onnx-Datei am '
          + '(angegebenen oder Standard-) Pfad.';
    } catch (err) {
      status.textContent = 'Prüfung fehlgeschlagen: ' + err;
    }
  });

  async function refreshCache() {
    try {
      const c = await GetCacheInfo();
      el('#st-cache-info').textContent =
        `${c.files} Datei(en), ${c.humanSize || '0 B'} — ${c.path}`;
    } catch (err) {
      el('#st-cache-info').textContent = 'nicht lesbar: ' + err;
    }
  }
  refreshCache();

  el('#st-cache-clear').addEventListener('click', async () => {
    try {
      await ClearCache();
      await refreshCache();
    } catch (err) {
      alert('Fehler: ' + err);
    }
  });

  el('#st-cache-exit').addEventListener('change', e =>
    saveSetting('cache.clearOnExit', e.target.checked));

  el('#st-hardware').addEventListener('click', async () => {
    const out = el('#st-hardware-out');
    out.textContent = 'Prüfe...';
    try {
      out.textContent = await GetHardwareInfo();
    } catch (err) {
      out.textContent = 'Nicht ermittelbar: ' + err;
    }
  });

  el('#st-model-info').addEventListener('click', async () => {
    const out = el('#st-report-out');
    out.textContent = 'Frage Modell ab...';
    try {
      out.textContent = await QualityModelInfo();
    } catch (err) {
      out.textContent = 'Nicht abrufbar: ' + err;
    }
  });

  el('#st-model-train').addEventListener('click', async () => {
    const out = el('#st-report-out');
    out.textContent = 'Lerne aus den Urteilen...';
    try {
      out.textContent = await TrainQualityModel();
    } catch (err) {
      out.textContent = 'Lernen fehlgeschlagen: ' + err;
    }
  });

  el('#st-report-summary').addEventListener('click', async () => {
    const out = el('#st-report-out');
    out.textContent = 'Werte aus...';
    try {
      out.textContent = await ReportSummary();
    } catch (err) {
      out.textContent = 'Keine Auswertung möglich: ' + err;
    }
  });
}
