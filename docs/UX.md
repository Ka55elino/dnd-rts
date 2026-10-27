# UX: cursors

The cursor shows what a click at this spot will do. The client decides, as a hint, using the same data as the right click: selection, what's under the mouse, who can attack whom. The server still checks every order itself.

- **Code:** `frontend/src/game/hud/cursors.ts` (images and click points), `Game.pickCursor()` (selection logic).
- **Rate:** recomputed on mouse move and per frame, no more than 10 times per second (the check projects soldiers onto the screen).
- **Images:** SVG placeholders in the HUD style (bronze with a dark outline). At startup they're turned into 32 and 64 px PNGs (`image-set`, sharp on Retina); while loading — system cursors.
- **Replacing the art:** edit the SVG in `CURSORS` under the same name (or later load a PNG); click points stay in code.

## States

### No selection

| Name | When |
|---|---|
| `default` | default |
| `select` | over an own squad or building |
| `inspect` | over an enemy or neutral object |
| `scrollN` … `scrollNW` | mouse at the screen edge (8 px): scrolling in 8 directions |
| `plot` | over an own free plot or tower slot |
| `rally` | an own building with recruitment is selected, cursor over the ground (right click — rally point) |

### Own squads selected (what a right click will do)

| Name | When |
|---|---|
| `move` | over the ground, over an own squad, over an own building with no special action |
| `attack` | over an enemy (squad or building) |
| `no` | over an enemy the selected squads can't reach: a flyer with no anti-air; on a wall or tower — only for melee |
| `climb` | infantry over an own wall or watchtower |
| `build` | workers over an own foundation |
| `repair` | workers over an own damaged building |
| `capture` | over a neutral outpost |

### Targeting mode (button on the command card)

| Name | When |
|---|---|
| `targetMove` | "Move" (G) |
| `targetAttack` | "Attack" (F) over an enemy |
| `attackMove` | "Attack" (F) over the ground: attack-move |
| `patrol` | "Patrol" (P) |
| `fireAt` | "Attack ground" (T) |
| `cast` / `castAlly` | ability on a point (or on an ally) |
| `no` | ability on a squad, and there's no valid target under the mouse |

### Construction

| Name | When |
|---|---|
| `place` | a building is being placed (the building ghost is visible) |

Over the HUD (ring, command card, console, minimap) — their own cursors from CSS.

## Later

- Own images instead of placeholders (art), an animated wait cursor.
- A "not allowed" cursor for unavailable buildings in construction mode (for now the red ghost shows this).
