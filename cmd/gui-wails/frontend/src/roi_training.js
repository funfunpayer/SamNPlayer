import {
  PickVideoFile, PickImageFile, LoadFirstFrame, LoadFrameAt,
  BootstrapRoiTrainingRegions, AddRoiStillTrainingSample, RunRoiModelTraining,
  ListRoiTrainingSamples, DiscardRoiTrainingSample, UpdateRoiTrainingSample, GetRoiDatasetSummary,
  GetRoiTrainingSampleImage, CheckRoiTrainingAvailable, CheckRoiTrainingStatus,
  InstallRoiTrainingDeps, ListRoiTrainingDevices,
  GetMotionProfileModelStatus, TrainMotionProfileModel,
} from '../wailsjs/go/main/App';
import { CLASS_PRESETS, MAX_REGIONS, normalizeClass, labelFor } from './bodyparts.js';
import { mountBodyFigure } from './body_figure.js';
import { EventsOn } from '../wailsjs/runtime/runtime';
import { getSettingsCache, saveSetting } from './settings.js';
import { wireDataHelp } from './help.js';
import { uiError, uiInfo, uiWarn } from './notify.js';

const MARK_COLORS = [
  { stroke: '#3dccc0', fill: 'rgba(61,204,192,0.16)' },
  { stroke: '#f2b03d', fill: 'rgba(242,176,61,0.18)' },
  { stroke: '#5ecf8a', fill: 'rgba(94,207,138,0.16)' },
  { stroke: '#e8eaed', fill: 'rgba(232,234,237,0.12)' },
  { stroke: '#7aa2ff', fill: 'rgba(122,162,255,0.16)' },
  { stroke: '#ff7a9a', fill: 'rgba(255,122,154,0.16)' },
  { stroke: '#c9a0ff', fill: 'rgba(201,160,255,0.14)' },
  { stroke: '#9ad6ff', fill: 'rgba(154,214,255,0.14)' },
  { stroke: '#ffd27a', fill: 'rgba(255,210,122,0.16)' },
];

export function initRoiTraining(root) {
  root.innerHTML = `
    <h2>AI training</h2>
    <p class="hint">
      Mark on video or still → sample → review → train locally.
      Up to nine body-part classes per image (Face, Mouth, Breasts, Nipples, Hand 1/2, Penis, Glans, Vagina). Audio is stored with the dataset.
    </p>
    <div class="card" style="margin-bottom:16px; padding:12px 14px;">
      <h3 style="margin-top:0; margin-bottom:8px;">From marks to a good script</h3>
      <ol class="hint" style="margin:0; padding-left:1.2em; line-height:1.55;">
        <li><b>Train here</b> — marks → “Use for training” → discard bad samples → start training. Result: <code>roi_detector.onnx</code>.</li>
        <li><b>Then Create</b> — Load video → enable “Smarter tip find” → “Find tip area”. AI suggests the box only.</li>
        <li><b>Verify/correct boxes</b> — never apply blindly. For Tf/Tj, tip + fixed target (e.g. Glans + Nipples).</li>
        <li><b>Create Emotion Script</b> — classic tracking (CSRT) writes the script. AI does not track by itself.</li>
        <li><b>Review in Play</b> — Feedback buttons (usable/…) improve Quality Doctor later, not region AI.</li>
      </ol>
    </div>

    <div class="card" id="rt-motion-profile-card" style="margin-bottom:16px; padding:12px 14px;">
      <h3 style="margin-top:0; margin-bottom:6px;">Learned generation profile (Go)</h3>
      <p class="hint" style="margin:0 0 8px;">
        Learns from scenes remembered in <b>Create → Power-user: scene memory</b> and the
        Style selected there. Training/classification are pure Go and offline; measuring a
        new video's signature uses the existing local OpenCV adapter. It suggests
        Normal / Soft / Autotune only; it never changes settings or writes a script without Apply/Create.
      </p>
      <div class="row" style="align-items:center; flex-wrap:wrap;">
        <button id="rt-profile-refresh" type="button">Refresh learning data</button>
        <button id="rt-profile-train" class="primary" type="button" disabled>Train Go profile model</button>
        <button id="rt-open-create" type="button">Generate in Create…</button>
      </div>
      <div class="path-label" id="rt-profile-status">Checking saved scenes…</div>
      <p class="hint" id="rt-profile-path" style="margin:4px 0 0;"></p>
    </div>

    <h3>1. Collect training data</h3>
    <div class="row">
      <button id="rt-pick-video" type="button">Choose video…</button>
      <button id="rt-pick-image" type="button"
        data-help="Single image without tracking — good to define classes; transfer to clips later.">Choose image…</button>
      <span class="path-label" id="rt-video-path">No source selected</span>
    </div>
    <div class="row" style="align-items:center; margin:6px 0;">
      <label style="width:auto;" data-help="Offset for black intro — pick frame to mark.">Time (s)</label>
      <input type="number" id="rt-seek" value="0" min="0" step="0.5" style="width:5em;" />
      <button id="rt-seek-btn" type="button" disabled>Load frame</button>
      <button id="rt-seek-plus" type="button" disabled title="+1 s">+1s</button>
      <button id="rt-seek-plus5" type="button" disabled title="+5 s">+5s</button>
    </div>
    <div id="rt-canvas-wrap">
      <canvas id="rt-canvas"></canvas>
    </div>
    <div class="row" style="align-items:center; flex-wrap:wrap;">
      <button id="rt-mark-next" type="button"
        data-help="Start next mark (up to 9 body-part boxes on the same image).">+ Mark</button>
      <button id="rt-clear-marks" type="button">Clear all</button>
      <span class="hint" style="margin:0">Active: <span id="rt-active-mark">1</span>/9 — Shift+drag = next class</span>
    </div>
    <div id="rt-mark-fields"></div>
    <datalist id="rt-class-list"></datalist>
    <div id="rt-body-figure" class="body-figure-host" aria-label="Human body map for class picking"></div>
    <div id="rt-class-chips" class="rt-chips" aria-label="Known classes"></div>
    <div class="field-row"><label data-help="Write every N-th frame as a sample. Smaller = denser; larger = less redundancy.">Sampling (every N-th frame)</label>
      <input type="number" id="rt-sample-every" value="12" min="1" max="120" style="width:5em;" />
    </div>
    <div class="field-row"><label data-help="Multiply marked box size for YOLO labels (1.0 = exact mark; 1.1–1.2 adds a small pad). Prefer correcting boxes in review over a large scale.">Box scale</label>
      <input type="number" id="rt-box-scale" value="1.0" min="0.5" max="2.0" step="0.05" style="width:5em;" />
    </div>
    <div class="checkbox-row"><input type="checkbox" id="rt-extract-audio" checked />
      <label for="rt-extract-audio" style="width:auto"
        data-help="Saves audio as WAV under audio/ in the dataset — for future learning.">Save audio too</label></div>
    <div class="row">
      <button id="rt-bootstrap" class="primary" disabled
        data-help="Video: tracks marks and writes YOLO samples. Image: saves a still annotation.">Use for training</button>
    </div>
    <div class="path-label" id="rt-bootstrap-status"></div>
    <div id="rt-bootstrap-progress-wrap" class="run-progress" style="display:none;">
      <div class="run-progress-track"><div id="rt-bootstrap-progress-bar" class="run-progress-bar"></div></div>
      <div id="rt-bootstrap-progress-text" class="hint" style="margin-top:4px;"></div>
    </div>
    <pre id="rt-bootstrap-log" class="run-log" aria-label="Bootstrap progress log"></pre>

    <h3>2. Review</h3>
    <div class="field-row"><label data-help="Folder with images/ and labels/ from bootstrap.">Dataset folder</label>
      <input type="text" id="rt-dataset-dir" style="flex:1" />
    </div>
    <div class="row" style="align-items:center;">
      <button id="rt-refresh-review" type="button">Refresh</button>
      <label class="hint" style="margin:0"
        data-help="Shows only samples from the last run."><input type="checkbox" id="rt-only-new" checked /> only newly added</label>
    </div>
    <div id="rt-review-grid" class="hint">No samples loaded yet.</div>

    <h3>3. Dataset summary</h3>
    <div class="row"><button id="rt-summary-refresh" type="button">Load summary</button></div>
    <div id="rt-summary" class="hint"></div>

    <h3>4. Train model</h3>
    <p class="hint">Optional. Only needed if you want your own ONNX region model.
      Play / Create work without this. Device: Auto picks CUDA → MPS → DirectML → CPU.</p>
    <div class="field-row"><label data-help="Training passes. 50–100 typical for small sets.">Epochs</label><input type="number" id="rt-epochs" value="100" min="1" /></div>
    <div class="field-row"><label data-help="auto = best available backend.">Device</label>
      <select id="rt-device">
        <option value="auto" selected>Automatic</option>
        <option value="cuda">NVIDIA CUDA</option>
        <option value="directml">DirectML (Windows)</option>
        <option value="mps">Apple MPS</option>
        <option value="cpu">CPU (very slow)</option>
      </select>
    </div>
    <p class="hint" id="rt-device-status" style="margin:0 0 8px;"></p>
    <div class="row"><button id="rt-train" class="primary" type="button" disabled>Start training</button>
      <button id="rt-install-deps" type="button"
        data-help="Installs ultralytics + onnx (+ PyTorch), then restores opencv-contrib-python so Use for training keeps CSRT. Prefer a real Python install (not the Windows Store stub). Large download.">Install AI train deps</button></div>
    <p class="hint" id="rt-train-unavailable" style="display:none; color:var(--danger);">
      AI train packages missing (optional). Use “Install AI train deps”
      only if you want to train a model (large: ultralytics/torch). That
      button also restores <code>opencv-contrib-python</code> so
      <b>Use for training</b> keeps CSRT after ultralytics. Prefer a real
      Python from python.org — not the Windows Store stub. Or:
      <code>pip uninstall opencv-python opencv-python-headless &amp;&amp; pip install opencv-contrib-python ultralytics onnx</code>.
      Details: <a href="#" id="rt-docs-link">docs/KI_TRAINING.md</a>
    </p>
    <p class="hint" id="rt-dataset-hint" style="display:none; color:var(--danger);">
      No training samples yet. Mark region(s) above and click <b>Use for training</b>
      (or add a photo still) before starting training.
    </p>
    <p class="hint" id="rt-status-detail" style="margin:0 0 8px;"></p>
    <div class="path-label" id="rt-train-status"></div>
    <div id="rt-train-progress-wrap" class="run-progress" style="display:none;">
      <div class="run-progress-track"><div id="rt-train-progress-bar" class="run-progress-bar"></div></div>
      <div id="rt-train-progress-text" class="hint" style="margin-top:4px;"></div>
    </div>
    <pre id="rt-train-log" class="run-log" aria-label="Training progress log"></pre>
  `;

  wireDataHelp(root);

  const el = sel => root.querySelector(sel);
  const canvas = el('#rt-canvas');
  const ctx = canvas.getContext('2d');

  async function refreshMotionProfileModelStatus() {
    const statusEl = el('#rt-profile-status');
    const pathEl = el('#rt-profile-path');
    const trainBtn = el('#rt-profile-train');
    try {
      const status = await GetMotionProfileModelStatus();
      const profiles = (status.profiles || []).map(p => `${p.profile}: ${p.count}`).join(' · ');
      const modelText = status.modelAvailable
        ? `Model ready (${status.trainedSamples} samples)`
        : 'No trained model yet';
      statusEl.textContent = `${status.usableSamples || 0} usable saved scenes · ${modelText}`
        + (profiles ? ` · ${profiles}` : '');
      pathEl.textContent = status.readyToTrain
        ? 'Ready to train. New remembered scenes are included on the next training run.'
        : 'Remember at least two scenes with a selected Style in Create before training.';
      if (status.modelWarning) {
        pathEl.textContent += ' Existing model is unreadable/incompatible; training will replace it.';
      }
      pathEl.title = `Labels: ${status.labelsPath || ''}\nModel: ${status.modelPath || ''}`;
      trainBtn.disabled = !status.readyToTrain;
      return status;
    } catch (err) {
      statusEl.textContent = 'Profile learning status unavailable: ' + err;
      trainBtn.disabled = true;
      return null;
    }
  }

  el('#rt-profile-refresh').addEventListener('click', refreshMotionProfileModelStatus);
  el('#rt-profile-train').addEventListener('click', async () => {
    const btn = el('#rt-profile-train');
    const statusEl = el('#rt-profile-status');
    btn.disabled = true;
    statusEl.textContent = 'Training Go profile model…';
    try {
      const status = await TrainMotionProfileModel();
      statusEl.textContent = `Model trained from ${status.trainedSamples || status.usableSamples || 0} scenes. `
        + 'Create can now ask it for a profile suggestion.';
      uiInfo('Local motion-profile model trained. Suggestions stay opt-in in Create.');
      window.dispatchEvent(new CustomEvent('samn-motion-profile-refresh'));
      await refreshMotionProfileModelStatus();
    } catch (err) {
      uiError('Profile model training: ' + err, statusEl);
      await refreshMotionProfileModelStatus();
    }
  });
  el('#rt-open-create').addEventListener('click', () => {
    document.querySelector('.tab-btn[data-tab="generator"]')?.click();
  });
  refreshMotionProfileModelStatus();

  let sourcePath = null;
  let sourceKind = null; // 'video' | 'image'
  let img = new Image();
  let nativeW = 0, nativeH = 0;
  let marks = Array(9).fill(null); // up to MAX_REGIONS
  let activeMark = 0;
  let dragging = false, startX = 0, startY = 0, curX = 0, curY = 0;
  let lastPrefix = '';
  let datasetDir = '';
  let seekSec = 0;

  function applyClassToActiveMark(classId) {
    const n = normalizeClass(classId);
    if (!n) return;
    // Prefer the first drawn box that still has no class — body-map clicks
    // usually happen right after marking, when activeMark already advanced.
    let idx = activeMark;
    for (let i = 0; i < MAX_REGIONS; i++) {
      if (marks[i] && !(el(`#rt-class${i + 1}`)?.value || '').trim()) {
        idx = i;
        break;
      }
    }
    const input = el(`#rt-class${idx + 1}`);
    if (input) {
      input.value = n;
      updateBootstrapEnabled();
    }
    bodyFigure?.setActive(n);
  }

  const bodyFigure = mountBodyFigure(el('#rt-body-figure'), {
    onSelect: applyClassToActiveMark,
    getActive: () => el(`#rt-class${activeMark + 1}`)?.value || '',
  });

  const DISPLAY_W = 560;

  function renderMarkFields() {
    const prev = [];
    for (let i = 0; i < MAX_REGIONS; i++) {
      prev[i] = el(`#rt-class${i + 1}`)?.value || '';
    }
    const wrap = el('#rt-mark-fields');
    wrap.innerHTML = '';
    for (let i = 0; i < MAX_REGIONS; i++) {
      const row = document.createElement('div');
      row.className = 'field-row';
      row.style.opacity = (i === 0 || marks[i] || i === activeMark) ? '1' : '0.55';
      const lab = document.createElement('label');
      lab.textContent = `Class ${i + 1}`;
      lab.style.borderLeft = `3px solid ${MARK_COLORS[i].stroke}`;
      lab.style.paddingLeft = '6px';
      const input = document.createElement('input');
      input.type = 'text';
      input.id = `rt-class${i + 1}`;
      input.setAttribute('list', 'rt-class-list');
      input.placeholder = CLASS_PRESETS[i] || 'e.g. glans';
      if (prev[i]) input.value = prev[i];
      input.addEventListener('input', updateBootstrapEnabled);
      const meta = document.createElement('span');
      meta.className = 'hint';
      meta.id = `rt-mark-label${i + 1}`;
      meta.style.margin = '0 0 0 8px';
      meta.textContent = marks[i]
        ? `${marks[i].w}×${marks[i].h}`
        : (i === activeMark ? 'draw…' : '—');
      row.appendChild(lab);
      row.appendChild(input);
      row.appendChild(meta);
      wrap.appendChild(row);
    }
    el('#rt-active-mark').textContent = String(activeMark + 1);
    bodyFigure?.setActive(el(`#rt-class${activeMark + 1}`)?.value || '');
    updateBootstrapEnabled();
  }

  function updateBootstrapEnabled() {
    const class1 = el('#rt-class1')?.value.trim();
    let ok = !!(sourcePath && marks[0] && class1);
    for (let i = 1; i < MAX_REGIONS; i++) {
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
    for (let i = 0; i < MAX_REGIONS; i++) {
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
      for (let i = 0; i < MAX_REGIONS; i++) {
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
    if (activeMark < MAX_REGIONS - 1) activeMark += 1;
    renderMarkFields();
    updateBootstrapEnabled();
    redraw();
  });

  el('#rt-mark-next').addEventListener('click', () => {
    for (let i = 0; i < MAX_REGIONS; i++) {
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
    el('#rt-bootstrap-status').textContent = 'Loading frame…';
    try {
      await showPreview(path, 0);
      el('#rt-bootstrap-status').textContent = 'Mark region(s). If the start is black, seek forward first.';
    } catch (err) {
      uiError('Load video: ' + err, el('#rt-bootstrap-status'));
    }
  }

  async function chooseImage() {
    const path = await PickImageFile();
    if (document.activeElement?.blur) document.activeElement.blur();
    window.focus();
    if (!path) return;
    sourcePath = path;
    sourceKind = 'image';
    el('#rt-video-path').textContent = path.split(/[\\/]/).pop() + ' (image)';
    marks = [null, null, null, null];
    activeMark = 0;
    renderMarkFields();
    updateBootstrapEnabled();
    el('#rt-seek-btn').disabled = true;
    el('#rt-seek-plus').disabled = true;
    el('#rt-seek-plus5').disabled = true;
    el('#rt-bootstrap-status').textContent = 'Loading image…';
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
      el('#rt-bootstrap-status').textContent = 'Still image: mark regions, then save.';
    } catch (err) {
      uiError('Load image: ' + err, el('#rt-bootstrap-status'));
    }
  }

  async function seekTo(sec) {
    if (!sourcePath || sourceKind !== 'video') return;
    seekSec = Math.max(0, sec);
    el('#rt-seek').value = String(seekSec);
    el('#rt-bootstrap-status').textContent = `Loading frame at ${seekSec}s…`;
    try {
      await showPreview(sourcePath, seekSec);
      el('#rt-bootstrap-status').textContent = `Frame at ${seekSec}s — mark region(s).`;
    } catch (err) {
      uiError('Seek failed: ' + err, el('#rt-bootstrap-status'));
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
    const classes = [];
    for (let i = 1; i <= MAX_REGIONS; i++) {
      classes.push(el(`#rt-class${i}`)?.value.trim() || '');
    }
    if (!classes[0]) return;
    el('#rt-bootstrap').disabled = true;
    el('#rt-bootstrap-status').textContent = 'Running…';
    el('#rt-bootstrap-log').textContent = '';
    showRunProgress('rt-bootstrap', true);
    try {
      if (sourceKind === 'image') {
        const regions = [];
        for (let i = 0; i < MAX_REGIONS; i++) {
          if (marks[i] && classes[i]) {
            regions.push({ ROI: roiArg(marks[i]), ClassName: normalizeClass(classes[i]) || classes[i] });
          }
        }
        lastPrefix = await AddRoiStillTrainingSample(sourcePath, regions);
        showRunProgress('rt-bootstrap', false);
        el('#rt-bootstrap-status').textContent = 'Still sample saved.';
        updateBootstrapEnabled();
        refreshReview();
        refreshClassList();
        refreshDatasetReadyHint();
      } else {
        const sampleEvery = parseInt(el('#rt-sample-every').value, 10) || 12;
        const extractAudio = el('#rt-extract-audio').checked;
        const boxScale = parseFloat(el('#rt-box-scale')?.value) || 1.0;
        const regions = [];
        for (let i = 0; i < MAX_REGIONS; i++) {
          if (marks[i] && classes[i]) {
            regions.push({ ROI: roiArg(marks[i]), ClassName: normalizeClass(classes[i]) || classes[i] });
          }
        }
        lastPrefix = await BootstrapRoiTrainingRegions(
          sourcePath, regions, sampleEvery, extractAudio, seekSec > 0 ? seekSec : 0, boxScale,
        );
      }
    } catch (err) {
      showRunProgress('rt-bootstrap', false);
      uiError('Save sample: ' + err, el('#rt-bootstrap-status'));
      updateBootstrapEnabled();
    }
  });

  function showRunProgress(prefix, show) {
    const wrap = el(`#${prefix}-progress-wrap`);
    if (!wrap) return;
    wrap.style.display = show ? 'block' : 'none';
    if (show) {
      const bar = el(`#${prefix}-progress-bar`);
      const text = el(`#${prefix}-progress-text`);
      if (bar) { bar.style.width = '0%'; bar.style.opacity = '1'; }
      if (text) text.textContent = 'Starting…';
    }
  }

  function applyRunPercent(prefix, pct) {
    const wrap = el(`#${prefix}-progress-wrap`);
    const bar = el(`#${prefix}-progress-bar`);
    const text = el(`#${prefix}-progress-text`);
    if (!wrap || !bar || !text) return;
    if (wrap.style.display === 'none') wrap.style.display = 'block';
    if (pct < 0) {
      bar.style.width = '100%';
      bar.style.opacity = '0.35';
      text.textContent = 'Running… (length unknown)';
      return;
    }
    bar.style.opacity = '1';
    bar.style.width = pct + '%';
    text.textContent = pct + ' %';
  }

  EventsOn('roitraining:bootstrap:progress', line => {
    const log = el('#rt-bootstrap-log');
    log.textContent += line + '\n';
    log.scrollTop = log.scrollHeight;
  });
  EventsOn('roitraining:bootstrap:percent', pct => applyRunPercent('rt-bootstrap', pct));
  EventsOn('roitraining:bootstrap:done', payload => {
    showRunProgress('rt-bootstrap', false);
    updateBootstrapEnabled();
    if (payload.error) {
      uiError('Bootstrap failed: ' + payload.error, el('#rt-bootstrap-status'));
      return;
    }
    lastPrefix = payload.prefix || lastPrefix;
    el('#rt-bootstrap-status').textContent = 'Done — samples in the review view.';
    refreshReview();
    refreshClassList();
    refreshDatasetReadyHint();
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
    grid.textContent = 'Loading…';
    try {
      const samples = await ListRoiTrainingSamples(datasetDir, prefix);
      if (!samples || samples.length === 0) {
        grid.textContent = onlyNew && lastPrefix
          ? 'No new samples — uncheck to use the full dataset.'
          : 'No samples found.';
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
        discardBtn.textContent = 'Discard';
        const editBtn = document.createElement('button');
        editBtn.type = 'button';
        editBtn.textContent = 'Correct box';
        editBtn.title = 'Redraw the box on the preview, then save';
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
        }).catch(() => { caption.textContent += ' (could not load image)'; });

        editBtn.addEventListener('click', async () => {
          try {
            await editSampleBoxes(s, imgEl, thumbWrap);
          } catch (err) {
            uiError('Correct box: ' + err);
          }
        });

        discardBtn.addEventListener('click', async () => {
          discardBtn.disabled = true;
          try {
            await DiscardRoiTrainingSample(datasetDir, s.split, s.name);
            card.remove();
            refreshDatasetReadyHint();
          } catch (err) {
            uiError('Discard sample: ' + err);
            discardBtn.disabled = false;
          }
        });
      }
    } catch (err) {
      grid.textContent = 'Error: ' + err;
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
    uiInfo('Drag on preview: new box for the first class. Escape cancels.');
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
      uiInfo('Box saved.');
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
    const known = fromData.length ? fromData : allNames.slice(0, MAX_REGIONS);
    if (!known.length) {
      wrap.innerHTML = '<span class="hint">No classes yet — use the English body-part presets or mark a region.</span>';
      return;
    }
    const label = document.createElement('span');
    label.className = 'hint';
    label.style.marginRight = '6px';
    label.textContent = fromData.length ? 'From dataset:' : 'Body parts:';
    wrap.appendChild(label);
    known.forEach(name => {
      const btn = document.createElement('button');
      btn.type = 'button';
      btn.className = 'rt-chip';
      btn.textContent = labelFor(name) || name;
      btn.title = name;
      btn.addEventListener('click', () => applyClassToActiveMark(name));
      wrap.appendChild(btn);
    });
  }

  el('#rt-summary-refresh').addEventListener('click', async () => {
    datasetDir = el('#rt-dataset-dir').value.trim();
    const box = el('#rt-summary');
    box.textContent = 'Loading…';
    try {
      const summary = await GetRoiDatasetSummary(datasetDir);
      if (!summary.classes || summary.classes.length === 0) {
        box.textContent = 'No classes yet — collect data first.';
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
      box.textContent = 'Error: ' + err;
    }
  });

  el('#rt-train').addEventListener('click', async () => {
    datasetDir = el('#rt-dataset-dir').value.trim();
    try {
      const summary = await GetRoiDatasetSummary(datasetDir);
      if (!summary.readyToTrain) {
        uiWarn('Collect samples first: mark region(s) → “Use for training”, then start training.',
          el('#rt-train-status'));
        const hint = el('#rt-dataset-hint');
        if (hint) hint.style.display = 'block';
        return;
      }
    } catch (_) { /* fall through; Go/Python will fail fast with a clear error */ }
    const epochs = parseInt(el('#rt-epochs').value, 10) || 100;
    const device = el('#rt-device').value;
    el('#rt-train').disabled = true;
    el('#rt-train-status').textContent = 'Running…';
    el('#rt-train-log').textContent = '';
    showRunProgress('rt-train', true);
    try {
      await RunRoiModelTraining(epochs, device);
    } catch (err) {
      showRunProgress('rt-train', false);
      uiError('ROI training: ' + err, el('#rt-train-status'));
      el('#rt-train').disabled = false;
    }
  });
  EventsOn('roitraining:train:progress', line => {
    const log = el('#rt-train-log');
    log.textContent += line + '\n';
    log.scrollTop = log.scrollHeight;
  });
  EventsOn('roitraining:train:percent', pct => applyRunPercent('rt-train', pct));
  EventsOn('roitraining:train:done', payload => {
    showRunProgress('rt-train', false);
    el('#rt-train').disabled = false;
    if (payload.error) {
      uiError('Training failed: ' + payload.error, el('#rt-train-status'));
      return;
    }
    el('#rt-train-status').textContent = 'Done: ' + payload.modelPath
      + ' — AI detection in the Create tab updates automatically.';
    window.dispatchEvent(new CustomEvent('samn-ai-roi-refresh'));
  });

  getSettingsCache().then(s => {
    datasetDir = s.roiDatasetDir || s.defaultRoiDatasetDir || '';
    el('#rt-dataset-dir').value = datasetDir;
    refreshClassList();
    refreshDatasetReadyHint();
  });

  function applyTrainAvailability(available, detail) {
    el('#rt-train').disabled = !available;
    el('#rt-train-unavailable').style.display = available ? 'none' : 'block';
    if (el('#rt-status-detail') && detail) {
      el('#rt-status-detail').textContent = detail;
    }
    if (available) refreshDatasetReadyHint();
  }

  async function refreshDatasetReadyHint() {
    const hint = el('#rt-dataset-hint');
    if (!hint) return;
    datasetDir = el('#rt-dataset-dir').value.trim();
    if (!datasetDir) {
      hint.style.display = 'block';
      return;
    }
    try {
      const summary = await GetRoiDatasetSummary(datasetDir);
      hint.style.display = summary.readyToTrain ? 'none' : 'block';
    } catch (_) {
      hint.style.display = 'block';
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
    el('#rt-train-status').textContent = 'Installing dependencies…';
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
      uiError('Install failed: ' + payload.error, el('#rt-train-status'));
      return;
    }
    const st = payload.status || {};
    applyTrainAvailability(!!st.ultralytics, st.detail || 'Dependencies installed');
    const bootstrapOk = !!st.opencv;
    el('#rt-train-status').textContent = bootstrapOk
      ? 'Dependencies OK — bootstrap (CSRT) and training ready.'
      : 'Ultralytics OK, but OpenCV trackers still missing — re-run Install or pip install opencv-contrib-python.';
    uiInfo(bootstrapOk
      ? 'AI training dependencies installed (CSRT restored).'
      : 'AI train deps installed; fix OpenCV (opencv-contrib-python) for Use for training.');
  });

  ListRoiTrainingDevices().then(devices => {
    if (!Array.isArray(devices) || !devices.length) return;
    const sel = el('#rt-device');
    const current = sel.value || 'auto';
    sel.innerHTML = devices.map(d => {
      const mark = d.available ? '' : ' — not detected';
      return `<option value="${d.id}">${d.label}${mark}</option>`;
    }).join('');
    if ([...sel.options].some(o => o.value === current)) sel.value = current;
    else sel.value = 'auto';
    const avail = devices.filter(d => d.available && d.id !== 'auto').map(d => d.id);
    el('#rt-device-status').textContent = avail.length
      ? ('Detected: ' + avail.join(', '))
      : 'No GPU backend detected — Auto falls back to CPU.';
  }).catch(() => {
    el('#rt-device-status').textContent = '';
  });

  renderMarkFields();
  el('#rt-class-list').innerHTML = CLASS_PRESETS.map(n => `<option value="${n}"></option>`).join('');
}
