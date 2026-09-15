#!/usr/bin/env python3
"""auto_roi.py - findet automatisch die Bildregion mit der stärksten
periodischen (rhythmischen) Bewegung, ohne dass der Nutzer von Hand
markieren muss.

Ansatz (entspricht Abschnitt 5+6 der Projekt-Dokumentation: Optical Flow,
Camera Motion Compensation, Motion Fusion):

  1. Video in einem groben Raster (Zellen) analysieren.
  2. Pro Frame-Paar dichten Optical Flow (Farneback) berechnen.
  3. GLOBALE Kamerabewegung schätzen (per Feature-Tracking + affinem
     Modell) und vom Flussfeld ABZIEHEN - sonst würde ein Kameraschwenk
     als "Bewegung überall" erkannt und die ROI wäre Zufall.
  4. Pro Zelle die vertikale Restbewegung über die Zeit sammeln.
  5. Bewertung: nicht die *stärkste* Bewegung gewinnt, sondern die am
     stärksten PERIODISCHE - gemessen über die spektrale Konzentration
     (FFT) im plausiblen Frequenzband. Eine zufällig zappelnde Region hat
     breit verteilte Energie, eine rhythmische einen klaren Peak.
  6. Die Zellen mit der besten Bewertung zu einer zusammenhängenden
     Region zusammenfassen.

Bewusst KEIN trainiertes Objekterkennungs-Modell: dieser Ansatz ist
inhaltsunabhängig (er sucht Rhythmus, nicht bestimmte Objekte), braucht
keine Modellgewichte im Programm und funktioniert auch bei Material, für
das kein passendes Modell trainiert wurde.

Nutzung (eigenständig oder aus generate_funscript.py importiert):
  python3 auto_roi.py --video input.mp4
    -> gibt "ROI x,y,w,h" auf stdout aus
"""

import argparse
import sys

import cv2
import numpy as np


def estimate_camera_motion(prev_gray, gray):
    """Schätzt die globale (Kamera-)Bewegung zwischen zwei Frames als
    affine Transformation. Gibt (dx, dy) der geschätzten Verschiebung
    zurück, oder (0,0) wenn keine verlässliche Schätzung möglich war."""
    prev_pts = cv2.goodFeaturesToTrack(
        prev_gray, maxCorners=200, qualityLevel=0.01, minDistance=20, blockSize=7
    )
    if prev_pts is None or len(prev_pts) < 10:
        return 0.0, 0.0

    next_pts, status, _ = cv2.calcOpticalFlowPyrLK(prev_gray, gray, prev_pts, None)
    if next_pts is None or status is None:
        return 0.0, 0.0

    good_prev = prev_pts[status.flatten() == 1]
    good_next = next_pts[status.flatten() == 1]
    if len(good_prev) < 10:
        return 0.0, 0.0

    matrix, _ = cv2.estimateAffinePartial2D(good_prev, good_next, method=cv2.RANSAC)
    if matrix is None:
        return 0.0, 0.0
    return float(matrix[0, 2]), float(matrix[1, 2])


def periodicity_score(signal, fps):
    """Bewertet, wie stark ein Signal von einer einzelnen Frequenz
    dominiert wird (= rhythmisch statt zufällig). Höher = periodischer.

    Beschränkt auf 0.1-4 Hz: darunter sind es eher langsame Drifts/
    Szenenwechsel, darüber Bildrauschen oder Flackern - beides soll die
    ROI-Wahl nicht gewinnen.
    """
    if len(signal) < 8:
        return 0.0
    sig = np.asarray(signal, dtype=float)
    sig = sig - sig.mean()
    if np.allclose(sig, 0):
        return 0.0

    spectrum = np.abs(np.fft.rfft(sig))
    freqs = np.fft.rfftfreq(len(sig), d=1.0 / fps)

    band = (freqs >= 0.1) & (freqs <= 4.0)
    if not band.any():
        return 0.0
    band_spectrum = spectrum[band]
    total = band_spectrum.sum()
    if total <= 0:
        return 0.0

    # Anteil der stärksten Frequenz an der Gesamtenergie im Band, gewichtet
    # mit der absoluten Bewegungsstärke - eine klar rhythmische, aber
    # winzige Bewegung soll nicht über eine kräftige rhythmische siegen.
    concentration = band_spectrum.max() / total
    amplitude = float(np.abs(sig).mean())
    return float(concentration * amplitude)


def _peak_regions(scores, min_cells=1, decay=0.4, max_regions=6):
    """Findet nacheinander die stärksten lokalen Bewegungsregionen.

    Nimmt die stärkste noch verfügbare Zelle als Kern und wächst von dort
    per Breitensuche nur in Nachbarzellen, deren Wert mindestens `decay`-mal
    den des Kerns erreicht; die gefundene Region wird dann aus der Menge
    entfernt, bevor die nächste gesucht wird. Das gibt jedem Objekt seine
    eigene Region, auch wenn zwei Objekte nah beieinander liegen - siehe
    find_two_rois()'s Docstring für die Messung, die das nötig gemacht hat.

    decay=0.4 ist GEMESSEN, nicht geraten, aber der Parameter reagiert
    unruhig statt glatt: an denselben zwei Testszenen schnitt 0.6 zahlenmäßig
    noch etwas besser ab (enger Fall +0.75 statt +0.63), aber 0.5 - direkt
    dazwischen - ließ die Korrelation im weiter auseinanderliegenden Fall
    auf -0.03 einbrechen (Vorzeichenwechsel), und 0.7 brach im engen Fall
    auf -0.80 ein. 0.3-0.4 ist der einzige der getesteten Werte, der auf
    beiden Szenen durchgehend deutlich positiv blieb - ein ruhiger Bereich
    statt eines scharfen Optimums, siehe docs/NEXT.md Priorität 8's
    "gentle upscaling"-Abschnitt für dieselbe Lehre (nicht von drei Punkten
    auf eine Kurve schließen). Deshalb hier der sichere Wert, nicht der
    zahlenmäßig beste einer einzelnen Messung.
    """
    rows, cols = scores.shape
    available = np.ones_like(scores, dtype=bool)
    regions = []
    for _ in range(max_regions):
        masked = np.where(available, scores, -np.inf)
        r0, c0 = np.unravel_index(np.argmax(masked), masked.shape)
        peak = masked[r0, c0]
        if not np.isfinite(peak) or peak <= 0:
            break
        core = peak * decay
        stack = [(r0, c0)]
        available[r0, c0] = False
        cells = []
        while stack:
            r, c = stack.pop()
            cells.append((r, c))
            for dr in (-1, 0, 1):
                for dc in (-1, 0, 1):
                    if dr == 0 and dc == 0:
                        continue
                    nr, nc = r + dr, c + dc
                    if (0 <= nr < rows and 0 <= nc < cols and available[nr, nc]
                            and scores[nr, nc] >= core):
                        available[nr, nc] = False
                        stack.append((nr, nc))
        if len(cells) >= min_cells:
            regions.append(cells)
    return regions


def find_two_rois(video_path, **kwargs):
    """Sucht ZWEI Regionen, deren ABSTAND die stärkste Bewegung zeigt.

    Hintergrund: Bei echtem Material ist meist die relative Bewegung zweier
    Körper das eigentliche Signal, nicht die absolute Bewegung eines
    Bildbereichs. Eine einzelne verfolgte Region misst nur, wie weit sich
    dieser Bereich im Bild verschiebt - das ist oft deutlich weniger als der
    Abstand zwischen zwei Bereichen sich ändert.

    Gemessen an einem Vergleich mit einem etablierten Fremdprogramm auf
    demselben Film: unsere Einzelpunktmessung lieferte rund viermal
    schwächere Bewegung, obwohl die Signalverarbeitung an synthetischem
    Material nachweislich verlustfrei arbeitet (Intensität 162,1 gegen ideal
    161,4). Der Unterschied steckt also in der Messgröße selbst.

    ACHTUNG - weiterhin NICHT als GUI-Standard verdrahtet (siehe
    docs/NEXT.md, Priorität 3): jetzt gemessen ausreichend gegen von Hand
    gewählte Regionen, aber nur an synthetischem Material - eine Validierung
    über mehrere echte Clips steht noch aus, bevor das zum Standardpfad wird.

    Ursprünglich (Zellenpaare aus einem starren Raster, jede Zelle ein
    eigener Kandidat, benachbarte Zellpaare ausgeschlossen) an einem
    Testvideo mit zwei nur rund 25px auseinanderliegenden Objekten gemessen:

        von Hand gewählte Regionen        +0.62
        automatisch, Raster 12x8          +0.18
        automatisch, Raster 20x16         +0.17
        automatisch, Raster 28x22         +0.28

    Die Ursache: die beiden Objekte fielen überwiegend in dieselben oder
    benachbarte Zellen, und die Regel "räumlich getrennt" (Zellabstand >= 2)
    schloss genau diese Paare aus - übrig blieben Paare, bei denen eine
    Zelle nur Hintergrund zeigte. Ein feineres Raster half kaum, weil sich
    das Problem nicht durch mehr Zellen löst, sondern dadurch, dass jede
    einzelne Zelle nur einen winzigen Ausschnitt eines Objekts sieht statt
    seiner ganzen Ausdehnung.

    Ersetzt durch `_peak_regions()`: statt einzelner Zellen als Kandidaten
    werden zusammenhängende Bewegungsregionen um die jeweils stärkste noch
    unverbrauchte Zelle gebildet (siehe deren Docstring) - jedes Objekt
    bekommt dadurch seine eigene Region, auch wenn beide nah beieinander
    liegen, weil die Region um die schwächere Zelle erst NACH Entfernen der
    stärkeren gesucht wird, statt beide in einem einzigen Schwellwert-Klumpen
    zu verschmelzen. Neu gemessen (generator/auto_roi_two_point_test.py,
    zwei synthetische Szenen - eng beieinander wie oben, und die bereits
    bestehenden two_point_test.py-Fixtures mit größerem Objektabstand):

        eng beieinander (~25px, wie oben)   alt +0.01   neu +0.63
        weiter auseinander (two_point_test) alt +0.17   neu +0.21

    Beide Fälle verbessert, der enge deutlich - genau das Szenario, das
    vorher am schlechtesten abschnitt. Für den Erzeugungspfad bleibt es
    trotzdem bei --roi2 von Hand (bzw. der KI-Regionsvorschlag mit
    Bestätigung, docs/AI_ADAPTER.md): eine Messung an synthetischem
    Material allein reicht laut Priorität 3's Abnahmekriterien nicht, um
    das zum automatischen Standard zu machen.
    """
    result = find_roi(video_path, _return_series=True, **kwargs)
    scores, series, geometry = result
    cell_w, cell_h, scale, width, height, fps = geometry

    regions = _peak_regions(scores)
    candidates = []
    for cells in regions:
        member_series = [np.asarray(series[r][c], dtype=float) for r, c in cells if len(series[r][c]) > 8]
        if not member_series:
            continue
        n = min(len(s) for s in member_series)
        weights = np.array([scores[r][c] for r, c in cells if len(series[r][c]) > 8])
        stacked = np.array([s[:n] for s in member_series])
        agg = np.average(stacked, axis=0, weights=weights)
        candidates.append((cells, agg))

    if len(candidates) < 2:
        raise RuntimeError("Zu wenige bewegte Regionen für eine Zwei-Punkt-Messung gefunden")

    # Auf gleiche Länge bringen: Regionen können unterschiedlich viele
    # verwertbare Messwerte haben.
    n = min(len(c[1]) for c in candidates)
    best = None
    for i in range(len(candidates)):
        for j in range(i + 1, len(candidates)):
            cellsA, sA = candidates[i]
            cellsB, sB = candidates[j]
            # Bewertet wird die RHYTHMIK der Differenz, nicht ihre
            # Auslenkung - das Aufsummieren von Flow driftet, nach der
            # größten Auslenkung auszuwählen kürt damit zuverlässig das
            # Paar mit der stärksten Drift statt dem stärksten Signal.
            difference = np.cumsum(sA[:n]) - np.cumsum(sB[:n])
            quality = periodicity_score(difference, fps)
            if best is None or quality > best[0]:
                best = (quality, cellsA, cellsB)

    if best is None:
        raise RuntimeError("Kein geeignetes Regionenpaar gefunden")

    def to_box(cells):
        rs = [r for r, c in cells]
        cs = [c for r, c in cells]
        r0, r1 = min(rs), max(rs) + 1
        c0, c1 = min(cs), max(cs) + 1
        x = int(c0 * cell_w / scale)
        y = int(r0 * cell_h / scale)
        w = max(24, int((c1 - c0) * cell_w / scale))
        h = max(24, int((r1 - r0) * cell_h / scale))
        x = max(0, min(x, width - w))
        y = max(0, min(y, height - h))
        return (x, y, w, h)

    return to_box(best[1]), to_box(best[2])


def find_roi(video_path, grid_cols=12, grid_rows=8, max_seconds=45, sample_every=2,
             start_frame=0, end_frame=None, report_progress=True, _return_series=False):
    """Analysiert das Video und liefert (x, y, w, h) der besten Region.

    max_seconds begrenzt die Analysedauer. sample_every überspringt Frames
    für Tempo.

    start_frame/end_frame beschränken die Analyse auf einen Abschnitt - so
    kann nach einem Szenenschnitt die Region für die NEUE Szene bestimmt
    werden, statt eine einmal am Videoanfang gefundene Region über
    Schnittgrenzen hinweg weiterzuverwenden. Ohne das ist der Tracker nach
    einem Schnitt auf die letzte bekannte Position angewiesen, die in der
    neuen Szene beliebig falsch sein kann.
    """
    cap = cv2.VideoCapture(video_path)
    if not cap.isOpened():
        raise RuntimeError(f"Video konnte nicht geöffnet werden: {video_path}")

    fps = cap.get(cv2.CAP_PROP_FPS) or 30.0
    width = int(cap.get(cv2.CAP_PROP_FRAME_WIDTH))
    height = int(cap.get(cv2.CAP_PROP_FRAME_HEIGHT))
    max_frames = int(max_seconds * fps)
    if end_frame is not None:
        max_frames = min(max_frames, max(1, end_frame - start_frame))
    if start_frame > 0:
        cap.set(cv2.CAP_PROP_POS_FRAMES, start_frame)

    # Für die Analyse verkleinern - Optical Flow auf voller Auflösung ist
    # unnötig langsam, die Bewegungsregion findet man auch grob.
    scale = 320.0 / width if width > 320 else 1.0
    small_w, small_h = int(width * scale), int(height * scale)
    cell_w, cell_h = small_w / grid_cols, small_h / grid_rows

    ok, prev = cap.read()
    if not ok:
        raise RuntimeError("Erster Frame konnte nicht gelesen werden")
    prev_small = cv2.resize(prev, (small_w, small_h))
    prev_gray = cv2.cvtColor(prev_small, cv2.COLOR_BGR2GRAY)

    # Pro Zelle eine Zeitreihe der vertikalen Restbewegung.
    series = [[[] for _ in range(grid_cols)] for _ in range(grid_rows)]

    frame_idx = 0
    analysed = 0
    # Die Analyse liest jeden Frame und rechnet dichten Optical Flow - das
    # dauert bei längeren Videos spürbar. Ohne Rückmeldung wirkt die
    # Oberfläche eingefroren, deshalb wird der Fortschritt maschinenlesbar
    # gemeldet ("PROGRESS <erledigt> <gesamt>"), damit die GUI daraus einen
    # Balken bauen kann. Die Zeile geht nach stderr wie alle Statusausgaben;
    # das Ergebnis steht weiterhin allein auf stdout.
    total_frames = max(1, max_frames)
    next_report = 0
    while frame_idx < max_frames:
        if report_progress and frame_idx >= next_report:
            print(f"PROGRESS {frame_idx} {total_frames}", file=sys.stderr, flush=True)
            next_report = frame_idx + max(1, total_frames // 100)
        for _ in range(sample_every):
            ok, frame = cap.read()
            frame_idx += 1
            if not ok:
                break
        if not ok:
            break

        small = cv2.resize(frame, (small_w, small_h))
        gray = cv2.cvtColor(small, cv2.COLOR_BGR2GRAY)

        cam_dx, cam_dy = estimate_camera_motion(prev_gray, gray)

        flow = cv2.calcOpticalFlowFarneback(
            prev_gray, gray, None,
            pyr_scale=0.5, levels=3, winsize=15,
            iterations=3, poly_n=5, poly_sigma=1.2, flags=0,
        )
        # Kamerabewegung abziehen -> übrig bleibt lokale Eigenbewegung.
        flow_y = flow[..., 1] - cam_dy

        for r in range(grid_rows):
            for c in range(grid_cols):
                y0, y1 = int(r * cell_h), int((r + 1) * cell_h)
                x0, x1 = int(c * cell_w), int((c + 1) * cell_w)
                series[r][c].append(float(flow_y[y0:y1, x0:x1].mean()))

        prev_gray = gray
        analysed += 1

    if report_progress:
        print(f"PROGRESS {total_frames} {total_frames}", file=sys.stderr, flush=True)
    cap.release()

    if analysed < 8:
        raise RuntimeError("Video zu kurz oder nicht lesbar für die automatische Analyse")

    effective_fps = fps / sample_every
    scores = np.zeros((grid_rows, grid_cols))
    for r in range(grid_rows):
        for c in range(grid_cols):
            scores[r, c] = periodicity_score(series[r][c], effective_fps)

    if _return_series:
        return scores, series, (cell_w, cell_h, scale, width, height, fps)

    if scores.max() <= 0:
        raise RuntimeError("Keine rhythmische Bewegung gefunden - bitte Region von Hand markieren")

    # Zellen oberhalb eines Anteils des Maximums als zusammenhängende
    # Region zusammenfassen (statt nur die eine beste Zelle zu nehmen -
    # die eigentliche Bewegungsregion ist meist mehrere Zellen groß).
    threshold = scores.max() * 0.5
    rows, cols = np.where(scores >= threshold)

    x0 = int(cols.min() * cell_w / scale)
    x1 = int((cols.max() + 1) * cell_w / scale)
    y0 = int(rows.min() * cell_h / scale)
    y1 = int((rows.max() + 1) * cell_h / scale)

    # Etwas einschrumpfen: die Rasterzellen sind grob, ein leicht engerer
    # Kasten trackt in der Praxis stabiler als einer mit viel Rand.
    pad_x = int((x1 - x0) * 0.1)
    pad_y = int((y1 - y0) * 0.1)
    x0, x1 = x0 + pad_x, x1 - pad_x
    y0, y1 = y0 + pad_y, y1 - pad_y

    w = max(16, x1 - x0)
    h = max(16, y1 - y0)
    x0 = max(0, min(x0, width - w))
    y0 = max(0, min(y0, height - h))
    return x0, y0, w, h


def main():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--video", required=True)
    ap.add_argument("--max-seconds", type=int, default=45)
    args = ap.parse_args()

    x, y, w, h = find_roi(args.video, max_seconds=args.max_seconds)
    # Maschinenlesbare Zeile für den Go-Aufrufer:
    print(f"ROI {x} {y} {w} {h}")
    print(f"Automatisch gefundene Region: x={x} y={y} w={w} h={h}", file=sys.stderr)


if __name__ == "__main__":
    main()
