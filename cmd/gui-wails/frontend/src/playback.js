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
      1–9 springen · + / − Offset · L Wiederholung · E Extended-O · O O-Marker setzen</p>
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
    <p class="hint">Tastenkürzel: Leertaste = Abspielen/Stop, E = Extended-O auslösen (wenn aktiv), O = O-Marker an aktueller Position setzen.</p>
    <div id="pb-log"></div>
  `;
