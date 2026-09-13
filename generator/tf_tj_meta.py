"""tf/tj-Rezept: Pos-Klammer und Metadata für generate_funscript.py.

tf und tj sind derselbe Abstand-Modus. Enge Distanz = hohe pos = starker Sog.
Playback liest metadata.profile / metadata.device_recipe.
"""

TF_TJ_POS_MIN = 20
TF_TJ_POS_MAX = 90


def is_distance_profile(name):
    return (name or "").strip().lower() in ("tf", "tj")


def device_recipe_for(profile):
    if not is_distance_profile(profile):
        return None
    return {
        "sync": "suction_position",
        "min_suction": 0.20,
        "tick_ms": 50,
        "max_speed": 0.50,
        "smoothing": 0.22,
    }


def clamp_actions_pos(actions, lo=TF_TJ_POS_MIN, hi=TF_TJ_POS_MAX):
    out = []
    for a in actions:
        pos = int(a["pos"])
        if pos < lo:
            pos = lo
        elif pos > hi:
            pos = hi
        out.append({"at": a["at"], "pos": pos})
    return out


def apply_profile_metadata(metadata, profile):
    """Ergänzt metadata um profile + device_recipe für Distanzprofile."""
    meta = dict(metadata)
    if is_distance_profile(profile):
        meta["profile"] = "tj"
        meta["device_recipe"] = device_recipe_for(profile)
    elif profile and profile != "standard":
        meta["profile"] = profile
    return meta
