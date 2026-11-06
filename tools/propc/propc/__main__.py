"""Command line:  python3 tools/propc <command>

  list                    props found in art/props/*.py (each file defines PROPS = [Prop(...), ...])
  build <id ...> | --all  write frontend/public/models/<id>/lod0..2.glb + model.json (+ index.json)
  render <id ...> | --all software-rendered PNG views to art/build/<id>/render.png
"""
import glob
import os
import runpy
import sys
import time

HERE = os.path.dirname(os.path.abspath(__file__))
TOOLS = os.path.dirname(os.path.dirname(HERE))
REPO = os.path.dirname(TOOLS)
sys.path[:0] = [os.path.join(TOOLS, "propc"), os.path.join(TOOLS, "modelc"), REPO]

from propc import writer  # noqa: E402

ART = os.path.join(REPO, "art")
OUT = os.path.join(REPO, "frontend", "public", "models")


def find_props():
    found = {}
    for f in sorted(glob.glob(os.path.join(ART, "props", "*.py"))):
        if os.path.basename(f).startswith("_"):
            continue
        for p in runpy.run_path(f, run_name="__propc__").get("PROPS", []):
            if p.id in found:
                sys.exit(f"duplicate prop id {p.id} in {f}")
            found[p.id] = p
    return found


def pick(props, args):
    if not args or args == ["--all"]:
        return list(props.values())
    missing = [a for a in args if a not in props]
    if missing:
        sys.exit(f"unknown props: {', '.join(missing)}")
    return [props[a] for a in args]


def cmd_build(args):
    props = find_props()
    chosen = pick(props, args)
    for p in chosen:
        t0 = time.time()
        meta = writer.write(p, OUT)
        tris = " / ".join(str(l["triangles"]) for l in meta["lods"])
        size = [round(meta["bounds"]["max"][i] - meta["bounds"]["min"][i], 1) for i in range(3)]
        print(f"{p.id}: {tris} tris, {size[0]}×{size[1]}×{size[2]} m — {time.time() - t0:.2f} s")
    writer.update_index(OUT, [p.id for p in chosen])


def cmd_render(args):
    import numpy as np
    from PIL import Image
    from modelc.preview import render
    props = find_props()
    for p in pick(props, args):
        colors = []
        pos, nrm, uv, tris = [], [], [], []
        for part in p.parts:
            col = (0.75, 0.2, 0.2) if part.kind == "team" else part.color
            if col not in colors:
                colors.append(col)
            ci = colors.index(col)
            v = part.verts
            for a, b, c in part.tris:
                A, B, C = v[a], v[b], v[c]
                n = np.cross(B - A, C - A)
                ln = np.linalg.norm(n)
                if ln < 1e-10:
                    continue
                k = len(pos)
                pos += [A, B, C]
                nrm += [n / ln] * 3
                uv += [((ci + 0.5) / 64, 0.5)] * 3
                tris.append((k, k + 1, k + 2))
        pal = np.zeros((1, 64, 3), np.uint8)
        for i, c in enumerate(colors[:64]):
            pal[0, i] = [int(255 * x) for x in c]
        img = render(np.array(pos), np.array(nrm), np.array(uv), np.array(tris), pal, None,
                     views=("front", "threequarter", "rts", "back"), height=420)
        d = os.path.join(ART, "build", p.id)
        os.makedirs(d, exist_ok=True)
        img.save(os.path.join(d, "render.png"))
        print("wrote", os.path.relpath(os.path.join(d, "render.png"), REPO))


def main(argv):
    if not argv or argv[0] in ("-h", "--help", "help"):
        print(__doc__)
        return
    cmd, args = argv[0], argv[1:]
    if cmd == "list":
        for pid, p in find_props().items():
            print(f"{pid:28s} {p.doc}")
    elif cmd == "build":
        cmd_build(args)
    elif cmd == "render":
        cmd_render(args)
    else:
        sys.exit(f"unknown command {cmd}\n{__doc__}")


if __name__ == "__main__":
    main(sys.argv[1:])
