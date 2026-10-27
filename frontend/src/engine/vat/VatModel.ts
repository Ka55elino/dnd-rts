import * as THREE from "three";

/** Level of detail: 0 = full mesh, 1 = ~40% triangles, 2 = ~10% (far away). */
export type Lod = 0 | 1 | 2;

/** One animation clip inside a VAT texture (see tools/vatbake). */
export interface VatClip {
    start: number; // first frame
    frames: number;
    loop: boolean;
    duration: number; // seconds at 1x
    groundSpeed?: number; // m/s at 1x (walk/run)
}

interface VatMeta {
    version: number;
    fps: number;
    vertexCount: number;
    frames: number;
    texture: { width: number; height: number; format: "rgba16f" };
    clips: Record<string, VatClip>;
    bounds: { min: [number, number, number]; max: [number, number, number] };
    geometry: {
        lod0: { offset: number; count: number };
        lod1: { offset: number; count: number };
        lod2?: { offset: number; count: number };
        colors: { offset: number };
        positions: { offset: number };
    };
}

/**
 * A character baked by tools/vatbake: one merged mesh (vertex colors + team
 * mask) and a texture holding every vertex position for every animation frame.
 */
export class VatModel {
    readonly geometry: THREE.BufferGeometry;
    readonly texture: THREE.DataTexture;
    /** Index buffers per level of detail (0 = full). Missing levels fall back to the previous one. */
    readonly lodIndex: THREE.BufferAttribute[];
    readonly clips: Record<string, VatClip>;

    private constructor(readonly id: string, readonly meta: VatMeta, geometry: ArrayBuffer, vat: ArrayBuffer) {
        this.clips = meta.clips;
        const g = meta.geometry;
        const V = meta.vertexCount;

        const geo = new THREE.BufferGeometry();
        geo.setIndex(new THREE.BufferAttribute(new Uint32Array(geometry, g.lod0.offset, g.lod0.count), 1));
        const lod1 = new THREE.BufferAttribute(new Uint32Array(geometry, g.lod1.offset, g.lod1.count), 1);
        const lod2 = g.lod2 ? new THREE.BufferAttribute(new Uint32Array(geometry, g.lod2.offset, g.lod2.count), 1) : lod1;
        this.lodIndex = [geo.index!, lod1, lod2];
        // Bind-pose positions: only used for bounds; the shader reads the VAT instead.
        geo.setAttribute("position", new THREE.BufferAttribute(new Float32Array(geometry, g.positions.offset, V * 3), 3));
        // RGBA bytes: rgb = color, a = team mask (255 = tinted with the player color).
        const rgba = new THREE.InterleavedBuffer(new Uint8Array(geometry, g.colors.offset, V * 4), 4);
        geo.setAttribute("color", new THREE.InterleavedBufferAttribute(rgba, 3, 0, true));
        geo.setAttribute("teamMask", new THREE.InterleavedBufferAttribute(rgba, 1, 3, true));
        const { min, max } = meta.bounds;
        geo.boundingBox = new THREE.Box3(new THREE.Vector3(...min), new THREE.Vector3(...max));
        geo.boundingSphere = geo.boundingBox.getBoundingSphere(new THREE.Sphere());
        this.geometry = geo;

        const t = meta.texture;
        this.texture = new THREE.DataTexture(new Uint16Array(vat), t.width, t.height, THREE.RGBAFormat, THREE.HalfFloatType);
        this.texture.minFilter = THREE.NearestFilter;
        this.texture.magFilter = THREE.NearestFilter;
        this.texture.generateMipmaps = false;
        this.texture.needsUpdate = true;
    }

    /** Loads frontend/public/models/<id>/. */
    static async load(id: string): Promise<VatModel> {
        const base = `/models/${id}`;
        const [meta, geometry, vat] = await Promise.all([
            fetch(`${base}/model.json`).then((r) => checked(r, id).json() as Promise<VatMeta>),
            fetch(`${base}/geometry.bin`).then((r) => checked(r, id).arrayBuffer()),
            fetch(`${base}/vat.bin`).then((r) => checked(r, id).arrayBuffer()),
        ]);
        if (meta.version !== 1) throw new Error(`model ${id}: unsupported version ${meta.version}`);
        return new VatModel(id, meta, geometry, vat);
    }

    clip(name: string): VatClip | undefined {
        return this.clips[name];
    }

    dispose(): void {
        this.geometry.dispose();
        this.texture.dispose();
    }
}

function checked(r: Response, id: string): Response {
    if (!r.ok) throw new Error(`model ${id}: ${r.url} → ${r.status}`);
    return r;
}

/**
 * Draws many instances of a VAT model in one call. Per instance: transform,
 * clip, clip start time, playback rate and team color. Animation is computed
 * entirely in the vertex shader.
 */
export class VatRenderer {
    readonly mesh: THREE.InstancedMesh;
    private readonly anim: THREE.InstancedBufferAttribute; // start frame, frame count, fps * rate, loop
    private readonly animStart: THREE.InstancedBufferAttribute; // seconds
    private readonly team: THREE.InstancedBufferAttribute; // rgb
    private readonly uniforms: { uTime: { value: number } };
    private count = 0;

    constructor(
        readonly model: VatModel,
        private readonly capacity: number,
        lod: Lod = 0,
    ) {
        // Share the model's buffers (no clone: cloning de-interleaves the color
        // attribute), but give this renderer its own instanced attributes.
        const geometry = new THREE.BufferGeometry();
        const src = model.geometry;
        geometry.setIndex(model.lodIndex[lod]);
        for (const [name, attr] of Object.entries(src.attributes)) geometry.setAttribute(name, attr);
        geometry.boundingBox = src.boundingBox;
        geometry.boundingSphere = src.boundingSphere;
        this.anim = new THREE.InstancedBufferAttribute(new Float32Array(capacity * 4), 4).setUsage(THREE.DynamicDrawUsage);
        this.animStart = new THREE.InstancedBufferAttribute(new Float32Array(capacity), 1).setUsage(THREE.DynamicDrawUsage);
        this.team = new THREE.InstancedBufferAttribute(new Float32Array(capacity * 3), 3).setUsage(THREE.DynamicDrawUsage);
        geometry.setAttribute("aAnim", this.anim);
        geometry.setAttribute("aAnimStart", this.animStart);
        geometry.setAttribute("aTeam", this.team);

        const { material, uniforms } = createVatMaterial(model);
        this.uniforms = uniforms;
        this.mesh = new THREE.InstancedMesh(geometry, material, capacity);
        this.mesh.instanceMatrix.setUsage(THREE.DynamicDrawUsage);
        this.mesh.count = 0;
        this.mesh.frustumCulled = false;
    }

    begin(): void {
        this.count = 0;
    }

    /**
     * Adds an instance. matrix: 16 floats (column-major) written by the caller via `matrixSlot`.
     * Returns the instance index, or -1 when full.
     */
    add(clip: VatClip, startSec: number, rate: number, team: THREE.Color): number {
        if (this.count >= this.capacity) return -1;
        const i = this.count++;
        const a = this.anim.array as Float32Array;
        a[i * 4] = clip.start;
        a[i * 4 + 1] = clip.frames;
        a[i * 4 + 2] = this.model.meta.fps * rate;
        a[i * 4 + 3] = clip.loop ? 1 : 0;
        (this.animStart.array as Float32Array)[i] = startSec;
        team.toArray(this.team.array as Float32Array, i * 3);
        return i;
    }

    /**
     * Sets instance i's animation directly (for callers that manage instance
     * slots themselves, e.g. the benchmark). Call commitAnimation() afterwards.
     */
    setInstance(i: number, clip: VatClip, startSec: number, rate: number, team: THREE.Color): void {
        const a = this.anim.array as Float32Array;
        a[i * 4] = clip.start;
        a[i * 4 + 1] = clip.frames;
        a[i * 4 + 2] = this.model.meta.fps * rate;
        a[i * 4 + 3] = clip.loop ? 1 : 0;
        (this.animStart.array as Float32Array)[i] = startSec;
        team.toArray(this.team.array as Float32Array, i * 3);
    }

    commitAnimation(): void {
        this.anim.needsUpdate = true;
        this.animStart.needsUpdate = true;
        this.team.needsUpdate = true;
    }

    /** Advances the animation clock without touching instance data. */
    setTime(nowSec: number): void {
        this.uniforms.uTime.value = nowSec;
    }

    /** The instance matrix array; instance i occupies [i * 16, i * 16 + 16). */
    get matrices(): Float32Array {
        return this.mesh.instanceMatrix.array as Float32Array;
    }

    end(nowSec: number): void {
        const n = this.count;
        this.mesh.count = n;
        this.uniforms.uTime.value = nowSec;
        // Upload only the used part of each buffer (capacity can be far larger).
        for (const attr of [this.mesh.instanceMatrix, this.anim, this.animStart, this.team]) {
            attr.clearUpdateRanges();
            if (n > 0) attr.addUpdateRange(0, n * attr.itemSize);
            attr.needsUpdate = n > 0;
        }
    }

    dispose(): void {
        this.mesh.geometry.dispose();
        (this.mesh.material as THREE.Material).dispose();
    }
}

/** Lambert material whose vertex positions come from the VAT. Flat shading
 *  (normals from screen-space derivatives) matches the low-poly style and
 *  means no normals need to be baked. */
function createVatMaterial(model: VatModel) {
    const uniforms = {
        uTime: { value: 0 },
        vatTex: { value: model.texture },
        vatWidth: { value: model.meta.texture.width },
        vatVerts: { value: model.meta.vertexCount },
    };
    const material = new THREE.MeshLambertMaterial({ vertexColors: true, flatShading: true });
    material.onBeforeCompile = (shader) => {
        Object.assign(shader.uniforms, uniforms);
        shader.vertexShader = shader.vertexShader
            .replace(
                "#include <common>",
                /* glsl */ `#include <common>
                attribute vec4 aAnim;       // start frame, frame count, frames per second, loop
                attribute float aAnimStart; // seconds
                attribute vec3 aTeam;
                attribute float teamMask;
                uniform sampler2D vatTex;
                uniform int vatWidth;
                uniform int vatVerts;
                uniform float uTime;
                vec3 vatFetch(float frame) {
                    int i = int(frame) * vatVerts + gl_VertexID;
                    return texelFetch(vatTex, ivec2(i % vatWidth, i / vatWidth), 0).xyz;
                }`,
            )
            .replace(
                "#include <begin_vertex>",
                /* glsl */ `
                float vt = max(uTime - aAnimStart, 0.0) * aAnim.z;
                float fc = aAnim.y;
                float ff = aAnim.w > 0.5 ? mod(vt, fc) : min(vt, fc - 1.0);
                float f0 = floor(ff);
                float f1 = aAnim.w > 0.5 ? mod(f0 + 1.0, fc) : min(f0 + 1.0, fc - 1.0);
                vec3 transformed = mix(vatFetch(aAnim.x + f0), vatFetch(aAnim.x + f1), ff - f0);`,
            )
            .replace(
                "#include <color_vertex>",
                /* glsl */ `
                float lum = dot(color.rgb, vec3(0.299, 0.587, 0.114));
                vec3 tinted = mix(color.rgb, aTeam * (0.55 + 0.6 * lum), teamMask);
                vColor = vec4(tinted, 1.0); // vColor is a vec4 in three r18x`,
            );
    };
    material.customProgramCacheKey = () => `vat-${model.id}`;
    return { material, uniforms };
}
