import * as THREE from "three";

/** A lighting setup: sky / fill light, the sun, the background and exposure. */
export interface LightPreset {
    id: string;
    label: string;
    /** Background and fog color. */
    sky: number;
    /** Hemisphere fill: light from the sky, light bounced from the ground. */
    hemiSky: number;
    hemiGround: number;
    hemiIntensity: number;
    sunColor: number;
    sunIntensity: number;
    /** Direction from the ground toward the sun (normalized on use). */
    sunDir: [number, number, number];
    /** Tone-mapping exposure. */
    exposure: number;
}

export const LIGHT_PRESETS: LightPreset[] = [
    {
        id: "day",
        label: "Day",
        sky: 0x9fb4c8,
        hemiSky: 0xcfe0ff,
        hemiGround: 0x5a4a32,
        hemiIntensity: 1.1,
        sunColor: 0xfff0d8,
        sunIntensity: 2.6,
        sunDir: [-0.45, 0.8, 0.4],
        exposure: 1.0,
    },
    {
        id: "golden",
        label: "Evening",
        sky: 0xd8a878,
        hemiSky: 0xffd2a0,
        hemiGround: 0x4a3420,
        hemiIntensity: 0.8,
        sunColor: 0xffb070,
        sunIntensity: 2.8,
        sunDir: [-0.8, 0.35, 0.35],
        exposure: 1.05,
    },
    {
        id: "overcast",
        label: "Overcast",
        sky: 0x8d949c,
        hemiSky: 0xdde4ee,
        hemiGround: 0x4c4a44,
        hemiIntensity: 1.9,
        sunColor: 0xf0f0f0,
        sunIntensity: 0.8,
        sunDir: [-0.3, 0.9, 0.3],
        exposure: 1.0,
    },
    {
        id: "studio",
        label: "Studio",
        sky: 0x26303c,
        hemiSky: 0xdfe8ff,
        hemiGround: 0x4a3e2a,
        hemiIntensity: 1.2,
        sunColor: 0xfff1dc,
        sunIntensity: 2.2,
        sunDir: [0.45, 0.75, 0.6],
        exposure: 1.0,
    },
];

export function lightPreset(id: string): LightPreset {
    return LIGHT_PRESETS.find((p) => p.id === id) ?? LIGHT_PRESETS[0];
}

/**
 * Hemisphere fill + a sun with a shadow map. The shadow covers a square of `area` meters around
 * the point given to follow() (the camera target), so its resolution stays useful at any map size.
 */
export class LightRig {
    readonly hemi = new THREE.HemisphereLight();
    readonly sun = new THREE.DirectionalLight();
    readonly objects: THREE.Object3D[];
    private preset: LightPreset = LIGHT_PRESETS[0];
    private readonly dir = new THREE.Vector3();
    private readonly focus = new THREE.Vector3();

    constructor(
        private area = 60,
        mapSize = 2048,
    ) {
        this.sun.castShadow = true;
        this.sun.shadow.mapSize.set(mapSize, mapSize);
        this.sun.shadow.bias = -0.0004;
        this.sun.shadow.normalBias = 0.03;
        this.setArea(area);
        this.objects = [this.hemi, this.sun, this.sun.target];
        this.apply(this.preset);
    }

    get current(): LightPreset {
        return this.preset;
    }

    /** Applies a preset; also sets the scene background / fog color and the renderer exposure when given. */
    apply(p: LightPreset, scene?: THREE.Scene, renderer?: THREE.WebGLRenderer): void {
        this.preset = p;
        this.hemi.color.set(p.hemiSky);
        this.hemi.groundColor.set(p.hemiGround);
        this.hemi.intensity = p.hemiIntensity;
        this.sun.color.set(p.sunColor);
        this.sun.intensity = p.sunIntensity;
        this.dir.set(...p.sunDir).normalize();
        if (scene) {
            if (scene.background instanceof THREE.Color) scene.background.set(p.sky);
            if (scene.fog) scene.fog.color.set(p.sky);
        }
        if (renderer) renderer.toneMappingExposure = p.exposure;
        this.follow(this.focus);
    }

    /** Size of the shadowed square (meters). Smaller = sharper shadows. */
    setArea(area: number): void {
        this.area = area;
        const c = this.sun.shadow.camera;
        c.left = -area / 2;
        c.right = area / 2;
        c.top = area / 2;
        c.bottom = -area / 2;
        c.near = 1;
        c.far = area * 4;
        c.updateProjectionMatrix();
    }

    set shadows(on: boolean) {
        this.sun.castShadow = on;
    }

    /** Keeps the sun (and its shadow box) centered on a point, e.g. the camera target. */
    follow(p: THREE.Vector3): void {
        this.focus.copy(p);
        // Snap to shadow-map texels so the shadows don't shimmer while the camera moves.
        const texel = this.area / this.sun.shadow.mapSize.x;
        const x = Math.round(p.x / texel) * texel;
        const z = Math.round(p.z / texel) * texel;
        this.sun.target.position.set(x, p.y, z);
        this.sun.position.set(x, p.y, z).addScaledVector(this.dir, this.area * 2);
        this.sun.target.updateMatrixWorld();
    }
}

/** Renderer settings that go with LightRig: soft shadow maps and filmic tone mapping. */
export function setupRenderer(renderer: THREE.WebGLRenderer, shadows = true): void {
    renderer.shadowMap.enabled = shadows;
    renderer.shadowMap.type = THREE.PCFSoftShadowMap;
    renderer.toneMapping = THREE.ACESFilmicToneMapping;
    renderer.outputColorSpace = THREE.SRGBColorSpace;
}
