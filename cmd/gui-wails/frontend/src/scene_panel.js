// Scene Map bottom panel (V1 target scene-map.png): the loaded script's
// chapters shown as scenes — a grid with a thumbnail, name, range and
// duration per scene, and a details pane for the selected scene (range,
// n / N, prev/next, go to, the linked video's technical data).
// Real data only: no tags or notes (there is no storage for them yet).
import { ProbeCreateVideo, SuggestChapterScenes, SaveChapterScenes } from '../wailsjs/go/main/App';

const FEELS = [['', 'Classic'], ['oral', 'Oral'], ['intercourse', 'Intercourse'], ['breast', 'Breast'], ['hand', 'Hand']];
const FEEL_NAME = Object.fromEntries(FEELS);

function fmtMs(ms) {
  const t = Math.max(0, Math.round(ms / 1000));
  const h = Math.floor(t / 3600);
  const m = Math.floor((t % 3600) / 60);
  const s = t % 60;
  const mm = String(m).padStart(h ? 2 : 1, '0');
  return (h ? h + ':' : '') + mm + ':' + String(s).padStart(2, '0');
}

function baseName(p) { return String(p).split(/[\\/]/).pop(); }

function make(tag, cls, text) {
  const e = document.createElement(tag);
  if (cls) e.className = cls;
  if (text != null) e.textContent = text;
  return e;
}

// Thumbnail closest to the scene start (thumbs are [{atMs, dataUrl}]).
export function thumbFor(thumbs, startMs, endMs) {
  let best = null;
  for (const t of thumbs || []) {
    if (!t || !t.dataUrl) continue;
    const inside = t.atMs >= startMs && t.atMs < endMs;
    const d = Math.abs(t.atMs - startMs);
    const score = inside ? d : d + 1e12;
    if (!best || score < best.score) best = { score, url: t.dataUrl };
  }
  return best ? best.url : '';
}

export function mountScenePanel(panel, root, api) {
  if (!panel || !api || typeof api.chapters !== 'function') return null;
  panel.classList.add('pb-scenes');
  const head = make('div', 'pb-scenes-head');
  const count = make('span', 'pb-scenes-count');
  count.id = 'pb-scenes-count';
  const heat = make('button', 'pb-scenes-heatmap', 'Rhythm heatmap (Generator)…');
  heat.type = 'button';
  heat.id = 'pb-scenes-heatmap';
  heat.title = 'Open the Generator\'s rhythm heatmap of the video';
  heat.addEventListener('click', () => {
    window.dispatchEvent(new CustomEvent('app:rhythm'));
  });
  // Step 4: a scene feel per chapter, suggested by the scene detection or
  // the contact mark — applied only when you say so.
  const applyAll = make('button', 'pb-scenes-apply', 'Apply suggestions');
  applyAll.type = 'button';
  applyAll.id = 'pb-scenes-apply';
  applyAll.title = 'Give every scene its suggested Sam Neo 2 feel';
  applyAll.hidden = true;
  head.append(count, applyAll, heat);

  const body = make('div', 'pb-scenes-body');
  const grid = make('div', 'pb-scenes-grid');
  grid.id = 'pb-scenes-grid';
  grid.setAttribute('role', 'listbox');
  grid.setAttribute('aria-label', 'Scenes');
  const empty = make('p', 'pb-scenes-empty',
    'No scenes yet. Scenes are the script\'s chapters — add one with "+" on the scene strip or in the Chapters tab.');
  empty.id = 'pb-scenes-empty';
  const details = make('aside', 'pb-scene-details');
  details.id = 'pb-scene-details';
  body.append(grid, details);
  panel.append(head, empty, body);

  let selected = 0;
  let feels = []; // ChapterScene[] from SuggestChapterScenes (same order as chapters)
  let feelsBusy = false;
  let feelsSig = null; // chapters the feels were fetched for
  const sigOf = scenes => scenes.map(c => `${c.start}-${c.end}`).join(',');
  async function refreshFeels() {
    feelsSig = sigOf(api.chapters());
    try {
      const r = await SuggestChapterScenes();
      feels = Array.isArray(r) ? r : [];
    } catch (_) {
      feels = [];
    }
    render();
  }
  const feelOf = i => (feels[i] && feels[i].hasFeel ? feels[i].feel : null); // null = not set
  async function saveFeels(change) {
    if (feelsBusy) return;
    feelsBusy = true;
    try {
      const next = feels.map((f, i) => ({ f, feel: change(i, feelOf(i), f) }))
        .filter(x => x.feel != null)
        .map(x => ({ start_ms: x.f.startMs, end_ms: x.f.endMs, feel: x.feel }));
      await SaveChapterScenes(next);
      await refreshFeels();
      api.refreshVisuals?.();
    } catch (err) {
      const st = details.querySelector('#pb-scene-feel-status');
      if (st) st.textContent = 'Could not save: ' + err;
    } finally {
      feelsBusy = false;
    }
  }
  applyAll.addEventListener('click', () => saveFeels((i, cur, f) => (f.suggested ? f.suggested : cur)));
  let thumbs = [];
  let thumbsFor = '';
  let info = null;
  let infoFor = '';

  const visible = () => panel.classList.contains('is-active');

  async function refreshVideo() {
    const path = api.videoPath();
    if (path !== thumbsFor) {
      thumbsFor = path;
      thumbs = [];
      info = null;
      infoFor = '';
      if (path) {
        try { thumbs = await api.getSceneThumbs(); } catch (_) { thumbs = []; }
        if (api.videoPath() !== path) return;
      }
    }
    if (path && infoFor !== path) {
      infoFor = path;
      try { info = await ProbeCreateVideo(path); } catch (_) { info = null; }
      if (api.videoPath() !== path) return;
    }
    render();
  }

  function select(i, focus) {
    selected = i;
    render();
    if (focus) grid.querySelector(`[data-scene="${i}"]`)?.focus();
  }

  function renderDetails(scenes) {
    details.replaceChildren();
    const s = scenes[selected];
    if (!s) { details.hidden = true; return; }
    details.hidden = false;
    details.append(make('h4', 'pb-scene-d-title', s.name));
    const dl = make('dl', 'pb-scene-d-list');
    const row = (k, v, id) => {
      const dd = make('dd', '', v);
      if (id) dd.id = id;
      dl.append(make('dt', '', k), dd);
    };
    row('Range', `${fmtMs(s.start)} – ${fmtMs(s.end)}`, 'pb-scene-d-range');
    row('Duration', fmtMs(s.end - s.start));
    row('Scene', `${selected + 1} / ${scenes.length}`, 'pb-scene-d-pos');
    details.append(dl);

    const nav = make('div', 'pb-scene-d-nav');
    const prev = make('button', 'pb-scene-prev', '‹ Prev');
    prev.type = 'button'; prev.id = 'pb-scene-prev';
    prev.disabled = selected <= 0;
    prev.addEventListener('click', () => select(selected - 1));
    const go = make('button', 'pb-scene-go', 'Go to scene');
    go.type = 'button'; go.id = 'pb-scene-go';
    go.addEventListener('click', async () => { await api.seekMs(s.start); markCurrent(); });
    const next = make('button', 'pb-scene-next', 'Next ›');
    next.type = 'button'; next.id = 'pb-scene-next';
    next.disabled = selected >= scenes.length - 1;
    next.addEventListener('click', () => select(selected + 1));
    nav.append(prev, go, next);
    details.append(nav);

    // Sam Neo 2 feel for this scene: saved value, suggestion, Use.
    const fs = feels[selected];
    if (fs) {
      details.append(make('h5', 'pb-scene-d-sub', 'Sam Neo 2 feel'));
      const frow = make('div', 'pb-scene-feel');
      const sel = make('select');
      sel.id = 'pb-scene-feel-sel';
      sel.setAttribute('aria-label', 'Scene feel');
      for (const [v, label] of FEELS) sel.append(new Option(label, v));
      sel.value = fs.hasFeel ? fs.feel : '';
      sel.addEventListener('change', () => saveFeels((i, cur) => (i === selected ? sel.value : cur)));
      frow.append(sel);
      if (fs.suggested) {
        const sug = make('span', 'pb-scene-feel-sug',
          `Suggested: ${FEEL_NAME[fs.suggested]} · ${fs.source} ${Math.round((fs.confidence || 0) * 100)}%`);
        sug.id = 'pb-scene-feel-sug';
        const use = make('button', 'pb-scene-feel-use', 'Use');
        use.type = 'button';
        use.id = 'pb-scene-feel-use';
        use.disabled = fs.hasFeel && fs.feel === fs.suggested;
        use.addEventListener('click', () => saveFeels((i, cur) => (i === selected ? fs.suggested : cur)));
        frow.append(sug, use);
      } else {
        frow.append(make('span', 'pb-scene-feel-sug hint', 'No suggestion — no scene detection for this part.'));
      }
      details.append(frow);
      const st = make('p', 'hint');
      st.id = 'pb-scene-feel-status';
      details.append(st);
    }

    const tech = make('dl', 'pb-scene-d-tech');
    tech.id = 'pb-scene-d-tech';
    const trow = (k, v) => tech.append(make('dt', '', k), make('dd', '', v));
    const path = api.videoPath();
    if (!path) {
      trow('Video', 'none linked');
    } else {
      trow('Video', baseName(path));
      if (info && info.width > 0 && info.height > 0) trow('Resolution', `${info.width} × ${info.height}`);
      if (info && info.fps > 0) trow('Frame rate', `${Math.round(info.fps * 100) / 100} fps${info.vfr ? ' (variable)' : ''}`);
      if (info && info.codec) trow('Codec', info.codec);
    }
    details.append(make('h5', 'pb-scene-d-sub', 'Video'), tech);
  }

  function render() {
    if (!visible()) return;
    const scenes = api.chapters();
    count.textContent = scenes.length === 1 ? '1 scene' : `${scenes.length} scenes`;
    empty.hidden = scenes.length > 0;
    body.hidden = scenes.length === 0;
    if (selected >= scenes.length) selected = Math.max(0, scenes.length - 1);
    const now = api.positionMs();
    grid.replaceChildren();
    scenes.forEach((s, i) => {
      const card = make('button', 'pb-scene-card');
      card.type = 'button';
      card.dataset.scene = String(i);
      card.setAttribute('role', 'option');
      card.setAttribute('aria-selected', i === selected ? 'true' : 'false');
      card.classList.add('c' + (i % 5));
      if (i === selected) card.classList.add('is-selected');
      if (now >= s.start && now < s.end) card.classList.add('is-current');
      const url = thumbFor(thumbs, s.start, s.end);
      const fig = make('span', 'pb-scene-card-thumb');
      if (url) {
        const img = make('img');
        img.alt = '';
        img.src = url;
        fig.append(img);
      } else {
        fig.classList.add('is-empty');
      }
      const f = feelOf(i);
      const name = make('span', 'pb-scene-card-name', s.name);
      if (f != null) name.append(make('b', 'pb-scene-card-feel', FEEL_NAME[f] || f));
      card.append(fig,
        name,
        make('span', 'pb-scene-card-range', `${fmtMs(s.start)} – ${fmtMs(s.end)} · ${fmtMs(s.end - s.start)}`));
      card.addEventListener('click', () => select(i));
      card.addEventListener('dblclick', async () => { await api.seekMs(s.start); markCurrent(); });
      grid.append(card);
    });
    // Feels follow the chapters by position; refetch once when they change.
    if (feelsSig !== sigOf(scenes) && !feelsBusy) refreshFeels();
    applyAll.hidden = !feels.some((f, i) => f.suggested && feelOf(i) !== f.suggested);
    renderDetails(scenes);
  }

  // The playing scene gets a dot; cheap class toggles while the tab shows.
  function markCurrent() {
    if (!visible()) return;
    const now = api.positionMs();
    const scenes = api.chapters();
    for (const card of grid.children) {
      const s = scenes[Number(card.dataset.scene)];
      const on = !!s && now >= s.start && now < s.end;
      if (card.classList.contains('is-current') !== on) card.classList.toggle('is-current', on);
    }
  }
  setInterval(markCurrent, 500);

  grid.addEventListener('keydown', e => {
    const n = api.chapters().length;
    if (!n) return;
    if (e.key === 'ArrowRight' || e.key === 'ArrowDown') { e.preventDefault(); select(Math.min(n - 1, selected + 1), true); }
    else if (e.key === 'ArrowLeft' || e.key === 'ArrowUp') { e.preventDefault(); select(Math.max(0, selected - 1), true); }
    else if (e.key === 'Enter') {
      e.preventDefault();
      const s = api.chapters()[selected];
      if (s) api.seekMs(s.start).then(markCurrent);
    }
  });

  root.addEventListener('pb:chapters', () => render());
  // A linked video changes data-pb-video-path; refresh thumbs and info then.
  new MutationObserver(() => { if (visible()) refreshVideo(); })
    .observe(root, { attributes: true, attributeFilter: ['data-pb-video-path'] });

  return {
    show() { render(); refreshVideo(); refreshFeels(); },
  };
}
