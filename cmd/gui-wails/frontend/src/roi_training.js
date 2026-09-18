import {
  PickVideoFile, PickImageFile, LoadFirstFrame, LoadFrameAt,
  BootstrapRoiTrainingSampleEx, AddRoiStillTrainingSample, RunRoiModelTraining,
  ListRoiTrainingSamples, DiscardRoiTrainingSample, UpdateRoiTrainingSample, GetRoiDatasetSummary,
  GetRoiTrainingSampleImage, CheckRoiTrainingAvailable, CheckRoiTrainingStatus,
  InstallRoiTrainingDeps, ListRoiTrainingDevices,
} from '../wailsjs/go/main/App';
import { EventsOn } from '../wailsjs/runtime/runtime';
import { getSettingsCache, saveSetting } from './settings.js';
import { wireDataHelp } from './help.js';
import { uiError, uiInfo } from './notify.js';

const CLASS_PRESETS = [
  'hand', 'mouth', 'brust', 'eichel', 'penis', 'tongue', 'toy', 'body', 'face', 'other',
];
const MARK_COLORS = [
  { stroke: '#3dccc0', fill: 'rgba(61,204,192,0.16)' },
  { stroke: '#f2b03d', fill: 'rgba(242,176,61,0.18)' },
  { stroke: '#5ecf8a', fill: 'rgba(94,207,138,0.16)' },
  { stroke: '#e8eaed', fill: 'rgba(232,234,237,0.12)' },
];

export function initRoiTraining(root) {
  root.innerHTML = `
    <h2>KI-Trainingssystem</h2>
    <p class="hint">
      Markierungen auf Video oder Still-Bild → abtasten → kontrollieren → lokal trainieren.
      Bis zu vier Klassen pro Bild. Sound wird neben dem Datensatz abgelegt.
    </p>
    <div class="card" style="margin-bottom:16px; padding:12px 14px;">
      <h3 style="margin-top:0; margin-bottom:8px;">So wird daraus ein gutes Skript</h3>
      <ol class="hint" style="margin:0; padding-left:1.2em; line-height:1.55;">
        <li><b>Hier trainieren</b> — Markierungen → „Für Training verwenden“ → schlechte Samples verwerfen → Training starten. Ergebnis: <code>roi_detector.onnx</code>.</li>
        <li><b>Dann Erzeugen</b> — Video laden → Häkchen „KI-Erkennung (ONNX)“ → „Region automatisch finden“. Die KI schlägt nur die Box vor.</li>
        <li><b>Box prüfen/korrigieren</b> — nie blind übernehmen. Bei Tf/Tj ggf. 2. Region setzen.</li>
        <li><b>Funscript generieren</b> — klassisches Tracking (CSRT/Flow/…) schreibt das Skript. Die KI trackt nicht selbst.</li>
        <li><b>In Wiedergabe prüfen</b> — Feedback-Buttons (brauchbar/…) verbessern später den Quality Doctor, nicht die Regions-KI.</li>
      </ol>
    </div>

    <h3>1. Trainingsdaten sammeln</h3>
    <div class="row">
      <button id="rt-pick-video" type="button">Video wählen…</button>
      <button id="rt-pick-image" type="button"
        data-help="Einzelbild ohne Tracking — gut, um Klassen zu erklären; später auf Clips übertragen.">Bild wählen…</button>
      <span class="path-label" id="rt-video-path">Keine Quelle gewählt</span>
    </div>
    <div class="row" style="align-items:center; margin:6px 0;">
      <label style="width:auto;" data-help="Vorsprung bei schwarzem Intro — Frame zum Markieren wählen.">Zeit (s)</label>
      <input type="number" id="rt-seek" value="0" min="0" step="0.5" style="width:5em;" />
      <button id="rt-seek-btn" type="button" disabled>Frame laden</button>
      <button id="rt-seek-plus" type="button" disabled title="+1 s">+1s</button>
      <button id="rt-seek-plus5" type="button" disabled title="+5 s">+5s</button>
    </div>
    <div id="rt-canvas-wrap">
      <canvas id="rt-canvas"></canvas>
    </div>
    <div class="row" style="align-items:center; flex-wrap:wrap;">
      <button id="rt-mark-next" type="button"
        data-help="Nächste Markierung starten (bis zu 4 Boxen im selben Bild).">+ Markierung</button>
      <button id="rt-clear-marks" type="button">Alle löschen</button>
      <span class="hint" style="margin:0">Aktiv: <span id="rt-active-mark">1</span>/4 — Shift+Ziehen = nächste Klasse</span>
    </div>
    <div id="rt-mark-fields"></div>
    <datalist id="rt-class-list"></datalist>
    <div id="rt-class-chips" class="rt-chips" aria-label="Bekannte Klassen"></div>
    <div class="field-row"><label data-help="Write every N-th frame as a sample. Smaller = denser; larger = less redundancy.">Sampling (every N-th frame)</label>
      <input type="number" id="rt-sample-every" value="12" min="1" max="120" style="width:5em;" />
    </div>
    <div class="field-row"><label data-help="Multiply marked box size for YOLO labels (1.0 = exact mark; 1.1–1.2 adds a small pad). Prefer correcting boxes in review over a large scale.">Box scale</label>
      <input type="number" id="rt-box-scale" value="1.0" min="0.5" max="2.0" step="0.05" style="width:5em;" />
    </div>
    <div class="checkbox-row"><input type="checkbox" id="rt-extract-audio" checked />
      <label for="rt-extract-audio" style="width:auto"
        data-help="Speichert die Tonspur als WAV unter audio/ im Datensatz — für späteres Lernen, was was ist.">Sound mit speichern</label></div>
    <div class="row">
      <button id="rt-bootstrap" class="primary" disabled
        data-help="Video: trackt Markierungen und schreibt YOLO-Beispiele. Bild: speichert eine Still-Annotation.">Für Training verwenden</button>
    </div>
    <div class="path-label" id="rt-bootstrap-status"></div>
    <pre id="rt-bootstrap-log" class="hint" style="max-height:120px; overflow:auto; white-space:pre-wrap; margin:0 0 10px;"></pre>

    <h3>2. Kontrollansicht</h3>
    <div class="field-row"><label data-help="Ordner mit images/ und labels/ aus dem Bootstrap.">Datensatzordner</label>
      <input type="text" id="rt-dataset-dir" style="flex:1" />
    </div>
    <div class="row" style="align-items:center;">
      <button id="rt-refresh-review" type="button">Aktualisieren</button>
      <label class="hint" style="margin:0"
        data-help="Zeigt nur Beispiele des letzten Laufs."><input type="checkbox" id="rt-only-new" checked /> nur zuletzt hinzugekommene</label>
    </div>
    <div id="rt-review-grid" class="hint">Noch keine Beispiele geladen.</div>

    <h3>3. Datensatz-Übersicht</h3>
    <div class="row"><button id="rt-summary-refresh" type="button">Übersicht laden</button></div>
    <div id="rt-summary" class="hint"></div>

    <h3>4. Modell trainieren</h3>
    <p class="hint">Lokal. Gerät: Auto wählt CUDA → MPS → DirectML → CPU.</p>
    <div class="field-row"><label data-help="Trainingsdurchläufe. 50–100 üblich für kleine Sets.">Epochen</label><input type="number" id="rt-epochs" value="100" min="1" /></div>
    <div class="field-row"><label data-help="auto = bestes Backend.">Gerät</label>
      <select id="rt-device">
        <option value="auto" selected>Automatisch</option>
        <option value="cuda">NVIDIA CUDA</option>
        <option value="directml">DirectML (Windows)</option>
        <option value="mps">Apple MPS</option>
        <option value="cpu">CPU (sehr langsam)</option>
      </select>
    </div>
    <p class="hint" id="rt-device-status" style="margin:0 0 8px;"></p>
    <div class="row"><button id="rt-train" class="primary" type="button" disabled>Training starten</button>
      <button id="rt-install-deps" type="button"
        data-help="Installiert ultralytics + onnx per pip in das erkannte Python (auch ohne Quellbaum).">Abhängigkeiten installieren</button></div>
    <p class="hint" id="rt-train-unavailable" style="display:none; color:var(--danger);">
      Training-Abhängigkeiten fehlen. Knopf „Abhängigkeiten installieren“ nutzen
      (oder manuell: <code>pip install ultralytics onnx</code>). Details:
      <a href="#" id="rt-docs-link">docs/KI_TRAINING.md</a>
    </p>
    <p class="hint" id="rt-status-detail" style="margin:0 0 8px;"></p>
    <div class="path-label" id="rt-train-status"></div>
    <pre id="rt-train-log" class="hint" style="max-height:240px; overflow:auto; white-space:pre-wrap;"></pre>
  `;

  wireDataHelp(root);

  const el = sel => root.querySelector(sel);
  const canvas = el('#rt-canvas');
  const ctx = canvas.getContext('2d');

  let sourcePath = null;
  let sourceKind = null; // 'video' | 'image'
  let img = new Image();
  let nativeW = 0, nativeH = 0;
  let marks = [null, null, null, null]; // up to 4
  let activeMark = 0;
  let dragging = false, startX = 0, startY = 0, curX = 0, curY = 0;
  let lastPrefix = '';
  let datasetDir = '';
  let seekSec = 0;

  const DISPLAY_W = 560;

  function renderMarkFields() {
    const prev = [];
    for (let i = 0; i < 4; i++) {
      prev[i] = el(`#rt-class${i + 1}`)?.value || '';
    }
    const wrap = el('#rt-mark-fields');
    wrap.innerHTML = '';
    for (let i = 0; i < 4; i++) {
      const row = document.createElement('div');
      row.className = 'field-row';
      row.style.opacity = (i === 0 || marks[i] || i === activeMark) ? '1' : '0.55';
      const lab = document.createElement('label');
      lab.textContent = `Klasse ${i + 1}`;
      lab.style.borderLeft = `3px solid ${MARK_COLORS[i].stroke}`;
      lab.style.paddingLeft = '6px';
      const input = document.createElement('input');
      input.type = 'text';
      input.id = `rt-class${i + 1}`;
      input.setAttribute('list', 'rt-class-list');
      input.placeholder = CLASS_PRESETS[i] || 'z.B. hand';
      if (prev[i]) input.value = prev[i];
      input.addEventListener('input', updateBootstrapEnabled);
      const meta = document.createElement('span');
      meta.className = 'hint';
      meta.id = `rt-mark-label${i + 1}`;
      meta.style.margin = '0 0 0 8px';
      meta.textContent = marks[i]
        ? `${marks[i].w}×${marks[i].h}`
        : (i === activeMark ? 'ziehen…' : '—');
      row.appendChild(lab);
      row.appendChild(input);
      row.appendChild(meta);
      wrap.appendChild(row);
    }
    el('#rt-active-mark').textContent = String(activeMark + 1);
    updateBootstrapEnabled();
  }

  function updateBootstrapEnabled() {
    const class1 = el('#rt-class1')?.value.trim();
    let ok = !!(sourcePath && marks[0] && class1);
    for (let i = 1; i < 4; i++) {
      if (marks[i] && !(el(`#rt-class${i + 1}`)?.value.trim())) ok = false;
    }
    el('#rt-bootstrap').disabled = !ok;
  }

  function drawNativeRect(r, stroke, fill) {
    if (!r || !nativeW || !nativeH) return;
    const scaleX = canvas.width / nativeW, scaleY = canvas.height / nativeH;
    ctx.strokeStyle = stroke;
    ctx.lineWidth = 2;
    ctx.strokeRect(r.x * scaleX, r.y * scaleY, r.w * scaleX, r.h * scaleY);
    ctx.fillStyle = fill;
    ctx.fillRect(r.x * scaleX, r.y * scaleY, r.w * scaleX, r.h * scaleY);
  }

  function redraw() {
    ctx.clearRect(0, 0, canvas.width, canvas.height);
    if (img.src) ctx.drawImage(img, 0, 0, canvas.width, canvas.height);
    for (let i = 0; i < 4; i++) {
      if (marks[i] && !(dragging && i === activeMark)) {
        drawNativeRect(marks[i], MARK_COLORS[i].stroke, MARK_COLORS[i].fill);
      }
    }
    if (dragging) {
      const c = MARK_COLORS[activeMark];
      const dx0 = Math.min(startX, curX), dy0 = Math.min(startY, curY);
      const dw = Math.abs(curX - startX), dh = Math.abs(curY - startY);
      ctx.strokeStyle = c.stroke;
      ctx.lineWidth = 2;
      ctx.strokeRect(dx0, dy0, dw, dh);
      ctx.fillStyle = c.fill;
      ctx.fillRect(dx0, dy0, dw, dh);
    }
  }

  canvas.addEventListener('mousedown', e => {
    const r = canvas.getBoundingClientRect();
    startX = curX = e.clientX - r.left;
    startY = curY = e.clientY - r.top;
    if (e.shiftKey) {
      for (let i = 0; i < 4; i++) {
        if (!marks[i]) { activeMark = i; break; }
      }
    }
    dragging = true;
    renderMarkFields();
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
    marks[activeMark] = {
      x: Math.round(x0 * scaleX), y: Math.round(y0 * scaleY),
      w: Math.round(w * scaleX), h: Math.round(h * scaleY),
    };
    if (activeMark < 3) activeMark += 1;
    renderMarkFields();
    updateBootstrapEnabled();
    redraw();
  });

  el('#rt-mark-next').addEventListener('click', () => {
    for (let i = 0; i < 4; i++) {
      if (!marks[i]) { activeMark = i; break; }
    }
    renderMarkFields();
  });
  el('#rt-clear-marks').addEventListener('click', () => {
    marks = [null, null, null, null];
    activeMark = 0;
    renderMarkFields();
    updateBootstrapEnabled();
    redraw();
  });

  async function showPreview(path, sec) {
    const preview = sec > 0 ? await LoadFrameAt(path, sec) : await LoadFirstFrame(path);
    nativeW = preview.width; nativeH = preview.height;
    const displayH = Math.round(DISPLAY_W * nativeH / nativeW);
    canvas.width = DISPLAY_W; canvas.height = displayH;
    img.onload = redraw;
    img.src = 'data:image/png;base64,' + preview.pngBase64;
  }

  async function chooseVideo() {
    const path = await PickVideoFile();
    if (document.activeElement?.blur) document.activeElement.blur();
    window.focus();
    if (!path) return;
    sourcePath = path;
    sourceKind = 'video';
    seekSec = 0;
    el('#rt-seek').value = '0';
    el('#rt-video-path').textContent = path.split(/[\\/]/).pop();
    marks = [null, null, null, null];
    activeMark = 0;
    renderMarkFields();
    updateBootstrapEnabled();
    el('#rt-seek-btn').disabled = false;
    el('#rt-seek-plus').disabled = false;
    el('#rt-seek-plus5').disabled = false;
    el('#rt-bootstrap-status').textContent = 'Lade Frame…';
    try {
      await showPreview(path, 0);
      el('#rt-bootstrap-status').textContent = 'Region(en) markieren. Bei schwarzem Anfang Zeit vorstellen.';
    } catch (err) {
      uiError('Video laden: ' + err, el('#rt-bootstrap-status'));
    }
  }

  async function chooseImage() {
    const path = await PickImageFile();
    if (document.activeElement?.blur) document.activeElement.blur();
    window.focus();
    if (!path) return;
    sourcePath = path;
    sourceKind = 'image';
    el('#rt-video-path').textContent = path.split(/[\\/]/).pop() + ' (Bild)';
    marks = [null, null, null, null];
    activeMark = 0;
    renderMarkFields();
    updateBootstrapEnabled();
    el('#rt-seek-btn').disabled = true;
    el('#rt-seek-plus').disabled = true;
    el('#rt-seek-plus5').disabled = true;
    el('#rt-bootstrap-status').textContent = 'Lade Bild…';
    try {
      // Still: LoadFrameAt won't work; read via temporary — use same PNG path through a tiny hack:
      // Go AddStill uses DecodeConfig; for preview we open as data URL via fetch isn't available.
      // Use LoadFirstFrame on non-video fails — instead draw from filesystem via a Go helper.
      // Fallback: ask user to mark after we get dimensions from a video-less path.
      // We reuse LoadFrameAt only for video; for images, create an Image from file:// is blocked.
      // So call a zero-second dump only works for video. For images, use img from path via Go:
      const preview = await GetRoiTrainingSampleImage(path);
      img.onload = () => {
        nativeW = img.naturalWidth || img.width;
        nativeH = img.naturalHeight || img.height;
        const displayH = Math.round(DISPLAY_W * nativeH / Math.max(nativeW, 1));
        canvas.width = DISPLAY_W; canvas.height = displayH;
        redraw();
      };
      const ext = path.toLowerCase();
      const mime = ext.endsWith('.png') ? 'image/png'
        : ext.endsWith('.webp') ? 'image/webp'
        : ext.endsWith('.gif') ? 'image/gif'
        : 'image/jpeg';
      img.src = `data:${mime};base64,` + preview;
      el('#rt-bootstrap-status').textContent = 'Still-Bild: Regionen markieren, dann speichern.';
    } catch (err) {
      uiError('Bild laden: ' + err, el('#rt-bootstrap-status'));
    }
  }

  async function seekTo(sec) {
    if (!sourcePath || sourceKind !== 'video') return;
    seekSec = Math.max(0, sec);
    el('#rt-seek').value = String(seekSec);
    el('#rt-bootstrap-status').textContent = `Lade Frame bei ${seekSec}s…`;
    try {
      await showPreview(sourcePath, seekSec);
      el('#rt-bootstrap-status').textContent = `Frame bei ${seekSec}s — markieren.`;
    } catch (err) {
      uiError('Seek fehlgeschlagen: ' + err, el('#rt-bootstrap-status'));
    }
  }

  el('#rt-pick-video').addEventListener('click', chooseVideo);
  el('#rt-pick-image').addEventListener('click', chooseImage);
  el('#rt-seek-btn').addEventListener('click', () => seekTo(parseFloat(el('#rt-seek').value) || 0));
  el('#rt-seek-plus').addEventListener('click', () => seekTo(seekSec + 1));
  el('#rt-seek-plus5').addEventListener('click', () => seekTo(seekSec + 5));

  function roiArg(m) {
    return m ? { X: m.x, Y: m.y, W: m.w, H: m.h } : null;
  }

  el('#rt-bootstrap').addEventListener('click', async () => {
    if (!sourcePath || !marks[0]) return;
    const classes = [1, 2, 3, 4].map(i => el(`#rt-class${i}`)?.value.trim() || '');
    if (!classes[0]) return;
    el('#rt-bootstrap').disabled = true;
    el('#rt-bootstrap-status').textContent = 'Läuft…';
    el('#rt-bootstrap-log').textContent = '';
    try {
      if (sourceKind === 'image') {
        const regions = [];
        for (let i = 0; i < 4; i++) {
          if (marks[i] && classes[i]) {
            regions.push({ ROI: roiArg(marks[i]), ClassName: classes[i] });
          }
        }
        lastPrefix = await AddRoiStillTrainingSample(sourcePath, regions);
        el('#rt-bootstrap-status').textContent = 'Still-Beispiel gespeichert.';
        updateBootstrapEnabled();
        refreshReview();
        refreshClassList();
      } else {
        const sampleEvery = parseInt(el('#rt-sample-every').value, 10) || 12;
        const extractAudio = el('#rt-extract-audio').checked;
        const boxScale = parseFloat(el('#rt-box-scale')?.value) || 1.0;
        lastPrefix = await BootstrapRoiTrainingSampleEx(
          sourcePath,
          roiArg(marks[0]), roiArg(marks[1]), roiArg(marks[2]), roiArg(marks[3]),
          classes[0], classes[1], classes[2], classes[3],
          sampleEvery, extractAudio, seekSec > 0 ? seekSec : 0, boxScale,
        );
      }
    } catch (err) {
      uiError('Sample speichern: ' + err, el('#rt-bootstrap-status'));
      updateBootstrapEnabled();
    }
  });

  EventsOn('roitraining:bootstrap:progress', line => {
    const log = el('#rt-bootstrap-log');
    log.textContent += line + '\n';
    log.scrollTop = log.scrollHeight;
  });
  EventsOn('roitraining:bootstrap:done', payload => {
    updateBootstrapEnabled();
    if (payload.error) {
      uiError('Bootstrap fehlgeschlagen: ' + payload.error, el('#rt-bootstrap-status'));
      return;
    }
    lastPrefix = payload.prefix || lastPrefix;
    el('#rt-bootstrap-status').textContent = 'Fertig — Beispiele in der Kontrollansicht.';
    refreshReview();
    refreshClassList();
  });

  function boxOverlay(box) {
    const wrap = document.createElement('div');
    wrap.className = 'rt-box';
    wrap.style.left = ((box.xc - box.w / 2) * 100) + '%';
    wrap.style.top = ((box.yc - box.h / 2) * 100) + '%';
    wrap.style.width = (box.w * 100) + '%';
    wrap.style.height = (box.h * 100) + '%';
    const label = document.createElement('span');
    label.className = 'rt-box-label';
    label.textContent = box.className || ('#' + box.classId);
    wrap.appendChild(label);
    return wrap;
  }

  async function refreshReview() {
    datasetDir = el('#rt-dataset-dir').value.trim();
    const onlyNew = el('#rt-only-new').checked;
    const prefix = onlyNew ? lastPrefix : '';
    const grid = el('#rt-review-grid');
    grid.textContent = 'Lädt…';
    try {
      const samples = await ListRoiTrainingSamples(datasetDir, prefix);
      if (!samples || samples.length === 0) {
        grid.textContent = onlyNew && lastPrefix
          ? 'Keine neuen Beispiele — Häkchen abwählen für den ganzen Datensatz.'
          : 'Keine Beispiele gefunden.';
        return;
      }
      grid.innerHTML = '';
      const gridEl = document.createElement('div');
      gridEl.className = 'rt-grid';
      grid.appendChild(gridEl);
      for (const s of samples) {
        const card = document.createElement('div');
        card.className = 'rt-card';
        const thumbWrap = document.createElement('div');
        thumbWrap.className = 'rt-thumb-wrap';
        const imgEl = document.createElement('img');
        imgEl.className = 'rt-thumb';
        thumbWrap.appendChild(imgEl);
        const caption = document.createElement('div');
        caption.className = 'hint';
        caption.style.margin = '4px 0';
        caption.textContent = `${s.split} / ${s.name}`;
        const discardBtn = document.createElement('button');
        discardBtn.type = 'button';
        discardBtn.className = 'danger';
        discardBtn.textContent = 'Verwerfen';
        const editBtn = document.createElement('button');
        editBtn.type = 'button';
        editBtn.textContent = 'Box korrigieren';
        editBtn.title = 'Box neu auf dem Vorschaubild ziehen und speichern';
        const btnRow = document.createElement('div');
        btnRow.className = 'row';
        btnRow.style.gap = '6px';
        btnRow.appendChild(editBtn);
        btnRow.appendChild(discardBtn);
        card.appendChild(thumbWrap);
        card.appendChild(caption);
        card.appendChild(btnRow);
        gridEl.appendChild(card);

        GetRoiTrainingSampleImage(s.imagePath).then(b64 => {
          const p = String(s.imagePath || '').toLowerCase();
          const mime = p.endsWith('.png') ? 'image/png'
            : p.endsWith('.webp') ? 'image/webp'
            : p.endsWith('.gif') ? 'image/gif'
            : 'image/jpeg';
          imgEl.src = `data:${mime};base64,` + b64;
          (s.boxes || []).forEach(box => thumbWrap.appendChild(boxOverlay(box)));
        }).catch(() => { caption.textContent += ' (Bild konnte nicht geladen werden)'; });

        editBtn.addEventListener('click', async () => {
          try {
            await editSampleBoxes(s, imgEl, thumbWrap);
          } catch (err) {
            uiError('Box korrigieren: ' + err);
          }
        });

        discardBtn.addEventListener('click', async () => {
          discardBtn.disabled = true;
          try {
            await DiscardRoiTrainingSample(datasetDir, s.split, s.name);
            card.remove();
          } catch (err) {
            uiError('Sample verwerfen: ' + err);
            discardBtn.disabled = false;
          }
        });
      }
    } catch (err) {
      grid.textContent = 'Fehler: ' + err;
    }
  }

  // Simple box re-draw: user drags on the thumbnail; first box is replaced.
  async function editSampleBoxes(sample, imgEl, thumbWrap) {
    const boxes = Array.isArray(sample.boxes) ? sample.boxes.slice() : [];
    if (!imgEl.naturalWidth) {
      await new Promise((resolve, reject) => {
        imgEl.onload = resolve;
        imgEl.onerror = reject;
        if (imgEl.complete && imgEl.naturalWidth) resolve();
      });
    }
    uiInfo('Auf dem Vorschaubild ziehen: neue Box für die erste Klasse. Escape bricht ab.');
    const rect = () => thumbWrap.getBoundingClientRect();
    let dragging = false, x0 = 0, y0 = 0, x1 = 0, y1 = 0;
    const overlay = document.createElement('div');
    overlay.className = 'rt-box';
    overlay.style.borderColor = 'var(--accent)';
    const onMove = (e) => {
      if (!dragging) return;
      const r = rect();
      x1 = Math.min(1, Math.max(0, (e.clientX - r.left) / r.width));
      y1 = Math.min(1, Math.max(0, (e.clientY - r.top) / r.height));
      const L = Math.min(x0, x1), T = Math.min(y0, y1);
      const W = Math.abs(x1 - x0), H = Math.abs(y1 - y0);
      overlay.style.left = (L * 100) + '%';
      overlay.style.top = (T * 100) + '%';
      overlay.style.width = (W * 100) + '%';
      overlay.style.height = (H * 100) + '%';
    };
    const cleanup = () => {
      thumbWrap.removeEventListener('mousemove', onMove);
      window.removeEventListener('mouseup', onUp);
      window.removeEventListener('keydown', onKey);
    };
    const onKey = (e) => {
      if (e.key === 'Escape') {
        cleanup();
        overlay.remove();
        uiInfo('Korrektur abgebrochen.');
      }
    };
    const onUp = async () => {
      if (!dragging) return;
      dragging = false;
      cleanup();
      const L = Math.min(x0, x1), T = Math.min(y0, y1);
      const W = Math.abs(x1 - x0), H = Math.abs(y1 - y0);
      if (W < 0.02 || H < 0.02) {
        overlay.remove();
        uiInfo('Box zu klein — erneut versuchen.');
        return;
      }
      const first = boxes[0] || { classId: 0, className: 'object' };
      const updated = [{
        classId: first.classId,
        className: first.className,
        xc: L + W / 2,
        yc: T + H / 2,
        w: W,
        h: H,
      }, ...boxes.slice(1)];
      await UpdateRoiTrainingSample(datasetDir, sample.split, sample.name, updated);
      overlay.remove();
      thumbWrap.querySelectorAll('.rt-box').forEach(n => n.remove());
      updated.forEach(box => thumbWrap.appendChild(boxOverlay(box)));
      uiInfo('Box gespeichert.');
    };
    thumbWrap.addEventListener('mousedown', (e) => {
      const r = rect();
      dragging = true;
      x0 = x1 = Math.min(1, Math.max(0, (e.clientX - r.left) / r.width));
      y0 = y1 = Math.min(1, Math.max(0, (e.clientY - r.top) / r.height));
      thumbWrap.appendChild(overlay);
      thumbWrap.addEventListener('mousemove', onMove);
      window.addEventListener('mouseup', onUp);
      window.addEventListener('keydown', onKey);
    }, { once: true });
  }

  el('#rt-refresh-review').addEventListener('click', refreshReview);
  el('#rt-dataset-dir').addEventListener('change', e => {
    saveSetting('generator.roiTrainingDatasetDir', e.target.value.trim());
  });

  async function refreshClassList() {
    let fromData = [];
    try {
      const summary = await GetRoiDatasetSummary(datasetDir || el('#rt-dataset-dir').value.trim());
      fromData = (summary.classes || []).map(c => c.className);
      const list = el('#rt-class-list');
      const names = [...new Set([...fromData, ...CLASS_PRESETS])];
      list.innerHTML = names.map(n => `<option value="${n}"></option>`).join('');
      renderClassChips(fromData, names);
      return summary;
    } catch {
      el('#rt-class-list').innerHTML = CLASS_PRESETS.map(n => `<option value="${n}"></option>`).join('');
      renderClassChips([], CLASS_PRESETS);
      return null;
    }
  }

  function renderClassChips(fromData, allNames) {
    const wrap = el('#rt-class-chips');
    if (!wrap) return;
    wrap.innerHTML = '';
    const known = fromData.length ? fromData : allNames.slice(0, 8);
    if (!known.length) {
      wrap.innerHTML = '<span class="hint">Noch keine Klassen — Presets unten tippen oder markieren.</span>';
      return;
    }
    const label = document.createElement('span');
    label.className = 'hint';
    label.style.marginRight = '6px';
    label.textContent = fromData.length ? 'Aus Datensatz:' : 'Vorschläge:';
    wrap.appendChild(label);
    known.forEach(name => {
      const btn = document.createElement('button');
      btn.type = 'button';
      btn.className = 'rt-chip';
      btn.textContent = name;
      btn.title = 'In aktive Klasse einsetzen';
      btn.addEventListener('click', () => {
        const input = el(`#rt-class${activeMark + 1}`);
        if (input) {
          input.value = name;
          updateBootstrapEnabled();
        }
      });
      wrap.appendChild(btn);
    });
  }

  el('#rt-summary-refresh').addEventListener('click', async () => {
    datasetDir = el('#rt-dataset-dir').value.trim();
    const box = el('#rt-summary');
    box.textContent = 'Lädt…';
    try {
      const summary = await GetRoiDatasetSummary(datasetDir);
      if (!summary.classes || summary.classes.length === 0) {
        box.textContent = 'Noch keine Klassen — erst Daten sammeln.';
        return;
      }
      box.innerHTML = '<table style="border-collapse:collapse; width:100%;">'
        + '<thead><tr><th style="text-align:left; padding:4px 8px;">Klasse</th>'
        + '<th style="text-align:right; padding:4px 8px;">Train</th>'
        + '<th style="text-align:right; padding:4px 8px;">Val</th></tr></thead><tbody>'
        + summary.classes.map(c => `<tr><td style="padding:4px 8px;">${c.className}</td>`
          + `<td style="text-align:right; padding:4px 8px;">${c.trainCount}</td>`
          + `<td style="text-align:right; padding:4px 8px;">${c.valCount}</td></tr>`).join('')
        + '</tbody></table>';
      refreshClassList();
    } catch (err) {
      box.textContent = 'Fehler: ' + err;
    }
  });

  el('#rt-train').addEventListener('click', async () => {
    const epochs = parseInt(el('#rt-epochs').value, 10) || 100;
    const device = el('#rt-device').value;
    el('#rt-train').disabled = true;
    el('#rt-train-status').textContent = 'Läuft…';
    el('#rt-train-log').textContent = '';
    try {
      await RunRoiModelTraining(epochs, device);
    } catch (err) {
      uiError('ROI-Training: ' + err, el('#rt-train-status'));
      el('#rt-train').disabled = false;
    }
  });
  EventsOn('roitraining:train:progress', line => {
    const log = el('#rt-train-log');
    log.textContent += line + '\n';
    log.scrollTop = log.scrollHeight;
  });
  EventsOn('roitraining:train:done', payload => {
    el('#rt-train').disabled = false;
    if (payload.error) {
      uiError('Training fehlgeschlagen: ' + payload.error, el('#rt-train-status'));
      return;
    }
    el('#rt-train-status').textContent = 'Fertig: ' + payload.modelPath
      + ' — KI-Erkennung im Erzeugen-Tab aktualisiert sich automatisch.';
    window.dispatchEvent(new CustomEvent('samn-ai-roi-refresh'));
  });

  getSettingsCache().then(s => {
    datasetDir = s.roiDatasetDir || s.defaultRoiDatasetDir || '';
    el('#rt-dataset-dir').value = datasetDir;
    refreshClassList();
  });

  function applyTrainAvailability(available, detail) {
    el('#rt-train').disabled = !available;
    el('#rt-train-unavailable').style.display = available ? 'none' : 'block';
    if (el('#rt-status-detail') && detail) {
      el('#rt-status-detail').textContent = detail;
    }
  }

  CheckRoiTrainingStatus().then(st => {
    applyTrainAvailability(!!st.ultralytics, st.detail || '');
  }).catch(() => {
    CheckRoiTrainingAvailable().then(available => {
      applyTrainAvailability(available, '');
    }).catch(() => applyTrainAvailability(false, ''));
  });

  el('#rt-install-deps')?.addEventListener('click', async () => {
    el('#rt-install-deps').disabled = true;
    el('#rt-train-status').textContent = 'Installiere Abhängigkeiten…';
    el('#rt-train-log').textContent = '';
    try {
      await InstallRoiTrainingDeps();
    } catch (err) {
      uiError('Installation: ' + err, el('#rt-train-status'));
      el('#rt-install-deps').disabled = false;
    }
  });
  EventsOn('roitraining:deps:progress', line => {
    const log = el('#rt-train-log');
    log.textContent += line + '\n';
    log.scrollTop = log.scrollHeight;
  });
  EventsOn('roitraining:deps:done', payload => {
    el('#rt-install-deps').disabled = false;
    if (payload.error) {
      uiError('Installation fehlgeschlagen: ' + payload.error, el('#rt-train-status'));
      return;
    }
    const st = payload.status || {};
    applyTrainAvailability(!!st.ultralytics, st.detail || 'Abhängigkeiten installiert');
    el('#rt-train-status').textContent = 'Abhängigkeiten OK — Training starten möglich.';
    uiInfo('KI-Trainingsabhängigkeiten installiert.');
  });

  ListRoiTrainingDevices().then(devices => {
    if (!Array.isArray(devices) || !devices.length) return;
    const sel = el('#rt-device');
    const current = sel.value || 'auto';
    sel.innerHTML = devices.map(d => {
      const mark = d.available ? '' : ' — nicht erkannt';
      return `<option value="${d.id}">${d.label}${mark}</option>`;
    }).join('');
    if ([...sel.options].some(o => o.value === current)) sel.value = current;
    else sel.value = 'auto';
    const avail = devices.filter(d => d.available && d.id !== 'auto').map(d => d.id);
    el('#rt-device-status').textContent = avail.length
      ? ('Erkannt: ' + avail.join(', '))
      : 'Kein GPU-Backend erkannt — Auto fällt auf CPU zurück.';
  }).catch(() => {
    el('#rt-device-status').textContent = '';
  });

  renderMarkFields();
  el('#rt-class-list').innerHTML = CLASS_PRESETS.map(n => `<option value="${n}"></option>`).join('');
}
