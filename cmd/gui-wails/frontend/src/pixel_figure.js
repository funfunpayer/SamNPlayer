/** Pixel-art human figures for the Training tab (stamina / stop-start).
 *
 * Not AI Train — that keeps the vector body map. These are small pixel
 * people that show arousal feedback (1–10) and live cycle intensity.
 */

/** 12×16 pixel silhouette — front-facing human (head, torso, arms, legs). */
const BASE_MASK = [
  // y=0..15, bits left→right (12 wide). 1 = body pixel.
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
].map(row => row.replace(/\./g, '0').replace(/#/g, '1'));

function heatColor(level01) {
  const t = Math.max(0, Math.min(1, level01));
  // cool teal → amber → warm coral
  if (t < 0.45) {
    const u = t / 0.45;
    return lerpHex('#2a6b66', '#2fd4c4', u);
  }
  if (t < 0.7) {
    const u = (t - 0.45) / 0.25;
    return lerpHex('#2fd4c4', '#f3b23c', u);
  }
  const u = (t - 0.7) / 0.3;
  return lerpHex('#f3b23c', '#ef5f5f', u);
}

function lerpHex(a, b, t) {
  const pa = hexToRgb(a), pb = hexToRgb(b);
  const r = Math.round(pa.r + (pb.r - pa.r) * t);
  const g = Math.round(pa.g + (pb.g - pa.g) * t);
  const bl = Math.round(pa.b + (pb.b - pa.b) * t);
  return `rgb(${r},${g},${bl})`;
}

function hexToRgb(hex) {
  const h = hex.replace('#', '');
  return {
    r: parseInt(h.slice(0, 2), 16),
    g: parseInt(h.slice(2, 4), 16),
    b: parseInt(h.slice(4, 6), 16),
  };
}

/**
 * Build a pixel-human SVG.
 * @param {{ level?: number, size?: number, label?: string, title?: string }} opts
 *   level 0..1 fill height from feet (intensity / arousal/10)
 */
export function pixelFigureSVG(opts = {}) {
  const level = Math.max(0, Math.min(1, opts.level ?? 0.5));
  const cell = opts.size || 3; // px per pixel
  const cols = 12, rows = 16;
  const w = cols * cell;
  const h = rows * cell;
  const fillFromRow = Math.floor((1 - level) * rows); // rows above this are dim
  const hot = heatColor(level);
  const cool = '#3a4558';
  const outline = '#1a2030';

  let rects = '';
  for (let y = 0; y < rows; y++) {
    const row = BASE_MASK[y];
    for (let x = 0; x < cols; x++) {
      if (row[x] !== '1') continue;
      const lit = y >= fillFromRow;
      const fill = lit ? hot : cool;
      rects += `<rect x="${x * cell}" y="${y * cell}" width="${cell}" height="${cell}" fill="${fill}" stroke="${outline}" stroke-width="0.35"/>`;
    }
  }

  const aria = opts.title || `Intensity ${Math.round(level * 100)}%`;
  return `<svg class="pixel-figure-svg" viewBox="0 0 ${w} ${h}" width="${w}" height="${h}"
    role="img" aria-label="${aria}" shape-rendering="crispEdges">${rects}</svg>`;
}

/** Compact markup for arousal button 1–10. */
export function arousalPixelButtonHTML(n) {
  const level = n / 10;
  const target = n === 7 ? ' is-target' : '';
  return `<button type="button" class="tr-arousal-pix${target}" data-arousal="${n}"
    title="Feedback ${n}${n === 7 ? ' (target)' : ''}">
    ${pixelFigureSVG({ level, size: 2, title: `Feedback ${n}` })}
    <span class="tr-arousal-pix-n">${n}</span>
  </button>`;
}

/**
 * Mount live training pixel stage (technique + intensity figure).
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
        <strong>Session figure</strong>
        <span class="hint" id="tr-pixel-hint">Fills with peak intensity · feedback tints the pose</span>
      </div>
      <div class="tr-pixel-body">
        <div class="tr-pixel-main" id="tr-pixel-main"></div>
        <div class="tr-pixel-meta">
          <div class="tr-pixel-tech" id="tr-pixel-tech">Stop-start</div>
          <div class="tr-pixel-readout"><span id="tr-pixel-pct">—</span></div>
          <div class="hint" id="tr-pixel-fb">No feedback yet</div>
        </div>
      </div>
    </div>`;

  let intensity = 0;
  let arousal = null;
  let technique = 'stopstart';

  function paint() {
    const level = arousal != null ? Math.max(intensity, arousal / 10) : intensity;
    const main = host.querySelector('#tr-pixel-main');
    if (main) {
      main.innerHTML = pixelFigureSVG({
        level,
        size: 5,
        title: `Training intensity ${Math.round(level * 100)}%`,
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
