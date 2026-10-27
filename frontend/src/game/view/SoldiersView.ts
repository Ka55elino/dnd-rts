import * as THREE from "three";
import { mergeGeometries } from "three/examples/jsm/utils/BufferGeometryUtils.js";
import {
    STATUS_GROUNDED,
    STATUS_ON_WALL,
    SOLDIER_WORDS,
    SoldierAttacking,
    SoldierKnocked,
    SoldierMoving,
    type GameSnapshot,
    type HordeSnap,
} from "../../net/protocol";
import { type VatClip, type VatModel } from "../../engine/vat/VatModel";
import { VatLodSet } from "../../engine/vat/VatLodSet";
import type { MachinesView } from "./MachinesView";
import { FLY_BOB, FLY_HEIGHT, GROUNDED_HEIGHT } from "./flight";

export const PLAYER_COLORS = [0x3a6ea5, 0xa53a3a, 0x3aa55a, 0xa5903a];
const SELECTION_COLOR = 0x7dff7a;
const HERO_TINT = new THREE.Color(0xffd36a);
const BASE_RADIUS = 0.4; // placeholder model is built for this unit radius

const CORPSE_CAPACITY = 3000;
const CORPSE_LIFETIME = 20; // seconds
const CORPSE_SINK = 4; // last seconds: sink into the ground
const LUNGE = 0.35; // placeholder attack animation: forward lunge distance
const SWING_MS = 300; // attack animation length (internal/sim swingTicks * 50 ms)
const KNOCK_MS = 1000; // knockdown length (internal/sim knockdownTicks * 50 ms)
const RUN_THRESHOLD = 1.8; // use the run clip when walk would play faster than this

// Client-side animation clips for VAT units.
const CLIP_IDLE = 0;
const CLIP_WALK = 1;
const CLIP_RUN = 2;
const CLIP_ATTACK = 3;
const CLIP_NAMES = ["idle", "walk", "run", "attack"];

export interface HordeTypeInfo {
    name: string;
    size: number;
    radius: number;
    hero: boolean;
    abilities: string[]; // ability ids in slot order (ContentService.Abilities)
    model: string; // VAT model id, "" = placeholder
    speed: number; // units per second
    worker?: boolean; // can construct free buildings
    heroXp?: number[]; // total XP per level (index 0 = level 1)
    flying?: boolean; // drawn FLY_HEIGHT above the ground, with a shadow
    ranged?: boolean; // has a ranged attack (attack-ground order)
    antiAir?: boolean; // an attack hits flyers
    climbs?: boolean; // goes up walls and watchtowers
}

/** Placeholder soldier: body, head and a shield on the +Z (facing) side. */
function soldierGeometry(): THREE.BufferGeometry {
    const body = new THREE.CylinderGeometry(0.32, 0.38, 1.3, 8).translate(0, 0.65, 0);
    const head = new THREE.SphereGeometry(0.22, 8, 6).translate(0, 1.5, 0);
    const shield = new THREE.BoxGeometry(0.5, 0.7, 0.08).translate(0, 0.85, 0.38);
    const merged = mergeGeometries([body, head, shield])!;
    body.dispose();
    head.dispose();
    shield.dispose();
    return merged;
}

/** How a horde type is drawn: a VAT model (with its clips) or the placeholder. */
interface KindRender {
    vat: VatLodSet | null;
    clips: (VatClip | undefined)[]; // indexed by CLIP_*
    rates: number[]; // playback rate per clip (locomotion matched to unit speed)
    die: VatClip | undefined;
}

/** Per-soldier render state for one snapshot (struct of arrays). */
class Frame {
    count = 0;
    ids: Uint32Array;
    hordeOf: Uint32Array;
    kind: Uint8Array;
    owner: Uint8Array;
    hero: Uint8Array;
    scale: Float32Array;
    state: Uint8Array;
    status: Uint16Array; // effect bits (STATUS_*)
    swingStart: Float64Array; // ms: start of the current attack / knockdown state
    clip: Uint8Array; // VAT units: current CLIP_*
    clipStart: Float64Array; // VAT units: ms when the clip started
    pos: Float32Array; // x, z, rot

    constructor(capacity: number) {
        this.ids = new Uint32Array(capacity);
        this.hordeOf = new Uint32Array(capacity);
        this.kind = new Uint8Array(capacity);
        this.owner = new Uint8Array(capacity);
        this.hero = new Uint8Array(capacity);
        this.scale = new Float32Array(capacity);
        this.state = new Uint8Array(capacity);
        this.status = new Uint16Array(capacity);
        this.swingStart = new Float64Array(capacity);
        this.clip = new Uint8Array(capacity);
        this.clipStart = new Float64Array(capacity);
        this.pos = new Float32Array(capacity * 3);
    }
}

/**
 * Renders all soldiers, interpolating between the two latest snapshots
 * (matched by soldier id, so deaths don't break smoothing). Unit types with a
 * baked model use a VAT renderer (one draw call per type, animated on the GPU);
 * the rest use the placeholder mesh. Also draws selection rings and corpses.
 */
export class SoldiersView {
    readonly mesh: THREE.InstancedMesh;
    readonly rings: THREE.InstancedMesh;
    /** Ground shadows under flyers. */
    readonly shadows: THREE.InstancedMesh;
    readonly corpses: THREE.InstancedMesh;
    /** Scene objects to add (placeholder + VAT meshes + rings + corpses). */
    readonly objects: THREE.Object3D[];

    /** Hordes from the latest snapshot. */
    readonly hordes = new Map<number, HordeSnap>();

    /** Interpolated render positions of the last frame (x, y, z per soldier), for picking. */
    renderPos: Float32Array;

    private curr: Frame;
    private next: Frame;
    /** Previous position of each soldier in `curr`, aligned by index (x, z, rot). */
    private prevPos: Float32Array;
    private currTime = 0;
    private readonly color = new THREE.Color();
    private readonly kinds: KindRender[];
    private readonly renderers: VatLodSet[] = [];

    // Corpses: ring buffer.
    private corpseData: Float32Array; // x, z, rot, scale, owner, bornAt(s), kind
    private corpseHead = 0;
    private corpseCount = 0;
    private readonly tmpMatrix = new THREE.Matrix4();
    private readonly tmpQuat = new THREE.Quaternion();
    private readonly tmpPos = new THREE.Vector3();
    private readonly tmpScale = new THREE.Vector3();
    private readonly tmpLie = new THREE.Quaternion();
    private readonly qLie = new THREE.Quaternion().setFromAxisAngle(new THREE.Vector3(1, 0, 0), -Math.PI / 2);
    private readonly up = new THREE.Vector3(0, 1, 0);
    private readonly axisX = new THREE.Vector3(1, 0, 0);
    private readonly teamColors = PLAYER_COLORS.map((c) => new THREE.Color(c));
    /** Walkway height of base walls: soldiers on a wall are drawn this high. */
    wallHeight = 0;
    /** Height of what a soldier "on a wall" stands on (walkway or watchtower), set by the game. */
    standHeight: ((x: number, z: number) => number) | null = null;
    /** Machines (static models with moving parts), set by the game. */
    machines: MachinesView | null = null;
    /** Current (eased) flight height per flying horde. */
    private readonly lift = new Map<number, { y: number; at: number }>();
    private lastFrame = 0;

    constructor(
        private readonly capacity: number,
        private readonly tickMs: number,
        private readonly types: HordeTypeInfo[],
        models: ReadonlyMap<string, VatModel> = new Map(),
    ) {
        const geometry = soldierGeometry();

        this.mesh = new THREE.InstancedMesh(geometry, new THREE.MeshLambertMaterial(), capacity);
        this.mesh.instanceMatrix.setUsage(THREE.DynamicDrawUsage);
        this.mesh.count = 0;
        this.mesh.frustumCulled = false;
        this.mesh.setColorAt(0, this.color.set(0xffffff)); // allocate instanceColor
        this.mesh.instanceColor!.setUsage(THREE.DynamicDrawUsage);

        this.corpses = new THREE.InstancedMesh(geometry, new THREE.MeshLambertMaterial(), CORPSE_CAPACITY);
        this.corpses.instanceMatrix.setUsage(THREE.DynamicDrawUsage);
        this.corpses.count = 0;
        this.corpses.frustumCulled = false;
        this.corpses.setColorAt(0, this.color);
        this.corpses.instanceColor!.setUsage(THREE.DynamicDrawUsage);

        const ringGeo = new THREE.RingGeometry(0.5, 0.62, 20).rotateX(-Math.PI / 2);
        const ringMat = new THREE.MeshBasicMaterial({
            color: SELECTION_COLOR,
            transparent: true,
            opacity: 0.85,
            depthWrite: false,
            polygonOffset: true,
            polygonOffsetFactor: -2,
        });
        this.rings = new THREE.InstancedMesh(ringGeo, ringMat, capacity);
        this.rings.instanceMatrix.setUsage(THREE.DynamicDrawUsage);
        this.rings.count = 0;
        this.rings.frustumCulled = false;

        const shadowGeo = new THREE.CircleGeometry(0.6, 16).rotateX(-Math.PI / 2);
        const shadowMat = new THREE.MeshBasicMaterial({
            color: 0x000000,
            transparent: true,
            opacity: 0.35,
            depthWrite: false,
            polygonOffset: true,
            polygonOffsetFactor: -1,
        });
        this.shadows = new THREE.InstancedMesh(shadowGeo, shadowMat, capacity);
        this.shadows.instanceMatrix.setUsage(THREE.DynamicDrawUsage);
        this.shadows.count = 0;
        this.shadows.frustumCulled = false;

        // One VAT renderer per model; horde kinds sharing a model share it.
        const byModel = new Map<string, VatLodSet>();
        this.kinds = types.map((t) => {
            const model = t.model ? models.get(t.model) : undefined;
            if (!model) return { vat: null, clips: [], rates: [], die: undefined };
            let vat = byModel.get(model.id);
            if (!vat) {
                vat = new VatLodSet(model, capacity + CORPSE_CAPACITY);
                byModel.set(model.id, vat);
                this.renderers.push(vat);
            }
            const clips = CLIP_NAMES.map((n) => model.clip(n));
            const rates = clips.map((c) => (c?.groundSpeed ? t.speed / c.groundSpeed : 1));
            return { vat, clips, rates, die: model.clip("die") };
        });

        this.objects = [this.mesh, this.corpses, this.rings, this.shadows, ...this.renderers.flatMap((r) => r.meshes)];
        this.curr = new Frame(capacity);
        this.next = new Frame(capacity);
        this.prevPos = new Float32Array(capacity * 3);
        this.renderPos = new Float32Array(capacity * 3);
        this.corpseData = new Float32Array(CORPSE_CAPACITY * 7);
    }

    get count(): number {
        return this.curr.count;
    }

    get hordeOf(): Uint32Array {
        return this.curr.hordeOf;
    }

    /** Effect bits per soldier, aligned with renderPos (STATUS_*). */
    get status(): Uint16Array {
        return this.curr.status;
    }

    pushSnapshot(s: GameSnapshot, now: number): void {
        this.hordes.clear();
        for (const h of s.hordes) this.hordes.set(h.id, h);

        const old = this.curr;
        const f = this.next;
        const n = Math.min(s.soldierCount, this.capacity);
        const nowSec = now / 1000;

        // Both lists are sorted by id: merge to match soldiers across snapshots.
        let p = 0;
        for (let i = 0, w = 0; i < n; i++, w += SOLDIER_WORDS) {
            const id = s.soldierU32[w];
            const o = i * 3;
            f.ids[i] = id;
            f.hordeOf[i] = s.soldierU32[w + 1];
            f.pos[o] = s.soldierF32[w + 2];
            f.pos[o + 1] = s.soldierF32[w + 3];
            f.pos[o + 2] = s.soldierF32[w + 4];
            f.state[i] = (s.soldierU32[w + 5] >> 8) & 0xff;
            f.status[i] = s.soldierU32[w + 5] >>> 16;

            const h = this.hordes.get(f.hordeOf[i]);
            const type = h ? this.types[h.kind] : undefined;
            f.kind[i] = h?.kind ?? 0;
            f.owner[i] = h?.owner ?? 0;
            f.hero[i] = type?.hero ? 1 : 0;
            f.scale[i] = ((type?.radius ?? BASE_RADIUS) / BASE_RADIUS) * (h?.scale ?? 1); // transformed: bigger / smaller

            // Old soldiers with smaller ids are gone: they died.
            while (p < old.count && old.ids[p] < id) {
                this.addCorpse(old, p, nowSec);
                p++;
            }
            const matched = p < old.count && old.ids[p] === id;

            // swingStart = when the current attack / knockdown state began.
            const animated = f.state[i] === SoldierAttacking || f.state[i] === SoldierKnocked;
            f.swingStart[i] = matched && animated && old.state[p] === f.state[i] ? old.swingStart[p] : now;

            this.chooseClip(f, i, matched ? old : null, p, now);

            if (matched) {
                // Interpolate from where it was drawn in the previous snapshot.
                this.prevPos[o] = old.pos[p * 3];
                this.prevPos[o + 1] = old.pos[p * 3 + 1];
                this.prevPos[o + 2] = old.pos[p * 3 + 2];
                p++;
            } else {
                // New soldier: no previous state.
                this.prevPos[o] = f.pos[o];
                this.prevPos[o + 1] = f.pos[o + 1];
                this.prevPos[o + 2] = f.pos[o + 2];
            }
        }
        while (p < old.count) {
            this.addCorpse(old, p, nowSec);
            p++;
        }
        f.count = n;

        this.curr = f;
        this.next = old;
        this.currTime = now;
    }

    /** VAT units: picks the animation clip from the simulation state, letting an
     *  attack clip play to the end even after the short attack state is over. */
    private chooseClip(f: Frame, i: number, old: Frame | null, p: number, now: number): void {
        const k = this.kinds[f.kind[i]];
        if (!k?.vat) return;
        const prevClip = old ? old.clip[p] : -1;
        const prevStart = old ? old.clipStart[p] : 0;
        const attack = k.clips[CLIP_ATTACK];

        let clip: number;
        if (prevClip === CLIP_ATTACK && attack && now - prevStart < attack.duration * 1000) {
            clip = CLIP_ATTACK; // finish the swing
        } else if (f.state[i] === SoldierAttacking && attack) {
            clip = CLIP_ATTACK;
            f.clip[i] = clip;
            f.clipStart[i] = now;
            return;
        } else if (f.state[i] === SoldierMoving && k.clips[CLIP_WALK]) {
            clip = k.clips[CLIP_RUN] && k.rates[CLIP_WALK] > RUN_THRESHOLD ? CLIP_RUN : CLIP_WALK;
        } else {
            clip = CLIP_IDLE;
        }

        f.clip[i] = clip;
        if (clip === prevClip) {
            f.clipStart[i] = prevStart;
        } else {
            // Desynchronise loops so a horde doesn't move in lockstep.
            const phase = clip === CLIP_ATTACK ? 0 : ((f.ids[i] * 0.618) % 1) * 1000;
            f.clipStart[i] = now - phase;
        }
    }

    /** Interpolates transforms and writes them straight into the instance buffers. */
    update(
        now: number,
        heightAt: (x: number, z: number) => number,
        selected: ReadonlySet<number>,
        camera: THREE.Camera,
    ): void {
        const alpha = Math.min(Math.max((now - this.currTime) / this.tickMs, 0), 1);
        const m = this.mesh.instanceMatrix.array as Float32Array;
        const pcolors = this.mesh.instanceColor!.array as Float32Array;
        const rm = this.rings.instanceMatrix.array as Float32Array;
        const sm = this.shadows.instanceMatrix.array as Float32Array;
        let shadows = 0;
        const dt = this.lastFrame ? Math.min((now - this.lastFrame) / 1000, 0.1) : 0;
        this.lastFrame = now;
        const f = this.curr;
        const prev = this.prevPos;
        const curr = f.pos;
        const renderPos = this.renderPos;
        const nowSec = now / 1000;
        let rings = 0;
        let placeholders = 0;

        for (const r of this.renderers) r.begin(camera);
        this.machines?.begin();

        for (let i = 0, o = 0; i < f.count; i++, o += 3) {
            let x = prev[o] + (curr[o] - prev[o]) * alpha;
            let z = prev[o + 1] + (curr[o + 1] - prev[o + 1]) * alpha;
            let dr = curr[o + 2] - prev[o + 2];
            if (dr > Math.PI) dr -= 2 * Math.PI;
            else if (dr < -Math.PI) dr += 2 * Math.PI;
            const r = prev[o + 2] + dr * alpha;
            const s = f.scale[i];
            const kind = this.kinds[f.kind[i]];
            const vat = kind?.vat ?? null;

            renderPos[o] = x;
            renderPos[o + 2] = z;

            if (this.machines?.has(f.kind[i])) {
                const y = heightAt(x, z);
                renderPos[o + 1] = y;
                this.machines.place(f.ids[i], f.kind[i], x, y, z, r, s / ((this.types[f.kind[i]]?.radius ?? BASE_RADIUS) / BASE_RADIUS), f.state[i] === SoldierAttacking ? f.swingStart[i] : -1, now);
                if (selected.has(f.hordeOf[i])) rings = this.writeRing(rm, rings, x, y, z, s);
                continue;
            }

            if (!vat && f.state[i] === SoldierAttacking) {
                // Placeholder has no attack animation: lunge forward instead.
                const t = Math.min((now - f.swingStart[i]) / SWING_MS, 1);
                const lunge = Math.sin(Math.PI * t) * LUNGE * s;
                x += Math.sin(r) * lunge; // forward = (sin r, cos r)
                z += Math.cos(r) * lunge;
            }
            let ground = heightAt(x, z);
            if (f.status[i] & STATUS_ON_WALL) ground += this.standHeight ? this.standHeight(x, z) : this.wallHeight; // the walkway is their ground
            let y = ground;
            if (this.types[f.kind[i]]?.flying) {
                y += this.flightHeight(f.hordeOf[i], (f.status[i] & STATUS_GROUNDED) !== 0, now, dt) + Math.sin(now * 0.002 + f.ids[i]) * FLY_BOB;
                shadows = this.writeRing(sm, shadows, x, ground, z, s * (1.4 - Math.min(y - ground, FLY_HEIGHT) / FLY_HEIGHT * 0.5));
            }
            renderPos[o + 1] = y;

            // Destination slot: VAT instance or placeholder instance.
            let target: Float32Array;
            let k: number;
            if (vat) {
                const clipId = f.state[i] === SoldierKnocked ? CLIP_IDLE : f.clip[i];
                const clip = kind.clips[clipId] ?? kind.clips[CLIP_IDLE]!;
                const slot = vat.slot(x, y, z, s, clip, f.clipStart[i] / 1000, kind.rates[clipId] ?? 1, this.teamColors[f.owner[i] % 4]);
                if (slot < 0) {
                    if (selected.has(f.hordeOf[i])) rings = this.writeRing(rm, rings, renderPos[o], ground, renderPos[o + 2], s);
                    continue; // culled: off screen
                }
                target = vat.matrices;
                k = slot;
            } else {
                target = m;
                k = placeholders * 16;
                this.color.copy(this.teamColors[f.owner[i] % 4]);
                if (f.hero[i]) this.color.lerp(HERO_TINT, 0.45); // heroes stand out
                this.color.toArray(pcolors, placeholders * 3);
                placeholders++;
            }

            if (f.state[i] === SoldierKnocked) {
                this.writeKnocked(target, k, x, y, z, r, s, (now - f.swingStart[i]) / KNOCK_MS);
            } else {
                writeYRotation(target, k, x, y, z, r, s);
            }

            if (selected.has(f.hordeOf[i])) rings = this.writeRing(rm, rings, renderPos[o], ground, renderPos[o + 2], s);
        }

        this.mesh.count = placeholders;
        this.mesh.instanceMatrix.needsUpdate = true;
        this.mesh.instanceColor!.needsUpdate = true;
        this.rings.count = rings;
        this.rings.instanceMatrix.needsUpdate = true;
        this.shadows.count = shadows;
        this.shadows.instanceMatrix.needsUpdate = true;

        this.machines?.end();
        this.updateCorpses(nowSec, heightAt);
        for (const r of this.renderers) r.end(nowSec);
    }

    /** Eased flight height of a flying horde (drops low while grounded). */
    private flightHeight(horde: number, grounded: boolean, now: number, dt: number): number {
        const target = grounded ? GROUNDED_HEIGHT : FLY_HEIGHT;
        let l = this.lift.get(horde);
        if (!l) {
            l = { y: target, at: now };
            this.lift.set(horde, l);
            if (this.lift.size > 1024) {
                for (const [id, v] of this.lift) if (now - v.at > 2000) this.lift.delete(id); // gone hordes
            }
        }
        if (l.at !== now) {
            l.y += (target - l.y) * Math.min(1, dt * 3);
            l.at = now;
        }
        return l.y;
    }

    private writeRing(rm: Float32Array, rings: number, x: number, y: number, z: number, s: number): number {
        const q = rings * 16;
        rm.fill(0, q, q + 16);
        rm[q] = s;
        rm[q + 5] = 1;
        rm[q + 10] = s;
        rm[q + 12] = x;
        rm[q + 13] = y + 0.08;
        rm[q + 14] = z;
        rm[q + 15] = 1;
        return rings + 1;
    }

    /** Draw-call statistics of the VAT renderers (for the debug panel). */
    lodStats(): { visible: number; culled: number; perLod: [number, number, number] } {
        const out = { visible: 0, culled: 0, perLod: [0, 0, 0] as [number, number, number] };
        for (const r of this.renderers) {
            out.culled += r.stats.culled;
            for (let l = 0; l < 3; l++) {
                out.perLod[l] += r.stats.perLod[l];
                out.visible += r.stats.perLod[l];
            }
        }
        return out;
    }

    /** Knocked down: falls back fast, lies, then gets up (t = 0..1 of the knockdown). */
    private writeKnocked(m: Float32Array, k: number, x: number, y: number, z: number, r: number, s: number, t: number): void {
        let tilt: number;
        if (t < 0.15) tilt = t / 0.15;
        else if (t < 0.75) tilt = 1;
        else tilt = Math.max(0, 1 - (t - 0.75) / 0.25);
        const angle = tilt * THREE.MathUtils.degToRad(80);

        this.tmpQuat.setFromAxisAngle(this.up, r);
        this.tmpLie.setFromAxisAngle(this.axisX, -angle); // fall backwards
        this.tmpQuat.multiply(this.tmpLie);
        this.tmpPos.set(x, y, z);
        this.tmpScale.setScalar(s);
        this.tmpMatrix.compose(this.tmpPos, this.tmpQuat, this.tmpScale).toArray(m, k);
    }

    typeOf(hordeId: number): HordeTypeInfo | undefined {
        const h = this.hordes.get(hordeId);
        return h ? this.types[h.kind] : undefined;
    }

    dispose(): void {
        for (const r of this.renderers) r.dispose();
    }

    private addCorpse(frame: Frame, i: number, nowSec: number): void {
        if (this.machines?.has(frame.kind[i])) return; // machines just vanish (wreck models later)
        const d = this.corpseData;
        const k = this.corpseHead * 7;
        d[k] = frame.pos[i * 3];
        d[k + 1] = frame.pos[i * 3 + 1];
        d[k + 2] = frame.pos[i * 3 + 2];
        d[k + 3] = frame.scale[i];
        d[k + 4] = frame.owner[i];
        d[k + 5] = nowSec;
        d[k + 6] = frame.kind[i];
        this.corpseHead = (this.corpseHead + 1) % CORPSE_CAPACITY;
        this.corpseCount = Math.min(this.corpseCount + 1, CORPSE_CAPACITY);
    }

    private updateCorpses(nowSec: number, heightAt: (x: number, z: number) => number): void {
        const d = this.corpseData;
        const colors = this.corpses.instanceColor!.array as Float32Array;
        let n = 0;

        // Oldest first.
        const start = (this.corpseHead - this.corpseCount + CORPSE_CAPACITY) % CORPSE_CAPACITY;
        let alive = 0;
        for (let j = 0; j < this.corpseCount; j++) {
            const k = ((start + j) % CORPSE_CAPACITY) * 7;
            const age = nowSec - d[k + 5];
            if (age > CORPSE_LIFETIME) continue;
            alive++;

            const x = d[k];
            const z = d[k + 1];
            const s = d[k + 3];
            const owner = d[k + 4];
            const sink = Math.max(0, age - (CORPSE_LIFETIME - CORPSE_SINK)) / CORPSE_SINK;
            const kind = this.kinds[d[k + 6]];

            if (kind?.vat && kind.die) {
                // Baked death animation, holding its last frame; sinks at the end.
                const y = heightAt(x, z) - sink * 0.6 * s;
                const slot = kind.vat.slot(x, y, z, s, kind.die, d[k + 5], 1, this.teamColors[owner % 4]);
                if (slot >= 0) writeYRotation(kind.vat.matrices, slot, x, y, z, d[k + 2], s);
                continue;
            }

            // Placeholder: tip over during the first 0.4 s.
            const fall = Math.min(age / 0.4, 1);
            this.tmpQuat.setFromAxisAngle(this.up, d[k + 2]);
            this.tmpLie.identity().slerp(this.qLie, fall);
            this.tmpQuat.multiply(this.tmpLie);
            this.tmpPos.set(x, heightAt(x, z) + 0.15 * s * fall - sink * 0.6 * s, z);
            this.tmpScale.setScalar(s);
            this.tmpMatrix.compose(this.tmpPos, this.tmpQuat, this.tmpScale);
            this.tmpMatrix.toArray(this.corpses.instanceMatrix.array as Float32Array, n * 16);

            this.color.copy(this.teamColors[owner % 4]).multiplyScalar(0.45);
            this.color.toArray(colors, n * 3);
            n++;
        }
        this.corpseCount = alive === 0 ? 0 : this.corpseCount;
        this.corpses.count = n;
        this.corpses.instanceMatrix.needsUpdate = true;
        this.corpses.instanceColor!.needsUpdate = true;
    }
}

/** Column-major matrix: rotation r around Y, uniform scale s, translation. */
function writeYRotation(m: Float32Array, k: number, x: number, y: number, z: number, r: number, s: number): void {
    const c = Math.cos(r) * s;
    const sn = Math.sin(r) * s;
    m[k] = c;
    m[k + 1] = 0;
    m[k + 2] = -sn;
    m[k + 3] = 0;
    m[k + 4] = 0;
    m[k + 5] = s;
    m[k + 6] = 0;
    m[k + 7] = 0;
    m[k + 8] = sn;
    m[k + 9] = 0;
    m[k + 10] = c;
    m[k + 11] = 0;
    m[k + 12] = x;
    m[k + 13] = y;
    m[k + 14] = z;
    m[k + 15] = 1;
}
