/** Isometric 3D voxel people for the Training tab (You + Partner).
 *
 * Part of Claude’s figure system (`figure_theme.js`). Flat 2D pixels were
 * rejected — these are proper voxel humans (top/left/right faces, depth sort)
 * so curve intensity reads as real motion, not a heat bar.
 */

import {
  FIGURE_AMBER,
  FIGURE_COOL,
  FIGURE_GROUND,
  FIGURE_TEAL,
  figureHeatColor,
} from './figure_theme.js';

/**
 * Voxel set for a standing adult (grid coords, Y up).
 * Clear head / torso / arms / legs so it reads as a person in isometric.
 */
function buildStandVoxels() {
  const v = [];
  const add = (x, y, z) => v.push({ x, y, z });
  // feet + legs (split)
  for (let y = 0; y <= 6; y++) {
    add(1, y, 2); add(2, y, 2); add(1, y, 3); add(2, y, 3);
    add(5, y, 2); add(6, y, 2); add(5, y, 3); add(6, y, 3);
  }
  // hips
  for (let x = 1; x <= 6; x++) {
    add(x, 7, 2); add(x, 7, 3);
    add(x, 8, 2); add(x, 8, 3);
  }
  // torso (thicker)
  for (let y = 9; y <= 14; y++) {
    for (let x = 2; x <= 5; x++) {
      add(x, y, 1); add(x, y, 2); add(x, y, 3);
    }
  }
  // chest flare + shoulders
  for (let x = 1; x <= 6; x++) {
    add(x, 14, 2); add(x, 14, 3); add(x, 15, 2); add(x, 15, 3);
  }
  // arms hanging
  for (let y = 10; y <= 14; y++) {
    add(0, y, 2); add(0, y, 3);
    add(7, y, 2); add(7, y, 3);
  }
  add(0, 9, 2); add(7, 9, 2);
  // neck
  add(3, 16, 2); add(4, 16, 2); add(3, 16, 3); add(4, 16, 3);
  // head (blocky but rounder footprint)
  for (let y = 17; y <= 21; y++) {
    for (let x = 2; x <= 5; x++) {
      for (let z = 1; z <= 3; z++) add(x, y, z);
    }
  }
  // hair / crown
  for (let x = 2; x <= 5; x++) add(x, 22, 2);
  return v;
}

/** Engaged: arms in, stance narrower. */
function buildEngagedVoxels() {
  const v = [];
  const add = (x, y, z) => v.push({ x, y, z });
  for (let y = 0; y <= 6; y++) {
    add(2, y, 2); add(3, y, 2); add(2, y, 3); add(3, y, 3);
    add(4, y, 2); add(5, y, 2); add(4, y, 3); add(5, y, 3);
  }
  for (let x = 2; x <= 5; x++) {
    add(x, 7, 2); add(x, 7, 3); add(x, 8, 2); add(x, 8, 3);
  }
  for (let y = 9; y <= 15; y++) {
    for (let x = 2; x <= 5; x++) {
      add(x, y, 1); add(x, y, 2); add(x, y, 3);
    }
  }
  for (let x = 1; x <= 6; x++) {
    add(x, 15, 2); add(x, 15, 3);
  }
  for (let y = 11; y <= 14; y++) {
    add(1, y, 2); add(6, y, 2);
  }
  add(3, 16, 2); add(4, 16, 2);
  for (let y = 17; y <= 21; y++) {
    for (let x = 2; x <= 5; x++) {
      for (let z = 1; z <= 3; z++) add(x, y, z);
    }
  }
  for (let x = 2; x <= 5; x++) add(x, 22, 2);
  return v;
}

const STAND = buildStandVoxels();
const ENGAGED = buildEngagedVoxels();

function hexToRgb(hex) {
  const s = String(hex).trim();
  const rgb = s.match(/^rgb\(\s*(\d+)\s*,\s*(\d+)\s*,\s*(\d+)\s*\)$/i);
  if (rgb) {
    return { r: +rgb[1], g: +rgb[2], b: +rgb[3] };
  }
  const h = s.replace('#', '');
  if (h.length !== 6) return { r: 60, g: 70, b: 90 };
  return {
    r: parseInt(h.slice(0, 2), 16),
    g: parseInt(h.slice(2, 4), 16),
    b: parseInt(h.slice(4, 6), 16),
  };
}

function toHex(r, g, b) {
  return `#${[clamp(r)|0, clamp(g)|0, clamp(b)|0].map(n => n.toString(16).padStart(2, '0')).join('')}`;
}

function clamp(n) {
  return Math.max(0, Math.min(255, n));
}

function shade(hex, factor) {
  const { r, g, b } = hexToRgb(hex);
  return toHex(r * factor, g * factor, b * factor);
}

function mixHex(a, b, t) {
  const pa = hexToRgb(a), pb = hexToRgb(b);
  const u = Math.max(0, Math.min(1, t));
  return toHex(
    pa.r + (pb.r - pa.r) * u,
    pa.g + (pb.g - pa.g) * u,
    pa.b + (pb.b - pa.b) * u,
  );
}

/** Body base color: cool idle → Claude heat at intensity. */
function bodyColor(level, partner) {
  const cool = partner ? '#5a6478' : FIGURE_COOL;
  const hot = figureHeatColor(Math.max(0.15, level));
  if (level < 0.08) return cool;
  return mixHex(cool, hot, Math.min(1, level * 1.2));
}

/**
 * Project voxel (x,y,z) → screen. Classic game isometric.
 * y is up. Returns top-face diamond origin.
 */
function project(x, y, z, scale) {
  const s = scale;
  const sx = (x - z) * (s * 0.866);
  const sy = (x + z) * (s * 0.5) - y * s;
  return { sx, sy };
}

/**
 * Draw one isometric cube as 3 polygons (top / left / right).
 */
function cubeSVG(x, y, z, scale, base, ox, oy) {
  const s = scale;
  const { sx, sy } = project(x, y, z, s);
  const cx = ox + sx;
  const cy = oy + sy;
  // diamond corners for top face
  const t = [
    [cx, cy - s * 0.5],
    [cx + s * 0.866, cy],
    [cx, cy + s * 0.5],
    [cx - s * 0.866, cy],
  ];
  const top = shade(base, 1.18);
  const left = shade(base, 0.72);
  const right = shade(base, 0.88);
  const h = s; // vertical edge length (matches -y * s step)
  const topP = t.map(p => p.join(',')).join(' ');
  const leftP = [
    t[3], t[2],
    [t[2][0], t[2][1] + h],
    [t[3][0], t[3][1] + h],
  ].map(p => p.join(',')).join(' ');
  const rightP = [
    t[2], t[1],
    [t[1][0], t[1][1] + h],
    [t[2][0], t[2][1] + h],
  ].map(p => p.join(',')).join(' ');
  return `<polygon points="${leftP}" fill="${left}" stroke="rgba(0,0,0,0.22)" stroke-width="0.4"/>`
    + `<polygon points="${rightP}" fill="${right}" stroke="rgba(0,0,0,0.18)" stroke-width="0.4"/>`
    + `<polygon points="${topP}" fill="${top}" stroke="rgba(0,0,0,0.12)" stroke-width="0.35"/>`;
}

/**
 * Render a voxel person into SVG markup + bounds.
 * @param {{ level?: number, scale?: number, flip?: boolean, partner?: boolean, engaged?: boolean, ox?: number, oy?: number }} opts
 */
function renderVoxelPerson(opts = {}) {
  const level = Math.max(0, Math.min(1, opts.level ?? 0));
  const scale = opts.scale || 5;
  const voxels = (opts.engaged ? ENGAGED : STAND).slice();
  // intensity: light more voxels from feet (y) upward
  const maxY = 22;
  const base = bodyColor(Math.max(level, 0.12), !!opts.partner);
  // Feet hotter than head — subtle vertical heat, whole body still readable
  function colorAt(y) {
    const footBoost = (1 - y / maxY) * 0.2 * level;
    const t = Math.min(1, level + footBoost);
    return bodyColor(Math.max(t, 0.12), !!opts.partner);
  }

  // flip around X for partner facing inward
  const mapped = voxels.map(v => ({
    x: opts.flip ? (7 - v.x) : v.x,
    y: v.y,
    z: v.z,
  }));

  // painter's algorithm: back → front
  mapped.sort((a, b) => (a.x + a.z + a.y * 0.01) - (b.x + b.z + b.y * 0.01));

  let minX = Infinity, minY = Infinity, maxX = -Infinity, maxY2 = -Infinity;
  const probe = (px, py) => {
    minX = Math.min(minX, px); maxX = Math.max(maxX, px);
    minY = Math.min(minY, py); maxY2 = Math.max(maxY2, py);
  };

  let mark = '';
  const ox = opts.ox || 0;
  const oy = opts.oy || 0;
  for (const v of mapped) {
    const col = level < 0.06 ? shade(base, 0.85) : colorAt(v.y);
    mark += cubeSVG(v.x, v.y, v.z, scale, col, ox, oy);
    const { sx, sy } = project(v.x, v.y, v.z, scale);
    probe(ox + sx - scale, oy + sy - scale);
    probe(ox + sx + scale, oy + sy + scale * 1.5);
  }
  return { mark, minX, minY, maxX, maxY: maxY2 };
}

/**
 * Single 3D voxel human SVG.
 */
export function pixelFigureSVG(opts = {}) {
  const level = Math.max(0, Math.min(1, opts.level ?? 0.5));
  const scale = opts.size || 5;
  const r = renderVoxelPerson({
    level,
    scale,
    flip: !!opts.flip,
    engaged: !!opts.engaged,
    partner: !!opts.partner,
    ox: 40,
    oy: 110,
  });
  const pad = 4;
  const w = Math.ceil(r.maxX - r.minX + pad * 2);
  const h = Math.ceil(r.maxY - r.minY + pad * 2);
  const aria = opts.title || `Intensity ${Math.round(level * 100)}%`;
  return `<svg class="pixel-figure-svg voxel-figure-svg" viewBox="${r.minX - pad} ${r.minY - pad} ${w} ${h}"
    width="${Math.max(48, w)}" height="${Math.max(64, h)}"
    role="img" aria-label="${aria}">${r.mark}</svg>`;
}

/**
 * Two facing 3D voxel people — motion viz for training intensity.
 */
export function pixelPairSVG(opts = {}) {
  const level = Math.max(0, Math.min(1, opts.level ?? 0));
  const scale = opts.size || 5;
  const engaged = level >= 0.55;
  // Gap in world-ish screen units: idle far, peak close
  const gap = Math.round(72 - level * 52);

  const left = renderVoxelPerson({
    level,
    scale,
    engaged,
    partner: false,
    flip: false,
    ox: 36,
    oy: 118,
  });
  const right = renderVoxelPerson({
    level: Math.min(1, level * 0.95 + 0.04),
    scale,
    engaged,
    partner: true,
    flip: true,
    ox: 36 + gap + 28,
    oy: 118,
  });

  const minX = Math.min(left.minX, right.minX) - 6;
  const minY = Math.min(left.minY, right.minY) - 6;
  const maxX = Math.max(left.maxX, right.maxX) + 6;
  const maxY = Math.max(left.maxY, right.maxY) + 10;
  const w = maxX - minX;
  const h = maxY - minY;

  // ground ellipse under feet
  const gx = (minX + maxX) / 2;
  const gy = maxY - 8;
  const ground = `<ellipse cx="${gx}" cy="${gy}" rx="${w * 0.38}" ry="${scale * 0.55}"
    fill="${FIGURE_GROUND}" opacity="0.85"/>`
    + `<ellipse cx="${gx}" cy="${gy}" rx="${w * 0.38}" ry="${scale * 0.55}"
    fill="none" stroke="${FIGURE_TEAL}" stroke-opacity="${0.15 + level * 0.35}" stroke-width="1"/>`;

  // soft amber rim at high intensity (ties to Claude Device vib)
  const glow = level > 0.55
    ? `<ellipse cx="${gx}" cy="${gy - 4}" rx="${w * 0.22}" ry="${scale * 0.35}"
        fill="${FIGURE_AMBER}" opacity="${(level - 0.55) * 0.35}"/>`
    : '';

  const aria = opts.title || `Pair motion ${Math.round(level * 100)}%`;
  return `<svg class="pixel-figure-svg pixel-pair-svg voxel-pair-svg"
    viewBox="${minX} ${minY} ${w} ${h}" width="${Math.round(w)}" height="${Math.round(h)}"
    role="img" aria-label="${aria}">${ground}${glow}${left.mark}${right.mark}</svg>`;
}

/**
 * Mount live training two-person 3D voxel stage.
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
        <span class="hint" id="tr-pixel-hint">3D voxel You + Partner — closer + warmer as peak rises</span>
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
    const main = host.querySelector('#tr-pixel-main');
    if (main) {
      main.innerHTML = pixelPairSVG({
        level: intensity,
        size: 5.5,
        title: `Training pair ${Math.round(intensity * 100)}%`,
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
