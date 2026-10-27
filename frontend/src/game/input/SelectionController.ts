import * as THREE from "three";
import type { SoldiersView } from "../view/SoldiersView";
import { NEUTRAL } from "../../net/protocol";

const DRAG_THRESHOLD = 5; // px before a click becomes a box selection
const CLICK_RADIUS = 16; // px around a soldier that counts as clicking it

export interface SelectionDeps {
    dom: HTMLElement; // canvas
    overlay: HTMLElement; // where the selection box is drawn
    camera: () => THREE.Camera;
    soldiers: SoldiersView;
    terrain: THREE.Object3D;
    localPlayer: () => number; // can change (dev console: control)
    onMove: (x: number, y: number, z: number, hordeIds: number[]) => void;
    onAttack: (targetHorde: number, at: THREE.Vector3, hordeIds: number[]) => void;
    /** Building under the cursor ray (ground = terrain hit, for footprint picking). */
    pickBuilding: (ray: THREE.Raycaster, ground: THREE.Vector3 | null) => { id: number; owner: number; point: THREE.Vector3 } | null;
    onSiege: (targetBuilding: number, at: THREE.Vector3, hordeIds: number[]) => void;
    /** Mouse down on the canvas before normal handling; return true to swallow it (placement mode). */
    intercept: (e: MouseEvent) => boolean;
    /** Right click on an own building; return true when handled (e.g. workers help build). */
    onOwnBuilding: (building: number, hordeIds: number[], point: THREE.Vector3) => boolean;
    /** Right click on the ground with no hordes selected (rally point of the shown building). */
    onRightClickIdle: (ground: THREE.Vector3) => void;
    /** Left click that hit no own soldier (buildings, plots). */
    onClickWorld: (ray: THREE.Raycaster, ground: THREE.Vector3 | null) => void;
    onChange: (selected: ReadonlySet<number>) => void;
}

/**
 * Left click / drag selects own hordes (Shift adds). Right click: on an enemy
 * soldier = attack its horde, on an enemy building = attack the building,
 * otherwise move there.
 */
export class SelectionController {
    readonly selected = new Set<number>();

    private dragStart: { x: number; y: number } | null = null;
    private dragging = false;
    private readonly box: HTMLDivElement;
    private readonly raycaster = new THREE.Raycaster();
    private readonly ndc = new THREE.Vector2();
    private readonly tmp = new THREE.Vector3();

    constructor(private readonly d: SelectionDeps) {
        this.box = document.createElement("div");
        this.box.className = "selection-box";
        this.box.hidden = true;
        d.overlay.appendChild(this.box);

        d.dom.addEventListener("mousedown", this.onMouseDown);
        window.addEventListener("mousemove", this.onMouseMove);
        window.addEventListener("mouseup", this.onMouseUp);
    }

    /**
     * What is under the mouse for a targeted order (command card): an enemy
     * horde, an enemy building, and the ground point.
     */
    pickTarget(e: { clientX: number; clientY: number }): {
        allyHorde: number;
        enemyHorde: number;
        /** Soldier index (renderPos / status) of the enemy under the mouse, -1 = none. */
        enemyIndex: number;
        enemyAt: THREE.Vector3 | null;
        building: { id: number; owner: number; point: THREE.Vector3 } | null;
        ground: THREE.Vector3 | null;
    } {
        const p = this.local(e);
        const enemy = this.soldierAt(p, (owner) => owner !== this.d.localPlayer());
        const ally = this.soldierAt(p, (owner) => owner === this.d.localPlayer());
        let enemyAt: THREE.Vector3 | null = null;
        if (enemy >= 0) {
            const pos = this.d.soldiers.renderPos;
            enemyAt = new THREE.Vector3(pos[enemy * 3], pos[enemy * 3 + 1], pos[enemy * 3 + 2]);
        }
        const ray = this.rayAt(p);
        const ground = ray.intersectObject(this.d.terrain, false)[0]?.point ?? null;
        const building = this.d.pickBuilding(ray, ground);
        return {
            allyHorde: ally >= 0 ? this.d.soldiers.hordeOf[ally] : 0,
            enemyHorde: enemy >= 0 ? this.d.soldiers.hordeOf[enemy] : 0,
            enemyIndex: enemy,
            enemyAt,
            building,
            ground,
        };
    }

    /** Replaces the selection (HUD portraits). */
    set(ids: number[]): void {
        this.selected.clear();
        for (const id of ids) this.selected.add(id);
        this.d.onChange(this.selected);
    }

    clear(): void {
        if (this.selected.size === 0) return;
        this.selected.clear();
        this.d.onChange(this.selected);
    }

    dispose(): void {
        this.d.dom.removeEventListener("mousedown", this.onMouseDown);
        window.removeEventListener("mousemove", this.onMouseMove);
        window.removeEventListener("mouseup", this.onMouseUp);
        this.box.remove();
    }

    private onMouseDown = (e: MouseEvent): void => {
        if (this.d.intercept(e)) return;
        if (e.button === 0) {
            const p = this.local(e);
            this.dragStart = p;
            this.dragging = false;
        } else if (e.button === 2) {
            this.issueMove(e);
        }
    };

    private onMouseMove = (e: MouseEvent): void => {
        if (!this.dragStart) return;
        const p = this.local(e);
        if (!this.dragging && Math.hypot(p.x - this.dragStart.x, p.y - this.dragStart.y) > DRAG_THRESHOLD) {
            this.dragging = true;
            this.box.hidden = false;
        }
        if (this.dragging) {
            const r = rect(this.dragStart, p);
            Object.assign(this.box.style, {
                left: `${r.x0}px`,
                top: `${r.y0}px`,
                width: `${r.x1 - r.x0}px`,
                height: `${r.y1 - r.y0}px`,
            });
        }
    };

    private onMouseUp = (e: MouseEvent): void => {
        if (e.button !== 0 || !this.dragStart) return;
        const p = this.local(e);
        const additive = e.shiftKey;

        if (this.dragging) {
            this.selectInRect(rect(this.dragStart, p), additive);
        } else {
            this.selectAt(p, additive);
        }
        this.dragStart = null;
        this.dragging = false;
        this.box.hidden = true;
    };

    /** Index of the soldier nearest to screen point p (within CLICK_RADIUS) whose owner passes `filter`, or -1. */
    private soldierAt(p: { x: number; y: number }, filter: (owner: number) => boolean): number {
        let best = -1;
        let bestDist = CLICK_RADIUS;
        this.forEachSoldierOnScreen(filter, (i, sx, sy) => {
            const dist = Math.hypot(sx - p.x, sy - p.y);
            if (dist < bestDist) {
                bestDist = dist;
                best = i;
            }
        });
        return best;
    }

    private selectAt(p: { x: number; y: number }, additive: boolean): void {
        const { soldiers } = this.d;
        const best = this.soldierAt(p, (owner) => owner === this.d.localPlayer());

        if (!additive) this.selected.clear();
        if (best >= 0) {
            const horde = soldiers.hordeOf[best];
            if (additive && this.selected.has(horde)) this.selected.delete(horde);
            else this.selected.add(horde);
        }
        this.d.onChange(this.selected);
        if (best < 0 && !additive) {
            const ray = this.rayAt(p);
            this.d.onClickWorld(ray, ray.intersectObject(this.d.terrain, false)[0]?.point ?? null);
        }
    }

    private selectInRect(r: Rect, additive: boolean): void {
        if (!additive) this.selected.clear();
        this.forEachSoldierOnScreen((owner) => owner === this.d.localPlayer(), (i, sx, sy) => {
            if (sx >= r.x0 && sx <= r.x1 && sy >= r.y0 && sy <= r.y1) {
                this.selected.add(this.d.soldiers.hordeOf[i]);
            }
        });
        this.d.onChange(this.selected);
    }

    private issueMove(e: MouseEvent): void {
        const p = this.local(e);
        if (this.selected.size === 0) {
            const hit = this.rayAt(p).intersectObject(this.d.terrain, false)[0];
            if (hit) this.d.onRightClickIdle(hit.point);
            return;
        }

        // Right click on an enemy soldier = attack its horde.
        const enemy = this.soldierAt(p, (owner) => owner !== this.d.localPlayer());
        if (enemy >= 0) {
            const pos = this.d.soldiers.renderPos;
            this.tmp.set(pos[enemy * 3], pos[enemy * 3 + 1], pos[enemy * 3 + 2]);
            this.d.onAttack(this.d.soldiers.hordeOf[enemy], this.tmp.clone(), [...this.selected]);
            return;
        }

        this.rayAt(p);
        const hit = this.raycaster.intersectObject(this.d.terrain, false)[0];
        const building = this.d.pickBuilding(this.raycaster, hit?.point ?? null);
        if (building && building.owner !== this.d.localPlayer() && building.owner !== NEUTRAL) {
            this.d.onSiege(building.id, building.point, [...this.selected]);
            return;
        }
        if (building && building.owner === this.d.localPlayer() && this.d.onOwnBuilding(building.id, [...this.selected], building.point)) return;
        if (!hit) return;
        this.d.onMove(hit.point.x, hit.point.y, hit.point.z, [...this.selected]);
    }

    /** Projects every soldier whose owner passes `filter` to screen pixels. */
    private forEachSoldierOnScreen(
        filter: (owner: number) => boolean,
        fn: (index: number, sx: number, sy: number) => void,
    ): void {
        const { soldiers } = this.d;
        const camera = this.d.camera();
        const { clientWidth: w, clientHeight: h } = this.d.dom;
        const pos = soldiers.renderPos;

        for (let i = 0; i < soldiers.count; i++) {
            const horde = soldiers.hordes.get(soldiers.hordeOf[i]);
            if (!horde || !filter(horde.owner)) continue;
            // Aim at the soldier's chest, not its feet.
            this.tmp.set(pos[i * 3], pos[i * 3 + 1] + 0.9, pos[i * 3 + 2]).project(camera);
            if (this.tmp.z > 1) continue; // behind the camera
            fn(i, (this.tmp.x + 1) * 0.5 * w, (1 - this.tmp.y) * 0.5 * h);
        }
    }

    private rayAt(p: { x: number; y: number }): THREE.Raycaster {
        const { clientWidth: w, clientHeight: h } = this.d.dom;
        this.ndc.set((p.x / w) * 2 - 1, -(p.y / h) * 2 + 1);
        this.raycaster.setFromCamera(this.ndc, this.d.camera());
        return this.raycaster;
    }

    private local(e: { clientX: number; clientY: number }): { x: number; y: number } {
        const r = this.d.dom.getBoundingClientRect();
        return { x: e.clientX - r.left, y: e.clientY - r.top };
    }
}

interface Rect {
    x0: number;
    y0: number;
    x1: number;
    y1: number;
}

function rect(a: { x: number; y: number }, b: { x: number; y: number }): Rect {
    return { x0: Math.min(a.x, b.x), y0: Math.min(a.y, b.y), x1: Math.max(a.x, b.x), y1: Math.max(a.y, b.y) };
}
