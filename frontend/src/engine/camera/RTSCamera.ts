import * as THREE from "three";

export interface RTSCameraOptions {
    /** Half extents of the area the camera target may move in. */
    boundsX: number;
    boundsZ: number;
    /** Ground height lookup so the camera follows the terrain. */
    heightAt: (x: number, z: number) => number;
    minDistance?: number;
    maxDistance?: number;
}

const EDGE_MARGIN = 8; // px from the screen edge that triggers scrolling
const ZOOM_STEP = 0.0015; // per wheel delta unit
const ROTATE_SPEED = 1.8; // rad/s for Q/E
const DRAG_ROTATE_SPEED = 0.005; // rad per px for middle-mouse drag
const SMOOTHING = 12; // higher = snappier zoom / height follow

/**
 * BFME-style RTS camera: a target point on the ground, viewed from a distance
 * at a pitch that flattens as you zoom in.
 *
 * Controls: WASD / arrows / screen edges pan, wheel zooms, Q/E or middle-drag
 * rotates, Home (or Backspace) resets rotation and zoom.
 */
export class RTSCamera {
    readonly camera: THREE.PerspectiveCamera;

    private readonly target = new THREE.Vector2(0, 0); // x, z on the ground
    private groundY = 0;
    private yaw = 0;
    private distance: number;
    private targetDistance: number;
    private readonly minDistance: number;
    private readonly maxDistance: number;
    private readonly defaultDistance: number;

    private readonly keys = new Set<string>();
    private mouse = { x: -1, y: -1, inside: false };
    private dragging = false;
    private lastDragX = 0;

    private readonly lookAt = new THREE.Vector3();

    constructor(
        private readonly dom: HTMLElement,
        private readonly opts: RTSCameraOptions,
    ) {
        this.camera = new THREE.PerspectiveCamera(40, 1, 1, 3000);
        this.minDistance = opts.minDistance ?? 30;
        this.maxDistance = opts.maxDistance ?? 320;
        this.defaultDistance = (this.minDistance + this.maxDistance) * 0.45;
        this.distance = this.targetDistance = this.defaultDistance;

        window.addEventListener("keydown", this.onKeyDown);
        window.addEventListener("keyup", this.onKeyUp);
        window.addEventListener("blur", this.onBlur);
        dom.addEventListener("wheel", this.onWheel, { passive: false });
        dom.addEventListener("mousemove", this.onMouseMove);
        dom.addEventListener("mouseleave", this.onMouseLeave);
        dom.addEventListener("mousedown", this.onMouseDown);
        window.addEventListener("mouseup", this.onMouseUp);

        this.groundY = opts.heightAt(0, 0);
        this.apply();
    }

    /** Moves the camera target to a world position (e.g. the player's start). */
    focus(x: number, z: number): void {
        this.target.set(x, z);
        this.clampTarget();
        this.groundY = this.opts.heightAt(this.target.x, this.target.y);
        this.apply();
    }

    setAspect(aspect: number): void {
        this.camera.aspect = aspect;
        this.camera.updateProjectionMatrix();
    }

    update(dt: number): void {
        dt = Math.min(dt, 0.1); // avoid jumps after a stall

        // --- rotation
        if (this.keys.has("KeyQ")) this.yaw += ROTATE_SPEED * dt;
        if (this.keys.has("KeyE")) this.yaw -= ROTATE_SPEED * dt;

        // --- pan direction in camera space (x = right, y = forward)
        let px = 0;
        let py = 0;
        if (this.keys.has("KeyW") || this.keys.has("ArrowUp")) py += 1;
        if (this.keys.has("KeyS") || this.keys.has("ArrowDown")) py -= 1;
        if (this.keys.has("KeyD") || this.keys.has("ArrowRight")) px += 1;
        if (this.keys.has("KeyA") || this.keys.has("ArrowLeft")) px -= 1;

        if (this.mouse.inside && !this.dragging && document.hasFocus()) {
            const { clientWidth: w, clientHeight: h } = this.dom;
            if (this.mouse.x <= EDGE_MARGIN) px -= 1;
            if (this.mouse.x >= w - EDGE_MARGIN) px += 1;
            if (this.mouse.y <= EDGE_MARGIN) py += 1;
            if (this.mouse.y >= h - EDGE_MARGIN) py -= 1;
        }

        if (px !== 0 || py !== 0) {
            const len = Math.hypot(px, py);
            px /= len;
            py /= len;
            // Faster when zoomed out.
            const speed = this.distance * 1.1 * dt;
            const sin = Math.sin(this.yaw);
            const cos = Math.cos(this.yaw);
            // forward = (-sin, -cos), right = (cos, -sin)
            this.target.x += (px * cos - py * sin) * speed;
            this.target.y += (-px * sin - py * cos) * speed;
            this.clampTarget();
        }

        // --- smooth zoom and terrain following
        const k = 1 - Math.exp(-SMOOTHING * dt);
        this.distance += (this.targetDistance - this.distance) * k;
        const ground = this.opts.heightAt(this.target.x, this.target.y);
        this.groundY += (ground - this.groundY) * k;

        this.apply();
    }

    dispose(): void {
        window.removeEventListener("keydown", this.onKeyDown);
        window.removeEventListener("keyup", this.onKeyUp);
        window.removeEventListener("blur", this.onBlur);
        this.dom.removeEventListener("wheel", this.onWheel);
        this.dom.removeEventListener("mousemove", this.onMouseMove);
        this.dom.removeEventListener("mouseleave", this.onMouseLeave);
        this.dom.removeEventListener("mousedown", this.onMouseDown);
        window.removeEventListener("mouseup", this.onMouseUp);
    }

    private apply(): void {
        // Pitch: steep (top-down-ish) when zoomed out, flatter when zoomed in.
        const t = (this.distance - this.minDistance) / (this.maxDistance - this.minDistance);
        const pitch = THREE.MathUtils.degToRad(THREE.MathUtils.lerp(38, 58, t));

        const horizontal = this.distance * Math.cos(pitch);
        const vertical = this.distance * Math.sin(pitch);

        this.lookAt.set(this.target.x, this.groundY, this.target.y);
        this.camera.position.set(
            this.target.x + Math.sin(this.yaw) * horizontal,
            this.groundY + vertical,
            this.target.y + Math.cos(this.yaw) * horizontal,
        );
        this.camera.lookAt(this.lookAt);
    }

    private clampTarget(): void {
        this.target.x = THREE.MathUtils.clamp(this.target.x, -this.opts.boundsX, this.opts.boundsX);
        this.target.y = THREE.MathUtils.clamp(this.target.y, -this.opts.boundsZ, this.opts.boundsZ);
    }

    private onKeyDown = (e: KeyboardEvent): void => {
        if (isTyping(e)) return;
        this.keys.add(e.code);
        if (e.code === "Home" || e.code === "Backspace") {
            this.yaw = 0;
            this.targetDistance = this.defaultDistance;
        }
    };

    private onKeyUp = (e: KeyboardEvent): void => {
        this.keys.delete(e.code);
    };

    private onBlur = (): void => {
        this.keys.clear();
        this.dragging = false;
    };

    private onWheel = (e: WheelEvent): void => {
        e.preventDefault();
        this.targetDistance = THREE.MathUtils.clamp(
            this.targetDistance * Math.exp(e.deltaY * ZOOM_STEP),
            this.minDistance,
            this.maxDistance,
        );
    };

    private onMouseMove = (e: MouseEvent): void => {
        const rect = this.dom.getBoundingClientRect();
        this.mouse.x = e.clientX - rect.left;
        this.mouse.y = e.clientY - rect.top;
        this.mouse.inside = true;
        if (this.dragging) {
            this.yaw -= (e.clientX - this.lastDragX) * DRAG_ROTATE_SPEED;
            this.lastDragX = e.clientX;
        }
    };

    private onMouseLeave = (): void => {
        this.mouse.inside = false;
    };

    private onMouseDown = (e: MouseEvent): void => {
        if (e.button === 1) {
            e.preventDefault(); // no autoscroll cursor
            this.dragging = true;
            this.lastDragX = e.clientX;
        }
    };

    private onMouseUp = (e: MouseEvent): void => {
        if (e.button === 1) this.dragging = false;
    };
}

function isTyping(e: KeyboardEvent): boolean {
    const el = e.target as HTMLElement | null;
    return !!el && (el.tagName === "INPUT" || el.tagName === "TEXTAREA" || el.isContentEditable);
}
