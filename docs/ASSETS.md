# Assets: icons, 3D models, animations.

## 1. Technical requirements

### Common to all 3D models

| Rule | Value |
|---|---|
| Format | `.glb` (glTF 2.0) in `art/source/...` |
| Scale | 1 unit = 1 meter |
| Axes | Y up, **model front is +Z** |
| Origin | at the center of the base (on the ground) |
| Size | by footprint (`footprint` in JSON) or unit radius, within ~10% |
| Moving parts | separate **named nodes**; pivot on the axis (hinge, wheel axle, lever axle) |
| LOD | not needed by hand: `tools/meshbake` and `tools/vatbake` make 3 levels themselves |

### Animated units (VAT, `tools/vatbake`)

Units are drawn with instancing, up to ~100k in a scene, so the budgets are strict.

| Rule | Value |
|---|---|
| Skeleton | one skinned mesh, one skeleton |
| Materials | **no textures**: each material is turned into vertex color (low-poly, flat shading). Team-color materials are listed in `teamMaterials` (cloak, shield, banner) |
| Normals | not needed: flat shading is computed on the GPU |
| Clip frame rate | 15 frames per second |
| Total clip length | ≤ 12 s per model (≈ 180 frames) |

LOD0 budgets (counted after vertex welding):

| Category | Triangles | Vertices |
|---|---|---|
| Infantry, mages, workers | ≤ 3,000 | ≤ 2,000 |
| Cavalry (rider and horse are one model, one skeleton) | ≤ 5,000 | ≤ 3,500 |
| Heroes | ≤ 8,000 | ≤ 5,000 |
| Summoned creatures (elementals) | ≤ 6,000 | ≤ 4,000 |

The swordsman is currently 5,900 triangles: over the infantry budget.

**Clips** (names are keys in `tools/vatbake/models.json`):

| Clip | Loop | Duration | For |
|---|---|---|---|
| `idle` | yes | 1–2 s | everyone |
| `walk` | yes | ~1 s | everyone |
| `run` | yes | ~0.7 s | everyone (gallop for cavalry) |
| `attack` | no | 0.6–1 s | everyone who attacks; a shot for ranged units |
| `die` | no | 1–1.5 s | everyone |
| `cast` | no | ~1 s | mages, healers, heroes |
| `work` | yes | ~1 s | workers: construction, repair (hammer blow) |
| `brace` | yes | ~1 s | swordsmen (shield wall), spearmen (spear wall): stance pose |

If a clip is missing, `vatbake` substitutes a procedural one (`@swing`, `@bow`, `@die`) — fine for placeholders.

### Static models: buildings and machines (`tools/meshbake`)

| Rule | Value |
|---|---|
| Textures | allowed: PBR, ≤ 1024×1024 per model, shared atlases welcome |
| Materials | ≤ 8 per model (each is a separate draw call) |
| Far LODs | on models with UV seams and flat shading, normal simplification doesn't work, so the coarse one (`sloppy`) is used. Better to make a simpler mesh from the start |

| Category | LOD0 triangles | Note |
|---|---|---|
| Wall segment | **≤ 3,000** | appears ~140 times on the map (2 rows each). Currently 16,600 — 5× over budget |
| Gatehouse | ≤ 15,000 | doors and portcullis are separate nodes |
| Wall tower (3 kinds) | ≤ 10,000 | footprint ≤ 7×7 m |
| Watchtower | ≤ 8,000 | platform for a squad at ~7–8 m height |
| Building S (farm) | ≤ 10,000 | footprint ≤ 9×9 m |
| Building M (barracks, archery range, stables, forge) | ≤ 20,000 | ≤ 12×12 m |
| Building L (workshop, mages' guild) | ≤ 25,000 | ≤ 16×16 m |
| Keep | ≤ 40,000 | footprint 14×14 m |
| Outposts (gold mine, sawmill, ore mine) | ≤ 20,000 | 8×8 m |
| Siege machines | ≤ 15,000 | wheels, arm, counterweight, sling, bow — separate nodes |

**Moving node names** (the code finds them by name suffix):
- `_wheel_…` — wheels, rotate around local X;
- `_arm` — trebuchet or catapult arm, around X;
- `_counterweight`, `_sling` — hang from the arm;
- `_crossbow`, `_ram_log` — recoil or strike along −Z;
- `_gate_left`, `_gate_right`, `_portcullis` — gate doors (around Y on the hinge) and portcullis (up);
- `_sliding_door_…` — sliding doors (along X).

## 2. Icons

PNG with transparent background, unified style (bronze, stone). File name is the id from `content/`.

### Unit portraits — 256×256 (heroes also 512×512 for the large card)

| id | What |
|---|---|
| `builders` | builder |
| `swordsmen` | swordsman with shield |
| `spearmen` | spearman |
| `archers` | archer |
| `knights` | knight (also the Royal Guard `royal_guards`) |
| `ballistas` | ballista |
| `trebuchets` | trebuchet |
| `fire_mages` | fire mage |
| `water_mages` | water mage |
| `lightning_mages` | lightning mage |
| `healers` | healer |
| `paladin` | Paladin (hero) |
| `archmage` | Archmage (hero) |
| `king` | King (hero) |
| `fire_elementals`, `ice_elementals`, `storm_elementals` | elementals (Archmage summon) |

### Buildings — 128×128

`keep`, `barracks`, `archery_range`, `stables`, `workshop`, `mage_guild`, `forge`, `farm`, `watchtower`, `archer_tower`, `ballista_tower`, `trebuchet_tower`, `castle_wall`, `castle_gate`; outposts `gold_mine`, `sawmill`, `ore_mine`.

### Abilities — 128×128

| Who | id |
|---|---|
| Swordsmen | `shield_wall` |
| Spearmen | `spear_wall` |
| Archers and archer towers | `fire_mode` (fire arrows) |
| Ballistas and their towers | `fire_bolts_mode` (fire bolts) |
| Knights | `wedge` (wedge) |
| Fire mages | `fire_pillar` |
| Water mages | `storm` |
| Lightning mages | `chain_lightning` |
| Healers | `heal` |
| Paladin | `holy_radiance`, `purity`, `divine_smite`, `warding_aura`, `divine_shield` |
| Archmage | `element` + 3 element variants (`element_fire`, `element_ice`, `element_lightning`), `elemental_ward`, `summon_elemental`, `arcane_focus`, `meteor_shower` |
| King | `royal_guard`, `royal_march`, `inspire`, `royal_oversight`, `mass_recall` |

Ultimate abilities get a gold frame in code — no need to draw it on the icon.

### Forge upgrades — 128×128

`forged_weapons_1`, `forged_weapons_2`, `reinforced_armor_1`, `reinforced_armor_2`, `fire_arrows`, `fire_bolts`, `improved_mechanism`.

### Orders (command card) — 128×128

`move`, `stop`, `hold`, `attack`, `patrol`, `attack_ground`, `rally`, `cancel`, `build`, `repair`, `gate_open`, `gate_close`, `revive` (hero revival), `research` (generic for research).

### Resources — 64×64

`gold`, `wood`, `metal`, `food`.

### Statuses — 32×32 (badges on portraits, later)

`burning`, `poisoned`, `slowed`, `stunned`, `acid`, `blinded`, `grounded`, `on_wall`.

### Other

- **Empire coat of arms** — 256×256: player banner, faction selection screen.
- **Cursors** — 28 of them per the list in [`UX.md`](UX.md): 32×32 and 64×64, click point as on the placeholders.
- **HUD frames** (9-slice PNG): minimap ring, resource plate, command card frame, portrait medallion, tooltip.

## 3. 3D models and animations

### Units (VAT, animated)

| Unit | Current model | Clips |
|---|---|---|
| Builder | placeholder (swordsman) | idle, walk, run, attack, **work**, die |
| Swordsman with shield | exists (grab3d) | idle, walk, run, attack, **brace**, die |
| Spearman | placeholder | idle, walk, run, attack (thrust), **brace**, die |
| Archer | exists (ranger) | idle, walk, run, attack (shot), die |
| Knight | placeholder | idle, walk, run (gallop), attack, die |
| Fire / water / lightning mage | placeholders | idle, walk, run, attack (projectile), **cast**, die. Can be one model with 3 color schemes — but that's 3 VAT models: colors are baked into vertices |
| Healer | placeholder | idle, walk, run, **cast**, die |
| Paladin | placeholder | idle, walk, run, attack, **cast**, die |
| Archmage | placeholder | idle, walk, run, attack (projectile), **cast**, die |
| King (mounted) | placeholder | idle, walk, run (gallop), attack, **cast**, die |
| Elementals ×3 | placeholders | idle, walk, attack, die (the storm one's attack is a projectile) |

### Siege machines (static, moving nodes)

| Model | Now | Needed |
|---|---|---|
| Ballista | exists (wheels, bow) | + wreck (`ballista_wreck`) |
| Trebuchet | exists (wheels, arm, counterweight, sling) | + wreck (`trebuchet_wreck`) |

### Buildings (static)

| Building | Now | Note |
|---|---|---|
| Keep | exists (`keep`) | |
| Barracks, archery range, forge | placeholder (mausoleum) | M |
| Stables, farm | placeholder (witch's hut) | M / S |
| Workshop | placeholder (tea house) | L, sliding doors — animation on recruitment |
| Mage Guild | placeholder (mausoleum) | L |
| Watchtower | exists (wooden) | platform at `walkHeight` |
| Archer / ballista / trebuchet towers | placeholder (stone tower) | on a tower slot ≤ 7×7 m, weapon visible from above |
| Wall segment | exists, **too heavy** | ≤ 3,000 triangles; length 6.2 m, thickness 7 m (or ~2.4 m — then two rows back to back, as now), walkway height 5.6 m |
| Gatehouse | exists | opening ≥ 4 m |
| Outposts: gold mine, sawmill, ore mine | placeholder (castle) | |

### Shared models

| Model | Purpose |
|---|---|
| Foundation / scaffolding S, M, L | building under construction (currently the model just "grows" out of the ground) |
| Ruins S, M, L | destroyed building instead of sinking |
| Wall breach rubble | in place of a destroyed segment |
| Projectiles: arrow, bolt, stone, fireball | currently dots |
| Rally point banner / flag | rally point marker |
