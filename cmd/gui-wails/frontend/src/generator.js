import { SubmitFeedback, PickVideoFile, LoadFirstFrame, GenerateScript, CheckGeneratorDependencies, ScriptExistsForVideo, AutoDetectROI, CheckAIRoiAvailable, CheckAudioCheckAvailable, SuggestProfile, LabelScene } from '../wailsjs/go/main/App';
import { EventsOn } from '../wailsjs/runtime/runtime';

export function initGenerator(root, playback) {
  root.innerHTML = `
    <h2>Skript erzeugen</h2>
    <div class="row">
      <button id="gen-choose">Video wählen...</button>
      <span class="path-label" id="gen-video-path">Kein Video gewählt</span>
      <button id="gen-check-deps">Abhängigkeiten prüfen</button>
    </div>
    <div class="row" style="align-items:center;">
      <button id="gen-autoroi" class="primary" disabled>Region automatisch finden</button>
      <span class="checkbox-row" style="margin:0"><input type="checkbox" id="gen-ai-roi" disabled />
        <label for="gen-ai-roi" style="width:auto">KI-Erkennung (ONNX)</label></span>
    </div>
    <p class="hint" id="gen-autoroi-hint" style="margin:0 0 6px 0">Analysiert die Bewegung im Video - danach lässt sich die Region trotzdem von Hand korrigieren.</p>

    <div id="roi-canvas-wrap">
      <canvas id="roi-canvas"></canvas>
    </div>
    <div class="path-label" id="gen-roi-label">Keine Region markiert</div>
    <div class="row" style="align-items:center; margin-top:6px;">
      <button id="gen-roi2-toggle" type="button">2. Region</button>
      <span class="hint" id="gen-roi2-hint" style="margin:0">Shift+Ziehen oder Knopf: zweite Region (violett). Für Tf/Tj (Abstand + Sog) nötig.</span>
    </div>
    <div class="path-label" id="gen-roi2-label">Keine 2. Region markiert</div>

    <div class="row" style="align-items:center;">
      <label style="width:auto;">Bewegungsart</label>
      <select id="gen-profile">
        <option value="standard">Hubbewegung (Standard)</option>
        <option value="weich">Weiches Gewebe (schwingt nach)</option>
        <option value="tf">Tf/Tj (Abstand + Sog)</option>
      </select>
    </div>
    <p class="hint" id="gen-profile-hint" style="margin:0 0 10px 0;">Tf und Tj sind intern dasselbe
      Rezept (Abstand zweier Regionen, Sog folgt) - darum nur ein Eintrag. Wird automatisch
      ausgewählt, sobald eine 2. Region markiert wird. „Weiches Gewebe" behandelt Nachschwingungen
      nicht als eigene Hübe. An einem Testvideo mit Anstoß alle 800 ms: 81 Keyframes werden
      zu 42 — den Anstößen selbst. Saubere Hubsignale bleiben davon unberührt.</p>
    <p class="hint" id="gen-tftj-hint" style="display:none; margin:0 0 6px 0;">
      Tf/Tj (Abstand + Sog): zwei Regionen markieren (erste Region ziehen, dann Shift+Ziehen
      oder „2. Region“ für die zweite, violett). Der Abstand zwischen beiden steuert den Hub;
      Sog folgt der Position. Vibration bleibt 0, außer „Kontakt-Vibration" unten ist
      aktiviert — kein Akt-Detektor, reine Abstandsmessung.
    </p>
    <div class="checkbox-row" id="gen-contact-vibration-row" style="display:none;">
      <input type="checkbox" id="gen-contact-vibration" />
      <label for="gen-contact-vibration">Kontakt-Vibration: vibriert zusätzlich zum Sog, sobald
        ROI1 nahe an ROI2 herankommt (z.B. Eichel an Brustwarze oder Zunge) - Dauer/Stärke
        richten sich nach dem gemessenen Abstand in diesem Video, kein fester Impuls</label>
    </div>

    <div class="row" style="align-items:center;">
      <button id="gen-suggest-profile" disabled>Profil vorschlagen</button>
      <span class="hint" id="gen-suggest-status" style="margin:0"></span>
    </div>
    <div class="row" style="align-items:center;">
      <input type="text" id="gen-scene-label" placeholder="Name für diese Szene (optional)" style="flex:1;" />
      <button id="gen-label-scene" disabled>Szene merken</button>
    </div>
    <p class="hint" style="margin:0 0 10px 0">
      „Szene merken" speichert die Bewegungssignatur unter diesem Namen - spätere ähnliche
      Videos bekommen dann automatisch dieses Profil vorgeschlagen (gemessen, ohne KI).
      „Profil vorschlagen" vergleicht zuerst gegen gemerkte Szenen, erst danach optional
      gegen einen lokalen KI-Server (Einstellungen → KI-Server-Adresse). Beides ein
      Vorschlag zum Bestätigen, nichts wird automatisch übernommen.
    </p>

    <details id="gen-advanced" style="margin:6px 0 10px 0;">
      <summary style="cursor:pointer;">Erweiterte Einstellungen</summary>
      <div style="margin-top:8px;">
        <div class="checkbox-row"><input type="checkbox" id="gen-invert" /><label for="gen-invert">Bewegungsrichtung umkehren</label></div>
        <div class="checkbox-row"><input type="checkbox" id="gen-camcomp" checked /><label for="gen-camcomp">Kamerabewegungs-Kompensation (empfohlen bei Kameraschwenks)</label></div>
        <div class="checkbox-row"><input type="checkbox" id="gen-scenecut" checked /><label for="gen-scenecut">Szenenschnitt-Erkennung (verankert Tracker bei harten Schnitten neu)</label></div>
        <div class="row" style="align-items:center;">
          <label style="width:auto;">Tracking-Verfahren</label>
          <select id="gen-backend">
            <option value="csrt">CSRT (Standard, robust)</option>
            <option value="flow">Flow (keine Region nötig, ca. 4x schneller)</option>
            <option value="grid_lk">Gitter/Optical-Flow (braucht Region wie CSRT, ca. 15x schneller)</option>
            <option value="region_fusion">Region-Fusion (4 Teilregionen, gewichtet verschmolzen)</option>
            <option value="region_fusion_auto">Region-Fusion Automatisch (4 Zonen, keine Region nötig)</option>
          </select>
        </div>
        <p class="hint" id="gen-backend-hint" style="margin:0 0 6px 0;">CSRT: bei ruhiger Kamera
          gleichwertig zu Flow, bei Kameraschwenks trifft es die Bewegungsstärke besser (gemessen
          113 gegen 145 bei 110 tatsächlicher Bewegung), weil dort die Kamerabewegung über
          Hintergrundmerkmale herausgerechnet wird. Gitter/Optical-Flow: verfolgt ein Raster aus
          Einzelpunkten statt einer Box - GEMESSEN auf einem realen Clip mindestens gleich gute
          Qualität wie CSRT bei ~15x der Geschwindigkeit, robuster als CSRT bei schwierigen
          (kleinen/unscharfen) Regionen (siehe docs/NEXT.md Abschnitt 8). Region-Fusion: teilt
          die Region in 4 Teilregionen und gewichtet sie je Frame nach aktueller
          Bewegungsstärke - GEMESSEN an echtem Material mindestens gleichauf mit CSRT, in
          einem von zwei getesteten Abschnitten deutlich besser als beide (siehe docs/NEXT.md).
          Region-Fusion Automatisch: wie Region-Fusion, aber ohne markierte Region - teilt
          automatisch das ganze Bild in 4 Zonen, wie Flow also ohne Markier-Schritt. Noch nicht
          gegen eine Referenz gemessen, ein Kandidat.</p>
        <div class="checkbox-row"><input type="checkbox" id="gen-dynrange" checked /><label for="gen-dynrange">Gleitende Dynamik (hebt schwache Abschnitte auf nutzbare Stärke)</label></div>
        <div class="checkbox-row"><input type="checkbox" id="gen-opencl" /><label for="gen-opencl">GPU-Beschleunigung nutzen, falls verfügbar (OpenCL)</label></div>
        <div class="checkbox-row"><input type="checkbox" id="gen-retry" checked /><label for="gen-retry">Auto-Retry (bei schlechter Qualität andere Signalparameter probieren)</label></div>
        <div class="checkbox-row"><input type="checkbox" id="gen-ai-quality" /><label for="gen-ai-quality">KI-Zweitmeinung zur Qualität einholen (lokaler KI-Server, optional - beeinflusst den Quality-Doctor-Wert nicht)</label></div>
        <div class="checkbox-row"><input type="checkbox" id="gen-audio-check" /><label for="gen-audio-check">Skript-Tempo gegen die Tonspur prüfen (braucht ffmpeg, rein klassisch - beeinflusst den Quality-Doctor-Wert nicht)</label></div>
        <div class="checkbox-row"><input type="checkbox" id="gen-auto-ozone" /><label for="gen-auto-ozone">O-Marker automatisch vorschlagen (letztes Achtel, höchste mittlere Position - klassisch aus dem Signal, kein KI-Modell; nur gesetzt, wenn das Ende deutlich hoch liegt)</label></div>
        <div class="field-row"><label>Bewegungsachse</label>
          <select id="gen-axis">
            <option value="" selected>Automatisch (empfohlen)</option>
            <option value="x">Waagerecht erzwingen</option>
            <option value="y">Senkrecht erzwingen</option>
          </select>
        </div>
        <p class="hint" style="margin:0 0 6px 0;">Waagerecht und senkrecht werden immer
          beide verfolgt - "Automatisch" wählt danach die Achse mit der deutlich größeren
          Spannweite. Nur bei einer erkennbar falschen automatischen Wahl fest erzwingen.</p>
        <div class="checkbox-row"><input type="checkbox" id="gen-adaptive" checked /><label for="gen-adaptive">Adaptive Keyframes (zusätzliche Punkte bei asymmetrischen Bewegungen)</label></div>
        <div class="checkbox-row"><input type="checkbox" id="gen-perscene" /><label for="gen-perscene">Region nach jedem Schnitt neu suchen (besser bei geschnittenem Material, dauert länger)</label></div>
        <div class="checkbox-row"><input type="checkbox" id="gen-native" /><label for="gen-native">Go-Pipeline (experimentell): CSRT-Tracking + Signalpfad ohne Python — nur CSRT, eine Region, ohne Tf/Tj/KI/Audio/Auto-Retry. Sonst Fallback auf Python.</label></div>
        <div class="field-row"><label>Glättungs-Fenster</label><input type="number" id="gen-smooth" value="11" /></div>
        <div class="field-row"><label>Min. Keyframe-Abstand (ms)</label><input type="number" id="gen-peakdist" value="150" /></div>
        <div class="field-row"><label>RDP-Toleranz (0 = aus)</label><input type="number" id="gen-rdp" value="0" step="0.5" min="0" /></div>
      </div>
    </details>

    <div class="row"><button id="gen-generate" class="primary" disabled>Funscript generieren</button></div>
    <div class="path-label" id="gen-status"></div>
    <div id="gen-progress-wrap" style="display:none; margin-top:8px;">
      <div style="height:10px; border-radius:5px; background:rgba(255,255,255,0.10); overflow:hidden;">
        <div id="gen-progress-bar" style="height:100%; width:0%; background:var(--accent, #00c8ff);
             transition:width .2s linear;"></div>
      </div>
      <div id="gen-progress-text" class="hint" style="margin-top:4px;"></div>
    </div>
    <div id="gen-feedback" style="display:none; margin-top:10px; padding:10px;
         border:1px solid var(--border); border-radius:4px;">
      <div style="margin-bottom:6px;">War das Ergebnis brauchbar? Dein Urteil hilft,
        die Qualitätsbewertung an echtem Material zu justieren.</div>
      <div class="row">
        <button data-verdict="brauchbar">brauchbar</button>
        <button data-verdict="grenzwertig">grenzwertig</button>
        <button data-verdict="unbrauchbar">unbrauchbar</button>
      </div>
      <input type="text" id="gen-fb-comment" placeholder="Kommentar (optional) - z.B. was genau nicht gepasst hat"
             style="width:100%; margin-top:8px;" />
      <div id="gen-fb-status" class="hint" style="margin-top:6px;"></div>
    </div>
    <div id="gen-quality" style="display:none; margin-top:8px; padding:8px; border-radius:4px;"></div>

    <p class="hint">
      Klassisches CV-Tracking als Grundlage - im Vorschaubild eine Region über
      das zu verfolgende Motiv ziehen, dann generieren. Optional dabei die lokale
      KI-Regionserkennung nutzen (Häkchen oben) oder ein gemerktes/vorgeschlagenes
      Profil übernehmen (unten) - beides bleibt ein Vorschlag, den du bestätigst
      oder korrigierst.
    </p>
  `;

  const el = id => root.querySelector(id);
  const canvas = el('#roi-canvas');
  const ctx = canvas.getContext('2d');

  let videoPath = null;
  let img = new Image();
  let nativeW = 0, nativeH = 0;
  let roi = null; // {x,y,w,h} in Videopixeln
  let roi2 = null; // zweite Region für Tf/Tj (Abstand + Sog)
  let roi2Mode = false; // Knopf „2. Region“ aktiv
  let dragging = false, draggingSecond = false, startX = 0, startY = 0, curX = 0, curY = 0;

  const DISPLAY_W = 560;

  const ROI1_STROKE = '#5fd0c8';
  const ROI1_FILL = 'rgba(95,208,200,0.15)';
  const ROI2_STROKE = '#8b7cff';
  const ROI2_FILL = 'rgba(139,124,255,0.18)';

  function isTfTj() {
    const p = el('#gen-profile').value;
    return p === 'tf' || p === 'tj';
  }

  // KI-Regionssuche (ai_roi.py, lokales ONNX-Modell) ist optional - ohne
  // installiertes onnxruntime oder ohne Modelldatei bleibt es bei der
  // klassischen Rhythmus-Heuristik (auto_roi.py). Einmal beim Öffnen des
  // Tabs geprüft (kostet einen Python-Start), nicht bei jedem Videoladen.
  CheckAIRoiAvailable().then(available => {
    const checkbox = el('#gen-ai-roi');
    checkbox.disabled = !available;
    el('#gen-autoroi-hint').textContent = available
      ? 'Häkchen "KI-Erkennung" setzt auf ein lokales ONNX-Objekterkennungsmodell statt der '
        + 'Rhythmus-Heuristik. Danach lässt sich die Region trotzdem von Hand korrigieren.'
      : 'Analysiert die Bewegung im Video (klassisch, ohne KI-Modell) - danach lässt sich die '
        + 'Region trotzdem von Hand korrigieren. KI-Erkennung: kein lokales ONNX-Modell '
        + 'gefunden (Einstellungen → KI-Modellpfad, oder Standardordner).';
  }).catch(() => {});

  // Audio-Tempo-Prüfung (audio_check.py) braucht nur ffmpeg auf dem PATH -
  // kein Modell, kein separates Python-Paket. Gleiches Muster wie oben:
  // einmal beim Öffnen des Tabs geprüft, Checkbox ausgegraut statt bei
  // jedem Versuch mit "nicht möglich" zu scheitern.
  CheckAudioCheckAvailable().then(available => {
    const checkbox = el('#gen-audio-check');
    checkbox.disabled = !available;
    if (!available) {
      checkbox.title = 'ffmpeg wurde nicht auf dem PATH gefunden';
    }
  }).catch(() => {});

  function setRoi2Mode(on) {
    roi2Mode = !!on;
    const btn = el('#gen-roi2-toggle');
    btn.style.outline = roi2Mode ? '2px solid #8b7cff' : '';
    btn.style.background = roi2Mode ? 'rgba(139,124,255,0.28)' : '';
  }

  function updateRoiLabels() {
    el('#gen-roi-label').textContent = roi
      ? `Region: x=${roi.x} y=${roi.y} w=${roi.w} h=${roi.h} (Videopixel)`
      : 'Keine Region markiert';
    el('#gen-roi2-label').textContent = roi2
      ? `2. Region: x=${roi2.x} y=${roi2.y} w=${roi2.w} h=${roi2.h} (Videopixel, violett)`
      : 'Keine 2. Region markiert';

    // Zwei-Punkt-Messung (2. Region gesetzt) hat einen eigenen Pfad in
    // generate_funscript.py, der weder die Schnitt-Neuerkennung noch das
    // Flow-Backend kennt - beide unten sichtbar abschalten, statt sie
    // anzubieten und dann stillschweigend zu ignorieren.
    const twoPoint = !!roi2;
    const perScene = el('#gen-perscene');
    perScene.disabled = twoPoint;
    if (twoPoint) perScene.checked = false;
    perScene.title = twoPoint
      ? 'Bei Zwei-Punkt-Messung (2. Region gesetzt) nicht verfügbar - die Region wird dort nicht neu gesucht.'
      : '';
    const flowOption = el('#gen-backend').querySelector('option[value="flow"]');
    flowOption.disabled = twoPoint;
    flowOption.title = twoPoint ? 'Bei Zwei-Punkt-Messung nicht verfügbar (kein Tracker/keine Region).' : '';
    const regionFusionOption = el('#gen-backend').querySelector('option[value="region_fusion"]');
    regionFusionOption.disabled = twoPoint;
    regionFusionOption.title = twoPoint
      ? 'Bei Zwei-Punkt-Messung nicht verfügbar (kein Zwei-Punkt-Pfad für dieses Verfahren) - CSRT wird stattdessen verwendet.'
      : '';
    const regionFusionAutoOption = el('#gen-backend').querySelector('option[value="region_fusion_auto"]');
    regionFusionAutoOption.disabled = twoPoint;
    regionFusionAutoOption.title = twoPoint
      ? 'Bei Zwei-Punkt-Messung nicht verfügbar (kein Zwei-Punkt-Pfad für dieses Verfahren) - CSRT wird stattdessen verwendet.'
      : '';
    if (twoPoint && ['flow', 'region_fusion', 'region_fusion_auto'].includes(el('#gen-backend').value)) {
      el('#gen-backend').value = 'csrt';
    }
    updateGenerateEnabled();
  }

  // Nur csrt/grid_lk/region_fusion/two_point (Tf/Tj) brauchen eine von Hand
  // markierte Region - flow und region_fusion_auto bestimmen ihre Zonen
  // selbst aus dem ganzen Bild, siehe generate_funscript.py/backends.py.
  function backendNeedsRoi() {
    return !['flow', 'region_fusion_auto'].includes(el('#gen-backend').value);
  }

  function updateGenerateEnabled() {
    // Ohne Video kein Ziel zum Generieren - bisher deckte "kein roi" das
    // implizit mit ab (roi startet null), das gilt seit backendNeedsRoi()
    // für flow/region_fusion_auto nicht mehr automatisch.
    if (!videoPath) {
      el('#gen-generate').disabled = true;
      return;
    }
    if (backendNeedsRoi() && !roi) {
      el('#gen-generate').disabled = true;
      return;
    }
    if (isTfTj() && !roi2) {
      el('#gen-generate').disabled = true;
      return;
    }
    el('#gen-generate').disabled = false;
  }

  function updateProfileUi() {
    const tftj = isTfTj();
    el('#gen-tftj-hint').style.display = tftj ? 'block' : 'none';
    el('#gen-contact-vibration-row').style.display = tftj ? 'flex' : 'none';
    if (tftj) setRoi2Mode(true);
    updateGenerateEnabled();
    if (tftj && videoPath) {
      el('#gen-status').textContent = roi2
        ? 'Tf/Tj: beide Regionen gesetzt — generieren möglich.'
        : 'Tf/Tj (Abstand + Sog): zweite Region markieren (Shift+Ziehen oder „2. Region“).';
    }
  }

  function drawNativeRect(r, stroke, fill) {
    if (!r || !nativeW || !nativeH) return;
    const scaleX = canvas.width / nativeW, scaleY = canvas.height / nativeH;
    const dx0 = r.x * scaleX, dy0 = r.y * scaleY, dw = r.w * scaleX, dh = r.h * scaleY;
    ctx.strokeStyle = stroke;
    ctx.lineWidth = 2;
    ctx.strokeRect(dx0, dy0, dw, dh);
    ctx.fillStyle = fill;
    ctx.fillRect(dx0, dy0, dw, dh);
  }

  function drawDragRect(stroke, fill) {
    const dx0 = Math.min(startX, curX), dy0 = Math.min(startY, curY);
    const dw = Math.abs(curX - startX), dh = Math.abs(curY - startY);
    ctx.strokeStyle = stroke;
    ctx.lineWidth = 2;
    ctx.strokeRect(dx0, dy0, dw, dh);
    ctx.fillStyle = fill;
    ctx.fillRect(dx0, dy0, dw, dh);
  }

  function redraw() {
    ctx.clearRect(0, 0, canvas.width, canvas.height);
    if (img.src) ctx.drawImage(img, 0, 0, canvas.width, canvas.height);
    if (roi && !(dragging && !draggingSecond)) drawNativeRect(roi, ROI1_STROKE, ROI1_FILL);
    if (roi2 && !(dragging && draggingSecond)) drawNativeRect(roi2, ROI2_STROKE, ROI2_FILL);
    if (dragging) {
      if (draggingSecond) drawDragRect(ROI2_STROKE, ROI2_FILL);
      else drawDragRect(ROI1_STROKE, ROI1_FILL);
    }
  }

  canvas.addEventListener('mousedown', e => {
    const r = canvas.getBoundingClientRect();
    startX = curX = e.clientX - r.left;
    startY = curY = e.clientY - r.top;
    dragging = true;
    draggingSecond = roi2Mode || e.shiftKey;
  });
  canvas.addEventListener('mousemove', e => {
    if (!dragging) return;
    const r = canvas.getBoundingClientRect();
    curX = e.clientX - r.left;
    curY = e.clientY - r.top;
    redraw();
  });
  window.addEventListener('mouseup', () => {
    if (!dragging) return;
    dragging = false;
    const wasSecond = draggingSecond;
    draggingSecond = false;
    const w = Math.abs(curX - startX), h = Math.abs(curY - startY);
    if (w < 4 || h < 4) { redraw(); return; }
    const scaleX = nativeW / canvas.width, scaleY = nativeH / canvas.height;
    const x0 = Math.min(startX, curX), y0 = Math.min(startY, curY);
    const box = {
      x: Math.round(x0 * scaleX), y: Math.round(y0 * scaleY),
      w: Math.round(w * scaleX), h: Math.round(h * scaleY),
    };
    if (wasSecond) {
      roi2 = box;
      setRoi2Mode(false);
      // Eine 2. Region ergibt nur im Tf/Tj-Modus (Abstand + Sog) einen Sinn
      // - kein anderes Verfahren wertet sie aus (siehe generate_funscript.py).
      // Automatisch erkennen statt den Nutzer zusätzlich noch die Dropdown
      // umstellen zu lassen: das Markieren der 2. Region IST die Auswahl.
      if (!isTfTj()) {
        el('#gen-profile').value = 'tf';
        updateProfileUi();
      }
    } else {
      roi = box;
      if (isTfTj() && !roi2) setRoi2Mode(true);
    }
    updateRoiLabels();
    updateGenerateEnabled();
    if (isTfTj() && videoPath && !roi2) {
      el('#gen-status').textContent = 'Erste Region gesetzt — jetzt 2. Region markieren (Shift+Ziehen oder „2. Region“).';
    }
    redraw();
  });

  // Fallengelassenes Video übernehmen. Teilt sich den Ladeweg mit der
  // Dateiauswahl, damit beide Wege garantiert dasselbe tun.
  window.addEventListener('drop:video', e => loadVideo(e.detail.path, e.detail.extraCount || 0));

  async function chooseVideo() {
    const path = await PickVideoFile();
    // Fokus zurück ins Fenster - siehe gleichnamige Behandlung in
    // playback.js: nach dem nativen Datei-Dialog bleibt der Tastaturfokus
    // sonst am Button hängen.
    if (document.activeElement && document.activeElement.blur) {
      document.activeElement.blur();
    }
    window.focus();
    if (!path) return;
    await loadVideo(path);
  }

  async function loadVideo(path, extraCount = 0) {
    videoPath = path;
    el('#gen-video-path').textContent = path.split(/[\\/]/).pop();
    el('#gen-status').textContent = 'Lade Vorschau-Frame...';
    roi = null;
    roi2 = null;
    setRoi2Mode(isTfTj());
    el('#gen-generate').disabled = true;
    updateRoiLabels();
    // Stapelverarbeitung mehrerer Videos gibt es noch nicht - vorher wurden
    // weitere abgelegte Videos einfach stillschweigend verworfen, ohne dass
    // sichtbar war, dass überhaupt mehr als eins ankam.
    const batchNote = extraCount > 0
      ? ` (${extraCount} weitere${extraCount === 1 ? 's' : ''} abgelegte${extraCount === 1 ? 's' : ''} Video${extraCount === 1 ? '' : 's'} ignoriert - Stapelverarbeitung gibt es noch nicht)`
      : '';
    try {
      const preview = await LoadFirstFrame(path);
      nativeW = preview.width; nativeH = preview.height;
      const displayH = Math.round(DISPLAY_W * nativeH / nativeW);
      canvas.width = DISPLAY_W; canvas.height = displayH;
      img.onload = redraw;
      img.src = 'data:image/png;base64,' + preview.pngBase64;
      el('#gen-autoroi').disabled = false;
      el('#gen-suggest-profile').disabled = false;
      el('#gen-label-scene').disabled = false;
      el('#gen-suggest-status').textContent = '';
      el('#gen-status').textContent = (isTfTj()
        ? 'Tf/Tj (Abstand + Sog): erste Region ziehen, dann Shift+Ziehen oder „2. Region“ für die zweite.'
        : 'Region automatisch finden lassen oder von Hand markieren (Maus ziehen).') + batchNote;
    } catch (err) {
      el('#gen-status').textContent = '';
      alert('Fehler: ' + err);
    }
  }

  async function checkDeps() {
    try {
      await CheckGeneratorDependencies();
      alert('Python und alle benötigten Pakete sind verfügbar.');
    } catch (err) {
      alert('Fehler: ' + err);
    }
  }

  async function generate() {
    if (!videoPath || (backendNeedsRoi() && !roi)) return;
    if (isTfTj() && !roi2) {
      alert('Tf/Tj (Abstand + Sog) braucht eine zweite Region. Shift+Ziehen oder Knopf „2. Region“.');
      return;
    }

    // Vorhandenes Skript nicht kommentarlos überschreiben - der Nutzer
    // könnte ein von Hand erstelltes oder heruntergeladenes Skript neben
    // dem Video liegen haben.
    let overwrite = false;
    try {
      if (await ScriptExistsForVideo(videoPath)) {
        const target = videoPath.replace(/\.[^.\\/]+$/, '') + '.funscript';
        if (!confirm(`Es existiert bereits ein Skript:\n${target}\n\nWirklich überschreiben?`)) {
          return;
        }
        overwrite = true;
      }
    } catch (err) {
      // Prüfung fehlgeschlagen - lieber nicht überschreiben, das Backend
      // lehnt dann ohnehin ab und meldet es sauber.
    }

    el('#gen-generate').disabled = true;
    el('#gen-status').textContent = 'Generiere...';
    // Ohne markierte Region (flow/region_fusion_auto) dieselbe "keine ROI"-
    // Platzhalter-Region wie die CLI ohne --roi fürs flow-Backend verschickt
    // (0,0,0,0) - beide Backends ignorieren sie ohnehin vollständig.
    const effectiveRoi = roi || { x: 0, y: 0, w: 0, h: 0 };
    const payload = {
      videoPath,
      x: effectiveRoi.x, y: effectiveRoi.y, w: effectiveRoi.w, h: effectiveRoi.h,
      invert: el('#gen-invert').checked,
      smoothWindow: parseInt(el('#gen-smooth').value, 10) || 11,
      minPeakDistanceMs: parseInt(el('#gen-peakdist').value, 10) || 150,
      disableCameraCompensation: !el('#gen-camcomp').checked,
      disableSceneCutDetection: !el('#gen-scenecut').checked,
      perSceneRoi: el('#gen-perscene').checked,
      adaptiveKeyframeError: el('#gen-adaptive').checked ? 6 : 0,
      autoRetry: el('#gen-retry').checked,
      backend: el('#gen-backend').value,
      useOpenCl: el('#gen-opencl').checked,
      dynamicRangeMs: el('#gen-dynrange').checked ? 3000 : 0,
      profile: el('#gen-profile').value,
      axis: el('#gen-axis').value,
      rdpTolerance: parseFloat(el('#gen-rdp').value) || 0,
      overwrite,
      aiQualityOpinion: el('#gen-ai-quality').checked,
      contactVibration: isTfTj() && el('#gen-contact-vibration').checked,
      autoOZoneMarker: el('#gen-auto-ozone').checked,
      audioCheck: el('#gen-audio-check').checked,
      nativePipeline: el('#gen-native').checked,
    };
    if (roi2) {
      payload.x2 = roi2.x;
      payload.y2 = roi2.y;
      payload.w2 = roi2.w;
      payload.h2 = roi2.h;
    }
    GenerateScript(payload);
  }

  EventsOn('generate:progress', line => { el('#gen-status').textContent = line; });

  // Ergebnis der automatischen Regionssuche übernehmen - die ROI wird
  // genauso gesetzt, als hätte der Nutzer sie gezogen, und lässt sich
  // danach frei korrigieren.
  EventsOn('generate:autoroi', result => {
    hideProgress();
    el('#gen-autoroi').disabled = false;
    if (result.error) {
      el('#gen-status').textContent = 'Automatische Suche fehlgeschlagen.';
      alert('Fehler: ' + result.error);
      return;
    }
    roi = { x: result.x, y: result.y, w: result.w, h: result.h };
    updateRoiLabels();
    const via = result.engine === 'ai' ? 'KI-Erkennung' : 'klassisch, automatisch';
    if (roi) {
      el('#gen-roi-label').textContent =
        `Region: x=${roi.x} y=${roi.y} w=${roi.w} h=${roi.h} (Videopixel, ${via} gefunden)`;
    }
    updateGenerateEnabled();
    el('#gen-status').textContent = isTfTj() && !roi2
      ? `Region gefunden (${via}) — für Tf/Tj noch die 2. Region markieren (Shift+Ziehen oder „2. Region“).`
      : `Region gefunden (${via}) - bei Bedarf von Hand korrigieren.`;
    redraw();
  });
  // Fortschritt: das Backend schickt 0-100, oder -1 wenn die Frame-Anzahl
  // des Videos nicht ermittelbar war. In dem Fall wird ein unbestimmter
  // Balken gezeigt statt eines erfundenen Prozentwerts.
  let progressStartedAt = 0;
  EventsOn('generate:percent', pct => {
    const wrap = el('#gen-progress-wrap');
    const bar = el('#gen-progress-bar');
    const text = el('#gen-progress-text');
    if (wrap.style.display === 'none') {
      wrap.style.display = 'block';
      progressStartedAt = Date.now();
    }
    if (pct < 0) {
      bar.style.width = '100%';
      bar.style.opacity = '0.35';
      text.textContent = 'Läuft... (Gesamtlänge des Videos nicht ermittelbar)';
      return;
    }
    bar.style.opacity = '1';
    bar.style.width = pct + '%';
    const elapsed = (Date.now() - progressStartedAt) / 1000;
    let rest = '';
    if (pct >= 3 && elapsed > 2) {
      const total = elapsed / (pct / 100);
      const remaining = Math.max(0, Math.round(total - elapsed));
      rest = `  ·  noch ca. ${remaining < 60 ? remaining + ' s' : Math.round(remaining / 60) + ' min'}`;
    }
    text.textContent = `${pct} %${rest}`;
  });

  function hideProgress() {
    el('#gen-progress-wrap').style.display = 'none';
    el('#gen-progress-bar').style.width = '0%';
    el('#gen-progress-text').textContent = '';
  }

  let lastOutputPath = null;

  // Urteil abschicken. Der Bereich erscheint erst nach einem erfolgreichen
  // Lauf - vorher gibt es nichts zu beurteilen.
  root.querySelectorAll('#gen-feedback button[data-verdict]').forEach(btn => {
    btn.addEventListener('click', async () => {
      if (!lastOutputPath) return;
      const status = el('#gen-fb-status');
      try {
        await SubmitFeedback({
          outputPath: lastOutputPath,
          verdict: btn.dataset.verdict,
          comment: el('#gen-fb-comment').value || '',
        });
        status.textContent = `Danke - als "${btn.dataset.verdict}" gespeichert.`;
        el('#gen-fb-comment').value = '';
      } catch (err) {
        status.textContent = 'Konnte nicht gespeichert werden: ' + err;
      }
    });
  });

  EventsOn('generate:done', result => {
    hideProgress();
    lastOutputPath = result.path || null;
    el('#gen-fb-status').textContent = '';
    el('#gen-feedback').style.display = lastOutputPath ? 'block' : 'none';
    updateGenerateEnabled();
    if (result.error) {
      el('#gen-status').textContent = 'Fehlgeschlagen.';
      alert('Fehler: ' + result.error);
      return;
    }
    el('#gen-status').textContent = 'Fertig: ' + result.path;
    if (typeof result.oZoneMarkerStartMs === 'number') {
      const s = Math.round(result.oZoneMarkerStartMs / 1000);
      const e = Math.round(result.oZoneMarkerEndMs / 1000);
      el('#gen-status').textContent += ` — O-Marker gesetzt: ${s}s–${e}s`;
    }

    const qualityBox = el('#gen-quality');
    if (typeof result.qualityScore === 'number') {
      const pct = Math.round(result.qualityScore * 100);
      // Das Urteil kommt vom Quality Doctor selbst, nicht aus dem Score: er
      // kennt harte Ausschlusskriterien, die trotz Score >= 0.5 zum Scheitern
      // führen. Nur bei älteren Skripten ohne das Feld auf den Score zurückfallen.
      const ok = typeof result.qualityPassed === 'boolean'
        ? result.qualityPassed
        : result.qualityScore >= 0.5;
      qualityBox.style.display = 'block';
      qualityBox.style.background = ok ? 'rgba(61,216,117,0.12)' : 'rgba(216,77,77,0.12)';
      qualityBox.style.border = `1px solid ${ok ? 'var(--ok)' : 'var(--danger)'}`;
      let html = `<b>Quality Doctor: ${pct}% ${ok ? '(unauffällig)' : '(bitte prüfen)'}</b>`;
      if (result.qualityWarnings && result.qualityWarnings.length > 0) {
        html += '<ul style="margin:6px 0 0 18px; padding:0;">' +
          result.qualityWarnings.map(w => `<li>${w}</li>`).join('') + '</ul>';
      }
      if (result.aiOpinionVerdict) {
        html += `<div style="margin-top:8px; padding-top:8px; border-top:1px solid var(--border);">`
          + `<b>KI-Zweitmeinung: ${result.aiOpinionVerdict}</b>`
          + (result.aiOpinionReason ? `<br>${result.aiOpinionReason}` : '')
          + `</div>`;
      }
      if (result.audioCheckWarnings && result.audioCheckWarnings.length > 0) {
        html += `<div style="margin-top:8px; padding-top:8px; border-top:1px solid var(--border);">`
          + `<b>Audio-Tempo-Prüfung:</b>`
          + '<ul style="margin:6px 0 0 18px; padding:0;">'
          + result.audioCheckWarnings.map(w => `<li>${w}</li>`).join('') + '</ul>'
          + `</div>`;
      }
      qualityBox.innerHTML = html;
    } else {
      qualityBox.style.display = 'none';
    }

    if (confirm(`${result.path}\n\nJetzt im Wiedergabe-Tab laden?`)) {
      playback.loadScriptPath(result.path);
    }
  });

  el('#gen-choose').addEventListener('click', chooseVideo);
  el('#gen-check-deps').addEventListener('click', checkDeps);
  el('#gen-generate').addEventListener('click', generate);
  el('#gen-roi2-toggle').addEventListener('click', () => {
    setRoi2Mode(!roi2Mode);
    if (roi2Mode) {
      el('#gen-status').textContent = '2. Region: Bereich im Vorschaubild ziehen (wird violett).';
    }
  });
  el('#gen-profile').addEventListener('change', updateProfileUi);
  el('#gen-backend').addEventListener('change', updateGenerateEnabled);
  el('#gen-autoroi').addEventListener('click', () => {
    if (!videoPath) return;
    const useAI = el('#gen-ai-roi').checked && !el('#gen-ai-roi').disabled;
    el('#gen-autoroi').disabled = true;
    el('#gen-status').textContent = useAI
      ? 'KI-Regionssuche läuft (ONNX-Modell)...'
      : 'Analysiere Bewegung im Video (dauert einige Sekunden)...';
    AutoDetectROI(videoPath, useAI ? 'ai' : 'auto');
  });

  const PROFILE_VALUES = ['standard', 'weich', 'tf'];

  el('#gen-suggest-profile').addEventListener('click', async () => {
    if (!videoPath) return;
    const status = el('#gen-suggest-status');
    status.textContent = 'Vergleiche mit gemerkten Szenen...';
    el('#gen-suggest-profile').disabled = true;
    try {
      const result = await SuggestProfile(videoPath);
      if (!result.found) {
        status.textContent = 'Kein Vorschlag (keine ähnliche gemerkte Szene, kein KI-Server erreichbar).';
        return;
      }
      const via = result.kind === 'ai'
        ? `KI, Konfidenz ${Math.round(result.confidence * 100)}%`
        : `gemessen, Abstand ${result.confidence.toFixed(3)}`;
      // "tj" war früher ein zweiter, identisch behandelter Dropdown-Eintrag
      // (siehe funscript/recipe.go NormalizeProfile) - inzwischen zu einem
      // Eintrag "tf" zusammengelegt. Ältere, lokal gemerkte Szenen können
      // noch mit "tj" beschriftet sein; hier auf den verbliebenen Wert
      // abbilden, statt beim Übernehmen an einer verschwundenen Option
      // stillschweigend hängenzubleiben.
      const label = result.label === 'tj' ? 'tf' : result.label;
      if (PROFILE_VALUES.includes(label)) {
        status.textContent = `Vorschlag: "${label}" (${via}) — `;
        const applyBtn = document.createElement('button');
        applyBtn.textContent = 'übernehmen';
        applyBtn.addEventListener('click', () => {
          el('#gen-profile').value = label;
          updateProfileUi();
          status.textContent = `Profil "${label}" übernommen (${via}).`;
        });
        status.appendChild(applyBtn);
      } else {
        status.textContent = `Ähnlich zu gemerkter Szene "${result.label}" (${via}) - kein `
          + 'direkter Profilname, keine automatische Übernahme.';
      }
    } catch (err) {
      status.textContent = 'Fehler: ' + err;
    } finally {
      el('#gen-suggest-profile').disabled = false;
    }
  });

  el('#gen-label-scene').addEventListener('click', async () => {
    if (!videoPath) return;
    const label = el('#gen-scene-label').value.trim();
    if (!label) {
      alert('Bitte einen Namen für die Szene eingeben.');
      return;
    }
    el('#gen-label-scene').disabled = true;
    try {
      await LabelScene(videoPath, label);
      el('#gen-suggest-status').textContent = `Szene als "${label}" gemerkt.`;
    } catch (err) {
      alert('Fehler: ' + err);
    } finally {
      el('#gen-label-scene').disabled = false;
    }
  });
}
