import * as THREE from "three";
import { StaticModel } from "../../engine/models/StaticModel";

/**
 * Machines (ballistas, trebuchets, catapults, rams) drawn with static models
 * (tools/meshbake, moving parts kept as named nodes) instead of VAT soldiers.
 * There are few of them, so each gets its own LOD object; the parts are moved
 * here: wheels roll with the distance driven, a trebuchet / catapult arm
 * swings when it shoots (counterweight and sling follow the arm), a ballista's
 * bow kicks back, a ram's log swings.
 */

interface Parts {
    wheels: { o: THREE.Object3D; r: number }[];
    arm: THREE.Object3D[]; // throwing arm (rotates about its own pivot, local x)
    riders: { o: THREE.Object3D; base: THREE.Vector3; pivot: THREE.Vector3; turns: boolean }[]; // follow the arm
    recoil: { o: THREE.Object3D; z: number }[]; // ballista bow / ram log
}

interface Machine {
    obj: THREE.LOD;
    parts: Parts;
    x: number;
    z: number;
    roll: number; // wheel angle
    shotAt: number; // ms when the current shot started (-1 = none)
    seen: number; // frame stamp
}

const ARM_SWING = -2.4; // radians: the long end goes up and over the front
const SWING_UP_MS = 450;
const SWING_BACK_MS = 2200;
const RECOIL_MS = 350;

export class MachinesView {
    readonly group = new THREE.Group();
    private readonly live = new Map<number, Machine>();
    private frame = 0;

    constructor(private readonly models: Map<number, StaticModel>) {} // by horde kind

    /** Loads the static models of the horde types that have one (mesh id per kind, "" = none). */
    static async load(meshes: string[]): Promise<MachinesView> {
        const byId = new Map<string, StaticModel>();
        await Promise.all(
            [...new Set(meshes.filter((m) => m))].map(async (id) => {
                try {
                    byId.set(id, await StaticModel.load(id));
                } catch (err) {
                    console.warn(`machine model ${id}:`, err);
                }
            }),
        );
        const models = new Map<number, StaticModel>();
        meshes.forEach((id, kind) => {
            const m = id && byId.get(id);
            if (m) models.set(kind, m);
        });
        return new MachinesView(models);
    }

    /** Whether soldiers of this horde kind are drawn here. */
    has(kind: number): boolean {
        return this.models.has(kind);
    }

    begin(): void {
        this.frame++;
    }

    /** One machine this frame (soldier id, render position, facing, shooting since `shotStart` ms or -1). */
    place(id: number, kind: number, x: number, y: number, z: number, rot: number, scale: number, shotStart: number, now: number): void {
        let m = this.live.get(id);
        if (!m) {
            const model = this.models.get(kind)!;
            const obj = model.instance(1);
            m = { obj, parts: findParts(obj), x, z, roll: 0, shotAt: -1, seen: 0 };
            this.live.set(id, m);
            this.group.add(obj);
        }
        m.seen = this.frame;
        const moved = Math.hypot(x - m.x, z - m.z);
        // Driving backwards is rare for an RTS unit: wheels roll forward.
        m.roll += moved;
        m.x = x;
        m.z = z;
        m.obj.position.set(x, y, z);
        m.obj.rotation.y = rot;
        m.obj.scale.setScalar(scale);
        if (shotStart >= 0 && shotStart !== m.shotAt) m.shotAt = shotStart;
        pose(m, now);
    }

    /** Drops machines not placed this frame (destroyed or gone). */
    end(): void {
        for (const [id, m] of this.live) {
            if (m.seen === this.frame) continue;
            this.group.remove(m.obj);
            this.live.delete(id);
        }
    }

    dispose(): void {
        for (const m of this.live.values()) this.group.remove(m.obj);
        this.live.clear();
        for (const model of new Set(this.models.values())) model.dispose();
    }
}

function findParts(obj: THREE.Object3D): Parts {
    const p: Parts = { wheels: [], arm: [], riders: [], recoil: [] };
    const arms: THREE.Object3D[] = [];
    obj.traverse((o) => {
        const n = o.name;
        if (/_wheel_/.test(n)) {
            const box = new THREE.Box3().setFromObject(o);
            p.wheels.push({ o, r: Math.max(0.1, (box.max.y - box.min.y) / 2) });
        } else if (/_arm$/.test(n)) {
            arms.push(o);
        } else if (/_crossbow$|_ram_log$/.test(n)) {
            p.recoil.push({ o, z: o.position.z });
        }
    });
    p.arm = arms;
    // Counterweight and sling hang off the arm: they follow its rotation about the arm's pivot.
    obj.traverse((o) => {
        if (!/_counterweight$|_sling$/.test(o.name)) return;
        const arm = arms.find((a) => a.parent === o.parent);
        if (!arm) return;
        p.riders.push({ o, base: o.position.clone(), pivot: arm.position.clone(), turns: /_sling$/.test(o.name) });
    });
    return p;
}

const tmp = new THREE.Vector3();
const axisX = new THREE.Vector3(1, 0, 0);

function pose(m: Machine, now: number): void {
    const p = m.parts;
    for (const w of p.wheels) {
        w.o.rotation.x = m.roll / w.r;
        w.o.updateMatrix();
    }
    const t = m.shotAt >= 0 ? now - m.shotAt : Infinity;
    let a = 0;
    if (t < SWING_UP_MS) {
        const k = t / SWING_UP_MS;
        a = ARM_SWING * k * k; // released: accelerates
    } else if (t < SWING_UP_MS + SWING_BACK_MS) {
        const k = (t - SWING_UP_MS) / SWING_BACK_MS;
        a = ARM_SWING * (1 - k * k * (3 - 2 * k)); // wound back down
    }
    for (const o of p.arm) {
        o.rotation.x = a;
        o.updateMatrix();
    }
    for (const r of p.riders) {
        tmp.copy(r.base).sub(r.pivot).applyAxisAngle(axisX, a).add(r.pivot);
        r.o.position.copy(tmp);
        r.o.rotation.x = r.turns ? a : 0; // the counterweight keeps hanging down
        r.o.updateMatrix();
    }
    const kick = t < RECOIL_MS ? Math.sin((Math.PI * t) / RECOIL_MS) : 0;
    for (const r of p.recoil) {
        r.o.position.z = r.z - 0.25 * kick;
        r.o.updateMatrix();
    }
}
