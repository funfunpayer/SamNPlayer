// Generator with editor (Owner 14 Oct: "der Generator soll ein richtiger
// Editor sein"): in Generator mode the Player's timeline (lanes, points,
// zoom, chapters, …) sits under the Generator video, and the edit tools
// (edit curve, undo / redo, Optimize for, Sam Neo 2 feel) join the settings
// column. A new script stays in the Generator and is edited right there.
//
// The parts are borrowed, not copied: the same nodes (IDs, listeners,
// canvases) move into the Generator while it is shown and go back to the
// Player when another mode opens. The Generator video and the timeline's
// playhead follow each other.

export function attachGeneratorEditor(genRoot, pbRoot, api) {
  if (!genRoot || !pbRoot) return null;
  const stage = genRoot.querySelector('.gl-stage');
  const side = genRoot.querySelector('.gl-side');
  if (!stage || !side) return null;

  const editor = document.createElement('section');
  editor.className = 'gl-editor card';
  editor.setAttribute('aria-label', 'Editor');
  editor.innerHTML = `<div class="gl-editor-head"><p class="gl-card-title">Editor</p>
    <span class="hint gl-editor-name" id="gl-editor-name"></span></div>
    <p class="hint gl-editor-empty" id="gl-editor-empty">Create a script (or open one) — it opens here for editing: drag, add or delete points on every lane.</p>`;
  stage.after(editor);

  const tools = document.createElement('section');
  tools.className = 'gl-edit-tools card';
  tools.setAttribute('aria-label', 'Edit tools');
  tools.append(Object.assign(document.createElement('p'), { className: 'gl-card-title', textContent: 'Edit' }));
  const run = side.querySelector('.gl-run');
  (run || side.lastElementChild)?.after(tools);
  tools.hidden = true;

  // Borrowed parts and where they go back to.
  const parts = [
    { sel: '[data-pb-region="bottom"]', into: editor },
    { sel: '#pb-insp-view', into: tools },
    { sel: '#pb-feel-card', into: tools },
  ];
  let borrowed = false;
  let editOn = false;
  // (Looked up in the whole document: the box travels with the edit tools.)
  const setEdit = on => {
    const box = document.querySelector('#pb-curve-edit');
    if (!box || box.disabled || box.checked === on) return false;
    box.checked = on;
    box.dispatchEvent(new Event('change'));
    return true;
  };

  function borrow() {
    if (borrowed) return;
    for (const p of parts) {
      const node = pbRoot.querySelector(p.sel) || document.querySelector(p.sel);
      if (!node || p.into.contains(node)) continue;
      p.node = node;
      p.mark = document.createComment('gen-editor');
      node.before(p.mark);
      p.into.append(node);
    }
    borrowed = true;
    genRoot.dataset.genEditor = '1';
    sync();
  }
  function giveBack() {
    if (!borrowed) return;
    if (editOn) { setEdit(false); editOn = false; }
    for (const p of parts) {
      if (p.node && p.mark && p.mark.parentNode) p.mark.replaceWith(p.node);
      p.node = null;
      p.mark = null;
    }
    borrowed = false;
    genRoot.dataset.genEditor = '';
  }
  function sync() {
    const loaded = pbRoot.dataset.pbState === 'loaded';
    editor.dataset.empty = loaded ? '' : '1';
    tools.hidden = !borrowed || !loaded;
    const path = pbRoot.dataset.pbScript || '';
    editor.querySelector('#gl-editor-name').textContent = loaded && path ? path.split(/[\\/]/).pop() : '';
    // In the Generator the loaded script is being edited: points on.
    if (borrowed && loaded && setEdit(true)) editOn = true;
  }

  const inGenerator = () => document.body.dataset.mode === 'generator';
  const follow = () => (inGenerator() ? borrow() : giveBack());
  new MutationObserver(follow).observe(document.body, { attributes: true, attributeFilter: ['data-mode'] });
  new MutationObserver(sync).observe(pbRoot, { attributes: true, attributeFilter: ['data-pb-state', 'data-pb-script'] });
  // The curve-edit row appears once the curve is drawn — turn editing on then.
  const editRow = document.querySelector('#pb-curve-edit-row');
  if (editRow) new MutationObserver(() => { if (borrowed) sync(); }).observe(editRow, { attributes: true, attributeFilter: ['style'] });

  // ---- Generator video ↔ timeline playhead ----------------------------------
  const genVideo = genRoot.querySelector('#gen-video');
  const pbVideo = pbRoot.querySelector('#pb-video');
  let syncing = false;
  if (genVideo && api && typeof api.seekMs === 'function') {
    const toTimeline = () => {
      if (!borrowed || syncing || pbRoot.dataset.pbState !== 'loaded') return;
      syncing = true;
      try { api.seekMs(Math.round(genVideo.currentTime * 1000)); } finally { syncing = false; }
    };
    genVideo.addEventListener('timeupdate', toTimeline);
    genVideo.addEventListener('seeked', toTimeline);
  }
  if (genVideo && pbVideo) {
    pbVideo.addEventListener('seeked', () => {
      // A Generator jump still in flight wins over an older timeline seek.
      if (!borrowed || syncing || !genVideo.paused || genVideo.seeking) return;
      if (Math.abs(genVideo.currentTime - pbVideo.currentTime) > 0.25) {
        syncing = true;
        genVideo.currentTime = pbVideo.currentTime;
        setTimeout(() => { syncing = false; }, 0);
      }
    });
  }

  follow();
  return { borrow, giveBack, sync };
}
