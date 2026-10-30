# Art technical specification

Technical requirements for every model, texture and icon in the game. **What** has to be made is listed in [`ART_LIST.md`](ART_LIST.md); the look of each unit is described in the faction files in [`lore/factions/`](lore/factions/).

## 1. Pipeline

```
art/source/...glb ──► tools/vatbake  (animated units) ──► frontend/public/models/<id>/  (VAT + geometry)
                  └─► tools/meshbake (static models)  ──► frontend/public/models/<id>/  (lod0/1/2.glb)
```

- The artist delivers **source files only** (`.glb` + textures). LODs, VAT textures and compression are produced by the tools; do not deliver LODs by hand.
- After baking, the tool prints the triangle and vertex counts of every LOD and the VAT size — this is how a delivery is checked against the budgets below.
- The renderer draws units with instancing: up to **~100,000 soldiers** in a scene. Every rule for units follows from this number.

### Folder layout and names

| What | Where |
|---|---|
| Units (all skins) | `art/source/<faction>/units/<unit>/<skin>.glb` + `<skin>_albedo.png` |
| Buildings | `art/source/<faction>/buildings/<building>.glb` + textures |
| Shared / neutral models | `art/source/shared/<group>/<name>.glb` (outposts, environment, projectiles) |
| Terrain textures | `art/source/terrain/<material>_albedo.png`, `_normal.png` |
| Effect textures | `art/source/effects/<name>.png` |
| Icons, HUD, cursors | `art/source/ui/<group>/<id>.png` |
| Concept art | `docs/lore/images/<faction>/units/<unit>/<skin>.png` (same folder names as the models) |

Factions: `empire`, `high-elves`, `dark-elves`, `alkars`, `dwarves`. Unit and building ids: snake_case, as in [`ART_LIST.md`](ART_LIST.md) (for implemented content they match `content/`). Node, material and file names: Latin letters, digits and `_` only.

## 2. Common rules for all 3D models

| Rule | Value |
|---|---|
| Format | `.glb` (glTF 2.0), one model per file |
| Scale | 1 unit = 1 meter, transforms applied (scale 1, no rotation on the root) |
| Axes | Y up, **model front is +Z** |
| Origin | center of the base, on the ground (y = 0) |
| Size | by the footprint (buildings) or the unit radius (units), within ~10%; see the size tables below |
| Geometry | closed, no hidden faces, no faces under the ground, no duplicate vertices, no n-gons (triangles / quads) |
| Moving parts | separate **named nodes**, pivot on the real axis (hinge, wheel axle, arm axle) |
| LOD | made automatically (3 levels); keep the silhouette readable with simple shapes so the far LODs survive simplification |
| Style | stylized, readable from the RTS camera (top-down ~45–60°, 30–150 m away): clear silhouettes, large shapes, exaggerated weapons and heads, contrasting colors between factions |

## 3. Animated units (VAT, `tools/vatbake`)

Units are baked into Vertex Animation Textures: every frame of every clip is stored as vertex positions in a texture, and all units of one type are drawn with one instanced draw call per LOD. The GPU cost is driven by **vertex count × number of units**, so the budgets are strict.

### Geometry

| Rule | Value |
|---|---|
| Mesh | **one** skinned mesh, one skeleton; rider and mount are one model with one skeleton |
| Skin weights | ≤ 4 bones per vertex |
| Bones | ≤ 60 (the skeleton does not exist at runtime, but the source must stay sane) |
| Normals | not used: shading is flat, computed on the GPU |
| Team color | marked areas (cloak, shield face, banner, trim) — see [Textures](#5-textures) |

LOD0 budgets (counted after vertex welding). LOD1 is 40% and LOD2 10% of LOD0, made automatically: LOD0 is drawn closer than 45 m from the camera, LOD1 up to 110 m, LOD2 beyond.

| Category | Triangles | Vertices | Typical height |
|---|---|---|---|
| Infantry, mages, workers (humans, elves) | ≤ 3,000 | ≤ 2,000 | 1.8 m (elves 1.9 m) |
| Dwarves (infantry, workers) | ≤ 3,000 | ≤ 2,000 | 1.3–1.4 m |
| Goblins, skeletons | ≤ 2,000 | ≤ 1,400 | 1.1 m (goblins), 1.8 m |
| Orcs, berserkers | ≤ 3,500 | ≤ 2,400 | 2.1 m |
| Cavalry and riders (horse, tiger, wolf, bear, spider, griffon) | ≤ 5,000 | ≤ 3,500 | 2.5–3 m |
| Large creatures (siege troll, demon, golem, elementals, harpy) | ≤ 6,000 | ≤ 4,000 | 2–4.5 m |
| Heroes | ≤ 8,000 | ≤ 5,000 | as their kind (mounted heroes as cavalry) |
| Dark dragon | ≤ 10,000 | ≤ 6,500 | wingspan ~14 m |

The current placeholder swordsman is 5,900 triangles — twice the infantry budget.

### Clips

Frame rate 15 fps. Total length of all clips ≤ 12 s per model (≈ 180 frames). The VAT of one model must stay **≤ 3 MB** (vertices × frames × 8 bytes; 2,000 vertices × 180 frames ≈ 2.9 MB).

| Clip | Loop | Duration | For |
|---|---|---|---|
| `idle` | yes | 1–2 s | everyone |
| `walk` | yes | ~1 s | everyone |
| `run` | yes | ~0.7 s | everyone (gallop for cavalry, flight for flyers) |
| `attack` | no | 0.6–1 s | everyone who attacks; a shot for ranged units |
| `attack_ranged` | no | 0.6–1 s | units with both melee and ranged attacks (tiger rider, orc hunter, gunner, harpy) |
| `die` | no | 1–1.5 s | everyone |
| `cast` | no | ~1 s | mages, priests, healers, acolytes, heroes |
| `work` | yes | ~1 s | workers: construction, repair |
| `brace` | yes | ~1 s | stances: shield wall, spear wall |

Walk and run cycles are in place (no root motion); feet must not slide visibly at the unit's speed. If a clip is missing, `vatbake` substitutes a procedural one — fine only for placeholders.

### Regional skins

Every unit of a faction comes in one skin per region of that faction:

| Faction | Skins | Skin ids |
|---|---|---|
| Empire | 5 | `central`, `northern`, `western`, `southern`, `eastern` |
| High Elves (Liénari) | 3 | `eldralinna`, `vaelralinna`, `merilinna` |
| Dwarves (Dragori) | 3 | `monsarx`, `jarngrod`, `skopeskala` |
| Dark Elves (Shaessari) | 2 | `melasarach`, `aterliman` |
| Alkars | 1 | `alkaris` |

Rules:
- All skins of a unit use **the same skeleton, the same clips, the same size and pivot**. Animations are made once per unit and reused by every skin.
- A skin may change the mesh (armor shape, helmet, cloak) — each skin is then baked as its own VAT model and has its own budget.
- Only the skins in use are loaded in a match (one skin per player), so the number of skins does not raise the in-game memory.
- Summoned units and elementals have one skin.
- In PvP the player will choose the skin in the lobby (later); in the campaign the skin is fixed per mission.

## 4. Static models: buildings, machines, props (`tools/meshbake`)

| Rule | Value |
|---|---|
| Materials | ≤ 4 per model recommended, ≤ 8 maximum (each is a separate draw call); shared atlases per faction welcome |
| Textures | see [Textures](#5-textures) |
| Far LODs | flat-shaded models with UV seams are simplified with the coarse `sloppy` mode; build simple, chunky shapes from the start |
| Construction | the building must read well when only its lower part is visible (it rises during construction) |

LOD0 budgets:

| Category | Triangles | Footprint |
|---|---|---|
| Wall segment | **≤ 3,000** | 6.2 × 7 m (or ~2.4 m thick, two rows back to back), walkway at 5.6 m; ~140 on the map. The current one is 16,600 |
| Gatehouse | ≤ 15,000 | 18.6 × 7 m, passage ≥ 5 m |
| Wall tower | ≤ 10,000 | ≤ 7 × 7 m (tower slot), weapon visible from above |
| Watchtower / field tower | ≤ 8,000 | ≤ 7 × 7 m, squad platform at ~7.6 m |
| Building S | ≤ 10,000 | ≤ 9 × 9 m |
| Building M | ≤ 20,000 | ≤ 12 × 12 m |
| Building L | ≤ 25,000 | ≤ 16 × 16 m |
| Keep | ≤ 40,000 | 14 × 14 m |
| Outposts (neutral) | ≤ 20,000 | 8 × 8 m |
| Machines (ballista, trebuchet, mortar, airship) | ≤ 15,000 | by unit radius; airship ≤ 12,000 |
| Construction scaffolding S / M / L | ≤ 3,000 / 5,000 / 7,000 | as the plot |
| Ruins S / M / L | ≤ 4,000 / 6,000 / 8,000 | as the plot |
| Trees | ≤ 1,500 (≤ 300 for far copies) | — |
| Rocks, props | ≤ 800 | — |
| Projectiles | ≤ 100 (arrow, bolt), ≤ 300 (stone, shell) | — |

**Moving node names** (the code finds them by name suffix):
- `_wheel_…` — wheels, rotate around local X;
- `_arm` — trebuchet or catapult arm, around X;
- `_counterweight`, `_sling` — hang from the arm;
- `_crossbow`, `_ram_log`, `_barrel` — recoil or strike along −Z;
- `_gate_left`, `_gate_right`, `_portcullis` — gate doors (around Y on the hinge) and portcullis (up);
- `_sliding_door_…` — sliding doors (along X);
- `_propeller_…` — airship propellers, around local Z.

Machines also need a wreck model (`<id>_wreck`, ≤ 30% of the machine's budget).

## 5. Textures

### Units

> ❓ **Not decided yet.** The goal: good-looking textures while keeping ~100,000 units on screen. The current renderer has no textures on units: material colors are baked into vertex colors. The recommended option below needs engine work (`vatbake` and the unit shader) before textured units can be shown.

| Option | Look | Cost at 100k units | Engine work |
|---|---|---|---|
| A. Vertex colors only (current) | flat low-poly, no painted detail | lowest | none |
| **B. Hand-painted albedo + vertex colors far away (recommended)** | painted detail (faces, armor trim, heraldry) up close | one texture fetch per pixel; vertex cost unchanged | texture arrays in the unit shader, UVs in `vatbake`, LOD2 colors baked from the texture |
| C. Full PBR (albedo, normal, roughness) | most detailed | normals/tangents double the VAT, many more texture fetches | large; not compatible with the budgets above |

Rules for option B (what the artist prepares now, so that the art works either way):
- **One UV set**, one texture per unit skin: **`<skin>_albedo.png`, 512 × 512** (heroes, cavalry and large creatures 1024 × 1024, the dark dragon 1024 × 1024). No overlapping UVs except mirrored halves; 4 px padding between islands.
- **Hand-painted albedo** with light and ambient occlusion painted in (no normal or roughness maps on units). sRGB, 8-bit PNG.
- **Team color mask** in the alpha channel of the albedo: 255 = fully team-colored, 0 = no team color. The team color tints the painted value, so team areas are painted in neutral grey tones.
- **Materials:** still one per region of the model with a flat base color (it is used for the far LOD and as a fallback); name team-colored materials `…_team`.
- Runtime (engine side, later): all albedo textures of the loaded units go into one texture array, compressed to KTX2 with mipmaps; LOD2 uses vertex colors averaged from the texture, so far units do no texture reads.

### Buildings, machines, props

- Hand-painted or stylized PBR — the same style as the units. Albedo (sRGB) + optional normal map (linear); no other maps.
- Size: S and M buildings, towers, machines **1024 × 1024**; L buildings, keep, gatehouse up to **2048 × 2048**; wall segment **512 × 512** (or a shared faction atlas).
- Team color: separate material named `…_team` (flags, banners, roof trim).
- A faction's buildings should share one or two atlases (stone, wood, metal, cloth), so that one material serves many buildings.

### Terrain

Tileable materials, **1024 × 1024**, albedo + normal (height in the alpha of the normal, optional). Seamless on all sides, no strong unique details (they repeat every 4–8 m). The current terrain is vertex-colored; textured terrain needs engine work.

### Effects

Particle and decal textures: PNG with alpha, power-of-two sizes, **≤ 256 × 256** per frame; animated effects as a flipbook sheet ≤ 1024 × 1024 (e.g. 4 × 4 frames of 256). Additive effects on black background are allowed. Effects are colored by element: fire orange, ice/water turquoise, lightning blue-white, acid yellow-green, poison green, necrotic dark green, light gold-white.

## 6. Icons, HUD, cursors

PNG, transparent background, one unified style (bronze and stone frames are drawn by the code — icons contain only the picture). File name is the id.

| Kind | Size | Note |
|---|---|---|
| Unit and hero portraits | 256 × 256; heroes also 512 × 512 | one per unit (not per skin) |
| Buildings | 128 × 128 | |
| Abilities, upgrades, orders | 128 × 128 | ultimates get a gold frame in code |
| Resources | 64 × 64 | |
| Statuses | 32 × 32 | small badges, must read at 16 px |
| Faction crests | 256 × 256 | player banner, faction selection |
| Cursors | 32 × 32 and 64 × 64 | click point as on the placeholders ([`UX.md`](UX.md)) |
| HUD frames | 9-slice PNG | minimap ring, resource plate, command card frame, portrait medallion, tooltip |

## 7. Delivery checklist

For every model:
1. File at the right path with the right name; one model per file.
2. Scale, axes and origin as in section 2; size matches the table.
3. Triangle / vertex counts within budget (checked by the bake log).
4. Units: all required clips with the exact names; one skeleton; ≤ 4 weights per vertex; same skeleton and clips in every skin.
5. Moving parts as separate nodes with the right suffixes and pivots.
6. Textures at the listed sizes; team color marked (alpha mask for units, `…_team` material for buildings).
7. A screenshot from the game camera (top-down ~50°, 40 m and 100 m away) to check readability.
