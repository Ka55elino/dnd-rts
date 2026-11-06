"""Writes a static model as the game reads it (the same output as tools/meshbake):
frontend/public/models/<id>/lod0.glb, lod1.glb, lod2.glb + model.json.

LOD0 keeps every part, LOD1 the parts with detail 0-1, LOD2 only detail 0.
Static parts are merged into one mesh (one primitive per material); every named node (a moving part)
is its own node at its pivot. Flat shading: every triangle has its own vertices. Materials are named
by colour and kind, so the game shares them between LODs by name; "team" parts are tinted per owner.
"""
import json
import os
import struct

import numpy as np

LOD_KEEP = [2, 1, 0]  # max detail per LOD


def _material(name, part):
    r, g, b = part.color
    m = {"name": name, "pbrMetallicRoughness": {"baseColorFactor": [r, g, b, 1.0], "metallicFactor": 0.0, "roughnessFactor": 0.85}}
    if part.kind == "metal":
        m["pbrMetallicRoughness"].update(metallicFactor=0.55, roughnessFactor=0.45)
    elif part.kind == "glow":
        m["emissiveFactor"] = [r, g, b]
    elif part.kind == "team":
        m["pbrMetallicRoughness"]["baseColorFactor"] = [0.8, 0.8, 0.8, 1.0]
    return m


def _flat(parts, origin):
    """Merged flat-shaded triangle soup of parts (positions relative to origin)."""
    pos, nrm = [], []
    for p in parts:
        v = p.verts - origin
        a, b, c = v[p.tris[:, 0]], v[p.tris[:, 1]], v[p.tris[:, 2]]
        n = np.cross(b - a, c - a)
        ln = np.linalg.norm(n, axis=1, keepdims=True)
        keep = ln[:, 0] > 1e-10
        a, b, c, n = a[keep], b[keep], c[keep], n[keep] / ln[keep]
        tri = np.stack([a, b, c], axis=1).reshape(-1, 3)
        pos.append(tri)
        nrm.append(np.repeat(n, 3, axis=0))
    if not pos:
        return None, None
    return np.concatenate(pos).astype(np.float32), np.concatenate(nrm).astype(np.float32)


def glb(parts):
    """A glb (bytes) of the parts and its triangle count."""
    bin_ = bytearray()
    gl = {"asset": {"version": "2.0", "generator": "propc"}, "scenes": [{"nodes": []}], "scene": 0,
          "nodes": [], "meshes": [], "materials": [], "accessors": [], "bufferViews": [], "buffers": []}
    mats = {}

    def mat(p):
        name = p.material
        if name not in mats:
            mats[name] = len(gl["materials"])
            gl["materials"].append(_material(name, p))
        return mats[name]

    def view(data, target):
        while len(bin_) % 4:
            bin_.append(0)
        off = len(bin_)
        bin_.extend(data)
        gl["bufferViews"].append({"buffer": 0, "byteOffset": off, "byteLength": len(data), "target": target})
        return len(gl["bufferViews"]) - 1

    def accessor(arr, kind, minmax=False):
        a = {"bufferView": view(arr.tobytes(), 34962), "componentType": 5126, "count": len(arr), "type": kind}
        if minmax:
            a["min"] = arr.min(0).tolist()
            a["max"] = arr.max(0).tolist()
        gl["accessors"].append(a)
        return len(gl["accessors"]) - 1

    tris = 0
    groups = {}
    for p in parts:
        groups.setdefault(p.node, []).append(p)
    for node, ps in sorted(groups.items(), key=lambda kv: (kv[0] is not None, kv[0] or "")):
        origin = np.array(ps[0].pivot if node and ps[0].pivot is not None else (0, 0, 0), float)
        prims = []
        by_mat = {}
        for p in ps:
            by_mat.setdefault(p.material, []).append(p)
        for name, mp in sorted(by_mat.items()):
            pos, nrm = _flat(mp, origin)
            if pos is None:
                continue
            tris += len(pos) // 3
            prims.append({"attributes": {"POSITION": accessor(pos, "VEC3", True), "NORMAL": accessor(nrm, "VEC3")},
                          "material": mat(mp[0])})
        if not prims:
            continue
        gl["meshes"].append({"name": node or "body", "primitives": prims})
        n = {"name": node or "body", "mesh": len(gl["meshes"]) - 1}
        if origin.any():
            n["translation"] = origin.tolist()
        gl["nodes"].append(n)
        gl["scenes"][0]["nodes"].append(len(gl["nodes"]) - 1)
    while len(bin_) % 4:
        bin_.append(0)
    gl["buffers"].append({"byteLength": len(bin_)})
    js = json.dumps(gl, separators=(",", ":")).encode()
    while len(js) % 4:
        js += b" "
    out = struct.pack("<III", 0x46546C67, 2, 12 + 8 + len(js) + 8 + len(bin_))
    out += struct.pack("<II", len(js), 0x4E4F534A) + js
    out += struct.pack("<II", len(bin_), 0x004E4942) + bytes(bin_)
    return out, tris


def write(prop, out_root):
    d = os.path.join(out_root, prop.id)
    os.makedirs(d, exist_ok=True)
    for f in os.listdir(d):
        os.remove(os.path.join(d, f))
    meta = {"version": 1, "lods": [], "bounds": None}
    allv = np.concatenate([p.verts for p in prop.parts])
    meta["bounds"] = {"min": [round(float(x), 3) for x in allv.min(0)], "max": [round(float(x), 3) for x in allv.max(0)]}
    for level, keep in enumerate(LOD_KEEP):
        parts = [p for p in prop.parts if p.detail <= keep]
        data, tris = glb(parts)
        name = f"lod{level}.glb"
        open(os.path.join(d, name), "wb").write(data)
        meta["lods"].append({"file": name, "triangles": int(tris)})
    meta["build"] = int(os.path.getmtime(os.path.join(d, "lod0.glb")) * 1000)
    with open(os.path.join(d, "model.json"), "w") as f:
        f.write(json.dumps(meta, indent=2) + "\n")
    return meta


def update_index(models_root, ids):
    """Marks ids as static models in frontend/public/models/index.json (Model Overview reads it)."""
    f = os.path.join(models_root, "index.json")
    try:
        data = json.load(open(f))
    except Exception:
        data = {"models": []}
    models = [m for m in data.get("models", []) if m["id"] not in ids and os.path.exists(os.path.join(models_root, m["id"], "model.json"))]
    models += [{"id": i, "type": "static"} for i in ids if os.path.exists(os.path.join(models_root, i, "model.json"))]
    out = {"$comment": "Generated by tools/vatbake, tools/meshbake, tools/modelc and tools/propc: every baked model. "
                       "vat = animated (VatModel), static = LODs (StaticModel).",
           "models": sorted(models, key=lambda m: m["id"])}
    with open(f, "w") as fh:
        fh.write(json.dumps(out, indent=2) + "\n")
