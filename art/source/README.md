# Art sources (not in git)

Everything in `art/source/` except this file is **ignored by git** (`.gitignore`). The sources
are big binaries (high-poly models, `.blend` files, textures, animation imports) and would
bloat the repository. Only this README is committed: it says what lives here and where.

## Where the files are

| Location | Status |
|---|---|
| Local working copy: `<repo>/art/source/` on the developer's machine | **current — the only copy, back it up** |
| NAS on the local network | planned; when it exists, mount or symlink it as `art/source/` and note the path here |

The tools read sources by repo-relative paths (`art/source/...` in `tools/vatbake/models.json`
and `tools/meshbake/models.json`), so whatever the storage is, it must appear at this path.

## What the game needs — and what it doesn't

- **Runtime** uses only the baked output in `frontend/public/models/<id>/` (VAT, geometry,
  albedo, LODs). That folder **is committed**, so a fresh clone builds and runs without
  `art/source/`.
- **Sources here** are needed only to change a model: edit → export → re-bake
  (`tools/vatbake`, `tools/meshbake`) → commit the new `frontend/public/models/<id>/`.

## Layout

Full layout and naming rules: [`docs/ASSETS.md`](../../docs/ASSETS.md) (see "Unit source layout");
what has to be made: [`docs/ART_LIST.md`](../../docs/ART_LIST.md). Concept art is not here —
it is committed in `docs/lore/images/`.

- `<faction>/units/<unit>/` — units (`empire`, `high-elves`, `dark-elves`, `alkars`, `dwarves`):
  `highpoly/`, `lowpoly/<skin>.blend`, `rig/rig.blend`, `animations/animations.blend`,
  `animations/source/`, `textures/<skin>_albedo.png` and the exported `<skin>.glb`.
- `<faction>/buildings/<building>.glb` — buildings and towers of the faction.
- `bases/` — base meshes to start new models from.
- `shared/` — neutral outposts, environment, projectiles.
- `terrain/`, `effects/`, `ui/` — terrain materials, effect textures, icons / HUD / cursors.

A fresh clone has none of these folders; create them as needed.

## Inventory

Update this table when files are added or removed (`find art/source -type f -not -name .DS_Store | sort`).

### Units

| Files | Used by | Notes |
|---|---|---|
| `empire/units/swordsman/central.glb` | (not baked; the game uses `swordsman_simple`) | export of the files below (`tools/blender/export_unit.py`) |
| `empire/units/swordsman/lowpoly/central.blend` | export | mesh, UVs, weights; rig linked from `rig/rig.blend` |
| `empire/units/swordsman/rig/rig.blend` | lowpoly, animations | skeleton (Mixamo, 33 bones) |
| `empire/units/swordsman/animations/animations.blend` | export | clips: idle, run, attack, die, brace, walk_brace |
| `empire/units/swordsman/animations/source/mixamo_*.fbx` | — | Mixamo imports the clips were made from (block, death, idle, run, slash, strafe) |
| `empire/units/swordsman/textures/central_albedo.png` | `vatbake` | 512×512, alpha = team mask; RGB baked from `central_high.glb` (Meshy) |
| `empire/units/swordsman/textures/central_albedo_meshy_1024.png` | — | the raw 1024×1024 bake from `central_high.glb` (before downscaling and adding the team mask) |
| `empire/units/swordsman/textures/central_albedo_trellis.png` | — | the previous albedo, baked from the TRELLIS.2 high-poly (backup) |
| `empire/units/swordsman/highpoly/central.glb` | texture bake | TRELLIS.2 generation from the concept (13 MB) |
| `empire/units/swordsman/highpoly/central_posed.glb` | texture bake | the same, moved into the low-poly's pose (20 MB) |
| `empire/units/swordsman/central_high.glb` | texture bake (current source) | Meshy generation "Crimson Sentinel" from the concept: 79k triangles, 3 × 2048² PBR textures (9 MB) |

### Machines, buildings

None: the stand-in buildings and siege machines are generated as code (`art/props/`, `tools/propc`); the downloaded placeholders were removed.

### Other

| File | Used by | Notes |
|---|---|---|
| `bases/humanoid.blend` | — | humanoid base mesh (the swordsman's low-poly started from it) |
