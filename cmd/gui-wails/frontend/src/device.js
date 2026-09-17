import { GetDeviceStatus, ConnectDevice, DisconnectDevice, TestVibration, TestSuction, TestStop, TestRawValue, ConnectDeviceVia, GetSettings, RunDeviceDiagnostics, GetDiagnosticsHistory } from '../wailsjs/go/main/App';
import { EventsOn } from '../wailsjs/runtime/runtime';
import { wireDataHelp } from './help.js';

// Zweck dieses Tabs: sichtbar machen, ob überhaupt ein Gerät gefunden und
// richtig erkannt wurde, und die Ansteuerung isoliert prüfen zu können -
// ohne dafür eine Wiedergabe starten zu müssen.

export function initDevice(root) {
  root.innerHTML = `
    <h2>Gerät</h2>

    <div id="dev-status" class="row">
      <div class="dev-visual" aria-hidden="true">
        <div class="dev-shell">
          <div class="dev-fill-suc" id="dev-fill-suc" style="height:0%"></div>
          <div class="dev-fill-vib" id="dev-fill-vib" style="height:0%"></div>
        </div>
      </div>
      <div class="dev-status-meta">
        <div class="dev-status-line">
          <span id="dev-dot"></span>
          <span id="dev-status-text">Status wird geladen...</span>
        </div>
        <p class="hint" id="dev-status-sub" style="margin:0">Noch nicht verbunden.</p>
      </div>
    </div>

    <div class="row">
      <button id="dev-connect" class="primary">Verbinden</button>
      <button id="dev-disconnect" disabled>Trennen</button>
      <select id="dev-transport" style="margin-left:12px;"
        data-help="BLE = direkt per Bluetooth. Intiface = über Buttplug-Server (auch andere Geräte). Mock = Oberfläche ohne Hardware prüfen.">
        <option value="ble">Direkt per Bluetooth</option>
        <option value="intiface">Über Intiface Central</option>
        <option value="mock">Mock-Gerät (ohne Hardware)</option>
      </select>
      <input type="text" id="dev-intiface-url" placeholder="z.B. 192.168.1.50 oder leer = dieser Rechner"
             style="display:none; width:220px;" />
    </div>
    <p class="hint" id="dev-transport-hint">Sucht bis zu 20 s nach „Sam Neo 2“.</p>

    <fieldset id="dev-test" disabled style="margin-top:16px; border:1px solid var(--border);
              border-radius:4px; padding:12px;">
      <legend style="padding:0 6px;">Funktionstest</legend>

      <div class="row" style="align-items:center;">
        <label style="width:110px;" data-help="Intern 11 Stufen (0–10). Prozent dazwischen ändert nichts.">Vibration</label>
        <input type="range" id="dev-vib" min="0" max="100" value="0" style="flex:1;">
        <span id="dev-vib-val" style="width:70px; text-align:right;">0 % (Stufe 0)</span>
      </div>

      <div class="row" style="align-items:center;">
        <label style="width:110px;" data-help="Intern 6 Stufen (0–5). Prozent dazwischen ändert nichts.">Sog</label>
        <input type="range" id="dev-suc" min="0" max="100" value="0" style="flex:1;">
        <span id="dev-suc-val" style="width:70px; text-align:right;">0 % (Stufe 0)</span>
      </div>

      <div class="row" style="margin-top:10px;">
        <button id="dev-pulse" data-help="Zwei Sekunden mittlere Vibration, dann aus.">Kurzer Testimpuls</button>
        <button id="dev-stop" class="danger">Alles aus</button>
      </div>
    </fieldset>

    <fieldset id="dev-raw" disabled style="margin-top:16px; border:1px solid var(--border);
              border-radius:4px; padding:12px;">
      <legend style="padding:0 6px;" data-help="Sendet den Stufenwert ohne Umrechnung. 0–10 / 0–5 stammen aus Buttplug — ob die Firmware mehr annimmt, ist unklar. Probiere 3 vs 4, dann 20/50/100.">Rohwert-Test</legend>
      <div class="row" style="align-items:center;">
        <select id="dev-raw-channel">
          <option value="vibration">Vibration</option>
          <option value="suction">Sog</option>
        </select>
        <input type="number" id="dev-raw-value" min="0" max="255" value="0" style="width:90px;" />
        <button id="dev-raw-send">Senden</button>
        <span id="dev-raw-hint" class="hint"></span>
      </div>
    </fieldset>

    <fieldset id="dev-diag" disabled style="margin-top:16px; border:1px solid var(--border);
              border-radius:4px; padding:12px;">
      <legend style="padding:0 6px;" data-help="Automatische Testreihe (Rohwert-Annahme, Update-Rate, Kanalinteraktion). Misst Schreib-Latenz und angenommene Werte — nicht gefühlte Intensität. Gerät bewegt sich dabei mehrfach kurz.">Geräte-Diagnose</legend>
      <div class="row"><button id="diag-run">Diagnose starten</button></div>
      <div id="diag-status" class="path-label"></div>
      <div id="diag-result" style="margin-top:8px;"></div>
      <h4 style="margin:14px 0 4px;">Verlauf</h4>
      <div id="diag-history" class="hint">Lädt...</div>
    </fieldset>

    <div id="dev-log" class="hint" style="margin-top:12px; white-space:pre-wrap;"></div>
  `;

  wireDataHelp(root);

  const el = id => root.querySelector(id);
  let busy = false;
  let searching = false;

  function log(msg) {
    el('#dev-log').textContent = new Date().toLocaleTimeString() + '  ' + msg;
  }

  function setFills(vibPct, sucPct) {
    el('#dev-fill-vib').style.height = Math.max(0, Math.min(100, vibPct)) + '%';
    el('#dev-fill-suc').style.height = Math.max(0, Math.min(100, sucPct * 0.85)) + '%';
  }

  // Das Gerät kennt intern feste Stufen (Vibration 0-10, Sog 0-5). Die
  // Prozentanzeige allein wäre irreführend, weil sich zwischen zwei Stufen
  // nichts ändert - deshalb wird die tatsächliche Stufe mit angezeigt.
  const vibStep = pct => Math.round(pct / 100 * 10);
  const sucStep = pct => Math.round(pct / 100 * 5);

  function render(st) {
    const dot = el('#dev-dot');
    const text = el('#dev-status-text');
    const sub = el('#dev-status-sub');
    const box = el('#dev-status');
    box.classList.remove('is-connected', 'is-searching', 'is-session');

    if (st.sessionActive) {
      box.classList.add('is-session');
      dot.style.background = 'var(--warn, #d9a441)';
      text.textContent = 'Wiedergabe/Training aktiv';
      sub.textContent = 'Gerätetest währenddessen nicht möglich.';
      box.style.borderColor = 'var(--warn, #d9a441)';
    } else if (searching) {
      box.classList.add('is-searching');
      dot.style.background = 'var(--warn, #d9a441)';
      text.textContent = 'Suche Gerät…';
      sub.textContent = 'Bis zu 20 Sekunden.';
      box.style.borderColor = 'var(--warn, #d9a441)';
    } else if (st.connected) {
      box.classList.add('is-connected');
      dot.style.background = 'var(--ok)';
      box.style.borderColor = 'var(--ok)';
      text.textContent = st.mock ? 'Mock verbunden' : 'Verbunden';
      const parts = [];
      if (st.name) parts.push(st.name);
      if (st.address) parts.push(st.address);
      if (st.rssi) parts.push(`Signal ${st.rssi} dBm`);
      sub.textContent = parts.length ? parts.join(' · ') : (st.mock ? 'Simuliertes Gerät' : 'Sam Neo 2');
    } else {
      dot.style.background = '#777';
      box.style.borderColor = 'var(--border)';
      text.textContent = 'Nicht verbunden';
      sub.textContent = 'Verbindung wählen und Verbinden tippen.';
      setFills(0, 0);
    }

    const canTest = st.connected && !st.sessionActive;
    el('#dev-test').disabled = !canTest;
    el('#dev-raw').disabled = !canTest;
    el('#dev-diag').disabled = !canTest;
    el('#dev-connect').disabled = st.connected || st.sessionActive || busy;
    el('#dev-disconnect').disabled = !st.connected || busy;
    el('#dev-transport').disabled = st.connected || busy;
    el('#dev-intiface-url').disabled = st.connected || busy;
  }

  async function refresh() {
    try {
      const st = await GetDeviceStatus();
      render(st);
      // sidebar.js zeigt denselben Status (LED/Name), fragt ihn aber nicht
      // mehr selbst beim Backend ab, um GetDeviceStatus() nicht doppelt so
      // oft wie nötig aufzurufen - dieser Tab bleibt wie der Wiedergabe-Tab
      // dauerhaft im DOM (nur display:none), sein Poll-Intervall reicht.
      window.dispatchEvent(new CustomEvent('device:status', { detail: st }));
    } catch (e) {
      log('Status nicht abrufbar: ' + e);
    }
  }

  // Erklärung und Adressfeld an die gewählte Verbindungsart anpassen.
  el('#dev-transport').addEventListener('change', e => {
    const intiface = e.target.value === 'intiface';
    el('#dev-intiface-url').style.display = intiface ? 'inline-block' : 'none';
    el('#dev-transport-hint').innerHTML = intiface
      ? 'Verbindet über einen laufenden Buttplug-Server. Dafür Intiface Central starten, '
      + 'dort das Gerät verbinden und den Server starten. Vorteil: kein eigener '
      + 'Bluetooth-Adapter nötig, und es funktioniert mit jedem von Buttplug unterstützten '
      + 'Gerät. Adresse leer lassen für den Standard ws://127.0.0.1:12345.'
      : e.target.value === 'mock'
        ? 'Simuliert ein Gerät, um die Oberfläche ohne Hardware zu prüfen.'
        : 'Sucht bis zu 20 Sekunden nach einem Gerät mit dem Namen "Sam Neo 2". Das Gerät '
        + 'muss eingeschaltet und nicht mit einer anderen App verbunden sein.';
  });

  el('#dev-connect').addEventListener('click', async () => {
    busy = true;
    searching = true;
    el('#dev-connect').disabled = true;
    render({ connected: false, sessionActive: false });
    try {
      const st = await ConnectDeviceVia(el('#dev-transport').value,
                                        el('#dev-intiface-url').value.trim());
      searching = false;
      busy = false;
      render(st);
      log('Verbunden.');
    } catch (e) {
      searching = false;
      busy = false;
      await refresh();
      log('Verbindung fehlgeschlagen: ' + e);
    }
  });

  el('#dev-disconnect').addEventListener('click', async () => {
    busy = true;
    try {
      const st = await DisconnectDevice();
      busy = false;
      el('#dev-vib').value = 0;
      el('#dev-suc').value = 0;
      el('#dev-vib-val').textContent = '0 % (Stufe 0)';
      el('#dev-suc-val').textContent = '0 % (Stufe 0)';
      setFills(0, 0);
      render(st);
      log('Getrennt.');
    } catch (e) {
      busy = false;
      await refresh();
      log('Trennen mit Fehler: ' + e);
    }
  });

  el('#dev-vib').addEventListener('input', async e => {
    const pct = Number(e.target.value);
    el('#dev-vib-val').textContent = `${pct} % (Stufe ${vibStep(pct)})`;
    setFills(pct, Number(el('#dev-suc').value));
    try {
      await TestVibration(pct / 100);
    } catch (err) {
      log('Vibration: ' + err);
    }
  });

  el('#dev-suc').addEventListener('input', async e => {
    const pct = Number(e.target.value);
    el('#dev-suc-val').textContent = `${pct} % (Stufe ${sucStep(pct)})`;
    setFills(Number(el('#dev-vib').value), pct);
    try {
      await TestSuction(pct / 100);
    } catch (err) {
      log('Sog: ' + err);
    }
  });

  el('#dev-pulse').addEventListener('click', async () => {
    try {
      log('Testimpuls läuft...');
      await TestVibration(0.5);
      setTimeout(async () => {
        try {
          await TestStop();
          el('#dev-vib').value = 0;
          el('#dev-vib-val').textContent = '0 % (Stufe 0)';
          log('Testimpuls beendet.');
        } catch (e) {
          log('Abschalten nach Impuls: ' + e);
        }
      }, 2000);
    } catch (e) {
      log('Testimpuls: ' + e);
    }
  });

  el('#dev-raw-send').addEventListener('click', async () => {
    const channel = el('#dev-raw-channel').value;
    const value = Number(el('#dev-raw-value').value);
    const expected = channel === 'vibration' ? 10 : 5;
    el('#dev-raw-hint').textContent = value > expected
      ? `über dem dokumentierten Maximum (${expected})`
      : '';
    try {
      await TestRawValue(channel, value);
      log(`Rohwert ${value} an ${channel} gesendet.`);
    } catch (err) {
      log('Rohwert: ' + err);
    }
  });

  el('#dev-stop').addEventListener('click', async () => {
    try {
      await TestStop();
      el('#dev-vib').value = 0;
      el('#dev-suc').value = 0;
      el('#dev-vib-val').textContent = '0 % (Stufe 0)';
      el('#dev-suc-val').textContent = '0 % (Stufe 0)';
      log('Alles aus.');
    } catch (e) {
      log('Stop: ' + e);
    }
  });

  // Geräte-Diagnose: siehe app_diagnostics.go/device/diagnostics.go.
  function phaseLabel(phase) {
    return phase.replace(/_/g, ' ');
  }

  function renderDiagReport(report) {
    if (!report) return '';
    const rows = (report.phases || []).map(p => `
      <tr><td>${phaseLabel(p.phase)}</td><td>${p.commands}</td>
        <td>${p.errors > 0 ? `<span style="color:var(--danger)">${p.errors}</span>` : '0'}</td>
        <td>${p.meanLatencyMs.toFixed(1)} ms</td><td>${p.maxLatencyMs.toFixed(1)} ms</td></tr>
    `).join('');
    const interrupted = report.interrupted
      ? '<p style="color:var(--warn, #d9a441)">Lauf wurde abgebrochen, unvollständig.</p>' : '';
    const notes = (report.notes || []).map(n => `<p class="hint">${n}</p>`).join('');
    return `
      ${interrupted}
      <table class="bench-table">
        <thead><tr><th>Phase</th><th>Kommandos</th><th>Fehler</th><th>Ø Latenz</th><th>Max Latenz</th></tr></thead>
        <tbody>${rows}</tbody>
      </table>
      ${notes}
    `;
  }

  function renderDiagHistoryRow(entry) {
    const r = entry.report || {};
    const errors = (r.phases || []).reduce((sum, p) => sum + p.errors, 0);
    const commands = (r.phases || []).reduce((sum, p) => sum + p.commands, 0);
    const when = new Date(entry.timestamp);
    const whenText = isNaN(when.getTime()) ? entry.timestamp : when.toLocaleString();
    return `<div>${whenText}${entry.mock ? ' (Mock)' : entry.deviceName ? ` (${entry.deviceName})` : ''} — `
      + `${commands} Kommandos, ${errors} Fehler${r.interrupted ? ', abgebrochen' : ''}</div>`;
  }

  async function refreshDiagHistory() {
    const box = el('#diag-history');
    try {
      const history = await GetDiagnosticsHistory();
      if (!Array.isArray(history) || history.length === 0) {
        box.textContent = 'Noch kein Diagnoselauf aufgezeichnet.';
        return;
      }
      box.innerHTML = history.map(renderDiagHistoryRow).join('');
    } catch (err) {
      box.textContent = 'Verlauf konnte nicht geladen werden: ' + err;
    }
  }

  el('#diag-run').addEventListener('click', async () => {
    el('#diag-run').disabled = true;
    el('#diag-status').textContent = 'Läuft… (Vibration/Sog bewegen sich kurz mehrfach)';
    el('#diag-result').innerHTML = '';
    try {
      await RunDeviceDiagnostics();
    } catch (err) {
      el('#diag-run').disabled = false;
      el('#diag-status').textContent = 'Fehlgeschlagen: ' + err;
    }
  });

  EventsOn('diagnostics:entry', e => {
    const value = typeof e.sentRaw === 'number' ? e.sentRaw : e.wantedValue;
    el('#diag-status').textContent =
      `Läuft… ${phaseLabel(e.phase)}${e.channel ? ' · ' + e.channel : ''}${value !== undefined ? ' · ' + value : ''}`
      + (e.error ? ` · Fehler: ${e.error}` : '');
  });
  EventsOn('diagnostics:done', entry => {
    el('#diag-run').disabled = false;
    el('#diag-status').textContent = 'Fertig.';
    el('#diag-result').innerHTML = renderDiagReport(entry.report);
    refreshDiagHistory();
  });

  refreshDiagHistory();

  // Zuletzt benutzte Verbindungsart und Adresse wiederherstellen.
  GetSettings().then(s => {
    if (s.deviceTransport) {
      el('#dev-transport').value = s.deviceTransport;
      el('#dev-transport').dispatchEvent(new Event('change'));
    }
    if (s.intifaceUrl) el('#dev-intiface-url').value = s.intifaceUrl;
  }).catch(() => {});

  refresh();
  // Der Zustand kann sich außerhalb dieses Tabs ändern (eine Wiedergabe
  // startet oder endet), deshalb regelmäßig nachfragen statt nur beim Laden.
  setInterval(refresh, 2000);

  return { refresh };
}
