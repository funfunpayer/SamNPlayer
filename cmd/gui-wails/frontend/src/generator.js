import { SubmitFeedback, PickVideoFile, LoadFirstFrame, LoadFrameAt, GenerateScript, CancelGenerate, CancelROIDetection, CheckGeneratorDependencies, ScriptExistsForVideo, AutoDetectROI, DetectExpectedTipROI, SuggestROICandidates, CheckAIRoiAvailable, CheckAudioCheckAvailable, SuggestProfile, SuggestPipeline, LabelSceneWithProfile, ImproveGeneratedScript, GetScriptCurve, ScanSceneMap, SceneMapAvailable } from '../wailsjs/go/main/App';
import {
  CONTACT_CLASS_ORDER, TIP_CLASS_ORDER,
  labelFor, normalizeClass, orderedCanonical,
} from './bodyparts.js';
import { EventsOn } from '../wailsjs/runtime/runtime';
import { uiError, uiInfo, uiWarn } from './notify.js';
import { wireDataHelp } from './help.js';

export function initGenerator(root, playback) {
  root.classList.add('tab-create');
  root.innerHTML = `
    <header class="create-head">
      <h2>Create Emotion Script</h2>
      <p class="create-lede">From a quiet video — motion becomes feel you can play.</p>
    </header>
    <nav class="gen-steps" id="gen-steps" aria-label="Create workflow">
      <ol class="gen-steps-list">
        <li class="gen-step-item is-current" data-step="1"><span class="gen-step-num">1</span> Video</li>
        <li class="gen-step-item" data-step="2"><span class="gen-step-num">2</span> Where</li>
        <li class="gen-step-item" data-step="3"><span class="gen-step-num">3</span> Feel</li>
        <li class="gen-step-item" data-step="4"><span class="gen-step-num">4</span> Create</li>
        <li class="gen-step-item" data-step="5"><span class="gen-step-num">5</span> Review</li>
      </ol>
    </nav>
    <p class="hint gen-step-prompt" id="gen-step-prompt" style="margin-top:0">
      Start gently — choose a video; we find the motion and shape your Emotion Script.
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
      <h3 class="gen-step-title">2 · Where it moves</h3>
      <p class="hint" style="margin-top:0">
        We pick the tip area automatically (you can adjust). Contact areas only appear when Contact vibration is on.
      </p>
      <div class="row" style="align-items:center;">
        <button id="gen-autoroi" class="primary" disabled
          data-help="Finds the tip start region from motion (or AI if checked). You can always correct the box.">Find tip area</button>
        <button id="gen-candidates" type="button" disabled
          data-help="Shows ranked motion regions. Click = Tip. Shift-click = optional contact area when Contact vibration is on.">Show other spots</button>
        <button id="gen-seed-suggest" type="button" disabled hidden
          data-help="Proposes Tip + optional contact area. Apply required.">Suggest Tip+2nd</button>
        <span class="hint" id="gen-seed-status" style="margin:0"></span>
        <span class="checkbox-row" style="margin:0"><input type="checkbox" id="gen-ai-roi" disabled />
          <label for="gen-ai-roi" style="width:auto"
            data-help="Optional AI tip box suggestion — never writes the curve. Needs Settings → AI model.">Smarter tip find (optional)</label></span>
        <select id="gen-ai-target-class" disabled style="min-width:10em;">
          <option value="">Expected body point…</option>
        </select>
      </div>
      <p class="hint" id="gen-autoroi-hint" style="margin:0 0 6px 0">After the video loads we look for a tip area automatically.</p>
      <div class="hint" id="gen-ai-target-status" style="margin:0 0 6px 0;"></div>

      <div class="row" style="align-items:center; margin:4px 0;">
        <label style="width:auto;" data-help="Seek past a black intro before marking the region.">Time (s)</label>
        <input type="number" id="gen-seek" value="0" min="0" step="0.5" style="width:5em;" disabled />
        <button id="gen-seek-btn" type="button" disabled>Frame</button>
        <button id="gen-seek-plus" type="button" disabled>+1s</button>
        <button id="gen-seek-plus5" type="button" disabled>+5s</button>
        <label class="checkbox-row" style="margin:0 0 0 8px;"
          data-help="0–100 stroke gauge over the preview. Moves with Time/Frame after Create. Turn off anytime.">
          <input type="checkbox" id="gen-pos-overlay-toggle" checked />
          0–100 on video
        </label>
      </div>
      <div id="roi-canvas-wrap">
        <canvas id="roi-canvas"></canvas>
        <div id="gen-pos-overlay" class="pos-gauge" hidden aria-hidden="true">
          <div class="pos-gauge-scale" aria-hidden="true">
            <span>100</span><span>50</span><span>0</span>
          </div>
          <div class="pos-gauge-track">
            <div class="pos-gauge-fill" id="gen-pos-fill"></div>
            <div class="pos-gauge-knob" id="gen-pos-knob"></div>
          </div>
          <div class="pos-gauge-value" id="gen-pos-value">—</div>
        </div>
      </div>
      <div class="path-label" id="gen-roi-label">No region marked</div>
      <p class="hint" id="gen-pipeline-auto" style="margin:4px 0 8px 0;"></p>

      <div id="gen-contact-marks-wrap" style="margin-top:8px; padding-top:8px; border-top:1px solid rgba(255,255,255,0.08);">
        <p class="hint" style="margin:0 0 6px 0;">
          Contact vibration is on — optional labels &amp; contact areas (not required to Create).
          Tip class = Glans/Penis for the tracked tip. Contact areas = where touch should feel (nipples…).
          Multiple contact areas OK. Vib still follows stroke depth today.
        </p>
        <div class="row" style="align-items:center; flex-wrap:wrap; gap:8px; margin:6px 0;">
          <label style="width:auto;" data-help="Optional. Label the tracked tip (Glans preferred, or whole Penis). Empty = any. Only shown while Contact vibration is on — not required for CSRT.">Tip class</label>
          <select id="gen-region-class" style="min-width:8em;">
            <option value="">(any)</option>
          </select>
          <label style="width:auto;" data-help="What the main contact area is (nipples, mouth, hand…). Defaults to Nipples when empty. Optional.">Contact type</label>
          <select id="gen-region-class2" style="min-width:8em;">
            <option value="">(pick class)</option>
          </select>
          <label class="checkbox-row" style="margin:0;"
            data-help="Fix contact area (static): keep the gold box where you drew it — do not track it. Use when nipples/mouth barely move and only the tip/camera moves. Off (default with Contact vib) = track that area so camera pans stay in sync.">
            <input type="checkbox" id="gen-roi2-fixed" /> Fix contact area (static)
          </label>
        </div>
        <div class="row" style="align-items:center; margin-top:6px;">
          <button id="gen-roi2-toggle" type="button"
            data-help="Mark the main contact area (gold). Example: one nipple, mouth, or hand. On Stroke, Contact vibration still follows stroke depth — the mark is for location/feel later. Tip↔partner distance needs Tf/Tj.">Mark contact area</button>
          <button id="gen-target-add" type="button"
            data-help="Add another contact area (magenta) — e.g. second nipple. Same idea as the first contact mark; you can mark several.">+ Another contact area</button>
          <label style="width:auto; margin:0;" data-help="Body-part type for the next extra contact mark.">Extra type</label>
          <select id="gen-target-class" style="min-width:7em;">
            <option value="">(any)</option>
          </select>
          <button id="gen-extras-clear" type="button"
            data-help="Clear extra contact areas and soft masks (keeps tip + first contact mark).">Clear extras</button>
          <span class="hint" id="gen-roi2-hint" style="margin:0">All optional. Vib = stroke depth unless Tf/Tj distance.</span>
        </div>
        <div class="path-label" id="gen-roi2-label">No contact area marked</div>
        <div class="path-label" id="gen-extras-label" style="display:none;"></div>
        <div class="row" style="align-items:center; margin-top:4px;">
          <button id="gen-mask-add" type="button"
            data-help="Advanced: soft-exclude mask (dashed gray). Punched out of camera/grid features — does not drive the stroke.">+ Soft mask (advanced)</button>
        </div>
      </div>
      <!-- 4-zone removed from product GUI (1-Zone CSRT Everyday). Backend kept for CLI / evidence experiments. -->
      <button id="gen-nomark" type="button" disabled hidden
        data-help="Removed from Create GUI — use tip CSRT. 4-zone remains CLI-only.">4-zone (advanced)</button>
    </section>

    <section class="gen-step-panel" id="gen-step-motion" data-step="3" hidden>
      <h3 class="gen-step-title">3 · How it feels</h3>
      <div class="row" style="align-items:center;">
        <label style="width:auto;" data-help="Normal = everyday feel. Soft = gentler. Autotune = extra cleanup for noisy clips.">Style</label>
        <select id="gen-profile">
          <option value="standard">Normal</option>
          <option value="weich">Soft</option>
          <option value="autotune">Autotune</option>
        </select>
      </div>
      <p class="hint" id="gen-profile-hint" style="margin:0 0 10px 0;">
        We follow the tip. Contact vibration (below) adds feel on deep strokes — on by default.
      </p>
      <p class="hint" id="gen-tftj-hint" style="display:none; margin:0 0 6px 0;"></p>
      <div id="gen-contact-vibration-wrap">
        <div class="checkbox-row" id="gen-contact-vibration-row">
          <input type="checkbox" id="gen-contact-vibration" checked />
          <label for="gen-contact-vibration"
            data-help="Extra vibration on deep strokes (high position / stroke depth). On by default. When on, Step 2 shows Contact area marks (nipples/mouth/…). Marks do not change Stroke vib yet (still depth-based) — they store where touch should feel.">Contact vibration (on by default)</label>
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

      <details id="gen-power-user" style="margin:6px 0 8px 0;">
        <summary style="cursor:pointer;">Power-user: scene memory</summary>
        <div class="row" style="align-items:center; margin-top:8px;">
          <button id="gen-suggest-profile" disabled
            data-help="Compares the motion signature to saved scenes first, optionally to a local AI server. Suggestion only — nothing is applied automatically.">Suggest profile</button>
          <span class="hint" id="gen-suggest-status" style="margin:0"></span>
        </div>
        <div class="row" style="align-items:center;">
          <input type="text" id="gen-scene-label" placeholder="Name for this scene (optional)" style="flex:1;" />
          <button id="gen-label-scene" disabled
            data-help="Saves the motion signature, this name and the Style currently selected above. The AI training tab can learn a local profile model from confirmed examples.">Remember scene + style</button>
        </div>
      </details>
    </section>

    <section class="gen-step-panel" id="gen-step-run" data-step="4" hidden>
      <h3 class="gen-step-title">4 · Create</h3>
      <details id="gen-advanced" style="margin:6px 0 10px 0;">
        <summary style="cursor:pointer;">Advanced settings</summary>
        <div style="margin-top:8px;">
          <div class="opt-group">Tracking</div>
          <div class="checkbox-row"><input type="checkbox" id="gen-invert" /><label for="gen-invert"
            data-help="Flips the stroke curve up↔down (100−pos). Use when the stroke feels inverted — not a tracker failure. Example: tip moves down but the script rises.">Invert motion direction</label></div>
          <div class="checkbox-row"><input type="checkbox" id="gen-camcomp" checked /><label for="gen-camcomp"
            data-help="Compensates camera pans using background features. Recommended for moving camera.">Camera motion compensation</label></div>
          <div class="checkbox-row"><input type="checkbox" id="gen-scenecut" checked /><label for="gen-scenecut"
            data-help="Detects hard cuts and re-anchors the tracker afterward.">Scene-cut detection</label></div>
          <div class="row" style="align-items:center;">
            <label style="width:auto;" data-help="CSRT tip tracking (Go path) — Everyday stroke writer. Whole-frame 4-zone stays CLI-only (weaker on measured clips; not a product stroke mode).">Tracking method</label>
            <select id="gen-backend">
              <option value="csrt" selected>CSRT (mark tip, Go path)</option>
            </select>
          </div>
          <p class="hint" id="gen-backend-hint" style="margin:0 0 6px 0;">CSRT needs a tip mark (auto-find or draw). Contact vibration is the feel layer — optional marks when vib is on.</p>
          <div class="checkbox-row"><input type="checkbox" id="gen-capture-trajectory" /><label for="gen-capture-trajectory"
            data-help="Records tip (x,y) per frame into the script. Needed for Feel Stage A (vib when tip grazes a contact mark) and the optional Play trajectory overlay. Soft-on with Contact vib; CSRT path only.">Record tip path (for contact feel + overlay)</label></div>
          <div class="checkbox-row"><input type="checkbox" id="gen-rhythm-grid" /><label for="gen-rhythm-grid"
            data-help="Starts inside the confirmed target box and follows only nearby cells with matching rhythm. A stronger unrelated body part cannot take over merely because CSRT drifts toward it. Opt-in; Go CSRT path only; ~+18% analysis time.">Rhythm-robust signal (target-locked, long clips)</label></div>
          <div class="row" style="align-items:center;gap:8px;flex-wrap:wrap;">
            <button type="button" class="secondary" id="gen-scene-map" disabled
              data-help="Quick rhythm heatmap (~6×8s windows) without running Generate. Explicit only — never auto before Create (Owner). Map drawing/marks = later Advanced step.">Show scene map</button>
            <span class="hint" id="gen-scene-map-status" style="margin:0;"></span>
          </div>

          <div class="opt-group">Signal &amp; quality</div>
          <div class="checkbox-row"><input type="checkbox" id="gen-dynrange" checked /><label for="gen-dynrange"
            data-help="Smoothly lifts weak sections to usable strength.">Sliding dynamics</label></div>
          <div class="checkbox-row"><input type="checkbox" id="gen-retry" checked /><label for="gen-retry"
            data-help="Automatically retries with other signal parameters when quality is poor.">Auto-Retry</label></div>
          <div class="checkbox-row"><input type="checkbox" id="gen-auto-ozone" /><label for="gen-auto-ozone"
            data-help="Suggests O-markers in the last eighth (highest mean position) only when the ending is clearly high. Classic from signal, no AI model.">Suggest O-markers automatically</label></div>
          <!-- Audio check lives in Review → Improve (post-generate). Still default-on at generate time via hidden input. -->
          <input type="checkbox" id="gen-audio-check" checked style="display:none" aria-hidden="true" />
          <!-- Ballast removed: AI second opinion + Flow downscale (no Everyday effect). -->

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
        </div>
      </details>

      <div class="row">
        <button id="gen-generate" class="primary" disabled>Create Emotion Script</button>
        <button id="gen-cancel" type="button" disabled>Cancel</button>
      </div>
      <div id="gen-progress-wrap" style="display:none; margin-top:8px;">
        <div style="height:10px; border-radius:5px; background:rgba(255,255,255,0.10); overflow:hidden;">
          <div id="gen-progress-bar" style="height:100%; width:0%; background:linear-gradient(90deg,var(--accent),var(--teal));
               transition:width .2s linear;"></div>
        </div>
        <div id="gen-progress-text" class="hint" style="margin-top:4px;"></div>
        <div id="gen-preview-steer-tip" class="hint" style="margin-top:4px;"></div>
      </div>
      <pre id="gen-log" class="run-log" aria-label="Creation progress"></pre>
      <p class="hint">
        Creates one <b>Emotion Script</b> for Play. Share to other apps is optional later.
      </p>
    </section>

    <section class="gen-step-panel" id="gen-step-result" data-step="5" hidden>
      <h3 class="gen-step-title">5 · Review &amp; improve</h3>
      <div id="gen-improve" style="display:none; margin-top:4px; padding:10px;
           border:1px solid var(--border); border-radius:4px;">
        <div style="margin-bottom:6px;">
          Soft polish on the CSRT result — trim ends, fill gaps, optional audio check.
          Gaps already get an auto pass right after Create; re-run here after trim or with audio spacing.
        </div>
        <div class="row" style="align-items:center; flex-wrap:wrap; gap:8px;">
          <label style="width:auto;" data-help="Cut black intro / late credits. 0 = keep from start.">Start (s)</label>
          <input type="number" id="gen-improve-start" value="0" min="0" step="0.5" style="width:5em;" />
          <label style="width:auto;" data-help="Cut after this time. 0 = keep to end.">End (s)</label>
          <input type="number" id="gen-improve-end" value="0" min="0" step="0.5" style="width:5em;" />
        </div>
        <div class="row" style="align-items:center; flex-wrap:wrap; gap:8px; margin-top:6px;">
          <label class="checkbox-row" style="margin:0;"
            data-help="Inserts linear points across long holes (tracker loss / sparse keyframes). Does not invent motion from audio.">
            <input type="checkbox" id="gen-improve-fill" checked /> Fill gaps
          </label>
          <label class="checkbox-row" style="margin:0;"
            data-help="When filling gaps, space new points using audio tempo (half-period) if ffmpeg finds a clear beat. Still linear positions — not audio→curve.">
            <input type="checkbox" id="gen-improve-audio-fill" checked /> Align fill to audio tempo
          </label>
          <label class="checkbox-row" style="margin:0;"
            data-help="Compares script Hz to audio Hz and stamps warnings. Toggle off to skip. Needs ffmpeg.">
            <input type="checkbox" id="gen-improve-audio" checked /> Audio check
          </label>
        </div>
        <div class="row" style="margin-top:8px;">
          <button id="gen-improve-apply" class="primary" type="button">Improve script</button>
          <span class="hint" id="gen-improve-status" style="margin:0 0 0 8px;"></span>
        </div>
      </div>
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
  let lastSceneMap = null;
  let img = new Image();
  let nativeW = 0, nativeH = 0;
  let roi = null; // {x,y,w,h} in videopixeln
  let roi2 = null; // zweite Region für Tf/Tj (distance + suction)
  let candidates = []; // TFTJ 4b / MT-Seed: [{x,y,w,h,score,index}, ...] dashed until pick
  let pendingSeed = null; // MT-Seed: { tip, partner } suggest ≠ auto-commit
  let pendingAITarget = null; // strict semantic proposal; Apply required
  let activeAITargetRequest = null; // {requestId, videoPath, expectedClass, timeSec}
  let aiTargetRequestSeq = 0;
  let extraTargets = []; // additional fixed Tf/Tj anchors (min-distance)
  let maskRois = []; // soft-exclude boxes
  let roi2Mode = false; // Knopf „2. Region“ aktiv
  let markMode = null; // null | 'target' | 'mask'
  let dragging = false, draggingSecond = false, startX = 0, startY = 0, curX = 0, curY = 0;
  let seekSec = 0;
  let generating = false;
  let lastOutputPath = null;
  // Everyday FunGen-like: Generate with no ROI → auto-find tip then generate.
  let pendingGenerateAfterRoi = false;
  let userCancelRequested = false;
  let activeCreateSeq = 0;
  let profileSuggestionSeq = 0;
  // Multi-drop batch note — keep visible through auto-find status updates.
  let videoBatchNote = '';
  // FunGen-like 0–100 gauge over the preview (after Generate).
  let genCurvePoints = null; // [{atMs, pos}, ...]
  const POS_OVERLAY_PREF = 'samn.genPosOverlay';

  function posOverlayWanted() {
    const cb = el('#gen-pos-overlay-toggle');
    if (!cb) return true;
    try {
      const saved = localStorage.getItem(POS_OVERLAY_PREF);
      if (saved === '0') { cb.checked = false; return false; }
      if (saved === '1') { cb.checked = true; return true; }
    } catch (_) { /* ignore */ }
    return !!cb.checked;
  }

  function setPosOverlayVisible(on) {
    const box = el('#gen-pos-overlay');
    if (!box) return;
    const show = on && posOverlayWanted() && genCurvePoints && genCurvePoints.length >= 2;
    box.hidden = !show;
    box.setAttribute('aria-hidden', show ? 'false' : 'true');
  }

  function interpGenPos(tMs) {
    const pts = genCurvePoints;
    if (!pts || pts.length < 1) return null;
    if (tMs <= pts[0].atMs) return pts[0].pos;
    const last = pts[pts.length - 1];
    if (tMs >= last.atMs) return last.pos;
    for (let i = 1; i < pts.length; i++) {
      const a = pts[i - 1], b = pts[i];
      if (tMs >= a.atMs && tMs <= b.atMs) {
        if (b.atMs === a.atMs) return b.pos;
        const f = (tMs - a.atMs) / (b.atMs - a.atMs);
        return a.pos + f * (b.pos - a.pos);
      }
    }
    return last.pos;
  }

  function updatePosOverlay() {
    const knob = el('#gen-pos-knob');
    const fill = el('#gen-pos-fill');
    const val = el('#gen-pos-value');
    if (!knob || !fill || !val) return;
    setPosOverlayVisible(true);
    if (el('#gen-pos-overlay')?.hidden) return;
    const pos = interpGenPos(Math.round(seekSec * 1000));
    if (pos == null || Number.isNaN(pos)) {
      val.textContent = '—';
      return;
    }
    const p = Math.max(0, Math.min(100, pos));
    // CSS: bottom = 0, top = 100
    const pct = p; // height from bottom
    knob.style.bottom = `calc(${pct}% - 7px)`;
    fill.style.height = pct + '%';
    val.textContent = String(Math.round(p));
  }

  async function loadGenCurveFromPlay() {
    try {
      const pts = await GetScriptCurve(800);
      if (Array.isArray(pts) && pts.length >= 2) {
        genCurvePoints = pts.map(p => ({
          atMs: p.atMs ?? p.AtMs ?? 0,
          pos: p.pos ?? p.Pos ?? 0,
        }));
        updatePosOverlay();
        return;
      }
    } catch (_) { /* no script loaded yet */ }
    genCurvePoints = null;
    setPosOverlayVisible(false);
  }
  const DISPLAY_W = 560;

  const ROI1_STROKE = '#3dccc0';
  const ROI1_FILL = 'rgba(61,204,192,0.16)';
  const ROI2_STROKE = '#f2b03d';
  const ROI2_FILL = 'rgba(242,176,61,0.18)';
  const TARGET_STROKE = '#e070a0';
  const TARGET_FILL = 'rgba(224,112,160,0.16)';
  const AI_TARGET_STROKE = '#7ee787';
  const AI_TARGET_FILL = 'rgba(126,231,135,0.14)';
  const MASK_STROKE = 'rgba(180,180,190,0.85)';
  const MASK_FILL = 'rgba(120,120,130,0.12)';

  function isTfTj() {
    // Legacy distance profile (CLI / old saves). Product GUI no longer offers it —
    // Contact vibration on Normal is the everyday feel layer (owner 21 Sep).
    const p = (el('#gen-profile').value || '').toLowerCase();
    return p === 'tf' || p === 'tj';
  }

  function normalizeProductProfile() {
    const sel = el('#gen-profile');
    if (!sel) return;
    const p = (sel.value || '').toLowerCase();
    if (p === 'tf' || p === 'tj') {
      sel.value = 'standard';
      sel.dataset.userTouched = '1';
    }
  }

  // KI-Regionssuche (ai_roi.py, lokales ONNX-Modell) ist optional - ohne
  // installiertes onnxruntime oder ohne Modelldatei bleibt es bei der
  // klassischen Rhythmus-Heuristik (auto_roi.py). Einmal beim Öffnen des
  // Tabs geprüft (kostet einen Python-Start), nicht bei jedem videoladen.
  function refreshAIRoiAvailability() {
    CheckAIRoiAvailable().then(available => {
      const checkbox = el('#gen-ai-roi');
      const target = el('#gen-ai-target-class');
      checkbox.disabled = !available;
      if (target) target.disabled = !available || !checkbox.checked;
      el('#gen-autoroi-hint').textContent = available
        ? 'AI mode checks the currently displayed frame for the expected body point and never '
          + 'falls back to another class. A matching box remains a proposal until you press Apply.'
        : 'Analyzes motion in the video (classic, no AI model) — you can still '
          + 'correct the region by hand. AI detection: no local ONNX model '
          + 'found (Settings → AI model path, or default folder).';
    }).catch(() => {});
  }
  refreshAIRoiAvailability();
  window.addEventListener('samn-ai-roi-refresh', refreshAIRoiAvailability);

  // Audio-Tempo-Prüfung braucht nur ffmpeg auf dem PATH (Go-native post-hoc
  // oder Python bei PreferPython). Default-on at generate time; Review step
  // exposes the user-facing toggle for improve / re-check.
  CheckAudioCheckAvailable().then(available => {
    const genCheck = el('#gen-audio-check');
    const improveCheck = el('#gen-improve-audio');
    const improveFill = el('#gen-improve-audio-fill');
    if (genCheck) {
      genCheck.disabled = !available;
      if (!available) {
        genCheck.checked = false;
        genCheck.title = 'ffmpeg not found — use portable release or Settings → Install video tools';
      } else {
        genCheck.checked = true;
      }
    }
    if (improveCheck) {
      improveCheck.disabled = !available;
      if (!available) improveCheck.checked = false;
      else improveCheck.checked = true;
    }
    if (improveFill) {
      improveFill.disabled = !available;
      if (!available) improveFill.checked = false;
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
      ? secondRegionLabel(roi2, 'gold')
      : 'No contact area marked';
    const extras = el('#gen-extras-label');
    if (extras) {
      const parts = [];
      if (extraTargets.length) {
        const named = extraTargets.map((t, i) => t.class ? t.class : `#${i + 1}`).join(', ');
        parts.push(`${extraTargets.length} extra contact (${named}, magenta)`);
      }
      if (maskRois.length) {
        parts.push(`${maskRois.length} soft mask${maskRois.length === 1 ? '' : 's'} (dashed)`);
      }
      extras.style.display = parts.length ? 'block' : 'none';
      extras.textContent = parts.length ? parts.join(' · ') : '';
    }

    // Per-scene ROI is unsupported on the real Tf/Tj two-point distance path.
    // Optional Zone 2 on Stroke is UX-only (not sent in payload) — do NOT
    // disable “Re-find region after each cut” just because Zone 2 is marked.
    const twoPoint = !!roi2 && isTfTj();
    const perScene = el('#gen-perscene');
    perScene.disabled = twoPoint;
    if (twoPoint) perScene.checked = false;
    perScene.title = twoPoint
      ? 'Not available for Tf/Tj tip↔partner distance — region is not re-searched there.'
      : '';
    // Product GUI: CSRT tip only (1-Zone). 4-zone / research backends = CLI.
    const be = el('#gen-backend').value;
    if (be !== 'csrt') {
      el('#gen-backend').value = 'csrt';
    }
    updateGenerateEnabled();
  }

  // CSRT needs a tip mark. (Legacy no-mark backends are CLI-only now.)
  function backendNeedsRoi() {
    return el('#gen-backend').value !== 'region_fusion_auto';
  }

  function isNoMarkMotion() {
    return el('#gen-backend').value === 'region_fusion_auto';
  }

  function setNoMarkMotion(on) {
    // Product: never enable 4-zone from the GUI — force CSRT.
    const backend = el('#gen-backend');
    backend.value = 'csrt';
    delete backend.dataset.userTouched;
    const btn = el('#gen-nomark');
    if (btn) {
      btn.style.outline = '';
      btn.style.background = '';
      btn.textContent = '4-zone (CLI only)';
    }
    if (on) {
      normalizeProductProfile();
      candidates = [];
      clearPendingSeed();
      setSeedSuggestEnabled(false);
      el('#gen-status').textContent =
        'Tip CSRT is the Everyday stroke writer — 4-zone is CLI-only (not a GUI mode).';
    }
    syncNoMarkButton();
    updateGenerateEnabled();
    redraw();
  }

  function syncNoMarkButton() {
    const btn = el('#gen-nomark');
    if (!btn) return;
    const on = isNoMarkMotion();
    btn.style.outline = on ? '2px solid #7ec8ff' : '';
    btn.style.background = on ? 'rgba(126,200,255,0.18)' : '';
  }

  function contactVibrationOn() {
    return !!el('#gen-contact-vibration')?.checked;
  }

  // Everyday Generate: tip mark (CSRT) or no-mark 4-zone. Zone 2 never required.
  // FunGen-like: video alone is enough — Generate will auto-find tip if missing.
  function regionReadyForGenerate() {
    if (!videoPath) return false;
    if (isNoMarkMotion()) return true;
    if (backendNeedsRoi() && !roi) return true; // Generate triggers auto-find
    return true;
  }

  function startAutoFindRegion() {
    if (!videoPath) return;
    setNoMarkMotion(false);
    const useAI = el('#gen-ai-roi').checked && !el('#gen-ai-roi').disabled;
    const expectedClass = normalizeClass(el('#gen-ai-target-class')?.value || '');
    if (useAI && !expectedClass) {
      pendingGenerateAfterRoi = false;
      el('#gen-status').textContent = (
        'Choose the expected body point for strict AI detection, or turn AI off for generic motion search.'
      ) + videoBatchNote;
      el('#gen-ai-target-class')?.focus();
      return;
    }
    clearPendingAITarget();
    redraw();
    el('#gen-autoroi').disabled = true;
    el('#gen-candidates').disabled = true;
    el('#gen-nomark').disabled = true;
    el('#gen-status').textContent = (useAI
      ? `Looking only for ${labelFor(expectedClass) || expectedClass} (strict AI; no class fallback)…`
      : 'Finding tip region automatically (CSRT)…') + videoBatchNote;
    if (useAI) {
      const requestId = ++aiTargetRequestSeq;
      activeAITargetRequest = { requestId, videoPath, expectedClass, timeSec: seekSec };
      DetectExpectedTipROI(videoPath, expectedClass, seekSec, requestId);
    } else {
      activeAITargetRequest = null;
      AutoDetectROI(videoPath, 'auto');
    }
  }

  // Default Fix Zone 2 = off when an optional contact partner is marked + vib on.
  function syncRoi2FixedDefault() {
    const fix = el('#gen-roi2-fixed');
    if (!fix || fix.dataset.userTouched) return;
    if (roi2 && contactVibrationOn()) {
      fix.checked = false;
    }
  }

  // Progressive steps: next panel appears only when the previous action is done.
  function syncWorkflowSteps() {
    const hasVideo = !!videoPath;
    const hasRoi1 = !!roi;
    const canRun = regionReadyForGenerate();
    const hasResult = !!lastOutputPath;
    const noMark = isNoMarkMotion();

    const showRegion = hasVideo;
    // Unlock motion/profile once tip is marked, 4-zone is on, OR video is loaded
    // (Generate will auto-find tip — FunGen-like everyday path).
    const showMotion = hasVideo;
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
    } else if (!hasRoi1 && !noMark) {
      prompt.textContent = 'Finding tip… or mark / pick a spot. Then Create.';
    } else if (!canRun && !generating) {
      prompt.textContent = 'Step 3: Contact vibration is on by default — Create unlocks when tracking is ready.';
    } else if (generating) {
      prompt.textContent = 'Step 4: creating… you can Cancel if needed.';
    } else if (!hasResult) {
      prompt.textContent = noMark
        ? 'Step 4: Create your Emotion Script.'
        : 'Step 4: Create Emotion Script — optional Advanced settings below.';
    } else {
      prompt.textContent = 'Step 5: Improve, then Play — edit dots on the soft curve.';
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

  let sceneMapAvailable = false;

  function updateSceneMapButton() {
    const btn = el('#gen-scene-map');
    if (!btn) return;
    // Explicit Advanced control only (Owner § 6) — needs video + OpenCV path.
    btn.disabled = !videoPath || !sceneMapAvailable || generating;
  }

  async function runSceneMapScan() {
    if (!videoPath || !sceneMapAvailable) return;
    const btn = el('#gen-scene-map');
    const status = el('#gen-scene-map-status');
    if (btn) btn.disabled = true;
    if (status) status.textContent = 'Scanning rhythm map…';
    try {
      const map = await ScanSceneMap(videoPath, 0);
      lastSceneMap = map || null;
      const n = Array.isArray(map?.windows) ? map.windows.length : 0;
      const grid = map?.cols && map?.rows ? `${map.cols}×${map.rows}` : '';
      if (status) {
        status.textContent = n
          ? `Map ready: ${n} windows${grid ? ` · ${grid}` : ''} (marks/draw = later)`
          : 'Map scan returned no windows.';
      }
      uiInfo(status?.textContent || 'Scene map scan done.', el('#gen-status'));
    } catch (err) {
      lastSceneMap = null;
      if (status) status.textContent = '';
      uiError('Scene map: ' + err, el('#gen-status'));
    } finally {
      updateSceneMapButton();
    }
  }

  let contactUserOverride = false;

  function updateContactVibrationOpts() {
    const on = el('#gen-contact-vibration').checked;
    el('#gen-contact-vibration-opts').style.display = on ? 'block' : 'none';
    const marks = el('#gen-contact-marks-wrap');
    if (marks) {
      marks.style.display = on ? 'block' : 'none';
      marks.hidden = !on;
    }
    if (on) {
      const tip = el('#gen-region-class');
      if (tip && !tip.value && !tip.dataset.userTouched) {
        // Soft default: Glans (tip) — optional, user can clear to (any).
        if ([...tip.options].some((o) => o.value === 'glans')) {
          tip.value = 'glans';
        }
      }
      const c2 = el('#gen-region-class2');
      if (c2 && !c2.value && !c2.dataset.userTouched) {
        // Default contact type: nipples (common dual-side touch target).
        if ([...c2.options].some((o) => o.value === 'nipples')) {
          c2.value = 'nipples';
        }
      }
      const tc = el('#gen-target-class');
      if (tc && !tc.value && !tc.dataset.userTouched) {
        if ([...tc.options].some((o) => o.value === 'nipples')) {
          tc.value = 'nipples';
        }
      }
      // Feel Stage A needs tip trajectory — soft-on with Contact vib.
      const traj = el('#gen-capture-trajectory');
      if (traj && !traj.dataset.userTouched) {
        traj.checked = true;
      }
    } else {
      // Leaving contact-mark mode when vib is off.
      if (roi2Mode) setRoi2Mode(false);
      if (markMode === 'target' || markMode === 'mask') setMarkMode(null);
      const traj = el('#gen-capture-trajectory');
      if (traj && !traj.dataset.userTouched) {
        traj.checked = false;
      }
    }
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
    normalizeProductProfile();
    el('#gen-tftj-hint').style.display = 'none';
    // Contact vib is the product feel layer — always shown.
    el('#gen-contact-vibration-wrap').style.display = 'block';
    el('#gen-contact-vibration-row').style.display = 'flex';
    if (!contactUserOverride) {
      el('#gen-contact-vibration').checked = true;
      if (!el('#gen-contact-curve').dataset.userTouched) {
        el('#gen-contact-curve').value = 'soft';
      }
    }
    syncRoi2FixedDefault();
    updateContactVibrationOpts();
    updateGenerateEnabled();
    if (videoPath && roi2) {
      el('#gen-status').textContent = isTfTj()
        ? (contactVibrationOn()
          ? 'Contact area set — tip↔partner distance; tracked unless “Fix contact area” is on.'
          : 'Contact area set (Contact vib off) — tip↔partner distance still uses both regions.')
        : 'Contact area marked — vib = depth and/or tip-near-mark when tip path is recorded.';
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

  // MT-Seed: IoU gate so Tip+2nd suggestions stay spatially distinct.
  function boxesOverlap(a, b, iouThresh = 0.25) {
    if (!a || !b) return false;
    const ax2 = a.x + a.w, ay2 = a.y + a.h;
    const bx2 = b.x + b.w, by2 = b.y + b.h;
    const ix = Math.max(0, Math.min(ax2, bx2) - Math.max(a.x, b.x));
    const iy = Math.max(0, Math.min(ay2, by2) - Math.max(a.y, b.y));
    const inter = ix * iy;
    if (inter <= 0) return false;
    const union = a.w * a.h + b.w * b.h - inter;
    return union > 0 && (inter / union) >= iouThresh;
  }

  function regionClass2Value() {
    return normalizeClass(el('#gen-region-class2')?.value || '');
  }

  function regionClass1Value() {
    return normalizeClass(el('#gen-region-class')?.value || '');
  }

  /** Apply optional body-part class on Zone 2 (suggest ≠ invent Partner). */
  function applySecondClassFromCandidate(c) {
    const sel = el('#gen-region-class2');
    if (!sel) return '';
    const fromCand = normalizeClass(c?.class || '');
    if (fromCand) {
      if (![...sel.options].some(o => o.value === fromCand)) {
        const opt = document.createElement('option');
        opt.value = fromCand;
        opt.textContent = labelFor(fromCand) || fromCand;
        sel.appendChild(opt);
      }
      sel.value = fromCand;
      return fromCand;
    }
    return regionClass2Value();
  }

  function secondRegionLabel(roiBox, via) {
    const cls = regionClass2Value();
    const clsTag = cls ? `, ${labelFor(cls) || cls}` : '';
    return `Contact area: x=${roiBox.x} y=${roiBox.y} w=${roiBox.w} h=${roiBox.h}`
      + ` (video pixels${via ? `, ${via}` : ''}${clsTag})`;
  }

  function nudgeZone2ClassIfEmpty() {
    const sel = el('#gen-region-class2');
    if (!sel || sel.value) return '';
    sel.style.outline = '2px solid rgba(242,176,61,0.85)';
    setTimeout(() => { if (sel) sel.style.outline = ''; }, 2400);
    return ' Pick contact type (nipples/mouth/hand…) — Partner is not the default.';
  }

  /** Ranked list → Tip (#1) + first non-overlapping partner. Suggest only. */
  function suggestTipPartnerPair(list) {
    if (!list || list.length < 2) return null;
    const tip = list[0];
    for (let i = 1; i < list.length; i++) {
      if (!boxesOverlap(tip, list[i])) return { tip, partner: list[i] };
    }
    return { tip, partner: list[1] };
  }

  function setSeedSuggestEnabled(on) {
    const btn = el('#gen-seed-suggest');
    if (!btn) return;
    btn.hidden = !on;
    btn.disabled = !on;
  }

  function clearPendingSeed() {
    pendingSeed = null;
    const status = el('#gen-seed-status');
    if (status) status.textContent = '';
  }

  function clearPendingAITarget() {
    pendingAITarget = null;
    const status = el('#gen-ai-target-status');
    if (status) status.textContent = '';
  }

  function cancelActiveAITargetRequest() {
    if (!activeAITargetRequest) return;
    activeAITargetRequest = null;
    pendingGenerateAfterRoi = false;
    CancelROIDetection().catch(() => {});
    hideProgress();
    if (videoPath) {
      el('#gen-autoroi').disabled = false;
      el('#gen-candidates').disabled = false;
      el('#gen-nomark').disabled = false;
    }
  }

  function renderPendingAITarget() {
    const status = el('#gen-ai-target-status');
    if (!status) return;
    status.textContent = '';
    if (!pendingAITarget) return;
    const cls = labelFor(pendingAITarget.matchedClass) || pendingAITarget.matchedClass;
    const confidence = Math.round((pendingAITarget.confidence || 0) * 100);
    status.appendChild(document.createTextNode(
      `Detected ${cls} · ${confidence}% — `));
    const apply = document.createElement('button');
    apply.type = 'button';
    apply.textContent = 'Apply target';
    apply.addEventListener('click', () => {
      if (!pendingAITarget) return;
      roi = {
        x: pendingAITarget.x, y: pendingAITarget.y,
        w: pendingAITarget.w, h: pendingAITarget.h,
      };
      const selectedClass = normalizeClass(pendingAITarget.matchedClass || pendingAITarget.expectedClass);
      const classSelect = el('#gen-region-class');
      if (classSelect && selectedClass) classSelect.value = selectedClass;
      pendingAITarget = null;
      status.textContent = `${cls} target applied — CSRT starts here. Optional target-locked rhythm remains under Advanced.`;
      updateRoiLabels();
      updateGenerateEnabled();
      redraw();
      autoApplyPipeline();
    });
    const dismiss = document.createElement('button');
    dismiss.type = 'button';
    dismiss.textContent = 'Dismiss';
    dismiss.style.marginLeft = '6px';
    dismiss.addEventListener('click', () => {
      clearPendingAITarget();
      redraw();
    });
    status.appendChild(apply);
    status.appendChild(dismiss);
  }

  function renderPendingSeedStatus() {
    const status = el('#gen-seed-status');
    if (!status) return;
    status.textContent = '';
    if (!pendingSeed) return;
    const tip = pendingSeed.tip, partner = pendingSeed.partner;
    status.appendChild(document.createTextNode(
      `Suggest Tip #${tip.index} + 2nd #${partner.index} — `));
    const applyBtn = document.createElement('button');
    applyBtn.type = 'button';
    applyBtn.textContent = 'Apply';
    applyBtn.addEventListener('click', () => applyPendingSeed());
    const dismissBtn = document.createElement('button');
    dismissBtn.type = 'button';
    dismissBtn.textContent = 'Dismiss';
    dismissBtn.style.marginLeft = '6px';
    dismissBtn.addEventListener('click', () => {
      clearPendingSeed();
      el('#gen-status').textContent =
        'Suggestion dismissed — Everyday tip-CSRT if you skip the contact mark.';
      redraw();
    });
    status.appendChild(applyBtn);
    status.appendChild(dismissBtn);
  }

  function applyPendingSeed() {
    if (!pendingSeed) return;
    const { tip, partner } = pendingSeed;
    roi = { x: tip.x, y: tip.y, w: tip.w, h: tip.h };
    roi2 = { x: partner.x, y: partner.y, w: partner.w, h: partner.h };
    const tipCls = normalizeClass(tip.class || '');
    if (tipCls && el('#gen-region-class')) {
      const sel = el('#gen-region-class');
      if (![...sel.options].some(o => o.value === tipCls)) {
        const opt = document.createElement('option');
        opt.value = tipCls;
        opt.textContent = labelFor(tipCls) || tipCls;
        sel.appendChild(opt);
      }
      sel.value = tipCls;
    }
    applySecondClassFromCandidate(partner);
    clearPendingSeed();
    setRoi2Mode(false);
    updateRoiLabels();
    updateProfileUi();
    updateGenerateEnabled();
    autoApplyPipeline();
    const tipTag = regionClass1Value()
      ? `, ${labelFor(regionClass1Value())}` : '';
    el('#gen-roi-label').textContent =
      `Region: x=${roi.x} y=${roi.y} w=${roi.w} h=${roi.h}`
      + ` (video pixels, Tip candidate #${tip.index}${tipTag})`;
    el('#gen-roi2-label').textContent =
      secondRegionLabel(roi2, `2nd candidate #${partner.index}`);
    const nudge = nudgeZone2ClassIfEmpty();
    el('#gen-status').textContent =
      `Tip #${tip.index} + contact #${partner.index} applied — correct by hand if needed.`
      + ` Contact mark was never auto-filled.${nudge}`;
    redraw();
  }

  function drawCandidate(c) {
    if (!c || !nativeW || !nativeH) return;
    const scaleX = canvas.width / nativeW, scaleY = canvas.height / nativeH;
    const dx0 = c.x * scaleX, dy0 = c.y * scaleY, dw = c.w * scaleX, dh = c.h * scaleY;
    const isTip = pendingSeed && pendingSeed.tip && pendingSeed.tip.index === c.index;
    const isPartner = pendingSeed && pendingSeed.partner && pendingSeed.partner.index === c.index;
    ctx.save();
    ctx.setLineDash([5, 4]);
    if (isTip) {
      ctx.strokeStyle = 'rgba(61, 204, 192, 0.98)';
      ctx.fillStyle = 'rgba(61, 204, 192, 0.22)';
    } else if (isPartner) {
      ctx.strokeStyle = 'rgba(242, 176, 61, 0.98)';
      ctx.fillStyle = 'rgba(242, 176, 61, 0.22)';
    } else {
      ctx.strokeStyle = 'rgba(120, 200, 255, 0.95)';
      ctx.fillStyle = 'rgba(120, 200, 255, 0.12)';
    }
    ctx.lineWidth = 2;
    ctx.strokeRect(dx0, dy0, dw, dh);
    ctx.fillRect(dx0, dy0, dw, dh);
    let label = '#' + (c.index || '?');
    if (isTip) label += ' Tip';
    else if (isPartner) label += ' 2nd';
    const cls = normalizeClass(c.class || '')
      || (isPartner ? regionClass2Value() : '')
      || (isTip ? regionClass1Value() : '');
    if (cls) label += ' ' + (labelFor(cls) || cls);
    ctx.setLineDash([]);
    ctx.font = '600 13px system-ui, sans-serif';
    ctx.fillStyle = 'rgba(10, 20, 30, 0.75)';
    ctx.fillRect(dx0 + 2, dy0 + 2, ctx.measureText(label).width + 8, 18);
    ctx.fillStyle = isPartner ? '#ffe9c2' : '#dff3ff';
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

  /** zone 1 = tip, zone 2 = body-part / contact. Never fills the other zone (issue #8). */
  function pickCandidate(c, zone = 1) {
    if (!c) return;
    if (zone === 2) {
      roi2 = { x: c.x, y: c.y, w: c.w, h: c.h };
      applySecondClassFromCandidate(c);
      setRoi2Mode(false);
      updateRoiLabels();
      updateGenerateEnabled();
      el('#gen-roi2-label').textContent =
        secondRegionLabel(roi2, `candidate #${c.index}`);
      const nudge = nudgeZone2ClassIfEmpty();
      const msg = `2nd mark set from candidate #${c.index} — correct by hand if needed.${nudge}`;
      el('#gen-status').textContent = msg;
      redraw();
      // Re-assert status after pipeline hint (SuggestPipeline → updateProfileUi).
      autoApplyPipeline().then(() => { el('#gen-status').textContent = msg; });
      return;
    }
    roi = { x: c.x, y: c.y, w: c.w, h: c.h };
    const tipCls = normalizeClass(c.class || '');
    if (tipCls && el('#gen-region-class')) {
      const sel = el('#gen-region-class');
      if (![...sel.options].some(o => o.value === tipCls)) {
        const opt = document.createElement('option');
        opt.value = tipCls;
        opt.textContent = labelFor(tipCls) || tipCls;
        sel.appendChild(opt);
      }
      sel.value = tipCls;
    }
    // Never auto-fill Zone 2 from a Zone 1 pick (issue #8 / TFTJ 4b / MT-Seed).
    updateRoiLabels();
    updateGenerateEnabled();
    const tipTag = regionClass1Value()
      ? `, ${labelFor(regionClass1Value())}` : '';
    el('#gen-roi-label').textContent =
      `Region: x=${roi.x} y=${roi.y} w=${roi.w} h=${roi.h}`
      + ` (video pixels, candidate #${c.index}${tipTag})`;
    const msg =
      `Tip set from candidate #${c.index} — optional: Shift-click a 2nd body-part mark, or Suggest Tip+2nd. Everyday tip alone is fine.`;
    el('#gen-status').textContent = msg;
    redraw();
    autoApplyPipeline().then(() => { el('#gen-status').textContent = msg; });
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
    if (pendingAITarget) drawNativeRect(pendingAITarget, AI_TARGET_STROKE, AI_TARGET_FILL, true);
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
    cancelActiveAITargetRequest();
    clearPendingAITarget();
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
    // Tiny press: pick a motion candidate if shown (TFTJ 4b / MT-Seed).
    // Click → Tip (Zone 1). Shift / Zone-2 mode → Partner (Zone 2). Never auto-fills the other.
    if (w < 8 && h < 8) {
      if (!mode && candidates.length) {
        const hit = hitCandidate(startX, startY);
        if (hit) {
          pickCandidate(hit, wasSecond ? 2 : 1);
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
      const tag = cls ? ` (${cls})` : '';
      el('#gen-status').textContent = `Extra contact #${extraTargets.length}${tag} added.`;
    } else if (mode === 'mask') {
      maskRois.push(box);
      setMarkMode(null);
      el('#gen-status').textContent = `Soft mask #${maskRois.length} added (feature exclude).`;
    } else if (wasSecond) {
      roi2 = box;
      setRoi2Mode(false);
      // Zone 2 is optional for Contact vib — do not switch to legacy Tf/Tj.
      el('#gen-status').textContent = contactVibrationOn()
        ? 'Optional contact zone set — Contact vib on (stroke depth / approach).'
        : 'Optional contact zone set.';
    } else {
      roi = box;
    }
    updateRoiLabels();
    updateGenerateEnabled();
    autoApplyPipeline();
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
        const prof = (s.Profile === 'tf' || s.Profile === 'tj') ? 'standard' : s.Profile;
        el('#gen-profile').value = prof;
        updateProfileUi();
      }
      const pipe = el('#gen-pipeline-auto');
      if (pipe) {
        pipe.textContent = (s.Reason || '') + (s.GoPath ? ' · Go path' : ' · Python path');
      }
      updateGenerateEnabled();
    } catch (err) {
      const pipe = el('#gen-pipeline-auto');
      if (pipe) pipe.textContent = 'Pipeline hint unavailable';
    }
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
    cancelActiveAITargetRequest();
    videoPath = path;
    lastSceneMap = null;
    const smStatus = el('#gen-scene-map-status');
    if (smStatus) smStatus.textContent = '';
    const loadSuggestionSeq = ++profileSuggestionSeq;
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
    clearPendingAITarget();
    extraTargets = [];
    maskRois = [];
    setMarkMode(null);
    // Neues video: Pipeline-Vorschläge wieder erlauben.
    delete el('#gen-backend').dataset.userTouched;
    delete el('#gen-profile').dataset.userTouched;
    setRoi2Mode(false);
    el('#gen-generate').disabled = true;
    updateRoiLabels();
    // Stapelverarbeitung mehrerer videos gibt es noch nicht - vorher wurden
    // more dropped videos einfach stillschweigend verworfen, ohne dass
    // sichtbar war, dass überhaupt mehr als eins ankam.
    const batchNote = extraCount > 0
      ? ` (${extraCount} more video${extraCount === 1 ? '' : 's'} ignored — batch processing not available yet)`
      : '';
    videoBatchNote = batchNote;
    try {
      await showFrame(path, 0);
      candidates = [];
      clearPendingSeed();
      setSeedSuggestEnabled(false);
      // Region buttons stay disabled until generate:autoroi (auto-find owns them).
      el('#gen-autoroi').disabled = true;
      el('#gen-candidates').disabled = true;
      el('#gen-nomark').disabled = true;
      syncNoMarkButton();
      el('#gen-suggest-profile').disabled = false;
      el('#gen-label-scene').disabled = false;
      updateSceneMapButton();
      el('#gen-suggest-status').textContent = '';
      el('#gen-status').textContent = (
        'Everyday path: finding tip region for CSRT. Contact marks appear when Contact vib is on.'
      ) + batchNote;
      lastOutputPath = null;
      genCurvePoints = null;
      setPosOverlayVisible(false);
      el('#gen-feedback').style.display = 'none';
      el('#gen-improve').style.display = 'none';
      el('#gen-improve-status').textContent = '';
      el('#gen-quality').style.display = 'none';
      syncWorkflowSteps();
      // Soft-Vorschlag: Profil nur anzeigen, nie automatisch Apply.
      SuggestProfile(path).then(result => {
        if (!result || !videoPath || videoPath !== path
          || loadSuggestionSeq !== profileSuggestionSeq) return;
        const status = el('#gen-suggest-status');
        const via = result.kind === 'local_model' ? 'learned locally'
          : result.kind === 'ai' ? 'local AI server'
          : 'saved scene';
        const label = result.label === 'tj' ? 'tf' : result.label;
        if (label && ['standard', 'weich', 'autotune', 'tf'].includes(label)) {
          status.textContent = `Suggestion: “${label}” (${via}) — use “Suggest profile” to apply.`;
        }
      }).catch(() => {});
      // FunGen-like: auto-find tip after preview loads (CSRT first choice).
      startAutoFindRegion();
    } catch (err) {
      // Keep batchNote so multi-drop "ignored" stays visible even if preview fails.
      uiError('Load video: ' + err + batchNote, el('#gen-status'));
    }
  }

  async function showFrame(path, sec) {
    const preview = sec > 0 ? await LoadFrameAt(path, sec) : await LoadFirstFrame(path);
    nativeW = preview.width; nativeH = preview.height;
    const displayH = Math.round(DISPLAY_W * nativeH / nativeW);
    canvas.width = DISPLAY_W; canvas.height = displayH;
    await new Promise((resolve, reject) => {
      img.onload = () => { redraw(); resolve(); };
      img.onerror = () => reject(new Error('Preview image could not be decoded'));
      img.src = 'data:image/png;base64,' + preview.pngBase64;
    });
  }

  async function seekTo(sec) {
    if (!videoPath) return;
    cancelActiveAITargetRequest();
    clearPendingAITarget();
    seekSec = Math.max(0, sec);
    el('#gen-seek').value = String(seekSec);
    el('#gen-status').textContent = `Loading frame at ${seekSec}s…`;
    try {
      await showFrame(videoPath, seekSec);
      el('#gen-status').textContent = genCurvePoints
        ? `Frame at ${seekSec}s — 0–100 gauge follows the curve.`
        : `Frame at ${seekSec}s — mark region.`;
      updatePosOverlay();
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
    if (!videoPath) return;
    if (activeAITargetRequest) {
      el('#gen-status').textContent = 'Wait for the body-point check, or change/dismiss it before creating.';
      return;
    }
    if (pendingAITarget) {
      el('#gen-status').textContent = 'Review the detected body point first: Apply target, or Dismiss to keep the current region.';
      return;
    }
    normalizeProductProfile();

    // Everyday: no tip yet + CSRT → auto-find then continue.
    if (backendNeedsRoi() && !roi) {
      const strictAI = el('#gen-ai-roi').checked && !el('#gen-ai-roi').disabled;
      if (strictAI) {
        pendingGenerateAfterRoi = false;
        el('#gen-status').textContent = 'Find the expected body point, review it, then press Apply target before Create.';
        startAutoFindRegion();
        return;
      }
      pendingGenerateAfterRoi = true;
    generating = true;
    el('#gen-generate').disabled = true;
    el('#gen-cancel').disabled = false;
    updateSceneMapButton();
      el('#gen-status').textContent = 'No tip yet — finding region, then creating…';
      syncWorkflowSteps();
      startAutoFindRegion();
      return;
    }

    // Do not silently overwrite an existing script beside the video.
    let overwrite = false;
    try {
      if (await ScriptExistsForVideo(videoPath)) {
        const target = videoPath.replace(/\.[^.\\/]+$/, '');
        if (!confirm(`A script already exists for this video. Creating again can replace these files:\n${target}.samn (Emotion Script)\n${target}.funscript (copy for other apps)\n\nReplace existing files?`)) {
          return;
        }
        overwrite = true;
      }
    } catch (err) {
      // Existence check failed — prefer not to overwrite; backend will refuse cleanly.
    }

    el('#gen-generate').disabled = true;
    el('#gen-cancel').disabled = false;
    generating = true;
    userCancelRequested = false;
    syncWorkflowSteps();
    updateSceneMapButton();
    el('#gen-status').textContent = 'Creating…';
    el('#gen-log').textContent = '';
    {
      const wrap = el('#gen-progress-wrap');
      wrap.style.display = 'block';
      el('#gen-progress-bar').style.width = '0%';
      el('#gen-progress-bar').style.opacity = '1';
      el('#gen-progress-text').textContent = 'Starting…';
      const tip = el('#gen-preview-steer-tip');
      if (tip) tip.textContent = '';
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
      backend: el('#gen-backend').value || 'csrt',
      dynamicRangeMs: el('#gen-dynrange').checked ? 3000 : 0,
      profile: el('#gen-profile').value || 'standard',
      axis: el('#gen-axis').value,
      rdpTolerance: parseFloat(el('#gen-rdp').value) || 0,
      maxSpeed: parseFloat(el('#gen-maxspeed')?.value) || 0,
      flowDownscale: 0,
      overwrite,
      aiQualityOpinion: false,
      contactVibration: el('#gen-contact-vibration').checked,
      contactVibrationSpan: el('#gen-contact-vibration').checked
        ? (parseInt(el('#gen-contact-span').value, 10) || 75) / 100
        : 0,
      contactVibrationCurve: el('#gen-contact-vibration').checked
        ? (el('#gen-contact-curve').value || 'linear')
        : '',
      autoOZoneMarker: el('#gen-auto-ozone').checked,
      audioCheck: el('#gen-audio-check').checked,
      captureTrajectory: !!el('#gen-capture-trajectory')?.checked,
      rhythmGrid: !!el('#gen-rhythm-grid')?.checked,
      startTimeSec: seekSec > 0 ? seekSec : 0,
    };
    // Contact marks: persist when Contact vib is on (or legacy Tf/Tj distance).
    // Everyday stroke still uses tip-only CSRT — marks go to metadata.contact_marks
    // for feel / later feel-decouple (native pipeline stamps them; drive_stroke=false).
    if (roi2 && (contactVibrationOn() || isTfTj())) {
      payload.x2 = roi2.x;
      payload.y2 = roi2.y;
      payload.w2 = roi2.w;
      payload.h2 = roi2.h;
      payload.roi2Fixed = !!el('#gen-roi2-fixed')?.checked;
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
    userCancelRequested = true;
    pendingGenerateAfterRoi = false;
    CancelGenerate();
    el('#gen-status').textContent = 'Cancel requested…';
  });

  const handleProgressLine = line => {
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
    // Stage B steers (stroke preview): keep Advanced checkboxes in sync with
    // what Generate enabled for this run (same lines as applyStrokePreviewSteers).
    const s = String(line);
    if (/STROKE_PREVIEW:.*enabling .Re-find region after each cut/i.test(s)) {
      const box = el('#gen-perscene');
      if (box && !box.checked) box.checked = true;
      const tip = el('#gen-preview-steer-tip');
      if (tip) tip.textContent = 'Stroke preview: enabled Re-find region after each cut (high cut rate).';
    }
    if (/STROKE_PREVIEW:.*enabling camera motion compensation/i.test(s)) {
      const box = el('#gen-camcomp');
      if (box && !box.checked) box.checked = true;
      const tip = el('#gen-preview-steer-tip');
      if (tip) tip.textContent = 'Stroke preview: enabled camera motion compensation (pan-like energy).';
    }
    const pipe = el('#gen-pipeline');
    if (!pipe) return;
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
  };
  EventsOn('generate:progress', handleProgressLine);
  EventsOn('generate:tip-progress', event => {
    const request = activeAITargetRequest;
    if (!request || Number(event?.requestId) !== request.requestId
      || event?.videoPath !== request.videoPath) return;
    handleProgressLine(String(event?.line || ''));
  });

  EventsOn('generate:tip-detection', result => {
    const request = activeAITargetRequest;
    if (!request || request.videoPath !== videoPath) return;
    const selected = normalizeClass(el('#gen-ai-target-class')?.value || '');
    const expected = normalizeClass(result.expectedClass || '');
    if (Number(result.requestId) !== request.requestId
      || !el('#gen-ai-roi').checked || selected !== request.expectedClass
      || expected !== request.expectedClass
      || (result.videoPath && result.videoPath !== request.videoPath)) return;
    activeAITargetRequest = null;
    hideProgress();
    el('#gen-autoroi').disabled = false;
    el('#gen-candidates').disabled = false;
    el('#gen-nomark').disabled = false;
    pendingGenerateAfterRoi = false;
    generating = false;
    el('#gen-cancel').disabled = true;
    if (result.error || !result.match) {
      clearPendingAITarget();
      redraw();
      const wanted = labelFor(expected || selected) || expected || selected || 'target';
      const reason = result.errorCode || result.status || 'target_not_detected';
      const guidance = ['manifest_missing', 'manifest_invalid', 'class_unresolved', 'class_conflict'].includes(reason)
        ? 'Check that classes.json beside the ONNX model contains this class.'
        : 'Mark the intended point manually or train/correct more examples.';
      uiWarn(`No confirmed ${wanted} (${reason}). ${guidance} Existing region kept.`, el('#gen-status'));
      updateGenerateEnabled();
      return;
    }
    const matched = normalizeClass(result.matchedClass || '');
    if (!matched || matched !== expected || !(result.w > 0 && result.h > 0)) {
      clearPendingAITarget();
      redraw();
      uiWarn('AI result did not match the requested body point. Existing region kept; mark manually.', el('#gen-status'));
      updateGenerateEnabled();
      return;
    }
    pendingAITarget = { ...result, matchedClass: matched, expectedClass: expected };
    renderPendingAITarget();
    el('#gen-status').textContent =
      `${labelFor(matched) || matched} found with ${Math.round((result.confidence || 0) * 100)}% confidence — Apply target to use it.`;
    redraw();
    updateGenerateEnabled();
  });

  // Ergebnis der automatischen Regionssuche Apply - die ROI wird
  // genauso gesetzt, als hätte der Nutzer sie gezogen, und lässt sich
  // danach frei korrigieren.
  EventsOn('generate:autoroi', result => {
    // Drop stale finds before touching UI — a late result for video A must not
    // hide progress / unlock buttons while video B is still searching.
    if (result.videoPath && videoPath && result.videoPath !== videoPath) {
      return;
    }
    hideProgress();
    el('#gen-autoroi').disabled = false;
    el('#gen-candidates').disabled = false;
    el('#gen-nomark').disabled = false;
    if (result.error) {
      pendingGenerateAfterRoi = false;
      generating = false;
      el('#gen-cancel').disabled = true;
      updateGenerateEnabled();
      updateSceneMapButton();
      syncWorkflowSteps();
      uiError('Automatic region search: ' + result.error, el('#gen-status'));
      return;
    }
    candidates = [];
    clearPendingSeed();
    setSeedSuggestEnabled(false);
    // Everyday first choice: tip CSRT — leave 4-zone only if user opted in.
    if (!el('#gen-backend').dataset.userTouched) {
      el('#gen-backend').value = 'csrt';
      setNoMarkMotion(false);
    } else {
      syncNoMarkButton();
    }
    if (!el('#gen-profile').dataset.userTouched) {
      el('#gen-profile').value = 'standard';
    }
    roi = { x: result.x, y: result.y, w: result.w, h: result.h };
    const hasRoi2 = result.w2 > 0 && result.h2 > 0;
    if (hasRoi2) {
      // Optional contact suggestion only — do not switch to legacy Tf/Tj.
      roi2 = { x: result.x2, y: result.y2, w: result.w2, h: result.h2 };
    }
    updateRoiLabels();
    const via = result.engine === 'ai' ? 'AI detection' : 'classic auto';
    if (roi) {
      el('#gen-roi-label').textContent =
        `Region: x=${roi.x} y=${roi.y} w=${roi.w} h=${roi.h} (video pixels, ${via} found)`;
    }
    if (hasRoi2 && roi2) {
      el('#gen-roi2-label').textContent =
        `Contact area: x=${roi2.x} y=${roi2.y} w=${roi2.w} h=${roi2.h} (video pixels, ${via})`;
    }
    updateProfileUi();
    updateGenerateEnabled();
    let status = hasRoi2
      ? `Tip + contact area found (${via}) — CSRT ready. Vib still follows stroke depth until feel-decouple.`
      : `Tip region found (${via}) — CSRT + Contact is the everyday path. Correct by hand if needed.`;
    if (result.verifyWarning) {
      status += ' ⚠ ' + result.verifyWarning;
      uiWarn(result.verifyWarning, el('#gen-status'));
    }
    el('#gen-status').textContent = status + videoBatchNote;
    redraw();
    const shouldGenerate = pendingGenerateAfterRoi;
    pendingGenerateAfterRoi = false;
    if (shouldGenerate && roi) {
      generate();
    }
  });
  // Fortschritt: das Backend schickt 0-100, oder -1 wenn die Frame-Anzahl
  // des videos unknown war. In dem Fall wird ein unbestimmter
  // Balken gezeigt statt eines erfundenen Prozentwerts.
  let progressStartedAt = 0;
  const handleProgressPercent = pct => {
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
  };
  EventsOn('generate:percent', handleProgressPercent);
  EventsOn('generate:tip-percent', event => {
    const request = activeAITargetRequest;
    if (!request || Number(event?.requestId) !== request.requestId
      || event?.videoPath !== request.videoPath) return;
    handleProgressPercent(Number(event?.percent));
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

  el('#gen-improve-apply')?.addEventListener('click', async () => {
    if (!lastOutputPath) return;
    const status = el('#gen-improve-status');
    const btn = el('#gen-improve-apply');
    btn.disabled = true;
    status.textContent = 'Improving…';
    try {
      const audioOn = !!el('#gen-improve-audio')?.checked;
      // Keep generate-time hidden flag in sync for any re-run.
      if (el('#gen-audio-check')) el('#gen-audio-check').checked = audioOn;
      const result = await ImproveGeneratedScript({
        path: lastOutputPath,
        videoPath: videoPath || '',
        startSec: parseFloat(el('#gen-improve-start')?.value) || 0,
        endSec: parseFloat(el('#gen-improve-end')?.value) || 0,
        fillGaps: !!el('#gen-improve-fill')?.checked,
        maxGapMs: 0,
        audioCheck: audioOn,
        useAudioForFill: !!el('#gen-improve-audio-fill')?.checked,
      });
      status.textContent = result.message || 'Done';
      if (result.audioWarnings && result.audioWarnings.length) {
        status.textContent += ' — ' + result.audioWarnings[0];
      }
      el('#gen-status').textContent =
        `Improved: ${result.afterCount} points` +
        (result.pointsAdded ? ` (+${result.pointsAdded} fill)` : '') +
        (result.trimmed ? ', trimmed' : '') +
        ' — open Play to edit dots/curve.';
      const reloadPath = result.path || lastOutputPath;
      if (reloadPath && playback && typeof playback.loadScriptPath === 'function') {
        await playback.loadScriptPath(reloadPath, { review: true });
        await loadGenCurveFromPlay();
      }
    } catch (err) {
      status.textContent = 'Improve failed: ' + err;
      uiError('Improve script: ' + err, el('#gen-status'));
    } finally {
      btn.disabled = false;
    }
  });

  EventsOn('generate:done', result => {
    // Ignore stale cancel from a superseded run (new Create already active).
    if (result && result.cancelled && !userCancelRequested) {
      return;
    }
    if (typeof result?.seq === 'number') {
      if (result.seq < activeCreateSeq) return;
      activeCreateSeq = result.seq;
    }
    userCancelRequested = false;
    hideProgress();
    el('#gen-cancel').disabled = true;
    generating = false;
    lastOutputPath = result.path || null;
    el('#gen-fb-status').textContent = '';
    el('#gen-improve-status').textContent = '';
    el('#gen-feedback').style.display = lastOutputPath ? 'block' : 'none';
    el('#gen-improve').style.display = lastOutputPath ? 'block' : 'none';
    updateGenerateEnabled();
    updateSceneMapButton();
    syncWorkflowSteps();
    if (result.error) {
      if (result.cancelled) {
        el('#gen-status').textContent = 'Canceled.';
        return;
      }
      el('#gen-status').textContent = 'Failed: ' + result.error;
      el('#gen-improve').style.display = 'none';
      return;
    }
    el('#gen-status').textContent = 'Done — Emotion Script ready in Play.';
    const pipe = el('#gen-pipeline');
    if (pipe) {
      pipe.textContent = '';
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
      if (result.trackingGapCount > 0 || result.trackingReason) {
        html += `<div style="margin-top:8px; padding-top:8px; border-top:1px solid var(--border);">`
          + `<b>Tracking:</b> `;
        const bits = [];
        if (result.trackingReason === 'tracker_lost_heavy') {
          bits.push('heavy tracker loss — Tip/Partner often lost; contact feel may mute in gaps');
        } else if (result.trackingReason === 'tracker_lost_elevated') {
          bits.push('elevated tracker loss — some Tip/Partner gaps');
        } else if (result.trackingReason) {
          bits.push(String(result.trackingReason));
        }
        if (result.trackingGapCount > 0) {
          bits.push(`${result.trackingGapCount} gap window(s) (contact vib muted there on Tf/Tj)`);
        }
        html += bits.join(' · ') + `</div>`;
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

    // Right after Generate: fill gaps (auto + tighter second pass in Go),
    // then open Play with dots. Review Improve can re-run with trim/audio.
    (async () => {
      let path = result.path;
      try {
        el('#gen-status').textContent += ' — filling gaps…';
        if (el('#gen-improve-status')) {
          el('#gen-improve-status').textContent = 'Auto fill-gaps after generate…';
        }
        // Force fill on; honor Review audio toggles (defaults checked).
        if (el('#gen-improve-fill')) el('#gen-improve-fill').checked = true;
        const polished = await ImproveGeneratedScript({
          path,
          videoPath: videoPath || '',
          startSec: 0,
          endSec: 0,
          fillGaps: true,
          maxGapMs: 0, // auto + second pass @ 400ms
          audioCheck: !!(el('#gen-improve-audio')?.checked || el('#gen-audio-check')?.checked),
          useAudioForFill: el('#gen-improve-audio-fill')
            ? !!el('#gen-improve-audio-fill').checked
            : true,
        });
        if (polished && polished.path) path = polished.path;
        const msg = (polished && polished.message) || 'Gaps checked';
        if (polished && polished.pointsAdded > 0) {
          el('#gen-status').textContent += ` (+${polished.pointsAdded} fill)`;
        } else {
          el('#gen-status').textContent += ' (gaps ok)';
        }
        if (el('#gen-improve-status')) {
          el('#gen-improve-status').textContent = 'After generate: ' + msg;
        }
      } catch (err) {
        // Non-fatal — still open Play with the raw generate output.
        console.warn('post-generate fill gaps:', err);
        if (el('#gen-improve-status')) {
          el('#gen-improve-status').textContent = 'Auto fill skipped: ' + err;
        }
      }
      lastOutputPath = path;
      el('#gen-status').textContent += ' — loaded in Play (dots + Edit curve).';
      try {
        await playback.loadScriptPath(path, { review: true });
        await loadGenCurveFromPlay();
      } catch (err) {
        console.warn('post-generate Play/overlay:', err);
        playback.loadScriptPath(path, { review: true });
      }
    })();
  });

  el('#gen-pos-overlay-toggle')?.addEventListener('change', () => {
    const on = !!el('#gen-pos-overlay-toggle').checked;
    try { localStorage.setItem(POS_OVERLAY_PREF, on ? '1' : '0'); } catch (_) { /* ignore */ }
    setPosOverlayVisible(on);
    if (on) updatePosOverlay();
  });
  // Restore toggle preference once DOM is ready.
  posOverlayWanted();
  setPosOverlayVisible(false);
  el('#gen-choose').addEventListener('click', chooseVideo);
  el('#gen-check-deps').addEventListener('click', checkDeps);
  el('#gen-generate').addEventListener('click', generate);
  el('#gen-scene-map')?.addEventListener('click', runSceneMapScan);
  SceneMapAvailable().then((ok) => {
    sceneMapAvailable = !!ok;
    updateSceneMapButton();
  }).catch(() => {
    sceneMapAvailable = false;
    updateSceneMapButton();
  });
  el('#gen-seek-btn').addEventListener('click', () => seekTo(parseFloat(el('#gen-seek').value) || 0));
  el('#gen-seek-plus').addEventListener('click', () => seekTo(seekSec + 1));
  el('#gen-seek-plus5').addEventListener('click', () => seekTo(seekSec + 5));
  el('#gen-roi2-toggle').addEventListener('click', () => {
    setRoi2Mode(!roi2Mode);
    if (roi2Mode) {
      el('#gen-status').textContent = 'Contact area: drag on preview (gold).';
    }
  });
  el('#gen-target-add')?.addEventListener('click', () => {
    if (markMode === 'target') {
      setMarkMode(null);
      return;
    }
    setMarkMode('target');
    el('#gen-status').textContent = 'Another contact area: drag on preview (magenta).';
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
    normalizeProductProfile();
    updateProfileUi();
  });
  el('#gen-contact-vibration').addEventListener('change', () => {
    contactUserOverride = true;
    updateContactVibrationOpts();
  });
  el('#gen-capture-trajectory')?.addEventListener('change', () => {
    el('#gen-capture-trajectory').dataset.userTouched = '1';
  });
  el('#gen-contact-curve').addEventListener('change', () => {
    el('#gen-contact-curve').dataset.userTouched = '1';
  });
  el('#gen-contact-span').addEventListener('input', updateContactSpanLabel);
  updateContactSpanLabel();
  el('#gen-backend').addEventListener('change', () => {
    el('#gen-backend').dataset.userTouched = '1';
    syncNoMarkButton();
    normalizeProductProfile();
    updateGenerateEnabled();
  });
  el('#gen-autoroi').addEventListener('click', () => {
    if (!videoPath) return;
    pendingGenerateAfterRoi = false;
    startAutoFindRegion();
  });

  el('#gen-candidates').addEventListener('click', () => {
    if (!videoPath) return;
    setNoMarkMotion(false);
    el('#gen-candidates').disabled = true;
    el('#gen-autoroi').disabled = true;
    el('#gen-nomark').disabled = true;
    setSeedSuggestEnabled(false);
    clearPendingSeed();
    el('#gen-status').textContent = 'Finding motion candidates (nothing applied until you click / Apply)…';
    SuggestROICandidates(videoPath);
  });

  el('#gen-seed-suggest')?.addEventListener('click', () => {
    if (candidates.length < 2) {
      el('#gen-status').textContent = 'Need at least two motion candidates for Tip+2nd.';
      return;
    }
    const pair = suggestTipPartnerPair(candidates);
    if (!pair) {
      el('#gen-status').textContent = 'Could not form a Tip+2nd pair — pick by hand.';
      return;
    }
    pendingSeed = pair;
    renderPendingSeedStatus();
    el('#gen-status').textContent =
      `Tip #${pair.tip.index} + 2nd #${pair.partner.index} suggested — Apply to seed, or Dismiss. Tip alone = Everyday.`;
    redraw();
  });

  el('#gen-nomark').addEventListener('click', () => {
    if (!videoPath) return;
    setNoMarkMotion(!isNoMarkMotion());
  });

  EventsOn('generate:roi-candidates', result => {
    hideProgress();
    el('#gen-candidates').disabled = false;
    el('#gen-autoroi').disabled = false;
    el('#gen-nomark').disabled = false;
    if (result.error) {
      uiError('Motion candidates: ' + result.error, el('#gen-status'));
      setSeedSuggestEnabled(false);
      clearPendingSeed();
      return;
    }
    const list = Array.isArray(result.candidates) ? result.candidates : [];
    candidates = list.map((c, i) => ({
      x: c.x, y: c.y, w: c.w, h: c.h,
      score: c.score || 0,
      index: c.index || (i + 1),
      class: normalizeClass(c.class || c.label || ''),
    }));
    clearPendingSeed();
    if (!candidates.length) {
      setSeedSuggestEnabled(false);
      el('#gen-status').textContent = 'No motion candidates — mark primary by hand.';
      redraw();
      return;
    }
    setSeedSuggestEnabled(candidates.length >= 2);
    el('#gen-status').textContent = candidates.length >= 2
      ? `${candidates.length} motion candidates — click Tip; optional Shift-click 2nd body-part or Suggest Tip+2nd (Apply). Tip alone = Everyday.`
      : `1 motion candidate — click to set Tip. Contact mark never auto-filled.`;
    redraw();
  });

  const PROFILE_VALUES = ['standard', 'weich', 'autotune'];

  el('#gen-suggest-profile').addEventListener('click', async () => {
    if (!videoPath) return;
    const requestSeq = ++profileSuggestionSeq;
    const requestPath = videoPath;
    const status = el('#gen-suggest-status');
    status.textContent = 'Comparing to saved scenes…';
    el('#gen-suggest-profile').disabled = true;
    try {
      const result = await SuggestProfile(videoPath);
      if (requestSeq !== profileSuggestionSeq || requestPath !== videoPath) return;
      if (!result.found) {
        status.textContent = 'No suggestion (no similar saved scene, AI server unreachable).';
        return;
      }
      const via = result.kind === 'local_model'
        ? `lokales Go-Modell, Konfidenz ${Math.round(result.confidence * 100)}%`
        : result.kind === 'ai'
          ? `KI, Konfidenz ${Math.round(result.confidence * 100)}%`
          : `gemessen, Abstand ${result.confidence.toFixed(3)}`;
      // Legacy "tf"/"tj" scene labels map to Stroke — Contact vib is the feel layer now.
      let label = result.label === 'tj' || result.label === 'tf' ? 'standard' : result.label;
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
      if (requestSeq === profileSuggestionSeq) status.textContent = 'Error: ' + err;
    } finally {
      if (requestSeq === profileSuggestionSeq) el('#gen-suggest-profile').disabled = false;
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
      const profile = el('#gen-profile').value || 'standard';
      await LabelSceneWithProfile(videoPath, label, profile);
      el('#gen-suggest-status').textContent = `Scene "${label}" saved with Style “${profile}”. `
        + 'Retrain the Go profile model in AI training to include it.';
    } catch (err) {
      uiError('Remember scene: ' + err, el('#gen-suggest-status'));
    } finally {
      el('#gen-label-scene').disabled = false;
    }
  });

  // Body-part class selects (docs/BODY_REGIONS.md) — tip-first / contact-first.
  const fillClassSelect = (selId, preferIds) => {
    const sel = el(selId);
    if (!sel) return;
    for (const p of orderedCanonical(preferIds)) {
      const opt = document.createElement('option');
      opt.value = p.id;
      opt.textContent = p.label;
      sel.appendChild(opt);
    }
  };
  fillClassSelect('#gen-region-class', TIP_CLASS_ORDER);
  fillClassSelect('#gen-region-class2', CONTACT_CLASS_ORDER);
  fillClassSelect('#gen-target-class', CONTACT_CLASS_ORDER);
  fillClassSelect('#gen-ai-target-class', TIP_CLASS_ORDER);
  el('#gen-ai-roi')?.addEventListener('change', () => {
    const enabled = el('#gen-ai-roi').checked && !el('#gen-ai-roi').disabled;
    const target = el('#gen-ai-target-class');
    if (target) target.disabled = !enabled;
    cancelActiveAITargetRequest();
    clearPendingAITarget();
    redraw();
    hideProgress();
    if (videoPath) {
      el('#gen-autoroi').disabled = false;
      el('#gen-candidates').disabled = false;
    }
    el('#gen-autoroi-hint').textContent = enabled
      ? 'Choose the exact expected body point on the displayed frame. AI fails closed instead of choosing another class.'
      : 'Generic motion search is active. Enable AI plus an expected body point for strict matching.';
  });
  el('#gen-ai-target-class')?.addEventListener('change', () => {
    cancelActiveAITargetRequest();
    clearPendingAITarget();
    redraw();
    hideProgress();
    if (videoPath) {
      el('#gen-autoroi').disabled = false;
      el('#gen-candidates').disabled = false;
    }
    const cls = normalizeClass(el('#gen-ai-target-class').value || '');
    el('#gen-ai-target-status').textContent = cls
      ? `Strict target: ${labelFor(cls) || cls}. Press Find tip area.`
      : 'Choose an expected body point before strict AI detection.';
  });
  el('#gen-region-class')?.addEventListener('change', () => {
    const sel = el('#gen-region-class');
    if (sel) sel.dataset.userTouched = '1';
  });
  el('#gen-region-class2')?.addEventListener('change', () => {
    const sel = el('#gen-region-class2');
    if (sel) {
      sel.dataset.userTouched = '1';
      sel.style.outline = '';
    }
    if (roi2) {
      el('#gen-roi2-label').textContent = secondRegionLabel(roi2);
      redraw();
    }
  });
  el('#gen-target-class')?.addEventListener('change', () => {
    const sel = el('#gen-target-class');
    if (sel) sel.dataset.userTouched = '1';
  });
  el('#gen-roi2-fixed')?.addEventListener('change', () => {
    el('#gen-roi2-fixed').dataset.userTouched = '1';
  });

  updateContactVibrationOpts();
  syncWorkflowSteps();
}
