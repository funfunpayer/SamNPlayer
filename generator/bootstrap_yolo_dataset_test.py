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

    print(("FEHLGESCHLAGEN: " + ", ".join(failures)) if failures else "Alle Prüfungen bestanden.")
    return 1 if failures else 0


if __name__ == "__main__":
    sys.exit(main())
