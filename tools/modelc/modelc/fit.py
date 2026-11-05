"""Fit an AI-painted atlas to the model's real UV layout.

AI image tools keep the rows of the layout and the order of the pieces in a row, but not the exact
coordinates: pieces drift sideways by a few percent, get wider or narrower. Here every piece of
paint is found and moved into its rectangle:

  1. content mask: pixels that differ from the flat background (brightness or local texture);
  2. the regions are grouped into shelves (rows of the atlas); in each shelf's band of the image
     the columns with paint form runs (pieces, or parts of pieces);
  3. the runs are split into as many consecutive groups as the shelf has regions, choosing the
     split whose groups best match the regions' spans (dynamic programming) — so a wrapped region
     painted as two pictures (FRONT and BACK) takes both, and a drifted row still maps in order;
  4. each group's box (its columns, the rows with paint inside them) is scaled into the region's
     rectangle. Outside the regions the result is the background color.

The wrapped-region fill at build time then removes what is left of the background inside a region.
"""
import numpy as np
from PIL import Image


def _box(x, r):
    c = np.pad(x, ((r + 1, r), (r + 1, r)), mode="edge").cumsum(0).cumsum(1)
    n = 2 * r + 1
    return (c[n:, n:] - c[:-n, n:] - c[n:, :-n] + c[:-n, :-n]) / (n * n)


def content_mask(rgb):
    g = rgb.astype(float).mean(2)
    bg = float(np.median(g[g < np.percentile(g, 40)]))
    m = _box(g, 2)
    sd = np.sqrt(np.maximum(_box(g * g, 2) - m * m, 0))
    return (np.abs(m - bg) > 5) | (sd > 3), bg


def _runs(profile, thresh, min_gap):
    """Index ranges where profile > thresh, gaps shorter than min_gap merged."""
    on = profile > thresh
    runs, start = [], None
    for i, v in enumerate(on):
        if v and start is None:
            start = i
        elif not v and start is not None:
            runs.append([start, i]); start = None
    if start is not None:
        runs.append([start, len(on)])
    merged = []
    for r in runs:
        if merged and r[0] - merged[-1][1] < min_gap:
            merged[-1][1] = r[1]
        else:
            merged.append(r)
    return [r for r in merged if r[1] - r[0] >= 3]


def _shelves(regions):
    """Regions grouped by their top edge (the packer fills the atlas shelf by shelf), left to right."""
    items = sorted(regions.items(), key=lambda kv: (round(kv[1][1], 3), kv[1][0]))
    shelves = []
    for name, r in items:
        if shelves and abs(shelves[-1][0][1][1] - r[1]) < 0.004:
            shelves[-1].append((name, r))
        else:
            shelves.append([(name, r)])
    return [sorted(s, key=lambda kv: kv[1][0]) for s in shelves]


def _split(runs, spans):
    """Split the runs (sorted, in pixels) into len(spans) consecutive non-empty groups. The shelf is
    first mapped onto the painted row (scale + offset from the row's ends); then a group should sit
    where its region sits and, above all, be as wide as the region (AI tools keep proportions better
    than positions). Returns the groups as (x0, x1)."""
    R, K = len(runs), len(spans)
    if R < K:
        return None
    a0, a1 = spans[0][0], spans[-1][1]
    b0, b1 = runs[0][0], runs[-1][1]
    s = (b1 - b0) / max(a1 - a0, 1e-9)
    spans = [(b0 + (x0 - a0) * s, b0 + (x1 - a0) * s) for x0, x1 in spans]
    INF = float("inf")
    cost = np.full((K + 1, R + 1), INF); cost[0, 0] = 0
    back = np.zeros((K + 1, R + 1), dtype=int)
    for k in range(1, K + 1):
        x0, x1 = spans[k - 1]
        for j in range(k, R - (K - k) + 1):
            for i in range(k - 1, j):
                if cost[k - 1, i] == INF:
                    continue
                g0, g1 = runs[i][0], runs[j - 1][1]
                c = cost[k - 1, i] + abs((g0 + g1) - (x0 + x1)) / 2 + 2 * abs((g1 - g0) - (x1 - x0))
                if c < cost[k, j]:
                    cost[k, j], back[k, j] = c, i
    out, j = [], R
    for k in range(K, 0, -1):
        i = back[k, j]
        out.append((runs[i][0], runs[j - 1][1]))
        j = i
    return out[::-1]


def fit_atlas(src, regions, size=None):
    """src: H x W x 3 uint8 (an AI atlas); regions: {name: (x, y, w, h)} in 0..1.
    Returns (image, report) — the image at the source resolution (or `size`)."""
    H, W = src.shape[:2]
    size = size or W
    mask, bg = content_mask(src)
    out = np.zeros((size, size, 3), dtype=np.uint8); out[:] = int(round(bg))
    report = []
    shelves = _shelves(regions)
    # Cut lines between the painted rows: the emptiest image row between the bottom of one shelf and
    # the top of the next (searched a bit wider, the picture may have drifted).
    prof = mask.mean(1)
    cuts = [0]
    for a, b in zip(shelves, shelves[1:]):
        lo = max(r[1] + r[3] for _, r in a) - 0.05
        hi = min(r[1] for _, r in b) + 0.03
        i0, i1 = int(max(lo, 0) * H), int(min(hi, 1) * H)
        cuts.append(i0 + int(np.argmin(prof[i0:i1])) if i1 > i0 else int(hi * H))
    cuts.append(H)
    for si, shelf in enumerate(shelves):
        top = min(r[1] for _, r in shelf)
        bottom = max(r[1] + r[3] for _, r in shelf)
        short = min(r[3] for _, r in shelf)
        # columns are read in the middle of the shelf's shortest region (clear of the rows above and
        # below even when the picture drifted a bit); rows in a wider band, to allow vertical drift
        c0, c1 = int((top + short * 0.15) * H), int((top + short * 0.85) * H)
        y0, y1 = cuts[si], cuts[si + 1]
        band = mask[y0:y1]
        runs = _runs(mask[c0:c1].mean(0), 0.02, max(3, W // 300))
        spans = [(r[0] * W, (r[0] + r[2]) * W) for _, r in shelf]
        groups = _split(runs, spans)
        if groups is None:
            report.append(f"shelf at y={top:.2f}: {len(runs)} pieces of paint for {len(shelf)} regions — left as background")
            continue
        for (name, (rx, ry, rw, rh)), (gx0, gx1) in zip(shelf, groups):
            # rows with paint in the group's columns, inside this shelf's painted row: the longest run
            col = band[:, gx0:gx1].mean(1)
            vr = _runs(col, 0.02, max(3, H // 150))
            if not vr:
                continue
            gy0, gy1 = max(vr, key=lambda r: r[1] - r[0])
            crop = src[y0 + gy0: y0 + gy1, gx0:gx1]
            X0, Y0 = int(round(rx * size)), int(round(ry * size))
            X1, Y1 = int(round((rx + rw) * size)), int(round((ry + rh) * size))
            out[Y0:Y1, X0:X1] = np.array(Image.fromarray(crop).resize((X1 - X0, Y1 - Y0), Image.LANCZOS))
            moved = (gx0 / W - rx, (y0 + gy0) / H - ry)
            report.append(f"{name:10} <- x {gx0 / W:.3f}..{gx1 / W:.3f} y {(y0 + gy0) / H:.3f}..{(y0 + gy1) / H:.3f}"
                          f"  (moved {moved[0]:+.3f}, {moved[1]:+.3f})")
    return out, report
