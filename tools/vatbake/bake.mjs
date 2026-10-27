// vatbake: bakes skinned glTF characters into Vertex Animation Textures.
//
//   cd tools/vatbake && npm install && npm run bake            # all models
//   npm run bake -- swordsman                                   # one model
//
// For every model in models.json it writes frontend/public/models/<id>/:
//   model.json    layout + clips (see writeModel below)
//   geometry.bin  indices (LOD0, LOD1), vertex colors + team mask, bind-pose positions
//   vat.bin       per-frame vertex positions, RGBA half floats, row-major texture
//
// All materials are merged into one mesh with vertex colors, so a unit type is
// drawn with a single instanced draw call. Normals are not baked: the runtime
// uses flat shading (low-poly look), computed from screen-space derivatives.

import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import * as THREE from "three";
import { GLTFLoader } from "three/examples/jsm/loaders/GLTFLoader.js";
import { MeshoptSimplifier } from "meshoptimizer";
import { generators, buildClip } from "./clips.mjs";

const here = path.dirname(fileURLToPath(import.meta.url));
const repo = path.resolve(here, "../..");
const config = JSON.parse(fs.readFileSync(path.join(here, "models.json"), "utf8"));
const TEX_WIDTH = 2048;
const LOD1_RATIO = 0.4; // fraction of triangles kept for the mid LOD
const LOD2_RATIO = 0.1; // fraction kept for the far LOD (silhouette only)

const only = process.argv[2];
await MeshoptSimplifier.ready;

for (const model of config.models) {
  if (only && model.id !== only) continue;
  await bakeModel(model, config.fps);
}

async function bakeModel(model, fps) {
  const file = path.join(repo, model.source);
  const gltf = await loadGLB(file);
  const root = gltf.scene;
  root.updateMatrixWorld(true);

  const meshes = [];
  root.traverse((o) => o.isSkinnedMesh && meshes.push(o));
  if (meshes.length === 0) throw new Error(`${model.id}: no skinned meshes`);
  const bones = new Map(meshes[0].skeleton.bones.map((b) => [b.name, b]));

  // Rest pose = first frame of idle (procedural clips are offsets from it).
  const mixer = new THREE.AnimationMixer(root);
  const sourceClip = (name) => {
    const c = gltf.animations.find((a) => a.name === name);
    if (!c) throw new Error(`${model.id}: clip "${name}" not found (have: ${gltf.animations.map((a) => a.name)})`);
    return c;
  };
  poseAt(mixer, root, sourceClip(model.clips.idle.from), 0);
  const rest = new Map([...bones].map(([n, b]) => [n, { position: b.position.clone(), quaternion: b.quaternion.clone() }]));

  // Resolve clips.
  const clips = [];
  for (const [name, spec] of Object.entries(model.clips)) {
    let clip;
    if (spec.from.startsWith("@")) {
      const gen = generators[spec.from.slice(1)];
      if (!gen) throw new Error(`${model.id}: unknown generator ${spec.from}`);
      clip = buildClip(name, gen(), rest, bones, root);
    } else {
      clip = sourceClip(spec.from);
    }
    const frames = spec.loop ? Math.max(2, Math.round(clip.duration * fps)) : Math.round(clip.duration * fps) + 1;
    clips.push({ name, clip, loop: !!spec.loop, frames });
  }

  // Merge all primitives into one deduplicated vertex list.
  const geo = mergeMeshes(meshes, new Set(model.teamMaterials ?? []));
  const V = geo.sources.length;

  // Sample every frame of every clip.
  const totalFrames = clips.reduce((n, c) => n + c.frames, 0);
  const texHeight = Math.ceil((V * totalFrames) / TEX_WIDTH);
  const vat = new Uint16Array(TEX_WIDTH * texHeight * 4);
  const v = new THREE.Vector3();
  const bmin = new THREE.Vector3(Infinity, Infinity, Infinity);
  const bmax = new THREE.Vector3(-Infinity, -Infinity, -Infinity);
  let frame = 0;
  const clipMeta = {};
  const foot = bones.get("footL");
  const fp = new THREE.Vector3();
  for (const c of clips) {
    clipMeta[c.name] = { start: frame, frames: c.frames, loop: c.loop, duration: round(c.clip.duration) };
    let footMin = Infinity;
    let footMax = -Infinity;
    for (let f = 0; f < c.frames; f++, frame++) {
      const t = c.loop ? (f / c.frames) * c.clip.duration : Math.min(f / fps, c.clip.duration);
      poseAt(mixer, root, c.clip, t);
      if (foot) {
        foot.getWorldPosition(fp);
        footMin = Math.min(footMin, fp.z);
        footMax = Math.max(footMax, fp.z);
      }
      for (let i = 0; i < V; i++) {
        const { mesh, index } = geo.sources[i];
        mesh.getVertexPosition(index, v).applyMatrix4(mesh.matrixWorld);
        bmin.min(v);
        bmax.max(v);
        const o = (frame * V + i) * 4;
        vat[o] = THREE.DataUtils.toHalfFloat(v.x);
        vat[o + 1] = THREE.DataUtils.toHalfFloat(v.y);
        vat[o + 2] = THREE.DataUtils.toHalfFloat(v.z);
        vat[o + 3] = THREE.DataUtils.toHalfFloat(1);
      }
    }
    // In-place locomotion: a foot slides back by its stride during the half
    // cycle it is planted, so ground speed ≈ 2 * stride / duration. The runtime
    // scales playback by unitSpeed / groundSpeed so feet don't skate.
    if (c.loop && foot && (c.name === "walk" || c.name === "run")) {
      clipMeta[c.name].groundSpeed = round((2 * (footMax - footMin)) / c.clip.duration);
    }
  }

  // LOD1: simplified index buffer over the same vertices.
  const target = Math.floor((geo.indices.length * LOD1_RATIO) / 3) * 3;
  const [lod1] = MeshoptSimplifier.simplify(geo.indices, geo.bindPositions, 3, target, 0.02);
  // LOD2: aggressive; seen from far away only. Ignore per-material borders (colors may bleed slightly).
  const target2 = Math.floor((geo.indices.length * LOD2_RATIO) / 3) * 3;
  const [lod2] = MeshoptSimplifier.simplify(geo.indices, geo.bindPositions, 3, target2, 0.08);

  writeModel(model.id, { fps, V, totalFrames, texHeight, clipMeta, geo, lod1, lod2, vat, bmin, bmax });
  console.log(
    `${model.id}: ${V} vertices, ${geo.indices.length / 3} tris (LOD1 ${lod1.length / 3}, LOD2 ${lod2.length / 3}), ` +
      `${totalFrames} frames [${clips.map((c) => `${c.name}:${c.frames}`).join(" ")}], ` +
      `VAT ${TEX_WIDTH}x${texHeight} (${(vat.byteLength / 1024 / 1024).toFixed(1)} MB)`,
  );
}

function poseAt(mixer, root, clip, t) {
  mixer.stopAllAction();
  const action = mixer.clipAction(clip);
  action.reset();
  action.setLoop(THREE.LoopOnce, 1);
  action.clampWhenFinished = true;
  action.play();
  action.time = t;
  mixer.update(0);
  root.updateMatrixWorld(true);
}

/** Merges skinned primitives into one vertex list, deduplicating vertices that
 *  differ only in normal (we shade flat). Keeps a pointer to each vertex's
 *  source mesh/index so it can be skinned per frame. */
function mergeMeshes(meshes, teamMaterials) {
  const sources = [];
  const colors = [];
  const bindPositions = [];
  const indices = [];
  const seen = new Map();
  const p = new THREE.Vector3();

  for (const mesh of meshes) {
    const g = mesh.geometry;
    const pos = g.attributes.position;
    const si = g.attributes.skinIndex;
    const sw = g.attributes.skinWeight;
    const c = mesh.material.color.clone().convertLinearToSRGB();
    const team = teamMaterials.has(mesh.material.name) ? 255 : 0;
    const rgb = [c.r, c.g, c.b].map((x) => Math.round(x * 255));
    const remap = new Uint32Array(pos.count);

    for (let i = 0; i < pos.count; i++) {
      p.fromBufferAttribute(pos, i).applyMatrix4(mesh.matrixWorld);
      const key = [
        p.x.toFixed(5), p.y.toFixed(5), p.z.toFixed(5),
        si.getX(i), si.getY(i), si.getZ(i), si.getW(i),
        sw.getX(i).toFixed(3), sw.getY(i).toFixed(3), sw.getZ(i).toFixed(3), sw.getW(i).toFixed(3),
        ...rgb, team,
      ].join("|");
      let id = seen.get(key);
      if (id === undefined) {
        id = sources.length;
        seen.set(key, id);
        sources.push({ mesh, index: i });
        colors.push(...rgb, team);
        bindPositions.push(p.x, p.y, p.z);
      }
      remap[i] = id;
    }
    const idx = g.index;
    const n = idx ? idx.count : pos.count;
    for (let k = 0; k < n; k++) indices.push(remap[idx ? idx.getX(k) : k]);
  }
  return {
    sources,
    colors: new Uint8Array(colors),
    bindPositions: new Float32Array(bindPositions),
    indices: new Uint32Array(indices),
  };
}

/**
 * model.json:
 * {
 *   version, fps, vertexCount, frames,
 *   texture: { width, height, format: "rgba16f" },  // texel (frame*vertexCount + vertex) at (i % width, i / width)
 *   clips: { name: { start, frames, loop, duration, groundSpeed? } },  // groundSpeed: m/s at 1x (walk/run)
 *   bounds: { min: [x,y,z], max: [x,y,z] },          // over all frames
 *   geometry: { lod0, lod1, lod2: {offset, count}, colors: {offset}, positions: {offset} }  // byte offsets into geometry.bin
 * }
 */
function writeModel(id, d) {
  const dir = path.join(repo, "frontend/public/models", id);
  fs.mkdirSync(dir, { recursive: true });

  const lod0Bytes = d.geo.indices.byteLength;
  const lod1Bytes = d.lod1.byteLength;
  const lod2Bytes = d.lod2.byteLength;
  const colorsOffset = lod0Bytes + lod1Bytes + lod2Bytes;
  const positionsOffset = colorsOffset + d.geo.colors.byteLength;
  const geometry = Buffer.concat([
    Buffer.from(d.geo.indices.buffer, d.geo.indices.byteOffset, lod0Bytes),
    Buffer.from(d.lod1.buffer, d.lod1.byteOffset, lod1Bytes),
    Buffer.from(d.lod2.buffer, d.lod2.byteOffset, lod2Bytes),
    Buffer.from(d.geo.colors.buffer, d.geo.colors.byteOffset, d.geo.colors.byteLength),
    Buffer.from(d.geo.bindPositions.buffer, d.geo.bindPositions.byteOffset, d.geo.bindPositions.byteLength),
  ]);

  const meta = {
    version: 1,
    fps: d.fps,
    vertexCount: d.V,
    frames: d.totalFrames,
    texture: { width: TEX_WIDTH, height: d.texHeight, format: "rgba16f" },
    clips: d.clipMeta,
    bounds: { min: d.bmin.toArray().map(round), max: d.bmax.toArray().map(round) },
    geometry: {
      lod0: { offset: 0, count: d.geo.indices.length },
      lod1: { offset: lod0Bytes, count: d.lod1.length },
      lod2: { offset: lod0Bytes + lod1Bytes, count: d.lod2.length },
      colors: { offset: colorsOffset },
      positions: { offset: positionsOffset },
    },
  };
  fs.writeFileSync(path.join(dir, "model.json"), JSON.stringify(meta, null, 2) + "\n");
  fs.writeFileSync(path.join(dir, "geometry.bin"), geometry);
  fs.writeFileSync(path.join(dir, "vat.bin"), Buffer.from(d.vat.buffer));
}

function loadGLB(file) {
  const buf = fs.readFileSync(file);
  const ab = buf.buffer.slice(buf.byteOffset, buf.byteOffset + buf.byteLength);
  return new Promise((resolve, reject) => new GLTFLoader().parse(ab, "", resolve, reject));
}

function round(x) {
  return Math.round(x * 1000) / 1000;
}
