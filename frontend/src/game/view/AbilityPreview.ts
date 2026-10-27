import * as THREE from "three";
import { fanGeometry } from "./Effects";

const AURA_SECONDS = 3;
const AURA_POOL = 8;

/** A flat circle: soft fill plus a brighter rim. Drawn on top (no depth test) so hills don't hide it. */
function circle(color: number, fillOpacity: number): THREE.Group {
    const g = new THREE.Group();
    const rim = new THREE.Mesh(
        new THREE.RingGeometry(0.965, 1, 96).rotateX(-Math.PI / 2),
        new THREE.MeshBasicMaterial({ color, transparent: true, opacity: 0.9, depthTest: false, depthWrite: false }),
    );
    const fill = new THREE.Mesh(
        new THREE.CircleGeometry(0.965, 96).rotateX(-Math.PI / 2),
        new THREE.MeshBasicMaterial({ color, transparent: true, opacity: fillOpacity, depthTest: false, depthWrite: false }),
    );
    rim.renderOrder = fill.renderOrder = 10;
    g.add(fill, rim);
    g.visible = false;
    return g;
}

function setColor(g: THREE.Group, color: number): void {
    for (const m of g.children) ((m as THREE.Mesh).material as THREE.MeshBasicMaterial).color.setHex(color);
}

function setOpacity(g: THREE.Group, k: number): void {
    const [fill, rim] = g.children as THREE.Mesh[];
    (fill.material as THREE.MeshBasicMaterial).opacity = 0.12 * k;
    (rim.material as THREE.MeshBasicMaterial).opacity = 0.9 * k;
}

/**
 * Ability previews (visual only):
 * - while aiming an area ability: its effect radius under the cursor and the
 *   cast range around the caster;
 * - after clicking an aura on the command card: the aura radius around each
 *   unit that has it, for a few seconds.
 * All circles are created once and reused.
 */
export class AbilityPreview {
    readonly group = new THREE.Group();
    private readonly area = circle(0xc89bff, 0.12);
    private readonly range = circle(0xffffff, 0.0);
    private readonly cone = new THREE.Mesh(
        new THREE.BufferGeometry(),
        new THREE.MeshBasicMaterial({ transparent: true, opacity: 0.22, depthTest: false, depthWrite: false, side: THREE.DoubleSide }),
    );
    private coneKey = "";
    private readonly auras: { g: THREE.Group; horde: number; left: number }[] = [];

    constructor() {
        setOpacity(this.range, 0.45);
        this.cone.visible = false;
        this.cone.renderOrder = 10;
        this.group.add(this.area, this.range, this.cone);
        for (let i = 0; i < AURA_POOL; i++) {
            const g = circle(0x7dff7a, 0.08);
            this.group.add(g);
            this.auras.push({ g, horde: 0, left: 0 });
        }
    }

    /** Aiming: area circle at the cursor (radius 0 hides it) and range circle around the caster. */
    aim(cursor: THREE.Vector3 | null, radius: number, color: number, caster: { x: number; y: number; z: number } | null, range: number): void {
        this.area.visible = !!cursor && radius > 0;
        if (cursor && radius > 0) {
            setColor(this.area, color);
            this.area.position.set(cursor.x, cursor.y + 0.25, cursor.z);
            this.area.scale.setScalar(radius);
        }
        this.range.visible = !!caster && range > 0;
        if (caster && range > 0) {
            this.range.position.set(caster.x, caster.y + 0.2, caster.z);
            this.range.scale.setScalar(range);
        }
    }

    stopAiming(): void {
        this.area.visible = false;
        this.range.visible = false;
        this.cone.visible = false;
    }

    /** Aiming a cone: a fan from the caster towards the cursor. */
    aimCone(cursor: THREE.Vector3 | null, caster: { x: number; y: number; z: number } | null, length: number, degrees: number, color: number): void {
        this.area.visible = false;
        this.range.visible = false;
        this.cone.visible = !!cursor && !!caster;
        if (!cursor || !caster) return;
        const facing = Math.atan2(cursor.x - caster.x, cursor.z - caster.z);
        const key = `${Math.round(facing * 50)}:${degrees}`; // rebuild only when it turns noticeably
        if (key !== this.coneKey) {
            this.coneKey = key;
            this.cone.geometry.dispose();
            this.cone.geometry = fanGeometry(facing, (degrees * Math.PI) / 180);
        }
        (this.cone.material as THREE.MeshBasicMaterial).color.setHex(color);
        this.cone.position.set(caster.x, caster.y + 0.25, caster.z);
        this.cone.scale.setScalar(length);
    }

    /** Shows an aura's radius around a horde for a few seconds. */
    showAura(horde: number, radius: number): void {
        const slot = this.auras.find((a) => a.horde === horde) ?? this.auras.find((a) => a.left <= 0) ?? this.auras[0];
        slot.horde = horde;
        slot.left = AURA_SECONDS;
        slot.g.scale.setScalar(radius);
        slot.g.visible = true;
    }

    /** Moves aura circles with their units and fades them out. */
    update(dt: number, posOf: (horde: number) => { x: number; y: number; z: number } | null): void {
        for (const a of this.auras) {
            if (a.left <= 0) continue;
            a.left -= dt;
            const p = posOf(a.horde);
            if (!p || a.left <= 0) {
                a.left = 0;
                a.g.visible = false;
                continue;
            }
            a.g.position.set(p.x, p.y + 0.25, p.z);
            setOpacity(a.g, Math.min(1, a.left / 0.5));
        }
    }

    dispose(): void {
        this.group.traverse((o) => {
            if (o instanceof THREE.Mesh) {
                o.geometry.dispose();
                (o.material as THREE.Material).dispose();
            }
        });
    }
}
