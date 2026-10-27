import * as THREE from "three";
import { PLAYER_COLORS, type SoldiersView } from "../view/SoldiersView";

const LIFT = 3.2; // world units above the tallest soldier of the horde

interface Label {
    el: HTMLDivElement;
    text: HTMLSpanElement;
    fill: HTMLDivElement;
    lastHp: number;
    lastMax: number;
}

/**
 * HTML plates above each horde: "current HP / max HP" and a health bar.
 * Positioned every frame by projecting the horde's centre (average of its
 * soldiers' render positions) to the screen.
 */
export class HordeLabels {
    private readonly labels = new Map<number, Label>();
    private readonly tmp = new THREE.Vector3();
    // Per-horde accumulators, reused every frame.
    private readonly sum = new Map<number, { x: number; z: number; top: number; n: number }>();

    constructor(private readonly overlay: HTMLElement) {}

    update(camera: THREE.Camera, soldiers: SoldiersView, width: number, height: number): void {
        // Average soldier position per horde.
        this.sum.forEach((s) => (s.n = 0));
        const pos = soldiers.renderPos;
        const hordeOf = soldiers.hordeOf;
        for (let i = 0; i < soldiers.count; i++) {
            const id = hordeOf[i];
            let s = this.sum.get(id);
            if (!s) {
                s = { x: 0, z: 0, top: -Infinity, n: 0 };
                this.sum.set(id, s);
            }
            if (s.n === 0) {
                s.x = 0;
                s.z = 0;
                s.top = -Infinity;
            }
            s.x += pos[i * 3];
            s.z += pos[i * 3 + 2];
            s.top = Math.max(s.top, pos[i * 3 + 1]);
            s.n++;
        }

        // Remove plates of hordes that no longer exist.
        for (const [id, label] of this.labels) {
            if (!soldiers.hordes.has(id)) {
                label.el.remove();
                this.labels.delete(id);
                this.sum.delete(id);
            }
        }

        for (const [id, h] of soldiers.hordes) {
            const s = this.sum.get(id);
            if (!s || s.n === 0) continue;
            const label = this.labels.get(id) ?? this.create(id, h.owner);

            this.tmp.set(s.x / s.n, s.top + LIFT, s.z / s.n).project(camera);
            const visible = this.tmp.z < 1 && Math.abs(this.tmp.x) < 1.1 && Math.abs(this.tmp.y) < 1.1;
            label.el.hidden = !visible;
            if (!visible) continue;

            const sx = (this.tmp.x + 1) * 0.5 * width;
            const sy = (1 - this.tmp.y) * 0.5 * height;
            label.el.style.transform = `translate(${sx.toFixed(1)}px, ${sy.toFixed(1)}px) translate(-50%, -100%)`;

            if (h.hp !== label.lastHp || h.maxHp !== label.lastMax) {
                label.lastHp = h.hp;
                label.lastMax = h.maxHp;
                label.text.textContent = `${h.hp}/${h.maxHp}`;
                label.fill.style.width = `${h.maxHp > 0 ? (100 * h.hp) / h.maxHp : 0}%`;
            }
        }
    }

    dispose(): void {
        for (const label of this.labels.values()) label.el.remove();
        this.labels.clear();
    }

    private create(id: number, owner: number): Label {
        const el = document.createElement("div");
        el.className = "horde-label";
        const color = `#${new THREE.Color(PLAYER_COLORS[owner % PLAYER_COLORS.length]).getHexString()}`;
        el.innerHTML = `
            <span class="horde-label__text"></span>
            <div class="horde-label__bar"><div class="horde-label__fill" style="background:${color}"></div></div>
        `;
        this.overlay.appendChild(el);
        const label: Label = {
            el,
            text: el.querySelector(".horde-label__text")!,
            fill: el.querySelector(".horde-label__fill")!,
            lastHp: -1,
            lastMax: -1,
        };
        this.labels.set(id, label);
        return label;
    }
}
