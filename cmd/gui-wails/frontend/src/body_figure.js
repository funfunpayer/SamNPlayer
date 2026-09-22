/** Human body map for AI Train class picking (stylized line figure).
 *
 * Same idea as the Device tab silhouette Claude shipped: a clear visual
 * instead of text-only chips. Click a region → select that body-part class.
 * Anatomical outline only — no photographic detail.
 * Training-tab motion uses separate pixel pair figures (`pixel_figure.js`).
 */

import { CANONICAL, labelFor, normalizeClass } from './bodyparts.js';

const REGION_HINTS = {
  face: 'Head / face box',
  mouth: 'Mouth / lips',
  breasts: 'Chest / breasts',
  nipples: 'Nipples (Tf/Tj contact)',
  hand_1: 'Hand 1',
  hand_2: 'Hand 2',
  penis: 'Penis / shaft',
  glans: 'Glans / tip (common tracked tip)',
  vagina: 'Vagina / vulva',
};

/** Inline SVG — viewBox 0 0 120 200, front-facing stylized adult. */
export function bodyFigureMarkup() {
  return `
<svg class="body-figure-svg" viewBox="0 0 120 200" role="img"
     aria-label="Body parts for AI training">
  <defs>
    <style>
      .bf-outline { fill: none; stroke: #5a6478; stroke-width: 1.6; stroke-linecap: round; stroke-linejoin: round; }
      .bf-zone { fill: rgba(61,204,192,0.08); stroke: #3dccc0; stroke-width: 1.2; cursor: pointer; transition: fill .12s ease; }
      .bf-zone:hover, .bf-zone.is-hot { fill: rgba(61,204,192,0.28); }
      .bf-zone.is-active { fill: rgba(242,176,61,0.35); stroke: #f2b03d; stroke-width: 1.8; }
      .bf-label { fill: #9aa3b5; font-size: 7px; font-family: inherit; pointer-events: none; }
    </style>
  </defs>
  <!-- soft body silhouette (non-interactive) -->
  <ellipse class="bf-outline" cx="60" cy="28" rx="16" ry="18"/>
  <path class="bf-outline" d="M44 44 Q60 52 76 44 L82 90 Q60 102 38 90 Z"/>
  <path class="bf-outline" d="M42 92 L38 150 M78 92 L82 150"/>
  <path class="bf-outline" d="M38 150 L34 196 M82 150 L86 196"/>
  <path class="bf-outline" d="M44 58 L22 88 M76 58 L98 88"/>

  <!-- clickable zones -->
  <ellipse class="bf-zone" data-class="face" cx="60" cy="26" rx="14" ry="15"/>
  <ellipse class="bf-zone" data-class="mouth" cx="60" cy="36" rx="7" ry="3.5"/>
  <ellipse class="bf-zone" data-class="breasts" cx="60" cy="68" rx="22" ry="14"/>
  <circle class="bf-zone" data-class="nipples" cx="50" cy="68" r="3.2"/>
  <circle class="bf-zone" data-class="nipples" cx="70" cy="68" r="3.2" data-alias="nipples-r"/>
  <ellipse class="bf-zone" data-class="hand_1" cx="20" cy="92" rx="9" ry="7"/>
  <ellipse class="bf-zone" data-class="hand_2" cx="100" cy="92" rx="9" ry="7"/>
  <ellipse class="bf-zone" data-class="penis" cx="60" cy="118" rx="7" ry="16"/>
  <ellipse class="bf-zone" data-class="glans" cx="60" cy="132" rx="5.5" ry="5"/>
  <ellipse class="bf-zone" data-class="vagina" cx="60" cy="108" rx="10" ry="6"/>
</svg>`;
}

/**
 * Mount interactive body map into `host`.
 * @param {HTMLElement} host
 * @param {{ onSelect?: (classId: string) => void, getActive?: () => string }} opts
 */
export function mountBodyFigure(host, opts = {}) {
  if (!host) return { setActive() {}, destroy() {} };
  host.classList.add('body-figure');
  host.innerHTML = `
    <div class="body-figure-card">
      <div class="body-figure-head">
        <strong>Body map</strong>
        <span class="hint">Click a region — same classes as chips (Claude-style silhouette, human).</span>
      </div>
      <div class="body-figure-row">
        ${bodyFigureMarkup()}
        <ul class="body-figure-legend"></ul>
      </div>
    </div>`;

  const svg = host.querySelector('.body-figure-svg');
  const legend = host.querySelector('.body-figure-legend');
  let active = normalizeClass(opts.getActive?.() || '') || '';

  function paint() {
    svg.querySelectorAll('.bf-zone').forEach(el => {
      const id = el.getAttribute('data-class');
      el.classList.toggle('is-active', id === active);
      el.setAttribute('aria-pressed', id === active ? 'true' : 'false');
    });
    legend.querySelectorAll('[data-class]').forEach(li => {
      li.classList.toggle('is-active', li.getAttribute('data-class') === active);
    });
  }

  function select(id) {
    const n = normalizeClass(id);
    if (!n) return;
    active = n;
    paint();
    opts.onSelect?.(n);
  }

  legend.innerHTML = CANONICAL.map(c =>
    `<li data-class="${c.id}" title="${REGION_HINTS[c.id] || c.label}">
       <button type="button" class="body-figure-leg-btn">${c.label}</button>
     </li>`).join('');

  svg.querySelectorAll('.bf-zone').forEach(el => {
    const id = el.getAttribute('data-class');
    el.setAttribute('role', 'button');
    el.setAttribute('tabindex', '0');
    el.setAttribute('aria-label', labelFor(id) || id);
    el.title = REGION_HINTS[id] || labelFor(id) || id;
    el.addEventListener('click', () => select(id));
    el.addEventListener('keydown', e => {
      if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); select(id); }
    });
  });
  legend.querySelectorAll('[data-class]').forEach(li => {
    li.querySelector('button')?.addEventListener('click', () => select(li.getAttribute('data-class')));
  });

  paint();
  return {
    setActive(id) {
      active = normalizeClass(id) || '';
      paint();
    },
    destroy() { host.innerHTML = ''; },
  };
}
