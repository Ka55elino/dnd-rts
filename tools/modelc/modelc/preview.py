"""A small software renderer for quick looks without Blender or the game: z-buffer, the albedo texture,
smooth normals, a sun + sky light and a specular highlight from the material map.

    python3 tools/modelc/modelc render <id> [clip@sec] [--views front,side,back,threequarter]
writes art/build/<id>/render_<clip>_<sec>.png
"""
import math
import numpy as np
from PIL import Image

VIEWS = {  # camera yaw around the vertical axis (degrees, 0 = looking at the character's front), pitch
    "front": (0, 8), "threequarter": (35, 12), "above": (150, 45), "rts": (30, 50), "side": (90, 5), "back": (180, 8), "left": (-90, 5),
}


def _rot(yaw, pitch):
    y, p = math.radians(yaw), math.radians(pitch)
    Ry = np.array([[math.cos(y), 0, math.sin(y)], [0, 1, 0], [-math.sin(y), 0, math.cos(y)]])
    Rx = np.array([[1, 0, 0], [0, math.cos(p), -math.sin(p)], [0, math.sin(p), math.cos(p)]])
    return Rx @ Ry


def render(pos, nrm, uv, tris, albedo, material, views=("front", "threequarter", "side", "back"), height=640, cull=-1):
    """pos / nrm: V x 3 in game axes (Y up, front +Z). albedo / material: H x W x 3|4 uint8.
    cull: back faces are skipped like in the game (one-sided triangles), so holes show up here too."""
    lo, hi = pos.min(0), pos.max(0)
    centre = (lo + hi) / 2
    size = float(max(hi[1] - lo[1], np.linalg.norm((hi - lo)[[0, 2]]))) * 1.08
    scale = height / size
    W = int(height * 0.72)
    sun = np.array([-0.45, 0.8, 0.55]); sun /= np.linalg.norm(sun)
    tiles = []
    A = albedo[..., :3].astype(float) / 255
    M = material.astype(float) / 255 if material is not None else None
    TH, TW = A.shape[:2]
    for v in views:
        R = _rot(*VIEWS[v])
        p = (pos - centre) @ R.T              # camera looks along -Z, so larger z = nearer
        n = nrm @ R.T
        sun_c = R @ sun
        X = W / 2 + p[:, 0] * scale
        Y = height / 2 - p[:, 1] * scale
        Z = p[:, 2]
        img = np.zeros((height, W, 3)); img[:] = (0.15, 0.17, 0.2)
        zb = np.full((height, W), -1e9)
        for t in tris:
            a, b, c = t
            x0, x1 = int(max(min(X[a], X[b], X[c]), 0)), int(min(max(X[a], X[b], X[c]) + 1, W))
            y0, y1 = int(max(min(Y[a], Y[b], Y[c]), 0)), int(min(max(Y[a], Y[b], Y[c]) + 1, height))
            if x1 <= x0 or y1 <= y0:
                continue
            gx, gy = np.meshgrid(np.arange(x0, x1) + 0.5, np.arange(y0, y1) + 0.5)
            d = (Y[b] - Y[c]) * (X[a] - X[c]) + (X[c] - X[b]) * (Y[a] - Y[c])
            if abs(d) < 1e-9:
                continue
            if cull and d * cull < 0:     # back face: the game does not draw it either
                continue
            w0 = ((Y[b] - Y[c]) * (gx - X[c]) + (X[c] - X[b]) * (gy - Y[c])) / d
            w1 = ((Y[c] - Y[a]) * (gx - X[c]) + (X[a] - X[c]) * (gy - Y[c])) / d
            w2 = 1 - w0 - w1
            inside = (w0 >= 0) & (w1 >= 0) & (w2 >= 0)
            if not inside.any():
                continue
            z = w0 * Z[a] + w1 * Z[b] + w2 * Z[c]
            sub = zb[y0:y1, x0:x1]
            vis = inside & (z > sub)
            if not vis.any():
                continue
            sub[vis] = z[vis]
            nn = (w0[..., None] * n[a] + w1[..., None] * n[b] + w2[..., None] * n[c])[vis]
            nn /= np.maximum(np.linalg.norm(nn, axis=1, keepdims=True), 1e-9)
            u = (w0 * uv[a, 0] + w1 * uv[b, 0] + w2 * uv[c, 0])[vis]
            vv = (w0 * uv[a, 1] + w1 * uv[b, 1] + w2 * uv[c, 1])[vis]
            tx = np.clip((u * TW).astype(int), 0, TW - 1); ty = np.clip((vv * TH).astype(int), 0, TH - 1)
            col = A[ty, tx]
            ndl = np.clip(nn @ sun_c, 0, 1)
            sky = 0.35 + 0.2 * nn[:, 1]
            shade = col * (sky[:, None] * np.array([0.8, 0.85, 1.0]) + ndl[:, None] * np.array([1.0, 0.95, 0.85]) * 0.75)
            if M is not None:
                spec_s, gloss = M[ty, tx, 0], M[ty, tx, 1]
                h = sun_c + np.array([0, 0, 1.0]); h /= np.linalg.norm(h)
                sp = np.clip(nn @ h, 0, 1) ** (6 + 84 * gloss)
                shade += (spec_s * sp)[:, None] * 0.9
            region = img[y0:y1, x0:x1]
            region[vis] = shade
        tiles.append((np.clip(img, 0, 1) ** (1 / 1.15) * 255).astype(np.uint8))
    return Image.fromarray(np.hstack(tiles))
