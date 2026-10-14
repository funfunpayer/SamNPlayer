// Training "Waves" (Owner 14 Oct, #533): the live training levels as calm,
// glowing curves — vibration and suction as they are sent to the device
// (training:levels, the same values the rings show). Nothing is simulated:
// before the first level arrives the panel shows a short hint, and after the
// session ends the curves stay visible.
//
// With a script plan (tr:plan carries the planned curves from
// TrainingScriptPreview) the x axis spans the whole session: the planned
// curves are drawn dashed as "Target", the live curves grow from the left
// and a playhead marks now. Without a plan (Custom form) the panel shows the
// last minute, growing from the left until the minute is full.

const WINDOW_MS = 60000;
// Same colours as the live rings (training_rings.css): vibration cyan,
// suction orange.
const VIB = '#36c8ff';
const SUC = '#ff8a23';
// ~50 ms levels: 60 000 points cover 50 minutes.
const MAX_POINTS = 60000;
// One drawn sample per this many CSS px (the mean of the levels inside),
// so the curve stays calm however dense the levels arrive.
const BIN_PX = 3;

// Smooth path through points (quadratic curves through the midpoints).
function smoothPath(ctx, pts) {
  ctx.moveTo(pts[0][0], pts[0][1]);
  if (pts.length === 2) { ctx.lineTo(pts[1][0], pts[1][1]); return; }
  for (let i = 1; i < pts.length - 1; i++) {
    const mx = (pts[i][0] + pts[i + 1][0]) / 2;
    const my = (pts[i][1] + pts[i + 1][1]) / 2;
    ctx.quadraticCurveTo(pts[i][0], pts[i][1], mx, my);
  }
  const last = pts[pts.length - 1];
  ctx.lineTo(last[0], last[1]);
}

// Mean level per bin of `binMs`; bins without a level stay gaps.
export function binLevels(points, key, fromMs, toMs, binMs) {
  const out = [];
  if (!(binMs > 0) || toMs <= fromMs) return out;
  let sum = 0, n = 0, bin = -1;
  const flush = () => { if (n) out.push([fromMs + (bin + 0.5) * binMs, sum / n]); sum = 0; n = 0; };
  for (const p of points) {
    const v = p[key];
    if (typeof v !== 'number' || p.t < fromMs || p.t > toMs) continue;
    const b = Math.floor((p.t - fromMs) / binMs);
    if (b !== bin) { flush(); bin = b; }
    sum += v; n++;
  }
  flush();
  return out;
}

// A light 3-bin average so the live curve reads calm (display only).
function soften(bins) {
  return bins.map(([t, v], i) => {
    const a = bins[i - 1], b = bins[i + 1];
    return [t, (v * 2 + (a ? a[1] : v) + (b ? b[1] : v)) / 4];
  });
}

export function createWaves(host) {
  host.classList.add('tr-waves');
  host.innerHTML = `
    <div class="tr-waves-head">
      <span class="tr-waves-title">Live training curves</span>
      <span class="tr-waves-legend"><i class="is-vib"></i>Vibration <i class="is-suc"></i>Suction
        <span class="tr-waves-target" hidden><b></b>Target</span></span>
    </div>
    <canvas class="tr-waves-canvas" id="tr-waves-canvas" aria-label="Live training curves: vibration and suction"></canvas>
    <p class="hint tr-waves-empty" id="tr-waves-empty">The curves follow the real levels once the training runs.</p>`;
  const canvas = host.querySelector('canvas');
  const empty = host.querySelector('.tr-waves-empty');
  const targetKey = host.querySelector('.tr-waves-target');
  let points = []; // {t (ms since session start), vibration, suction}
  let t0 = 0;
  // Paused time does not count (the session clock stands still), so the
  // live curve stays aligned with the planned target.
  let pausedAt = 0, pausedTotal = 0;
  window.addEventListener('tr:paused', e => {
    const now = performance.now();
    if (e.detail?.paused && !pausedAt) pausedAt = now;
    else if (!e.detail?.paused && pausedAt) { pausedTotal += now - pausedAt; pausedAt = 0; }
  });
  let plan = null; // {totalMs, vibration:[{atMs,level}], suction:[...]}
  let raf = 0;

  const hasTarget = () => !!(plan && plan.totalMs > 0 && (plan.vibration?.length || plan.suction?.length));

  function draw() {
    raf = 0;
    const dpr = window.devicePixelRatio || 1;
    const w = Math.max(1, Math.round((canvas.clientWidth || 600) * dpr));
    const h = Math.max(1, Math.round((canvas.clientHeight || 220) * dpr));
    if (canvas.width !== w || canvas.height !== h) { canvas.width = w; canvas.height = h; }
    const ctx = canvas.getContext('2d');
    ctx.clearRect(0, 0, w, h);
    const left = 38 * dpr, right = 6 * dpr, top = 8 * dpr, bottom = 8 * dpr;
    const pw = w - left - right, ph = h - top - bottom;
    const yOf = v => top + (1 - Math.max(0, Math.min(1, v))) * ph;

    // Axis: 100 / 50 / 0 % with calm grid lines.
    ctx.font = `${11 * dpr}px system-ui, sans-serif`;
    ctx.textBaseline = 'middle';
    ctx.textAlign = 'right';
    for (const v of [1, 0.75, 0.5, 0.25, 0]) {
      const y = Math.round(yOf(v)) + 0.5;
      ctx.strokeStyle = v === 0 || v === 0.5 || v === 1 ? 'rgba(160,200,255,0.12)' : 'rgba(160,200,255,0.05)';
      ctx.lineWidth = 1;
      ctx.beginPath(); ctx.moveTo(left, y); ctx.lineTo(w - right, y); ctx.stroke();
      if (v === 0 || v === 0.5 || v === 1) {
        ctx.fillStyle = 'rgba(200,220,245,0.55)';
        ctx.fillText(`${Math.round(v * 100)}%`, left - 8 * dpr, y);
      }
    }

    const full = hasTarget();
    const now = points.length ? points[points.length - 1].t : 0;
    const from = full ? 0 : Math.max(0, now - WINDOW_MS);
    const span = full ? plan.totalMs : WINDOW_MS;
    const xOf = t => left + ((t - from) / span) * pw;
    for (let i = 1; i < 6; i++) {
      const x = Math.round(left + (pw * i) / 6) + 0.5;
      ctx.strokeStyle = 'rgba(160,200,255,0.04)';
      ctx.beginPath(); ctx.moveTo(x, top); ctx.lineTo(x, top + ph); ctx.stroke();
    }

    targetKey.hidden = !full;
    if (full) {
      for (const [key, color] of [['suction', SUC], ['vibration', VIB]]) {
        const pts = (plan[key] || []).map(p => [xOf(p.atMs), yOf(p.level)]);
        if (pts.length < 2) continue;
        ctx.save();
        ctx.strokeStyle = color; ctx.globalAlpha = 0.45; ctx.lineWidth = 1.4 * dpr;
        ctx.setLineDash([5 * dpr, 5 * dpr]);
        ctx.beginPath(); smoothPath(ctx, pts); ctx.stroke();
        ctx.restore();
      }
    }

    empty.hidden = points.length > 0;
    if (points.length >= 2) {
      const binMs = (span / pw) * BIN_PX * dpr;
      for (const [key, color] of [['suction', SUC], ['vibration', VIB]]) {
        const pts = soften(binLevels(points, key, from, now, binMs)).map(([t, v]) => [xOf(t), yOf(v)]);
        if (pts.length < 2) continue;
        // Soft fill under the curve.
        ctx.save();
        ctx.beginPath(); smoothPath(ctx, pts);
        ctx.lineTo(pts[pts.length - 1][0], yOf(0)); ctx.lineTo(pts[0][0], yOf(0)); ctx.closePath();
        const g = ctx.createLinearGradient(0, yOf(1), 0, yOf(0));
        g.addColorStop(0, color); g.addColorStop(1, 'transparent');
        ctx.globalAlpha = 0.22; ctx.fillStyle = g; ctx.fill();
        ctx.restore();
        // Glow: a wide faint stroke, then the bright line.
        ctx.save();
        ctx.lineJoin = 'round'; ctx.lineCap = 'round'; ctx.strokeStyle = color;
        ctx.globalAlpha = 0.25; ctx.lineWidth = 7 * dpr; ctx.shadowColor = color; ctx.shadowBlur = 14 * dpr;
        ctx.beginPath(); smoothPath(ctx, pts); ctx.stroke();
        ctx.globalAlpha = 1; ctx.lineWidth = 2 * dpr; ctx.shadowBlur = 8 * dpr;
        ctx.beginPath(); smoothPath(ctx, pts); ctx.stroke();
        ctx.restore();
      }
      if (full) {
        const x = Math.round(xOf(now)) + 0.5;
        ctx.strokeStyle = 'rgba(255,255,255,0.55)'; ctx.lineWidth = 1 * dpr;
        ctx.beginPath(); ctx.moveTo(x, top); ctx.lineTo(x, top + ph); ctx.stroke();
      }
    }
  }
  const schedule = () => { if (!raf) raf = requestAnimationFrame(draw); };

  const setPlan = p => { plan = p || null; schedule(); };
  setPlan(window.__trPlan);
  window.addEventListener('tr:plan', e => setPlan(e.detail));
  if (typeof ResizeObserver === 'function') new ResizeObserver(schedule).observe(canvas);
  schedule();

  return {
    // Real levels (0–1) from training:levels; a missing channel stays a gap.
    push({ vibration, suction } = {}) {
      const now = performance.now();
      if (!points.length) { t0 = now; pausedTotal = 0; }
      points.push({
        t: (pausedAt || now) - t0 - pausedTotal,
        vibration: Number.isFinite(vibration) ? vibration : undefined,
        suction: Number.isFinite(suction) ? suction : undefined,
      });
      if (points.length > MAX_POINTS) points = points.slice(-MAX_POINTS);
      schedule();
    },
    reset() { points = []; schedule(); },
    redraw: schedule,
    setPlan,
  };
}
