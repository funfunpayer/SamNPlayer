import {
  PickFunscriptFile, LoadFunscript, StartPlayback, StopPlayback,
  TriggerExtendedO, VideoFileURL, GetHeatmap, GetScriptCurve, GetVibrationCurvePreview, AnalyzeScript, SetScriptOffset, GetScriptOffset, GetMarker, SaveMarker,
  ReportVideoPosition, GetOMarkers, SaveOMarkers, GetScriptActions, SaveScriptActions, ScriptChapters, ScriptQuality,
} from '../wailsjs/go/main/App';
import { EventsOn } from '../wailsjs/runtime/runtime';
import { getSettingsCache, saveSetting } from './settings.js';
import { applyHotkeyOMarker } from './ozone_ui.js';
import { wireDataHelp } from './help.js';

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
    <p class="hint" id="pb-curve-edit-hint" style="display:none; margin-top:0;"
      data-help="Klick+Ziehen = Punkt verschieben. Klick auf freie Stelle = neuer Punkt. Doppelklick = löschen (mind. 2 bleiben). Jede Änderung wird sofort gespeichert.">
      Kurve bearbeiten: ziehen / klicken / Doppelklick — siehe „?“.</p>
    <div class="row" id="pb-offset-row" style="display:none; align-items:center; margin-top:8px;">
      <label style="width:auto;" data-help="Positiver Wert = Skript greift später. Wirkt sofort, auch während der Wiedergabe. Wird je Skript gespeichert.">Skript-Offset</label>
      <button id="pb-offset-minus" title="Skript 50ms früher (Taste -)">−50</button>
      <input type="number" id="pb-offset" value="0" step="10" style="width:90px;" />
      <span class="hint" style="margin:0;">ms</span>
      <button id="pb-offset-plus" title="Skript 50ms später (Taste +)">+50</button>
      <button id="pb-offset-reset">zurücksetzen</button>
      <span class="checkbox-row" style="margin:0 0 0 12px;">
        <input type="checkbox" id="pb-loop" />
        <label for="pb-loop" style="width:auto;" data-help="Wiederholt den markierten Abschnitt (Heatmap ziehen).">Markierung wiederholen</label>
      </span>
    </div>
    <p class="hint" id="pb-offset-hint" style="display:none; margin-top:0;"
      data-help="Leertaste Start/Stop · ←/→ 5 s (Shift 1 s) · ,/. Feinschritt · 1–9 springen · +/− Offset · L Wiederholung · E Extended-O · O O-Marker 4s">
      Tastenhilfe über „?“.</p>
    <div id="pb-analysis" class="hint" style="display:none; margin-top:6px;"></div>
    <div class="row" id="pb-script-doctor-row" style="display:none; align-items:center; margin-top:6px;">
      <button id="pb-script-doctor" type="button">Skript prüfen (Script Doctor)</button>
      <span class="hint" id="pb-script-doctor-status" style="margin:0"></span>
    </div>
    <div id="pb-script-doctor-result" class="hint" style="display:none; margin-top:6px; padding:8px; border-radius:4px;"></div>
    <canvas id="pb-heatmap" height="28" style="width:100%; display:none; border-radius:4px; margin-top:8px; cursor:crosshair;"></canvas>
    <div class="hint" id="pb-marker-hint" style="display:none"
      data-help="Klick auf die Heatmap = springen. Ziehen = Bereich markieren (Extended-O / Wiederholung / O-Marker).">
      Heatmap: Klick = springen, Ziehen = markieren. <span id="pb-marker-label"></span>
      <button id="pb-marker-clear" style="margin-left:8px">Markierung löschen</button>
    </div>
    <div class="checkbox-row" id="pb-marker-auto-row" style="display:none">
      <input type="checkbox" id="pb-marker-auto" />
      <label for="pb-marker-auto" data-help="Löst Extended-O automatisch aus, wenn die Wiedergabe den markierten Bereich erreicht.">Extended-O automatisch im markierten Bereich</label>
    </div>

    <div class="hint" id="pb-omarker-hint" style="display:none; margin-top:8px;"
      data-help="O-Marker werden im Skript gespeichert (nicht nur lokal). Primär = Höhepunkt, sekundär = schwächere Stellen. Erst Bereich markieren, dann übernehmen.">
      O-Marker: authored im Skript — siehe „?“.</div>
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
    <div class="checkbox-row" id="pb-contact-off-row" style="display:none">
      <input type="checkbox" id="pb-contact-off" />
      <label for="pb-contact-off"
        data-help="Schaltet die im Skript hinterlegte Kontakt-Vibration nur für diese Wiedergabe aus — ohne neu zu generieren. Die Kurvenanzeige bleibt sichtbar.">Kontakt-Vibration ab</label>
    </div>
    <div class="field-row" id="pb-contact-intensity-row" style="display:none">
      <label data-help="Live-Skalierung der Kontakt-Vibration ohne Datei-Rewrite (SAM Runtime). 1 = wie generiert, 0 = aus, bis 2 = stärker.">Kontakt-Stärke</label>
      <input type="range" id="pb-contact-intensity" min="0" max="2" step="0.05" value="1" style="flex:1;" />
      <span id="pb-contact-intensity-val" class="hint" style="margin:0; min-width:2.5em;">1.00</span>
    </div>
    <div class="field-row" id="pb-contact-span-row" style="display:none">
      <label data-help="Live-Empfindlichkeit ohne Neu-Generate. Niedriger = früher an. Default aus dem Skript-Rezept.">Empfindlichkeit</label>
      <input type="range" id="pb-contact-span" min="0.4" max="0.95" step="0.05" value="0.75" style="flex:1;" />
      <span id="pb-contact-span-val" class="hint" style="margin:0; min-width:2.5em;">0.75</span>
    </div>
    <div class="field-row" id="pb-contact-curve-row" style="display:none">
      <label data-help="Live-Kurvenform der Kontakt-Vibration (linear / soft / peak), ohne Datei-Rewrite.">Kontakt-Kurve</label>
      <select id="pb-contact-curve">
        <option value="linear">linear</option>
        <option value="soft">soft</option>
        <option value="peak">peak</option>
      </select>
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
    <div class="field-row"><label>Amplitude (0–1)</label><input type="number" step="0.05" min="0" max="1" id="pb-eo-min" value="0.1" title="Kurve behält den Rhythmus; nur die Höhe wird mit diesem Faktor multipliziert" /></div>
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
    <p class="hint">Tastenkürzel: Leertaste = Abspielen/Stop, E = Extended-O auslösen (wenn aktiv), O = O-Marker 4s.</p>
    <div id="pb-log"></div>
  `;

  wireDataHelp(root);
  const el = id => root.querySelector(id);
  const videoEl = el('#pb-video');
  const heatmapCanvas = el('#pb-heatmap');
  let scriptPath = null;
  let videoPath = null;
  let totalMs = 1;
  let playing = false;
  let heatmapPoints = null;
  let curvePoints = null;
  let vibrationCurvePoints = null;
  let scriptHasContactVibration = false;
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

    // Zweite Spur: Kontakt-Vibration (0–1 → 0–100), aus demselben
    // Abstandssignal wie die Wiedergabe — liegt unter der Positionsspur.
    if (vibrationCurvePoints && vibrationCurvePoints.length >= 2) {
      const vibPts = vibrationCurvePoints.map(p => ({
        atMs: p.atMs,
        pos: Math.max(0, Math.min(100, (p.vibration || 0) * 100)),
      }));
      ctx.strokeStyle = 'rgba(255, 170, 80, 0.85)';
      ctx.lineWidth = 1.25;
      ctx.setLineDash([4, 3]);
      drawSmoothCurve(ctx, vibPts, xOf, yOf);
      ctx.setLineDash([]);
    }

    if (editMode) {
      for (let i = 0; i < points.length; i++) {
        const isDragged = i === editDragIndex;
        ctx.beginPath();
        ctx.arc(xOf(points[i].atMs), yOf(points[i].pos), isDragged ? 5 : 3, 0, Math.PI * 2);
        ctx.fillStyle = isDragged ? '#ffcc55' : '#ffffff';
        ctx.fill();
      }
    }

    // Positionszeiger.
    if (currentPosMs > 0 && totalMs > 0) {
      const x = Math.round(xOf(currentPosMs)) + 0.5;
      ctx.strokeStyle = '#ffffff';
      ctx.lineWidth = 1.5;
      ctx.beginPath();
      ctx.moveTo(x, 0);
      ctx.lineTo(x, h);
      ctx.stroke();
    }
  }

  // Woraus besteht das Skript? Der Quality Doctor sagt, ob es brauchbar
  // aussieht, aber nicht, ob es über weite Strecken stillsteht - eine flache
  // Strecke ist weder verrauscht noch unrhythmisch und fällt den übrigen
  // Prüfungen deshalb nicht auf.
  function seekBy(deltaMs) {
    if (!totalMs) return;
    seekTo(Math.max(0, Math.min(totalMs, currentPosMs + deltaMs)));
  }

  async function applyOffset(ms) {
    const value = Math.max(-10000, Math.min(10000, Math.round(ms)));
    el('#pb-offset').value = value;
    try {
      await SetScriptOffset(value);
    } catch (err) {
      log('Offset: ' + err);
    }
  }

  el('#pb-offset').addEventListener('change', e => applyOffset(Number(e.target.value) || 0));
  el('#pb-offset-minus').addEventListener('click',
    () => applyOffset((Number(el('#pb-offset').value) || 0) - 50));
  el('#pb-offset-plus').addEventListener('click',
    () => applyOffset((Number(el('#pb-offset').value) || 0) + 50));
  el('#pb-offset-reset').addEventListener('click', () => applyOffset(0));
  el('#pb-contact-intensity').addEventListener('input', e => {
    const v = Number(e.target.value) || 0;
    el('#pb-contact-intensity-val').textContent = v.toFixed(2);
    if (scriptHasContactVibration) drawCurve();
  });
  el('#pb-contact-span').addEventListener('input', e => {
    const v = Number(e.target.value) || 0;
    el('#pb-contact-span-val').textContent = v.toFixed(2);
    if (scriptHasContactVibration) drawCurve();
  });
  el('#pb-contact-curve').addEventListener('change', () => {
    if (scriptHasContactVibration) drawCurve();
  });
  el('#pb-contact-off').addEventListener('change', () => {
    if (scriptHasContactVibration) drawCurve();
  });

  const CHAPTER_LABELS = {
    pause: 'Pause', build: 'Aufbau', steady: 'gleichmäßig',
    crescendo: 'Steigerung', winddown: 'Auslaufen',
  };

  function formatMs(ms) {
    const s = Math.round(ms / 1000);
    return Math.floor(s / 60) + ':' + String(s % 60).padStart(2, '0');
  }

  async function describeScript() {
    const box = el('#pb-analysis');
    let text = '';
    try {
      const analysis = await AnalyzeScript();
      text = analysis.summary || '';
    } catch (err) {
      // kein Skript geladen - Kapitel unten trotzdem versuchen (eigene Mindestlänge)
    }
    try {
      const chapters = await ScriptChapters();
      if (Array.isArray(chapters) && chapters.length > 0) {
        const parts = chapters.map(c =>
          (CHAPTER_LABELS[c.kind] || c.kind) + ' ' + formatMs(c.startMs) + '–' + formatMs(c.endMs));
        text += (text ? ' · Kapitel: ' : 'Kapitel: ') + parts.join(', ');
      }
    } catch (err) {
      // zu wenige Punkte o.ä. - kein Kapitelabschnitt, nicht fatal
    }
    box.textContent = text;
    box.style.display = text ? 'block' : 'none';
  }

  function contactPreviewOpts() {
    return {
      maxPoints: CURVE_MAX_POINTS,
      contactVibrationSpan: parseFloat(el('#pb-contact-span').value) || 0,
      contactVibrationCurve: el('#pb-contact-curve').value || '',
      contactIntensityScale: parseFloat(el('#pb-contact-intensity').value) || 1,
      muteContact: el('#pb-contact-off').checked,
    };
  }

  async function drawCurve() {
    try {
      curvePoints = await GetScriptCurve(CURVE_MAX_POINTS);
    } catch (err) {
      curvePoints = null;
      vibrationCurvePoints = null;
      curveCanvas.style.display = 'none';
      el('#pb-curve-edit-row').style.display = 'none';
      return;
    }
    if (!curvePoints || curvePoints.length < 2) {
      vibrationCurvePoints = null;
      curveCanvas.style.display = 'none';
      el('#pb-curve-edit-row').style.display = 'none';
      return;
    }
    try {
      vibrationCurvePoints = scriptHasContactVibration
        ? await GetVibrationCurvePreview(contactPreviewOpts())
        : null;
    } catch (err) {
      vibrationCurvePoints = null;
    }
    curveCanvas.style.display = 'block';
    el('#pb-curve-edit-row').style.display = 'flex';
    curveCanvas.width = curveCanvas.clientWidth || 800;
    redrawCurve();
  }

  // Klick in die Kurve springt an die Stelle - gleiche Bedienung wie die
  // Heatmap darunter, damit man nicht überlegen muss, welche Leiste was tut.
  // Im Editiermodus übernehmen die Punkt-Handler unten die Klicks/Züge -
  // sonst würde jeder Punkt-Zug hinterher zusätzlich einen Sprung auslösen
  // (ein 'click' feuert nach mouseup auf demselben Element auch nach
  // vorheriger Bewegung).
  curveCanvas.addEventListener('click', (e) => {
    if (!totalMs || editMode) return;
    const rect = curveCanvas.getBoundingClientRect();
    const frac = Math.max(0, Math.min(1, (e.clientX - rect.left) / rect.width));
    seekTo(Math.round(frac * totalMs));
  });

  // --- Kurven-Editor: Punkte ziehen, hinzufügen, löschen -----------------
  curveCanvas.addEventListener('mousedown', (e) => {
    if (!editMode || !rawActions) return;
    const [mx, my] = curveMouseXY(e);
    const idx = findNearestActionIndex(mx, my);
    if (idx >= 0) {
      editDragIndex = idx;
      editDragStartValue = { ...rawActions[idx] };
    } else {
      // Freie Stelle getroffen - neuer Punkt genau dort, sofort greifbar
      // für dieselbe Zieh-Geste (Setzen und Feinjustieren in einem Zug).
      rawActions.push({ atMs: curveMsOfX(e.clientX), pos: curvePosOfY(e.clientY) });
      editDragIndex = rawActions.length - 1;
      editDragStartValue = null; // neuer Punkt - immer speichern, nichts zum Vergleichen
    }
    redrawCurve();
    seekTo(rawActions[editDragIndex].atMs);
  });

  curveCanvas.addEventListener('mousemove', (e) => {
    if (!editMode || editDragIndex === null) return;
    const atMs = Math.max(0, Math.min(totalMs, curveMsOfX(e.clientX)));
    rawActions[editDragIndex] = { atMs, pos: curvePosOfY(e.clientY) };
    redrawCurve();
    // Video folgt beim Ziehen mit - man sieht, welcher Moment gerade
    // markiert wird, statt blind auf Zeitwerte zu vertrauen. seekTo() ist
    // bereits ein no-op ohne Video, also kein zusätzlicher Guard nötig.
    seekTo(atMs);
  });

  window.addEventListener('mouseup', async () => {
    if (!editMode || editDragIndex === null) return;
    const idx = editDragIndex, startValue = editDragStartValue;
    editDragIndex = null;
    editDragStartValue = null;
    redrawCurve();
    // Ein Doppelklick zum Löschen (siehe unten) besteht browserseitig aus
    // zwei normalen mousedown/mouseup-Paaren VOR dem eigentlichen
    // 'dblclick' - ohne diesen Vergleich würde jeder reine Klick auf einen
    // bestehenden Punkt (ohne Bewegung) unnötig zweimal denselben,
    // unveränderten Stand speichern.
    const current = rawActions[idx];
    const unchanged = startValue && current
      && current.atMs === startValue.atMs && current.pos === startValue.pos;
    if (unchanged) return;
    await persistRawActions();
  });

  curveCanvas.addEventListener('dblclick', async (e) => {
    if (!editMode || !rawActions) return;
    const [mx, my] = curveMouseXY(e);
    const idx = findNearestActionIndex(mx, my);
    if (idx < 0) return;
    if (rawActions.length <= 2) {
      log('Editor: mindestens 2 Punkte müssen im Skript bleiben.');
      return;
    }
    rawActions.splice(idx, 1);
    redrawCurve();
    await persistRawActions();
  });

  // persistRawActions speichert den aktuellen Punktstand sofort - keine
  // separate "Speichern"-Aktion, dieselbe Sofort-Speicher-Logik wie bei
  // den O-Markern oben. Bei Fehler (z.B. Datei zwischenzeitlich entfernt)
  // bleibt der bearbeitete Stand im Editor sichtbar, wird aber nicht als
  // gespeichert angenommen - ein erneuter Zug versucht es wieder.
  async function persistRawActions() {
    if (!scriptPath || !rawActions) return;
    const sorted = [...rawActions].sort((a, b) => a.atMs - b.atMs);
    try {
      await SaveScriptActions(sorted.map(p => ({ at: p.atMs, pos: p.pos })));
    } catch (err) {
      log('Kurve speichern: ' + err);
      return;
    }
    rawActions = sorted;
    redrawCurve();
    drawHeatmap();
  }

  async function setEditMode(on) {
    if (on) {
      try {
        const actions = await GetScriptActions();
        rawActions = (Array.isArray(actions) ? actions : []).map(a => ({ atMs: a.at, pos: a.pos }));
      } catch (err) {
        log('Editor: Punkte laden fehlgeschlagen: ' + err);
        el('#pb-curve-edit').checked = false;
        return;
      }
      editMode = true;
    } else {
      editMode = false;
      editDragIndex = null;
      rawActions = null;
    }
    el('#pb-curve-edit-hint').style.display = editMode ? 'block' : 'none';
    redrawCurve();
  }

  el('#pb-curve-edit').addEventListener('change', e => setEditMode(e.target.checked));

  async function drawHeatmap() {
    try {
      heatmapPoints = await GetHeatmap(HEATMAP_BUCKETS);
    } catch (err) {
      heatmapPoints = null;
      heatmapCanvas.style.display = 'none';
      el('#pb-marker-hint').style.display = 'none';
      el('#pb-marker-auto-row').style.display = 'none';
      return;
    }
    if (!heatmapPoints || heatmapPoints.length === 0) {
      heatmapCanvas.style.display = 'none';
      el('#pb-marker-hint').style.display = 'none';
      el('#pb-marker-auto-row').style.display = 'none';
      return;
    }
    heatmapCanvas.style.display = 'block';
    el('#pb-marker-hint').style.display = 'block';
    el('#pb-marker-auto-row').style.display = 'flex';
    heatmapCanvas.width = heatmapCanvas.clientWidth || 800;
    redrawHeatmap();
  }

  function canvasXToMs(clientX) {
    const rect = heatmapCanvas.getBoundingClientRect();
    const frac = Math.max(0, Math.min(1, (clientX - rect.left) / rect.width));
    return Math.round(frac * totalMs);
  }

  heatmapCanvas.addEventListener('mousedown', (e) => {
    markerDragStartMs = canvasXToMs(e.clientX);
    markerDragMoved = false;
  });
  heatmapCanvas.addEventListener('mousemove', (e) => {
    if (markerDragStartMs === null) return;
    const cur = canvasXToMs(e.clientX);
    if (Math.abs(cur - markerDragStartMs) > 150) markerDragMoved = true;
    if (!markerDragMoved) return;
    marker = { startMs: Math.min(markerDragStartMs, cur), endMs: Math.max(markerDragStartMs, cur) };
    redrawHeatmap();
    redrawCurve();
  });
  window.addEventListener('mouseup', () => {
    if (markerDragStartMs === null) return;
    const clickedMs = markerDragStartMs;
    const wasDrag = markerDragMoved;
    markerDragStartMs = null;
    markerDragMoved = false;

    // Klick ohne Ziehen = an diese Stelle springen (statt zu markieren).
    // Markiert wird nur beim echten Ziehen - so lässt sich dieselbe Leiste
    // für beides nutzen, ohne Moduswechsel.
    if (!wasDrag) {
      seekTo(clickedMs);
      return;
    }

    updateMarkerHint();
    redrawHeatmap();
    if (scriptPath) {
      SaveMarker(scriptPath, marker ? marker.startMs : 0, marker ? marker.endMs : 0)
        .catch(err => log('Markierung speichern: ' + err));
    }
  });

  // seekTo springt im Video an die angegebene Stelle. Bei aktivem
  // Video-Sync folgt das Gerät automatisch mit, da die Videoposition
  // ohnehin die Quelle ist (siehe player/sync.go) - es ist kein
  // zusätzliches Zutun nötig. Ohne Video gibt es nichts zu spulen; dann
  // bleibt der Klick wirkungslos (die eigene Uhr in Play() lässt sich
  // nicht nachträglich verschieben).
  function seekTo(atMs) {
    if (!videoPath || !videoEl.duration) return;
    videoEl.currentTime = Math.max(0, Math.min(videoEl.duration, atMs / 1000));
    // Auto-Extended-O darf nach einem Sprung erneut auslösen, wenn der
    // markierte Bereich danach nochmal erreicht wird.
    if (marker && atMs < marker.startMs) autoEOTriggeredForMarker = false;
    redrawHeatmap();
  }

  el('#pb-marker-clear').addEventListener('click', () => {
    marker = null;
    updateMarkerHint();
    redrawHeatmap();
    if (scriptPath) SaveMarker(scriptPath, 0, 0).catch(err => log('Markierung speichern: ' + err));
  });

  el('#pb-omarker-kind').addEventListener('change', e => {
    el('#pb-omarker-intensity-row').style.display = e.target.value === 'secondary' ? 'flex' : 'none';
  });

  el('#pb-omarker-add').addEventListener('click', async () => {
    if (!marker || !scriptPath) return;
    const kind = el('#pb-omarker-kind').value;
    const intensity = kind === 'primary'
      ? 1.0
      : Math.max(0, Math.min(1, Number(el('#pb-omarker-intensity').value) || 0));
    const next = [...oMarkers, { startMs: marker.startMs, endMs: marker.endMs, kind, intensity }];
    try {
      await SaveOMarkers(scriptPath, next);
    } catch (err) {
      log('O-Marker hinzufügen: ' + err);
      return;
    }
    oMarkers = next;
    renderOMarkerList();
    redrawHeatmap();
    redrawCurve();
  });

  // checkAutoExtendedO wird bei jedem Fortschritts-Update aufgerufen -
  // löst Extended-O einmal pro Wiedergabe aus, sobald die Position in den
  // markierten Bereich eintritt (falls aktiviert).
  function checkAutoExtendedO(atMs) {
    if (!marker || !el('#pb-marker-auto').checked || autoEOTriggeredForMarker) return;
    if (atMs >= marker.startMs && atMs <= marker.endMs) {
      autoEOTriggeredForMarker = true;
      if (!el('#pb-eo-trigger').disabled) triggerEO();
    }
  }



  // Fallengelassenes Skript übernehmen - gleicher Ladeweg wie die
  // Dateiauswahl.
  window.addEventListener('drop:script', e => loadScript(e.detail.path, e.detail.extraCount || 0));

  async function chooseScript() {
    const path = await PickFunscriptFile();
    // Fokus zurück ins Fenster holen: nach dem Schließen des nativen
    // Datei-Dialogs bleibt der Tastaturfokus sonst am Auswahl-Button
    // hängen (oder ganz außerhalb der Webview), und die Tastenkürzel
    // wirken scheinbar nicht mehr. In der Testumgebung reproduzierbar,
    // darum hier defensiv behandelt.
    restoreKeyboardFocus();
    if (!path) return;
    await loadScript(path);
  }

  // restoreKeyboardFocus nimmt den Fokus von einem eventuell fokussierten
  // Bedienelement und legt ihn aufs Dokument, damit globale Tastenkürzel
  // (Leertaste, E) wieder greifen.
  function restoreKeyboardFocus() {
    if (document.activeElement && document.activeElement.blur) {
      document.activeElement.blur();
    }
    window.focus();
  }

  async function loadScript(path, extraCount = 0) {
    const info = await LoadFunscript(path);
    scriptPath = info.path;
    totalMs = Math.max(info.durationMs, 1);
    // Stapelverarbeitung mehrerer Skripte gibt es noch nicht - vorher
    // wurden weitere abgelegte Skripte einfach stillschweigend verworfen,
    // ohne dass sichtbar war, dass überhaupt mehr als eins ankam (siehe
    // dieselbe Behandlung für Videos in generator.js).
    const batchNote = extraCount > 0
      ? ` (${extraCount} weitere${extraCount === 1 ? 's' : ''} abgelegte${extraCount === 1 ? 's' : ''} Skript${extraCount === 1 ? '' : 'e'} ignoriert - Stapelverarbeitung gibt es noch nicht)`
      : '';
    el('#pb-script-path').textContent = scriptPath + batchNote;
    scriptHasContactVibration = !!info.contactVibration;
    const showContact = scriptHasContactVibration;
    el('#pb-contact-off-row').style.display = showContact ? 'flex' : 'none';
    el('#pb-contact-intensity-row').style.display = showContact ? 'flex' : 'none';
    el('#pb-contact-span-row').style.display = showContact ? 'flex' : 'none';
    el('#pb-contact-curve-row').style.display = showContact ? 'flex' : 'none';
    if (!showContact) {
      el('#pb-contact-off').checked = false;
      el('#pb-contact-intensity').value = '1';
      el('#pb-contact-intensity-val').textContent = '1.00';
    } else {
      // Rezept-Defaults in die Live-Controls übernehmen (Datei bleibt Quelle).
      const span = (info.contactVibrationSpan > 0) ? info.contactVibrationSpan : 0.75;
      el('#pb-contact-span').value = String(span);
      el('#pb-contact-span-val').textContent = Number(span).toFixed(2);
      const curve = info.contactVibrationCurve || 'linear';
      el('#pb-contact-curve').value = ['linear', 'soft', 'peak'].includes(curve) ? curve : 'linear';
    }
    if (info.hasVideo) {
      videoPath = info.videoPath;
      videoEl.src = await VideoFileURL();
      videoEl.style.display = 'block';
      el('#pb-novideo').style.display = 'none';
      el('#pb-video-sync-row').style.display = 'flex';
    } else {
      videoPath = null;
      videoEl.style.display = 'none';
      el('#pb-novideo').style.display = 'block';
      el('#pb-video-sync-row').style.display = 'none';
    }
    try {
      marker = await GetMarker(scriptPath);
    } catch (err) {
      marker = null;
    }
    try {
      const result = await GetOMarkers(scriptPath);
      oMarkers = Array.isArray(result) ? result : [];
    } catch (err) {
      oMarkers = [];
    }
    el('#pb-omarker-hint').style.display = 'block';
    el('#pb-omarker-add-row').style.display = 'flex';
    updateMarkerHint();
    renderOMarkerList();
    // Ein neu geladenes Skript hat andere Punkte - ein noch aktiver
    // Editiermodus vom vorherigen Skript würde sonst dessen (falsche)
    // rawActions weiterbenutzen.
    el('#pb-curve-edit').checked = false;
    setEditMode(false);
    drawHeatmap();
    drawCurve();
    describeScript();
    el('#pb-script-doctor-row').style.display = 'flex';
    el('#pb-script-doctor-result').style.display = 'none';
    el('#pb-script-doctor-status').textContent = '';
    el('#pb-offset-row').style.display = 'flex';
    el('#pb-offset-hint').style.display = 'block';
    GetScriptOffset().then(v => { el('#pb-offset').value = v || 0; }).catch(() => {});
  }

  // startScriptPlayback startet nur das Funscript/Gerät (StartPlayback +
  // Statuswechsel), ohne das <video>-Element anzufassen - gemeinsamer Kern
  // für den App-eigenen "Abspielen"-Knopf (play(), der zusätzlich das
  // Video von vorn startet) und den nativen Video-Play-Listener weiter
  // unten (der das Video NICHT zurückspulen darf, weil es dort schon an
  // seiner Position läuft). Gibt zurück, ob der Start geklappt hat.
  async function startScriptPlayback() {
    if (!scriptPath) { alert('Bitte zuerst eine .funscript-Datei wählen.'); return false; }
    el('#pb-log').textContent = '';

    const useVideoSync = videoPath && el('#pb-use-video-sync').checked;
    const opts = {
      mock: el('#pb-mock').checked,
      syncMode: el('#pb-sync').value,
      tickMs: parseInt(el('#pb-tick').value, 10) || 50,
      maxSpeed: parseFloat(el('#pb-maxspeed').value) || 0.6,
      smoothing: parseFloat(el('#pb-smoothing').value) || 0,
      softStartMs: parseInt(el('#pb-softstart').value, 10) || 0,
      useVideoSync: !!useVideoSync,
      extendedOEnabled: el('#pb-eo-enabled').checked,
      extendedOMin: parseFloat(el('#pb-eo-min').value) || 0.1,
      extendedOHoldS: parseFloat(el('#pb-eo-hold').value) || 10,
      extendedORestoreMs: parseFloat(el('#pb-eo-restore').value) || 500,
      disableContactVibration: scriptHasContactVibration && el('#pb-contact-off').checked,
      contactIntensityScale: scriptHasContactVibration
        ? parseFloat(el('#pb-contact-intensity').value) || 1
        : 1,
      contactExtraSmooth: 0,
      contactVibrationSpan: scriptHasContactVibration
        ? parseFloat(el('#pb-contact-span').value) || 0
        : 0,
      contactVibrationCurve: scriptHasContactVibration
        ? (el('#pb-contact-curve').value || '')
        : '',
    };

    try {
      await StartPlayback(opts);
    } catch (err) {
      alert('Fehler: ' + err);
      return false;
    }
    setPlayingState(true);
    return true;
  }

  async function play() {
    const useVideoSync = videoPath && el('#pb-use-video-sync').checked;
    if (!(await startScriptPlayback())) return;

    if (useVideoSync) {
      videoEl.currentTime = 0;
      videoEl.play().catch(() => {});
    }
  }

  async function stop() {
    await StopPlayback();
    if (videoPath) videoEl.pause();
    setPlayingState(false);
  }

  async function triggerEO() {
    await TriggerExtendedO(
      parseFloat(el('#pb-eo-min').value) || 0.1,
      parseFloat(el('#pb-eo-hold').value) || 10,
      parseFloat(el('#pb-eo-restore').value) || 500,
    );
  }

  // Video-Position ans Backend melden, solange Sync aktiv ist - das treibt
  // player.Sync() an (siehe player/sync.go). timeupdate feuert im Browser
  // ca. alle 250ms, das reicht für flüssiges Gerätefeedback.
  videoEl.addEventListener('timeupdate', () => {
    currentPosMs = Math.round(videoEl.currentTime * 1000);

    // Markierten Abschnitt wiederholen. Das ist das Werkzeug, mit dem sich
    // der Offset überhaupt einstellen lässt: ohne Wiederholung müsste man
    // nach jeder Korrektur von Hand zurückspulen, und bis man wieder an der
    // fraglichen Stelle ist, hat man den Vergleich verloren.
    if (el('#pb-loop').checked && marker && currentPosMs >= marker.endMs) {
      videoEl.currentTime = marker.startMs / 1000;
      currentPosMs = marker.startMs;
    }
    if (!playing) { redrawHeatmap(); redrawCurve(); }
    if (playing && el('#pb-use-video-sync').checked) {
      ReportVideoPosition(Math.round(videoEl.currentTime * 1000));
    }
  });

  // Historische video:pause/resume-Events. Extended-O skaliert nur noch die
  // Amplitude und pausiert das Video nicht mehr - Listener bleiben harmlos.
  EventsOn('video:pause', () => videoEl.pause());
  EventsOn('video:resume', () => videoEl.play().catch(() => {}));

  // Video-Ende soll auch unsere Wiedergabe sauber beenden - sonst bleibt
  // player.Sync() im Leerlauf hängen (wartet ewig auf weitere Positionen,
  // die nach Videoende nicht mehr kommen).
  videoEl.addEventListener('ended', () => {
    if (playing && el('#pb-use-video-sync').checked) stop();
  });

  // Das native Play im <video>-Element (Browser-eigene Steuerung, per
  // Klick oder Leertaste mit Fokus auf dem Video) soll das Funscript
  // gleich mitstarten, statt zwei getrennte Aktionen zu verlangen. Die
  // playing-Prüfung verhindert eine Rückkopplungsschleife: play() selbst
  // ruft bei aktivem Video-Sync videoEl.play() auf, was dieses 'play'
  // erneut auslösen würde - zu dem Zeitpunkt steht playing aber schon auf
  // true (setPlayingState lief vor dem videoEl.play()-Aufruf), also bricht
  // die Prüfung hier sofort ab. currentTime wird bewusst NICHT
  // zurückgesetzt (anders als play()) - das Video läuft hier schon an
  // seiner aktuellen Position, ein Sprung auf 0 wäre ein sichtbarer Bug.
  videoEl.addEventListener('play', () => {
    if (playing || !scriptPath) return;
    startScriptPlayback();
  });

  EventsOn('playback:log', log);
  EventsOn('playback:error', msg => log('FEHLER: ' + msg));
  EventsOn('playback:done', () => setPlayingState(false));
  EventsOn('playback:frame', f => {
    totalMs = Math.max(f.totalMs, 1);
    currentPosMs = f.atMs;
    el('#pb-progress').style.width = Math.min(100, (f.atMs / totalMs) * 100) + '%';
    el('#pb-vib').textContent = Math.round(f.vibration * 100) + '%';
    el('#pb-suc').textContent = Math.round(f.suction * 100) + '%';
    checkAutoExtendedO(f.atMs);
    redrawHeatmap();
    redrawCurve();
  });

  // Canvas-Breite ist eine feste Pixelangabe, keine CSS-Größe - bei einer
  // Fenstergrößenänderung muss sie neu gesetzt werden, sonst wird die Kurve
  // verzerrt skaliert dargestellt.
  window.addEventListener('resize', () => {
    if (curvePoints) {
      curveCanvas.width = curveCanvas.clientWidth || 800;
      redrawCurve();
    }
    if (heatmapPoints) {
      heatmapCanvas.width = heatmapCanvas.clientWidth || 800;
      redrawHeatmap();
    }
  });

  el('#pb-script-doctor').addEventListener('click', async () => {
    const status = el('#pb-script-doctor-status');
    const box = el('#pb-script-doctor-result');
    const btn = el('#pb-script-doctor');
    btn.disabled = true;
    status.textContent = 'Prüfe...';
    try {
      const result = await ScriptQuality();
      status.textContent = '';
      const pct = Math.round((result.score || 0) * 100);
      box.style.display = 'block';
      box.style.background = result.passed ? 'rgba(61,216,117,0.12)' : 'rgba(216,77,77,0.12)';
      box.style.border = `1px solid ${result.passed ? 'var(--ok)' : 'var(--danger)'}`;
      let html = `<b>Signal Quality (Script Doctor): ${pct}% ${result.passed ? '(unauffällig)' : '(bitte prüfen)'}</b>`;
      html += '<br><span style="opacity:0.7;">Nur Signalqualität — Jitter, Sprünge, Lücken, Geräte-Dichte. '
        + 'Kein Beweis, dass die Kurve zum Video passt (das wäre Motion Fidelity / Phase-Vergleich). '
        + 'Geschätzt nur aus dem Skript, ohne Video; trackingbasierte Prüfungen fehlen hier, '
        + 'anders als direkt nach einer Generierung.</span>';
      if (result.warnings && result.warnings.length > 0) {
        html += '<ul style="margin:6px 0 0 18px; padding:0;">'
          + result.warnings.map(w => `<li>${w}</li>`).join('') + '</ul>';
      }
      box.innerHTML = html;
    } catch (err) {
      status.textContent = 'Prüfung fehlgeschlagen: ' + err;
    } finally {
      btn.disabled = false;
    }
  });
  el('#pb-choose').addEventListener('click', chooseScript);
  el('#pb-play').addEventListener('click', play);
  el('#pb-stop').addEventListener('click', stop);
  el('#pb-eo-trigger').addEventListener('click', triggerEO);
  el('#pb-eo-enabled').addEventListener('change', () => {
    el('#pb-eo-trigger').disabled = !playing || !el('#pb-eo-enabled').checked;
  });

  // Tastenkürzel: Leertaste = Play/Stop, E = Extended-O. Greift nicht ein,
  // solange in einem Eingabefeld getippt wird.
  document.addEventListener('keydown', (e) => {
    const tag = (e.target.tagName || '').toLowerCase();
    if (tag === 'input' || tag === 'select' || tag === 'textarea') return;

    if (e.code === 'Space') {
      e.preventDefault();
      if (playing) stop(); else play();
    } else if (e.key === 'ArrowLeft' || e.key === 'ArrowRight') {
      // Springen, ohne die Maus zu benutzen. Der Player wird typischerweise
      // einhändig bedient - alles, was einen Mausweg spart, zählt hier mehr
      // als anderswo.
      e.preventDefault();
      const delta = (e.key === 'ArrowRight' ? 1 : -1) * (e.shiftKey ? 1 : 5);
      seekBy(delta * 1000);
    } else if (e.key === ',' || e.key === '.') {
      e.preventDefault();
      seekBy(e.key === '.' ? 200 : -200);
    } else if (e.key === '+' || e.key === '=' || e.key === '-') {
      // Offset live nachjustieren, während der markierte Abschnitt läuft.
      e.preventDefault();
      const step = e.key === '-' ? -10 : 10;
      applyOffset((Number(el('#pb-offset').value) || 0) + step);
    } else if (e.key === 'l') {
      el('#pb-loop').checked = !el('#pb-loop').checked;
      log(el('#pb-loop').checked ? 'Wiederholung an.' : 'Wiederholung aus.');
    } else if (e.key >= '1' && e.key <= '9') {
      if (totalMs > 0) seekTo(Math.round(totalMs * (Number(e.key) - 1) / 9));
    } else if (e.key.toLowerCase() === 'e') {
      if (!el('#pb-eo-trigger').disabled) triggerEO();
    }
  });

  // Gespeicherte Standardwerte übernehmen, sobald settings.js sie geladen hat.
  getSettingsCache().then(s => {
    el('#pb-mock').checked = s.playbackMock;
    el('#pb-sync').value = s.playbackSync;
    el('#pb-tick').value = s.playbackTickMs;
    el('#pb-maxspeed').value = s.playbackMaxSpeed;
    el('#pb-smoothing').value = s.playbackSmoothing;
    el('#pb-softstart').value = s.playbackSoftStartMs;
    el('#pb-eo-enabled').checked = s.playbackEOEnabled;
    el('#pb-eo-min').value = s.playbackEOMin;
    el('#pb-eo-hold').value = s.playbackEOHoldS;
    el('#pb-eo-restore').value = s.playbackEORestoreMs;
  });
  el('#pb-mock').addEventListener('change', e => saveSetting('playback.mock', e.target.checked));
  el('#pb-sync').addEventListener('change', e => saveSetting('playback.sync_mode', e.target.value));
  el('#pb-tick').addEventListener('change', e => saveSetting('playback.tick_ms', parseFloat(e.target.value)));
  el('#pb-maxspeed').addEventListener('change', e => saveSetting('playback.max_speed', parseFloat(e.target.value)));
  el('#pb-smoothing').addEventListener('change', e => saveSetting('playback.smoothing', parseFloat(e.target.value)));
  el('#pb-softstart').addEventListener('change', e => saveSetting('playback.soft_start_ms', parseFloat(e.target.value)));
  el('#pb-eo-enabled').addEventListener('change', e => saveSetting('playback.extended_o_enabled', e.target.checked));
  el('#pb-eo-min').addEventListener('change', e => saveSetting('playback.extended_o_min', parseFloat(e.target.value)));
  el('#pb-eo-hold').addEventListener('change', e => saveSetting('playback.extended_o_hold_seconds', parseFloat(e.target.value)));
  el('#pb-eo-restore').addEventListener('change', e => saveSetting('playback.extended_o_restore_ms', parseFloat(e.target.value)));

  async function refreshScriptVisuals() {
    if (!scriptPath) return;
    try {
      const [heat, curve, vib, markers] = await Promise.all([
        GetHeatmap(HEATMAP_BUCKETS),
        GetScriptCurve(CURVE_MAX_POINTS),
        scriptHasContactVibration ? GetVibrationCurvePreview(contactPreviewOpts()) : Promise.resolve(null),
        GetOMarkers(scriptPath),
      ]);
      heatmapPoints = heat;
      curvePoints = curve;
      vibrationCurvePoints = vib;
      oMarkers = Array.isArray(markers) ? markers : [];
      if (el('#pb-curve-edit').checked) {
        const acts = await GetScriptActions();
        rawActions = Array.isArray(acts) ? acts : rawActions;
      }
      redrawHeatmap();
      redrawCurve();
      renderOMarkerList();
      describeScript();
    } catch (err) {
      log('Aktualisieren: ' + err);
    }
  }

  window.addEventListener('ozone:hotkey', async (ev) => {
    if (!scriptPath) return;
    const nowMs = (ev.detail && ev.detail.nowMs) || 0;
    try {
      oMarkers = await applyHotkeyOMarker(scriptPath, nowMs, oMarkers);
      renderOMarkerList();
      redrawHeatmap();
      redrawCurve();
      log('O-Marker gesetzt: ' + (nowMs / 1000).toFixed(1) + 's–' + ((nowMs + 4000) / 1000).toFixed(1) + 's');
    } catch (err) {
      log('O-Taste: ' + err);
    }
  });
  window.addEventListener('ozone:suggested', () => { refreshScriptVisuals(); });
  window.addEventListener('polarity:inverted', () => { refreshScriptVisuals(); });
  window.addEventListener('ringdown:applied', () => { refreshScriptVisuals(); });

  return {
    // Von generator.js genutzt, um ein Ergebnis direkt zu übernehmen.
    loadScriptPath: (path) => loadScript(path).then(() => switchToPlaybackTab()),
  };
}

function switchToPlaybackTab() {
  document.querySelector('.tab-btn[data-tab="playback"]').click();
}
