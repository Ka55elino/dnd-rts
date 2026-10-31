"""Exports one unit skin to the .glb that tools/vatbake reads.

Layout (docs/ASSETS.md, "Unit source layout"):
    art/source/<faction>/units/<unit>/
        lowpoly/<skin>.blend         mesh + UV + weights, rig linked from ../rig/rig.blend
        animations/animations.blend  all clips (actions), animated on an override of the rig
        <skin>.glb                   ← written here

Run headless from the repo root:
    blender -b art/source/empire/units/swordsman/lowpoly/central.blend -P tools/blender/export_unit.py

or inside Blender with the lowpoly file open (Scripting tab → Run Script); the open
file is reverted afterwards, nothing is saved.
"""
import os

import bpy

blend = bpy.data.filepath
if not blend or os.path.basename(os.path.dirname(blend)) != "lowpoly":
    raise SystemExit("open art/source/<faction>/units/<unit>/lowpoly/<skin>.blend first")
unit_dir = os.path.dirname(os.path.dirname(blend))
skin = os.path.splitext(os.path.basename(blend))[0]
anim_file = os.path.join(unit_dir, "animations", "animations.blend")
out = os.path.join(unit_dir, skin + ".glb")

# 1. The rig is linked from rig/rig.blend: make it local so actions can be assigned.
rigs = [o for o in bpy.data.objects if o.type == "ARMATURE" and o.users_scene]
if len(rigs) != 1:
    raise SystemExit(f"expected one armature in the scene, found {[o.name for o in rigs]}")
rig = rigs[0]
meshes = [o for o in bpy.data.objects if o.type == "MESH" and o.users_scene and o.find_armature() is not None]
if rig.library:
    # Never re-parent by hand: that loses matrix_parent_inverse (the rig is scaled 0.01)
    # and the exported mesh ends up 100x too small. Let Blender remap the users instead.
    linked = rig
    rig = linked.make_local()
    if rig is not linked:
        linked.user_remap(rig)
    if rig.data.library:
        rig.data = rig.data.make_local()
for m in meshes:
    if m.parent is not rig or not any(mod.type == "ARMATURE" and mod.object is rig for mod in m.modifiers):
        raise SystemExit(f"{m.name}: not bound to the local rig after make_local")

# 2. Clips from animations/animations.blend (every action with a fake user).
with bpy.data.libraries.load(anim_file, link=False) as (src, dst):
    dst.actions = list(src.actions)
clips = [a for a in dst.actions if a is not None]
if not clips:
    raise SystemExit(f"no actions in {anim_file}")
rig.animation_data_create()
rig.animation_data.action = clips[0]
if hasattr(rig.animation_data, "action_suitable_slots") and rig.animation_data.action_suitable_slots:
    rig.animation_data.action_slot = rig.animation_data.action_suitable_slots[0]
for p in rig.pose.bones:
    p.matrix_basis.identity()

# 3. Export rig + skinned meshes with every action as a glTF animation.
bpy.ops.object.select_all(action="DESELECT")
for o in [rig, *meshes]:
    o.select_set(True)
bpy.context.view_layer.objects.active = rig
props = bpy.ops.export_scene.gltf.get_rna_type().properties.keys()
kw = dict(filepath=out, export_format="GLB", use_selection=True, export_animations=True,
          export_skins=True, export_yup=True, export_apply=False)
if "export_animation_mode" in props:
    kw["export_animation_mode"] = "ACTIONS"
if "export_image_format" in props:
    kw["export_image_format"] = "NONE"  # the albedo goes to vatbake as textures/<skin>_albedo.png
bpy.ops.export_scene.gltf(**kw)
print(f"exported {out}: {len(meshes)} mesh(es), clips {[a.name for a in clips]}")

if not bpy.app.background:
    bpy.ops.wm.revert_mainfile()
