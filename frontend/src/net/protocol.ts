// Binary wire format. Mirror of internal/protocol/protocol.go — change both together.

/** Wire format version: fixed at 1 until release 0.0.1 (see internal/protocol.Version). */
export const PROTOCOL_VERSION = 1;

export const MsgSnapshot = 1;
export const MsgBenchSetCount = 100;

export const SNAPSHOT_HEADER_SIZE = 16;
/** Floats per entity in a snapshot: x, z, rotation. */
export const SNAPSHOT_ENTITY_FLOATS = 3;

export interface Snapshot {
    tick: number;
    count: number;
    /** Server time spent on this tick, microseconds. */
    simMicros: number;
    /** View over the frame: [x0, z0, rot0, x1, z1, rot1, ...]. No copy. */
    entities: Float32Array;
    byteLength: number;
}

export function decodeSnapshot(buf: ArrayBuffer): Snapshot | null {
    if (buf.byteLength < SNAPSHOT_HEADER_SIZE) return null;
    const view = new DataView(buf);
    if (view.getUint8(0) !== MsgSnapshot) return null;
    if (view.getUint8(1) !== PROTOCOL_VERSION) {
        console.warn(`protocol version mismatch: got ${view.getUint8(1)}, want ${PROTOCOL_VERSION}`);
        return null;
    }
    const tick = view.getUint32(4, true);
    const count = view.getUint32(8, true);
    const simMicros = view.getUint32(12, true);

    // Float32Array uses platform endianness; all target platforms (x86-64, arm64) are little endian.
    const entities = new Float32Array(buf, SNAPSHOT_HEADER_SIZE, count * SNAPSHOT_ENTITY_FLOATS);
    return { tick, count, simMicros, entities, byteLength: buf.byteLength };
}

export function encodeBenchSetCount(count: number): ArrayBuffer {
    const buf = new ArrayBuffer(8);
    const view = new DataView(buf);
    view.setUint8(0, MsgBenchSetCount);
    view.setUint8(1, PROTOCOL_VERSION);
    view.setUint32(4, count, true);
    return buf;
}

// ---------------------------------------------------------------------------
// Game messages (mirror of internal/protocol/game.go)

export const MsgGameSnapshot = 2;
export const MsgConsoleOutput = 3;
export const MsgHello = 110;
export const MsgCmdMove = 111;
export const MsgCmdAttack = 112;
export const MsgCmdAbility = 113;
export const MsgCmdConsole = 114;
export const MsgCmdSiege = 115;
export const MsgCmdPlot = 116;
export const MsgCmdCancel = 117;
export const MsgCmdBuild = 118;
export const MsgCmdHelp = 119;
export const MsgCmdTrain = 120;
export const MsgCmdUntrain = 121;
export const MsgCmdRally = 122;
export const MsgCmdStop = 123;
export const MsgCmdHold = 124;
export const MsgCmdAMove = 125;
export const MsgCmdPatrol = 126;
export const MsgCmdFireAt = 133;
export const MsgCmdCast = 127;
export const MsgCmdToggle = 128;
export const MsgCmdAuto = 129;
export const MsgCmdResearch = 130;
export const MsgCmdGate = 131;
export const MsgCmdBToggle = 132;
/** Queue entry kind flag: research (kind & 0x7fff = upgrade index). */
export const QUEUE_RESEARCH = 0x8000;

/** PlayerSnap.flags */
export const PLAYER_AI = 1;
export const PLAYER_DEFEATED = 2;
export const PLAYER_LOCAL = 4;

export interface PlayerSnap {
    id: number;
    /** Index into the sorted faction list (ContentService.Factions). */
    faction: number;
    flags: number;
}

/** Local player's resource, in sorted resource id order (ContentService.Resources). */
export interface ResourceSnap {
    /** Stock: amount. Cap (food): used. */
    amount: number;
    /** Cap: provided. Stock: 0. */
    limit: number;
}

export interface QueueSnap {
    building: number;
    /** Horde kinds, front first. */
    hordes: number[];
    /** Front entry progress, 0..1. */
    progress: number;
}

/** BuildingSnap.flags helpers (internal/protocol BuildingFlags). */
export const isConstructing = (flags: number): boolean => (flags & 1) !== 0;
export const isCapturing = (flags: number): boolean => (flags & 2) !== 0;
/** Construction progress (1 when finished), or capture progress while capturing. */
export const buildProgress = (flags: number): number => (flags & 3 ? ((flags >>> 8) & 0xff) / 255 : 1);
export const plotOccupied = (b: { flags: number; plotsHi: number }, plot: number): boolean =>
    plot < 16 ? ((b.flags >>> 16) & (1 << plot)) !== 0 : plot < 48 && (b.plotsHi & (1 << (plot - 16))) !== 0;

/** Soldier animation states (internal/protocol Soldier*). */
export const SoldierIdle = 0;
export const SoldierMoving = 1;
export const SoldierAttacking = 2;
export const SoldierKnocked = 3;

const GAME_HEADER_SIZE = 64;
const GAME_HERO_SIZE = 8;
const GAME_EVENT_SIZE = 24;
const GAME_HORDE_SIZE = 32;
/** Soldier status bits (word 5 >> 16; internal/sim Status*). */
export const STATUS_BURNING = 1;
export const STATUS_POISONED = 2;
export const STATUS_SLOWED = 4;
export const STATUS_STUNNED = 8;
export const STATUS_ACID = 16;
/** Flying unit pulled down: ground attacks can hit it. */
export const STATUS_GROUNDED = 32;
/** Standing on a wall walkway: drawn at the wall's walk height. */
export const STATUS_ON_WALL = 64;
/** BuildingSnap.flags bit: on fire. */
export const isBurning = (flags: number): boolean => (flags & 4) !== 0;
/** BuildingSnap.flags bit: a gate is open. */
export const isGateOpen = (flags: number): boolean => (flags & 8) !== 0;
/** BuildingSnap.flags bit: a tower's toggle (fire arrows) is on. */
export const isToggled = (flags: number): boolean => (flags & 16) !== 0;

/** u32 words per soldier: id, hordeId, x(f32), z(f32), rot(f32), [hp u8 | state u8 | status u16]. */
export const SOLDIER_WORDS = 6;

export interface HordeSnap {
    id: number;
    owner: number;
    kind: number;
    alive: number;
    x: number;
    z: number;
    hp: number;
    maxHp: number;
    /** Hero level 1..5, 0 = not a hero. */
    level: number;
    xp: number;
    /** HORDE_TRANSFORMED | HORDE_SUMMONED. */
    flags: number;
    /** Form size relative to the base unit (1 = normal). */
    scale: number;
}

/** HordeSnap.flags */
export const HORDE_TRANSFORMED = 1;
export const HORDE_SUMMONED = 2;

/** A unique horde of the local player: on the field or waiting for revival. */
export interface HeroSnap {
    /** Index into the horde type list. */
    kind: number;
    level: number;
    xp: number;
    alive: boolean;
}

/** An ability went off this tick (visuals). */
/** CastEventSnap.kind */
export const EVENT_CIRCLE = 0;
export const EVENT_CHAIN = 1;
export const EVENT_CONE = 2;

export interface CastEventSnap {
    /** Index into ContentService.Abilities. */
    ability: number;
    owner: number;
    /** EVENT_CIRCLE / EVENT_CHAIN / EVENT_CONE. */
    kind: number;
    /** Circle: centre; chain: jump start; cone: apex. */
    x: number;
    z: number;
    /** Circle: radius; chain: jump end x; cone: length. */
    radius: number;
    /** Circle: zone seconds (0 = instant); chain: jump end z; cone: width in degrees. */
    duration: number;
    /** Cone: direction (Y rotation, like soldiers). */
    facing: number;
    /** Damage type that went off ("" = none): follows the caster's element. */
    damageType: string;
}

/** AbilitySlotSnap.flags */
export const ABILITY_TOGGLED = 1;
export const ABILITY_AUTOCAST = 2;
export const ABILITY_PENDING = 4;
/** Hero level too low for this slot. */
export const ABILITY_LOCKED = 8;

export interface AbilitySlotSnap {
    cooldown: number; // ticks
    flags: number;
    /** Mode abilities: index of the chosen mode. */
    mode: number;
}

/** Mirror of internal/data.DamageTypes (order matters: events send an index). */
export const DAMAGE_TYPES = ["pierce", "slash", "crush", "fire", "lightning", "ice", "acid", "poison", "necrotic", "radiant"];

/** ProjectileSnap.flags */
export const PROJECTILE_FROM_AIR = 1;
export const PROJECTILE_TO_AIR = 2;
export const PROJECTILE_FROM_WALL = 4;
export const PROJECTILE_SPLASH = 8;

export interface ProjectileSnap {
    id: number;
    owner: number;
    flags: number;
    arrows: number;
    fromX: number;
    fromZ: number;
    toX: number;
    toZ: number;
    launchTick: number;
    impactTick: number;
}

/** BuildingSnap.owner of neutral buildings (internal/sim.Neutral). */
export const NEUTRAL = 255;

export interface BuildingSnap {
    id: number;
    /** Player id, or NEUTRAL. */
    owner: number;
    /** Player capturing a neutral outpost, or NEUTRAL. */
    capturer: number;
    /** Index into the building type list (ContentService.BuildingTypes). */
    kind: number;
    x: number;
    z: number;
    rot: number;
    hp: number;
    maxHp: number;
    flags: number;
    /** Occupied plots 16–47 (tower sockets of a base wall come last). */
    plotsHi: number;
}

const GAME_SOLDIER_SIZE = SOLDIER_WORDS * 4;
const GAME_PROJECTILE_SIZE = 32;
const GAME_BUILDING_SIZE = 36;

export interface GameSnapshot {
    tick: number;
    /** Tick of the latest state hash and the hash itself (hex), for desync checks. */
    hashTick: number;
    hash: string;
    /** The match has ended; winner is a player id or NEUTRAL (draw). */
    over: boolean;
    winner: number;
    /** The player this client controls (can change through the dev console). */
    localPlayer: number;
    players: PlayerSnap[];
    resources: ResourceSnap[];
    queues: QueueSnap[];
    events: CastEventSnap[];
    /** Local player's hordes: ability slots in the order of their type's abilities. */
    abilities: Map<number, AbilitySlotSnap[]>;
    /** Upgrade kinds the local player has researched. */
    upgrades: Set<number>;
    /** Local player's unique hordes, in hire order. */
    heroes: HeroSnap[];
    projectiles: ProjectileSnap[];
    buildings: BuildingSnap[];
    simMicros: number;
    hordes: HordeSnap[];
    soldierCount: number;
    /** Soldier records as u32 words (id, hordeId at +0, +1; hp = w5 & 0xff, state = (w5 >> 8) & 0xff). */
    soldierU32: Uint32Array;
    /** Same memory as f32 (x, z, rot at +2, +3, +4). */
    soldierF32: Float32Array;
    byteLength: number;
}

export function decodeGameSnapshot(buf: ArrayBuffer): GameSnapshot | null {
    if (buf.byteLength < GAME_HEADER_SIZE) return null;
    const v = new DataView(buf);
    if (v.getUint8(0) !== MsgGameSnapshot || v.getUint8(1) !== PROTOCOL_VERSION) return null;

    const hordeCount = v.getUint16(2, true);
    const tick = v.getUint32(4, true);
    const soldierCount = v.getUint32(8, true);
    const simMicros = v.getUint32(12, true);
    const projectileCount = v.getUint32(16, true);
    const hashTick = v.getUint32(20, true);
    const hash =
        v.getUint32(28, true).toString(16).padStart(8, "0") + v.getUint32(24, true).toString(16).padStart(8, "0");
    const buildingCount = v.getUint32(32, true);
    const over = v.getUint8(36) === 1;
    const winner = v.getUint8(37);
    const localPlayer = v.getUint8(38);
    const playerCount = v.getUint8(39);
    const resourceCount = v.getUint8(40);
    const queueCount = v.getUint16(42, true);
    const eventCount = v.getUint16(48, true);
    const abilityCount = v.getUint16(50, true);
    const upgradeCount = v.getUint8(41);
    const heroCount = v.getUint16(56, true);

    const hordes: HordeSnap[] = new Array(hordeCount);
    let off = GAME_HEADER_SIZE;
    for (let i = 0; i < hordeCount; i++, off += GAME_HORDE_SIZE) {
        hordes[i] = {
            id: v.getUint32(off, true),
            owner: v.getUint8(off + 4),
            kind: v.getUint8(off + 5),
            alive: v.getUint16(off + 6, true),
            x: v.getFloat32(off + 8, true),
            z: v.getFloat32(off + 12, true),
            hp: v.getUint32(off + 16, true),
            maxHp: v.getUint32(off + 20, true),
            level: v.getUint8(off + 24),
            flags: v.getUint8(off + 25),
            scale: v.getUint16(off + 26, true) / 100 || 1,
            xp: v.getUint32(off + 28, true),
        };
    }

    // `off` is 4-byte aligned (40 + n*32), so typed views are safe.
    const words = soldierCount * SOLDIER_WORDS;
    const soldierOff = off;

    off += soldierCount * GAME_SOLDIER_SIZE;
    const projectiles: ProjectileSnap[] = new Array(projectileCount);
    for (let i = 0; i < projectileCount; i++, off += GAME_PROJECTILE_SIZE) {
        projectiles[i] = {
            id: v.getUint32(off, true),
            owner: v.getUint8(off + 4),
            flags: v.getUint8(off + 5),
            arrows: v.getUint16(off + 6, true),
            fromX: v.getFloat32(off + 8, true),
            fromZ: v.getFloat32(off + 12, true),
            toX: v.getFloat32(off + 16, true),
            toZ: v.getFloat32(off + 20, true),
            launchTick: v.getUint32(off + 24, true),
            impactTick: v.getUint32(off + 28, true),
        };
    }

    const buildings: BuildingSnap[] = new Array(buildingCount);
    for (let i = 0; i < buildingCount; i++, off += GAME_BUILDING_SIZE) {
        buildings[i] = {
            id: v.getUint32(off, true),
            owner: v.getUint8(off + 4),
            capturer: v.getUint8(off + 5),
            kind: v.getUint16(off + 6, true),
            x: v.getFloat32(off + 8, true),
            z: v.getFloat32(off + 12, true),
            rot: v.getFloat32(off + 16, true),
            hp: v.getUint32(off + 20, true),
            maxHp: v.getUint32(off + 24, true),
            flags: v.getUint32(off + 28, true),
            plotsHi: v.getUint32(off + 32, true),
        };
    }

    const players: PlayerSnap[] = new Array(playerCount);
    for (let i = 0; i < playerCount; i++, off += 4) {
        players[i] = { id: v.getUint8(off), faction: v.getUint8(off + 1), flags: v.getUint8(off + 2) };
    }
    const resources: ResourceSnap[] = new Array(resourceCount);
    for (let i = 0; i < resourceCount; i++, off += 8) {
        resources[i] = { amount: v.getInt32(off, true), limit: v.getInt32(off + 4, true) };
    }
    const queues: QueueSnap[] = new Array(queueCount);
    for (let i = 0; i < queueCount; i++) {
        const n = v.getUint8(off + 4);
        const hordes: number[] = [];
        for (let k = 0; k < n; k++) hordes.push(v.getUint16(off + 8 + k * 2, true));
        queues[i] = { building: v.getUint32(off, true), hordes, progress: v.getUint16(off + 6, true) / 65535 };
        off += 8 + Math.ceil((n * 2) / 4) * 4;
    }

    const events: CastEventSnap[] = new Array(eventCount);
    for (let i = 0; i < eventCount; i++, off += GAME_EVENT_SIZE) {
        events[i] = {
            ability: v.getUint16(off, true),
            owner: v.getUint8(off + 2),
            kind: v.getUint8(off + 3) & 0x0f,
            damageType: DAMAGE_TYPES[(v.getUint8(off + 3) >> 4) - 1] ?? "",
            x: v.getFloat32(off + 4, true),
            z: v.getFloat32(off + 8, true),
            radius: v.getFloat32(off + 12, true),
            duration: v.getFloat32(off + 16, true),
            facing: v.getFloat32(off + 20, true),
        };
    }
    const abilities = new Map<number, AbilitySlotSnap[]>();
    for (let i = 0; i < abilityCount; i++) {
        const horde = v.getUint32(off, true);
        const n = v.getUint8(off + 4);
        const slots: AbilitySlotSnap[] = [];
        for (let k = 0; k < n; k++) {
            slots.push({ cooldown: v.getUint16(off + 8 + k * 4, true), flags: v.getUint8(off + 8 + k * 4 + 2), mode: v.getUint8(off + 8 + k * 4 + 3) });
        }
        abilities.set(horde, slots);
        off += 8 + 4 * n;
    }
    const upgrades = new Set<number>();
    for (let i = 0; i < upgradeCount; i++) upgrades.add(v.getUint16(off + i * 2, true));
    off += Math.ceil((upgradeCount * 2) / 4) * 4;
    const heroes: HeroSnap[] = new Array(heroCount);
    for (let i = 0; i < heroCount; i++, off += GAME_HERO_SIZE) {
        heroes[i] = {
            kind: v.getUint16(off, true),
            level: v.getUint8(off + 2),
            alive: (v.getUint8(off + 3) & 1) !== 0,
            xp: v.getUint32(off + 4, true),
        };
    }

    return {
        tick,
        hashTick,
        hash,
        over,
        winner,
        localPlayer,
        players,
        resources,
        queues,
        events,
        abilities,
        upgrades,
        heroes,
        projectiles,
        buildings,
        simMicros,
        hordes,
        soldierCount,
        soldierU32: new Uint32Array(buf, soldierOff, words),
        soldierF32: new Float32Array(buf, soldierOff, words),
        byteLength: buf.byteLength,
    };
}

export function encodeHello(mapId: string): ArrayBuffer {
    const name = new TextEncoder().encode(mapId);
    const buf = new ArrayBuffer(4 + name.length);
    const v = new DataView(buf);
    v.setUint8(0, MsgHello);
    v.setUint8(1, PROTOCOL_VERSION);
    v.setUint16(2, name.length, true);
    new Uint8Array(buf, 4).set(name);
    return buf;
}

export interface ConsoleOutput {
    error: boolean;
    text: string;
}

export function decodeConsoleOutput(buf: ArrayBuffer): ConsoleOutput | null {
    if (buf.byteLength < 8) return null;
    const v = new DataView(buf);
    if (v.getUint8(0) !== MsgConsoleOutput || v.getUint8(1) !== PROTOCOL_VERSION) return null;
    const len = Math.min(v.getUint32(4, true), buf.byteLength - 8);
    return { error: v.getUint8(2) === 1, text: new TextDecoder().decode(new Uint8Array(buf, 8, len)) };
}

/** Dev console line with context: cursor world position and selected hordes. */
export function encodeConsole(text: string, cursorX: number, cursorZ: number, selected: number[]): ArrayBuffer {
    const bytes = new TextEncoder().encode(text).subarray(0, 1024);
    const n = Math.min(selected.length, 256);
    const buf = new ArrayBuffer(16 + n * 4 + bytes.length);
    const v = new DataView(buf);
    v.setUint8(0, MsgCmdConsole);
    v.setUint8(1, PROTOCOL_VERSION);
    v.setUint16(2, n, true);
    v.setInt32(4, toFixedRaw(cursorX), true);
    v.setInt32(8, toFixedRaw(cursorZ), true);
    v.setUint16(12, bytes.length, true);
    for (let i = 0; i < n; i++) v.setUint32(16 + i * 4, selected[i], true);
    new Uint8Array(buf, 16 + n * 4).set(bytes);
    return buf;
}

/** World coordinates are sent as Q16.16 fixed-point, the simulation's native format. */
export function toFixedRaw(v: number): number {
    return Math.round(v * 65536) | 0;
}

export function encodeMove(x: number, z: number, hordeIds: number[]): ArrayBuffer {
    const n = Math.min(hordeIds.length, 256);
    const buf = new ArrayBuffer(12 + n * 4);
    const v = new DataView(buf);
    v.setUint8(0, MsgCmdMove);
    v.setUint8(1, PROTOCOL_VERSION);
    v.setUint16(2, n, true);
    v.setInt32(4, toFixedRaw(x), true);
    v.setInt32(8, toFixedRaw(z), true);
    for (let i = 0; i < n; i++) v.setUint32(12 + i * 4, hordeIds[i], true);
    return buf;
}

export function encodeAttack(targetHorde: number, hordeIds: number[]): ArrayBuffer {
    const n = Math.min(hordeIds.length, 256);
    const buf = new ArrayBuffer(8 + n * 4);
    const v = new DataView(buf);
    v.setUint8(0, MsgCmdAttack);
    v.setUint8(1, PROTOCOL_VERSION);
    v.setUint16(2, n, true);
    v.setUint32(4, targetHorde, true);
    for (let i = 0; i < n; i++) v.setUint32(8 + i * 4, hordeIds[i], true);
    return buf;
}

/** Attack an enemy building (same layout as encodeAttack). */
export function encodeSiege(targetBuilding: number, hordeIds: number[]): ArrayBuffer {
    const buf = encodeAttack(targetBuilding, hordeIds);
    new DataView(buf).setUint8(0, MsgCmdSiege);
    return buf;
}

/** Build building type `kind` on plot `plot` of fortress / outpost `host`. */
export function encodePlot(host: number, plot: number, kind: number): ArrayBuffer {
    const buf = new ArrayBuffer(12);
    const v = new DataView(buf);
    v.setUint8(0, MsgCmdPlot);
    v.setUint8(1, PROTOCOL_VERSION);
    v.setUint16(2, plot, true);
    v.setUint32(4, host, true);
    v.setUint16(8, kind, true);
    return buf;
}

/** Workers put up building `kind` at (x, z) facing `angle` (radians, simulation convention). */
export function encodeBuild(kind: number, x: number, z: number, angle: number, workers: number[]): ArrayBuffer {
    const n = Math.min(workers.length, 256);
    const buf = new ArrayBuffer(20 + n * 4);
    const v = new DataView(buf);
    v.setUint8(0, MsgCmdBuild);
    v.setUint8(1, PROTOCOL_VERSION);
    v.setUint16(2, n, true);
    v.setUint16(4, kind, true);
    v.setInt32(8, toFixedRaw(x), true);
    v.setInt32(12, toFixedRaw(z), true);
    v.setInt32(16, toFixedRaw(angle), true);
    for (let i = 0; i < n; i++) v.setUint32(20 + i * 4, workers[i], true);
    return buf;
}

/** Workers help build an unfinished building (same layout as encodeAttack). */
export function encodeHelp(building: number, workers: number[]): ArrayBuffer {
    const buf = encodeAttack(building, workers);
    new DataView(buf).setUint8(0, MsgCmdHelp);
    return buf;
}

export function encodeTrain(building: number, hordeKind: number): ArrayBuffer {
    const buf = new ArrayBuffer(8);
    const v = new DataView(buf);
    v.setUint8(0, MsgCmdTrain);
    v.setUint8(1, PROTOCOL_VERSION);
    v.setUint16(2, hordeKind, true);
    v.setUint32(4, building, true);
    return buf;
}

export function encodeResearch(building: number, upgradeKind: number): ArrayBuffer {
    const buf = encodeTrain(building, upgradeKind);
    new DataView(buf).setUint8(0, MsgCmdResearch);
    return buf;
}

export function encodeUntrain(building: number, index: number): ArrayBuffer {
    const buf = new ArrayBuffer(8);
    const v = new DataView(buf);
    v.setUint8(0, MsgCmdUntrain);
    v.setUint8(1, PROTOCOL_VERSION);
    v.setUint16(2, index, true);
    v.setUint32(4, building, true);
    return buf;
}

export function encodeRally(building: number, x: number, z: number): ArrayBuffer {
    const buf = new ArrayBuffer(16);
    const v = new DataView(buf);
    v.setUint8(0, MsgCmdRally);
    v.setUint8(1, PROTOCOL_VERSION);
    v.setUint32(4, building, true);
    v.setInt32(8, toFixedRaw(x), true);
    v.setInt32(12, toFixedRaw(z), true);
    return buf;
}

/** Switch toggle ability `ability` of a tower (fire arrows). */
export function encodeBuildingToggle(building: number, ability: number): ArrayBuffer {
    const buf = new ArrayBuffer(8);
    const v = new DataView(buf);
    v.setUint8(0, MsgCmdBToggle);
    v.setUint8(1, PROTOCOL_VERSION);
    v.setUint16(2, ability, true);
    v.setUint32(4, building, true);
    return buf;
}

/** Open (true) or close a gate of the local player. */
export function encodeGate(building: number, open: boolean): ArrayBuffer {
    const buf = new ArrayBuffer(12);
    const v = new DataView(buf);
    v.setUint8(0, MsgCmdGate);
    v.setUint8(1, PROTOCOL_VERSION);
    v.setUint32(4, building, true);
    v.setUint8(8, open ? 1 : 0);
    return buf;
}

export function encodeCancel(building: number): ArrayBuffer {
    const buf = new ArrayBuffer(8);
    const v = new DataView(buf);
    v.setUint8(0, MsgCmdCancel);
    v.setUint8(1, PROTOCOL_VERSION);
    v.setUint32(4, building, true);
    return buf;
}

/** Stop / hold: same layout as encodeAbility with another type byte. */
export function encodeHordeList(type: number, hordeIds: number[]): ArrayBuffer {
    const buf = encodeAbility(hordeIds);
    new DataView(buf).setUint8(0, type);
    return buf;
}

/** Attack-move / patrol: same layout as encodeMove with another type byte. */
export function encodeMoveLike(type: number, x: number, z: number, hordeIds: number[]): ArrayBuffer {
    const buf = encodeMove(x, z, hordeIds);
    new DataView(buf).setUint8(0, type);
    return buf;
}

/** Cast ability `kind` (index into ContentService.Abilities) at a point / on a horde (0 = none). */
export function encodeCast(kind: number, x: number, z: number, targetHorde: number, hordeIds: number[]): ArrayBuffer {
    const n = Math.min(hordeIds.length, 256);
    const buf = new ArrayBuffer(20 + n * 4);
    const v = new DataView(buf);
    v.setUint8(0, MsgCmdCast);
    v.setUint8(1, PROTOCOL_VERSION);
    v.setUint16(2, n, true);
    v.setUint16(4, kind, true);
    v.setInt32(8, toFixedRaw(x), true);
    v.setInt32(12, toFixedRaw(z), true);
    v.setUint32(16, targetHorde, true);
    for (let i = 0; i < n; i++) v.setUint32(20 + i * 4, hordeIds[i], true);
    return buf;
}

/** Toggle (MsgCmdToggle) or set autocast (MsgCmdAuto, on) of ability `kind`. */
export function encodeSwitch(type: number, kind: number, on: boolean | number, hordeIds: number[]): ArrayBuffer {
    const n = Math.min(hordeIds.length, 256);
    const buf = new ArrayBuffer(8 + n * 4);
    const v = new DataView(buf);
    v.setUint8(0, type);
    v.setUint8(1, PROTOCOL_VERSION);
    v.setUint16(2, n, true);
    v.setUint16(4, kind, true);
    v.setUint16(6, typeof on === "number" ? on : on ? 1 : 0, true); // autocast: on; mode abilities: the mode index
    for (let i = 0; i < n; i++) v.setUint32(8 + i * 4, hordeIds[i], true);
    return buf;
}

export function encodeAbility(hordeIds: number[]): ArrayBuffer {
    const n = Math.min(hordeIds.length, 256);
    const buf = new ArrayBuffer(4 + n * 4);
    const v = new DataView(buf);
    v.setUint8(0, MsgCmdAbility);
    v.setUint8(1, PROTOCOL_VERSION);
    v.setUint16(2, n, true);
    for (let i = 0; i < n; i++) v.setUint32(4 + i * 4, hordeIds[i], true);
    return buf;
}
