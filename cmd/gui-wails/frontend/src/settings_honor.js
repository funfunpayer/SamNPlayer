import { getSettingsCache, saveSetting } from './settings.js';
import { wireDataHelp } from './help.js';

const HELP =
  'Some .funscript files (e.g. from OpenFunscripter) carry "inverted": true. On: Play flips those files (100 \u2212 position) as the format intends. Off: they play exactly as the points are written \u2014 use this if such a file feels upside down. Review \u2192 Invert always flips what you currently see and removes the flag.';

export function enhanceHonorInverted(root) {
  const host = root || document.getElementById('tab-settings');
  if (!host || host.querySelector('#st-honor-inverted')) return;

  const block = document.createElement('div');
  block.id = 'st-honor-inverted-block';
  block.innerHTML =
    '<h3>Play</h3>' +
    '<div class="checkbox-row">' +
    '<input type="checkbox" id="st-honor-inverted" />' +
    '<label for="st-honor-inverted">Honor "inverted" flag in .funscript files</label>' +
    '</div>' +
    '<p class="hint" style="margin-top:0">On by default (same as since v0.5.44). Changing this reloads the open script.</p>';

  const label = block.querySelector('label');
  if (label) label.setAttribute('data-help', HELP);

  const connectRow = host.querySelector('#st-connect-test');
  const insertAfter = connectRow && connectRow.closest('.checkbox-row');
  const hintAfter = insertAfter && insertAfter.nextElementSibling;
  if (hintAfter && hintAfter.classList.contains('hint')) {
    hintAfter.after(block);
  } else if (insertAfter) {
    insertAfter.after(block);
  } else {
    const h2 = host.querySelector('h2');
    if (h2) h2.after(block);
    else host.prepend(block);
  }

  wireDataHelp(block);

  const box = host.querySelector('#st-honor-inverted');
  getSettingsCache().then((s) => {
    if (box) box.checked = s.playbackHonorInverted !== false;
  }).catch(() => {
    if (box) box.checked = true;
  });
  box?.addEventListener('change', (e) => {
    saveSetting('playback.honor_inverted', e.target.checked);
  });
}
