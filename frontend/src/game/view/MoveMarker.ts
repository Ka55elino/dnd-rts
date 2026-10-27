import * as THREE from "three";

const DURATION = 0.6; // seconds

/** Short expanding ring where a move order was given. */
export class MoveMarker {
    readonly mesh: THREE.Mesh;
    private age = DURATION;

    constructor() {
        const geo = new THREE.RingGeometry(0.8, 1.1, 32).rotateX(-Math.PI / 2);
        const mat = new THREE.MeshBasicMaterial({
            color: 0x7dff7a,
            transparent: true,
            depthWrite: false,
            polygonOffset: true,
            polygonOffsetFactor: -2,
        });
        this.mesh = new THREE.Mesh(geo, mat);
        this.mesh.visible = false;
    }

    show(x: number, y: number, z: number, color = 0x7dff7a): void {
        (this.mesh.material as THREE.MeshBasicMaterial).color.set(color);
        this.mesh.position.set(x, y + 0.1, z);
        this.age = 0;
        this.mesh.visible = true;
    }

    update(dt: number): void {
        if (this.age >= DURATION) return;
        this.age += dt;
        const t = Math.min(this.age / DURATION, 1);
        this.mesh.scale.setScalar(1 + t * 2);
        (this.mesh.material as THREE.MeshBasicMaterial).opacity = 1 - t;
        this.mesh.visible = t < 1;
    }
}
