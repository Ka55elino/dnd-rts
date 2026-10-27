/**
 * HUD icon set: original line glyphs (24×24, drawn with currentColor) used
 * until real portraits / building art exist. iconFor() picks a glyph for a
 * content id; unknown ids get their initial.
 */

const P: Record<string, string> = {
    // Units
    sword: `<path d="M18.5 3.5 20.5 5.5 9 17l-2-2zM6.5 14.5l3 3M4 17l3 3M5 20l-1.5 1.5"/><path d="M12.5 7.5 16.5 11.5"/>`,
    bow: `<path d="M6 3c7 2 11 7 13 15M6 3l13 15"/><path d="M5 9 14 12M11 3l1.5 7.5M19 18l1.5 2.5M3.5 7.5 6 3l4 1"/>`,
    horse: `<path d="M7 21v-6l-3-2 2-5 4-2 2-3 2 4 3 1 2 5-2 1-2-2-1 3v6"/><circle cx="12.5" cy="7.5" r=".8" fill="currentColor"/>`,
    crown: `<path d="M3.5 18h17l1-10-5.5 4L12 5l-4 7-5.5-4z"/><path d="M4 21h16"/>`,
    hammer: `<path d="M14 4l6 6-2.5 2.5-6-6z"/><path d="M12.5 7.5 4 16a1.8 1.8 0 0 0 2.5 2.5L15 10"/><path d="M17 4.5l2-1.5 2 2-1.5 2"/>`,
    raider: `<path d="M5 19 16 8M14 4l6 2-2 6M7 13l4 4M3.5 20.5 6 18"/><path d="M15 6.5l2.5 2.5"/>`,
    wolf: `<path d="M4 20l3-8-2-6 5 3h4l5-3-2 6 3 8"/><path d="M9 14l3 3 3-3"/><circle cx="9.5" cy="11.5" r=".8" fill="currentColor"/><circle cx="14.5" cy="11.5" r=".8" fill="currentColor"/>`,
    // Buildings
    keep: `<path d="M5 21V9h3V6h2v3h4V6h2v3h3v12z"/><path d="M10 21v-4a2 2 0 0 1 4 0v4M4 6V3l2 1"/>`,
    barracks: `<path d="M3 20 12 4l9 16z"/><path d="M12 4v16M8.5 20 12 13l3.5 7"/>`,
    farm: `<path d="M12 21V9M12 9c-3-1-4-4-4-6 3 1 4 3 4 6zM12 9c3-1 4-4 4-6-3 1-4 3-4 6z"/><path d="M12 14c-3-1-5-3-5-6M12 14c3-1 5-3 5-6M7 21h10"/>`,
    tower: `<path d="M7 21V8h10v13M6 8V4h2v2h2V4h4v2h2V4h2v4"/><path d="M11 12h2v3h-2z"/>`,
    mine: `<path d="M4 20h16l-3-7-5-3-5 3z"/><path d="M14 4l5 5M17.5 2.5 21 6M11 7l3-3"/>`,
    sawmill: `<circle cx="12" cy="12" r="7"/><path d="M12 5v2M12 17v2M5 12h2M17 12h2M7 7l1.4 1.4M15.6 15.6 17 17M7 17l1.4-1.4M15.6 8.4 17 7"/><circle cx="12" cy="12" r="2"/>`,
    ore: `<path d="M5 15l4-8 6 1 4 6-5 6-7-1z"/><path d="M9 7l2 7 8 0M11 14l-4 4"/>`,
    // Commands
    cancel: `<path d="M6 6l12 12M18 6 6 18"/>`,
    fire: `<path d="M12 21c-4 0-6-3-6-6 0-4 4-6 4-10 3 2 4 4 4 6 1-1 2-2 2-4 2 2 2 5 2 8 0 3-2 6-6 6z"/><path d="M12 21c-1.5 0-2.5-1-2.5-2.5S12 15 12 14c1 1.5 2.5 3 2.5 4.5S13.5 21 12 21z"/>`,
    gateOpen: `<path d="M4 21V9a8 8 0 0 1 16 0v12"/><path d="M8 21v-8l3 1v7M16 21v-8l-3 1v7"/>`,
    gateClosed: `<path d="M4 21V9a8 8 0 0 1 16 0v12"/><path d="M8 21V9M12 21V5M16 21V9M6 13h12M6 17h12"/>`,
    rally: `<path d="M6 21V3M6 4h11l-3 4 3 4H6"/>`,
    build: `<path d="M3 21h18M5 21V11l7-6 7 6v10"/><path d="M9 21v-6h6v6"/>`,
    move: `<path d="M4 20 20 4M20 4h-8M20 4v8"/><path d="M4 14v6h6" opacity=".55"/>`,
    stop: `<path d="M8 3h8l5 5v8l-5 5H8l-5-5V8z"/><path d="M8.5 12h7"/>`,
    hold: `<path d="M12 3 20 6v6c0 5-3.5 8-8 9-4.5-1-8-4-8-9V6z"/><path d="M12 8v8M8.5 12h7"/>`,
    attack: `<path d="M4 4l9 9M4 4h4M4 4v4M20 4l-9 9M20 4h-4M20 4v4"/><path d="M9 15l-4 4M15 15l4 4M7 13l4 4M17 13l-4 4"/>`,
    fireAt: `<circle cx="12" cy="12" r="7"/><circle cx="12" cy="12" r="2.5"/><path d="M12 2v3M12 19v3M2 12h3M19 12h3"/>`,
    patrol: `<path d="M5 9h12l-3-3M19 15H7l3 3"/>`,
    gold: `<circle cx="12" cy="12" r="7"/><path d="M12 8v8M9.5 10h4a1.5 1.5 0 0 1 0 3h-3a1.5 1.5 0 0 0 0 3h4"/>`,
};

/** id / tag → glyph name. */
const MAP: Record<string, string> = {
    swordsmen: "sword",
    archers: "bow",
    knights: "horse",
    wolf_riders: "wolf",
    raiders: "raider",
    captain: "crown",
    warlord: "crown",
    builders: "hammer",
    keep: "keep",
    barracks: "barracks",
    farm: "farm",
    tower: "tower",
    gold_mine: "mine",
    sawmill: "sawmill",
    ore_mine: "ore",
};

/** SVG markup of a named glyph. */
export function icon(name: string, cls = "hud-icon"): string {
    const body = P[name];
    if (!body) return "";
    return `<svg class="${cls}" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round">${body}</svg>`;
}

/** Icon for a content id (horde or building), or its initial. */
export function iconFor(id: string, name = id): string {
    const glyph = MAP[id];
    return glyph ? icon(glyph) : `<span class="hud-icon hud-icon--letter">${(name[0] ?? "?").toUpperCase()}</span>`;
}
