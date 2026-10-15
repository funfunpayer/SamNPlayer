// Video first (Owner 14 Oct): "Open…" in the Player (and Editor) takes a
// video or a script, and so does a drop on the window. A video finds its
// script by itself (same name beside it, FindScriptForVideo); without one the
// empty Player shows the video with "Create script" — the Generator takes the
// same video — or "Choose script…". "Always create right away" skips the
// question.
//
// v0.8: the search also covers the script folders (Settings → Play) and
// tolerant names; when one video has several scripts (Soft / Hard / another
// author) a "Script" choice in the video bar switches between them.
//
// Cock Hero / Fap Hero (Owner 15 Oct): "From the beat bar" reads the strip of
// scrolling dots such videos show and writes one stroke per beat, as
// "<video> (Beat bar).funscript" beside the video.
import { PickMediaFile, PickFunscriptFile, FindScriptForVideo, LoadFrameAt, AddScriptFolder, CreateBeatBarScript } from '../wailsjs/go/main/App';

const SCRIPT_RE = /\.(samn|funscript)$/i;
const AUTO_KEY = 'open.autoCreate';
const baseName = p => String(p || '').split(/[\\/]/).pop();

function readAuto() {
  try { return localStorage.getItem(AUTO_KEY) === '1'; } catch (_) { return false; }
}
function writeAuto(on) {
  try { localStorage.setItem(AUTO_KEY, on ? '1' : '0'); } catch (_) { /* ignore */ }
}

export function initOpenMedia(pbRoot, api) {
  if (!pbRoot || !api) return null;
  const empty = pbRoot.querySelector('#pb-empty');
  const inner = empty && empty.querySelector('.pb-empty-inner');
  if (!inner) return null;
  pbRoot.dataset.openMedia = '1';

  // ---- "No script yet" card (inside the empty Player) ----------------------
  const card = document.createElement('div');
  card.className = 'pb-noscript';
  card.id = 'pb-noscript';
  card.hidden = true;
  card.innerHTML = `
    <div class="pb-noscript-frame"><img id="pb-noscript-img" alt="" hidden /></div>
    <p class="pb-empty-kicker">New video</p>
    <p class="pb-noscript-name" id="pb-noscript-name"></p>
    <p class="hint">No script for this video yet. SamNPlayer can create one — in the Generator, with this video.</p>
    <div class="pb-noscript-actions">
      <button type="button" class="primary pb-open-big" id="pb-noscript-create">Create script</button>
      <button type="button" id="pb-noscript-beatbar" title="Cock Hero / Fap Hero: read the strip of dots scrolling to a target and make one stroke per beat">From the beat bar</button>
      <button type="button" id="pb-noscript-choose">Choose script…</button>
      <button type="button" id="pb-noscript-other">Open other…</button>
      <button type="button" id="pb-noscript-folder" title="Add a folder where your scripts live; SamNPlayer looks there too">Add a script folder…</button>
      <button type="button" id="pb-noscript-back" hidden>Back to the script</button>
    </div>
    <div class="checkbox-row pb-noscript-auto">
      <input type="checkbox" id="pb-noscript-auto" />
      <label for="pb-noscript-auto">Always create right away when a video has no script</label>
    </div>
    <p class="hint" id="pb-noscript-status" aria-live="polite"></p>`;
  inner.after(card);
  const status = card.querySelector('#pb-noscript-status');
  const auto = card.querySelector('#pb-noscript-auto');
  auto.checked = readAuto();
  auto.addEventListener('change', () => writeAuto(auto.checked));

  const loadedEl = pbRoot.querySelector('#pb-loaded');
  const isLoaded = () => pbRoot.dataset.pbState === 'loaded';
  let pending = '';
  // Shown in place of a loaded script, too: it stays loaded until the user
  // picks something ("Back to the script" returns to it).
  function showNoScript(path) {
    pending = path;
    pbRoot.dataset.pbPendingVideo = path;
    empty.dataset.noscript = '1';
    inner.hidden = true;
    card.hidden = false;
    empty.hidden = false;
    if (loadedEl) loadedEl.hidden = true;
    card.querySelector('#pb-noscript-back').hidden = !isLoaded();
    status.textContent = '';
    card.querySelector('#pb-noscript-name').textContent = baseName(path);
    const img = card.querySelector('#pb-noscript-img');
    img.hidden = true;
    img.removeAttribute('src');
    // A still from a few seconds in (the very first frame is often black).
    LoadFrameAt(path, 3).then(f => {
      if (pending !== path || !f || !f.pngBase64) return;
      img.src = 'data:image/png;base64,' + f.pngBase64;
      img.hidden = false;
    }).catch(() => {});
  }
  function clearNoScript() {
    pending = '';
    delete pbRoot.dataset.pbPendingVideo;
    delete empty.dataset.noscript;
    inner.hidden = false;
    card.hidden = true;
    empty.hidden = isLoaded();
    if (loadedEl) loadedEl.hidden = !isLoaded();
  }

  // ---- Create in the Generator ---------------------------------------------
  function createFor(path, { start = false } = {}) {
    window.dispatchEvent(new CustomEvent('ws:set', { detail: { view: 'create' } }));
    window.dispatchEvent(new CustomEvent('drop:video', { detail: { path, paths: [path], extraCount: 0 } }));
    if (!start) return;
    // Start once the Generator is ready (the button enables after the
    // automatic tip search).
    const btn = document.getElementById('gen-generate');
    if (!btn) return;
    let done = false;
    const go = () => {
      if (done || btn.disabled) return;
      done = true;
      obs.disconnect();
      btn.click();
    };
    const obs = new MutationObserver(go);
    obs.observe(btn, { attributes: true, attributeFilter: ['disabled'] });
    setTimeout(() => { done = true; obs.disconnect(); }, 120000);
    setTimeout(go, 0);
  }

  // ---- Open a path (picker, drop, Library) ---------------------------------
  async function openPath(path) {
    if (!path) return;
    if (SCRIPT_RE.test(path)) {
      clearNoScript();
      await api.openScriptPath(path);
      return;
    }
    let match = null;
    try { match = await FindScriptForVideo(path); } catch (_) { match = null; }
    if (match && match.found && match.scriptPath) {
      clearNoScript();
      await api.openScriptPath(match.scriptPath);
      // Same name, other container (clip.mkv next to clip.mp4): keep the one opened.
      if (pbRoot.dataset.pbState === 'loaded' && pbRoot.dataset.pbVideoPath !== path) {
        await api.attachVideoPath(path);
      }
      return;
    }
    if (readAuto()) {
      clearNoScript();
      createFor(path, { start: true });
      return;
    }
    if (document.body.dataset.mode === 'generator') {
      createFor(path);
      return;
    }
    // Player / Editor: the Player asks.
    showNoScript(path);
    window.dispatchEvent(new CustomEvent('ws:set', { detail: { view: document.body.dataset.mode === 'editor' ? 'edit' : 'play' } }));
  }

  // A script with a given video (both dropped together).
  async function openPair(video, script) {
    clearNoScript();
    await api.openScriptPath(script);
    if (video && pbRoot.dataset.pbState === 'loaded' && pbRoot.dataset.pbVideoPath !== video) {
      await api.attachVideoPath(video);
    }
  }

  async function pickAndOpen() {
    let path = '';
    try { path = await PickMediaFile(); } catch (err) { status.textContent = 'Open: ' + err; return; }
    if (typeof path === 'string' && path) await openPath(path);
  }

  card.querySelector('#pb-noscript-create').addEventListener('click', () => {
    const p = pending;
    if (!p) return;
    clearNoScript();
    createFor(p);
  });
  const beatBtn = card.querySelector('#pb-noscript-beatbar');
  beatBtn.addEventListener('click', async () => {
    const video = pending;
    if (!video) return;
    beatBtn.disabled = true;
    status.textContent = 'Reading the beat bar… a long video takes a minute or two.';
    let res = null;
    try {
      res = await CreateBeatBarScript(video);
    } catch (err) {
      status.textContent = 'Beat bar: ' + err;
      return;
    } finally {
      beatBtn.disabled = false;
    }
    if (pending !== video) return;
    if (res && res.found && res.scriptPath) {
      // Opens like any variant; the video bar's Script choice shows it.
      await openPair(video, res.scriptPath);
    } else {
      status.textContent = (res && res.message) || 'No beat bar found in this video.';
    }
  });
  card.querySelector('#pb-noscript-choose').addEventListener('click', async () => {
    const video = pending;
    let script = '';
    try { script = await PickFunscriptFile(); } catch (_) { script = ''; }
    if (script) await openPair(video, script);
  });
  card.querySelector('#pb-noscript-other').addEventListener('click', pickAndOpen);
  card.querySelector('#pb-noscript-back').addEventListener('click', clearNoScript);
  // A folder added here is searched right away for this video.
  card.querySelector('#pb-noscript-folder').addEventListener('click', async () => {
    const video = pending;
    let dirs = null;
    try { dirs = await AddScriptFolder(); } catch (err) { status.textContent = 'Could not add the folder: ' + err; return; }
    if (!video || !Array.isArray(dirs)) return;
    let match = null;
    try { match = await FindScriptForVideo(video); } catch (_) { match = null; }
    if (match && match.found && match.scriptPath) await openPair(video, match.scriptPath);
    else status.textContent = 'No script for this video in that folder either.';
  });

  // ---- Script variants (one video, several scripts) ------------------------
  const vbar = pbRoot.querySelector('.pb-videobar');
  const variantSel = document.createElement('select');
  variantSel.id = 'pb-variant';
  variantSel.className = 'pb-variant';
  variantSel.hidden = true;
  variantSel.title = 'This video has several scripts — choose one';
  variantSel.setAttribute('aria-label', 'Script variant');
  const recentBtn = vbar && vbar.querySelector('#pb-vb-recent');
  if (recentBtn) recentBtn.after(variantSel);
  else if (vbar) vbar.prepend(variantSel);
  const samePath = (a, b) => String(a || '').toLowerCase() === String(b || '').toLowerCase();
  let variantTimer = 0;
  let variantSeq = 0;
  async function refreshVariants() {
    const seq = ++variantSeq;
    const video = pbRoot.dataset.pbVideoPath || '';
    const script = pbRoot.dataset.pbScript || '';
    if (!video || pbRoot.dataset.pbState !== 'loaded') { variantSel.hidden = true; return; }
    let match = null;
    try { match = await FindScriptForVideo(video); } catch (_) { match = null; }
    if (seq !== variantSeq) return; // a newer load is under way
    const list = (match && Array.isArray(match.variants)) ? match.variants : [];
    if (list.length < 2) { variantSel.hidden = true; return; }
    variantSel.innerHTML = list.map((v, i) => {
      const name = v.label ? v.label.replace(/^./, c => c.toUpperCase()) : 'Main script';
      const where = v.beside ? '' : ' · script folder';
      return `<option value="${i}">${name.replace(/[<&]/g, '')}${where}</option>`;
    }).join('');
    variantSel._list = list;
    variantSel._video = video;
    const cur = list.findIndex(v => samePath(v.path, script));
    if (cur < 0) {
      const other = document.createElement('option');
      other.value = '';
      other.textContent = baseName(script) || 'Other script';
      variantSel.prepend(other);
      variantSel.value = '';
    } else {
      variantSel.value = String(cur);
    }
    variantSel.hidden = false;
  }
  variantSel.addEventListener('change', async () => {
    const v = (variantSel._list || [])[Number(variantSel.value)];
    if (v && variantSel.value !== '') await openPair(variantSel._video, v.path);
  });
  new MutationObserver(() => {
    clearTimeout(variantTimer);
    variantTimer = setTimeout(refreshVariants, 150);
  }).observe(pbRoot, { attributes: true, attributeFilter: ['data-pb-video-path', 'data-pb-script', 'data-pb-state'] });
  for (const id of ['#pb-choose', '#pb-choose-empty']) {
    pbRoot.querySelector(id)?.addEventListener('click', pickAndOpen);
  }
  // A script that loads ends the question.
  window.addEventListener('pb:loaded', () => { if (pending) clearNoScript(); });

  return { openPath, openPair, pickAndOpen, showNoScript, clearNoScript, refreshVariants };
}
