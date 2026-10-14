// Mode switch (Owner 14 Oct: "oben ein Wählmenü … nicht einfach Dropdown"):
// the top bar shows one glowing Mode pill with the current mode; it opens
// the Mode deck — large glass cards for Player, Editor, Generator, Training
// and Scene Map over the silk ribbons, keys 1–5, arrows, Esc. Tools stays
// its own button next to the pill. The Play · Edit toggle is gone: Editor
// is a card.
//
// The existing top tab buttons become the deck's cards (moved, same IDs,
// same switchTab wiring and ARIA); Editor is a new card that asks main.js
// for the edit view (ws:set), Player asks for the play view.
import { ICONS } from './flow_icons.js';

const MODES = [
  { mode: 'player', tab: 'playback', name: 'Player', icon: 'player', desc: 'Play scripts with video and your device', hue: 'cyan' },
  { mode: 'editor', name: 'Editor', icon: 'edit', desc: 'Edit points, axes and curves', hue: 'violet' },
  { mode: 'generator', tab: 'generator', name: 'Generator', icon: 'create', desc: 'Create a script from a video', hue: 'orange' },
  { mode: 'training', tab: 'training', name: 'Training', icon: 'training', desc: 'Guided sessions with live curves', hue: 'flow' },
  { mode: 'scenemap', tab: 'scenemap', name: 'Scene Map', icon: 'scenemap', desc: 'Scenes, chapters and rhythm', hue: 'teal' },
];

export function initModeSwitch() {
  const rail = document.getElementById('rail');
  if (!rail || document.getElementById('mode-pill')) return null;

  // ---- Pill -------------------------------------------------------------
  const pill = document.createElement('button');
  pill.type = 'button';
  pill.id = 'mode-pill';
  pill.className = 'mode-pill';
  pill.setAttribute('aria-haspopup', 'dialog');
  pill.setAttribute('aria-expanded', 'false');
  pill.setAttribute('aria-controls', 'mode-deck');
  pill.title = 'Choose mode (M)';
  pill.innerHTML = `
    <span class="mode-pill-orb" aria-hidden="true"></span>
    <span class="mode-pill-text"><small>Mode</small><b id="mode-pill-name">Player</b></span>
    <svg class="mode-pill-chev" viewBox="0 0 24 24" aria-hidden="true"><path d="m7 10 5 5 5-5"/></svg>`;
  rail.prepend(pill);
  rail.classList.add('has-mode-switch');
  document.body.classList.add('has-mode-switch');

  // ---- Deck ---------------------------------------------------------------
  const deck = document.createElement('div');
  deck.id = 'mode-deck';
  deck.className = 'mode-deck';
  deck.setAttribute('role', 'dialog');
  deck.setAttribute('aria-label', 'Choose mode');
  deck.hidden = true;
  deck.innerHTML = `<div class="mode-deck-backdrop"></div>
    <div class="mode-deck-panel">
      <p class="mode-deck-title">Choose a mode <span>1–5 · Esc</span></p>
      <div class="mode-deck-cards"></div>
    </div>`;
  document.body.append(deck);
  const cards = deck.querySelector('.mode-deck-cards');

  MODES.forEach((m, i) => {
    let card = m.tab ? rail.querySelector(`.tab-btn[data-tab="${m.tab}"]`) : null;
    if (!card) {
      card = document.createElement('button');
      card.type = 'button';
      card.innerHTML = `<span>${m.name}</span>`;
    }
    card.classList.add('mode-card');
    card.dataset.mode = m.mode;
    card.dataset.hue = m.hue;
    card.style.setProperty('--i', String(i));
    // One large icon in a glowing ring, the name, a line, the key.
    card.querySelector(':scope > svg')?.remove();
    card.querySelector(':scope > .dot')?.remove();
    const ring = document.createElement('span');
    ring.className = 'mode-card-ring';
    ring.innerHTML = ICONS[m.icon] || '';
    card.prepend(ring);
    const desc = document.createElement('small');
    desc.className = 'mode-card-desc';
    desc.textContent = m.desc;
    const key = document.createElement('kbd');
    key.textContent = String(i + 1);
    card.append(desc, key);
    card.title = `${m.name} — ${m.desc} (${i + 1})`;
    card.addEventListener('click', () => {
      if (m.mode === 'player') window.dispatchEvent(new CustomEvent('ws:set', { detail: { view: 'play' } }));
      if (m.mode === 'editor') window.dispatchEvent(new CustomEvent('ws:set', { detail: { view: 'edit' } }));
      close(false);
    });
    cards.append(card);
  });

  // ---- Open / close -------------------------------------------------------
  let lastFocus = null;
  function open() {
    if (!deck.hidden) return;
    lastFocus = document.activeElement;
    const r = pill.getBoundingClientRect();
    deck.style.setProperty('--deck-top', `${Math.round(r.bottom + 10)}px`);
    deck.hidden = false;
    requestAnimationFrame(() => deck.classList.add('is-open'));
    pill.setAttribute('aria-expanded', 'true');
    (cards.querySelector('.mode-card.is-current') || cards.querySelector('.mode-card'))?.focus();
  }
  function close(restoreFocus = true) {
    if (deck.hidden) return;
    deck.classList.remove('is-open');
    deck.hidden = true;
    pill.setAttribute('aria-expanded', 'false');
    if (restoreFocus) (lastFocus && lastFocus.focus ? lastFocus : pill).focus();
  }
  pill.addEventListener('click', () => (deck.hidden ? open() : close()));
  deck.querySelector('.mode-deck-backdrop').addEventListener('click', () => close());
  deck.addEventListener('keydown', e => {
    const list = [...cards.querySelectorAll('.mode-card')].filter(c => !c.hidden);
    const i = list.indexOf(document.activeElement);
    if (e.key === 'Escape') { e.preventDefault(); close(); return; }
    if (/^[1-9]$/.test(e.key) && list[+e.key - 1]) { e.preventDefault(); list[+e.key - 1].click(); return; }
    let j = -1;
    if (e.key === 'ArrowRight' || e.key === 'ArrowDown') j = (i + 1) % list.length;
    else if (e.key === 'ArrowLeft' || e.key === 'ArrowUp') j = (i - 1 + list.length) % list.length;
    else if (e.key === 'Home') j = 0;
    else if (e.key === 'End') j = list.length - 1;
    else if (e.key === 'Tab') { // keep focus inside the deck
      e.preventDefault();
      j = e.shiftKey ? (i - 1 + list.length) % list.length : (i + 1) % list.length;
    }
    if (j < 0) return;
    e.preventDefault();
    e.stopPropagation();
    list[j].focus();
  });
  // M opens the deck (not while typing).
  document.addEventListener('keydown', e => {
    if (e.key !== 'm' && e.key !== 'M') return;
    if (e.ctrlKey || e.metaKey || e.altKey) return;
    const t = e.target;
    if (t && (t.isContentEditable || /^(INPUT|TEXTAREA|SELECT)$/.test(t.tagName))) return;
    if (!deck.hidden) return;
    e.preventDefault();
    open();
  });

  // ---- Current mode -------------------------------------------------------
  const nameEl = pill.querySelector('#mode-pill-name');
  let lastMode = 'player';
  function sync() {
    const mode = document.body.dataset.mode || 'player';
    const inTools = mode === 'tools';
    if (!inTools) lastMode = mode;
    const m = MODES.find(x => x.mode === lastMode) || MODES[0];
    nameEl.textContent = m.name;
    pill.dataset.hue = m.hue;
    pill.querySelector('.mode-pill-orb').innerHTML = ICONS[m.icon] || '';
    pill.classList.toggle('is-dim', inTools);
    cards.querySelectorAll('.mode-card').forEach(c => {
      const on = !inTools && c.dataset.mode === mode;
      c.classList.toggle('is-current', on);
      c.setAttribute('aria-current', on ? 'true' : 'false');
    });
  }
  new MutationObserver(sync).observe(document.body, { attributes: true, attributeFilter: ['data-mode'] });
  sync();
  return { open, close };
}
