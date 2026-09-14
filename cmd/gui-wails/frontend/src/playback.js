import {
  PickFunscriptFile, LoadFunscript, StartPlayback, StopPlayback,
  TriggerExtendedO, VideoFileURL, GetHeatmap, GetScriptCurve, AnalyzeScript, SetScriptOffset, GetScriptOffset, GetMarker, SaveMarker,
} from '../wailsjs/go/main/App';
import { EventsOn } from '../wailsjs/runtime/runtime';
import { getSettingsCache, saveSetting } from './settings.js';

const HEATMAP_BUCKETS = 300;

export function initPlayback(root) {
  root.innerHTML = `
    <h2>Wiedergabe</h2>
    <div class="row">
      <button id="pb-choose">Funscript wählen...</button>
      <span class="path-label" id="pb-script-path">Kein Skript gewählt</span>
    </div>

    <video id="pb-video" controls style="display:none"></video>
    <div id="pb-novideo" class="hint">Kein passendes Video im selben Ordner gefunden.</div>

    <canvas id="pb-curve" height="110" style="width:100%; display:none; border-radius:4px;
            margin-top:8px; background:rgba(255,255,255,0.04); cursor:crosshair;"></canvas>
    <div class="row" id="pb-offset-row" style="display:none; align-items:center; margin-top:8px;">
      <label style="width:auto;">Skript-Offset</label>
      <button id="pb-offset-minus" title="Skript 50ms früher (Taste -)">−50</button>
      <input type="number" id="pb-offset" value="0" step="10" style="width:90px;" />
      <span class="hint" style="margin:0;">ms</span>
      <button id="pb-offset-plus" title="Skript 50ms später (Taste +)">+50</button>
      <button id="pb-offset-reset">zurücksetzen</button>
      <span class="checkbox-row" style="margin:0 0 0 12px;">
        <input type="checkbox" id="pb-loop" />
        <label for="pb-loop" style="width:auto;">Markierung wiederholen</label>
      </span>
    </div>
    <p class="hint" id="pb-offset-hint" style="display:none; margin-top:0;">
      Positiver Wert = Skript greift später. Wirkt sofort, auch während der Wiedergabe.
      Wird je Skript gespeichert. Zum Einstellen: Abschnitt markieren, Wiederholung
      einschalten und mit + / − nachjustieren.<br>
      <b>Tasten:</b> Leertaste Start/Stop · ←/→ 5 s (mit Shift 1 s) · , und . Feinschritt ·
      1–9 springen · + / − Offset · L Wiederholung · E Extended-O</p>
    <div id="pb-analysis" class="hint" style="display:none; margin-top:6px;"></div>
    <canvas id="pb-heatmap" height="28" style="width:100%; display:none; border-radius:4px; margin-top:8px; cursor:crosshair;"></canvas>
    <div class="hint" id="pb-marker-hint" style="display:none">
      Klick auf die Leiste = an diese Stelle springen. Ziehen = Bereich markieren, in dem Extended-O automatisch auslöst.
      <span id="pb-marker-label"></span>
      <button id="pb-marker-clear" style="margin-left:8px">Markierung löschen</button>
    </div>
    <div class="checkbox-row" id="pb-marker-auto-row" style="display:none">
      <input type="checkbox" id="pb-marker-auto" />
      <label for="pb-marker-auto">Extended-O automatisch im markierten Bereich auslösen</label>
    </div>

    <div class="checkbox-row" id="pb-video-sync-row" style="display:none">
      <input type="checkbox" id="pb-use-video-sync" checked />
      <label for="pb-use-video-sync">Gerät folgt der echten Videoposition (empfohlen, statt eigener Uhr)</label>
    </div>

    <div class="field-row"><label>Gerät</label>
      <span class="checkbox-row" style="margin:0"><input type="checkbox" id="pb-mock" /> <label for="pb-mock" style="width:auto">Mock (ohne Gerät testen)</label></span>
    </div>
    <div class="field-row"><label>Sync-Modus</label>
      <select id="pb-sync">
        <option value="independent">independent</option>
        <option value="synchronized">synchronized</option>
        <option value="alternating">alternating</option>
        <option value="vibration_only">nur Vibration</option>
        <option value="suction_only">nur Sog</option>
        <option value="suction_position">Sog aus Position</option>
      </select>
    </div>
    <div class="field-row"><label>Tick (ms)</label><input type="number" id="pb-tick" value="50" /></div>
    <div class="field-row"><label>Max-Speed</label><input type="number" step="0.1" id="pb-maxspeed" value="0.6" /></div>
    <div class="field-row"><label>Glättung (0-1)</label><input type="number" step="0.05" min="0" max="1" id="pb-smoothing" value="0.3" /></div>
    <div class="field-row"><label>Soft-Start (ms)</label><input type="number" step="100" min="0" id="pb-softstart" value="500" /></div>

    <div class="checkbox-row"><input type="checkbox" id="pb-eo-enabled" checked /><label for="pb-eo-enabled">Extended-O aktiv</label></div>
    <div class="field-row"><label>Min-Intensität</label><input type="number" step="0.05" id="pb-eo-min" value="0.1" /></div>
    <div class="field-row"><label>Hold (s)</label><input type="number" id="pb-eo-hold" value="10" /></div>
    <div class="field-row"><label>Restore (ms)</label><input type="number" id="pb-eo-restore" value="500" /></div>

    <div class="row">
      <button id="pb-play" class="primary">▶ Abspielen</button>
      <button id="pb-stop" disabled>■ Stop</button>
      <button id="pb-eo-trigger" disabled>Extended-O auslösen</button>
    </div>

    <div class="progress-bar"><div class="progress-bar-fill" id="pb-progress"></div></div>
    <div class="stat-row">
      <span>Vibration: <b id="pb-vib">-</b></span>
      <span>Sog: <b id="pb-suc">-</b></span>
    </div>
    <p class="hint">Tastenkürzel: Leertaste = Abspielen/Stop, E = Extended-O auslösen (wenn aktiv).</p>
    <div id="pb-log"></div>
  `;

  const el = id => root.querySelector(id);
  const videoEl = el('#pb-video');
  const heatmapCanvas = el('#pb-heatmap');
  let scriptPath = null;
  let videoPath = null;
  let totalMs = 1;
  let playing = false;
  let heatmapPoints = null;
  let curvePoints = null;
  const curveCanvas = el('#pb-curve');
  const CURVE_MAX_POINTS = 1200;
  let marker = null; // {startMs, endMs} oder null
  let markerDragStartMs = null;
  let markerDragMoved = false;
  let autoEOTriggeredForMarker = false;
  let currentPosMs = 0;

  function log(line) {
    const box = el('#pb-log');
    box.textContent += (box.textContent ? '\n' : '') + line;
    box.scrollTop = box.scrollHeight;
  }

  function setPlayingState(isPlaying) {
    playing = isPlaying;
    el('#pb-play').disabled = isPlaying;
    el('#pb-stop').disabled = !isPlaying;
    el('#pb-eo-trigger').disabled = !isPlaying || !el('#pb-eo-enabled').checked;
    if (!isPlaying) el('#pb-progress').style.width = '0%';
    if (isPlaying) autoEOTriggeredForMarker = false;
  }

  function updateMarkerHint() {
    if (marker) {
      el('#pb-marker-label').textContent =
        `Markiert: ${(marker.startMs / 1000).toFixed(1)}s - ${(marker.endMs / 1000).toFixed(1)}s`;
    } else {
      el('#pb-marker-label').textContent = '(keine Markierung)';
    }
  }

  // drawHeatmap zeichnet die grob gerasterte Intensitätskurve als
  // Farbverlauf (blau=ruhig -> rot=intensiv) - dieselbe Idee wie die
  // Heatmap-Leisten in MultiFunPlayer & Co, zeigt auf einen Blick, wo im
  // Skript viel/wenig passiert. Ein markierter Bereich wird als heller
  // Rahmen darüber gezeichnet.
  function redrawHeatmap() {
    if (!heatmapPoints || heatmapPoints.length === 0) return;
    const ctx = heatmapCanvas.getContext('2d');
    const w = heatmapCanvas.width, h = heatmapCanvas.height;
    const barWidth = w / heatmapPoints.length;
    for (let i = 0; i < heatmapPoints.length; i++) {
      const intensity = Math.max(0, Math.min(1, heatmapPoints[i].intensity));
      const hue = 220 - intensity * 220;
      ctx.fillStyle = `hsl(${hue}, 75%, ${35 + intensity * 20}%)`;
      ctx.fillRect(i * barWidth, 0, barWidth + 1, h);
    }
    if (marker && totalMs > 0) {
      const x0 = (marker.startMs / totalMs) * w;
      const x1 = (marker.endMs / totalMs) * w;
      ctx.strokeStyle = '#ffffff';
      ctx.lineWidth = 2;
      ctx.strokeRect(x0, 1, Math.max(1, x1 - x0), h - 2);
    }
    // Aktuelle Position als senkrechter Strich - macht die Leiste zur
    // vollwertigen Zeitleiste, nicht nur zur Übersicht.
    if (currentPosMs > 0 && totalMs > 0) {
      const x = (currentPosMs / totalMs) * w;
      ctx.strokeStyle = '#ffffff';
      ctx.lineWidth = 2;
      ctx.beginPath();
      ctx.moveTo(x, 0);
      ctx.lineTo(x, h);
      ctx.stroke();
    }
  }


  // --- Funscript-Kurve unter dem Video -----------------------------------
  // Zeigt den tatsächlichen Positionsverlauf (0-100) über die Zeit, plus
  // einen mitlaufenden Positionszeiger. Die Heatmap-Leiste darunter bleibt
  // erhalten: sie gibt den groben Überblick, die Kurve die genaue Form.
  function redrawCurve() {
    if (!curvePoints || curvePoints.length < 2) return;
    const ctx = curveCanvas.getContext('2d');
    const w = curveCanvas.width, h = curveCanvas.height;
    const pad = 6;
    const usableH = h - pad * 2;
    ctx.clearRect(0, 0, w, h);

    const xOf = ms => (ms / Math.max(1, totalMs)) * w;
    const yOf = pos => pad + (1 - pos / 100) * usableH;

    // Hilfslinien bei 0 / 50 / 100 - ohne Bezug ist die Amplitude nicht
    // einzuschätzen.
    ctx.strokeStyle = 'rgba(255,255,255,0.10)';
    ctx.lineWidth = 1;
    for (const pos of [0, 50, 100]) {
      const y = Math.round(yOf(pos)) + 0.5;
      ctx.beginPath();
      ctx.moveTo(0, y);
      ctx.lineTo(w, y);
      ctx.stroke();
    }

    // Markierter Bereich (dieselbe Markierung wie in der Heatmap).
    if (marker) {
      ctx.fillStyle = 'rgba(0,200,255,0.12)';
      ctx.fillRect(xOf(marker.startMs), 0, xOf(marker.endMs) - xOf(marker.startMs), h);
    }

    // Die Kurve selbst.
    ctx.strokeStyle = 'var(--accent)';
    ctx.strokeStyle = '#00c8ff';
    ctx.lineWidth = 1.5;
    ctx.lineJoin = 'round';
    ctx.beginPath();
    ctx.moveTo(xOf(curvePoints[0].atMs), yOf(curvePoints[0].pos));
    for (let i = 1; i < curvePoints.length; i++) {
      ctx.lineTo(xOf(curvePoints[i].atMs), yOf(curvePoints[i].pos));
    }
    ctx.stroke();

    // Positionszeiger.
    if (currentPosMs > 0 && totalMs > 0) {
      const x = Math.round(xOf(currentPosMs)) + 0.5;
      ctx.strokeStyle = '#ffffff';
      ctx.lineWidth = 1.5;
      ctx.beginPath();
      ctx.moveTo(x, 0);
      ctx.lineTo(x, h);
      ctx.stroke();
    }
  }

  // Woraus besteht das Skript? Der Quality Doctor sagt, ob es brauchbar
  // aussieht, aber nicht, ob es über weite Strecken stillsteht - eine flache
  // Strecke ist weder verrauscht noch unrhythmisch und fällt den übrigen
  // Prüfungen deshalb nicht auf.
  function seekBy(deltaMs) {
    if (!totalMs) return;
    seekTo(Math.max(0, Math.min(totalMs, currentPosMs + deltaMs)));
  }

  async function applyOffset(ms) {
    const value = Math.max(-10000, Math.min(10000, Math.round(ms)));
    el('#pb-offset').value = value;
    try {
      await SetScriptOffset(value);
    } catch (err) {
      log('Offset: ' + err);
    }
  }

  el('#pb-offset').addEventListener('change', e => applyOffset(Number(e.target.value) || 0));
  el('#pb-offset-minus').addEventListener('click',
    () => applyOffset((Number(el('#pb-offset').value) || 0) - 50));
  el('#pb-offset-plus').addEventListener('click',
    () => applyOffset((Number(el('#pb-offset').value) || 0) + 50));
  el('#pb-offset-reset').addEventListener('click', () => applyOffset(0));

  async function describeScript() {
    const box = el('#pb-analysis');
    try {
      const analysis = await AnalyzeScript();
      box.textContent = analysis.summary || '';
      box.style.display = analysis.summary ? 'block' : 'none';
    } catch (err) {
      box.style.display = 'none';
    }
  }

  async function drawCurve() {
    try {
      curvePoints = await GetScriptCurve(CURVE_MAX_POINTS);
    } catch (err) {
      curvePoints = null;
      curveCanvas.style.display = 'none';
      return;
    }
    if (!curvePoints || curvePoints.length < 2) {
      curveCanvas.style.display = 'none';
      return;
    }
    curveCanvas.style.display = 'block';
    curveCanvas.width = curveCanvas.clientWidth || 800;
    redrawCurve();
  }

  // Klick in die Kurve springt an die Stelle - gleiche Bedienung wie die
  // Heatmap darunter, damit man nicht überlegen muss, welche Leiste was tut.
  curveCanvas.addEventListener('click', (e) => {
    if (!totalMs) return;
    const rect = curveCanvas.getBoundingClientRect();
    const frac = Math.max(0, Math.min(1, (e.clientX - rect.left) / rect.width));
    seekTo(Math.round(frac * totalMs));
  });

  async function drawHeatmap() {
    try {
      heatmapPoints = await GetHeatmap(HEATMAP_BUCKETS);
    } catch (err) {
      heatmapPoints = null;
      heatmapCanvas.style.display = 'none';
      el('#pb-marker-hint').style.display = 'none';
      el('#pb-marker-auto-row').style.display = 'none';
      return;
    }
    if (!heatmapPoints || heatmapPoints.length === 0) {
      heatmapCanvas.style.display = 'none';
      el('#pb-marker-hint').style.display = 'none';
      el('#pb-marker-auto-row').style.display = 'none';
      return;
    }
    heatmapCanvas.style.display = 'block';
    el('#pb-marker-hint').style.display = 'block';
    el('#pb-marker-auto-row').style.display = 'flex';
    heatmapCanvas.width = heatmapCanvas.clientWidth || 800;
    redrawHeatmap();
  }

  function canvasXToMs(clientX) {
    const rect = heatmapCanvas.getBoundingClientRect();
    const frac = Math.max(0, Math.min(1, (clientX - rect.left) / rect.width));
    return Math.round(frac * totalMs);
  }

  heatmapCanvas.addEventListener('mousedown', (e) => {
    markerDragStartMs = canvasXToMs(e.clientX);
    markerDragMoved = false;
  });
  heatmapCanvas.addEventListener('mousemove', (e) => {
    if (markerDragStartMs === null) return;
    const cur = canvasXToMs(e.clientX);
    if (Math.abs(cur - markerDragStartMs) > 150) markerDragMoved = true;
    if (!markerDragMoved) return;
    marker = { startMs: Math.min(markerDragStartMs, cur), endMs: Math.max(markerDragStartMs, cur) };
    redrawHeatmap();
    redrawCurve();
  });
  window.addEventListener('mouseup', () => {
    if (markerDragStartMs === null) return;
    const clickedMs = markerDragStartMs;
    const wasDrag = markerDragMoved;
    markerDragStartMs = null;
    markerDragMoved = false;

    // Klick ohne Ziehen = an diese Stelle springen (statt zu markieren).
    // Markiert wird nur beim echten Ziehen - so lässt sich dieselbe Leiste
    // für beides nutzen, ohne Moduswechsel.
    if (!wasDrag) {
      seekTo(clickedMs);
      return;
    }

    updateMarkerHint();
    redrawHeatmap();
    if (scriptPath) {
      SaveMarker(scriptPath, marker ? marker.startMs : 0, marker ? marker.endMs : 0).catch(() => {});
    }
  });

  // seekTo springt im Video an die angegebene Stelle. Bei aktivem
  // Video-Sync folgt das Gerät automatisch mit, da die Videoposition
  // ohnehin die Quelle ist (siehe player/sync.go) - es ist kein
  // zusätzliches Zutun nötig. Ohne Video gibt es nichts zu spulen; dann
  // bleibt der Klick wirkungslos (die eigene Uhr in Play() lässt sich
  // nicht nachträglich verschieben).
  function seekTo(atMs) {
    if (!videoPath || !videoEl.duration) return;
    videoEl.currentTime = Math.max(0, Math.min(videoEl.duration, atMs / 1000));
    // Auto-Extended-O darf nach einem Sprung erneut auslösen, wenn der
    // markierte Bereich danach nochmal erreicht wird.
    if (marker && atMs < marker.startMs) autoEOTriggeredForMarker = false;
    redrawHeatmap();
  }

  el('#pb-marker-clear').addEventListener('click', () => {
    marker = null;
    updateMarkerHint();
    redrawHeatmap();
    if (scriptPath) SaveMarker(scriptPath, 0, 0).catch(() => {});
  });

  // checkAutoExtendedO wird bei jedem Fortschritts-Update aufgerufen -
  // löst Extended-O einmal pro Wiedergabe aus, sobald die Position in den
  // markierten Bereich eintritt (falls aktiviert).
  function checkAutoExtendedO(atMs) {
    if (!marker || !el('#pb-marker-auto').checked || autoEOTriggeredForMarker) return;
    if (atMs >= marker.startMs && atMs <= marker.endMs) {
      autoEOTriggeredForMarker = true;
      if (!el('#pb-eo-trigger').disabled) triggerEO();
    }
  }



  // Fallengelassenes Skript übernehmen - gleicher Ladeweg wie die
  // Dateiauswahl.
  window.addEventListener('drop:script', e => loadScript(e.detail));

  async function chooseScript() {
    const path = await PickFunscriptFile();
    // Fokus zurück ins Fenster holen: nach dem Schließen des nativen
    // Datei-Dialogs bleibt der Tastaturfokus sonst am Auswahl-Button
    // hängen (oder ganz außerhalb der Webview), und die Tastenkürzel
    // wirken scheinbar nicht mehr. In der Testumgebung reproduzierbar,
    // darum hier defensiv behandelt.
    restoreKeyboardFocus();
    if (!path) return;
    await loadScript(path);
  }

  // restoreKeyboardFocus nimmt den Fokus von einem eventuell fokussierten
  // Bedienelement und legt ihn aufs Dokument, damit globale Tastenkürzel
  // (Leertaste, E) wieder greifen.
  function restoreKeyboardFocus() {
    if (document.activeElement && document.activeElement.blur) {
      document.activeElement.blur();
    }
    window.focus();
  }

  async function loadScript(path) {
    const info = await LoadFunscript(path);
    scriptPath = info.path;
    totalMs = Math.max(info.durationMs, 1);
    el('#pb-script-path').textContent = scriptPath;
    if (info.hasVideo) {
      videoPath = info.videoPath;
      videoEl.src = await VideoFileURL();
      videoEl.style.display = 'block';
      el('#pb-novideo').style.display = 'none';
      el('#pb-video-sync-row').style.display = 'flex';
    } else {
      videoPath = null;
      videoEl.style.display = 'none';
      el('#pb-novideo').style.display = 'block';
      el('#pb-video-sync-row').style.display = 'none';
    }
    try {
      marker = await GetMarker(scriptPath);
    } catch (err) {
      marker = null;
    }
    updateMarkerHint();
    drawHeatmap();
    drawCurve();
    describeScript();
    el('#pb-offset-row').style.display = 'flex';
    el('#pb-offset-hint').style.display = 'block';
    GetScriptOffset().then(v => { el('#pb-offset').value = v || 0; }).catch(() => {});
  }

  async function play() {
    if (!scriptPath) { alert('Bitte zuerst eine .funscript-Datei wählen.'); return; }
    el('#pb-log').textContent = '';

    const useVideoSync = videoPath && el('#pb-use-video-sync').checked;
    const opts = {
      mock: el('#pb-mock').checked,
      syncMode: el('#pb-sync').value,
      tickMs: parseInt(el('#pb-tick').value, 10) || 50,
      maxSpeed: parseFloat(el('#pb-maxspeed').value) || 0.6,
      smoothing: parseFloat(el('#pb-smoothing').value) || 0,
      softStartMs: parseInt(el('#pb-softstart').value, 10) || 0,
      useVideoSync: !!useVideoSync,
      extendedOEnabled: el('#pb-eo-enabled').checked,
      extendedOMin: parseFloat(el('#pb-eo-min').value) || 0.1,
      extendedOHoldS: parseFloat(el('#pb-eo-hold').value) || 10,
      extendedORestoreMs: parseFloat(el('#pb-eo-restore').value) || 500,
    };

    try {
      await StartPlayback(opts);
    } catch (err) {
      alert('Fehler: ' + err);
      return;
    }
    setPlayingState(true);

    if (useVideoSync) {
      videoEl.currentTime = 0;
      videoEl.play().catch(() => {});
    }
  }

  async function stop() {
    await StopPlayback();
    if (videoPath) videoEl.pause();
    setPlayingState(false);
  }

  async function triggerEO() {
    await TriggerExtendedO(
      parseFloat(el('#pb-eo-min').value) || 0.1,
      parseFloat(el('#pb-eo-hold').value) || 10,
      parseFloat(el('#pb-eo-restore').value) || 500,
    );
  }

  // Video-Position ans Backend melden, solange Sync aktiv ist - das treibt
  // player.Sync() an (siehe player/sync.go). timeupdate feuert im Browser
  // ca. alle 250ms, das reicht für flüssiges Gerätefeedback.
  videoEl.addEventListener('timeupdate', () => {
    currentPosMs = Math.round(videoEl.currentTime * 1000);

    // Markierten Abschnitt wiederholen. Das ist das Werkzeug, mit dem sich
    // der Offset überhaupt einstellen lässt: ohne Wiederholung müsste man
    // nach jeder Korrektur von Hand zurückspulen, und bis man wieder an der
    // fraglichen Stelle ist, hat man den Vergleich verloren.
    if (el('#pb-loop').checked && marker && currentPosMs >= marker.endMs) {
      videoEl.currentTime = marker.startMs / 1000;
      currentPosMs = marker.startMs;
    }
    if (!playing) { redrawHeatmap(); redrawCurve(); }
    if (playing && el('#pb-use-video-sync').checked) {
      import('../wailsjs/go/main/App').then(m => m.ReportVideoPosition(Math.round(videoEl.currentTime * 1000)));
    }
  });

  // Extended-O soll (wenn gewünscht) das Video pausieren/fortsetzen -
  // siehe Player.PauseVideo/ResumeVideo-Hooks im Go-Backend.
  EventsOn('video:pause', () => videoEl.pause());
  EventsOn('video:resume', () => videoEl.play().catch(() => {}));

  // Video-Ende soll auch unsere Wiedergabe sauber beenden - sonst bleibt
  // player.Sync() im Leerlauf hängen (wartet ewig auf weitere Positionen,
  // die nach Videoende nicht mehr kommen).
  videoEl.addEventListener('ended', () => {
    if (playing && el('#pb-use-video-sync').checked) stop();
  });

  EventsOn('playback:log', log);
  EventsOn('playback:error', msg => log('FEHLER: ' + msg));
  EventsOn('playback:done', () => setPlayingState(false));
  EventsOn('playback:frame', f => {
    totalMs = Math.max(f.totalMs, 1);
    currentPosMs = f.atMs;
    el('#pb-progress').style.width = Math.min(100, (f.atMs / totalMs) * 100) + '%';
    el('#pb-vib').textContent = Math.round(f.vibration * 100) + '%';
    el('#pb-suc').textContent = Math.round(f.suction * 100) + '%';
    checkAutoExtendedO(f.atMs);
    redrawHeatmap();
    redrawCurve();
  });

  // Canvas-Breite ist eine feste Pixelangabe, keine CSS-Größe - bei einer
  // Fenstergrößenänderung muss sie neu gesetzt werden, sonst wird die Kurve
  // verzerrt skaliert dargestellt.
  window.addEventListener('resize', () => {
    if (curvePoints) {
      curveCanvas.width = curveCanvas.clientWidth || 800;
      redrawCurve();
    }
    if (heatmapPoints) {
      heatmapCanvas.width = heatmapCanvas.clientWidth || 800;
      redrawHeatmap();
    }
  });

  el('#pb-choose').addEventListener('click', chooseScript);
  el('#pb-play').addEventListener('click', play);
  el('#pb-stop').addEventListener('click', stop);
  el('#pb-eo-trigger').addEventListener('click', triggerEO);
  el('#pb-eo-enabled').addEventListener('change', () => {
    el('#pb-eo-trigger').disabled = !playing || !el('#pb-eo-enabled').checked;
  });

  // Tastenkürzel: Leertaste = Play/Stop, E = Extended-O. Greift nicht ein,
  // solange in einem Eingabefeld getippt wird.
  document.addEventListener('keydown', (e) => {
    const tag = (e.target.tagName || '').toLowerCase();
    if (tag === 'input' || tag === 'select' || tag === 'textarea') return;

    if (e.code === 'Space') {
      e.preventDefault();
      if (playing) stop(); else play();
    } else if (e.key === 'ArrowLeft' || e.key === 'ArrowRight') {
      // Springen, ohne die Maus zu benutzen. Der Player wird typischerweise
      // einhändig bedient - alles, was einen Mausweg spart, zählt hier mehr
      // als anderswo.
      e.preventDefault();
      const delta = (e.key === 'ArrowRight' ? 1 : -1) * (e.shiftKey ? 1 : 5);
      seekBy(delta * 1000);
    } else if (e.key === ',' || e.key === '.') {
      e.preventDefault();
      seekBy(e.key === '.' ? 200 : -200);
    } else if (e.key === '+' || e.key === '=' || e.key === '-') {
      // Offset live nachjustieren, während der markierte Abschnitt läuft.
      e.preventDefault();
      const step = e.key === '-' ? -10 : 10;
      applyOffset((Number(el('#pb-offset').value) || 0) + step);
    } else if (e.key === 'l') {
      el('#pb-loop').checked = !el('#pb-loop').checked;
      log(el('#pb-loop').checked ? 'Wiederholung an.' : 'Wiederholung aus.');
    } else if (e.key >= '1' && e.key <= '9') {
      if (totalMs > 0) seekTo(Math.round(totalMs * (Number(e.key) - 1) / 9));
    } else if (e.key.toLowerCase() === 'e') {
      if (!el('#pb-eo-trigger').disabled) triggerEO();
    }
  });

  // Gespeicherte Standardwerte übernehmen, sobald settings.js sie geladen hat.
  getSettingsCache().then(s => {
    el('#pb-mock').checked = s.playbackMock;
    el('#pb-sync').value = s.playbackSync;
    el('#pb-tick').value = s.playbackTickMs;
    el('#pb-maxspeed').value = s.playbackMaxSpeed;
    el('#pb-smoothing').value = s.playbackSmoothing;
    el('#pb-softstart').value = s.playbackSoftStartMs;
    el('#pb-eo-enabled').checked = s.playbackEOEnabled;
    el('#pb-eo-min').value = s.playbackEOMin;
    el('#pb-eo-hold').value = s.playbackEOHoldS;
    el('#pb-eo-restore').value = s.playbackEORestoreMs;
  });
  el('#pb-mock').addEventListener('change', e => saveSetting('playback.mock', e.target.checked));
  el('#pb-sync').addEventListener('change', e => saveSetting('playback.sync_mode', e.target.value));
  el('#pb-tick').addEventListener('change', e => saveSetting('playback.tick_ms', parseFloat(e.target.value)));
  el('#pb-maxspeed').addEventListener('change', e => saveSetting('playback.max_speed', parseFloat(e.target.value)));
  el('#pb-smoothing').addEventListener('change', e => saveSetting('playback.smoothing', parseFloat(e.target.value)));
  el('#pb-softstart').addEventListener('change', e => saveSetting('playback.soft_start_ms', parseFloat(e.target.value)));
  el('#pb-eo-enabled').addEventListener('change', e => saveSetting('playback.extended_o_enabled', e.target.checked));
  el('#pb-eo-min').addEventListener('change', e => saveSetting('playback.extended_o_min', parseFloat(e.target.value)));
  el('#pb-eo-hold').addEventListener('change', e => saveSetting('playback.extended_o_hold_seconds', parseFloat(e.target.value)));
  el('#pb-eo-restore').addEventListener('change', e => saveSetting('playback.extended_o_restore_ms', parseFloat(e.target.value)));

  return {
    // Von generator.js genutzt, um ein Ergebnis direkt zu übernehmen.
    loadScriptPath: (path) => loadScript(path).then(() => switchToPlaybackTab()),
  };
}

function switchToPlaybackTab() {
  document.querySelector('.tab-btn[data-tab="playback"]').click();
}
