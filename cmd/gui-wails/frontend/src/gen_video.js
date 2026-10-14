// Generator video (Owner 14 Oct, #533): the clip to create from plays in a
// real <video> with transport and seeking, like the Player. Pausing shows
// the tip-box canvas over the video at exactly that frame — the box is
// drawn on the film, not on a separate still. Playing hides the canvas.
// The clip has its own URL (SetGeneratorVideo), so the Player's linked
// video never changes.
import { SetGeneratorVideo } from '../wailsjs/go/main/App';

function fmt(sec) {
  if (!Number.isFinite(sec) || sec < 0) sec = 0;
  const m = Math.floor(sec / 60), s = Math.floor(sec % 60);
  return `${m}:${String(s).padStart(2, '0')}`;
}

function btn(id, label, title) {
  const b = document.createElement('button');
  b.type = 'button';
  b.id = id;
  b.className = 'gl-vt-btn';
  b.textContent = label;
  b.title = title;
  b.setAttribute('aria-label', title);
  return b;
}

// host: the preview card; canvasWrap: #roi-canvas-wrap (moved in, kept).
export function attachGeneratorVideo(root, host, canvasWrap) {
  const box = document.createElement('div');
  box.className = 'gl-video';
  box.dataset.state = 'none';
  const video = document.createElement('video');
  video.id = 'gen-video';
  video.preload = 'metadata';
  video.playsInline = true;
  box.append(video);
  if (canvasWrap) box.append(canvasWrap);

  const bar = document.createElement('div');
  bar.className = 'gl-vt';
  bar.setAttribute('role', 'group');
  bar.setAttribute('aria-label', 'Video transport');
  const back = btn('gen-video-back', '−5s', 'Back 5 seconds');
  const play = btn('gen-video-play', '▶', 'Play');
  play.classList.add('gl-vt-play');
  const fwd = btn('gen-video-fwd', '+5s', 'Forward 5 seconds');
  const seek = document.createElement('input');
  seek.type = 'range';
  seek.id = 'gen-video-seek';
  seek.min = '0';
  seek.max = '0';
  seek.step = '0.1';
  seek.value = '0';
  seek.setAttribute('aria-label', 'Position');
  const time = document.createElement('span');
  time.className = 'gl-vt-time';
  time.id = 'gen-video-time';
  time.textContent = '0:00 / 0:00';
  bar.append(back, play, fwd, seek, time);
  for (const c of [back, play, fwd, seek]) c.disabled = true;
  host.append(box, bar);

  const q = sel => root.querySelector(sel) || document.querySelector(sel);
  let frameTimer = 0;
  // Paused at a new spot → the tip canvas loads exactly that frame
  // (Generator's own Time/Frame control, so all its logic stays in charge).
  const showFrameHere = () => {
    clearTimeout(frameTimer);
    frameTimer = setTimeout(() => {
      const t = Math.round(video.currentTime * 10) / 10;
      const input = q('#gen-seek');
      const go = q('#gen-seek-btn');
      if (!input || !go || go.disabled) return;
      if (Math.abs((parseFloat(input.value) || 0) - t) < 0.05 && box.dataset.state === 'paused') return;
      input.value = String(t);
      go.click();
    }, 180);
  };
  const sync = () => {
    const d = Number.isFinite(video.duration) ? video.duration : 0;
    seek.max = String(d);
    if (document.activeElement !== seek) seek.value = String(video.currentTime || 0);
    time.textContent = `${fmt(video.currentTime)} / ${fmt(d)}`;
  };
  const setState = s => {
    box.dataset.state = s;
    play.textContent = s === 'playing' ? '❚❚' : '▶';
    play.title = s === 'playing' ? 'Pause' : 'Play';
    play.setAttribute('aria-label', play.title);
  };

  video.addEventListener('loadedmetadata', () => {
    for (const c of [back, play, fwd, seek]) c.disabled = false;
    sync();
  });
  video.addEventListener('timeupdate', sync);
  video.addEventListener('play', () => setState('playing'));
  video.addEventListener('pause', () => { setState('paused'); showFrameHere(); });
  video.addEventListener('seeked', () => { sync(); if (video.paused) showFrameHere(); });
  video.addEventListener('error', () => { setState('none'); for (const c of [back, play, fwd, seek]) c.disabled = true; });

  play.addEventListener('click', () => { if (video.paused) video.play().catch(() => {}); else video.pause(); });
  back.addEventListener('click', () => { video.currentTime = Math.max(0, video.currentTime - 5); });
  fwd.addEventListener('click', () => { video.currentTime = Math.min(video.duration || 0, video.currentTime + 5); });
  seek.addEventListener('input', () => { video.currentTime = parseFloat(seek.value) || 0; });
  box.addEventListener('keydown', e => {
    if (e.key === ' ') { e.preventDefault(); play.click(); }
  });

  async function load(path) {
    video.pause();
    setState('none');
    for (const c of [back, play, fwd, seek]) c.disabled = true;
    if (!path) { video.removeAttribute('src'); video.load(); return; }
    try {
      const url = await SetGeneratorVideo(path);
      if (!url) return;
      video.src = url;
      setState('paused');
    } catch (_) {
      // No playable URL (e.g. file gone): the still-frame canvas still works.
    }
  }
  // The Generator's own Time / Frame / +1s / +5s controls move the video too.
  for (const id of ['#gen-seek-btn', '#gen-seek-plus', '#gen-seek-plus5']) {
    q(id)?.addEventListener('click', () => setTimeout(() => {
      const t = parseFloat(q('#gen-seek')?.value) || 0;
      if (box.dataset.state !== 'none' && Math.abs(video.currentTime - t) > 0.1) {
        video.pause();
        video.currentTime = t;
      }
    }, 0));
  }
  window.addEventListener('gen:video', e => load(e.detail && e.detail.path));
  return { video, load };
}
