# Art production list

Everything that has to be drawn or modeled, by faction. Technical requirements (budgets, sizes, clips, textures, naming): [`ASSETS.md`](ASSETS.md). The look of each unit: the faction files in [`lore/factions/`](lore/factions/) (descriptions, appearance and Pinterest references in comments).

Celestials are not included yet. Building size classes and the ids of buildings, abilities and upgrades for factions other than the Empire are proposals: they become final when the faction is implemented in `content/`.

## Summary

| Faction | Skins | Unit and hero models (× skins) | One-skin models (summoned, elementals) | Buildings, towers, traps | Icons |
|---|---|---|---|---|---|
| Empire | 5 | 14 × 5 = **70** | 3 | 14 | 69 |
| High Elves (Liénari) | 3 | 8 × 3 = **24** | 4 | 13 | 48 |
| Dark Elves (Shaessari) | 2 | 11 × 2 = **22** | 1 | 15 | 52 |
| Alkars | 1 | 14 × 1 = **14** | 4 | 14 | 60 |
| Dwarves (Dragori) | 3 | 10 × 3 = **30** | 0 | 14 | 55 |
| **Total** | | **160** | **12** | **70** | **284** |

Unit models are counted per skin: each skin is a separate textured model with the same skeleton and clips (see [`ASSETS.md`](ASSETS.md), Regional skins). Animations are made once per unit: 69 animated or machine units in total. Icons are counted without the shared ones below.

## Empire

Skins: `central`, `northern`, `western`, `southern`, `eastern`. Sources: `art/source/empire/units/<unit>/<skin>.glb` + `<skin>_albedo.png`; concept art: `docs/lore/images/empire/units/<unit>/`.

### Units

Each unit: one model per skin (same skeleton and clips), one albedo texture per skin.

| id | Unit | Category · LOD0 tris / verts | Clips / nodes | Now |
|---|---|---|---|---|
| `worker` | Builder | Infantry · 3,000 / 2,000 | idle, walk, run, work, die | placeholder (swordsman) |
| `swordsman` | Swordsman | Infantry · 3,000 / 2,000 | idle, walk, run, attack, brace, die | placeholder (grab3d swordsman) |
| `spearman` | Spearman | Infantry · 3,000 / 2,000 | idle, walk, run, attack, brace, die | — |
| `archer` | Archer | Infantry · 3,000 / 2,000 | idle, walk, run, attack (shot), die | placeholder (grab3d ranger) |
| `knight` | Knight | Cavalry · 5,000 / 3,500 | idle, walk, run (gallop), attack, die | — |
| `ballista` | Ballista | Machine (static) · ≤ 15,000 tris (static model) | nodes `_wheel_…`, `_crossbow`; + `ballista_wreck` | placeholder (with nodes) |
| `trebuchet` | Trebuchet | Machine (static) · ≤ 15,000 tris (static model) | nodes `_wheel_…`, `_arm`, `_counterweight`, `_sling`; + `trebuchet_wreck` | placeholder (with nodes) |
| `fire_mage` | Fire Mage | Infantry · 3,000 / 2,000 | idle, walk, run, attack (projectile), cast, die | — |
| `water_mage` | Water Mage | Infantry · 3,000 / 2,000 | idle, walk, run, attack (projectile), cast, die | — |
| `lightning_mage` | Lightning Mage | Infantry · 3,000 / 2,000 | idle, walk, run, attack (projectile), cast, die | — |
| `healer` | Healer | Infantry · 3,000 / 2,000 | idle, walk, run, cast, die | — |

### Heroes

| id | Hero | LOD0 tris / verts | Clips |
|---|---|---|---|
| `paladin` | Paladin | 8,000 / 5,000 | idle, walk, run, attack, cast, die |
| `archmage` | Archmage | 8,000 / 5,000 | idle, walk, run, attack (projectile), cast, die |
| `emperor` | Emperor (mounted) | 8,000 / 5,000 | idle, walk, run (gallop), attack, cast, die |

### Summoned (one skin)

| id | Creature | Category · LOD0 tris / verts | Clips |
|---|---|---|---|
| `fire_elemental` | Fire Elemental (Archmage) | Large creature · 6,000 / 4,000 | idle, walk, attack, die |
| `ice_elemental` | Ice Elemental (Archmage) | Large creature · 6,000 / 4,000 | idle, walk, attack, die |
| `storm_elemental` | Storm Elemental (Archmage) | Large creature · 6,000 / 4,000 | idle, walk, attack (projectile), die |

### Buildings, towers, traps

Sources: `art/source/empire/buildings/<id>.glb`. One look per faction (no regional skins).

| id | Building | Class · LOD0 tris · footprint | Note |
|---|---|---|---|
| `keep` | Keep | Keep · ≤ 40,000 · 14×14 m |  |
| `barracks` | Barracks | M · ≤ 20,000 · ≤ 12×12 m |  |
| `archery_range` | Archery Range | M · ≤ 20,000 · ≤ 12×12 m |  |
| `stables` | Stables | M · ≤ 20,000 · ≤ 12×12 m |  |
| `workshop` | Workshop | L · ≤ 25,000 · ≤ 16×16 m | `_sliding_door_…` opens on recruitment |
| `mage_guild` | Mage Guild | L · ≤ 25,000 · ≤ 16×16 m | dome and four towers (fire, lightning, water, light), as in Kordkayer |
| `forge` | Forge | M · ≤ 20,000 · ≤ 12×12 m |  |
| `farm` | Farm | S · ≤ 10,000 · ≤ 9×9 m |  |
| `castle_wall` | Wall segment | ≤ 3,000 · 6.2×7 m | current model is 16,600 triangles |
| `castle_gate` | Gatehouse | ≤ 15,000 · 18.6×7 m | `_gate_left`, `_gate_right`, `_portcullis` |
| `archer_tower` | Archer tower | ≤ 10,000 · ≤ 7×7 m |  |
| `ballista_tower` | Ballista tower | ≤ 10,000 · ≤ 7×7 m | ballista visible on top |
| `trebuchet_tower` | Trebuchet tower | ≤ 10,000 · ≤ 7×7 m | trebuchet visible on top |
| `watchtower` | Watchtower (field) | ≤ 8,000 · ≤ 7×7 m | squad platform at 7.6 m |

Also in the faction's style: construction scaffolding S / M / L, ruins S / M / L, wall breach rubble.

**Projectiles:** arrow, ballista bolt, trebuchet stone.

### Icons

- **Portraits** 256×256: `worker`, `swordsman`, `spearman`, `archer`, `knight`, `ballista`, `trebuchet`, `fire_mage`, `water_mage`, `lightning_mage`, `healer`; summoned `fire_elemental`, `ice_elemental`, `storm_elemental`.
- **Hero portraits** 256×256 and 512×512: `paladin`, `archmage`, `emperor`.
- **Buildings** 128×128: `keep`, `barracks`, `archery_range`, `stables`, `workshop`, `mage_guild`, `forge`, `farm`, `castle_wall`, `castle_gate`, `archer_tower`, `ballista_tower`, `trebuchet_tower`, `watchtower`.
- **Unit abilities** 128×128: `shield_wall`, `spear_wall`, `fire_mode`, `fire_bolts_mode`, `wedge`, `fire_pillar`, `storm`, `chain_lightning`, `heal`.
- **Paladin** 128×128: `holy_radiance`, `purity`, `divine_smite`, `warding_aura`, `divine_shield`.
- **Archmage** 128×128: `element`, `element_fire`, `element_ice`, `element_lightning`, `elemental_ward`, `summon_elemental`, `arcane_focus`, `meteor_shower`.
- **Emperor** 128×128: `imperial_guard`, `imperial_march`, `inspire`, `imperial_oversight`, `mass_recall`.
- **Upgrades** 128×128: `forged_weapons_1`, `forged_weapons_2`, `reinforced_armor_1`, `reinforced_armor_2`, `fire_arrows`, `fire_bolts`, `improved_mechanism`.
- **Faction crest** 256×256.

## High Elves (Liénari)

Skins: `eldralinna`, `vaelralinna`, `merilinna`. Sources: `art/source/high-elves/units/<unit>/<skin>.glb` + `<skin>_albedo.png`; concept art: `docs/lore/images/high-elves/units/<unit>/`.

### Units

Each unit: one model per skin (same skeleton and clips), one albedo texture per skin.

| id | Unit | Category · LOD0 tris / verts | Clips / nodes | Now |
|---|---|---|---|---|
| `spirit` | Spirit (builder) | Infantry · 3,000 / 2,000 | idle, walk, run, work, die | temporary reference image |
| `twinblade` | Swordsman (double-bladed sword and shield) | Infantry · 3,000 / 2,000 | idle, walk, run, attack, die | — |
| `archer` | Archer | Infantry · 3,000 / 2,000 | idle, walk, run, attack (shot), die | — |
| `griffon_rider` | Griffon Rider | Cavalry · 5,000 / 3,500 | idle, walk, run (flight), attack, attack_ranged, die | — |
| `tiger_rider` | Tiger Rider | Cavalry · 5,000 / 3,500 | idle, walk, run, attack, attack_ranged, die | — |
| `fire_elemental` | Fire Elemental | Large creature · 6,000 / 4,000 | idle, walk, attack, die | one skin |
| `water_elemental` | Water Elemental | Large creature · 6,000 / 4,000 | idle, walk, attack, die | one skin |
| `storm_elemental` | Storm Elemental | Large creature · 6,000 / 4,000 | idle, walk, attack (projectile), die | one skin |

### Heroes

| id | Hero | LOD0 tris / verts | Clips |
|---|---|---|---|
| `elf_ranger` | Elf Ranger | 8,000 / 5,000 | idle, walk, run, attack (shot), cast, die |
| `elf_king` | Elf King (stag rider) | 8,000 / 5,000 | idle, walk, run (gallop), attack, cast, die |
| `priestess` | Priestess | 8,000 / 5,000 | idle, walk, run, attack (projectile), cast, die |

### Summoned (one skin)

| id | Creature | Category · LOD0 tris / verts | Clips |
|---|---|---|---|
| `ent` | Ent (Awakening of the Ents) | Large creature · 6,000 / 4,000 | idle, walk, attack, die |

### Buildings, towers, traps

Sources: `art/source/high-elves/buildings/<id>.glb`. One look per faction (no regional skins).

| id | Building | Class · LOD0 tris · footprint | Note |
|---|---|---|---|
| `keep` | Keep | Keep · ≤ 40,000 · 14×14 m |  |
| `order_of_the_sword` | Order of the Sword | M · ≤ 20,000 · ≤ 12×12 m |  |
| `order_of_archers` | Order of Archers | M · ≤ 20,000 · ≤ 12×12 m |  |
| `griffon_tower` | Griffon Tower | L · ≤ 25,000 · ≤ 16×16 m |  |
| `tiger_pen` | Tiger Pen | M · ≤ 20,000 · ≤ 12×12 m |  |
| `spirit_garden` | Spirit Garden | L · ≤ 25,000 · ≤ 16×16 m |  |
| `enchantment_tower` | Enchantment Tower | M · ≤ 20,000 · ≤ 12×12 m |  |
| `moonwell` | Moonwell | S · ≤ 10,000 · ≤ 9×9 m | food |
| `wall` | Wall segment | ≤ 3,000 · 6.2×7 m |  |
| `gate` | Gatehouse | ≤ 15,000 · 18.6×7 m | `_gate_left`, `_gate_right` |
| `archer_tower` | Archer tower | ≤ 10,000 · ≤ 7×7 m |  |
| `magic_tower` | Magic tower | ≤ 10,000 · ≤ 7×7 m | radiant |
| `sacred_tree` | Sacred Tree (Elf King ability) | ≤ 6,000 | summoned structure |

Also in the faction's style: construction scaffolding S / M / L, ruins S / M / L, wall breach rubble.

**Projectiles:** elven arrow.

### Icons

- **Portraits** 256×256: `spirit`, `twinblade`, `archer`, `griffon_rider`, `tiger_rider`, `fire_elemental`, `water_elemental`, `storm_elemental`; summoned `ent`.
- **Hero portraits** 256×256 and 512×512: `elf_ranger`, `elf_king`, `priestess`.
- **Buildings** 128×128: `keep`, `order_of_the_sword`, `order_of_archers`, `griffon_tower`, `tiger_pen`, `spirit_garden`, `enchantment_tower`, `moonwell`, `wall`, `gate`, `archer_tower`, `magic_tower`, `sacred_tree`.
- **Elf Ranger** 128×128: `elven_agility`, `forest_shadow`, `hail_of_arrows`, `eagle_eye`, `sniper_shot`.
- **Elf King** 128×128: `thicket`, `child_of_the_forest`, `sacred_tree`, `power_of_growth`, `awakening_of_the_ents`.
- **Priestess** 128×128: `moonbeam`, `blessing_of_the_moon`, `cleansing`, `aura_of_the_moon_shield`, `sacred_weapons`.
- **Upgrades** 128×128: `elemental_blades`, `elemental_arrows`, `camouflage`, `true_sight`.
- **Faction crest** 256×256.

## Dark Elves (Shaessari)

Skins: `melasarach`, `aterliman`. Sources: `art/source/dark-elves/units/<unit>/<skin>.glb` + `<skin>_albedo.png`; concept art: `docs/lore/images/dark-elves/units/<unit>/`.

### Units

Each unit: one model per skin (same skeleton and clips), one albedo texture per skin.

| id | Unit | Category · LOD0 tris / verts | Clips / nodes | Now |
|---|---|---|---|---|
| `webweaver` | Webweaver (builder) | Infantry · 3,000 / 2,000 | idle, walk, run, work, die | — |
| `blade_dancer` | Blade Dancer | Infantry · 3,000 / 2,000 | idle, walk, run, attack, die | — |
| `crossbowman` | Crossbowman (crossbow in each hand) | Infantry · 3,000 / 2,000 | idle, walk, run, attack (shot), die | — |
| `scout` | Scout | Infantry · 3,000 / 2,000 | idle, walk, run, attack, work (sabotage), die | — |
| `spider_rider` | Spider Rider | Cavalry · 5,000 / 3,500 | idle, walk, run, attack, cast (web), die | — |
| `dark_priest` | Dark Priest | Infantry · 3,000 / 2,000 | idle, walk, run, cast, die | — |
| `dark_mage` | Dark Mage | Infantry · 3,000 / 2,000 | idle, walk, run, attack (projectile), cast, die | — |
| `dark_dragon` | Dark Dragon | Dark dragon · 10,000 / 6,500 | idle (hover), run (flight), attack (breath), die | — |

### Heroes

| id | Hero | LOD0 tris / verts | Clips |
|---|---|---|---|
| `dark_paladin` | Dark Paladin | 8,000 / 5,000 | idle, walk, run, attack, cast, die |
| `dark_archer` | Dark Archer | 8,000 / 5,000 | idle, walk, run, attack (shot), cast, die |
| `spider_queen` | Spider Queen | 8,000 / 5,000 | idle, walk, run, attack, cast, die; the spider form is a second model (same budget) |

### Summoned (one skin)

| id | Creature | Category · LOD0 tris / verts | Clips |
|---|---|---|---|
| `spider` | Spider (Summon Spiders) | Cavalry · 5,000 / 3,500 | idle, walk, run, attack, die |

### Buildings, towers, traps

Sources: `art/source/dark-elves/buildings/<id>.glb`. One look per faction (no regional skins).

| id | Building | Class · LOD0 tris · footprint | Note |
|---|---|---|---|
| `keep` | Keep | Keep · ≤ 40,000 · 14×14 m |  |
| `fighting_pits` | Fighting Pits | M · ≤ 20,000 · ≤ 12×12 m |  |
| `shooting_range` | Shooting Range | M · ≤ 20,000 · ≤ 12×12 m |  |
| `scout_post` | Scout Post | S · ≤ 10,000 · ≤ 9×9 m |  |
| `spider_caves` | Spider Caves | L · ≤ 25,000 · ≤ 16×16 m |  |
| `dark_church` | Dark Church | L · ≤ 25,000 · ≤ 16×16 m | temple of the Great Spider, web-like arches |
| `dark_dragon_lair` | Dark Dragon Lair | L · ≤ 25,000 · ≤ 16×16 m |  |
| `alchemy_workshop` | Alchemy Workshop | M · ≤ 20,000 · ≤ 12×12 m |  |
| `spider_farm` | Spider Farm | S · ≤ 10,000 · ≤ 9×9 m | food |
| `wall` | Wall segment | ≤ 3,000 · 6.2×7 m |  |
| `gate` | Gatehouse | ≤ 15,000 · 18.6×7 m | `_gate_left`, `_gate_right` |
| `crossbow_tower` | Crossbow tower | ≤ 10,000 · ≤ 7×7 m |  |
| `trap_mine` | Mine (trap) | prop · ≤ 300 | hidden from the enemy |
| `trap_poison` | Poison trap | prop · ≤ 300 |  |
| `trap_paralysis` | Paralysis trap | prop · ≤ 300 |  |

Also in the faction's style: construction scaffolding S / M / L, ruins S / M / L, wall breach rubble.

**Projectiles:** crossbow bolt.

### Icons

- **Portraits** 256×256: `webweaver`, `blade_dancer`, `crossbowman`, `scout`, `spider_rider`, `dark_priest`, `dark_mage`, `dark_dragon`; summoned `spider`.
- **Hero portraits** 256×256 and 512×512: `dark_paladin`, `dark_archer`, `spider_queen`.
- **Buildings** 128×128: `keep`, `fighting_pits`, `shooting_range`, `scout_post`, `spider_caves`, `dark_church`, `dark_dragon_lair`, `alchemy_workshop`, `spider_farm`, `wall`, `gate`, `crossbow_tower`.
- **Unit abilities** 128×128: `deny_death`, `debuff`, `web`, `sabotage`.
- **Dark Paladin** 128×128: `dark_bolt`, `harvest`, `dark_shield`, `aura_of_darkness`, `cursed_ground`.
- **Dark Archer** 128×128: `paralyzing_arrow`, `shadow`, `ice_fan`, `aura_of_precision`, `black_hail`.
- **Spider Queen** 128×128: `summon_spiders`, `web_run`, `spider_net`, `lair`, `spider_form`.
- **Upgrades** 128×128: `poisoned_weapons`, `camouflage`, `active_camouflage`, `sabotage_kit`, `spider_boots`.
- **Faction crest** 256×256.

## Alkars

Skins: `alkaris`. Sources: `art/source/alkars/units/<unit>/<skin>.glb` + `<skin>_albedo.png`; concept art: `docs/lore/images/alkars/units/<unit>/`.

### Units

Each unit: one model per skin (same skeleton and clips), one albedo texture per skin.

| id | Unit | Category · LOD0 tris / verts | Clips / nodes | Now |
|---|---|---|---|---|
| `goblin` | Goblin (builder) | Goblins, skeletons · 2,000 / 1,400 | idle, walk, run, work, die | — |
| `orc` | Orc | Orcs · 3,500 / 2,400 | idle, walk, run, attack, die | — |
| `berserker` | Berserker | Orcs · 3,500 / 2,400 | idle, walk, run, attack, die | — |
| `orc_hunter` | Orc Hunter | Orcs · 3,500 / 2,400 | idle, walk, run, attack, attack_ranged (spear throw), die | — |
| `orc_spearman` | Orc Spearman (spear and shield) | Orcs · 3,500 / 2,400 | idle, walk, run, attack, brace, die | — |
| `alkar_swordsman` | Alkar Swordsman | Infantry · 3,000 / 2,000 | idle, walk, run, attack, die | — |
| `alkar_archer` | Alkar Archer | Infantry · 3,000 / 2,000 | idle, walk, run, attack (shot), die | — |
| `harpy` | Harpy (with crossbow) | Large creature · 6,000 / 4,000 | idle (hover), run (flight), attack, attack_ranged, die | — |
| `goblin_wolf_rider` | Goblin Wolf Rider | Cavalry · 5,000 / 3,500 | idle, walk, run, attack, die | — |
| `dark_acolyte` | Dark Acolyte | Infantry · 3,000 / 2,000 | idle, walk, run, attack (projectile), cast, die | — |
| `siege_troll` | Siege Troll | Large creature · 6,000 / 4,000 | idle, walk, run, attack, attack_ranged (rock throw), die | — |

### Heroes

| id | Hero | LOD0 tris / verts | Clips |
|---|---|---|---|
| `dark_lord` | Dark Lord | 8,000 / 5,000 | idle, walk, run, attack, cast, die |
| `dark_mage` | Dark Mage | 8,000 / 5,000 | idle, walk, run, attack (projectile), cast, die |
| `wraith_king` | Wraith King | 8,000 / 5,000 | idle, walk, run, attack, cast, die |

### Summoned (one skin)

| id | Creature | Category · LOD0 tris / verts | Clips |
|---|---|---|---|
| `wraith` | Wraith | Infantry · 3,000 / 2,000 | idle, walk, run, attack, die |
| `skeleton_swordsman` | Skeleton Swordsman | Goblins, skeletons · 2,000 / 1,400 | idle, walk, run, attack, die |
| `skeleton_archer` | Skeleton Archer | Goblins, skeletons · 2,000 / 1,400 | idle, walk, run, attack (shot), die |
| `demon` | Demon (Dark Mage) | Large creature · 6,000 / 4,000 | idle, walk, run, attack, die |

### Buildings, towers, traps

Sources: `art/source/alkars/buildings/<id>.glb`. One look per faction (no regional skins).

| id | Building | Class · LOD0 tris · footprint | Note |
|---|---|---|---|
| `keep` | Keep | Keep · ≤ 40,000 · 14×14 m | dark volcanic stone, black metal |
| `orc_pits` | Orc Pits | M · ≤ 20,000 · ≤ 12×12 m |  |
| `dark_order` | Dark Order | M · ≤ 20,000 · ≤ 12×12 m |  |
| `harpy_nest` | Harpy Nest | M · ≤ 20,000 · ≤ 12×12 m |  |
| `wolf_den` | Wolf Den | M · ≤ 20,000 · ≤ 12×12 m |  |
| `dark_cult` | Dark Cult | M · ≤ 20,000 · ≤ 12×12 m |  |
| `troll_arena` | Troll Arena | L · ≤ 25,000 · ≤ 16×16 m |  |
| `workshop` | Workshop | M · ≤ 20,000 · ≤ 12×12 m |  |
| `dark_order_enchantments` | Dark Order Enchantments | M · ≤ 20,000 · ≤ 12×12 m |  |
| `goblin_farm` | Goblin Farm | S · ≤ 10,000 · ≤ 9×9 m | food |
| `wall` | Wall segment | ≤ 3,000 · 6.2×7 m |  |
| `gate` | Gatehouse | ≤ 15,000 · 18.6×7 m | `_gate_left`, `_gate_right` |
| `crossbow_tower` | Crossbow tower | ≤ 10,000 · ≤ 7×7 m |  |
| `wraith_tower` | Wraith tower | ≤ 10,000 · ≤ 7×7 m | necrotic |

Also in the faction's style: construction scaffolding S / M / L, ruins S / M / L, wall breach rubble.

**Projectiles:** hunting spear, arrow, troll rock, acid bomb.

### Icons

- **Portraits** 256×256: `goblin`, `orc`, `berserker`, `orc_hunter`, `orc_spearman`, `alkar_swordsman`, `alkar_archer`, `harpy`, `goblin_wolf_rider`, `dark_acolyte`, `siege_troll`; summoned `wraith`, `skeleton_swordsman`, `skeleton_archer`, `demon`.
- **Hero portraits** 256×256 and 512×512: `dark_lord`, `dark_mage`, `wraith_king`.
- **Buildings** 128×128: `keep`, `orc_pits`, `dark_order`, `harpy_nest`, `wolf_den`, `dark_cult`, `troll_arena`, `workshop`, `dark_order_enchantments`, `goblin_farm`, `wall`, `gate`, `crossbow_tower`, `wraith_tower`.
- **Unit abilities** 128×128: `curse_the_ground`, `shields_mode`, `rage`.
- **Dark Lord** 128×128: `scattering_wave`, `dark_might`, `plague`, `cursed_presence`, `dominion_of_darkness`.
- **Dark Mage** 128×128: `cursed_blob`, `rot`, `scorched_earth`, `fiery_death`, `summon_demon`.
- **Wraith King** 128×128: `wraith_retinue`, `chill_of_the_grave`, `soul_harvest`, `dominion_over_the_dead`, `wraith_horde`.
- **Upgrades** 128×128: `burning_weapons`, `sack_of_rocks`, `acid_bombs`, `wards_against_magic`, `cursed_weapons`, `invisibility_detection`.
- **Faction crest** 256×256.

## Dwarves (Dragori)

Skins: `monsarx`, `jarngrod`, `skopeskala`. Sources: `art/source/dwarves/units/<unit>/<skin>.glb` + `<skin>_albedo.png`; concept art: `docs/lore/images/dwarves/units/<unit>/`.

### Units

Each unit: one model per skin (same skeleton and clips), one albedo texture per skin.

| id | Unit | Category · LOD0 tris / verts | Clips / nodes | Now |
|---|---|---|---|---|
| `worker` | Worker | Dwarves · 3,000 / 2,000 | idle, walk, run, work, die | — |
| `axe_dwarf` | Axe Dwarf (two-handed axe) | Dwarves · 3,000 / 2,000 | idle, walk, run, attack, die | — |
| `gunner` | Gunner | Dwarves · 3,000 / 2,000 | idle, walk, run, attack, attack_ranged (shot), die | — |
| `airship` | Airship | Machine (static) · ≤ 15,000 tris (static model) | nodes `_propeller_…`, `_barrel` (swivel cannon); + `airship_wreck` | — |
| `mortar` | Mortar | Machine (static) · ≤ 15,000 tris (static model) | node `_barrel`; crew is part of the model; + `mortar_wreck` | — |
| `bear_rider` | Bear Rider | Cavalry · 5,000 / 3,500 | idle, walk, run, attack, die | — |
| `golem` | Golem | Large creature · 6,000 / 4,000 | idle, walk, run, attack, attack_ranged (fire rune), die | — |

### Heroes

| id | Hero | LOD0 tris / verts | Clips |
|---|---|---|---|
| `dwarf_king` | Dwarf King (King under the Mountain) | 8,000 / 5,000 | idle, walk, run, attack, cast, die |
| `engineer` | Dwarf Engineer | 8,000 / 5,000 | idle, walk, run, attack, cast, work, die |
| `runemaster` | Rune Master | 8,000 / 5,000 | idle, walk, run, attack, cast, die |

### Buildings, towers, traps

Sources: `art/source/dwarves/buildings/<id>.glb`. One look per faction (no regional skins).

| id | Building | Class · LOD0 tris · footprint | Note |
|---|---|---|---|
| `keep` | Keep | Keep · ≤ 40,000 · 14×14 m | built into the rock |
| `barracks` | Barracks | M · ≤ 20,000 · ≤ 12×12 m |  |
| `workshop` | Workshop | L · ≤ 25,000 · ≤ 16×16 m |  |
| `bear_grounds` | Bear Grounds | L · ≤ 25,000 · ≤ 16×16 m |  |
| `golem_workshop` | Golem Workshop | L · ≤ 25,000 · ≤ 16×16 m |  |
| `forge` | Forge | M · ≤ 20,000 · ≤ 12×12 m |  |
| `rune_hall` | Rune Hall | M · ≤ 20,000 · ≤ 12×12 m |  |
| `tunnel` | Tunnel | S · ≤ 10,000 · ≤ 9×9 m | entrance into the rock |
| `farm` | Farm | S · ≤ 10,000 · ≤ 9×9 m | fungus farm |
| `wall` | Wall segment | ≤ 3,000 · 6.2×7 m |  |
| `gate` | Gatehouse | ≤ 15,000 · 18.6×7 m | `_gate_left`, `_gate_right` |
| `mortar_tower` | Mortar tower | ≤ 10,000 · ≤ 7×7 m |  |
| `gunner_tower` | Gunner tower | ≤ 10,000 · ≤ 7×7 m |  |
| `rune_tower` | Rune tower | ≤ 10,000 · ≤ 7×7 m | glowing rune core; fire / frost / lightning by color |

Also in the faction's style: construction scaffolding S / M / L, ruins S / M / L, wall breach rubble.

**Projectiles:** bullet (tracer effect), mortar shell, cannonball.

### Icons

- **Portraits** 256×256: `worker`, `axe_dwarf`, `gunner`, `airship`, `mortar`, `bear_rider`, `golem`.
- **Hero portraits** 256×256 and 512×512: `dwarf_king`, `engineer`, `runemaster`.
- **Buildings** 128×128: `keep`, `barracks`, `workshop`, `bear_grounds`, `golem_workshop`, `forge`, `rune_hall`, `tunnel`, `farm`, `wall`, `gate`, `mortar_tower`, `gunner_tower`, `rune_tower`.
- **Unit abilities** 128×128: `smoke_screen`, `phosphorus_shells`, `golem_frost`, `golem_fire`.
- **Dwarf King** 128×128: `wrath_of_the_ancestors`, `ancestral_shield_wall`, `stoneskin`, `steadfast`, `earthquake`.
- **Dwarf Engineer** 128×128: `blinding_charges`, `field_repair`, `recon_shell`, `mechanism_tuning`, `undermine`.
- **Rune Master** 128×128: `rune_swap`, `rune_ward`, `rune_trap`, `rune_aura`, `rune_fusion`.
- **Upgrades** 128×128: `smoke_grenades`, `insulated_armor`, `incendiary_powder`, `phosphorus_shells`, `rune_axes`, `frost_rune`, `fire_rune`, `rune_sight`.
- **Faction crest** 256×256.

## Shared and neutral

### Neutral outposts

| id | Model | Budget | Now |
|---|---|---|---|
| `gold_mine` | Gold mine | ≤ 20,000 · 8×8 m | placeholder (castle) |
| `sawmill` | Sawmill | ≤ 20,000 · 8×8 m | placeholder (castle) |
| `ore_mine` | Ore mine | ≤ 20,000 · 8×8 m | placeholder (castle) |

### Environment

| Group | Models | LOD0 budget |
|---|---|---|
| Trees | deciduous ×3, conifer ×3, birch ×2, elven forest giant ×2, dead / dark-land tree ×2, desert palm ×2 | ≤ 1,500 each |
| Bushes, grass clumps | ×6 (green, dry, snowy) | ≤ 300 |
| Rocks and cliffs | rocks ×6, cliff pieces ×4, volcanic rocks ×3 (Alkar lands) | ≤ 800 (cliffs ≤ 2,000) |
| Props | fences ×2, carts ×2, barrels and crates ×4, signpost, campfire, red tents of the orc camps ×2 | ≤ 800 |
| Markers | rally point banner | ≤ 300 |

### Terrain materials

1024×1024, tileable, albedo + normal: grass, dry grass, meadow with flowers, dirt, mud, rock, cliff, sand, desert sand, snow, ice, volcanic ash (Alkar lands), dark soil, cobblestone road, dirt road, forest floor — 16 materials.

### Effects (particles and decals)

| Group | Textures |
|---|---|
| Elements | fire (flipbook), smoke, embers; ice shards, frost decal; lightning arcs (flipbook), sparks; acid splash, acid puddle decal; poison cloud; necrotic wisps; light rays, holy glow |
| Combat | hit sparks, dust puff, debris (stone, wood), explosion (flipbook), muzzle flash, scorch decal, crater decal |
| Abilities | aura ring, area circle, web decal, smoke screen, roots / thicket, rune circle (fire, frost, lightning), summon portal |
| In the world | selection circle, move marker, plot outline, construction grid |

### Icons and HUD (all factions)

- **Orders** 128×128: `move`, `stop`, `hold`, `attack`, `patrol`, `attack_ground`, `rally`, `cancel`, `build`, `repair`, `gate_open`, `gate_close`, `climb`, `revive`, `research`.
- **Resources** 64×64: `gold`, `wood`, `metal`, `food`.
- **Statuses** 32×32: `burning`, `poisoned`, `slowed`, `stunned`, `acid`, `blinded`, `grounded`, `on_wall`, `invisible`, `paralyzed`.
- **Cursors** 32×32 and 64×64: the 28 states in [`UX.md`](UX.md).
- **HUD frames** (9-slice): minimap ring, resource plate, command card frame, portrait medallion, tooltip — one color variant per faction (5).
- **Regional crests** 256×256: one per skin region (14), for the skin choice in the lobby.
