# Models as code: pipeline design

Status: `draft` — prototype working (`tools/modelc`, test model `proto_knight`); the old pipeline (Blender → `vatbake`) stays until the new one is accepted.

## 1. Goals

- **Models are text.** Every unit is described by formulas and parameters in a Python file — readable, editable, diffable in git. No `.blend`, no `.fbx`, no high-poly models, no baking.
- **Animations are text.** Clips are formulas (cycles) and key poses (actions), written by hand or by AI. No Mixamo.
- **Model and texture are separate.** The model defines the shape and its UV atlas; the look comes from a PNG painted for that atlas. Semi-realism comes from the texture.
- **The only binary source is the PNG.** The compiled output (`frontend/public/models/<id>/`) is generated and committed, as today.
- **The engine does not change.** The compiler writes exactly the files `vatbake` writes; renderer, instancing, LODs and Model Overview stay.

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
art/models/proto/proto_knight.py ──┐
art/skeletons/humanoid.py ─────────┤
art/anims/humanoid.py ─────────────┤   python3 tools/modelc/modelc build proto_knight
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
- Team color: by default the parts with `team=True` are tinted with the player color. With `Model(team_tint=False)` nothing is tinted (the painted colors stay) and the player is shown by the colored ring the game draws under every soldier — the choice for semi-realistic units (`proto_knight`).
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

## 8. Test model: `proto_knight`

`art/models/proto/proto_knight.py`: a shape study after the BFME2 look (a Gondor-style lord): slender proportions, a narrow cuirass and a gorget up to the jaw, one smooth pauldron plate over the shoulder and down the upper arm (its lames are painted), long flared vambraces, leather gloves, a long black surcoat in two panels (wrapped `Extrude`s, each weighted to the hips and its own thigh) open at the front over a mail skirt, mail sleeves and chausses, knee cops, greaves, leather boots, bare head with long hair (the crown is painted), a long straight sword and a scabbard at the left hip. No texture (`texture=None`: flat part colors). ~1,940 triangles. Rule: a part is geometry only when it changes the silhouette; plate edges, lames, ridges and ornaments are painted. The earlier Empire swordsman version (Corinthian helmet, crest, shield) is in git history; its texture stays in `art/textures/proto/proto_knight_v2/`.

## 9. Next steps

1. Look at `proto_knight` in Model Overview: shapes, joints when bending, clips, LODs.
2. Primitives `Shell` / `Revolve`, modifiers; nicer pauldrons and helmet.
3. A fixed humanoid UV atlas and a template with front/back silhouettes; the first painted texture.
4. More clips: `brace`, `walk_brace`, `cast`, `work`; spear and bow attacks.
5. Port the swordsman from the concept; compare with the current one; decide whether to switch.
6. Optional: Jupyter + PyVista lab for shaping parts interactively.
