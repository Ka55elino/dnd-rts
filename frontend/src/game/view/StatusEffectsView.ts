import * as THREE from "three";
import {
    STATUS_ACID,
    STATUS_BURNING,
    STATUS_POISONED,
    STATUS_SLOWED,
    STATUS_STUNNED,
    isBurning,
    type BuildingSnap,
} from "../../net/protocol";
import type { SoldiersView } from "./SoldiersView";

/** Particles in the shared pool (one buffer, one draw call, never reallocated). */
const CAPACITY = 12_000;
/** Soldiers farther than this from the camera get no status particles. */
const MAX_DISTANCE = 160;

interface Look {
    bit: number;
    color: THREE.Color;
    count: number; // particles per soldier
    size: number;
}

const LOOKS: Look[] = [
    { bit: STATUS_BURNING, color: new THREE.Color(0xff8a2a), count: 3, size: 0.55 },
    { bit: STATUS_POISONED, color: new THREE.Color(0x7be05a), count: 2, size: 0.4 },
    { bit: STATUS_ACID, color: new THREE.Color(0xd8e84a), count: 2, size: 0.35 },
    { bit: STATUS_SLOWED, color: new THREE.Color(0x8fd4ff), count: 2, size: 0.45 },
    { bit: STATUS_STUNNED, color: new THREE.Color(0xffe46a), count: 3, size: 0.35 },
];

const FIRE = new THREE.Color(0xff7a20);
const SMOKE = new THREE.Color(0x6a5a4a);

/**
 * Visuals of status effects (state comes from the Go simulation; nothing is
 * decided here): soft glowing particles around burning, poisoned, slowed,
 * stunned and acid-covered soldiers, and fire on burning buildings.
 *
 * All particles live in one fixed-size pooled buffer drawn with a single
 * Points call; each frame refills it from the current state, so effects never
 * allocate objects. Far soldiers and particles beyond the pool are skipped.
 */
export class StatusEffectsView {
    readonly points: THREE.Points;
    private readonly pos: Float32Array;
    private readonly col: Float32Array;
    private readonly size: Float32Array;
    private readonly geo = new THREE.BufferGeometry();
    private readonly texture: THREE.Texture;
    private readonly tmp = new THREE.Vector3();

    constructor() {
        this.pos = new Float32Array(CAPACITY * 3);
        this.col = new Float32Array(CAPACITY * 3);
        this.size = new Float32Array(CAPACITY);
        this.geo.setAttribute("position", new THREE.BufferAttribute(this.pos, 3).setUsage(THREE.DynamicDrawUsage));
        this.geo.setAttribute("color", new THREE.BufferAttribute(this.col, 3).setUsage(THREE.DynamicDrawUsage));
        this.geo.setAttribute("pSize", new THREE.BufferAttribute(this.size, 1).setUsage(THREE.DynamicDrawUsage));
        this.texture = softDot();
        const material = new THREE.PointsMaterial({
            size: 6, // world units at distance 1 (attenuated); per-particle pSize scales it
            map: this.texture,
            vertexColors: true,
            transparent: true,
            depthWrite: false,
            blending: THREE.AdditiveBlending,
            sizeAttenuation: true,
        });
        // Per-particle size: the "pSize" attribute scales the material size.
        material.onBeforeCompile = (shader) => {
            shader.vertexShader = shader.vertexShader
                .replace("#include <common>", "#include <common>\nattribute float pSize;")
                .replace("gl_PointSize = size;", "gl_PointSize = size * pSize;");
        };
        material.customProgramCacheKey = () => "status-fx";
        this.points = new THREE.Points(this.geo, material);
        this.points.frustumCulled = false;
        this.geo.setDrawRange(0, 0);
    }

    /** Refills the pool from soldiers (render positions) and burning buildings. */
    update(nowMs: number, soldiers: SoldiersView, buildings: Iterable<BuildingSnap>, heightAt: (x: number, z: number) => number, camera: THREE.Camera): void {
        const t = nowMs / 1000;
        const cam = camera.position;
        const maxSq = MAX_DISTANCE * MAX_DISTANCE;
        let n = 0;

        const rp = soldiers.renderPos;
        const st = soldiers.status;
        for (let i = 0; i < soldiers.count && n < CAPACITY; i++) {
            const bits = st[i];
            if (!bits) continue;
            const x = rp[i * 3];
            const y = rp[i * 3 + 1];
            const z = rp[i * 3 + 2];
            const dx = x - cam.x;
            const dz = z - cam.z;
            if (dx * dx + dz * dz > maxSq) continue;
            for (const look of LOOKS) {
                if (!(bits & look.bit)) continue;
                for (let k = 0; k < look.count && n < CAPACITY; k++) {
                    const seed = i * 7.31 + k * 2.17 + look.bit;
                    const phase = fract(t * 0.9 + seed * 0.37);
                    let px = x, py = y, pz = z;
                    switch (look.bit) {
                        case STATUS_BURNING: // flames rising from the body
                            px += Math.sin(seed * 12.9) * 0.35;
                            pz += Math.cos(seed * 7.7) * 0.35;
                            py += 0.4 + phase * 1.6;
                            break;
                        case STATUS_STUNNED: { // stars circling above the head
                            const a = t * 4 + (k * Math.PI * 2) / look.count;
                            px += Math.cos(a) * 0.45;
                            pz += Math.sin(a) * 0.45;
                            py += 2.1;
                            break;
                        }
                        case STATUS_SLOWED: // frost at the feet
                            px += Math.sin(seed * 5.1 + t) * 0.45;
                            pz += Math.cos(seed * 3.3 + t) * 0.45;
                            py += 0.15 + phase * 0.3;
                            break;
                        default: // poison / acid: bubbles drifting up
                            px += Math.sin(seed * 9.3) * 0.3;
                            pz += Math.cos(seed * 4.1) * 0.3;
                            py += 0.6 + phase * 1.1;
                    }
                    this.put(n++, px, py, pz, look.color, look.size * (look.bit === STATUS_BURNING ? 1.2 - phase * 0.6 : 1));
                }
            }
        }

        // Burning buildings: a cluster of flames and some smoke.
        for (const b of buildings) {
            if (!isBurning(b.flags)) continue;
            const ground = heightAt(b.x, b.z);
            this.tmp.set(b.x - cam.x, 0, b.z - cam.z);
            if (this.tmp.lengthSq() > maxSq * 2) continue;
            for (let k = 0; k < 40 && n < CAPACITY; k++) {
                const seed = b.id * 3.7 + k * 1.93;
                const phase = fract(t * 0.6 + seed * 0.21);
                const r = 1.5 + fract(seed * 0.77) * 3;
                const a = seed * 2.4;
                const smoke = k % 4 === 0;
                this.put(
                    n++,
                    b.x + Math.cos(a) * r,
                    ground + 1 + phase * (smoke ? 10 : 6),
                    b.z + Math.sin(a) * r,
                    smoke ? SMOKE : FIRE,
                    smoke ? 2.2 : 1.4 - phase * 0.8,
                );
            }
        }

        this.geo.setDrawRange(0, n);
        for (const name of ["position", "color", "pSize"] as const) {
            const attr = this.geo.getAttribute(name) as THREE.BufferAttribute;
            attr.clearUpdateRanges();
            if (n > 0) attr.addUpdateRange(0, n * attr.itemSize);
            attr.needsUpdate = n > 0;
        }
    }

    dispose(): void {
        this.geo.dispose();
        (this.points.material as THREE.Material).dispose();
        this.texture.dispose();
    }

    private put(i: number, x: number, y: number, z: number, c: THREE.Color, size: number): void {
        this.pos[i * 3] = x;
        this.pos[i * 3 + 1] = y;
        this.pos[i * 3 + 2] = z;
        this.col[i * 3] = c.r;
        this.col[i * 3 + 1] = c.g;
        this.col[i * 3 + 2] = c.b;
        this.size[i] = size;
    }
}

function fract(v: number): number {
    return v - Math.floor(v);
}

/** Soft round sprite. */
function softDot(): THREE.Texture {
    const c = document.createElement("canvas");
    c.width = c.height = 32;
    const g = c.getContext("2d")!;
    const grad = g.createRadialGradient(16, 16, 0, 16, 16, 16);
    grad.addColorStop(0, "rgba(255,255,255,1)");
    grad.addColorStop(0.4, "rgba(255,255,255,0.55)");
    grad.addColorStop(1, "rgba(255,255,255,0)");
    g.fillStyle = grad;
    g.fillRect(0, 0, 32, 32);
    const tex = new THREE.CanvasTexture(c);
    tex.colorSpace = THREE.SRGBColorSpace;
    return tex;
}
