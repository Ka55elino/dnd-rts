import { Stream } from "@wailsio/runtime";
import {
    MsgConsoleOutput,
    decodeConsoleOutput,
    decodeGameSnapshot,
    encodeAttack,
    encodeConsole,
    encodeHello,
    encodeMove,
    encodeSiege,
    encodePlot,
    encodeCancel,
    encodeBuild,
    encodeHelp,
    encodeTrain,
    encodeResearch,
    encodeUntrain,
    encodeRally,
    encodeGate,
    encodeBuildingToggle,
    encodeHordeList,
    encodeMoveLike,
    encodeCast,
    encodeSwitch,
    MsgCmdToggle,
    MsgCmdAuto,
    MsgCmdStop,
    MsgCmdHold,
    MsgCmdAMove,
    MsgCmdPatrol,
    MsgCmdFireAt,
    type ConsoleOutput,
    type GameSnapshot,
} from "./protocol";

const STREAM_NAME = "game"; // internal/game.StreamName

/**
 * Connection to a game session in Go. Today it is a local Wails stream;
 * in a server build the same API runs over a real WebSocket.
 */
export class GameConnection {
    private socket: ReturnType<typeof Stream> | null;

    /** Dev console replies from the server. */
    onConsole: ((out: ConsoleOutput) => void) | null = null;

    constructor(mapId: string, onSnapshot: (snap: GameSnapshot, receivedAt: number) => void) {
        const socket = Stream(STREAM_NAME);
        socket.binaryType = "arraybuffer";
        socket.onopen = () => socket.send(encodeHello(mapId));
        socket.onmessage = (ev: MessageEvent) => {
            const buf = ev.data as ArrayBuffer;
            if (buf.byteLength === 0) return;
            if (new Uint8Array(buf, 0, 1)[0] === MsgConsoleOutput) {
                const out = decodeConsoleOutput(buf);
                if (out) this.onConsole?.(out);
                return;
            }
            const snap = decodeGameSnapshot(buf);
            if (snap) onSnapshot(snap, performance.now());
        };
        socket.onclose = () => console.info("game stream closed");
        this.socket = socket;
    }

    move(x: number, z: number, hordeIds: number[]): void {
        if (hordeIds.length === 0) return;
        this.send(encodeMove(x, z, hordeIds));
    }

    attack(targetHorde: number, hordeIds: number[]): void {
        if (hordeIds.length === 0) return;
        this.send(encodeAttack(targetHorde, hordeIds));
    }

    siege(targetBuilding: number, hordeIds: number[]): void {
        if (hordeIds.length === 0) return;
        this.send(encodeSiege(targetBuilding, hordeIds));
    }

    buildOnPlot(host: number, plot: number, kind: number): void {
        this.send(encodePlot(host, plot, kind));
    }

    build(kind: number, x: number, z: number, angle: number, workers: number[]): void {
        if (workers.length === 0) return;
        this.send(encodeBuild(kind, x, z, angle, workers));
    }

    helpBuild(building: number, workers: number[]): void {
        if (workers.length === 0) return;
        this.send(encodeHelp(building, workers));
    }

    train(building: number, hordeKind: number): void {
        this.send(encodeTrain(building, hordeKind));
    }

    research(building: number, upgradeKind: number): void {
        this.send(encodeResearch(building, upgradeKind));
    }

    untrain(building: number, index: number): void {
        this.send(encodeUntrain(building, index));
    }

    buildingToggle(building: number, ability: number): void {
        this.send(encodeBuildingToggle(building, ability));
    }

    gate(building: number, open: boolean): void {
        this.send(encodeGate(building, open));
    }

    rally(building: number, x: number, z: number): void {
        this.send(encodeRally(building, x, z));
    }

    cancelBuild(building: number): void {
        this.send(encodeCancel(building));
    }

    stop(hordeIds: number[]): void {
        if (hordeIds.length > 0) this.send(encodeHordeList(MsgCmdStop, hordeIds));
    }

    hold(hordeIds: number[]): void {
        if (hordeIds.length > 0) this.send(encodeHordeList(MsgCmdHold, hordeIds));
    }

    attackMove(x: number, z: number, hordeIds: number[]): void {
        if (hordeIds.length > 0) this.send(encodeMoveLike(MsgCmdAMove, x, z, hordeIds));
    }

    /** Ranged hordes shoot at the ground point until told otherwise. */
    attackGround(x: number, z: number, hordeIds: number[]): void {
        if (hordeIds.length > 0) this.send(encodeMoveLike(MsgCmdFireAt, x, z, hordeIds));
    }

    patrol(x: number, z: number, hordeIds: number[]): void {
        if (hordeIds.length > 0) this.send(encodeMoveLike(MsgCmdPatrol, x, z, hordeIds));
    }

    cast(kind: number, x: number, z: number, targetHorde: number, hordeIds: number[]): void {
        if (hordeIds.length > 0) this.send(encodeCast(kind, x, z, targetHorde, hordeIds));
    }

    /** Toggle an ability, or pick mode `mode` of a mode ability (an element). */
    toggle(kind: number, hordeIds: number[], mode = 0): void {
        if (hordeIds.length > 0) this.send(encodeSwitch(MsgCmdToggle, kind, mode, hordeIds));
    }

    autocast(kind: number, on: boolean, hordeIds: number[]): void {
        if (hordeIds.length > 0) this.send(encodeSwitch(MsgCmdAuto, kind, on, hordeIds));
    }

    console(text: string, cursorX: number, cursorZ: number, selected: number[]): void {
        this.send(encodeConsole(text, cursorX, cursorZ, selected));
    }

    close(): void {
        this.socket?.close();
        this.socket = null;
    }

    private send(buf: ArrayBuffer): void {
        if (this.socket?.readyState === 1) this.socket.send(buf);
    }
}
