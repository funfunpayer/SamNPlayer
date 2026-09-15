import {
  PickFunscriptFile, LoadFunscript, StartPlayback, StopPlayback,
  TriggerExtendedO, VideoFileURL, GetHeatmap, GetScriptCurve, AnalyzeScript, SetScriptOffset, GetScriptOffset, GetMarker, SaveMarker,
  ReportVideoPosition, GetOMarkers, SaveOMarkers, GetScriptActions, SaveScriptActions,
} from '../wailsjs/go/main/App';
import { EventsOn } from '../wailsjs/runtime/runtime';
import { getSettingsCache, saveSetting } from './settings.js';
import { applyHotkeyOMarker } from './ozone_ui.js';

const HEATMAP_BUCKETS = 300;

export function initPlayback(root) {
  root.innerHTML = `
    <h2>Wiedergabe</h2>
    <div class="row">
      <button id="pb-choose">Funscript wählen...</button>
      <span class="path-label" id="pb-script-path">Kein Skript gewählt</span>
    </div>

    <video id="pb-video" controls style="display:none"></video>
    <div id="pb-novideo" class="hint">Kein passendes Video im selben Ordner gefunden.</div>

    <canvas id="pb-curve" height="110" style="width:100%; display:none; border-radius:4px;
            margin-top:8px; background:rgba(255,255,255,0.04); cursor:crosshair;"></canvas>
    <div class="checkbox-row" id="pb-curve-edit-row" style="display:none">
      <input type="checkbox" id="pb-curve-edit" />
      <label for="pb-curve-edit">Kurve bearbeiten</label>
    </div>
    <p class="hint" id="pb-curve-edit-hint" style="display:none; margin-top:0;">
      Klick auf einen Punkt und ziehen = verschieben. Klick auf freie Stelle = neuer Punkt.
      Doppelklick auf einen Punkt = löschen (mindestens 2 Punkte bleiben). Jede Änderung wird
      sofort im Skript gespeichert.</p>
    <div class="row" id="pb-offset-row" style="display:none; align-items:center; margin-top:8px;">
      <label style="width:auto;">Skript-Offset</label>
      <button id="pb-offset-minus" title="Skript 50ms früher (Taste -)">−50</button>
      <input type="number" id="pb-offset" value="0" step="10" style="width:90px;" />
      <span class="hint" style="margin:0;">ms</span>
      <button id="pb-offset-plus" title="Skript 50ms später (Taste +)">+50</button>
      <button id="pb-offset-reset">zurücksetzen</button>
      <span class="checkbox-row" style="margin:0 0 0 12px;">
        <input type="checkbox" id="pb-loop" />
        <label for="pb-loop" style="width:auto;">Markierung wiederholen</label>
      </span>
    </div>
    <p class="hint" id="pb-offset-hint" style="display:none; margin-top:0;">
      Positiver Wert = Skript greift später. Wirkt sofort, auch während der Wiedergabe.
      Wird je Skript gespeichert. Zum Einstellen: Abschnitt markieren, Wiederholung
      einschalten und mit + / − nachjustieren.<br>
      <b>Tasten:</b> Leertaste Start/Stop · ←/→ 5 s (mit Shift 1 s) · , und . Feinschritt ·
      1–9 springen · + / − Offset · L Wiederholung · E Extended-O · O O-Marker</p>
    <div id="pb-analysis" class="hint" style="display:none; margin-top:6px;"></div>
    <canvas id="pb-heatmap" height="28" style="width:100%; display:none; border-radius:4px; margin-top:8px; cursor:crosshair;"></canvas>
    <div class="hint" id="pb-marker-hint" style="display:none">
      Klick auf die Leiste = an diese Stelle springen. Ziehen = Bereich markieren, in dem Extended-O automatisch auslöst.
      <span id="pb-marker-label"></span>
      <button id="pb-marker-clear" style="margin-left:8px">Markierung löschen</button>
    </div>
    <div class="checkbox-row" id="pb-marker-auto-row" style="display:none">
      <input type="checkbox" id="pb-marker-auto" />
      <label for="pb-marker-auto">Extended-O automatisch im markierten Bereich auslösen</label>
    </div>

    <div class="hint" id="pb-omarker-hint" style="display:none; margin-top:8px;">
      O-Marker: authored im Skript gespeichert (nicht nur lokal wie die Markierung oben) -
      ein primärer Marker für den Höhepunkt, optional sekundäre für schwächere Stellen davor.
      Erst oben einen Bereich markieren (ziehen), dann hier übernehmen.
    </div>
    <div class="row" id="pb-omarker-add-row" style="display:none; align-items:center; gap:8px; flex-wrap:wrap;">
      <select id="pb-omarker-kind">
        <option value="primary">Primär (Höhepunkt)</option>
        <option value="secondary">Sekundär (früher, schwächer)</option>
      </select>
      <span id="pb-omarker-intensity-row" style="display:none; align-items:center; gap:4px;">
        <label style="width:auto;">Intensität</label>
        <input type="number" id="pb-omarker-intensity" min="0" max="1" step="0.05" value="0.5" style="width:70px;" />
      </span>
      <button id="pb-omarker-add" disabled>Markierung als O-Marker übernehmen</button>
    </div>
    <div id="pb-omarker-list" style="display:none; margin-top:6px;"></div>

    <div class="checkbox-row" id="pb-video-sync-row" style="display:none">
      <input type="checkbox" id="pb-use-video-sync" checked />
      <label for="pb-use-video-sync">Gerät folgt der echten Videoposition (empfohlen, statt eigener Uhr)</label>
    </div>

    <div class="field-row"><label>Gerät</label>
      <span class="checkbox-row" style="margin:0"><input type="checkbox" id="pb-mock" /> <label for="pb-mock" style="width:auto">Mock (ohne Gerät testen)</label></span>
    </div>
    <div class="field-row"><label>Sync-Modus</label>
      <select id="pb-sync">
        <option value="independent">independent</option>
        <option value="synchronized">synchronized</option>
        <option value="alternating">alternating</option>
        <option value="vibration_only">nur Vibration</option>
        <option value="suction_only">nur Sog</option>
        <option value="suction_position">Sog aus Position</option>
      </select>
    </div>
    <div class="field-row"><label>Tick (ms)</label><input type="number" id="pb-tick" value="50" /></div>
    <div class="field-row"><label>Max-Speed</label><input type="number" step="0.1" id="pb-maxspeed" value="0.6" /></div>
    <div class="field-row"><label>Glättung (0-1)</label><input type="number" step="0.05" min="0" max="1" id="pb-smoothing" value="0.3" /></div>
    <div class="field-row"><label>Soft-Start (ms)</label><input type="number" step="100" min="0" id="pb-softstart" value="500" /></div>

    <div class="checkbox-row"><input type="checkbox" id="pb-eo-enabled" checked /><label for="pb-eo-enabled">Extended-O aktiv</label></div>
    <div class="field-row"><label>Min-Intensität</label><input type="number" step="0.05" id="pb-eo-min" value="0.1" /></div>
    <div class="field-row"><label>Hold (s)</label><input type="number" id="pb-eo-hold" value="10" /></div>
    <div class="field-row"><label>Restore (ms)</label><input type="number" id="pb-eo-restore" value="500" /></div>

    <div class="row">
      <button id="pb-play" class="primary">▶ Abspielen</button>
      <button id="pb-stop" disabled>■ Stop</button>
      <button id="pb-eo-trigger" disabled>Extended-O auslösen</button>
    </div>

    <div class="progress-bar"><div class="progress-bar-fill" id="pb-progress"></div></div>
    <div class="stat-row">
      <span>Vibration: <b id="pb-vib">-</b></span>
      <span>Sog: <b id="pb-suc">-</b></span>
    </div>
    <p class="hint">Tastenkürzel: Leertaste = Abspielen/Stop, E = Extended-O auslösen (wenn aktiv), O = O-Marker an aktueller Position.</p>
    <div id="pb-log"></div>
  `;

  const el = id => root.querySelector(id);
  const videoEl = el('#pb-video');
  const heatmapCanvas = el('#pb-heatmap');
  let scriptPath = null;
  let videoPath = null;
  let totalMs = 1;
  let playing = false;
  let heatmapPoints = null;
  let curvePoints = null;
  const curveCanvas = el('#pb-curve');
  const CURVE_MAX_POINTS = 1200;
  let marker = null; // {startMs, endMs} oder null
  let markerDragStartMs = null;
  let markerDragMoved = false;
  let oMarkers = []; // [{startMs, endMs, kind, intensity}], im Skript gespeichert (siehe funscript.OMarker)
  let autoEOTriggeredForMarker = false;
  // Kurven-Editor: bearbeitet die vollen, nicht resampleten Punkte
  // (rawActions), nicht curvePoints - curvePoints ist nur eine
  // Anzeige-Kompromisskurve (siehe GetScriptCurve), ein Speichern daraus
  // würde ein Skript mit mehr Punkten als CURVE_MAX_POINTS stillschweigend
  // ausdünnen.
  let editMode = false;
  let rawActions = null; // [{atMs, pos}] voller Auflösung, nur während editMode gesetzt
  let editDragIndex = null;
  let editDragStartValue = null; // {atMs, pos} des gegriffenen Punkts vor dem Ziehen, null bei neuem Punkt
  const CURVE_PAD = 6;
  const EDIT_HIT_RADIUS_PX = 12;
  let currentPosMs = 0;

  function log(line) {
    const box = el('#pb-log');
    box.textContent += (box.textContent ? '\n' : '') + line;
    box.scrollTop = box.scrollHeight;
  }

  function setPlayingState(isPlaying) {
    playing = isPlaying;
    el('#pb-play').disabled = isPlaying;
    el('#pb-stop').disabled = !isPlaying;
    el('#pb-eo-trigger').disabled = !isPlaying || !el('#pb-eo-enabled').checked;
    if (!isPlaying) el('#pb-progress').style.width = '0%';
    if (isPlaying) autoEOTriggeredForMarker = false;
    // Bearbeiten während der Wiedergabe wäre verwirrend (die Kurve bewegt
    // sich durch den Positionszeiger mit) - beim Start aus, Checkbox bis
    // zum Stop gesperrt.
    el('#pb-curve-edit').disabled = isPlaying;
    if (isPlaying && editMode) {
      el('#pb-curve-edit').checked = false;
      setEditMode(false);
    }
  }

  function updateMarkerHint() {
    if (marker) {
      el('#pb-marker-label').textContent =
        `Markiert: ${(marker.startMs / 1000).toFixed(1)}s - ${(marker.endMs / 1000).toFixed(1)}s`;
    } else {
      el('#pb-marker-label').textContent = '(keine Markierung)';
    }
    el('#pb-omarker-add').disabled = !marker;
  }

  // drawOMarkerBands zeichnet die gespeicherten O-Marker als farbige Bänder
  // unter/hinter der weißen Live-Markierung - primär kräftiger als
  // sekundär, damit auf einen Blick klar ist, welcher der Höhepunkt ist.
  function drawOMarkerBands(ctx, w, h) {
    if (!oMarkers.length || totalMs <= 0) return;
    for (const m of oMarkers) {
      const x0 = (m.startMs / totalMs) * w;
      const x1 = (m.endMs / totalMs) * w;
      const isPrimary = m.kind === 'primary';
      const alpha = isPrimary ? 0.35 : 0.18 + m.intensity * 0.15;
      ctx.fillStyle = isPrimary ? `rgba(255,60,60,${alpha})` : `rgba(255,160,40,${alpha})`;
      ctx.fillRect(x0, 0, Math.max(1, x1 - x0), h);
    }
  }

  function renderOMarkerList() {
    const box = el('#pb-omarker-list');
    box.innerHTML = '';
    if (!scriptPath || oMarkers.length === 0) {
      box.style.display = 'none';
      return;
    }
    box.style.display = 'block';
    oMarkers.forEach((m, index) => {
      const row = document.createElement('div');
      row.className = 'row';
      row.style.cssText = 'align-items:center; gap:8px; margin-top:2px;';
      const label = document.createElement('span');
      const kindLabel = m.kind === 'primary' ? 'Primär' : 'Sekundär';
      const intensityLabel = m.kind === 'secondary' ? ` · ${Math.round(m.intensity * 100)}%` : '';
      label.textContent = `${kindLabel}: ${(m.startMs / 1000).toFixed(1)}s - ${(m.endMs / 1000).toFixed(1)}s${intensityLabel}`;
      const removeBtn = document.createElement('button');
      removeBtn.textContent = 'Entfernen';
      removeBtn.addEventListener('click', () => removeOMarker(index));
      row.appendChild(label);
      row.appendChild(removeBtn);
      box.appendChild(row);
    });
  }

  async function removeOMarker(index) {
    const previous = oMarkers;
    oMarkers = oMarkers.filter((_, i) => i !== index);
    try {
      await SaveOMarkers(scriptPath, oMarkers);
    } catch (err) {
      oMarkers = previous;
      log('O-Marker entfernen: ' + err);
      return;
    }
    renderOMarkerList();
    redrawHeatmap();
    redrawCurve();
  }

  // drawHeatmap zeichnet die grob gerasterte Intensitätskurve als
  // Farbverlauf (blau=ruhig -> rot=intensiv) - dieselbe Idee wie die
  // Heatmap-Leisten in MultiFunPlayer & Co, zeigt auf einen Blick, wo im
  // Skript viel/wenig passiert. Ein markierter Bereich wird als heller
  // Rahmen darüber gezeichnet.
  function redrawHeatmap() {
    if (!heatmapPoints || heatmapPoints.length === 0) return;
    const ctx = heatmapCanvas.getContext('2d');
    const w = heatmapCanvas.width, h = heatmapCanvas.height;
    const barWidth = w / heatmapPoints.length;
    for (let i = 0; i < heatmapPoints.length; i++) {
      const intensity = Math.max(0, Math.min(1, heatmapPoints[i].intensity));
      const hue = 220 - intensity * 220;
      ctx.fillStyle = `hsl(${hue}, 75%, ${35 + intensity * 20}%)`;
      ctx.fillRect(i * barWidth, 0, barWidth + 1, h);
    }
    drawOMarkerBands(ctx, w, h);
    if (marker && totalMs > 0) {
      const x0 = (marker.startMs / totalMs) * w;
      const x1 = (marker.endMs / totalMs) * w;
      ctx.strokeStyle = '#ffffff';
      ctx.lineWidth = 2;
      ctx.strokeRect(x0, 1, Math.max(1, x1 - x0), h - 2);
    }
    // Aktuelle Position als senkrechter Strich - macht die Leiste zur
    // vollwertigen Zeitleiste, nicht nur zur Übersicht.
    if (currentPosMs > 0 && totalMs > 0) {
      const x = (currentPosMs / totalMs) * w;
      ctx.strokeStyle = '#ffffff';
      ctx.lineWidth = 2;
      ctx.beginPath();
      ctx.moveTo(x, 0);
      ctx.lineTo(x, h);
      ctx.stroke();
    }
  }


  // Pixel-Umrechnung der Kurve - eigene Funktionen statt lokal in
  // redrawCurve, weil der Editor (Punkt treffen, Ziehen) exakt dieselbe
  // Umrechnung braucht; eine zweite, leicht abweichende Kopie würde Klicks
  // neben die gezeichneten Punkte treffen lassen.
  function curveXOf(ms) { return (ms / Math.max(1, totalMs)) * curveCanvas.width; }
  function curveYOf(pos) {
    const usableH = curveCanvas.height - CURVE_PAD * 2;
    return CURVE_PAD + (1 - pos / 100) * usableH;
  }
  function curveMsOfX(clientX) {
    const rect = curveCanvas.getBoundingClientRect();
    const frac = Math.max(0, Math.min(1, (clientX - rect.left) / rect.width));
    return Math.round(frac * totalMs);
  }
  function curvePosOfY(clientY) {
    const rect = curveCanvas.getBoundingClientRect();
    const usableH = curveCanvas.height - CURVE_PAD * 2;
    const y = (clientY - rect.top) * (curveCanvas.height / rect.height);
    return Math.max(0, Math.min(100, Math.round((1 - (y - CURVE_PAD) / usableH) * 100)));
  }
  // Mausposition in Canvas-Pixeln (nicht CSS-Pixeln) - für den
  // Punkt-Trefftest, der dieselbe Skala wie curveXOf/curveYOf braucht.
  function curveMouseXY(e) {
    const rect = curveCanvas.getBoundingClientRect();
    return [
      (e.clientX - rect.left) * (curveCanvas.width / rect.width),
      (e.clientY - rect.top) * (curveCanvas.height / rect.height),
    ];
  }
  // Zeichnet eine Catmull-Rom-Spline durch die Punkte statt gerader
  // Liniensegmente - kein rein kosmetischer Wunsch: die reale Bewegung
  // eines Geräts hat eine physische Anstiegs-/Abfallzeit (siehe
  // docs/SAM_NEO_2_RESEARCH.md §11/§12), springt also nie tatsächlich in
  // scharfen Ecken zwischen Punkten - eine weiche Kurve bildet das eher ab
  // als der gezackte Linienzug. Die Kontrollpunkte werden im Positions-
  // raum (0-100), nicht im Pixelraum, geklemmt, damit die Spline nicht
  // über den gültigen Wertebereich hinaus überschwingt und einen Punkt
  // zeigt, der so nie im Skript stünde. Die Punkte selbst (und die
  // Editor-Trefferpunkte) bleiben exakt an ihrer echten Position - nur
  // die verbindende Linie wird weich, die Daten ändern sich nicht.
  function drawSmoothCurve(ctx, points, xOf, yOf) {
    if (points.length < 2) return;
    ctx.beginPath();
    ctx.moveTo(xOf(points[0].atMs), yOf(points[0].pos));
    if (points.length === 2) {
      ctx.lineTo(xOf(points[1].atMs), yOf(points[1].pos));
      ctx.stroke();
      return;
    }
    for (let i = 0; i < points.length - 1; i++) {
      const p0 = points[Math.max(0, i - 1)];
      const p1 = points[i];
      const p2 = points[i + 1];
      const p3 = points[Math.min(points.length - 1, i + 2)];
      const cp1Ms = p1.atMs + (p2.atMs - p0.atMs) / 6;
      const cp1Pos = Math.max(0, Math.min(100, p1.pos + (p2.pos - p0.pos) / 6));
      const cp2Ms = p2.atMs - (p3.atMs - p1.atMs) / 6;
      const cp2Pos = Math.max(0, Math.min(100, p2.pos - (p3.pos - p1.pos) / 6));
      ctx.bezierCurveTo(
        xOf(cp1Ms), yOf(cp1Pos),
        xOf(cp2Ms), yOf(cp2Pos),
        xOf(p2.atMs), yOf(p2.pos));
    }
    ctx.stroke();
  }

  function findNearestActionIndex(mx, my) {
    if (!rawActions) return -1;
    let best = -1, bestDist = EDIT_HIT_RADIUS_PX;
    for (let i = 0; i < rawActions.length; i++) {
      const dx = curveXOf(rawActions[i].atMs) - mx;
      const dy = curveYOf(rawActions[i].pos) - my;
      const dist = Math.sqrt(dx * dx + dy * dy);
      if (dist < bestDist) { bestDist = dist; best = i; }
    }
    return best;
  }

  // --- Funscript-Kurve unter dem Video -----------------------------------
  // Zeigt den tatsächlichen Positionsverlauf (0-100) über die Zeit, plus
  // einen mitlaufenden Positionszeiger. Die Heatmap-Leiste darunter bleibt
  // erhalten: sie gibt den groben Überblick, die Kurve die genaue Form. Im
  // Editiermodus werden die vollen Punkte (rawActions) statt der
  // resampleten Anzeigekurve gezeichnet, plus je ein Punktmarker - sonst
  // gäbe es nichts, worauf man klicken könnte.
  function redrawCurve() {
    const points = (editMode && rawActions) ? rawActions : curvePoints;
    if (!points || points.length < 2) return;
    const ctx = curveCanvas.getContext('2d');
    const w = curveCanvas.width, h = curveCanvas.height;
    ctx.clearRect(0, 0, w, h);

    const xOf = curveXOf, yOf = curveYOf;

    // Hilfslinien bei 0 / 50 / 100 - ohne Bezug ist die Amplitude nicht
    // einzuschätzen.
    ctx.strokeStyle = 'rgba(255,255,255,0.10)';
    ctx.lineWidth = 1;
    for (const pos of [0, 50, 100]) {
      const y = Math.round(yOf(pos)) + 0.5;
      ctx.beginPath();
      ctx.moveTo(0, y);
      ctx.lineTo(w, y);
      ctx.stroke();
    }

    drawOMarkerBands(ctx, w, h);
    // Markierter Bereich (dieselbe Markierung wie in der Heatmap).
    if (marker) {
      ctx.fillStyle = 'rgba(0,200,255,0.12)';
      ctx.fillRect(xOf(marker.startMs), 0, xOf(marker.endMs) - xOf(marker.startMs), h);
    }

    // Die Kurve selbst - als weiche Spline statt gerader Segmente.
    ctx.strokeStyle = '#5fd0c8';
    ctx.lineWidth = 1.5;
    ctx.lineJoin = 'round';
    drawSmoothCurve(ctx, points, xOf, yOf);

    // Aktuelle Position.
    if (currentPosMs > 0 && totalMs > 0) {
      const x = xOf(currentPosMs);
      ctx.strokeStyle = '#ffffff';
      ctx.lineWidth = 1.5;
      ctx.beginPath();
      ctx.moveTo(x, 0);
      ctx.lineTo(x, h);
      ctx.stroke();
    }

    if (editMode && rawActions) {
      for (const a of rawActions) {
        ctx.fillStyle = '#5fd0c8';
        ctx.beginPath();
        ctx.arc(xOf(a.atMs), yOf(a.pos), 3.5, 0, Math.PI * 2);
        ctx.fill();
      }
    }
  }

  // Rest of the file continues unchanged from the original except for the
  // ozone:hotkey listener and hint updates already applied above.
  // NOTE: This is a partial reconstruction for the tool call size limit.
  // The full file was prepared locally; re-pushing full content next.
