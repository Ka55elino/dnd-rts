# Bases: shape, size, plots

A base is the main building (keep) with a wall outline, gates, tower slots on the wall and plots for buildings inside. As in BFME2: round (sometimes square or polygonal) bases with rings of plots.

Two ways to describe a base:

| Mode | Where the base comes from | Why |
|---|---|---|
| **PvP / skirmish** | the **generator** in the faction keep's JSON (`base`) | the same rules for all players, size and plot count are tuned with a couple of numbers, easy to balance |
| **Single-player campaign** | **by hand in the map editor** (phase 2): outline, gates, plots at the player slot on a specific map | a base fitted to the terrain and mission story (by a river, on a hill, irregular) |

Both produce the same data (outline, gates, tower slots, list of plots). The simulation and the client don't know where the base came from.

## Generator (`base` on the keep)

```json
"base": {
  "shape": "circle",
  "radius": 70,
  "rings": [
    { "radius": 26, "count": 4, "offset": 45, "sizes": ["m"] },
    { "radius": 39, "count": 6, "offset": 30, "sizes": ["s", "l", "s"] },
    { "radius": 53, "count": 6, "offset": 30, "sizes": ["m"] }
  ],
  "gates": [0, 180],
  "towers": 8,
  "wall": "castle_wall",
  "gate": "castle_gate",
  "stairs": 4
}
```

Angles are degrees clockwise from the keep's front (0 — forward, 90 — right). Everything is relative to the keep; rotation comes from the map slot.

| Field | What it sets |
|---|---|
| `shape` | wall outline: `circle`, `square` or `polygon` (with `sides` ≥ 5) |
| `radius` | circle radius, half the side of a square, circumradius of a polygon |
| `rotate` | rotation of the square / polygon outline, degrees |
| `rings` | rings of plots: `count` plots on a circle of `radius`, the first at angle `offset`. `sizes` — plot classes, repeated around the ring (`["s", "l", "s"]` → S, L, S, S, L, S) |
| `gates` | gate directions: an opening in the wall |
| `towers` | tower slots on the wall, evenly around the circle between the "spokes". Those too close to a gate (closer than 12°) are skipped |
| `wall` / `gate` | the wall segment building (kind `wall`) and the gatehouse (kind `gate`) |
| `gateWidth` | how many wall segments the gatehouse takes up (default — its length / segment length, 3 for the Empire) |
| `stairs` | stairs up onto the wall from inside: one on each side of each gate plus `stairs` more evenly around the circle |

Explicit plots (`plots` with `offset` and `class`) can be added on top of the rings: e.g. a large plot by the gate.

**Load-time checks:**
- plots don't overlap each other or the keep and don't extend past the wall;
- a keep has no more than 48 plots (including tower slots): occupancy is sent as a 48-bit mask.

**Free-standing buildings** can't be placed inside the wall: the keep's `noBuildRadius` is automatically at least the base radius + 4 m.

**Terrain:** the ground around the keep is leveled (80 m radius), like at an army start point. The army starts inside the base, in front of the keep.

## Plots S / M / L

Sizes are in `content/plots.json`: S = 9, M = 12, L = 16 (square side, m). A building fits on a plot if its footprint fits. Large buildings don't fit on small plots; small ones fit on any.

| Class | Examples (Empire) |
|---|---|
| S (9) | farm |
| M (12) | barracks, archery range, stables, forge |
| L (16) | mages' guild, later the workshop |

Outpost plots are described the same way: `"class": "m"`.

## Where to build: plot and / or field

A building's `placement` is `["plot"]`, `["free"]` or both:
- `plot` — only on a base or outpost plot, builds itself;
- `free` — in the field, by workers (outside the base wall);
- both — either way.

Currently for the Empire: barracks, archery range, stables, mages' guild, farm — both ways; forge — plot only.

## Base wall

The wall **stands from the start**, together with the keep, along the generator outline. The Empire keep has two gatehouses exactly in the gate directions, two arcs of 33 segments (~6.2 m) between them, 8 tower slots, 8 stairs.

**Segments** are ordinary buildings of kind `wall` / `gate` with HP, armor and material:
- `content/buildings/castle_wall.json`: footprint 6.2 × 7 m, 3000 HP;
- `castle_gate.json`: gatehouse 18.6 × 7 m (3 segments wide), 6000 HP, a 5 m `passage` in the middle;
- `walkHeight` — walkway height, 5.6 m.

Walls block navigation: every cell a segment touches counts as occupied, so nothing can slip through the wall. A destroyed segment is a breach: the cells open up and the enemy gets through.

**The gate is a toggle, as in BFME:** an own gatehouse has an "Open / Close the gate" button in the ring.
- An open gate lets **everyone** through, the enemy too, via the passage in the middle (5 m). The sides of the gatehouse always block.
- A closed gate lets no one through.
- It closes only when nobody is standing in the passage: the command waits until the opening is clear.
- At the start of a match the gates are open so the army can get out.

Gate state is computed by the simulation (a flag in the snapshot), so it's the same for all players.

**Walking along the wall** is interrupted at gatehouses: with two gates the wall is two separate arcs, each with its own stairs.

**Infantry can go up on the wall:** tag `infantry`; no cavalry, monsters, machines or flyers.
- A "move" order to a point on an own wall, or a right-click on it — the squad goes to the nearest suitable stairs, climbs up and walks along the wall to the point.
- A "move" order to the ground — the squad walks along the wall to the stairs nearest the target, goes down and continues. Attack-move and patrol from the wall also go down first.
- On the wall a squad stands in formation along the wall (up to 4 ranks), faces outward and doesn't pursue the enemy.
- Ranged units on the wall get +25% range.
- Melee only reaches its own "level": units on the ground can't hit those on the wall and vice versa. Flyers hit everyone; ranged attacks reach everyone.
- If the segment under a squad is destroyed, the squad falls down, inside the base.

**Towers** are built on tower slots (plot class `tower`, 7 m): the building has `placement: ["tower"]`.
- Archer tower: piercing damage, hits both ground and air, a volley of 5 arrows every 1.5 s, range 32 m +25% on the wall.
- Attacking buildings are defined with the `attack` field (like a unit's ranged attack) and `volley` (arrows per volley).

**Rendering:**
- a segment is two rows of the `castle_wall` model back to back (crenellations on both sides) with a stone walkway on top;
- the gate is the `gatehouse` model (`gatehouse-light.glb`), stretched in width to its 18.6 m. The doors and portcullis are separate nodes (`meshbake`: `keepNamed`). Opening takes 1.2 s: first the portcullis rises, then the doors swing inward; closing goes in reverse order;
- walls are drawn with instancing by LOD (about 16k / 2k / 370 triangles) with frustum culling;
- a segment's HP bar is visible only when it's damaged;
- units on the wall are drawn at walkway height; arrows from the wall and towers fly from above.

## Campaign (map editor, phase 2)

A player slot on the map will have an optional `base` block. It holds either the same generator with different numbers, or explicit `outline` (outline points), `gates`, `towers` and `plots`. If there's no block, the faction keep's base is used.

## Watchtower (in the field)

Standalone walls aren't built in the field. Instead there's the **watchtower** (`content/buildings/watchtower.json`, model `wooden_watchtower`):
- placed by workers in the field;
- `garrison: true`, `walkHeight` — platform height, 7.6 m.

Infantry gets onto the tower the same way as onto the wall: right-click an own tower, or a "move" order to its point. The same rules apply: +25% range for ranged units, melee from the ground can't reach.
- The tower holds **one squad**; others refuse to climb up.
- Cavalry, machines and monsters can't go up the tower.
- If the tower is destroyed, the squad falls down.
