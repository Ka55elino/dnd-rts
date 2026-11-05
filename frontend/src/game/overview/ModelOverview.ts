import * as THREE from "three";
import { OrbitControls } from "three/examples/jsm/controls/OrbitControls.js";
import { ContentService } from "../../../bindings/dnd-rts/internal/services";
import { RenderContext } from "../../engine/renderer";
import { VatModel, VatRenderer, type Lod, type VatClip } from "../../engine/vat/VatModel";
import { StaticModel } from "../../engine/models/StaticModel";
import { PLAYER_COLORS } from "../view/SoldiersView";
import { LIGHT_PRESETS, LightRig, lightPreset, setupRenderer } from "../../engine/lighting";

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
/** Poses shown side by side in strip mode. */
const STRIP = 6;
/** How often the selected model's model.json is checked for a rebuild (ms). */
const POLL_MS = 1000;
/** Backgrounds: "sky" follows the light preset. */
const BACKGROUNDS: { id: string; label: string; color?: number }[] = [
    { id: "sky", label: "Sky" },
    { id: "dark", label: "Dark", color: 0x16191e },
    { id: "grey", label: "Grey", color: 0x7c838b },
    { id: "light", label: "Light", color: 0xd9dde2 },
];
const GROUNDS: { id: string; label: string; base?: number; speck?: [number, number] }[] = [
    { id: "grass", label: "Grass", base: 0x3f5a32, speck: [0x2f4626, 0x5a7442] },
    { id: "dirt", label: "Dirt", base: 0x6b5a44, speck: [0x55462f, 0x857258] },
    { id: "none", label: "None" },
];

/**
 * Dev screen: pick any unit / building / machine (or any baked model), look at it
 * from all sides and play its animation clips: pause, scrub, step frame by frame, or show
 * a strip of poses side by side. Models built by tools/modelc reload automatically after
 * a rebuild (`modelc watch <id>`), keeping the camera, clip and time.
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
    private paused = false;
    private pausedT = 0; // clip time (s) while paused
    private strip = false;
    private stripSpacing = 1;
    private pollTimer = 0;
    private reloading = false;
    private reloadNote = "";
    private readonly lights = new LightRig(6, 2048);
    private light = "day";
    private background = "sky";
    private groundId = "grass";
    private smooth = true;
    private shadows = true;
    private ground: THREE.Mesh | null = null;
    private grid: THREE.GridHelper | null = null;

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
        window.addEventListener("keydown", this.onKey);
        this.pollTimer = window.setInterval(() => void this.poll(), POLL_MS);
        this.loop();
    }

    stop(): void {
        cancelAnimationFrame(this.frameId);
        window.removeEventListener("keydown", this.onKey);
        clearInterval(this.pollTimer);
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

    /** Current time in the clip (seconds at 1x). */
    private clipTime(): number {
        return this.paused ? this.pausedT : (this.now() - this.clipStart) * this.speed;
    }

    private loop = (): void => {
        this.frameId = requestAnimationFrame(this.loop);
        this.controls?.update();
        if (this.vat) {
            const { model, renderer } = this.vat;
            const clip = model.clip(this.clip);
            const now = this.now();
            const t = this.clipTime();
            renderer.begin();
            if (clip) {
                const team = new THREE.Color(PLAYER_COLORS[this.team]);
                const n = this.strip ? STRIP : 1;
                for (let k = 0; k < n; k++) {
                    // Strip: a looping clip shows n phases moving together; a one-shot clip n fixed moments.
                    const tk = n === 1 ? t : clip.loop ? t + (k * clip.duration) / n : (k * clip.duration) / (n - 1);
                    const i = renderer.add(clip, now - tk, 1, team); // the time is passed as the start
                    if (i < 0) break;
                    this.poseMatrix(k, n).toArray(renderer.matrices, i * 16);
                }
            }
            renderer.end(now);
            this.updateTime(t, clip);
        }
        this.ctx!.renderer.render(this.scene, this.camera);
    };

    /** Strip: poses in a row along +X, turned to face +X (side view from the default camera). */
    private poseMatrix(k: number, n: number): THREE.Matrix4 {
        const m = new THREE.Matrix4();
        if (n === 1) return m;
        return m.makeRotationY(Math.PI / 2).setPosition((k - (n - 1) / 2) * this.stripSpacing, 0, 0);
    }

    private onKey = (e: KeyboardEvent): void => {
        if (!this.vat || (e.target as HTMLElement).closest("select, input, textarea")) return;
        if (e.key === " ") this.togglePause();
        else if (e.key === "ArrowLeft") this.step(-1);
        else if (e.key === "ArrowRight") this.step(1);
        else return;
        e.preventDefault();
    };

    // ---------- hot reload ----------

    /** Reloads the selected model when tools/modelc rebuilt it (the "build" stamp in model.json changed). */
    private async poll(): Promise<void> {
        const model = this.vat?.model;
        if (!model || this.reloading || !model.meta.build) return;
        let build: string | undefined;
        try {
            const r = await fetch(`/models/${model.id}/model.json`, { cache: "no-store" });
            if (!r.ok) return;
            build = ((await r.json()) as { build?: string }).build;
        } catch {
            return; // mid-write or the server is gone: try again next time
        }
        if (!build || build === model.meta.build || this.vat?.model !== model) return;
        this.reloading = true;
        try {
            const fresh = await VatModel.load(model.id, true);
            this.vatModels.set(model.id, fresh);
            if (this.vat?.model === model) {
                this.clearSelection();
                this.reloadNote = `reloaded ${new Date().toLocaleTimeString()}`;
                this.showVat(fresh, true);
            }
            model.dispose();
        } catch (err) {
            console.warn(`reload ${model.id}:`, err); // half-written files: the next poll retries
        } finally {
            this.reloading = false;
        }
    }

    private buildScene(): void {
        this.scene.background = new THREE.Color();
        setupRenderer(this.ctx!.renderer);
        this.scene.add(...this.lights.objects);

        const ground = new THREE.Mesh(new THREE.CircleGeometry(40, 64), new THREE.MeshLambertMaterial());
        ground.rotation.x = -Math.PI / 2;
        ground.position.y = -0.002;
        ground.receiveShadow = true;
        this.scene.add(ground);
        this.ground = ground;
        const grid = new THREE.GridHelper(40, 40, 0x6f8a5a, 0x4c6640); // 1 m cells
        (grid.material as THREE.Material).transparent = true;
        (grid.material as THREE.Material).opacity = 0.25;
        this.scene.add(grid);
        this.grid = grid;
        this.camera.position.set(3, 2.5, 5);
        this.applyLook();
    }

    /** Light preset, background, ground and shadows from the current settings. */
    private applyLook(): void {
        const renderer = this.ctx?.renderer;
        if (!renderer) return;
        const preset = lightPreset(this.light);
        this.lights.apply(preset, this.scene, renderer);
        const bg = BACKGROUNDS.find((b) => b.id === this.background);
        (this.scene.background as THREE.Color).set(bg?.color ?? preset.sky);
        this.lights.shadows = this.shadows;
        renderer.shadowMap.enabled = this.shadows;
        const g = GROUNDS.find((x) => x.id === this.groundId) ?? GROUNDS[0];
        if (this.ground) {
            this.ground.visible = g.base !== undefined;
            const mat = this.ground.material as THREE.MeshLambertMaterial;
            mat.map?.dispose();
            mat.map = g.base !== undefined ? groundTexture(g.base, g.speck!) : null;
            mat.needsUpdate = true;
        }
        if (this.grid) this.grid.visible = g.base !== undefined;
        this.markActive();
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

    /** keep: a hot reload — keep the clip, its time and the camera. */
    private showVat(model: VatModel, keep = false): void {
        const renderer = new VatRenderer(model, STRIP, this.lod, { smooth: this.smooth, shadows: true });
        this.vat = { model, renderer };
        this.scene.add(renderer.mesh);

        const clips = Object.entries(model.clips).sort((a, b) => a[1].start - b[1].start);
        if (!keep || !model.clip(this.clip)) {
            this.clip = model.clip("idle") ? "idle" : (clips[0]?.[0] ?? "");
            this.clipStart = this.now();
            this.pausedT = 0;
        }
        if (!keep) this.reloadNote = "";
        this.renderClips(clips.map(([name]) => name), clips.map(([, c]) => c.loop));
        this.renderLods(3);

        // Strip spacing: the model's depth over all frames (it becomes the row direction).
        const { min, max } = model.meta.bounds;
        this.stripSpacing = Math.max(0.8, (max[2] - min[2]) * 1.05);
        if (!keep) this.frameVat();
        const m = model.meta;
        const tris = (model.lodIndex[this.lod]?.count ?? 0) / 3;
        this.setInfo(
            `vertices ${m.vertexCount} · triangles ${tris}\n` +
                `clips ${clips.length} · frames ${m.frames} @ ${m.fps} fps\n` +
                `VAT ${m.texture.width}×${m.texture.height}` +
                (m.albedo ? ` · albedo ${m.albedo.width}×${m.albedo.height}` : " · vertex colors") +
                (m.build ? `\nbuilt ${m.build.replace("T", " ")}` : "") +
                (this.reloadNote ? ` · ${this.reloadNote}` : ""),
        );
    }

    /** Frames the rest pose, or the whole strip (bounds cover every frame of every clip, which is too wide). */
    private frameVat(): void {
        if (!this.vat) return;
        const { min, max } = this.vat.model.meta.bounds;
        const height = Math.max(0.5, Math.min(max[1] - min[1], 6));
        this.lights.setArea(this.strip ? STRIP * this.stripSpacing + 3 : Math.max(4, height * 3));
        this.lights.follow(new THREE.Vector3(0, 0, 0));
        const center = new THREE.Vector3(0, height / 2, 0);
        if (this.strip) this.frame(center, STRIP * this.stripSpacing * 0.75, new THREE.Vector3(0, 0.15, 1));
        else this.frame(center, height);
    }

    private showStatic(model: StaticModel): void {
        const group = new THREE.Group();
        const level = Math.min(this.lod, model.levels - 1);
        for (const p of model.parts(level)) {
            const mesh = new THREE.Mesh(p.geometry, p.material);
            mesh.castShadow = mesh.receiveShadow = true;
            mesh.matrixAutoUpdate = false;
            mesh.matrix.copy(p.matrix);
            group.add(mesh);
        }
        this.staticGroup = group;
        this.scene.add(group);
        this.clip = "";
        this.renderClips([], []);
        this.renderLods(model.levels);
        this.updateTime(0, undefined);

        const size = model.bounds.getSize(new THREE.Vector3());
        const center = model.bounds.getCenter(new THREE.Vector3());
        this.frame(center, Math.max(size.x, size.y, size.z));
        this.lights.setArea(Math.max(size.x, size.z) * 1.6 + 4);
        this.lights.follow(new THREE.Vector3(center.x, 0, center.z));
        let tris = 0;
        group.traverse((o) => {
            if (o instanceof THREE.Mesh) tris += (o.geometry.index?.count ?? o.geometry.attributes.position.count) / 3;
        });
        this.setInfo(`static model · LOD ${level} of ${model.levels}\ntriangles ${tris}\nsize ${size.x.toFixed(1)} × ${size.z.toFixed(1)} m, height ${size.y.toFixed(1)} m`);
    }

    /** Points the camera at `center` from a 3/4 view, far enough to see `size`. */
    private frame(center: THREE.Vector3, size: number, from = new THREE.Vector3(0.6, 0.45, 1)): void {
        const dist = (size / 2 / Math.tan(THREE.MathUtils.degToRad(this.camera.fov / 2))) * 1.4;
        const dir = from.clone().normalize();
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
            <div class="overview__time" data-vat-only data-scrub title="click or drag to scrub"><div class="overview__bar" data-bar></div></div>
            <div class="overview__row" data-vat-only>
                <button class="segmented__btn overview__replay" data-step="-1" title="previous frame (←)">◀</button>
                <button class="segmented__btn overview__replay overview__play" data-pause title="play / pause (Space)">❚❚</button>
                <button class="segmented__btn overview__replay" data-step="1" title="next frame (→)">▶</button>
                <span class="overview__frame" data-frame></span>
            </div>
            <div class="overview__row" data-vat-only>
                <button class="segmented__btn overview__replay" data-replay>Replay</button>
                <div class="bench-overlay__counts">
                    ${SPEEDS.map((s) => `<button class="segmented__btn" data-speed="${s}">${s}×</button>`).join("")}
                </div>
            </div>
            <div class="overview__row" data-vat-only>
                <button class="segmented__btn overview__replay" data-strip title="${STRIP} poses side by side, seen from the side">Strip</button>
            </div>
            <div class="overview__label">Light</div>
            <div class="bench-overlay__counts overview__wrap">
                ${LIGHT_PRESETS.map((p) => `<button class="segmented__btn" data-light="${p.id}">${p.label}</button>`).join("")}
            </div>
            <div class="overview__label">Background</div>
            <div class="bench-overlay__counts overview__wrap">
                ${BACKGROUNDS.map((b) => `<button class="segmented__btn" data-bg="${b.id}">${b.label}</button>`).join("")}
            </div>
            <div class="overview__label">Ground</div>
            <div class="bench-overlay__counts overview__wrap">
                ${GROUNDS.map((g) => `<button class="segmented__btn" data-ground="${g.id}">${g.label}</button>`).join("")}
            </div>
            <div class="overview__label">Shading</div>
            <div class="bench-overlay__counts overview__wrap">
                <button class="segmented__btn" data-smooth="1" title="baked normals + specular (modelc models)">Smooth</button>
                <button class="segmented__btn" data-smooth="0" title="flat low-poly faces">Flat</button>
                <button class="segmented__btn" data-shadows title="sun shadows">Shadows</button>
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
            else if (t.dataset.replay !== undefined) {
                this.paused = false;
                this.playClip(this.clip);
            } else if (t.dataset.pause !== undefined) this.togglePause();
            else if (t.dataset.step) this.step(Number(t.dataset.step));
            else if (t.dataset.strip !== undefined) {
                this.strip = !this.strip;
                this.frameVat();
                this.markActive();
            }
            else if (t.dataset.speed) this.setSpeed(Number(t.dataset.speed));
            else if (t.dataset.lod) this.setLod(Number(t.dataset.lod) as Lod);
            else if (t.dataset.light) {
                this.light = t.dataset.light;
                this.applyLook();
            } else if (t.dataset.bg) {
                this.background = t.dataset.bg;
                this.applyLook();
            } else if (t.dataset.ground) {
                this.groundId = t.dataset.ground;
                this.applyLook();
            } else if (t.dataset.shadows !== undefined) {
                this.shadows = !this.shadows;
                this.applyLook();
            } else if (t.dataset.smooth) {
                this.smooth = t.dataset.smooth === "1";
                if (this.vat) {
                    const model = this.vat.model;
                    this.clearSelection();
                    this.showVat(model, true);
                }
                this.markActive();
            } else if (t.dataset.team) {
                this.team = Number(t.dataset.team);
                this.markActive();
            }
        });
        const scrub = el.querySelector<HTMLElement>("[data-scrub]")!;
        const scrubTo = (e: PointerEvent) => {
            const r = scrub.getBoundingClientRect();
            this.scrub((e.clientX - r.left) / Math.max(r.width, 1));
        };
        scrub.addEventListener("pointerdown", (e) => {
            scrub.setPointerCapture(e.pointerId);
            scrubTo(e);
        });
        scrub.addEventListener("pointermove", (e) => {
            if (scrub.hasPointerCapture(e.pointerId)) scrubTo(e);
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
        this.pausedT = 0;
        this.markActive();
    }

    private togglePause(): void {
        if (this.paused) {
            this.clipStart = this.now() - this.pausedT / this.speed;
            this.paused = false;
        } else {
            this.pausedT = this.clipTime();
            this.paused = true;
        }
        this.markActive();
    }

    private pause(): void {
        if (!this.paused) this.togglePause();
    }

    /** One frame back / forward (pauses). Loops wrap around; one-shot clips stop at the ends. */
    private step(d: number): void {
        const clip = this.vat?.model.clip(this.clip);
        if (!clip) return;
        this.pause();
        const fps = this.vat!.model.meta.fps;
        let f = this.frameOf(this.pausedT, clip) + d;
        f = clip.loop ? (f + clip.frames) % clip.frames : Math.max(0, Math.min(f, clip.frames - 1));
        this.pausedT = (f + 0.001) / fps;
    }

    /** Scrub to a fraction 0..1 of the clip (pauses). */
    private scrub(frac: number): void {
        const clip = this.vat?.model.clip(this.clip);
        if (!clip) return;
        this.pause();
        this.pausedT = Math.max(0, Math.min(frac, clip.loop ? 0.9999 : 1)) * clip.duration;
    }

    /** The VAT frame shown at clip time t (as the shader picks it, before blending to the next). */
    private frameOf(t: number, clip: VatClip): number {
        const vt = t * this.vat!.model.meta.fps;
        return Math.floor(clip.loop ? ((vt % clip.frames) + clip.frames) % clip.frames : Math.min(vt, clip.frames - 1));
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
        const play = el.querySelector<HTMLElement>("[data-pause]");
        if (play) play.textContent = this.paused ? "▶︎ Play" : "❚❚";
        el.querySelector<HTMLElement>("[data-strip]")?.classList.toggle("is-active", this.strip);
        el.querySelectorAll<HTMLElement>("[data-light]").forEach((b) => b.classList.toggle("is-active", b.dataset.light === this.light));
        el.querySelectorAll<HTMLElement>("[data-bg]").forEach((b) => b.classList.toggle("is-active", b.dataset.bg === this.background));
        el.querySelectorAll<HTMLElement>("[data-ground]").forEach((b) => b.classList.toggle("is-active", b.dataset.ground === this.groundId));
        el.querySelectorAll<HTMLElement>("[data-smooth]").forEach((b) => b.classList.toggle("is-active", (b.dataset.smooth === "1") === this.smooth));
        el.querySelector<HTMLElement>("[data-shadows]")?.classList.toggle("is-active", this.shadows);
    }

    private lastTimeText = "";

    private updateTime(t: number, clip: VatClip | undefined): void {
        const bar = this.overlay?.querySelector<HTMLElement>("[data-bar]");
        if (!bar) return;
        const d = clip?.duration ?? 0;
        const local = clip && d > 0 ? (clip.loop ? ((t % d) + d) % d : Math.min(t, d)) : 0;
        bar.style.width = `${(d > 0 ? (local / d) * 100 : 0).toFixed(1)}%`;
        const text = clip ? `frame ${this.frameOf(t, clip) + 1} / ${clip.frames} · ${local.toFixed(2)} s` : "";
        if (text !== this.lastTimeText) {
            this.lastTimeText = text;
            const el = this.overlay?.querySelector<HTMLElement>("[data-frame]");
            if (el) el.textContent = text;
        }
    }

    private setInfo(text: string): void {
        const info = this.overlay?.querySelector<HTMLElement>("[data-info]");
        if (info) info.textContent = text;
    }
}

/** A tiling ground texture: a base color with soft specks (reads as grass / earth at a glance). */
function groundTexture(base: number, speck: [number, number]): THREE.CanvasTexture {
    const size = 256;
    const canvas = document.createElement("canvas");
    canvas.width = canvas.height = size;
    const g = canvas.getContext("2d")!;
    g.fillStyle = `#${base.toString(16).padStart(6, "0")}`;
    g.fillRect(0, 0, size, size);
    let seed = 7;
    const rnd = () => ((seed = (seed * 16807) % 2147483647) / 2147483647);
    for (let i = 0; i < 2600; i++) {
        const c = speck[i % 2];
        g.globalAlpha = 0.25 + rnd() * 0.45;
        g.fillStyle = `#${c.toString(16).padStart(6, "0")}`;
        const r = 0.6 + rnd() * 2.2;
        g.beginPath();
        g.ellipse(rnd() * size, rnd() * size, r, r * (0.4 + rnd()), rnd() * Math.PI, 0, Math.PI * 2);
        g.fill();
    }
    const tex = new THREE.CanvasTexture(canvas);
    tex.colorSpace = THREE.SRGBColorSpace;
    tex.wrapS = tex.wrapT = THREE.RepeatWrapping;
    tex.repeat.set(20, 20);
    tex.anisotropy = 4;
    return tex;
}
