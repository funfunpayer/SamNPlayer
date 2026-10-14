"""Generates the Flow background ribbons (visual targets docs/visual-target/*.png):
silky blue / orange light waves in the top-right and bottom-left corners.

Run:  python3 scripts/art/flow_background.py
Writes cmd/gui-wails/frontend/src/assets/flow/bg-top.svg, bg-bottom.svg and
stage-ribbons.svg (behind the training rings).
Deterministic (fixed seeds), so re-running gives the same files.

A ribbon is many thin strands around one path (two joined cubic Béziers),
pinched where the light is brightest and fanned out towards the ends, over a
blurred glow and a white-hot core — the "silk" look of the mockups.
"""
import pathlib
import random

OUT = pathlib.Path(__file__).resolve().parents[2] / "cmd/gui-wails/frontend/src/assets/flow"

BLUE = ("#0636ff", "#1677ff", "#3fd0ff", "#e9fbff")
ORANGE = ("#ff2a00", "#ff6410", "#ffa030", "#fff0d0")


def grad(gid, x1, x2, stops):
    s = "".join(f'<stop offset="{o:.2f}" stop-color="{c}" stop-opacity="{a}"/>' for o, c, a in stops)
    return (f'<linearGradient id="{gid}" gradientUnits="userSpaceOnUse" x1="{x1}" y1="0" x2="{x2}" y2="0">'
            f"{s}</linearGradient>")


def ribbon(rng, gid, colors, pts, x1, x2, peak, *, n=44, spread=(30, 12, 2, 6, 18, 34, 46),
           width=1.0, opacity=1.0, glow=1.0):
    """pts: 7 points (start, c1, c2, mid, c3, c4, end). spread: fan-out per
    point (pixels, perpendicular-ish = vertical). peak: 0–1 where along x1..x2
    the light is brightest."""
    deep, mid, light, hot = colors
    a, b = max(0.0, peak - 0.42), min(1.0, peak + 0.42)
    defs = [
        grad(gid, x1, x2, [(0, deep, 0), (a, deep, 0.35), (peak - 0.08, mid, 1), (peak + 0.06, light, 1),
                           (b, mid, 0.5), (1, deep, 0)]),
        grad(gid + "h", x1, x2, [(max(0, peak - 0.25), hot, 0), (peak, hot, 1), (min(1, peak + 0.22), hot, 0)]),
    ]

    def path(t, jitter):
        q = []
        for (x, y), s in zip(pts, spread):
            q.append((x + rng.uniform(-jitter, jitter), y + t * s + rng.uniform(-jitter, jitter) * s / 20))
        f = lambda p: f"{p[0]:.1f} {p[1]:.1f}"
        return f"M{f(q[0])} C{f(q[1])} {f(q[2])} {f(q[3])} C{f(q[4])} {f(q[5])} {f(q[6])}"

    body = [f'<path d="{path(0, 0)}" stroke="url(#{gid})" stroke-width="{26 * width:.0f}" '
            f'opacity="{0.55 * glow * opacity:.2f}" filter="url(#glow)"/>']
    for i in range(n):
        t = (i / (n - 1)) * 2 - 1           # −1 … 1 across the ribbon
        op = opacity * (1 - 0.75 * abs(t)) * rng.uniform(0.55, 1.0)
        sw = width * rng.uniform(0.35, 1.1)
        body.append(f'<path d="{path(t, 1.5)}" stroke="url(#{gid})" stroke-width="{sw:.2f}" opacity="{op:.2f}"/>')
    body.append(f'<path d="{path(0, 0)}" stroke="url(#{gid}h)" stroke-width="{3.2 * width:.1f}" '
                f'opacity="{opacity:.2f}" filter="url(#soft)"/>')
    body.append(f'<path d="{path(0, 0)}" stroke="url(#{gid}h)" stroke-width="{1.1 * width:.1f}" opacity="{opacity:.2f}"/>')
    return defs, body


def svg(w, h, parts):
    defs = [f'<filter id="glow" filterUnits="userSpaceOnUse" x="-100" y="-100" width="{w + 200}" height="{h + 200}">'
            '<feGaussianBlur stdDeviation="9"/></filter>',
            f'<filter id="soft" filterUnits="userSpaceOnUse" x="-100" y="-100" width="{w + 200}" height="{h + 200}">'
            '<feGaussianBlur stdDeviation="2.2"/></filter>']
    body = []
    for d, b in parts:
        defs += d
        body += b
    return (f'<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 {w} {h}" width="{w}" height="{h}">'
            f'<defs>{"".join(defs)}</defs><g fill="none" stroke-linecap="round">{"".join(body)}</g></svg>\n')


def top():
    """Top-right corner, 700 × 120 (anchored right / top)."""
    r = random.Random(7)
    return svg(700, 120, [
        # faint upper sheet
        ribbon(r, "t3", BLUE, [(120, -10), (240, -8), (300, 20), (380, 30), (470, 40), (580, 10), (720, -5)],
               120, 720, 0.45, n=26, spread=(10, 8, 6, 8, 12, 16, 20), width=0.7, opacity=0.45, glow=0.6),
        # main blue: dives from the top edge, bright where it bends
        ribbon(r, "t1", BLUE, [(140, -12), (260, 6), (320, 58), (380, 74), (440, 88), (560, 58), (720, 32)],
               140, 720, 0.38, n=56, spread=(60, 34, 6, 3, 10, 26, 40), width=1.25, glow=1.7),
        # orange: low and long, brightest just right of the blue bend
        ribbon(r, "t2", ORANGE, [(110, 96), (240, 92), (330, 76), (420, 76), (500, 76), (590, 54), (720, 48)],
               110, 720, 0.55, spread=(20, 12, 3, 4, 10, 18, 26)),
        # faint lower blue echo
        ribbon(r, "t4", BLUE, [(300, 118), (420, 110), (500, 96), (580, 92), (640, 90), (690, 80), (720, 76)],
               300, 720, 0.6, n=18, spread=(8, 6, 4, 6, 8, 10, 12), width=0.6, opacity=0.4, glow=0.5),
    ])


def bottom():
    """Bottom-left corner, 600 × 160 (anchored left / bottom)."""
    r = random.Random(11)
    return svg(600, 160, [
        ribbon(r, "b3", BLUE, [(-20, 20), (40, 10), (90, 40), (140, 80), (180, 110), (220, 140), (280, 170)],
               -20, 280, 0.3, n=24, spread=(14, 12, 10, 8, 8, 10, 12), width=0.7, opacity=0.45, glow=0.6),
        ribbon(r, "b1", BLUE, [(-20, 66), (50, 52), (110, 80), (160, 112), (200, 136), (250, 156), (320, 172)],
               -20, 320, 0.35, n=56, spread=(44, 26, 8, 3, 6, 12, 18), width=1.2, glow=1.6),
        ribbon(r, "b2", ORANGE, [(110, 172), (170, 150), (220, 118), (280, 104), (340, 92), (420, 100), (520, 118)],
               110, 520, 0.38, spread=(24, 12, 4, 6, 14, 22, 30)),
        ribbon(r, "b4", BLUE, [(200, 170), (300, 150), (360, 140), (430, 140), (480, 142), (540, 150), (620, 158)],
               200, 620, 0.4, n=16, spread=(6, 6, 6, 8, 8, 10, 10), width=0.6, opacity=0.35, glow=0.4),
    ])


def stage():
    """Training stage, 1000 × 360: a blue wave from the left through the
    vibration ring and an orange one from the right through the suction ring
    (visual target training.png). Stretched to the stage behind the rings."""
    r = random.Random(23)
    return svg(1000, 360, [
        ribbon(r, "s3", BLUE, [(-20, 120), (90, 60), (200, 230), (320, 170), (420, 110), (500, 230), (600, 200)],
               -20, 600, 0.5, n=22, spread=(18, 14, 10, 8, 10, 14, 18), width=0.7, opacity=0.4, glow=0.5),
        ribbon(r, "s1", BLUE, [(-20, 250), (110, 300), (190, 110), (320, 180), (430, 250), (520, 130), (640, 170)],
               -20, 640, 0.48, n=56, spread=(60, 40, 18, 4, 16, 30, 44), width=1.2, glow=1.5),
        ribbon(r, "s4", ORANGE, [(400, 210), (500, 110), (580, 250), (680, 180), (800, 120), (900, 250), (1020, 150)],
               400, 1020, 0.5, n=22, spread=(18, 14, 10, 8, 10, 14, 18), width=0.7, opacity=0.4, glow=0.5),
        ribbon(r, "s2", ORANGE, [(360, 180), (480, 240), (570, 120), (680, 180), (790, 250), (890, 80), (1020, 120)],
               360, 1020, 0.52, n=56, spread=(44, 30, 16, 4, 18, 40, 60), width=1.2, glow=1.5),
    ])


if __name__ == "__main__":
    OUT.mkdir(parents=True, exist_ok=True)
    (OUT / "bg-top.svg").write_text(top())
    (OUT / "bg-bottom.svg").write_text(bottom())
    (OUT / "stage-ribbons.svg").write_text(stage())
    print("wrote", OUT / "bg-top.svg", OUT / "bg-bottom.svg", OUT / "stage-ribbons.svg")
