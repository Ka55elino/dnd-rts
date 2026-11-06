"""A model: a skeleton + parts + clips. build() evaluates everything into one skinned mesh
with a packed UV atlas (one region per part / shared region name)."""
import numpy as np


def swap_side(name):
    if name is None:
        return None
    if name.endswith(".L"):
        return name[:-2] + ".R"
    if name.endswith(".R"):
        return name[:-2] + ".L"
    return name


class Model:
    def __init__(self, id, skeleton, parts, clips, skins=("default",), texture=None, texture_size=512,
                 budget_tris=3000, foot_bone="foot.L", team_tint=True, wrap_views=2, fps=15, scale=1.0):
        """team_tint=False: no part is tinted with the player color (the team is shown by the ring
        under the unit); parts marked team=True keep their painted colors."""
        self.id, self.skeleton, self.parts, self.clips = id, skeleton, parts, clips
        self.team_tint = team_tint
        # uniform size factor applied to the baked output (a big elemental, a large hero): shapes and
        # clips are authored at human size on the shared skeletons
        self.scale = scale
        # animation samples per second in the VAT (the shader blends linearly between them): 15 is enough
        # for cycles; 30 keeps fast eased moves (brace, attacks) smooth at twice the VAT size
        self.fps = fps
        # wrapped parts (Loft) in the texture: 2 = FRONT | BACK halves, 4 = FRONT | LEFT | BACK | RIGHT quarters
        self.wrap_views = wrap_views
        self.skins, self.texture, self.texture_size = list(skins), texture, texture_size
        self.budget_tris, self.foot_bone = budget_tris, foot_bone


class Built:
    """Evaluated geometry: positions (rest), triangles, atlas UVs, 4 bone indices/weights, colors."""
    pass


def build(model):
    sk = model.skeleton
    pieces = []  # (part, mesh, side_swapped)
    for p in model.parts:
        p.wrap_views = model.wrap_views
        m = p.mesh()
        pieces.append((p, m, False))
        if p.mirror == "x":
            pieces.append((p, m.mirrored_x(), True))

    regions = _pack_atlas(pieces)
    locked = _locked_atlas(model)
    if locked is not None:
        missing = [r for r in regions if r not in locked]
        if missing:
            print(f"WARNING {model.id}: regions {missing} are not in the locked atlas — they get no texture space; "
                  f"re-lock with `modelc atlas {model.id} --lock` (and repaint) to include them")
            for r in missing:
                locked[r] = (0.0, 0.0, 0.004, 0.004)
        regions = {r: tuple(locked[r]) for r in regions}
    V, T, UV, BI, BW, COL, TEAM, PART, PIECE = [], [], [], [], [], [], [], [], []
    base = 0
    for pi, (p, m, swapped) in enumerate(pieces):
        r = regions[p.uv_region]
        uv = np.column_stack([r[0] + m.uv[:, 0] * r[2], r[1] + m.uv[:, 1] * r[3]])
        bi, bw = _weights(sk, p, m, swapped)
        V.append(m.verts); T.append(m.tris + base); UV.append(uv); BI.append(bi); BW.append(bw)
        COL += [p.color] * len(m.verts); TEAM += [p.team and model.team_tint] * len(m.verts); PART += [p.id] * len(m.verts)
        PIECE += [pi] * len(m.verts)
        base += len(m.verts)
    b = Built()
    b.verts = np.concatenate(V); b.tris = np.concatenate(T); b.uv = np.concatenate(UV)
    b.bone_idx = np.concatenate(BI); b.bone_w = np.concatenate(BW)
    b.colors = np.array(COL, dtype=float); b.team = np.array(TEAM, dtype=bool); b.part = PART
    b.piece = np.array(PIECE, dtype=np.int64)   # index into b.pieces per vertex
    b.regions = regions
    b.wrap_views = model.wrap_views
    b.pieces = pieces
    if len(b.tris) > model.budget_tris:
        print(f"WARNING {model.id}: {len(b.tris)} triangles, budget {model.budget_tris}")
    return b


def atlas_file(model):
    import os
    if not model.texture:
        return None
    repo = os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__)))))
    return os.path.join(repo, "art", "textures", model.texture, "atlas.json")


def _locked_atlas(model):
    """A painted texture fixes the layout: art/textures/<texture>/atlas.json keeps every region's
    rectangle, so changing a part's shape or size does not move its pixels."""
    import json, os
    f = atlas_file(model)
    if f and os.path.exists(f):
        return {k: v for k, v in json.load(open(f))["regions"].items()}
    return None


def _weights(sk, part, mesh, swapped):
    n = len(mesh.verts)
    bi = np.zeros((n, 4), dtype=np.int64); bw = np.zeros((n, 4))
    if part.bone:
        name = swap_side(part.bone) if swapped else part.bone
        bi[:, 0] = sk.index[name]; bw[:, 0] = 1.0
        return bi, bw
    names = [swap_side(x) if swapped else x for x in part.bones]
    segs = [(sk[x].head, sk[x].tail) for x in names]
    D = np.zeros((n, len(names)))
    for j, (a, b) in enumerate(segs):
        ab = b - a
        t = np.clip(((mesh.verts - a) @ ab) / max(ab @ ab, 1e-9), 0, 1)
        D[:, j] = np.linalg.norm(mesh.verts - (a + np.outer(t, ab)), axis=1)
    W = 1.0 / (D + 0.01) ** 6        # sharp falloff: one bone in the middle of a segment, a blend at joints
    order = np.argsort(-W, axis=1)[:, :2]
    for k in range(2):
        bi[:, k] = [sk.index[names[j]] for j in order[:, k]]
        bw[:, k] = W[np.arange(n), order[:, k]]
    bw /= bw.sum(1, keepdims=True)
    return bi, bw


def _pack_atlas(pieces, pad=0.012):
    """One rectangle per UV region, sized by surface area, aspect from the primitive's UV span.
    Shelf packing into the unit square; the scale is searched so everything fits."""
    info = {}
    for p, m, swapped in pieces:
        if swapped:
            continue  # mirrored copies share the region
        area = m.area() * p.uv_scale
        span_u, span_v = _uv_span(m)
        aspect = max(0.15, min(6.0, span_u / max(span_v, 1e-6)))
        prev = info.get(p.uv_region)
        if prev is None or area > prev[0]:
            info[p.uv_region] = (area, aspect)
    names = list(info)
    def layout(s):
        rects, x, y, shelf = {}, pad, pad, 0.0
        for n in sorted(names, key=lambda k: -np.sqrt(info[k][0] / info[k][1])):
            area, aspect = info[n]
            h = np.sqrt(area / aspect) * s; w = h * aspect
            if x + w + pad > 1.0:
                x, y, shelf = pad, y + shelf + pad, 0.0
            if w + 2 * pad > 1.0 or y + h + pad > 1.0:
                return None
            rects[n] = (x, y, w, h); x += w + pad; shelf = max(shelf, h)
        return rects
    lo, hi = 0.0, 50.0
    best = None
    for _ in range(40):
        mid = (lo + hi) / 2
        r = layout(mid)
        if r:
            best, lo = r, mid
        else:
            hi = mid
    return best


def _uv_span(m):
    """Approximate world length of the mesh along its local U and V (for the region aspect)."""
    du, dv = [], []
    for t in m.tris[: min(len(m.tris), 400)]:
        for a, b in ((t[0], t[1]), (t[1], t[2])):
            d = np.linalg.norm(m.verts[a] - m.verts[b])
            u = abs(m.uv[a, 0] - m.uv[b, 0]); v = abs(m.uv[a, 1] - m.uv[b, 1])
            if u > 1e-6 and v < 1e-6:
                du.append(d / u)
            if v > 1e-6 and u < 1e-6:
                dv.append(d / v)
    su = float(np.median(du)) if du else 1.0
    sv = float(np.median(dv)) if dv else su
    return su, sv
