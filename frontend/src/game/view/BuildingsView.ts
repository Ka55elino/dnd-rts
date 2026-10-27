import * as THREE from "three";
import { WallsView } from "./WallsView";
import { NEUTRAL, buildProgress, isCapturing, isConstructing, isGateOpen, plotOccupied, type BuildingSnap, type GameSnapshot } from "../../net/protocol";
import { StaticModel } from "../../engine/models/StaticModel";
import { PLAYER_COLORS } from "./SoldiersView";

/** Client-side building type (mirror of ContentService.BuildingType). */
export interface BuildingTypeInfo {
    id: string;
    name: string;
    kind: string; // "fortress" | "outpost" | "building"
    model: string;
    modelScale: number;
    width: number;
    depth: number;
    placement: string[];
    buildTime: number;
    noBuildRadius: number; // free buildings may not reach into this circle
    plots: { x: number; y: number; size: number; class: string }[];
    /** Fortress base: wall contour, gates, tower sockets (local x = right, y = front). */
    base: BaseLayoutInfo | null;
    /** Walls and gates: height of the walkway on top. */
    walkHeight: number;
    /** Towers: toggle abilities (fire arrows). */
    toggles: string[];
    /** Watchtowers: foot soldiers stand on the platform (at walkHeight). */
    garrison: boolean;
    cost: Partial<Record<string, number>>;
    trains: string[];
    research: string[];
    requires: import("../hud/tech").Requires;
}

export interface BaseLayoutInfo {
    outline: { x: number; y: number }[];
    gates: { x: number; y: number }[];
    towers: { x: number; y: number }[];
}

/** A free plot under the cursor. */
export interface PlotPick {
    host: number; // fortress / outpost building id
    plot: number;
    size: number;
    class: string; // s / m / l, "" = custom
}


const NEUTRAL_COLOR = 0xb8b8b8;
const COLLAPSE_SECONDS = 2.5; // destroyed buildings sink into the ground
const LABEL_LIFT = 2; // world units above the model top

interface Placed {
    snap: BuildingSnap;
    root: THREE.Group; // positioned on the terrain, rotated
    plots: THREE.Mesh[];
    label: HTMLDivElement;
    text: HTMLSpanElement;
    fill: HTMLDivElement;
    top: number; // label height above the base
    owner: number;
    lastHp: number;
    lastFlags: number;
    lastPlotsHi: number;
    model: THREE.Object3D; // scaled vertically while under construction
    isWall: boolean; // wall segment (drawn by WallsView)
    quiet: boolean; // walls and gates: the plate shows only once damaged
    gate: GateParts | null;
}

/** The moving parts of a gatehouse (in every LOD level) and how far it is open. */
interface GateParts {
    left: THREE.Object3D[];
    right: THREE.Object3D[];
    portcullis: THREE.Object3D[];
    portY: number[]; // closed heights of the portcullis nodes
    open: number; // 0 closed .. 1 open (animated)
    target: number;
}

const GATE_SECONDS = 1.2; // portcullis up, then the leaves swing in (closing: the reverse)
const PORTCULLIS_RISE = 4.6;

/**
 * Buildings from game snapshots: static models with LOD, build plots of
 * fortresses / outposts, and an HTML plate (name + health) above each one.
 * Buildings rarely move, so objects are created once and only updated when
 * the snapshot changes them.
 */
export class BuildingsView {
    readonly group = new THREE.Group();
    private readonly placed = new Map<number, Placed>();
    private readonly plotGeo = plotGeometry();
    private readonly plotMats = new Map<number, THREE.MeshBasicMaterial>();
    readonly walls: WallsView;
    /** Walkway height of the base walls (tower sockets sit on top). */
    private readonly wallHeight: number;
    private readonly highlightMat = new THREE.MeshBasicMaterial({
        color: 0xffffff,
        transparent: true,
        opacity: 0.95,
        depthWrite: false,
        side: THREE.DoubleSide,
        polygonOffset: true,
        polygonOffsetFactor: -2,
    });
    private selectedPlot: PlotPick | null = null;
    private selectedBuilding = 0;
    private readonly tmp = new THREE.Vector3();
    /** Destroyed buildings playing their collapse. */
    private readonly falling: { root: THREE.Group; age: number; baseY: number; depth: number; tilt: THREE.Vector3 }[] = [];

    private constructor(
        private readonly types: BuildingTypeInfo[],
        private readonly models: Map<string, StaticModel>,
        private readonly overlay: HTMLElement,
        private readonly heightAt: (x: number, z: number) => number,
    ) {
        const wall = types.find((t) => t.kind === "wall");
        this.wallHeight = wall?.walkHeight ?? 0;
        this.walls = new WallsView((wall && models.get(wall.model)) ?? null);
        this.group.add(this.walls.group);
    }

    private isWall(type: BuildingTypeInfo | undefined): boolean {
        return type?.kind === "wall"; // gates are gatehouse models (animated), not instanced wall rows
    }

    /** Finds the leaves and the portcullis in every LOD level of a gatehouse instance. */
    private gateParts(lod: THREE.LOD, open: boolean): GateParts {
        const parts: GateParts = { left: [], right: [], portcullis: [], portY: [], open: open ? 1 : 0, target: open ? 1 : 0 };
        lod.traverse((o) => {
            if (o.name.endsWith("gate_left")) parts.left.push(o);
            else if (o.name.endsWith("gate_right")) parts.right.push(o);
            else if (o.name.endsWith("portcullis")) {
                parts.portcullis.push(o);
                parts.portY.push(o.position.y);
            }
        });
        this.poseGate(parts);
        return parts;
    }

    private poseGate(g: GateParts): void {
        const port = Math.min(1, g.open * 2); // first half: the portcullis
        const leaves = Math.max(0, g.open * 2 - 1); // second half: the leaves (they open inwards)
        const ease = (t: number) => t * t * (3 - 2 * t);
        g.portcullis.forEach((o, i) => {
            o.position.y = g.portY[i] + ease(port) * PORTCULLIS_RISE;
            o.updateMatrix();
        });
        for (const o of g.left) {
            o.rotation.y = (ease(leaves) * Math.PI) / 2;
            o.updateMatrix();
        }
        for (const o of g.right) {
            o.rotation.y = (-ease(leaves) * Math.PI) / 2;
            o.updateMatrix();
        }
    }

    static async create(
        types: BuildingTypeInfo[],
        overlay: HTMLElement,
        heightAt: (x: number, z: number) => number,
    ): Promise<BuildingsView> {
        const models = new Map<string, StaticModel>();
        await Promise.all(
            [...new Set(types.map((t) => t.model))].map(async (id) => {
                try {
                    models.set(id, await StaticModel.load(id));
                } catch (err) {
                    console.warn(`building model ${id}:`, err); // drawn as a placeholder box
                }
            }),
        );
        return new BuildingsView(types, models, overlay, heightAt);
    }

    pushSnapshot(snap: GameSnapshot): void {
        const seen = new Set<number>();
        for (const b of snap.buildings) {
            seen.add(b.id);
            const p = this.placed.get(b.id) ?? this.place(b);
            p.snap = b;
            if (b.owner !== p.owner) this.setOwner(p, b.owner);
            if (b.flags !== p.lastFlags || b.plotsHi !== p.lastPlotsHi) this.applyFlags(p);
        }
        for (const [id, p] of this.placed) if (!seen.has(id)) this.remove(id, p);
    }

    /**
     * Building under a ray (models first, then footprints on the ground for
     * clicks that miss thin geometry). Destroyed buildings are ignored.
     */
    pick(ray: THREE.Raycaster, ground: THREE.Vector3 | null): { id: number; owner: number; point: THREE.Vector3 } | null {
        const wall = this.walls.pick(ray);
        const wp = wall && this.placed.get(wall.id);
        if (wall && wp) return { id: wall.id, owner: wp.owner, point: wall.point };
        const roots = [...this.placed.values()].filter((p) => !p.isWall).map((p) => p.root);
        const hit = ray.intersectObjects(roots, true)[0];
        if (hit) {
            let o: THREE.Object3D | null = hit.object;
            while (o && o.userData.buildingId === undefined) o = o.parent;
            const p = o ? this.placed.get(o.userData.buildingId as number) : undefined;
            if (p) return { id: p.snap.id, owner: p.owner, point: hit.point };
        }
        if (ground) {
            for (const p of this.placed.values()) {
                const type = this.types[p.snap.kind];
                if (!type || p.isWall) continue;
                // Ground point in building-local axes (x = right, z = front).
                const dx = ground.x - p.snap.x;
                const dz = ground.z - p.snap.z;
                const c = Math.cos(p.snap.rot);
                const s = Math.sin(p.snap.rot);
                const lx = dx * c - dz * s;
                const lz = dx * s + dz * c;
                if (Math.abs(lx) <= type.width / 2 + 1 && Math.abs(lz) <= type.depth / 2 + 1) {
                    return { id: p.snap.id, owner: p.owner, point: ground.clone() };
                }
            }
        }
        return null;
    }

    /** Free plot of a building owned by `owner` under the ground point, or null. */
    pickPlot(ground: THREE.Vector3 | null, owner: number, ray?: THREE.Raycaster): PlotPick | null {
        if (!ground) return null;
        // Tower sockets sit on the walkway: test them where the ray crosses that height.
        let onWall: THREE.Vector3 | null = null;
        if (ray && this.wallHeight > 0 && Math.abs(ray.ray.direction.y) > 1e-4) {
            const t = (ground.y + this.wallHeight - ray.ray.origin.y) / ray.ray.direction.y;
            if (t > 0) onWall = ray.ray.at(t, new THREE.Vector3());
        }
        for (const p of this.placed.values()) {
            if (p.owner !== owner) continue;
            const plots = this.types[p.snap.kind]?.plots ?? [];
            const c = Math.cos(p.snap.rot);
            const s = Math.sin(p.snap.rot);
            const local = (q: THREE.Vector3) => {
                const dx = q.x - p.snap.x;
                const dz = q.z - p.snap.z;
                return [dx * c - dz * s, dx * s + dz * c];
            };
            const [gx, gz] = local(ground);
            const [wx, wz] = onWall ? local(onWall) : [gx, gz];
            for (let i = 0; i < plots.length; i++) {
                const plot = plots[i];
                if (plotOccupied(p.snap, i)) continue;
                const [lx, lz] = plot.class === "tower" ? [wx, wz] : [gx, gz];
                if (Math.abs(lx - plot.x) <= plot.size / 2 && Math.abs(lz - plot.y) <= plot.size / 2) {
                    return { host: p.snap.id, plot: i, size: plot.size, class: plot.class };
                }
            }
        }
        return null;
    }

    /**
     * Height of the platform a soldier on a "wall" stands on: a watchtower's
     * if (x, z) is on one, else the base walls' walkway.
     */
    standHeight(x: number, z: number): number {
        for (const p of this.placed.values()) {
            const t = this.types[p.snap.kind];
            if (!t?.garrison) continue;
            const r = Math.max(t.width, t.depth) / 2 + 1.5;
            if (Math.abs(x - p.snap.x) <= r && Math.abs(z - p.snap.z) <= r) return t.walkHeight;
        }
        return this.wallHeight;
    }

    /** Every standing building with its type (for placement checks). */
    *all(): IterableIterator<{ snap: BuildingSnap; type: BuildingTypeInfo | undefined }> {
        for (const p of this.placed.values()) yield { snap: p.snap, type: this.types[p.snap.kind] };
    }

    /**
     * A see-through copy of a building type's model for placement previews.
     * Materials are owned by the ghost: call the returned dispose().
     */
    ghost(kind: number, material: THREE.Material): THREE.Object3D {
        const type = this.types[kind];
        const model = type && this.models.get(type.model);
        if (!model || !type) {
            const w = type?.width ?? 4;
            const d = type?.depth ?? 4;
            return new THREE.Mesh(new THREE.BoxGeometry(w, 6, d).translate(0, 3, 0), material);
        }
        const obj = model.instance(type.modelScale);
        obj.traverse((o) => {
            if (o instanceof THREE.Mesh) o.material = material;
        });
        return obj;
    }

    /** World position of plot i of a fortress / outpost. */
    plotWorld(host: number, i: number): { x: number; z: number } | null {
        const p = this.placed.get(host);
        const plot = p && this.types[p.snap.kind]?.plots[i];
        if (!p || !plot) return null;
        // Local (x = right, z = front) rotated by the snapshot's Y rotation.
        const c = Math.cos(p.snap.rot);
        const s = Math.sin(p.snap.rot);
        return { x: p.snap.x + plot.x * c + plot.y * s, z: p.snap.z - plot.x * s + plot.y * c };
    }

    /** Current snapshot and type of a building (undefined once it is gone). */
    info(id: number): { snap: BuildingSnap; type: BuildingTypeInfo | undefined } | undefined {
        const p = this.placed.get(id);
        return p && { snap: p.snap, type: this.types[p.snap.kind] };
    }

    /** Highlights a plot or a building (0 / null = none). */
    select(plot: PlotPick | null, building = 0): void {
        const old = this.selectedPlot && this.placed.get(this.selectedPlot.host);
        if (old) old.plots[this.selectedPlot!.plot].material = this.plotMaterial(old.owner);
        this.placed.get(this.selectedBuilding)?.label.classList.remove("is-selected");
        this.selectedPlot = plot;
        this.selectedBuilding = building;
        const host = plot && this.placed.get(plot.host);
        if (host) host.plots[plot!.plot].material = this.highlightMat;
        this.placed.get(building)?.label.classList.add("is-selected");
    }

    /** Animates collapses; call once per frame. */
    animate(dt: number): void {
        for (const p of this.placed.values()) {
            const g = p.gate;
            if (!g || g.open === g.target) continue;
            const step = dt / GATE_SECONDS;
            g.open = g.target > g.open ? Math.min(g.target, g.open + step) : Math.max(g.target, g.open - step);
            this.poseGate(g);
        }
        for (let i = this.falling.length - 1; i >= 0; i--) {
            const f = this.falling[i];
            f.age += dt;
            const t = Math.min(f.age / COLLAPSE_SECONDS, 1);
            const ease = t * t;
            f.root.position.y = f.baseY - f.depth * ease;
            f.root.rotation.x = f.tilt.x * ease;
            f.root.rotation.z = f.tilt.z * ease;
            // Shake while it goes down.
            f.root.children[0].position.x = Math.sin(f.age * 40) * 0.15 * (1 - t);
            if (t >= 1) {
                this.group.remove(f.root);
                this.falling.splice(i, 1);
            }
        }
    }

    /** Positions the HTML plates; call after the camera moved. */
    update(camera: THREE.Camera, width: number, height: number): void {
        this.walls.update(camera);
        for (const p of this.placed.values()) {
            const b = p.snap;
            if (p.quiet && b.hp >= b.maxHp && this.selectedBuilding !== b.id) {
                p.label.hidden = true; // intact wall segments: no plate
                continue;
            }
            this.tmp.set(b.x, p.root.position.y + p.top, b.z).project(camera);
            const visible = this.tmp.z < 1 && Math.abs(this.tmp.x) < 1.1 && Math.abs(this.tmp.y) < 1.1;
            p.label.hidden = !visible;
            if (!visible) continue;
            const sx = (this.tmp.x + 1) * 0.5 * width;
            const sy = (1 - this.tmp.y) * 0.5 * height;
            p.label.style.transform = `translate(${sx.toFixed(1)}px, ${sy.toFixed(1)}px) translate(-50%, -100%)`;
            if (b.hp !== p.lastHp && !isCapturing(b.flags)) {
                p.lastHp = b.hp;
                p.fill.style.width = `${b.maxHp > 0 ? (100 * b.hp) / b.maxHp : 0}%`;
            }
        }
    }

    dispose(): void {
        for (const [id, p] of this.placed) this.remove(id, p, false);
        for (const f of this.falling) this.group.remove(f.root);
        this.falling.length = 0;
        for (const m of this.models.values()) m.dispose();
        this.models.clear();
        this.plotGeo.dispose();
        for (const m of this.plotMats.values()) m.dispose();
        this.highlightMat.dispose();
        this.walls.dispose();
    }

    private place(b: BuildingSnap): Placed {
        const type = this.types[b.kind];
        const root = new THREE.Group();
        // Sit on the lowest footprint corner so the base never floats on slopes.
        const c = Math.cos(b.rot);
        const s = Math.sin(b.rot);
        let ground = Infinity;
        for (const [dx, dz] of [[-1, -1], [1, -1], [-1, 1], [1, 1], [0, 0]]) {
            const lx = (dx * (type?.width ?? 4)) / 2;
            const lz = (dz * (type?.depth ?? 4)) / 2;
            // Model space: x = right, z = front; rotate by the Y rotation.
            ground = Math.min(ground, this.heightAt(b.x + lx * c + lz * s, b.z - lx * s + lz * c));
        }
        root.position.set(b.x, ground, b.z);
        root.rotation.y = b.rot;
        root.userData.buildingId = b.id;

        const model = type && !this.isWall(type) ? this.models.get(type.model) : undefined;
        let top = 6;
        let body: THREE.Object3D;
        if (this.isWall(type)) {
            body = new THREE.Group();
            top = type!.walkHeight + 1;
        } else if (model && type) {
            body = model.instance(type.modelScale);
            top = model.bounds.max.y * type.modelScale;
            if (type.kind === "gate") {
                // Stretch the gatehouse over its footprint (a wider passage than the model's).
                const size = model.bounds.getSize(new THREE.Vector3());
                body.scale.set(type.width / size.x, type.modelScale, type.depth / size.z);
            }
        } else {
            const w = type?.width ?? 4;
            const d = type?.depth ?? 4;
            body = new THREE.Mesh(new THREE.BoxGeometry(w, 6, d).translate(0, 3, 0), new THREE.MeshLambertMaterial({ color: 0x8a7a66 }));
        }
        root.add(body);
        this.group.add(root);

        // Build plots: flat square outlines in the building's local frame. Plot
        // offsets are (x = right, y = front); with the snapshot's Y rotation the
        // model's front is local +Z and its right is local +X.
        const plots = (type?.plots ?? []).map((plot) => {
            const mesh = new THREE.Mesh(this.plotGeo, this.plotMaterial(b.owner));
            mesh.scale.setScalar(plot.size);
            const lx = plot.x;
            const lz = plot.y;
            const wx = b.x + lx * c + lz * s;
            const wz = b.z - lx * s + lz * c;
            const lift = plot.class === "tower" ? this.wallHeight + 0.1 : 0.05; // tower sockets: on the walkway
            mesh.position.set(lx, this.plotHeight(wx, wz, plot.size) - ground + lift, lz);
            root.add(mesh);
            return mesh;
        });

        const label = document.createElement("div");
        label.className = "horde-label building-label";
        label.innerHTML = `
            <span class="horde-label__text"></span>
            <div class="horde-label__bar building-label__bar"><div class="horde-label__fill"></div></div>
        `;
        const text = label.querySelector<HTMLSpanElement>(".horde-label__text")!;
        text.textContent = type?.name ?? `building ${b.kind}`;
        this.overlay.appendChild(label);

        const p: Placed = {
            snap: b,
            root,
            plots,
            label,
            text,
            fill: label.querySelector(".horde-label__fill")!,
            top: top + LABEL_LIFT,
            owner: -1,
            lastHp: -1,
            lastFlags: -1,
            lastPlotsHi: -1,
            model: body,
            isWall: false,
            quiet: false,
            gate: null,
        };
        if (type?.kind === "gate" && body instanceof THREE.LOD) p.gate = this.gateParts(body, isGateOpen(b.flags));
        p.quiet = type?.kind === "wall" || type?.kind === "gate";
        if (this.isWall(type)) {
            // Drawn by WallsView (instanced); the plate shows only once damaged.
            p.isWall = true;
            label.classList.add("is-wall");
            this.walls.set({
                id: b.id, x: b.x, y: ground, z: b.z, rot: b.rot, length: type!.width, thickness: type!.depth,
                walkHeight: type!.walkHeight, gate: type!.kind === "gate",
            });
        }
        this.setOwner(p, b.owner);
        this.applyFlags(p);
        this.placed.set(b.id, p);
        return p;
    }

    /** Construction progress (model rises out of the ground) and plot occupancy. */
    private applyFlags(p: Placed): void {
        const f = p.snap.flags;
        p.lastFlags = f;
        p.lastPlotsHi = p.snap.plotsHi;
        if (p.gate) p.gate.target = isGateOpen(f) ? 1 : 0;
        const type = this.types[p.snap.kind];
        const building = isConstructing(f);
        const progress = buildProgress(f);
        const sy = building ? 0.12 + 0.88 * progress : 1;
        p.model.scale.y = (type?.modelScale ?? 1) * sy;
        const name = type?.name ?? `building ${p.snap.kind}`;
        p.text.textContent = building ? `${name} ${Math.round(progress * 100)}%` : name;
        p.label.classList.toggle("is-constructing", building);
        // Capture: the bar shows the progress in the capturer's color.
        const capturing = isCapturing(f);
        if (capturing) {
            p.text.textContent = `${name} — capturing ${Math.round(progress * 100)}%`;
            p.fill.style.width = `${progress * 100}%`;
            p.fill.style.background = `#${new THREE.Color(ownerColor(p.snap.capturer)).getHexString()}`;
            p.lastHp = -1; // restore the health bar when the capture stops
        } else if (p.label.classList.contains("is-capturing")) {
            p.fill.style.background = `#${new THREE.Color(ownerColor(p.owner)).getHexString()}`;
        }
        p.label.classList.toggle("is-capturing", capturing);
        p.plots.forEach((m, i) => (m.visible = !plotOccupied(p.snap, i)));
    }

    private setOwner(p: Placed, owner: number): void {
        p.owner = owner;
        const color = ownerColor(owner);
        p.fill.style.background = `#${new THREE.Color(color).getHexString()}`;
        p.plots.forEach((m, i) => {
            const selected = this.selectedPlot?.host === p.snap.id && this.selectedPlot.plot === i;
            m.material = selected ? this.highlightMat : this.plotMaterial(owner);
        });
    }

    private plotMaterial(owner: number): THREE.MeshBasicMaterial {
        let m = this.plotMats.get(owner);
        if (!m) {
            m = new THREE.MeshBasicMaterial({
                color: ownerColor(owner),
                transparent: true,
                opacity: 0.7,
                depthWrite: false,
                side: THREE.DoubleSide,
                polygonOffset: true,
                polygonOffsetFactor: -2,
            });
            this.plotMats.set(owner, m);
        }
        return m;
    }

    /** Highest terrain point of a plot (the flat marker then never sinks into it). */
    private plotHeight(x: number, z: number, size: number): number {
        const h = size / 2;
        let top = -Infinity;
        for (const [dx, dz] of [[-h, -h], [h, -h], [-h, h], [h, h], [0, 0]]) top = Math.max(top, this.heightAt(x + dx, z + dz));
        return top;
    }

    /** Destroyed (or gone): drop the plate and plots now, let the model collapse. */
    private remove(id: number, p: Placed, animate = true): void {
        p.label.remove();
        this.placed.delete(id);
        for (const m of p.plots) p.root.remove(m);
        if (p.isWall) {
            this.walls.remove(id);
            this.group.remove(p.root);
            return; // no collapse animation for a segment (it just leaves a gap)
        }
        if (!animate) {
            this.group.remove(p.root);
            return;
        }
        delete p.root.userData.buildingId;
        // Rotation order: Y first (facing), then a small random-ish tilt.
        p.root.rotation.order = "YXZ";
        const k = (id * 0.618) % 1;
        this.falling.push({ root: p.root, age: 0, baseY: p.root.position.y, depth: p.top, tilt: new THREE.Vector3(0.25 * (k - 0.5), 0, 0.25 * (0.5 - k)) });
    }
}

function ownerColor(owner: number): number {
    return owner === NEUTRAL ? NEUTRAL_COLOR : PLAYER_COLORS[owner % PLAYER_COLORS.length];
}

/** Unit square outline on the XZ plane. */
export function plotGeometry(): THREE.BufferGeometry {
    const outer = new THREE.Shape([
        new THREE.Vector2(-0.5, -0.5),
        new THREE.Vector2(0.5, -0.5),
        new THREE.Vector2(0.5, 0.5),
        new THREE.Vector2(-0.5, 0.5),
    ]);
    const inner = new THREE.Path([
        new THREE.Vector2(-0.44, -0.44),
        new THREE.Vector2(-0.44, 0.44),
        new THREE.Vector2(0.44, 0.44),
        new THREE.Vector2(0.44, -0.44),
    ]);
    outer.holes.push(inner);
    return new THREE.ShapeGeometry(outer).rotateX(-Math.PI / 2);
}
