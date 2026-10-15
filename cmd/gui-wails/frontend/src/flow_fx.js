/**
 * SamNPlayer Flow — decorative idle-stage visual for Play and Create.
 *
 * Claude wiring contract:
 *   const fx = mountFlowStage(document.querySelector('#pb-stage-idle'));
 *   fx.setState('playing' | 'paused' | 'idle');
 *   fx.destroy();
 *
 * This module does not import Wails, load media, run timers, or set player state.
 * Import './flow_fx.css' from the host entrypoint or the mounting module.
 *
 * Artwork (Owner mockup / visual-target/player.png): lush cyan↔orange silk
 * ribbons behind the SamNPlayer brand mark — not thin crossing stroke lines.
 */
const STATES = new Set(['idle', 'playing', 'paused']);
const LOGO_URL = new URL('./assets/images/logo-universal.png', import.meta.url).href;
const SILK_URL = new URL('./assets/flow/stage-ribbons.svg', import.meta.url).href;
const SILK_TOP_URL = new URL('./assets/flow/bg-top.svg', import.meta.url).href;
const SILK_BOT_URL = new URL('./assets/flow/bg-bottom.svg', import.meta.url).href;
let nextInstanceId = 0;

function visibilityEnabled(root, inViewport, reducedMotion) {
  return !reducedMotion && inViewport && !document.hidden && root.isConnected;
}

/**
 * Mount decorative graphics into a host element.
 * @param {HTMLElement} el
 * @param {{state?: 'idle'|'playing'|'paused', label?: string}} [opts]
 * @returns {{setState: (state: 'idle'|'playing'|'paused') => void, destroy: () => void}}
 */
export function mountFlowStage(el, opts = {}) {
  if (!(el instanceof HTMLElement)) throw new TypeError('mountFlowStage expects an HTMLElement');
  const root = document.createElement('div');
  root.className = 'flow-fx';
  root.setAttribute('role', 'img');
  root.setAttribute('aria-label', opts.label || 'SamNPlayer Flow');
  // Unique filter/id scope if any SVG is added later; keep instance stamp.
  root.dataset.fxId = String(++nextInstanceId);

  // Corner silk (same lush assets as app chrome) + stage ribbons bridge.
  const silkAura = document.createElement('div');
  silkAura.className = 'flow-fx__silk-aura';
  silkAura.setAttribute('aria-hidden', 'true');
  silkAura.style.setProperty('--flow-fx-silk-top', `url("${SILK_TOP_URL}")`);
  silkAura.style.setProperty('--flow-fx-silk-bot', `url("${SILK_BOT_URL}")`);

  const silk = document.createElement('div');
  silk.className = 'flow-fx__silk';
  silk.setAttribute('aria-hidden', 'true');
  silk.style.setProperty('--flow-fx-silk', `url("${SILK_URL}")`);

  // Soft cyan / amber glow layers under the brand mark (mockup energy).
  const glow = document.createElement('div');
  glow.className = 'flow-fx__glow';
  glow.setAttribute('aria-hidden', 'true');

  const mark = document.createElement('img');
  mark.className = 'flow-fx__mark';
  mark.src = LOGO_URL;
  mark.alt = '';
  mark.draggable = false;

  root.append(silkAura, silk, glow, mark);
  el.appendChild(root);

  let destroyed = false;
  let inViewport = true;
  const mq = typeof window.matchMedia === 'function'
    ? window.matchMedia('(prefers-reduced-motion: reduce)')
    : null;
  const refresh = () => {
    if (destroyed) return;
    root.classList.toggle('flow-fx--animate', visibilityEnabled(root, inViewport, !!mq?.matches));
  };
  const setState = (value) => {
    if (destroyed) return;
    if (!STATES.has(value)) throw new RangeError('Unknown Flow stage state: ' + value);
    root.dataset.state = value;
  };
  const onVisibility = () => refresh();
  const observer = typeof IntersectionObserver === 'function'
    ? new IntersectionObserver((entries) => {
      inViewport = entries.some((entry) => entry.isIntersecting);
      refresh();
    }, { threshold: 0 })
    : null;

  document.addEventListener('visibilitychange', onVisibility);
  mq?.addEventListener?.('change', onVisibility);
  observer?.observe(root);
  setState(STATES.has(opts.state) ? opts.state : 'idle');
  refresh();

  return {
    setState,
    destroy() {
      if (destroyed) return;
      destroyed = true;
      observer?.disconnect();
      document.removeEventListener('visibilitychange', onVisibility);
      mq?.removeEventListener?.('change', onVisibility);
      root.remove();
    },
  };
}
