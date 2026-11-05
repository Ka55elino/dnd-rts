"""Textures: the albedo PNG painted for the model's UV atlas, the team mask, the template to paint
on, and a placeholder painter for models that have no texture yet. UV convention: v = 0 is the top row."""
import os
import numpy as np
from PIL import Image, ImageDraw, ImageFont


def team_mask(built, size):
    img = Image.new("L", (size, size), 0)
    d = ImageDraw.Draw(img)
    for t in built.tris:
        if built.team[t].all():
            d.polygon([(built.uv[i, 0] * size, built.uv[i, 1] * size) for i in t], fill=255)
    return np.array(img)


def placeholder(built, size, seed=1):
    """Flat part colors with simple painted shading: lighter at the top of each region,
    darker rims, a little noise — enough to judge shapes and UVs in the game."""
    rng = np.random.default_rng(seed)
    img = np.zeros((size, size, 3))
    regions = built.regions
    colors = {}
    for p, m, _ in built.pieces:
        colors.setdefault(p.uv_region, (0.7, 0.7, 0.7) if p.team else p.color)
    for name, (x, y, w, h) in regions.items():
        x0, y0 = int(x * size) - 2, int(y * size) - 2
        x1, y1 = int((x + w) * size) + 2, int((y + h) * size) + 2
        x0, y0 = max(x0, 0), max(y0, 0)
        hh, ww = y1 - y0, x1 - x0
        yy, xx = np.mgrid[0:hh, 0:ww]
        v = yy / max(hh - 1, 1); u = xx / max(ww - 1, 1)
        shade = 1.08 - 0.25 * v                                   # light from above
        rim = 1 - 0.25 * np.clip(1 - np.minimum.reduce([u, 1 - u, v, 1 - v]) * 12, 0, 1)
        noise = 1 + rng.normal(0, 0.035, (hh, ww))
        c = np.array(colors[name])
        img[y0:y1, x0:x1] = np.clip(c[None, None, :] * (shade * rim * noise)[..., None], 0, 1)
    return (img * 255).astype(np.uint8)


def template(built, size, clean=False):
    """The sheet to paint the albedo on: every region with its name, the used area (filled) and
    the UV wireframe. Lofts (wrapped parts) are split into FRONT | BACK halves or FRONT | LEFT | BACK | RIGHT
    quarters (Model.wrap_views); v = 0 is the top."""
    from .geometry import Loft
    img = Image.new("RGB", (size, size), (24, 24, 28))
    d = ImageDraw.Draw(img)
    kind = {}
    for p, m, _ in built.pieces:
        kind.setdefault(p.uv_region, "wrap" if isinstance(p, Loft) else "face")
    for t in built.tris:
        d.polygon([(built.uv[i, 0] * size, built.uv[i, 1] * size) for i in t], fill=(58, 60, 68))
    if not clean:
        for t in built.tris:
            pts = [(built.uv[i, 0] * size, built.uv[i, 1] * size) for i in t]
            d.line(pts + [pts[0]], fill=(95, 120, 150), width=1)
    font = ImageFont.load_default()
    if clean:
        # for AI image tools: no wireframe and no text (they copy both into the picture)
        for name, (x, y, w, h) in built.regions.items():
            X0, Y0, X1, Y1 = x * size, y * size, (x + w) * size, (y + h) * size
            d.rectangle([X0, Y0, X1, Y1], outline=(120, 100, 50), width=1)
            if kind[name] == "wrap":
                nv = getattr(built, "wrap_views", 2)
                for k in range(1, nv):
                    xa = X0 + (X1 - X0) * k / nv
                    d.line([(xa, Y0), (xa, Y1)], fill=(90, 60, 40), width=1)
        return img
    for name, (x, y, w, h) in built.regions.items():
        X0, Y0, X1, Y1 = x * size, y * size, (x + w) * size, (y + h) * size
        d.rectangle([X0, Y0, X1, Y1], outline=(220, 180, 60), width=2)
        d.text((X0 + 4, Y0 + 3), name, fill=(255, 230, 120), font=font)
        if kind[name] == "wrap":
            nv = getattr(built, "wrap_views", 2)
            names = ["FRONT", "BACK"] if nv == 2 else ["FRONT", "LEFT", "BACK", "RIGHT"]
            for k in range(nv):
                xa = X0 + (X1 - X0) * k / nv
                if k:
                    d.line([(xa, Y0), (xa, Y1)], fill=(220, 120, 60), width=2)
                d.text((xa + 4, Y1 - 13), names[k], fill=(255, 160, 90), font=font)
            d.text((X1 - 26, Y0 + 3), "top", fill=(160, 200, 255), font=font)
        else:
            d.text((X0 + 4, Y1 - 13), "FACE", fill=(160, 220, 160), font=font)
    return img


def fill_wrapped(rgb, built, tol=22, skip=()):
    """AI-painted atlases draw each part as an object on a dark background. On a wrapped part that
    background lands on the joins between the views and on the ends as dark seams. Two steps:
      1. fit: every view (FRONT, BACK) is cropped to the box of its drawing and scaled to fill its
         half — one scale per axis, so the drawing is not distorted;
      2. fill: what is still background (rounded corners of the silhouette) is filled with a soft
         blur of the nearby paint; it wraps across the join between the halves and the outer edges.
    background = pixels of the region's most common edge color connected to its border, so dark
    paint inside the drawing stays."""
    from .geometry import Loft
    H, W = rgb.shape[:2]
    out = rgb.copy()
    wrapped = {p.uv_region for p, m, _ in built.pieces if isinstance(p, Loft)}
    nv = getattr(built, "wrap_views", 2)
    for name in wrapped - set(skip):
        x, y, w, h = built.regions[name]
        X0, Y0, X1, Y1 = int(x * W), int(y * H), int((x + w) * W), int((y + h) * H)
        if X1 - X0 < 8 * nv or Y1 - Y0 < 8:
            continue
        block = out[Y0:Y1, X0:X1].astype(float)
        bg = _background(block, tol)
        if bg.mean() < 0.02 or bg.mean() > 0.97:
            continue
        hh, ww = bg.shape
        cuts = [ww * k // nv for k in range(nv + 1)]
        fit = np.zeros_like(block); paint = np.zeros((hh, ww), dtype=bool)
        for c0, c1 in zip(cuts, cuts[1:]):
            vb, vm = block[:, c0:c1], ~bg[:, c0:c1]
            box = _content_box(vm)
            if box is None:
                continue
            r0, r1, k0, k1 = box
            img = Image.fromarray(np.clip(vb[r0:r1, k0:k1], 0, 255).astype(np.uint8))
            fit[:, c0:c1] = np.array(img.resize((c1 - c0, hh), Image.LANCZOS), dtype=float)
            msk = Image.fromarray((vm[r0:r1, k0:k1] * 255).astype(np.uint8))
            paint[:, c0:c1] = np.array(msk.resize((c1 - c0, hh), Image.BILINEAR)) > 200
        if not paint.any():
            continue
        out[Y0:Y1, X0:X1] = np.clip(_soft_fill(fit, paint), 0, 255).astype(np.uint8)
    return out


def _content_box(mask, frac=0.03):
    """Box of the drawing: rows / columns where more than frac of the pixels are painted (stray
    specks and copied labels do not count)."""
    rows = np.nonzero(mask.mean(1) > frac)[0]
    cols = np.nonzero(mask.mean(0) > frac)[0]
    if len(rows) < 4 or len(cols) < 4:
        return None
    return rows[0], rows[-1] + 1, cols[0], cols[-1] + 1


def _box_blur(a, r):
    """Box blur of radius r along both axes; wraps around in x (the region is a ring), clamps in y."""
    for axis, mode in ((1, "wrap"), (0, "edge")):
        pad = [(0, 0)] * a.ndim; pad[axis] = (r + 1, r)
        c = np.cumsum(np.pad(a, pad, mode=mode), axis=axis)
        n = a.shape[axis]
        a = (np.take(c, range(2 * r + 1, 2 * r + 1 + n), axis=axis) - np.take(c, range(0, n), axis=axis)) / (2 * r + 1)
    return a


def _soft_fill(img, known):
    """Fill the unknown pixels with a blur of the known ones: normalized blurs of growing radius,
    each filling what the smaller ones could not reach; the filled area is smoothed once more."""
    w = known.astype(float)
    out = img * w[..., None]
    done = known.copy()
    r = 2
    while not done.all() and r < max(img.shape[:2]):
        cw = w
        cc = img * w[..., None]
        for _ in range(3):  # 3 box blurs ≈ a gaussian
            cw = _box_blur(cw, r); cc = _box_blur(cc, r)
        ok = (~done) & (cw > 1e-4)
        out[ok] = cc[ok] / cw[ok, None]
        done |= ok
        r *= 2
    # soften the steps between radii; keep the painted pixels as they are
    s = _box_blur(_box_blur(out, 3), 3)
    out[~known] = s[~known]
    return out


def _background(block, tol):
    """Background mask of a region: close to the background color and connected to the border.
    A thin band along the edges is always background (AI tools often copy the layout's outline
    there); the background color is the most common color in a wider band."""
    k = max(4, min(block.shape[:2]) // 30)
    band = np.zeros(block.shape[:2], dtype=bool)
    band[:k], band[-k:], band[:, :k], band[:, -k:] = True, True, True, True
    wide = np.zeros_like(band)
    wide[:3 * k], wide[-3 * k:], wide[:, :3 * k], wide[:, -3 * k:] = True, True, True, True
    px = block[wide]
    q = (px // 16).astype(int) @ np.array([256, 16, 1])
    bins, counts = np.unique(q, return_counts=True)
    near = np.linalg.norm(block - px[q == bins[np.argmax(counts)]].mean(0), axis=2) < tol
    near |= band
    bg = band.copy()
    while True:  # flood fill from the border through background-colored pixels
        grow = bg.copy()
        grow[1:] |= bg[:-1]; grow[:-1] |= bg[1:]; grow[:, 1:] |= bg[:, :-1]; grow[:, :-1] |= bg[:, 1:]
        grow &= near
        if (grow == bg).all():
            return bg
        bg = grow


def load_albedo(path, built, size, fill=True):
    """RGBA8: RGB from the PNG (or a placeholder), alpha = team mask of the team parts."""
    if path and os.path.exists(path):
        rgb = np.array(Image.open(path).convert("RGB"))
        src = path
    else:
        rgb = placeholder(built, 1024)
        src = None
    if fill:
        rgb = fill_wrapped(rgb, built)
    rgb = np.array(Image.fromarray(rgb).resize((size, size), Image.LANCZOS))
    a = team_mask(built, size)
    return np.dstack([rgb, a]).astype(np.uint8), src


def sample_colors(albedo, built):
    """Average color around each vertex's UV (the far LOD has no texture), + the team flag."""
    H, W = albedo.shape[:2]
    R = max(1, W // 64)
    out = np.zeros((len(built.verts), 4), dtype=np.uint8)
    for i, (u, v) in enumerate(built.uv):
        cx, cy = int(u * W), int(v * H)
        patch = albedo[max(0, cy - R): cy + R + 1, max(0, cx - R): cx + R + 1].reshape(-1, 4).astype(float)
        out[i, :3] = patch[:, :3].mean(0).round()
        out[i, 3] = 255 if built.team[i] or patch[:, 3].mean() > 127 else 0
    return out
