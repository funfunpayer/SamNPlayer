"""Tests für das tf/tj-Rezept, ohne Video und ohne OpenCV."""

import tf_tj_meta as m


def test_is_distance_profile():
    assert m.is_distance_profile("tf")
    assert m.is_distance_profile("TJ")
    assert not m.is_distance_profile("weich")
    assert not m.is_distance_profile("")


def test_device_recipe():
    r = m.device_recipe_for("tj")
    assert r["sync"] == "suction_position"
    assert r["min_suction"] == 0.20
    assert "contact_vibration" not in r
    assert m.device_recipe_for("standard") is None


def test_device_recipe_contact_vibration():
    r = m.device_recipe_for("tj", contact_vibration=True)
    assert r["contact_vibration"] is True
    # Nicht angefordert -> Feld fehlt ganz, statt explizit False zu sein -
    # ältere Player, die das Feld nicht kennen, sollen nichts lesen müssen.
    assert "contact_vibration" not in m.device_recipe_for("tj", contact_vibration=False)
    assert m.device_recipe_for("standard", contact_vibration=True) is None


def test_clamp():
    out = m.clamp_actions_pos([{"at": 0, "pos": 0}, {"at": 1, "pos": 100}])
    assert out[0]["pos"] == 20
    assert out[1]["pos"] == 90


def test_metadata():
    meta = m.apply_profile_metadata({"creator": "x"}, "tf")
    assert meta["profile"] == "tj"
    assert meta["device_recipe"]["sync"] == "suction_position"
    assert "contact_vibration" not in meta["device_recipe"]


def test_metadata_contact_vibration():
    meta = m.apply_profile_metadata({"creator": "x"}, "tj", contact_vibration=True)
    assert meta["device_recipe"]["contact_vibration"] is True


if __name__ == "__main__":
    test_is_distance_profile()
    test_device_recipe()
    test_device_recipe_contact_vibration()
    test_clamp()
    test_metadata()
    test_metadata_contact_vibration()
    print("ok")
