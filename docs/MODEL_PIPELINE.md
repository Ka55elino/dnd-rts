# Models as code: pipeline design

Status: `in use for stand-in models` — `tools/modelc` builds simple units from plain shapes (boxes, cylinders, spheres) to develop game mechanics with readable silhouettes. Final art goes the classic way (low poly → high poly → UV → bake, then `vatbake`); a shape study showed that models as code cannot reach BFME-level characters (proportions, organic forms).

## 1. Goals

- **Stand-ins fast.** A unit is a short Python file: plain shapes on the shared humanoid skeleton, a weapon, team-colored parts. Build in a second, see it in the game and in Model Overview.
- **Animations are text.** Clips are formulas (cycles), key poses and stances solved with IK (`art/anims/humanoid.py`); they work for any model on the humanoid skeleton.
- **Same output as `vatbake`.** The compiler writes the files the game already reads (`frontend/public/models/<id>/`); renderer, instancing, LODs and Model Overview are shared.
- Texture tools (templates, `fit` for AI-painted atlases) stay available but stand-ins use flat part colors (`texture=None`).

Why Python: an open "Wolfram": SymPy (symbolic math), NumPy (fast geometry), Jupyter + PyVista (an interactive lab, later), build123d (parametric CAD for hard-surface shapes, later). Python is needed only on the developer's machine at build time; the game never sees it. YAML was rejected: formulas would be unchecked strings with YAML's quoting traps and no functions.

## 2. Layout

```
art/                                   committed: text + PNG textures
├── skeletons/humanoid.py              bones as data (A-pose), .L mirrored to .R
├── anims/humanoid.py                  idle / walk / run (formulas + IK), attack_sword / die (key poses)
├── models/<group>/<id>.py             a model: parts, clips, skins, texture
├── textures/<group>/<id>/<skin>.png   albedo painted on the model's UV atlas
└── build/                             generated, not in git: UV templates, OBJ snapshots

tools/modelc/                          the compiler (Python): README, requirements.txt, modelc/
frontend/public/models/<id>/           compiled output, committed (model.json, geometry.bin, vat.bin, albedo.bin)
```

`art/source/` (not in git) keeps the old pipeline's files; it is untouched.

## 3. The full path: from a file to the model in the game

```
art/models/simple/swordsman_simple.py ┐
art/skeletons/humanoid.py ─────────┤
art/anims/humanoid.py ─────────────┤   python3 tools/modelc/modelc build swordsman_simple
art/textures/…/central.png ────────┘
  1. run the model file → Model (skeleton, parts, clips, skins, texture)
  2. geometry: every part evaluates its formulas → triangles; mirror="x" adds the other side
  3. UV atlas: one region per part (or a shared region name), sized by surface area, shelf-packed
  4. weights: rigid parts → their bone; parts with several bones → blend by distance (joints)
  5. budget check (triangles)
  6. texture: the PNG (or a generated placeholder) + team mask from the parts marked team=True
  7. animations: every clip sampled at 15 fps → bone matrices → skinned positions → VAT
  8. LOD1 / LOD2: simplified index buffers over the same vertices; far-LOD colors from the texture
  9. write frontend/public/models/<id>/ and add it to index.json
                                   │
content/units/<unit>.json  "model": "<id>"  ──►  VatModel loads it ──►  drawn instanced (as today)
```

Commands: [`tools/modelc/README.md`](../tools/modelc/README.md). `watch` rebuilds on every save, and Model Overview reloads the open model by itself (it polls the `build` stamp in `model.json`), keeping the camera, clip and time. Model Overview's timeline: play / pause (Space), frame step (← →), scrubbing, the frame number, and **Strip** — 6 poses of the clip side by side, seen from the side.

## 4. Skeleton

```python
HUMANOID = Skeleton([
    # name         parent        head              tail
    ("hips",       None,         (0, 0, 0.98),     (0, 0, 1.12)),
    ("upperarm.L", "chest",      (0.19, 0, 1.45),  (0.40, 0, 1.24)),
    ...
])
```

~20 bones (no fingers). Rest orientation of every bone is the identity, so a pose is a rotation in degrees around the bone's head in its parent's frame. Every humanoid model (humans, elves, orcs, dwarves) uses the same bone names, so the clips are shared; a body only changes proportions.

Axes: Z up, the character faces −Y, its left is +X. The writer converts to the game (Y up, front +Z).

## 5. Geometry: primitives and formulas

| Primitive | Defined by | Typical use |
|---|---|---|
| `Loft` | a path, radius `r` / `rx`, `ry` as functions of `t` (0..1), superellipse section `n`, `offset(t)`, segments, rings, caps, `shell` (thickness of an open end: a rim and a short inner wall, so an open tube is not see-through — the game draws one side of a triangle only; `shell_inner=False` = the rim only, when the inside is filled anyway) | torso, limbs, head, helmet, cuirass, greaves, blade |
| `Extrude` | a 2D outline + depth, placed by origin and two axes; `wrap=(rx, ry, flare[, angle])` bends it around the body (the face is then split into narrow triangles, so it follows the curve instead of cutting inside it) | shield, crest, strips, guard, plates |

Loft rings are carried along the path (parallel transport), so a path that turns a lot (a pauldron over the shoulder) does not twist. `modelc render <id> [clip@sec]` draws one-sided triangles like the game, so holes show up there before the game.

Later: `Revolve`, `Shell` (part of a superellipsoid — pauldrons, plates), `Box`, modifiers (`bend`, `taper`, `noise`), and build123d for complex hard-surface parts.

Formula values: a number, a Python function of `t`, `curve([(t, value), ...])` (smooth through control points), or a SymPy expression.

Common part options: `bone` (rigid) or `bones` (blend), `color` (far-LOD color and placeholder), `team=True`, `mirror="x"`, `uv="region"` (share a texture region), `material` (`"metal"`, `"gold"`, `"leather"`, `"cloth"` — the default —, `"skin"`, `"wood"`, or `(specular, gloss)`).

## 6. UV atlas and texture

- The compiler packs one rectangle per region (sized by surface area, aspect from the part's own U/V lengths). Mirrored parts share their region, so the left and right sides use the same pixels.
- `modelc template <id>` draws the sheet to paint on: every region with its name, the used area and the UV wireframe.
  - **Wrapped parts** (`Loft`: cuirass, greaves, helmet, limbs) are split into **FRONT | BACK** halves: the left half is the part seen straight from the front (the character's right on the left), the right half the part seen from behind. `top` marks the top. The picture is the *unrolled surface* — no perspective, no lighting from the side. (`Model(wrap_views=4)` gives FRONT | LEFT | BACK | RIGHT quarters, but AI tools do not keep four columns reliably, so every model uses two.)
  - **Flat parts** (`Extrude`: shield, crest, strips, guard) are marked `FACE`: the outline is drawn in the region, centred; paint the face inside it (both faces use it).
  - Parts hidden under armor (torso, head) get tiny regions; a plain color is enough there.
- The texture: `art/textures/<group>/<id>/<skin>.png`. Without a PNG the compiler paints a placeholder from the part colors.
- Shading: the compiler bakes the vertex normals of every frame (`normals.bin`; smooth across each `Loft`, hard edges on `Extrude` faces) and a material map over the atlas (`material.bin`, from each part's `material`). The game shades these models smoothly (Phong) with specular highlights where the material says metal; models without normals stay flat (Lambert). Model Overview can switch Smooth / Flat to compare.
- Lighting (`frontend/src/engine/lighting.ts`): presets Day / Evening / Overcast / Studio (hemisphere fill + a warm sun), filmic tone mapping, and a sun shadow map that follows the camera target. In the game only the near LODs (0 and 1) cast shadows — the shadow pass draws every instance again. Model Overview: Light, Background, Ground and Shadows controls.
- Team color: by default the parts with `team=True` are tinted with the player color. With `Model(team_tint=False)` nothing is tinted (the painted colors stay) and the player is shown by the colored ring the game draws under every soldier.
- Dark seams: AI tools draw each part as an object on a dark background, which would end up on the joins between FRONT and BACK and at the ends of wrapped parts. When the compiler loads the PNG it (1) crops every view to the box of its drawing and scales it to fill its half — one scale per axis, so the drawing is not distorted — and (2) fills what is still background (rounded corners of the silhouette) with a soft blur of the nearby paint, wrapping across the join and the outer edges. Background = pixels of the region's most common edge color connected to its border, so dark paint inside the drawing stays and a copied outline along the edge is ignored. The source PNG is not changed.
- For AI image tools use `modelc template <id> --clean`: no wireframe and no text (the tools copy both into the picture); the command prints the region list for the prompt.
- `modelc fit <id> <image.png>`: AI tools keep the rows and the order of the pieces but not their coordinates (pieces drift sideways by up to ~0.1, get wider or narrower). `fit` finds every piece of paint, splits each row of pieces between the row's regions (in order, matching widths), scales each piece into its rectangle, saves the result as the skin's PNG and locks the atlas. The prompt therefore only has to get the rows, the order and the content right; name the content of every region in it. Labels copied from the labelled template (region names, `top`/`back` on the dark margin above a piece) are removed. When a row is split wrong (an extra picture, a piece drawn in its neighbour's place), give the columns by hand: `--box blade=0.51,0.67` (fractions of the image width); the rest of the row is split automatically.
- Ways to make the PNG: paint on the template by hand, or fill regions with a 2D AI image tool.
- Prompt for 2D AI tools. Attach two images: `art/build/<id>/template_clean.png` (never `template.png` — the tools copy its grid and labels) and the concept art. Replace `<…>` with the unit's look and paste the region list printed by `modelc template <id> --clean`, adding one line of content per region:

  > Paint a 1024×1024 texture atlas for a low-poly real-time-strategy game unit. The first attached image is the layout: thin rectangles on a dark background. The second is the concept art to match: <short description of the unit>.
  >
  > LAYOUT — the most important rules: (1) Paint only inside the rectangles. Keep every rectangle exactly where it is, with the same size; do not add, remove, move, merge or resize rectangles, especially the small ones. (2) Outside the rectangles leave plain flat dark gray (#202024) — no extra objects, no copies of parts. (3) Draw no lines, frames, grids, labels, letters or numbers; the layout's outlines must not be visible. (4) Fill each rectangle edge to edge with the surface of its part — not an object floating on a background: no margins, no drop shadows, no vignette, no outline.
  >
  > WRAPPED REGIONS (a thin vertical line splits the rectangle into two equal halves): the surface wraps around the part like a label around a bottle. Left half: the part seen straight from the FRONT; right half: the same part seen from the BACK; the top edge is the top of the part. Flat, unrolled, orthographic: no perspective. Make each view as wide as its half; the two halves meet on the sides of the part, so colors, brightness and metal tone must match along the outer edges of both halves.
  >
  > FLAT REGIONS (no line): one flat face filling the whole rectangle, centered, seen straight on.
  >
  > LIGHTING AND MATERIAL: soft even light from slightly above, the same in every region; no light from one side, no strong cast shadows, no dark corners. Ambient occlusion only in real recesses (under plate edges, between lames, along seams and straps). Polished metal gets bright painted highlights and light edge wear — the game shows no reflections. Neutral colors, no blue or other tint over everything. Hand-painted semi-realistic style like a classic RTS (Battle for Middle-earth), readable from far away.
  >
  > REGIONS (x, y, width, height as fractions of the image, origin top-left) and their content: <one line per region, e.g. "foot — flat 0.01 0.88 0.13 0.11 — STEEL armored boot seen from the side, toe pointing right; not cloth">
  >
  > FINAL CHECK: exactly these filled rectangles at these positions, nothing outside them, no lines and no text anywhere.

  Mistakes seen so far, to check before using a result: grid and labels copied (wrong template attached), small regions moved, extra pictures between regions, wrong content (a cloth foot, a sword grip in the glove region), one half of a wrapped region darker than the other. Name the content of small or ambiguous regions explicitly ("a glove, not a grip"). A region that came out wrong can be pasted from a previous version (the rectangles are in `art/textures/<texture>/atlas.json`).
- Open: a fixed shared atlas for all humanoids (so a texture fits any humanoid model) instead of a per-model packing.

## 7. Animations: formulas and key poses

**Cycles** (`idle`, `walk`, `run`): a function of the phase `p` (0..1). Locomotion moves each foot along a formula path (planted half the cycle, then a lifted swing) and places the leg with two-bone IK, so the ground speed is exact: `2 * stride / duration`.

```python
def walk(duration=1.0, stride=0.62, lift=0.09, swing=22, bob=0.03, lean=3):
    ...  # foot(p) path, leg_ik for both legs, hips bob / sway, arms swing opposite
```

**Poses from targets** (world-space IK, `modelc.anim`): `plant_foot(sk, pose, side, ankle, yaw)` keeps a foot flat on the ground at a point, `place_hand(sk, pose, side, wrist, elbow, hand_rot)` puts a wrist at a point with the elbow bending toward a direction, `point_on(sk, pose, bone, p)` makes a target ride with a bone (hands follow the chest), `aim(axis, direction, up_axis, up)` orients a hand (a blade pointed at the enemy). Set the torso first, then solve the limbs. Stances in `art/anims/humanoid.py` are written this way — feet, body turn, shield wrist, sword wrist and blade direction as numbers with comments: `GUARD` (the base), `RELAXED` (idle: upright, the shield lowered to the side and turned out, the sword low) and `BRACE` (the raise-shields ability: the sword hand at the shoulder, the elbow out, the blade level over the shield, the front foot stepped forward, the shield high, the body bent forward). `idle` is `RELAXED` plus breathing, a weight shift and a small circle of the sword point. `brace` is a `Motion` (one-shot clip from a function) that interpolates the stance *parameters* from `RELAXED` to `BRACE`, not bone angles, with everything moving together on one easing curve (`ease=(x1, y1, x2, y2)`, the same as CSS `cubic-bezier`): the shield rises (a little higher on the way), the body drops and turns, the front foot steps forward (lifted mid-step), the sword rises over the shield; the game holds its last pose. `unbrace` = `Reverse(brace)`.

**Actions** (`attack`, `die`, `cast`, `brace`, `work`): key poses with times and easing (`linear`, `in`, `out`, `in_out`, `out_back`, or CSS-like cubic-bezier points `(x1, y1, x2, y2)`); `Model(fps=30)` bakes 30 samples per second instead of 15 for fast eased moves (twice the VAT size); bones are interpolated with slerp.

```python
Action(0.8, [(0.00, READY, "linear"), (0.32, windup, "in_out"), (0.47, strike, "in"), (0.80, READY, "in_out")])
```

Parameters give every unit its own character (a heavy soldier: short stride, more bob; an elf: long stride). Compared with motion capture the motion is cleaner and more "designed"; from the RTS camera this fits the genre (BFME's units were hand-keyed).

## 8. Stand-in models

`art/models/simple/`, one short file per unit, built from the kit `art/models/simple/kit.py`:

- shapes: `box(id, a, b, rx, ry)`, `cyl(id, a, b, r)`, `cone(id, a, b, r0, r1)`, `ball(id, c, r)` — `Loft`s with few segments; a flat outline is an `Extrude`;
- `humanoid(torso, legs, arms, …, head="helmet"|"hood"|"hat"|"bare"|"crown", tabard, robe, sash, build)` — a body on the humanoid skeleton (A-pose); `tabard` / `sash` are team-colored;
- weapons in the right hand (grip `G`, the business end along −Z): `sword`, `spear`, `lance`, `staff(head_color)`, `hammer`, `mace`; in the left hand: `shield` (team-colored face), `bow` (+ quiver);
- cavalry: `horse(coat, mane, barding, scale, beak)` on a mount skeleton from `art/skeletons/cavalry.py` — `mount(scale)`: the mount's body is the root, the humanoid is lifted onto the saddle (`sk.rider`); `CAVALRY` (horse, griffon, beast), `WOLF` (×0.62); `mount_rider(parts, sk)` moves humanoid parts onto the saddle; `wings(color)` for griffons, `arm_wings(color)` for harpies; `elemental(core, glow)`; `axe()`.
- `Model(scale=…)`: the baked output is scaled (a big hero, an elemental); shapes and clips stay authored at human size.

Stances and clips (`art/anims/humanoid.py`): every stance places both hands with IK — `shield_*` = the left hand (shield, bow, free hand), `sword_*` = the weapon hand, `sword_aim` = where the weapon's business end points.

| Stance set | Units | Clips |
|---|---|---|
| `RELAXED` / `BRACE` (sword or mace + shield) | swordsman, paladin | idle, walk, run, attack (`attack_sword`: overhead cut), brace / unbrace (shield wall), die |
| `SPEAR_RELAXED` / `SPEAR_BRACE` / `SPEAR_CARRY` | spearman | idle, walk / run carrying the spear upright, attack (`attack_spear`: thrust), brace / unbrace (spear wall: level over the shield), die |
| `BOW_RELAXED` / `BOW_AIM` / `BOW_CARRY` | archer | idle, walk / run with the bow, attack (`attack_bow`: nock, draw, release), die |
| `STAFF_RELAXED` / `STAFF_CARRY` | fire / water / lightning mages, healer, archmage | idle, walk / run with the staff, attack (`cast_staff`), die |
| `TOOL_RELAXED` | builder | idle, walk, run, attack (hammer swing, also the work), die |
| cavalry (`art/anims/cavalry.py`: `SWORD`, `LANCE`, `LANCE_COUCHED`) | knight (lance), emperor (sword) | idle, walk (four-beat), run (gallop), attack (`attack_lance` / `attack_sword` from the saddle), knight: brace / unbrace / walk_brace (wedge: the lance couched), die (the horse rolls over) |

| Model | Unit | Look |
|---|---|---|
| `swordsman_simple` | swordsman | steel torso and shins, helmet, tabard, sword, shield |
| `spearman_simple` | spearman | leather armour, steel forearm plates, helmet, tabard, spear, shield |
| `archer_simple` | archer | leather jacket, green hood, tabard, bow, quiver |
| `worker_simple` | builder | linen clothes, bare head, tabard (apron), hammer |
| `fire_mage_simple` / `water_mage_simple` / `battlemage_simple` | mages | robe and hood (dark red / blue-turquoise / purple), steel bracers (fire, lightning), team sash, staff with a colored orb |
| `healer_simple` | healer | cream robe, bare head, team sash, staff with a golden orb |
| `archmage_simple` | archmage | purple robe, tall hat with a brim, team sash, a longer staff with a big orb |
| `paladin_simple` | paladin | white plate with a gold helmet, broader build, tabard, mace, shield |
| `knight_simple` | knight | armoured rider, brown horse with a team caparison, lance |
| `emperor_simple` | emperor | gold-and-steel rider with a crown, white horse with a team caparison, sword |
| `raider_simple` | raider (warband) | green skin, leather, axe |
| `warlord_simple` | warlord (warband hero) | big green warrior (×1.2), dark armour, horned helmet, axe, shield |
| `captain_simple` | captain (test hero) | steel and gold, sword, shield |
| `wolf_rider_simple` | wolf rider (warband) | green raider with an axe on a grey wolf (mount skeleton ×0.62) |
| `harpy_simple` | harpy (flying) | lean grey body, feathered wings along the arms; every moving clip is a wing beat (`fly`), attack = claw dive |
| `griffon_rider_simple` | griffon rider (flying) | rider with a sword on a griffon (tawny body, beak, wings on `wing.L/R`), `cavalry.fly` |
| `warbeast_simple` | warbeast (warlord's beast form) | a huge dark four-legged beast (×1.3), `beast_attack` |
| `fire_` / `ice_` / `storm_elemental_simple` | elementals | a column without legs, big shoulders and fists (×1.4); ice has shards; storm casts |

Other factions (one file per unit in `art/models/simple/`, `<unit>_simple`):

- **High Elves:** `spirit` (glowing figure without legs), `twinblade` (double-bladed sword + shield), `elf_archer`, `tiger_rider` (tiger mount `TIGER`), `water_elemental`, heroes `elf_ranger`, `elf_king` (white stag with antlers), `priestess`; `griffon_rider`, fire / storm elementals shared.
- **Dark Elves:** `webweaver`, `blade_dancer` (two blades), `crossbowman` (a crossbow in each hand, `GUN_*` stances), `scout`, `spider_rider` (spider mount `SPIDER`, `spider_body`), `dark_priest`, `dark_mage`, `dark_dragon` (×2, `dragon_parts`, flies), heroes `dark_paladin`, `dark_archer`, `spider_queen`.
- **Alkars:** `goblin` (×0.72), `orc`, `berserker` (great axe), `orc_hunter`, `orc_spearman`, `alkar_swordsman`, `alkar_archer`, `goblin_wolf_rider`, `dark_acolyte`, `siege_troll` (×1.7, club), heroes `dark_lord`, `alkar_dark_mage`, `wraith_king` (floating, crowned); `harpy` shared.
- **Dwarves** (×0.78, `build=1.35`, beards): `dwarf_worker`, `axe_dwarf`, `gunner` (`GUN_*`), `bear_rider` (bear mount `BEAR`), `golem` (×1.7), `airship` and `mortar` (rigid skeletons in `art/skeletons/machines.py`, clips in `art/anims/machines.py`), heroes `dwarf_king`, `engineer`, `runemaster`.
- **Traps** (rigid skeleton `TRAP`: a flat `base` and a `lid` that springs up; clips `trap_idle` / `trap_spring`): `mine` (a buried iron pot with a fuse spike), `poison_trap` (a green flask in ooze), `paralysis_trap` (open iron jaws with teeth), `rune_trap_fire` / `_ice` / `_lightning` (a carved stone with a glowing cone / orb / bar). The team color is a band, cork, plate or rim.

Add a unit: copy a file, change the body, weapon and stances, `python3 tools/modelc/modelc build <id>`, set `"model": "<id>"` in `content/units/<unit>.json`.

## 9. Buildings and machines as code (`tools/propc`)

Every building of every faction, the base wall and gate, the outposts and the siege machines are stand-ins built from plain shapes, like the units: `art/props/<faction>.py` (`empire`, `high_elves`, `dark_elves`, `alkars`, `dwarves`, `neutral`), shared pieces in `art/props/_style.py`. Each file lists `PROPS = [Prop(id, parts, doc), ...]`; the model id is the building id (`content/buildings/<id>.json` → `"model": "<id>"`), sized to the building's footprint (no `modelScale`).

```sh
python3 tools/propc/propc list                  # every prop and its one-line look
python3 tools/propc/propc build [id ...]        # all by default → frontend/public/models/<id>/lod0..2.glb + model.json + index.json
python3 tools/propc/propc render [id ...]       # software preview → art/build/<id>/render.png
```

- Shapes (`tools/propc/propc/shapes.py`, game axes: x right, y up, front +z, meters): `box`, `cyl` (cylinder / frustum / cone), `cone`, `rod` (between two points: posts, beams, axles, discs), `spike`, `sphere`, `dome`, `gable`, `hip`, `prism`; composites `battlements`, `crown`, `banner`, `flag`; `move`, `turn`, `mirror_x`, `scaled`, `tagged`. Faction helpers: `hall`, `door`, `windows`, `round_tower`, `fence`, `palisade`, `tree`, `target`, `logs`, `anvil`, `glow_pool`, `rocks`, `wall_row`.
- Options on every shape: `color`; `kind` = `matte` | `metal` | `glow` (emissive: runes, magic, fire) | `team` (the owner's colour — banners, flags, plates; the game tints it per owner); `detail` 0 / 1 / 2 (LOD1 drops detail 2, LOD2 also detail 1); `node` + `pivot` for moving parts.
- Output: flat-shaded triangles, one mesh per material for the static body; every named node is its own mesh at its pivot. Moving parts the game looks for: `gate_left` / `gate_right` / `portcullis` (gatehouse), `<id>_wheel_<n>` (roll about x), `<id>_arm` (trebuchet arm about x), `<id>_counterweight` / `<id>_sling` (follow the arm), `<id>_crossbow` (ballista bow, kicks back).
- The keep is shared by every faction but looks different per owner: `"factionModels": {"high_elves": "keep_high_elves", ...}` on the building (default `keep`, the Empire's).
- `castle_wall` is one row of a base wall: the game draws two back to back per segment, outer face +z, the walkway between them.
