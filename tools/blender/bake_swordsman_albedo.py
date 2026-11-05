"""Bakes the Empire swordsman albedo from the high-poly (Meshy) onto the low-poly UVs.

Run inside Blender with art/source/empire/units/swordsman/lowpoly/central.blend open:
    exec(open("<repo>/tools/blender/bake_swordsman_albedo.py").read())
Writes textures/central_albedo_meshy_1024.png (RGB only). The file is reverted afterwards — nothing is saved.
Then: compose the 512 albedo with the team mask (alpha) and run vatbake (see the end of this file).

The high-poly stands in a different pose, so a temporary copy of the low-poly is matched to it:
the arms are posed with the rig, the sword and the shield are moved rigidly onto the high-poly's.
The landmarks below were measured on central_high.glb.
"""
import bpy, bmesh
import numpy as np
from mathutils import Vector, Matrix

UNIT = "/Users/denysromanenko/Projects/dnd-rts/art/source/empire/units/swordsman"
HIGH = UNIT + "/central_high.glb"
OUT = UNIT + "/textures/central_albedo_meshy_1024.png"

# --- 1. high-poly: import, scale to the low-poly height, align -------------------------------------
before = set(bpy.data.objects)
bpy.ops.import_scene.gltf(filepath=HIGH)
high = [o for o in bpy.data.objects if o not in before and o.type == "MESH"][0]
k = 2.06 / 1.899
high.scale = (k, k, k)
high.location = (0, -0.17, 0.951 * k)
# metals have almost no diffuse color in PBR: bake the base color as if non-metallic
for m in high.data.materials:
    b = next(n for n in m.node_tree.nodes if n.type == "BSDF_PRINCIPLED")
    for l in list(b.inputs["Metallic"].links):
        m.node_tree.links.remove(l)
    b.inputs["Metallic"].default_value = 0.0
bpy.context.view_layer.update()

# --- 2. pose the arms like the high-poly --------------------------------------------------------------
rig = bpy.data.objects["rig"]
if rig.library:
    loc = rig.make_local()
    if loc is not rig:
        rig.user_remap(loc)
    rig = bpy.data.objects["rig"]
    if rig.data.library:
        rig.data = rig.data.make_local()
W = rig.matrix_world
Wi = W.inverted()


def aim(name, target):
    pb = rig.pose.bones[name]
    bpy.context.view_layer.update()
    cur = (Wi.to_3x3() @ ((W @ pb.tail) - (W @ pb.head))).normalized()
    want = (Wi.to_3x3() @ (Vector(target) - (W @ pb.head))).normalized()
    q = cur.rotation_difference(want)
    h = pb.head.copy()
    pb.matrix = Matrix.Translation(h) @ q.to_matrix().to_4x4() @ Matrix.Translation(-h) @ pb.matrix
    bpy.context.view_layer.update()


for pb in rig.pose.bones:
    pb.matrix_basis.identity()
aim("mixamorig:RightArm", (-0.45, 0.10, 1.22))
aim("mixamorig:RightForeArm", (-0.38, -0.03, 0.88))
aim("mixamorig:LeftArm", (0.36, 0.10, 1.18))
aim("mixamorig:LeftForeArm", (0.47, 0.0, 0.98))

# --- 3. posed copy of the low-poly; move the sword and the shield onto the high-poly's --------------
lp = bpy.data.objects["swordsman"]
dg = bpy.context.evaluated_depsgraph_get()
me = bpy.data.meshes.new_from_object(lp.evaluated_get(dg), depsgraph=dg)
tmp = bpy.data.objects.new("swordsman_bake", me)
bpy.context.scene.collection.objects.link(tmp)
bm = bmesh.new(); bm.from_mesh(me); bm.verts.ensure_lookup_table()
seen, parts = set(), []
for v in bm.verts:
    if v.index in seen:
        continue
    st, comp = [v], []
    seen.add(v.index)
    while st:
        x = st.pop(); comp.append(x.index)
        for e in x.link_edges:
            o = e.other_vert(x)
            if o.index not in seen:
                seen.add(o.index); st.append(o)
    parts.append(comp)
SWORD, GUARD, SHIELD = parts[28], parts[29], parts[30]
assert (len(SWORD), len(GUARD), len(SHIELD)) == (32, 12, 20), "low-poly topology changed: re-check the part indices"


def frame(a, b):
    a = a / np.linalg.norm(a); b = b - a * (b @ a); b /= np.linalg.norm(b)
    return np.stack([a, b, np.cross(a, b)], 1)


def pts(idx):
    return np.array([tuple(bm.verts[i].co) for i in idx])


def move(idx, R, src, dst):
    for i in idx:
        bm.verts[i].co = Vector(R @ (np.array(bm.verts[i].co) - src) + dst)


S, G = pts(SWORD), pts(GUARD)
gc = G.mean(0)
a_l = np.linalg.svd(S - S.mean(0))[2][0]
if abs(((S - gc) @ a_l).min()) > abs(((S - gc) @ a_l).max()):
    a_l = -a_l
w_l = np.linalg.svd(G - gc)[2][0]
ax = np.array([-0.61, 0.616, 0.499]); c = np.array([0.064, -0.478, 0.498]); wh = np.array([-0.383, 0.322, -0.866])
move(SWORD + GUARD, frame(-ax, wh) @ frame(a_l, w_l).T, gc, c + 0.62 * ax)

P = pts(SHIELD); cl = P.mean(0); vt = np.linalg.svd(P - cl)[2]
long_l, n_l = vt[0], vt[2]
if n_l @ (cl - np.array([0, 0, cl[2]])) < 0:
    n_l = -n_l
pr = (P - cl) @ long_l
wide = np.linalg.norm((P - cl) - np.outer(pr, long_l), axis=1)
up_l = -long_l if wide[pr < 0].mean() > wide[pr > 0].mean() else long_l
cen_l = cl + long_l * (pr.min() + pr.max()) / 2
long_h = np.array([0.071, 0.075, 0.995]); n_h = np.array([0.454, -0.89, 0.035]); n_h /= np.linalg.norm(n_h)
move(SHIELD, frame(long_h, n_h) @ frame(up_l, n_l).T, cen_l, np.array([0.449, 0.04, 1.042]))
bm.to_mesh(me); bm.free(); me.update()

# --- 4. bake the base color, selected (high) to active (posed low-poly copy) ------------------------
sc = bpy.context.scene
sc.render.engine = "CYCLES"
sc.cycles.samples = 4
img = bpy.data.images.new("bake_albedo", 1024, 1024, alpha=True)
for m in tmp.data.materials:
    n = m.node_tree.nodes.new("ShaderNodeTexImage"); n.image = img
    m.node_tree.nodes.active = n
for o in bpy.context.selected_objects:
    o.select_set(False)
high.select_set(True); tmp.select_set(True)
bpy.context.view_layer.objects.active = tmp
bpy.ops.object.bake(type="DIFFUSE", pass_filter={"COLOR"}, use_selected_to_active=True,
                    cage_extrusion=0.04, max_ray_distance=0.12, margin=8)
img.filepath_raw = OUT; img.file_format = "PNG"; img.save()
print("baked", OUT)
bpy.ops.wm.revert_mainfile()

# --- 5. afterwards (outside Blender) --------------------------------------------------------------------
#   python: RGB of OUT resized to 512 + alpha (team mask) of the previous albedo → textures/central_albedo.png
#   cd tools/vatbake && npm run bake -- swordsman
