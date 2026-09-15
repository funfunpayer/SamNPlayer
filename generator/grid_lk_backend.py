"""Grid-LK-Backend: ein Gitter unabhängiger Punkte statt einer einzelnen
Tracker-Bounding-Box.

Alternative zum CSRT-Weg in generate_funscript.py, für dieselbe markierte ROI
wie --backend csrt (im Gegensatz zu --backend flow, das keine Region
braucht). Liefert dieselbe Rückgabeform wie track_roi()/flow_backend.analyze,
damit die gesamte nachgelagerte Verarbeitung unverändert weiterläuft.

IDEE: statt einer Bounding Box (CSRT/KCF/MOSSE - "alle Eier in einem Korb",
siehe docs/NEXT.md Abschnitt 8 zur KCF/MOSSE-Messung) wird ein N x N-Gitter
aus Punkten in der ROI gesät und jeder Punkt einzeln per Sparse Optical Flow
(cv2.calcOpticalFlowPyrLK) verfolgt - derselbe Baustein, den
generate_funscript.py bereits für die Kamerabewegungs-Kompensation benutzt
(estimate_camera_motion). Pro Frame wird der MEDIAN der noch verfolgten
Punkte als Position genommen: ein einzelner verlorener Punkt kippt das
Ergebnis nicht, anders als eine einzelne verlorene Bounding Box.

GEMESSEN (September 15, 2026, echter 256x144-Clip, docs/NEXT.md Abschnitt 8
- volle Zahlen dort): die Hypothese trägt, aber nur ab einer Mindestdichte.

  * Leichte, gut texturierte ROI (154x72px "hub"): CSRT 92.3s/Score 0.90,
    Grid-LK (jede getestete Dichte) 5.9-6.2s (~15x schneller)/Score
    0.90-1.00 - deutlich schneller UND mindestens gleich gut.

  * Kleine, harte ROI (18x16px, Tip-Region): über den vollen Clip CSRT
    269.5s/Score 0.95 (auf einem 800-Frame-Ausschnitt dagegen nur Score
    0.45 "PRÜFEN" - dieselbe Instabilität, die schon bei der
    KCF/MOSSE-Messung auffiel). Ein 3x3-Gitter (9 Punkte) bricht auf
    diesem Clip WIRKLICH zusammen: 1498/2527 Frames (59.3%) ohne
    überlebenden Punkt, davon 1488 Frames am Stück ab Frame 1039 - kein
    kurzer Ausrutscher, sondern derselbe Fehlermodus wie bei KCF/MOSSE, nur
    nicht ganz so vollständig. Ein 4x4-Gitter (16 Punkte) und ein
    6x6-Gitter (36 Punkte) zeigen GENAU in diesem Abschnitt (Frame
    1039-2527) dagegen KEINEN einzigen Aussetzer - beide bleiben bei 0.4%
    verlorenen Frames (Score 1.00), bei praktisch identischer Laufzeit wie
    das 3x3-Gitter (6-7s). Der Mehrpreis für mehr Punkte ist hier
    vernachlässigbar (Frame-Dekodierung und Kamerakompensation dominieren
    die Kosten, nicht die Punktzahl) - es gibt also keinen Geschwindigkeits-
    grund, die knappere, nachweislich riskantere Dichte zu wählen.

  Deshalb: die Gitterdichte ist hier bewusst NICHT als CLI-Option
  freigegeben (anders als z.B. --smooth-window). Eine ungeeignete Dichte zu
  wählen ist genau die Art unsichtbarer Falle, die bei KCF/MOSSE zum
  Nicht-Ausliefern führte - dort gab es aber keinen Hebel, sie zu vermeiden.
  Hier gibt es einen (mehr Punkte), und er kostet praktisch nichts, also
  wird er fest auf einen gemessen sicheren Wert gesetzt statt dem Nutzer
  überlassen.

Wiederbesetzung verlorener Punkte: jeden Frame werden Punkte mit status==0
verworfen; fehlen dadurch Punkte zum Sollbestand, werden per
cv2.goodFeaturesToTrack frische Ecken NUR innerhalb einer Maske um die
aktuelle bbox-Schätzung nachgesetzt (nicht im ganzen Bild - das würde bei
einem Wiederfund irgendwo im Bild landen, nicht in der verfolgten Region).
Verlieren ALLE Punkte gleichzeitig (0 Überlebende) oder wird ein harter
Szenenschnitt erkannt (detect_scene_cut, wiederverwendet aus
generate_funscript.py), wird das gesamte Gitter an der zuletzt bekannten
Position neu gesät - dieselbe Idee wie CSRTs Neuverankerung am Schnitt in
track_roi(), nur ohne Erscheinungsgedächtnis (das gibt es hier nicht).
"""

import sys

import numpy as np
import cv2
from scipy.signal import savgol_filter

from generate_funscript import estimate_camera_motion, detect_scene_cut

# Gitterdichte. GEMESSEN sicher (siehe Modulkommentar) - 4x4 (16 Punkte) war
# bereits fehlerfrei auf der harten Testregion, 6x6 kostete auf demselben
# Clip praktisch nichts mehr (6.8s statt 6.2s) und ist der etwas größere
# Sicherheitsabstand zum gemessenen 3x3-Totalausfall. Nicht als CLI-Option
# freigegeben - siehe Begründung oben.
GRID_N = 6
GRID_M = 6

# Anteil des Sollbestands, unter dem ein Frame als "angeschlagen" statt nur
# "nicht perfekt" gilt - informativ (stats["grid_degraded_frames"]), fließt
# nicht in die Qualitätsbewertung ein. Ein Viertel des Gitters weg ist schon
# ein spürbar schwächerer Median, aber noch ein echter Median mehrerer
# Punkte, kein einzelner Wert.
DEGRADED_QUORUM_FRACTION = 0.25


def _seed_grid(roi, n, m):
    x, y, w, h = [float(v) for v in roi]
    pad_x = max(1.0, w * 0.12)
    pad_y = max(1.0, h * 0.12)
    xs = np.linspace(x + pad_x, x + w - pad_x, n)
    ys = np.linspace(y + pad_y, y + h - pad_y, m)
    return np.array([[[px, py]] for py in ys for px in xs], dtype=np.float32)


def _reseed(gray, cx, cy, roi_w, roi_h, n_needed):
    """Frische Ecken NUR innerhalb einer Maske um (cx, cy) - siehe
    Modulkommentar zur Wiederbesetzung."""
    h, w = gray.shape[:2]
    half_w, half_h = roi_w * 0.6, roi_h * 0.6
    x0, x1 = max(0, int(cx - half_w)), min(w, int(cx + half_w))
    y0, y1 = max(0, int(cy - half_h)), min(h, int(cy + half_h))
    if x1 <= x0 or y1 <= y0:
        return None
    mask = np.zeros(gray.shape[:2], dtype=np.uint8)
    mask[y0:y1, x0:x1] = 255
    min_dist = max(2, int(min(roi_w, roi_h) // max(2, n_needed)))
    return cv2.goodFeaturesToTrack(
        gray, maxCorners=max(n_needed * 3, 10), qualityLevel=0.01,
        minDistance=min_dist, blockSize=5, mask=mask)


def analyze(video_path, roi, options):
    """Verfolgt ein Punktgitter in roi=(x,y,w,h) durchs Video. Rückgabe:
    (timestamps_ms, positions, (width, height), scene_cuts, stats) -
    derselbe Vertrag wie backends.py ihn verlangt."""
    max_frames = options.get("max_frames")
    camera_compensation = options.get("camera_compensation", True)
    scene_cut_detection = options.get("scene_cut_detection", True)
    axis = options.get("axis", "y")

    target_n = GRID_N * GRID_M
    quorum = max(2, int(np.ceil(target_n * DEGRADED_QUORUM_FRACTION)))

    cap = cv2.VideoCapture(video_path)
    if not cap.isOpened():
        raise RuntimeError(f"Video konnte nicht geöffnet werden: {video_path}")
    fps = cap.get(cv2.CAP_PROP_FPS) or 30.0
    width = int(cap.get(cv2.CAP_PROP_FRAME_WIDTH))
    height = int(cap.get(cv2.CAP_PROP_FRAME_HEIGHT))

    ok, first_frame = cap.read()
    if not ok:
        cap.release()
        raise RuntimeError("Erster Frame konnte nicht gelesen werden")

    x0, y0, w0, h0 = [float(v) for v in roi]
    last_bbox = (x0, y0, w0, h0)
    cur_cx, cur_cy = x0 + w0 / 2.0, y0 + h0 / 2.0

    prev_gray = cv2.cvtColor(first_frame, cv2.COLOR_BGR2GRAY)
    pts = _seed_grid(roi, GRID_N, GRID_M)

    timestamps_ms = [0]
    y_positions = [y0 + h0 / 2.0]
    x_positions = [x0 + w0 / 2.0]
    camera_dy_cumulative = [0.0]

    tracker_lost_frames = 0  # 0 überlebende Punkte in diesem Frame
    degraded_frames = 0      # >0, aber unter Quorum
    survivor_counts = []
    camera_frames_lost = 0
    scene_cuts = []

    frame_idx = 1
    total_frames = int(cap.get(cv2.CAP_PROP_FRAME_COUNT) or 0)
    if max_frames:
        total_frames = min(total_frames, max_frames) if total_frames else max_frames
    next_report = 0

    while True:
        if frame_idx >= next_report:
            print(f"PROGRESS {frame_idx} {total_frames}", file=sys.stderr, flush=True)
            next_report = frame_idx + max(1, (total_frames or 1000) // 100)
        ok, frame = cap.read()
        if not ok:
            break
        gray = cv2.cvtColor(frame, cv2.COLOR_BGR2GRAY)

        is_cut = scene_cut_detection and detect_scene_cut(prev_gray, gray)
        if is_cut:
            scene_cuts.append(frame_idx)

        if is_cut or pts is None or len(pts) == 0:
            fresh = _reseed(gray, cur_cx, cur_cy, w0, h0, target_n)
            if fresh is not None and len(fresh) > 0:
                pts = fresh[:target_n]
            else:
                pts = _seed_grid((cur_cx - w0 / 2.0, cur_cy - h0 / 2.0, w0, h0), GRID_N, GRID_M)
            survivors_xy = None
        else:
            new_pts, status, _err = cv2.calcOpticalFlowPyrLK(prev_gray, gray, pts, None)
            if new_pts is None or status is None:
                survivors_xy, pts = None, None
            else:
                status = status.reshape(-1)
                survivors_xy = new_pts.reshape(-1, 2)[status == 1]

        n_survivors = 0 if survivors_xy is None else len(survivors_xy)
        survivor_counts.append(n_survivors)

        if n_survivors == 0:
            tracker_lost_frames += 1
            y_positions.append(y_positions[-1])
            x_positions.append(x_positions[-1])
            if pts is None or len(pts) == 0:
                fresh = _reseed(gray, cur_cx, cur_cy, w0, h0, target_n)
                pts = fresh[:target_n] if fresh is not None and len(fresh) > 0 else None
        else:
            if n_survivors < quorum:
                degraded_frames += 1
            med_x = float(np.median(survivors_xy[:, 0]))
            med_y = float(np.median(survivors_xy[:, 1]))
            cur_cx, cur_cy = med_x, med_y
            last_bbox = (med_x - w0 / 2.0, med_y - h0 / 2.0, w0, h0)
            y_positions.append(med_y)
            x_positions.append(med_x)

            missing = target_n - n_survivors
            if missing > 0:
                fresh = _reseed(gray, cur_cx, cur_cy, w0, h0, missing)
                if fresh is not None and len(fresh) > 0:
                    take = fresh[:missing].reshape(-1, 1, 2).astype(np.float32)
                    pts = np.concatenate(
                        [survivors_xy.reshape(-1, 1, 2).astype(np.float32), take], axis=0)
                else:
                    pts = survivors_xy.reshape(-1, 1, 2).astype(np.float32)
            else:
                pts = survivors_xy[:target_n].reshape(-1, 1, 2).astype(np.float32)

        if camera_compensation:
            if is_cut:
                camera_dy_cumulative.append(0.0)
            else:
                dy = estimate_camera_motion(prev_gray, gray, last_bbox)
                if dy == 0.0:
                    camera_frames_lost += 1
                camera_dy_cumulative.append(camera_dy_cumulative[-1] + dy)
        else:
            camera_dy_cumulative.append(0.0)

        prev_gray = gray
        timestamps_ms.append(int(frame_idx * 1000 / fps))
        frame_idx += 1
        if max_frames and frame_idx >= max_frames:
            break

    print(f"PROGRESS {frame_idx} {frame_idx}", file=sys.stderr, flush=True)
    cap.release()

    y_positions = np.array(y_positions)
    x_positions = np.array(x_positions)
    if camera_compensation:
        camera_dy = np.array(camera_dy_cumulative)
        segment_bounds = [0] + list(scene_cuts) + [len(camera_dy)]
        for seg_start, seg_end in zip(segment_bounds[:-1], segment_bounds[1:]):
            seg_len = seg_end - seg_start
            if seg_len >= 9:
                camera_dy[seg_start:seg_end] = savgol_filter(
                    camera_dy[seg_start:seg_end], 9, polyorder=2)
        y_positions = y_positions - camera_dy
        if camera_frames_lost > 0:
            print(f"Kamerakompensation: {camera_frames_lost}/{frame_idx} Frames ohne "
                  "verlässliche Hintergrund-Features (unverändert übernommen)", file=sys.stderr)

    if scene_cuts:
        print(f"{len(scene_cuts)} Szenenschnitt(e) erkannt und Gitter dort neu gesät "
              f"(Frames: {scene_cuts[:10]}{'...' if len(scene_cuts) > 10 else ''})",
              file=sys.stderr)

    if tracker_lost_frames > 0:
        share = tracker_lost_frames / max(1, frame_idx)
        print(f"Gitter hat alle Punkte in {tracker_lost_frames}/{frame_idx} Frames "
              f"({share*100:.0f}%) verloren - dort wurde die letzte bekannte Position "
              "fortgeschrieben", file=sys.stderr)

    vertical_range = float(np.ptp(y_positions)) if len(y_positions) else 0.0
    horizontal_range = float(np.ptp(x_positions)) if len(x_positions) else 0.0
    mean_survivors = float(np.mean(survivor_counts)) if survivor_counts else 0.0

    stats = {
        "tracker_lost_frames": tracker_lost_frames,
        "camera_frames_lost": camera_frames_lost,
        "total_frames": frame_idx,
        "vertical_range": round(vertical_range, 1),
        "horizontal_range": round(horizontal_range, 1),
        "grid_target_points": target_n,
        "grid_quorum": quorum,
        "grid_degraded_frames": degraded_frames,
        "grid_mean_survivors": round(mean_survivors, 2),
    }
    return (np.array(timestamps_ms),
            x_positions if axis == "x" else y_positions,
            (width, height), scene_cuts, stats)
