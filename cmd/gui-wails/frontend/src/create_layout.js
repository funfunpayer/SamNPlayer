// V1 Create layout (target docs/visual-target/create.png): a three-column
// workspace instead of one long scrolling wizard.
//   left   — Project: source clips, recent videos, generated scripts, analysis
//   centre — project header, step bar, preview + source clips + curve preview
//   right  — the settings (Feel), step tools (Tip & contact / Create / Review)
// Existing elements are moved, not re-created: IDs, listeners and tests stay.
// The Scene Map view (#tab-generator[data-view="scenemap"]) keeps its own
// rules; the grid only applies to the plain Create view.
import { readRecentVideos, readScripts } from './player_layout.js';

function baseName(p) { return String(p || '').split(/[\\/]/).pop(); }

function make(tag, cls, text) {
  const e = document.createElement(tag);
  if (cls) e.className = cls;
  if (text != null) e.textContent = text;
  return e;
}

// Step bar: number + name + what the step does (target: "Import Clip /
// Add video or audio" …). Names stay ours — they match what each step does.
const STEP_TEXT = {
  1: ['Video', 'Add a clip'],
  2: ['Tip & contact', 'Find the motion'],
  3: ['Feel', 'Shape the curves'],
  4: ['Create', 'Build the script'],
  5: ['Review', 'Refine and play'],
};

export function applyCreateLayoutV1(root, { playback } = {}) {
  if (!root || root.dataset.genLayout === 'v1') return null;
  root.dataset.genLayout = 'v1';
  const q = sel => root.querySelector(sel);

  // ---- Step bar with sub-labels -----------------------------------------
  root.querySelectorAll('.gen-step-item').forEach(li => {
    const [name, sub] = STEP_TEXT[li.dataset.step] || [];
    if (!name) return;
    const num = li.querySelector('.gen-step-num');
    const text = make('span', 'gen-step-text');
    text.append(make('b', '', name), make('small', '', sub));
    li.replaceChildren(num, text);
  });

  // ---- Left: Project navigation (real lists only) -------------------------
  const nav = make('nav', 'gl-nav');
  nav.setAttribute('aria-label', 'Project');
  nav.innerHTML = `
    <p class="gl-nav-title">Project</p>
    <button type="button" class="gl-nav-item is-active" id="gl-nav-clips">Source clips</button>
    <p class="gl-nav-sub">Recent videos</p>
    <ul class="gl-nav-list" id="gl-nav-recent"></ul>
    <p class="gl-nav-sub">Generated scripts</p>
    <ul class="gl-nav-list" id="gl-nav-generated"></ul>
    <p class="gl-nav-title">Analysis</p>
    <button type="button" class="gl-nav-item" id="gl-nav-scenemap">Scene map</button>
    <div class="gl-nav-drop">Drag videos here<br><span>or use Choose video…</span></div>`;
  root.prepend(nav);

  function fillList(listEl, paths, emptyText, onPick) {
    listEl.replaceChildren();
    if (!paths.length) {
      listEl.append(make('li', 'gl-nav-empty', emptyText));
      return;
    }
    for (const p of paths.slice(0, 6)) {
      const li = make('li');
      const b = make('button', 'gl-nav-link', baseName(p));
      b.type = 'button';
      b.title = p;
      b.addEventListener('click', () => onPick(p));
      li.append(b);
      listEl.append(li);
    }
  }
  function renderNav() {
    fillList(q('#gl-nav-recent'), readRecentVideos(), 'No videos yet',
      p => window.dispatchEvent(new CustomEvent('gen:load-video', { detail: { path: p } })));
    fillList(q('#gl-nav-generated'), readScripts('generated'), 'Nothing created yet',
      p => playback?.loadScriptPath?.(p, { review: true }));
  }
  // Handbook moves into the Project column; the page title goes (the
  // project header names the work).
  const handbook = q('#gen-open-handbook');
  if (handbook) {
    handbook.classList.add('gl-nav-item');
    nav.insertBefore(handbook, q('.gl-nav-drop'));
  }
  q('.create-head')?.classList.add('gl-hidden-head');
  renderNav();
  window.addEventListener('pb:loaded', renderNav);
  window.addEventListener('gen:video', () => setTimeout(renderNav, 0)); // after the recent list updates
  window.addEventListener('ws:view', e => { if (e.detail?.view === 'create') renderNav(); });
  q('#gl-nav-clips').addEventListener('click', () => q('.gl-clips')?.scrollIntoView({ block: 'nearest' }));
  q('#gl-nav-scenemap').addEventListener('click', () => document.getElementById('ws-scenemap')?.click());

  // ---- Centre: preview + source clips side by side ---------------------
  const centre = make('div', 'gl-centre');
  const stage = make('div', 'gl-stage');
  const preview = make('section', 'gl-preview card');
  preview.setAttribute('aria-label', 'Preview');
  const empty = make('p', 'gl-preview-empty', 'Choose a video — its frame appears here.');
  const canvasWrap = q('#roi-canvas-wrap');
  preview.append(empty);
  if (canvasWrap) preview.append(canvasWrap);
  const clips = make('section', 'gl-clips card');
  clips.setAttribute('aria-label', 'Source clips');
  const clipsHead = make('div', 'gl-clips-head');
  const clipsTitle = make('p', 'gl-card-title', 'Source clips');
  clipsHead.append(clipsTitle);
  // One clip, no queue: a single card mirrors the project header (poster,
  // name, size) — the queue grid takes over as soon as there are several.
  const single = make('div', 'gen-clip gl-single-clip is-current');
  single.hidden = true;
  const singleThumb = make('div', 'gen-clip-thumb');
  const singleImg = make('img');
  singleImg.alt = '';
  singleThumb.append(singleImg);
  const singleName = make('span', 'gen-clip-name');
  const singleMeta = make('span', 'hint gen-clip-meta');
  single.append(singleThumb, singleName, singleMeta);
  const video = q('#gen-step-video');
  clips.append(clipsHead, single);
  if (video) clips.append(video);
  stage.append(preview, clips);

  // Curve preview tabs (visual target): Curve Preview / Motion / Audio / Scene Markers.
  // Full interactive curves open in Edit after Create (as decided). This is the
  // visual placeholder that matches the target layout.
  const curve = make('section', 'gl-curve card');
  curve.setAttribute('aria-label', 'Curve preview');
  const tabs = make('div', 'gl-curve-tabs');
  const tabNames = ['Curve Preview', 'Motion Analysis', 'Audio Analysis', 'Scene Markers'];
  tabNames.forEach((name, i) => {
    const b = make('button', 'gl-curve-tab' + (i === 0 ? ' is-active' : ''), name);
    b.type = 'button';
    b.dataset.tab = String(i);
    b.addEventListener('click', () => {
      tabs.querySelectorAll('.gl-curve-tab').forEach(t => t.classList.toggle('is-active', t === b));
      curve.querySelectorAll('.gl-curve-pane').forEach(p => p.hidden = p.dataset.tab !== b.dataset.tab);
    });
    tabs.append(b);
  });
  const panes = make('div', 'gl-curve-panes');
  tabNames.forEach((name, i) => {
    const pane = make('div', 'gl-curve-pane');
    pane.dataset.tab = String(i);
    pane.hidden = i !== 0;
    if (i === 0) {
      pane.append(make('p', 'hint', 'Generated curves appear here after Create. Open the result in Edit to refine and play.'));
      // Placeholder multi-axis sketch (matches target colours)
      const sketch = make('div', 'gl-curve-sketch');
      sketch.innerHTML = `
        <svg viewBox="0 0 400 120" xmlns="http://www.w3.org/2000/svg" aria-hidden="true">
          <path d="M0,60 Q50,30 100,55 T200,40 T300,70 T400,45" fill="none" stroke="#ff6a10" stroke-width="2"/>
          <path d="M0,70 Q60,50 120,65 T240,50 T360,75 T400,60" fill="none" stroke="#1a8cff" stroke-width="1.5"/>
          <path d="M0,50 Q40,80 100,45 T200,70 T300,40 T400,55" fill="none" stroke="#b896e8" stroke-width="1.5"/>
          <path d="M0,80 Q80,40 160,75 T280,50 T400,65" fill="none" stroke="#5ad4c4" stroke-width="1.5"/>
        </svg>`;
      pane.append(sketch);
    } else {
      pane.append(make('p', 'hint', `${name} will show here once a script is generated. Full analysis opens in Edit.`));
    }
    panes.append(pane);
  });
  curve.append(tabs, panes);

  // Step tools (Tip & contact, Create, Review) join the Feel settings on the
  // right, so the centre stays the project header + step bar + preview next
  // to Source clips. Closer to the visual target.
  const tools = make('div', 'gl-tools');
  tools.setAttribute('aria-label', 'Step tools');
  const region = q('#gen-step-region');
  const run = q('#gen-step-run');
  const result = q('#gen-step-result');
  if (region) tools.append(region);
  if (run) tools.append(run);
  if (result) tools.append(result);

  centre.append(stage, curve);
  const steps = q('#gen-steps');
  const prompt = q('#gen-step-prompt');
  (prompt || steps)?.after(centre);

  // ---- Right: settings column ---------------------------------------------
  const side = make('aside', 'gl-side');
  side.setAttribute('aria-label', 'Create settings');
  const sideEmpty = make('div', 'gl-side-empty card');
  sideEmpty.append(make('p', 'gl-side-title', 'Settings'),
    make('p', 'hint', 'Feel, style and channels appear here once a clip is chosen.'));
  side.append(sideEmpty);
  const motion = q('#gen-step-motion');
  if (motion) side.append(motion);
  // Step tools join Feel on the right.
  if (tools.children.length) side.append(tools);
  // Create / Cancel and the progress live at the bottom of the settings, like
  // the target's "Generate" button; they stay disabled until a clip is ready.
  const runCard = make('section', 'gl-run card');
  runCard.setAttribute('aria-label', 'Create');
  const genRow = q('#gen-generate')?.parentElement;
  const progress = q('#gen-progress-wrap');
  if (genRow) { genRow.classList.add('gl-run-row'); runCard.append(genRow); }
  if (progress) runCard.append(progress);
  side.append(runCard);
  root.append(side);

  // Placeholders, single card and counts follow the real state.
  const head = q('#gen-project-head');
  const queue = q('#gen-queue');
  const pathEl = q('#gen-video-path');
  const sync = () => {
    const hasVideo = !!pathEl && !/^\s*No video/i.test(pathEl.textContent || '');
    const hasFrame = hasVideo && !!head && !head.hidden;
    empty.hidden = hasVideo;
    if (canvasWrap) canvasWrap.hidden = !hasVideo;
    sideEmpty.hidden = !!motion && !motion.hidden;
    const many = !!queue && !queue.hidden;
    const n = many ? q("#gen-queue-list").children.length : (hasVideo ? 1 : 0);
    clipsTitle.textContent = n ? `Source clips (${n})` : 'Source clips';
    single.hidden = many || !hasFrame;
    if (!single.hidden) {
      const poster = q('#gen-project-poster');
      singleImg.hidden = !poster || poster.hidden || !poster.src;
      if (!singleImg.hidden) singleImg.src = poster.src;
      singleName.textContent = q('#gen-project-name')?.textContent || '';
      singleMeta.textContent = q('#gen-project-meta')?.textContent || '';
    }
  };
  const mo = new MutationObserver(sync);
  for (const n of [head, queue, motion, pathEl].filter(Boolean)) {
    mo.observe(n, { attributes: true, attributeFilter: ['hidden', 'src'], childList: true, subtree: true, characterData: true });
  }
  sync();

  return { renderNav };
}
