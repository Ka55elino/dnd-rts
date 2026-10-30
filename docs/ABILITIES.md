# Abilities: data format

An ability is a file `content/abilities/<id>.json`. A unit lists its abilities in `"abilities": ["id", ...]`. Everything is computed in Go (`internal/sim/abilities.go`, `mods.go`); the frontend only shows buttons, cooldowns and visuals.

## Kinds

| `kind` | How it works |
|---|---|
| `active` | used on command (or by autocast), then goes on `cooldown` (s) |
| `toggle` | switched on / off; while on — `modifiers` on its own squad |
| `passive` | always on: `modifiers` on its own squad |
| `aura` | always on: `modifiers` on units within `radius` around the bearer (own units by default); `tags` — only on units with these tags (e.g. `["mage"]`) |
| `mode` | a choice of one of the `modes` (element): the selected mode's modifiers apply to its own squad, and the unit's other abilities follow the mode. Switching has a `cooldown`. One such mode per unit |

## Target (`active`)

| `target` | What the player picks |
|---|---|
| `self` | nothing — centered on the caster |
| `point` | a point on the ground within `range` |
| `unit` | a squad within `range`: an enemy (default) or an ally (`"affects": "allies"`) |

If the target is farther than `range`, the squad walks up on its own and casts once in reach. Any new order cancels the wait.

For `self` abilities, effects with `radius: 0` apply to the caster's own squad (shields, self-buffs).

`"autocast": true` — the ability can be set to autocast (RMB on the button): heals go to the most wounded own squad in radius, damage goes to the nearest enemy.

## Effects (`effects`)

Effects fire in order. `radius: 0` — the target only (the whole target squad); otherwise everyone in the circle allowed by `affects` (`enemies` / `allies` / `all`; by default enemies for damage and statuses, own units for heals, buffs and dispel).

| `type` | Fields | What it does |
|---|---|---|
| `damage` | `damageType`, `amount`, `radius` | damage by the common rules (`combat.json`), with the damage type's effect; as an area effect it also hits enemy buildings |
| `heal` | `amount`, `radius` | heals living units (not `mechanical`, `construct`, `elemental`) |
| `status` | `status`, `duration`, `radius` | `stun`, `slow`, `blind` (attacks miss), `burn`, `acid`, `ground` (flyers land: ground attacks can hit them) |
| `knockback` | `force`, `radius` | pushes away from the center (stronger near the center); heroes and flyers are not knocked back |
| `dispel` | `radius` | removes all negative statuses |
| `buff` | `modifiers`, `duration`, `radius` | temporary modifiers |
| `zone` | `radius`, `duration`, `interval`, `effects` | an area on the ground: every `interval` s it applies the nested `effects` (their `radius` defaults to the zone radius) |
| `chain` | `jumps`, `jumpRange`, `falloff`, `effects` | hits a soldier at the target, then jumps to the nearest one not yet hit (up to `jumps` times, no farther than `jumpRange`); each jump is weaker by `falloff` (0.2 = −20%). Nested effects: only `damage` and `status`. Does not hit buildings |
| `summon` | `horde`, `count`, `duration`, `limit` | summons `count` squads (1 by default) at the center for `duration` s (0 — until killed). Summoned squads don't eat food and, when time runs out, vanish without giving XP to the enemy. `limit: true` — only one summon of this ability at a time, and its cooldown starts only after the summoned squads die (the emperor's retinue) |
| `teleport` | `radius`, `toBuilding` | the caster's squad (and own squads within `radius` around it) instantly moves to the point, keeping formation. `toBuilding: true` — only to an own completed building (point no farther than 12 m from it), otherwise the cast fails |
| `transform` | `unit`, `duration` | the squad temporarily becomes another unit (`content/units`): its attacks, armor, speed, size. Health fraction, abilities, hero level and per-level growth are kept; when it ends — back to normal |

**Area shape** — `"shape": "cone"` with `angle` (width, degrees) on `damage` / `heal` / `status` / `knockback` / `dispel` / `buff`: a cone from the caster toward the chosen point, `radius` long. Default is a circle. Cones don't hit buildings.

## Modifiers (`modifiers`)

They stack (two +15% damage auras = +30%).

| Field | Meaning |
|---|---|
| `armor`, `magicDefense` | + to armor / magic defense |
| `damage` | + fraction of attack and ability damage (0.15 = +15%) |
| `attackSpeed` | + fraction of attack speed |
| `moveSpeed` | + fraction of speed (−0.4 = 40% slower) |
| `range` | + fraction of ranged attack range |
| `damageTaken` | + fraction of damage taken (−0.3 = 30% less) |
| `rangedTaken` | same, but only from ranged attacks |
| `invulnerable`, `stunImmune` | invulnerability, stun immunity |
| `extraType` + `extraDamage` | extra damage of another type on every hit (fraction), e.g. fire on arrows |
| `attackType` | replaces the damage type of the unit's attacks (element choice) |
| `resistType` + `resist` | fraction of damage of this type that doesn't go through (elemental protection); only one, the strongest, applies at a time |
| `abilityHaste` | + fraction of ability cooldown speed (0.33 ≈ −25% time) |
| `immune` | a status effect that has no effect: `burn`, `slow`, `stun`, `acid`, `poison` (the paladin's "Purity" passive) |
| `workSpeed` | auras only: + fraction of construction and recruitment speed of **own buildings** in radius (imperial oversight) |
| `rooted` | the squad holds its ground: doesn't move or pursue, can't be trampled, a charge into it is stopped |
| `speedDamage` | piercing damage to enemies running into the squad, per m/s of their speed (once per second per enemy, with attack bonuses, e.g. ×1.6 vs. cavalry) — spear wall |
| `formation` | `"wedge"`: triangular formation, point forward |
| `charge` | + fraction of charge damage and knockback; in a wedge a rider also loses half as much momentum on each enemy knocked down — goes deeper |

## Modes (elements)

```json
{ "id": "element", "kind": "mode", "cooldown": 5, "modes": [
  { "id": "fire", "name": "Fire", "modifiers": { "attackType": "fire" } },
  { "id": "ice",  "name": "Ice",  "modifiers": { "attackType": "ice" } } ] }
```

The unit's current mode determines:
- **modifiers** of auras, passives and toggles — `"byMode": { "fire": {...}, "ice": {...} }` (modes not in the list use plain `modifiers`);
- **effects** — `"mode": "fire"` on an effect: it fires only while the unit is in that mode (a zone remembers its effects at cast time);
- cast visuals — color by the damage type that actually fired.

All modes referenced by a unit's abilities must exist in its mode ability (checked at load time).

**Ultimate** — `"ultimate": true`: gold frame on the button. The rules are set with ordinary numbers (long cooldown, hero level 5 by ability order).

## Examples (in `content/abilities/`)

- `shockwave` — `active`, `self`: crushing damage in a circle + knockback
- `shield_wall` — `toggle`: −40% speed, +3 armor, −60% damage from ranged attacks
- `spear_wall` — `toggle` (spearmen): hold their ground, +1 armor, 5 damage per m/s of a charging enemy's speed; knights at full gallop (7 m/s) take ~56 per hit
- `wedge` — `toggle` (knights): wedge, +15% speed, +30% charge, +15% damage taken (flanks)
- `battle_cry` — `aura`: +15% damage to own units within radius 15
- `fire_pillar` — `active`, `point`, autocast: fire damage in a circle (sets on fire)
- `storm` — `active`, `point`: 6 s zone, ice + slow every second
- `heal` — `active`, `unit` (ally), autocast: heal in a circle
- `chain_lightning` — `active`, `unit`, autocast: chain of 5 targets, −20% per jump, lightning + 1 s stun (battle mages)
- `blink` — `active`, `point`: teleports the squad 25 m (battle mages)
- `frost_fan` — `active`, `point`: 60° × 12 m cone, ice + slow (pack leader, lvl 3)
- `pack_call` — `active`, `self`: summons a wolf rider squad, limited (pack leader, lvl 4)
- `beast_form` — `active`, `self`: transforms into a beast for 30 s (pack leader, lvl 5)
- `mass_recall` — `active`, `point`, map-wide range, ultimate: mass teleport to an own building (nobody has it yet; for the future emperor)
- Archmage (Empire, keep): `element` (mode: fire / ice / lightning — attack type), `elemental_ward` (aura: −30% damage of the chosen element), `summon_elemental` (elemental of the chosen element for 45 s), `arcane_focus` (aura on mages: faster cooldowns), `meteor_shower` (ultimate: 12 m zone for 6 s, damage and effect by element)

---

# Attacks: minimum range, area damage, towers

An attack (a unit's `melee` / `ranged` or a building's `attack`) also has:

| Field | Meaning |
|---|---|
| `minRange` | dead zone: doesn't shoot closer than this. A squad looks for targets farther away, and a tower skips close ones |
| `splash` | radius: every shot hits all enemies around the impact point (100% at the center, 50% at the edge) and damages buildings there |

**Attack ground** (`T` on the command card, for all squads with a ranged attack):
- click on the ground — the squad moves into range (or backs out of the dead zone if it's too close) and fires volleys at that point until it gets another order;
- arrows hit any enemies at the impact point and the enemy building under the point;
- on a wall, on a tower and in a stance the squad doesn't move — it fires only if the point is already in reach.

**Attacking buildings** (towers): `attack` (like a ranged attack) and `volley` (arrows per volley), firing at the nearest enemy. On a wall, range is +25%.
- `toggles` — one toggle ability (fire mode): switched on with a button in the ring if its requirements are met.
- Upgrades with `"buildings": [...]` apply to these towers' shots, the same way as to units from `units`.

**Repair:**
- workers repair own damaged buildings, walls included: right-click a building with workers, ~2% health per second per squad, several squads are faster;
- own machines (tag `mechanical`) within 8 m of idle workers get repaired automatically.

---

# Upgrades and requirements

## Upgrade — `content/upgrades/<id>.json`

```json
{
  "id": "forged_weapons_2",
  "name": "Forged Weapons II",
  "cost": { "gold": 400, "metal": 100 },
  "researchTime": 45,
  "requires": { "upgrades": ["forged_weapons_1"] },
  "units": ["swordsman", "knight"],
  "modifiers": { "damage": 0.15 }
}
```

- Researched once in a building that has the upgrade in `"research": [...]`; goes into the same queue as recruitment (cancel refunds).
- After research, the `modifiers` (the same as for abilities) apply to all matching units of the player — including those already recruited.
- Who: `units` (unit ids) and / or `tags`; empty — everyone.
- `group`: upgrades of one group replace each other (element choice): researching one removes the rest.
- An upgrade may change nothing itself and instead unlock an ability: the ability has `"requires": { "upgrades": ["fire_arrows"] }` (example — `fire_mode`, the archers' fire mode).

## Requirements — `requires`

`{ "buildings": [...], "upgrades": [...] }` — own **completed** buildings of these types and researched upgrades. Supported on buildings (construction), squads (recruitment), upgrades (research) and abilities (use / toggling). The HUD shows unavailable buttons grayed out with a red "Requires: …"; the server checks every command itself.

Examples: the forge requires barracks; knights — Forged Weapons I; Forged Weapons II — Forged Weapons I.

---

# Heroes: experience, levels, revival

A unit becomes a hero with a `hero` block (`content/units/<id>.json`):

```json
"hero": {
  "xp": [0, 150, 400, 800, 1400],
  "perLevel": { "damage": 0.1, "armor": 1 },
  "healthPerLevel": 0.1
}
```

- `xp[i]` — total experience needed for level i+1; the array length is the max level (5).
- `perLevel` — the same modifiers as for abilities, × (level − 1); `healthPerLevel` — fraction added to max health per level. On level-up the hero is fully healed.
- Hero abilities unlock by level: `abilities[i]` is available from level i+1 (pattern skill / aura / skill / aura / ultimate). Locked ones can't be used or toggled; passives and auras don't apply.

**Experience** (`combat.json` → `xp`):
- every enemy soldier who dies within `radius` (25) of a hero gives its `xp` from the unit JSON (default — health / 10) to every such hero;
- capturing an outpost gives `capture` (100) to the capturer's heroes nearby.

**Recruitment and revival** — the hero squad has `"unique": true`:
- a player has one hero of each type at a time (in the field or in the queue);
- after death, the building where it was recruited offers "Revive …": cost `reviveCost` (default — recruitment cost), time `reviveTime` (default — 2 × `buildTime`); the hero returns with its previous level and experience.

Dev console: `xp <n>` — experience to the selected heroes.
