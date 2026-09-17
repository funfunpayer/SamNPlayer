"""tf/tj-Rezept: Pos-Klammer und Metadata für generate_funscript.py.

tf und tj sind derselbe Abstand-Modus. Enge Distanz = hohe pos = starker Sog.
Playback liest metadata.profile / metadata.device_recipe.
"""

TF_TJ_POS_MIN = 20
TF_TJ_POS_MAX = 90

DEFAULT_CONTACT_VIBRATION_SPAN = 0.75
CONTACT_CURVES = ("linear", "soft", "peak")


def is_distance_profile(name):
    return (name or "").strip().lower() in ("tf", "tj")


def normalize_contact_curve(name):
    n = (name or "").strip().lower()
    if n in ("soft", "soft_entry", "weicher"):
        return "soft"
    if n in ("peak", "strong_peak", "peaky"):
        return "peak"
    return "linear"


def clamp_contact_span(span):
    """0 / None / ungültig → Default; sonst in [0.4, 0.95]."""
    if span is None:
        return DEFAULT_CONTACT_VIBRATION_SPAN
    try:
        v = float(span)
    except (TypeError, ValueError):
        return DEFAULT_CONTACT_VIBRATION_SPAN
    if v <= 0 or v >= 1:
        return DEFAULT_CONTACT_VIBRATION_SPAN
    return max(0.4, min(0.95, v))


def device_recipe_for(profile, contact_vibration=False,
                      contact_vibration_span=None,
                      contact_vibration_curve=None):
    if not is_distance_profile(profile):
        return None
    recipe = {
        "sync": "suction_position",
        "min_suction": 0.20,
        "tick_ms": 50,
        "max_speed": 0.50,
        "smoothing": 0.22,
    }
    # contact_vibration: opt-in, siehe funscript.MapOptions.ContactVibration
    # (mapper.go) - Vibration bleibt 0 solange der Abstand ROI1<->ROI2 nicht
    # nahe seinem für dieses Video beobachteten Minimum liegt. Nur gesetzt,
    # wenn angefordert, damit ältere Player, die das Feld nicht kennen, beim
    # bisherigen (vibrationslosen) Verhalten bleiben.
    if contact_vibration:
        recipe["contact_vibration"] = True
        span = clamp_contact_span(contact_vibration_span)
        curve = normalize_contact_curve(contact_vibration_curve)
        # Nur nicht-Default-Werte schreiben, damit ältere Dateien klein bleiben
        # und Default-Verhalten (0.75 / linear) ohne Extrafelder gilt.
        if abs(span - DEFAULT_CONTACT_VIBRATION_SPAN) > 1e-9:
            recipe["contact_vibration_span"] = span
        if curve != "linear":
            recipe["contact_vibration_curve"] = curve
    return recipe


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


def apply_profile_metadata(metadata, profile, contact_vibration=False,
                           contact_vibration_span=None,
                           contact_vibration_curve=None):
    """Ergänzt metadata um profile + device_recipe für Distanzprofile."""
    meta = dict(metadata)
    if is_distance_profile(profile):
        meta["profile"] = "tj"
        meta["device_recipe"] = device_recipe_for(
            profile, contact_vibration,
            contact_vibration_span=contact_vibration_span,
            contact_vibration_curve=contact_vibration_curve)
    elif profile and profile != "standard":
        meta["profile"] = profile
    return meta
