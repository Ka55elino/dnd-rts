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
- `buildings/`, `grab3d/` — current placeholders (referenced by the bake configs).

A fresh clone has none of these folders; create them as needed.

## Inventory

Update this table when files are added or removed (`find art/source -type f -not -name .DS_Store | sort`).

### Units

| Files | Used by | Notes |
|---|---|---|
| `empire/units/swordsman/central.glb` | `vatbake` → `swordsman` | export of the files below (`tools/blender/export_unit.py`) |
| `empire/units/swordsman/lowpoly/central.blend` | export | mesh, UVs, weights; rig linked from `rig/rig.blend` |
| `empire/units/swordsman/rig/rig.blend` | lowpoly, animations | skeleton (Mixamo, 33 bones) |
| `empire/units/swordsman/animations/animations.blend` | export | clips: idle, run, attack, die, brace, walk_brace |
| `empire/units/swordsman/animations/source/mixamo_*.fbx` | — | Mixamo imports the clips were made from (block, death, idle, run, slash, strafe) |
| `empire/units/swordsman/textures/central_albedo.png` | `vatbake` | 512×512, alpha = team mask |
| `empire/units/swordsman/highpoly/central.glb` | texture bake | TRELLIS.2 generation from the concept (13 MB) |
| `empire/units/swordsman/highpoly/central_posed.glb` | texture bake | the same, moved into the low-poly's pose (20 MB) |
| `grab3d/ranger.glb` | `vatbake` → `ranger` | placeholder, grab3d.com Character Mixer, CC0 1.0 |

### Machines, buildings (placeholders)

All are development placeholders from free model sites; for the release they will be replaced with original art.

| File | Used by | Source / license |
|---|---|---|
| `grab3d/ballista.glb`, `trebuchet.glb`, `catapult.glb`, `battering-ram.glb` | `meshbake` | free model site, open license (siege machines; wheels / arm / bow are separate nodes) |
| `buildings/keep.glb` | `meshbake` | grab3d.com, CC0 1.0 (keep, outposts) |
| `buildings/mausoleum.glb` | `meshbake` | grab3d.com, CC0 1.0 (plot buildings) |
| `buildings/witch-cottage.glb` | `meshbake` | grab3d.com, CC0 1.0 (free buildings) |
| `buildings/castle-wall.glb` | `meshbake` | free model site, open license (base wall segments) |
| `buildings/gatehouse-light.glb` | `meshbake` | free model site, open license (gatehouse; leaves and portcullis are separate nodes) |
| `buildings/castle-tower.glb` | `meshbake` | free model site, open license (ballista / trebuchet towers) |
| `buildings/wooden-watchtower.glb` | `meshbake` | free model site, open license (archer tower) |
| `buildings/tea-house-light.glb` | `meshbake` | free model site, open license (workshop) |

### Other

| File | Used by | Notes |
|---|---|---|
| `bases/humanoid.blend` | — | humanoid base mesh (the swordsman's low-poly started from it) |
