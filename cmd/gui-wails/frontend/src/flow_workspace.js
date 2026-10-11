/* Flow workspace: independent GUI-only sidebar visibility.
 * No Go/Wails calls. Never intercept shortcuts while editing text.
 */
const STORAGE_KEY = 'samn.flow.sidebarHidden';
export function initFlowWorkspace() {
  const sidebar = document.getElementById('sidebar');
  const topbar = document.getElementById('topbar');
  if (!sidebar || !topbar || document.getElementById('flow-sidebar-toggle')) return;
  const button = document.createElement('button');
  button.type = 'button';
  button.id = 'flow-sidebar-toggle';
  button.className = 'flow-workspace-toggle';
  button.setAttribute('aria-controls', 'sidebar');
  button.textContent = 'Panels';
  button.title = 'Show or hide side panels (Alt+B)';
  topbar.prepend(button);
  const apply = (hidden) => {
    document.body.classList.toggle('flow-sidebar-hidden', hidden);
    button.setAttribute('aria-expanded', String(!hidden));
    button.setAttribute('aria-pressed', String(!hidden));
    button.textContent = hidden ? 'Show panels' : 'Hide panels';
  };
  let hidden = false;
  try { hidden = localStorage.getItem(STORAGE_KEY) === '1'; } catch (_) {}
  apply(hidden);
  const toggle = () => {
    hidden = !hidden;
    apply(hidden);
    try { localStorage.setItem(STORAGE_KEY, hidden ? '1' : '0'); } catch (_) {}
  };
  button.addEventListener('click', toggle);
  window.addEventListener('keydown', (event) => {
    if (!event.altKey || event.ctrlKey || event.metaKey || event.repeat || event.code !== 'KeyB') return;
    const target = event.target;
    if (target instanceof Element && (target.closest('input,textarea,select,[contenteditable="true"]'))) return;
    event.preventDefault();
    toggle();
  });
}
