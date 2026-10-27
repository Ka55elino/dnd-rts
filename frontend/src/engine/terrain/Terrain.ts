import * as THREE from "three";

/** Heightfield as received from Go (ContentService.Terrain). */
export interface TerrainInfo {
    width: number;
    height: number;
    cols: number;
    rows: number;
    cellSize: number;
    maxHeight: number;
    /** Little-endian uint16 samples, row-major (decoded from base64). */
    samples: Uint16Array;
}

const GRASS_LOW = new THREE.Color(0x3f5a2c);
const GRASS_HIGH = new THREE.Color(0x6b7a3a);
const DIRT = new THREE.Color(0x6e5a3c);
const ROCK = new THREE.Color(0x77736b);
const BLOCKED_TINT = new THREE.Color(0xd03030);

/** Walkability grid from Go (ContentService.NavGrid), for the debug overlay. */
export interface NavOverlay {
    cols: number;
    rows: number;
    cellSize: number;
    blocked: Uint8Array; // 1 = blocked, row-major
}

/** Decodes a base64 blob produced by Go's []byte JSON encoding. */
export function decodeBase64(base64: string): Uint8Array {
    const bin = atob(base64);
    const bytes = new Uint8Array(bin.length);
    for (let i = 0; i < bin.length; i++) bytes[i] = bin.charCodeAt(i);
    return bytes;
}

/** Decodes heightfield samples (little-endian uint16; matches all target platforms). */
export function decodeSamples(base64: string): Uint16Array {
    const bytes = decodeBase64(base64);
    return new Uint16Array(bytes.buffer, 0, bytes.length / 2);
}

/**
 * Terrain mesh built from a heightfield. World layout matches Go:
 * the map is centered at (0, 0), sim Y → world Z.
 */
export class Terrain {
    readonly mesh: THREE.Mesh;
    private readonly heights: Float32Array;
    private readonly baseColors: Float32Array;

    constructor(readonly info: TerrainInfo) {
        const { cols, rows, cellSize, maxHeight, samples, width, height } = info;

        this.heights = new Float32Array(cols * rows);
        for (let i = 0; i < samples.length; i++) this.heights[i] = (samples[i] / 65535) * maxHeight;

        const positions = new Float32Array(cols * rows * 3);
        for (let r = 0, i = 0; r < rows; r++) {
            for (let c = 0; c < cols; c++, i++) {
                positions[i * 3] = c * cellSize - width / 2;
                positions[i * 3 + 1] = this.heights[i];
                positions[i * 3 + 2] = r * cellSize - height / 2;
            }
        }

        const indices = new Uint32Array((cols - 1) * (rows - 1) * 6);
        for (let r = 0, k = 0; r < rows - 1; r++) {
            for (let c = 0; c < cols - 1; c++) {
                const a = r * cols + c;
                const b = a + 1;
                const d = a + cols;
                const e = d + 1;
                // Counter-clockwise seen from above (+Y).
                indices[k++] = a; indices[k++] = d; indices[k++] = b;
                indices[k++] = b; indices[k++] = d; indices[k++] = e;
            }
        }

        const geometry = new THREE.BufferGeometry();
        geometry.setAttribute("position", new THREE.BufferAttribute(positions, 3));
        geometry.setIndex(new THREE.BufferAttribute(indices, 1));
        geometry.computeVertexNormals();
        this.baseColors = this.vertexColors(geometry);
        geometry.setAttribute("color", new THREE.BufferAttribute(this.baseColors.slice(), 3));
        geometry.computeBoundingSphere();

        this.mesh = new THREE.Mesh(geometry, new THREE.MeshLambertMaterial({ vertexColors: true }));
        this.mesh.receiveShadow = true;
    }

    /** Bilinear height at a world position (same formula as Go's HeightAt). */
    heightAt(x: number, z: number): number {
        const { cols, rows, cellSize, width, height } = this.info;
        const gx = (x + width / 2) / cellSize;
        const gz = (z + height / 2) / cellSize;
        const c = Math.floor(gx);
        const r = Math.floor(gz);
        const tx = gx - c;
        const tz = gz - r;
        const h = (cc: number, rr: number) =>
            this.heights[Math.max(0, Math.min(rr, rows - 1)) * cols + Math.max(0, Math.min(cc, cols - 1))];
        const top = h(c, r) + (h(c + 1, r) - h(c, r)) * tx;
        const bottom = h(c, r + 1) + (h(c + 1, r + 1) - h(c, r + 1)) * tx;
        return top + (bottom - top) * tz;
    }

    /** Tints vertices inside blocked nav cells red; pass null to restore normal colours. */
    setNavOverlay(nav: NavOverlay | null): void {
        const attr = this.mesh.geometry.getAttribute("color") as THREE.BufferAttribute;
        const colors = attr.array as Float32Array;
        colors.set(this.baseColors);

        if (nav) {
            const { cols, rows, cellSize, width, height } = this.info;
            const c = new THREE.Color();
            for (let r = 0, i = 0; r < rows; r++) {
                for (let col = 0; col < cols; col++, i++) {
                    const x = col * cellSize - width / 2;
                    const z = r * cellSize - height / 2;
                    const nc = Math.min(Math.floor((x + width / 2) / nav.cellSize), nav.cols - 1);
                    const nr = Math.min(Math.floor((z + height / 2) / nav.cellSize), nav.rows - 1);
                    if (nav.blocked[nr * nav.cols + nc]) {
                        c.fromArray(colors, i * 3).lerp(BLOCKED_TINT, 0.6).toArray(colors, i * 3);
                    }
                }
            }
        }
        attr.needsUpdate = true;
    }

    dispose(): void {
        this.mesh.geometry.dispose();
        (this.mesh.material as THREE.Material).dispose();
    }

    /** Colors by height and slope: grass in lowlands, dirt on slopes, rock on steep parts. */
    private vertexColors(geometry: THREE.BufferGeometry): Float32Array {
        const normals = geometry.getAttribute("normal") as THREE.BufferAttribute;
        const colors = new Float32Array(normals.count * 3);
        const c = new THREE.Color();
        const { maxHeight } = this.info;

        for (let i = 0; i < normals.count; i++) {
            const slope = 1 - normals.getY(i); // 0 flat … 1 vertical
            const h = this.heights[i] / maxHeight;

            c.copy(GRASS_LOW).lerp(GRASS_HIGH, Math.min(h * 1.6, 1));
            c.lerp(DIRT, smoothstep(0.04, 0.12, slope));
            c.lerp(ROCK, smoothstep(0.14, 0.3, slope));
            // Subtle per-vertex noise to break up flat colour.
            const n = (hash(i) - 0.5) * 0.06;
            colors[i * 3] = c.r + n;
            colors[i * 3 + 1] = c.g + n;
            colors[i * 3 + 2] = c.b + n;
        }
        return colors;
    }
}

function smoothstep(a: number, b: number, x: number): number {
    const t = Math.min(Math.max((x - a) / (b - a), 0), 1);
    return t * t * (3 - 2 * t);
}

function hash(i: number): number {
    const s = Math.sin(i * 12.9898) * 43758.5453;
    return s - Math.floor(s);
}
