import * as THREE from "three";
import { PROJECTILE_FROM_AIR, PROJECTILE_FROM_WALL, PROJECTILE_SPLASH, PROJECTILE_TO_AIR, type GameSnapshot, type ProjectileSnap } from "../../net/protocol";
import { FLY_HEIGHT } from "./flight";

const CAPACITY = 512;
const LAUNCH_HEIGHT = 1.5; // above the ground at the firing horde
const IMPACT_HEIGHT = 0.3;
const WALL_LAUNCH_HEIGHT = 8; // archers on a wall walkway / tower tops
const ARC = 0.3; // apex height as a fraction of the distance

/**
 * Volleys in flight. Each simulated projectile stands for a whole volley; for
 * now it is drawn as a single point on a ballistic arc. Later this view can
 * spawn a cloud of arrows per volley without changing the simulation.
 */
export class ProjectilesView {
    readonly points: THREE.Points;
    /** Area shots (trebuchet stones): bigger, lighter points. */
    readonly stones: THREE.Points;
    /** Called when an area shot lands (dust ring). */
    onImpact: ((x: number, z: number) => void) | null = null;
    private readonly stonePos = new Float32Array(64 * 3);

    private readonly flying = new Map<number, ProjectileSnap>();
    private readonly positions: Float32Array;
    private snapTick = 0;
    private snapTime = 0;

    constructor(private readonly tickMs: number) {
        this.positions = new Float32Array(CAPACITY * 3);
        const geo = new THREE.BufferGeometry();
        geo.setAttribute("position", new THREE.BufferAttribute(this.positions, 3).setUsage(THREE.DynamicDrawUsage));
        geo.setDrawRange(0, 0);
        const mat = new THREE.PointsMaterial({ color: 0x1a1208, size: 6, sizeAttenuation: false });
        this.points = new THREE.Points(geo, mat);
        this.points.frustumCulled = false;
        const sg = new THREE.BufferGeometry();
        sg.setAttribute("position", new THREE.BufferAttribute(this.stonePos, 3).setUsage(THREE.DynamicDrawUsage));
        sg.setDrawRange(0, 0);
        this.stones = new THREE.Points(sg, new THREE.PointsMaterial({ color: 0x6b6458, size: 14, sizeAttenuation: false }));
        this.stones.frustumCulled = false;
    }

    pushSnapshot(s: GameSnapshot, now: number): void {
        this.snapTick = s.tick;
        this.snapTime = now;
        for (const p of s.projectiles) {
            if (!this.flying.has(p.id)) this.flying.set(p.id, p);
        }
    }

    update(now: number, heightAt: (x: number, z: number) => number): void {
        // Estimated server time in ticks (fractional), from the latest snapshot.
        const tick = this.snapTick + (now - this.snapTime) / this.tickMs;
        const pos = this.positions;
        let n = 0;
        let ns = 0;

        for (const [id, p] of this.flying) {
            const t = (tick - p.launchTick) / (p.impactTick - p.launchTick);
            const splash = (p.flags & PROJECTILE_SPLASH) !== 0;
            if (t >= 1) {
                this.flying.delete(id);
                if (splash) this.onImpact?.(p.toX, p.toZ);
                continue;
            }
            if (splash ? ns >= 64 : n >= CAPACITY) continue;
            const k = Math.max(t, 0);
            const x = p.fromX + (p.toX - p.fromX) * k;
            const z = p.fromZ + (p.toZ - p.fromZ) * k;
            const fromAir = (p.flags & PROJECTILE_FROM_AIR) !== 0;
            const toAir = (p.flags & PROJECTILE_TO_AIR) !== 0;
            const fromWall = (p.flags & PROJECTILE_FROM_WALL) !== 0;
            const y0 = heightAt(p.fromX, p.fromZ) + (fromAir ? FLY_HEIGHT : fromWall ? WALL_LAUNCH_HEIGHT : LAUNCH_HEIGHT);
            const y1 = heightAt(p.toX, p.toZ) + (toAir ? FLY_HEIGHT : IMPACT_HEIGHT);
            // Shots up into the air fly flatter.
            const apex = Math.hypot(p.toX - p.fromX, p.toZ - p.fromZ) * (toAir ? ARC * 0.3 : ARC);
            const yy = y0 + (y1 - y0) * k + 4 * apex * k * (1 - k);
            if (splash) {
                this.stonePos.set([x, yy, z], ns * 3);
                ns++;
                continue;
            }
            pos[n * 3] = x;
            pos[n * 3 + 1] = yy;
            pos[n * 3 + 2] = z;
            n++;
        }

        this.stones.geometry.setDrawRange(0, ns);
        (this.stones.geometry.getAttribute("position") as THREE.BufferAttribute).needsUpdate = true;
        const geo = this.points.geometry;
        geo.setDrawRange(0, n);
        (geo.getAttribute("position") as THREE.BufferAttribute).needsUpdate = true;
    }
}
