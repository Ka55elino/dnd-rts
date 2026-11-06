"""Shared building blocks for the faction buildings (art/props/*.py). Front of a building is +z."""
from propc import *  # noqa: F401,F403
from propc.shapes import box, cyl, cone, rod, spike, sphere, dome, gable, hip, prism, battlements, crown, ring_of, banner, flag, move, turn, both_x, tagged  # noqa: F401

# Common colours
DOOR = (0.24, 0.16, 0.09)
DARK = (0.09, 0.08, 0.08)       # openings, shadows in doorways
WOOD = (0.47, 0.34, 0.20)
WOOD_D = (0.33, 0.23, 0.14)
STRAW = (0.80, 0.68, 0.38)
DIRT = (0.42, 0.34, 0.24)
SAND = (0.72, 0.64, 0.46)
LEAF = (0.27, 0.48, 0.22)
LEAF_L = (0.38, 0.60, 0.28)
IRON = (0.32, 0.33, 0.36)
BONE = (0.86, 0.83, 0.74)
GOLD = (0.85, 0.66, 0.24)
FIRE = (1.0, 0.55, 0.15)


def hall(w, d, h, wall, roof_col, roof="gable", roof_h=None, at=(0, 0, 0), over=0.35, ridge=None, ry=0):
    """A rectangular building: walls w (x) × h × d (z) and a roof (gable / hip: the ridge along the
    longer side; flat; none)."""
    x, y, z = at
    out = box((x, y, z), (w, h, d), color=wall)
    rh = roof_h if roof_h is not None else min(w, d) * 0.45
    W, D = w + 2 * over, d + 2 * over
    along_x = w > d
    if roof == "gable":
        out += gable((x, y + h, z), D, W, rh, ry=90, color=roof_col) if along_x else gable((x, y + h, z), W, D, rh, color=roof_col)
    elif roof == "hip":
        r = abs(w - d) if ridge is None else ridge
        out += hip((x, y + h, z), D, W, rh, ridge=r, ry=90, color=roof_col) if along_x else hip((x, y + h, z), W, D, rh, ridge=r, color=roof_col)
    elif roof == "flat":
        out += box((x, y + h, z), (w + 2 * over, 0.3, d + 2 * over), color=roof_col)
    return turn(out, ry, (x, 0, z)) if ry else out


def door(x, z, w=1.6, h=2.4, y=0.0, color=DOOR, ry=0, arch=True):
    """A door on a facade at (x, z) facing +z (turned ry°): a slab standing a little proud of the wall."""
    out = box((x, y, z), (w, h, 0.25), color=color)
    if arch:  # a rounded top: a disc facing +z
        out += rod((x, y + h, z - 0.125), (x, y + h, z + 0.125), w / 2, seg=10, color=color)
    return turn(out, ry, (x, 0, z)) if ry else out


def windows(xs, y, z, w=0.6, h=1.0, color=DARK, kind="matte", ry=0, about=(0, 0, 0), detail=1):
    """A row of windows on a facade facing +z at depth z."""
    out = []
    for x in xs:
        out += box((x, y, z), (w, h, 0.12), color=color, kind=kind, detail=detail)
    return turn(out, ry, about) if ry else out


def round_tower(x, z, r, h, wall, roof_col=None, roof_h=None, seg=10, merlons=None, y=0.0):
    """A round tower: a cylinder, then a cone roof (roof_col) or a crown of merlons (merlons = colour)."""
    out = cyl((x, y, z), r, h, seg=seg, color=wall)
    if merlons:
        out += cyl((x, y + h, z), r + 0.25, 0.35, seg=seg, color=merlons)
        out += crown((x, y + h + 0.35, z), r + 0.25, n=max(6, seg - 2), size=0.5, h=0.6, color=merlons)
    if roof_col:
        out += cone((x, y + h + (0.35 if merlons else 0), z), r + 0.35, roof_h or r * 2.2, seg=seg, color=roof_col)
    return out


def fence(points, h=1.1, color=WOOD_D, post=0.09, detail=2):
    """Posts and two rails along a polyline of (x, z) points."""
    out = []
    for (x0, z0), (x1, z1) in zip(points, points[1:]):
        out += rod((x0, 0, z0), (x0, h, z0), post, seg=4, color=color, detail=detail)
        for hh in (h * 0.45, h * 0.9):
            out += rod((x0, hh, z0), (x1, hh, z1), post * 0.6, seg=4, color=color, detail=detail)
    x, z = points[-1]
    out += rod((x, 0, z), (x, h, z), post, seg=4, color=color, detail=detail)
    return out


def palisade(points, h=3.0, r=0.22, color=WOOD, step=0.42, detail=0):
    """Pointed logs side by side along a polyline (orc and goblin walls)."""
    import math
    out = []
    for (x0, z0), (x1, z1) in zip(points, points[1:]):
        L = math.hypot(x1 - x0, z1 - z0)
        n = max(1, int(L / step))
        for k in range(n):
            t = k / n
            x, z = x0 + (x1 - x0) * t, z0 + (z1 - z0) * t
            hh = h * (0.9 + 0.2 * ((k * 7919) % 5) / 5)
            out += cyl((x, 0, z), r, hh, seg=5, color=color, detail=detail)
            out += cone((x, hh, z), r, 0.5, seg=5, color=color, detail=max(detail, 1))
    return out


def circle(n, r, start=0.0):
    import math
    return [(r * math.cos(start + 2 * math.pi * k / n), r * math.sin(start + 2 * math.pi * k / n)) for k in range(n + 1)]


def tree(x, z, h=4.0, r=1.4, trunk=WOOD_D, leaf=LEAF, y=0.0, detail=0):
    return (cyl((x, y, z), 0.18 * h / 4, h * 0.55, seg=5, color=trunk, detail=detail) +
            sphere((x, y + h * 0.65, z), r, seg=7, rings=4, color=leaf, detail=detail))


def target(x, z, ry=0, r=0.7, h=1.2):
    """An archery target: a straw disc with a red centre on a stand, facing +z (turned ry°)."""
    out = rod((x, h, z), (x, h, z + 0.25), r, seg=10, color=STRAW, detail=1)
    out += rod((x, h, z + 0.25), (x, h, z + 0.28), r * 0.35, seg=8, color=(0.7, 0.15, 0.12), detail=2)
    out += rod((x - 0.3, 0, z - 0.2), (x - 0.15, h, z), 0.06, seg=4, color=WOOD_D, detail=2)
    out += rod((x + 0.3, 0, z - 0.2), (x + 0.15, h, z), 0.06, seg=4, color=WOOD_D, detail=2)
    return turn(out, ry, (x, 0, z)) if ry else out


def logs(x, z, n=3, length=2.4, r=0.22, ry=0, color=WOOD):
    """A small pile of logs lying along x."""
    out = []
    k = 0
    for row in range(n):
        for i in range(n - row):
            ox = (i - (n - row - 1) / 2) * r * 2.05
            out += rod((x - length / 2, r + row * r * 1.75, z + ox), (x + length / 2, r + row * r * 1.75, z + ox), r, seg=6, color=color, detail=1)
            k += 1
    return turn(out, ry, (x, 0, z)) if ry else out


def anvil(x, z, ry=0):
    out = box((x, 0, z), (0.45, 0.55, 0.45), color=WOOD_D, detail=1)
    out += box((x, 0.55, z), (0.9, 0.3, 0.38), color=IRON, kind="metal", detail=1)
    return turn(out, ry, (x, 0, z)) if ry else out


def glow_pool(x, z, r, color, rim, y=0.0, rim_h=0.6, seg=12):
    return cyl((x, y, z), r + 0.35, rim_h, seg=seg, color=rim) + cyl((x, y + rim_h - 0.05, z), r, 0.08, seg=seg, color=color, kind="glow")


def rocks(x, z, r=1.5, n=4, color=(0.42, 0.40, 0.38), y=0.0, detail=0):
    """A clump of rounded rocks (a mound): n spheres around (x, z)."""
    import math
    out = sphere((x, y, z), r, seg=7, rings=4, ry_scale=0.7, color=color, detail=detail)
    for k in range(n - 1):
        a = 2.4 * k
        rr = r * (0.55 + 0.1 * (k % 3))
        out += sphere((x + math.cos(a) * r * 0.8, y, z + math.sin(a) * r * 0.8), rr, seg=6, rings=3, ry_scale=0.75, color=color, detail=max(detail, 1))
    return out


def wall_row(length=6.1, depth=2.44, h=5.6, stone=(0.6, 0.58, 0.54), cap=(0.66, 0.64, 0.6), merlon=None):
    """One row of a base wall (the game puts two back to back, the walkway between): the outer face at +z."""
    out = box((0, 0, 0), (length, h, depth), color=stone)
    out += box((0, 0, depth / 2 - 0.2), (length + 0.02, 0.9, 0.5), color=cap)  # plinth
    out += box((0, h - 0.25, 0), (length + 0.02, 0.25, depth + 0.1), color=cap)
    m = merlon or cap
    n = 4
    step = length / n
    for k in range(n):
        out += box((-length / 2 + step * (k + 0.5), h, depth / 2 - 0.3), (step * 0.55, 1.0, 0.6), color=m)
    return out
