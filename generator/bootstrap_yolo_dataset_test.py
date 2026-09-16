"""Test für bootstrap_yolo_dataset.py - den CSRT-Bootstrap für YOLO-Datensätze.

Prüft nur die reinen Funktionen (Box->YOLO-Label-Umrechnung, Klassen-
Registrierung über mehrere Aufrufe hinweg): kein echtes Video, kein Tracking,
keine ultralytics/onnx-Abhängigkeit - genau wie ai_roi_test.py deckt das die
Nachverarbeitung ab, ohne die teuren/optionalen Teile (echtes Tracking, echtes
Training) in einen Unittest zu zwingen.

Ausführen: python3 generator/bootstrap_yolo_dataset_test.py
"""

import sys
import tempfile
from pathlib import Path

import bootstrap_yolo_dataset as b


def main():
    failures = []

    def check(name, cond, detail=""):
        print(("  OK   " if cond else "  FAIL ") + name + (f"  [{detail}]" if not cond else ""))
        if not cond:
            failures.append(name)

    # --- _box_to_yolo_label: normale Box mittig im Bild ---------------------
    label = b._box_to_yolo_label(cx=50, cy=25, box_w=20, box_h=10, width=100, height=50)
    check("normale Box liefert erwartete normalisierte Werte",
          label == (0.5, 0.5, 0.2, 0.2), str(label))

    # --- _box_to_yolo_label: an den Bildrand geklammert ----------------------
    edge = b._box_to_yolo_label(cx=95, cy=45, box_w=20, box_h=20, width=100, height=50)
    check("am rechten/unteren Rand wird auf die Bildgrenze geklammert",
          edge is not None and edge[0] + edge[2] / 2 <= 1.0 and edge[1] + edge[3] / 2 <= 1.0,
          str(edge))

    # --- _box_to_yolo_label: Mittelpunkt komplett außerhalb -> entartet -----
    outside = b._box_to_yolo_label(cx=-50, cy=25, box_w=20, box_h=10, width=100, height=50)
    check("Box komplett außerhalb des Bildes liefert None statt einer entarteten Box",
          outside is None, str(outside))

    # --- Klassen-Registrierung: neue Namen bekommen fortlaufende IDs --------
    with tempfile.TemporaryDirectory() as tmp:
        out = Path(tmp) / "dataset"

        id_bj = b.register_class(str(out), "blowjob")
        id_tftj = b.register_class(str(out), "tf_tj_mix")
        check("erster Name bekommt ID 0", id_bj == 0, str(id_bj))
        check("zweiter Name bekommt die nächste freie ID", id_tftj == 1, str(id_tftj))

        # --- derselbe Name in einem späteren Aufruf bekommt dieselbe ID -----
        id_bj_again = b.register_class(str(out), "blowjob")
        check("wiederholter Aufruf mit demselben Namen liefert dieselbe ID",
              id_bj_again == id_bj, str(id_bj_again))

        # --- data.yaml enthält am Ende ALLE bisher angelegten Kategorien ----
        b.write_data_yaml(str(out))
        yaml_text = (out / "data.yaml").read_text()
        check("data.yaml enthält beide Kategorien, nicht nur die zuletzt registrierte",
              "blowjob" in yaml_text and "tf_tj_mix" in yaml_text, yaml_text)
        check("data.yaml ordnet den Namen die richtige ID zu",
              "0: blowjob" in yaml_text and "1: tf_tj_mix" in yaml_text, yaml_text)

        # --- explizite --class-id erzwingt eine bestimmte ID ----------------
        id_forced = b.register_class(str(out), "extra", class_id=5)
        check("explizite class_id wird übernommen statt automatisch vergeben",
              id_forced == 5, str(id_forced))

    # --- write_data_yaml ohne jede Registrierung: sinnvoller Standardwert ---
    with tempfile.TemporaryDirectory() as tmp2:
        b.write_data_yaml(tmp2)
        yaml_text2 = Path(tmp2, "data.yaml").read_text()
        check("ohne classes.json fällt write_data_yaml auf eine Standardklasse zurück",
              "motion_region" in yaml_text2, yaml_text2)

    # --- _frame_label_lines: mehrere Regionen im selben Bild ----------------
    lines = b._frame_label_lines(
        [(0, 50, 25, 20, 10), (1, 80, 40, 10, 10)], width=100, height=50)
    check("zwei Regionen ergeben zwei Zeilen", len(lines) == 2, str(lines))
    check("erste Zeile trägt Klasse 0", lines[0].startswith("0 "), lines[0])
    check("zweite Zeile trägt Klasse 1", lines[1].startswith("1 "), lines[1])

    # --- _frame_label_lines: eine Region außerhalb wird einzeln übersprungen
    lines_partial = b._frame_label_lines(
        [(0, 50, 25, 20, 10), (1, -50, 25, 10, 10)], width=100, height=50)
    check("nur die im Bild liegende Region bleibt übrig, der Frame wird nicht verworfen",
          len(lines_partial) == 1 and lines_partial[0].startswith("0 "), str(lines_partial))

    # --- Kontrollansicht: Label lesen/schreiben/auflisten/verwerfen ---------
    with tempfile.TemporaryDirectory() as tmp3:
        out = Path(tmp3) / "dataset"
        img_dir = out / "images" / "train"
        lbl_dir = out / "labels" / "train"
        img_dir.mkdir(parents=True)
        lbl_dir.mkdir(parents=True)

        video_path = "/videos/clip.mp4"
        stem = b._video_stem(video_path)
        image_path = img_dir / f"{stem}_000000.jpg"
        label_path = lbl_dir / f"{stem}_000000.txt"
        image_path.write_bytes(b"fake jpg bytes")

        check("_label_path_for_image bildet images/ auf labels/ ab, Endung .txt",
              b._label_path_for_image(str(image_path)) == str(label_path),
              b._label_path_for_image(str(image_path)))

        check("read_label_file liefert leere Liste, wenn die Labeldatei fehlt",
              b.read_label_file(str(label_path)) == [], "")

        boxes = [(0, 0.5, 0.5, 0.2, 0.2), (1, 0.8, 0.4, 0.1, 0.1)]
        b.write_label_file(str(label_path), boxes)
        reread = b.read_label_file(str(label_path))
        check("write_label_file/read_label_file: Rundlauf liefert dieselben Boxen",
              reread == boxes, str(reread))

        samples = b.list_samples(str(out))
        check("list_samples findet das eine Beispiel mit seinen Boxen",
              len(samples) == 1 and samples[0]["boxes"] == boxes, str(samples))

        # zweites Beispiel von einem ANDEREN Video - video_path-Filter grenzt ein
        other_video = "/videos/other.mp4"
        other_stem = b._video_stem(other_video)
        other_image = img_dir / f"{other_stem}_000000.jpg"
        other_image.write_bytes(b"fake jpg bytes")
        b.write_label_file(str(b._label_path_for_image(str(other_image))), [(0, 0.5, 0.5, 0.1, 0.1)])
        check("list_samples ohne video_path findet beide Beispiele",
              len(b.list_samples(str(out))) == 2, "")
        filtered = b.list_samples(str(out), video_path=video_path)
        check("list_samples MIT video_path grenzt auf dessen eigene Beispiele ein",
              len(filtered) == 1 and filtered[0]["name"] == image_path.name, str(filtered))

        b.discard_sample(str(image_path))
        check("discard_sample entfernt Bild und Labeldatei",
              not image_path.exists() and not label_path.exists(), "")
        check("discard_sample lässt das andere Beispiel unberührt",
              len(b.list_samples(str(out))) == 1, "")

    print(("FEHLGESCHLAGEN: " + ", ".join(failures)) if failures else "Alle Prüfungen bestanden.")
    return 1 if failures else 0


if __name__ == "__main__":
    sys.exit(main())
