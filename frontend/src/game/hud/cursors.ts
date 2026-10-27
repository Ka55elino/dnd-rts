/**
 * Game cursors (docs/UX.md): placeholder SVGs in the HUD style (bronze on dark
 * outline), turned into PNG data URLs at start (WebKit doesn't take SVG
 * cursors everywhere), at 1× and 2× for sharp cursors on Retina screens.
 * Replace the art later by editing the SVG bodies (or loading PNG files under
 * the same names); hotspots stay here.
 */

export type CursorName =
    | "default"
    | "select"
    | "inspect"
    | "scrollN"
    | "scrollNE"
    | "scrollE"
    | "scrollSE"
    | "scrollS"
    | "scrollSW"
    | "scrollW"
    | "scrollNW"
    | "move"
    | "attack"
    | "attackMove"
    | "no"
    | "climb"
    | "build"
    | "repair"
    | "capture"
    | "rally"
    | "targetMove"
    | "targetAttack"
    | "patrol"
    | "fireAt"
    | "cast"
    | "castAlly"
    | "place"
    | "plot";

const OUT = "#1c130a"; // dark outline under every stroke
const BRONZE = "#e2bd72";
const RED = "#e0533a";
const GREEN = "#7fd36a";
const BLUE = "#7cc4ff";
const ORANGE = "#ff9a3c";
const VIOLET = "#c89bff";

/** Strokes drawn twice: a wide dark outline, then the colour on top. */
function stroked(d: string, color: string, width = 2): string {
    return `<path d="${d}" fill="none" stroke="${OUT}" stroke-width="${width + 2.4}" stroke-linecap="round" stroke-linejoin="round"/>` +
        `<path d="${d}" fill="none" stroke="${color}" stroke-width="${width}" stroke-linecap="round" stroke-linejoin="round"/>`;
}

const ARROW = "M3 2 L3 24 L9 18.5 L13 27 L16.5 25.5 L12.5 17 L20.5 17 Z";
const arrow = (fill: string) => `<path d="${ARROW}" fill="${fill}" stroke="${OUT}" stroke-width="1.6" stroke-linejoin="round"/>`;

const crosshair = (color: string) =>
    stroked("M16 3 V10 M16 22 V29 M3 16 H10 M22 16 H29", color) + stroked("M16 16 m-7 0 a7 7 0 1 0 14 0 a7 7 0 1 0 -14 0", color, 1.6);

const chevron = (rot: number) => `<g transform="rotate(${rot} 16 16)">${stroked("M8 19 L16 9 L24 19", BRONZE, 3)}${stroked("M10.5 25 L16 18.5 L21.5 25", BRONZE, 2)}</g>`;

const CURSORS: Record<CursorName, { svg: string; hot: [number, number] }> = {
    default: { svg: arrow(BRONZE), hot: [3, 2] },
    select: { svg: arrow("#fff2c8") + stroked("M23 23 m-4 0 a4 4 0 1 0 8 0 a4 4 0 1 0 -8 0", GREEN, 1.6), hot: [3, 2] },
    inspect: { svg: arrow(BRONZE) + stroked("M22 21 m-4.5 0 a4.5 4.5 0 1 0 9 0 a4.5 4.5 0 1 0 -9 0 M25.5 24.5 L29 28", BLUE, 1.8), hot: [3, 2] },
    scrollN: { svg: chevron(0), hot: [16, 4] },
    scrollNE: { svg: chevron(45), hot: [27, 5] },
    scrollE: { svg: chevron(90), hot: [28, 16] },
    scrollSE: { svg: chevron(135), hot: [27, 27] },
    scrollS: { svg: chevron(180), hot: [16, 28] },
    scrollSW: { svg: chevron(225), hot: [5, 27] },
    scrollW: { svg: chevron(270), hot: [4, 16] },
    scrollNW: { svg: chevron(315), hot: [5, 5] },
    move: { svg: stroked("M16 4 V20 M10 14 L16 20 L22 14", GREEN, 2.6) + stroked("M7 26 Q16 30 25 26", GREEN, 1.8), hot: [16, 22] },
    attack: { svg: stroked("M5 5 L21 21 M27 5 L11 21 M18 24 L24 18 M8 18 L14 24 M21 21 L27 27 M11 21 L5 27", RED, 2.2), hot: [16, 13] },
    attackMove: { svg: stroked("M5 5 L19 19 M15 23 L23 15 M19 19 L25 25", ORANGE, 2.2) + stroked("M22 4 V12 M18 8 L22 12 L26 8", ORANGE, 1.8), hot: [16, 16] },
    no: { svg: stroked("M16 16 m-11 0 a11 11 0 1 0 22 0 a11 11 0 1 0 -22 0 M8.5 8.5 L23.5 23.5", RED, 2.6), hot: [16, 16] },
    climb: { svg: stroked("M10 4 V28 M22 4 V28 M10 9 H22 M10 15 H22 M10 21 H22 M10 27 H22", BRONZE, 2) + stroked("M16 1 L16 5", GREEN, 2), hot: [16, 16] },
    build: { svg: stroked("M18 5 L27 14 L23.5 17.5 L14.5 8.5 Z M16 11 L4 23 A2.5 2.5 0 0 0 7.5 26.5 L19.5 14.5", BRONZE, 2), hot: [6, 25] },
    repair: { svg: stroked("M18 5 L27 14 L23.5 17.5 L14.5 8.5 Z M16 11 L4 23 A2.5 2.5 0 0 0 7.5 26.5 L19.5 14.5", BRONZE, 2) + stroked("M24 22 V30 M20 26 H28", GREEN, 2.2), hot: [6, 25] },
    capture: { svg: stroked("M8 29 V4 M8 5 H24 L20 10 L24 15 H8", BRONZE, 2.2), hot: [8, 29] },
    rally: { svg: stroked("M8 29 V4 M8 5 H24 L20 10 L24 15 H8", BLUE, 2.2), hot: [8, 29] },
    targetMove: { svg: crosshair(GREEN), hot: [16, 16] },
    targetAttack: { svg: crosshair(RED), hot: [16, 16] },
    patrol: { svg: crosshair(BLUE) + stroked("M9 9 Q16 4 23 9 M21 6 L23 9 L20 11", BLUE, 1.6), hot: [16, 16] },
    fireAt: { svg: crosshair(ORANGE) + stroked("M16 16 m-2 0 a2 2 0 1 0 4 0 a2 2 0 1 0 -4 0", ORANGE, 2), hot: [16, 16] },
    cast: { svg: stroked("M16 3 L18.5 13.5 L29 16 L18.5 18.5 L16 29 L13.5 18.5 L3 16 L13.5 13.5 Z", VIOLET, 1.8), hot: [16, 16] },
    castAlly: { svg: stroked("M16 3 L18.5 13.5 L29 16 L18.5 18.5 L16 29 L13.5 18.5 L3 16 L13.5 13.5 Z", GREEN, 1.8), hot: [16, 16] },
    place: { svg: stroked("M16 9 V23 M9 16 H23", BRONZE, 1.6), hot: [16, 16] },
    plot: { svg: arrow(BRONZE) + stroked("M17 17 H29 V29 H17 Z", GREEN, 1.6), hot: [3, 2] },
};

/** Fallback CSS keywords while the images load (or if they fail). */
const FALLBACK: Partial<Record<CursorName, string>> = {
    default: "default",
    select: "pointer",
    inspect: "help",
    scrollN: "n-resize",
    scrollNE: "ne-resize",
    scrollE: "e-resize",
    scrollSE: "se-resize",
    scrollS: "s-resize",
    scrollSW: "sw-resize",
    scrollW: "w-resize",
    scrollNW: "nw-resize",
    no: "not-allowed",
    targetMove: "crosshair",
    targetAttack: "crosshair",
    patrol: "crosshair",
    fireAt: "crosshair",
    cast: "crosshair",
    castAlly: "crosshair",
    place: "crosshair",
};

/** CSS cursor values, ready after load(). */
const css = new Map<CursorName, string>();

function toPng(svg: string, size: number): Promise<string> {
    const src = `data:image/svg+xml;charset=utf-8,${encodeURIComponent(`<svg xmlns="http://www.w3.org/2000/svg" width="${size}" height="${size}" viewBox="0 0 32 32">${svg}</svg>`)}`;
    return new Promise((resolve, reject) => {
        const img = new Image();
        img.onload = () => {
            const c = document.createElement("canvas");
            c.width = c.height = size;
            c.getContext("2d")!.drawImage(img, 0, 0, size, size);
            resolve(c.toDataURL("image/png"));
        };
        img.onerror = reject;
        img.src = src;
    });
}

/** Renders every cursor to PNG once (call at game start). */
export async function loadCursors(): Promise<void> {
    await Promise.all(
        (Object.keys(CURSORS) as CursorName[]).map(async (name) => {
            const { svg, hot } = CURSORS[name];
            try {
                const [x1, x2] = await Promise.all([toPng(svg, 32), toPng(svg, 64)]);
                const fallback = FALLBACK[name] ?? "default";
                // image-set: sharp on Retina; WebKit wants the prefixed form. Plain url() as the last resort.
                const probe = document.createElement("div").style;
                for (const v of [
                    `image-set(url("${x1}") 1x, url("${x2}") 2x) ${hot[0]} ${hot[1]}, ${fallback}`,
                    `-webkit-image-set(url("${x1}") 1x, url("${x2}") 2x) ${hot[0]} ${hot[1]}, ${fallback}`,
                    `url("${x1}") ${hot[0]} ${hot[1]}, ${fallback}`,
                ]) {
                    probe.cursor = "";
                    probe.cursor = v;
                    if (probe.cursor) {
                        css.set(name, v);
                        break;
                    }
                }
            } catch {
                /* keep the keyword fallback */
            }
        }),
    );
}

/** The CSS cursor value for a name. */
export function cursorCss(name: CursorName): string {
    return css.get(name) ?? FALLBACK[name] ?? "default";
}

/** Sets an element's cursor (no-op when unchanged). */
export function setCursor(el: HTMLElement, name: CursorName): void {
    if (el.dataset.cursor === name) return;
    el.dataset.cursor = name;
    el.style.cursor = cursorCss(name);
}
