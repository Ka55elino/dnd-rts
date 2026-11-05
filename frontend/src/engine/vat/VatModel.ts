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

export interface VatMeta {
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
        uvs?: { offset: number }; // textured models: 2 floats per vertex
    };
    /** Textured models: raw RGBA8 albedo (rows top to bottom), alpha = team mask. */
    albedo?: { file: string; width: number; height: number };
    /** Optional build time; Model Overview reloads the model when it changes. */
    build?: string;
    /** Optional per-frame vertex normals, same layout as the VAT (RGBA8, n * 0.5 + 0.5) -> smooth shading. */
    normals?: { file: string; format: "rgba8" };
    /** Optional material map over the UV atlas (RGBA8: R = specular, G = gloss). */
    material?: { file: string; width: number; height: number };
}

/**
 * A character baked by tools/vatbake: one merged mesh (vertex colors + team
 * mask) and a texture holding every vertex position for every animation frame.
 * Textured models also carry UVs and an albedo texture (alpha = team mask); the
 * vertex colors are then averaged from it and used only by the far LOD.
 */
export class VatModel {
    readonly geometry: THREE.BufferGeometry;
    readonly texture: THREE.DataTexture;
    /** Albedo (sRGB, mipmapped), or null for vertex-colored models. */
    readonly albedo: THREE.DataTexture | null;
    /** Per-frame normals (same layout as `texture`), or null: the model is flat-shaded. */
    readonly normals: THREE.DataTexture | null;
    /** Specular (R) and gloss (G) over the UV atlas, or null: a weak uniform highlight. */
    readonly material: THREE.DataTexture | null;
    /** Index buffers per level of detail (0 = full). Missing levels fall back to the previous one. */
    readonly lodIndex: THREE.BufferAttribute[];
    readonly clips: Record<string, VatClip>;

    private constructor(
        readonly id: string,
        readonly meta: VatMeta,
        geometry: ArrayBuffer,
        vat: ArrayBuffer,
        albedo: ArrayBuffer | null,
        normals: ArrayBuffer | null = null,
        material: ArrayBuffer | null = null,
    ) {
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
        if (g.uvs) geo.setAttribute("uv", new THREE.BufferAttribute(new Float32Array(geometry, g.uvs.offset, V * 2), 2));
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

        const a = meta.albedo;
        if (a && albedo && g.uvs) {
            // glTF UVs have v = 0 at the top row, which is the first row of the data: no flip.
            const tex = new THREE.DataTexture(new Uint8Array(albedo), a.width, a.height, THREE.RGBAFormat, THREE.UnsignedByteType);
            tex.colorSpace = THREE.SRGBColorSpace;
            tex.flipY = false;
            tex.premultiplyAlpha = false; // alpha is the team mask, not coverage
            tex.wrapS = tex.wrapT = THREE.ClampToEdgeWrapping;
            tex.magFilter = THREE.LinearFilter;
            tex.minFilter = THREE.LinearMipmapLinearFilter;
            tex.generateMipmaps = true;
            tex.anisotropy = 4;
            tex.needsUpdate = true;
            this.albedo = tex;
        } else {
            this.albedo = null;
        }

        if (normals) {
            const n = new THREE.DataTexture(new Uint8Array(normals), t.width, t.height, THREE.RGBAFormat, THREE.UnsignedByteType);
            n.minFilter = n.magFilter = THREE.NearestFilter;
            n.generateMipmaps = false;
            n.needsUpdate = true;
            this.normals = n;
        } else {
            this.normals = null;
        }
        const m = meta.material;
        if (m && material) {
            const mt = new THREE.DataTexture(new Uint8Array(material), m.width, m.height, THREE.RGBAFormat, THREE.UnsignedByteType);
            mt.flipY = false;
            mt.wrapS = mt.wrapT = THREE.ClampToEdgeWrapping;
            mt.magFilter = THREE.LinearFilter;
            mt.minFilter = THREE.LinearMipmapLinearFilter;
            mt.generateMipmaps = true;
            mt.needsUpdate = true;
            this.material = mt;
        } else {
            this.material = null;
        }
    }

    /** Loads frontend/public/models/<id>/. fresh: bypass the HTTP cache (hot reload after a rebuild). */
    static async load(id: string, fresh = false): Promise<VatModel> {
        const base = `/models/${id}`;
        const init: RequestInit | undefined = fresh ? { cache: "no-store" } : undefined;
        const [meta, geometry, vat] = await Promise.all([
            fetch(`${base}/model.json`, init).then((r) => checked(r, id).json() as Promise<VatMeta>),
            fetch(`${base}/geometry.bin`, init).then((r) => checked(r, id).arrayBuffer()),
            fetch(`${base}/vat.bin`, init).then((r) => checked(r, id).arrayBuffer()),
        ]);
        if (meta.version !== 1) throw new Error(`model ${id}: unsupported version ${meta.version}`);
        const optional = (file?: string) =>
            file ? fetch(`${base}/${file}`, init).then((r) => checked(r, id).arrayBuffer()) : Promise.resolve(null);
        const [albedo, normals, material] = await Promise.all([
            optional(meta.albedo?.file),
            optional(meta.normals?.file),
            optional(meta.material?.file),
        ]);
        return new VatModel(id, meta, geometry, vat, albedo, normals, material);
    }

    clip(name: string): VatClip | undefined {
        return this.clips[name];
    }

    dispose(): void {
        this.geometry.dispose();
        this.texture.dispose();
        this.albedo?.dispose();
        this.normals?.dispose();
        this.material?.dispose();
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

    /**
     * opts.smooth: smooth shading with the baked normals and specular highlights (default: when the
     * model has normals); false = the flat low-poly look. opts.shadows: cast shadows (default false;
     * it draws every instance a second time, so the game enables it only for the near LODs).
     */
    constructor(
        readonly model: VatModel,
        private readonly capacity: number,
        lod: Lod = 0,
        opts: { smooth?: boolean; shadows?: boolean } = {},
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

        // The far LOD keeps vertex colors (averaged from the albedo by vatbake): no texture reads.
        const textured = model.albedo !== null && lod < 2;
        const smooth = (opts.smooth ?? true) && model.normals !== null;
        const { material, depth, uniforms } = createVatMaterial(model, textured, smooth);
        this.uniforms = uniforms;
        this.mesh = new THREE.InstancedMesh(geometry, material, capacity);
        this.mesh.customDepthMaterial = depth;
        this.mesh.castShadow = opts.shadows ?? false;
        this.mesh.receiveShadow = true;
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

    /** Turn shadow casting on / off (it costs a second draw of every instance). */
    set castShadow(on: boolean) {
        this.mesh.castShadow = on;
    }

    dispose(): void {
        this.mesh.geometry.dispose();
        (this.mesh.material as THREE.Material).dispose();
        this.mesh.customDepthMaterial?.dispose();
    }
}

/** GLSL shared by the color and the shadow-depth materials: the VAT frame of this vertex. */
const VAT_PARS = /* glsl */ `
    attribute vec4 aAnim;       // start frame, frame count, frames per second, loop
    attribute float aAnimStart; // seconds
    uniform sampler2D vatTex;
    uniform int vatWidth;
    uniform int vatVerts;
    uniform float uTime;
    ivec2 vatTexel(float frame) {
        int i = int(frame) * vatVerts + gl_VertexID;
        return ivec2(i % vatWidth, i / vatWidth);
    }
    // The two frames around the current time and the blend between them.
    vec3 vatFrames() {
        float vt = max(uTime - aAnimStart, 0.0) * aAnim.z;
        float fc = aAnim.y;
        float ff = aAnim.w > 0.5 ? mod(vt, fc) : min(vt, fc - 1.0);
        float f0 = floor(ff);
        float f1 = aAnim.w > 0.5 ? mod(f0 + 1.0, fc) : min(f0 + 1.0, fc - 1.0);
        return vec3(aAnim.x + f0, aAnim.x + f1, ff - f0);
    }`;

const VAT_POSITION = /* glsl */ `
    vec3 vf = vatFrames();
    vec3 transformed = mix(texelFetch(vatTex, vatTexel(vf.x), 0).xyz, texelFetch(vatTex, vatTexel(vf.y), 0).xyz, vf.z);`;

/**
 * Material whose vertex positions come from the VAT.
 * - smooth (the model has baked normals): Phong — normals from the normal VAT, specular highlights
 *   scaled by the material map (R = specular, G = gloss -> shininess);
 * - otherwise Lambert with flat shading (normals from screen-space derivatives): the low-poly look.
 * Textured: the color comes from the albedo and its alpha is the team mask; otherwise from the
 * vertex colors. Also returns the depth material that casts the animated shadow.
 */
function createVatMaterial(model: VatModel, textured: boolean, smooth: boolean) {
    const uniforms = {
        uTime: { value: 0 },
        vatTex: { value: model.texture },
        vatNrm: { value: model.normals },
        vatWidth: { value: model.meta.texture.width },
        vatVerts: { value: model.meta.vertexCount },
    };
    const gloss = smooth && textured && model.material !== null;
    let material: THREE.MeshLambertMaterial | THREE.MeshPhongMaterial;
    if (smooth) {
        material = new THREE.MeshPhongMaterial({
            map: textured ? model.albedo : null,
            vertexColors: !textured,
            specularMap: gloss ? model.material : null,
            specular: new THREE.Color(gloss ? 0xffffff : 0x161616),
            shininess: 30,
        });
    } else {
        material = textured
            ? new THREE.MeshLambertMaterial({ map: model.albedo, flatShading: true })
            : new THREE.MeshLambertMaterial({ vertexColors: true, flatShading: true });
    }
    material.onBeforeCompile = (shader) => {
        if (textured) patchTextured(shader);
        Object.assign(shader.uniforms, uniforms);
        shader.vertexShader = shader.vertexShader
            .replace(
                "#include <common>",
                `#include <common>
                ${VAT_PARS}
                uniform sampler2D vatNrm;
                attribute vec3 aTeam;
                attribute float teamMask;`,
            )
            .replace("#include <begin_vertex>", VAT_POSITION)
            .replace(
                "#include <color_vertex>",
                /* glsl */ `
                float lum = dot(color.rgb, vec3(0.299, 0.587, 0.114));
                vec3 tinted = mix(color.rgb, aTeam * (0.55 + 0.6 * lum), teamMask);
                vColor = vec4(tinted, 1.0); // vColor is a vec4 in three r18x`,
            );
        if (smooth) {
            shader.vertexShader = shader.vertexShader.replace(
                "#include <beginnormal_vertex>",
                /* glsl */ `
                vec3 nf = vatFrames();
                vec3 objectNormal = normalize(mix(texelFetch(vatNrm, vatTexel(nf.x), 0).xyz,
                                                  texelFetch(vatNrm, vatTexel(nf.y), 0).xyz, nf.z) * 2.0 - 1.0);`,
            );
        }
        if (gloss) {
            // G of the material map = gloss: from a broad sheen (cloth) to a tight highlight (polished metal).
            shader.fragmentShader = shader.fragmentShader.replace(
                "#include <lights_phong_fragment>",
                /* glsl */ `#include <lights_phong_fragment>
                material.specularShininess = mix(6.0, 90.0, texture2D(specularMap, vSpecularMapUv).g);`,
            );
        }
    };
    material.customProgramCacheKey = () => `vat-${model.id}-${textured ? "albedo" : "colors"}-${smooth ? "smooth" : "flat"}`;

    // Shadow caster: the same VAT positions, depth only.
    const depth = new THREE.MeshDepthMaterial({ depthPacking: THREE.RGBADepthPacking });
    depth.onBeforeCompile = (shader) => {
        Object.assign(shader.uniforms, uniforms);
        shader.vertexShader = shader.vertexShader
            .replace("#include <common>", `#include <common>\n${VAT_PARS}`)
            .replace("#include <begin_vertex>", VAT_POSITION);
    };
    depth.customProgramCacheKey = () => `vat-depth-${model.id}`;
    return { material, depth, uniforms };
}

/** Textured variant: pass the team color to the fragment shader and tint the
 *  albedo by its alpha (team mask). Runs before the common VAT patch, so the
 *  vertex-color tint below never sees <color_vertex>. */
function patchTextured(shader: THREE.WebGLProgramParametersWithUniforms): void {
    shader.vertexShader = shader.vertexShader
        .replace("#include <common>", "#include <common>\nvarying vec3 vTeamColor;")
        .replace("#include <color_vertex>", "vTeamColor = aTeam;");
    shader.fragmentShader = shader.fragmentShader
        .replace("#include <common>", "#include <common>\nvarying vec3 vTeamColor;")
        .replace(
            "#include <map_fragment>",
            /* glsl */ `
            vec4 albedo = texture2D(map, vMapUv); // linear (sRGB texture)
            // Same tint as the vertex-color path, which works on sRGB-encoded values.
            float lum = dot(pow(albedo.rgb, vec3(1.0 / 2.2)), vec3(0.299, 0.587, 0.114));
            diffuseColor.rgb *= mix(albedo.rgb, vTeamColor * (0.55 + 0.6 * lum), albedo.a);`,
        );
}
