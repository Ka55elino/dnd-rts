import { ABILITY_AUTOCAST, ABILITY_LOCKED, ABILITY_PENDING, ABILITY_TOGGLED, type AbilitySlotSnap, type HordeSnap } from "../../net/protocol";
import type { HordeTypeInfo } from "../view/SoldiersView";
import type { BuildingTypeInfo } from "../view/BuildingsView";
import type { Economy } from "./economy";
import { icon, iconFor } from "./icons";

/** Orders that need a target on the map (the next click picks it). */
export type TargetMode = "move" | "attack" | "patrol" | "fireAt" | "cast";

/** Client-side ability info (mirror of ContentService.AbilityInfo; index = kind). */
export interface AbilityInfo {
    id: string;
    name: string;
    kind: string; // active / toggle / passive / aura
    target: string; // self / point / unit
    affects: string;
    range: number;
    cooldown: number;
    autocast: boolean;
    radius: number;
    damageType: string;
    requires: import("./tech").Requires;
    /** Cone width in degrees (0 = circle). */
    cone: number;
    /** Special effects in words (chain, summon, teleport, transform). */
    description: string;
    /** Mode abilities: the choices (elements). */
    modes: { id: string; name: string; attackType: string }[] | null;
    ultimate: boolean;
}

export interface CommandCardDeps {
    overlay: HTMLElement;
    economy: Economy;
    tech: import("./tech").Tech;
    tickMs: number;
    /** All building types (index = snapshot kind). */
    buildings: BuildingTypeInfo[];
    /** Building ids the local faction may construct. */
    buildable: () => string[];
    typeOf: (hordeId: number) => HordeTypeInfo | undefined;
    onTarget: (mode: TargetMode) => void;
    onStop: () => void;
    onHold: () => void;
    /** All abilities (index = kind) and the latest slots of a horde. */
    abilities: AbilityInfo[];
    slotsOf: (hordeId: number) => AbilitySlotSnap[] | undefined;
    /** Active ability clicked: cast now (self) or wait for a target (point / unit). */
    onCast: (kind: number, hordeIds: number[]) => void;
    /** Toggle; for mode abilities `mode` is the mode to switch to. */
    onToggle: (kind: number, hordeIds: number[], mode?: number) => void;
    onAutocast: (kind: number, on: boolean, hordeIds: number[]) => void;
    /** Aura / passive clicked: show what it covers (aura radius). */
    onInspect: (kind: number, hordeIds: number[]) => void;
    /** Workers: start placing a free building. */
    onPlace: (kind: number, workers: number[]) => void;
}

interface Cell {
    id: string; // stable key: "move", "build:3", ...
    glyph: string;
    title: string;
    key: string; // hotkey label
    code: string; // KeyboardEvent.code
    detail?: string; // tooltip extra (HTML)
    enabled?: boolean;
    active?: boolean;
    cd?: number; // cooldown fraction 0..1
    toggled?: boolean;
    autocast?: boolean; // undefined = not available
    passive?: boolean;
    ultimate?: boolean;
    run: () => void;
    alt?: () => void; // right click (autocast)
}

const COLS = 4;
const ROWS = 3;

/**
 * Command card (bottom right, WC3 style 4×3 grid) for the selected hordes:
 * - row 1: Move, Stop, Hold, Attack; row 2: Patrol (more orders later);
 *   row 3: abilities (heroes now, more abilities later);
 * - workers only: Move and Stop, then the free buildings they can put up.
 * Hotkeys are shown on each cell (G, X, H, F, P, 1–9). WASD / Q / E stay
 * with the camera.
 */
export class CommandCard {
    private readonly el: HTMLDivElement;
    private readonly grid: HTMLDivElement;
    private readonly tip: HTMLDivElement;
    private cells: (Cell | null)[] = [];
    private lastKey = "";
    private lastTipDetail = "";
    private hover = -1;
    private mode: string | null = null;

    constructor(private readonly d: CommandCardDeps) {
        this.el = document.createElement("div");
        this.el.className = "hud-card";
        this.el.innerHTML = `<div class="hud-card__tip" hidden></div><div class="hud-card__grid"></div>`;
        this.grid = this.el.querySelector(".hud-card__grid")!;
        this.tip = this.el.querySelector(".hud-card__tip")!;
        this.el.addEventListener("mousedown", (e) => e.stopPropagation());
        this.grid.addEventListener("click", (e) => {
            const i = (e.target as HTMLElement).closest<HTMLElement>("[data-cell]")?.dataset.cell;
            const cell = i === undefined ? null : this.cells[Number(i)];
            if (cell && cell.enabled !== false) cell.run();
        });
        this.grid.addEventListener("contextmenu", (e) => {
            e.preventDefault();
            const i = (e.target as HTMLElement).closest<HTMLElement>("[data-cell]")?.dataset.cell;
            const cell = i === undefined ? null : this.cells[Number(i)];
            cell?.alt?.();
        });
        this.grid.addEventListener("mouseover", (e) => {
            const i = (e.target as HTMLElement).closest<HTMLElement>("[data-cell]")?.dataset.cell;
            this.setHover(i === undefined ? -1 : Number(i));
        });
        this.grid.addEventListener("mouseleave", () => this.setHover(-1));
        d.overlay.appendChild(this.el);
    }

    /** Highlights the order waiting for a target (null = none). */
    setMode(mode: TargetMode | null, castKind = -1): void {
        this.mode = mode === "cast" ? `ability:${castKind}` : mode;
    }

    /** Rebuilds the cells for the selection (a few times a second). */
    render(selected: number[], hordes: ReadonlyMap<number, HordeSnap>, localPlayer: number): void {
        const own = selected.filter((id) => hordes.get(id)?.owner === localPlayer);
        const workers = own.filter((id) => this.d.typeOf(id)?.worker);
        const fighters = own.filter((id) => !this.d.typeOf(id)?.worker);
        const cells: (Cell | null)[] = new Array(COLS * ROWS).fill(null);

        if (own.length > 0) {
            cells[0] = this.order("move", "Move", "G", "KeyG", "Move to the clicked point (ignores enemies on the way)");
            cells[1] = { id: "stop", glyph: icon("stop"), title: "Stop", key: "X", code: "KeyX", detail: "Stand still and forget the current order", run: () => this.d.onStop() };
            if (fighters.length > 0) {
                cells[2] = { id: "hold", glyph: icon("hold"), title: "Hold position", key: "H", code: "KeyH", detail: "Fight enemies in reach, never chase", run: () => this.d.onHold() };
                cells[3] = this.order("attack", "Attack", "F", "KeyF", "Click an enemy to attack it, or the ground to attack-move: fight everything met on the way");
                cells[4] = this.order("patrol", "Patrol", "P", "KeyP", "Walk back and forth between here and the clicked point, fighting enemies met");
            }
            if (own.some((id) => this.d.typeOf(id)?.ranged)) {
                cells[5] = this.order("fireAt", "Attack ground", "T", "KeyT", "Shoot volleys at the clicked spot until given another order (walks into range first)");
            }
        }

        if (fighters.length > 0) {
            // Abilities of the selection: the bottom row, then free cells of row 2.
            const slots = [2 * COLS, 2 * COLS + 1, 2 * COLS + 2, 2 * COLS + 3, COLS + 1, COLS + 2, COLS + 3].filter((s) => !cells[s]);
            this.abilityCells(fighters, hordes).forEach((cell, i) => {
                if (i < slots.length) cells[slots[i]] = cell;
            });
        } else if (workers.length > 0) {
            // Free buildings, filling the cells after Move / Stop.
            const buildable = this.d.buildable();
            let slot = COLS; // second row
            this.d.buildings.forEach((type, kind) => {
                if (type.kind !== "building" || !type.placement.includes("free") || !buildable.includes(type.id)) return;
                if (slot >= cells.length) return;
                const n = slot - COLS + 1;
                cells[slot++] = {
                    id: `build:${kind}`,
                    glyph: iconFor(type.id, type.name),
                    title: `Build ${type.name}`,
                    key: `${n}`,
                    code: `Digit${n}`,
                    detail: `${this.d.economy.costHtml(type.cost)}<br>${type.width}×${type.depth} · ${type.buildTime.toFixed(0)}s${this.req(type.requires)}`,
                    enabled: this.d.economy.canAfford(type.cost) && this.d.tech.missing(type.requires).length === 0,
                    run: () => this.d.onPlace(kind, workers),
                };
            });
        }

        this.cells = cells;
        // Cooldowns and "ready in" counters are not part of the key: they are
        // written into the existing buttons, so the grid isn't rebuilt (no flicker).
        const key = cells
            .map((c) =>
                c ? `${c.id}|${c.enabled !== false}|${c.id === this.mode}|${c.toggled}|${c.autocast}|${(c.detail ?? "").replace(/ready in \d+s/, "")}` : "-",
            )
            .join(";");
        if (key === this.lastKey) {
            this.applyLive();
            return;
        }
        this.lastKey = key;
        this.grid.innerHTML = cells
            .map((c, i) => {
                if (!c) return `<span class="hud-card__cell is-empty"></span>`;
                const cls = `hud-card__cell${c.enabled === false ? " is-disabled" : ""}${c.id === this.mode || c.toggled ? " is-active" : ""}${c.autocast ? " is-autocast" : ""}${c.passive ? " is-passive" : ""}${c.ultimate ? " is-ultimate" : ""}`;
                return `<button class="${cls}" data-cell="${i}" style="--cd:${(c.cd ?? 0).toFixed(3)}">
                    ${c.glyph}<span class="hud-card__key">${c.key}</span>
                </button>`;
            })
            .join("");
        this.renderTip();
    }

    /** Cooldown sweeps and the tooltip's countdown, without rebuilding the grid. */
    private applyLive(): void {
        this.grid.querySelectorAll<HTMLElement>("[data-cell]").forEach((el) => {
            const cd = (this.cells[Number(el.dataset.cell)]?.cd ?? 0).toFixed(3);
            if (el.style.getPropertyValue("--cd") !== cd) el.style.setProperty("--cd", cd);
        });
        const c = this.cells[this.hover];
        const detail = c?.detail ?? "";
        if (c && detail !== this.lastTipDetail) this.renderTip();
    }

    /** Hotkey: returns true when it ran a command. */
    hotkey(code: string): boolean {
        const cell = this.cells.find((c) => c?.code === code);
        if (!cell || cell.enabled === false) return false;
        cell.run();
        return true;
    }

    dispose(): void {
        this.el.remove();
    }

    /** One cell per ability the selected hordes have (grouped by ability). */
    private abilityCells(ids: number[], hordes: ReadonlyMap<number, HordeSnap>): Cell[] {
        const groups = new Map<number, { hordes: number[]; ready: number[]; cd: number; toggled: boolean; auto: boolean; pending: boolean; unlockAt: number; mode: number }>();
        for (const id of ids) {
            const type = this.d.typeOf(id);
            const slots = this.d.slotsOf(id);
            if (!type || !slots || !hordes.has(id)) continue;
            type.abilities.forEach((abId, i) => {
                const kind = this.d.abilities.findIndex((a) => a.id === abId);
                const slot = slots[i];
                if (kind < 0 || !slot) return;
                let g = groups.get(kind);
                if (!g) {
                    g = { hordes: [], ready: [], cd: Infinity, toggled: true, auto: true, pending: false, unlockAt: 0, mode: slot.mode };
                    groups.set(kind, g);
                }
                // Hero ability i opens at level i + 1: locked slots only say when.
                if (slot.flags & ABILITY_LOCKED) {
                    g.unlockAt = g.unlockAt ? Math.min(g.unlockAt, i + 1) : i + 1;
                    return;
                }
                g.hordes.push(id);
                if (slot.cooldown === 0) g.ready.push(id);
                g.cd = Math.min(g.cd, slot.cooldown);
                g.toggled &&= (slot.flags & ABILITY_TOGGLED) !== 0;
                g.auto &&= (slot.flags & ABILITY_AUTOCAST) !== 0;
                g.pending ||= (slot.flags & ABILITY_PENDING) !== 0;
            });
        }
        const cells: Cell[] = [];
        let hot = 1;
        for (const [kind, g] of [...groups].sort((a, b) => a[0] - b[0])) {
            const a = this.d.abilities[kind];
            const key = a.kind === "active" || a.kind === "toggle" || a.kind === "mode" ? `${hot++}` : "";
            const code = key ? `Digit${key}` : "";
            const secs = Math.ceil((g.cd * this.d.tickMs) / 1000);
            const base: Omit<Cell, "run"> = { id: `ability:${kind}`, glyph: icon("crown"), title: a.name, key, code, ultimate: a.ultimate };
            if (g.hordes.length === 0) {
                cells.push({ ...base, detail: `${a.kind === "toggle" ? "Toggle" : a.kind === "aura" ? "Aura" : a.kind === "passive" ? "Passive" : "Ability"}<br><span class="hud-card__tip-req">Unlocks at hero level ${g.unlockAt}</span>`, enabled: false, run: () => {} });
                continue;
            }
            const locked = this.d.tech.missing(a.requires).length > 0;
            if (locked) {
                cells.push({ ...base, detail: `${a.kind === "toggle" ? "Toggle" : "Ability"}${this.req(a.requires)}`, enabled: false, run: () => {} });
                continue;
            }
            switch (a.kind) {
                case "active": {
                    const ready = g.ready.length > 0;
                    const target = a.target === "self" ? "on itself" : a.target === "point" ? "at a point" : "on a unit";
                    cells.push({
                        ...base,
                        detail: `Cast ${target}${a.range ? ` · range ${a.range >= 1000 ? "any" : a.range}` : ""}${a.radius && !a.cone ? ` · radius ${a.radius}` : ""} · cooldown ${a.cooldown}s${a.description ? `<br>${a.description}` : ""}${ready ? "" : `<br>ready in ${secs}s`}${g.pending ? "<br>moving into range…" : ""}${a.autocast ? "<br>right click: autocast" : ""}`,
                        enabled: ready,
                        cd: ready ? 0 : Math.min(1, (g.cd * this.d.tickMs) / 1000 / Math.max(a.cooldown, 0.01)),
                        autocast: a.autocast ? g.auto : undefined,
                        run: () => this.d.onCast(kind, g.ready),
                        alt: a.autocast ? () => this.d.onAutocast(kind, !g.auto, g.hordes) : undefined,
                    });
                    break;
                }
                case "mode": {
                    // Element choice: click → next mode, right click → previous (switching has a cooldown).
                    const modes = a.modes ?? [];
                    const cur = modes[g.mode];
                    const ready = g.cd === 0;
                    const next = (g.mode + 1) % Math.max(modes.length, 1);
                    const prev = (g.mode + modes.length - 1) % Math.max(modes.length, 1);
                    const list = modes.map((m, i) => (i === g.mode ? `<b>${m.name}</b>` : m.name)).join(" · ");
                    cells.push({
                        ...base,
                        title: `${a.name}: ${cur?.name ?? "?"}`,
                        detail: `${list}<br>click: ${modes[next]?.name ?? ""} · right click: ${modes[prev]?.name ?? ""}${a.cooldown ? ` · switch ${a.cooldown}s` : ""}${ready ? "" : `<br>ready in ${secs}s`}`,
                        enabled: ready,
                        cd: ready ? 0 : Math.min(1, (g.cd * this.d.tickMs) / 1000 / Math.max(a.cooldown, 0.01)),
                        toggled: true,
                        run: () => this.d.onToggle(kind, g.hordes, next),
                        alt: () => this.d.onToggle(kind, g.hordes, prev),
                    });
                    break;
                }
                case "toggle":
                    cells.push({ ...base, detail: g.toggled ? "On — click to turn off" : "Off — click to turn on", toggled: g.toggled, run: () => this.d.onToggle(kind, g.hordes) });
                    break;
                default:
                    cells.push({
                        ...base,
                        detail: a.kind === "aura" ? `Aura · radius ${a.radius}<br>click: show the radius` : "Passive",
                        passive: true,
                        run: () => this.d.onInspect(kind, g.hordes),
                    });
            }
        }
        return cells;
    }

    /** "<br>Requires: …" in red, or "". */
    private req(r: import("./tech").Requires | undefined): string {
        const m = this.d.tech.missingText(r);
        return m ? `<br><span class="hud-card__tip-req">${m}</span>` : "";
    }

    private order(mode: TargetMode, title: string, key: string, code: string, detail: string): Cell {
        return { id: mode, glyph: icon(mode), title, key, code, detail, run: () => this.d.onTarget(mode) };
    }

    private setHover(i: number): void {
        if (i === this.hover) return;
        this.hover = i;
        this.renderTip();
    }

    private renderTip(): void {
        const c = this.cells[this.hover];
        this.lastTipDetail = c?.detail ?? "";
        this.tip.hidden = !c;
        if (!c) return;
        this.tip.innerHTML = `<div class="hud-card__tip-title">${escapeHtml(c.title)} <span class="hud-card__tip-key">${c.key}</span></div>
            ${c.detail ? `<div class="hud-card__tip-detail">${c.detail}</div>` : ""}`;
    }
}

function escapeHtml(s: string): string {
    return s.replace(/[&<>"']/g, (c) => `&#${c.charCodeAt(0)};`);
}
