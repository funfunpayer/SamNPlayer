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
    assert m.device_recipe_for("standard") is None


def test_clamp():
    out = m.clamp_actions_pos([{"at": 0, "pos": 0}, {"at": 1, "pos": 100}])
    assert out[0]["pos"] == 20
    assert out[1]["pos"] == 90


def test_metadata():
    meta = m.apply_profile_metadata({"creator": "x"}, "tf")
    assert meta["profile"] == "tj"
    assert meta["device_recipe"]["sync"] == "suction_position"


if __name__ == "__main__":
    test_is_distance_profile()
    test_device_recipe()
    test_clamp()
    test_metadata()
    print("ok")
