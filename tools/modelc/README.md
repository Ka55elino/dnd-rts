# modelc — models and animations as code

Compiles units described in Python (formulas, key poses) into the files the game reads —
the same format as `tools/vatbake` (`frontend/public/models/<id>/`). Design: [`docs/MODEL_PIPELINE.md`](../../docs/MODEL_PIPELINE.md).

## Setup (once)

```sh
python3 -m pip install -r tools/modelc/requirements.txt     # numpy, sympy, pillow
```

## Commands (from the repo root)

```sh
python3 tools/modelc/modelc list                      # models in art/models/
python3 tools/modelc/modelc build proto_knight        # compile → frontend/public/models/proto_knight/ + index.json
python3 tools/modelc/modelc build --all
python3 tools/modelc/modelc watch proto_knight        # rebuild on every save (art/ and tools/modelc/); Model Overview reloads the model by itself
python3 tools/modelc/modelc render proto_knight idle@0 brace@0.7  # software-rendered PNG views in art/build/<id>/ (no Blender needed)
python3 tools/modelc/modelc fit proto_knight ai.png [--box leg=0.68,0.90]  # move the pieces of an AI-painted atlas into the UV rectangles, save as the skin, lock the atlas
python3 tools/modelc/modelc template proto_knight     # art/build/<id>/template.png: UV regions + wireframe, to paint on
python3 tools/modelc/modelc paint proto_knight        # placeholder texture → art/textures/<texture>/<skin>.png
python3 tools/modelc/modelc obj proto_knight rest walk@0.25 attack@0.47   # OBJ snapshots → art/build/<id>/
```

Then look at it in the game: main menu → **Model Overview**.

## Library (`tools/modelc/modelc/`)

| Module | What |
|---|---|
| `skeleton.py` | `Skeleton`: bones as data (name, parent, head, tail); `.L` bones are mirrored to `.R` |
| `geometry.py` | primitives: `Loft` (rings along a path, radius formulas, superellipse section), `Extrude` (2D outline + depth) |
| `model.py` | `Model` and `build()`: parts → one mesh, mirrored copies, UV atlas packing, skin weights, budget check |
| `anim.py` | `Cycle` (formula of the phase), `Action` (key poses + easing), `leg_ik` (two-bone IK), `merge` |
| `texture.py` | team mask from `team=True` parts, the template, the placeholder painter |
| `writer.py` | VAT, geometry (LOD0–2), albedo, `model.json`, `index.json` |
| `math3d.py` | rotations, quaternions, easing, `curve()` (smooth profile through control points) |

Formulas: any parameter that varies along a part (`r`, `rx`, `ry`, `offset`) can be a number, a Python
function of `t` (0..1 along the path), a `curve([(t, value), ...])` or a SymPy expression in `t`.

## Content (`art/`, committed)

```
art/skeletons/humanoid.py          the humanoid skeleton (A-pose)
art/anims/humanoid.py              idle, walk, run (formulas + IK), attack_sword, die (key poses)
art/models/proto/proto_knight.py   the test model
art/textures/proto/proto_knight/central.png
art/build/                         generated (not in git): templates, OBJ snapshots
```

Axes in model files: Z up, the character faces −Y, its left is +X (as in Blender). The writer converts
to the game's axes (Y up, front +Z).
