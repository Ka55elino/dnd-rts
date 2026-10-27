import * as THREE from "three";
import { PLAYER_AI, PLAYER_DEFEATED, PLAYER_LOCAL, type GameSnapshot, type HeroSnap, type HordeSnap } from "../../net/protocol";
import { PLAYER_COLORS, type HordeTypeInfo } from "../view/SoldiersView";
import type { Economy } from "./economy";
import { iconFor } from "./icons";

export interface HudDeps {
    overlay: HTMLElement;
    economy: Economy;
    factions: { id: string; name: string }[];
    tickMs: number;
    /** Horde type by horde id. */
    typeOf: (hordeId: number) => (HordeTypeInfo & { id: string }) | undefined;
    /** Horde type by snapshot kind. */
    typeOfKind: (kind: number) => (HordeTypeInfo & { id: string }) | undefined;
    onSelect: (hordeIds: number[], focus: boolean) => void;
}

/**
 * The game HUD frame (DOM only; game state comes in through update calls):
 * - bottom left: round minimap in a bronze frame and the resource plate
 *   beneath it (orders and abilities live on the command card, bottom right);
 * - bottom centre: portraits of the selected hordes (count, health ring);
 * - left edge: own heroes (click selects, double click centres the camera),
 *   with level and experience; fallen ones greyed out until revived;
 * - top left: player banners with their faction and You / AI tag.
 */
export class Hud {
    readonly minimapFrame: HTMLDivElement;
    private readonly root: HTMLDivElement;
    private readonly playersEl: HTMLDivElement;
    private readonly resEl: HTMLDivElement;
    private readonly selectionEl: HTMLDivElement;
    private readonly heroesEl: HTMLDivElement;
    private lastPlayers = "";
    private lastSelection = "";
    private lastHeroes = "";
    private lastAmounts: number[] = [];
    private lastLocal = -1;

    constructor(private readonly d: HudDeps) {
        this.root = document.createElement("div");
        this.root.className = "hud";
        this.root.innerHTML = `
            <div class="hud-players"></div>
            <div class="hud-heroes"></div>
            <div class="hud-console">
                <div class="hud-resources"></div>
                <div class="hud-minimap"><div class="hud-minimap__frame"></div></div>
            </div>
            <div class="hud-selection"></div>
        `;
        this.playersEl = this.root.querySelector(".hud-players")!;
        this.heroesEl = this.root.querySelector(".hud-heroes")!;
        this.resEl = this.root.querySelector(".hud-resources")!;
        this.minimapFrame = this.root.querySelector(".hud-minimap__frame")!;
        this.selectionEl = this.root.querySelector(".hud-selection")!;

        this.resEl.innerHTML = d.economy.ordered
            .map(
                ({ info, index }) =>
                    `<div class="hud-res" data-res="${index}" title="${info.name}">
                        <i class="hud-res__gem hud-res__gem--${info.id}"></i>
                        <span class="hud-res__value">–</span>
                        <span class="hud-res__delta"></span>
                    </div>`,
            )
            .join("");

        // Clicks on HUD elements must not reach the 3D view.
        for (const el of [this.selectionEl, this.heroesEl, this.resEl]) {
            el.addEventListener("mousedown", (e) => e.stopPropagation());
        }
        this.selectionEl.addEventListener("click", (e) => this.clickHorde(e, false));
        this.heroesEl.addEventListener("click", (e) => this.clickHorde(e, false));
        this.heroesEl.addEventListener("dblclick", (e) => this.clickHorde(e, true));
        d.overlay.appendChild(this.root);
    }

    /** Players and resources (every snapshot). */
    update(snap: GameSnapshot): void {
        const players = snap.players
            .map((p) => {
                const color = cssColor(p.id);
                const faction = escapeHtml(this.d.factions[p.faction]?.name ?? "?");
                const tag = p.flags & PLAYER_LOCAL ? "You" : p.flags & PLAYER_AI ? "AI" : "";
                const cls = `hud-player${p.flags & PLAYER_LOCAL ? " is-local" : ""}${p.flags & PLAYER_DEFEATED ? " is-defeated" : ""}`;
                return `<div class="${cls}" style="--team:${color}">
                    <span class="hud-player__banner"></span>
                    <span class="hud-player__name">${faction}</span>
                    <span class="hud-player__tag">${tag}</span>
                </div>`;
            })
            .join("");
        if (players !== this.lastPlayers) {
            this.playersEl.innerHTML = players;
            this.lastPlayers = players;
        }

        const switched = snap.localPlayer !== this.lastLocal;
        this.lastLocal = snap.localPlayer;
        for (const { info, index } of this.d.economy.ordered) {
            const r = snap.resources[index];
            if (!r) continue;
            const row = this.resEl.querySelector<HTMLElement>(`[data-res="${index}"]`)!;
            const value = row.querySelector<HTMLElement>(".hud-res__value")!;
            const text = info.kind === "cap" ? `${r.amount}/${r.limit}` : `${r.amount}`;
            if (value.textContent !== text) value.textContent = text;
            row.classList.toggle("is-full", info.kind === "cap" && r.amount >= r.limit);
            const prev = this.lastAmounts[index];
            if (!switched && info.kind === "stock" && prev !== undefined && r.amount > prev) {
                const delta = row.querySelector<HTMLElement>(".hud-res__delta")!;
                delta.textContent = `+${r.amount - prev}`;
                delta.classList.remove("is-on");
                void delta.offsetWidth;
                delta.classList.add("is-on");
            }
            this.lastAmounts[index] = r.amount;
        }
    }

    /** Selection portraits and hero column (a few times a second). */
    render(selected: number[], hordes: ReadonlyMap<number, HordeSnap>, localPlayer: number, fallen: HeroSnap[] = []): void {
        // Selection portraits.
        const sel = selected
            .map((id) => {
                const h = hordes.get(id);
                const t = this.d.typeOf(id);
                return h && t ? this.portrait(h, t, "hud-portrait") : "";
            })
            .join("");
        if (sel !== this.lastSelection) {
            this.selectionEl.innerHTML = sel;
            this.selectionEl.classList.toggle("is-empty", sel === "");
            this.lastSelection = sel;
        }

        // Heroes: every own hero, selected ones highlighted.
        const heroes = [...hordes.values()]
            .filter((h) => h.owner === localPlayer && this.d.typeOf(h.id)?.hero)
            .sort((a, b) => a.id - b.id)
            .map((h) => this.portrait(h, this.d.typeOf(h.id)!, `hud-hero${selected.includes(h.id) ? " is-selected" : ""}`))
            .join("") +
            fallen
                .filter((r) => !r.alive)
                .map((r) => {
                    const t = this.d.typeOfKind(r.kind);
                    if (!t) return "";
                    return `<div class="hud-hero is-fallen" title="${escapeHtml(t.name)} has fallen — revive at the castle (level ${r.level})">
                        <span class="hud-portrait__face">${iconFor(t.id, t.name)}</span>
                        <span class="hud-portrait__level">${r.level}</span>
                    </div>`;
                })
                .join("");
        if (heroes !== this.lastHeroes) {
            this.heroesEl.innerHTML = heroes;
            this.lastHeroes = heroes;
        }
    }

    dispose(): void {
        this.root.remove();
    }

    private portrait(h: HordeSnap, t: HordeTypeInfo & { id: string }, cls: string): string {
        const hp = h.maxHp > 0 ? h.hp / h.maxHp : 0;
        const count = t.size > 1 ? `<span class="hud-portrait__count">${h.alive}</span>` : "";
        let level = "";
        let title = `${escapeHtml(t.name)} — ${h.hp}/${h.maxHp}`;
        if (h.level > 0) {
            // XP bar: progress inside the current level (full at max level).
            const xp = t.heroXp ?? [];
            const from = xp[h.level - 1] ?? 0;
            const to = xp[h.level];
            const p = to === undefined ? 1 : Math.min(1, (h.xp - from) / Math.max(1, to - from));
            level = `<span class="hud-portrait__level">${h.level}</span><span class="hud-portrait__xp" style="--xp:${p.toFixed(3)}"></span>`;
            title += ` · level ${h.level} · ${to === undefined ? `${h.xp} XP (max)` : `${h.xp}/${to} XP`}`;
        }
        return `<button class="${cls}" data-horde="${h.id}" style="--hp:${hp.toFixed(3)};--team:${cssColor(h.owner)}" title="${title}">
            <span class="hud-portrait__face">${iconFor(t.id, t.name)}</span>${count}${level}
        </button>`;
    }

    private clickHorde(e: MouseEvent, focus: boolean): void {
        const id = (e.target as HTMLElement).closest<HTMLElement>("[data-horde]")?.dataset.horde;
        if (id) this.d.onSelect([Number(id)], focus);
    }
}

function cssColor(owner: number): string {
    return `#${new THREE.Color(PLAYER_COLORS[owner % PLAYER_COLORS.length]).getHexString()}`;
}

function escapeHtml(s: string): string {
    return s.replace(/[&<>"']/g, (c) => `&#${c.charCodeAt(0)};`);
}
