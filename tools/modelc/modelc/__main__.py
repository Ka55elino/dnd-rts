"""Command line:  python3 tools/modelc <command> [model id]

  list                      models found in art/models/
  build <id> | --all        compile into frontend/public/models/<id>/ (+ index.json); also refreshes
                            art/build/<id>/template.png and template_clean.png
  watch <id>                rebuild on every change of art/ or tools/modelc/
  template <id> [--clean]   write art/build/<id>/template.png (UV regions + wireframe) to paint on;
                            --clean: template_clean.png without wireframe and text, for AI image tools
  paint <id> [--force]      write the placeholder texture to art/textures/<texture>/<skin>.png
  atlas <id> --lock         freeze the current UV layout into art/textures/<texture>/atlas.json
                            (after a texture is painted, so later shape changes keep its pixels in place)
  obj <id> [clip@sec ...]   write OBJ snapshots (+ texture) to art/build/<id>/ to look at in any 3D viewer
  fit <id> <image.png> [--skin name] [--box region=x0,x1 ...]
                            move the pieces of an AI-painted atlas into the model's UV rectangles
                            (AI tools keep the rows and the order, not the coordinates) and save it
                            as art/textures/<texture>/<skin>.png; locks the atlas. --box: where a
                            region's painting really is (columns, 0..1) when the automatic split
                            gets a row wrong
  render <id> [clip@sec ...] [--views front,threequarter,side,back]
                            software-rendered PNG (texture, smooth shading, specular) to art/build/<id>/
"""
import glob
import os
import runpy
import sys
import time

HERE = os.path.dirname(os.path.abspath(__file__))
TOOLS = os.path.dirname(HERE)
REPO = os.path.dirname(os.path.dirname(TOOLS))
sys.path[:0] = [TOOLS, REPO]

from modelc.model import build  # noqa: E402
from modelc import texture, writer  # noqa: E402

ART = os.path.join(REPO, "art")
OUT = os.path.join(REPO, "frontend", "public", "models")


def find_models():
    found = {}
    for f in sorted(glob.glob(os.path.join(ART, "models", "**", "*.py"), recursive=True)):
        if os.path.basename(f).startswith("_"):
            continue
        found[os.path.splitext(os.path.basename(f))[0]] = f
    return found


def load(mid):
    models = find_models()
    if mid not in models:
        sys.exit(f"no model '{mid}' (have: {', '.join(models)})")
    ns = runpy.run_path(models[mid], run_name="__modelc__")
    if "MODEL" not in ns:
        sys.exit(f"{models[mid]}: defines no MODEL")
    return ns["MODEL"]


def texture_path(model, skin):
    return os.path.join(ART, "textures", model.texture, f"{skin}.png") if model.texture else None


def cmd_build(mid):
    t0 = time.time()
    model = load(mid)
    built = build(model)
    skin = model.skins[0]
    albedo, src = texture.load_albedo(texture_path(model, skin), built, model.texture_size)
    info = writer.bake(model, built, albedo, OUT, fps=getattr(model, "fps", 15))
    # keep the paint templates in step with the shapes (they used to be written only by `template`)
    d = os.path.join(ART, "build", model.id)
    os.makedirs(d, exist_ok=True)
    for clean in (False, True):
        texture.template(built, max(1024, model.texture_size), clean=clean).save(
            os.path.join(d, "template_clean.png" if clean else "template.png"))
    writer.update_index(OUT, [model.id])
    print(f"{model.id}: {info['vertices']} vertices, {info['tris']} tris (LOD1 {info['lod1']}, LOD2 {info['lod2']}), "
          f"{info['frames']} frames {info['clips']}, VAT {info['vat_mb']:.1f} MB, "
          f"texture {src.replace(REPO + os.sep, '') if src else 'placeholder'} — {time.time() - t0:.1f} s")


def cmd_template(mid, clean=False):
    model = load(mid)
    built = build(model)
    d = os.path.join(ART, "build", model.id)
    os.makedirs(d, exist_ok=True)
    f = os.path.join(d, "template_clean.png" if clean else "template.png")
    texture.template(built, max(1024, model.texture_size), clean=clean).save(f)
    print("wrote", os.path.relpath(f, REPO))
    if clean:  # the region list goes into the prompt instead of text on the picture
        from modelc.geometry import Loft
        kinds = {p.uv_region: ("FRONT|BACK" if isinstance(p, Loft) else "FACE") for p, m, _ in built.pieces}
        for n, (x, y, w, h) in built.regions.items():
            print(f"  {n:10} {kinds[n]:10} x={x:.2f} y={y:.2f} w={w:.2f} h={h:.2f}")


def cmd_paint(mid, force):
    from PIL import Image
    model = load(mid)
    built = build(model)
    for skin in model.skins:
        f = texture_path(model, skin)
        if os.path.exists(f) and not force:
            print("exists (use --force):", os.path.relpath(f, REPO)); continue
        os.makedirs(os.path.dirname(f), exist_ok=True)
        Image.fromarray(texture.placeholder(built, model.texture_size)).save(f)
        print("wrote", os.path.relpath(f, REPO))


def cmd_atlas(mid, lock):
    import json
    from modelc.model import atlas_file
    model = load(mid)
    f = atlas_file(model)
    if lock and os.path.exists(f):
        os.remove(f)  # re-pack from the current parts
    built = build(model)
    if lock:
        os.makedirs(os.path.dirname(f), exist_ok=True)
        json.dump({"regions": {k: [round(float(x), 5) for x in v] for k, v in built.regions.items()}}, open(f, "w"), indent=1)
        print("locked", os.path.relpath(f, REPO))
    for n, (x, y, w, h) in built.regions.items():
        print(f"  {n:10} x={x:.3f} y={y:.3f} w={w:.3f} h={h:.3f}")


def cmd_fit(mid, args):
    from PIL import Image
    import numpy as np
    from modelc.fit import fit_atlas
    skin = None
    boxes = {}
    files = []
    it = iter(args)
    for a in it:
        if a == "--skin":
            skin = next(it)
        elif a == "--box":           # --box blade=0.51,0.67 : that region's painting is in these columns
            n, v = next(it).split("=")
            boxes[n] = tuple(float(x) for x in v.split(","))
        else:
            files.append(a)
    if not files:
        sys.exit("usage: modelc fit <id> <image.png> [--skin name]")
    model = load(mid)
    built = build(model)
    src = np.array(Image.open(files[0]).convert("RGB"))
    out, report = fit_atlas(src, built.regions, size=max(src.shape[:2]), boxes=boxes)
    for line in report:
        print(" ", line)
    dst = texture_path(model, skin or model.skins[0])
    os.makedirs(os.path.dirname(dst), exist_ok=True)
    Image.fromarray(out).save(dst)
    print("wrote", os.path.relpath(dst, REPO))
    cmd_atlas(mid, True)   # the painted texture now fixes the layout


def cmd_render(mid, args):
    from modelc.writer import skin_frame, to_game, smoothing_groups, vertex_normals, material_map
    from modelc.preview import render
    views = ("front", "threequarter", "side", "back")
    shots = []
    for a in args:
        if a.startswith("--views="):
            views = tuple(a.split("=", 1)[1].split(","))
        else:
            shots.append(a)
    model = load(mid)
    built = build(model)
    albedo, _ = texture.load_albedo(texture_path(model, model.skins[0]), built, model.texture_size)
    mat = material_map(built, albedo.shape[0])
    group = smoothing_groups(built)
    d = os.path.join(ART, "build", model.id)
    os.makedirs(d, exist_ok=True)
    for s in shots or ["rest"]:
        if s == "rest":
            pos = to_game(built.verts)
        else:
            name, sec = s.split("@")
            pos = to_game(skin_frame(built, model.skeleton, model.clips[name].pose(float(sec), model.skeleton)))
        img = render(pos, vertex_normals(pos, built.tris, group), built.uv, built.tris, albedo, mat, views)
        f = os.path.join(d, "render_" + s.replace("@", "_") + ".png")
        img.save(f)
        print("wrote", os.path.relpath(f, REPO))


def cmd_obj(mid, shots):
    from PIL import Image
    from modelc.writer import skin_frame, to_game
    model = load(mid)
    built = build(model)
    albedo, _ = texture.load_albedo(texture_path(model, model.skins[0]), built, model.texture_size)
    d = os.path.join(ART, "build", model.id)
    os.makedirs(d, exist_ok=True)
    Image.fromarray(albedo[..., :3]).save(os.path.join(d, "albedo.png"))
    with open(os.path.join(d, "model.mtl"), "w") as f:
        f.write("newmtl albedo\nKd 1 1 1\nmap_Kd albedo.png\n")
    shots = shots or ["rest"]
    for s in shots:
        if s == "rest":
            pos = to_game(built.verts)
        else:
            name, sec = s.split("@")
            pos = to_game(skin_frame(built, model.skeleton, model.clips[name].pose(float(sec), model.skeleton)))
        f = os.path.join(d, s.replace("@", "_") + ".obj")
        with open(f, "w") as fh:
            fh.write("mtllib model.mtl\nusemtl albedo\n")
            for p in pos:
                fh.write(f"v {p[0]:.5f} {p[1]:.5f} {p[2]:.5f}\n")
            for u, v in built.uv:
                fh.write(f"vt {u:.5f} {1 - v:.5f}\n")
            for a, b, c in built.tris + 1:
                fh.write(f"f {a}/{a} {b}/{b} {c}/{c}\n")
        print("wrote", os.path.relpath(f, REPO))


def cmd_watch(mid):
    def stamp():
        files = glob.glob(os.path.join(ART, "**", "*.py"), recursive=True) + \
            glob.glob(os.path.join(ART, "textures", "**", "*.png"), recursive=True) + \
            glob.glob(os.path.join(HERE, "*.py"))
        return max(os.path.getmtime(f) for f in files)
    last = 0
    print(f"watching {mid} (Ctrl+C to stop)")
    while True:
        s = stamp()
        if s != last:
            last = s
            try:
                # fresh import of the library each time, so edits to modelc itself apply too
                for k in [k for k in sys.modules if k == "modelc" or k.startswith("modelc.") or k.startswith("art")]:
                    del sys.modules[k]
                global build, texture, writer
                from modelc.model import build as b2
                from modelc import texture as t2, writer as w2
                build, texture, writer = b2, t2, w2
                cmd_build(mid)
            except Exception as e:  # keep watching after a mistake in a model file
                import traceback; traceback.print_exc()
        time.sleep(0.5)


def main(argv):
    if not argv or argv[0] in ("-h", "--help", "help"):
        print(__doc__); return
    cmd, rest = argv[0], argv[1:]
    if cmd == "list":
        for k, f in find_models().items():
            print(f"{k:24} {os.path.relpath(f, REPO)}")
    elif cmd == "build":
        ids = list(find_models()) if rest[:1] == ["--all"] else rest[:1]
        for i in ids:
            cmd_build(i)
    elif cmd == "watch":
        cmd_watch(rest[0])
    elif cmd == "template":
        cmd_template(rest[0], "--clean" in rest)
    elif cmd == "atlas":
        cmd_atlas(rest[0], "--lock" in rest)
    elif cmd == "obj":
        cmd_obj(rest[0], rest[1:])
    elif cmd == "fit":
        cmd_fit(rest[0], rest[1:])
    elif cmd == "render":
        cmd_render(rest[0], rest[1:])
    elif cmd == "paint":
        cmd_paint(rest[0], "--force" in rest)
    else:
        sys.exit(f"unknown command {cmd}\n{__doc__}")


if __name__ == "__main__":
    main(sys.argv[1:])
