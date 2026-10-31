import * as THREE from "three";
import { OrbitControls } from "three/examples/jsm/controls/OrbitControls.js";
import { ContentService } from "../../../bindings/dnd-rts/internal/services";
import { RenderContext } from "../../engine/renderer";
import { VatModel, VatRenderer, type Lod } from "../../engine/vat/VatModel";
import { StaticModel } from "../../engine/models/StaticModel";
import { PLAYER_COLORS } from "../view/SoldiersView";

/** frontend/public/models/index.json, written by tools/vatbake and tools/meshbake. */
interface ModelsIndex {
    models: { id: string; type: "vat" | "static" }[];
}

/** One entry of the select: a game entity (or a bare model) and the model that draws it. */
interface Entry {
    key: string; // option value
    label: string;
    group: string;
    model: string;
    type: "vat" | "static";
}

const SPEEDS = [0.25, 0.5, 1, 2];

/**
 * Dev screen: pick any unit / building / machine (or any baked model), look at it
 * from all sides and play its animation clips.
 */
export class ModelOverview {
    private ctx: RenderContext | null = null;
    private controls: OrbitControls | null = null;
    private overlay: HTMLElement | null = null;
    private frameId = 0;

    private readonly scene = new THREE.Scene();
    private readonly camera = new THREE.PerspectiveCamera(40, 1, 0.05, 500);
    private readonly clock = { start: performance.now() };

    private entries: Entry[] = [];
    private readonly vatModels = new Map<string, VatModel>();
    private readonly staticModels = new Map<string, StaticModel>();

    // Current selection.
    private token = 0;
    private vat: { model: VatModel; renderer: VatRenderer } | null = null;
    private staticGroup: THREE.Group | null = null;
    private clip = "";
    private clipStart = 0; // seconds on the overview clock
    private speed = 1;
    private lod: Lod = 0;
    private team = 0;

    constructor(private readonly container: HTMLElement) {}

    start(): void {
        const ctx = new RenderContext(this.container);
        this.ctx = ctx;
        ctx.onResize((w, h) => {
            this.camera.aspect = w / h;
            this.camera.updateProjectionMatrix();
        });
        this.buildScene();
        this.controls = new OrbitControls(this.camera, ctx.renderer.domElement);
        this.controls.enableDamping = true;
        this.buildOverlay();
        void this.loadEntries();
        this.loop();
    }

    stop(): void {
        cancelAnimationFrame(this.frameId);
        this.token++;
        this.controls?.dispose();
        this.controls = null;
        this.overlay?.remove();
        this.overlay = null;
        this.clearSelection();
        for (const m of this.vatModels.values()) m.dispose();
        for (const m of this.staticModels.values()) m.dispose();
        this.vatModels.clear();
        this.staticModels.clear();
        RenderContext.disposeScene(this.scene);
        this.ctx?.dispose();
        this.ctx = null;
    }

    private now(): number {
        return (performance.now() - this.clock.start) / 1000;
    }

    private loop = (): void => {
        this.frameId = requestAnimationFrame(this.loop);
        this.controls?.update();
        if (this.vat) {
            const { model, renderer } = this.vat;
            const clip = model.clip(this.clip);
            renderer.begin();
            if (clip) {
                const i = renderer.add(clip, this.clipStart, this.speed, new THREE.Color(PLAYER_COLORS[this.team]));
                new THREE.Matrix4().toArray(renderer.matrices, i * 16);
            }
            renderer.end(this.now());
            this.updateTime(clip ? (this.now() - this.clipStart) * this.speed : 0, clip?.duration ?? 0, clip?.loop ?? false);
        }
        this.ctx!.renderer.render(this.scene, this.camera);
    };

    private buildScene(): void {
        const sky = new THREE.Color(0x26303c);
        this.scene.background = sky;
        this.scene.add(new THREE.HemisphereLight(0xdfe8ff, 0x4a3e2a, 1.1));
        const sun = new THREE.DirectionalLight(0xfff1dc, 1.6);
        sun.position.set(6, 12, 8);
        this.scene.add(sun);

        const ground = new THREE.Mesh(new THREE.CircleGeometry(40, 64), new THREE.MeshLambertMaterial({ color: 0x3f5a32 }));
        ground.rotation.x = -Math.PI / 2;
        ground.position.y = -0.002;
        this.scene.add(ground);
        const grid = new THREE.GridHelper(40, 40, 0x6f8a5a, 0x4c6640); // 1 m cells
        (grid.material as THREE.Material).transparent = true;
        (grid.material as THREE.Material).opacity = 0.5;
        this.scene.add(grid);
        this.camera.position.set(3, 2.5, 5);
    }

    // ---------- entity list ----------

    private async loadEntries(): Promise<void> {
        let index: ModelsIndex = { models: [] };
        try {
            const res = await fetch("/models/index.json");
            if (res.ok) index = (await res.json()) as ModelsIndex;
        } catch (err) {
            console.warn("models/index.json:", err);
        }
        const typeOf = new Map(index.models.map((m) => [m.id, m.type]));
        const used = new Set<string>();
        const entries: Entry[] = [];
        const add = (group: string, label: string, model: string, key: string) => {
            const type = typeOf.get(model);
            if (!type) return; // not baked (placeholder in the game)
            used.add(model);
            entries.push({ key, label: `${label} · ${model}`, group, model, type });
        };

        try {
            const [hordes, buildings] = await Promise.all([ContentService.HordeTypes(), ContentService.BuildingTypes()]);
            for (const h of hordes ?? []) {
                if (h.model) add(h.hero ? "Heroes" : "Units", h.name, h.model, `horde:${h.id}`);
                if (h.mesh) add("Machines", h.name, h.mesh, `machine:${h.id}`);
            }
            for (const b of buildings ?? []) if (b.model) add("Buildings", b.name, b.model, `building:${b.id}`);
        } catch (err) {
            console.warn("content not available, listing models only:", err);
        }
        for (const m of index.models) {
            if (!used.has(m.id)) entries.push({ key: `model:${m.id}`, label: m.id, group: "Other models", model: m.id, type: m.type });
        }
        const order = ["Units", "Heroes", "Machines", "Buildings", "Other models"];
        entries.sort((a, b) => order.indexOf(a.group) - order.indexOf(b.group) || a.label.localeCompare(b.label));
        this.entries = entries;

        const select = this.overlay?.querySelector<HTMLSelectElement>("[data-entity]");
        if (!select) return;
        select.innerHTML = order
            .map((g) => {
                const items = entries.filter((e) => e.group === g);
                if (!items.length) return "";
                return `<optgroup label="${g}">${items.map((e) => `<option value="${e.key}">${e.label}</option>`).join("")}</optgroup>`;
            })
            .join("");
        if (entries.length) {
            const first = entries.find((e) => e.type === "vat") ?? entries[0];
            select.value = first.key;
            void this.select(first);
        } else {
            this.setInfo("no baked models (frontend/public/models/index.json)");
        }
    }

    // ---------- selection ----------

    private clearSelection(): void {
        if (this.vat) {
            this.scene.remove(this.vat.renderer.mesh);
            this.vat.renderer.dispose();
            this.vat = null;
        }
        if (this.staticGroup) {
            this.scene.remove(this.staticGroup); // geometry / materials belong to the StaticModel
            this.staticGroup = null;
        }
    }

    private async select(entry: Entry): Promise<void> {
        const token = ++this.token;
        this.setInfo("loading…");
        try {
            if (entry.type === "vat") {
                let model = this.vatModels.get(entry.model);
                if (!model) {
                    model = await VatModel.load(entry.model);
                    this.vatModels.set(entry.model, model);
                }
                if (token !== this.token) return;
                this.clearSelection();
                this.showVat(model);
            } else {
                let model = this.staticModels.get(entry.model);
                if (!model) {
                    model = await StaticModel.load(entry.model);
                    this.staticModels.set(entry.model, model);
                }
                if (token !== this.token) return;
                this.clearSelection();
                this.showStatic(model);
            }
        } catch (err) {
            if (token === this.token) this.setInfo(`failed: ${String(err)}`);
            console.error(err);
        }
    }

    private showVat(model: VatModel): void {
        const renderer = new VatRenderer(model, 1, this.lod);
        this.vat = { model, renderer };
        this.scene.add(renderer.mesh);

        const clips = Object.entries(model.clips).sort((a, b) => a[1].start - b[1].start);
        this.clip = model.clip("idle") ? "idle" : (clips[0]?.[0] ?? "");
        this.clipStart = this.now();
        this.renderClips(clips.map(([name]) => name), clips.map(([, c]) => c.loop));
        this.renderLods(3);

        // Frame the rest pose (bounds cover every frame of every clip, which is too wide).
        const { min, max } = model.meta.bounds;
        const height = Math.max(0.5, Math.min(max[1] - min[1], 6));
        this.frame(new THREE.Vector3(0, height / 2, 0), height);
        const m = model.meta;
        const tris = (model.lodIndex[this.lod]?.count ?? 0) / 3;
        this.setInfo(
            `vertices ${m.vertexCount} · triangles ${tris}\n` +
                `clips ${clips.length} · frames ${m.frames} @ ${m.fps} fps\n` +
                `VAT ${m.texture.width}×${m.texture.height}` +
                (m.albedo ? ` · albedo ${m.albedo.width}×${m.albedo.height}` : " · vertex colors"),
        );
    }

    private showStatic(model: StaticModel): void {
        const group = new THREE.Group();
        const level = Math.min(this.lod, model.levels - 1);
        for (const p of model.parts(level)) {
            const mesh = new THREE.Mesh(p.geometry, p.material);
            mesh.matrixAutoUpdate = false;
            mesh.matrix.copy(p.matrix);
            group.add(mesh);
        }
        this.staticGroup = group;
        this.scene.add(group);
        this.clip = "";
        this.renderClips([], []);
        this.renderLods(model.levels);
        this.updateTime(0, 0, false);

        const size = model.bounds.getSize(new THREE.Vector3());
        const center = model.bounds.getCenter(new THREE.Vector3());
        this.frame(center, Math.max(size.x, size.y, size.z));
        let tris = 0;
        group.traverse((o) => {
            if (o instanceof THREE.Mesh) tris += (o.geometry.index?.count ?? o.geometry.attributes.position.count) / 3;
        });
        this.setInfo(`static model · LOD ${level} of ${model.levels}\ntriangles ${tris}\nsize ${size.x.toFixed(1)} × ${size.z.toFixed(1)} m, height ${size.y.toFixed(1)} m`);
    }

    /** Points the camera at `center` from a 3/4 view, far enough to see `size`. */
    private frame(center: THREE.Vector3, size: number): void {
        const dist = (size / 2 / Math.tan(THREE.MathUtils.degToRad(this.camera.fov / 2))) * 1.4;
        const dir = new THREE.Vector3(0.6, 0.45, 1).normalize();
        this.camera.position.copy(center).addScaledVector(dir, dist);
        this.camera.near = Math.max(0.01, dist / 200);
        this.camera.far = dist * 20;
        this.camera.updateProjectionMatrix();
        if (this.controls) {
            this.controls.target.copy(center);
            this.controls.minDistance = size * 0.2;
            this.controls.maxDistance = dist * 5;
            this.controls.update();
        }
    }

    // ---------- overlay ----------

    private buildOverlay(): void {
        const el = document.createElement("div");
        el.className = "bench-overlay overview";
        el.innerHTML = `
            <div class="bench-overlay__title">Model overview</div>
            <select class="overview__select" data-entity><option>loading…</option></select>
            <div class="overview__label">Animation</div>
            <div class="overview__clips" data-clips></div>
            <div class="overview__time" data-vat-only><div class="overview__bar" data-bar></div></div>
            <div class="overview__row" data-vat-only>
                <button class="segmented__btn overview__replay" data-replay>Replay</button>
                <div class="bench-overlay__counts">
                    ${SPEEDS.map((s) => `<button class="segmented__btn" data-speed="${s}">${s}×</button>`).join("")}
                </div>
            </div>
            <div class="overview__label">LOD</div>
            <div class="bench-overlay__counts" data-lods></div>
            <div class="overview__label" data-vat-only>Team</div>
            <div class="bench-overlay__counts" data-vat-only>
                ${PLAYER_COLORS.map((c, i) => `<button class="segmented__btn overview__team" data-team="${i}" style="--team:#${c.toString(16).padStart(6, "0")}"></button>`).join("")}
            </div>
            <pre class="bench-overlay__stats" data-info></pre>
            <div class="bench-overlay__hint">LMB rotate · RMB pan · wheel zoom · Esc menu</div>
        `;
        el.querySelector<HTMLSelectElement>("[data-entity]")!.addEventListener("change", (e) => {
            const entry = this.entries.find((x) => x.key === (e.target as HTMLSelectElement).value);
            if (entry) void this.select(entry);
        });
        el.addEventListener("click", (e) => {
            const t = (e.target as HTMLElement).closest<HTMLElement>("button");
            if (!t) return;
            if (t.dataset.clip !== undefined) this.playClip(t.dataset.clip);
            else if (t.dataset.replay !== undefined) this.playClip(this.clip);
            else if (t.dataset.speed) this.setSpeed(Number(t.dataset.speed));
            else if (t.dataset.lod) this.setLod(Number(t.dataset.lod) as Lod);
            else if (t.dataset.team) {
                this.team = Number(t.dataset.team);
                this.markActive();
            }
        });
        this.container.appendChild(el);
        this.overlay = el;
        this.markActive();
    }

    private renderClips(names: string[], loops: boolean[]): void {
        // Playback and team color only mean something for animated (VAT) models.
        this.overlay?.querySelectorAll<HTMLElement>("[data-vat-only]").forEach((e) => (e.hidden = names.length === 0));
        const box = this.overlay?.querySelector<HTMLElement>("[data-clips]");
        if (!box) return;
        box.innerHTML = names.length
            ? names.map((n, i) => `<button class="segmented__btn" data-clip="${n}" title="${loops[i] ? "loop" : "once"}">${n}${loops[i] ? " ⟳" : ""}</button>`).join("")
            : `<span class="overview__none">no animations</span>`;
        this.markActive();
    }

    private renderLods(levels: number): void {
        const box = this.overlay?.querySelector<HTMLElement>("[data-lods]");
        if (!box) return;
        const labels = ["Full", "Mid", "Far"];
        box.innerHTML = labels
            .slice(0, levels)
            .map((l, i) => `<button class="segmented__btn" data-lod="${i}">${l}</button>`)
            .join("");
        this.markActive();
    }

    private playClip(name: string): void {
        if (!name) return;
        this.clip = name;
        this.clipStart = this.now();
        this.markActive();
    }

    private setSpeed(speed: number): void {
        // Keep the current position in the clip when changing speed.
        const t = (this.now() - this.clipStart) * this.speed;
        this.speed = speed;
        this.clipStart = this.now() - t / speed;
        this.markActive();
    }

    private setLod(lod: Lod): void {
        this.lod = lod;
        const key = this.overlay?.querySelector<HTMLSelectElement>("[data-entity]")?.value;
        const entry = this.entries.find((e) => e.key === key);
        if (entry) {
            const clip = this.clip;
            const start = this.clipStart;
            void this.select(entry).then(() => {
                if (clip && this.vat?.model.clip(clip)) {
                    this.clip = clip;
                    this.clipStart = start;
                    this.markActive();
                }
            });
        }
        this.markActive();
    }

    private markActive(): void {
        const el = this.overlay;
        if (!el) return;
        el.querySelectorAll<HTMLElement>("[data-clip]").forEach((b) => b.classList.toggle("is-active", b.dataset.clip === this.clip));
        el.querySelectorAll<HTMLElement>("[data-speed]").forEach((b) => b.classList.toggle("is-active", Number(b.dataset.speed) === this.speed));
        el.querySelectorAll<HTMLElement>("[data-lod]").forEach((b) => b.classList.toggle("is-active", Number(b.dataset.lod) === this.lod));
        el.querySelectorAll<HTMLElement>("[data-team]").forEach((b) => b.classList.toggle("is-active", Number(b.dataset.team) === this.team));
    }

    private updateTime(t: number, duration: number, loop: boolean): void {
        const bar = this.overlay?.querySelector<HTMLElement>("[data-bar]");
        if (!bar) return;
        const p = duration > 0 ? (loop ? (t % duration) / duration : Math.min(t / duration, 1)) : 0;
        bar.style.width = `${(p * 100).toFixed(1)}%`;
    }

    private setInfo(text: string): void {
        const info = this.overlay?.querySelector<HTMLElement>("[data-info]");
        if (info) info.textContent = text;
    }
}
