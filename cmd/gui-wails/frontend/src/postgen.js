import { ReviewGeneratedScript, InvertScriptAtPath } from '../wailsjs/go/main/App';
import { EventsOn } from '../wailsjs/runtime/runtime';

export function initPostGenerateReview() {
  EventsOn('generate:done', async (payload) => {
    if (!payload || payload.error || !payload.path) return;
    let review;
    try {
      review = await ReviewGeneratedScript(payload.path);
    } catch (err) {
      return;
    }
    const pol = review.polarity || {};
    const ozone = review.ozone || {};
    const bits = [];
    if (pol.reason) bits.push('Richtung: ' + pol.reason);
    if (ozone.ok) bits.push('O-Zone vorgeschlagen: ' + Math.round(ozone.startMs / 1000) + 's–' + Math.round(ozone.endMs / 1000) + 's');
    else if (ozone.reason) bits.push('O-Zone: ' + ozone.reason);
    const status = document.querySelector('#gen-status');
    if (status && bits.length) {
      status.textContent = (status.textContent ? status.textContent + ' \n' : '') + bits.join(' ');
    }
    if (pol.suggestInvert) {
      if (confirm((pol.reason || 'Richtung unsicher') + '\n\nSkript jetzt umkehren (100 − pos)? Quality-Doctor und FunGen-r danach neu lesen.')) {
        try {
          await InvertScriptAtPath(payload.path);
          if (status) status.textContent += ' Richtung umgekehrt.';
        } catch (err) {
          if (status) status.textContent += ' Invert fehlgeschlagen: ' + err;
        }
      }
    }
  });
}
