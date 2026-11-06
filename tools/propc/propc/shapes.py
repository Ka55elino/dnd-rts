"""Plain shapes for static models (buildings, machines). Game axes: x right, y up, front +z; meters.

Every shape returns a list of Parts (so shapes, groups and transforms combine with `+`). Common options:
    color   (r, g, b) 0..1
    kind    "matte" (default) | "metal" | "glow" (emissive: runes, lights, magic) | "team" (player color)
    detail  0 = always, 1 = dropped in the far LOD, 2 = only in the near LOD (small decorations)
    node    name of a moving part (a separate node: gate leaves, wheels, arms); pivot = its axis point
"""
import math
import numpy as np


class Part:
    def __init__(self, verts, tris, color=(0.6, 0.6, 0.6), kind="matte", detail=0, node=None, pivot=None):
        self.verts = np.asarray(verts, dtype=float).reshape(-1, 3)
        self.tris = np.asarray(tris, dtype=np.int64).reshape(-1, 3)
        self.color = tuple(float(c) for c in color)
        self.kind = kind
        self.detail = detail
        self.node = node
        self.pivot = tuple(pivot) if pivot is not None else None

    def copy(self, verts=None, **kw):
        p = Part(self.verts if verts is None else verts, self.tris, self.color, self.kind, self.detail, self.node, self.pivot)
        for k, v in kw.items():
            setattr(p, k, v)
        return p

    @property
    def material(self):
        if self.kind == "team":
            return "team"
        r, g, b = (min(255, max(0, int(round(c * 255)))) for c in self.color)
        return f"{self.kind}_{r:02x}{g:02x}{b:02x}"


def _part(verts, tris, kw):
    """Every primitive is convex: triangles are turned to face outwards here (one-sided in the game)."""
    p = Part(verts, tris, kw.get("color", (0.6, 0.6, 0.6)), kw.get("kind", "matte"), kw.get("detail", 0),
             kw.get("node"), kw.get("pivot"))
    _fix_outward(p)
    return [p]


def _ry(v, deg, about=(0, 0, 0)):
    if not deg:
        return v
    a = math.radians(deg)
    c, s = math.cos(a), math.sin(a)
    v = np.asarray(v, dtype=float) - about
    out = v.copy()
    out[:, 0] = c * v[:, 0] + s * v[:, 2]
    out[:, 2] = -s * v[:, 0] + c * v[:, 2]
    return out + about


# ------------------------------------------------------------------------------------------- shapes
def box(c, size, ry=0, **kw):
    """A box standing on c = (x, y_bottom, z), size (width x, height y, depth z), turned ry° about y."""
    x, y, z = c
    w, h, d = size
    v = np.array([[x + sx * w / 2, y + sy * h, z + sz * d / 2] for sy in (0, 1) for sz in (-1, 1) for sx in (-1, 1)])
    # 0..3 bottom (z-,x-)(z-,x+)(z+,x-)(z+,x+); 4..7 top
    t = [(0, 1, 3), (0, 3, 2),  # bottom (faces down)
         (4, 6, 7), (4, 7, 5),  # top
         (2, 3, 7), (2, 7, 6),  # front +z
         (1, 0, 4), (1, 4, 5),  # back -z
         (0, 2, 6), (0, 6, 4),  # left -x
         (3, 1, 5), (3, 5, 7)]  # right +x
    return _part(_ry(v, ry, (x, 0, z)), t, kw)


def _ring(cx, y, cz, r, seg, rot=0.0):
    return [(cx + r * math.cos(rot + 2 * math.pi * k / seg), y, cz + r * math.sin(rot + 2 * math.pi * k / seg)) for k in range(seg)]


def cyl(c, r, h, r_top=None, seg=8, rot=0.0, caps=True, **kw):
    """A cylinder / frustum / cone standing on c (r_top = 0: a cone). rot: turn of the sides (radians)."""
    x, y, z = c
    rt = r if r_top is None else r_top
    v = _ring(x, y, z, r, seg, rot)
    t = []
    if rt > 0:
        v += _ring(x, y + h, z, rt, seg, rot)
        for k in range(seg):
            a, b = k, (k + 1) % seg
            t += [(a, seg + b, b), (a, seg + a, seg + b)]
        if caps:
            v += [(x, y, z), (x, y + h, z)]
            cb, ct = 2 * seg, 2 * seg + 1
            for k in range(seg):
                a, b = k, (k + 1) % seg
                t += [(cb, a, b), (ct, seg + b, seg + a)]
    else:
        v += [(x, y + h, z)]
        tip = seg
        for k in range(seg):
            a, b = k, (k + 1) % seg
            t.append((a, tip, b))
        if caps:
            v += [(x, y, z)]
            cb = seg + 1
            for k in range(seg):
                a, b = k, (k + 1) % seg
                t.append((cb, a, b))
    return _part(v, t, kw)


def cone(c, r, h, seg=8, **kw):
    return cyl(c, r, h, r_top=0, seg=seg, **kw)


def rod(a, b, r, r_b=None, seg=6, caps=True, **kw):
    """A cylinder (or a cone with r_b = 0) from point a to point b: posts, beams, spikes, axles."""
    a, b = np.array(a, float), np.array(b, float)
    d = b - a
    L = float(np.linalg.norm(d))
    if L < 1e-9:
        return []
    p = cyl((0, 0, 0), r, L, r_top=r if r_b is None else r_b, seg=seg, caps=caps, **kw)[0]
    # local y → d
    yv = d / L
    ref = np.array([1.0, 0, 0]) if abs(yv[0]) < 0.9 else np.array([0, 0, 1.0])
    xv = np.cross(yv, ref); xv /= np.linalg.norm(xv)
    zv = np.cross(xv, yv)
    m = np.stack([xv, yv, zv])  # rows
    p.verts = p.verts @ m + a
    return [p]


def spike(a, b, r, **kw):
    return rod(a, b, r, r_b=0, seg=kw.pop("seg", 5), **kw)


def sphere(c, r, seg=10, rings=6, ry_scale=1.0, half=False, **kw):
    """A low-poly sphere centred on c (half=True: a dome standing on c)."""
    x, y, z = c
    v, t = [], []
    lo = 0.0 if half else -math.pi / 2
    for i in range(rings + 1):
        phi = lo + (math.pi / 2 - lo) * i / rings
        for k in range(seg):
            th = 2 * math.pi * k / seg
            v.append((x + r * math.cos(phi) * math.cos(th), y + r * ry_scale * math.sin(phi), z + r * math.cos(phi) * math.sin(th)))
    for i in range(rings):
        for k in range(seg):
            a, b = i * seg + k, i * seg + (k + 1) % seg
            t += [(a, b + seg, b), (a, a + seg, b + seg)]
    if half:
        v.append((x, y, z))
        cb = len(v) - 1
        for k in range(seg):
            t.append((cb, k, (k + 1) % seg))
    return _part(v, t, kw)


def dome(c, r, h=None, seg=10, rings=4, **kw):
    return sphere(c, r, seg=seg, rings=rings, ry_scale=(h or r) / r, half=True, **kw)


def gable(c, w, d, h, ry=0, **kw):
    """A gable roof on c = (x, y_eaves, z): w across x, d along z (the ridge runs along z)."""
    x, y, z = c
    v = [(x - w / 2, y, z - d / 2), (x + w / 2, y, z - d / 2), (x - w / 2, y, z + d / 2), (x + w / 2, y, z + d / 2),
         (x, y + h, z - d / 2), (x, y + h, z + d / 2)]
    t = [(0, 2, 5), (0, 5, 4), (3, 1, 4), (3, 4, 5), (2, 3, 5), (1, 0, 4), (0, 1, 3), (0, 3, 2)]
    return _part(_ry(v, ry, (x, 0, z)), t, kw)


def hip(c, w, d, h, ridge=0.0, ry=0, **kw):
    """A hipped / pyramid roof on c = (x, y_eaves, z); ridge: length of the top edge along z (0 = pyramid)."""
    x, y, z = c
    v = [(x - w / 2, y, z - d / 2), (x + w / 2, y, z - d / 2), (x - w / 2, y, z + d / 2), (x + w / 2, y, z + d / 2),
         (x, y + h, z - ridge / 2), (x, y + h, z + ridge / 2)]
    t = [(0, 2, 5), (0, 5, 4), (3, 1, 4), (3, 4, 5), (2, 3, 5), (1, 0, 4), (0, 1, 3), (0, 3, 2)]
    return _part(_ry(v, ry, (x, 0, z)), t, kw)


def prism(c, points, h, **kw):
    """A vertical prism: a convex outline (x, z) points (counter-clockwise seen from above), from c's y up h."""
    x0, y, z0 = c
    n = len(points)
    v = [(x0 + px, y, z0 + pz) for px, pz in points] + [(x0 + px, y + h, z0 + pz) for px, pz in points]
    t = []
    for k in range(n):
        a, b = k, (k + 1) % n
        t += [(a, b, n + b), (a, n + b, n + a)]
    for k in range(1, n - 1):
        t += [(0, k + 1, k), (n, n + k, n + k + 1)]
    return _part(v, t, kw)


def _fix_outward(p):
    """Flips triangles that face the inside of a convex shape."""
    c = p.verts.mean(0)
    for i, (a, b, cc) in enumerate(p.tris):
        A, B, C = p.verts[a], p.verts[b], p.verts[cc]
        n = np.cross(B - A, C - A)
        if np.dot(n, (A + B + C) / 3 - c) < 0:
            p.tris[i] = (a, cc, b)


# ------------------------------------------------------------------------------------- composites
def battlements(c, w, d, size=0.6, gap=0.6, h=0.8, sides="nsew", **kw):
    """Merlons along the edges of a w × d rectangle centred on c = (x, y_top_of_wall, z)."""
    x, y, z = c
    out = []
    step = size + gap
    for side in sides:
        length = w if side in "ns" else d
        n = max(2, int((length + gap) // step))
        span = n * step - gap
        for k in range(n):
            o = -span / 2 + size / 2 + k * step
            if side == "n":
                p = (x + o, y, z + d / 2 - size / 2)
            elif side == "s":
                p = (x + o, y, z - d / 2 + size / 2)
            elif side == "e":
                p = (x + w / 2 - size / 2, y, z + o)
            else:
                p = (x - w / 2 + size / 2, y, z + o)
            out += box(p, (size, h, size), **{"detail": 1, **kw})
    return out


def crown(c, r, n=8, size=0.6, h=0.8, **kw):
    """Merlons around a round tower top: c = (x, y_top, z)."""
    x, y, z = c
    out = []
    for k in range(n):
        a = 2 * math.pi * k / n
        out += box((x + (r - size / 2) * math.cos(a), y, z + (r - size / 2) * math.sin(a)), (size, h, size),
                   ry=-math.degrees(a), **{"detail": 1, **kw})
    return out


def ring_of(n, r, fn, c=(0, 0, 0), start=0.0):
    """fn(x, z, angle_deg) for n points on a circle around c: repeats a detail around a building."""
    out = []
    for k in range(n):
        a = start + 2 * math.pi * k / n
        out += fn(c[0] + r * math.cos(a), c[2] + r * math.sin(a), math.degrees(a))
    return out


def banner(c, w=0.8, h=1.6, ry=0, pole=True, **kw):
    """A team-colored hanging banner on c = (x, y_top, z), facing +z (turned ry°)."""
    x, y, z = c
    out = box((x, y - h, z), (w, h, 0.06), ry=ry, kind="team", detail=1)
    if pole:
        out += rod(_ry([(x - w / 2 - 0.1, y + 0.05, z)], ry, (x, 0, z))[0], _ry([(x + w / 2 + 0.1, y + 0.05, z)], ry, (x, 0, z))[0],
                   0.05, seg=4, color=kw.get("pole", (0.3, 0.22, 0.14)), detail=2)
    return out


def flag(c, h=3.0, w=1.2, ry=0, **kw):
    """A pole with a team flag on top, standing on c."""
    x, y, z = c
    return (rod((x, y, z), (x, y + h, z), 0.06, seg=4, color=(0.35, 0.27, 0.18), detail=1) +
            box(_ry([(x + w / 2, y + h - 0.75, z)], ry, (x, 0, z))[0], (w, 0.7, 0.05), ry=ry, kind="team", detail=1))


# --------------------------------------------------------------------------------------- transforms
def move(parts, dx=0.0, dy=0.0, dz=0.0):
    return [p.copy(p.verts + (dx, dy, dz), pivot=None if p.pivot is None else tuple(np.add(p.pivot, (dx, dy, dz)))) for p in parts]


def turn(parts, deg, about=(0, 0, 0)):
    """Turns parts about the vertical axis through `about`."""
    out = []
    for p in parts:
        piv = None if p.pivot is None else tuple(_ry([p.pivot], deg, about)[0])
        out.append(p.copy(_ry(p.verts, deg, about), pivot=piv))
    return out


def mirror_x(parts):
    out = []
    for p in parts:
        v = p.verts.copy(); v[:, 0] *= -1
        q = p.copy(v)
        q.tris = p.tris[:, ::-1].copy()
        if p.pivot is not None:
            q.pivot = (-p.pivot[0], p.pivot[1], p.pivot[2])
        out.append(q)
    return out


def both_x(parts):
    """The parts and their mirror image across x = 0."""
    return parts + mirror_x(parts)


def scaled(parts, s):
    return [p.copy(p.verts * s, pivot=None if p.pivot is None else tuple(np.multiply(p.pivot, s))) for p in parts]


def tagged(parts, **kw):
    """Sets options (node, pivot, detail, color, kind) on every part."""
    return [p.copy(**kw) for p in parts]
