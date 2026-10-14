// Training "Waves" stage (Owner 14 Oct, #533): the live training levels as
// glowing waves over the last minute — vibration and suction as they are
// sent to the device (training:levels, the same values the rings show).
// Nothing is simulated: before the first level arrives the stage shows a
// short hint, and after the session ends the last minute stays visible.

const WINDOW_MS = 60000;
// Same colours as the live rings (training_rings.css): vibration cyan,
// suction orange.
const VIB = '#36c8ff';
const SUC = '#ff8a23';
const MAX_POINTS = 1500;

export function createWaves(host) {
  host.classList.add('tr-waves');
  host.innerHTML = `
    <div class="tr-waves-head">
      <span class="tr-waves-title">Live training curves</span>
      <span class="tr-waves-legend"><i class="is-vib"></i>Vibration <i class="is-suc"></i>Suction</span>
    </div>
    <canvas class="tr-waves-canvas" id="tr-waves-canvas" aria-label="Live training curves: vibration and suction over the last minute"></canvas>
    <p class="hint tr-waves-empty" id="tr-waves-empty">The waves follow the real levels once the training runs.</p>`;
  const canvas = host.querySelector('canvas');
  const empty = host.querySelector('.tr-waves-empty');
  let points = []; // {t, vibration, suction}
  let raf = 0;

  function draw() {
    raf = 0;
    const dpr = window.devicePixelRatio || 1;
    const w = Math.max(1, Math.round((canvas.clientWidth || 600) * dpr));
    const h = Math.max(1, Math.round((canvas.clientHeight || 220) * dpr));
    if (canvas.width !== w || canvas.height !== h) { canvas.width = w; canvas.height = h; }
    const ctx = canvas.getContext('2d');
    ctx.clearRect(0, 0, w, h);
    const pad = 10 * dpr;
    const yOf = v => pad + (1 - Math.max(0, Math.min(1, v))) * (h - pad * 2);
    ctx.strokeStyle = 'rgba(255,255,255,0.08)';
    ctx.lineWidth = 1;
    for (const v of [0, 0.5, 1]) {
      const y = Math.round(yOf(v)) + 0.5;
      ctx.beginPath(); ctx.moveTo(0, y); ctx.lineTo(w, y); ctx.stroke();
    }
    empty.hidden = points.length > 0;
    if (points.length < 2) return;
    const end = points[points.length - 1].t;
    const xOf = t => w - ((end - t) / WINDOW_MS) * w;
    const line = (key, color) => {
      const pts = points.filter(p => typeof p[key] === 'number');
      if (pts.length < 2) return;
      const path = () => {
        ctx.beginPath();
        pts.forEach((p, i) => (i ? ctx.lineTo(xOf(p.t), yOf(p[key])) : ctx.moveTo(xOf(p.t), yOf(p[key]))));
      };
      path();
      ctx.lineTo(xOf(pts[pts.length - 1].t), yOf(0));
      ctx.lineTo(xOf(pts[0].t), yOf(0));
      ctx.closePath();
      const g = ctx.createLinearGradient(0, yOf(1), 0, yOf(0));
      g.addColorStop(0, color);
      g.addColorStop(1, 'transparent');
      ctx.save(); ctx.globalAlpha = 0.18; ctx.fillStyle = g; ctx.fill(); ctx.restore();
      ctx.save();
      ctx.strokeStyle = color; ctx.lineWidth = 2 * dpr; ctx.lineJoin = 'round';
      ctx.shadowColor = color; ctx.shadowBlur = 10 * dpr;
      path(); ctx.stroke();
      ctx.restore();
    };
    line('suction', SUC);
    line('vibration', VIB);
  }
  const schedule = () => { if (!raf) raf = requestAnimationFrame(draw); };

  if (typeof ResizeObserver === 'function') new ResizeObserver(schedule).observe(canvas);
  schedule();

  return {
    // Real levels (0–1) from training:levels; a missing channel stays a gap.
    push({ vibration, suction } = {}) {
      const t = performance.now();
      points.push({
        t,
        vibration: Number.isFinite(vibration) ? vibration : undefined,
        suction: Number.isFinite(suction) ? suction : undefined,
      });
      while (points.length && t - points[0].t > WINDOW_MS) points.shift();
      if (points.length > MAX_POINTS) points = points.slice(-MAX_POINTS);
      schedule();
    },
    reset() { points = []; schedule(); },
    redraw: schedule,
  };
}
