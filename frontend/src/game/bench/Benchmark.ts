import * as THREE from "three";
import { OrbitControls } from "three/examples/jsm/controls/OrbitControls.js";
import { Stream } from "@wailsio/runtime";
import { RenderContext } from "../../engine/renderer";
import { UnitsView } from "../view/UnitsView";
import { decodeSnapshot, encodeBenchSetCount } from "../../net/protocol";
import { VatModel, type Lod, type VatClip } from "../../engine/vat/VatModel";
import { VatLodSet } from "../../engine/vat/VatLodSet";

const STREAM_NAME = "bench"; // internal/bench.StreamName
const TICK_MS = 50; // 20 Hz, internal/bench tickRate
const MAP_SIZE = 512; // internal/bench.MapSize
const CAPACITY = 100_000; // internal/bench.MaxEntities
const COUNTS = [1_000, 5_000, 10_000, 20_000, 50_000, 100_000];
const BENCH_SPEED = 3; // internal/bench speed (units per second), to match run playback

/** What to draw for each entity. */
const UNITS: { id: string; label: string }[] = [
    { id: "cubes", label: "Cubes" },
    { id: "swordsman", label: "Swordsman" },
    { id: "ranger", label: "Ranger" },
];
const LODS: { lod: Lod | -1; label: string }[] = [
    { lod: -1, label: "Auto" },
    { lod: 0, label: "Full" },
    { lod: 1, label: "Mid" },
    { lod: 2, label: "Far" },
];
const TEAM_COLORS = [new THREE.Color(0x3a6ea5), new THREE.Color(0xa53a3a)];

/** Rolling stats over ~1 second windows. */
class Stats {
    frames = 0;
    frameMs = 0;
    interpMs = 0;
    snapshots = 0;
    snapshotBytes = 0;
    decodeMs = 0;
    simMicros = 0;
    maxGapMs = 0;
    lastSnapshotAt = 0;

    reset(): void {
        this.frames = this.frameMs = this.interpMs = 0;
        this.snapshots = this.snapshotBytes = this.decodeMs = this.simMicros = this.maxGapMs = 0;
    }
}

/**
 * Phase 0 benchmark: Go simulates N wandering entities at 20 Hz and streams binary
 * snapshots; JS interpolates and renders them with one InstancedMesh.
 */
export class Benchmark {
    private ctx: RenderContext | null = null;
    private controls: OrbitControls | null = null;
    private socket: ReturnType<typeof Stream> | null = null;
    private overlay: HTMLElement | null = null;
    private statsEl: HTMLElement | null = null;
    private frameId = 0;
    private statsTimer = 0;

    private readonly scene = new THREE.Scene();
    private readonly camera = new THREE.PerspectiveCamera(50, 1, 1, 2000);
    private readonly units = new UnitsView(
        CAPACITY,
        TICK_MS,
        new THREE.BoxGeometry(0.6, 1.6, 0.6).translate(0, 0.8, 0),
        new THREE.MeshLambertMaterial({ color: 0xc8a060 }),
    );
    private readonly stats = new Stats();
    private requestedCount = 10_000;
    private readonly cubes: THREE.InstancedMesh;
    private unit = "cubes";
    private lod: Lod | -1 = -1; // -1 = automatic by distance
    private readonly models = new Map<string, VatModel>();
    private readonly lodSets = new Map<string, VatLodSet>();
    private active: { set: VatLodSet; clip: VatClip; rate: number } | null = null;
    private readonly interp = new Float32Array(CAPACITY * 3);
    private switchToken = 0;

    constructor(private readonly container: HTMLElement) {
        this.cubes = this.units.mesh;
    }

    start(): void {
        const ctx = new RenderContext(this.container);
        this.ctx = ctx;
        ctx.onResize((w, h) => {
            this.camera.aspect = w / h;
            this.camera.updateProjectionMatrix();
        });

        this.buildScene();

        this.controls = new OrbitControls(this.camera, ctx.renderer.domElement);
        this.controls.target.set(0, 0, 0);
        this.controls.maxPolarAngle = Math.PI / 2.2;
        this.controls.update();

        this.buildOverlay();
        this.connect();

        this.statsTimer = window.setInterval(() => this.renderStats(), 1000);
        this.loop();
    }

    stop(): void {
        cancelAnimationFrame(this.frameId);
        clearInterval(this.statsTimer);
        this.socket?.close();
        this.socket = null;
        this.controls?.dispose();
        this.controls = null;
        this.overlay?.remove();
        this.overlay = null;
        RenderContext.disposeScene(this.scene);
        for (const r of this.lodSets.values()) r.dispose();
        for (const m of this.models.values()) m.dispose();
        this.lodSets.clear();
        this.models.clear();
        this.ctx?.dispose();
        this.ctx = null;
    }

    private connect(): void {
        const socket = Stream(STREAM_NAME);
        socket.binaryType = "arraybuffer";
        socket.onopen = () => this.setCount(this.requestedCount);
        socket.onmessage = (ev: MessageEvent) => {
            const t0 = performance.now();
            const snap = decodeSnapshot(ev.data as ArrayBuffer);
            if (!snap) return;
            this.units.pushSnapshot(snap, t0);
            const t1 = performance.now();

            const s = this.stats;
            if (s.lastSnapshotAt) s.maxGapMs = Math.max(s.maxGapMs, t0 - s.lastSnapshotAt);
            s.lastSnapshotAt = t0;
            s.snapshots++;
            s.snapshotBytes += snap.byteLength;
            s.decodeMs += t1 - t0;
            s.simMicros += snap.simMicros;
        };
        socket.onclose = () => console.info("bench stream closed");
        this.socket = socket;
    }

    private setCount(n: number): void {
        this.requestedCount = n;
        if (this.socket?.readyState === 1) this.socket.send(encodeBenchSetCount(n));
        this.overlay?.querySelectorAll<HTMLButtonElement>("[data-count]").forEach((b) => {
            b.classList.toggle("is-active", Number(b.dataset.count) === n);
        });
    }

    private loop = (): void => {
        this.frameId = requestAnimationFrame(this.loop);
        const t0 = performance.now();
        if (this.active) this.updateVat(t0);
        else this.units.update(t0);
        const t1 = performance.now();
        this.ctx!.renderer.render(this.scene, this.camera);
        const t2 = performance.now();

        this.stats.frames++;
        this.stats.interpMs += t1 - t0;
        this.stats.frameMs += t2 - t0;
    };

    private buildScene(): void {
        this.scene.background = new THREE.Color(0x1d2330);
        this.camera.position.set(0, 180, 220);

        this.scene.add(new THREE.HemisphereLight(0xbfd4ff, 0x3a2f20, 1.0));
        const sun = new THREE.DirectionalLight(0xffffff, 1.2);
        sun.position.set(100, 200, 80);
        this.scene.add(sun);

        const ground = new THREE.Mesh(
            new THREE.PlaneGeometry(MAP_SIZE, MAP_SIZE),
            new THREE.MeshLambertMaterial({ color: 0x3f5a32 }),
        );
        ground.rotation.x = -Math.PI / 2;
        this.scene.add(ground);

        this.scene.add(this.units.mesh);
    }

    private buildOverlay(): void {
        const el = document.createElement("div");
        el.className = "bench-overlay";
        el.innerHTML = `
            <div class="bench-overlay__title">Benchmark</div>
            <div class="bench-overlay__counts">
                ${COUNTS.map((n) => `<button class="segmented__btn" data-count="${n}">${n / 1000}k</button>`).join("")}
            </div>
            <div class="bench-overlay__counts">
                ${UNITS.map((u) => `<button class="segmented__btn" data-unit="${u.id}">${u.label}</button>`).join("")}
            </div>
            <div class="bench-overlay__counts">
                ${LODS.map((l) => `<button class="segmented__btn" data-lod="${l.lod}">LOD ${l.label}</button>`).join("")}
            </div>
            <pre class="bench-overlay__stats">waiting for data…</pre>
            <div class="bench-overlay__hint">LMB rotate · RMB pan · wheel zoom · Esc menu</div>
        `;
        el.addEventListener("click", (e) => {
            const t = e.target as HTMLElement;
            const n = t.closest<HTMLElement>("[data-count]")?.dataset.count;
            if (n) this.setCount(Number(n));
            const unit = t.closest<HTMLElement>("[data-unit]")?.dataset.unit;
            if (unit) void this.setUnit(unit);
            const lod = t.closest<HTMLElement>("[data-lod]")?.dataset.lod;
            if (lod) this.setLod(Number(lod) as Lod | -1);
        });
        this.container.appendChild(el);
        this.overlay = el;
        this.statsEl = el.querySelector(".bench-overlay__stats");
        this.setCount(this.requestedCount);
        this.markActive();
    }

    /** Switches what each entity is drawn as: cubes or an animated VAT model. */
    private async setUnit(unit: string): Promise<void> {
        const token = ++this.switchToken;
        this.unit = unit;
        this.markActive();

        let next: typeof this.active = null;
        if (unit !== "cubes") {
            try {
                next = await this.vatSet(unit);
            } catch (err) {
                console.error(err);
                return;
            }
            if (token !== this.switchToken || !this.ctx) return; // switched again meanwhile
        }

        // Detach the previous renderer(s), attach the new one(s).
        if (this.active) for (const m of this.active.set.meshes) this.scene.remove(m);
        else this.scene.remove(this.cubes);
        this.active = next;
        if (next) {
            next.set.forceLod = this.lod < 0 ? null : (this.lod as Lod);
            for (const m of next.set.meshes) this.scene.add(m);
        } else {
            this.scene.add(this.cubes);
        }
    }

    private setLod(lod: Lod | -1): void {
        this.lod = lod;
        if (this.active) this.active.set.forceLod = lod < 0 ? null : (lod as Lod);
        this.markActive();
    }

    private async vatSet(id: string): Promise<NonNullable<typeof this.active>> {
        let set = this.lodSets.get(id);
        let model = this.models.get(id);
        if (!model) {
            model = await VatModel.load(id);
            this.models.set(id, model);
        }
        if (!set) {
            set = new VatLodSet(model, CAPACITY);
            this.lodSets.set(id, set);
        }
        const clip = model.clip("run") ?? model.clip("walk") ?? model.clip("idle")!;
        const rate = clip.groundSpeed ? BENCH_SPEED / clip.groundSpeed : 1;
        return { set, clip, rate };
    }

    /** VAT mode: interpolate, cull, pick LOD and write instances. */
    private updateVat(now: number): void {
        const { set, clip, rate } = this.active!;
        const n = this.units.interpolate(now, this.interp);
        const p = this.interp;
        set.begin(this.camera);
        for (let i = 0, o = 0; i < n; i++, o += 3) {
            const x = p[o];
            const z = p[o + 1];
            const phase = ((i * 0.618) % 1) * clip.duration; // crowd not in lockstep
            const k = set.slot(x, 0, z, 1, clip, -phase, rate, TEAM_COLORS[i & 1]);
            if (k < 0) continue;
            const m = set.matrices;
            const c = Math.cos(p[o + 2]);
            const s = Math.sin(p[o + 2]);
            m[k] = c; m[k + 1] = 0; m[k + 2] = -s; m[k + 3] = 0;
            m[k + 4] = 0; m[k + 5] = 1; m[k + 6] = 0; m[k + 7] = 0;
            m[k + 8] = s; m[k + 9] = 0; m[k + 10] = c; m[k + 11] = 0;
            m[k + 12] = x; m[k + 13] = 0; m[k + 14] = z; m[k + 15] = 1;
        }
        set.end(now / 1000);
    }

    private markActive(): void {
        this.overlay?.querySelectorAll<HTMLButtonElement>("[data-unit]").forEach((b) => {
            b.classList.toggle("is-active", b.dataset.unit === this.unit);
        });
        this.overlay?.querySelectorAll<HTMLButtonElement>("[data-lod]").forEach((b) => {
            b.classList.toggle("is-active", Number(b.dataset.lod) === this.lod);
        });
    }

    private lodLines(): string[] {
        if (!this.active) return [];
        const st = this.active.set.stats;
        const mode = this.lod < 0 ? "auto" : LODS.find((l) => l.lod === this.lod)!.label.toLowerCase();
        return [
            `lod mode      ${mode}`,
            `drawn         ${(st.perLod[0] + st.perLod[1] + st.perLod[2]).toLocaleString()} (culled ${st.culled.toLocaleString()})`,
            `  full/mid/far ${st.perLod.map((v) => v.toLocaleString()).join(" / ")}`,
        ];
    }

    private renderStats(): void {
        const s = this.stats;
        const info = this.ctx?.renderer.info.render;
        const per = (v: number, n: number) => (n ? v / n : 0);
        const lines = [
            `entities      ${this.units.entityCount.toLocaleString()}`,
            `fps           ${s.frames}`,
            `frame (cpu)   ${per(s.frameMs, s.frames).toFixed(2)} ms`,
            `  interp      ${per(s.interpMs, s.frames).toFixed(2)} ms`,
            `draw calls    ${info?.calls ?? 0}`,
            `triangles     ${((info?.triangles ?? 0) / 1e6).toFixed(1)} M`,
            `unit          ${this.unit}`,
            ...this.lodLines(),
            ``,
            `snapshots/s   ${s.snapshots}`,
            `snapshot      ${(per(s.snapshotBytes, s.snapshots) / 1024).toFixed(1)} KB`,
            `bandwidth     ${(s.snapshotBytes / 1024 / 1024).toFixed(2)} MB/s`,
            `decode+copy   ${per(s.decodeMs, s.snapshots).toFixed(2)} ms`,
            `max gap       ${s.maxGapMs.toFixed(0)} ms`,
            `go sim tick   ${(per(s.simMicros, s.snapshots) / 1000).toFixed(2)} ms`,
        ];
        if (this.statsEl) this.statsEl.textContent = lines.join("\n");
        s.reset();
    }
}
