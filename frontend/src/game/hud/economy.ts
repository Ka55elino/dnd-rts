import type { ResourceSnap } from "../../net/protocol";

/** Client-side resource info (mirror of ContentService.ResourceInfo; index = snapshot order). */
export interface ResourceInfo {
    id: string;
    name: string;
    icon: string;
    kind: string; // "stock" | "cap"
    order: number;
}

export type Amounts = Partial<Record<string, number>>;

/** Icon colors per resource id (placeholder art). */
const ICON_COLORS: Record<string, string> = {
    gold: "#e8c34a",
    wood: "#a5713f",
    metal: "#a9b3bc",
    food: "#8bc34a",
};

/** Local player's economy as of the latest snapshot, plus formatting helpers. */
export class Economy {
    resources: ResourceSnap[] = [];

    constructor(readonly infos: ResourceInfo[]) {}

    /** Display order (by `order`). */
    get ordered(): { info: ResourceInfo; index: number }[] {
        return this.infos.map((info, index) => ({ info, index })).sort((a, b) => a.info.order - b.info.order);
    }

    amount(id: string): number {
        const i = this.infos.findIndex((r) => r.id === id);
        return this.resources[i]?.amount ?? 0;
    }

    canAfford(cost: Amounts | null | undefined): boolean {
        return this.missing(cost).length === 0;
    }

    /** Stock resources the player lacks for cost. */
    missing(cost: Amounts | null | undefined): string[] {
        if (!cost) return [];
        return Object.entries(cost).filter(([id, n]) => (n ?? 0) > this.amount(id)).map(([id]) => id);
    }

    /** Whether cap resources (food) have room for `uses` more. */
    hasRoom(uses: Amounts | null | undefined): boolean {
        if (!uses) return true;
        return Object.entries(uses).every(([id, n]) => {
            const i = this.infos.findIndex((r) => r.id === id);
            const r = this.resources[i];
            return !r || r.amount + (n ?? 0) <= r.limit;
        });
    }

    /** "● 150 ● 50" with shortfalls marked. */
    costHtml(cost: Amounts | null | undefined, uses?: Amounts | null): string {
        const parts: string[] = [];
        for (const { info } of this.ordered) {
            const n = cost?.[info.id] ?? uses?.[info.id];
            if (!n) continue;
            const short = info.kind === "stock" ? n > this.amount(info.id) : !this.hasRoom({ [info.id]: n });
            parts.push(`<span class="res-cost${short ? " is-short" : ""}" title="${info.name}">${iconHtml(info.id)}${n}</span>`);
        }
        return parts.join(" ");
    }
}

export function iconHtml(id: string): string {
    return `<i class="res-icon" style="background:${ICON_COLORS[id] ?? "#ccc"}"></i>`;
}
