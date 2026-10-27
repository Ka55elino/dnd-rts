import { isConstructing, QUEUE_RESEARCH, type GameSnapshot, type HeroSnap } from "../../net/protocol";

/** Requirements as the HUD knows them (mirror of ContentService.RequiresInfo). */
export interface Requires {
    buildings: string[] | null;
    upgrades: string[] | null;
}

/** Client-side upgrade info (mirror of ContentService.UpgradeInfo; index = kind). */
export interface UpgradeInfo {
    id: string;
    name: string;
    cost: Partial<Record<string, number>> | null;
    researchTime: number;
    group: string;
    requires: Requires;
    description: string;
}

/**
 * What the local player has (researched upgrades, finished building types),
 * refreshed from every snapshot, to grey out buttons and say what is missing.
 * Display only: the simulation checks every command itself.
 */
export class Tech {
    private owned = new Set<string>();
    private builtTypes = new Set<string>();
    /** Local player's unique hordes (alive or fallen), by horde kind. */
    private heroes = new Map<number, HeroSnap>();
    /** Horde kinds in any of the local player's queues. */
    private queuedKinds = new Set<number>();

    constructor(
        readonly upgrades: UpgradeInfo[],
        private readonly buildingTypes: { id: string; name: string }[],
    ) {}

    update(snap: GameSnapshot): void {
        this.heroes = new Map(snap.heroes.map((h) => [h.kind, h]));
        this.queuedKinds.clear();
        for (const q of snap.queues) for (const k of q.hordes) if (!(k & QUEUE_RESEARCH)) this.queuedKinds.add(k);
        this.owned = new Set([...snap.upgrades].map((k) => this.upgrades[k]?.id).filter((id): id is string => !!id));
        this.builtTypes.clear();
        for (const b of snap.buildings) {
            if (b.owner === snap.localPlayer && !isConstructing(b.flags)) {
                const t = this.buildingTypes[b.kind];
                if (t) this.builtTypes.add(t.id);
            }
        }
    }

    /** Record of a unique horde the player has hired before (undefined = never). */
    hero(kind: number): HeroSnap | undefined {
        return this.heroes.get(kind);
    }

    /** The horde kind is being trained somewhere. */
    queued(kind: number): boolean {
        return this.queuedKinds.has(kind);
    }

    has(upgradeId: string): boolean {
        return this.owned.has(upgradeId);
    }

    /** Names of what is still missing (empty = all requirements met). */
    missing(r: Requires | null | undefined): string[] {
        if (!r) return [];
        const out: string[] = [];
        for (const id of r.buildings ?? []) {
            if (!this.builtTypes.has(id)) out.push(this.buildingTypes.find((t) => t.id === id)?.name ?? id);
        }
        for (const id of r.upgrades ?? []) {
            if (!this.owned.has(id)) out.push(this.upgrades.find((u) => u.id === id)?.name ?? id);
        }
        return out;
    }

    /** "Requires: Barracks, Forged Weapons I" or "". */
    missingText(r: Requires | null | undefined): string {
        const m = this.missing(r);
        return m.length ? `Requires: ${m.join(", ")}` : "";
    }
}
