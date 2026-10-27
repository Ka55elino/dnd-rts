import * as THREE from "three";
import { plotGeometry, type BuildingTypeInfo, type BuildingsView } from "../view/BuildingsView";

/** Rotation step for R / Shift+R. */
const ROTATE_STEP = Math.PI / 4;

export interface NavBlocked {
    cols: number;
    rows: number;
    cellSize: number;
    blocked: Uint8Array; // terrain only, row-major
}

export interface PlacementDeps {
    scene: THREE.Object3D;
    buildings: BuildingsView;
    types: BuildingTypeInfo[];
    heightAt: (x: number, z: number) => number;
    mapWidth: number;
    mapHeight: number;
    nav: NavBlocked | null;
    localPlayer: () => number;
    /** Place: world x/z and the facing in simulation convention (front = (cos a, sin a)). */
    onPlace: (kind: number, x: number, z: number, angle: number, workers: number[]) => void;
}

/**
 * Free-building placement (workers): a ghost of the building follows the
 * cursor, green where it can stand and red where it can't. No-build circles
 * of existing buildings (and of the ghost itself) are drawn while placing.
 * Left click places (Shift keeps placing), right click / Esc cancels,
 * R / Shift+R rotates.
 *
 * The check mirrors sim.CanPlaceFree; the server has the final word.
 */
export class Placement {
    private active: { kind: number; workers: number[]; type: BuildingTypeInfo } | null = null;
    private readonly group = new THREE.Group();
    private ghost: THREE.Object3D | null = null;
    private readonly outline: THREE.Mesh;
    private readonly zones = new THREE.Group();
    private readonly okMat = translucent(0x7dff7a, 0.45);
    private readonly badMat = translucent(0xff5040, 0.45);
    private readonly zoneMat = translucent(0xff8a5a, 0.55);
    private readonly ownZone: THREE.Mesh;
    private readonly outlineGeo = plotGeometry();
    private readonly zoneGeo = new THREE.RingGeometry(0.985, 1, 96).rotateX(-Math.PI / 2);
    private rotation = 0; // manual offset added to the default (face the map center)
    private x = 0;
    private z = 0;
    private valid = false;

    constructor(private readonly d: PlacementDeps) {
        this.outline = new THREE.Mesh(this.outlineGeo, this.okMat);
        this.ownZone = new THREE.Mesh(this.zoneGeo, this.okMat);
        this.group.add(this.outline, this.zones, this.ownZone);
        this.group.visible = false;
        d.scene.add(this.group);
    }

    get placing(): boolean {
        return this.active !== null;
    }

    start(kind: number, workers: number[]): void {
        const type = this.d.types[kind];
        if (!type || workers.length === 0) return;
        this.cancel();
        this.active = { kind, workers, type };
        this.ghost = this.d.buildings.ghost(kind, this.okMat);
        this.group.add(this.ghost);
        this.outline.scale.set(type.width, 1, type.depth);
        this.ownZone.visible = type.noBuildRadius > 0;
        this.ownZone.scale.setScalar(Math.max(type.noBuildRadius, 0.01));
        this.rebuildZones();
        this.group.visible = true;
    }

    cancel(): void {
        if (this.ghost) this.group.remove(this.ghost);
        this.ghost = null;
        this.active = null;
        this.group.visible = false;
    }

    rotate(dir: 1 | -1): void {
        this.rotation += dir * ROTATE_STEP;
        this.update(this.x, this.z);
    }

    /** Follow the cursor (ground point). */
    update(x: number, z: number): void {
        const a = this.active;
        if (!a || !this.ghost) return;
        this.x = x;
        this.z = z;
        const angle = this.angle();
        this.valid = this.check(a.type, x, z, angle);
        const rot = Math.PI / 2 - angle; // three.js Y rotation (model faces +Z)
        const y = this.groundFor(a.type, x, z, rot);
        this.ghost.position.set(x, y, z);
        this.ghost.rotation.y = rot;
        this.outline.position.set(x, y + 0.08, z);
        this.outline.rotation.y = rot;
        const mat = this.valid ? this.okMat : this.badMat;
        this.outline.material = mat;
        this.ownZone.position.set(x, y + 0.3, z);
        this.ownZone.material = mat;
        this.ghost.traverse((o) => {
            if (o instanceof THREE.Mesh) o.material = mat;
        });
    }

    /** Left click: place if valid. Returns true when placement mode should end. */
    place(keep: boolean): boolean {
        const a = this.active;
        if (!a) return true;
        if (!this.valid) return false;
        this.d.onPlace(a.kind, this.x, this.z, this.angle(), a.workers);
        if (!keep) this.cancel();
        return !keep;
    }

    dispose(): void {
        this.cancel();
        this.d.scene.remove(this.group);
        for (const m of [this.okMat, this.badMat, this.zoneMat]) m.dispose();
        this.outlineGeo.dispose();
        this.zoneGeo.dispose();
    }

    private angle(): number {
        const base = this.x === 0 && this.z === 0 ? 0 : Math.atan2(-this.z, -this.x);
        return base + this.rotation;
    }

    private rebuildZones(): void {
        this.zones.clear();
        for (const { snap, type } of this.d.buildings.all()) {
            if (!type || type.noBuildRadius <= 0) continue;
            const ring = new THREE.Mesh(this.zoneGeo, this.zoneMat);
            ring.scale.setScalar(type.noBuildRadius);
            ring.position.set(snap.x, this.d.heightAt(snap.x, snap.z) + 0.3, snap.z);
            this.zones.add(ring);
        }
    }

    /** Lowest terrain point under the footprint, so the ghost sits like the real building. */
    private groundFor(type: BuildingTypeInfo, x: number, z: number, rot: number): number {
        const c = Math.cos(rot);
        const s = Math.sin(rot);
        let y = Infinity;
        for (const [dx, dz] of [[-1, -1], [1, -1], [-1, 1], [1, 1], [0, 0]]) {
            const lx = (dx * type.width) / 2;
            const lz = (dz * type.depth) / 2;
            y = Math.min(y, this.d.heightAt(x + lx * c + lz * s, z - lx * s + lz * c));
        }
        return y;
    }

    /** Mirror of sim.CanPlaceFree (floats). `angle` in simulation convention. */
    private check(type: BuildingTypeInfo, x: number, z: number, angle: number): boolean {
        const ghost = rect(x, z, angle, type.width / 2, type.depth / 2);

        const nav = this.d.nav;
        const cell = nav?.cellSize ?? 4;
        const r = ghost.hx + ghost.hz;
        const hw = this.d.mapWidth / 2 - cell;
        const hh = this.d.mapHeight / 2 - cell;
        if (x - r < -hw || x + r > hw || z - r < -hh || z + r > hh) return false;

        if (nav) {
            const c0 = Math.max(0, Math.floor((x - r + this.d.mapWidth / 2) / cell));
            const c1 = Math.min(nav.cols - 1, Math.floor((x + r + this.d.mapWidth / 2) / cell));
            const r0 = Math.max(0, Math.floor((z - r + this.d.mapHeight / 2) / cell));
            const r1 = Math.min(nav.rows - 1, Math.floor((z + r + this.d.mapHeight / 2) / cell));
            for (let row = r0; row <= r1; row++) {
                for (let col = c0; col <= c1; col++) {
                    if (!nav.blocked[row * nav.cols + col]) continue;
                    const cx = (col + 0.5) * cell - this.d.mapWidth / 2;
                    const cz = (row + 0.5) * cell - this.d.mapHeight / 2;
                    if (covers(ghost, cx, cz, cell / 2)) return false;
                }
            }
        }

        for (const { snap, type: t } of this.d.buildings.all()) {
            if (!t) continue;
            const a = Math.PI / 2 - snap.rot; // back to simulation convention
            const other = rect(snap.x, snap.z, a, t.width / 2, t.depth / 2);
            if (t.noBuildRadius > 0 && distTo(ghost, snap.x, snap.z) < t.noBuildRadius) return false;
            if (type.noBuildRadius > 0 && distTo(other, x, z) < type.noBuildRadius) return false;
            if (overlaps(ghost, other)) return false;
            const right = { x: Math.sin(a), z: -Math.cos(a) };
            const fwd = { x: Math.cos(a), z: Math.sin(a) };
            for (const p of t.plots) {
                const px = snap.x + right.x * p.x + fwd.x * p.y;
                const pz = snap.z + right.z * p.x + fwd.z * p.y;
                if (overlaps(ghost, rect(px, pz, a, p.size / 2, p.size / 2))) return false;
            }
        }
        return true;
    }
}

interface Rect {
    x: number;
    z: number;
    rx: number; // right axis
    rz: number;
    fx: number; // front axis
    fz: number;
    hx: number; // half width (along right)
    hz: number; // half depth (along front)
}

/** Oriented rectangle; angle in simulation convention (front = (cos a, sin a), right = (sin a, -cos a)). */
function rect(x: number, z: number, a: number, hx: number, hz: number): Rect {
    const fx = Math.cos(a);
    const fz = Math.sin(a);
    return { x, z, rx: fz, rz: -fx, fx, fz, hx, hz };
}

function covers(r: Rect, px: number, pz: number, margin: number): boolean {
    const dx = px - r.x;
    const dz = pz - r.z;
    return Math.abs(dx * r.rx + dz * r.rz) <= r.hx + margin && Math.abs(dx * r.fx + dz * r.fz) <= r.hz + margin;
}

/** Distance from a point to the rectangle (0 inside). */
function distTo(r: Rect, px: number, pz: number): number {
    const dx = px - r.x;
    const dz = pz - r.z;
    const lx = dx * r.rx + dz * r.rz;
    const lz = dx * r.fx + dz * r.fz;
    const ox = Math.max(Math.abs(lx) - r.hx, 0);
    const oz = Math.max(Math.abs(lz) - r.hz, 0);
    return Math.hypot(ox, oz);
}

function overlaps(a: Rect, b: Rect): boolean {
    const dx = b.x - a.x;
    const dz = b.z - a.z;
    for (const [ax, az] of [
        [a.rx, a.rz],
        [a.fx, a.fz],
        [b.rx, b.rz],
        [b.fx, b.fz],
    ]) {
        const ra = a.hx * Math.abs(a.rx * ax + a.rz * az) + a.hz * Math.abs(a.fx * ax + a.fz * az);
        const rb = b.hx * Math.abs(b.rx * ax + b.rz * az) + b.hz * Math.abs(b.fx * ax + b.fz * az);
        if (Math.abs(dx * ax + dz * az) >= ra + rb) return false;
    }
    return true;
}

function translucent(color: number, opacity: number): THREE.MeshBasicMaterial {
    return new THREE.MeshBasicMaterial({
        color,
        transparent: true,
        opacity,
        depthWrite: false,
        side: THREE.DoubleSide,
        polygonOffset: true,
        polygonOffsetFactor: -2,
    });
}
