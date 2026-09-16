import {
  PickVideoFile, LoadFirstFrame, BootstrapRoiTrainingSample, RunRoiModelTraining,
  ListRoiTrainingSamples, DiscardRoiTrainingSample, GetRoiDatasetSummary,
  GetRoiTrainingSampleImage, CheckRoiTrainingAvailable,
} from '../wailsjs/go/main/App';
import { EventsOn } from '../wailsjs/runtime/runtime';
import { getSettingsCache, saveSetting } from './settings.js';

// KI-Trainingssystem: sammelt Trainingsdaten für ein YOLO-Regionserkennungs-
// modell direkt aus Videos, die durch die App laufen (statt nur über die
// eigenständigen CLI-Skripte generator/bootstrap_yolo_dataset.py und
// train_yolo_model.py) - siehe app_roi_training.go für den Vertrag.
//
// bootstrap_yolo_dataset.py ist explizit dokumentiert: der Detektor lernt,
// die klassische Tracking-Box nachzuahmen, wird also nie zuverlässiger als
// der Tracker, der die Trainingsdaten geliefert hat. Deshalb ist die
// Kontrollansicht (Abschnitt 2) kein optionales Extra, sondern der Schritt,
// der das eigentlich erst brauchbar macht: falsch getrackte Beispiele lassen
// sich vor dem Training verwerfen, statt unkontrolliert einzufließen.
//
// Box-Korrektur von Hand (statt nur Verwerfen) ist bewusst nicht Teil dieser
// ersten Version - Verwerfen deckt den wichtigsten Fall (Tracker hat
// komplett danebengelegen) ab, ohne eine eigene Zeichen-UI für die
// Kontrollansicht zu brauchen.
export function initRoiTraining(root) {
  root.innerHTML = `
    <h2>KI-Trainingssystem</h2>
    <p class="hint">
      Sammelt Trainingsbeispiele für ein eigenes YOLO-Regionserkennungsmodell aus Videos,
      die hier durchlaufen - klassisches Tracking markiert die Region automatisch, die
      Kontrollansicht darunter lässt falsch getrackte Beispiele vor dem Training aussortieren.
      Das bestehende manuelle Markieren im Generator-Tab bleibt davon unberührt und weiterhin
      der zuverlässige Weg.
    </p>

    <h3>1. Trainingsdaten sammeln</h3>
    <div class="row">
      <button id="rt-pick-video" type="button">Video wählen…</button>
      <span class="path-label" id="rt-video-path">Kein Video gewählt</span>
    </div>
    <div id="rt-canvas-wrap">
      <canvas id="rt-canvas"></canvas>
    </div>
    <div class="row" style="align-items:center;">
      <button id="rt-roi2-toggle" type="button">2. Region</button>
      <span class="hint" style="margin:0">Shift+Ziehen oder Knopf: zweite Region (violett), optional.</span>
    </div>
    <div class="field-row"><label>Klasse (Region 1)</label>
      <input type="text" id="rt-class1" list="rt-class-list" placeholder="z.B. brust" />
    </div>
    <div class="field-row"><label>Klasse (Region 2)</label>
      <input type="text" id="rt-class2" list="rt-class-list" placeholder="z.B. hand" disabled />
    </div>
    <datalist id="rt-class-list"></datalist>
    <div class="path-label" id="rt-roi-label">Keine Region markiert</div>
    <div class="path-label" id="rt-roi2-label">Keine 2. Region markiert</div>
    <div class="row"><button id="rt-bootstrap" class="primary" disabled>Für Training verwenden</button></div>
    <div class="path-label" id="rt-bootstrap-status"></div>
    <pre id="rt-bootstrap-log" class="hint" style="max-height:120px; overflow:auto; white-space:pre-wrap; margin:0 0 10px;"></pre>

    <h3>2. Kontrollansicht</h3>
    <div class="field-row"><label>Datensatzordner</label>
      <input type="text" id="rt-dataset-dir" style="flex:1" />
    </div>
    <div class="row" style="align-items:center;">
      <button id="rt-refresh-review" type="button">Aktualisieren</button>
      <label class="hint" style="margin:0"><input type="checkbox" id="rt-only-new" checked /> nur zuletzt hinzugekommene</label>
    </div>
    <div id="rt-review-grid" class="hint">Noch keine Beispiele geladen.</div>

    <h3>3. Datensatz-Übersicht</h3>
    <div class="row"><button id="rt-summary-refresh" type="button">Übersicht laden</button></div>
    <div id="rt-summary" class="hint"></div>

    <h3>4. Modell trainieren</h3>
    <p class="hint">Läuft lokal auf dieser Maschine (braucht eine CUDA-fähige GPU für
      brauchbare Trainingszeiten - auf der CPU dauert selbst ein kleiner Datensatz sehr lange).
      Das trainierte Modell landet direkt am KI-Modellpfad der Einstellungen und ist danach
      ohne weiteren Schritt im Generator-Tab nutzbar (Häkchen "KI-Erkennung").</p>
    <div class="field-row"><label>Epochen</label><input type="number" id="rt-epochs" value="100" min="1" /></div>
    <div class="field-row"><label>Gerät</label>
      <select id="rt-device">
        <option value="cuda" selected>GPU (CUDA)</option>
        <option value="cpu">CPU (sehr langsam)</option>
      </select>
    </div>
    <div class="row"><button id="rt-train" class="primary" type="button" disabled>Training starten</button></div>
    <p class="hint" id="rt-train-unavailable" style="display:none; color:var(--danger);">
      ultralytics ist nicht installiert (nur fürs Trainieren nötig, nicht fürs Sammeln von
      Trainingsdaten oben) - im Terminal einmalig:
      <code>pip install -r generator/requirements-ai-train.txt</code>
    </p>
    <div class="path-label" id="rt-train-status"></div>
    <pre id="rt-train-log" class="hint" style="max-height:240px; overflow:auto; white-space:pre-wrap;"></pre>
  `;

  const el = sel => root.querySelector(sel);
  const canvas = el('#rt-canvas');
  const ctx = canvas.getContext('2d');

  let videoPath = null;
  let img = new Image();
  let nativeW = 0, nativeH = 0;
  let roi = null;
  let roi2 = null;
  let roi2Mode = false;
  let dragging = false, draggingSecond = false, startX = 0, startY = 0, curX = 0, curY = 0;
  let lastPrefix = '';
  let datasetDir = '';

  const DISPLAY_W = 560;
  const ROI1_STROKE = '#5fd0c8';
  const ROI1_FILL = 'rgba(95,208,200,0.15)';
  const ROI2_STROKE = '#8b7cff';
  const ROI2_FILL = 'rgba(139,124,255,0.18)';

  function setRoi2Mode(on) {
    roi2Mode = !!on;
    const btn = el('#rt-roi2-toggle');
    btn.style.outline = roi2Mode ? '2px solid #8b7cff' : '';
    btn.style.background = roi2Mode ? 'rgba(139,124,255,0.28)' : '';
  }

  function updateLabels() {
    el('#rt-roi-label').textContent = roi
      ? `Region: x=${roi.x} y=${roi.y} w=${roi.w} h=${roi.h} (Videopixel)`
      : 'Keine Region markiert';
    el('#rt-roi2-label').textContent = roi2
      ? `2. Region: x=${roi2.x} y=${roi2.y} w=${roi2.w} h=${roi2.h} (Videopixel, violett)`
      : 'Keine 2. Region markiert';
    el('#rt-class2').disabled = !roi2;
  }

  function updateBootstrapEnabled() {
    const class1 = el('#rt-class1').value.trim();
    const class2 = el('#rt-class2').value.trim();
    el('#rt-bootstrap').disabled = !(videoPath && roi && class1 && (!roi2 || class2));
  }

  function drawNativeRect(r, stroke, fill) {
    if (!r || !nativeW || !nativeH) return;
    const scaleX = canvas.width / nativeW, scaleY = canvas.height / nativeH;
    const dx0 = r.x * scaleX, dy0 = r.y * scaleY, dw = r.w * scaleX, dh = r.h * scaleY;
    ctx.strokeStyle = stroke;
    ctx.lineWidth = 2;
    ctx.strokeRect(dx0, dy0, dw, dh);
    ctx.fillStyle = fill;
    ctx.fillRect(dx0, dy0, dw, dh);
  }

  function drawDragRect(stroke, fill) {
    const dx0 = Math.min(startX, curX), dy0 = Math.min(startY, curY);
    const dw = Math.abs(curX - startX), dh = Math.abs(curY - startY);
    ctx.strokeStyle = stroke;
    ctx.lineWidth = 2;
    ctx.strokeRect(dx0, dy0, dw, dh);
    ctx.fillStyle = fill;
    ctx.fillRect(dx0, dy0, dw, dh);
  }

  function redraw() {
    ctx.clearRect(0, 0, canvas.width, canvas.height);
    if (img.src) ctx.drawImage(img, 0, 0, canvas.width, canvas.height);
    if (roi && !(dragging && !draggingSecond)) drawNativeRect(roi, ROI1_STROKE, ROI1_FILL);
    if (roi2 && !(dragging && draggingSecond)) drawNativeRect(roi2, ROI2_STROKE, ROI2_FILL);
    if (dragging) {
      if (draggingSecond) drawDragRect(ROI2_STROKE, ROI2_FILL);
      else drawDragRect(ROI1_STROKE, ROI1_FILL);
    }
  }

  canvas.addEventListener('mousedown', e => {
    const r = canvas.getBoundingClientRect();
    startX = curX = e.clientX - r.left;
    startY = curY = e.clientY - r.top;
    dragging = true;
    draggingSecond = roi2Mode || e.shiftKey;
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
    const wasSecond = draggingSecond;
    draggingSecond = false;
    const w = Math.abs(curX - startX), h = Math.abs(curY - startY);
    if (w < 4 || h < 4) { redraw(); return; }
    const scaleX = nativeW / canvas.width, scaleY = nativeH / canvas.height;
    const x0 = Math.min(startX, curX), y0 = Math.min(startY, curY);
    const box = {
      x: Math.round(x0 * scaleX), y: Math.round(y0 * scaleY),
      w: Math.round(w * scaleX), h: Math.round(h * scaleY),
    };
    if (wasSecond) {
      roi2 = box;
      setRoi2Mode(false);
    } else {
      roi = box;
    }
    updateLabels();
    updateBootstrapEnabled();
    redraw();
  });

  el('#rt-roi2-toggle').addEventListener('click', () => setRoi2Mode(!roi2Mode));

  async function chooseVideo() {
    const path = await PickVideoFile();
    if (document.activeElement && document.activeElement.blur) document.activeElement.blur();
    window.focus();
    if (!path) return;
    videoPath = path;
    el('#rt-video-path').textContent = path.split(/[\\/]/).pop();
    roi = null; roi2 = null;
    setRoi2Mode(false);
    updateLabels();
    updateBootstrapEnabled();
    el('#rt-bootstrap-status').textContent = 'Lade Vorschau-Frame...';
    try {
      const preview = await LoadFirstFrame(path);
      nativeW = preview.width; nativeH = preview.height;
      const displayH = Math.round(DISPLAY_W * nativeH / nativeW);
      canvas.width = DISPLAY_W; canvas.height = displayH;
      img.onload = redraw;
      img.src = 'data:image/png;base64,' + preview.pngBase64;
      el('#rt-bootstrap-status').textContent = 'Region markieren (Maus ziehen), optional 2. Region (Shift+Ziehen oder Knopf).';
    } catch (err) {
      el('#rt-bootstrap-status').textContent = '';
      alert('Fehler: ' + err);
    }
  }
  el('#rt-pick-video').addEventListener('click', chooseVideo);
  el('#rt-class1').addEventListener('input', updateBootstrapEnabled);
  el('#rt-class2').addEventListener('input', updateBootstrapEnabled);

  el('#rt-bootstrap').addEventListener('click', async () => {
    if (!videoPath || !roi) return;
    const class1 = el('#rt-class1').value.trim();
    const class2 = el('#rt-class2').value.trim();
    el('#rt-bootstrap').disabled = true;
    el('#rt-bootstrap-status').textContent = 'Läuft… (kann bei langen Videos Minuten dauern)';
    el('#rt-bootstrap-log').textContent = '';
    try {
      const roiArg = { X: roi.x, Y: roi.y, W: roi.w, H: roi.h };
      const roi2Arg = roi2 ? { X: roi2.x, Y: roi2.y, W: roi2.w, H: roi2.h } : null;
      lastPrefix = await BootstrapRoiTrainingSample(videoPath, roiArg, roi2Arg, class1, class2);
      // Knopf bleibt gesperrt, bis roitraining:bootstrap:done ankommt -
      // Go lässt ohnehin nur einen Bootstrap-/Trainingslauf gleichzeitig zu.
    } catch (err) {
      el('#rt-bootstrap-status').textContent = 'Fehlgeschlagen.';
      alert('Fehler: ' + err);
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
      el('#rt-bootstrap-status').textContent = 'Fehlgeschlagen.';
      alert('Bootstrap fehlgeschlagen: ' + payload.error);
      return;
    }
    lastPrefix = payload.prefix || lastPrefix;
    el('#rt-bootstrap-status').textContent = 'Fertig - Beispiele zur Kontrollansicht hinzugefügt.';
    refreshReview();
    refreshClassList();
  });

  // --- Kontrollansicht -----------------------------------------------
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
          ? 'Keine neuen Beispiele gefunden - Häkchen "nur zuletzt hinzugekommene" abwählen, um den ganzen Datensatz zu sehen.'
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
        card.appendChild(thumbWrap);
        card.appendChild(caption);
        card.appendChild(discardBtn);
        gridEl.appendChild(card);

        GetRoiTrainingSampleImage(s.imagePath).then(b64 => {
          imgEl.src = 'data:image/jpeg;base64,' + b64;
          (s.boxes || []).forEach(box => thumbWrap.appendChild(boxOverlay(box)));
        }).catch(() => { caption.textContent += ' (Bild konnte nicht geladen werden)'; });

        discardBtn.addEventListener('click', async () => {
          discardBtn.disabled = true;
          try {
            await DiscardRoiTrainingSample(datasetDir, s.split, s.name);
            card.remove();
          } catch (err) {
            alert('Konnte nicht verworfen werden: ' + err);
            discardBtn.disabled = false;
          }
        });
      }
    } catch (err) {
      grid.textContent = 'Fehler: ' + err;
    }
  }
  el('#rt-refresh-review').addEventListener('click', refreshReview);
  el('#rt-dataset-dir').addEventListener('change', e => {
    saveSetting('generator.roiTrainingDatasetDir', e.target.value.trim());
  });

  // --- Datensatz-Übersicht + Klassen-Autocomplete ---------------------
  async function refreshClassList() {
    try {
      const summary = await GetRoiDatasetSummary(datasetDir || el('#rt-dataset-dir').value.trim());
      const list = el('#rt-class-list');
      list.innerHTML = (summary.classes || [])
        .map(c => `<option value="${c.className}"></option>`).join('');
      return summary;
    } catch {
      return null;
    }
  }

  el('#rt-summary-refresh').addEventListener('click', async () => {
    datasetDir = el('#rt-dataset-dir').value.trim();
    const box = el('#rt-summary');
    box.textContent = 'Lädt…';
    try {
      const summary = await GetRoiDatasetSummary(datasetDir);
      if (!summary.classes || summary.classes.length === 0) {
        box.textContent = 'Noch keine Klassen im Datensatz - erst Trainingsdaten sammeln.';
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
      const list = el('#rt-class-list');
      list.innerHTML = summary.classes.map(c => `<option value="${c.className}"></option>`).join('');
    } catch (err) {
      box.textContent = 'Fehler: ' + err;
    }
  });

  // --- Training ---------------------------------------------------------
  el('#rt-train').addEventListener('click', async () => {
    const epochs = parseInt(el('#rt-epochs').value, 10) || 100;
    const device = el('#rt-device').value;
    el('#rt-train').disabled = true;
    el('#rt-train-status').textContent = 'Läuft… (kann je nach Datensatzgröße und Gerät lange dauern)';
    el('#rt-train-log').textContent = '';
    try {
      await RunRoiModelTraining(epochs, device);
    } catch (err) {
      el('#rt-train-status').textContent = 'Fehlgeschlagen.';
      alert('Fehler: ' + err);
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
      el('#rt-train-status').textContent = 'Fehlgeschlagen.';
      alert('Training fehlgeschlagen: ' + payload.error);
      return;
    }
    el('#rt-train-status').textContent = 'Fertig: ' + payload.modelPath;
  });

  getSettingsCache().then(s => {
    datasetDir = s.roiDatasetDir || s.defaultRoiDatasetDir || '';
    el('#rt-dataset-dir').value = datasetDir;
    refreshClassList();
  });

  // ultralytics ist eine schwere, separate Installation (zieht u.a. PyTorch
  // nach sich, siehe requirements-ai-train.txt) - den Knopf anzubieten und
  // dann mit einem rohen ModuleNotFoundError-Traceback scheitern zu lassen
  // wäre schlechter als ihn vorher zu sperren (gleiches Muster wie
  // CheckAIRoiAvailable im Generator-Tab).
  CheckRoiTrainingAvailable().then(available => {
    el('#rt-train').disabled = !available;
    el('#rt-train-unavailable').style.display = available ? 'none' : 'block';
  }).catch(() => {
    el('#rt-train-unavailable').style.display = 'block';
  });
}
