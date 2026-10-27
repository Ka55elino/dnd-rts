import * as THREE from "three";
import { SNAPSHOT_ENTITY_FLOATS, type Snapshot } from "../../net/protocol";

/**
 * Renders entities from snapshots with one InstancedMesh, interpolating
 * between the two latest snapshots so motion is smooth at any frame rate.
 */
export class UnitsView {
    /** Target mesh; can be swapped for another InstancedMesh with the same capacity. */
    mesh: THREE.InstancedMesh;

    private prev: Float32Array;
    private curr: Float32Array;
    private count = 0;
    private currTime = 0;

    constructor(
        private readonly capacity: number,
        private readonly tickMs: number,
        geometry: THREE.BufferGeometry,
        material: THREE.Material,
    ) {
        this.mesh = new THREE.InstancedMesh(geometry, material, capacity);
        this.mesh.instanceMatrix.setUsage(THREE.DynamicDrawUsage);
        this.mesh.count = 0;
        // Bounding sphere is not recomputed for moving instances.
        this.mesh.frustumCulled = false;

        this.prev = new Float32Array(capacity * SNAPSHOT_ENTITY_FLOATS);
        this.curr = new Float32Array(capacity * SNAPSHOT_ENTITY_FLOATS);
    }

    pushSnapshot(s: Snapshot, now: number): void {
        const n = Math.min(s.count, this.capacity);
        // Swap buffers, then copy the new frame into `curr`.
        [this.prev, this.curr] = [this.curr, this.prev];
        this.curr.set(s.entities.subarray(0, n * SNAPSHOT_ENTITY_FLOATS));
        if (n > this.count) {
            // New entities have no previous state: start them at their current position.
            this.prev.set(this.curr.subarray(this.count * SNAPSHOT_ENTITY_FLOATS, n * SNAPSHOT_ENTITY_FLOATS), this.count * SNAPSHOT_ENTITY_FLOATS);
        }
        this.count = n;
        this.currTime = now;
    }

    /** Writes interpolated transforms straight into the instance matrix buffer. */
    update(now: number): void {
        // Render one tick behind the latest snapshot: lerp prev → curr over one tick.
        const alpha = Math.min(Math.max((now - this.currTime) / this.tickMs, 0), 1);
        const m = this.mesh.instanceMatrix.array as Float32Array;
        const prev = this.prev;
        const curr = this.curr;

        for (let i = 0, o = 0, k = 0; i < this.count; i++, o += SNAPSHOT_ENTITY_FLOATS, k += 16) {
            const x = prev[o] + (curr[o] - prev[o]) * alpha;
            const z = prev[o + 1] + (curr[o + 1] - prev[o + 1]) * alpha;

            let dr = curr[o + 2] - prev[o + 2];
            if (dr > Math.PI) dr -= 2 * Math.PI;
            else if (dr < -Math.PI) dr += 2 * Math.PI;
            const r = prev[o + 2] + dr * alpha;

            const c = Math.cos(r);
            const s = Math.sin(r);
            // Column-major rotation around Y + translation.
            m[k] = c;   m[k + 1] = 0; m[k + 2] = -s;  m[k + 3] = 0;
            m[k + 4] = 0; m[k + 5] = 1; m[k + 6] = 0;  m[k + 7] = 0;
            m[k + 8] = s; m[k + 9] = 0; m[k + 10] = c; m[k + 11] = 0;
            m[k + 12] = x; m[k + 13] = 0; m[k + 14] = z; m[k + 15] = 1;
        }

        this.mesh.count = this.count;
        this.mesh.instanceMatrix.clearUpdateRanges();
        this.mesh.instanceMatrix.addUpdateRange(0, this.count * 16);
        this.mesh.instanceMatrix.needsUpdate = true;
    }

    /**
     * Interpolated (x, z, rotation) per entity into `out` (3 floats each), for
     * callers that render through something other than `mesh`. Returns the count.
     */
    interpolate(now: number, out: Float32Array): number {
        const alpha = Math.min(Math.max((now - this.currTime) / this.tickMs, 0), 1);
        const prev = this.prev;
        const curr = this.curr;
        for (let o = 0, n = this.count * SNAPSHOT_ENTITY_FLOATS; o < n; o += SNAPSHOT_ENTITY_FLOATS) {
            out[o] = prev[o] + (curr[o] - prev[o]) * alpha;
            out[o + 1] = prev[o + 1] + (curr[o + 1] - prev[o + 1]) * alpha;
            let dr = curr[o + 2] - prev[o + 2];
            if (dr > Math.PI) dr -= 2 * Math.PI;
            else if (dr < -Math.PI) dr += 2 * Math.PI;
            out[o + 2] = prev[o + 2] + dr * alpha;
        }
        return this.count;
    }

    get entityCount(): number {
        return this.count;
    }
}
