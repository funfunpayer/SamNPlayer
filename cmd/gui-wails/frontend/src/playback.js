import {
  PickFunscriptFile, LoadFunscript, StartPlayback, StopPlayback,
  TriggerExtendedO, VideoFileURL, GetHeatmap, GetScriptCurve, GetVibrationCurvePreview, AnalyzeScript, SetScriptOffset, GetScriptOffset, GetMarker, SaveMarker,
  ReportVideoPosition, ReportVideoSyncIdle, GetOMarkers, SaveOMarkers, GetSpeedHighlights,
  GetScriptAxisActions, SaveScriptAxisActions, SetPlaybackSource,
  GetStrengthPresets, SetActiveStrength, ExportLoadedFunscript, SaveLoadedAsSamn, BakeNeoAxesOnLoaded,
  OptimizeLoadedForNeo2,
  ExportScriptHeatmapPNG, SavePlaybackProject, LoadPlaybackProject, PickPlaybackProject,
  EditCapSpeedRange, EditDeleteRange, EditScaleRange, SnapTimeMs,
  ScriptChapters, ScriptQuality,
  SaveContactSettings, PreviewContactVibration, PickVideoFile, SetPlaybackVideo, ClearPlaybackVideo,
  ProbePlaybackVideo, EnsurePlayablePlaybackVideo, GetTrajectory,
  GetScriptBookmarks, SaveScriptBookmarks,
  GetScriptChapterMarks, SaveScriptChapterMarks,
} from '../wailsjs/go/main/App';
import { EventsOn } from '../wailsjs/runtime/runtime';
import { getSettingsCache, saveSetting } from './settings.js';
import { applyHotkeyOMarker } from './ozone_ui.js';
import { wireDataHelp } from './help.js';
import { uiError, uiInfo, uiWarn } from './notify.js';

const HEATMAP_BUCKETS = 300;

export function initPlayback(root) {
  root.classList.add('tab-playback');
  root.innerHTML = `
    <div class="pb-head">
      <div class="pb-head-text">
        <h2>Play</h2>
        <span class="path-label" id="pb-script-path">No Emotion Script selected</span>
      </div>
      <div class="pb-head-actions">
        <button id="pb-choose" class="primary" type="button">Choose Emotion Script…</button>
        <button id="pb-queue-add" type="button" title="Add another Emotion Script to the list" hidden>Add to list</button>
      </div>
    </div>

    <div class="pb-empty" id="pb-empty">
      <div class="pb-empty-inner">
        <div class="pb-brand-moment" aria-hidden="true">
          <img class="pb-brand-mark" src="/src/assets/images/logo-universal.png" alt="" />
          <p class="pb-brand-word">
            <span class="wm-sam">Sam</span><span class="wm-n">N</span><span class="wm-rest">Player</span>
          </p>
        </div>
        <p class="pb-empty-kicker">Play</p>
        <p class="pb-empty-title">Ready when you are</p>
        <p class="hint">Open an Emotion Script — with a film or without. Feel still reaches the device. Add more for a quiet list. Other apps’ files are welcome too.</p>
        <button type="button" id="pb-choose-empty" class="primary">Choose Emotion Script…</button>
      </div>
    </div>

    <div class="pb-loaded" id="pb-loaded" hidden>
      <div class="pb-media">
        <div class="pb-video-stage" id="pb-video-stage">
          <video id="pb-video" controls playsinline></video>
          <canvas id="pb-trajectory" class="pb-trajectory-canvas" hidden aria-hidden="true"></canvas>
          <div id="pb-pos-overlay" class="pos-gauge" hidden aria-hidden="true">
            <div class="pos-gauge-scale" aria-hidden="true">
              <span>100</span><span>50</span><span>0</span>
            </div>
            <div class="pos-gauge-track">
              <div class="pos-gauge-fill" id="pb-pos-fill"></div>
              <div class="pos-gauge-knob" id="pb-pos-knob"></div>
            </div>
            <div class="pos-gauge-value" id="pb-pos-value">—</div>
          </div>
          <div class="pb-video-chrome" id="pb-video-chrome">
            <button type="button" id="pb-video-fs" title="Fullscreen (double-click)">Fullscreen</button>
            <button type="button" id="pb-video-change" title="Choose another video">Video…</button>
            <button type="button" id="pb-video-convert" title="Convert to H.264/AAC MP4 (ffmpeg)" hidden>Make playable</button>
          </div>
          <div id="pb-video-warn" class="pb-video-warn" hidden></div>
          <p class="pb-keys-hint hint" id="pb-keys-hint">Space play/stop · ←/→ seek · ,/. fine · +/- offset · E Extended-O · Esc leave fullscreen</p>
          <div class="checkbox-row" id="pb-contact-marks-row" style="display:none; margin:0.35rem 0 0;">
            <input type="checkbox" id="pb-contact-marks-toggle" checked />
            <label for="pb-contact-marks-toggle"
              data-help="Draws contact areas stored at Create (gold = primary, magenta = extras). Feel labels only — Everyday stroke still follows tip CSRT depth.">Show contact marks</label>
          </div>
          <p class="hint" id="pb-contact-marks-hint" style="display:none; margin:0.2rem 0 0;"></p>
          <div id="pb-novideo" class="pb-novideo">
            <p class="pb-novideo-title">Feel without film</p>
            <p class="hint">Device and curve alone — video is optional. Play starts immediately.</p>
            <div class="pb-novideo-actions">
              <button type="button" id="pb-play-novideo" class="primary">Play</button>
              <button type="button" id="pb-pick-video">Link video…</button>
            </div>
          </div>
        </div>
        <canvas id="pb-curve" height="176" class="pb-curve" style="display:none"></canvas>
        <canvas id="pb-heatmap" height="56" class="pb-heatmap" style="display:none"></canvas>
        <div class="row pb-curve-zoom" id="pb-curve-zoom" style="display:none; align-items:center; flex-wrap:wrap; gap:8px; margin-top:4px;">
          <button type="button" id="pb-zoom-in" title="Zoom in on the curve / heatmap window">Zoom in</button>
          <button type="button" id="pb-zoom-out" title="Zoom out">Zoom out</button>
          <button type="button" id="pb-zoom-sel" title="Zoom to the current heatmap selection" disabled>Zoom selection</button>
          <button type="button" id="pb-zoom-reset" title="Show the full script">Reset zoom</button>
          <span class="hint" id="pb-zoom-label" style="margin:0">Full</span>
        </div>
        <div id="pb-chart-tooltip" class="pb-chart-tooltip" hidden></div>
        <div class="pb-transport">
          <div class="row pb-transport-btns">
            <button id="pb-play" class="primary pb-btn-icon" type="button" title="Play">
              <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M8 5v14l11-7z"/></svg>
              <span>Play</span>
            </button>
            <button id="pb-stop" class="pb-btn-icon" type="button" disabled title="Stop">
              <svg viewBox="0 0 24 24" aria-hidden="true"><rect x="6" y="6" width="12" height="12"/></svg>
              <span>Stop</span>
            </button>
            <button id="pb-eo-trigger" type="button" disabled>Extended-O</button>
            <button id="pb-next" type="button" hidden title="Next item in the list">Next</button>
          </div>
          <div class="progress-bar"><div class="progress-bar-fill" id="pb-progress"></div></div>
          <div class="stat-row pb-live">
            <span>Vibration <b id="pb-vib">-</b></span>
            <span>Suction <b id="pb-suc">-</b></span>
          </div>
        </div>
        <div class="pb-playlist" id="pb-playlist" hidden>
          <div class="pb-playlist-head">
            <span class="pb-playlist-title">Playlist</span>
            <div class="pb-playlist-opts">
              <label class="checkbox-row pb-playlist-auto" style="margin:0">
                <input type="checkbox" id="pb-playlist-auto" checked />
                <span>Auto-advance</span>
              </label>
              <label class="checkbox-row pb-playlist-shuffle" style="margin:0">
                <input type="checkbox" id="pb-playlist-shuffle" />
                <span>Shuffle</span>
              </label>
              <label class="checkbox-row pb-playlist-repeat" style="margin:0">
                <input type="checkbox" id="pb-playlist-repeat" />
                <span>Repeat playlist</span>
              </label>
            </div>
          </div>
          <ol class="pb-playlist-list" id="pb-playlist-list"></ol>
        </div>
      </div>

      <div class="pb-tools">
        <div class="checkbox-row" id="pb-curve-edit-row" style="display:none">
          <input type="checkbox" id="pb-curve-edit" />
          <label for="pb-curve-edit">Edit curve (dots)</label>
        </div>
        <label class="checkbox-row" id="pb-pos-overlay-row" style="display:none; margin:0;"
          data-help="0–100 stroke gauge over the video. Follows playhead. Turn off anytime.">
          <input type="checkbox" id="pb-pos-overlay-toggle" checked />
          0–100 on video
        </label>
        <div class="row" id="pb-axis-row" style="display:none; align-items:center; gap:8px; flex-wrap:wrap;">
          <label style="width:auto;" data-help="Motion = main stroke. Vibration / Suction = Neo 2 feel channels on Emotion Scripts.">Curve</label>
          <select id="pb-axis" style="width:auto;">
            <option value="general">General</option>
            <option value="vibration">Vibration</option>
            <option value="suction">Suction</option>
          </select>
          <label style="width:auto;" data-help="Recipe derives Neo 2 from general. Axes plays explicit vibration/suction curves.">Drive</label>
          <select id="pb-playback-source" style="width:auto;">
            <option value="recipe">Recipe</option>
            <option value="axes">Axes</option>
          </select>
          <label style="width:auto;" data-help="Soft / normal / strong feel scales on Emotion Scripts.">Strength</label>
          <select id="pb-strength" style="width:auto;">
            <option value="">—</option>
          </select>
          <button type="button" id="pb-optimize-neo2" class="primary"
            title="Imported file → polish, Contact on, bake Neo 2 feel, save as Emotion Script."
            data-help="One click for files from other apps: polish gaps, enable Contact vibration, bake vibe+suction for Sam Neo 2, save as Emotion Script. Then edit each curve if you want.">Optimize for Neo 2</button>
          <button type="button" id="pb-save-samn" title="Save Emotion Script">Save Emotion Script</button>
        </div>
        <details class="pb-tools-details" id="pb-export-bake-details" style="display:none;">
          <summary>Bake &amp; share</summary>
          <div class="row" style="align-items:center; gap:8px; flex-wrap:wrap; margin-top:6px;">
            <button type="button" id="pb-bake-axes" title="Bake vibration/suction into this Emotion Script">Bake feel channels</button>
            <button type="button" id="pb-export-funscript" title="Share stroke for other apps (.funscript)">Share for other apps</button>
          </div>
        </details>
        <p class="hint" id="pb-optimize-neo2-status" style="display:none; margin:4px 0 0 0;"></p>
        <p class="hint" id="pb-curve-edit-hint" style="display:none; margin-top:0;"
          data-help="Soft curve + keyframe dots. Click+drag = move. Click empty = new. Double-click = delete (keep ≥2). Saves immediately.">
          Edit dots on the curve: drag / click / double-click — see “?”.</p>

        <div class="row" id="pb-offset-row" style="display:none; align-items:center;">
          <label style="width:auto;" data-help="Positive value = script applies later. Takes effect immediately, including during playback. Saved per script.">Script offset</label>
          <button id="pb-offset-minus" title="Script 50 ms earlier (− key)">−50</button>
          <input type="number" id="pb-offset" value="0" step="10" style="width:90px;" />
          <span class="hint" style="margin:0;">ms</span>
          <button id="pb-offset-plus" title="Script 50 ms later (+ key)">+50</button>
          <button id="pb-offset-reset">Reset</button>
          <span class="checkbox-row" style="margin:0 0 0 12px;">
            <input type="checkbox" id="pb-loop" />
            <label for="pb-loop" style="width:auto;" data-help="Repeats the marked section (drag on heatmap).">Loop selection</label>
          </span>
        </div>
        <p class="hint" id="pb-offset-hint" style="display:none; margin-top:0;"
          data-help="Space play/stop · ←/→ 5 s (Shift 1 s) · ,/. fine step · 1–9 jump · +/− offset · L loop · E Extended-O · O O-marker 4s">
          Keyboard help via “?”.</p>

        <details class="pb-tools-details" id="pb-quality-details" style="display:none;">
          <summary>Script check &amp; analysis</summary>
          <div id="pb-analysis" class="hint" style="display:none; margin-top:6px;"></div>
          <div class="row" id="pb-script-doctor-row" style="display:none; align-items:center; margin-top:6px;">
            <button id="pb-script-doctor" type="button">Check script</button>
            <span class="hint" id="pb-script-doctor-status" style="margin:0"></span>
          </div>
          <div id="pb-script-doctor-result" class="hint" style="display:none; margin-top:6px; padding:8px; border-radius:4px;"></div>
        </details>

        <div id="pb-ofs-row" style="display:none; margin-top:8px;">
          <details class="pb-tools-details" id="pb-project-details">
            <summary>Project &amp; heatmap export</summary>
            <div class="row" style="flex-wrap:wrap; gap:8px; margin-top:6px; align-items:center;">
              <button type="button" id="pb-heatmap-export" title="Intensity heatmap as PNG (chapters as ticks)">Heatmap PNG</button>
              <button type="button" id="pb-project-save" title="Save video+script+offset as .snp.json">Save project</button>
              <button type="button" id="pb-project-load" title="Open a .snp.json project (script, video, offset, seek, loop)">Load project</button>
              <span class="hint" id="pb-ofs-status" style="margin:0"></span>
            </div>
          </details>
        </div>
        <details class="pb-tools-details" id="pb-ofs-edit-details" style="display:none;">
          <summary>Curve edit (Cap · Scale · Speed HL · BPM)</summary>
          <div class="row pb-ofs-edit-row" style="flex-wrap:wrap; gap:8px; margin-top:6px; align-items:center;">
            <label class="hint" style="margin:0; display:inline-flex; align-items:center; gap:6px;"
              data-help="Max community intensity (500×|Δpos|/|Δt|) for Speed-cap on the heatmap selection. Lower = more stretch / calmer. Play edit only — does not change Create Expert max-speed.">
              Cap
              <input type="range" id="pb-cap-intensity" min="150" max="800" step="25" value="400" style="width:7em;" />
              <span id="pb-cap-intensity-val">400</span>
            </label>
            <button type="button" id="pb-cap-speed" title="Time-stretch segments that are too fast in the heatmap selection (active Curve axis)">Speed-cap selection</button>
            <label class="checkbox-row" style="margin:0;"
              data-help="Paint red “too fast” bands on the Play curve (OFS community intensity). Display only — separate from Speed-cap edit.">
              <input type="checkbox" id="pb-speed-hl" checked /> Speed highlights
            </label>
            <label class="hint" style="margin:0; display:inline-flex; align-items:center; gap:6px;"
              data-help="Intensity threshold for Speed highlights on the curve (500×|Δpos|/|Δt|). Lower = more red bands. Does not change Cap edit or Create defaults.">
              HL
              <input type="range" id="pb-speed-hl-thresh" min="150" max="800" step="25" value="400" style="width:7em;" />
              <span id="pb-speed-hl-thresh-val">400</span>
            </label>
            <label class="hint" style="margin:0; display:inline-flex; align-items:center; gap:6px;"
              data-help="Scale factor for the heatmap selection around 50. Applies to the active Curve axis (General / Vibration / Suction).">
              Scale
              <input type="range" id="pb-scale-factor" min="0.5" max="1.5" step="0.05" value="0.8" style="width:7em;" />
              <span id="pb-scale-factor-val">×0.80</span>
            </label>
            <button type="button" id="pb-scale-range" title="Scale positions in the heatmap selection around 50 (active Curve axis)">Scale selection</button>
            <label class="checkbox-row" style="margin:0;"
              data-help="Ramp scale from ×1 at the selection edges to the chosen factor at the midpoint — gentler on Vibration / Suction.">
              <input type="checkbox" id="pb-scale-soft-edges" /> Soft edges
            </label>
            <button type="button" id="pb-del-range" title="Delete points in the heatmap selection (active Curve axis)">Delete range</button>
            <label class="hint" style="margin:0; display:inline-flex; align-items:center; gap:6px;"
              data-help="If >0: seeks and new curve points snap to that frame rate. 0 = off. Play edit only — does not change Create.">
              FPS-Snap
              <input type="range" id="pb-fps-snap" min="0" max="60" step="1" value="0" style="width:7em;" />
              <span id="pb-fps-snap-val">off</span>
            </label>
            <label class="checkbox-row" style="margin:0;"
              data-help="Draw beat lines on the Play curve from BPM (or audio tempo when BPM is blank). Display only — never snaps or rewrites the stroke.">
              <input type="checkbox" id="pb-bpm-grid" /> BPM grid
            </label>
            <label class="hint" style="margin:0; display:inline-flex; align-items:center; gap:6px;"
              data-help="Beats per minute for the curve grid. Leave blank to use audio_check tempo (Hz×60) when available.">
              BPM
              <input type="number" id="pb-bpm" value="" min="20" max="240" step="1" placeholder="auto" style="width:4.5em;" />
            </label>
          </div>
        </details>

        <div class="hint" id="pb-marker-hint" style="display:none"
          data-help="Click the heatmap to seek. Drag to mark a range (Extended-O / loop / O-marker).">
          Heatmap: click = seek, drag = mark. <span id="pb-marker-label"></span>
          <button id="pb-marker-clear" style="margin-left:8px">Clear selection</button>
        </div>
        <div class="checkbox-row" id="pb-marker-auto-row" style="display:none">
          <input type="checkbox" id="pb-marker-auto" />
          <label for="pb-marker-auto" data-help="Triggers Extended-O automatically when playback reaches the marked range.">Auto Extended-O in marked range</label>
        </div>

        <details class="pb-tools-details" id="pb-omarker-details" style="display:none;">
          <summary>O-markers</summary>
          <div class="hint" id="pb-omarker-hint" style="display:none; margin-top:6px;"
            data-help="O-markers are saved in the script (not local only). Primary = peak; secondary = softer spots. Mark a range first, then apply.">
            O-markers: authored in script — see “?”.</div>
          <div class="row" id="pb-omarker-add-row" style="display:none; align-items:center; gap:8px; flex-wrap:wrap;">
            <select id="pb-omarker-kind">
              <option value="primary">Primary (peak)</option>
              <option value="secondary">Secondary (earlier, softer)</option>
            </select>
            <span id="pb-omarker-intensity-row" style="display:none; align-items:center; gap:6px;">
              <label style="width:auto;" data-help="Secondary O-marker strength (0–1). Default 0.5. Authoring only — does not change Create CSRT.">Intensity</label>
              <input type="range" id="pb-omarker-intensity" min="0" max="1" step="0.05" value="0.5" style="width:7em;" />
              <span class="hint" id="pb-omarker-intensity-val" style="margin:0; min-width:2.5em;">0.5</span>
            </span>
            <button id="pb-omarker-add" disabled>Apply as O-marker</button>
          </div>
          <div id="pb-omarker-list" style="display:none; margin-top:6px;"></div>
        </details>

        <details class="pb-tools-details" id="pb-bookmark-details" style="display:none;">
          <summary>Bookmarks</summary>
          <div class="hint" id="pb-bookmark-hint" style="display:none; margin-top:6px;"
            data-help="Named times saved in the script (OFS-style metadata.bookmarks / .samn). Add at the current playhead, seek, rename, or remove. Separate from O-markers and heatmap selection.">
            Bookmarks: named times in the script — see “?”.</div>
          <div class="row" id="pb-bookmark-add-row" style="display:none; align-items:center; gap:8px; flex-wrap:wrap;">
            <input type="text" id="pb-bookmark-name" placeholder="Name (optional)" maxlength="80" style="width:11em;" />
            <button type="button" id="pb-bookmark-add"
              data-help="Saves a bookmark at the current playhead time into the loaded script.">Add at playhead</button>
          </div>
          <div id="pb-bookmark-list" style="display:none; margin-top:6px;"></div>
        </details>

        <details class="pb-tools-details" id="pb-chapter-details" style="display:none;">
          <summary>Chapters</summary>
          <div class="hint" id="pb-chapter-hint" style="display:none; margin-top:6px;"
            data-help="Named ranges saved in the script (OFS-style metadata.chapters / .samn). Mark a heatmap range first, then add. Rename or remove later. Stored chapters replace auto chapter summary in analysis. Separate from O-markers and bookmarks.">
            Chapters: named ranges in the script — see “?”.</div>
          <div class="row" id="pb-chapter-add-row" style="display:none; align-items:center; gap:8px; flex-wrap:wrap;">
            <input type="text" id="pb-chapter-name" placeholder="Name (optional)" maxlength="80" style="width:11em;" />
            <button type="button" id="pb-chapter-add" disabled
              data-help="Saves a chapter from the current heatmap selection into the loaded script.">Add from selection</button>
          </div>
          <div id="pb-chapter-list" style="display:none; margin-top:6px;"></div>
        </details>

        <div id="pb-audio-segments" class="gen-audio-segments" hidden>
          <div class="gen-audio-segments-head">
            <b>Speech-Hold / Feel segments</b>
            <span class="hint" id="pb-audio-hold-summary"></span>
          </div>
          <p class="hint" style="margin:4px 0 6px;">
            From script <code>audio_check</code> (Create Review / Audio check). Click to seek.
            Optional: add visible blocks as chapter marks — never rewrites the stroke.
          </p>
          <details class="pb-tools-details" id="pb-feel-filters-details">
            <summary>Filters &amp; heatmap bands</summary>
            <div class="row gen-audio-seg-filters" style="align-items:center; flex-wrap:wrap; gap:10px; margin:6px 0;">
              <label class="checkbox-row" style="margin:0;"><input type="checkbox" id="pb-seg-f-holding" checked /> Hold</label>
              <label class="checkbox-row" style="margin:0;"><input type="checkbox" id="pb-seg-f-gentle" checked /> Gentle</label>
              <label class="checkbox-row" style="margin:0;"><input type="checkbox" id="pb-seg-f-intense" checked /> Intense</label>
              <label class="checkbox-row" style="margin:0;"><input type="checkbox" id="pb-seg-f-climax" checked /> Climax</label>
              <label class="checkbox-row" style="margin:0;"
                data-help="When on, only dialogue/quiet Hold blocks stay visible.">
                <input type="checkbox" id="pb-seg-f-speech-only" /> Speech-hold only
              </label>
              <label class="checkbox-row" style="margin:0;"
                data-help="Paint filtered Feel segments as translucent bands on the Play heatmap (and a thin curve underlay). Display only — never rewrites the stroke.">
                <input type="checkbox" id="pb-seg-heatbands" checked /> Show on heatmap
              </label>
              <label class="field-row" id="pb-seg-heatbands-opacity-row" style="margin:0; align-items:center; gap:6px;"
                data-help="Opacity of Feel heatmap bands (and curve underlay). Display only.">
                Opacity
                <input type="range" id="pb-seg-heatbands-opacity" min="0.2" max="1" step="0.05" value="1" style="width:7em;" />
                <span class="hint" id="pb-seg-heatbands-opacity-val" style="margin:0; min-width:2.5em;">1.00</span>
              </label>
            </div>
          </details>
          <div id="pb-audio-seg-strip" class="gen-audio-seg-strip" role="list"></div>
          <div class="row" style="margin-top:8px; align-items:center; flex-wrap:wrap; gap:8px;">
            <button type="button" id="pb-audio-seg-chapters">Add visible as chapters</button>
            <span class="hint" id="pb-audio-seg-status"></span>
          </div>
        </div>

        <div class="pb-contact" id="pb-contact-block" hidden>
          <h3>Contact vibration</h3>
          <p class="hint" style="margin-top:0">Follows proximity like contact — adjust live, optionally save to script.</p>
          <div class="checkbox-row" id="pb-contact-off-row">
            <input type="checkbox" id="pb-contact-off" />
            <label for="pb-contact-off"
              data-help="Turns off contact vibration stored in the script for this play only — without creating again. The curve preview stays visible.">Contact vibration off</label>
          </div>
          <details class="pb-tools-details" id="pb-contact-tune-details">
            <summary>Strength · Sensitivity · Curve</summary>
            <div class="field-row" id="pb-contact-intensity-row">
              <label data-help="Live scaling of contact vibration without rewriting the file (SAM runtime). 1 = as created, 0 = off, up to 2 = stronger.">Contact strength</label>
              <input type="range" id="pb-contact-intensity" min="0" max="2" step="0.05" value="1" style="flex:1;" />
              <span id="pb-contact-intensity-val" class="hint" style="margin:0; min-width:2.5em;">1.00</span>
            </div>
            <div class="field-row" id="pb-contact-span-row">
              <label data-help="Live sensitivity without creating again. Lower = engages earlier. Default from the script recipe.">Sensitivity</label>
              <input type="range" id="pb-contact-span" min="0.4" max="0.95" step="0.05" value="0.75" style="flex:1;" />
              <span id="pb-contact-span-val" class="hint" style="margin:0; min-width:2.5em;">0.75</span>
            </div>
            <div class="field-row" id="pb-contact-curve-row">
              <label data-help="Live contact-vibration curve shape (linear / soft / peak / impulse), without rewriting the file.">Contact curve</label>
              <select id="pb-contact-curve">
                <option value="linear">linear</option>
                <option value="soft">soft (contact-like)</option>
                <option value="peak">peak</option>
                <option value="impulse">impulse (peaks only)</option>
              </select>
            </div>
            <details class="pb-tools-details" id="pb-contact-probe-details">
              <summary>Contact feel probe</summary>
              <div class="pb-contact-vib-probe" id="pb-contact-vib-probe" aria-live="polite">
                <div class="pb-contact-vib-badges" id="pb-contact-vib-badges">
                  <span class="pb-cv-badge" id="pb-cv-badge-active">— active</span>
                  <span class="pb-cv-badge" id="pb-cv-badge-peak">— peak</span>
                </div>
                <svg id="pb-contact-vib-svg" class="pb-contact-vib-svg" viewBox="0 0 320 72" preserveAspectRatio="none" aria-hidden="true">
                  <polyline id="pb-contact-vib-stroke" class="pb-contact-vib-stroke" fill="none" stroke-width="1.4" points="" />
                  <polyline id="pb-contact-vib-poly" class="pb-contact-vib-poly" fill="none" stroke-width="1.8" points="" />
                  <g id="pb-contact-vib-peaks" class="pb-contact-vib-peaks"></g>
                </svg>
                <p class="hint" id="pb-contact-vib-preview" style="margin:4px 0 0 0;">
                  Contact probe: change Strength / Sensitivity / Curve for live vib feedback (synthetic bounce — not your clip).
                </p>
              </div>
            </details>
          </details>
          <div class="row" style="margin-top:8px;">
            <button type="button" id="pb-contact-save">Save to script</button>
            <span class="hint" id="pb-contact-save-status" style="margin:0"></span>
          </div>
        </div>

        <details class="pb-advanced">
          <summary>Playback &amp; Extended-O</summary>
          <div class="checkbox-row" id="pb-video-sync-row" style="display:none">
            <input type="checkbox" id="pb-use-video-sync" checked />
            <label for="pb-use-video-sync">Device follows video position</label>
          </div>
          <div class="checkbox-row" id="pb-video-autostart-row" style="display:none">
            <input type="checkbox" id="pb-video-play-autostart" checked />
            <label for="pb-video-play-autostart"
              data-help="Off: the video's own play button/spacebar (with the video focused) plays the video only, without starting the device/curve — use the app's Play button for that. On (default): native video play also starts device playback, same as before.">Video ▶ also starts device</label>
          </div>
          <div class="checkbox-row" id="pb-trajectory-row" style="display:none">
            <input type="checkbox" id="pb-trajectory-toggle" />
            <label for="pb-trajectory-toggle"
              data-help="Draws the tip/partner track path over the video (MT-Debug) — only available on scripts created with 'Record tip path' on. Off by default.">Show tip/partner trajectory</label>
          </div>
          <p class="hint" id="pb-trajectory-hint" style="display:none; margin-top:0;">This script has no recorded trajectory — regenerate with “Record tip/partner trajectory (Debug overlay)” on to use this.</p>
          <div class="field-row"><label>Device</label>
            <span class="checkbox-row" style="margin:0"><input type="checkbox" id="pb-mock" /> <label for="pb-mock" style="width:auto">Mock (no device)</label></span>
          </div>
          <div class="field-row"><label>Sync mode</label>
            <select id="pb-sync">
              <option value="independent">independent</option>
              <option value="synchronized">synchronized</option>
              <option value="alternating">alternating</option>
              <option value="vibration_only">vibration only</option>
              <option value="suction_only">suction only</option>
              <option value="suction_position">Suction from position</option>
            </select>
          </div>
          <div class="pb-adv-grid">
            <div class="field-row" style="align-items:center;">
              <label data-help="Device command interval in milliseconds. Lower = denser stream. Default 50. Playback feel only — not Create CSRT.">Tick (ms)</label>
              <input type="range" id="pb-tick" min="10" max="200" step="10" value="50" style="flex:1;" />
              <span class="hint" id="pb-tick-val" style="margin:0; min-width:2.5em;">50</span>
            </div>
            <div class="field-row" style="align-items:center;">
              <label data-help="Max position change rate for device mapping (0–100 scale per second fraction). Default 0.6. Playback feel only — not Create max-speed.">Max-Speed</label>
              <input type="range" id="pb-maxspeed" min="0.1" max="2" step="0.1" value="0.6" style="flex:1;" />
              <span class="hint" id="pb-maxspeed-val" style="margin:0; min-width:2.5em;">0.6</span>
            </div>
            <div class="field-row" style="align-items:center;">
              <label data-help="Live curve smoothing toward the next sample (0–1). Default 0.3. Playback feel only.">Smoothing</label>
              <input type="range" id="pb-smoothing" min="0" max="1" step="0.05" value="0.3" style="flex:1;" />
              <span class="hint" id="pb-smoothing-val" style="margin:0; min-width:2.5em;">0.3</span>
            </div>
            <div class="field-row" style="align-items:center;">
              <label data-help="Ramp-in duration at play start (ms). Default 500. Playback feel only.">Soft-Start</label>
              <input type="range" id="pb-softstart" min="0" max="3000" step="100" value="500" style="flex:1;" />
              <span class="hint" id="pb-softstart-val" style="margin:0; min-width:3em;">500</span>
            </div>
          </div>
          <details class="pb-tools-details" id="pb-eo-details">
            <summary>Extended-O</summary>
            <div class="checkbox-row"><input type="checkbox" id="pb-eo-enabled" checked /><label for="pb-eo-enabled">Extended-O enabled</label></div>
            <div class="pb-adv-grid">
              <div class="field-row" style="align-items:center;">
                <label data-help="Extended-O amplitude multiplier (curve keeps rhythm; only height is scaled). Default 0.1. Playback feel only.">Amplitude</label>
                <input type="range" id="pb-eo-min" min="0" max="1" step="0.05" value="0.1" style="flex:1;" />
                <span class="hint" id="pb-eo-min-val" style="margin:0; min-width:2.5em;">0.1</span>
              </div>
              <div class="field-row" style="align-items:center;">
                <label data-help="Extended-O hold duration in seconds. Default 10. Playback feel only.">Hold (s)</label>
                <input type="range" id="pb-eo-hold" min="1" max="60" step="1" value="10" style="flex:1;" />
                <span class="hint" id="pb-eo-hold-val" style="margin:0; min-width:2.5em;">10</span>
              </div>
              <div class="field-row" style="align-items:center;">
                <label data-help="Extended-O restore ramp in milliseconds. Default 500. Playback feel only.">Restore (ms)</label>
                <input type="range" id="pb-eo-restore" min="0" max="3000" step="100" value="500" style="flex:1;" />
                <span class="hint" id="pb-eo-restore-val" style="margin:0; min-width:3em;">500</span>
              </div>
            </div>
          </details>
        </details>

        <div id="pb-log" class="pb-log"></div>
      </div>
    </div>
  `;

  wireDataHelp(root);
  const el = id => root.querySelector(id);
  const videoEl = el('#pb-video');
  const heatmapCanvas = el('#pb-heatmap');
  const trajectoryCanvas = el('#pb-trajectory');
  let scriptPath = null;
  let videoPath = null;
  let totalMs = 1;
  // Shared curve/heatmap time window (display/nav only — never rewrites stroke).
  let viewStartMs = 0;
  let viewEndMs = 1;
  let playing = false;
  let playlist = []; // [{ path, name }]
  let playlistIndex = 0;
  let advancingPlaylist = false;
  let userStopRequested = false;
  let heatmapPoints = null;
  let curvePoints = null;
  let speedHighlights = [];
  let vibrationCurvePoints = null;
  let scriptHasContactVibration = false;
  let trajectoryData = null; // MT-Debug: {width,height,tip:[{atMs,x,y}],partner:[...]} or null
  let contactMarksData = null; // { tip_class, primary:{x,y,w,h,class,fixed}, extras:[...], drive_stroke }
  /** @type {Array<{label:string,startMs:number,endMs:number,speechHold:boolean}>} */
  let audioSegments = [];
  let speechHoldMs = 0;
  let audioHzHint = 0;
  const AUDIO_SEG_LABELS = ['holding', 'gentle', 'intense', 'climax'];
  const curveCanvas = el('#pb-curve');
  const chartTooltip = el('#pb-chart-tooltip');

  // sizeCanvasForDPR setzt die Backing-Store-Auflösung auf CSS-Größe ×
  // devicePixelRatio, statt 1:1 auf clientWidth/Height - sonst wird auf
  // HiDPI/Retina-Displays ein 1x-Bitmap hochskaliert und wirkt unscharf,
  // während der Rest der UI (CSS) gestochen scharf bleibt. Die gesamte
  // Koordinatenmathematik unten (curveXOf/curveYOf/curveMouseXY/...)
  // rechnet bereits mit canvas.width/height statt mit clientWidth/Height,
  // bleibt also automatisch korrekt, sobald der Backing-Store größer ist -
  // kein zusätzlicher ctx-Transform nötig.
  function sizeCanvasForDPR(canvas, fallbackW, fallbackH) {
    const dpr = window.devicePixelRatio || 1;
    const cssW = canvas.clientWidth || fallbackW;
    const cssH = canvas.clientHeight || fallbackH;
    canvas.width = Math.round(cssW * dpr);
    canvas.height = Math.round(cssH * dpr);
  }
  const CURVE_MAX_POINTS = 1200;
  const PLAYLIST_SHUFFLE_KEY = 'pb.playlist.shuffle';
  const PLAYLIST_REPEAT_KEY = 'pb.playlist.repeat';
  let marker = null; // {startMs, endMs} oder null
  let markerDragStartMs = null;
  let markerDragMoved = false;
  let oMarkers = []; // [{startMs, endMs, kind, intensity}], im Skript gespeichert (siehe funscript.OMarker)
  let bookmarks = []; // [{name, time}] ms — OFS metadata.bookmarks / .samn
  let chapterMarks = []; // [{name, startTime, endTime}] ms — OFS metadata.chapters / .samn
  let autoEOTriggeredForMarker = false;
  // Kurven-Editor: bearbeitet die vollen, nicht resampleten Punkte
  // (rawActions), nicht curvePoints - curvePoints ist nur eine
  // Anzeige-Kompromisskurve (siehe GetScriptCurve), ein Speichern daraus
  // würde ein Skript mit mehr Punkten als CURVE_MAX_POINTS stillschweigend
  // ausdünnen.
  let editMode = false;
  let rawActions = null; // [{atMs, pos}] voller Auflösung, nur während editMode gesetzt
  // FunGen-like: keyframe dots stay visible on the soft curve (and while playing).
  let keyframeDots = null; // [{atMs, pos}] — display only when not editing
  let editDragIndex = null;
  let editDragStartValue = null; // {atMs, pos} des gegriffenen Punkts vor dem Ziehen, null bei neuem Punkt
  let editAxis = 'general'; // general | vibration | suction
  let scriptNativeFormat = false;
  let scriptHasNeoAxes = false;
  const CURVE_PAD = 6;
  const EDIT_HIT_RADIUS_PX = 12;
  const DOT_MAX_DRAW = 600; // dense scripts: subsample dots for draw cost
  let currentPosMs = 0;
  const POS_OVERLAY_PREF = 'samn.pbPosOverlay';

  function posOverlayWanted() {
    const cb = el('#pb-pos-overlay-toggle');
    if (!cb) return true;
    try {
      const saved = localStorage.getItem(POS_OVERLAY_PREF);
      if (saved === '0') { cb.checked = false; return false; }
      if (saved === '1') { cb.checked = true; return true; }
    } catch (_) { /* ignore */ }
    return !!cb.checked;
  }

  function updatePbPosOverlay(livePos) {
    const box = el('#pb-pos-overlay');
    const knob = el('#pb-pos-knob');
    const fill = el('#pb-pos-fill');
    const val = el('#pb-pos-value');
    const row = el('#pb-pos-overlay-row');
    if (!box || !knob || !fill || !val) return;
    const hasCurve = curvePoints && curvePoints.length >= 2;
    if (row) row.style.display = hasCurve ? 'flex' : 'none';
    const show = hasCurve && posOverlayWanted();
    box.hidden = !show;
    box.setAttribute('aria-hidden', show ? 'false' : 'true');
    if (!show) return;
    let pos = livePos;
    if (pos == null) pos = interpPosAt(curvePoints, currentPosMs);
    if (pos == null || Number.isNaN(pos)) {
      val.textContent = '—';
      return;
    }
    const p = Math.max(0, Math.min(100, pos));
    knob.style.bottom = `calc(${p}% - 7px)`;
    fill.style.height = p + '%';
    val.textContent = String(Math.round(p));
  }

  function setScriptLoaded(loaded) {
    el('#pb-empty').hidden = !!loaded;
    el('#pb-loaded').hidden = !loaded;
    root.classList.toggle('has-script', !!loaded);
    el('#pb-queue-add').hidden = !loaded;
  }

  function scriptBaseName(path) {
    const base = String(path || '').split(/[/\\]/).pop() || path;
    return base.replace(/\.(funscript|samn)$/i, '');
  }

  function renderPlaylist() {
    const wrap = el('#pb-playlist');
    const list = el('#pb-playlist-list');
    const nextBtn = el('#pb-next');
    if (!wrap || !list) return;
    if (playlist.length < 2) {
      wrap.hidden = true;
      list.innerHTML = '';
      if (nextBtn) nextBtn.hidden = true;
      return;
    }
    wrap.hidden = false;
    const repeat = el('#pb-playlist-repeat') && el('#pb-playlist-repeat').checked;
    if (nextBtn) nextBtn.hidden = playlistIndex >= playlist.length - 1 && !repeat;
    list.innerHTML = playlist.map((item, i) => {
      const active = i === playlistIndex ? ' is-active' : '';
      return `<li class="pb-playlist-item${active}" data-idx="${i}">`
        + `<button type="button" class="pb-playlist-pick" data-idx="${i}">`
        + `<span class="pb-playlist-idx">${i + 1}</span>`
        + `<span class="pb-playlist-name">${item.name}</span>`
        + `</button>`
        + `<button type="button" class="pb-playlist-remove" data-idx="${i}" title="Remove">×</button>`
        + `</li>`;
    }).join('');
  }

  function replacePlaylist(paths, startIndex = 0) {
    const uniq = [];
    const seen = new Set();
    for (const p of paths || []) {
      if (!p || seen.has(p)) continue;
      seen.add(p);
      uniq.push({ path: p, name: scriptBaseName(p) });
    }
    playlist = uniq;
    playlistIndex = Math.max(0, Math.min(startIndex, Math.max(0, playlist.length - 1)));
    renderPlaylist();
  }

  function shuffleArrayInPlace(arr) {
    for (let i = arr.length - 1; i > 0; i--) {
      const j = Math.floor(Math.random() * (i + 1));
      [arr[i], arr[j]] = [arr[j], arr[i]];
    }
    return arr;
  }

  /** Reshuffle queue with the current item first (when shuffle is turned on). */
  function reshufflePlaylistKeepingCurrent() {
    if (playlist.length < 2) return;
    const cur = playlist[playlistIndex];
    const rest = playlist.filter((_, i) => i !== playlistIndex);
    shuffleArrayInPlace(rest);
    playlist = [cur, ...rest];
    playlistIndex = 0;
    renderPlaylist();
  }

  /** New cycle at end of list: full reshuffle when shuffle is on. */
  function reshufflePlaylistForRepeatCycle() {
    if (playlist.length < 2) return;
    shuffleArrayInPlace(playlist);
    playlistIndex = 0;
    renderPlaylist();
  }

  function appendToPlaylist(path) {
    if (!path) return;
    if (playlist.some(p => p.path === path)) {
      log('Already in list: ' + scriptBaseName(path));
      return;
    }
    if (playlist.length === 0 && scriptPath) {
      playlist.push({ path: scriptPath, name: scriptBaseName(scriptPath) });
      playlistIndex = 0;
    }
    playlist.push({ path, name: scriptBaseName(path) });
    renderPlaylist();
    log('Added to list: ' + scriptBaseName(path));
  }

  function log(line) {
    const box = el('#pb-log');
    box.textContent += (box.textContent ? '\n' : '') + line;
    box.scrollTop = box.scrollHeight;
  }

  function logError(line) {
    log(line);
    uiError(line);
  }

  function setPlayingState(isPlaying) {
    playing = isPlaying;
    el('#pb-play').disabled = isPlaying;
    el('#pb-stop').disabled = !isPlaying;
    el('#pb-eo-trigger').disabled = !isPlaying || !el('#pb-eo-enabled').checked;
    const playAlone = el('#pb-play-novideo');
    if (playAlone) playAlone.disabled = isPlaying;
    if (!isPlaying) el('#pb-progress').style.width = '0%';
    if (isPlaying) autoEOTriggeredForMarker = false;
    el('#pb-video-stage').classList.toggle('is-playing', !!isPlaying);
    // Bearbeiten während der Playback wäre verwirrend (die Kurve bewegt
    // sich durch den Positionszeiger mit) - beim Start aus, Checkbox bis
    // zum Stop gesperrt.
    el('#pb-curve-edit').disabled = isPlaying;
    if (isPlaying && editMode) {
      el('#pb-curve-edit').checked = false;
      setEditMode(false);
    }
  }

  function markerIsValid(m) {
    return !!(m
      && Number.isFinite(m.startMs)
      && Number.isFinite(m.endMs)
      && m.endMs > m.startMs);
  }

  function updateMarkerHint() {
    if (markerIsValid(marker)) {
      el('#pb-marker-label').textContent =
        `Marked: ${(marker.startMs / 1000).toFixed(1)}s - ${(marker.endMs / 1000).toFixed(1)}s`;
      // Selection unlocks Cap/Scale/O-marker/Chapters — open those panels.
      const edit = el('#pb-ofs-edit-details');
      if (edit) edit.open = true;
      const om = el('#pb-omarker-details');
      if (om) om.open = true;
      const ch = el('#pb-chapter-details');
      if (ch) ch.open = true;
    } else {
      el('#pb-marker-label').textContent = '(no selection)';
      // Clear leaves Cap/O/Chapters closed — no disabled CTAs in open panels.
      const edit = el('#pb-ofs-edit-details');
      if (edit) edit.open = false;
      const om = el('#pb-omarker-details');
      if (om) om.open = false;
      const ch = el('#pb-chapter-details');
      if (ch) ch.open = false;
    }
    el('#pb-omarker-add').disabled = !markerIsValid(marker);
    const chAdd = el('#pb-chapter-add');
    if (chAdd) chAdd.disabled = !markerIsValid(marker);
    updateZoomChrome();
  }

  function viewSpanMs() {
    return Math.max(1, viewEndMs - viewStartMs);
  }
  function isZoomed() {
    return totalMs > 0 && (viewStartMs > 0 || viewEndMs < totalMs - 1);
  }
  function clampViewWindow() {
    const dur = Math.max(1, totalMs);
    if (viewEndMs <= viewStartMs) {
      viewStartMs = 0;
      viewEndMs = dur;
    }
    viewStartMs = Math.max(0, Math.min(viewStartMs, dur - 1));
    viewEndMs = Math.max(viewStartMs + 1, Math.min(viewEndMs, dur));
  }
  function resetViewWindow() {
    viewStartMs = 0;
    viewEndMs = Math.max(1, totalMs);
    updateZoomChrome();
  }
  function timeToX(ms, w) {
    return ((ms - viewStartMs) / viewSpanMs()) * w;
  }
  function fracToMs(frac) {
    return Math.round(viewStartMs + Math.max(0, Math.min(1, frac)) * viewSpanMs());
  }
  function updateZoomChrome() {
    const row = el('#pb-curve-zoom');
    const lab = el('#pb-zoom-label');
    const zoomSel = el('#pb-zoom-sel');
    if (row) row.style.display = scriptPath ? 'flex' : 'none';
    if (zoomSel) zoomSel.disabled = !markerIsValid(marker);
    if (!lab) return;
    if (!isZoomed()) {
      lab.textContent = 'Full';
      return;
    }
    lab.textContent = `${(viewStartMs / 1000).toFixed(1)}s–${(viewEndMs / 1000).toFixed(1)}s`;
  }
  function setViewWindow(startMs, endMs) {
    viewStartMs = startMs;
    viewEndMs = endMs;
    clampViewWindow();
    updateZoomChrome();
    redrawHeatmap();
    redrawCurve();
  }
  function zoomByFactor(factor, anchorMs) {
    const span = viewSpanMs();
    const next = Math.max(250, Math.min(totalMs, span / factor));
    if (next >= totalMs - 1) {
      resetViewWindow();
      redrawHeatmap();
      redrawCurve();
      return;
    }
    const anchor = Number.isFinite(anchorMs) ? anchorMs : (viewStartMs + viewEndMs) / 2;
    const rel = span > 0 ? (anchor - viewStartMs) / span : 0.5;
    let start = Math.round(anchor - rel * next);
    let end = Math.round(start + next);
    if (start < 0) {
      end -= start;
      start = 0;
    }
    if (end > totalMs) {
      start -= (end - totalMs);
      end = totalMs;
    }
    setViewWindow(Math.max(0, start), Math.min(totalMs, end));
  }
  function zoomToSelection() {
    if (!markerIsValid(marker)) return;
    const pad = Math.max(200, Math.round((marker.endMs - marker.startMs) * 0.08));
    setViewWindow(Math.max(0, marker.startMs - pad), Math.min(totalMs, marker.endMs + pad));
  }

  // drawOMarkerBands zeichnet die gespeicherten O-Marker als farbige Bänder
  // unter/hinter der weißen Live-Markierung - primär kräftiger als
  // sekundär, damit auf einen Blick klar ist, welcher der Höhepunkt ist.
  function drawOMarkerBands(ctx, w, h) {
    if (!oMarkers.length || totalMs <= 0) return;
    for (const m of oMarkers) {
      if (m.endMs < viewStartMs || m.startMs > viewEndMs) continue;
      const x0 = timeToX(m.startMs, w);
      const x1 = timeToX(m.endMs, w);
      const isPrimary = m.kind === 'primary';
      const alpha = isPrimary ? 0.35 : 0.18 + m.intensity * 0.15;
      ctx.fillStyle = isPrimary ? `rgba(255,60,60,${alpha})` : `rgba(255,160,40,${alpha})`;
      ctx.fillRect(x0, 0, Math.max(1, x1 - x0), h);
    }
  }

  // Feel / Speech-Hold taxonomy bands (display only — synced with strip filters).
  const FEEL_BAND_RGBA = {
    holding: [232, 188, 96, 0.38],
    gentle: [90, 212, 196, 0.34],
    intense: [184, 150, 232, 0.36],
    climax: [232, 110, 110, 0.40],
  };
  function feelHeatbandsEnabled() {
    const box = el('#pb-seg-heatbands');
    return !box || !!box.checked;
  }
  function feelHeatbandsOpacity() {
    const v = parseFloat(el('#pb-seg-heatbands-opacity')?.value);
    if (!Number.isFinite(v)) return 1;
    return Math.max(0.2, Math.min(1, v));
  }
  function feelBandFill(label, opacity) {
    const c = FEEL_BAND_RGBA[label] || [200, 210, 230, 0.28];
    return `rgba(${c[0]},${c[1]},${c[2]},${(c[3] * opacity).toFixed(3)})`;
  }
  function syncFeelHeatbandsOpacityUi() {
    const on = feelHeatbandsEnabled();
    const row = el('#pb-seg-heatbands-opacity-row');
    if (row) row.style.display = on ? 'flex' : 'none';
    const lab = el('#pb-seg-heatbands-opacity-val');
    if (lab) lab.textContent = feelHeatbandsOpacity().toFixed(2);
  }
  function drawFeelSegmentBands(ctx, w, h, { fullHeight = true } = {}) {
    if (!feelHeatbandsEnabled() || totalMs <= 0 || !audioSegments.length) return;
    const visible = visibleAudioSegments();
    if (!visible.length) return;
    const opacity = feelHeatbandsOpacity();
    const bandH = fullHeight ? h : Math.max(3, Math.round(h * 0.14));
    const y = fullHeight ? 0 : h - bandH;
    for (const s of visible) {
      if (s.endMs < viewStartMs || s.startMs > viewEndMs) continue;
      const x0 = timeToX(s.startMs, w);
      const x1 = timeToX(s.endMs, w);
      ctx.fillStyle = feelBandFill(s.label, opacity);
      ctx.fillRect(x0, y, Math.max(1, x1 - x0), bandH);
      if (s.speechHold && fullHeight) {
        ctx.strokeStyle = `rgba(14, 17, 26,${(0.35 * opacity).toFixed(3)})`;
        ctx.lineWidth = 1;
        ctx.setLineDash([3, 2]);
        ctx.strokeRect(x0 + 0.5, 0.5, Math.max(1, x1 - x0) - 1, h - 1);
        ctx.setLineDash([]);
      }
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
      const kindLabel = m.kind === 'primary' ? 'Primary' : 'Secondary';
      const intensityLabel = m.kind === 'secondary' ? ` · ${Math.round(m.intensity * 100)}%` : '';
      label.textContent = `${kindLabel}: ${(m.startMs / 1000).toFixed(1)}s - ${(m.endMs / 1000).toFixed(1)}s${intensityLabel}`;
      const removeBtn = document.createElement('button');
      removeBtn.textContent = 'Remove';
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
      logError('Remove O-marker: ' + err);
      return;
    }
    renderOMarkerList();
    redrawHeatmap();
    redrawCurve();
  }

  function normalizeBookmark(b) {
    if (!b || typeof b !== 'object') return null;
    const name = String(b.name ?? b.Name ?? '').trim() || 'Bookmark';
    const time = Number(b.time ?? b.Time ?? 0);
    if (!Number.isFinite(time) || time < 0) return null;
    return { name, time: Math.round(time) };
  }

  function renderBookmarkList() {
    const box = el('#pb-bookmark-list');
    if (!box) return;
    box.innerHTML = '';
    if (!scriptPath || bookmarks.length === 0) {
      box.style.display = 'none';
      return;
    }
    box.style.display = 'block';
    bookmarks.forEach((b, index) => {
      const row = document.createElement('div');
      row.className = 'row';
      row.style.cssText = 'align-items:center; gap:8px; margin-top:2px; flex-wrap:wrap;';
      const label = document.createElement('span');
      label.textContent = `${b.name} · ${(b.time / 1000).toFixed(1)}s`;
      const seekBtn = document.createElement('button');
      seekBtn.type = 'button';
      seekBtn.textContent = 'Seek';
      seekBtn.addEventListener('click', () => seekTo(b.time));
      const renameBtn = document.createElement('button');
      renameBtn.type = 'button';
      renameBtn.textContent = 'Rename';
      renameBtn.addEventListener('click', () => renameBookmark(index));
      const removeBtn = document.createElement('button');
      removeBtn.type = 'button';
      removeBtn.textContent = 'Remove';
      removeBtn.addEventListener('click', () => removeBookmark(index));
      row.appendChild(label);
      row.appendChild(seekBtn);
      row.appendChild(renameBtn);
      row.appendChild(removeBtn);
      box.appendChild(row);
    });
  }

  async function persistBookmarks(next, rollback) {
    try {
      await SaveScriptBookmarks(next);
      bookmarks = next;
      renderBookmarkList();
      return true;
    } catch (err) {
      bookmarks = rollback;
      renderBookmarkList();
      logError('Bookmarks: ' + err);
      return false;
    }
  }

  async function renameBookmark(index) {
    const b = bookmarks[index];
    if (!b) return;
    const raw = window.prompt('Rename bookmark', b.name);
    if (raw == null) return;
    const name = String(raw).trim() || 'Bookmark';
    if (name === b.name) return;
    const previous = bookmarks.slice();
    const next = bookmarks.map((item, i) => (i === index ? { ...item, name } : item));
    await persistBookmarks(next, previous);
  }

  async function removeBookmark(index) {
    const previous = bookmarks.slice();
    const next = bookmarks.filter((_, i) => i !== index);
    await persistBookmarks(next, previous);
  }

  function normalizeChapterMark(c) {
    if (!c || typeof c !== 'object') return null;
    const name = String(c.name ?? c.Name ?? '').trim() || 'Chapter';
    const startTime = Number(c.startTime ?? c.StartTime ?? 0);
    let endTime = Number(c.endTime ?? c.EndTime ?? 0);
    if (!Number.isFinite(startTime) || startTime < 0) return null;
    if (!Number.isFinite(endTime) || endTime < 0) endTime = 0;
    const start = Math.round(startTime);
    let end = Math.round(endTime);
    if (end > 0 && end < start) {
      const tmp = start;
      return { name, startTime: end, endTime: tmp };
    }
    return { name, startTime: start, endTime: end };
  }

  function formatChapterRange(c) {
    const a = `${(c.startTime / 1000).toFixed(1)}s`;
    if (!c.endTime) return `${a} → end`;
    return `${a} - ${(c.endTime / 1000).toFixed(1)}s`;
  }

  function renderChapterList() {
    const box = el('#pb-chapter-list');
    if (!box) return;
    box.innerHTML = '';
    if (!scriptPath || chapterMarks.length === 0) {
      box.style.display = 'none';
      return;
    }
    box.style.display = 'block';
    chapterMarks.forEach((c, index) => {
      const row = document.createElement('div');
      row.className = 'row';
      row.style.cssText = 'align-items:center; gap:8px; margin-top:2px; flex-wrap:wrap;';
      const label = document.createElement('span');
      label.textContent = `${c.name} · ${formatChapterRange(c)}`;
      const seekBtn = document.createElement('button');
      seekBtn.type = 'button';
      seekBtn.textContent = 'Seek';
      seekBtn.addEventListener('click', () => seekTo(c.startTime));
      const renameBtn = document.createElement('button');
      renameBtn.type = 'button';
      renameBtn.textContent = 'Rename';
      renameBtn.addEventListener('click', () => renameChapterMark(index));
      const removeBtn = document.createElement('button');
      removeBtn.type = 'button';
      removeBtn.textContent = 'Remove';
      removeBtn.addEventListener('click', () => removeChapterMark(index));
      row.appendChild(label);
      row.appendChild(seekBtn);
      row.appendChild(renameBtn);
      row.appendChild(removeBtn);
      box.appendChild(row);
    });
  }

  async function persistChapterMarks(next, rollback) {
    try {
      await SaveScriptChapterMarks(next);
      chapterMarks = next;
      renderChapterList();
      describeScript();
      return true;
    } catch (err) {
      chapterMarks = rollback;
      renderChapterList();
      logError('Chapters: ' + err);
      return false;
    }
  }

  async function renameChapterMark(index) {
    const c = chapterMarks[index];
    if (!c) return;
    const raw = window.prompt('Rename chapter', c.name);
    if (raw == null) return;
    const name = String(raw).trim() || 'Chapter';
    if (name === c.name) return;
    const previous = chapterMarks.slice();
    const next = chapterMarks.map((item, i) => (i === index ? { ...item, name } : item));
    await persistChapterMarks(next, previous);
  }

  async function removeChapterMark(index) {
    const previous = chapterMarks.slice();
    const next = chapterMarks.filter((_, i) => i !== index);
    await persistChapterMarks(next, previous);
  }

  function normalizeAudioSegment(s) {
    if (!s || typeof s !== 'object') return null;
    const label = String(s.label ?? s.Label ?? '').toLowerCase().trim() || 'gentle';
    let start = Number(s.start_ms ?? s.StartMs ?? s.startMs ?? 0);
    let end = Number(s.end_ms ?? s.EndMs ?? s.endMs ?? 0);
    if (!Number.isFinite(start) || !Number.isFinite(end)) return null;
    start = Math.round(start);
    end = Math.round(end);
    if (end < start) {
      const tmp = start;
      start = end;
      end = tmp;
    }
    if (end <= start) return null;
    const speechHold = !!(s.speech_hold ?? s.SpeechHold ?? s.speechHold);
    return { label, startMs: start, endMs: end, speechHold };
  }

  function chapterNameForAudioSegment(s) {
    if (s.label === 'holding') return s.speechHold ? 'Hold (speech)' : 'Hold';
    if (s.label === 'gentle') return 'Gentle';
    if (s.label === 'intense') return 'Intense';
    if (s.label === 'climax') return 'Climax';
    return s.label ? s.label.charAt(0).toUpperCase() + s.label.slice(1) : 'Segment';
  }

  function audioSegFilterOn(label) {
    const box = el(`#pb-seg-f-${label}`);
    return !box || !!box.checked;
  }

  function visibleAudioSegments() {
    const speechOnly = !!el('#pb-seg-f-speech-only')?.checked;
    return audioSegments.filter((s) => {
      if (!AUDIO_SEG_LABELS.includes(s.label)) return false;
      if (!audioSegFilterOn(s.label)) return false;
      if (speechOnly && !s.speechHold) return false;
      return true;
    });
  }

  function rememberAudioSegmentsFromInfo(info) {
    const segs = (info && (info.audioSegments || info.AudioSegments)) || [];
    const hold = Number(info?.speechHoldMs ?? info?.SpeechHoldMs ?? 0);
    const hz = Number(info?.audioHz ?? info?.AudioHz ?? 0);
    audioSegments = Array.isArray(segs) ? segs.map(normalizeAudioSegment).filter(Boolean) : [];
    speechHoldMs = Number.isFinite(hold) ? hold : 0;
    audioHzHint = Number.isFinite(hz) && hz > 0.2 && hz < 8 ? hz : 0;
    const bpmInput = el('#pb-bpm');
    if (bpmInput && !bpmInput.value && audioHzHint > 0) {
      bpmInput.placeholder = `auto ${Math.round(audioHzHint * 60)}`;
    }
    renderAudioSegmentsPanel();
    redrawCurve();
  }

  function bpmPeriodMs() {
    const raw = el('#pb-bpm')?.value;
    const bpm = parseFloat(raw);
    if (Number.isFinite(bpm) && bpm >= 20 && bpm <= 240) {
      return 60000 / bpm;
    }
    if (audioHzHint > 0) {
      return 1000 / audioHzHint;
    }
    return 0;
  }

  function drawBpmGrid(ctx, w, h) {
    if (!el('#pb-bpm-grid')?.checked || totalMs <= 0) return;
    const period = bpmPeriodMs();
    if (!(period > 0)) return;
    const maxLines = 240;
    let n = Math.floor(viewStartMs / period);
    let t = n * period;
    let drawn = 0;
    while (t <= viewEndMs && drawn < maxLines) {
      if (t >= viewStartMs) {
        const x = Math.round(timeToX(t, w)) + 0.5;
        const bar = n % 4 === 0;
        ctx.strokeStyle = bar ? 'rgba(184, 150, 232, 0.38)' : 'rgba(184, 150, 232, 0.16)';
        ctx.lineWidth = bar ? 1.25 : 1;
        ctx.beginPath();
        ctx.moveTo(x, 0);
        ctx.lineTo(x, h);
        ctx.stroke();
        drawn += 1;
      }
      n += 1;
      t = n * period;
    }
  }

  function renderAudioSegmentsPanel() {
    const panel = el('#pb-audio-segments');
    const strip = el('#pb-audio-seg-strip');
    const summary = el('#pb-audio-hold-summary');
    if (!panel || !strip) return;
    if (!scriptPath || !audioSegments.length) {
      panel.hidden = true;
      strip.innerHTML = '';
      if (summary) summary.textContent = '';
      redrawHeatmap();
      redrawCurve();
      return;
    }
    panel.hidden = false;
    if (summary) {
      summary.textContent = speechHoldMs > 0
        ? `· Speech-Hold ≈ ${(speechHoldMs / 1000).toFixed(1)}s · ${audioSegments.length} segment(s)`
        : `· ${audioSegments.length} segment(s)`;
    }
    const visible = visibleAudioSegments();
    const tMin = Math.min(...audioSegments.map((s) => s.startMs));
    const tMax = Math.max(...audioSegments.map((s) => s.endMs));
    const span = Math.max(1, tMax - tMin);
    strip.innerHTML = '';
    if (!visible.length) {
      const empty = document.createElement('span');
      empty.className = 'hint';
      empty.textContent = 'No segments match the filters.';
      strip.appendChild(empty);
      redrawHeatmap();
      redrawCurve();
      return;
    }
    visible.forEach((s) => {
      const btn = document.createElement('button');
      btn.type = 'button';
      btn.className = `gen-audio-seg gen-audio-seg-${s.label}` + (s.speechHold ? ' is-speech-hold' : '');
      btn.setAttribute('role', 'listitem');
      const left = ((s.startMs - tMin) / span) * 100;
      const width = Math.max(1.5, ((s.endMs - s.startMs) / span) * 100);
      btn.style.left = `${left}%`;
      btn.style.width = `${width}%`;
      btn.title = `${chapterNameForAudioSegment(s)} · ${(s.startMs / 1000).toFixed(1)}s–${(s.endMs / 1000).toFixed(1)}s`
        + (s.speechHold ? ' · speech-hold' : '');
      btn.textContent = chapterNameForAudioSegment(s);
      btn.addEventListener('click', () => {
        seekTo(s.startMs);
        const st = el('#pb-audio-seg-status');
        if (st) st.textContent = `Seek ${(s.startMs / 1000).toFixed(1)}s — ${chapterNameForAudioSegment(s)}`;
      });
      strip.appendChild(btn);
    });
    redrawHeatmap();
    redrawCurve();
  }

  async function addVisibleAudioSegmentsAsChapters() {
    const status = el('#pb-audio-seg-status');
    const visible = visibleAudioSegments();
    if (!visible.length) {
      if (status) status.textContent = 'Nothing visible to add — adjust filters.';
      return;
    }
    if (!scriptPath) {
      showChooseScriptRetry('Load a script first.', status);
      return;
    }
    try {
      const existing = await GetScriptChapterMarks().catch(() => []) || [];
      const mapped = visible.map((s) => ({
        name: chapterNameForAudioSegment(s),
        startTime: s.startMs,
        endTime: s.endMs,
      }));
      const next = existing.slice();
      let added = 0;
      for (const ch of mapped) {
        const clash = next.some((e) => Math.abs((e.startTime ?? e.StartTime ?? 0) - ch.startTime) < 250);
        if (clash) continue;
        next.push(ch);
        added += 1;
      }
      if (await persistChapterMarks(next, existing)) {
        if (status) {
          status.textContent = added > 0
            ? `Added ${added} chapter mark(s)`
            : 'Chapters already present near those times';
        }
        if (added > 0) uiInfo(`Added ${added} audio segment chapter(s)`, el('#pb-log'));
      }
    } catch (err) {
      if (status) status.textContent = 'Chapters failed: ' + err;
      uiError('Add chapters: ' + err, el('#pb-log'));
    }
  }

  // --- MT-Debug: Tip/Partner-Trajektorie über dem Video -------------------
  // videoContentRect berechnet die tatsächlich sichtbare Videofläche
  // innerhalb von #pb-video - object-fit:contain lässt bei abweichendem
  // Seitenverhältnis Letterbox-Balken entstehen, und das Overlay muss auf
  // den eingebetteten Pixeln liegen, nicht auf der ganzen Box (sonst
  // driftet die Linie in den Balken hinein).
  function videoContentRect() {
    const vw = videoEl.videoWidth, vh = videoEl.videoHeight;
    const boxW = videoEl.clientWidth, boxH = videoEl.clientHeight;
    if (!vw || !vh || !boxW || !boxH) return null;
    const videoRatio = vw / vh, boxRatio = boxW / boxH;
    let w, h, x, y;
    if (videoRatio > boxRatio) {
      w = boxW; h = boxW / videoRatio; x = 0; y = (boxH - h) / 2;
    } else {
      h = boxH; w = boxH * videoRatio; y = 0; x = (boxW - w) / 2;
    }
    return { x, y, w, h };
  }

  function sizeTrajectoryCanvas(rect) {
    const dpr = window.devicePixelRatio || 1;
    trajectoryCanvas.style.left = rect.x + 'px';
    trajectoryCanvas.style.top = rect.y + 'px';
    trajectoryCanvas.style.width = rect.w + 'px';
    trajectoryCanvas.style.height = rect.h + 'px';
    trajectoryCanvas.width = Math.max(1, Math.round(rect.w * dpr));
    trajectoryCanvas.height = Math.max(1, Math.round(rect.h * dpr));
  }

  function drawTrajectoryPath(ctx, points, w, h, color) {
    if (!points || points.length < 2) return;
    ctx.strokeStyle = color;
    ctx.lineWidth = 2 * (window.devicePixelRatio || 1);
    ctx.lineJoin = 'round';
    ctx.beginPath();
    points.forEach((p, i) => {
      const x = (p.x / trajectoryData.width) * w;
      const y = (p.y / trajectoryData.height) * h;
      if (i === 0) ctx.moveTo(x, y); else ctx.lineTo(x, y);
    });
    ctx.stroke();
  }

  // nearestTrajectoryPoint: points sind chronologisch (gleiche Reihenfolge
  // wie beim Tracking) - lineare Suche reicht, typische Skriptlänge macht
  // das nicht spürbar teuer, und es läuft nur bei Toggle+Daten aktiv.
  function nearestTrajectoryPoint(points, atMs) {
    if (!points || points.length === 0) return null;
    let best = points[0], bestDist = Math.abs(points[0].atMs - atMs);
    for (let i = 1; i < points.length; i++) {
      const d = Math.abs(points[i].atMs - atMs);
      if (d < bestDist) { bestDist = d; best = points[i]; }
    }
    return best;
  }

  function trajectoryWanted() {
    return !!(videoPath && trajectoryData && el('#pb-trajectory-toggle').checked);
  }

  function contactMarksWanted() {
    return !!(videoPath && contactMarksData && el('#pb-contact-marks-toggle')?.checked
      && (contactMarksData.tip || contactMarksData.primary
        || (contactMarksData.extras && contactMarksData.extras.length)));
  }

  function overlayWanted() {
    return trajectoryWanted() || contactMarksWanted();
  }

  function drawContactMarkBox(ctx, box, vw, vh, canvasW, canvasH, color, label) {
    if (!box || !(box.w > 0) || !(box.h > 0) || !(vw > 0) || !(vh > 0)) return;
    const dpr = window.devicePixelRatio || 1;
    const x = (box.x / vw) * canvasW;
    const y = (box.y / vh) * canvasH;
    const w = (box.w / vw) * canvasW;
    const h = (box.h / vh) * canvasH;
    ctx.strokeStyle = color;
    ctx.lineWidth = 2 * dpr;
    ctx.setLineDash([]);
    ctx.strokeRect(x, y, w, h);
    if (label) {
      ctx.fillStyle = color;
      ctx.font = `${11 * dpr}px sans-serif`;
      ctx.fillText(label, x + 3 * dpr, Math.max(12 * dpr, y - 4 * dpr));
    }
  }

  function redrawTrajectory() {
    if (!overlayWanted()) {
      trajectoryCanvas.hidden = true;
      return;
    }
    const rect = videoContentRect();
    if (!rect) {
      // Video-Metadaten (videoWidth/Height) noch nicht geladen - erneut
      // via 'loadedmetadata' versucht, hier nur sauber nichts zeichnen.
      trajectoryCanvas.hidden = true;
      return;
    }
    trajectoryCanvas.hidden = false;
    sizeTrajectoryCanvas(rect);
    const ctx = trajectoryCanvas.getContext('2d');
    const w = trajectoryCanvas.width, h = trajectoryCanvas.height;
    ctx.clearRect(0, 0, w, h);

    if (trajectoryWanted()) {
      drawTrajectoryPath(ctx, trajectoryData.tip, w, h, 'rgba(184, 150, 232, 0.88)');
      drawTrajectoryPath(ctx, trajectoryData.partner, w, h, 'rgba(95, 208, 200, 0.85)');
      const tipNow = nearestTrajectoryPoint(trajectoryData.tip, currentPosMs);
      if (tipNow) {
        const dpr = window.devicePixelRatio || 1;
        const x = (tipNow.x / trajectoryData.width) * w;
        const y = (tipNow.y / trajectoryData.height) * h;
        ctx.beginPath();
        ctx.arc(x, y, 5 * dpr, 0, Math.PI * 2);
        ctx.fillStyle = '#b896e8';
        ctx.fill();
        ctx.lineWidth = 1.5 * dpr;
        ctx.strokeStyle = '#fff';
        ctx.stroke();
      }
    }

    if (contactMarksWanted()) {
      const vw = videoEl.videoWidth || trajectoryData?.width || 0;
      const vh = videoEl.videoHeight || trajectoryData?.height || 0;
      const tip = contactMarksData.tip;
      if (tip) {
        const label = tip.class || contactMarksData.tip_class || 'tip';
        // Tip is always tracked — move the mark with trajectory when present.
        const box = boxFollowTrajectory(tip, trajectoryData?.tip, currentPosMs);
        drawContactMarkBox(ctx, box, vw, vh, w, h, 'rgba(80,180,255,0.95)', label);
      }
      const primary = contactMarksData.primary;
      if (primary) {
        const label = primary.class || 'contact';
        // Fixed contact stays put; tracked contact follows partner trajectory.
        const box = primary.fixed
          ? primary
          : boxFollowTrajectory(primary, trajectoryData?.partner || trajectoryData?.tip, currentPosMs);
        drawContactMarkBox(ctx, box, vw, vh, w, h, 'rgba(242,176,61,0.95)', label);
      }
      const extras = contactMarksData.extras || [];
      for (let i = 0; i < extras.length; i++) {
        const e = extras[i];
        const label = e.class || `extra ${i + 1}`;
        // Marks S2: same rule as primary — Fixed stays put; else follow path.
        // (omitempty Fixed:false omitted from JSON must still follow.)
        const box = e.fixed
          ? e
          : boxFollowTrajectory(e, trajectoryData?.partner || trajectoryData?.tip, currentPosMs);
        drawContactMarkBox(ctx, box, vw, vh, w, h, 'rgba(220,80,200,0.9)', label);
      }
    }
  }

  // Center a contact mark box on the nearest trajectory sample (same w/h).
  // Root cause of "mark stays fixed": Play drew metadata start rects only.
  function boxFollowTrajectory(box, points, atMs) {
    if (!box || !points || !points.length) return box;
    const pt = nearestTrajectoryPoint(points, atMs);
    if (!pt) return box;
    return {
      ...box,
      x: Math.round(pt.x - (box.w || 0) / 2),
      y: Math.round(pt.y - (box.h || 0) / 2),
    };
  }

  function applyContactMarksInfo(info) {
    contactMarksData = info?.contactMarks || null;
    const row = el('#pb-contact-marks-row');
    const hint = el('#pb-contact-marks-hint');
    const has = !!(contactMarksData && (contactMarksData.tip || contactMarksData.primary
      || (contactMarksData.extras && contactMarksData.extras.length)
      || contactMarksData.tip_class));
    if (row) row.style.display = (videoPath && has) ? 'flex' : 'none';
    if (hint) {
      if (videoPath && has) {
        const tip = contactMarksData.tip_class ? `Tip: ${contactMarksData.tip_class}` : '';
        const n = (contactMarksData.tip ? 1 : 0)
          + (contactMarksData.primary ? 1 : 0)
          + ((contactMarksData.extras && contactMarksData.extras.length) || 0);
        const drive = contactMarksData.drive_stroke
          ? 'distance drives stroke'
          : 'feel only (tip CSRT stroke)';
        hint.textContent = [tip, n ? `${n} marked box${n === 1 ? '' : 'es'}` : '', drive]
          .filter(Boolean).join(' · ');
        hint.style.display = 'block';
      } else {
        hint.style.display = 'none';
        hint.textContent = '';
      }
    }
    redrawTrajectory();
  }

  // loadTrajectory holt die optionale MT-Debug-Trajektorie fürs geladene
  // Skript (null, wenn ohne "Record tip/partner trajectory" erzeugt) -
  // eigener Aufruf statt Teil von loadScript()'s großem try/catch-Block,
  // damit ein Fehlschlag hier nie den restlichen Skript-Ladevorgang stört.
  async function loadTrajectory() {
    trajectoryData = null;
    if (videoPath) {
      try {
        const data = await GetTrajectory();
        trajectoryData = (data && Array.isArray(data.tip) && data.tip.length >= 2) ? data : null;
      } catch (err) {
        trajectoryData = null;
      }
    }
    const hint = el('#pb-trajectory-hint');
    if (hint) hint.style.display = (el('#pb-trajectory-toggle').checked && !trajectoryData) ? 'block' : 'none';
    redrawTrajectory();
  }
  // drawHeatmap zeichnet die grob gerasterte Intensityskurve als
  // Farbverlauf (blau=ruhig -> rot=intensiv) - dieselbe Idee wie die
  // Heatmap-Leisten in MultiFunPlayer & Co, zeigt auf einen Blick, wo im
  // Skript viel/wenig passiert. Ein markierter Bereich wird als heller
  // Rahmen darüber gezeichnet.
  function redrawHeatmap() {
    if (!heatmapPoints || heatmapPoints.length === 0) return;
    const ctx = heatmapCanvas.getContext('2d');
    const w = heatmapCanvas.width, h = heatmapCanvas.height;
    ctx.clearRect(0, 0, w, h);
    const span = viewSpanMs();
    for (let i = 0; i < heatmapPoints.length; i++) {
      const p = heatmapPoints[i];
      const at = Number(p.atMs ?? p.AtMs ?? 0);
      const nextAt = i + 1 < heatmapPoints.length
        ? Number(heatmapPoints[i + 1].atMs ?? heatmapPoints[i + 1].AtMs ?? at)
        : at + (totalMs / Math.max(1, heatmapPoints.length));
      if (nextAt < viewStartMs || at > viewEndMs) continue;
      const x0 = timeToX(at, w);
      const x1 = timeToX(Math.max(at + 1, nextAt), w);
      const intensity = Math.max(0, Math.min(1, p.intensity));
      const hue = 220 - intensity * 220;
      ctx.fillStyle = `hsl(${hue}, 75%, ${35 + intensity * 20}%)`;
      ctx.fillRect(x0, 0, Math.max(1, x1 - x0) + 1, h);
    }
    drawFeelSegmentBands(ctx, w, h, { fullHeight: true });
    drawOMarkerBands(ctx, w, h);
    if (marker && totalMs > 0) {
      const x0 = timeToX(marker.startMs, w);
      const x1 = timeToX(marker.endMs, w);
      ctx.strokeStyle = '#ffffff';
      ctx.lineWidth = 2;
      ctx.strokeRect(x0, 1, Math.max(1, x1 - x0), h - 2);
    }
    // Aktuelle Position als senkrechter Strich - macht die Leiste zur
    // vollwertigen Zeitleiste, nicht nur zur Übersicht.
    if (currentPosMs >= 0 && totalMs > 0
        && currentPosMs >= viewStartMs && currentPosMs <= viewEndMs) {
      const x = timeToX(currentPosMs, w);
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
  function curveXOf(ms) { return timeToX(ms, curveCanvas.width); }
  function curveYOf(pos) {
    const usableH = curveCanvas.height - CURVE_PAD * 2;
    return CURVE_PAD + (1 - pos / 100) * usableH;
  }
  function curveMsOfX(clientX) {
    const rect = curveCanvas.getBoundingClientRect();
    const frac = Math.max(0, Math.min(1, (clientX - rect.left) / rect.width));
    return fracToMs(frac);
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
  // Editor-Trefferpunkte) bleiben exakt to ihrer echten Position - nur
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
    // mx/my kommen aus curveMouseXY() in Canvas-Pixeln (= physische Pixel
    // seit sizeCanvasForDPR) - EDIT_HIT_RADIUS_PX ist als CSS-Pixel-Wert
    // gedacht, sonst schrumpft der Trefferradius auf HiDPI/Retina effektiv
    // auf EDIT_HIT_RADIUS_PX/devicePixelRatio CSS-Pixel.
    let best = -1, bestDist = EDIT_HIT_RADIUS_PX * (window.devicePixelRatio || 1);
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
  // erhalten: sie gibt den groben Überblick, die Kurve die genaue Form.
  // FunGen-like: soft curve + keyframe dots always (edit mode = drag those dots).
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

    drawFeelSegmentBands(ctx, w, h, { fullHeight: false });
    drawBpmGrid(ctx, w, h);
    drawOMarkerBands(ctx, w, h);
    // Markierter Bereich (dieselbe Markierung wie in der Heatmap).
    if (marker) {
      ctx.fillStyle = 'rgba(0,200,255,0.12)';
      ctx.fillRect(xOf(marker.startMs), 0, xOf(marker.endMs) - xOf(marker.startMs), h);
    }

    // OFS-Stil: Abschnitte über Community-Intensitysschwelle (Max-Speed).
    if (speedHighlightsEnabled() && Array.isArray(speedHighlights) && speedHighlights.length > 0) {
      ctx.fillStyle = 'rgba(239, 95, 95, 0.22)';
      for (const seg of speedHighlights) {
        const x0 = xOf(seg.fromMs ?? seg.FromMs ?? 0);
        const x1 = xOf(seg.toMs ?? seg.ToMs ?? 0);
        if (x1 > x0) ctx.fillRect(x0, 0, x1 - x0, h);
      }
    }

    // Soft stroke curve (Catmull-Rom) — FunGen-like “schwingen”.
    ctx.strokeStyle = '#5fd0c8';
    ctx.lineWidth = 1.5;
    ctx.lineJoin = 'round';
    drawSmoothCurve(ctx, points, xOf, yOf);

    // Zweite Spur: Contact vibration (0–1 → 0–100), aus demselben
    // Abstandssignal wie die Playback — liegt unter der Positionsspur.
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

    // Keyframe dots — always on (FunGen2-style); brighter / larger when editing.
    const dots = (editMode && rawActions) ? rawActions : keyframeDots;
    if (dots && dots.length) {
      const step = dots.length > DOT_MAX_DRAW
        ? Math.ceil(dots.length / DOT_MAX_DRAW) : 1;
      for (let i = 0; i < dots.length; i += step) {
        const isDragged = editMode && i === editDragIndex;
        ctx.beginPath();
        ctx.arc(xOf(dots[i].atMs), yOf(dots[i].pos), isDragged ? 5 : (editMode ? 3.5 : 2.2), 0, Math.PI * 2);
        ctx.fillStyle = isDragged ? '#ffcc55' : (editMode ? '#ffffff' : 'rgba(255,255,255,0.85)');
        ctx.fill();
      }
      // Always draw the active drag index even if subsampled away.
      if (editMode && editDragIndex != null && editDragIndex < dots.length && editDragIndex % step !== 0) {
        const d = dots[editDragIndex];
        ctx.beginPath();
        ctx.arc(xOf(d.atMs), yOf(d.pos), 5, 0, Math.PI * 2);
        ctx.fillStyle = '#ffcc55';
        ctx.fill();
      }
    }

    // Positionszeiger + live height marker (“schwingen” playhead).
    if (totalMs > 0) {
      const x = Math.round(xOf(Math.max(0, currentPosMs))) + 0.5;
      if (currentPosMs > 0) {
        ctx.strokeStyle = '#ffffff';
        ctx.lineWidth = 1.5;
        ctx.beginPath();
        ctx.moveTo(x, 0);
        ctx.lineTo(x, h);
        ctx.stroke();
      }
      const livePos = interpPosAt(points, currentPosMs);
      if (livePos != null && currentPosMs > 0) {
        ctx.beginPath();
        ctx.arc(x, yOf(livePos), 5, 0, Math.PI * 2);
        ctx.fillStyle = '#5fd0c8';
        ctx.strokeStyle = '#ffffff';
        ctx.lineWidth = 1.25;
        ctx.fill();
        ctx.stroke();
      }
      updatePbPosOverlay(livePos);
    } else {
      updatePbPosOverlay(null);
    }
  }

  function interpPosAt(points, tMs) {
    if (!points || points.length < 1) return null;
    if (tMs <= points[0].atMs) return points[0].pos;
    const last = points[points.length - 1];
    if (tMs >= last.atMs) return last.pos;
    for (let i = 1; i < points.length; i++) {
      const a = points[i - 1], b = points[i];
      if (tMs >= a.atMs && tMs <= b.atMs) {
        if (b.atMs === a.atMs) return b.pos;
        const f = (tMs - a.atMs) / (b.atMs - a.atMs);
        return a.pos + f * (b.pos - a.pos);
      }
    }
    return last.pos;
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
      logError('Offset: ' + err);
    }
  }

  el('#pb-offset').addEventListener('change', e => applyOffset(Number(e.target.value) || 0));
  el('#pb-offset-minus').addEventListener('click',
    () => applyOffset((Number(el('#pb-offset').value) || 0) - 50));
  el('#pb-offset-plus').addEventListener('click',
    () => applyOffset((Number(el('#pb-offset').value) || 0) + 50));
  el('#pb-offset-reset').addEventListener('click', () => applyOffset(0));
  let pbContactVibPreviewTimer = 0;
  async function refreshPlayContactVibPreview() {
    const out = el('#pb-contact-vib-preview');
    const vibPoly = el('#pb-contact-vib-poly');
    const strokePoly = el('#pb-contact-vib-stroke');
    const peaksG = el('#pb-contact-vib-peaks');
    const badgeActive = el('#pb-cv-badge-active');
    const badgePeak = el('#pb-cv-badge-peak');
    const probe = el('#pb-contact-vib-probe');
    const block = el('#pb-contact-block');
    if (!out || !probe || !block || block.hidden) return;
    if (el('#pb-contact-off')?.checked) {
      probe.hidden = true;
      return;
    }
    probe.hidden = false;
    const span = parseFloat(el('#pb-contact-span')?.value) || 0.75;
    const curve = el('#pb-contact-curve')?.value || 'soft';
    // finiteOr (not ||): Strength slider min=0 must stay 0 in the probe.
    const intensity = Math.max(0, finiteOr(el('#pb-contact-intensity')?.value, 1));
    try {
      const res = await PreviewContactVibration({ span, curve });
      let hint = res.hint || res.Hint || 'Contact probe';
      if (intensity !== 1) hint += ` · strength ×${intensity.toFixed(2)}`;
      out.textContent = hint;
      const activePct = Number(res.activePct ?? res.ActivePct ?? 0);
      const peakVib = Number(res.peakVib ?? res.PeakVib ?? 0) * intensity;
      if (badgeActive) badgeActive.textContent = `${Math.round(activePct)}% active`;
      if (badgePeak) badgePeak.textContent = `peak ${Math.round(Math.min(100, peakVib * 100))}`;
      const sample = res.sample || res.Sample || [];
      const w = 320;
      const h = 72;
      const pad = 4;
      if (peaksG) peaksG.innerHTML = '';
      if (Array.isArray(sample) && sample.length >= 2) {
        const t0 = Number(sample[0].atMs ?? sample[0].AtMs ?? 0);
        const t1 = Number(sample[sample.length - 1].atMs ?? sample[sample.length - 1].AtMs ?? 1);
        const tSpan = Math.max(1, t1 - t0);
        const vibOf = (p) => Math.max(0, Math.min(100, Number(p.vib ?? p.Vib ?? 0) * intensity));
        const toPts = (key, Key, scale = 1) => sample.map((p) => {
          const t = Number(p.atMs ?? p.AtMs ?? 0);
          let pos = Math.max(0, Math.min(100, Number(p[key] ?? p[Key] ?? 0)));
          if (key === 'vib' || key === 'Vib') {
            pos = Math.max(0, Math.min(100, pos * scale));
          }
          const x = pad + ((t - t0) / tSpan) * (w - 2 * pad);
          const y = pad + (1 - pos / 100) * (h - 2 * pad);
          return `${x.toFixed(1)},${y.toFixed(1)}`;
        }).join(' ');
        if (strokePoly) strokePoly.setAttribute('points', toPts('stroke', 'Stroke', 1));
        if (vibPoly) vibPoly.setAttribute('points', toPts('vib', 'Vib', intensity));
        if (peaksG) {
          for (let i = 1; i < sample.length - 1; i++) {
            const v = vibOf(sample[i]);
            if (v < 15) continue;
            if (v < vibOf(sample[i - 1]) || v < vibOf(sample[i + 1])) continue;
            const t = Number(sample[i].atMs ?? sample[i].AtMs ?? 0);
            const x = pad + ((t - t0) / tSpan) * (w - 2 * pad);
            const y = pad + (1 - v / 100) * (h - 2 * pad);
            const c = document.createElementNS('http://www.w3.org/2000/svg', 'circle');
            c.setAttribute('cx', x.toFixed(1));
            c.setAttribute('cy', y.toFixed(1));
            c.setAttribute('r', '2.4');
            peaksG.appendChild(c);
          }
        }
      } else {
        if (strokePoly) strokePoly.setAttribute('points', '');
        if (vibPoly) vibPoly.setAttribute('points', '');
      }
    } catch (err) {
      out.textContent = 'Contact probe unavailable: ' + err;
      if (strokePoly) strokePoly.setAttribute('points', '');
      if (vibPoly) vibPoly.setAttribute('points', '');
      if (peaksG) peaksG.innerHTML = '';
      if (badgeActive) badgeActive.textContent = '— active';
      if (badgePeak) badgePeak.textContent = '— peak';
    }
  }
  function schedulePlayContactVibPreview() {
    clearTimeout(pbContactVibPreviewTimer);
    pbContactVibPreviewTimer = setTimeout(refreshPlayContactVibPreview, 160);
  }
  function openPlayContactProbe() {
    const tune = el('#pb-contact-tune-details');
    // Do not re-force-open tune if the user collapsed it this session.
    if (tune && !tune.dataset.userCollapsed) tune.open = true;
    const d = el('#pb-contact-probe-details');
    if (d && (!tune || tune.open)) d.open = true;
  }

  el('#pb-contact-tune-details')?.addEventListener('toggle', (e) => {
    if (!e.target.open) e.target.dataset.userCollapsed = '1';
    else delete e.target.dataset.userCollapsed;
  });

  el('#pb-contact-intensity').addEventListener('input', e => {
    // finiteOr (not ||): Strength min=0 must stay 0 in the label + prefs.
    const v = finiteOr(e.target.value, 0);
    el('#pb-contact-intensity-val').textContent = v.toFixed(2);
    if (scriptHasContactVibration) drawCurve();
    openPlayContactProbe();
    schedulePlayContactVibPreview();
  });
  el('#pb-contact-intensity').addEventListener('change', e => {
    saveSetting('playback.contact_intensity', finiteOr(e.target.value, 1));
  });
  el('#pb-contact-span').addEventListener('input', e => {
    const v = Number(e.target.value) || 0;
    el('#pb-contact-span-val').textContent = v.toFixed(2);
    if (scriptHasContactVibration) drawCurve();
    openPlayContactProbe();
    schedulePlayContactVibPreview();
  });
  el('#pb-contact-curve').addEventListener('change', () => {
    if (scriptHasContactVibration) drawCurve();
    openPlayContactProbe();
    schedulePlayContactVibPreview();
  });
  el('#pb-contact-off').addEventListener('change', () => {
    if (scriptHasContactVibration) drawCurve();
    schedulePlayContactVibPreview();
  });
  el('#pb-contact-save').addEventListener('click', async () => {
    if (!scriptPath || !scriptHasContactVibration) return;
    const status = el('#pb-contact-save-status');
    status.textContent = 'Saving…';
    try {
      await SaveContactSettings(
        !el('#pb-contact-off').checked,
        parseFloat(el('#pb-contact-span').value) || 0.75,
        el('#pb-contact-curve').value || 'soft',
      );
      status.textContent = 'Saved to script.';
      await refreshScriptVisuals();
    } catch (err) {
      uiError('Save contact: ' + err, status);
      log('Save contact: ' + err);
    }
  });

  const CHAPTER_LABELS = {
    pause: 'Pause', build: 'Build-up', steady: 'steady',
    crescendo: 'Crescendo', winddown: 'Wind-down',
  };

  function formatMs(ms) {
    const s = Math.round(ms / 1000);
    return Math.floor(s / 60) + ':' + String(s % 60).padStart(2, '0');
  }

  function formatTooltipTime(ms) {
    const sec = Math.max(0, ms) / 1000;
    if (sec < 60) return sec.toFixed(1) + 's';
    const m = Math.floor(sec / 60);
    const sWhole = Math.floor(sec % 60);
    const tenths = Math.round((sec % 1) * 10);
    return m + ':' + String(sWhole).padStart(2, '0') + '.' + tenths;
  }

  function hideChartTooltip() {
    if (chartTooltip) chartTooltip.hidden = true;
  }

  function showChartTooltip(e, text) {
    if (!chartTooltip) return;
    chartTooltip.textContent = text;
    chartTooltip.hidden = false;
    const pad = 12;
    const rect = chartTooltip.getBoundingClientRect();
    let left = e.clientX + pad;
    let top = e.clientY + pad;
    if (left + rect.width > window.innerWidth - 4) left = e.clientX - rect.width - pad;
    if (top + rect.height > window.innerHeight - 4) top = e.clientY - rect.height - pad;
    chartTooltip.style.left = left + 'px';
    chartTooltip.style.top = top + 'px';
  }

  function curvePosAtMs(atMs, points) {
    if (!points || points.length === 0) return 0;
    if (atMs <= points[0].atMs) return points[0].pos;
    const last = points[points.length - 1];
    if (atMs >= last.atMs) return last.pos;
    for (let i = 0; i < points.length - 1; i++) {
      const a = points[i];
      const b = points[i + 1];
      if (atMs >= a.atMs && atMs <= b.atMs) {
        const span = Math.max(1, b.atMs - a.atMs);
        const t = (atMs - a.atMs) / span;
        return a.pos + t * (b.pos - a.pos);
      }
    }
    return last.pos;
  }

  function heatmapIntensityAtMs(atMs) {
    if (!heatmapPoints || heatmapPoints.length === 0) return null;
    let best = heatmapPoints[0];
    let bestDist = Math.abs(best.atMs - atMs);
    for (let i = 1; i < heatmapPoints.length; i++) {
      const p = heatmapPoints[i];
      const d = Math.abs(p.atMs - atMs);
      if (d < bestDist) { bestDist = d; best = p; }
    }
    return best.intensity;
  }

  function updateHeatmapChartTooltip(e) {
    if (!heatmapPoints || heatmapPoints.length === 0 || !totalMs) {
      hideChartTooltip();
      return;
    }
    const atMs = canvasXToMs(e.clientX);
    const intensity = heatmapIntensityAtMs(atMs);
    if (intensity == null) {
      hideChartTooltip();
      return;
    }
    const pct = Math.round(Math.max(0, Math.min(1, intensity)) * 100);
    showChartTooltip(e, formatTooltipTime(atMs) + ' · ' + pct + '%');
  }

  function updateCurveChartTooltip(e) {
    const points = (editMode && rawActions) ? rawActions : curvePoints;
    if (!points || points.length < 2 || !totalMs) {
      hideChartTooltip();
      return;
    }
    const atMs = Math.max(0, Math.min(totalMs, curveMsOfX(e.clientX)));
    const pos = curvePosAtMs(atMs, points);
    const posLabel = Math.round(Math.max(0, Math.min(100, pos)));
    showChartTooltip(e, formatTooltipTime(atMs) + ' · ' + posLabel);
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
        text += (text ? ' · Chapters: ' : 'Chapters: ') + parts.join(', ');
      }
    } catch (err) {
      // zu wenige Punkte o.ä. - kein Kapitelabschnitt, nicht fatal
    }
    box.textContent = text;
    box.style.display = text ? 'block' : 'none';
    const sum = el('#pb-quality-details > summary');
    if (sum) {
      sum.textContent = text
        ? 'Script check & analysis · summary ready'
        : 'Script check & analysis';
    }
  }

  function contactPreviewOpts() {
    return {
      maxPoints: CURVE_MAX_POINTS,
      contactVibrationSpan: finiteOr(el('#pb-contact-span').value, 0),
      contactVibrationCurve: el('#pb-contact-curve').value || '',
      contactIntensityScale: finiteOr(el('#pb-contact-intensity').value, 1),
      muteContact: el('#pb-contact-off').checked,
    };
  }

  /** parseFloat that keeps 0 (unlike `x || fallback`). */
  function finiteOr(raw, fallback) {
    const n = parseFloat(raw);
    return Number.isFinite(n) ? n : fallback;
  }

  async function drawCurve() {
    try {
      if (editAxis && editAxis !== 'general') {
        const acts = await GetScriptAxisActions(editAxis);
        const pts = Array.isArray(acts) ? acts : [];
        curvePoints = pts.length
          ? pts.map(a => ({ atMs: a.at, pos: a.pos }))
          : await GetScriptCurve(CURVE_MAX_POINTS);
      } else {
        curvePoints = await GetScriptCurve(CURVE_MAX_POINTS);
      }
    } catch (err) {
      curvePoints = null;
      keyframeDots = null;
      vibrationCurvePoints = null;
      speedHighlights = [];
      curveCanvas.style.display = 'none';
      el('#pb-curve-edit-row').style.display = 'none';
      updatePbPosOverlay(null);
      return;
    }
    if (!curvePoints || curvePoints.length < 2) {
      keyframeDots = null;
      vibrationCurvePoints = null;
      speedHighlights = [];
      curveCanvas.style.display = 'none';
      el('#pb-curve-edit-row').style.display = 'none';
      updatePbPosOverlay(null);
      return;
    }
    // FunGen-like dots: load full keyframes even when not editing.
    if (!editMode) {
      try {
        const acts = await GetScriptAxisActions(editAxis || 'general');
        keyframeDots = (Array.isArray(acts) ? acts : []).map(a => ({ atMs: a.at, pos: a.pos }));
        if (!keyframeDots.length) keyframeDots = null;
      } catch (err) {
        keyframeDots = null;
      }
    }
    try {
      vibrationCurvePoints = scriptHasContactVibration
        ? await GetVibrationCurvePreview(contactPreviewOpts())
        : null;
    } catch (err) {
      vibrationCurvePoints = null;
    }
    await refreshSpeedHighlights();
    curveCanvas.style.display = 'block';
    el('#pb-curve-edit-row').style.display = 'flex';
    sizeCanvasForDPR(curveCanvas, 800, 176);
    redrawCurve();
  }

  function speedHighlightsEnabled() {
    const box = el('#pb-speed-hl');
    return !box || !!box.checked;
  }
  function speedHighlightThresh() {
    const v = parseFloat(el('#pb-speed-hl-thresh')?.value);
    if (!Number.isFinite(v) || v <= 0) return 400;
    return Math.max(50, Math.min(2000, v));
  }
  function updateSpeedHighlightLabel() {
    const lab = el('#pb-speed-hl-thresh-val');
    if (lab) lab.textContent = `${Math.round(speedHighlightThresh())}`;
  }
  async function refreshSpeedHighlights() {
    if (!scriptPath) {
      speedHighlights = [];
      return;
    }
    if (!speedHighlightsEnabled()) {
      speedHighlights = [];
      redrawCurve();
      return;
    }
    try {
      const hl = await GetSpeedHighlights(speedHighlightThresh());
      speedHighlights = Array.isArray(hl) ? hl : [];
    } catch (err) {
      speedHighlights = [];
    }
    redrawCurve();
  }

  // Klick in die Kurve springt to die Stelle - gleiche Bedienung wie die
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
    if (editMode && editDragIndex !== null) {
      let atMs = Math.max(0, Math.min(totalMs, curveMsOfX(e.clientX)));
      // Nicht über die direkten Nachbarn hinausziehen lassen - sonst
      // "springt" der Punkt beim Loslassen im Index, weil persistRawActions
      // danach chronologisch neu sortiert (siehe dort).
      const prev = rawActions[editDragIndex - 1];
      const next = rawActions[editDragIndex + 1];
      const lower = prev ? prev.atMs + 1 : 0;
      const upper = next ? next.atMs - 1 : totalMs;
      // Nachbarn <=1ms auseinander lassen kein gültiges Intervall übrig
      // (lower > upper) - dann lieber nicht klemmen, statt den Punkt aus
      // Versehen auf einen Wert außerhalb [lower, upper] zu zwingen.
      if (lower <= upper) atMs = Math.max(lower, Math.min(upper, atMs));
      rawActions[editDragIndex] = { atMs, pos: curvePosOfY(e.clientY) };
      redrawCurve();
      hideChartTooltip();
      // Video folgt beim Ziehen mit - man sieht, welcher Moment gerade
      // markiert wird, statt blind auf Zeitwerte zu vertrauen. seekTo() ist
      // bereits ein no-op ohne Video, also kein zusätzlicher Guard nötig.
      seekTo(atMs);
      return;
    }
    updateCurveChartTooltip(e);
  });
  curveCanvas.addEventListener('mouseleave', hideChartTooltip);

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
      log('Editor: at least 2 points must remain in the script.');
      return;
    }
    rawActions.splice(idx, 1);
    redrawCurve();
    await persistRawActions();
  });

  // persistRawActions speichert den aktuellen Punktstand sofort - keine
  // separate "Speichern"-Aktion, dieselbe Sofort-Speicher-Logik wie bei
  // den O-Markern oben. Bei errors (z.B. Datei zwischenzeitlich entfernt)
  // bleibt der bearbeitete Stand im Editor sichtbar, wird aber nicht als
  // gespeichert angenommen - ein erneuter Zug versucht es wieder.
  async function persistRawActions() {
    if (!scriptPath || !rawActions) return;
    const sorted = [...rawActions].sort((a, b) => a.atMs - b.atMs);
    try {
      await SaveScriptAxisActions(editAxis || 'general', sorted.map(p => ({ at: p.atMs, pos: p.pos })));
    } catch (err) {
      logError('Save curve: ' + err);
      return;
    }
    rawActions = sorted;
    keyframeDots = sorted.map(p => ({ ...p }));
    redrawCurve();
    drawHeatmap();
  }

  async function setEditMode(on) {
    if (on) {
      try {
        const actions = await GetScriptAxisActions(editAxis || 'general');
        rawActions = (Array.isArray(actions) ? actions : []).map(a => ({ atMs: a.at, pos: a.pos }));
        if (!rawActions.length) {
          // Empty Neo-2 axis: seed from general so the user can start editing.
          if (editAxis !== 'general') {
            const gen = await GetScriptAxisActions('general');
            rawActions = (Array.isArray(gen) ? gen : []).map(a => ({ atMs: a.at, pos: 0 }));
          }
        }
        keyframeDots = rawActions.map(p => ({ ...p }));
      } catch (err) {
        logError('Editor: failed to load points: ' + err);
        el('#pb-curve-edit').checked = false;
        return;
      }
      editMode = true;
    } else {
      editMode = false;
      editDragIndex = null;
      if (rawActions && rawActions.length) {
        keyframeDots = rawActions.map(p => ({ ...p }));
      }
      rawActions = null;
    }
    el('#pb-curve-edit-hint').style.display = editMode ? 'block' : 'none';
    redrawCurve();
  }

  el('#pb-curve-edit').addEventListener('change', e => setEditMode(e.target.checked));
  el('#pb-pos-overlay-toggle')?.addEventListener('change', () => {
    const on = !!el('#pb-pos-overlay-toggle').checked;
    try { localStorage.setItem(POS_OVERLAY_PREF, on ? '1' : '0'); } catch (_) { /* ignore */ }
    updatePbPosOverlay(null);
  });
  posOverlayWanted();

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
    sizeCanvasForDPR(heatmapCanvas, 800, 56);
    redrawHeatmap();
  }

  function canvasXToMs(clientX) {
    const rect = heatmapCanvas.getBoundingClientRect();
    const frac = Math.max(0, Math.min(1, (clientX - rect.left) / rect.width));
    return fracToMs(frac);
  }

  heatmapCanvas.addEventListener('mousedown', (e) => {
    markerDragStartMs = canvasXToMs(e.clientX);
    markerDragMoved = false;
  });
  heatmapCanvas.addEventListener('mousemove', (e) => {
    if (markerDragStartMs !== null) {
      const cur = canvasXToMs(e.clientX);
      if (Math.abs(cur - markerDragStartMs) > 150) markerDragMoved = true;
      if (!markerDragMoved) {
        hideChartTooltip();
        return;
      }
      hideChartTooltip();
      marker = { startMs: Math.min(markerDragStartMs, cur), endMs: Math.max(markerDragStartMs, cur) };
      redrawHeatmap();
      redrawCurve();
      return;
    }
    updateHeatmapChartTooltip(e);
  });
  heatmapCanvas.addEventListener('mouseleave', hideChartTooltip);
  window.addEventListener('mouseup', () => {
    if (markerDragStartMs === null) return;
    const clickedMs = markerDragStartMs;
    const wasDrag = markerDragMoved;
    markerDragStartMs = null;
    markerDragMoved = false;

    // Klick ohne Ziehen = to diese Stelle springen (statt zu markieren).
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
        .catch(err => log('Save selection: ' + err));
    }
  });

  // seekTo springt im Video to die angegebene Stelle. Bei aktivem
  // Video-Sync folgt das Gerät automatisch mit, da die Videoposition
  // ohnehin die Quelle ist (siehe player/sync.go) - es ist kein
  // zusätzliches Zutun nötig. Ohne Video gibt es nichts zu spulen; dann
  // bleibt der Klick wirkungslos (die eigene Uhr in Play() lässt sich
  // nicht nachträglich verschieben).
  //
  // Curve + heatmap redraw immediately (do not wait for timeupdate — that
  // event can lag or miss while paused). Device sync is pushed on 'seeked'.
  async function seekTo(atMs) {
    let t = atMs;
    const fps = parseFloat(el('#pb-fps-snap')?.value) || 0;
    if (fps > 0) {
      try { t = await SnapTimeMs(t, fps); } catch (_) { /* keep t */ }
    }
    currentPosMs = t;
    if (!videoPath || !videoEl.duration) {
      redrawHeatmap();
      redrawCurve();
      redrawTrajectory();
      return;
    }
    videoEl.currentTime = Math.max(0, Math.min(videoEl.duration, t / 1000));
    // Auto-Extended-O darf nach einem Sprung erneut auslösen, wenn der
    // markierte Bereich danach nochmal erreicht wird.
    if (marker && t < marker.startMs) autoEOTriggeredForMarker = false;
    redrawHeatmap();
    redrawCurve();
    redrawTrajectory();
  }

  el('#pb-marker-clear').addEventListener('click', () => {
    marker = null;
    updateMarkerHint();
    redrawHeatmap();
    if (scriptPath) SaveMarker(scriptPath, 0, 0).catch(err => log('Save selection: ' + err));
  });

  function syncPlayOMarkerIntensityLabel() {
    const inp = el('#pb-omarker-intensity');
    const lab = el('#pb-omarker-intensity-val');
    if (!inp || !lab) return;
    const n = Number(inp.value);
    lab.textContent = Number.isFinite(n) ? (Math.round(n * 100) / 100).toString() : '0.5';
  }
  syncPlayOMarkerIntensityLabel();
  el('#pb-omarker-intensity')?.addEventListener('input', syncPlayOMarkerIntensityLabel);
  el('#pb-omarker-intensity')?.addEventListener('change', syncPlayOMarkerIntensityLabel);

  el('#pb-omarker-kind').addEventListener('change', e => {
    el('#pb-omarker-intensity-row').style.display = e.target.value === 'secondary' ? 'flex' : 'none';
    syncPlayOMarkerIntensityLabel();
  });

  el('#pb-omarker-add').addEventListener('click', async () => {
    if (!markerIsValid(marker) || !scriptPath) return;
    const kind = el('#pb-omarker-kind').value;
    const intensity = kind === 'primary'
      ? 1.0
      : Math.max(0, Math.min(1, Number(el('#pb-omarker-intensity').value) || 0));
    const next = [...oMarkers, { startMs: marker.startMs, endMs: marker.endMs, kind, intensity }];
    try {
      await SaveOMarkers(scriptPath, next);
    } catch (err) {
      logError('Add O-marker: ' + err);
      return;
    }
    oMarkers = next;
    renderOMarkerList();
    redrawHeatmap();
    redrawCurve();
  });

  el('#pb-bookmark-add')?.addEventListener('click', async () => {
    if (!scriptPath) {
      showChooseScriptRetry('Load a script first.', el('#pb-log'));
      return;
    }
    const nameInput = el('#pb-bookmark-name');
    const name = ((nameInput && nameInput.value) || '').trim() || 'Bookmark';
    const time = Math.max(0, Math.round(currentPosMs || 0));
    const previous = bookmarks.slice();
    const next = [...bookmarks, { name, time }];
    if (await persistBookmarks(next, previous)) {
      if (nameInput) nameInput.value = '';
      uiInfo(`Bookmark “${name}” at ${(time / 1000).toFixed(1)}s`, el('#pb-log'));
    }
  });

  el('#pb-chapter-add')?.addEventListener('click', async () => {
    if (!scriptPath) {
      showChooseScriptRetry('Load a script first.', el('#pb-log'));
      return;
    }
    if (!markerIsValid(marker)) {
      uiWarn('Mark a range on the heatmap first.', el('#pb-log'));
      return;
    }
    const nameInput = el('#pb-chapter-name');
    const name = ((nameInput && nameInput.value) || '').trim() || 'Chapter';
    const startTime = Math.round(marker.startMs);
    const endTime = Math.round(marker.endMs);
    const previous = chapterMarks.slice();
    const next = [...chapterMarks, { name, startTime, endTime }];
    if (await persistChapterMarks(next, previous)) {
      if (nameInput) nameInput.value = '';
      uiInfo(`Chapter “${name}” ${formatChapterRange({ startTime, endTime })}`, el('#pb-log'));
    }
  });

  // checkAutoExtendedO wird bei jedem Fortschritts-Update aufgerufen -
  // löst Extended-O einmal pro Playback aus, sobald die Position in den
  // markierten Bereich eintritt (falls aktiviert).
  function checkAutoExtendedO(atMs) {
    if (!markerIsValid(marker) || !el('#pb-marker-auto').checked || autoEOTriggeredForMarker) return;
    if (atMs >= marker.startMs && atMs <= marker.endMs) {
      autoEOTriggeredForMarker = true;
      if (!el('#pb-eo-trigger').disabled) triggerEO();
    }
  }



  // Fallengelassene Skripte: eines oder mehrere → Playlist.
  window.addEventListener('drop:script', e => {
    const paths = (e.detail && e.detail.paths && e.detail.paths.length)
      ? e.detail.paths
      : [e.detail.path];
    replacePlaylist(paths, 0);
    loadScript(paths[0], { keepPlaylist: true }).catch(err => {
      showChooseScriptRetry('Could not load script: ' + err, el('#pb-log'), 'error');
    });
  });

  // C7: load failures and no-script actions name Choose Emotion Script.
  function showChooseScriptRetry(reason, statusEl, level) {
    const msg = String(reason || 'Load a script first.').replace(/\s+/g, ' ').trim();
    if (level === 'error') uiError(msg);
    else uiWarn(msg);
    const host = statusEl || el('#pb-log');
    if (!host) return;
    host.replaceChildren();
    const text = document.createElement('span');
    text.textContent = msg + ' ';
    const btn = document.createElement('button');
    btn.type = 'button';
    btn.className = 'secondary pb-choose-retry';
    btn.textContent = 'Choose Emotion Script…';
    btn.addEventListener('click', () => { chooseScript(); });
    host.append(text, btn);
  }

  async function chooseScript() {
    const path = await PickFunscriptFile();
    restoreKeyboardFocus();
    if (!path) return;
    replacePlaylist([path], 0);
    try {
      await loadScript(path, { keepPlaylist: true });
    } catch (err) {
      showChooseScriptRetry('Could not load script: ' + err, el('#pb-log'), 'error');
    }
  }

  async function queueAddScript() {
    const path = await PickFunscriptFile();
    restoreKeyboardFocus();
    if (!path) return;
    appendToPlaylist(path);
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

  // Force the <video> element to pick up a new stream. VideoFileURL is a
  // fixed localhost path; even with ?v=epoch, clearing first avoids stale
  // decode buffers when the user swaps films after the first load.
  function setVideoSrc(url) {
    const next = (url || '').trim();
    videoEl.pause();
    videoEl.removeAttribute('src');
    videoEl.load();
    if (!next) return;
    videoEl.src = next;
  }

  async function attachVideoFromPicker() {
    if (!scriptPath) return;
    try {
      const path = await PickVideoFile();
      if (!path) return;
      const url = await SetPlaybackVideo(path);
      videoPath = path;
      setVideoSrc(url);
      const stage = el('#pb-video-stage');
      stage.classList.add('has-video');
      stage.classList.remove('no-video');
      el('#pb-video-sync-row').style.display = 'flex';
      el('#pb-video-autostart-row').style.display = 'flex';
      log('Video linked: ' + path);
      await refreshVideoPlayability(path);
    } catch (err) {
      uiError('Link video: ' + err);
    }
  }

  async function refreshVideoPlayability(path) {
    const warn = el('#pb-video-warn');
    const conv = el('#pb-video-convert');
    if (!warn || !conv) return;
    try {
      const info = await ProbePlaybackVideo(path || '');
      if (info.likelyPlayable && !info.warning) {
        warn.hidden = true;
        conv.hidden = true;
        return;
      }
      warn.hidden = false;
      warn.textContent = info.warning
        || (`Codec ${info.codec || '?'} — playback may fail`);
      conv.hidden = false;
    } catch (err) {
      warn.hidden = false;
      warn.textContent = 'Could not read video metadata: ' + err;
      conv.hidden = false;
    }
  }

  async function convertPlaybackVideo() {
    const conv = el('#pb-video-convert');
    const warn = el('#pb-video-warn');
    if (conv) conv.disabled = true;
    if (warn) {
      warn.hidden = false;
      warn.textContent = 'Converting to H.264/AAC (may take a while)…';
    }
    try {
      const info = await EnsurePlayablePlaybackVideo();
      videoPath = info.path;
      setVideoSrc(await VideoFileURL());
      if (warn) {
        warn.textContent = info.usingProxy
          ? ('Playable copy ready' + (info.proxyPath ? ': ' + info.proxyPath.split(/[\\/]/).pop() : ''))
          : 'Video should be playable now.';
      }
      if (conv) conv.hidden = true;
      uiInfo('Video prepared for the player.');
    } catch (err) {
      uiError('Conversion: ' + err, warn);
    } finally {
      if (conv) conv.disabled = false;
    }
  }

  function toggleVideoFullscreen() {
    const stage = el('#pb-video-stage');
    if (!stage.classList.contains('has-video')) return;
    const on = stage.classList.toggle('is-fs');
    el('#pb-video-fs').textContent = on ? 'Exit fullscreen' : 'Fullscreen';
    redrawTrajectory();
  }

  async function refreshSamnControls(info) {
    scriptNativeFormat = !!info.nativeFormat;
    scriptHasNeoAxes = !!info.hasNeoAxes;
    const row = el('#pb-axis-row');
    if (row) row.style.display = 'flex';
    const bakeShare = el('#pb-export-bake-details');
    if (bakeShare) bakeShare.style.display = 'block';
    if (el('#pb-playback-source')) {
      el('#pb-playback-source').value = info.playbackSource === 'axes' ? 'axes' : 'recipe';
    }
    const strength = el('#pb-strength');
    if (strength) {
      strength.innerHTML = '<option value="">—</option>';
      try {
        const pack = await GetStrengthPresets();
        const presets = (pack && pack.presets) || [];
        const active = (pack && pack.active) || '';
        for (const p of presets) {
          const opt = document.createElement('option');
          opt.value = p.name;
          opt.textContent = p.name;
          if (p.name === active) opt.selected = true;
          strength.appendChild(opt);
        }
        strength.disabled = presets.length === 0;
      } catch (_) {
        strength.disabled = true;
      }
    }
  }

  async function loadScript(path, extraCountOrOpts = 0, opts = {}) {
    // Rückwärtskompatibel: früher (path, extraCount), jetzt auch (path, opts).
    if (extraCountOrOpts && typeof extraCountOrOpts === 'object') {
      opts = extraCountOrOpts;
    }
    let info;
    try {
      info = await LoadFunscript(path);
    } catch (err) {
      showChooseScriptRetry('Could not load script: ' + err, el('#pb-log'), 'error');
      throw err;
    }
    scriptPath = info.path;
    totalMs = Math.max(info.durationMs, 1);
    resetViewWindow();
    if (!opts.keepPlaylist) {
      replacePlaylist([scriptPath], 0);
    } else {
      const idx = playlist.findIndex(p => p.path === scriptPath);
      if (idx >= 0) playlistIndex = idx;
      renderPlaylist();
    }
    el('#pb-script-path').textContent = scriptPath;
    setScriptLoaded(true);
    scriptHasContactVibration = !!info.contactVibration;
    await refreshSamnControls(info);
    // Contact controls stay available in axes mode (save re-bakes vibe).
    const showContact = scriptHasContactVibration || info.hasNeoAxes || info.profile === 'tj' || info.profile === 'tf';
    el('#pb-contact-block').hidden = !showContact;
    if (el('#pb-contact-save-status')) el('#pb-contact-save-status').textContent = '';
    if (!showContact) {
      el('#pb-contact-off').checked = false;
      const probe = el('#pb-contact-vib-probe');
      if (probe) probe.hidden = true;
    } else {
      // Rezept-Defaults in die Live-Controls übernehmen (Datei bleibt Quelle).
      // Contact strength stays from settings/DOM (session preference; not in recipe).
      const span = (info.contactVibrationSpan > 0) ? info.contactVibrationSpan : 0.75;
      el('#pb-contact-span').value = String(span);
      el('#pb-contact-span-val').textContent = Number(span).toFixed(2);
      const curve = info.contactVibrationCurve || 'soft';
      el('#pb-contact-curve').value = ['linear', 'soft', 'peak', 'impulse'].includes(curve) ? curve : 'soft';
      schedulePlayContactVibPreview();
    }
    const stage = el('#pb-video-stage');
    if (info.hasVideo) {
      videoPath = info.videoPath;
      setVideoSrc(await VideoFileURL());
      stage.classList.add('has-video');
      stage.classList.remove('no-video');
      el('#pb-video-sync-row').style.display = 'flex';
      el('#pb-video-autostart-row').style.display = 'flex';
      el('#pb-trajectory-row').style.display = 'flex';
      refreshVideoPlayability(videoPath);
    } else {
      videoPath = null;
      setVideoSrc('');
      stage.classList.remove('has-video', 'is-fs', 'is-playing');
      stage.classList.add('no-video');
      el('#pb-video-sync-row').style.display = 'none';
      el('#pb-video-autostart-row').style.display = 'none';
      el('#pb-trajectory-row').style.display = 'none';
      el('#pb-trajectory-hint').style.display = 'none';
      if (el('#pb-contact-marks-row')) el('#pb-contact-marks-row').style.display = 'none';
      if (el('#pb-contact-marks-hint')) el('#pb-contact-marks-hint').style.display = 'none';
      el('#pb-video-fs').textContent = 'Fullscreen';
      const warn = el('#pb-video-warn');
      const conv = el('#pb-video-convert');
      if (warn) warn.hidden = true;
      if (conv) conv.hidden = true;
      try { ClearPlaybackVideo().catch(() => {}); } catch (_) {}
    }
    applyContactMarksInfo(info);
    rememberAudioSegmentsFromInfo(info);
    try {
      marker = await GetMarker(scriptPath);
      if (!markerIsValid(marker)) marker = null;
    } catch (err) {
      marker = null;
    }
    try {
      const result = await GetOMarkers(scriptPath);
      oMarkers = Array.isArray(result) ? result : [];
    } catch (err) {
      oMarkers = [];
    }
    try {
      const bm = await GetScriptBookmarks();
      bookmarks = Array.isArray(bm)
        ? bm.map(normalizeBookmark).filter(Boolean)
        : [];
    } catch (err) {
      bookmarks = [];
    }
    try {
      const ch = await GetScriptChapterMarks();
      chapterMarks = Array.isArray(ch)
        ? ch.map(normalizeChapterMark).filter(Boolean)
        : [];
    } catch (err) {
      chapterMarks = [];
    }
    updateMarkerHint();
    renderOMarkerList();
    renderBookmarkList();
    renderChapterList();
    // Ein neu geladenes Skript hat andere Punkte - ein noch aktiver
    // Editiermodus vom vorherigen Skript würde sonst dessen (falsche)
    // rawActions weiterbenutzen.
    // review: show FunGen-like dots + contact focus — Edit stays off until
    // the user checks “Edit curve (dots)” (avoids accidental saves).
    el('#pb-curve-edit').checked = false;
    await setEditMode(false);
    drawHeatmap();
    drawCurve();
    describeScript();
    loadTrajectory();
    el('#pb-omarker-details') && (el('#pb-omarker-details').style.display = 'block');
    el('#pb-omarker-hint').style.display = 'block';
    el('#pb-omarker-add-row').style.display = 'flex';
    el('#pb-bookmark-details') && (el('#pb-bookmark-details').style.display = 'block');
    el('#pb-bookmark-hint').style.display = 'block';
    el('#pb-bookmark-add-row').style.display = 'flex';
    el('#pb-chapter-details') && (el('#pb-chapter-details').style.display = 'block');
    el('#pb-chapter-hint').style.display = 'block';
    el('#pb-chapter-add-row').style.display = 'flex';
    el('#pb-script-doctor-row').style.display = 'flex';
    el('#pb-quality-details') && (el('#pb-quality-details').style.display = 'block');
    el('#pb-ofs-row').style.display = 'flex';
    el('#pb-ofs-edit-details') && (el('#pb-ofs-edit-details').style.display = 'block');
    el('#pb-script-doctor-result').style.display = 'none';
    el('#pb-script-doctor-status').textContent = '';
    if (el('#pb-ofs-status')) el('#pb-ofs-status').textContent = '';
    el('#pb-offset-row').style.display = 'flex';
    el('#pb-offset-hint').style.display = 'block';
    GetScriptOffset().then(v => { el('#pb-offset').value = v || 0; }).catch(() => {});
    if (opts.project) {
      await applyPlaybackProject(opts.project);
    }
    if (opts.review) {
      log('Freshly generated — soft curve + dots visible. Enable “Edit curve (dots)” to adjust points.');
      if (showContact) {
        el('#pb-contact-block').scrollIntoView({ block: 'nearest', behavior: 'smooth' });
      }
    } else if (!info.hasVideo && !(opts.project && opts.project.videoPath)) {
      log('Script loaded without video — Play drives device + curve only.');
    }
  }

  async function applyPlaybackProject(p) {
    if (!p || typeof p !== 'object') return;
    if (typeof p.offsetMs === 'number') {
      await applyOffset(p.offsetMs);
    }
    if (p.loopMarker && typeof p.loopMarker.startMs === 'number' && typeof p.loopMarker.endMs === 'number') {
      const m = { startMs: p.loopMarker.startMs, endMs: p.loopMarker.endMs };
      marker = markerIsValid(m) ? m : null;
      updateMarkerHint();
      redrawHeatmap();
    }
    const wantVideo = (p.videoPath || '').trim();
    if (wantVideo && wantVideo !== videoPath) {
      try {
        const url = await SetPlaybackVideo(wantVideo);
        videoPath = wantVideo;
        setVideoSrc(url);
        const stage = el('#pb-video-stage');
        stage.classList.add('has-video');
        stage.classList.remove('no-video');
        el('#pb-video-sync-row').style.display = 'flex';
        el('#pb-video-autostart-row').style.display = 'flex';
        el('#pb-trajectory-row').style.display = 'flex';
        await refreshVideoPlayability(wantVideo);
      } catch (err) {
        uiWarn('Project video: ' + err);
      }
    }
    if (typeof p.seekMs === 'number' && p.seekMs > 0) {
      await seekTo(p.seekMs);
    }
    ofsStatus('Project loaded');
  }

  function nextPlaylistIndexAfterAdvance() {
    const repeat = el('#pb-playlist-repeat') && el('#pb-playlist-repeat').checked;
    if (playlist.length < 2) return playlistIndex;
    if (playlistIndex < playlist.length - 1) return playlistIndex + 1;
    if (repeat) return 0;
    return playlistIndex;
  }

  async function playNextInPlaylist() {
    if (advancingPlaylist) return;
    const repeat = el('#pb-playlist-repeat') && el('#pb-playlist-repeat').checked;
    const atEnd = playlistIndex >= playlist.length - 1;
    if (atEnd && !repeat) {
      setPlayingState(false);
      return;
    }
    advancingPlaylist = true;
    try {
      if (playing) await stop({ user: false });
      if (atEnd && repeat) {
        if (el('#pb-playlist-shuffle') && el('#pb-playlist-shuffle').checked) {
          reshufflePlaylistForRepeatCycle();
        } else {
          playlistIndex = 0;
          renderPlaylist();
        }
      } else {
        playlistIndex += 1;
        renderPlaylist();
      }
      await loadScript(playlist[playlistIndex].path, { keepPlaylist: true });
      await play();
    } catch (err) {
      logError('Next item: ' + err);
      setPlayingState(false);
    } finally {
      advancingPlaylist = false;
    }
  }

  function onPlaybackFinished(payload) {
    if (advancingPlaylist) return;
    // Stop-Knopf / manuelles Beenden: nicht automatisch weiter in der Liste.
    if (userStopRequested) {
      userStopRequested = false;
      setPlayingState(false);
      return;
    }
    // Connect-/Play-Errors: Meter Reset, aber Playlist nicht weiter.
    if (payload && payload.failed) {
      setPlayingState(false);
      return;
    }
    const auto = el('#pb-playlist-auto') && el('#pb-playlist-auto').checked;
    const repeat = el('#pb-playlist-repeat') && el('#pb-playlist-repeat').checked;
    if (auto && playlist.length > 1 && (playlistIndex < playlist.length - 1 || repeat)) {
      playNextInPlaylist();
      return;
    }
    setPlayingState(false);
  }

  // startScriptPlayback startet nur das Funscript/Gerät (StartPlayback +
  // Statuswechsel), ohne das <video>-Element anzufassen - gemeinsamer Kern
  // für den App-eigenen "Play"-Knopf (play(), der zusätzlich das
  // Video von vorn startet) und den nativen Video-Play-Listener weiter
  // unten (der das Video NICHT zurückspulen darf, weil es dort schon an
  // seiner Position läuft). Gibt zurück, ob der Start geklappt hat.
  async function startScriptPlayback() {
    if (!scriptPath) { log('Choose an Emotion Script first.'); return false; }
    el('#pb-log').textContent = '';
    userStopRequested = false;

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
      // finiteOr (not ||): Amplitude / Restore allow 0 (#418 slider min).
      extendedOMin: finiteOr(el('#pb-eo-min').value, 0.1),
      extendedOHoldS: finiteOr(el('#pb-eo-hold').value, 10),
      extendedORestoreMs: finiteOr(el('#pb-eo-restore').value, 500),
      disableContactVibration: scriptHasContactVibration && el('#pb-contact-off').checked,
      contactIntensityScale: scriptHasContactVibration
        ? finiteOr(el('#pb-contact-intensity').value, 1)
        : 1,
      contactExtraSmooth: 0,
      contactVibrationSpan: scriptHasContactVibration
        ? finiteOr(el('#pb-contact-span').value, 0)
        : 0,
      contactVibrationCurve: scriptHasContactVibration
        ? (el('#pb-contact-curve').value || '')
        : '',
    };

    try {
      await StartPlayback(opts);
    } catch (err) {
      logError('Playback start: ' + err);
      return false;
    }
    setPlayingState(true);
    if (!useVideoSync) {
      log(videoPath
        ? 'Script only (video sync off) — internal clock.'
        : 'Script only without video — device + curve.');
    }
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

  // user=true: Stop-Knopf / manuelles Beenden — Playlist nicht weiter.
  // user=false: natürliches Video-/Skript-Ende — Auto-Next darf greifen.
  async function stop({ user = true } = {}) {
    if (user) userStopRequested = true;
    await StopPlayback();
    // Clear playing before pause so the video 'pause' handler does not
    // report sync-idle after the position channel is already closed.
    setPlayingState(false);
    if (videoPath) videoEl.pause();
  }

  async function triggerEO() {
    await TriggerExtendedO(
      finiteOr(el('#pb-eo-min').value, 0.1),
      finiteOr(el('#pb-eo-hold').value, 10),
      finiteOr(el('#pb-eo-restore').value, 500),
    );
  }

  // Video-Position ans Backend melden, solange Sync aktiv ist - das treibt
  // player.Sync() to (siehe player/sync.go). timeupdate feuert im Browser
  // ca. alle 250ms, das reicht für flüssiges Gerätefeedback.
  function pushVideoSyncPosition() {
    currentPosMs = Math.round(videoEl.currentTime * 1000);
    if (playing && el('#pb-use-video-sync').checked) {
      ReportVideoPosition(currentPosMs);
    }
  }

  videoEl.addEventListener('timeupdate', () => {
    currentPosMs = Math.round(videoEl.currentTime * 1000);

    // Markierten Abschnitt wiederholen. Das ist das Werkzeug, mit dem sich
    // der Offset überhaupt einstellen lässt: ohne Wiederholung müsste man
    // nach jeder Korrektur von Hand zurückspulen, und bis man wieder to der
    // fraglichen Stelle ist, hat man den Vergleich verloren.
    if (el('#pb-loop').checked && markerIsValid(marker) && currentPosMs >= marker.endMs) {
      videoEl.currentTime = marker.startMs / 1000;
      currentPosMs = marker.startMs;
    }
    if (!playing) { redrawHeatmap(); redrawCurve(); }
    if (playing && el('#pb-use-video-sync').checked) {
      ReportVideoPosition(Math.round(videoEl.currentTime * 1000));
    }
    redrawTrajectory();
  });
  // After seekTo(), 'seeked' fires once the decoder lands — push device sync
  // immediately so script/device do not lag ~250ms on the next timeupdate.
  videoEl.addEventListener('seeked', () => {
    currentPosMs = Math.round(videoEl.currentTime * 1000);
    redrawHeatmap();
    redrawCurve();
    redrawTrajectory();
    pushVideoSyncPosition();
  });
  // Pause / buffering: zero the device immediately (Sync idle sentinel).
  // Watchdog in player.Sync still covers lost timeupdate without these events.
  function pushVideoSyncIdle() {
    if (playing && el('#pb-use-video-sync')?.checked) {
      ReportVideoSyncIdle().catch(() => {});
    }
  }
  videoEl.addEventListener('pause', () => {
    if (videoEl.ended) return; // ended handler stops playback separately
    pushVideoSyncIdle();
  });
  videoEl.addEventListener('waiting', pushVideoSyncIdle);
  videoEl.addEventListener('playing', () => {
    // Resume after pause/buffer — push current position so Sync soft-starts.
    pushVideoSyncPosition();
  });
  // Video-Maße (videoWidth/Height) stehen erst nach 'loadedmetadata' fest -
  // vorher liefert videoContentRect() null und das Overlay bleibt versteckt.
  videoEl.addEventListener('loadedmetadata', () => redrawTrajectory());
  // Clear a prior decode warning once the element can actually play.
  videoEl.addEventListener('canplay', () => {
    const warn = el('#pb-video-warn');
    const conv = el('#pb-video-convert');
    if (!warn || warn.hidden) return;
    if (warn.textContent && /decode failed|not playable|MEDIA_ERR/i.test(warn.textContent)) {
      warn.hidden = true;
      if (conv) conv.hidden = true;
    }
  });

  // Historische video:pause/resume-Events. Extended-O skaliert nur noch die
  // Amplitude und pausiert das Video nicht mehr - Listener bleiben harmlos.
  EventsOn('video:pause', () => videoEl.pause());
  EventsOn('video:resume', () => videoEl.play().catch(() => {}));

  // Video-Ende soll auch unsere Playback sauber beenden - sonst bleibt
  // player.Sync() im Leerlauf hängen (wartet ewig auf weitere Positionen,
  // die nach Videoende nicht mehr kommen). user:false, damit die Playlist
  // bei „automatisch weiter“ noch weiterschalten darf.
  videoEl.addEventListener('ended', () => {
    if (playing && el('#pb-use-video-sync').checked) stop({ user: false });
  });
  videoEl.addEventListener('error', async () => {
    const warn = el('#pb-video-warn');
    const conv = el('#pb-video-convert');
    const code = videoEl.error && videoEl.error.code;
    const detail = code === 1 ? 'aborted'
      : code === 2 ? 'network error'
      : code === 3 ? 'decode failed (codec)'
      : code === 4 ? 'format/source not supported'
      : 'unknown';
    if (warn) {
      warn.hidden = false;
      warn.textContent = `Video not playable (${detail}). “Make playable” creates an H.264 copy.`;
    }
    if (conv) conv.hidden = false;
    if (playing) {
      try { await stop({ user: true }); } catch (_) { /* ignore */ }
    }
    uiWarn('Video load error — ' + detail + '. Conversion offered.');
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
  //
  // pb-video-play-autostart ist ein eigener Schalter (getrennt von "Device
  // follows video position", das nur die Positions-Verfolgung WÄHREND der
  // Wiedergabe betrifft): vorher startete jedes native Video-Play immer
  // auch das Gerät, selbst bei ausgeschaltetem Sync - wer nur das Video
  // ansehen wollte, ohne das Gerät zu starten, hatte keine Möglichkeit,
  // das zu verhindern.
  videoEl.addEventListener('play', () => {
    if (playing || !scriptPath) return;
    if (!el('#pb-video-play-autostart').checked) return;
    startScriptPlayback();
  });

  EventsOn('playback:log', log);
  EventsOn('playback:error', msg => log('ERROR: ' + msg));
  EventsOn('playback:done', (payload) => onPlaybackFinished(payload || {}));
  EventsOn('playback:frame', f => {
    totalMs = Math.max(f.totalMs, 1);
    currentPosMs = f.atMs;
    el('#pb-progress').style.width = Math.min(100, (f.atMs / totalMs) * 100) + '%';
    const vibRaw = Number(f && f.vibration);
    const sucRaw = Number(f && f.suction);
    const vib = Math.round((Number.isFinite(vibRaw) ? vibRaw : 0) * 100);
    const suc = Math.round((Number.isFinite(sucRaw) ? sucRaw : 0) * 100);
    el('#pb-vib').textContent = vib + '%';
    el('#pb-suc').textContent = suc + '%';
    checkAutoExtendedO(f.atMs);
    redrawHeatmap();
    redrawCurve();
  });

  // Canvas-Breite ist eine feste Pixelangabe, keine CSS-Größe - bei einer
  // Fenstergrößenänderung muss sie neu gesetzt werden, sonst wird die Kurve
  // verzerrt skaliert dargestellt.
  window.addEventListener('resize', () => {
    if (curvePoints) {
      sizeCanvasForDPR(curveCanvas, 800, 176);
      redrawCurve();
    }
    if (heatmapPoints) {
      sizeCanvasForDPR(heatmapCanvas, 800, 56);
      redrawHeatmap();
    }
    redrawTrajectory();
  });

  el('#pb-script-doctor').addEventListener('click', async () => {
    const status = el('#pb-script-doctor-status');
    const box = el('#pb-script-doctor-result');
    const btn = el('#pb-script-doctor');
    btn.disabled = true;
    status.textContent = 'Checking…';
    try {
      const result = await ScriptQuality();
      status.textContent = '';
      const pct = Math.round((result.score || 0) * 100);
      const qd = el('#pb-quality-details');
      if (qd) qd.open = true;
      box.style.display = 'block';
      box.style.background = result.passed ? 'rgba(61,216,117,0.12)' : 'rgba(216,77,77,0.12)';
      box.style.border = `1px solid ${result.passed ? 'var(--ok)' : 'var(--danger)'}`;
      let html = `<b>Signal Quality (Script Doctor): ${pct}% ${result.passed ? '(within normal)' : '(review recommended)'}</b>`;
      html += '<br><span style="opacity:0.7;">Signal quality only — jitter, jumps, gaps, device density. '
        + 'Not proof the curve matches the video (that would be motion fidelity / phase comparison). '
        + 'Estimated from the script only, without video; tracking-based checks are missing here, '
        + 'unlike right after generation.</span>';
      if (result.warnings && result.warnings.length > 0) {
        html += '<ul style="margin:6px 0 0 18px; padding:0;">'
          + result.warnings.map(w => `<li>${w}</li>`).join('') + '</ul>';
      }
      box.innerHTML = html;
    } catch (err) {
      const qd = el('#pb-quality-details');
      if (qd) qd.open = true;
      uiError('Quality check: ' + err, status);
    } finally {
      btn.disabled = false;
    }
  });

  function ofsStatus(msg) {
    const s = el('#pb-ofs-status');
    if (s) s.textContent = msg || '';
  }
  el('#pb-heatmap-export')?.addEventListener('click', async () => {
    try {
      const path = await ExportScriptHeatmapPNG();
      ofsStatus('Heatmap: ' + path);
      uiInfo('Heatmap saved: ' + path);
    } catch (err) {
      uiError('Heatmap: ' + err, el('#pb-ofs-status'));
    }
  });
  el('#pb-project-save')?.addEventListener('click', async () => {
    try {
      const path = await SavePlaybackProject({
        videoPath: videoPath || '',
        scriptPath: scriptPath || '',
        offsetMs: parseInt(el('#pb-offset')?.value, 10) || 0,
        seekMs: currentPosMs || 0,
        loopMarker: markerIsValid(marker) ? { startMs: marker.startMs, endMs: marker.endMs } : null,
      });
      ofsStatus('Project: ' + path);
      uiInfo('Project saved: ' + path);
    } catch (err) {
      uiError('Project: ' + err, el('#pb-ofs-status'));
    }
  });
  el('#pb-project-load')?.addEventListener('click', async () => {
    try {
      const path = await PickPlaybackProject();
      restoreKeyboardFocus();
      if (!path) return;
      const project = await LoadPlaybackProject(path);
      const script = (project && project.scriptPath) || '';
      if (!script) {
        uiError('Project has no script path', el('#pb-ofs-status'));
        return;
      }
      replacePlaylist([script], 0);
      await loadScript(script, { keepPlaylist: true, project });
      ofsStatus('Project: ' + path);
      uiInfo('Project loaded: ' + path);
    } catch (err) {
      uiError('Load project: ' + err, el('#pb-ofs-status'));
    }
  });
  function activeEditAxis() {
    return (editAxis && String(editAxis).trim()) || el('#pb-axis')?.value || 'general';
  }
  function scaleFactorValue() {
    const v = parseFloat(el('#pb-scale-factor')?.value);
    return Number.isFinite(v) ? v : 0.8;
  }
  function updateScaleFactorLabel() {
    const lab = el('#pb-scale-factor-val');
    if (lab) lab.textContent = `×${scaleFactorValue().toFixed(2)}`;
  }
  el('#pb-scale-factor')?.addEventListener('input', updateScaleFactorLabel);
  updateScaleFactorLabel();

  function capIntensityValue() {
    const v = parseFloat(el('#pb-cap-intensity')?.value);
    if (!Number.isFinite(v) || v <= 0) return 400;
    return Math.max(50, Math.min(2000, v));
  }
  function updateCapIntensityLabel() {
    const lab = el('#pb-cap-intensity-val');
    if (lab) lab.textContent = `${Math.round(capIntensityValue())}`;
  }
  el('#pb-cap-intensity')?.addEventListener('input', updateCapIntensityLabel);
  updateCapIntensityLabel();

  function syncPlayFpsSnapLabel() {
    const inp = el('#pb-fps-snap');
    const lab = el('#pb-fps-snap-val');
    if (!inp || !lab) return;
    const n = Math.round(Number(inp.value) || 0);
    lab.textContent = n > 0 ? String(n) : 'off';
  }
  syncPlayFpsSnapLabel();
  el('#pb-fps-snap')?.addEventListener('input', syncPlayFpsSnapLabel);
  el('#pb-fps-snap')?.addEventListener('change', syncPlayFpsSnapLabel);

  updateSpeedHighlightLabel();
  el('#pb-speed-hl-thresh')?.addEventListener('input', () => {
    updateSpeedHighlightLabel();
    if (speedHighlightsEnabled()) refreshSpeedHighlights();
  });
  el('#pb-speed-hl')?.addEventListener('change', () => refreshSpeedHighlights());

  el('#pb-bpm-grid')?.addEventListener('change', () => redrawCurve());
  el('#pb-bpm')?.addEventListener('input', () => {
    if (el('#pb-bpm-grid')?.checked) redrawCurve();
  });

  el('#pb-zoom-in')?.addEventListener('click', () => zoomByFactor(1.6, currentPosMs));
  el('#pb-zoom-out')?.addEventListener('click', () => zoomByFactor(1 / 1.6, currentPosMs));
  el('#pb-zoom-sel')?.addEventListener('click', () => zoomToSelection());
  el('#pb-zoom-reset')?.addEventListener('click', () => {
    resetViewWindow();
    redrawHeatmap();
    redrawCurve();
  });
  curveCanvas.addEventListener('wheel', (e) => {
    if (!scriptPath || !totalMs) return;
    e.preventDefault();
    const factor = e.deltaY < 0 ? 1.25 : 1 / 1.25;
    zoomByFactor(factor, curveMsOfX(e.clientX));
  }, { passive: false });

  el('#pb-cap-speed')?.addEventListener('click', async () => {
    if (!markerIsValid(marker)) {
      uiWarn('Mark a range on the heatmap first.');
      return;
    }
    const axis = activeEditAxis();
    const maxI = capIntensityValue();
    try {
      await EditCapSpeedRange(axis, marker.startMs, marker.endMs, maxI);
      ofsStatus(`Speed cap @ ${Math.round(maxI)} (${axis})`);
      // Reload duration + curve/edit buffer — CapSpeedRange can stretch At.
      await reloadAfterRangeEdit();
    } catch (err) {
      uiError('Speed cap: ' + err, el('#pb-ofs-status'));
    }
  });
  el('#pb-scale-range')?.addEventListener('click', async () => {
    if (!markerIsValid(marker)) {
      uiWarn('Mark a range on the heatmap first.');
      return;
    }
    const axis = activeEditAxis();
    const factor = scaleFactorValue();
    const soft = !!el('#pb-scale-soft-edges')?.checked;
    try {
      await EditScaleRange(axis, marker.startMs, marker.endMs, factor, soft);
      ofsStatus(`Scale ×${factor.toFixed(2)}${soft ? ' soft' : ''} (${axis})`);
      await reloadAfterRangeEdit();
    } catch (err) {
      uiError('Scale: ' + err, el('#pb-ofs-status'));
    }
  });
  el('#pb-del-range')?.addEventListener('click', async () => {
    if (!markerIsValid(marker)) {
      uiWarn('Mark a range on the heatmap first.');
      return;
    }
    const axis = activeEditAxis();
    try {
      await EditDeleteRange(axis, marker.startMs, marker.endMs);
      ofsStatus(`Range deleted (${axis})`);
      marker = null;
      await reloadAfterRangeEdit();
    } catch (err) {
      uiError('Delete: ' + err, el('#pb-ofs-status'));
    }
  });
  el('#pb-choose').addEventListener('click', chooseScript);
  el('#pb-choose-empty').addEventListener('click', chooseScript);
  el('#pb-queue-add').addEventListener('click', queueAddScript);
  el('#pb-next').addEventListener('click', () => playNextInPlaylist());
  el('#pb-play-novideo').addEventListener('click', play);
  el('#pb-playlist-list').addEventListener('click', async e => {
    const remove = e.target.closest('.pb-playlist-remove');
    if (remove) {
      const idx = Number(remove.dataset.idx);
      if (Number.isNaN(idx)) return;
      const removingCurrent = idx === playlistIndex;
      playlist.splice(idx, 1);
      if (playlist.length === 0) {
        playlistIndex = 0;
        renderPlaylist();
        return;
      }
      if (playlistIndex >= playlist.length) playlistIndex = Math.max(0, playlist.length - 1);
      else if (idx < playlistIndex) playlistIndex -= 1;
      renderPlaylist();
      // Active row removed: load the script now highlighted, else UI lies.
      if (removingCurrent && playlist[playlistIndex]) {
        if (playing) await stop({ user: false });
        await loadScript(playlist[playlistIndex].path, { keepPlaylist: true });
      }
      return;
    }
    const pick = e.target.closest('.pb-playlist-pick');
    if (!pick) return;
    const idx = Number(pick.dataset.idx);
    if (Number.isNaN(idx) || idx === playlistIndex) return;
    if (playing) await stop({ user: false });
    playlistIndex = idx;
    renderPlaylist();
    await loadScript(playlist[idx].path, { keepPlaylist: true });
  });
  el('#pb-pick-video').addEventListener('click', attachVideoFromPicker);
  el('#pb-video-change').addEventListener('click', attachVideoFromPicker);
  el('#pb-video-fs').addEventListener('click', toggleVideoFullscreen);
  el('#pb-video-convert')?.addEventListener('click', convertPlaybackVideo);
  videoEl.addEventListener('dblclick', e => {
    e.preventDefault();
    toggleVideoFullscreen();
  });
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

    if (e.key === 'Escape' && el('#pb-video-stage').classList.contains('is-fs')) {
      e.preventDefault();
      toggleVideoFullscreen();
      return;
    }

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
      log(el('#pb-loop').checked ? 'Loop on.' : 'Loop off.');
    } else if (e.key >= '1' && e.key <= '9') {
      // Taste n -> n*10% der Laufzeit, wie in VLC/YouTube üblich (statt
      // (n-1)/9, was Taste 1 auf 0% und Taste 9 nur auf ~88.9% legte).
      if (totalMs > 0) seekTo(Math.round(totalMs * Number(e.key) / 10));
    } else if (e.key.toLowerCase() === 'e') {
      if (!el('#pb-eo-trigger').disabled) triggerEO();
    }
  });

  // Gespeicherte Standardwerte übernehmen, sobald settings.js sie geladen hat.
  try {
    el('#pb-playlist-shuffle').checked = sessionStorage.getItem(PLAYLIST_SHUFFLE_KEY) === '1';
    el('#pb-playlist-repeat').checked = sessionStorage.getItem(PLAYLIST_REPEAT_KEY) === '1';
  } catch (_) { /* private mode */ }
  el('#pb-playlist-shuffle').addEventListener('change', e => {
    try { sessionStorage.setItem(PLAYLIST_SHUFFLE_KEY, e.target.checked ? '1' : '0'); } catch (_) {}
    if (e.target.checked) reshufflePlaylistKeepingCurrent();
  });
  el('#pb-playlist-repeat').addEventListener('change', e => {
    try { sessionStorage.setItem(PLAYLIST_REPEAT_KEY, e.target.checked ? '1' : '0'); } catch (_) {}
    renderPlaylist();
  });

  function syncPlayAdvKnobLabels() {
    const tick = el('#pb-tick');
    const tickLab = el('#pb-tick-val');
    if (tick && tickLab) tickLab.textContent = String(Math.round(Number(tick.value) || 0));
    const max = el('#pb-maxspeed');
    const maxLab = el('#pb-maxspeed-val');
    if (max && maxLab) {
      const n = Number(max.value);
      maxLab.textContent = Number.isFinite(n) ? n.toFixed(1) : '0.6';
    }
    const sm = el('#pb-smoothing');
    const smLab = el('#pb-smoothing-val');
    if (sm && smLab) {
      const n = Number(sm.value);
      smLab.textContent = Number.isFinite(n) ? (Math.round(n * 100) / 100).toString() : '0.3';
    }
    const soft = el('#pb-softstart');
    const softLab = el('#pb-softstart-val');
    if (soft && softLab) softLab.textContent = String(Math.round(Number(soft.value) || 0));
  }
  syncPlayAdvKnobLabels();

  function syncPlayEOKnobLabels() {
    const amp = el('#pb-eo-min');
    const ampLab = el('#pb-eo-min-val');
    if (amp && ampLab) {
      const n = Number(amp.value);
      ampLab.textContent = Number.isFinite(n) ? (Math.round(n * 100) / 100).toString() : '0.1';
    }
    const hold = el('#pb-eo-hold');
    const holdLab = el('#pb-eo-hold-val');
    if (hold && holdLab) holdLab.textContent = String(Math.round(Number(hold.value) || 0));
    const restore = el('#pb-eo-restore');
    const restoreLab = el('#pb-eo-restore-val');
    if (restore && restoreLab) restoreLab.textContent = String(Math.round(Number(restore.value) || 0));
  }
  syncPlayEOKnobLabels();

  getSettingsCache().then(s => {
    el('#pb-mock').checked = s.playbackMock;
    el('#pb-sync').value = s.playbackSync;
    el('#pb-tick').value = s.playbackTickMs;
    el('#pb-maxspeed').value = s.playbackMaxSpeed;
    el('#pb-smoothing').value = s.playbackSmoothing;
    el('#pb-softstart').value = s.playbackSoftStartMs;
    syncPlayAdvKnobLabels();
    el('#pb-eo-enabled').checked = s.playbackEOEnabled;
    el('#pb-eo-min').value = s.playbackEOMin;
    el('#pb-eo-hold').value = s.playbackEOHoldS;
    el('#pb-eo-restore').value = s.playbackEORestoreMs;
    syncPlayEOKnobLabels();
    // !== false statt einer direkten Zuweisung: das Backend-Default ist
    // true (siehe prefPlaybackVideoPlayAutostart), und ein fehlendes Feld
    // (älterer Settings-Stand, Test-Stub ohne GetSettings-Override) soll
    // nicht durch undefined -> false versehentlich das native Video-Play
    // stummschalten.
    el('#pb-video-play-autostart').checked = s.playbackVideoPlayAutostart !== false;
    el('#pb-trajectory-toggle').checked = !!s.playbackTrajectoryOverlay;
    redrawTrajectory();

    // OFS / Feel knobs — restore when field present (older stubs omit them).
    if (s.playbackCapIntensity != null && el('#pb-cap-intensity')) {
      el('#pb-cap-intensity').value = s.playbackCapIntensity;
      updateCapIntensityLabel();
    }
    if (typeof s.playbackSpeedHL === 'boolean' && el('#pb-speed-hl')) {
      el('#pb-speed-hl').checked = s.playbackSpeedHL;
    }
    if (s.playbackSpeedHLThresh != null && el('#pb-speed-hl-thresh')) {
      el('#pb-speed-hl-thresh').value = s.playbackSpeedHLThresh;
      updateSpeedHighlightLabel();
    }
    if (s.playbackScaleFactor != null && el('#pb-scale-factor')) {
      el('#pb-scale-factor').value = s.playbackScaleFactor;
      updateScaleFactorLabel();
    }
    if (typeof s.playbackScaleSoftEdges === 'boolean' && el('#pb-scale-soft-edges')) {
      el('#pb-scale-soft-edges').checked = s.playbackScaleSoftEdges;
    }
    if (s.playbackFpsSnap != null && el('#pb-fps-snap')) {
      el('#pb-fps-snap').value = s.playbackFpsSnap;
      syncPlayFpsSnapLabel();
    }
    if (typeof s.playbackBpmGrid === 'boolean' && el('#pb-bpm-grid')) {
      el('#pb-bpm-grid').checked = s.playbackBpmGrid;
    }
    if (s.playbackOMarkerIntensity != null && el('#pb-omarker-intensity')) {
      el('#pb-omarker-intensity').value = s.playbackOMarkerIntensity;
      syncPlayOMarkerIntensityLabel();
    }
    if (typeof s.playbackFeelHeatbands === 'boolean' && el('#pb-seg-heatbands')) {
      el('#pb-seg-heatbands').checked = s.playbackFeelHeatbands;
    }
    if (s.playbackFeelHeatbandsOpacity != null && el('#pb-seg-heatbands-opacity')) {
      el('#pb-seg-heatbands-opacity').value = s.playbackFeelHeatbandsOpacity;
    }
    syncFeelHeatbandsOpacityUi();
    if (s.playbackContactIntensity != null && el('#pb-contact-intensity')) {
      const ci = finiteOr(s.playbackContactIntensity, 1);
      el('#pb-contact-intensity').value = String(ci);
      el('#pb-contact-intensity-val').textContent = ci.toFixed(2);
    }
  });
  el('#pb-mock').addEventListener('change', e => saveSetting('playback.mock', e.target.checked));
  el('#pb-sync').addEventListener('change', e => saveSetting('playback.sync_mode', e.target.value));
  el('#pb-tick').addEventListener('input', syncPlayAdvKnobLabels);
  el('#pb-maxspeed').addEventListener('input', syncPlayAdvKnobLabels);
  el('#pb-smoothing').addEventListener('input', syncPlayAdvKnobLabels);
  el('#pb-softstart').addEventListener('input', syncPlayAdvKnobLabels);
  el('#pb-tick').addEventListener('change', e => {
    syncPlayAdvKnobLabels();
    saveSetting('playback.tick_ms', parseFloat(e.target.value));
  });
  el('#pb-maxspeed').addEventListener('change', e => {
    syncPlayAdvKnobLabels();
    saveSetting('playback.max_speed', parseFloat(e.target.value));
  });
  el('#pb-smoothing').addEventListener('change', e => {
    syncPlayAdvKnobLabels();
    saveSetting('playback.smoothing', parseFloat(e.target.value));
  });
  el('#pb-softstart').addEventListener('change', e => {
    syncPlayAdvKnobLabels();
    saveSetting('playback.soft_start_ms', parseFloat(e.target.value));
  });
  el('#pb-eo-enabled').addEventListener('change', e => saveSetting('playback.extended_o_enabled', e.target.checked));
  el('#pb-eo-min').addEventListener('input', syncPlayEOKnobLabels);
  el('#pb-eo-hold').addEventListener('input', syncPlayEOKnobLabels);
  el('#pb-eo-restore').addEventListener('input', syncPlayEOKnobLabels);
  el('#pb-eo-min').addEventListener('change', e => {
    syncPlayEOKnobLabels();
    saveSetting('playback.extended_o_min', parseFloat(e.target.value));
  });
  el('#pb-eo-hold').addEventListener('change', e => {
    syncPlayEOKnobLabels();
    saveSetting('playback.extended_o_hold_seconds', parseFloat(e.target.value));
  });
  el('#pb-eo-restore').addEventListener('change', e => {
    syncPlayEOKnobLabels();
    saveSetting('playback.extended_o_restore_ms', parseFloat(e.target.value));
  });
  el('#pb-video-play-autostart').addEventListener('change', e => saveSetting('playback.video_play_autostart', e.target.checked));
  el('#pb-trajectory-toggle').addEventListener('change', e => {
    saveSetting('playback.trajectory_overlay', e.target.checked);
    const hint = el('#pb-trajectory-hint');
    if (hint) hint.style.display = (e.target.checked && !trajectoryData) ? 'block' : 'none';
    redrawTrajectory();
  });
  el('#pb-cap-intensity')?.addEventListener('change', () => {
    saveSetting('playback.cap_intensity', capIntensityValue());
  });
  el('#pb-speed-hl')?.addEventListener('change', e => {
    saveSetting('playback.speed_highlights', e.target.checked);
  });
  el('#pb-speed-hl-thresh')?.addEventListener('change', () => {
    const v = parseFloat(el('#pb-speed-hl-thresh')?.value);
    saveSetting('playback.speed_highlight_thresh', Number.isFinite(v) ? v : 400);
  });
  el('#pb-scale-factor')?.addEventListener('change', () => {
    saveSetting('playback.scale_factor', scaleFactorValue());
  });
  el('#pb-scale-soft-edges')?.addEventListener('change', e => {
    saveSetting('playback.scale_soft_edges', e.target.checked);
  });
  el('#pb-fps-snap')?.addEventListener('change', e => {
    saveSetting('playback.fps_snap', finiteOr(e.target.value, 0));
  });
  el('#pb-bpm-grid')?.addEventListener('change', e => {
    saveSetting('playback.bpm_grid', e.target.checked);
  });
  el('#pb-omarker-intensity')?.addEventListener('change', e => {
    saveSetting('playback.omarker_intensity', finiteOr(e.target.value, 0.5));
  });
  el('#pb-seg-heatbands')?.addEventListener('change', e => {
    saveSetting('playback.feel_heatbands', e.target.checked);
  });
  el('#pb-seg-heatbands-opacity')?.addEventListener('change', e => {
    saveSetting('playback.feel_heatbands_opacity', finiteOr(e.target.value, 1));
  });
  el('#pb-contact-marks-toggle')?.addEventListener('change', () => {
    redrawTrajectory();
  });

  async function refreshScriptVisuals() {
    if (!scriptPath) return;
    try {
      const [heat, markers] = await Promise.all([
        GetHeatmap(HEATMAP_BUCKETS),
        GetOMarkers(scriptPath),
      ]);
      heatmapPoints = heat;
      oMarkers = Array.isArray(markers) ? markers : [];
      if (el('#pb-curve-edit').checked) {
        const acts = await GetScriptAxisActions(editAxis || 'general');
        rawActions = Array.isArray(acts) ? acts.map(a => ({ atMs: a.at, pos: a.pos })) : rawActions;
      }
      await drawCurve();
      redrawHeatmap();
      renderOMarkerList();
      describeScript();
    } catch (err) {
      logError('Refresh: ' + err);
    }
  }

  // Nach Speed-Cap / Bereich-Delete: Dauer neu lesen und Editor-Puffer
  // neu laden — sonst schreibt Editieren die alten Punkte wieder zurück.
  async function reloadAfterRangeEdit() {
    if (!scriptPath) return;
    try {
      const info = await LoadFunscript(scriptPath);
      totalMs = Math.max(info.durationMs, 1);
      clampViewWindow();
      updateZoomChrome();
      scriptHasContactVibration = !!info.contactVibration;
    } catch (err) {
      logError('Reload script after edit: ' + err);
    }
    await refreshScriptVisuals();
  }

  window.addEventListener('ozone:hotkey', async (ev) => {
    if (!scriptPath) return;
    const nowMs = (ev.detail && ev.detail.nowMs) || 0;
    try {
      oMarkers = await applyHotkeyOMarker(scriptPath, nowMs, oMarkers);
      renderOMarkerList();
      redrawHeatmap();
      redrawCurve();
      log('O-marker set: ' + (nowMs / 1000).toFixed(1) + 's–' + ((nowMs + 4000) / 1000).toFixed(1) + 's');
    } catch (err) {
      logError('O-marker hotkey: ' + err);
    }
  });
  window.addEventListener('ozone:suggested', () => { refreshScriptVisuals(); });
  window.addEventListener('polarity:inverted', () => { refreshScriptVisuals(); });
  window.addEventListener('ringdown:applied', () => { refreshScriptVisuals(); });
  // Create Review Speech-Hold strip → seek Play without switching APIs.
  window.addEventListener('playback:seek-ms', (ev) => {
    const ms = Number(ev.detail && ev.detail.ms);
    if (!Number.isFinite(ms) || ms < 0) return;
    seekTo(Math.round(ms));
  });
  window.addEventListener('playback:chapters-changed', async () => {
    if (!scriptPath) return;
    try {
      const ch = await GetScriptChapterMarks();
      chapterMarks = Array.isArray(ch)
        ? ch.map(normalizeChapterMark).filter(Boolean)
        : [];
      renderChapterList();
      describeScript();
    } catch (err) {
      logError('Chapters refresh: ' + err);
    }
  });
  ['holding', 'gentle', 'intense', 'climax'].forEach((lab) => {
    el(`#pb-seg-f-${lab}`)?.addEventListener('change', () => renderAudioSegmentsPanel());
  });
  el('#pb-seg-f-speech-only')?.addEventListener('change', () => renderAudioSegmentsPanel());
  el('#pb-seg-heatbands')?.addEventListener('change', () => {
    syncFeelHeatbandsOpacityUi();
    redrawHeatmap();
    redrawCurve();
  });
  el('#pb-seg-heatbands-opacity')?.addEventListener('input', () => {
    syncFeelHeatbandsOpacityUi();
    redrawHeatmap();
    redrawCurve();
  });
  syncFeelHeatbandsOpacityUi();
  el('#pb-audio-seg-chapters')?.addEventListener('click', () => {
    addVisibleAudioSegmentsAsChapters();
  });

  if (el('#pb-axis')) {
    el('#pb-axis').addEventListener('change', async () => {
      editAxis = el('#pb-axis').value || 'general';
      if (editMode) {
        el('#pb-curve-edit').checked = false;
        await setEditMode(false);
        el('#pb-curve-edit').checked = true;
        await setEditMode(true);
      } else {
        await drawCurve();
      }
    });
  }
  if (el('#pb-playback-source')) {
    el('#pb-playback-source').addEventListener('change', async () => {
      try {
        await SetPlaybackSource(el('#pb-playback-source').value);
        log('Playback drive: ' + el('#pb-playback-source').value);
        const info = await LoadFunscript(scriptPath);
        await refreshSamnControls(info);
        scriptHasContactVibration = !!info.contactVibration;
        el('#pb-contact-block').hidden = !(scriptHasContactVibration && info.playbackSource !== 'axes');
        await refreshScriptVisuals();
      } catch (err) {
        logError('Playback source: ' + err);
      }
    });
  }
  if (el('#pb-strength')) {
    el('#pb-strength').addEventListener('change', async () => {
      const name = el('#pb-strength').value;
      if (!name) return;
      try {
        await SetActiveStrength(name);
        log('Strength preset: ' + name);
      } catch (err) {
        logError('Strength: ' + err);
      }
    });
  }
  if (el('#pb-bake-axes')) {
    el('#pb-bake-axes').addEventListener('click', async () => {
      try {
        const out = await BakeNeoAxesOnLoaded();
        log('Baked Neo axes → ' + out);
        await loadScript(out, 0, { keepPlaylist: true });
      } catch (err) {
        logError('Bake axes: ' + err);
      }
    });
  }
  if (el('#pb-optimize-neo2')) {
    el('#pb-optimize-neo2').addEventListener('click', async () => {
      const status = el('#pb-optimize-neo2-status');
      const btn = el('#pb-optimize-neo2');
      btn.disabled = true;
      if (status) {
        status.style.display = 'block';
        status.textContent = 'Optimizing for Neo 2 (fill gaps → contact → bake)…';
      }
      try {
        const res = await OptimizeLoadedForNeo2(true);
        if (status) status.textContent = res.message || 'Neo 2 ready';
        log(res.message || 'Optimized for Neo 2');
        if (res.path) {
          await loadScript(res.path, 0, { keepPlaylist: true, review: true });
        }
      } catch (err) {
        if (status) status.textContent = 'Optimize failed: ' + err;
        logError('Optimize for Neo 2: ' + err);
      } finally {
        btn.disabled = false;
      }
    });
  }
  if (el('#pb-export-funscript')) {
    el('#pb-export-funscript').addEventListener('click', async () => {
      try {
        const out = await ExportLoadedFunscript('');
        log('Exported funscript → ' + out);
      } catch (err) {
        logError('Export: ' + err);
      }
    });
  }
  if (el('#pb-save-samn')) {
    el('#pb-save-samn').addEventListener('click', async () => {
      try {
        const out = await SaveLoadedAsSamn();
        log('Saved native script → ' + out);
        await loadScript(out, 0, { keepPlaylist: true });
      } catch (err) {
        logError('Save Emotion Script: ' + err);
      }
    });
  }

  return {
    // Von generator.js genutzt, um ein Ergebnis direkt zu übernehmen.
    // opts.review: open Play with dots visible; Edit curve stays off until checked.
    loadScriptPath: (path, opts = {}) => loadScript(path, 0, opts || {}).then(() => switchToPlaybackTab()),
    seekMs: (atMs) => seekTo(atMs),
    nextPlaylistIndexAfterAdvance,
    getPlaylistIndex: () => playlistIndex,
    setPlaylistForTest: (paths, startIndex = 0) => replacePlaylist(paths, startIndex),
  };
}

function switchToPlaybackTab() {
  document.querySelector('.tab-btn[data-tab="playback"]').click();
}
