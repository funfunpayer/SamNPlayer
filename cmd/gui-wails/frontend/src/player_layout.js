// Player layout v1 (Owner's final mockup, docs/V1_PLAN.md S3).
//
// Arranges the existing Player controls into the 1.0 shell:
//   Library (left) · video + transport + bottom tabs (centre) · Inspector (right)
// Nothing is re-implemented here: existing elements are *moved* into place,
// so every id, listener and binding stays the same (playback.js queries are
// scoped to the Player root, which still contains everything). Panels are
// switched with classes; the real stylesheet hides inactive ones, so tests
// that run without CSS still reach every control.

const RECENT_KEY = 'pb.recent';
const RECENT_MAX = 8;
const INSPECTOR_KEY = 'pb.inspectorTab';
const BOTTOM_KEY = 'pb.bottomTab';

function store(key, value) { try { localStorage.setItem(key, value); } catch (_) { /* ignore */ } }
function load(key) { try { return localStorage.getItem(key); } catch (_) { return null; } }

export function readRecent() {
  try {
    const list = JSON.parse(load(RECENT_KEY) || '[]');
    return Array.isArray(list) ? list.filter(p => typeof p === 'string' && p) : [];
  } catch (_) { return []; }
}

export function pushRecent(path) {
  if (!path) return readRecent();
  const list = [path, ...readRecent().filter(p => p !== path)].slice(0, RECENT_MAX);
  store(RECENT_KEY, JSON.stringify(list));
  return list;
}

function baseName(p) { return String(p).split(/[\\/]/).pop(); }

function icon(path) {
  return `<svg viewBox="0 0 24 24" aria-hidden="true">${path}</svg>`;
}

const ICONS = {
  folder: icon('<path d="M3.5 7.2a1.7 1.7 0 0 1 1.7-1.7h4.1l2 2.1h7.5a1.7 1.7 0 0 1 1.7 1.7v8.2a1.7 1.7 0 0 1-1.7 1.7H5.2a1.7 1.7 0 0 1-1.7-1.7z"/>'),
  recent: icon('<circle cx="12" cy="12" r="8"/><path d="M12 7.5V12l3 2"/>'),
  list: icon('<path d="M8 7h11M8 12h11M8 17h11"/><circle cx="4.5" cy="7" r="1"/><circle cx="4.5" cy="12" r="1"/><circle cx="4.5" cy="17" r="1"/>'),
  map: icon('<path d="M12 21s-6.2-5.6-6.2-10.6a6.2 6.2 0 0 1 12.4 0C18.2 15.4 12 21 12 21z"/><circle cx="12" cy="10.4" r="2.3"/>'),
  wave: icon('<path d="M3 12h2l2-6 3 12 3-9 2 5 2-2h4"/>'),
  bpm: icon('<path d="M5 20V10M10 20V5M15 20v-7M20 20V8"/>'),
  drop: icon('<path d="M12 4v10M7.5 9.5 12 14l4.5-4.5"/><path d="M5 17.5v1.5h14v-1.5"/>'),
};

/**
 * @param {HTMLElement} root  Player root (#tab-playback)
 * @param {{ loadScriptPath: (p: string) => Promise<void> }} api
 */
export function applyPlayerLayoutV1(root, api) {
  if (!root || root.dataset.pbLayout === 'v1') return;
  root.dataset.pbLayout = 'v1';
  const q = sel => root.querySelector(sel);

  // ---- Shell: library column + main column -------------------------------
  const main = document.createElement('div');
  main.className = 'pb-main-col';
  main.dataset.pbRegion = 'main';
  for (const sel of ['.pb-head', '#pb-empty', '#pb-loaded']) {
    const n = q(sel);
    if (n) main.appendChild(n);
  }

  const lib = document.createElement('aside');
  lib.className = 'pb-library';
  lib.dataset.pbRegion = 'library';
  lib.setAttribute('aria-label', 'Library');
  lib.innerHTML = `
    <p class="pb-lib-title">Library</p>
    <button type="button" class="pb-lib-item is-primary" id="pb-lib-local">${ICONS.folder}<span>Local Files</span></button>
    <button type="button" class="pb-lib-item" id="pb-lib-recent-btn" aria-expanded="true">${ICONS.recent}<span>Recent</span></button>
    <ol class="pb-lib-recent" id="pb-lib-recent"></ol>
    <button type="button" class="pb-lib-item" id="pb-lib-playlist">${ICONS.list}<span>Playlists</span><b class="pb-lib-count" id="pb-lib-playlist-count"></b></button>
    <p class="pb-lib-title">Tools</p>
    <button type="button" class="pb-lib-item" id="pb-lib-scenemap">${ICONS.map}<span>Scene Map</span></button>
    <button type="button" class="pb-lib-item" id="pb-lib-audio">${ICONS.wave}<span>Audio Analysis</span></button>
    <button type="button" class="pb-lib-item" id="pb-lib-bpm">${ICONS.bpm}<span>BPM Detection</span></button>
    <p class="pb-lib-title">Quick Access</p>
    <div class="pb-lib-drop" id="pb-lib-drop">${ICONS.drop}<span>Drag files here</span>
      <small>Video → Generator · Script → Player</small></div>
  `;
  root.prepend(lib);
  lib.after(main);

  // ---- Inspector (right): tabs Devices · Audio · Playback · View ----------
  const tools = q('#pb-tools');
  const head = tools.querySelector('.pb-tools-head');
  const title = head?.querySelector('.pb-tools-title');
  if (title) title.textContent = 'Inspector';
  const tabs = [['devices', 'Devices'], ['audio', 'Audio'], ['playback', 'Playback'], ['view', 'View']];
  const bar = document.createElement('div');
  bar.className = 'pb-insp-tabs';
  bar.dataset.pbRegion = 'inspector-tabs';
  bar.setAttribute('role', 'tablist');
  bar.innerHTML = tabs.map(([k, label]) =>
    `<button type="button" role="tab" class="pb-insp-tab" data-insp="${k}" id="pb-insp-tab-${k}">${label}</button>`).join('');
  const panels = {};
  for (const [k] of tabs) {
    const p = document.createElement('div');
    p.className = 'pb-insp-panel';
    p.dataset.inspPanel = k;
    p.id = `pb-insp-${k}`;
    p.setAttribute('role', 'tabpanel');
    p.setAttribute('aria-labelledby', `pb-insp-tab-${k}`);
    panels[k] = p;
  }
  // Devices: live channel rows (real values mirrored from the transport).
  const live = document.createElement('div');
  live.className = 'pb-chan-rows';
  live.innerHTML = [['stroke', 'Stroke', '#pb-pos-value'], ['vibration', 'Vibration', '#pb-vib'], ['suction', 'Suction', '#pb-suc']]
    .map(([k, label]) => `<div class="pb-chan-row" data-axis="${k}">
        <i class="pb-chan-dot"></i><span class="pb-chan-name">${label}</span>
        <span class="pb-chan-bar"><b id="pb-chan-${k}-fill"></b></span>
        <span class="pb-chan-val" id="pb-chan-${k}-val">–</span></div>`).join('');
  panels.devices.appendChild(live);

  const move = (sel, panel) => {
    const n = typeof sel === 'string' ? q(sel) : sel;
    if (n) panels[panel].appendChild(n);
    return n;
  };
  // Devices: script → device mapping.
  ['#pb-axis-row', '#pb-optimize-neo2-status', '#pb-export-bake-details', '#pb-contact-block'].forEach(s => move(s, 'devices'));
  // Audio: audio vibration + BPM card.
  const bpmCard = document.createElement('div');
  bpmCard.className = 'pb-card';
  bpmCard.id = 'pb-bpm-card';
  bpmCard.innerHTML = '<p class="pb-card-title">BPM</p><div class="pb-card-body"></div>';
  const bpmBody = bpmCard.querySelector('.pb-card-body');
  for (const n of [q('#pb-bpm')?.closest('label'), q('#pb-bpm-auto'), q('#pb-bpm-auto-status'), q('#pb-bpm-grid')?.closest('label')]) {
    if (n) bpmBody.appendChild(n);
  }
  const avCard = document.createElement('div');
  avCard.className = 'pb-card';
  avCard.id = 'pb-audio-vib-card';
  avCard.innerHTML = '<p class="pb-card-title">Audio Vibration</p><div class="pb-card-body"></div>';
  for (const s of ['#pb-audio-vibration-row', '#pb-audio-vibration-hint']) {
    const n = q(s);
    if (n) avCard.querySelector('.pb-card-body').appendChild(n);
  }
  panels.audio.append(bpmCard, avCard);
  // Playback: script tools (offset), playback settings, checks, log.
  const st = document.createElement('div');
  st.className = 'pb-card';
  st.id = 'pb-script-tools-card';
  st.innerHTML = '<p class="pb-card-title">Script Tools</p><div class="pb-card-body"></div>';
  for (const s of ['#pb-offset-row', '#pb-offset-hint']) {
    const n = q(s);
    if (n) st.querySelector('.pb-card-body').appendChild(n);
  }
  panels.playback.appendChild(st);
  const adv = [...tools.querySelectorAll(':scope > details.pb-advanced')][0];
  if (adv) panels.playback.appendChild(adv);
  ['#pb-quality-details', '#pb-ofs-row', '#pb-marker-auto-row', '#pb-log'].forEach(s => move(s, 'playback'));
  // View: curve display / editing.
  ['#pb-curve-edit-row', '#pb-curve-edit-hint', '#pb-pos-overlay-row', '#pb-ofs-edit-details',
    '#pb-marker-hint', '#pb-omarker-details'].forEach(s => move(s, 'view'));
  // Anything left (future additions) lands in Playback so nothing is lost.
  [...tools.children].forEach(n => {
    if (n !== head && !n.classList.contains('pb-insp-panel')) panels.playback.appendChild(n);
  });
  tools.append(bar, ...Object.values(panels));

  function showInspector(k) {
    if (!panels[k]) k = 'devices';
    for (const [key, p] of Object.entries(panels)) p.classList.toggle('is-active', key === k);
    bar.querySelectorAll('[data-insp]').forEach(b => {
      const on = b.dataset.insp === k;
      b.classList.toggle('is-active', on);
      b.setAttribute('aria-selected', on ? 'true' : 'false');
      b.tabIndex = on ? 0 : -1;
    });
    root.dataset.pbInspector = k;
    store(INSPECTOR_KEY, k);
  }
  bar.addEventListener('click', e => {
    const b = e.target.closest('[data-insp]');
    if (b) showInspector(b.dataset.insp);
  });
  bar.addEventListener('keydown', e => {
    const keys = tabs.map(t => t[0]);
    const i = keys.indexOf(root.dataset.pbInspector);
    let j = i;
    if (e.key === 'ArrowRight') j = (i + 1) % keys.length;
    else if (e.key === 'ArrowLeft') j = (i - 1 + keys.length) % keys.length;
    else return;
    e.preventDefault();
    showInspector(keys[j]);
    bar.querySelector(`[data-insp="${keys[j]}"]`)?.focus();
  });
  showInspector(load(INSPECTOR_KEY) || 'devices');

  // ---- Bottom tabs: Timeline · Chapters · Bookmarks · Scene Map · Audio Analysis
  const timeline = q('.pb-timeline');
  const bottom = document.createElement('div');
  bottom.className = 'pb-bottom';
  bottom.dataset.pbRegion = 'bottom';
  const btabs = [['timeline', 'Timeline'], ['chapters', 'Chapters'], ['bookmarks', 'Bookmarks'],
    ['scenemap', 'Scene Map'], ['audio', 'Audio Analysis']];
  const bbar = document.createElement('div');
  bbar.className = 'pb-bottom-tabs';
  bbar.dataset.pbRegion = 'bottom-tabs';
  bbar.setAttribute('role', 'tablist');
  bbar.innerHTML = btabs.map(([k, label]) =>
    `<button type="button" role="tab" class="pb-bottom-tab" data-btab="${k}" id="pb-btab-${k}">${label}</button>`).join('');
  timeline.before(bottom);
  bottom.appendChild(bbar);
  const bpanels = {};
  const mk = k => {
    const p = document.createElement('div');
    p.className = 'pb-bottom-panel';
    p.dataset.btabPanel = k;
    p.setAttribute('role', 'tabpanel');
    bottom.appendChild(p);
    bpanels[k] = p;
    return p;
  };
  mk('timeline').appendChild(timeline);
  const openDetails = (sel, k) => {
    const d = q(sel);
    if (!d) return;
    d.open = true;
    d.classList.add('pb-in-bottom');
    // As a bottom tab the section has no summary to reopen it, and Play
    // closes these panels when the selection clears — keep it open.
    d.addEventListener('toggle', () => { if (!d.open) d.open = true; });
    mk(k).appendChild(d);
  };
  openDetails('#pb-chapter-details', 'chapters');
  openDetails('#pb-bookmark-details', 'bookmarks');
  const segs = q('#pb-audio-segments');
  if (segs) mk('audio').appendChild(segs);
  function showBottom(k) {
    if (k === 'scenemap') {
      document.querySelector('#rail .tab-btn[data-tab="scenemap"]')?.click();
      return;
    }
    if (!bpanels[k]) k = 'timeline';
    for (const [key, p] of Object.entries(bpanels)) p.classList.toggle('is-active', key === k);
    bbar.querySelectorAll('[data-btab]').forEach(b => {
      const on = b.dataset.btab === k;
      b.classList.toggle('is-active', on);
      b.setAttribute('aria-selected', on ? 'true' : 'false');
    });
    root.dataset.pbBottom = k;
    store(BOTTOM_KEY, k);
    // Canvases inside a re-shown panel redraw via their ResizeObserver.
  }
  bbar.addEventListener('click', e => {
    const b = e.target.closest('[data-btab]');
    if (b) showBottom(b.dataset.btab);
  });
  showBottom(load(BOTTOM_KEY) || 'timeline');

  // ---- Library actions ----------------------------------------------------
  q('#pb-lib-local').addEventListener('click', () => q('#pb-choose')?.click());
  q('#pb-lib-scenemap').addEventListener('click', () => showBottom('scenemap'));
  q('#pb-lib-audio').addEventListener('click', () => showBottom('audio'));
  q('#pb-lib-bpm').addEventListener('click', () => {
    showInspector('audio');
    q('#pb-bpm-auto')?.focus();
  });
  q('#pb-lib-playlist').addEventListener('click', () => {
    const pl = q('#pb-playlist');
    if (pl && !pl.hidden) pl.scrollIntoView({ block: 'nearest' });
    else q('#pb-queue-add')?.focus();
  });
  const recentList = q('#pb-lib-recent');
  function renderRecent() {
    const list = readRecent();
    recentList.innerHTML = list.length
      ? list.map((p, i) => `<li><button type="button" class="pb-lib-recent-item" data-i="${i}" title="${p.replace(/"/g, '&quot;')}">${baseName(p)}</button></li>`).join('')
      : '<li class="pb-lib-empty">Nothing yet</li>';
  }
  recentList.addEventListener('click', async e => {
    const b = e.target.closest('.pb-lib-recent-item');
    if (!b) return;
    const p = readRecent()[Number(b.dataset.i)];
    if (p) await api.loadScriptPath(p).catch(() => {});
  });
  q('#pb-lib-recent-btn').addEventListener('click', () => {
    const open = recentList.hidden;
    recentList.hidden = !open;
    q('#pb-lib-recent-btn').setAttribute('aria-expanded', open ? 'true' : 'false');
  });
  window.addEventListener('pb:loaded', e => {
    pushRecent(e.detail && e.detail.path);
    renderRecent();
  });
  renderRecent();
  const plCount = q('#pb-lib-playlist-count');
  const syncCount = () => {
    const n = q('#pb-playlist-list')?.children.length || 0;
    plCount.textContent = n > 1 ? String(n) : '';
  };
  const plList = q('#pb-playlist-list');
  if (plList) new MutationObserver(syncCount).observe(plList, { childList: true });
  syncCount();

  // ---- Live channel rows: mirror real values ------------------------------
  const mirror = (srcSel, k, scale) => {
    const src = q(srcSel);
    if (!src) return;
    const fill = q(`#pb-chan-${k}-fill`);
    const val = q(`#pb-chan-${k}-val`);
    const sync = () => {
      const n = parseFloat(String(src.textContent).replace(',', '.'));
      const pct = Number.isFinite(n) ? Math.max(0, Math.min(100, n * scale)) : null;
      fill.style.width = pct == null ? '0%' : pct + '%';
      val.textContent = pct == null ? '–' : Math.round(pct) + '%';
    };
    new MutationObserver(sync).observe(src, { childList: true, characterData: true, subtree: true });
    sync();
  };
  // All three show 0–100 already (#pb-vib / #pb-suc as "58%").
  mirror('#pb-pos-value', 'stroke', 1);
  mirror('#pb-vib', 'vibration', 1);
  mirror('#pb-suc', 'suction', 1);

  return { showInspector, showBottom, renderRecent };
}
