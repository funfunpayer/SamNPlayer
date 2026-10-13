"""Generates the Flow background ribbons (visual target docs/visual-target/*.png):
silky blue / orange light waves in the top-right and bottom-left corners.

Run:  python3 scripts/art/flow_background.py
Writes cmd/gui-wails/frontend/src/assets/flow/bg-top.svg and bg-bottom.svg.
Deterministic (fixed seed), so re-running gives the same files.
"""
import pathlib
import random

OUT = pathlib.Path(__file__).resolve().parents[2] / "cmd/gui-wails/frontend/src/assets/flow"


def band(rng, gid, pts, n, spread, width, opacity, wobble):
    """n thin strands around one cubic path (pts: start, c1, c2, end, c3, c4,
    end2 — two joined cubics) plus a blurred glow under them."""
    def path(off, w):
        q = [(x + off[0] * k + rng.uniform(-w, w), y + off[1] * k + rng.uniform(-w, w))
             for k, (x, y) in zip((0.2, 0.9, 1.2, 1.0, 1.1, 0.8, 0.4), pts)]
        f = lambda p: f"{p[0]:.1f} {p[1]:.1f}"
        return f"M{f(q[0])} C{f(q[1])} {f(q[2])} {f(q[3])} C{f(q[4])} {f(q[5])} {f(q[6])}"
    out = [f'<path d="{path((0, 0), 0)}" stroke="url(#{gid})" stroke-width="{width * 9:.0f}" '
           f'opacity="{opacity * 0.55:.2f}" filter="url(#glow)"/>']
    for i in range(n):
        t = (i / max(1, n - 1)) - 0.5
        off = (t * spread * 0.35, t * spread)
        sw = rng.uniform(0.35, 1.0) * width / 4
        op = opacity * (1 - abs(t) * 1.3) * rng.uniform(0.5, 1.0)
        if op <= 0.03:
            continue
        out.append(f'<path d="{path(off, wobble)}" stroke="url(#{gid})" stroke-width="{sw:.2f}" opacity="{op:.2f}"/>')
    # hot core
    out.append(f'<path d="{path((0, 0), 0)}" stroke="url(#{gid}h)" stroke-width="{width / 3:.1f}" '
               f'opacity="{opacity:.2f}" filter="url(#soft)"/>')
    return out


def grad(gid, x1, x2, stops):
    s = "".join(f'<stop offset="{o}" stop-color="{c}" stop-opacity="{a}"/>' for o, c, a in stops)
    return f'<linearGradient id="{gid}" gradientUnits="userSpaceOnUse" x1="{x1}" y1="0" x2="{x2}" y2="0">{s}</linearGradient>'


def svg(w, h, defs, body):
    return (f'<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 {w} {h}" width="{w}" height="{h}">'
            f'<defs><filter id="glow" filterUnits="userSpaceOnUse" x="-100" y="-100" width="{w + 200}" height="{h + 200}"><feGaussianBlur stdDeviation="14"/></filter>'
            f'<filter id="soft" filterUnits="userSpaceOnUse" x="-100" y="-100" width="{w + 200}" height="{h + 200}"><feGaussianBlur stdDeviation="1.6"/></filter>'
            + "".join(defs) + '</defs><g fill="none" stroke-linecap="round">' + "".join(body) + "</g></svg>\n")


def blue(gid, x1, x2, fade_in=0.0, fade_out=1.0):
    return [grad(gid, x1, x2, [(fade_in, "#0a3cff", 0), ((fade_in + fade_out) / 2 - .1, "#1a8cff", 1),
                               ((fade_in + fade_out) / 2 + .1, "#36c8ff", 1), (fade_out, "#0a50ff", 0)]),
            grad(gid + "h", x1, x2, [(fade_in, "#bff0ff", 0), ((fade_in + fade_out) / 2, "#e8fbff", .9), (fade_out, "#7fd8ff", 0)])]


def orange(gid, x1, x2, fade_in=0.0, fade_out=1.0):
    return [grad(gid, x1, x2, [(fade_in, "#ff3d00", 0), ((fade_in + fade_out) / 2 - .1, "#ff6a10", 1),
                               ((fade_in + fade_out) / 2 + .1, "#ff9a2a", 1), (fade_out, "#ff4a00", 0)]),
            grad(gid + "h", x1, x2, [(fade_in, "#ffe0b0", 0), ((fade_in + fade_out) / 2, "#fff1d6", .9), (fade_out, "#ffb060", 0)])]


def top(rng):
    w, h = 900, 170
    defs = blue("b", 120, 900, 0, 1) + orange("o", 300, 900, 0, 1) + blue("b2", 400, 900, 0, 1)
    body = []
    body += band(rng, "b", [(120, -20), (330, -5), (430, 70), (560, 82), (680, 92), (800, 60), (930, 40)], 34, 46, 5, .95, 2.5)
    body += band(rng, "o", [(300, 140), (420, 120), (500, 96), (600, 92), (700, 88), (800, 55), (930, 30)], 30, 36, 5, .95, 2.5)
    body += band(rng, "b2", [(420, 165), (560, 150), (640, 120), (760, 118), (820, 116), (880, 100), (930, 95)], 14, 22, 3, .45, 3)
    return svg(w, h, defs, body)


def bottom(rng):
    w, h = 620, 260
    defs = blue("b", -20, 520, 0, 1) + orange("o", 40, 620, 0, 1) + blue("b2", -20, 300, 0, 1)
    body = []
    body += band(rng, "b", [(-30, 120), (60, 90), (130, 150), (220, 200), (300, 245), (380, 250), (460, 290)], 34, 50, 5, .95, 2.5)
    body += band(rng, "o", [(60, 290), (160, 250), (240, 230), (330, 236), (420, 242), (520, 250), (650, 270)], 30, 40, 5, .95, 2.5)
    body += band(rng, "b2", [(-30, 60), (40, 40), (100, 90), (160, 140), (200, 170), (240, 200), (290, 270)], 12, 26, 3, .4, 3)
    return svg(w, h, defs, body)


if __name__ == "__main__":
    OUT.mkdir(parents=True, exist_ok=True)
    (OUT / "bg-top.svg").write_text(top(random.Random(7)))
    (OUT / "bg-bottom.svg").write_text(bottom(random.Random(11)))
    print("wrote", OUT / "bg-top.svg", OUT / "bg-bottom.svg")
