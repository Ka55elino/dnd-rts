"""Writes the compiled model in the format the game reads (the same as tools/vatbake):
frontend/public/models/<id>/model.json, geometry.bin, vat.bin, albedo.bin; updates index.json."""
import json
import math
import os
import numpy as np
from .anim import to_matrices

TEX_WIDTH = 2048
LOD1_RATIO, LOD2_RATIO = 0.4, 0.1


def to_game(p):
    """Authoring (Z up, front -Y) -> game (Y up, front +Z)."""
    return np.column_stack([p[:, 0], p[:, 2], -p[:, 1]])


def skin_frame(built, sk, pose):
    M = sk.skin_matrices(to_matrices(pose))
    P = np.column_stack([built.verts, np.ones(len(built.verts))])
    out = np.zeros((len(P), 3))
    for k in range(built.bone_idx.shape[1]):
        w = built.bone_w[:, k]
        if not w.any():
            continue
        Mk = M[built.bone_idx[:, k]]                     # (n, 4, 4)
        out += w[:, None] * np.einsum("nij,nj->ni", Mk, P)[:, :3]
    return out


MATERIALS = {   # (specular, gloss) — R and G of material.bin
    "metal": (0.85, 0.75), "gold": (0.8, 0.6), "leather": (0.25, 0.3), "wood": (0.12, 0.2),
    "cloth": (0.04, 0.1), "skin": (0.15, 0.25),
}


def smoothing_groups(built):
    """Vertex -> smoothing group. Lofts are smooth (vertices at the same place on the same piece, like
    the seam of a ring, share a normal); Extrudes keep hard edges (each face has its own vertices)."""
    from .geometry import Loft
    group = np.arange(len(built.verts))
    keys = {}
    for i, (v, pi) in enumerate(zip(built.verts, built.piece)):
        if not isinstance(built.pieces[pi][0], Loft):
            continue
        k = (int(pi), *np.round(v / 1e-5).astype(np.int64))
        group[i] = keys.setdefault(k, i)
    return group


def vertex_normals(pos, tris, group):
    """Area-weighted smooth normals of a posed frame (pos: V x 3), shared inside each smoothing group."""
    a, b, c = pos[tris[:, 0]], pos[tris[:, 1]], pos[tris[:, 2]]
    fn = np.cross(b - a, c - a)
    acc = np.zeros_like(pos)
    for k in range(3):
        np.add.at(acc, group[tris[:, k]], fn)
    n = acc[group]
    return n / np.maximum(np.linalg.norm(n, axis=1, keepdims=True), 1e-9)


def material_map(built, size):
    """RGBA8 texture over the UV atlas: R = specular, G = gloss of each part's material."""
    from PIL import Image, ImageDraw
    img = Image.new("RGBA", (size, size), (int(0.04 * 255), int(0.1 * 255), 0, 255))
    d = ImageDraw.Draw(img)
    for t in built.tris:
        p = built.pieces[built.piece[t[0]]][0]
        m = p.material
        spec, gloss = MATERIALS[m] if isinstance(m, str) else m
        d.polygon([(built.uv[i, 0] * size, built.uv[i, 1] * size) for i in t],
                  fill=(int(spec * 255), int(gloss * 255), 0, 255))
    return np.array(img)


def bake(model, built, albedo, out_root, fps=15):
    sk = model.skeleton
    V = len(built.verts)
    frames_all, normals_all, meta_clips, frame = [], [], {}, 0
    group = smoothing_groups(built)
    for name, clip in model.clips.items():
        n = max(2, round(clip.duration * fps)) if clip.loop else round(clip.duration * fps) + 1
        meta_clips[name] = {"start": frame, "frames": n, "loop": clip.loop, "duration": round(clip.duration, 3)}
        if clip.ground_speed:
            meta_clips[name]["groundSpeed"] = round(clip.ground_speed, 3)
        for f in range(n):
            t = (f / n) * clip.duration if clip.loop else min(f / fps, clip.duration)
            pos = to_game(skin_frame(built, sk, clip.pose(t, sk))) * getattr(model, "scale", 1.0)
            frames_all.append(pos)
            normals_all.append(vertex_normals(pos, built.tris, group))
        frame += n
    total = frame
    allpos = np.stack(frames_all)                              # (frames, V, 3)
    tex_h = math.ceil(V * total / TEX_WIDTH)
    vat = np.zeros((TEX_WIDTH * tex_h, 4), dtype=np.float16)
    vat[: V * total, :3] = allpos.reshape(-1, 3)
    vat[: V * total, 3] = 1
    # normals: same layout as the VAT, RGBA8, n * 0.5 + 0.5
    nrm = np.zeros((TEX_WIDTH * tex_h, 4), dtype=np.uint8)
    nrm[: V * total, :3] = np.clip(np.round((np.stack(normals_all).reshape(-1, 3) * 0.5 + 0.5) * 255), 0, 255)
    nrm[: V * total, 3] = 255
    mat = material_map(built, albedo.shape[0])

    bind = (to_game(built.verts) * getattr(model, "scale", 1.0)).astype(np.float32)
    lod0 = built.tris.astype(np.uint32).reshape(-1)
    lod1 = cluster_lod(bind, built.tris, int(len(built.tris) * LOD1_RATIO))
    lod2 = cluster_lod(bind, built.tris, int(len(built.tris) * LOD2_RATIO))

    from .texture import sample_colors
    colors = sample_colors(albedo, built)
    uvs = built.uv.astype(np.float32)

    b0, b1, b2 = lod0.nbytes, lod1.nbytes, lod2.nbytes
    col_off = b0 + b1 + b2
    pos_off = col_off + colors.nbytes
    uv_off = pos_off + bind.nbytes
    out = os.path.join(out_root, model.id)
    os.makedirs(out, exist_ok=True)
    with open(os.path.join(out, "geometry.bin"), "wb") as f:
        for a in (lod0, lod1, lod2, colors, bind, uvs):
            f.write(np.ascontiguousarray(a).tobytes())
    with open(os.path.join(out, "vat.bin"), "wb") as f:
        f.write(vat.tobytes())
    with open(os.path.join(out, "albedo.bin"), "wb") as f:
        f.write(albedo.tobytes())
    with open(os.path.join(out, "normals.bin"), "wb") as f:
        f.write(nrm.tobytes())
    with open(os.path.join(out, "material.bin"), "wb") as f:
        f.write(mat.tobytes())
    meta = {
        "version": 1, "fps": fps, "vertexCount": V, "frames": total,
        "texture": {"width": TEX_WIDTH, "height": tex_h, "format": "rgba16f"},
        "clips": meta_clips,
        "bounds": {"min": [round(float(x), 3) for x in allpos.reshape(-1, 3).min(0)],
                   "max": [round(float(x), 3) for x in allpos.reshape(-1, 3).max(0)]},
        "geometry": {
            "lod0": {"offset": 0, "count": int(lod0.size)},
            "lod1": {"offset": b0, "count": int(lod1.size)},
            "lod2": {"offset": b0 + b1, "count": int(lod2.size)},
            "colors": {"offset": col_off}, "positions": {"offset": pos_off}, "uvs": {"offset": uv_off},
        },
        "albedo": {"file": "albedo.bin", "width": int(albedo.shape[1]), "height": int(albedo.shape[0])},
        # optional (modelc): per-frame vertex normals in the VAT layout (RGBA8, n * 0.5 + 0.5) -> smooth
        # shading; material map over the UV atlas (RGBA8: R = specular, G = gloss)
        "normals": {"file": "normals.bin", "format": "rgba8"},
        "material": {"file": "material.bin", "width": int(mat.shape[1]), "height": int(mat.shape[0])},
        "source": "modelc",
        # build time: Model Overview polls model.json and reloads the model when this changes
        "build": __import__("time").strftime("%Y-%m-%dT%H:%M:%S"),
    }
    with open(os.path.join(out, "model.json"), "w") as f:
        f.write(json.dumps(meta, indent=2) + "\n")
    return {"vertices": V, "tris": len(built.tris), "lod1": lod1.size // 3, "lod2": lod2.size // 3,
            "frames": total, "clips": {k: v["frames"] for k, v in meta_clips.items()},
            "vat_mb": (vat.nbytes + nrm.nbytes) / 1048576}


def cluster_lod(pos, tris, target):
    """Simplified index buffer over the SAME vertices (the VAT stores every vertex once):
    vertices are snapped to a grid and each cell keeps its first vertex; degenerate triangles go."""
    lo, hi = 1e-4, float(np.ptp(pos, axis=0).max())
    best = tris
    for _ in range(30):
        c = (lo + hi) / 2
        keys = np.floor(pos / c).astype(np.int64)
        _, first, inv = np.unique(keys, axis=0, return_index=True, return_inverse=True)
        rep = first[inv.reshape(-1)]
        t = rep[tris]
        ok = (t[:, 0] != t[:, 1]) & (t[:, 1] != t[:, 2]) & (t[:, 0] != t[:, 2])
        t = t[ok]
        keep = np.unique(np.sort(t, axis=1), axis=0, return_index=True)[1]
        t = t[np.sort(keep)]
        if len(t) <= target:
            best, hi = t, c
        else:
            lo = c
    return best.astype(np.uint32).reshape(-1)


def update_index(models_root, ids):
    f = os.path.join(models_root, "index.json")
    try:
        data = json.load(open(f))
    except Exception:
        data = {"models": []}
    models = [m for m in data.get("models", []) if m["id"] not in ids]
    models += [{"id": i, "type": "vat"} for i in ids if os.path.exists(os.path.join(models_root, i, "model.json"))]
    data["models"] = sorted(models, key=lambda m: m["id"])
    data["$comment"] = ("Generated by tools/vatbake, tools/meshbake and tools/modelc: every baked model. "
                        "vat = animated (VatModel), static = LODs (StaticModel).")
    with open(f, "w") as fh:
        fh.write(json.dumps({"$comment": data["$comment"], "models": data["models"]}, indent=2) + "\n")
