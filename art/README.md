# Art sources

Source assets (not shipped). Baked, game-ready versions are produced by `tools/vatbake`
(animated characters) and `tools/meshbake` (static models) into `frontend/public/models/`.

## Layout

Full layout and naming rules: [`../docs/ASSETS.md`](../docs/ASSETS.md); what has to be made: [`../docs/ART_LIST.md`](../docs/ART_LIST.md).

- `source/<faction>/units/<unit>/<skin>.glb` (+ `<skin>_albedo.png`) — units by faction (`empire`, `high-elves`, `dark-elves`, `alkars`, `dwarves`), one file per regional skin.
- `source/<faction>/buildings/<building>.glb` — buildings and towers of the faction.
- `source/shared/` — neutral outposts, environment, projectiles.
- `source/terrain/`, `source/effects/`, `source/ui/` — terrain materials, effect textures, icons / HUD / cursors.
- `source/buildings/`, `source/grab3d/` — current placeholders (referenced by the bake configs).

All current assets are development placeholders from free model sites with open licenses; for the release they will be replaced with original art.

| File | Source | License |
|---|---|---|
| `source/grab3d/swordsman.glb` | grab3d.com Character Mixer | CC0 1.0 |
| `source/grab3d/ranger.glb` | grab3d.com Character Mixer | CC0 1.0 |
| `source/buildings/keep.glb` | grab3d.com | CC0 1.0 (placeholder: keep, outposts) |
| `source/buildings/mausoleum.glb` | grab3d.com | CC0 1.0 (placeholder: plot buildings) |
| `source/buildings/witch-cottage.glb` | grab3d.com | CC0 1.0 (placeholder: free buildings) |
| `source/buildings/castle-wall.glb` | free model site | open license (placeholder: base wall segments) |
| `source/buildings/gatehouse-light.glb` | free model site | open license (placeholder: gatehouse; leaves and portcullis are separate nodes) |
| `source/buildings/castle-tower.glb` | free model site | open license (placeholder: ballista / trebuchet towers) |
| `source/buildings/wooden-watchtower.glb` | free model site | open license (placeholder: archer tower) |
| `source/buildings/tea-house-light.glb` | free model site | open license (placeholder: workshop) |
| `source/grab3d/ballista.glb`, `trebuchet.glb`, `catapult.glb`, `battering-ram.glb` | free model site | open license (placeholder: siege machines; wheels / arm / bow are separate nodes) |
