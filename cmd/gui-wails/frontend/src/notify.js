// Zentraler UI-Hinweis ohne Browser-Popups: schreibt ins Protokoll-Tab
// und optional in ein Status-Element der aktuellen Ansicht.
export function uiLog(level, message) {
  const msg = String(message ?? '');
  if (!msg) return;
  window.dispatchEvent(new CustomEvent('ui:notify', {
    detail: {
      level: level || 'INFO',
      message: msg,
      time: new Date().toLocaleTimeString(),
    },
  }));
}

export function uiError(message, statusEl) {
  const msg = String(message ?? '');
  uiLog('ERROR', msg);
  if (statusEl) statusEl.textContent = msg;
}

export function uiWarn(message, statusEl) {
  const msg = String(message ?? '');
  uiLog('WARN', msg);
  if (statusEl) statusEl.textContent = msg;
}

export function uiInfo(message, statusEl) {
  const msg = String(message ?? '');
  uiLog('INFO', msg);
  if (statusEl) statusEl.textContent = msg;
}
