// Video first (Owner 14 Oct): "Open…" in the Player (and Editor) takes a
// video or a script, and so does a drop on the window. A video finds its
// script by itself (same name beside it, FindScriptForVideo); without one the
// empty Player shows the video with "Create script" — the Generator takes the
// same video — or "Choose script…". "Always create right away" skips the
// question.
import { PickMediaFile, PickFunscriptFile, FindScriptForVideo, LoadFrameAt } from '../wailsjs/go/main/App';

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
      <button type="button" id="pb-noscript-choose">Choose script…</button>
      <button type="button" id="pb-noscript-other">Open other…</button>
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
  card.querySelector('#pb-noscript-choose').addEventListener('click', async () => {
    const video = pending;
    let script = '';
    try { script = await PickFunscriptFile(); } catch (_) { script = ''; }
    if (script) await openPair(video, script);
  });
  card.querySelector('#pb-noscript-other').addEventListener('click', pickAndOpen);
  card.querySelector('#pb-noscript-back').addEventListener('click', clearNoScript);
  for (const id of ['#pb-choose', '#pb-choose-empty']) {
    pbRoot.querySelector(id)?.addEventListener('click', pickAndOpen);
  }
  // A script that loads ends the question.
  window.addEventListener('pb:loaded', () => { if (pending) clearNoScript(); });

  return { openPath, openPair, pickAndOpen, showNoScript, clearNoScript };
}
