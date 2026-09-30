import { CancelRoiTraining } from '../wailsjs/go/main/App';
import { EventsOn } from '../wailsjs/runtime/runtime';
import { uiError, uiWarn } from './notify.js';

function isCancelError(err) {
  return /cancel|abgebrochen|context canceled/i.test(String(err || ''));
}

function addCancelButton(nextTo, id, title) {
  if (!nextTo || document.getElementById(id)) return document.getElementById(id);
  const btn = document.createElement('button');
  btn.id = id;
  btn.type = 'button';
  btn.textContent = 'Cancel';
  btn.hidden = true;
  btn.setAttribute('data-help', title);
  nextTo.after(btn);
  btn.addEventListener('click', async () => {
    try {
      const ok = await CancelRoiTraining();
      if (!ok) uiWarn('No AI train run to cancel.');
    } catch (err) {
      uiError('Cancel training: ' + err);
    }
  });
  return btn;
}

export function enhanceRoiCancel(root) {
  const host = root || document.getElementById('tab-roi-training');
  if (!host) return;

  const bootBtn = host.querySelector('#rt-bootstrap');
  const trainBtn = host.querySelector('#rt-train');
  const cancelBoot = addCancelButton(
    bootBtn,
    'rt-cancel-run',
    'Stops the running bootstrap or YOLO training. Closing the app does the same.',
  );
  const cancelTrain = addCancelButton(
    trainBtn,
    'rt-cancel-train',
    'Stops the running YOLO training (minutes to hours). Closing the app also cancels.',
  );

  function setVisible(show) {
    if (cancelBoot) cancelBoot.hidden = !show;
    if (cancelTrain) cancelTrain.hidden = !show;
  }

  const origClick = (el) => {
    if (!el) return;
    el.addEventListener('click', () => {
      if (!el.disabled) setVisible(true);
    });
  };
  origClick(bootBtn);
  origClick(trainBtn);

  EventsOn('roitraining:bootstrap:done', (payload) => {
    setVisible(false);
    if (payload && payload.error && isCancelError(payload.error)) {
      uiWarn('Bootstrap cancelled: ' + payload.error, host.querySelector('#rt-bootstrap-status'));
    }
  });
  EventsOn('roitraining:train:done', (payload) => {
    setVisible(false);
    if (payload && payload.error && isCancelError(payload.error)) {
      uiWarn('Training cancelled: ' + payload.error, host.querySelector('#rt-train-status'));
    }
  });
}
