// In-app user handbook (Emotion GUI). Source of truth for agents also lives in
// docs/USER_HANDBOOK.md — keep the FAQ here in sync when product behavior changes.

const SECTIONS = [
  {
    title: 'Everyday Create (recommended)',
    body: `1. Open Create → choose a video.
2. Find tip area (or paint the box). Optional: Smarter tip find if an AI ROI model is set in Settings.
3. Leave Contact vibration on. Optionally mark nipples/mouth for feel later.
4. Click Create / Generate.
5. Review & improve: trim · Fill gaps · Heal tracking gaps · audio check.
6. Open Play — soft curve + dots. Edit if needed. Connect device when ready.

Auto after Generate: fill gaps + heal known tracker-loss windows (linear bridge only).
Default stroke writer is classical CSRT — AI never silently invents the curve.`,
  },
  {
    title: 'Is everything in the GUI?',
    body: `Almost for everyday use. Create, Play, Device, Training, Settings, Scene map (Advanced), learning export, classical AI-training export, Load project / Scale range, Optimize for Neo 2 — yes.

Still CLI / advanced / not Everyday GUI:
• Whole-frame 4-zone stroke mode (experiments; weaker on measured clips)
• Flow tracker as Generate default (CLI)
• ONNX vision AI draft writer (S2+ model inference — not shipped; imitation draft is available after Export classical run)`,
  },
  {
    title: 'Fill gaps vs Heal tracking gaps',
    body: `Fill gaps — linear points across long time holes between actions.
Heal tracking gaps — uses tracking_gaps from Generate: strips junk inside loss windows, bridges only those windows, clears metadata so Contact vib is not muted forever.

Neither re-runs CSRT. Neither invents motion from audio (audio only spaces steps).`,
  },
  {
    title: 'Can AI write the Funscript?',
    body: `Path is open, opt-in, local only:
• S0 plan + stub — shipped
• S1 Export classical run (Create → Advanced) — after Create, status reminds you; builds a local imitation library
• S2-imitation — with ≥1 export, AI draft script stretches the best duration + tip-aspect match → Quality Doctor → Keep/Discard (experimental; not Everyday default)
• ONNX vision draft writer — not shipped yet

Cloud “ChatGPT writes positions” is out of product. Everyday Create stays CSRT.`,
  },
  {
    title: 'Play — quick map',
    body: `Play / Stop — device + curve (default). Video ▶ can also start device (toggle).
Edit curve — drag dots; click empty = add; double-click = delete (≥2 remain).
Optimize for Neo 2 — import path: fill/heal gaps → contact on → bake vibe/suction → .samn
Load / Save project — .snp.json
Scale range ×0.8 — soften marked range intensity
Playlist — queue multiple scripts
Bookmarks — named times: add at playhead, seek, remove
Chapters — named ranges: mark heatmap, add from selection, seek, remove
Heatmap / markers — seek, loop, Extended-O, O-markers

Keyboard: Space · ←/→ · ,/. · 1–9 · +/− offset · L loop · E Extended-O · O marker.`,
  },
  {
    title: 'Device & Training',
    body: `Device — connect Sam Neo 2 / Handy / etc.; connection test optional in Settings.
Intiface Central — start the server, then Via Intiface Central (empty address = this machine). After a single socket drop, one automatic reconnect is tried; if it stays down, tap Connect again.
Training — practice patterns without a video (technique, channel, cycles).
AI Train — label tip ROIs / train ONNX, and Train Go profile model (helpers only — not the stroke writer).`,
  },
  {
    title: 'Settings that matter',
    body: `Open user handbook — this FAQ.
AI ROI model path — enables Smarter tip find. Use Open AI training to jump to the AI Train tab, then Check availability.
Collect learning data — allows Scene map Export for learning.
Colibri / AI server URL — optional local prose for Suggest profile (see docs/COLIBRI_SETUP.md). Use Test AI server after coli serve.

AI draft model path is not in Settings yet (reserved for a future ONNX writer). Until then, use Export classical run → AI draft script from the imitation library.`,
  },
  {
    title: 'Also in the GUI',
    body: `Play Playlist — queue multiple scripts.
Create Review — Align fill to audio tempo (optional when filling gaps).
Create Advanced — Sliding dynamics, Auto-Retry, Suggest O-markers, Export classical run (needs a Create result), Scene map, Rhythm-robust signal, Use contact points (teachers JSON).`,
  },
  {
    title: 'Use contact points (Advanced)',
    body: `Opt-in after VLM1 engine: when Rhythm-robust signal is on, Advanced → Use contact points loads a teachers JSON from generator/contact_points.py (NudeNet / VLM probe consensus). The rhythm grid may search near those points only when the tip box is far away (>3 cells). Empty/off = bit-identical Everyday Create. Build the JSON via CLI; the GUI only picks a path — it does not run teachers.`,
  },
  {
    title: 'Troubleshooting',
    body: `Flat / stuck curve — re-find tip · Invert motion · Heal tracking gaps · check tracking_gaps muted Contact.
Feels inverted — Advanced → Invert motion direction.
Camera pans drift — Camera motion compensation · Fix contact area (static) off.
Long-clip drift — Advanced → Rhythm-robust signal (opt-in). Optional: Use contact points (teachers JSON) when Rhythm-robust is on.
Audio “wrong tempo” — warn only; fix ROI/axis; does not rewrite the curve.
AI draft greyed out — Export classical run once after Create (≥1 sample), then draft enables. ONNX model path is not required for imitation.`,
  },
];

let overlay = null;

function closeHandbook() {
  if (overlay) {
    overlay.remove();
    overlay = null;
  }
}

/** Open the in-app handbook modal. */
export function openHandbook() {
  closeHandbook();
  overlay = document.createElement('div');
  overlay.className = 'handbook-overlay';
  overlay.setAttribute('role', 'dialog');
  overlay.setAttribute('aria-modal', 'true');
  overlay.setAttribute('aria-label', 'User handbook');

  const panel = document.createElement('div');
  panel.className = 'handbook-panel';

  const head = document.createElement('header');
  head.className = 'handbook-head';
  head.innerHTML = `
    <div>
      <p class="handbook-kicker">Emotion Script</p>
      <h2>User handbook</h2>
      <p class="handbook-sub">How to use Create, Play, gaps, and the opt-in AI path. Default curve writer stays CSRT.</p>
    </div>
    <button type="button" class="handbook-close" aria-label="Close handbook">Close</button>
  `;

  const nav = document.createElement('nav');
  nav.className = 'handbook-nav';
  const body = document.createElement('div');
  body.className = 'handbook-body';

  SECTIONS.forEach((sec, i) => {
    const id = 'hb-sec-' + i;
    const a = document.createElement('button');
    a.type = 'button';
    a.className = 'handbook-nav-btn' + (i === 0 ? ' is-active' : '');
    a.textContent = sec.title;
    a.addEventListener('click', () => {
      nav.querySelectorAll('.handbook-nav-btn').forEach(b => b.classList.remove('is-active'));
      a.classList.add('is-active');
      const el = body.querySelector('#' + id);
      if (el) el.scrollIntoView({ behavior: 'smooth', block: 'start' });
    });
    nav.appendChild(a);

    const article = document.createElement('article');
    article.className = 'handbook-section';
    article.id = id;
    const h = document.createElement('h3');
    h.textContent = sec.title;
    const p = document.createElement('pre');
    p.className = 'handbook-text';
    p.textContent = sec.body.trim();
    article.appendChild(h);
    article.appendChild(p);
    body.appendChild(article);
  });

  const foot = document.createElement('p');
  foot.className = 'handbook-foot hint';
  foot.textContent = 'Full markdown for agents: docs/USER_HANDBOOK.md · Look: dark ink + lilac accent';

  panel.appendChild(head);
  panel.appendChild(nav);
  panel.appendChild(body);
  panel.appendChild(foot);
  overlay.appendChild(panel);
  document.body.appendChild(overlay);

  head.querySelector('.handbook-close').addEventListener('click', closeHandbook);
  overlay.addEventListener('click', (e) => {
    if (e.target === overlay) closeHandbook();
  });
}

document.addEventListener('keydown', (e) => {
  if (e.key === 'Escape' && overlay) {
    e.stopPropagation();
    closeHandbook();
  }
});
