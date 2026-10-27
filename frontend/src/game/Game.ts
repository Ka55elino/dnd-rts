import * as THREE from "three";
import { ContentService } from "../../bindings/dnd-rts/internal/services";
import { RenderContext } from "../engine/renderer";
import { RTSCamera } from "../engine/camera/RTSCamera";
import { Terrain, decodeBase64, decodeSamples } from "../engine/terrain/Terrain";
import { GameConnection } from "../net/gameConnection";
import { SoldiersView, type HordeTypeInfo } from "./view/SoldiersView";
import { MoveMarker } from "./view/MoveMarker";
import { ProjectilesView } from "./view/ProjectilesView";
import { HordeLabels } from "./hud/HordeLabels";
import { Effects } from "./view/Effects";
import { StatusEffectsView } from "./view/StatusEffectsView";
import { AbilityPreview } from "./view/AbilityPreview";
import { EVENT_CHAIN, EVENT_CONE, NEUTRAL, STATUS_GROUNDED, STATUS_ON_WALL, isConstructing, type GameSnapshot } from "../net/protocol";
import { VatModel } from "../engine/vat/VatModel";
import { SelectionController } from "./input/SelectionController";
import { BuildingsView, type BuildingTypeInfo } from "./view/BuildingsView";
import { DevConsole } from "./hud/DevConsole";
import { MachinesView } from "./view/MachinesView";
import { loadCursors, setCursor, type CursorName } from "./hud/cursors";
import { BuildPanel } from "./hud/BuildPanel";
import { Placement } from "./input/Placement";
import { Economy } from "./hud/economy";
import { Tech } from "./hud/tech";
import { Hud } from "./hud/Hud";
import { Minimap } from "./hud/Minimap";
import { CommandCard, type TargetMode } from "./hud/CommandCard";

const DEFAULT_MAP = "plains";

/** Ring colors of ability visuals by damage type. */
const DAMAGE_COLORS: Record<string, number> = {
    fire: 0xff7a2a,
    ice: 0x8fd4ff,
    lightning: 0xfff07a,
    acid: 0xc8e84a,
    poison: 0x7be05a,
    necrotic: 0x9a6aff,
    radiant: 0xfff4c8,
    crush: 0xffc860,
    slash: 0xffc860,
    pierce: 0xffc860,
};
const TICK_MS = 50; // 20 Hz, internal/sim.TickRate
const MAX_SOLDIERS = 20_000;

/**
 * Game client lifecycle: start() loads the map, connects to the Go session and
 * builds the scene; stop() releases everything (stream, WebGL, listeners).
 */
export class Game {
    private ctx: RenderContext | null = null;
    private lastHeroes: import("../net/protocol").HeroSnap[] = [];
    private rtsCamera: RTSCamera | null = null;
    private terrain: Terrain | null = null;
    private soldiers: SoldiersView | null = null;
    private buildings: BuildingsView | null = null;
    private devConsole: DevConsole | null = null;
    private buildPanel: BuildPanel | null = null;
    /** Last mouse position over the canvas (client px), for the cursor. */
    private mouseClient: { x: number; y: number } | null = null;
    private cursorAt = 0;
    private readonly cursorRay = new THREE.Raycaster();
    private placement: Placement | null = null;
    private hudView: Hud | null = null;
    private minimap: Minimap | null = null;
    private card: CommandCard | null = null;
    /** An order from the command card waiting for its target click. */
    private targetMode: TargetMode | null = null;
    private readonly anchorTmp = new THREE.Vector3();
    /** The player this client controls (from snapshots; the dev console can switch it). */
    private localPlayer = 0;
    private localFaction = -1; // index into factions
    /** Last mouse position over the canvas (NDC), for console commands that act "at the cursor". */
    private readonly mouseNdc = new THREE.Vector2();
    private readonly raycaster = new THREE.Raycaster();
    private selection: SelectionController | null = null;
    private connection: GameConnection | null = null;
    private marker = new MoveMarker();
    private projectiles = new ProjectilesView(TICK_MS);
    private effects = new Effects();
    private statusFx = new StatusEffectsView();
    private preview = new AbilityPreview();
    private models = new Map<string, VatModel>();
    private debugPanel: HTMLElement | null = null;
    private debugTimer = 0;
    private frames = 0;
    private lastSnap: GameSnapshot | null = null;
    /** Last ability cast tick seen per horde, to trigger effects once. */
    /** Abilities (index = kind) and the ability waiting for its target. */
    private abilityInfos: import("./hud/CommandCard").AbilityInfo[] = [];
    private castKind = -1;
    private castHordes: number[] = [];
    private hud: HTMLElement | null = null;
    private labels: HordeLabels | null = null;
    private frameId = 0;
    private panelTimer = 0;
    private stopped = false;
    private mapId = DEFAULT_MAP;
    private navOverlay = false;

    private readonly scene = new THREE.Scene();
    private readonly clock = new THREE.Clock();

    constructor(private readonly container: HTMLElement) {}

    start(mapId = DEFAULT_MAP): void {
        this.mapId = mapId;
        window.addEventListener("keydown", this.onKeyDown);
        this.hud = document.createElement("div");
        this.hud.className = "game-hud";
        this.hud.innerHTML = `
            <pre class="game-hud__debug" hidden></pre>
            <div class="game-hud__status">Loading map…</div>
        `;
        this.debugPanel = this.hud.querySelector(".game-hud__debug");
        this.labels = new HordeLabels(this.hud);
        this.container.appendChild(this.hud);

        this.load(mapId).catch((err) => {
            console.error(err);
            this.setStatus(`Failed to load map: ${err?.message ?? err}`);
        });
    }

    /** Esc: clears the selection first; returns false when there was nothing to clear. */
    handleEscape(): boolean {
        if (this.targetMode) {
            this.setTargetMode(null);
            return true;
        }
        if (this.placement?.placing) {
            this.placement.cancel();
            return true;
        }
        if (this.buildPanel?.active) {
            this.buildPanel.clear();
            return true;
        }
        if (this.selection && this.selection.selected.size > 0) {
            this.selection.clear();
            return true;
        }
        return false;
    }

    stop(): void {
        this.stopped = true;
        window.removeEventListener("keydown", this.onKeyDown);
        cancelAnimationFrame(this.frameId);
        clearInterval(this.panelTimer);
        clearInterval(this.debugTimer);
        this.connection?.close();
        this.connection = null;
        this.selection?.dispose();
        this.selection = null;
        this.rtsCamera?.dispose();
        this.rtsCamera = null;
        this.soldiers?.machines?.dispose();
        this.soldiers?.dispose();
        this.buildings?.dispose();
        this.buildings = null;
        this.statusFx.dispose();
        this.preview.dispose();
        for (const m of this.models.values()) m.dispose();
        RenderContext.disposeScene(this.scene);
        this.ctx?.dispose();
        this.ctx = null;
        this.labels?.dispose();
        this.labels = null;
        this.devConsole?.dispose();
        this.devConsole = null;
        this.buildPanel?.dispose();
        this.buildPanel = null;
        this.placement?.dispose();
        this.placement = null;
        this.hudView?.dispose();
        this.hudView = null;
        this.minimap?.dispose();
        this.minimap = null;
        this.card?.dispose();
        this.card = null;
        this.hud?.remove();
        this.hud = null;
    }

    private async load(mapId: string): Promise<void> {
        const [data, hordeTypes, buildingTypes, factions, navGrid, resourceInfos, abilityInfos, upgradeInfos] = await Promise.all([
            ContentService.Terrain(mapId),
            ContentService.HordeTypes(),
            ContentService.BuildingTypes(),
            ContentService.Factions(),
            ContentService.NavGrid(mapId),
            ContentService.Resources(),
            ContentService.Abilities(),
            ContentService.Upgrades(),
        ]);
        if (this.stopped || !data) return; // left the screen while loading

        const terrain = new Terrain({
            width: data.width,
            height: data.height,
            cols: data.cols,
            rows: data.rows,
            cellSize: data.cellSize,
            maxHeight: data.maxHeight,
            samples: decodeSamples(data.samples ?? ""),
        });
        this.terrain = terrain;

        const types: HordeTypeInfo[] = (hordeTypes ?? []).map((t) => ({
            name: t.name,
            size: t.size,
            radius: t.radius,
            hero: t.hero,
            abilities: t.abilities ?? [],
            model: t.model,
            speed: t.speed,
            worker: t.tags?.includes("worker") ?? false,
            heroXp: t.heroXp ?? [],
            flying: t.flying,
            ranged: t.ranged ?? false,
            antiAir: t.antiAir ?? false,
            climbs: t.climbs ?? false,
        }));
        const btypes: BuildingTypeInfo[] = (buildingTypes ?? []).map((t) => ({
            id: t.id,
            name: t.name,
            kind: t.kind,
            model: t.model,
            modelScale: t.modelScale,
            width: t.width,
            depth: t.depth,
            placement: t.placement ?? [],
            buildTime: t.buildTime,
            noBuildRadius: t.noBuildRadius,
            plots: t.plots ?? [],
            base: t.base ? { outline: t.base.outline ?? [], gates: t.base.gates ?? [], towers: t.base.towers ?? [] } : null,
            walkHeight: t.walkHeight ?? 0,
            toggles: t.toggles ?? [],
            garrison: t.garrison ?? false,
            cost: t.cost ?? {},
            trains: t.trains ?? [],
            research: t.research ?? [],
            requires: t.requires,
        }));
        const economy = new Economy(resourceInfos ?? []);
        this.abilityInfos = (abilityInfos ?? []).map((a) => ({ ...a }));
        const tech = new Tech((upgradeInfos ?? []).map((u) => ({ ...u })), btypes);
        const factionList = factions ?? [];
        let queues = new Map<number, import("../net/protocol").QueueSnap>();
        const [models, buildings] = await Promise.all([
            loadModels(types),
            BuildingsView.create(btypes, this.hud!, (x, z) => terrain.heightAt(x, z)),
        ]);
        this.models = models;
        this.buildings = buildings;
        const wallType = btypes.find((t) => t.kind === "wall");
        if (this.stopped) {
            buildings.dispose();
            return;
        }
        const soldiers = new SoldiersView(MAX_SOLDIERS, TICK_MS, types, models);
        soldiers.wallHeight = wallType?.walkHeight ?? 0;
        soldiers.standHeight = (x, z) => buildings.standHeight(x, z);
        const machines = await MachinesView.load((hordeTypes ?? []).map((t) => t.mesh ?? ""));
        soldiers.machines = machines;
        this.soldiers = soldiers;

        const ctx = new RenderContext(this.container);
        this.ctx = ctx;
        this.container.prepend(ctx.renderer.domElement); // canvas under the HUD

        const margin = 20;
        const camera = new RTSCamera(ctx.renderer.domElement, {
            boundsX: data.width / 2 - margin,
            boundsZ: data.height / 2 - margin,
            heightAt: (x, z) => terrain.heightAt(x, z),
        });
        this.rtsCamera = camera;
        ctx.onResize((w, h) => camera.setAspect(w / h));

        this.buildScene(terrain, soldiers, buildings);

        this.connection = new GameConnection(mapId, (snap, at) => {
            soldiers.pushSnapshot(snap, at);
            buildings.pushSnapshot(snap);
            economy.resources = snap.resources;
            tech.update(snap);
            this.lastHeroes = snap.heroes;
            queues = new Map(snap.queues.map((q) => [q.building, q]));
            this.localFaction = snap.players.find((p) => p.id === snap.localPlayer)?.faction ?? -1;
            if (snap.localPlayer !== this.localPlayer) this.switchPlayer(snap.localPlayer, snap);
            this.hudView?.update(snap);
            this.projectiles.pushSnapshot(snap, at);
            this.spawnCastEffects(snap, terrain);
            this.lastSnap = snap;
            if (snap.over) this.showResult(snap.winner);
        });
        this.devConsole = new DevConsole({
            overlay: this.hud!,
            hordeTypes: (hordeTypes ?? []).map((t) => t.id),
            buildingTypes: btypes.map((t) => t.id),
            resources: (resourceInfos ?? []).filter((r) => r.kind === "stock").map((r) => r.id),
            send: (line) => {
                const at = this.cursorGround();
                this.connection?.console(line, at?.x ?? 0, at?.z ?? 0, [...(this.selection?.selected ?? [])]);
            },
        });
        const devConsole = this.devConsole;
        const hordeIds = (hordeTypes ?? []).map((t) => t.id);
        this.hudView = new Hud({
            overlay: this.hud!,
            economy,
            factions: factionList,
            tickMs: TICK_MS,
            typeOfKind: (kind) => (types[kind] ? { ...types[kind], id: hordeIds[kind] ?? "" } : undefined),
            typeOf: (id) => {
                const t = soldiers.typeOf(id);
                const kind = soldiers.hordes.get(id)?.kind;
                return t && kind !== undefined ? { ...t, id: hordeIds[kind] ?? "" } : undefined;
            },
            onSelect: (ids, focus) => {
                this.selection?.set(ids);
                const h = soldiers.hordes.get(ids[0]);
                if (focus && h) camera.focus(h.x, h.z);
            },
        });
        const navBlocked = navGrid
            ? { cols: navGrid.cols, rows: navGrid.rows, cellSize: navGrid.cellSize, blocked: decodeBase64(navGrid.blocked ?? "") }
            : null;
        this.minimap = new Minimap({
            parent: this.hudView.minimapFrame,
            mapWidth: data.width,
            mapHeight: data.height,
            heightAt: (x, z) => terrain.heightAt(x, z),
            maxHeight: data.maxHeight,
            nav: navBlocked,
            camera: () => camera.camera,
            onFocus: (x, z) => camera.focus(x, z),
            onCommand: (x, z) => {
                const ids = [...(this.selection?.selected ?? [])];
                if (ids.length === 0) return;
                this.connection?.move(x, z, ids);
                this.marker.show(x, terrain.heightAt(x, z), z);
            },
        });
        // Plot classes as the content defines them (class → side), smallest first.
        const plotClasses = [...new Map(btypes.flatMap((t) => t.plots.filter((p) => p.class).map((p) => [p.class, p.size] as const))).entries()].sort(
            (a, b) => a[1] - b[1],
        );
        this.buildPanel = new BuildPanel({
            plotClassFor: (w, d) => plotClasses.find(([, size]) => Math.max(w, d) <= size)?.[0] ?? "",
            overlay: this.hud!,
            buildings,
            economy,
            tech,
            onResearch: (b, kind) => this.connection?.research(b, kind),
            onGate: (b, open) => this.connection?.gate(b, open),
            onBuildingToggle: (b, ability) => this.connection?.buildingToggle(b, ability),
            abilities: () => this.abilityInfos,
            types: btypes,
            hordes: (hordeTypes ?? []).map((t) => ({
                id: t.id,
                name: t.name,
                cost: t.cost ?? {},
                uses: t.uses ?? {},
                buildTime: t.buildTime,
                requires: t.requires,
                unique: t.unique ?? false,
                reviveCost: t.reviveCost ?? t.cost ?? {},
                reviveTime: t.reviveTime ?? t.buildTime * 2,
            })),
            faction: () => {
                const f = factionList[this.localFaction];
                return f && { buildings: f.buildings ?? [], hordes: f.hordes ?? [] };
            },
            queue: (id) => queues.get(id),
            hordePos: (id) => soldiers.hordes.get(id),
            onTrain: (b, kind) => this.connection?.train(b, kind),
            onUntrain: (b, i) => this.connection?.untrain(b, i),
            onBuild: (host, plot, kind) => this.connection?.buildOnPlot(host, plot, kind),
            onCancel: (id) => this.connection?.cancelBuild(id),
            onPlace: (kind, workers) => {
                placement.start(kind, workers);
                buildPanel.setSuppressed(true);
            },
        });
        const buildPanel = this.buildPanel;
        const placement = new Placement({
            scene: this.scene,
            buildings,
            types: btypes,
            heightAt: (x, z) => terrain.heightAt(x, z),
            mapWidth: data.width,
            mapHeight: data.height,
            nav: navBlocked,
            localPlayer: () => this.localPlayer,
            onPlace: (kind, x, z, angle, workers) => this.connection?.build(kind, x, z, angle, workers),
        });
        this.placement = placement;
        const ownSelected = (): number[] => [...(this.selection?.selected ?? [])].filter((id) => soldiers.hordes.get(id)?.owner === this.localPlayer);
        this.card = new CommandCard({
            overlay: this.hud!,
            economy,
            tech,
            tickMs: TICK_MS,
            buildings: btypes,
            buildable: () => factionList[this.localFaction]?.buildings ?? [],
            typeOf: (id) => soldiers.typeOf(id),
            onTarget: (mode) => this.setTargetMode(mode),
            onStop: () => this.connection?.stop(ownSelected()),
            onHold: () => this.connection?.hold(ownSelected()),
            abilities: this.abilityInfos,
            slotsOf: (id) => this.lastSnap?.abilities.get(id),
            onCast: (kind, ids) => {
                const a = this.abilityInfos[kind];
                if (!a || ids.length === 0) return;
                if (a.target === "self") {
                    this.connection?.cast(kind, 0, 0, 0, ids);
                    return;
                }
                this.castKind = kind;
                this.castHordes = ids;
                this.setTargetMode("cast");
            },
            onToggle: (kind, ids, mode) => this.connection?.toggle(kind, ids, mode),
            onAutocast: (kind, on, ids) => this.connection?.autocast(kind, on, ids),
            onInspect: (kind, ids) => {
                const a = this.abilityInfos[kind];
                if (a?.kind !== "aura" || a.radius <= 0) return;
                for (const id of ids) this.preview.showAura(id, a.radius);
            },
            onPlace: (kind, workers) => {
                this.setTargetMode(null);
                placement.start(kind, workers);
            },
        });
        this.connection.onConsole = (out) => devConsole.print(out.text, out.error ? "error" : "out");
        void loadCursors().then(() => this.updateCursor(true));
        ctx.renderer.domElement.addEventListener("pointerleave", () => (this.mouseClient = null));
        ctx.renderer.domElement.addEventListener("pointermove", (e) => {
            this.mouseClient = { x: e.clientX, y: e.clientY };
            this.updateCursor();
            const r = ctx.renderer.domElement.getBoundingClientRect();
            this.mouseNdc.set(((e.clientX - r.left) / r.width) * 2 - 1, -((e.clientY - r.top) / r.height) * 2 + 1);
            if (this.placement?.placing) {
                const at = this.cursorGround();
                if (at) this.placement.update(at.x, at.z);
            }
        });

        this.selection = new SelectionController({
            dom: ctx.renderer.domElement,
            overlay: this.hud!,
            camera: () => camera.camera,
            soldiers,
            terrain: terrain.mesh,
            localPlayer: () => this.localPlayer,
            onMove: (x, y, z, ids) => {
                this.connection?.move(x, z, ids);
                this.marker.show(x, y, z);
            },
            onAttack: (target, at, ids) => {
                this.connection?.attack(target, ids);
                this.marker.show(at.x, at.y, at.z, 0xff5040);
            },
            pickBuilding: (ray, ground) => buildings.pick(ray, ground),
            onSiege: (target, at, ids) => {
                this.connection?.siege(target, ids);
                this.marker.show(at.x, terrain.heightAt(at.x, at.z), at.z, 0xff5040);
            },
            onChange: (sel) => {
                const workers = [...sel].filter((id) => soldiers.typeOf(id)?.worker);
                if (sel.size > 0) buildPanel.clear();
                if (workers.length === 0) placement.cancel();
                if (sel.size === 0) this.setTargetMode(null);
                this.renderSelectionPanel();
            },
            intercept: (e) => {
                if (placement.placing) {
                    if (e.button === 0) placement.place(e.shiftKey);
                    else if (e.button === 2) placement.cancel();
                    return true;
                }
                if (this.targetMode) {
                    if (e.button === 0) this.issueTargeted(e, ownSelected(), terrain);
                    if (!e.shiftKey || e.button !== 0) this.setTargetMode(null);
                    return true;
                }
                return false;
            },
            onOwnBuilding: (id, ids, point) => {
                const b = buildings.info(id);
                if (b?.type?.kind === "wall" || b?.type?.kind === "gate") {
                    // Onto the wall (the server sends foot soldiers up the stairs; others walk to the spot below).
                    this.connection?.move(point.x, point.z, ids);
                    this.marker.show(point.x, point.y, point.z);
                    return true;
                }
                const workers = ids.filter((h) => soldiers.typeOf(h)?.worker);
                // Watchtower: foot soldiers go up (the server picks who can).
                if (b?.type?.garrison && !isConstructing(b.snap.flags) && workers.length < ids.length) {
                    const up = ids.filter((h) => !soldiers.typeOf(h)?.worker);
                    this.connection?.move(b.snap.x, b.snap.z, up);
                    this.marker.show(b.snap.x, point.y, b.snap.z);
                    return true;
                }
                // Workers help build a foundation, or repair a damaged building.
                if (!b || workers.length === 0 || !(isConstructing(b.snap.flags) || b.snap.hp < b.snap.maxHp)) return false;
                this.connection?.helpBuild(id, workers);
                this.marker.show(b.snap.x, terrain.heightAt(b.snap.x, b.snap.z), b.snap.z, 0xffd76a);
                return true;
            },
            onRightClickIdle: (ground) => {
                const id = buildPanel.building;
                const b = id ? buildings.info(id) : undefined;
                if (!b || !b.type?.trains.length || isConstructing(b.snap.flags)) return;
                this.connection?.rally(id, ground.x, ground.z);
                this.marker.show(ground.x, ground.y, ground.z, 0x7dc8ff);
            },
            onClickWorld: (ray, ground) => {
                const plot = buildings.pickPlot(ground, this.localPlayer, ray);
                if (plot) return buildPanel.showPlot(plot);
                const b = buildings.pick(ray, ground);
                if (b && b.owner === this.localPlayer) return buildPanel.showBuilding(b.id);
                buildPanel.clear();
            },
        });
        // Alive counts change over time (combat later): refresh the panel a few times per second.
        this.panelTimer = window.setInterval(() => {
            this.renderSelectionPanel();
            buildPanel.setSuppressed(placement.placing);
            buildPanel.refresh();
            this.minimap?.draw(soldiers.hordes.values(), this.lastSnap?.buildings ?? [], this.selection?.selected ?? new Set());
        }, 150);

        const first = data.starts?.[this.localPlayer];
        if (first) camera.focus(first.x, first.y);

        this.setStatus("");
        this.showHint();
        this.clock.start();
        this.loop();
    }

    private buildScene(terrain: Terrain, soldiers: SoldiersView, buildings: BuildingsView): void {
        const sky = new THREE.Color(0x9fb4c8);
        this.scene.background = sky;
        this.scene.fog = new THREE.Fog(sky, 350, 900);

        this.scene.add(new THREE.HemisphereLight(0xdfe8ff, 0x4a3e2a, 1.1));
        const sun = new THREE.DirectionalLight(0xfff1dc, 1.6);
        sun.position.set(-150, 260, 120);
        this.scene.add(sun);

        this.scene.add(terrain.mesh);
        for (const o of soldiers.objects) this.scene.add(o);
        if (soldiers.machines) this.scene.add(soldiers.machines.group);
        this.scene.add(this.marker.mesh);
        this.scene.add(this.projectiles.points, this.projectiles.stones);
        this.projectiles.onImpact = (x, z) => this.effects.cast(x, terrain.heightAt(x, z), z, 4, 0x9b8a6a, 0);
        this.scene.add(this.effects.group);
        this.scene.add(this.statusFx.points);
        this.scene.add(this.preview.group);
        this.scene.add(buildings.group);
    }

    private loop = (): void => {
        this.frameId = requestAnimationFrame(this.loop);
        const dt = this.clock.getDelta();
        const now = performance.now();
        this.frames++;
        const terrain = this.terrain!;

        this.rtsCamera!.update(dt);
        this.soldiers!.update(now, (x, z) => terrain.heightAt(x, z), this.selection!.selected, this.rtsCamera!.camera);
        this.projectiles.update(now, (x, z) => terrain.heightAt(x, z));
        this.marker.update(dt);
        this.buildings?.animate(dt);
        this.effects.update(dt);
        this.updatePreview(dt, terrain);
        this.statusFx.update(now, this.soldiers!, this.lastSnap?.buildings ?? [], (x, z) => terrain.heightAt(x, z), this.rtsCamera!.camera);
        this.updateCursor(); // targets move under a still mouse too
        this.ctx!.renderer.render(this.scene, this.rtsCamera!.camera);

        const canvas = this.ctx!.renderer.domElement;
        this.labels?.update(this.rtsCamera!.camera, this.soldiers!, canvas.clientWidth, canvas.clientHeight);
        this.buildings?.update(this.rtsCamera!.camera, canvas.clientWidth, canvas.clientHeight);
        this.positionRadial(canvas.clientWidth, canvas.clientHeight);
    };

    private onKeyDown = (e: KeyboardEvent): void => {
        if (this.devConsole?.open) return;
        if (e.code === "KeyR" && this.placement?.placing) {
            this.placement.rotate(e.shiftKey ? -1 : 1);
            return;
        }
        if (!e.repeat && !e.metaKey && !e.ctrlKey && !this.placement?.placing && this.card?.hotkey(e.code)) {
            e.preventDefault();
            return;
        }
        if (e.code === "F4") {
            e.preventDefault();
            this.toggleDebug();
            return;
        }
        if (e.code === "F3") {
            e.preventDefault();
            void this.toggleNavOverlay();
        }
    };

    /** F3: show blocked (unwalkable) ground in red. */
    private async toggleNavOverlay(): Promise<void> {
        if (!this.terrain) return;
        this.navOverlay = !this.navOverlay;
        if (!this.navOverlay) {
            this.terrain.setNavOverlay(null);
            return;
        }
        const nav = await ContentService.NavGrid(this.mapId);
        if (!nav || !this.navOverlay || this.stopped) return;
        this.terrain.setNavOverlay({
            cols: nav.cols,
            rows: nav.rows,
            cellSize: nav.cellSize,
            blocked: decodeBase64(nav.blocked ?? ""),
        });
    }

    /** F4: FPS, tick, simulation time and state hash. */
    private toggleDebug(): void {
        const panel = this.debugPanel;
        if (!panel) return;
        panel.hidden = !panel.hidden;
        clearInterval(this.debugTimer);
        if (panel.hidden) return;

        let last = performance.now();
        this.frames = 0;
        const render = () => {
            const now = performance.now();
            const fps = (this.frames * 1000) / (now - last);
            this.frames = 0;
            last = now;
            const s = this.lastSnap;
            panel.textContent = [
                `fps        ${fps.toFixed(0)}`,
                `tick       ${s?.tick ?? "-"}`,
                `sim        ${s ? (s.simMicros / 1000).toFixed(2) : "-"} ms`,
                `snapshot   ${s ? (s.byteLength / 1024).toFixed(1) : "-"} KB`,
                `soldiers   ${s?.soldierCount ?? "-"}`,
                `volleys    ${s?.projectiles.length ?? "-"}`,
                ...this.lodLines(),
                `hash @${s?.hashTick ?? "-"}`,
                `  ${s?.hash ?? "-"}`,
            ].join("\n");
        };
        render();
        this.debugTimer = window.setInterval(render, 500);
    }

    private lodLines(): string[] {
        const st = this.soldiers?.lodStats();
        if (!st) return [];
        return [`models     ${st.visible} drawn, ${st.culled} culled`, `  lod      ${st.perLod.join(" / ")}`];
    }

    /** Casts the ability of every given horde that has one ready. */
    /** The client now controls another player (dev console "control"). */
    private switchPlayer(player: number, snap: GameSnapshot): void {
        this.localPlayer = player;
        this.selection?.clear();
        this.buildPanel?.clear();
        this.placement?.cancel();
        const fortress = snap.buildings.find((b) => b.owner === player && this.buildings?.info(b.id)?.type?.kind === "fortress");
        if (fortress) this.rtsCamera?.focus(fortress.x, fortress.z);
    }

    /** Ability previews: area + range while aiming, aura circles after clicking an aura. */
    private updatePreview(dt: number, terrain: Terrain): void {
        const hordes = this.soldiers?.hordes;
        const posOf = (id: number) => {
            const h = hordes?.get(id);
            return h ? { x: h.x, y: terrain.heightAt(h.x, h.z), z: h.z } : null;
        };
        this.preview.update(dt, posOf);
        const a = this.targetMode === "cast" ? this.abilityInfos[this.castKind] : undefined;
        if (!a) {
            this.preview.stopAiming();
            return;
        }
        const caster = this.castHordes.map(posOf).find((p) => p) ?? null;
        const color = DAMAGE_COLORS[a.damageType] ?? (a.affects === "allies" ? 0x7dff7a : 0xc89bff);
        if (a.cone > 0) this.preview.aimCone(this.cursorGround(), caster, a.radius, a.cone, color);
        else this.preview.aim(this.cursorGround(), a.radius, color, caster, a.range);
    }

    /**
     * Picks the canvas cursor for what a click would do here (docs/UX.md).
     * A display hint only: the server checks every order itself.
     */
    private updateCursor(force = false): void {
        const now = performance.now();
        if (!force && now - this.cursorAt < 100) return; // 10 Hz: the probe projects every soldier
        this.cursorAt = now;
        const canvas = this.ctx?.renderer.domElement;
        if (!canvas) return;
        setCursor(canvas, this.pickCursor(canvas));
    }

    private pickCursor(canvas: HTMLCanvasElement): CursorName {
        const m = this.mouseClient;
        const sel = this.selection;
        const soldiers = this.soldiers;
        const buildings = this.buildings;
        if (!m || !sel || !soldiers || !buildings) return "default";
        if (this.placement?.placing) return "place";

        // Screen edges scroll the camera (same margin as RTSCamera).
        const r = canvas.getBoundingClientRect();
        const EDGE = 8;
        const n = m.y - r.top <= EDGE;
        const s = r.bottom - m.y <= EDGE;
        const w = m.x - r.left <= EDGE;
        const e = r.right - m.x <= EDGE;
        if (n || s || w || e) {
            return (n ? (w ? "scrollNW" : e ? "scrollNE" : "scrollN") : s ? (w ? "scrollSW" : e ? "scrollSE" : "scrollS") : w ? "scrollW" : "scrollE") as CursorName;
        }

        const t = sel.pickTarget({ clientX: m.x, clientY: m.y });
        const own = [...sel.selected].filter((id) => soldiers.hordes.get(id)?.owner === this.localPlayer);
        const types = own.map((id) => soldiers.typeOf(id)).filter((x): x is NonNullable<typeof x> => !!x);
        // Can any selected horde hit the enemy soldier under the mouse (flyers, units on walls)?
        const canHit = (): boolean => {
            const i = t.enemyIndex;
            if (i < 0) return true;
            const enemyType = soldiers.typeOf(t.enemyHorde);
            const status = soldiers.status[i] ?? 0;
            if (enemyType?.flying && !(status & STATUS_GROUNDED)) return types.some((x) => x.antiAir);
            if (status & STATUS_ON_WALL) return types.some((x) => x.ranged || x.flying);
            return true;
        };
        const b = t.building ? buildings.info(t.building.id) : undefined;

        switch (this.targetMode) {
            case "move":
                return "targetMove";
            case "patrol":
                return "patrol";
            case "fireAt":
                return "fireAt";
            case "attack":
                if (t.enemyHorde) return canHit() ? "targetAttack" : "no";
                if (b && t.building!.owner !== this.localPlayer && t.building!.owner !== NEUTRAL) return "targetAttack";
                return "attackMove";
            case "cast": {
                const a = this.abilityInfos[this.castKind];
                const allies = a?.affects === "allies";
                if (a?.target === "unit") return allies ? (t.allyHorde ? "castAlly" : "no") : t.enemyHorde ? "cast" : "no";
                return allies ? "castAlly" : "cast";
            }
        }

        if (own.length === 0) {
            const panelBuilding = this.buildPanel?.building ?? 0;
            const pb = panelBuilding ? buildings.info(panelBuilding) : undefined;
            if (buildings.pickPlot(t.ground, this.localPlayer, this.selectionRay(m))) return "plot";
            if (pb && pb.type?.trains.length && !isConstructing(pb.snap.flags) && !t.building && !t.allyHorde && !t.enemyHorde) return "rally";
            if (t.allyHorde || (t.building && t.building.owner === this.localPlayer)) return "select";
            if (t.enemyHorde || t.building) return "inspect";
            return "default";
        }

        if (t.enemyHorde) return canHit() ? "attack" : "no";
        if (b && t.building) {
            const owner = t.building.owner;
            if (owner === NEUTRAL) return b.type?.kind === "outpost" ? "capture" : "move";
            if (owner !== this.localPlayer) return "attack";
            const workers = types.some((x) => x.worker);
            if (workers && isConstructing(b.snap.flags)) return "build";
            if (workers && b.snap.hp < b.snap.maxHp) return "repair";
            const climbable = b.type?.kind === "wall" || b.type?.garrison;
            if (climbable && !isConstructing(b.snap.flags) && types.some((x) => x.climbs && !x.worker)) return "climb";
            return "move";
        }
        return "move";
    }

    /** A ray under client point m (for plots on the wall walkway). */
    private selectionRay(m: { x: number; y: number }): THREE.Raycaster | undefined {
        const canvas = this.ctx?.renderer.domElement;
        const cam = this.rtsCamera?.camera;
        if (!canvas || !cam) return undefined;
        const r = canvas.getBoundingClientRect();
        this.cursorRay.setFromCamera(new THREE.Vector2(((m.x - r.left) / r.width) * 2 - 1, -((m.y - r.top) / r.height) * 2 + 1), cam);
        return this.cursorRay;
    }

    /** Enters / leaves "click a target" mode for a command card order. */
    private setTargetMode(mode: TargetMode | null): void {
        this.targetMode = mode;
        this.updateCursor(true);
        this.card?.setMode(mode, this.castKind);
        this.ctx?.renderer.domElement.classList.toggle("is-targeting", mode !== null);
        this.renderSelectionPanel();
    }

    /** Resolves a targeted order at the clicked point. */
    private issueTargeted(e: MouseEvent, ids: number[], terrain: Terrain): void {
        const sel = this.selection;
        if (!sel || ids.length === 0) return;
        const t = sel.pickTarget(e);
        const g = t.ground;
        switch (this.targetMode) {
            case "move":
                if (g) {
                    this.connection?.move(g.x, g.z, ids);
                    this.marker.show(g.x, g.y, g.z);
                }
                break;
            case "attack":
                if (t.enemyHorde && t.enemyAt) {
                    this.connection?.attack(t.enemyHorde, ids);
                    this.marker.show(t.enemyAt.x, t.enemyAt.y, t.enemyAt.z, 0xff5040);
                } else if (t.building && t.building.owner !== this.localPlayer && t.building.owner !== NEUTRAL) {
                    this.connection?.siege(t.building.id, ids);
                    const p = t.building.point;
                    this.marker.show(p.x, terrain.heightAt(p.x, p.z), p.z, 0xff5040);
                } else if (g) {
                    this.connection?.attackMove(g.x, g.z, ids);
                    this.marker.show(g.x, g.y, g.z, 0xffa040);
                }
                break;
            case "patrol":
                if (g) {
                    this.connection?.patrol(g.x, g.z, ids);
                    this.marker.show(g.x, g.y, g.z, 0x7dc8ff);
                }
                break;
            case "fireAt": {
                const shooters = ids.filter((id) => this.soldiers?.typeOf(id)?.ranged);
                if (g && shooters.length > 0) {
                    this.connection?.attackGround(g.x, g.z, shooters);
                    this.marker.show(g.x, g.y, g.z, 0xff7a2a);
                }
                break;
            }
            case "cast": {
                const a = this.abilityInfos[this.castKind];
                const casters = this.castHordes.filter((id) => this.soldiers?.hordes.has(id));
                if (!a || casters.length === 0) break;
                if (a.target === "unit") {
                    const target = a.affects === "allies" ? t.allyHorde : t.enemyHorde;
                    const h = target ? this.soldiers?.hordes.get(target) : undefined;
                    if (!h) break;
                    this.connection?.cast(this.castKind, h.x, h.z, target, casters);
                    this.marker.show(h.x, terrain.heightAt(h.x, h.z), h.z, 0xc89bff);
                } else if (g) {
                    this.connection?.cast(this.castKind, g.x, g.z, 0, casters);
                    this.marker.show(g.x, g.y, g.z, 0xc89bff);
                }
                break;
            }
        }
    }

    /** Keeps the radial command ring on its plot / building / workers. */
    private positionRadial(width: number, height: number): void {
        const panel = this.buildPanel;
        const a = panel?.anchor();
        if (!panel || !a || !this.terrain || !this.rtsCamera) return;
        const p = this.anchorTmp.set(a.x, this.terrain.heightAt(a.x, a.z) + a.lift, a.z).project(this.rtsCamera.camera);
        const visible = p.z < 1 && Math.abs(p.x) < 1.05 && Math.abs(p.y) < 1.05;
        panel.position((p.x + 1) * 0.5 * width, (1 - p.y) * 0.5 * height, visible);
    }

    /** Terrain point under the last known mouse position. */
    private cursorGround(): THREE.Vector3 | null {
        if (!this.rtsCamera || !this.terrain) return null;
        this.raycaster.setFromCamera(this.mouseNdc, this.rtsCamera.camera);
        return this.raycaster.intersectObject(this.terrain.mesh, false)[0]?.point ?? null;
    }

    /** Visuals for this tick's ability casts (rings colored by damage type; zones linger). */
    private spawnCastEffects(snap: GameSnapshot, terrain: Terrain): void {
        for (const e of snap.events) {
            const a = this.abilityInfos[e.ability];
            const color = DAMAGE_COLORS[e.damageType || (a?.damageType ?? "")] ?? 0xffc860;
            switch (e.kind) {
                case EVENT_CHAIN:
                    this.effects.bolt(e.x, terrain.heightAt(e.x, e.z), e.z, e.radius, terrain.heightAt(e.radius, e.duration), e.duration, color);
                    break;
                case EVENT_CONE:
                    this.effects.cone(e.x, terrain.heightAt(e.x, e.z), e.z, e.radius, e.duration, e.facing, color);
                    break;
                default:
                    this.effects.cast(e.x, terrain.heightAt(e.x, e.z), e.z, e.radius, color, e.duration);
            }
        }
    }


    private renderSelectionPanel(): void {
        if (!this.selection || !this.soldiers || !this.hudView) return;
        // Drop hordes that were wiped out.
        for (const id of this.selection.selected) {
            if (!this.soldiers.hordes.has(id)) this.selection.selected.delete(id);
        }
        const ids = [...this.selection.selected].sort((a, b) => a - b);
        this.hudView.render(ids, this.soldiers.hordes, this.localPlayer, this.lastHeroes);
        this.card?.render(ids, this.soldiers.hordes, this.localPlayer);
    }

    /** End-of-match banner (shown once). */
    private showResult(winner: number): void {
        if (!this.hud || this.hud.querySelector(".game-hud__result")) return;
        const el = document.createElement("div");
        el.className = "game-hud__result";
        const [title, kind] =
            winner === this.localPlayer ? ["Victory", "is-win"] : winner === NEUTRAL ? ["Draw", "is-draw"] : ["Defeat", "is-loss"];
        el.classList.add(kind);
        el.innerHTML = `<div class="game-hud__result-title">${title}</div><div class="game-hud__result-hint">Esc — back to menu</div>`;
        this.hud.appendChild(el);
    }

    private setStatus(text: string): void {
        const el = this.hud?.querySelector<HTMLElement>(".game-hud__status");
        if (el) {
            el.textContent = text;
            el.hidden = text === "";
        }
    }

    private showHint(): void {
        const hint = document.createElement("div");
        hint.className = "game-hud__hint";
        hint.textContent =
            "LMB select / drag box · Shift add · RMB move / attack · click plots and buildings for the command ring · minimap: LMB move camera, RMB send selection · 1 hero ability · WASD / edges pan · wheel zoom · Q/E rotate · F2 console · F3 walkability · F4 debug · Esc back";
        this.hud?.appendChild(hint);
        window.setTimeout(() => hint.classList.add("is-faded"), 12_000);
    }
}

/** Loads every baked VAT model referenced by a horde type. A model that fails
 *  to load falls back to the placeholder (logged, not fatal). */
async function loadModels(types: HordeTypeInfo[]): Promise<Map<string, VatModel>> {
    const ids = [...new Set(types.map((t) => t.model).filter((m) => m))];
    const models = new Map<string, VatModel>();
    await Promise.all(
        ids.map(async (id) => {
            try {
                models.set(id, await VatModel.load(id));
            } catch (err) {
                console.warn(`model ${id} not loaded, using placeholder:`, err);
            }
        }),
    );
    return models;
}
