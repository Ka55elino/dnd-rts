import * as THREE from "three";
import { GLTFLoader } from "three/examples/jsm/loaders/GLTFLoader.js";

/** Camera distances (world units, before model scale) where LOD1 / LOD2 take over. */
const LOD_DISTANCES = [0, 140, 300];

interface StaticMeta {
    version: number;
    lods: { file: string; triangles: number }[];
    bounds: { min: [number, number, number]; max: [number, number, number] };
}

/**
 * A static model baked by tools/meshbake: up to three LOD scenes that share
 * materials and textures. `instance()` returns a cheap THREE.LOD clone
 * (geometry and materials are shared, never copied).
 */
export class StaticModel {
    private constructor(
        readonly id: string,
        readonly bounds: THREE.Box3,
        private readonly lods: THREE.Object3D[],
    ) {}

    /** Loads frontend/public/models/<id>/. */
    static async load(id: string, loader = new GLTFLoader()): Promise<StaticModel> {
        const base = `/models/${id}`;
        const res = await fetch(`${base}/model.json`);
        if (!res.ok) throw new Error(`model ${id}: ${res.url} → ${res.status}`);
        const meta = (await res.json()) as StaticMeta;
        if (meta.version !== 1) throw new Error(`model ${id}: unsupported version ${meta.version}`);

        const scenes = await Promise.all(meta.lods.map((l) => loader.loadAsync(`${base}/${l.file}`).then((g) => g.scene)));

        // Each LOD file carries its own copy of the materials and textures:
        // keep LOD0's and point the others at them by material name.
        const shared = new Map<string, THREE.Material>();
        scenes[0].traverse((o) => {
            if (o instanceof THREE.Mesh) for (const m of materials(o)) shared.set(m.name, m);
        });
        for (const scene of scenes.slice(1)) {
            scene.traverse((o) => {
                if (!(o instanceof THREE.Mesh)) return;
                const own = materials(o);
                const mapped = own.map((m) => shared.get(m.name) ?? m);
                for (const m of own) if (!mapped.includes(m)) disposeMaterial(m); // replaced duplicate
                o.material = Array.isArray(o.material) ? mapped : mapped[0];
            });
        }
        for (const scene of scenes) {
            scene.traverse((o) => {
                if (o instanceof THREE.Mesh) o.matrixAutoUpdate = false;
            });
            scene.updateMatrixWorld(true);
        }

        const { min, max } = meta.bounds;
        return new StaticModel(id, new THREE.Box3(new THREE.Vector3(...min), new THREE.Vector3(...max)), scenes);
    }

    /** Number of LOD levels. */
    get levels(): number {
        return this.lods.length;
    }

    /**
     * The meshes of one LOD level with their model-space transforms, for
     * drawing many copies with InstancedMesh (walls). Geometry and materials
     * are shared with the model.
     */
    parts(level: number): { geometry: THREE.BufferGeometry; material: THREE.Material | THREE.Material[]; matrix: THREE.Matrix4 }[] {
        const out: { geometry: THREE.BufferGeometry; material: THREE.Material | THREE.Material[]; matrix: THREE.Matrix4 }[] = [];
        this.lods[level]?.traverse((o) => {
            if (o instanceof THREE.Mesh) out.push({ geometry: o.geometry, material: o.material, matrix: o.matrixWorld.clone() });
        });
        return out;
    }

    /** Camera distance where LOD `level` starts (before scale). */
    static lodDistance(level: number): number {
        return LOD_DISTANCES[level] ?? Infinity;
    }

    /** A new LOD object for one placed instance. `scale` also scales LOD distances. */
    instance(scale = 1): THREE.LOD {
        const lod = new THREE.LOD();
        this.lods.forEach((scene, i) => lod.addLevel(scene.clone(true), LOD_DISTANCES[i] * scale));
        lod.scale.setScalar(scale);
        return lod;
    }

    dispose(): void {
        const geometries = new Set<THREE.BufferGeometry>();
        const mats = new Set<THREE.Material>();
        for (const scene of this.lods) {
            scene.traverse((o) => {
                if (!(o instanceof THREE.Mesh)) return;
                geometries.add(o.geometry);
                for (const m of materials(o)) mats.add(m);
            });
        }
        geometries.forEach((g) => g.dispose());
        mats.forEach(disposeMaterial);
    }
}

function materials(mesh: THREE.Mesh): THREE.Material[] {
    return Array.isArray(mesh.material) ? mesh.material : [mesh.material];
}

function disposeMaterial(m: THREE.Material): void {
    for (const value of Object.values(m)) if (value instanceof THREE.Texture) value.dispose();
    m.dispose();
}
