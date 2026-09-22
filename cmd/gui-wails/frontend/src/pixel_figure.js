/** Pixel-art two-person figures for the Training tab (stamina / stop-start).
 *
 * Part of Claude’s figure system (`figure_theme.js`): same teal/amber heat as
 * Device shell + AI Train body map. Not AI Train itself — that stays vector.
 * These 12×16 people visualize cycle motion (curve → gap + fill).
 */

import {
  FIGURE_COOL,
  FIGURE_COOL_PARTNER,
  FIGURE_GROUND,
  FIGURE_OUTLINE,
  figureHeatColor,
} from './figure_theme.js';

/** Front-facing human mask (12×16). `.` empty, `#` body. */
const MASK_STAND = [
  '....####....',
  '...######...',
  '...##..##...',
  '...######...',
  '....####....',
  '..########..',
  '.##########.',
  '.##.####.##.',
  '.##.####.##.',
  '..########..',
  '..###..###..',
  '..###..###..',
  '..###..###..',
  '..##....##..',
  '..##....##..',
  '..##....##..',
];

/** Slightly “engaged” pose — arms in, stance narrower (peak intensity). */
const MASK_ENGAGED = [
  '....####....',
  '...######...',
  '...##..##...',
  '...######...',
  '....####....',
  '...######...',
  '..########..',
  '.##########.',
  '..########..',
  '...######...',
  '..###..###..',
  '..###..###..',
  '..###..###..',
  '..##....##..',
  '..##....##..',
  '..##....##..',
];

function bits(mask) {
  return mask.map(row => row.replace(/\./g, '0').replace(/#/g, '1'));
}

const STAND = bits(MASK_STAND);
const ENGAGED = bits(MASK_ENGAGED);

/**
 * Paint one figure into an SVG string of <rect>s.
 * @param {{ level?: number, size?: number, mask?: string[], flip?: boolean, ox?: number, oy?: number, cool?: string }} opts
 */
function figureRects(opts = {}) {
  const level = Math.max(0, Math.min(1, opts.level ?? 0.5));
  const cell = opts.size || 3;
  const cols = 12, rows = 16;
  const mask = opts.mask || STAND;
  const fillFromRow = Math.floor((1 - level) * rows);
  const hot = figureHeatColor(level);
  const cool = opts.cool || FIGURE_COOL;
  const ox = opts.ox || 0;
  const oy = opts.oy || 0;
  let rects = '';
  for (let y = 0; y < rows; y++) {
    const row = mask[y];
    for (let x = 0; x < cols; x++) {
      const sx = opts.flip ? (cols - 1 - x) : x;
      if (row[sx] !== '1') continue;
      const lit = y >= fillFromRow;
      rects += `<rect x="${ox + x * cell}" y="${oy + y * cell}" width="${cell}" height="${cell}" fill="${lit ? hot : cool}" stroke="${FIGURE_OUTLINE}" stroke-width="0.35"/>`;
    }
  }
  return { rects, w: cols * cell, h: rows * cell };
}

/**
 * Single pixel-human SVG (shared silhouette language with AI body map colors).
 * @param {{ level?: number, size?: number, title?: string, engaged?: boolean, flip?: boolean }} opts
 */
export function pixelFigureSVG(opts = {}) {
  const level = Math.max(0, Math.min(1, opts.level ?? 0.5));
  const cell = opts.size || 3;
  const mask = opts.engaged ? ENGAGED : STAND;
  const { rects, w, h } = figureRects({
    level, size: cell, mask, flip: !!opts.flip,
  });
  const aria = opts.title || `Intensity ${Math.round(level * 100)}%`;
  return `<svg class="pixel-figure-svg" viewBox="0 0 ${w} ${h}" width="${w}" height="${h}"
    role="img" aria-label="${aria}" shape-rendering="crispEdges">${rects}</svg>`;
}

/**
 * Two facing pixel people — motion viz for training intensity / curve.
 * Gap shrinks as intensity rises (pair moves together).
 * @param {{ level?: number, size?: number, title?: string }} opts
 */
export function pixelPairSVG(opts = {}) {
  const level = Math.max(0, Math.min(1, opts.level ?? 0));
  const cell = opts.size || 4;
  const cols = 12, rows = 16;
  const figW = cols * cell;
  const figH = rows * cell;
  // Gap: idle ~10 cells, peak ~2 cells between people
  const gapCells = Math.round(10 - level * 8);
  const gap = gapCells * cell;
  const engaged = level >= 0.55;
  const mask = engaged ? ENGAGED : STAND;
  // Partner (right) can read slightly cooler / offset for “curve response”
  const left = figureRects({ level, size: cell, mask, ox: 0 });
  const right = figureRects({
    level: Math.min(1, level * 0.92 + 0.05),
    size: cell,
    mask,
    flip: true,
    ox: figW + gap,
    cool: FIGURE_COOL_PARTNER,
  });
  const w = figW * 2 + gap;
  const h = figH;
  const aria = opts.title || `Pair motion ${Math.round(level * 100)}%`;
  // Soft ground line between feet (curve → motion cue)
  const groundY = h - cell * 0.5;
  const ground = `<rect x="0" y="${groundY}" width="${w}" height="${Math.max(1, cell * 0.35)}" fill="${FIGURE_GROUND}"/>`;
  return `<svg class="pixel-figure-svg pixel-pair-svg" viewBox="0 0 ${w} ${h}" width="${w}" height="${h}"
    role="img" aria-label="${aria}" shape-rendering="crispEdges">${ground}${left.rects}${right.rects}</svg>`;
}

/**
 * Mount live training two-person pixel stage (technique + intensity motion).
 * @returns {{ setIntensity(level01: number): void, setArousal(n: number|null): void, setTechnique(t: string): void }}
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
        <span class="hint" id="tr-pixel-hint">Two pixel people — closer + warmer as peak intensity rises (curve → motion)</span>
      </div>
      <div class="tr-pixel-body">
        <div class="tr-pixel-main" id="tr-pixel-main"></div>
        <div class="tr-pixel-meta">
          <div class="tr-pixel-tech" id="tr-pixel-tech">Stop-start</div>
          <div class="tr-pixel-readout"><span id="tr-pixel-pct">—</span></div>
          <div class="hint" id="tr-pixel-fb">No feedback yet</div>
        </div>
      </div>
      <div class="tr-pixel-labels">
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
      main.innerHTML = pixelPairSVG({
        level,
        size: 5,
        title: `Training pair motion ${Math.round(level * 100)}%`,
      });
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
