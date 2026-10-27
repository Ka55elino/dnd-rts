import { QUEUE_RESEARCH, buildProgress, isConstructing, isGateOpen, isToggled, plotOccupied, type QueueSnap } from "../../net/protocol";
import type { Requires, Tech } from "./tech";
import type { BuildingTypeInfo, BuildingsView, PlotPick } from "../view/BuildingsView";
import type { Amounts, Economy } from "./economy";
import { icon, iconFor } from "./icons";

/** What the panel needs to know about a horde type (index = snapshot kind). */
export interface TrainableInfo {
    id: string;
    name: string;
    cost: Amounts;
    uses: Amounts;
    buildTime: number;
    requires: Requires;
    /** One per player; a fallen one is revived for reviveCost in reviveTime. */
    unique: boolean;
    reviveCost: Amounts;
    reviveTime: number;
}

export interface BuildPanelDeps {
    overlay: HTMLElement;
    buildings: BuildingsView;
    economy: Economy;
    tech: Tech;
    onResearch: (building: number, upgradeKind: number) => void;
    /** Open (true) / close a gate. */
    onGate: (building: number, open: boolean) => void;
    /** Switch a tower's toggle ability (index into the ability list). */
    onBuildingToggle: (building: number, ability: number) => void;
    /** Ability infos (index = kind), for tower toggles. */
    abilities: () => { id: string; name: string; requires: Requires }[];
    /** Smallest plot class (s / m / l) a footprint fits on, "" if none. */
    plotClassFor: (width: number, depth: number) => string;
    /** All building types (index = snapshot kind). */
    types: BuildingTypeInfo[];
    /** All horde types (index = snapshot kind). */
    hordes: TrainableInfo[];
    /** Building / horde ids the local player's faction may use (changes with the dev console). */
    faction: () => { buildings: string[]; hordes: string[] } | undefined;
    /** Latest training queue of a building. */
    queue: (building: number) => QueueSnap | undefined;
    /** World position of a horde (worker menus follow the workers). */
    hordePos: (hordeId: number) => { x: number; z: number } | undefined;
    onBuild: (host: number, plot: number, kind: number) => void;
    onCancel: (building: number) => void;
    /** Workers: start placing a free building of this kind. */
    onPlace: (kind: number, workers: number[]) => void;
    onTrain: (building: number, hordeKind: number) => void;
    onUntrain: (building: number, index: number) => void;
}

type Target = { plot: PlotPick } | { building: number } | { workers: number[] } | null;

/** One button on the ring. */
interface Item {
    action: string; // data attribute: kind | place | train | cancel | research | gate
    value: number;
    glyph: string; // icon markup
    title: string;
    cost?: Amounts;
    uses?: Amounts;
    meta?: string;
    enabled: boolean;
}

const MAX_QUEUE = 5; // internal/sim maxQueue
const RADIUS = 84; // px from the ring centre to the button centres

/**
 * Radial command ring (BFME style) drawn around what it acts on:
 * - a free plot: what to build there (BFME1 style, builds itself);
 * - selected workers: free buildings to place;
 * - an own building: train buttons (queue shown beneath; right click on the
 *   ground sets the rally point), or progress and cancel while it is built.
 * Hovering a button shows a parchment tooltip with name, price (shortfalls in
 * red) and time.
 */
export class BuildPanel {
    private readonly el: HTMLDivElement;
    private target: Target = null;
    private lastHtml = "";
    /** Changing numbers of the open ring, applied without rebuilding the markup. */
    private live = { queue: 0, build: 0, hp: "" };
    private items: Item[] = [];
    private hover = -1;

    constructor(private readonly d: BuildPanelDeps) {
        this.el = document.createElement("div");
        this.el.className = "radial";
        this.el.hidden = true;
        this.el.addEventListener("mousedown", (e) => e.stopPropagation());
        this.el.addEventListener("click", this.onClick);
        this.el.addEventListener("mouseover", (e) => {
            const i = (e.target as HTMLElement).closest<HTMLElement>("[data-item]")?.dataset.item;
            this.setHover(i === undefined ? -1 : Number(i));
        });
        this.el.addEventListener("mouseleave", () => this.setHover(-1));
        d.overlay.appendChild(this.el);
    }

    get active(): boolean {
        return this.target !== null;
    }

    /** The own building shown, or 0. */
    get building(): number {
        return this.target && "building" in this.target ? this.target.building : 0;
    }

    get showingWorkers(): boolean {
        return this.target !== null && "workers" in this.target;
    }

    showPlot(plot: PlotPick): void {
        this.target = { plot };
        this.d.buildings.select(plot, 0);
        this.refresh();
    }

    /** Workers selected: list free buildings. */
    showWorkers(workers: number[]): void {
        this.target = { workers };
        this.d.buildings.select(null, 0);
        this.refresh();
    }

    showBuilding(id: number): void {
        this.target = { building: id };
        this.d.buildings.select(null, id);
        this.refresh();
    }

    clear(): void {
        if (!this.target) return;
        this.target = null;
        this.d.buildings.select(null, 0);
        this.refresh();
    }

    /** Hide while placing a building (the ring would cover the ghost). */
    setSuppressed(on: boolean): void {
        this.el.classList.toggle("is-suppressed", on);
    }

    /** World point the ring is centred on (y = 0; the caller adds terrain height). */
    anchor(): { x: number; z: number; lift: number } | null {
        const t = this.target;
        if (!t) return null;
        if ("plot" in t) {
            const p = this.d.buildings.plotWorld(t.plot.host, t.plot.plot);
            return p && { ...p, lift: 0 };
        }
        if ("building" in t) {
            const b = this.d.buildings.info(t.building);
            return b ? { x: b.snap.x, z: b.snap.z, lift: 4 } : null;
        }
        let x = 0, z = 0, n = 0;
        for (const id of t.workers) {
            const p = this.d.hordePos(id);
            if (p) {
                x += p.x;
                z += p.z;
                n++;
            }
        }
        return n ? { x: x / n, z: z / n, lift: 1 } : null;
    }

    /** Screen position of the anchor (from the game loop). */
    position(x: number, y: number, visible: boolean): void {
        this.el.style.transform = `translate(${x.toFixed(1)}px, ${y.toFixed(1)}px)`;
        this.el.classList.toggle("is-offscreen", !visible);
    }

    /** Re-renders from the latest snapshot; closes when the target is gone. */
    refresh(): void {
        const html = this.render();
        if (html === null) {
            if (this.target) this.clear();
            this.el.hidden = true;
            this.lastHtml = "";
            this.items = [];
            return;
        }
        this.el.hidden = false;
        // The markup carries no progress numbers, so it only changes when the
        // options do; rebuilding it on every snapshot made the icons flicker.
        if (html !== this.lastHtml) {
            this.el.innerHTML = html;
            this.lastHtml = html;
            this.renderTip();
        }
        this.applyLive();
    }

    /** Writes the changing numbers (queue / construction progress) into the existing nodes. */
    private applyLive(): void {
        const front = this.el.querySelector<HTMLElement>('.radial__queued[data-untrain="0"]');
        if (front) {
            const p = this.live.queue.toFixed(3);
            if (front.style.getPropertyValue("--p") !== p) front.style.setProperty("--p", p);
        }
        const pct = this.el.querySelector<HTMLElement>(".radial__pct");
        if (pct) {
            const text = `${this.live.build}%`;
            if (pct.textContent !== text) pct.textContent = text;
            const bar = this.el.querySelector<HTMLElement>(".radial__bar > span");
            if (bar) bar.style.width = text;
        }
        const hp = this.el.querySelector<HTMLElement>(".radial__hp");
        if (hp && hp.textContent !== this.live.hp) hp.textContent = this.live.hp;
    }

    dispose(): void {
        this.el.remove();
    }

    private render(): string | null {
        const t = this.target;
        if (!t) return null;
        const faction = this.d.faction();
        const eco = this.d.economy;
        const items: Item[] = [];
        let title = "";
        let below = "";

        if ("workers" in t) {
            title = "Build";
            this.d.types.forEach((type, kind) => {
                if (type.kind !== "building" || !type.placement.includes("free") || !faction?.buildings.includes(type.id)) return;
                items.push({
                    action: "place",
                    value: kind,
                    glyph: iconFor(type.id, type.name),
                    title: type.name,
                    cost: type.cost,
                    meta: this.withReq(`${type.width}×${type.depth} · ${type.buildTime.toFixed(0)}s · built by workers`, type.requires),
                    enabled: eco.canAfford(type.cost) && this.d.tech.missing(type.requires).length === 0,
                });
            });
        } else if ("plot" in t) {
            const host = this.d.buildings.info(t.plot.host);
            if (!host || plotOccupied(host.snap, t.plot.plot)) return null;
            const tower = t.plot.class === "tower";
            const cls = t.plot.class.toUpperCase();
            title = tower ? "Tower socket" : cls ? `Plot ${cls}` : "Plot";
            this.d.types.forEach((type, kind) => {
                // Tower sockets on the wall take towers; ground plots the plot buildings.
                if (type.kind !== "building" || !type.placement.includes(tower ? "tower" : "plot") || !faction?.buildings.includes(type.id)) return;
                // Too big for this plot: shown greyed so the player knows a larger plot is needed.
                const fits = type.width <= t.plot.size && type.depth <= t.plot.size;
                const need = tower ? "" : this.d.plotClassFor(type.width, type.depth);
                items.push({
                    action: "kind",
                    value: kind,
                    glyph: iconFor(type.id, type.name),
                    title: type.name,
                    cost: type.cost,
                    meta: this.withReq(`${type.buildTime.toFixed(0)}s${need ? ` · plot ${need.toUpperCase()}` : ""}`, type.requires) +
                        (fits ? "" : `<br><span class="radial__tip-req">Needs a larger plot</span>`),
                    enabled: fits && eco.canAfford(type.cost) && this.d.tech.missing(type.requires).length === 0,
                });
            });
        } else {
            const b = this.d.buildings.info(t.building);
            if (!b) return null;
            title = b.type?.name ?? "Building";
            if (isConstructing(b.snap.flags)) {
                this.live.build = Math.round(buildProgress(b.snap.flags) * 100);
                const waiting = b.type?.placement.includes("free") && !b.type.placement.includes("plot") ? " · needs workers" : "";
                items.push({ action: "cancel", value: b.snap.id, glyph: icon("cancel"), title: "Cancel construction", meta: "full refund", enabled: true });
                below = `<div class="radial__status">Under construction <span class="radial__pct"></span>${waiting}
                    <div class="radial__bar"><span></span></div></div>`;
            } else if (b.type?.kind === "gate") {
                // Gatehouse: one switch. Open lets everyone through (enemies too); closing waits for an empty passage.
                const open = isGateOpen(b.snap.flags);
                items.push({
                    action: "gate",
                    value: open ? 0 : 1,
                    glyph: icon(open ? "gateClosed" : "gateOpen"),
                    title: open ? "Close the gate" : "Open the gate",
                    meta: open ? "Open: anyone can pass, enemies too.<br>Closes once nobody stands in the passage." : "Closed: nobody passes.",
                    enabled: true,
                });
                this.live.hp = `${b.snap.hp}/${b.snap.maxHp}`;
                below = `<div class="radial__status">${open ? "Gate open" : "Gate closed"} · <span class="radial__hp"></span></div>`;
            } else {
                // Tower toggles (fire arrows): need their upgrade to turn on.
                for (const id of b.type?.toggles ?? []) {
                    const kind = this.d.abilities().findIndex((a) => a.id === id);
                    const a = this.d.abilities()[kind];
                    if (!a) continue;
                    const on = isToggled(b.snap.flags);
                    items.push({
                        action: "toggle",
                        value: kind,
                        glyph: icon("fire"),
                        title: `${a.name}: ${on ? "on" : "off"}`,
                        meta: this.withReq(on ? "Click to switch off (pierce ignores armor again)" : "Click to switch on: + fire damage, no armor pierce", a.requires),
                        enabled: on || this.d.tech.missing(a.requires).length === 0,
                    });
                }
                const queue = this.d.queue(b.snap.id);
                const full = (queue?.hordes.length ?? 0) >= MAX_QUEUE;
                for (const id of b.type?.trains ?? []) {
                    const kind = this.d.hordes.findIndex((h) => h.id === id);
                    if (kind < 0 || !faction?.hordes.includes(id)) continue;
                    const h = this.d.hordes[kind];
                    // Unique heroes: one at a time; a fallen one is revived at its level.
                    const rec = h.unique ? this.d.tech.hero(kind) : undefined;
                    const busy = h.unique && (rec?.alive || this.d.tech.queued(kind));
                    const revive = !!rec && !rec.alive;
                    const cost = revive ? h.reviveCost : h.cost;
                    const time = revive ? h.reviveTime : h.buildTime;
                    let note = full ? " · queue full" : "";
                    if (busy) note = rec?.alive ? " · on the field" : " · in training";
                    items.push({
                        action: "train",
                        value: kind,
                        glyph: iconFor(h.id, h.name),
                        title: revive ? `Revive ${h.name} (level ${rec!.level})` : h.name,
                        cost,
                        uses: h.uses,
                        meta: this.withReq(`${time.toFixed(0)}s${note}`, h.requires),
                        enabled: !full && !busy && eco.canAfford(cost) && eco.hasRoom(h.uses) && this.d.tech.missing(h.requires).length === 0,
                    });
                }
                // Research: upgrades not owned yet; ones already queued are disabled.
                const queued = new Set((queue?.hordes ?? []).filter((k) => k & QUEUE_RESEARCH).map((k) => k & ~QUEUE_RESEARCH));
                for (const id of b.type?.research ?? []) {
                    const kind = this.d.tech.upgrades.findIndex((u) => u.id === id);
                    const g = this.d.tech.upgrades[kind];
                    if (!g || this.d.tech.has(id)) continue;
                    const inQueue = queued.has(kind);
                    items.push({
                        action: "research",
                        value: kind,
                        glyph: icon("build"),
                        title: g.name,
                        cost: g.cost ?? {},
                        meta: this.withReq(`${g.description}<br>${g.researchTime.toFixed(0)}s${inQueue ? " · researching" : ""}`, g.requires),
                        enabled: !full && !inQueue && eco.canAfford(g.cost) && this.d.tech.missing(g.requires).length === 0,
                    });
                }
                if (queue && queue.hordes.length > 0) {
                    this.live.queue = queue.progress;
                    below = `<div class="radial__queue">${queue.hordes
                        .map((k, i) => {
                            const pct = 0; // the front entry's progress is set live (applyLive)
                            if (k & QUEUE_RESEARCH) {
                                const g = this.d.tech.upgrades[k & ~QUEUE_RESEARCH];
                                return `<button class="radial__queued" data-untrain="${i}" style="--p:${pct.toFixed(3)}" title="${escapeHtml(g?.name ?? "?")} — click to cancel">
                                    ${icon("build")}</button>`;
                            }
                            const h = this.d.hordes[k];
                            return `<button class="radial__queued" data-untrain="${i}" style="--p:${pct.toFixed(3)}" title="${escapeHtml(h?.name ?? "?")} — click to cancel">
                                ${iconFor(h?.id ?? "", h?.name ?? "?")}</button>`;
                        })
                        .join("")}</div>`;
                }
                if (items.length === 0 && !below) {
                    this.live.hp = `${b.snap.hp}/${b.snap.maxHp}`;
                    below = `<div class="radial__status radial__hp"></div>`;
                }
            }
        }

        this.items = items;
        if (items.length === 0 && !below) {
            below = `<div class="radial__status">Nothing available</div>`;
        }
        // Buttons around the ring, starting at the top, clockwise.
        const n = Math.max(items.length, 1);
        const ring = items
            .map((it, i) => {
                const a = -Math.PI / 2 + (i * 2 * Math.PI) / Math.max(n, 6);
                const x = Math.cos(a) * RADIUS;
                const y = Math.sin(a) * RADIUS;
                return `<button class="radial__item${it.enabled ? "" : " is-disabled"}" data-item="${i}" data-${it.action}="${it.value}"
                    style="transform: translate(${x.toFixed(1)}px, ${y.toFixed(1)}px)">${it.glyph}</button>`;
            })
            .join("");
        return `<div class="radial__hub"><span>${escapeHtml(title)}</span></div>
                ${ring}
                <div class="radial__below">${below}<div class="radial__tip" hidden></div></div>`;
    }

    /** Appends "Requires: …" (red) when requirements are missing. */
    private withReq(meta: string, r: Requires | undefined): string {
        const m = this.d.tech.missingText(r);
        return m ? `${meta}<br><span class="radial__tip-req">${escapeHtml(m)}</span>` : meta;
    }

    private setHover(i: number): void {
        if (i === this.hover) return;
        this.hover = i;
        this.renderTip();
    }

    private renderTip(): void {
        const tip = this.el.querySelector<HTMLElement>(".radial__tip");
        if (!tip) return;
        const it = this.items[this.hover];
        tip.hidden = !it;
        if (!it) return;
        const cost = this.d.economy.costHtml(it.cost, it.uses);
        tip.innerHTML = `<div class="radial__tip-title">${escapeHtml(it.title)}</div>
            ${cost ? `<div class="radial__tip-cost">${cost}</div>` : ""}
            ${it.meta ? `<div class="radial__tip-meta">${it.meta}</div>` : ""}`;
    }

    private onClick = (e: MouseEvent): void => {
        const el = (e.target as HTMLElement).closest<HTMLElement>("button");
        if (!el || el.classList.contains("is-disabled")) return;
        const t = this.target;
        if (el.dataset.kind !== undefined && t && "plot" in t) {
            this.d.onBuild(t.plot.host, t.plot.plot, Number(el.dataset.kind));
            this.clear();
            return;
        }
        if (el.dataset.place !== undefined && t && "workers" in t) {
            this.d.onPlace(Number(el.dataset.place), t.workers);
            return;
        }
        if (el.dataset.toggle !== undefined && t && "building" in t) {
            this.d.onBuildingToggle(t.building, Number(el.dataset.toggle));
            return;
        }
        if (el.dataset.gate !== undefined && t && "building" in t) {
            this.d.onGate(t.building, el.dataset.gate === "1");
            return;
        }
        if (el.dataset.train !== undefined && t && "building" in t) {
            this.d.onTrain(t.building, Number(el.dataset.train));
            return;
        }
        if (el.dataset.research !== undefined && t && "building" in t) {
            this.d.onResearch(t.building, Number(el.dataset.research));
            return;
        }
        if (el.dataset.untrain !== undefined && t && "building" in t) {
            this.d.onUntrain(t.building, Number(el.dataset.untrain));
            return;
        }
        if (el.dataset.cancel !== undefined) {
            this.d.onCancel(Number(el.dataset.cancel));
            this.clear();
        }
    };
}

function escapeHtml(s: string): string {
    return s.replace(/[&<>"']/g, (c) => `&#${c.charCodeAt(0)};`);
}
