// meshbake: prepares static glTF models (buildings, props) for the game.
//
//   cd tools/meshbake && npm install && npm run bake      # all models
//   npm run bake -- keep                                   # one model
//
// For every model in models.json it writes frontend/public/models/<id>/:
//   lod0.glb, lod1.glb, lod2.glb   same materials and textures, fewer triangles
//                                  (the runtime shares materials between LODs by name)
//   model.json                     bounds and triangle counts per LOD
//
// Each LOD: flatten the node tree, join primitives that share a material
// (fewer draw calls), weld vertices, then simplify with meshoptimizer.

import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { Logger, NodeIO } from "@gltf-transform/core";
import { ALL_EXTENSIONS } from "@gltf-transform/extensions";
import { dedup, flatten, join, prune, simplify, weld, getBounds } from "@gltf-transform/functions";
import { MeshoptSimplifier } from "meshoptimizer";

const here = path.dirname(fileURLToPath(import.meta.url));
const repo = path.resolve(here, "../..");
const config = JSON.parse(fs.readFileSync(path.join(here, "models.json"), "utf8"));
const only = process.argv[2];

await MeshoptSimplifier.ready;
const io = new NodeIO().registerExtensions(ALL_EXTENSIONS).setLogger(new Logger(Logger.Verbosity.WARN));

for (const model of config.models) {
  if (only && model.id !== only) continue;
  await bakeModel(model);
}

async function bakeModel(model) {
  const out = path.join(repo, "frontend/public/models", model.id);
  fs.mkdirSync(out, { recursive: true });
  const meta = { version: 1, lods: [], bounds: null };

  // Per-model overrides: "lods" (ratios) and "error" (max deviation for LOD1+;
  // repeated props like wall segments need much coarser far LODs).
  const lods = model.lods ?? config.lods;
  for (const [level, ratio] of lods.entries()) {
    const doc = await io.read(path.join(repo, model.source));
    doc.setLogger(new Logger(Logger.Verbosity.WARN));
    // keepNamed: named nodes stay separate meshes (moving parts like gate leaves).
    await doc.transform(dedup(), flatten(), join({ keepNamed: !!model.keepNamed }), weld());
    if (ratio < 1) {
      // error: max deviation relative to the mesh size; looser for farther LODs.
      const error = model.error ?? (ratio > 0.2 ? 0.02 : 0.2);
      if (model.sloppy) sloppy(doc, ratio, error);
      else await doc.transform(simplify({ simplifier: MeshoptSimplifier, ratio, error, lockBorder: false }));
    }
    await doc.transform(prune());

    const scene = doc.getRoot().getDefaultScene() ?? doc.getRoot().listScenes()[0];
    if (level === 0) {
      const { min, max } = getBounds(scene);
      meta.bounds = { min: min.map(r3), max: max.map(r3) };
    }
    const tris = triangles(doc);
    meta.lods.push({ file: `lod${level}.glb`, triangles: tris });
    await io.write(path.join(out, `lod${level}.glb`), doc);
    console.log(`${model.id} lod${level}: ${tris} triangles, ${(fs.statSync(path.join(out, `lod${level}.glb`)).size / 1024).toFixed(0)} KB`);
  }
  fs.writeFileSync(path.join(out, "model.json"), JSON.stringify(meta, null, 2) + "\n");
  const { min, max } = meta.bounds;
  console.log(`${model.id}: size ${r3(max[0] - min[0])} x ${r3(max[2] - min[2])}, height ${r3(max[1] - min[1])}`);
}

function triangles(doc) {
  let n = 0;
  for (const mesh of doc.getRoot().listMeshes()) {
    for (const p of mesh.listPrimitives()) {
      const idx = p.getIndices();
      n += (idx ? idx.getCount() : p.getAttribute("POSITION").getCount()) / 3;
    }
  }
  return n;
}

function r3(v) {
  return Math.round(v * 1000) / 1000;
}

// sloppy: seam-ignoring simplification per primitive. Vertices are welded by
// position only (flat shading and UV islands split them and stop the regular
// simplifier), the welded mesh is simplified, and each kept corner maps back to
// one original vertex. UVs may smear a little: fine for far LODs of props.
function sloppy(doc, ratio, error) {
  for (const mesh of doc.getRoot().listMeshes()) {
    for (const prim of mesh.listPrimitives()) {
      const idx = prim.getIndices();
      const pos = prim.getAttribute("POSITION");
      if (!idx || !pos) continue;
      const src = pos.getArray();
      const ids = new Map();
      const remap = new Uint32Array(pos.getCount());
      const rep = [];
      const welded = [];
      for (let v = 0; v < pos.getCount(); v++) {
        const key = `${Math.round(src[v * 3] * 1e4)},${Math.round(src[v * 3 + 1] * 1e4)},${Math.round(src[v * 3 + 2] * 1e4)}`;
        let w = ids.get(key);
        if (w === undefined) {
          w = rep.length;
          ids.set(key, w);
          rep.push(v);
          welded.push(src[v * 3], src[v * 3 + 1], src[v * 3 + 2]);
        }
        remap[v] = w;
      }
      const indices = Uint32Array.from(idx.getArray(), (v) => remap[v]);
      const target = Math.max(3, Math.floor((indices.length * ratio) / 3) * 3);
      const [out] = MeshoptSimplifier.simplify(indices, new Float32Array(welded), 3, target, error, []);
      idx.setArray(Uint32Array.from(out, (w) => rep[w]));
    }
  }
}
