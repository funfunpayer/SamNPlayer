#!/usr/bin/env python3
"""bootstrap_yolo_dataset.py - erzeugt einen YOLO-Trainingsdatensatz aus
bereits vorhandenem klassischen Tracking (CSRT, track_roi), statt Boxen von
Hand zu zeichnen.

Hintergrund (docs/AI_ADAPTER.md, "Still open: no bundled/recommended ONNX
model"): `generator/ai_roi.py` erwartet ein ONNX-Modell mit dem Vertrag
"(N,6) normalisierte [x0,y0,x1,y1,confidence,class_id]", aber es existiert
keins. Ein Objekterkenner von Grund auf zu trainieren braucht annotierte
Frames - CSRT liefert genau das schon: eine Box pro Frame, für jeden Clip,
den wir bereits erfolgreich getrackt haben. Der Nutzer wollte beides
kombinieren (Chat, 15. September 2026): dieses Skript liefert die
Bootstrap-Basis aus dem Tracking, danach lässt sich der Datensatz mit einem
Standard-Werkzeug (LabelImg, CVAT, makesense.ai - alle können YOLO-Format
lesen/schreiben) von Hand nachbessern, bevor trainiert wird.

WICHTIG - was dieses Modell NICHT genauer machen kann als das Original: der
Detektor lernt, die CSRT-Box nachzuahmen. Er wird nie zuverlässiger als der
Tracker, der die Trainingsdaten geliefert hat - sein Nutzen ist Geschwindigkeit
(ein Einzelbild-Vorschlag statt iterativem Tracking über viele Frames) und
Generalisierung auf neue Clips ohne erneutes Tracking, nicht höhere Präzision
auf demselben Material. Von Hand nachgebesserte Beispiele (falsche/verlorene
CSRT-Boxen korrigiert) sind deshalb wichtig, nicht nur Fleißarbeit.

Format (Ultralytics-YOLO-Standard, siehe
https://docs.ultralytics.com/datasets/detect/):
  <output_dir>/
    images/train/*.jpg, images/val/*.jpg
    labels/train/*.txt, labels/val/*.txt   (eine Zeile "class xc yc w h",
                                             alle Werte 0..1 relativ zur
                                             Bildgröße)
    data.yaml

Wiederholte Aufrufe auf denselben output_dir mit verschiedenen Videos/ROIs
HÄNGEN AN (splittet neu in train/val, überschreibt bestehende Bilder nicht -
Dateinamen enthalten den Video-Stem), damit sich der Datensatz über mehrere
Clips aufbauen lässt, wie vom Nutzer gewünscht.

Mehrere Inhaltskategorien im selben Datensatz (Chat, 15. September 2026 -
"komplett kombinieren was zusammen passt", mit besonderem Augenmerk auf
TF/TJ-Mischformen): --class-name ordnet jedem Aufruf eine Kategorie zu
(z.B. "blowjob", "tf_tj_mix"); die Zuordnung Name->ID wird in
<output_dir>/classes.json über mehrere Aufrufe hinweg gemerkt, damit
data.yaml am Ende ALLE bisher angelegten Kategorien enthält statt nur die
des letzten Aufrufs. Eine eigene Klasse für TF/TJ-Inhalte bedeutet: der
Detektor lernt diese Bewegungsform getrennt von reinem Blowjob-Material zu
erkennen, statt sie in einer gemeinsamen Klasse zu verwässern - wie viele
Beispiele jede Kategorie bekommt (und damit wie stark sie im Ergebnis
gewichtet ist), bestimmt allein, wie oft mit dem jeweiligen --class-name
aufgerufen wird.

Nutzung:
  python3 bootstrap_yolo_dataset.py --video clip.mp4 --roi 61,63,154,72 \
      --output-dir ./yolo_dataset --class-name tf_tj_mix
"""

import argparse
import hashlib
import json
import os
import random
import sys

import cv2

import generate_funscript as g


def track_center_path(video_path, roi, max_frames=None, cache_dir=None,
                       camera_compensation=True, scene_cut_detection=True):
    """Liefert (timestamps_ms, x_centers, y_centers, frame_size) - track_roi
    gibt öffentlich nur EINE Achse zurück (je nach axis-Parameter), für eine
    Bounding-Box braucht es beide. Trackt darum zweimal (x, y) statt
    generate_funscript.py's getestete track_roi()/track_roi_cached()
    anzufassen, um deren Vertrag und Tests unverändert zu lassen - einmalige
    Mehrkosten für einen Offline-Datensatzlauf, nicht Teil des
    interaktiven Erzeugungspfads."""
    ts, y_centers, frame_size, _cuts, _stats = g.track_roi_cached(
        video_path, roi, max_frames=max_frames, cache_dir=cache_dir,
        camera_compensation=camera_compensation,
        scene_cut_detection=scene_cut_detection, axis="y")
    _ts2, x_centers, _fs2, _cuts2, _stats2 = g.track_roi_cached(
        video_path, roi, max_frames=max_frames, cache_dir=cache_dir,
        camera_compensation=camera_compensation,
        scene_cut_detection=scene_cut_detection, axis="x")
    return ts, x_centers, y_centers, frame_size


def _box_to_yolo_label(cx, cy, box_w, box_h, width, height):
    """Wandelt eine Boxmitte+Größe in Pixeln in eine YOLO-Labelzeile
    (xc,yc,w,h, alle 0..1 relativ zur Bildgröße) um, geklammert an die
    Bildgrenzen. Gibt None zurück, wenn die geklammerte Box entartet ist
    (Mittelpunkt liegt vollständig außerhalb des Bildes) - dieser Frame wird
    dann übersprungen statt einer leeren/negativen Box."""
    x0 = max(0.0, cx - box_w / 2.0)
    y0 = max(0.0, cy - box_h / 2.0)
    x1 = min(float(width), cx + box_w / 2.0)
    y1 = min(float(height), cy + box_h / 2.0)
    if x1 <= x0 or y1 <= y0:
        return None
    xc_n = (x0 + x1) / 2.0 / width
    yc_n = (y0 + y1) / 2.0 / height
    w_n = (x1 - x0) / width
    h_n = (y1 - y0) / height
    return xc_n, yc_n, w_n, h_n


def _video_stem(video_path):
    """Kurzer, stabiler Dateiname-Präfix - Hash statt Klartext-Videoname,
    damit Sonderzeichen/Länge im Originalnamen keine Rolle spielen."""
    h = hashlib.sha1(os.path.abspath(video_path).encode("utf-8")).hexdigest()[:10]
    base = os.path.splitext(os.path.basename(video_path))[0]
    safe = "".join(c if c.isalnum() else "_" for c in base)[:40]
    return f"{safe}_{h}"


def build_dataset(video_path, roi, output_dir, class_id=0, sample_every=12,
                   max_frames=None, cache_dir=None, val_fraction=0.15, seed=0):
    """Trackt roi durchs Video, schreibt jeden sample_every-ten Frame plus
    YOLO-Label (feste Boxgröße = roi[2],roi[3], Mittelpunkt = getrackte
    Position) in output_dir. sample_every=12 ist ein Kompromiss: bei 25fps
    etwa alle 0.5s ein Bild - benachbarte Videoframes sind sich fast
    identisch, zu dichte Abtastung bläht den Datensatz nur mit redundanten,
    stark korrelierten Beispielen auf, ohne das Modell robuster zu machen."""
    roi = tuple(int(v) for v in roi)
    print(f"Tracke {video_path} (Referenz-ROI {roi})...", file=sys.stderr)
    _ts, x_centers, y_centers, (width, height) = track_center_path(
        video_path, roi, max_frames=max_frames, cache_dir=cache_dir)
    n = min(len(x_centers), len(y_centers))
    print(f"{n} Frames getrackt, Videogröße {width}x{height}", file=sys.stderr)

    img_train_dir = os.path.join(output_dir, "images", "train")
    img_val_dir = os.path.join(output_dir, "images", "val")
    lbl_train_dir = os.path.join(output_dir, "labels", "train")
    lbl_val_dir = os.path.join(output_dir, "labels", "val")
    for d in (img_train_dir, img_val_dir, lbl_train_dir, lbl_val_dir):
        os.makedirs(d, exist_ok=True)

    stem = _video_stem(video_path)
    rng = random.Random(seed)
    box_w, box_h = roi[2], roi[3]

    cap = cv2.VideoCapture(video_path)
    if not cap.isOpened():
        raise RuntimeError(f"Video konnte nicht geöffnet werden: {video_path}")

    written = 0
    frame_idx = 0
    sample_idx = 0
    try:
        while frame_idx < n:
            ok, frame = cap.read()
            if not ok:
                break
            if frame_idx % sample_every == 0:
                is_val = rng.random() < val_fraction
                img_dir = img_val_dir if is_val else img_train_dir
                lbl_dir = lbl_val_dir if is_val else lbl_train_dir

                cx, cy = x_centers[frame_idx], y_centers[frame_idx]
                label = _box_to_yolo_label(cx, cy, box_w, box_h, width, height)
                if label is None:
                    frame_idx += 1
                    continue
                xc_n, yc_n, w_n, h_n = label

                name = f"{stem}_{sample_idx:06d}"
                cv2.imwrite(os.path.join(img_dir, name + ".jpg"), frame)
                with open(os.path.join(lbl_dir, name + ".txt"), "w") as f:
                    f.write(f"{class_id} {xc_n:.6f} {yc_n:.6f} {w_n:.6f} {h_n:.6f}\n")
                written += 1
                sample_idx += 1
            frame_idx += 1
    finally:
        cap.release()

    print(f"{written} annotierte Frames geschrieben ({video_path})", file=sys.stderr)
    return written


CLASS_REGISTRY_FILENAME = "classes.json"


def _load_class_registry(output_dir):
    path = os.path.join(output_dir, CLASS_REGISTRY_FILENAME)
    if not os.path.exists(path):
        return {}
    with open(path) as f:
        return json.load(f)


def _save_class_registry(output_dir, registry):
    path = os.path.join(output_dir, CLASS_REGISTRY_FILENAME)
    with open(path, "w") as f:
        json.dump(registry, f, indent=2, sort_keys=True)
    return path


def _assign_class_id(registry, class_name, class_id=None):
    """Reiner Teil der Klassenverwaltung (kein Dateizugriff): liefert
    (id, aktualisierte_registry) für class_name aus registry (Name -> ID).
    Ohne class_id wird eine bereits vorhandene ID wiederverwendet oder die
    nächste freie vergeben; mit class_id wird genau diese erzwungen. Gibt
    bei unveränderter Zuordnung dasselbe registry-Objekt zurück (nicht nur
    einen gleichen Wert), damit der Aufrufer per `is` erkennt, ob
    gespeichert werden muss."""
    if class_id is None:
        if class_name in registry:
            return registry[class_name], registry
        class_id = max(registry.values(), default=-1) + 1
    if registry.get(class_name) == class_id:
        return class_id, registry
    updated = dict(registry)
    updated[class_name] = class_id
    return class_id, updated


def register_class(output_dir, class_name, class_id=None):
    """Trägt class_name in <output_dir>/classes.json ein (legt die Datei bei
    Bedarf an) und liefert die tatsächlich verwendete ID zurück - siehe
    Moduldoc zu mehreren Inhaltskategorien im selben Datensatz."""
    registry = _load_class_registry(output_dir)
    resolved_id, updated = _assign_class_id(registry, class_name, class_id)
    if updated is not registry:
        os.makedirs(output_dir, exist_ok=True)
        _save_class_registry(output_dir, updated)
    return resolved_id


def write_data_yaml(output_dir, class_names=None):
    """Ultralytics-Datensatzkonfiguration - siehe
    https://docs.ultralytics.com/datasets/detect/#dataset-yaml-format.
    Ohne class_names werden die Namen aus classes.json übernommen (nach ID
    sortiert) - damit data.yaml immer ALLE über mehrere Aufrufe hinweg
    angelegten Kategorien enthält, nicht nur die des letzten Aufrufs."""
    if class_names is None:
        registry = _load_class_registry(output_dir) or {"motion_region": 0}
        class_names = [name for name, _id in sorted(registry.items(), key=lambda kv: kv[1])]
    path = os.path.join(output_dir, "data.yaml")
    abs_dir = os.path.abspath(output_dir)
    lines = [
        f"path: {abs_dir}",
        "train: images/train",
        "val: images/val",
        "names:",
    ]
    for i, name in enumerate(class_names):
        lines.append(f"  {i}: {name}")
    with open(path, "w") as f:
        f.write("\n".join(lines) + "\n")
    print(f"Geschrieben: {path}", file=sys.stderr)
    return path


def main():
    ap = argparse.ArgumentParser(description=__doc__,
                                  formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--video", required=True, help="Pfad zum Video")
    ap.add_argument("--roi", required=True, metavar="x,y,w,h",
                     help="Start-ROI, von Hand markiert oder aus einem früheren Lauf übernommen")
    ap.add_argument("--output-dir", required=True, help="Zielordner für den YOLO-Datensatz")
    ap.add_argument("--sample-every", type=int, default=12,
                     help="Nur jeden N-ten Frame als Trainingsbeispiel schreiben (Standard: 12)")
    ap.add_argument("--max-frames", type=int, default=None)
    ap.add_argument("--cache-dir", default=None,
                     help="Wie generate_funscript.py --cache-dir - vermeidet erneutes Tracking bei Wiederholung")
    ap.add_argument("--val-fraction", type=float, default=0.15)
    ap.add_argument("--class-id", type=int, default=None,
                     help="Feste Klassen-ID erzwingen - normalerweise nicht nötig, "
                          "die ID wird über classes.json automatisch je --class-name vergeben/wiederverwendet")
    ap.add_argument("--class-name", default="motion_region",
                     help="Inhaltskategorie dieses Clips (z.B. 'blowjob', 'tf_tj_mix') - "
                          "mehrere Kategorien im selben --output-dir werden zu eigenen Klassen kombiniert")
    args = ap.parse_args()

    try:
        roi = tuple(int(v) for v in args.roi.split(","))
        if len(roi) != 4:
            raise ValueError
    except ValueError:
        ap.error('--roi muss "x,y,w,h" sein')

    class_id = register_class(args.output_dir, args.class_name, class_id=args.class_id)
    build_dataset(args.video, roi, args.output_dir, class_id=class_id,
                  sample_every=args.sample_every, max_frames=args.max_frames,
                  cache_dir=args.cache_dir, val_fraction=args.val_fraction)
    write_data_yaml(args.output_dir)


if __name__ == "__main__":
    main()
