import * as THREE from "three";
import type { TreeSnap } from "../../net/protocol";

const MAX_TREES = 4096;
const SAPLING = 0.35; // a felled / newly planted tree is drawn this small until it grows

/**
 * Trees from game snapshots: instanced stand-ins (a trunk and two stacked cones). Grown trees
 * full size, saplings and stumps small. Rebuilt only when the list or a tree's state changes.
 */
export class TreesView {
    readonly group = new THREE.Group();
    private readonly trunks: THREE.InstancedMesh;
    private readonly crowns: THREE.InstancedMesh;
    private signature = "";
    private readonly m = new THREE.Matrix4();
    private readonly q = new THREE.Quaternion();
    private readonly up = new THREE.Vector3(0, 1, 0);
    private readonly v = new THREE.Vector3();
    private readonly s = new THREE.Vector3();

    constructor(private readonly heightAt: (x: number, z: number) => number) {
        const trunkGeo = new THREE.CylinderGeometry(0.18, 0.28, 2.2, 6);
        trunkGeo.translate(0, 1.1, 0);
        const crownGeo = mergeCones();
        this.trunks = new THREE.InstancedMesh(trunkGeo, new THREE.MeshLambertMaterial({ color: 0x5a4030 }), MAX_TREES);
        this.crowns = new THREE.InstancedMesh(crownGeo, new THREE.MeshLambertMaterial({ color: 0x2f5a2a }), MAX_TREES);
        for (const m of [this.trunks, this.crowns]) {
            m.count = 0;
            m.castShadow = true;
            m.receiveShadow = true;
            m.frustumCulled = false;
            this.group.add(m);
        }
    }

    /** Materials to darken with the fog of war. */
    get materials(): THREE.Material[] {
        return [this.trunks.material as THREE.Material, this.crowns.material as THREE.Material];
    }

    push(trees: TreeSnap[]): void {
        let sig = `${trees.length}:`;
        for (const t of trees) sig += t.grown ? "1" : "0";
        if (sig === this.signature) return;
        this.signature = sig;
        const n = Math.min(trees.length, MAX_TREES);
        for (let i = 0; i < n; i++) {
            const t = trees[i];
            const h = hash(t.x, t.z);
            const size = (0.85 + 0.35 * h) * (t.grown ? 1 : SAPLING);
            this.q.setFromAxisAngle(this.up, h * Math.PI * 2);
            this.v.set(t.x, this.heightAt(t.x, t.z) - 0.1, t.z);
            this.s.set(size, size * (0.9 + 0.3 * hash(t.z, t.x)), size);
            this.m.compose(this.v, this.q, this.s);
            this.trunks.setMatrixAt(i, this.m);
            this.crowns.setMatrixAt(i, this.m);
        }
        this.trunks.count = n;
        this.crowns.count = n;
        this.trunks.instanceMatrix.needsUpdate = true;
        this.crowns.instanceMatrix.needsUpdate = true;
    }

    dispose(): void {
        for (const m of [this.trunks, this.crowns]) {
            m.geometry.dispose();
            (m.material as THREE.Material).dispose();
        }
    }
}

/** Two stacked cones (a fir crown) as one geometry. */
function mergeCones(): THREE.BufferGeometry {
    const a = new THREE.ConeGeometry(1.6, 3.2, 7).toNonIndexed();
    a.translate(0, 3.4, 0);
    const b = new THREE.ConeGeometry(1.15, 2.6, 7).toNonIndexed();
    b.translate(0, 5.0, 0);
    const pos = new Float32Array([...(a.getAttribute("position").array as Float32Array), ...(b.getAttribute("position").array as Float32Array)]);
    const g = new THREE.BufferGeometry();
    g.setAttribute("position", new THREE.BufferAttribute(pos, 3));
    g.computeVertexNormals();
    a.dispose();
    b.dispose();
    return g;
}

/** Stable pseudo-random 0..1 from a position (variety without state). */
function hash(x: number, z: number): number {
    const s = Math.sin(x * 12.9898 + z * 78.233) * 43758.5453;
    return s - Math.floor(s);
}
