/** FunGen-style mosaic body pairs for the Training tab (You + Partner).
 *
 * Not Minecraft voxels — a photoreal torso shown as chunky mosaic pixels
 * (same look as FunGen 2 marketing). Intensity drives proximity + warmth.
 * Colors still share Claude’s figure theme for overlays.
 */

import { FIGURE_AMBER, FIGURE_TEAL, figureHeatColor } from './figure_theme.js';

const YOU_SRC = new URL('./assets/images/training-mosaic-you.png', import.meta.url).href;
const PARTNER_SRC = new URL('./assets/images/training-mosaic-partner.png', import.meta.url).href;

/**
 * Mosaic pair markup (img-based FunGen look).
 * @param {{ level?: number, title?: string }} opts
 */
export function pixelPairSVG(opts = {}) {
  // Kept name for callers/tests — returns SVG-wrapped foreignObject OR a
  // compact SVG badge; primary UI uses mountTrainingPixelStage HTML.
  const level = Math.max(0, Math.min(1, opts.level ?? 0));
  const gap = Math.round(28 - level * 20);
  const warm = figureHeatColor(Math.max(0.2, level));
  const aria = opts.title || `Pair motion ${Math.round(level * 100)}%`;
  const filter = level < 0.08
    ? 'saturate(0.75) brightness(0.92)'
    : `saturate(${0.9 + level * 0.45}) brightness(${0.95 + level * 0.12}) sepia(${level * 0.35})`;
  return `<svg class="pixel-figure-svg pixel-pair-svg mosaic-pair-svg" viewBox="0 0 220 120"
    width="220" height="120" role="img" aria-label="${aria}">
    <defs>
      <filter id="mosaicWarm"><feColorMatrix type="matrix" values="1 0 0 0 0  0 1 0 0 0  0 0 1 0 0  0 0 0 1 0"/></filter>
    </defs>
    <rect x="0" y="0" width="220" height="120" fill="#0a0c10" rx="6"/>
    <image href="${YOU_SRC}" x="8" y="8" width="90" height="90"
      style="filter:${filter}" preserveAspectRatio="xMidYMid slice"/>
    <image href="${PARTNER_SRC}" x="${102 + gap}" y="8" width="90" height="90"
      style="filter:${filter}" preserveAspectRatio="xMidYMid slice"/>
    <ellipse cx="110" cy="108" rx="${70 - level * 18}" ry="5"
      fill="${FIGURE_TEAL}" opacity="${0.12 + level * 0.25}"/>
    ${level > 0.5 ? `<ellipse cx="110" cy="106" rx="36" ry="3" fill="${FIGURE_AMBER}" opacity="${(level - 0.5) * 0.4}"/>` : ''}
    <rect x="8" y="8" width="90" height="90" fill="${warm}" opacity="${level * 0.12}" style="mix-blend-mode:overlay"/>
    <rect x="${102 + gap}" y="8" width="90" height="90" fill="${warm}" opacity="${level * 0.12}" style="mix-blend-mode:overlay"/>
  </svg>`;
}

/** Single mosaic figure (uses You asset). */
export function pixelFigureSVG(opts = {}) {
  const level = Math.max(0, Math.min(1, opts.level ?? 0.5));
  const filter = level < 0.08
    ? 'saturate(0.75) brightness(0.92)'
    : `saturate(${0.9 + level * 0.45}) brightness(${0.95 + level * 0.12}) sepia(${level * 0.35})`;
  const aria = opts.title || `Intensity ${Math.round(level * 100)}%`;
  const src = opts.partner ? PARTNER_SRC : YOU_SRC;
  return `<svg class="pixel-figure-svg mosaic-figure-svg" viewBox="0 0 100 100" width="72" height="72"
    role="img" aria-label="${aria}">
    <image href="${src}" x="0" y="0" width="100" height="100"
      style="filter:${filter}" preserveAspectRatio="xMidYMid slice"/>
  </svg>`;
}

/**
 * Mount live Training mosaic stage (FunGen-style pixel torsos).
 */
export function mountTrainingPixelStage(host) {
  if (!host) {
    return { setIntensity() {}, setArousal() {}, setTechnique() {} };
  }
  host.classList.add('tr-pixel-stage');
  host.innerHTML = `
    <div class="tr-pixel-card">
      <div class="tr-pixel-head">
        <strong>Motion (pair)</strong>
        <span class="hint" id="tr-pixel-hint">FunGen-style mosaic bodies — closer + warmer as peak rises</span>
      </div>
      <div class="tr-pixel-body">
        <div class="tr-pixel-main tr-mosaic-main" id="tr-pixel-main"></div>
        <div class="tr-pixel-meta">
          <div class="tr-pixel-tech" id="tr-pixel-tech">Stop-start</div>
          <div class="tr-pixel-readout"><span id="tr-pixel-pct">—</span></div>
          <div class="hint" id="tr-pixel-fb">No feedback yet</div>
        </div>
      </div>
      <div class="tr-pixel-labels tr-mosaic-labels">
        <span>You</span>
        <span>Partner</span>
      </div>
    </div>`;

  let intensity = 0;
  let arousal = null;
  let technique = 'stopstart';

  function paint() {
    const level = intensity;
    const main = host.querySelector('#tr-pixel-main');
    if (main) {
      const gap = Math.round(36 - level * 28);
      const warm = figureHeatColor(Math.max(0.15, level));
      const filter = level < 0.06
        ? 'saturate(0.7) brightness(0.9)'
        : `saturate(${0.85 + level * 0.55}) brightness(${0.94 + level * 0.14}) sepia(${level * 0.4})`;
      main.innerHTML = `
        <div class="tr-mosaic-stage" style="--mosaic-gap:${gap}px; --mosaic-warm:${warm}; --mosaic-overlay:${(level * 0.18).toFixed(3)};"
             role="img" aria-label="Training pair ${Math.round(level * 100)}%">
          <div class="tr-mosaic-slot">
            <img class="tr-mosaic-img" src="${YOU_SRC}" alt="" draggable="false" style="filter:${filter}" />
            <span class="tr-mosaic-heat" aria-hidden="true"></span>
          </div>
          <div class="tr-mosaic-slot">
            <img class="tr-mosaic-img" src="${PARTNER_SRC}" alt="" draggable="false" style="filter:${filter}" />
            <span class="tr-mosaic-heat" aria-hidden="true"></span>
          </div>
        </div>`;
    }
    const pct = host.querySelector('#tr-pixel-pct');
    if (pct) pct.textContent = intensity > 0 ? `${Math.round(intensity * 100)}% peak` : 'Idle';
    const tech = host.querySelector('#tr-pixel-tech');
    if (tech) {
      tech.textContent = technique === 'plateau' ? 'Plateau (edge)' : 'Stop-start';
      tech.dataset.tech = technique;
    }
    const fb = host.querySelector('#tr-pixel-fb');
    if (fb) {
      fb.textContent = arousal != null
        ? `Last feedback ${arousal}${arousal === 7 ? ' · on target' : ''}`
        : 'No feedback yet';
    }
  }

  paint();
  return {
    setIntensity(level01) {
      intensity = Math.max(0, Math.min(1, Number(level01) || 0));
      paint();
    },
    setArousal(n) {
      arousal = n == null ? null : Math.max(1, Math.min(10, Math.round(n)));
      paint();
    },
    setTechnique(t) {
      technique = t === 'plateau' ? 'plateau' : 'stopstart';
      paint();
    },
  };
}
