// Script folders (v0.8, docs/ROADMAP_2026-10.md §3 item 1): where SamNPlayer
// also looks for a video's script, besides the video's own folder. The list
// lives in the Go settings (GetScriptFolders / AddScriptFolder /
// RemoveScriptFolder); this card shows and edits it in Settings → Play.
import { GetScriptFolders, AddScriptFolder, RemoveScriptFolder } from '../wailsjs/go/main/App';

const esc = s => String(s).replace(/[&<>"]/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;' }[c]));

export function mountScriptFolders(host, { before = null } = {}) {
  if (!host) return null;
  const box = document.createElement('div');
  box.className = 'st-script-folders';
  box.id = 'st-script-folders';
  box.innerHTML = `
    <h3>Script folders</h3>
    <p class="hint" style="margin-top:0">When you open a video, SamNPlayer looks for its script next to the video and in these
      folders (with subfolders). Names may differ a little — “Clip (2019) [1080p]” finds “clip.funscript” — and
      variants such as “Clip (Soft)” or “Clip - by Someone” show up as a choice in the Player.</p>
    <ul class="st-folder-list" id="st-folder-list"></ul>
    <div class="row" style="align-items:center;">
      <button type="button" id="st-folder-add">Add folder…</button>
      <span class="hint" id="st-folder-status" style="margin:0" aria-live="polite"></span>
    </div>`;
  if (before && before.parentNode === host) host.insertBefore(box, before);
  else host.append(box);
  const list = box.querySelector('#st-folder-list');
  const status = box.querySelector('#st-folder-status');

  function render(dirs) {
    dirs = Array.isArray(dirs) ? dirs : [];
    list.innerHTML = dirs.length
      ? dirs.map((d, i) => `<li><span class="st-folder-path" title="${esc(d)}">${esc(d)}</span>
          <button type="button" class="secondary st-folder-remove" data-i="${i}" aria-label="Remove ${esc(d)}">Remove</button></li>`).join('')
      : '<li class="hint st-folder-empty">No folders yet — only the video\'s own folder is searched.</li>';
    list._dirs = dirs;
  }
  async function load() {
    try { render(await GetScriptFolders()); } catch (_) { render([]); }
  }
  box.querySelector('#st-folder-add').addEventListener('click', async () => {
    try {
      render(await AddScriptFolder());
      status.textContent = '';
    } catch (err) {
      status.textContent = 'Could not add: ' + err;
    }
  });
  list.addEventListener('click', async e => {
    const b = e.target.closest('.st-folder-remove');
    if (!b) return;
    const dir = (list._dirs || [])[Number(b.dataset.i)];
    if (!dir) return;
    try { render(await RemoveScriptFolder(dir)); } catch (err) { status.textContent = 'Could not remove: ' + err; }
  });
  load();
  return { load };
}
