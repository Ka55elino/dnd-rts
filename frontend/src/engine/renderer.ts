import * as THREE from "three";

/** Max device pixel ratio. Keeps 4K / Retina screens from killing the GPU. */
export const MAX_PIXEL_RATIO = 1.5;

/**
 * Owns a WebGLRenderer attached to a container: keeps it sized to the container
 * and releases the WebGL context on dispose().
 */
export class RenderContext {
    readonly renderer: THREE.WebGLRenderer;
    private readonly observer: ResizeObserver;
    private onResizeCb: ((w: number, h: number) => void) | null = null;

    constructor(private readonly container: HTMLElement, opts: { antialias?: boolean } = {}) {
        this.renderer = new THREE.WebGLRenderer({ antialias: opts.antialias ?? true, powerPreference: "high-performance" });
        this.renderer.setPixelRatio(Math.min(window.devicePixelRatio, MAX_PIXEL_RATIO));
        this.renderer.domElement.style.display = "block";
        this.renderer.domElement.style.width = "100%";
        this.renderer.domElement.style.height = "100%";
        container.appendChild(this.renderer.domElement);

        this.observer = new ResizeObserver(() => this.resize());
        this.observer.observe(container);
    }

    /** Called with the new size whenever the container resizes (and once immediately). */
    onResize(cb: (w: number, h: number) => void): void {
        this.onResizeCb = cb;
        this.resize();
    }

    /** Releases GPU resources of a scene graph (geometries, materials, textures). */
    static disposeScene(scene: THREE.Object3D): void {
        scene.traverse((obj) => {
            const mesh = obj as THREE.Mesh;
            if (mesh.geometry) mesh.geometry.dispose();
            if (mesh.material) {
                const materials = Array.isArray(mesh.material) ? mesh.material : [mesh.material];
                for (const m of materials) {
                    for (const value of Object.values(m)) {
                        if (value instanceof THREE.Texture) value.dispose();
                    }
                    m.dispose();
                }
            }
            if (obj instanceof THREE.InstancedMesh) obj.dispose();
        });
        scene.clear();
    }

    dispose(): void {
        this.observer.disconnect();
        this.onResizeCb = null;
        this.renderer.dispose();
        this.renderer.forceContextLoss();
        this.renderer.domElement.remove();
    }

    private resize(): void {
        const { clientWidth: w, clientHeight: h } = this.container;
        if (w === 0 || h === 0) return;
        this.renderer.setSize(w, h, false);
        this.onResizeCb?.(w, h);
    }
}
