import * as THREE from "three";
import { NEUTRAL, type BuildingSnap, type HordeSnap } from "../../net/protocol";
import { PLAYER_COLORS } from "../view/SoldiersView";

const SIZE = 256; // canvas pixels (drawn into a round frame)
const NEUTRAL_COLOR = "#cfc6b0";

export interface MinimapDeps {
    parent: HTMLElement; // the round frame
    mapWidth: number;
    mapHeight: number;
    heightAt: (x: number, z: number) => number;
    maxHeight: number;
    /** Blocked terrain cells (cliffs), drawn darker. */
    nav: { cols: number; rows: number; cellSize: number; blocked: Uint8Array } | null;
    camera: () => THREE.PerspectiveCamera;
    onFocus: (x: number, z: number) => void;
    /** Right click: order the selection to this ground point. */
    onCommand: (x: number, z: number) => void;
}

/**
 * Round minimap: shaded terrain (rendered once), buildings and hordes in
 * player colors, and the camera's view on the ground. Left click / drag moves
 * the camera, right click sends the selection there.
 */
export class Minimap {
    private readonly canvas: HTMLCanvasElement;
    private readonly ctx: CanvasRenderingContext2D;
    private readonly base: HTMLCanvasElement;
    private readonly ray = new THREE.Raycaster();
    private readonly plane = new THREE.Plane(new THREE.Vector3(0, 1, 0), 0);
    private readonly tmp = new THREE.Vector3();
    private dragging = false;

    constructor(private readonly d: MinimapDeps) {
        this.canvas = document.createElement("canvas");
        this.canvas.width = this.canvas.height = SIZE;
        this.canvas.className = "hud-minimap__canvas";
        this.ctx = this.canvas.getContext("2d")!;
        this.base = this.renderBase();
        d.parent.appendChild(this.canvas);
        this.canvas.addEventListener("mousedown", this.onDown);
        window.addEventListener("mousemove", this.onMove);
        window.addEventListener("mouseup", this.onUp);
        this.canvas.addEventListener("contextmenu", (e) => e.preventDefault());
    }

    /** Redraws units, buildings and the camera view. */
    draw(hordes: Iterable<HordeSnap>, buildings: Iterable<BuildingSnap>, selected: ReadonlySet<number>): void {
        const c = this.ctx;
        c.drawImage(this.base, 0, 0);

        for (const b of buildings) {
            const [x, y] = this.toMap(b.x, b.z);
            c.fillStyle = color(b.owner);
            c.strokeStyle = "rgba(0,0,0,0.75)";
            c.lineWidth = 1.5;
            c.beginPath();
            c.rect(x - 4, y - 4, 8, 8);
            c.fill();
            c.stroke();
        }
        for (const h of hordes) {
            const [x, y] = this.toMap(h.x, h.z);
            const r = Math.max(2, Math.min(4, Math.sqrt(h.alive)));
            c.fillStyle = color(h.owner);
            c.beginPath();
            c.arc(x, y, r, 0, Math.PI * 2);
            c.fill();
            if (selected.has(h.id)) {
                c.strokeStyle = "#fff";
                c.lineWidth = 1.5;
                c.stroke();
            }
        }
        this.drawView();
    }

    dispose(): void {
        window.removeEventListener("mousemove", this.onMove);
        window.removeEventListener("mouseup", this.onUp);
        this.canvas.remove();
    }

    /** The camera's footprint: screen corners projected onto the ground plane. */
    private drawView(): void {
        const cam = this.d.camera();
        const pts: [number, number][] = [];
        for (const [nx, ny] of [[-1, 1], [1, 1], [1, -1], [-1, -1]]) {
            this.ray.setFromCamera(new THREE.Vector2(nx, ny), cam);
            let hit = this.ray.ray.intersectPlane(this.plane, this.tmp);
            if (!hit) {
                // Looking above the horizon: clamp far along the ray.
                hit = this.ray.ray.at(1500, this.tmp);
            }
            pts.push(this.toMap(hit.x, hit.z));
        }
        const c = this.ctx;
        c.strokeStyle = "rgba(255, 244, 214, 0.95)";
        c.lineWidth = 1.5;
        c.beginPath();
        pts.forEach(([x, y], i) => (i === 0 ? c.moveTo(x, y) : c.lineTo(x, y)));
        c.closePath();
        c.stroke();
    }

    private renderBase(): HTMLCanvasElement {
        const base = document.createElement("canvas");
        base.width = base.height = SIZE;
        const c = base.getContext("2d")!;
        const img = c.createImageData(SIZE, SIZE);
        const { mapWidth: w, mapHeight: h, maxHeight } = this.d;
        const step = w / SIZE;
        const light = new THREE.Vector3(-0.6, 0.75, -0.5).normalize();
        for (let py = 0; py < SIZE; py++) {
            for (let px = 0; px < SIZE; px++) {
                const x = (px + 0.5) * step - w / 2;
                const z = ((py + 0.5) * h) / SIZE - h / 2;
                const y = this.d.heightAt(x, z);
                const dx = this.d.heightAt(x + step, z) - this.d.heightAt(x - step, z);
                const dz = this.d.heightAt(x, z + step) - this.d.heightAt(x, z - step);
                const n = new THREE.Vector3(-dx, 2 * step, -dz).normalize();
                const shade = 0.55 + 0.6 * Math.max(0, n.dot(light));
                const t = Math.min(1, Math.max(0, y / maxHeight));
                // Low meadows → dry grass → rock.
                let r = 70 + 70 * t, g = 104 + 40 * t, b = 52 + 40 * t;
                if (this.blocked(x, z)) {
                    r = 96; g = 86; b = 74;
                }
                const i = (py * SIZE + px) * 4;
                img.data[i] = r * shade;
                img.data[i + 1] = g * shade;
                img.data[i + 2] = b * shade;
                img.data[i + 3] = 255;
            }
        }
        c.putImageData(img, 0, 0);
        // Vignette towards the frame.
        const grad = c.createRadialGradient(SIZE / 2, SIZE / 2, SIZE * 0.3, SIZE / 2, SIZE / 2, SIZE * 0.52);
        grad.addColorStop(0, "rgba(0,0,0,0)");
        grad.addColorStop(1, "rgba(0,0,0,0.45)");
        c.fillStyle = grad;
        c.fillRect(0, 0, SIZE, SIZE);
        return base;
    }

    private blocked(x: number, z: number): boolean {
        const nav = this.d.nav;
        if (!nav) return false;
        const col = Math.floor((x + this.d.mapWidth / 2) / nav.cellSize);
        const row = Math.floor((z + this.d.mapHeight / 2) / nav.cellSize);
        if (col < 0 || row < 0 || col >= nav.cols || row >= nav.rows) return true;
        return nav.blocked[row * nav.cols + col] === 1;
    }

    private toMap(x: number, z: number): [number, number] {
        return [((x + this.d.mapWidth / 2) / this.d.mapWidth) * SIZE, ((z + this.d.mapHeight / 2) / this.d.mapHeight) * SIZE];
    }

    private toWorld(e: MouseEvent): [number, number] {
        const r = this.canvas.getBoundingClientRect();
        const u = (e.clientX - r.left) / r.width;
        const v = (e.clientY - r.top) / r.height;
        return [(u - 0.5) * this.d.mapWidth, (v - 0.5) * this.d.mapHeight];
    }

    private onDown = (e: MouseEvent): void => {
        e.stopPropagation();
        e.preventDefault();
        const [x, z] = this.toWorld(e);
        if (e.button === 2) {
            this.d.onCommand(x, z);
            return;
        }
        if (e.button !== 0) return;
        this.dragging = true;
        this.d.onFocus(x, z);
    };

    private onMove = (e: MouseEvent): void => {
        if (!this.dragging) return;
        const [x, z] = this.toWorld(e);
        this.d.onFocus(x, z);
    };

    private onUp = (): void => {
        this.dragging = false;
    };
}

function color(owner: number): string {
    return owner === NEUTRAL ? NEUTRAL_COLOR : `#${new THREE.Color(PLAYER_COLORS[owner % PLAYER_COLORS.length]).getHexString()}`;
}
