import { SubmitFeedback, PickVideoFile, LoadFirstFrame, LoadFrameAt, GenerateScript, CancelGenerate, CheckGeneratorDependencies, ScriptExistsForVideo, AutoDetectROI, SuggestROICandidates, CheckAIRoiAvailable, CheckAudioCheckAvailable, SuggestProfile, SuggestPipeline, LabelScene } from '../wailsjs/go/main/App';
import { CANONICAL } from './bodyparts.js';
import { EventsOn } from '../wailsjs/runtime/runtime';
import { uiError, uiInfo, uiWarn } from './notify.js';
import { wireDataHelp } from './help.js';

export function initGenerator(root, playback) {
  root.innerHTML = `
    <h2>Generate script</h2>
    <nav class="gen-steps" id="gen-steps" aria-label="Generate workflow">
      <ol class="gen-steps-list">
        <li class="gen-step-item is-current" data-step="1"><span class="gen-step-num">1</span> Video</li>
        <li class="gen-step-item" data-step="2"><span class="gen-step-num">2</span> Region</li>
        <li class="gen-step-item" data-step="3"><span class="gen-step-num">3</span> Motion</li>
        <li class="gen-step-item" data-step="4"><span class="gen-step-num">4</span> Generate</li>
        <li class="gen-step-item" data-step="5"><span class="gen-step-num">5</span> Review</li>
      </ol>
    </nav>
    <p class="hint gen-step-prompt" id="gen-step-prompt" style="margin-top:0">
      Start here: choose a video. The next step appears when this one is done.
    </p>
    <div class="path-label" id="gen-status"></div>
    <div class="hint" id="gen-pipeline" style="margin-top:4px;"></div>

    <section class="gen-step-panel" id="gen-step-video" data-step="1">
      <h3 class="gen-step-title">1 · Video</h3>
      <div class="row">
        <button id="gen-choose" class="primary">Choose video…</button>
        <span class="path-label" id="gen-video-path">No video selected</span>
        <button id="gen-check-deps">Check dependencies</button>
      </div>
    </section>

    <section class="gen-step-panel" id="gen-step-region" data-step="2" hidden>
      <h3 class="gen-step-title">2 · Mark region</h3>
      <p class="hint" style="margin-top:0">
        Drag on the frame (or find automatically). Classical CSRT tracks this box —
        AI may only propose the start region (optional). For Tf/Tj add contact zones
        (Zone 2 / Zone 3+) — e.g. nipples — so suction + contact vibration follow approach.
      </p>
      <div class="row" style="align-items:center;">
        <button id="gen-autoroi" class="primary" disabled
          data-help="Finds a start region from motion in the frame (for Tf/Tj, both regions as a suggestion). You can always correct the box by hand — never applied silently.">Find region automatically</button>
        <button id="gen-candidates" type="button" disabled
          data-help="Shows all ranked motion regions as dashed boxes. Click one to set Zone 1 (primary stroke). Nothing is applied until you pick — Zone 2 is never auto-filled.">Show motion candidates</button>
        <span class="checkbox-row" style="margin:0"><input type="checkbox" id="gen-ai-roi" disabled />
          <label for="gen-ai-roi" style="width:auto"
            data-help="Uses a local ONNX model instead of classic motion search. Needs a trained model under Settings → AI region detection. Stays off if onnxruntime or the model file is missing.">AI detection (ONNX)</label></span>
      </div>
      <p class="hint" id="gen-autoroi-hint" style="margin:0 0 6px 0">Analyzes motion in the video — you can still correct the region by hand. Candidates: pick primary yourself.</p>

      <div class="row" style="align-items:center; margin:4px 0;">
        <label style="width:auto;" data-help="Seek past a black intro before marking the region.">Time (s)</label>
        <input type="number" id="gen-seek" value="0" min="0" step="0.5" style="width:5em;" disabled />
        <button id="gen-seek-btn" type="button" disabled>Frame</button>
        <button id="gen-seek-plus" type="button" disabled>+1s</button>
        <button id="gen-seek-plus5" type="button" disabled>+5s</button>
      </div>
      <div id="roi-canvas-wrap">
        <canvas id="roi-canvas"></canvas>
      </div>
      <div class="path-label" id="gen-roi-label">No region marked</div>
      <div class="row" style="align-items:center; margin-top:6px;">
        <button id="gen-roi2-toggle" type="button"
          data-help="Zone 2: primary contact (e.g. nipples). Tf/Tj distance + contact vibration use tip→this zone (and Zone 3+). Also via Shift+drag.">Zone 2 (contact)</button>
        <button id="gen-target-add" type="button"
          data-help="Zone 3+: extra Tf/Tj contact (magenta). Stroke = min distance tip → all contact zones. Defaults to fixed.">+ Zone 3+ (contact)</button>
        <label style="width:auto; margin:0;" data-help="Body-part class applied to the next Zone 3+ mark (e.g. mouth, nipples).">Zone 3+ class</label>
        <select id="gen-target-class" style="min-width:7em;">
          <option value="">(any)</option>
        </select>
        <button id="gen-mask-add" type="button"
          data-help="Soft-exclude mask (dashed gray). Punched out of camera/grid feature masks — does not drive the stroke.">+ Mask</button>
        <button id="gen-extras-clear" type="button"
          data-help="Clear all extra targets and soft masks (keeps Zone 1/Zone 2).">Clear extras</button>
        <span class="hint" id="gen-roi2-hint" style="margin:0">Tf/Tj: tip + contact zones. Extras optional.</span>
      </div>
      <div class="path-label" id="gen-roi2-label">No 2nd region marked</div>
      <div class="path-label" id="gen-extras-label" style="display:none;"></div>
      <div class="row" style="align-items:center; flex-wrap:wrap; gap:8px; margin:6px 0;">
        <label style="width:auto;" data-help="Zone 1 class — tip / tracked part. Prefer glans for Tf/Tj; whole penis also works — distance uses the end toward the contact zone.">Zone 1 (tip)</label>
        <select id="gen-region-class" style="min-width:8em;">
          <option value="">(any)</option>
        </select>
        <label style="width:auto;" data-help="Zone 2 class — contact target (e.g. nipples). Contact vibration fires when tip approaches this zone.">Zone 2 (contact)</label>
        <select id="gen-region-class2" style="min-width:8em;">
          <option value="">(any)</option>
        </select>
        <label class="checkbox-row" style="margin:0;"
          data-help="Keep Zone 2 at the marked box (static). Off = track the partner (default when contact vibration is on — scene/camera motion stays in sync). On only when the contact target barely moves.">
          <input type="checkbox" id="gen-roi2-fixed" /> Fix Zone 2 (static)
        </label>
      </div>
      <p class="hint" id="gen-pipeline-auto" style="margin:4px 0 8px 0;"></p>
    </section>

    <section class="gen-step-panel" id="gen-step-motion" data-step="3" hidden>
      <h3 class="gen-step-title">3 · Motion type</h3>
      <div class="row" style="align-items:center;">
        <label style="width:auto;" data-help="Default = classic stroke motion. Soft tissue filters ringing. Autotune = detrend+bandpass+speed cap (FunGen/Flow-inspired). Tf/Tj needs two regions.">Motion type</label>
        <select id="gen-profile">
          <option value="standard">Stroke motion (default)</option>
          <option value="weich">Soft tissue (rings)</option>
          <option value="autotune">Autotune (detrend + bandpass + speed)</option>
          <option value="tf">Tf/Tj (distance + suction)</option>
        </select>
      </div>
      <p class="hint" id="gen-profile-hint" style="margin:0 0 10px 0;">
        CSRT tracks the marked region(s). Optional Autotune / audio check live under Advanced in step 4.
        Tf/Tj needs a 2nd region (step 2).
      </p>
      <p class="hint" id="gen-tftj-hint" style="display:none; margin:0 0 6px 0;">
        Zones: (1) tip — glans or whole penis, tracked; (2) primary contact e.g. nipples;
        (3+) optional extra contacts. Nearest contact drives stroke + vibration.
        Whole-penis mark is OK — distance uses the tip end toward the partner.
        Contact vibration on → Zone 2 tracked by default (not a frozen box).
        Zone 2 is always required for Tf/Tj (two markers: tip + partner).
      </p>
      <div id="gen-contact-vibration-wrap">
        <div class="checkbox-row" id="gen-contact-vibration-row">
          <input type="checkbox" id="gen-contact-vibration" checked />
          <label for="gen-contact-vibration"
            data-help="Extra vibration on deep strokes (high position). For Tf/Tj with a contact zone, strength follows tip→partner distance; for Normal/Autotune it follows stroke depth. On by default — turn off anytime.">Contact vibration (on by default)</label>
        </div>
        <div id="gen-contact-vibration-opts" style="display:none; margin:4px 0 10px 22px;">
          <div class="field-row" style="align-items:center;">
            <label style="width:auto;" data-help="Lower = engages earlier (wider contact window). Higher = deep only (near peak position). Default 0.75 = top quarter of the video signal.">Sensitivity</label>
            <input type="range" id="gen-contact-span" min="40" max="95" step="5" value="75" style="flex:1;" />
            <span class="hint" id="gen-contact-span-label" style="margin:0; min-width:7em;">deep only</span>
          </div>
          <div class="field-row" style="align-items:center;">
            <label style="width:auto;" data-help="linear = 1:1. soft = gentle onset (t²) — closer to contact feel. peak = stronger peak (√t).">Curve</label>
            <select id="gen-contact-curve">
              <option value="linear">Linear</option>
              <option value="soft" selected>Soft onset (contact-like)</option>
              <option value="peak">Stronger peak</option>
            </select>
          </div>
        </div>
      </div>

      <div class="row" style="align-items:center;">
        <button id="gen-suggest-profile" disabled
          data-help="Compares the motion signature to saved scenes first, optionally to a local AI server. Suggestion only — nothing is applied automatically.">Suggest profile</button>
        <span class="hint" id="gen-suggest-status" style="margin:0"></span>
      </div>
      <div class="row" style="align-items:center;">
        <input type="text" id="gen-scene-label" placeholder="Name for this scene (optional)" style="flex:1;" />
        <button id="gen-label-scene" disabled
          data-help="Saves the motion signature under this name. Similar videos later get this profile as a suggestion (classic measurement, no AI).">Remember scene</button>
      </div>
    </section>

    <section class="gen-step-panel" id="gen-step-run" data-step="4" hidden>
      <h3 class="gen-step-title">4 · Generate</h3>
      <details id="gen-advanced" style="margin:6px 0 10px 0;">
        <summary style="cursor:pointer;">Advanced settings</summary>
        <div style="margin-top:8px;">
          <div class="opt-group">Tracking</div>
          <div class="checkbox-row"><input type="checkbox" id="gen-invert" /><label for="gen-invert"
            data-help="Inverts motion direction (polarity). Often the FunGen difference — not a tracking bug.">Invert motion direction</label></div>
          <div class="checkbox-row"><input type="checkbox" id="gen-camcomp" checked /><label for="gen-camcomp"
            data-help="Compensates camera pans using background features. Recommended for moving camera.">Camera motion compensation</label></div>
          <div class="checkbox-row"><input type="checkbox" id="gen-scenecut" checked /><label for="gen-scenecut"
            data-help="Detects hard cuts and re-anchors the tracker afterward.">Scene-cut detection</label></div>
          <div class="row" style="align-items:center;">
            <label style="width:auto;" data-help="Product tracking is CSRT on the Go path (portable: no Python). Research backends (flow, grid_lk, fusion) stay CLI --backend only — measured quality first.">Tracking method</label>
            <select id="gen-backend">
              <option value="csrt" selected>CSRT (standard, Go path)</option>
            </select>
          </div>
          <p class="hint" id="gen-backend-hint" style="margin:0 0 6px 0;">CSRT only in the GUI. Weaker/faster research methods are CLI-only so Play/Generate stay dependency-light.</p>

          <div class="opt-group">Signal &amp; quality</div>
          <div class="checkbox-row"><input type="checkbox" id="gen-dynrange" checked /><label for="gen-dynrange"
            data-help="Smoothly lifts weak sections to usable strength.">Sliding dynamics</label></div>
          <div class="checkbox-row"><input type="checkbox" id="gen-retry" checked /><label for="gen-retry"
            data-help="Automatically retries with other signal parameters when quality is poor.">Auto-Retry</label></div>
          <div class="checkbox-row"><input type="checkbox" id="gen-ai-quality" /><label for="gen-ai-quality"
            data-help="Optionally asks a local AI server for a second opinion. Does not change the Quality Doctor score.">AI second opinion on quality</label></div>
          <div class="checkbox-row"><input type="checkbox" id="gen-audio-check" /><label for="gen-audio-check"
            data-help="Compares script tempo to the audio track. Needs ffmpeg (portable release or Settings → Install video tools). Classic; does not change Quality Doctor score.">Check script tempo against audio</label></div>
          <div class="checkbox-row"><input type="checkbox" id="gen-auto-ozone" /><label for="gen-auto-ozone"
            data-help="Suggests O-markers in the last eighth (highest mean position) only when the ending is clearly high. Classic from signal, no AI model.">Suggest O-markers automatically</label></div>

          <div class="opt-group">Keyframes</div>
          <div class="field-row"><label data-help="Both axes are tracked; Auto picks the larger span. Force only when clearly wrong.">Motion axis</label>
            <select id="gen-axis">
              <option value="" selected>Automatic (recommended)</option>
              <option value="x">Force horizontal</option>
              <option value="y">Force vertical</option>
            </select>
          </div>
          <div class="checkbox-row"><input type="checkbox" id="gen-adaptive" checked /><label for="gen-adaptive"
            data-help="Adds extra keyframes for asymmetric motion.">Adaptive Keyframes</label></div>
          <div class="checkbox-row"><input type="checkbox" id="gen-perscene" /><label for="gen-perscene"
            data-help="Re-searches region after each cut. Better for heavily edited material, slower.">Re-find region after each cut</label></div>
          <div class="field-row"><label data-help="Signal smoothing window width in frames. Larger = calmer but slower.">Smoothing window</label><input type="number" id="gen-smooth" value="11" /></div>
          <div class="field-row"><label data-help="Minimum spacing between keyframes in milliseconds.">Min keyframe spacing (ms)</label><input type="number" id="gen-peakdist" value="150" /></div>
          <div class="field-row"><label data-help="Ramer–Douglas–Peucker tolerance for thinning. 0 = off.">RDP tolerance (0 = off)</label><input type="number" id="gen-rdp" value="0" step="0.5" min="0" /></div>
          <div class="field-row"><label data-help="Max position change per second (0–100 scale). 0 = off. Protects the device. Autotune sets 400.">Max speed (0 = off)</label><input type="number" id="gen-maxspeed" value="0" step="50" min="0" /></div>
          <div class="field-row" style="display:none;"><label>Flow downscale</label><input type="number" id="gen-flow-downscale" value="0" step="0.1" min="0" max="1" /></div>
        </div>
      </details>

      <div class="row">
        <button id="gen-generate" class="primary" disabled>Generate Funscript</button>
        <button id="gen-cancel" type="button" disabled>Cancel</button>
      </div>
      <div id="gen-progress-wrap" style="display:none; margin-top:8px;">
        <div style="height:10px; border-radius:5px; background:rgba(255,255,255,0.10); overflow:hidden;">
          <div id="gen-progress-bar" style="height:100%; width:0%; background:linear-gradient(90deg,var(--accent),var(--teal));
               transition:width .2s linear;"></div>
        </div>
        <div id="gen-progress-text" class="hint" style="margin-top:4px;"></div>
      </div>
      <pre id="gen-log" class="run-log" aria-label="Generation progress log"></pre>
      <p class="hint">
        Classic CV tracking on <b>CSRT</b> — one strong product path.
        Go CSRT when OpenCV is linked; otherwise Python CSRT (Windows today).
      </p>
    </section>

    <section class="gen-step-panel" id="gen-step-result" data-step="5" hidden>
      <h3 class="gen-step-title">5 · Review</h3>
      <div id="gen-feedback" style="display:none; margin-top:4px; padding:10px;
           border:1px solid var(--border); border-radius:4px;">
        <div style="margin-bottom:6px;">Was the result usable? Your rating helps
          tune quality scoring on real material.</div>
        <div class="row">
          <button data-verdict="brauchbar" type="button">usable</button>
          <button data-verdict="grenzwertig" type="button">borderline</button>
          <button data-verdict="unbrauchbar" type="button">unusable</button>
        </div>
        <input type="text" id="gen-fb-comment" placeholder="Comment (optional) - e.g. what did not fit"
               style="width:100%; margin-top:8px;" />
        <div id="gen-fb-status" class="hint" style="margin-top:6px;"></div>
      </div>
      <div id="gen-quality" style="display:none; margin-top:8px; padding:8px; border-radius:4px;"></div>
    </section>
  `;

  wireDataHelp(root);

  const el = id => root.querySelector(id);
  const canvas = el('#roi-canvas');
  const ctx = canvas.getContext('2d');

  let videoPath = null;
  let img = new Image();
  let nativeW = 0, nativeH = 0;
  let roi = null; // {x,y,w,h} in videopixeln
  let roi2 = null; // zweite Region für Tf/Tj (distance + suction)
  let candidates = []; // TFTJ 4b: [{x,y,w,h,score,index}, ...] dashed until pick
  let extraTargets = []; // additional fixed Tf/Tj anchors (min-distance)
  let maskRois = []; // soft-exclude boxes
  let roi2Mode = false; // Knopf „2. Region“ aktiv
  let markMode = null; // null | 'target' | 'mask'
  let dragging = false, draggingSecond = false, startX = 0, startY = 0, curX = 0, curY = 0;
  let seekSec = 0;
  let generating = false;
  let lastOutputPath = null;

  const DISPLAY_W = 560;

  const ROI1_STROKE = '#3dccc0';
  const ROI1_FILL = 'rgba(61,204,192,0.16)';
  const ROI2_STROKE = '#f2b03d';
  const ROI2_FILL = 'rgba(242,176,61,0.18)';
  const TARGET_STROKE = '#e070a0';
  const TARGET_FILL = 'rgba(224,112,160,0.16)';
  const MASK_STROKE = 'rgba(180,180,190,0.85)';
  const MASK_FILL = 'rgba(120,120,130,0.12)';

  function isTfTj() {
    const p = el('#gen-profile').value;
    return p === 'tf' || p === 'tj';
  }

  // KI-Regionssuche (ai_roi.py, lokales ONNX-Modell) ist optional - ohne
  // installiertes onnxruntime oder ohne Modelldatei bleibt es bei der
  // klassischen Rhythmus-Heuristik (auto_roi.py). Einmal beim Öffnen des
  // Tabs geprüft (kostet einen Python-Start), nicht bei jedem videoladen.
  function refreshAIRoiAvailability() {
    CheckAIRoiAvailable().then(available => {
      const checkbox = el('#gen-ai-roi');
      checkbox.disabled = !available;
      el('#gen-autoroi-hint').textContent = available
        ? 'Enabling “AI detection” uses a local ONNX detector instead of the '
          + 'rhythm heuristic. You can still correct the region by hand afterward.'
        : 'Analyzes motion in the video (classic, no AI model) — you can still '
          + 'correct the region by hand. AI detection: no local ONNX model '
          + 'found (Settings → AI model path, or default folder).';
    }).catch(() => {});
  }
  refreshAIRoiAvailability();
  window.addEventListener('samn-ai-roi-refresh', refreshAIRoiAvailability);

  // Audio-Tempo-Prüfung braucht nur ffmpeg auf dem PATH (Go-native post-hoc
  // oder Python bei PreferPython). Gleiches Muster wie oben: einmal beim
  // Öffnen des Tabs geprüft, Checkbox ausgegraut statt bei jedem Versuch
  // mit "nicht möglich" zu scheitern.
  CheckAudioCheckAvailable().then(available => {
    const checkbox = el('#gen-audio-check');
    checkbox.disabled = !available;
    if (!available) {
      checkbox.checked = false;
      checkbox.title = 'ffmpeg not found — use portable release or Settings → Install video tools';
    } else {
      // Part of the normal workflow when ffmpeg is present (docs/AUDIO_WORKFLOW.md).
      checkbox.checked = true;
    }
  }).catch(() => {});

  function setRoi2Mode(on) {
    roi2Mode = !!on;
    if (roi2Mode) markMode = null;
    const btn = el('#gen-roi2-toggle');
    btn.style.outline = roi2Mode ? '2px solid #f2b03d' : '';
    btn.style.background = roi2Mode ? 'rgba(242,176,61,0.22)' : '';
    syncMarkModeButtons();
  }

  function setMarkMode(mode) {
    markMode = mode || null;
    if (markMode) roi2Mode = false;
    const btn = el('#gen-roi2-toggle');
    btn.style.outline = '';
    btn.style.background = '';
    syncMarkModeButtons();
  }

  function syncMarkModeButtons() {
    const tBtn = el('#gen-target-add');
    const mBtn = el('#gen-mask-add');
    if (tBtn) {
      tBtn.style.outline = markMode === 'target' ? '2px solid #e070a0' : '';
      tBtn.style.background = markMode === 'target' ? 'rgba(224,112,160,0.22)' : '';
    }
    if (mBtn) {
      mBtn.style.outline = markMode === 'mask' ? '2px solid rgba(180,180,190,0.9)' : '';
      mBtn.style.background = markMode === 'mask' ? 'rgba(120,120,130,0.25)' : '';
    }
  }

  function updateRoiLabels() {
    el('#gen-roi-label').textContent = roi
      ? `Region: x=${roi.x} y=${roi.y} w=${roi.w} h=${roi.h} (video pixels)`
      : 'No region marked';
    el('#gen-roi2-label').textContent = roi2
      ? `2nd region: x=${roi2.x} y=${roi2.y} w=${roi2.w} h=${roi2.h} (video pixels, gold)`
      : 'No 2nd region marked';
    const extras = el('#gen-extras-label');
    if (extras) {
      const parts = [];
      if (extraTargets.length) {
        const named = extraTargets.map((t, i) => t.class ? t.class : `#${i + 1}`).join(', ');
        parts.push(`${extraTargets.length} Zone 3+ (${named}, magenta, fixed)`);
      }
      if (maskRois.length) {
        parts.push(`${maskRois.length} soft mask${maskRois.length === 1 ? '' : 's'} (dashed)`);
      }
      extras.style.display = parts.length ? 'block' : 'none';
      extras.textContent = parts.length ? parts.join(' · ') : '';
    }

    // Zwei-Punkt-Messung (2. Region gesetzt): Per-Scene-ROI abschalten —
    // der Zwei-Punkt-Pfad sucht die Region nicht neu.
    const twoPoint = !!roi2;
    const perScene = el('#gen-perscene');
    perScene.disabled = twoPoint;
    if (twoPoint) perScene.checked = false;
    perScene.title = twoPoint
      ? 'Not available for two-point measurement (2nd region set) — region is not re-searched there.'
      : '';
    // Product GUI only exposes CSRT (Go path). Research backends are CLI-only.
    if (el('#gen-backend').value !== 'csrt') {
      el('#gen-backend').value = 'csrt';
    }
    updateGenerateEnabled();
  }

  // Product path always needs a marked region (CSRT / Tf/Tj).
  function backendNeedsRoi() {
    return true;
  }

  function contactVibrationOn() {
    return !!el('#gen-contact-vibration')?.checked;
  }

  // TFTJ: Zone 2 always required for Tf/Tj (two markers). Contact vib on →
  // partner is tracked by default (Fix Zone 2 off) — see syncRoi2FixedDefault.
  function regionReadyForGenerate() {
    if (!videoPath) return false;
    if (backendNeedsRoi() && !roi) return false;
    if (isTfTj() && !roi2) return false;
    return true;
  }

  // Default Fix Zone 2 = off when vib needs a tracked partner (step 3).
  // Respect user override via dataset.userTouched.
  function syncRoi2FixedDefault() {
    const fix = el('#gen-roi2-fixed');
    if (!fix || fix.dataset.userTouched) return;
    if (isTfTj() && contactVibrationOn()) {
      fix.checked = false;
    }
  }

  // Progressive steps: next panel appears only when the previous action is done.
  function syncWorkflowSteps() {
    const hasVideo = !!videoPath;
    const hasRoi1 = !!roi;
    const canRun = regionReadyForGenerate();
    const hasResult = !!lastOutputPath;

    const showRegion = hasVideo;
    const showMotion = hasRoi1;
    const showRun = canRun || generating;
    const showResult = hasResult;

    const panel = (id, on) => {
      const node = el(id);
      if (!node) return;
      node.hidden = !on;
    };
    panel('#gen-step-region', showRegion);
    panel('#gen-step-motion', showMotion);
    panel('#gen-step-run', showRun);
    panel('#gen-step-result', showResult);

    let current = 1;
    if (showResult) current = 5;
    else if (showRun || generating) current = 4;
    else if (showMotion) current = 3;
    else if (showRegion) current = 2;

    root.querySelectorAll('.gen-step-item').forEach(item => {
      const n = parseInt(item.dataset.step, 10);
      item.classList.toggle('is-current', n === current);
      item.classList.toggle('is-done', n < current);
    });

    const prompt = el('#gen-step-prompt');
    if (!prompt) return;
    if (!hasVideo) {
      prompt.textContent = 'Start here: choose a video. The next step appears when this one is done.';
    } else if (!hasRoi1) {
      prompt.textContent = 'Step 2: mark a region on the frame (drag) or find one automatically.';
    } else if (isTfTj() && !roi2) {
      prompt.textContent = 'Tf/Tj: mark Zone 2 / partner (two markers), then Generate appears.';
    } else if (!canRun && !generating) {
      prompt.textContent = 'Step 3: pick motion type if needed — Generate unlocks when regions are ready.';
    } else if (generating) {
      prompt.textContent = 'Step 4: generating… you can Cancel if needed.';
    } else if (!hasResult) {
      prompt.textContent = 'Step 4: Generate Funscript (Advanced optional). Review appears after a successful run.';
    } else {
      prompt.textContent = 'Step 5: rate Signal Quality / usability — script is also in Play.';
    }
  }

  function updateGenerateEnabled() {
    // Ohne video kein Ziel zum Generieren.
    if (!regionReadyForGenerate()) {
      el('#gen-generate').disabled = true;
      syncWorkflowSteps();
      return;
    }
    el('#gen-generate').disabled = generating;
    syncWorkflowSteps();
  }

  let contactUserOverride = false;

  function updateContactVibrationOpts() {
    const on = el('#gen-contact-vibration').checked;
    el('#gen-contact-vibration-opts').style.display = on ? 'block' : 'none';
    syncRoi2FixedDefault();
    updateGenerateEnabled();
  }

  function updateContactSpanLabel() {
    const v = parseInt(el('#gen-contact-span').value, 10) || 75;
    const label = el('#gen-contact-span-label');
    if (v <= 50) label.textContent = 'earlier';
    else if (v >= 85) label.textContent = 'deep only';
    else label.textContent = (v / 100).toFixed(2);
  }

  function updateProfileUi() {
    const tftj = isTfTj();
    el('#gen-tftj-hint').style.display = tftj ? 'block' : 'none';
    // Contact vib is available on every profile (Normal/Auto + Tf/Tj); always shown.
    el('#gen-contact-vibration-wrap').style.display = 'block';
    el('#gen-contact-vibration-row').style.display = 'flex';
    // Default on for all profiles unless the user explicitly turned it off.
    if (!contactUserOverride) {
      el('#gen-contact-vibration').checked = true;
      if (!el('#gen-contact-curve').dataset.userTouched) {
        el('#gen-contact-curve').value = 'soft';
      }
    }
    syncRoi2FixedDefault();
    updateContactVibrationOpts();
    if (tftj) setRoi2Mode(true);
    updateGenerateEnabled();
    if (tftj && videoPath) {
      if (roi2) {
        el('#gen-status').textContent = contactVibrationOn()
          ? 'Tf/Tj: both markers set — partner tracked unless “Fix Zone 2” is on.'
          : 'Tf/Tj: both markers set — ready (contact vib off).';
      } else {
        el('#gen-status').textContent =
          'Tf/Tj needs two markers: tip (Zone 1) + partner (Zone 2). Shift+drag or “Zone 2”.';
      }
    }
  }

  function drawNativeRect(r, stroke, fill, dashed) {
    if (!r || !nativeW || !nativeH) return;
    const scaleX = canvas.width / nativeW, scaleY = canvas.height / nativeH;
    const dx0 = r.x * scaleX, dy0 = r.y * scaleY, dw = r.w * scaleX, dh = r.h * scaleY;
    ctx.save();
    if (dashed) ctx.setLineDash([6, 4]);
    ctx.strokeStyle = stroke;
    ctx.lineWidth = 2;
    ctx.strokeRect(dx0, dy0, dw, dh);
    ctx.fillStyle = fill;
    ctx.fillRect(dx0, dy0, dw, dh);
    ctx.restore();
  }

  function drawCandidate(c) {
    if (!c || !nativeW || !nativeH) return;
    const scaleX = canvas.width / nativeW, scaleY = canvas.height / nativeH;
    const dx0 = c.x * scaleX, dy0 = c.y * scaleY, dw = c.w * scaleX, dh = c.h * scaleY;
    ctx.save();
    ctx.setLineDash([5, 4]);
    ctx.strokeStyle = 'rgba(120, 200, 255, 0.95)';
    ctx.lineWidth = 2;
    ctx.strokeRect(dx0, dy0, dw, dh);
    ctx.fillStyle = 'rgba(120, 200, 255, 0.12)';
    ctx.fillRect(dx0, dy0, dw, dh);
    const label = '#' + (c.index || '?');
    ctx.setLineDash([]);
    ctx.font = '600 13px system-ui, sans-serif';
    ctx.fillStyle = 'rgba(10, 20, 30, 0.75)';
    ctx.fillRect(dx0 + 2, dy0 + 2, ctx.measureText(label).width + 8, 18);
    ctx.fillStyle = '#dff3ff';
    ctx.fillText(label, dx0 + 6, dy0 + 15);
    ctx.restore();
  }

  function hitCandidate(canvasX, canvasY) {
    if (!candidates.length || !nativeW || !nativeH) return null;
    const scaleX = canvas.width / nativeW, scaleY = canvas.height / nativeH;
    const vx = canvasX / scaleX, vy = canvasY / scaleY;
    // Prefer smallest containing box (most specific).
    let best = null, bestArea = Infinity;
    for (const c of candidates) {
      if (vx >= c.x && vx <= c.x + c.w && vy >= c.y && vy <= c.y + c.h) {
        const area = c.w * c.h;
        if (area < bestArea) {
          best = c;
          bestArea = area;
        }
      }
    }
    return best;
  }

  function pickCandidate(c) {
    if (!c) return;
    roi = { x: c.x, y: c.y, w: c.w, h: c.h };
    // Never auto-fill Zone 2 from candidates (issue #8 / TFTJ 4b).
    updateRoiLabels();
    updateProfileUi();
    updateGenerateEnabled();
    el('#gen-roi-label').textContent =
      `Region: x=${roi.x} y=${roi.y} w=${roi.w} h=${roi.h} (video pixels, candidate #${c.index})`;
    el('#gen-status').textContent = isTfTj() && !roi2
      ? `Primary set from candidate #${c.index} — Tf/Tj still needs Zone 2.`
      : `Primary set from candidate #${c.index} — correct by hand if needed.`;
    redraw();
  }

  function drawDragRect(stroke, fill, dashed) {
    const dx0 = Math.min(startX, curX), dy0 = Math.min(startY, curY);
    const dw = Math.abs(curX - startX), dh = Math.abs(curY - startY);
    ctx.save();
    if (dashed) ctx.setLineDash([6, 4]);
    ctx.strokeStyle = stroke;
    ctx.lineWidth = 2;
    ctx.strokeRect(dx0, dy0, dw, dh);
    ctx.fillStyle = fill;
    ctx.fillRect(dx0, dy0, dw, dh);
    ctx.restore();
  }

  function redraw() {
    ctx.clearRect(0, 0, canvas.width, canvas.height);
    if (img.src) ctx.drawImage(img, 0, 0, canvas.width, canvas.height);
    for (const c of candidates) drawCandidate(c);
    const dragRoi2 = dragging && draggingSecond && !markMode;
    const dragRoi1 = dragging && !draggingSecond && !markMode;
    if (roi && !dragRoi1) drawNativeRect(roi, ROI1_STROKE, ROI1_FILL);
    if (roi2 && !dragRoi2) drawNativeRect(roi2, ROI2_STROKE, ROI2_FILL);
    for (const t of extraTargets) drawNativeRect(t, TARGET_STROKE, TARGET_FILL);
    for (const m of maskRois) drawNativeRect(m, MASK_STROKE, MASK_FILL, true);
    if (dragging) {
      if (markMode === 'mask') drawDragRect(MASK_STROKE, MASK_FILL, true);
      else if (markMode === 'target') drawDragRect(TARGET_STROKE, TARGET_FILL);
      else if (draggingSecond) drawDragRect(ROI2_STROKE, ROI2_FILL);
      else drawDragRect(ROI1_STROKE, ROI1_FILL);
    }
  }

  canvas.addEventListener('mousedown', e => {
    const r = canvas.getBoundingClientRect();
    startX = curX = e.clientX - r.left;
    startY = curY = e.clientY - r.top;
    dragging = true;
    draggingSecond = !markMode && (roi2Mode || e.shiftKey);
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
    const mode = markMode;
    draggingSecond = false;
    const w = Math.abs(curX - startX), h = Math.abs(curY - startY);
    // Tiny press: pick a motion candidate if shown (TFTJ 4b).
    if (w < 8 && h < 8) {
      if (!mode && !wasSecond && candidates.length) {
        const hit = hitCandidate(startX, startY);
        if (hit) {
          pickCandidate(hit);
          return;
        }
      }
      redraw();
      return;
    }
    const scaleX = nativeW / canvas.width, scaleY = nativeH / canvas.height;
    const x0 = Math.min(startX, curX), y0 = Math.min(startY, curY);
    const box = {
      x: Math.round(x0 * scaleX), y: Math.round(y0 * scaleY),
      w: Math.round(w * scaleX), h: Math.round(h * scaleY),
    };
    if (mode === 'target') {
      const cls = el('#gen-target-class')?.value || '';
      extraTargets.push({ ...box, fixed: true, class: cls });
      setMarkMode(null);
      if (!isTfTj()) {
        el('#gen-profile').value = 'tf';
        updateProfileUi();
      }
      const tag = cls ? ` (${cls})` : '';
      el('#gen-status').textContent = `Zone 3+ #${extraTargets.length}${tag} added (min-distance partner).`;
    } else if (mode === 'mask') {
      maskRois.push(box);
      setMarkMode(null);
      el('#gen-status').textContent = `Soft mask #${maskRois.length} added (feature exclude).`;
    } else if (wasSecond) {
      roi2 = box;
      setRoi2Mode(false);
      // Eine 2. Region ergibt nur im Tf/Tj-Modus (Abstand + Sog) einen Sinn
      // - kein anderes Verfahren wertet sie aus (siehe generate_funscript.py).
      // Automatisch erkennen statt den Nutzer zusätzlich noch die Dropdown
      // umstellen zu lassen: das Markieren der 2. Region IST die Auswahl.
      if (!isTfTj()) {
        el('#gen-profile').value = 'tf';
        updateProfileUi();
      }
    } else {
      roi = box;
      if (isTfTj() && !roi2) setRoi2Mode(true);
    }
    updateRoiLabels();
    updateGenerateEnabled();
    autoApplyPipeline();
    if (isTfTj() && videoPath && !roi2) {
      el('#gen-status').textContent =
        'First marker set — mark Zone 2 / partner (Shift+drag or “Zone 2”).';
    }
    redraw();
  });

  async function autoApplyPipeline() {
    const w = roi?.w || 0, h = roi?.h || 0;
    const w2 = roi2?.w || 0, h2 = roi2?.h || 0;
    try {
      const s = await SuggestPipeline(w, h, w2, h2);
      if (!s) return;
      // Manual backend/profile choices survive ROI redraws; only auto-fill
      // when the user has not touched the dropdowns yet.
      const backendTouched = el('#gen-backend').dataset.userTouched === '1';
      const profileTouched = el('#gen-profile').dataset.userTouched === '1';
      if (s.Backend && !backendTouched) el('#gen-backend').value = s.Backend;
      if (s.Profile && !profileTouched) {
        el('#gen-profile').value = s.Profile;
        updateProfileUi();
      }
      const pipe = el('#gen-pipeline-auto');
      if (pipe) {
        pipe.textContent = (s.Reason || '') + (s.GoPath ? ' · Go path' : ' · Python path');
      }
      updateGenerateEnabled();
    } catch (_) { /* ignore */ }
  }

  // Fallengelassenes video Apply. Teilt sich den Ladeweg mit der
  // Dateiauswahl, damit beide Wege garantiert dasselbe tun.
  window.addEventListener('drop:video', e => loadVideo(e.detail.path, e.detail.extraCount || 0));

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

  async function loadVideo(path, extraCount = 0) {
    videoPath = path;
    seekSec = 0;
    el('#gen-seek').value = '0';
    el('#gen-seek').disabled = false;
    el('#gen-seek-btn').disabled = false;
    el('#gen-seek-plus').disabled = false;
    el('#gen-seek-plus5').disabled = false;
    el('#gen-video-path').textContent = path.split(/[\\/]/).pop();
    el('#gen-status').textContent = 'Loading preview frame…';
    roi = null;
    roi2 = null;
    extraTargets = [];
    maskRois = [];
    setMarkMode(null);
    // Neues video: Pipeline-Vorschläge wieder erlauben.
    delete el('#gen-backend').dataset.userTouched;
    delete el('#gen-profile').dataset.userTouched;
    setRoi2Mode(isTfTj());
    el('#gen-generate').disabled = true;
    updateRoiLabels();
    // Stapelverarbeitung mehrerer videos gibt es noch nicht - vorher wurden
    // more dropped videos einfach stillschweigend verworfen, ohne dass
    // sichtbar war, dass überhaupt mehr als eins ankam.
    const batchNote = extraCount > 0
      ? ` (${extraCount} more video${extraCount === 1 ? '' : 's'} ignored — batch processing not available yet)`
      : '';
    try {
      await showFrame(path, 0);
      candidates = [];
      el('#gen-autoroi').disabled = false;
      el('#gen-candidates').disabled = false;
      el('#gen-suggest-profile').disabled = false;
      el('#gen-label-scene').disabled = false;
      el('#gen-suggest-status').textContent = '';
      el('#gen-status').textContent = (isTfTj()
        ? 'Tf/Tj: draw tip (Zone 1), then Shift+drag or “Zone 2” for contact (2nd region / nipples…). Or show motion candidates and click primary.'
        : 'Find region / show motion candidates / mark by hand. Seek time if the start is black.') + batchNote;
      lastOutputPath = null;
      el('#gen-feedback').style.display = 'none';
      el('#gen-quality').style.display = 'none';
      syncWorkflowSteps();
      // Soft-Vorschlag: Profil nur anzeigen, nie automatisch Apply.
      SuggestProfile(path).then(result => {
        if (!result || !videoPath || videoPath !== path) return;
        const status = el('#gen-suggest-status');
        const via = result.via || 'signature';
        const label = result.label === 'tj' ? 'tf' : result.label;
        if (label && ['standard', 'weich', 'autotune', 'tf'].includes(label)) {
          status.textContent = `Suggestion: “${label}” (${via}) — use “Suggest profile” to apply.`;
        }
      }).catch(() => {});
    } catch (err) {
      uiError('Load video: ' + err, el('#gen-status'));
    }
  }

  async function showFrame(path, sec) {
    const preview = sec > 0 ? await LoadFrameAt(path, sec) : await LoadFirstFrame(path);
    nativeW = preview.width; nativeH = preview.height;
    const displayH = Math.round(DISPLAY_W * nativeH / nativeW);
    canvas.width = DISPLAY_W; canvas.height = displayH;
    img.onload = redraw;
    img.src = 'data:image/png;base64,' + preview.pngBase64;
  }

  async function seekTo(sec) {
    if (!videoPath) return;
    seekSec = Math.max(0, sec);
    el('#gen-seek').value = String(seekSec);
    el('#gen-status').textContent = `Loading frame at ${seekSec}s…`;
    try {
      await showFrame(videoPath, seekSec);
      el('#gen-status').textContent = `Frame at ${seekSec}s — mark region.`;
    } catch (err) {
      uiError('Seek failed: ' + err, el('#gen-status'));
    }
  }

  async function checkDeps() {
    try {
      await CheckGeneratorDependencies();
      uiInfo('Python and required packages are available.', el('#gen-status'));
    } catch (err) {
      uiError('Dependencies: ' + err, el('#gen-status'));
    }
  }

  async function generate() {
    if (!videoPath || (backendNeedsRoi() && !roi)) return;
    if (isTfTj() && !roi2) {
      el('#gen-status').textContent =
        'Tf/Tj needs two markers: Zone 1 (tip) + Zone 2 (partner).';
      return;
    }

    // Vorhandenes Skript nicht kommentarlos überschreiben - der Nutzer
    // könnte ein von Hand erstelltes oder heruntergeladenes Skript neben
    // dem video liegen haben.
    let overwrite = false;
    try {
      if (await ScriptExistsForVideo(videoPath)) {
        const target = videoPath.replace(/\.[^.\\/]+$/, '') + '.funscript (and .samn)';
        if (!confirm(`A script already exists:\n${target}\n\nOverwrite it?`)) {
          return;
        }
        overwrite = true;
      }
    } catch (err) {
      // Prüfung fehlgeschlagen - lieber nicht überschreiben, das Backend
      // lehnt dann ohnehin ab und meldet es sauber.
    }

    el('#gen-generate').disabled = true;
    el('#gen-cancel').disabled = false;
    generating = true;
    syncWorkflowSteps();
    el('#gen-status').textContent = 'Generating…';
    el('#gen-log').textContent = '';
    {
      const wrap = el('#gen-progress-wrap');
      wrap.style.display = 'block';
      el('#gen-progress-bar').style.width = '0%';
      el('#gen-progress-bar').style.opacity = '1';
      el('#gen-progress-text').textContent = 'Starting…';
      progressStartedAt = Date.now();
    }
    // Ohne markierte Region (flow/region_fusion_auto) dieselbe "keine ROI"-
    // Platzhalter-Region wie die CLI ohne --roi fürs flow-Backend verschickt
    // (0,0,0,0) - beide Backends ignorieren sie ohnehin vollständig.
    const effectiveRoi = roi || { x: 0, y: 0, w: 0, h: 0 };
    const payload = {
      videoPath,
      x: effectiveRoi.x, y: effectiveRoi.y, w: effectiveRoi.w, h: effectiveRoi.h,
      invert: el('#gen-invert').checked,
      smoothWindow: parseInt(el('#gen-smooth').value, 10) || 11,
      minPeakDistanceMs: parseInt(el('#gen-peakdist').value, 10) || 150,
      disableCameraCompensation: !el('#gen-camcomp').checked,
      disableSceneCutDetection: !el('#gen-scenecut').checked,
      perSceneRoi: el('#gen-perscene').checked,
      adaptiveKeyframeError: el('#gen-adaptive').checked ? 6 : 0,
      autoRetry: el('#gen-retry').checked,
      backend: el('#gen-backend').value,
      dynamicRangeMs: el('#gen-dynrange').checked ? 3000 : 0,
      profile: el('#gen-profile').value,
      axis: el('#gen-axis').value,
      rdpTolerance: parseFloat(el('#gen-rdp').value) || 0,
      maxSpeed: parseFloat(el('#gen-maxspeed')?.value) || 0,
      flowDownscale: parseFloat(el('#gen-flow-downscale')?.value) || 0,
      overwrite,
      aiQualityOpinion: el('#gen-ai-quality').checked,
      contactVibration: el('#gen-contact-vibration').checked,
      contactVibrationSpan: el('#gen-contact-vibration').checked
        ? (parseInt(el('#gen-contact-span').value, 10) || 75) / 100
        : 0,
      contactVibrationCurve: el('#gen-contact-vibration').checked
        ? (el('#gen-contact-curve').value || 'linear')
        : '',
      autoOZoneMarker: el('#gen-auto-ozone').checked,
      audioCheck: el('#gen-audio-check').checked,
      startTimeSec: seekSec > 0 ? seekSec : 0,
    };
    if (roi2 && isTfTj()) {
      payload.x2 = roi2.x;
      payload.y2 = roi2.y;
      payload.w2 = roi2.w;
      payload.h2 = roi2.h;
      payload.roi2Fixed = !!el('#gen-roi2-fixed')?.checked;
    } else if (roi2 && !isTfTj()) {
      el('#gen-status').textContent =
        'Note: Zone 2 ignored for Autotune/stroke — tip ROI only. Switch profile to Tf/Tj for distance.';
    }
    payload.regionClass = el('#gen-region-class')?.value || '';
    payload.regionClass2 = el('#gen-region-class2')?.value || '';
    if (extraTargets.length) {
      payload.extraTargets = extraTargets.map(t => ({
        x: t.x, y: t.y, w: t.w, h: t.h,
        fixed: t.fixed !== false,
        class: t.class || '',
      }));
    }
    if (maskRois.length) {
      payload.maskRois = maskRois.map(m => ({ x: m.x, y: m.y, w: m.w, h: m.h }));
    }
    GenerateScript(payload);
  }

  el('#gen-cancel').addEventListener('click', () => {
    CancelGenerate();
    el('#gen-status').textContent = 'Cancel requested…';
  });

  EventsOn('generate:progress', line => {
    el('#gen-status').textContent = line;
    const log = el('#gen-log');
    if (log) {
      log.textContent += line + '\n';
      log.scrollTop = log.scrollHeight;
    }
    const wrap = el('#gen-progress-wrap');
    if (wrap && wrap.style.display === 'none') {
      wrap.style.display = 'block';
      progressStartedAt = Date.now();
    }
    const pipe = el('#gen-pipeline');
    if (!pipe) return;
    const s = String(line);
    if (/Go-Pipeline|Go-native|trackcv/i.test(s) && !/simpletrack|NCC/i.test(s)) {
      pipe.textContent = 'Path: Go CSRT';
    } else if (/PreferSimpletrack|simpletrack|NCC/i.test(s)) {
      pipe.textContent = 'Path: Go simpletrack (experimental)';
    } else if (/Python CSRT|product path/i.test(s)) {
      pipe.textContent = 'Path: Python CSRT';
    } else if (/Fallback auf Python|starte Generierung/i.test(s) && /Python/i.test(s)) {
      pipe.textContent = 'Path: Python';
    } else if (/Fallback auf Python/i.test(s)) {
      pipe.textContent = 'Path: Python';
    }
  });

  // Ergebnis der automatischen Regionssuche Apply - die ROI wird
  // genauso gesetzt, als hätte der Nutzer sie gezogen, und lässt sich
  // danach frei korrigieren.
  EventsOn('generate:autoroi', result => {
    hideProgress();
    el('#gen-autoroi').disabled = false;
    el('#gen-candidates').disabled = false;
    if (result.error) {
      uiError('Automatic region search: ' + result.error, el('#gen-status'));
      return;
    }
    candidates = [];
    roi = { x: result.x, y: result.y, w: result.w, h: result.h };
    const hasRoi2 = result.w2 > 0 && result.h2 > 0;
    if (hasRoi2) {
      roi2 = { x: result.x2, y: result.y2, w: result.w2, h: result.h2 };
      setRoi2Mode(true);
      if (!isTfTj()) el('#gen-profile').value = 'tf';
    }
    updateRoiLabels();
    const via = result.engine === 'ai' ? 'AI detection' : 'classic auto';
    if (roi) {
      el('#gen-roi-label').textContent =
        `Region: x=${roi.x} y=${roi.y} w=${roi.w} h=${roi.h} (video pixels, ${via} found)`;
    }
    if (hasRoi2 && roi2) {
      el('#gen-roi2-label').textContent =
        `2nd region: x=${roi2.x} y=${roi2.y} w=${roi2.w} h=${roi2.h} (video pixels, ${via} — please review)`;
    }
    updateProfileUi();
    updateGenerateEnabled();
    let status = hasRoi2
      ? `Both regions found (${via}) — suggestion, please review/correct.`
      : (isTfTj() && !roi2
        ? `Region found (${via}) — Tf/Tj still needs Zone 2 (second marker).`
        : `Region found (${via}) — correct by hand if needed.`);
    if (result.verifyWarning) {
      status += ' ⚠ ' + result.verifyWarning;
      uiWarn(result.verifyWarning, el('#gen-status'));
    }
    el('#gen-status').textContent = status;
    redraw();
  });
  // Fortschritt: das Backend schickt 0-100, oder -1 wenn die Frame-Anzahl
  // des videos unknown war. In dem Fall wird ein unbestimmter
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
      text.textContent = 'Running… (could not determine video length)';
      return;
    }
    bar.style.opacity = '1';
    bar.style.width = pct + '%';
    const elapsed = (Date.now() - progressStartedAt) / 1000;
    let rest = '';
    if (pct >= 3 && elapsed > 2) {
      const total = elapsed / (pct / 100);
      const remaining = Math.max(0, Math.round(total - elapsed));
      rest = `  ·  ~${remaining < 60 ? remaining + ' s' : Math.round(remaining / 60) + ' min'} left`;
    }
    text.textContent = `${pct} %${rest}`;
  });

  function hideProgress() {
    el('#gen-progress-wrap').style.display = 'none';
    el('#gen-progress-bar').style.width = '0%';
    el('#gen-progress-text').textContent = '';
  }

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
        status.textContent = `Thanks - saved as "${btn.textContent.trim()}".`;
        el('#gen-fb-comment').value = '';
      } catch (err) {
        status.textContent = 'Could not save: ' + err;
      }
    });
  });

  EventsOn('generate:done', result => {
    hideProgress();
    el('#gen-cancel').disabled = true;
    generating = false;
    lastOutputPath = result.path || null;
    el('#gen-fb-status').textContent = '';
    el('#gen-feedback').style.display = lastOutputPath ? 'block' : 'none';
    updateGenerateEnabled();
    syncWorkflowSteps();
    if (result.error) {
      if (result.cancelled) {
        el('#gen-status').textContent = 'Canceled.';
        return;
      }
      el('#gen-status').textContent = 'Failed: ' + result.error;
      return;
    }
    el('#gen-status').textContent = result.samPath
      ? `Done: ${result.path} (+ SAM model)`
      : 'Done: ' + result.path;
    const pipe = el('#gen-pipeline');
    if (pipe) {
      if (result.pipeline === 'go') {
        pipe.textContent = `Path: Go (${result.tracking || 'native'} / ${result.backend || '?'})`;
      } else {
        pipe.textContent = 'Path: Python';
      }
    }
    if (typeof result.oZoneMarkerStartMs === 'number') {
      const s = Math.round(result.oZoneMarkerStartMs / 1000);
      const e = Math.round(result.oZoneMarkerEndMs / 1000);
      el('#gen-status').textContent += ` — O-marker set: ${s}s–${e}s`;
    }

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
      let html = `<b>Signal Quality (Quality Doctor): ${pct}% ${ok ? '(within normal)' : '(review recommended)'}</b>`;
      html += '<br><span style="opacity:0.7;">Technical signal quality — not motion fidelity '
        + '(Does the curve match the video?).</span>';
      if (result.qualityWarnings && result.qualityWarnings.length > 0) {
        html += '<ul style="margin:6px 0 0 18px; padding:0;">' +
          result.qualityWarnings.map(w => `<li>${w}</li>`).join('') + '</ul>';
      }
      if (result.aiOpinionVerdict) {
        html += `<div style="margin-top:8px; padding-top:8px; border-top:1px solid var(--border);">`
          + `<b>AI second opinion: ${result.aiOpinionVerdict}</b>`
          + (result.aiOpinionReason ? `<br>${result.aiOpinionReason}` : '')
          + `</div>`;
      }
      if (result.audioCheckWarnings && result.audioCheckWarnings.length > 0) {
        html += `<div style="margin-top:8px; padding-top:8px; border-top:1px solid var(--border);">`
          + `<b>Audio tempo check:</b>`
          + '<ul style="margin:6px 0 0 18px; padding:0;">'
          + result.audioCheckWarnings.map(w => `<li>${w}</li>`).join('') + '</ul>'
          + `</div>`;
      }
      qualityBox.innerHTML = html;
    } else {
      qualityBox.style.display = 'none';
    }

    // Fertiges Skript immer zum Anschauen/Bearbeiten laden — kein Popup.
    el('#gen-status').textContent += ' — loaded in Playback (review & adjust).';
    playback.loadScriptPath(result.path, { review: true });
  });

  el('#gen-choose').addEventListener('click', chooseVideo);
  el('#gen-check-deps').addEventListener('click', checkDeps);
  el('#gen-generate').addEventListener('click', generate);
  el('#gen-seek-btn').addEventListener('click', () => seekTo(parseFloat(el('#gen-seek').value) || 0));
  el('#gen-seek-plus').addEventListener('click', () => seekTo(seekSec + 1));
  el('#gen-seek-plus5').addEventListener('click', () => seekTo(seekSec + 5));
  el('#gen-roi2-toggle').addEventListener('click', () => {
    // If Tf/Tj still needs ROI2, force mark mode ON (do not toggle off by accident).
    if (isTfTj() && !roi2 && !roi2Mode) {
      setRoi2Mode(true);
    } else if (isTfTj() && !roi2 && roi2Mode) {
      // already ready to mark — keep ON
      setRoi2Mode(true);
    } else {
      setRoi2Mode(!roi2Mode);
    }
    if (roi2Mode) {
      el('#gen-status').textContent = 'Zone 2 (contact): drag on preview (gold).';
    }
  });
  el('#gen-target-add')?.addEventListener('click', () => {
    if (markMode === 'target') {
      setMarkMode(null);
      return;
    }
    setMarkMode('target');
    el('#gen-status').textContent = 'Extra target: drag on preview (magenta, fixed min-distance partner).';
  });
  el('#gen-mask-add')?.addEventListener('click', () => {
    if (markMode === 'mask') {
      setMarkMode(null);
      return;
    }
    setMarkMode('mask');
    el('#gen-status').textContent = 'Soft mask: drag on preview (dashed — feature exclude only).';
  });
  el('#gen-extras-clear')?.addEventListener('click', () => {
    extraTargets = [];
    maskRois = [];
    setMarkMode(null);
    updateRoiLabels();
    redraw();
    el('#gen-status').textContent = 'Extra targets and soft masks cleared.';
  });
  el('#gen-profile').addEventListener('change', () => {
    el('#gen-profile').dataset.userTouched = '1';
    updateProfileUi();
    if (isTfTj()) {
      const c1 = el('#gen-region-class');
      const c2 = el('#gen-region-class2');
      if (c1 && !c1.value) c1.value = 'glans';
      if (c2 && !c2.value) c2.value = 'nipples';
      // Step 3: vib on → tracked partner (Fix off). Do not auto-check static.
      syncRoi2FixedDefault();
    }
  });
  el('#gen-contact-vibration').addEventListener('change', () => {
    contactUserOverride = true;
    updateContactVibrationOpts();
  });
  el('#gen-contact-curve').addEventListener('change', () => {
    el('#gen-contact-curve').dataset.userTouched = '1';
  });
  el('#gen-contact-span').addEventListener('input', updateContactSpanLabel);
  updateContactSpanLabel();
  el('#gen-backend').addEventListener('change', () => {
    el('#gen-backend').dataset.userTouched = '1';
    updateGenerateEnabled();
  });
  el('#gen-autoroi').addEventListener('click', () => {
    if (!videoPath) return;
    const useAI = el('#gen-ai-roi').checked && !el('#gen-ai-roi').disabled;
    const two = isTfTj();
    el('#gen-autoroi').disabled = true;
    el('#gen-candidates').disabled = true;
    el('#gen-status').textContent = useAI
      ? (two ? 'AI searching both regions (ONNX)…' : 'AI region search running (ONNX model)…')
      : (two ? 'Searching both regions (distance/Tf/Tj suggestion)…'
        : 'Analyzing motion in video (may take a few seconds)…');
    const engine = useAI ? (two ? 'ai_two' : 'ai') : (two ? 'auto_two' : 'auto');
    AutoDetectROI(videoPath, engine);
  });

  el('#gen-candidates').addEventListener('click', () => {
    if (!videoPath) return;
    el('#gen-candidates').disabled = true;
    el('#gen-autoroi').disabled = true;
    el('#gen-status').textContent = 'Finding motion candidates (nothing applied until you click one)…';
    SuggestROICandidates(videoPath);
  });

  EventsOn('generate:roi-candidates', result => {
    hideProgress();
    el('#gen-candidates').disabled = false;
    el('#gen-autoroi').disabled = false;
    if (result.error) {
      uiError('Motion candidates: ' + result.error, el('#gen-status'));
      return;
    }
    const list = Array.isArray(result.candidates) ? result.candidates : [];
    candidates = list.map((c, i) => ({
      x: c.x, y: c.y, w: c.w, h: c.h,
      score: c.score || 0,
      index: c.index || (i + 1),
    }));
    if (!candidates.length) {
      el('#gen-status').textContent = 'No motion candidates — mark primary by hand.';
      redraw();
      return;
    }
    el('#gen-status').textContent =
      `${candidates.length} motion candidate${candidates.length === 1 ? '' : 's'} — click one to set Zone 1 (primary). Zone 2 never auto-filled.`;
    redraw();
  });

  const PROFILE_VALUES = ['standard', 'weich', 'autotune', 'tf'];

  el('#gen-suggest-profile').addEventListener('click', async () => {
    if (!videoPath) return;
    const status = el('#gen-suggest-status');
    status.textContent = 'Comparing to saved scenes…';
    el('#gen-suggest-profile').disabled = true;
    try {
      const result = await SuggestProfile(videoPath);
      if (!result.found) {
        status.textContent = 'No suggestion (no similar saved scene, AI server unreachable).';
        return;
      }
      const via = result.kind === 'ai'
        ? `KI, Konfidenz ${Math.round(result.confidence * 100)}%`
        : `gemessen, Abstand ${result.confidence.toFixed(3)}`;
      // "tj" war früher ein zweiter, identisch behandelter Dropdown-Eintrag
      // (siehe funscript/recipe.go NormalizeProfile) - inzwischen zu einem
      // Eintrag "tf" zusammengelegt. Ältere, lokal gemerkte Szenen können
      // noch mit "tj" beschriftet sein; hier auf den verbliebenen Wert
      // abbilden, statt beim Übernehmen an einer verschwundenen Option
      // stillschweigend hängenzubleiben.
      const label = result.label === 'tj' ? 'tf' : result.label;
      if (PROFILE_VALUES.includes(label)) {
        status.textContent = `Suggestion: "${label}" (${via}) — `;
        const applyBtn = document.createElement('button');
        applyBtn.textContent = 'Apply';
        applyBtn.addEventListener('click', () => {
          el('#gen-profile').value = label;
          updateProfileUi();
          status.textContent = `Profile “${label}” applied (${via}).`;
        });
        status.appendChild(applyBtn);
      } else {
        status.textContent = `Similar to saved scene “${result.label}” (${via}) — no `
          + 'direct profile name; not applied automatically.';
      }
    } catch (err) {
      status.textContent = 'Error: ' + err;
    } finally {
      el('#gen-suggest-profile').disabled = false;
    }
  });

  el('#gen-label-scene').addEventListener('click', async () => {
    if (!videoPath) return;
    const label = el('#gen-scene-label').value.trim();
    if (!label) {
      uiWarn('Enter a name for the scene.', el('#gen-suggest-status'));
      return;
    }
    el('#gen-label-scene').disabled = true;
    try {
      await LabelScene(videoPath, label);
      el('#gen-suggest-status').textContent = `Scene saved as "${label}".`;
    } catch (err) {
      uiError('Remember scene: ' + err, el('#gen-suggest-status'));
    } finally {
      el('#gen-label-scene').disabled = false;
    }
  });

  // Body-part class selects (docs/BODY_REGIONS.md)
  for (const selId of ['#gen-region-class', '#gen-region-class2', '#gen-target-class']) {
    const sel = el(selId);
    if (!sel) continue;
    for (const p of CANONICAL) {
      const opt = document.createElement('option');
      opt.value = p.id;
      opt.textContent = p.label;
      sel.appendChild(opt);
    }
  }
  el('#gen-roi2-fixed')?.addEventListener('change', () => {
    el('#gen-roi2-fixed').dataset.userTouched = '1';
  });

  syncWorkflowSteps();
}
