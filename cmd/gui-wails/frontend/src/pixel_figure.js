/** Training motion — CANONICAL Vorlage = pixelated clip stroke.
 *
 * Soft mosaic assets are derived FROM the clip frames (for pixelFigureSVG
 * callers only). The stage shows the clip strip + Claude’s intensity ring
 * in ONE card (curve → frame; ring lives in #tr-ring-slot).
 */

import { figureHeatColor } from './figure_theme.js';

const YOU_SRC = new URL('./assets/images/training-mosaic-you.png', import.meta.url).href;
const PARTNER_SRC = new URL('./assets/images/training-mosaic-partner.png', import.meta.url).href;

// Each frame as its own static `new URL` so Vite emits all 16 into dist.
// (A directory-based `new URL(.../clip/, …)` does not — 0/16 in production.)
const CLIP_COUNT = 16;
const CLIP_FRAMES = [
  new URL('./assets/images/training-mosaic-clip/f00.png', import.meta.url).href,
  new URL('./assets/images/training-mosaic-clip/f01.png', import.meta.url).href,
  new URL('./assets/images/training-mosaic-clip/f02.png', import.meta.url).href,
  new URL('./assets/images/training-mosaic-clip/f03.png', import.meta.url).href,
  new URL('./assets/images/training-mosaic-clip/f04.png', import.meta.url).href,
  new URL('./assets/images/training-mosaic-clip/f05.png', import.meta.url).href,
  new URL('./assets/images/training-mosaic-clip/f06.png', import.meta.url).href,
  new URL('./assets/images/training-mosaic-clip/f07.png', import.meta.url).href,
  new URL('./assets/images/training-mosaic-clip/f08.png', import.meta.url).href,
  new URL('./assets/images/training-mosaic-clip/f09.png', import.meta.url).href,
  new URL('./assets/images/training-mosaic-clip/f10.png', import.meta.url).href,
  new URL('./assets/images/training-mosaic-clip/f11.png', import.meta.url).href,
  new URL('./assets/images/training-mosaic-clip/f12.png', import.meta.url).href,
  new URL('./assets/images/training-mosaic-clip/f13.png', import.meta.url).href,
  new URL('./assets/images/training-mosaic-clip/f14.png', import.meta.url).href,
  new URL('./assets/images/training-mosaic-clip/f15.png', import.meta.url).href,
];

/** Map training curve level (0..1) → clip frame index.
 * f00 = retracted (penis low between breasts), fN = peak (tip toward face).
 */
function frameForLevel(level) {
  const t = Math.max(0, Math.min(1, Number(level) || 0));
  return Math.min(CLIP_COUNT - 1, Math.max(0, Math.round(t * (CLIP_COUNT - 1))));
}

/** @deprecated use clip strip; kept for callers */
export function pixelPairSVG(opts = {}) {
  const level = Math.max(0, Math.min(1, opts.level ?? 0));
  const fi = frameForLevel(level);
  const aria = opts.title || `Pair motion ${Math.round(level * 100)}%`;
  return `<svg class="pixel-figure-svg pixel-pair-svg mosaic-pair-svg" viewBox="0 0 140 140"
    width="140" height="140" role="img" aria-label="${aria}">
    <image href="${CLIP_FRAMES[fi]}" x="0" y="0" width="140" height="140" preserveAspectRatio="xMidYMid meet"/>
  </svg>`;
}

/** Clip-derived soft mosaics (static callers only — stage uses clip strip). */
export function pixelFigureSVG(opts = {}) {
  const src = opts.partner ? PARTNER_SRC : YOU_SRC;
  return `<svg class="pixel-figure-svg mosaic-figure-svg" viewBox="0 0 100 100" width="72" height="72"
    role="img" aria-label="${opts.title || 'figure'}">
    <image href="${src}" x="0" y="0" width="100" height="100" preserveAspectRatio="xMidYMid slice"/>
  </svg>`;
}

/**
 * Mount Training stage: clip Vorlage only — frames follow stroke curve.
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
        <span class="hint" id="tr-pixel-hint">Pixel clip — penis between breasts follows training curve</span>
      </div>
      <div class="tr-pixel-body tr-pixel-body-with-ring">
        <div class="tr-pixel-clip-col">
          <div class="tr-pixel-main tr-mosaic-main" id="tr-pixel-main">
            <div class="tr-mosaic-stack" role="img" aria-label="Clip stroke Vorlage">
              <div class="tr-mosaic-clip-wrap">
                <img class="tr-mosaic-clip" id="tr-mosaic-clip" src="${CLIP_FRAMES[0]}" alt="" draggable="false" />
              </div>
            </div>
          </div>
          <div class="tr-pixel-labels tr-mosaic-labels">
            <span>Clip (pixel)</span>
            <span>Curve → frame</span>
          </div>
        </div>
        <div class="tr-pixel-meta">
          <div class="tr-pixel-tech" id="tr-pixel-tech">Stop-start</div>
          <div class="tr-pixel-readout"><span id="tr-pixel-pct">—</span></div>
          <div class="hint" id="tr-pixel-fb">No feedback yet</div>
        </div>
        <div class="tr-meter tr-meter-inline" id="tr-ring-slot"></div>
      </div>
    </div>`;

  // Prefetch clip frames
  CLIP_FRAMES.forEach((src) => { const im = new Image(); im.src = src; });

  let intensity = 0;
  let arousal = null;
  let technique = 'stopstart';
  let displayLevel = 0;
  let raf = 0;

  const clipEl = () => host.querySelector('#tr-mosaic-clip');
  const stackEl = () => host.querySelector('.tr-mosaic-stack');

  function applyLevel(level) {
    displayLevel = level;
    const warm = figureHeatColor(Math.max(0.12, level));

    // Curve → frame: penis between breasts moves with training intensity
    const fi = frameForLevel(level);
    const clip = clipEl();
    if (clip && clip.dataset.fi !== String(fi)) {
      clip.dataset.fi = String(fi);
      clip.src = CLIP_FRAMES[fi];
    }

    const st = stackEl();
    if (st) {
      st.style.setProperty('--mosaic-warm', warm);
      st.style.setProperty('--mosaic-overlay', (level * 0.06).toFixed(3));
      st.setAttribute('aria-label', `Clip stroke ${Math.round(level * 100)}% frame ${fi}`);
    }

    const pct = host.querySelector('#tr-pixel-pct');
    if (pct) {
      pct.textContent = intensity > 0
        ? `${Math.round(intensity * 100)}% · ${level > 0.55 ? 'up' : 'down'} · f${String(fi).padStart(2, '0')}`
        : 'Idle';
    }
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

  function stopAnim() {
    if (raf) {
      cancelAnimationFrame(raf);
      raf = 0;
    }
  }

  function runStrokeToward(peak) {
    stopAnim();
    if (peak <= 0.02) {
      applyLevel(0);
      return;
    }
    const hold = technique === 'plateau';
    let t0 = performance.now();
    const period = hold ? 2200 : 1600;
    const tick = (now) => {
      const phase = ((now - t0) % period) / period;
      let wave;
      if (hold) {
        wave = 0.72 + 0.18 * Math.sin(phase * Math.PI * 2);
      } else {
        wave = phase < 0.45
          ? phase / 0.45
          : phase < 0.55
            ? 1
            : Math.max(0, 1 - (phase - 0.55) / 0.45);
      }
      applyLevel(Math.max(0, Math.min(1, wave * peak)));
      raf = requestAnimationFrame(tick);
    };
    raf = requestAnimationFrame(tick);
  }

  applyLevel(0);
  return {
    setIntensity(level01) {
      intensity = Math.max(0, Math.min(1, Number(level01) || 0));
      runStrokeToward(intensity);
    },
    /** Live device level (training:levels) — frame follows immediately, no stroke RAF. */
    setLiveLevel(level01) {
      stopAnim();
      intensity = Math.max(0, Math.min(1, Number(level01) || 0));
      applyLevel(intensity);
    },
    setArousal(n) {
      arousal = n == null ? null : Math.max(1, Math.min(10, Math.round(n)));
      applyLevel(displayLevel);
    },
    setTechnique(t) {
      technique = t === 'plateau' ? 'plateau' : 'stopstart';
      if (intensity > 0) runStrokeToward(intensity);
      else applyLevel(displayLevel);
    },
  };
}
