import { SuggestBackend } from '../wailsjs/go/main/App';

export function enhanceGeneratorPreview(root) {
  const canvas = root.querySelector('#roi-canvas');
  const wrap = root.querySelector('#roi-canvas-wrap');
  const roiLabel = root.querySelector('#gen-roi-label');
  const roi2Label = root.querySelector('#gen-roi2-label');
  if (!canvas || !wrap) return;

  let hint = root.querySelector('#gen-roi-coach');
  if (!hint) {
    hint = document.createElement('p');
    hint.id = 'gen-roi-coach';
    hint.className = 'hint';
    hint.style.margin = '0 0 8px 0';
    wrap.insertAdjacentElement('afterend', hint);
  }

  const invertRow = root.querySelector('#gen-invert') && root.querySelector('#gen-invert').closest('.checkbox-row');
  if (invertRow && !root.querySelector('#gen-invert-visible')) {
    const clone = invertRow.cloneNode(true);
    const input = clone.querySelector('input');
    const label = clone.querySelector('label');
    if (input && label) {
      input.id = 'gen-invert-visible';
      label.htmlFor = 'gen-invert-visible';
      label.textContent = 'Invert motion direction (polarity — often FunGen difference, not a tracking bug)';
      input.addEventListener('change', () => {
        const orig = root.querySelector('#gen-invert');
        if (orig) orig.checked = input.checked;
      });
      hint.insertAdjacentElement('afterend', clone);
    }
  }

  const overlay = document.createElement('canvas');
  overlay.id = 'roi-help-overlay';
  overlay.style.cssText = 'position:absolute;left:0;top:0;pointer-events:none;';
  wrap.appendChild(overlay);

  let lastBackendHint = '';

  function parseBox(text) {
    const m = text && text.match(/x=(\d+)\s+y=(\d+)\s+w=(\d+)\s+h=(\d+)/);
    if (!m) return null;
    return { x: +m[1], y: +m[2], w: +m[3], h: +m[4] };
  }

  function backendHintLocal(r1) {
    if (!r1) return '';
    if (r1.w * r1.h < 800 || r1.w < 24 || r1.h < 24) {
      return 'Small ROI: grid/optical-flow tracking is usually more robust than CSRT.';
    }
    return 'ROI size favors CSRT.';
  }

  async function refreshBackendHint(r1) {
    if (!r1) { lastBackendHint = ''; return; }
    try {
      const name = await SuggestBackend(r1.w, r1.h);
      if (name && typeof name === 'string') {
        if (name.toLowerCase().includes('grid') || name.toLowerCase().includes('flow') || name.toLowerCase().includes('lk')) {
          lastBackendHint = 'SuggestBackend: ' + name + ' (small ROI).';
        } else {
          lastBackendHint = 'SuggestBackend: ' + name + '.';
        }
        return;
      }
    } catch (_) { /* fallback below */ }
    lastBackendHint = backendHintLocal(r1);
  }

  function coachText(r1, r2) {
    const bits = [];
    if (r1) {
      if (r1.w * r1.h < 400) bits.push('ROI1 is very small — tracking may lose lock.');
      if (r1.x < 4 || r1.y < 4) bits.push('ROI1 is flush with the image edge.');
      if (lastBackendHint) bits.push(lastBackendHint);
      else bits.push(backendHintLocal(r1));
    } else {
      bits.push('Set ROI1 on moving stroke, not just the tip.');
    }
    if (r1 && r2) {
      const dx = (r1.x + r1.w / 2) - (r2.x + r2.w / 2);
      const dy = (r1.y + r1.h / 2) - (r2.y + r2.h / 2);
      if (Math.hypot(dx, dy) < 12) bits.push('ROI2 almost on ROI1 — distance signal collapses.');
      if (Math.abs(dy) < 4 && Math.abs(dx) > 8) {
        bits.push('ROI2 is only shifted sideways (same height) — poor anchor.');
      }
      bits.push('Line = measured distance (Tf/Tj). ROI2 should be a counter-anchor, not a copy.');
    } else if (r1 && !r2) {
      bits.push('For Tf/Tj set a second anchor — do not copy ROI1 sideways.');
    }
    return bits.filter(Boolean).join(' ');
  }

  function draw() {
    overlay.width = canvas.width;
    overlay.height = canvas.height;
    const ctx = overlay.getContext('2d');
    ctx.clearRect(0, 0, overlay.width, overlay.height);
    const r1 = parseBox(roiLabel && roiLabel.textContent);
    const r2 = parseBox(roi2Label && roi2Label.textContent);
    if (r1) refreshBackendHint(r1).then(() => { hint.textContent = coachText(r1, r2); });
    else hint.textContent = coachText(r1, r2);
    const nw = Number(canvas.dataset.nativeW || 0);
    const nh = Number(canvas.dataset.nativeH || 0);
    if (!r1 || !r2 || !nw || !nh) return;
    const sx = overlay.width / nw, sy = overlay.height / nh;
    const a = { x: (r1.x + r1.w / 2) * sx, y: (r1.y + r1.h / 2) * sy };
    const b = { x: (r2.x + r2.w / 2) * sx, y: (r2.y + r2.h / 2) * sy };
    ctx.strokeStyle = 'rgba(242,176,61,0.9)';
    ctx.lineWidth = 2;
    ctx.setLineDash([5, 4]);
    ctx.beginPath();
    ctx.moveTo(a.x, a.y);
    ctx.lineTo(b.x, b.y);
    ctx.stroke();
    ctx.setLineDash([]);
    ctx.fillStyle = '#f2b03d';
    ctx.beginPath(); ctx.arc(a.x, a.y, 4, 0, Math.PI * 2); ctx.fill();
    ctx.beginPath(); ctx.arc(b.x, b.y, 4, 0, Math.PI * 2); ctx.fill();
  }

  const obs = new MutationObserver(draw);
  if (roiLabel) obs.observe(roiLabel, { childList: true, characterData: true, subtree: true });
  if (roi2Label) obs.observe(roi2Label, { childList: true, characterData: true, subtree: true });
  canvas.addEventListener('mouseup', () => setTimeout(draw, 0));
  window.addEventListener('mouseup', () => setTimeout(draw, 0));
  draw();
}

export function rememberNativeSize(canvas, nativeW, nativeH) {
  if (!canvas) return;
  canvas.dataset.nativeW = String(nativeW);
  canvas.dataset.nativeH = String(nativeH);
}
