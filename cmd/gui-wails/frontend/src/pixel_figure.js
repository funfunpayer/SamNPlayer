/** FunGen-style mosaic: woman on man, stroke motion up/down.
 *
 * You (man) = base. Partner (woman) on top moves vertically with intensity;
 * bust layer scales/bobs with the stroke. Bigger-bust partner asset.
 */

import { figureHeatColor } from './figure_theme.js';

const YOU_SRC = new URL('./assets/images/training-mosaic-you.png', import.meta.url).href;
const PARTNER_SRC = new URL('./assets/images/training-mosaic-partner.png', import.meta.url).href;
const BUST_SRC = new URL('./assets/images/training-mosaic-partner-bust.png', import.meta.url).href;

/** @deprecated kept for callers — primary UI is mountTrainingPixelStage */
export function pixelPairSVG(opts = {}) {
  const level = Math.max(0, Math.min(1, opts.level ?? 0));
  const y = Math.round(28 - level * 36);
  const aria = opts.title || `Pair motion ${Math.round(level * 100)}%`;
  return `<svg class="pixel-figure-svg pixel-pair-svg mosaic-pair-svg" viewBox="0 0 140 150"
    width="140" height="150" role="img" aria-label="${aria}">
    <rect width="140" height="150" fill="#0a0c10" rx="6"/>
    <image href="${YOU_SRC}" x="20" y="40" width="100" height="100" preserveAspectRatio="xMidYMid slice"/>
    <image href="${PARTNER_SRC}" x="22" y="${y}" width="96" height="96" opacity="0.92" preserveAspectRatio="xMidYMid slice"/>
  </svg>`;
}

export function pixelFigureSVG(opts = {}) {
  const level = Math.max(0, Math.min(1, opts.level ?? 0.5));
  const src = opts.partner ? PARTNER_SRC : YOU_SRC;
  const aria = opts.title || `Intensity ${Math.round(level * 100)}%`;
  return `<svg class="pixel-figure-svg mosaic-figure-svg" viewBox="0 0 100 100" width="72" height="72"
    role="img" aria-label="${aria}">
    <image href="${src}" x="0" y="0" width="100" height="100" preserveAspectRatio="xMidYMid slice"/>
  </svg>`;
}

/**
 * Mount Training stage: woman-on-man mosaic with up/down stroke motion.
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
        <span class="hint" id="tr-pixel-hint">Woman on man — she moves up/down with intensity; breasts follow</span>
      </div>
      <div class="tr-pixel-body">
        <div class="tr-pixel-main tr-mosaic-main" id="tr-pixel-main">
          <div class="tr-mosaic-stack" role="img" aria-label="Woman on man">
            <div class="tr-mosaic-base">
              <img class="tr-mosaic-img tr-mosaic-you" src="${YOU_SRC}" alt="" draggable="false" />
            </div>
            <div class="tr-mosaic-partner" id="tr-mosaic-partner">
              <img class="tr-mosaic-img tr-mosaic-woman" src="${PARTNER_SRC}" alt="" draggable="false" />
              <img class="tr-mosaic-bust" id="tr-mosaic-bust" src="${BUST_SRC}" alt="" draggable="false" />
              <span class="tr-mosaic-heat" aria-hidden="true"></span>
            </div>
            <div class="tr-mosaic-ground" aria-hidden="true"></div>
          </div>
        </div>
        <div class="tr-pixel-meta">
          <div class="tr-pixel-tech" id="tr-pixel-tech">Stop-start</div>
          <div class="tr-pixel-readout"><span id="tr-pixel-pct">—</span></div>
          <div class="hint" id="tr-pixel-fb">No feedback yet</div>
        </div>
      </div>
      <div class="tr-pixel-labels tr-mosaic-labels tr-mosaic-labels-stack">
        <span>You (base)</span>
        <span>Partner ♀ (moves)</span>
      </div>
    </div>`;

  let intensity = 0;
  let arousal = null;
  let technique = 'stopstart';
  let animTimer = null;
  let displayLevel = 0;
  let raf = 0;

  const partnerEl = () => host.querySelector('#tr-mosaic-partner');
  const bustEl = () => host.querySelector('#tr-mosaic-bust');
  const youImg = () => host.querySelector('.tr-mosaic-you');
  const herImg = () => host.querySelector('.tr-mosaic-woman');
  const stackEl = () => host.querySelector('.tr-mosaic-stack');
  const heatEl = () => host.querySelector('.tr-mosaic-heat');

  function applyLevel(level) {
    displayLevel = level;
    const bodyY = Math.round(36 - level * 72);
    const bustBob = Math.round(-6 - level * 14 - Math.sin(level * Math.PI) * 6);
    const bustScale = 1.08 + level * 0.18;
    const warm = figureHeatColor(Math.max(0.12, level));
    const youFilter = level < 0.05
      ? 'saturate(0.75) brightness(0.9)'
      : `saturate(${0.85 + level * 0.35}) brightness(${0.95 + level * 0.08})`;
    const herFilter = level < 0.05
      ? 'saturate(0.95) brightness(1.06) hue-rotate(-8deg)'
      : `saturate(${1.05 + level * 0.35}) brightness(${1.02 + level * 0.1}) hue-rotate(-12deg) sepia(${0.1 + level * 0.2})`;

    const p = partnerEl();
    if (p) p.style.transform = `translate(-50%, ${bodyY}px)`;
    const b = bustEl();
    if (b) b.style.transform = `translateY(${bustBob}px) scale(${bustScale})`;
    const yi = youImg();
    if (yi) yi.style.filter = youFilter;
    const hi = herImg();
    if (hi) hi.style.filter = herFilter;
    if (b) b.style.filter = herFilter;
    const st = stackEl();
    if (st) {
      st.style.setProperty('--mosaic-warm', warm);
      st.style.setProperty('--mosaic-overlay', (level * 0.14).toFixed(3));
      st.setAttribute('aria-label', `Woman on man ${Math.round(level * 100)}%`);
    }
    const heat = heatEl();
    if (heat) heat.style.opacity = String(level * 0.14);

    const pct = host.querySelector('#tr-pixel-pct');
    if (pct) {
      pct.textContent = intensity > 0
        ? `${Math.round(intensity * 100)}% · ${level > 0.55 ? 'up' : 'down'}`
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
    if (animTimer) {
      clearInterval(animTimer);
      animTimer = null;
    }
    if (raf) {
      cancelAnimationFrame(raf);
      raf = 0;
    }
  }

  /** Animate woman up→down (or hold high on plateau) toward peak intensity. */
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
