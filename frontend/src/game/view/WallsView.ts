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
    /** Construction progress 0..1 (1 = finished): a foundation rises out of the ground. */
    progress?: number;
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
    /** Round joints where a wall turns: a stone drum to the walkway and a parapet ring on top. */
    private readonly joints: THREE.InstancedMesh;
    private readonly jointRims: THREE.InstancedMesh;
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
        this.joints = this.instanced(new THREE.CylinderGeometry(1, 1, 1, 24).translate(0, 0.5, 0), new THREE.MeshLambertMaterial({ color: 0xa19b91 }));
        this.jointRims = this.instanced(
            new THREE.CylinderGeometry(1, 1, 1, 24, 1, true).translate(0, 0.5, 0),
            new THREE.MeshLambertMaterial({ color: 0xb8b3a8, side: THREE.DoubleSide }),
        );
        this.picker = new THREE.InstancedMesh(new THREE.BoxGeometry(1, 1, 1).translate(0, 0.5, 0), new THREE.MeshBasicMaterial({ visible: false }), CAPACITY);
        this.picker.count = 0;
        this.group.add(this.picker);
    }

    private instanced(geo: THREE.BufferGeometry, mat: THREE.Material): THREE.InstancedMesh {
        const mesh = new THREE.InstancedMesh(geo, mat, CAPACITY);
        mesh.count = 0;
        mesh.frustumCulled = false;
        mesh.castShadow = true;
        mesh.receiveShadow = true;
        this.group.add(mesh);
        return mesh;
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

    /** A segment under construction grew (or finished). */
    setProgress(id: number, progress: number): void {
        const s = this.segs.get(id);
        if (!s || s.progress === progress) return;
        s.progress = progress;
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
                this.row.compose(this.v.set(0, 0, side * off), this.q.setFromAxisAngle(this.up, side > 0 ? 0 : Math.PI), this.s.set(sx, height(s), 1));
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
        for (const b of [this.floor, this.posts, this.picker, this.joints, this.jointRims]) {
            b.geometry.dispose();
            (b.material as THREE.Material).dispose();
        }
    }

    /**
     * Joints (mirror of sim.WallJoints): where the centre-line tips of two finished wall segments
     * meet at an angle, a drum of the wall's thickness fills the corner, up to the walkway, with a
     * parapet ring above.
     */
    private rebuildJoints(): void {
        const tips: { x: number; z: number; ax: number; az: number; s: WallSegment }[] = [];
        for (const s of this.segs.values()) {
            if (s.gate || height(s) < 1) continue;
            const ax = Math.cos(s.rot);
            const az = -Math.sin(s.rot); // the segment's axis along the wall
            for (const k of [-1, 1]) tips.push({ x: s.x + (k * ax * s.length) / 2, z: s.z + (k * az * s.length) / 2, ax, az, s });
        }
        let n = 0;
        for (let i = 0; i < tips.length; i++) {
            for (let j = i + 1; j < tips.length; j++) {
                const a = tips[i];
                const b = tips[j];
                if (a.s === b.s || Math.hypot(a.x - b.x, a.z - b.z) > 0.5) continue;
                if (Math.abs(a.ax * b.ax + a.az * b.az) >= 0.985 || n >= CAPACITY) continue; // straight on
                const r = a.s.thickness / 2;
                const x = (a.x + b.x) / 2;
                const z = (a.z + b.z) / 2;
                const y = Math.min(a.s.y, b.s.y);
                this.m.compose(this.v.set(x, y, z), this.q.identity(), this.s.set(r, a.s.walkHeight, r));
                this.joints.setMatrixAt(n, this.m);
                this.m.compose(this.v.set(x, y + a.s.walkHeight, z), this.q.identity(), this.s.set(r, 1.0, r));
                this.jointRims.setMatrixAt(n, this.m);
                n++;
            }
        }
        this.joints.count = n;
        this.jointRims.count = n;
        this.joints.instanceMatrix.needsUpdate = true;
        this.jointRims.instanceMatrix.needsUpdate = true;
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
            // Walkway slab on top (over gates too: the arch); a foundation's rises with it.
            this.m.compose(this.v.set(0, s.walkHeight * height(s) - 0.35, 0), this.q.identity(), this.s.set(s.length * 1.02, 0.35, s.thickness - 0.4)).premultiply(this.seg);
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
        this.rebuildJoints();
        this.picker.count = this.pickIds.length;
        for (const b of [this.floor, this.posts, this.picker]) b.instanceMatrix.needsUpdate = true;
        this.picker.computeBoundingSphere();
    }
}

/** Height factor of a segment: a foundation grows from 15% to full. */
function height(s: WallSegment): number {
    return s.progress === undefined || s.progress >= 1 ? 1 : 0.15 + 0.85 * s.progress;
}
