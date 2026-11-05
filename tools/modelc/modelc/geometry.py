"""Primitives: every part of a model is one primitive evaluated from formulas.

A primitive produces a Mesh: vertices, triangles and local UVs in 0..1 (the compiler places
them into the part's region of the texture atlas).
"""
import math
import numpy as np
from .math3d import as_func


class Mesh:
    def __init__(self, verts, tris, uv):
        self.verts = np.asarray(verts, dtype=float).reshape(-1, 3)
        self.tris = np.asarray(tris, dtype=np.int64).reshape(-1, 3)
        self.uv = np.asarray(uv, dtype=float).reshape(-1, 2)

    def mirrored_x(self):
        v = self.verts.copy(); v[:, 0] *= -1
        return Mesh(v, self.tris[:, ::-1].copy(), self.uv.copy())

    def area(self):
        a, b, c = (self.verts[self.tris[:, k]] for k in range(3))
        return float(0.5 * np.linalg.norm(np.cross(b - a, c - a), axis=1).sum())


class Part:
    """Common part options:
    bone:    rigid binding to one bone, or
    bones:   a list of bones the part blends between by distance (the body, limbs, cuirass)
    color:   flat color (r, g, b in 0..1) — far-LOD color and placeholder texture
    team:    True = tinted with the player color
    mirror:  "x" = also build the mirrored copy (left ↔ right, bone names .L ↔ .R)
    uv:      name of a texture region shared with other parts (default: the part's own)
    """
    def __init__(self, id, bone=None, bones=None, color=(0.6, 0.6, 0.6), team=False, mirror=None, uv=None, uv_scale=1.0):
        self.id, self.bone, self.bones = id, bone, bones
        self.color, self.team, self.mirror = color, team, mirror
        self.uv_region = uv or id
        self.uv_scale = uv_scale

    def mesh(self):
        raise NotImplementedError


def _resample(points, n):
    """n points evenly spaced by arc length along a polyline, plus the tangents."""
    P = np.asarray(points, dtype=float)
    seg = np.linalg.norm(np.diff(P, axis=0), axis=1)
    s = np.concatenate([[0], np.cumsum(seg)])
    L = s[-1]
    out, tan = [], []
    for k in range(n):
        d = L * k / (n - 1)
        i = min(np.searchsorted(s, d, side="right") - 1, len(P) - 2)
        f = (d - s[i]) / max(seg[i], 1e-9)
        out.append(P[i] + (P[i + 1] - P[i]) * f)
        tan.append((P[i + 1] - P[i]) / max(seg[i], 1e-9))
    tan = np.array(tan)
    # smooth tangents at the joints of the polyline
    for k in range(1, n - 1):
        t = out[k + 1] - out[k - 1]
        tan[k] = t / max(np.linalg.norm(t), 1e-9)
    return np.array(out), tan, L


def _frame(t):
    """Ring axes for a path tangent t: ay = the front (-Y) made perpendicular to t, ax = the
    character's left (+X) made perpendicular to both — the same orientation whichever way the path runs."""
    F, U, X = np.array([0.0, -1.0, 0.0]), np.array([0.0, 0.0, 1.0]), np.array([1.0, 0.0, 0.0])
    ay = F - t * (F @ t)
    if np.linalg.norm(ay) < 0.3:
        ay = U - t * (U @ t)
    ay /= np.linalg.norm(ay)
    ax = X - t * (X @ t) - ay * (X @ ay)
    if np.linalg.norm(ax) < 0.3:
        ax = np.cross(t, ay)
    return ax / np.linalg.norm(ax), ay


class Loft(Part):
    """Rings along a path. r = radius(t) or rx/ry (t = 0..1 along the path, numbers, Python
    functions or SymPy expressions). section exponent n: 2 = ellipse, > 2 = squarer, 4 = rounded box.
    offset(t) -> (dx, dy, dz) bends the shape off the path (e.g. a chest pushed forward)."""
    def __init__(self, id, path, r=None, rx=None, ry=None, n=2.0, segments=10, rings=8,
                 caps=(True, True), offset=None, twist=0.0, **kw):
        super().__init__(id, **kw)
        self.path = path
        self.rx = as_func(rx if rx is not None else r)
        self.ry = as_func(ry if ry is not None else (r if r is not None else rx))
        self.n, self.segments, self.rings = n, segments, rings
        self.caps, self.offset, self.twist = caps, offset, twist

    def mesh(self):
        P, T, L = _resample(self.path, self.rings)
        S = self.segments
        verts, uv, tris, centres = [], [], [], []
        e = 2.0 / self.n
        # UV, 2 views: u = 0..0.5 is the front (from the character's right side to its left, as seen
        # in a front view), u = 0.5..1 the back (as seen from behind). 4 views: FRONT | LEFT | BACK | RIGHT,
        # each quarter as seen from that side. v = 0 is the top of the part.
        upright = self.path[-1][2] > self.path[0][2]
        # 4 views: FRONT | LEFT | BACK | RIGHT quarters (the seam on the front-right diagonal)
        off = 0.125 if getattr(self, "wrap_views", 2) == 4 else 0.0
        for i in range(self.rings):
            t = i / (self.rings - 1)
            vrow = 1 - t if upright else t
            ax, ay = _frame(T[i])
            rx, ry = self.rx(t), self.ry(t)
            c = P[i] + (np.array(self.offset(t)) if self.offset else 0)
            for k in range(S + 1):
                a = math.pi - 2 * math.pi * (k / S + off) + math.radians(self.twist * t)
                ca, sa = math.cos(a), math.sin(a)
                x = math.copysign(abs(ca) ** e, ca) * rx
                y = math.copysign(abs(sa) ** e, sa) * ry
                verts.append(c + ax * x + ay * y)
                uv.append((k / S, vrow))
                centres.append(c)
        row = S + 1
        for i in range(self.rings - 1):
            for k in range(S):
                a, b = i * row + k, i * row + k + 1
                c, d = a + row, b + row
                tris += [(a, c, b), (b, c, d)]
        for end, cap in ((0, self.caps[0]), (self.rings - 1, self.caps[1])):
            if not cap:
                continue
            centre = len(verts)
            t = end / (self.rings - 1)
            verts.append(P[end] + (np.array(self.offset(t)) if self.offset else 0))
            uv.append((0.25 - off, 1 - t if upright else t))
            centres.append(verts[-1] - T[end] * (1 if end else -1))  # outward = along the path at the ends
            for k in range(S):
                a, b = end * row + k, end * row + k + 1
                tris.append((centre, b, a) if end == 0 else (centre, a, b))
        m = Mesh(verts, tris, uv)
        _fix_winding(m, np.array(centres))
        return m


class Extrude(Part):
    """A flat 2D outline [(u, v), ...] extruded by depth. origin + u_axis, v_axis place it in space
    (default: in the XZ plane facing front, -Y). UVs: planar, both faces share the region."""
    def __init__(self, id, outline, depth, origin=(0, 0, 0), u_axis=(1, 0, 0), v_axis=(0, 0, 1), wrap=None, **kw):
        """wrap=(rx, ry, flare[, angle]): bend the outline around a vertical ellipse centred at origin
        (u = meters along the front of the ellipse, v = height; flare widens it downwards) —
        a skirt or a belt plate that hugs the body instead of floating in front of it. angle (degrees)
        turns it around the body: 0 = centred at the front, 180 = at the back."""
        super().__init__(id, **kw)
        self.wrap = wrap
        self.outline = [tuple(p) for p in outline]
        self.depth, self.origin = depth, np.array(origin, dtype=float)
        self.u_axis = np.array(u_axis, dtype=float); self.v_axis = np.array(v_axis, dtype=float)

    def mesh(self):
        O = self.outline
        if _signed_area(O) < 0:
            O = O[::-1]
        nrm = np.cross(self.u_axis, self.v_axis); nrm /= np.linalg.norm(nrm)
        us = [p[0] for p in O]; vs = [p[1] for p in O]
        u0, u1, v0, v1 = min(us), max(us), min(vs), max(vs)
        span = max(u1 - u0, v1 - v0)
        cu, cv = (1 - (u1 - u0) / span) / 2, (1 - (v1 - v0) / span) / 2   # centred in the region
        def puv(p):
            return (cu + (p[0] - u0) / span, 1 - cv - (p[1] - v0) / span)
        def pos(p, side):
            if self.wrap:
                rx, ry, flare = self.wrap[:3]
                a0 = math.radians(self.wrap[3]) if len(self.wrap) > 3 else 0.0   # 0 = front, 180 = back
                a = a0 + p[0] / ((rx + ry) / 2)
                s = 1 + flare * max(0.0, -p[1])
                off = side * self.depth / 2
                return self.origin + np.array([(rx * s + off) * math.sin(a), -(ry * s + off) * math.cos(a), p[1]])
            return self.origin + self.u_axis * p[0] + self.v_axis * p[1] + nrm * side * self.depth / 2
        verts, uv, tris, centres = [], [], [], []
        n = len(O)
        c2 = (sum(p[0] for p in O) / n, sum(p[1] for p in O) / n)
        for side in (1, -1):
            base = len(verts)
            for p in O:
                verts.append(pos(p, side)); uv.append(puv(p)); centres.append(pos(p, 0))
            for a, b, c in _ear_clip(O):
                tris.append((base + a, base + b, base + c) if side == 1 else (base + a, base + c, base + b))
        for k in range(n):
            p, q = O[k], O[(k + 1) % n]
            base = len(verts)
            verts += [pos(p, 1), pos(q, 1), pos(q, -1), pos(p, -1)]
            uv += [puv(p), puv(q), puv(q), puv(p)]
            centres += [pos(c2, 1), pos(c2, 1), pos(c2, -1), pos(c2, -1)]
            tris += [(base, base + 2, base + 1), (base, base + 3, base + 2)]
        m = Mesh(verts, tris, uv)
        _fix_winding(m, np.array(centres))
        return m


def _signed_area(O):
    return 0.5 * sum(O[i][0] * O[(i + 1) % len(O)][1] - O[(i + 1) % len(O)][0] * O[i][1] for i in range(len(O)))


def _ear_clip(O):
    idx = list(range(len(O))); out = []
    def cross(o, a, b):
        return (a[0] - o[0]) * (b[1] - o[1]) - (a[1] - o[1]) * (b[0] - o[0])
    def inside(p, a, b, c):
        return cross(a, b, p) >= 0 and cross(b, c, p) >= 0 and cross(c, a, p) >= 0
    guard = 0
    while len(idx) > 3 and guard < 10000:
        guard += 1
        for j in range(len(idx)):
            i0, i1, i2 = idx[j - 1], idx[j], idx[(j + 1) % len(idx)]
            a, b, c = O[i0], O[i1], O[i2]
            if cross(a, b, c) <= 1e-12:
                continue
            if any(inside(O[k], a, b, c) for k in idx if k not in (i0, i1, i2)):
                continue
            out.append((i0, i1, i2)); idx.pop(j); break
        else:
            break
    if len(idx) == 3:
        out.append(tuple(idx))
    return out


def _fix_winding(m, centres=None):
    """Make every triangle face outward: away from its ring centre (lofts) or the part centre."""
    a, b, d = (m.verts[m.tris[:, k]] for k in range(3))
    n = np.cross(b - a, d - a)
    if centres is None:
        out = (a + b + d) / 3 - m.verts.mean(0)
    else:
        out = (a + b + d) / 3 - centres[m.tris].mean(1)
    flip = (n * out).sum(1) < 0
    m.tris[flip] = m.tris[flip][:, ::-1]
