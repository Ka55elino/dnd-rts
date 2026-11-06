# Bases: walls and building

A base is the keep and whatever its owner builds around it. There are no build plots and no ready-made wall: workers put up every building wherever there is room, and they build the walls too — as lines dragged across the map, with gatehouses set into them and defensive towers on top.

## Building

Every building is put up by **workers** (the faction's builders): select them, pick a building on the command card (bottom right, 6 × 3: Move and Stop, then every building of the faction), place the ghost (R rotates), they walk over and build; more workers build faster. Nothing builds itself. Cancelling a foundation refunds it in full. A building's `placement`:
- `free` — anywhere on open ground: not on terrain that blocks movement, trees or other buildings, not on the foot of a wall stair (3 m clear), outside every building's no-build circle (`noBuildRadius`: keep 12 m, gatehouse 8 m — the gate passage stays open, farms 12 m) and with nothing inside its own;
- `wall` — wall segments, see below;
- `gate` — a gatehouse over a stretch of the own finished wall, see below;
- `tower` — defensive towers, **on the own wall only**: the ghost snaps to the centre line of the nearest own finished wall segment (within 8 m of the click), facing out. Not on a gatehouse (2 m clear), not on another tower, not where a stair comes up. Workers build it from the ground.

**Terrain:** the ground around the keep is leveled (80 m radius). The army starts in front of the keep.

## Walls

**Laying a wall:** pick the wall on the command card, click the start, then the end: the line is cut into segments one footprint long (6.2 m, at most 40 per order), the ghost shows them green while they fit and red from the first one that doesn't (the wall stops there: at a building, a tree, blocked ground). Each segment is a foundation paid on its own (`castle_wall`: 15 gold, 10 wood, 6 s); if the money runs out the line stops. The workers build the segments one after another, the nearest first. A finished segment already blocks and can be walked on.

- The wall's front (`out`) faces away from the owner's keep; the **stairs** are on the keep's side: at both ends of the line and every 8 segments.
- **Continuing a wall:** a line started on (or just past) an end segment of an own wall continues that wall: the new segments go on from the tip of the end segment in the dragged direction (turns up to 120°), it stays one wall with one walkway; the ghost snaps to the end. Where the wall turns (more than ~10°) a **round joint** fills the corner: a stone drum of the wall's thickness up to the walkway with a parapet ring; it blocks movement like the wall (`sim.WallJoints`). The stair at the joined end goes away if the new wall covers it. Lines started elsewhere are separate walls (their walkways don't connect).
- **Unfinished wall:** right click own foundations with workers — they build that segment, then the rest of the wall's unfinished segments, nearest first (the same for repairing a damaged one: right click it). A worker builds when it is within 1.5 cells (+1 m) of the footprint.
- **Gatehouse** (`castle_gate`: 120 gold, 60 wood, 25 s): pick it and click an own finished wall — it replaces as many segments as it is long (3) around the clicked one; they must all be finished plain wall (no tower on them). Workers build it.

**Segments** are ordinary buildings of kind `wall` / `gate` with HP, armor and material:
- `content/buildings/castle_wall.json`: footprint 6.2 × 7 m, 3000 HP;
- `castle_gate.json`: gatehouse 18.6 × 7 m (3 segments wide), 6000 HP, a 5 m `passage` in the middle;
- `walkHeight` — walkway height, 5.6 m.

Walls block navigation: every cell a segment touches counts as occupied, so nothing can slip through the wall. A destroyed segment is a breach: the cells open up and the enemy gets through.

**The gate is a toggle, as in BFME:** an own gatehouse has an "Open / Close the gate" button in the ring.
- An open gate lets **everyone** through, the enemy too, via the passage in the middle (5 m). The sides of the gatehouse always block.
- A closed gate lets no one through.
- It closes only when nobody is standing in the passage: the command waits until the opening is clear.
- A new gatehouse is open.

Gate state is computed by the simulation (a flag in the snapshot), so it's the same for all players.

**Walking along the wall** is interrupted at gatehouses and gaps (unfinished or destroyed segments).

**Infantry can go up on the wall:** tag `infantry`; no cavalry, monsters, machines or flyers.
- A "move" order to a point on an own wall, or a right-click on it — the squad goes to the nearest suitable stairs, climbs up and walks along the wall to the point.
- A "move" order to the ground — the squad walks along the wall to the stairs nearest the target, goes down and continues. Attack-move and patrol from the wall also go down first.
- On the wall a squad stands in formation along the wall (up to 4 ranks), faces outward and doesn't pursue the enemy.
- Ranged units on the wall get +25% range.
- Melee only reaches its own "level": units on the ground can't hit those on the wall and vice versa. Flyers hit everyone; ranged attacks reach everyone.
- If the segment under a squad is destroyed, the squad falls down on the inner side.

**Towers** stand on the wall wherever the player put them (`placement: ["tower"]`, see Building).
- Archer tower: piercing damage, hits both ground and air, a volley of 5 arrows every 1.5 s, range 32 m +25% on the wall.
- Attacking buildings are defined with the `attack` field (like a unit's ranged attack) and `volley` (arrows per volley).

**Rendering:**
- a segment is two rows of the `castle_wall` model back to back (crenellations on both sides) with a stone walkway on top; a foundation rises from 15% of its height as it is built;
- the gate is the `gatehouse` model (`art/props/empire.py`), stretched in width to its 18.6 m. The doors and portcullis are separate nodes (`gate_left`, `gate_right`, `portcullis`). Opening takes 1.2 s: first the portcullis rises, then the doors swing inward; closing goes in reverse order;
- walls are drawn with instancing by LOD with frustum culling;
- a segment's HP bar is visible only when it's damaged;
- units on the wall are drawn at walkway height; arrows from the wall and towers fly from above.

## Prepared bases (campaign, map editor)

A fortress may come with a ready-made walled base: a `base` generator on the building (or, later, on the map's player slot), expanded at load time into a closed ring of wall segments with gatehouses and stairs. The standard keep has none. `data.TestBaseLayout()` is the one the tests use:

```json
"base": { "shape": "circle", "radius": 70, "gates": [0, 180], "wall": "castle_wall", "gate": "castle_gate", "stairs": 4 }
```

| Field | What it sets |
|---|---|
| `shape` | outline: `circle`, `square` or `polygon` (with `sides` ≥ 5) |
| `radius` | circle radius, half the side of a square, circumradius of a polygon |
| `rotate` | rotation of the square / polygon outline, degrees |
| `gates` | gate directions (degrees clockwise from the keep's front) |
| `wall` / `gate` | the wall segment building (kind `wall`) and the gatehouse (kind `gate`) |
| `gateWidth` | how many wall segments the gatehouse takes up (default — its length / segment length) |
| `stairs` | stairs from inside: one on each side of each gate plus `stairs` more evenly around |


## Watchtower (in the field)

Standalone walls aren't built in the field. Instead there's the **watchtower** (`content/buildings/watchtower.json`, model `watchtower`):
- placed by workers in the field;
- `garrison: true`, `walkHeight` — platform height, 7.6 m.

Infantry gets onto the tower the same way as onto the wall: right-click an own tower, or a "move" order to its point. The same rules apply: +25% range for ranged units, melee from the ground can't reach.
- The tower holds **one squad**; others refuse to climb up.
- Cavalry, machines and monsters can't go up the tower.
- If the tower is destroyed, the squad falls down.
