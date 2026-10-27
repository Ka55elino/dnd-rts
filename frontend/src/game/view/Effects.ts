import * as THREE from "three";

const POOL = 32;
const BOLTS = 48; // chain jumps on screen at once
const BOLT_POINTS = 5; // jagged polyline: 4 segments
const BOLT_SECONDS = 0.35;
const CONES = 8;

/**
 * Flat fan (cone of an ability) of unit radius, `angle` radians wide, centred
 * on three.js Y rotation `facing` (forward = (sin r, cos r), like soldiers).
 */
export function fanGeometry(facing: number, angle: number): THREE.BufferGeometry {
    const half = angle / 2;
    // CircleGeometry's θ maps to (cos θ, −sin θ) on the ground after rotateX(−π/2).
    return new THREE.CircleGeometry(1, 24, facing - Math.PI / 2 - half, angle).rotateX(-Math.PI / 2);
}

interface Cone {
    mesh: THREE.Mesh;
    age: number;
}

interface Wave {
    mesh: THREE.Mesh;
    age: number;
    duration: number;
    radius: number;
    linger: boolean; // zones: stay at full size, fade out over the duration
}

/** Short-lived world effects: expanding rings, chain lightning bolts, cone fans. All pooled. */
export class Effects {
    readonly group = new THREE.Group();
    private readonly waves: Wave[] = [];
    private readonly boltPos = new Float32Array(BOLTS * (BOLT_POINTS - 1) * 2 * 3);
    private readonly boltCol = new Float32Array(BOLTS * (BOLT_POINTS - 1) * 2 * 3);
    private readonly boltAge = new Float32Array(BOLTS).fill(BOLT_SECONDS);
    private readonly boltColor: THREE.Color[] = Array.from({ length: BOLTS }, () => new THREE.Color());
    private readonly bolts: THREE.LineSegments;
    private nextBolt = 0;
    private readonly cones: Cone[] = [];
    private readonly tmpColor = new THREE.Color();

    constructor() {
        const bg = new THREE.BufferGeometry();
        bg.setAttribute("position", new THREE.BufferAttribute(this.boltPos, 3).setUsage(THREE.DynamicDrawUsage));
        bg.setAttribute("color", new THREE.BufferAttribute(this.boltCol, 3).setUsage(THREE.DynamicDrawUsage));
        this.bolts = new THREE.LineSegments(
            bg,
            new THREE.LineBasicMaterial({ vertexColors: true, transparent: true, blending: THREE.AdditiveBlending, depthWrite: false }),
        );
        this.bolts.frustumCulled = false;
        this.group.add(this.bolts);
        for (let i = 0; i < CONES; i++) {
            const mesh = new THREE.Mesh(
                new THREE.BufferGeometry(),
                new THREE.MeshBasicMaterial({ transparent: true, depthWrite: false, side: THREE.DoubleSide, polygonOffset: true, polygonOffsetFactor: -3 }),
            );
            mesh.visible = false;
            this.group.add(mesh);
            this.cones.push({ mesh, age: 1 });
        }

        const geo = new THREE.RingGeometry(0.85, 1, 48).rotateX(-Math.PI / 2);
        for (let i = 0; i < POOL; i++) {
            const mat = new THREE.MeshBasicMaterial({
                color: 0xffc860,
                transparent: true,
                depthWrite: false,
                side: THREE.DoubleSide,
                polygonOffset: true,
                polygonOffsetFactor: -3,
            });
            const mesh = new THREE.Mesh(geo, mat);
            mesh.visible = false;
            this.group.add(mesh);
            this.waves.push({ mesh, age: 1, duration: 1, radius: 1, linger: false });
        }
    }

    shockwave(x: number, y: number, z: number, radius: number): void {
        this.cast(x, y, z, radius, 0xffc860, 0);
    }

    /**
     * Visual of an ability: an expanding ring (instant) or, for zones
     * (duration > 0), a ring that stays at full size and fades out. Rings come
     * from a fixed pool; the oldest is reused when all are busy.
     */
    cast(x: number, y: number, z: number, radius: number, color: number, duration: number): void {
        const w = this.waves.find((w) => !w.mesh.visible) ?? this.waves.reduce((a, b) => (a.age / a.duration > b.age / b.duration ? a : b));
        w.mesh.position.set(x, y + 0.3, z);
        (w.mesh.material as THREE.MeshBasicMaterial).color.setHex(color);
        w.age = 0;
        w.linger = duration > 0;
        w.duration = duration > 0 ? duration : 0.55;
        w.radius = Math.max(radius, 1);
        w.mesh.visible = true;
    }

    /** One jump of a chain: a jagged bolt from a to b (heights included). */
    bolt(ax: number, ay: number, az: number, bx: number, by: number, bz: number, color: number): void {
        const i = this.nextBolt;
        this.nextBolt = (i + 1) % BOLTS;
        this.boltAge[i] = 0;
        this.boltColor[i].setHex(color);
        const len = Math.hypot(bx - ax, bz - az);
        // Sideways direction for the zigzag.
        const sx = len > 0 ? -(bz - az) / len : 1;
        const sz = len > 0 ? (bx - ax) / len : 0;
        const pts: number[] = [];
        for (let k = 0; k < BOLT_POINTS; k++) {
            const t = k / (BOLT_POINTS - 1);
            const j = k === 0 || k === BOLT_POINTS - 1 ? 0 : (Math.random() - 0.5) * Math.min(len * 0.25, 2);
            pts.push(ax + (bx - ax) * t + sx * j, ay + (by - ay) * t + 1.2 + Math.abs(j) * 0.5, az + (bz - az) * t + sz * j);
        }
        let o = i * (BOLT_POINTS - 1) * 6;
        for (let k = 0; k < BOLT_POINTS - 1; k++) {
            for (let v = 0; v < 2; v++) {
                const p = (k + v) * 3;
                this.boltPos[o++] = pts[p];
                this.boltPos[o++] = pts[p + 1];
                this.boltPos[o++] = pts[p + 2];
            }
        }
        (this.bolts.geometry.getAttribute("position") as THREE.BufferAttribute).needsUpdate = true;
    }

    /** A cone fan from (x, y, z): `length` long, `degrees` wide, facing Y rotation `facing`. */
    cone(x: number, y: number, z: number, length: number, degrees: number, facing: number, color: number): void {
        const c = this.cones.find((c) => !c.mesh.visible) ?? this.cones.reduce((a, b) => (a.age > b.age ? a : b));
        c.mesh.geometry.dispose();
        c.mesh.geometry = fanGeometry(facing, (degrees * Math.PI) / 180);
        c.mesh.position.set(x, y + 0.3, z);
        c.mesh.scale.setScalar(Math.max(length, 0.5));
        (c.mesh.material as THREE.MeshBasicMaterial).color.setHex(color);
        c.age = 0;
        c.mesh.visible = true;
    }

    update(dt: number): void {
        // Bolts: fade by darkening (additive blending: black = invisible).
        let o = 0;
        for (let i = 0; i < BOLTS; i++) {
            const a = this.boltAge[i];
            const k = a < BOLT_SECONDS ? 1 - a / BOLT_SECONDS : 0;
            this.boltAge[i] = Math.min(a + dt, BOLT_SECONDS);
            const c = this.tmpColor.copy(this.boltColor[i]).multiplyScalar(k * (0.7 + Math.random() * 0.3));
            for (let v = 0; v < (BOLT_POINTS - 1) * 2; v++) {
                this.boltCol[o++] = c.r;
                this.boltCol[o++] = c.g;
                this.boltCol[o++] = c.b;
            }
        }
        (this.bolts.geometry.getAttribute("color") as THREE.BufferAttribute).needsUpdate = true;
        for (const c of this.cones) {
            if (!c.mesh.visible) continue;
            c.age += dt;
            const t = Math.min(c.age / 0.6, 1);
            (c.mesh.material as THREE.MeshBasicMaterial).opacity = 0.55 * (1 - t);
            if (t >= 1) c.mesh.visible = false;
        }

        for (const w of this.waves) {
            if (!w.mesh.visible) continue;
            w.age += dt;
            const t = Math.min(w.age / w.duration, 1);
            const eased = w.linger ? Math.min(w.age / 0.3, 1) : 1 - (1 - t) * (1 - t);
            w.mesh.scale.setScalar(Math.max(0.01, eased * w.radius));
            (w.mesh.material as THREE.MeshBasicMaterial).opacity = w.linger ? 0.8 * (1 - t * t) : 1 - t;
            if (t >= 1) w.mesh.visible = false;
        }
    }
}
