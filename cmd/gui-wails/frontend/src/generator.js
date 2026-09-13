import { SubmitFeedback, PickVideoFile, LoadFirstFrame, GenerateScript, CheckGeneratorDependencies, ScriptExistsForVideo, AutoDetectROI } from '../wailsjs/go/main/App';
import { EventsOn } from '../wailsjs/runtime/runtime';

export function initGenerator(root, playback) {
  root.innerHTML = `
    <h2>Skript erzeugen</h2>
    <div class="row">
      <button id="gen-choose">Video wählen...</button>
      <span class="path-label" id="gen-video-path">Kein Video gewählt</span>
      <button id="gen-check-deps">Abhängigkeiten prüfen</button>
    </div>
    <div class="row">
      <button id="gen-autoroi" class="primary" disabled>Region automatisch finden</button>
      <span class="hint" style="margin:0">Analysiert die Bewegung im Video - danach lässt sich die Region trotzdem von Hand korrigieren.</span>
    </div>

    <div id="roi-canvas-wrap">
      <canvas id="roi-canvas"></canvas>
    </div>
    <div class="path-label" id="gen-roi-label">Keine Region markiert</div>

    <div class="checkbox-row"><input type="checkbox" id="gen-invert" /><label for="gen-invert">Bewegungsrichtung umkehren</label></div>
    <div class="checkbox-row"><input type="checkbox" id="gen-camcomp" checked /><label for="gen-camcomp">Kamerabewegungs-Kompensation (empfohlen bei Kameraschwenks)</label></div>
    <div class="checkbox-row"><input type="checkbox" id="gen-scenecut" checked /><label for="gen-scenecut">Szenenschnitt-Erkennung (verankert Tracker bei harten Schnitten neu)</label></div>
    <div class="checkbox-row"><input type="checkbox" id="gen-flow" /><label for="gen-flow">Flow-Backend verwenden (keine Region nötig, ca. 4x schneller)</label></div>
    <p class="hint" style="margin:0 0 6px 24px;">Bei ruhiger Kamera gleichwertig. Bei
      Kameraschwenks trifft der Tracker-Weg die Bewegungsstärke besser gemessen 113 gegen
      145 bei 110 tatsächlicher Bewegung —, weil dort die Kamerabewegung über
      Hintergrundmerkmale herausgerechnet wird.</p>
    <div class="row" style="align-items:center;">
      <label style="width:auto;">Bewegungsart</label>
      <select id="gen-profile">
        <option value="standard">Hubbewegung (Standard)</option>
        <option value="weich">Weiches Gewebe (schwingt nach)</option>
      </select>
    </div>
    <p class="hint" style="margin:0 0 10px 0;">„Weiches Gewebe" behandelt Nachschwingungen
      nicht als eigene Hübe. An einem Testvideo mit Anstoß alle 800 ms: 81 Keyframes werden
      zu 42 — den Anstößen selbst. Saubere Hubsignale bleiben davon unberührt.</p>
    <div class="checkbox-row"><input type="checkbox" id="gen-dynrange" checked /><label for="gen-dynrange">Gleitende Dynamik (hebt schwache Abschnitte auf nutzbare Stärke)</label></div>
    <div class="checkbox-row"><input type="checkbox" id="gen-opencl" /><label for="gen-opencl">GPU-Beschleunigung nutzen, falls verfügbar (OpenCL)</label></div>
    <div class="checkbox-row"><input type="checkbox" id="gen-retry" checked /><label for="gen-retry">Auto-Retry (bei schlechter Qualität andere Signalparameter probieren)</label></div>
    <div class="checkbox-row"><input type="checkbox" id="gen-axis-x" /><label for="gen-axis-x">Waagerechte Bewegung auswerten statt senkrechter</label></div>
    <div class="checkbox-row"><input type="checkbox" id="gen-adaptive" checked /><label for="gen-adaptive">Adaptive Keyframes (zusätzliche Punkte bei asymmetrischen Bewegungen)</label></div>
    <div class="checkbox-row"><input type="checkbox" id="gen-perscene" /><label for="gen-perscene">Region nach jedem Schnitt neu suchen (besser bei geschnittenem Material, dauert länger)</label></div>
    <div class="field-row"><label>Glättungs-Fenster</label><input type="number" id="gen-smooth" value="11" /></div>
    <div class="field-row"><label>Min. Keyframe-Abstand (ms)</label><input type="number" id="gen-peakdist" value="150" /></div>
    <div class="field-row"><label>RDP-Toleranz (0 = aus)</label><input type="number" id="gen-rdp" value="0" step="0.5" min="0" /></div>

    <div class="row"><button id="gen-generate" class="primary" disabled>Funscript generieren</button></div>
    <div class="path-label" id="gen-status"></div>
    <div id="gen-progress-wrap" style="display:none; margin-top:8px;">
      <div style="height:10px; border-radius:5px; background:rgba(255,255,255,0.10); overflow:hidden;">
        <div id="gen-progress-bar" style="height:100%; width:0%; background:var(--accent, #00c8ff);
             transition:width .2s linear;"></div>
      </div>
      <div id="gen-progress-text" class="hint" style="margin-top:4px;"></div>
    </div>
    <div id="gen-feedback" style="display:none; margin-top:10px; padding:10px;
         border:1px solid var(--border); border-radius:4px;">
      <div style="margin-bottom:6px;">War das Ergebnis brauchbar? Dein Urteil hilft,
        die Qualitätsbewertung an echtem Material zu justieren.</div>
      <div class="row">
        <button data-verdict="brauchbar">brauchbar</button>
        <button data-verdict="grenzwertig">grenzwertig</button>
        <button data-verdict="unbrauchbar">unbrauchbar</button>
      </div>
      <input type="text" id="gen-fb-comment" placeholder="Kommentar (optional) - z.B. was genau nicht gepasst hat"
             style="width:100%; margin-top:8px;" />
      <div id="gen-fb-status" class="hint" style="margin-top:6px;"></div>
    </div>
    <div id="gen-quality" style="display:none; margin-top:8px; padding:8px; border-radius:4px;"></div>

    <p class="hint">
      Klassisches CV-Tracking (kein KI-Modell) - im Vorschaubild eine Region über
      das zu verfolgende Motiv ziehen, dann generieren. Für deutlich bessere
      Ergebnisse (KI-Objekterkennung, VR, ganze Ordner automatisch): fungen.app -
      erzeugt ebenfalls .funscript-Dateien, die dieser Player direkt abspielen kann.
    </p>
  `;

  const el = id => root.querySelector(id);
  const canvas = el('#roi-canvas');
  const ctx = canvas.getContext('2d');

  let videoPath = null;
  let img = new Image();
  let nativeW = 0, nativeH = 0;
  let roi = null; // {x,y,w,h} in Videopixeln
  let dragging = false, startX = 0, startY = 0, curX = 0, curY = 0;

  const DISPLAY_W = 560;

  function redraw() {
    ctx.clearRect(0, 0, canvas.width, canvas.height);
    if (img.src) ctx.drawImage(img, 0, 0, canvas.width, canvas.height);
    if (dragging || roi) {
      const scaleX = canvas.width / nativeW, scaleY = canvas.height / nativeH;
      let dx0, dy0, dw, dh;
      if (dragging) {
        dx0 = Math.min(startX, curX); dy0 = Math.min(startY, curY);
        dw = Math.abs(curX - startX); dh = Math.abs(curY - startY);
      } else {
        dx0 = roi.x * scaleX; dy0 = roi.y * scaleY; dw = roi.w * scaleX; dh = roi.h * scaleY;
      }
      ctx.strokeStyle = '#00c8ff';
      ctx.lineWidth = 2;
      ctx.strokeRect(dx0, dy0, dw, dh);
      ctx.fillStyle = 'rgba(0,200,255,0.15)';
      ctx.fillRect(dx0, dy0, dw, dh);
    }
  }

  canvas.addEventListener('mousedown', e => {
    const r = canvas.getBoundingClientRect();
    startX = curX = e.clientX - r.left;
    startY = curY = e.clientY - r.top;
    dragging = true;
  });
  canvas.addEventListener('mousemove', e => {
    if (!dragging) return;
    const r = canvas.getBoundingClientRect();
    curX = e.clientX - r.left;
    curY = e.clientY - r.top;
    redraw();
  });
  window.addEventListener('mouseup', () => {
    if (!dragging) return;
    dragging = false;
    const w = Math.abs(curX - startX), h = Math.abs(curY - startY);
    if (w < 4 || h < 4) { redraw(); return; }
    const scaleX = nativeW / canvas.width, scaleY = nativeH / canvas.height;
    const x0 = Math.min(startX, curX), y0 = Math.min(startY, curY);
    roi = {
      x: Math.round(x0 * scaleX), y: Math.round(y0 * scaleY),
      w: Math.round(w * scaleX), h: Math.round(h * scaleY),
    };
    el('#gen-roi-label').textContent = `Region: x=${roi.x} y=${roi.y} w=${roi.w} h=${roi.h} (Videopixel)`;
    el('#gen-generate').disabled = false;
    redraw();
  });

  // Fallengelassenes Video übernehmen. Teilt sich den Ladeweg mit der
  // Dateiauswahl, damit beide Wege garantiert dasselbe tun.
  window.addEventListener('drop:video', e => loadVideo(e.detail));

  async function chooseVideo() {
    const path = await PickVideoFile();
    // Fokus zurück ins Fenster - siehe gleichnamige Behandlung in
    // playback.js: nach dem nativen Datei-Dialog bleibt der Tastaturfokus
    // sonst am Button hängen.
    if (document.activeElement && document.activeElement.blur) {
      document.activeElement.blur();
    }
    window.focus();
    if (!path) return;
    await loadVideo(path);
  }

  async function loadVideo(path) {
    videoPath = path;
    el('#gen-video-path').textContent = path.split(/[\\/]/).pop();
    el('#gen-status').textContent = 'Lade Vorschau-Frame...';
    roi = null;
    el('#gen-generate').disabled = true;
    el('#gen-roi-label').textContent = 'Keine Region markiert';
    try {
      const preview = await LoadFirstFrame(path);
      nativeW = preview.width; nativeH = preview.height;
      const displayH = Math.round(DISPLAY_W * nativeH / nativeW);
      canvas.width = DISPLAY_W; canvas.height = displayH;
      img.onload = redraw;
      img.src = 'data:image/png;base64,' + preview.pngBase64;
      el('#gen-autoroi').disabled = false;
      el('#gen-status').textContent = 'Region automatisch finden lassen oder von Hand markieren (Maus ziehen).';
    } catch (err) {
      el('#gen-status').textContent = '';
      alert('Fehler: ' + err);
    }
  }

  async function checkDeps() {
    try {
      await CheckGeneratorDependencies();
      alert('Python und alle benötigten Pakete sind verfügbar.');
    } catch (err) {
      alert('Fehler: ' + err);
    }
  }

  async function generate() {
    if (!videoPath || !roi) return;

    // Vorhandenes Skript nicht kommentarlos überschreiben - der Nutzer
    // könnte ein von Hand erstelltes oder heruntergeladenes Skript neben
    // dem Video liegen haben.
    let overwrite = false;
    try {
      if (await ScriptExistsForVideo(videoPath)) {
        const target = videoPath.replace(/\.[^.\\/]+$/, '') + '.funscript';
        if (!confirm(`Es existiert bereits ein Skript:\n${target}\n\nWirklich überschreiben?`)) {
          return;
        }
        overwrite = true;
      }
    } catch (err) {
      // Prüfung fehlgeschlagen - lieber nicht überschreiben, das Backend
      // lehnt dann ohnehin ab und meldet es sauber.
    }

    el('#gen-generate').disabled = true;
    el('#gen-status').textContent = 'Generiere...';
    GenerateScript({
      videoPath,
      x: roi.x, y: roi.y, w: roi.w, h: roi.h,
      invert: el('#gen-invert').checked,
      smoothWindow: parseInt(el('#gen-smooth').value, 10) || 11,
      minPeakDistanceMs: parseInt(el('#gen-peakdist').value, 10) || 150,
      disableCameraCompensation: !el('#gen-camcomp').checked,
      disableSceneCutDetection: !el('#gen-scenecut').checked,
      perSceneRoi: el('#gen-perscene').checked,
      adaptiveKeyframeError: el('#gen-adaptive').checked ? 6 : 0,
      autoRetry: el('#gen-retry').checked,
      backend: el('#gen-flow').checked ? 'flow' : 'csrt',
      useOpenCl: el('#gen-opencl').checked,
      dynamicRangeMs: el('#gen-dynrange').checked ? 3000 : 0,
      profile: el('#gen-profile').value,
      axis: el('#gen-axis-x').checked ? 'x' : 'y',
      rdpTolerance: parseFloat(el('#gen-rdp').value) || 0,
      overwrite,
    });
  }

  EventsOn('generate:progress', line => { el('#gen-status').textContent = line; });

  // Ergebnis der automatischen Regionssuche übernehmen - die ROI wird
  // genauso gesetzt, als hätte der Nutzer sie gezogen, und lässt sich
  // danach frei korrigieren.
  EventsOn('generate:autoroi', result => {
    hideProgress();
    el('#gen-autoroi').disabled = false;
    if (result.error) {
      el('#gen-status').textContent = 'Automatische Suche fehlgeschlagen.';
      alert('Fehler: ' + result.error);
      return;
    }
    roi = { x: result.x, y: result.y, w: result.w, h: result.h };
    el('#gen-roi-label').textContent = `Region: x=${roi.x} y=${roi.y} w=${roi.w} h=${roi.h} (Videopixel, automatisch gefunden)`;
    el('#gen-generate').disabled = false;
    el('#gen-status').textContent = 'Region automatisch gefunden - bei Bedarf von Hand korrigieren.';
    redraw();
  });
  // Fortschritt: das Backend schickt 0-100, oder -1 wenn die Frame-Anzahl
  // des Videos nicht ermittelbar war. In dem Fall wird ein unbestimmter
  // Balken gezeigt statt eines erfundenen Prozentwerts.
  let progressStartedAt = 0;
  EventsOn('generate:percent', pct => {
    const wrap = el('#gen-progress-wrap');
    const bar = el('#gen-progress-bar');
    const text = el('#gen-progress-text');
    if (wrap.style.display === 'none') {
      wrap.style.display = 'block';
      progressStartedAt = Date.now();
    }
    if (pct < 0) {
      bar.style.width = '100%';
      bar.style.opacity = '0.35';
      text.textContent = 'Läuft... (Gesamtlänge des Videos nicht ermittelbar)';
      return;
    }
    bar.style.opacity = '1';
    bar.style.width = pct + '%';
    const elapsed = (Date.now() - progressStartedAt) / 1000;
    let rest = '';
    if (pct >= 3 && elapsed > 2) {
      const total = elapsed / (pct / 100);
      const remaining = Math.max(0, Math.round(total - elapsed));
      rest = `  ·  noch ca. ${remaining < 60 ? remaining + ' s' : Math.round(remaining / 60) + ' min'}`;
    }
    text.textContent = `${pct} %${rest}`;
  });

  function hideProgress() {
    el('#gen-progress-wrap').style.display = 'none';
    el('#gen-progress-bar').style.width = '0%';
    el('#gen-progress-text').textContent = '';
  }

  let lastOutputPath = null;

  // Urteil abschicken. Der Bereich erscheint erst nach einem erfolgreichen
  // Lauf - vorher gibt es nichts zu beurteilen.
  root.querySelectorAll('#gen-feedback button[data-verdict]').forEach(btn => {
    btn.addEventListener('click', async () => {
      if (!lastOutputPath) return;
      const status = el('#gen-fb-status');
      try {
        await SubmitFeedback({
          outputPath: lastOutputPath,
          verdict: btn.dataset.verdict,
          comment: el('#gen-fb-comment').value || '',
        });
        status.textContent = `Danke - als "${btn.dataset.verdict}" gespeichert.`;
        el('#gen-fb-comment').value = '';
      } catch (err) {
        status.textContent = 'Konnte nicht gespeichert werden: ' + err;
      }
    });
  });

  EventsOn('generate:done', result => {
    hideProgress();
    lastOutputPath = result.path || null;
    el('#gen-fb-status').textContent = '';
    el('#gen-feedback').style.display = lastOutputPath ? 'block' : 'none';
    el('#gen-generate').disabled = false;
    if (result.error) {
      el('#gen-status').textContent = 'Fehlgeschlagen.';
      alert('Fehler: ' + result.error);
      return;
    }
    el('#gen-status').textContent = 'Fertig: ' + result.path;

    const qualityBox = el('#gen-quality');
    if (typeof result.qualityScore === 'number') {
      const pct = Math.round(result.qualityScore * 100);
      // Das Urteil kommt vom Quality Doctor selbst, nicht aus dem Score: er
      // kennt harte Ausschlusskriterien, die trotz Score >= 0.5 zum Scheitern
      // führen. Nur bei älteren Skripten ohne das Feld auf den Score zurückfallen.
      const ok = typeof result.qualityPassed === 'boolean'
        ? result.qualityPassed
        : result.qualityScore >= 0.5;
      qualityBox.style.display = 'block';
      qualityBox.style.background = ok ? 'rgba(61,216,117,0.12)' : 'rgba(216,77,77,0.12)';
      qualityBox.style.border = `1px solid ${ok ? 'var(--ok)' : 'var(--danger)'}`;
      let html = `<b>Quality Doctor: ${pct}% ${ok ? '(unauffällig)' : '(bitte prüfen)'}</b>`;
      if (result.qualityWarnings && result.qualityWarnings.length > 0) {
        html += '<ul style="margin:6px 0 0 18px; padding:0;">' +
          result.qualityWarnings.map(w => `<li>${w}</li>`).join('') + '</ul>';
      }
      qualityBox.innerHTML = html;
    } else {
      qualityBox.style.display = 'none';
    }

    if (confirm(`${result.path}\n\nJetzt im Wiedergabe-Tab laden?`)) {
      playback.loadScriptPath(result.path);
    }
  });

  el('#gen-choose').addEventListener('click', chooseVideo);
  el('#gen-check-deps').addEventListener('click', checkDeps);
  el('#gen-generate').addEventListener('click', generate);
  el('#gen-autoroi').addEventListener('click', () => {
    if (!videoPath) return;
    el('#gen-autoroi').disabled = true;
    el('#gen-status').textContent = 'Analysiere Bewegung im Video (dauert einige Sekunden)...';
    AutoDetectROI(videoPath);
  });
}
