import * as THREE from "three";
import { VatRenderer, type Lod, type VatClip, type VatModel } from "./VatModel";

/** Distance (world units, camera to unit) where each LOD ends. Beyond the last: LOD 2. */
export const LOD_DISTANCES: [number, number] = [45, 110];
/** Bounding sphere of a unit (scaled by instance scale), centered at chest height. */
const SPHERE_RADIUS = 1.3;
const SPHERE_CENTER_Y = 0.9;

export interface LodStats {
    submitted: number;
    culled: number;
    perLod: [number, number, number];
}

/**
 * Draws one VAT model at three levels of detail with frustum culling. Per frame:
 * begin(camera) → slot(...) for every unit → end(time). slot() picks the LOD from
 * the distance to the camera, skips units outside the view, writes the
 * animation attributes and returns where the caller writes the 4x4 matrix.
 */
export class VatLodSet {
    readonly renderers: [VatRenderer, VatRenderer, VatRenderer];
    /** Force a single LOD for every unit (benchmark comparisons); null = automatic. */
    forceLod: Lod | null = null;
    readonly stats: LodStats = { submitted: 0, culled: 0, perLod: [0, 0, 0] };

    /** Matrix array of the renderer chosen by the last slot() call. */
    matrices: Float32Array;

    private readonly planes = new Float32Array(24); // 6 x (nx, ny, nz, d)
    private readonly frustum = new THREE.Frustum();
    private readonly projView = new THREE.Matrix4();
    private camX = 0;
    private camY = 0;
    private camZ = 0;

    constructor(readonly model: VatModel, capacity: number) {
        this.renderers = [new VatRenderer(model, capacity, 0), new VatRenderer(model, capacity, 1), new VatRenderer(model, capacity, 2)];
        this.matrices = this.renderers[0].matrices;
    }

    get meshes(): THREE.Object3D[] {
        return this.renderers.map((r) => r.mesh);
    }

    begin(camera: THREE.Camera): void {
        camera.updateMatrixWorld();
        this.projView.multiplyMatrices(camera.projectionMatrix, camera.matrixWorldInverse);
        this.frustum.setFromProjectionMatrix(this.projView);
        this.frustum.planes.forEach((p, i) => {
            this.planes[i * 4] = p.normal.x;
            this.planes[i * 4 + 1] = p.normal.y;
            this.planes[i * 4 + 2] = p.normal.z;
            this.planes[i * 4 + 3] = p.constant;
        });
        const e = camera.matrixWorld.elements;
        this.camX = e[12];
        this.camY = e[13];
        this.camZ = e[14];

        for (const r of this.renderers) r.begin();
        this.stats.submitted = 0;
        this.stats.culled = 0;
        this.stats.perLod[0] = this.stats.perLod[1] = this.stats.perLod[2] = 0;
    }

    /**
     * Reserves an instance for a unit at (x, y, z) with scale s. Returns the
     * offset into `this.matrices` where the caller must write the unit's matrix,
     * or -1 if the unit is culled (or the renderer is full).
     */
    slot(x: number, y: number, z: number, s: number, clip: VatClip, startSec: number, rate: number, team: THREE.Color): number {
        this.stats.submitted++;
        const cy = y + SPHERE_CENTER_Y * s;
        const r = SPHERE_RADIUS * s;
        const p = this.planes;
        for (let i = 0; i < 24; i += 4) {
            if (p[i] * x + p[i + 1] * cy + p[i + 2] * z + p[i + 3] < -r) {
                this.stats.culled++;
                return -1;
            }
        }

        let lod: Lod = this.forceLod ?? 2;
        if (this.forceLod === null) {
            const dx = x - this.camX;
            const dy = cy - this.camY;
            const dz = z - this.camZ;
            const d2 = dx * dx + dy * dy + dz * dz;
            if (d2 < LOD_DISTANCES[0] * LOD_DISTANCES[0]) lod = 0;
            else if (d2 < LOD_DISTANCES[1] * LOD_DISTANCES[1]) lod = 1;
        }

        const renderer = this.renderers[lod];
        const i = renderer.add(clip, startSec, rate, team);
        if (i < 0) return -1;
        this.stats.perLod[lod]++;
        this.matrices = renderer.matrices;
        return i * 16;
    }

    end(nowSec: number): void {
        for (const r of this.renderers) r.end(nowSec);
    }

    dispose(): void {
        for (const r of this.renderers) r.dispose();
    }
}
