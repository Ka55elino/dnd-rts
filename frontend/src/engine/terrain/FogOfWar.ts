import * as THREE from "three";

/** Brightness of never-explored and explored-but-unseen ground (1 = in sight). */
const UNEXPLORED = 0;
const EXPLORED = 0.45;
const FADE_PER_SECOND = 3; // how fast a cell brightens / darkens

/**
 * Fog of war on the ground: the server's per-player grid (unexplored / explored / visible) as a
 * small texture that patched materials (terrain, trees) sample by world xz to darken what the
 * player can't see. Cells fade in and out instead of popping.
 */
export class FogOfWar {
    readonly uniforms = {
        fogMap: { value: null as THREE.DataTexture | null },
        fogOn: { value: 0 },
        fogSize: { value: new THREE.Vector2(1, 1) },
    };
    private cols = 0;
    private rows = 0;
    private cells: Uint8Array | null = null;
    private target = new Float32Array(0);
    private current = new Float32Array(0);
    private texData = new Uint8Array(0);

    constructor(readonly width: number, readonly height: number) {
        this.uniforms.fogSize.value.set(width, height);
    }

    /** Fog grid from a snapshot (null = no fog of war). */
    push(fog: { cols: number; rows: number; cells: Uint8Array } | null): void {
        if (!fog) {
            this.cells = null;
            this.uniforms.fogOn.value = 0;
            return;
        }
        if (fog.cols !== this.cols || fog.rows !== this.rows || !this.uniforms.fogMap.value) {
            this.cols = fog.cols;
            this.rows = fog.rows;
            const n = fog.cols * fog.rows;
            this.target = new Float32Array(n);
            this.current = new Float32Array(n);
            this.texData = new Uint8Array(n);
            this.uniforms.fogMap.value?.dispose();
            const tex = new THREE.DataTexture(this.texData, fog.cols, fog.rows, THREE.RedFormat, THREE.UnsignedByteType);
            tex.magFilter = THREE.LinearFilter;
            tex.minFilter = THREE.LinearFilter;
            tex.needsUpdate = true;
            this.uniforms.fogMap.value = tex;
            for (let i = 0; i < n; i++) this.current[i] = level(fog.cells[i]); // no fade on the first frame
        }
        this.cells = fog.cells;
        for (let i = 0; i < fog.cells.length; i++) this.target[i] = level(fog.cells[i]);
        this.uniforms.fogOn.value = 1;
    }

    /** Fades the texture towards the latest grid. */
    update(dt: number): void {
        const tex = this.uniforms.fogMap.value;
        if (!tex || !this.cells) return;
        const step = FADE_PER_SECOND * dt;
        let changed = false;
        for (let i = 0; i < this.current.length; i++) {
            const c = this.current[i];
            const t = this.target[i];
            if (c === t) continue;
            this.current[i] = c < t ? Math.min(t, c + step) : Math.max(t, c - step);
            this.texData[i] = Math.round(this.current[i] * 255);
            changed = true;
        }
        if (changed) tex.needsUpdate = true;
    }

    get enabled(): boolean {
        return this.cells !== null;
    }

    /** The cell at a world point was seen at least once (always true without fog). */
    explored(x: number, z: number): boolean {
        const c = this.cell(x, z);
        return c < 0 || (this.cells?.[c] ?? 1) > 0;
    }

    /** The cell at a world point is in sight now (always true without fog). */
    visible(x: number, z: number): boolean {
        const c = this.cell(x, z);
        return c < 0 || (this.cells?.[c] ?? 2) > 1;
    }

    /**
     * Makes a lit material darken by the fog: world xz → fog texture, the final colour scaled
     * down where the player doesn't see. Works with instanced meshes.
     */
    patch(material: THREE.Material): void {
        const u = this.uniforms;
        material.onBeforeCompile = (shader) => {
            shader.uniforms.fogMap = u.fogMap;
            shader.uniforms.fogOn = u.fogOn;
            shader.uniforms.fogSize = u.fogSize;
            shader.vertexShader = shader.vertexShader
                .replace("#include <common>", "#include <common>\nuniform vec2 fogSize;\nvarying vec2 vFogUv;")
                .replace(
                    "#include <project_vertex>",
                    `#include <project_vertex>
                    vec4 fogWorld = vec4(transformed, 1.0);
                    #ifdef USE_INSTANCING
                    fogWorld = instanceMatrix * fogWorld;
                    #endif
                    fogWorld = modelMatrix * fogWorld;
                    vFogUv = (fogWorld.xz + fogSize * 0.5) / fogSize;`,
                );
            shader.fragmentShader = shader.fragmentShader
                .replace("#include <common>", "#include <common>\nuniform sampler2D fogMap;\nuniform float fogOn;\nvarying vec2 vFogUv;")
                .replace(
                    "#include <dithering_fragment>",
                    `#include <dithering_fragment>
                    if (fogOn > 0.5) gl_FragColor.rgb *= mix(0.12, 1.0, texture2D(fogMap, vFogUv).r);`,
                );
        };
        material.needsUpdate = true;
    }

    dispose(): void {
        this.uniforms.fogMap.value?.dispose();
    }

    private cell(x: number, z: number): number {
        if (!this.cells) return -1;
        const c = Math.floor(((x + this.width / 2) / this.width) * this.cols);
        const r = Math.floor(((z + this.height / 2) / this.height) * this.rows);
        if (c < 0 || r < 0 || c >= this.cols || r >= this.rows) return -1;
        return r * this.cols + c;
    }
}

function level(state: number): number {
    return state >= 2 ? 1 : state === 1 ? EXPLORED : UNEXPLORED;
}
