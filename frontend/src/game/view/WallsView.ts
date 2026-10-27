import * as THREE from "three";
import { StaticModel } from "../../engine/models/StaticModel";

/** One wall or gate segment, from the building snapshot. */
export interface WallSegment {
    id: number;
    x: number;
    y: number; // ground under it
    z: number;
    rot: number; // three.js Y rotation; local +Z points out of the base
    length: number; // footprint along the wall
    thickness: number; // footprint across
    walkHeight: number;
    gate: boolean;
}

const CAPACITY = 1200; // wall rows (two per segment): enough for 8 bases
const POSTS = 2; // gate posts per gate segment
const CULL_RADIUS = 8;

/**
 * Base walls, drawn with instancing: every segment is two rows of the
 * castle-wall model back to back (crenellations on both sides) with a stone
 * walkway on top; gates are an arch of two posts under the walkway. One
 * InstancedMesh per model part and LOD level; each frame the rows go to the
 * LOD of their camera distance, rows outside the view are skipped. An
 * invisible box per segment is the click target.
 */
export class WallsView {
    readonly group = new THREE.Group();
    private readonly segs = new Map<number, WallSegment>();
    private readonly levels: { mesh: THREE.InstancedMesh; local: THREE.Matrix4 }[][] = [];
    private readonly floor: THREE.InstancedMesh;
    private readonly posts: THREE.InstancedMesh;
    private readonly picker: THREE.InstancedMesh;
    private pickIds: number[] = [];
    private dirty = true;
    private readonly modelLength: number;
    private readonly modelDepth: number;
    private readonly m = new THREE.Matrix4();
    private readonly seg = new THREE.Matrix4();
    private readonly row = new THREE.Matrix4();
    private readonly q = new THREE.Quaternion();
    private readonly v = new THREE.Vector3();
    private readonly s = new THREE.Vector3();
    private readonly up = new THREE.Vector3(0, 1, 0);
    private readonly frustum = new THREE.Frustum();
    private readonly sphere = new THREE.Sphere();
    private readonly pv = new THREE.Matrix4();

    constructor(model: StaticModel | null) {
        this.modelLength = model ? model.bounds.max.x - model.bounds.min.x : 6;
        this.modelDepth = model ? model.bounds.max.z - model.bounds.min.z : 2.4;
        for (let l = 0; model && l < model.levels; l++) {
            this.levels.push(
                model.parts(l).map((p) => {
                    const mesh = new THREE.InstancedMesh(p.geometry, p.material, CAPACITY);
                    mesh.instanceMatrix.setUsage(THREE.DynamicDrawUsage);
                    mesh.count = 0;
                    mesh.frustumCulled = false;
                    this.group.add(mesh);
                    return { mesh, local: p.matrix };
                }),
            );
        }
        const stone = new THREE.MeshLambertMaterial({ color: 0x8f8778 });
        this.floor = this.boxes(stone, CAPACITY);
        this.posts = this.boxes(stone, CAPACITY);
        this.picker = new THREE.InstancedMesh(new THREE.BoxGeometry(1, 1, 1).translate(0, 0.5, 0), new THREE.MeshBasicMaterial({ visible: false }), CAPACITY);
        this.picker.count = 0;
        this.group.add(this.picker);
    }

    private boxes(mat: THREE.Material, n: number): THREE.InstancedMesh {
        const mesh = new THREE.InstancedMesh(new THREE.BoxGeometry(1, 1, 1).translate(0, 0.5, 0), mat, n);
        mesh.instanceMatrix.setUsage(THREE.DynamicDrawUsage);
        mesh.count = 0;
        mesh.frustumCulled = false;
        this.group.add(mesh);
        return mesh;
    }

    set(s: WallSegment): void {
        this.segs.set(s.id, s);
        this.dirty = true;
    }

    remove(id: number): void {
        if (this.segs.delete(id)) this.dirty = true;
    }

    /** Segment under a ray (its box, walkway included), or null. */
    pick(ray: THREE.Raycaster): { id: number; point: THREE.Vector3 } | null {
        if (this.dirty) this.rebuildStatic();
        const hit = ray.intersectObject(this.picker, false)[0];
        if (!hit || hit.instanceId === undefined) return null;
        return { id: this.pickIds[hit.instanceId], point: hit.point };
    }

    /** Assigns rows to LODs by camera distance and skips those out of view. */
    update(camera: THREE.Camera): void {
        if (this.dirty) this.rebuildStatic();
        if (this.levels.length === 0) return;
        this.pv.multiplyMatrices(camera.projectionMatrix, camera.matrixWorldInverse);
        this.frustum.setFromProjectionMatrix(this.pv);
        const cam = camera.position;
        const counts = this.levels.map(() => 0);
        for (const s of this.segs.values()) {
            if (s.gate) continue;
            this.sphere.center.set(s.x, s.y + s.walkHeight / 2, s.z);
            this.sphere.radius = CULL_RADIUS;
            if (!this.frustum.intersectsSphere(this.sphere)) continue;
            const d = cam.distanceTo(this.sphere.center);
            let level = 0;
            while (level + 1 < this.levels.length && d >= StaticModel.lodDistance(level + 1)) level++;
            this.segmentMatrix(s);
            const off = s.thickness / 2 - this.modelDepth / 2;
            const sx = (s.length / this.modelLength) * 1.03; // a touch of overlap hides the joints
            for (const side of [1, -1]) {
                this.row.compose(this.v.set(0, 0, side * off), this.q.setFromAxisAngle(this.up, side > 0 ? 0 : Math.PI), this.s.set(sx, 1, 1));
                this.row.premultiply(this.seg);
                const i = counts[level]++;
                for (const part of this.levels[level]) {
                    this.m.multiplyMatrices(this.row, part.local);
                    part.mesh.setMatrixAt(i, this.m);
                }
            }
        }
        this.levels.forEach((parts, l) => {
            for (const p of parts) {
                p.mesh.count = counts[l];
                p.mesh.instanceMatrix.needsUpdate = true;
            }
        });
    }

    dispose(): void {
        for (const parts of this.levels) for (const p of parts) this.group.remove(p.mesh); // geometry / materials belong to the model
        for (const b of [this.floor, this.posts, this.picker]) {
            b.geometry.dispose();
            (b.material as THREE.Material).dispose();
        }
    }

    private segmentMatrix(s: WallSegment): void {
        this.seg.compose(this.v.set(s.x, s.y, s.z), this.q.setFromAxisAngle(this.up, s.rot), this.s.set(1, 1, 1));
    }

    /** Walkway slabs, gate posts and click boxes change only with the segments. */
    private rebuildStatic(): void {
        this.dirty = false;
        let f = 0;
        let p = 0;
        this.pickIds = [];
        for (const s of this.segs.values()) {
            this.segmentMatrix(s);
            // Walkway slab on top (over gates too: the arch).
            this.m.compose(this.v.set(0, s.walkHeight - 0.35, 0), this.q.identity(), this.s.set(s.length * 1.02, 0.35, s.thickness - 0.4)).premultiply(this.seg);
            this.floor.setMatrixAt(f++, this.m);
            if (s.gate) {
                for (let k = 0; k < POSTS; k++) {
                    const x = (k === 0 ? -1 : 1) * (s.length / 2 - 0.6);
                    this.m.compose(this.v.set(x, 0, 0), this.q.identity(), this.s.set(1.2, s.walkHeight, s.thickness)).premultiply(this.seg);
                    this.posts.setMatrixAt(p++, this.m);
                }
            }
            this.m.compose(this.v.set(0, 0, 0), this.q.identity(), this.s.set(s.length, s.walkHeight, s.thickness)).premultiply(this.seg);
            this.picker.setMatrixAt(this.pickIds.length, this.m);
            this.pickIds.push(s.id);
        }
        this.floor.count = f;
        this.posts.count = p;
        this.picker.count = this.pickIds.length;
        for (const b of [this.floor, this.posts, this.picker]) b.instanceMatrix.needsUpdate = true;
        this.picker.computeBoundingSphere();
    }
}
