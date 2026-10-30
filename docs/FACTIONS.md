# Factions

Working design document. Everything below is a **draft for discussion**: ideas, roles and approximate numbers. The numbers here are relative (compared to each other); exact values will appear in `content/` during implementation.

Workflow: faction idea → unit roles → buildings → heroes → abilities → faction powers. Every item that needs a new mechanic is marked **[mechanic: …]** — these marks are used to build the work list for the simulation (section at the end).

## 1. General Rules for All Factions

### Resources (see `RESOURCES.md`)

| Resource | Source | Mainly used for |
|---|---|---|
| Gold | gold mine (outpost) | everything |
| Wood | lumber mill (outpost) | buildings, archers, siege engines |
| Metal | ore mine (outpost) | heavy infantry, cavalry, armor and weapon upgrades |
| Food | farms (each faction has its own equivalent) | army cap |

Factions have different "appetites": some need more metal, others more wood. This gives a reason to fight over different outposts.

### Common Roster Framework

Every faction has roughly the same set of roles, so balance has a common baseline for comparison. The content differs for each.

| Slot | Role |
|---|---|
| Worker | builds free-placement buildings, captures outposts |
| Basic infantry | cheap, holds the line |
| Anti-cavalry infantry | spears / pikes |
| Ranged | damage from afar, weak in melee |
| Cavalry | charge, raids, finishing off archers |
| Elite | expensive, strong late-game squad |
| Special unit | monster, mage, flyer — the face of the faction |
| Siege | against buildings and walls |
| Heroes | 3 per faction (1 available from the start) |

### Decisions

- Factions have **no** unique resources: everyone has gold, wood, metal, food.
- Heroes and units have **cooldowns only**, no mana.
- **Flying units — right away**, in the first version of the shared mechanics.
- **Melee and ranged attacks.** A unit can have only a melee attack, only a ranged one, or both. If both, the unit shoots from afar and switches to melee when the enemy comes close (or by stance, like the Elf Ranger). Each attack has its own damage type, damage, range and cooldown **[mechanic: multiple attacks per unit]**.
- **Walls for all factions**: segments placed by dragging, gates (only own units pass), tower plots on walls **[mechanic: walls — done for the base]**.
- **Day and night.** The game has a day/night cycle; at night unit vision is reduced (except for those who see at night as well as by day) **[mechanic: day / night, vision and fog of war]**.
- **Own land** — the area around one's own buildings (and special objects, e.g. the elves' sacred trees). The radius is set per building / object in JSON. Some factions get bonuses on it (elves heal) **[mechanic: territory]**.
- **Trees** — map objects: impassable for squads (forest), provide cover, can burn and be cut down, regrow over time **[mechanic: trees as objects]**.

### Damage and Armor

Details are in section 1.1 below. Each unit and each attack is described by: **damage type** (+ its parameters), **armor class**, **armor** (against physical damage) and **magic defense** (against magical damage).

### Progression

- **Squad experience** — 3 veterancy levels: +damage, +armor, heal on level-up **[mechanic: experience and levels]**
- **Heroes** — **5 levels**, each unlocks one ability following a common scheme:

  | Level | What unlocks |
  |---|---|
  | 1 | active ability |
  | 2 | passive or aura |
  | 3 | active ability |
  | 4 | passive or aura |
  | 5 | ultimate ability (long cooldown) |

  - **Experience** — for kills (by the hero and near them) and for **capturing outposts** **[mechanic: hero experience and levels — done, see `ABILITIES.md`]**.
  - **Recruitment and respawn** — in the main building; after death the hero respawns there for **gold and a long time** (like construction) **[mechanic: hero respawn — done]**.
  - **Auras** affect own squads within a radius around the hero **[mechanic: auras]**.
- **Upgrades** in a special building (forge and equivalents): weapons, armor, special faction upgrades **[mechanic: upgrades]**
- **Faction powers** — points accumulate from kills and holding outposts, spent on the powers tree **[mechanic: faction powers]**


## 1.1 Damage and Armor Types

**[mechanic: damage and armor types]** — a shared system for all factions.

### How a Hit Is Calculated

1. **Evasion**: if the target has light armor and the hit is physical — a chance to miss.
2. **Base damage** of the attack × bonuses (by target tag, veterancy, upgrades).
3. **Defense**: the target's **armor** (against physical damage) or **magic defense** (against magical) is subtracted from the damage — but not below ¼ of the hit. Heavy armor additionally reduces physical damage by a percentage.
4. **Damage type special**: penetration, bonus vs buildings, effect (burning, stun, slow, etc.).
5. Damage-over-time effects tick every 0.5 s.

All "chances" are deterministic (a hash of the tick, attacker id and target id), like arrow spread — otherwise replays and network play would desync.

### Physical Damage

| Type | Special | Parameters (starting) |
|---|---|---|
| **Piercing** (spears, arrows, crossbows) | chance to **ignore armor**: the armor value and heavy-armor resistance are not applied | 25% chance |
| **Slashing** (swords, axes) | standard damage, no special | — |
| **Crushing** (hammers, trolls, siege) | **bonus damage vs structures** | ×2.5 vs buildings |

### Magical Damage

| Type | Special | Parameters (starting) |
|---|---|---|
| **Fire** | **burning**: damage over time to units and structures | 20% of the hit per second, 3 s; does not stack, refreshes duration |
| **Lightning** | chance to **stun**; **deals no damage** to stone and wooden structures | 15% chance, 1 s stun; a stunned unit cannot move or attack |
| **Water / ice** | **slows** movement and attack | −30% speed and attack speed, 2 s |
| **Acid** | damage + **armor reduction** | −2 armor per hit, up to 3 stacks, 6 s |
| **Poison** | damage + **damage over time**, chance to **spread** to nearby units | 15% of the hit per second, 5 s; 15% chance per tick to infect a neighbor within 3 m; **one hit infects at most 2 additional units** (units infected by spreading do not pass it on) |
| **Radiant** | **pure damage**: not reduced by anything — not armor, magic defense, resistances or heavy armor; **deals no damage to structures** | — |
| **Necrotic** | **lifesteal**: the attacker heals for part of the damage dealt | 30% of damage goes to the attacker's health (for a squad — to the soldier who struck); no healing from hitting structures |

Magical damage cannot be evaded (light armor does not help) and is not reduced by heavy armor — only **magic defense** reduces it.

**Radiant** damage has been brought back in a new form: not "damage vs undead" but **pure damage** against units that goes through any defense but is useless against structures. The Paladin's "Divine Smite" is also radiant damage.

### Armor Classes

| Class | Special | Parameters (starting) |
|---|---|---|
| **Light** | chance to **evade** a physical hit | 15% |
| **Medium** | standard, no special | — |
| **Heavy** | **resistance to all physical damage** | −35% physical damage |

Armor class affects only physical damage. A separate parameter protects against magic:

| Parameter | Against | How it works |
|---|---|---|
| **Armor** (number) | physical damage | subtracted from every physical hit (not below ¼ of the hit); reduced by acid |
| **Magic defense** (number) | magical damage | subtracted from every magical hit and effect tick (not below ¼) |

Typical combinations: heavy infantry — high armor, low magic defense (vulnerable to mages); mages and priests — light armor, high magic defense; monsters — medium armor and medium magic defense, lots of health. Structures also have both values.

**Structures** are a separate class with a **material**, because lightning, fire and crushing damage depend on it:

| Material | Special |
|---|---|
| Wood | burns more (fire ×1.5), lightning has no effect |
| Stone | fire ×0.5, lightning has no effect |
| Magical (crystal, light, etc.) | lightning works; for Celestial buildings, magic towers |

### Described Separately

- **Tag bonuses**: spearmen vs cavalry, axes vs monsters. This is an attack multiplier by target tag, not a damage type. **[mechanic: damage bonus by tag]**
- **Flyers**: only attacks with the `anti-air` tag can hit them (each faction has its own set of such units and towers). Other attacks do not hit flyers. **[mechanic: flying units — done: `flying` on the unit, `targets` on the attack, `ground` status]**
- **Resistances** of individual units to a specific type (e.g. golem: −50% lightning, fire elemental: immune to fire) — targeted modifiers in the unit data, on top of magic defense.

### Damage Decisions

- Necrotic damage is **lifesteal**; radiant is **pure damage** with no reduction, does not hit structures.
- Ice slows **both movement and attack**.
- Poison from one hit infects **at most 2 additional units**.
- Units and structures have **separate magic defense** in addition to armor.

---

## 2. Human Empire

**Idea.** A disciplined army and stone fortresses: heavy infantry, knights, archers and siege engines. The focus is on **physical damage**, formations and fortifications.

- **Play style:** wall in the settlement, hold the line in formation, advance slowly under cover of archers and engines
- **Strengths:** physical damage of all three types, heavy armor on the core infantry, the best fortifications (walls and three kinds of towers), strong siege
- **Weaknesses:** **no air units** (against air — only archers, ballistae, mages and towers), little magic (mages are the only source), heavy armor is vulnerable to magic (low magic defense)
- **Resources:** gold + metal (heavy armor), wood (archers, engines, walls)
- **Visuals:** steel, blue and gold, stone walls and towers, banners

### Faction Feature — Walls

Settlements can be walled in **[mechanic: walls — done for the base]**:

- Workers build **wall segments** between posts (by dragging, as in Stronghold / BFME2); a segment is a structure with HP, stone
- **Gates** — a segment that own squads can pass through, but enemies cannot **[mechanic: passability by owner — done for the base]**
- Walls have **tower slots** — plots like those around the keep, but on the wall (e.g. on the posts between segments). A tower is built on such a plot and stands on the wall **[mechanic: plots on walls — done for the base]**
- Walls block navigation; the enemy must break them (crushing damage) or go around

### Buildings

| Building | Placement | What it does | Requires |
|---|---|---|---|
| **Keep** | map (player slot) | main building; workers, heroes | — |
| **Barracks** | plot | swordsmen, spearmen | — |
| **Archery Range** | plot | archers | — |
| **Stables** | free | knights | barracks **[mechanic: requirements — done]** |
| **Workshop** | free | ballistae, trebuchets | — |
| **Mage Guild** | free | mages, healers | — |
| **Forge** | plot | weapon, armor and engine upgrades (see below) **[mechanic: upgrades]** | — |
| **Farm** | free | food | — (needed for the army cap) |
| **Walls and gates** | free, by dragging | see above | — |

Placement (plot or free) is a proposal, to be adjusted.

**Forge upgrades** — bought once and apply to all of the player's squads and structures, including those already recruited. Level 2 requires level 1. Numbers are starting values.

| Upgrade | Levels | Applies to | Effect |
|---|---|---|---|
| **Forged Weapons** | 2 | swordsmen, spearmen, knights | +15% damage per level |
| **Reinforced Armor** | 2 | swordsmen, spearmen, knights, archers | +2 armor per level |
| **Fire Arrows** | 1 | archers and **archer towers** | unlocks a toggleable **fire mode** |
| **Fire Bolts** | 1 | ballistae and **ballista towers** | unlocks a toggleable **fire mode** |
| **Improved Mechanism** | 1 | trebuchets — both units and towers | −25% reload time |

Builders, healers, mages and heroes are not upgraded by the forge.

**Fire mode** (after the upgrade, toggled separately for each squad and each tower): **fire** is added to piercing damage (burning, sets wooden structures alight), but piercing damage **loses its chance to ignore armor**. The choice: fire against crowds, light armor and wooden structures; regular arrows against heavy armor.

**Defensive structures** **[mechanic: attacking buildings — done]** — built on plots around the keep and on **tower slots on walls**:

| Tower | Damage | Targets | Special |
|---|---|---|---|
| **Archer tower** | piercing, ranged | ground and **air** (anti-air) | cheap, fast fire |
| **Ballista tower** | piercing, ranged, high single-target damage | ground and **air** (anti-air) | against monsters, cavalry and air |
| **Trebuchet tower** | crushing, very long range, area | ground only | against siege engines, buildings and dense formations; has a dead zone up close **[mechanic: minimum range — done]** |

### Units

| Unit | Where | Armor | Damage | Ability |
|---|---|---|---|---|
| **Builder** | keep | light | — | builds buildings and walls, **repairs** structures, ballistae and trebuchets; captures outposts |
| **Swordsman** | barracks | heavy | slashing | Shield Wall |
| **Spearman** | barracks | heavy | piercing | Spear Wall |
| **Archer** | archery range | medium | piercing, ranged, **anti-air** | Fire Arrows |
| **Knight** | stables | heavy | slashing (+ charge) | Wedge Formation |
| **Ballista** | workshop | heavy | piercing, ranged, **anti-air** | — |
| **Trebuchet** | workshop | heavy | crushing, very long range | bonus damage vs buildings |
| **Fire mage** | mages' guild | light, high magic defense | fire, ranged, **anti-air** | Pillar of Fire |
| **Water mage** | mages' guild | light, high magic defense | water / ice, ranged, **anti-air** | Storm |
| **Lightning mage** | mages' guild | light, high magic defense | lightning, ranged, **anti-air** | Chain Lightning |
| **Healer** | mages' guild | light | no attack | Heal |

**Abilities** (starting parameters, adjust during balancing) **[mechanic: toggleable modes / stances — done]**:

- **Shield Wall** (swordsmen, toggle): −40% speed, +armor, −60% damage from ranged attacks. The squad holds a tight formation.
- **Spear Wall** (spearmen, toggle): the squad stops and cannot move. An enemy entering the formation takes bonus piercing damage that grows with the **enemy's speed** (cavalry at full gallop takes the most); a cavalry charge into a spear wall is negated **[mechanic: damage from target speed — done]**.
- **Fire Arrows** (archers): first the **upgrade is bought** (once, **at the forge**), after which **fire mode** is toggled per squad: + fire (burning, sets wooden structures alight), but without piercing damage's chance to ignore armor.
- **Wedge Formation** (knights, toggle): the squad reforms into a wedge — higher speed and charge damage, pushes deeper through formations, but takes more damage on the flanks **[mechanic: squad formations — done]**.
- **Trebuchet**: crushing damage already gives ×2.5 vs buildings; plus long range and a dead zone up close.
- **Pillar of Fire** (fire mages, point-targeted): a pillar of fire erupts at the chosen point — fire damage in a small radius and **burning** on everything hit, including structures (wooden ones burn more). Starting: 3 m radius, 25 s cooldown.
- **Storm** (water mages, area): a storm starts in the chosen area for several seconds — a **very strong slow** to movement and attack of all enemies inside, and **water / ice damage** every second while the enemy is in the zone. Starting: 8 m radius, −60% speed and attack speed, 6 s, damage once per second, 35 s cooldown **[mechanic: ground zones]**.
- **Chain Lightning** (lightning mages, unit-targeted): lightning strikes an enemy and jumps to the nearest enemies nearby; each one hit takes damage and is **stunned**. Starting: up to 5 targets, jump up to 6 m, −20% damage per jump, 1 s stun, 20 s cooldown. Has no effect on stone and wooden structures **[mechanic: chain effects — done]**.
- **Heal** (healers): an ability — heals wounded **living** allies nearby; does not heal engines (ballistae, trebuchets) or structures. Can be cast manually or set to **autocast** — then healers heal the nearest wounded on their own whenever the ability is ready **[mechanic: healing, ability autocast — done]**.
- **Repair** (builders): restore HP of structures and engines — ballistae and trebuchets **[mechanic: repair — done]**.

**Living and mechanical.** Units have a tag: `living` (people, horses) or `mechanical` (ballista, trebuchet). Healers heal only `living`; builders repair structures and `mechanical` **[mechanic: living / mechanical tags — done]**.

**Against air** the Empire has only: archers, ballistae, mages (attack and abilities), archer and ballista towers. Swordsmen, spearmen, knights and trebuchets cannot hit flyers.

### Heroes

Three heroes, 5 levels (scheme in "Progression"). Numbers are starting values.

#### Paladin — Support Warrior

Heavy armor, slashing damage, melee.

| Lvl | Ability | Type | Description | Starting |
|---|---|---|---|---|
| 1 | **Holy Radiance** | ability, area around self | enemies in the radius **miss** for a time (blind) | 8 m radius, 100% misses for 4 s, 40 s cooldown **[mechanic: blind / misses — done]** |
| 2 | **Purity** | passive | **immune to poison** | — |
| 3 | **Divine Smite** | ability, unit-targeted | heavy **radiant** damage — ignores all defenses and resistances | 45 s cooldown **[mechanic: pure damage — done]** |
| 4 | **Warding Aura** | aura | +physical defense (armor) to own units in the radius | 15 m radius, +3 armor |
| 5 | **Divine Shield** | ultimate, self | **invulnerable** for a time | 6 s, 120 s cooldown **[mechanic: invulnerability — done]** |

#### Archmage — Elemental Mage

Light armor, high magic defense, magical damage of the chosen element, ranged, **anti-air**. Almost all abilities depend on the **chosen element**.

| Lvl | Ability | Type | Description | Starting |
|---|---|---|---|---|
| 1 | **Element** | toggle | attack damage type: **fire** (burning), **water / ice** (slow) or **lightning** (stun chance) | switch with a short cooldown, 5 s **[mechanic: switchable damage type — done]** |
| 2 | **Elemental Ward** | aura | to own units in the radius — protection from magic of the **chosen element**; changes **immediately** when the element is switched | 15 m radius, −30% damage of that element |
| 3 | **Summon Elemental** | ability | summons an elemental of the **chosen element** (fire / water / storm) for a time | 1 elemental, 45 s, 60 s cooldown **[mechanic: summoning — done]** |
| 4 | **Arcane Focus** | aura | **all mages** in the radius (guild mages, healers, mage heroes) recover abilities faster | 15 m radius, −25% cooldown |
| 5 | **Meteor Shower** | ultimate, area | a rain of meteors; damage type and effect follow the **chosen element**: fire — burning, ice — strong slow, lightning — stun | 12 m radius, 6 s, 150 s cooldown |

#### Emperor — Cavalry Commander

Mounted. Heavy armor, slashing damage, charge (like the knights).

| Lvl | Ability | Type | Description | Starting |
|---|---|---|---|---|
| 1 | **Imperial Guard** | ability | a squad of cavalrymen appears around the Emperor; the squad stays **as long as it lives**; **cannot be summoned again** while the squad is alive or the ability is on cooldown | 1 squad (like knights, 6 riders), 90 s cooldown after the squad dies **[mechanic: limited summoning — done]** |
| 2 | **Imperial March** | aura | +movement speed to own units in the radius | 15 m radius, +15% |
| 3 | **Inspire** | ability, area around self | for a time, +attack speed to units that were in the radius at the moment of casting | 15 m radius, +30% for 12 s, 50 s cooldown |
| 4 | **Imperial Oversight** | aura | +**construction and production** speed (recruitment in buildings) in the radius | 20 m radius, +30% **[mechanic: auras on structures — done]** |
| 5 | **Mass Recall** | ultimate | transports own units from the radius around the Emperor (together with him) **to a chosen own building** | 15 m radius, 180 s cooldown **[mechanic: teleport — done]** |

### Decisions

No open questions for the Empire.

**Decided (heroes):** experience for kills and capturing outposts; respawn in the keep for gold and a long time; mass teleport — only to own buildings; the Emperor's retinue lives until it dies; the focus aura affects all mages; the elemental protection aura changes immediately with the element.

**Decided:** fire arrows for ballistae are toggled, like the archers'; both fire arrow upgrades also apply to the corresponding towers, and fire mode on towers is toggleable too; in fire mode piercing damage does not ignore armor; forge upgrades — weapons ×2, armor ×2, fire arrows, fire arrows for ballistae, improved mechanism; the storm deals damage; mages hit air (anti-air); three kinds of mages — fire (Pillar of Fire), water (Storm), lightning (Chain Lightning); heal is an ability with autocast; fire arrows are bought at the forge; healers heal only the living; builders repair structures, ballistae and trebuchets; fire arrows — a purchasable upgrade + a toggleable ability; towers are built on separate plots on walls; the forge is needed; the trebuchet existing as both a unit and a tower is fine.

## 3. High Elves

**Idea.** An ancient forest people: the best archers, fast riders on griffons and tigers, spirits and nature magic. Specialization — **ranged attacks**.

- **Play style:** keep distance, shoot the enemy down on approach, retreat to own land, heal and come back
- **Racial features:**
  - **Heal on own land** — gradual health regeneration near own buildings and **sacred trees** (created by the Elf King); radii in JSON **[mechanic: territory, regeneration]**
  - **See at night as well as by day** — vision is not reduced at night **[mechanic: day / night]**
  - **Walls** around settlements (like all factions)
- **Strengths:** ranged damage, air (griffons), mobility, healing, night
- **Weaknesses:** **no siege engines** (buildings are destroyed only by units and heroes), fragile against crushing damage and numbers, expensive
- **Resources:** gold + wood
- **Visuals:** white stone and silver, greenery and turquoise, trees inside buildings

### Buildings

| Building | Placement | What it does |
|---|---|---|
| **Keep** | map (player slot) | main building; **spirits** (builders), heroes |
| **Order of the Sword** | plot | swordsmen with a double-bladed sword and a shield |
| **Order of Archers** | plot | archers |
| **Griffon tower** | free | griffon riders (air) |
| **Tiger pen** | free | tiger riders (cavalry) |
| **Spirit garden** | free | elementals: water, fire, storm |
| **Enchantment tower** | plot | upgrades (see below) |
| **Moonwell** | free | food; **grows trees** around itself |
| **Walls and gates** | free, by dragging | as for the Empire |

Placement is a proposal, to be adjusted.

**Defensive structures** (plots around the keep and on walls):

| Tower | Damage | Targets |
|---|---|---|
| **Archer tower** | piercing, ranged | ground and air (anti-air) |
| **Magic tower** | **radiant**, ranged | ground and air (anti-air); **sees invisible units** (with the "True Sight" upgrade) |

### Units

| Unit | Where | Armor | Melee attack | Ranged attack | Special |
|---|---|---|---|---|---|
| **Spirit** | keep | **none** (no armor) | — | — | **no attack**; **the elves' builder**: constructs buildings, **grows trees**, **removes negative effects** from allies (dispel) |
| **Swordsman with a double-bladed sword and a shield** | Order of the Sword | medium | slashing | — | — |
| **Archer** | Order of Archers | light | slashing | piercing | anti-air (with the ranged attack) |
| **Griffon rider** | griffon tower | medium | slashing — **air targets only** | piercing | **flying** **[mechanic: flying units, attacks with restricted targets]** |
| **Tiger rider** | tiger pen | medium | slashing | piercing | cavalry; shoots and slashes |
| **Fire elemental** | spirit garden | **no defense** | fire (burning) | fire (burning) | resistance and immunity to burning |
| **Water elemental** | spirit garden | **no defense** | water / ice (slow) | water / ice (slow) | resistance and immunity to slow |
| **Storm elemental** | spirit garden | **no defense** | lightning | lightning (stun chance) | resistance and immunity to stun |

**All elementals have both attacks** — melee and ranged, both of their own element. All elemental attacks also hit **air** (anti-air).

- **No defense** — armor 0 and magic defense 0: elementals hit hard but die fast.
- **Own element**: −75% damage from that element and **immunity to its effect** (a fire elemental cannot be set on fire, a water one cannot be slowed, a storm one cannot be stunned).
- **Not living**: not healed by healers, on own land, or by sacred trees (tag is not `living`).

**Enchantment tower upgrades.** Elemental upgrades are **swappable**: one is bought, and the chosen element can be changed (fire, water / ice, lightning) **[mechanic: swappable upgrades]**. Numbers are starting values.

| Upgrade | Applies to | Effect |
|---|---|---|
| **Elemental Blades** | swordsmen | + damage of the chosen element to the melee attack (fire — burning, ice — slow, lightning — stun chance); **its own** chosen element |
| **Elemental Arrows** | all ranged attacks: archers, griffons, tigers, **archer towers** | + damage of the chosen element to the ranged attack; **its own** chosen element, independent of the blades |
| **Camouflage** | foot units | become **invisible in trees** while not moving or attacking |
| **True Sight** | magic towers | towers see invisible units **[mechanic: invisibility detection]** |

### Heroes

#### Elf Ranger — Archer and Scout

Light armor. **Two attacks**, damage depends on the **stance**: melee — slashing, ranged (bow) — piercing **[mechanic: stances with different attacks]**.

| Lvl | Ability | Type | Description | Starting |
|---|---|---|---|---|
| 1 | **Elven Agility** | ability, self | for a time moves very fast and **ignores terrain** (passes through forest and steep slopes) | +100% speed, 6 s, 40 s cooldown **[mechanic: ignoring passability]** |
| 2 | **Forest Shadow** | passive | **invisible in trees** while not moving or attacking | **[mechanic: invisibility, trees]** |
| 3 | **Hail of Arrows** | ability, area | area damage | 6 m radius, 30 s cooldown |
| 4 | **Eagle Eye** | aura | +range for own ranged units in the radius | 15 m radius, +20% range |
| 5 | **Sniper Shot** | ultimate, point-targeted | heavy damage to **any visible point on the map** | 150 s cooldown **[mechanic: "whole map" range within vision]** |

#### Elf King — Stag Rider

Head of the council of elders and keepers (lore: [`lore/factions/high-elves.md`](lore/factions/high-elves.md)). Mounted on a stag. Medium armor, slashing damage, melee.

| Lvl | Ability | Type | Description | Starting |
|---|---|---|---|---|
| 1 | **Thicket** | ability, area | creates an area of thicket — **difficult terrain** for enemies | 8 m radius, −50% enemy speed, 15 s, 45 s cooldown **[mechanic: zones that change passability]** |
| 2 | **Child of the Forest** | passive | **immune to poison and necrotic damage** | — |
| 3 | **Sacred Tree** | ability, tree-targeted | enchants a tree: around it is an area where **all elves heal** | 10 m radius, 20 s, 60 s cooldown |
| 4 | **Power of Growth** | aura | trees in the radius **recover faster** (regrow) | 20 m radius **[mechanic: trees regrow]** |
| 5 | **Awakening of the Ents** | ultimate, tree-targeted | **summons ents** from nearby trees: **1 tree — 1 ent**, the trees **remain** in place; cannot be cast with no trees nearby | up to 3 ents, 60 s, 180 s cooldown **[mechanic: summoning from map objects]** |

#### Priestess — Healer and Light Mage

Ranged attack — **radiant** damage, melee — slashing **[mechanic: multiple attacks per unit]**. Armor — **to be decided** (light?).

| Lvl | Ability | Type | Description | Starting |
|---|---|---|---|---|
| 1 | **Moonbeam** | ability, area | **radiant** area damage (no effect on structures) | 5 m radius, 30 s cooldown |
| 2 | **Blessing of the Moon** | passive | **immune to poison and necrotic damage** | — |
| 3 | **Cleansing** | ability, area | **removes negative effects** (burning, poison, slow, stun, blind, armor reduction) from allies in the area | 10 m radius, 35 s cooldown **[mechanic: dispel]** |
| 4 | **Aura of the Moon Shield** | aura | +**magic defense** to own units in the radius | 15 m radius, +3 magic defense |
| 5 | **Sacred Weapons** | ultimate, area around self | allies around deal **additional radiant damage** for a time | 15 m radius, +25% radiant damage, 15 s, 150 s cooldown |

### Decisions

No open questions for the High Elves.

**Decided:** all elementals have a melee and a ranged attack of their own element, both hit air; resistance to own element −75% and immunity to its effect; elementals do not heal; the spirit garden has water, fire and storm elementals with no defense, with damage and resistance of their own element; spirits have no armor and no attack; the magic tower sees invisible units after the upgrade; spirits are recruited at the keep; elemental blades and arrows each have their own element; elemental arrows also apply to archer towers; ents — 1 tree per ent, the trees remain; unit stats (table above); spirits are builders (build, grow trees, dispel); magic tower — radiant damage, anti-air; moonwell — food and trees; own land — around buildings and sacred trees, radii in JSON; trees can be destroyed; ents — only near trees.

## 4. Dark Elves

**Concept.** A people of shadows and intrigue: **sabotage**, ambushes, poison, webs and necrotic magic. They win not head-on, but by weakening the enemy before the battle.

- **Playstyle:** scouting, invisible strikes against the economy and buildings, webs and traps on the enemy's path, poison and curses before the decisive clash
- **Racial traits:**
  - **See at night as well as by day** **[mechanic: day / night]**
  - **Walls** around settlements (segments, gates, tower slots)
  - **Web** — a ground surface that slows all units inside it (except friendly units with "Spider Boots"); created by spider riders and the Spider Queen **[mechanic: ground surfaces]**
- **Strengths:** sabotage and invisibility, poison and necrotic damage, webs and traps, control (paralysis, grounding flyers)
- **Weaknesses:** **head-on fights**, **small squads**, light armor on almost everything
- **Resources:** gold + metal
- **Visuals:** black and purple, spikes, obsidian, lilac lights, webs

### Buildings

| Building | Placement | What it does |
|---|---|---|
| **Keep** | map (player slot) | main building; webweavers, heroes |
| **Fighting Pits** | plot | blade dancers |
| **Shooting Range** | plot | crossbowmen |
| **Scout Post** | free | scouts |
| **Spider Caves** | free | spider riders |
| **Dark Church** | free | dark priests, dark mages |
| **Dark Dragon Lair** | free | dark dragon — **maximum 1 per lair** **[mechanic: per-building unit limit]** |
| **Alchemy Workshop** | plot | upgrades (see below) |
| **Spider Farm** | free | food |
| **Walls and Gates** | free, drag-placed | same as the Empire |

Placement is a proposal, to be adjusted.

**Defensive structures:**

| Structure | What it does |
|---|---|
| **Crossbow Tower** | piercing, ranged; **anti-air** |
| **Traps and Mines** | hidden ground structures, **placed by webweavers**; trigger when an enemy steps on them; **the enemy cannot see them** unless it has **invisibility detection** **[mechanic: traps, invisible structures]** |

**Trap types:**

| Trap | Effect |
|---|---|
| **Mine** | **bludgeoning** damage (area around it) |
| **Poison Trap** | **poison** damage (ticks, spreads per the poison rules) |
| **Paralysis Trap** | **paralyzes** the unit / squad that stepped on it for a time |

### Units

| Unit | Where | Armor | Melee attack | Ranged attack | Special |
|---|---|---|---|---|---|
| **Webweaver** | keep | light | — | — | no attack; **builds, repairs, captures** outposts |
| **Blade Dancer** | fighting pits | light | slashing | — | — |
| **Crossbowman** | shooting range | light | slashing | piercing | **anti-air** (with ranged attack) |
| **Scout** | scout post | light | slashing | — | with upgrade — **building sabotage** |
| **Spider Rider** | spider caves | light | slashing + **poison** | piercing | creates **web surfaces** (slow all units) |
| **Dark Priest** | dark church | light | — | necrotic | "deny death" ability: an ally cannot drop below 1 HP for a time; weak healing **[mechanic: death prevention]** |
| **Dark Mage** | dark church | light | — | necrotic | **debuffs** on enemies: −damage or −armor **[mechanic: debuffs]** |
| **Dark Dragon** | dragon lair | heavy, **magic immune** | — | fire + necrotic | **flying**; 1 per lair |

**Alchemy Workshop upgrades** (numbers are starting values):

| Upgrade | Applies to | Effect |
|---|---|---|
| **Poisoned Weapons** | all **weapon** attacks — swords and arrows / bolts (dancers, crossbowmen, scouts, riders; heroes with weapons too) | + **poison** damage to those attacks (magic and the dragon — no) |
| **Camouflage** | foot units | invisible while standing still and not attacking |
| **Active Camouflage** | foot units, requires "Camouflage" | the same, but **at night** they can move without losing invisibility |
| **Sabotage Kit** | scouts | unlocks **building sabotage** — a **channeled ability**: while the scout is sabotaging, the building **slowly burns and crumbles**; sabotage **does not break** the scout's invisibility; it can only be stopped by **killing the scout** **[mechanic: channeled abilities, sabotage]** |
| **Spider Boots** | all foot units | **ignore webs** (are not slowed) |

### Heroes

#### Dark Paladin — necrotic warrior

Heavy armor, slashing damage, melee.

| Lvl | Ability | Type | Description | Starting values |
|---|---|---|---|---|
| 1 | **Dark Bolt** | ability, targeted | projectile: **necrotic** damage, and heals the paladin | cooldown 20 s |
| 2 | **Harvest** | passive | when an **enemy** unit dies nearby — restores health | radius 10 m, +5% HP per unit **[mechanic: on-death effects]** |
| 3 | **Dark Shield** | ability, self | for a time **reflects all damage** back to the attacker; the paladin **still takes** the damage herself | 5 s, cooldown 60 s **[mechanic: damage reflection]** |
| 4 | **Aura of Darkness** | aura | friendly units nearby deal **extra necrotic** damage (with lifesteal) | radius 15 m, +15% damage as necrotic |
| 5 | **Cursed Ground** | ultimate, area | an area that **periodically** deals necrotic damage to enemies | radius 10 m, 12 s, cooldown 150 s |

#### Dark Archer — huntress from the shadows

**Light** armor. Ranged — piercing, melee — slashing.

| Lvl | Ability | Type | Description | Starting values |
|---|---|---|---|---|
| 1 | **Paralyzing Arrow** | ability, targeted | temporarily **disables** a unit or a **whole squad** (cannot move or attack); **does not affect buildings** | 4 s, cooldown 35 s **[mechanic: paralysis]** |
| 2 | **Shadow** | passive | **invisible** while not attacking; after attacking — a delay before becoming invisible again | delay 3 s |
| 3 | **Ice Fan** | ability, cone | a cone of **ice** arrows: damage and slow | 60° cone, 12 m, cooldown 30 s **[mechanic: cone areas — done]** |
| 4 | **Aura of Precision** | aura | +attack speed for friendly **ranged** units nearby | radius 15 m, +20% |
| 5 | **Black Hail** | ultimate, area | a hail of arrows: **piercing + necrotic** damage in an area | radius 10 m, cooldown 150 s |

#### Spider Queen — mistress of the web

In the lore she is the high priestess of the Great Spider, the deity of the drow cult: not a goddess, but a living being of great power (see [`lore/factions/dark-elves.md`](lore/factions/dark-elves.md)).

- **Normal form:** no armor, **ranged attack only** — necrotic damage.
- **Spider form** (ultimate): **heavy** armor, **melee attack only** — slashing + **poison** + **acid**.

| Lvl | Ability | Type | Description | Starting values |
|---|---|---|---|---|
| 1 | **Summon Spiders** | ability | summons spiders next to her | 6 spiders, 40 s, cooldown 50 s **[mechanic: summoning — done]** |
| 2 | **Web Run** | aura | +movement speed for allies **standing in a web** | +30% |
| 3 | **Spider Net** | ability, targeted / area | **brings flying** units down to the ground (for a time they become ground units and can be hit in melee) | 6 s, cooldown 40 s **[mechanic: grounding flyers]** |
| 4 | **Lair** | **toggleable** aura | while active — **web** around the queen | radius 8 m |
| 5 | **Spider Form** | ultimate, self | transforms into a **giant spider** for a time: **heavy armor**, more HP and damage, different attacks | 30 s, cooldown 180 s **[mechanic: transformation — done]** |

### Decisions

No open questions on the Dark Elves.

**Decided:** traps are placed by webweavers — mines (bludgeoning), poison and paralysis traps; sabotage is channeled, does not break invisibility, interrupted only by the scout's death; the archer's melee is slashing; the queen — ranged necrotic, in spider form melee slashing + poison + acid; food — spider farm; traps and mines are invisible without invisibility detection; sabotage — the building slowly burns and crumbles; Poisoned Weapons — all weapon attacks (swords, arrows); the dark dragon flies, ranged combat; crossbowmen and their tower hit air; slaves build, repair and capture; the paladin also takes damage while Dark Shield is up; the archer has light armor, the queen has no armor, heavy in spider form; the queen's web is a toggleable aura.

## 5. Alkars

**Concept.** A pale aristocratic civilization ruled by the strongest; next to their city lies a vast necropolis (lore: [`lore/factions/alkars.md`](lore/factions/alkars.md)). The core of the army is **created creatures** — orcs, goblins and trolls, made by the Alkars from stone and wood with dark energy — and wild **harpies** the Alkars have attracted or subjugated: an endless stream of cheap troops. On top of them stand the **elite — the Alkars themselves**, aristocrats serving as expensive units: Alkar swordsmen, Alkar archers and dark acolytes. Undead rise from fallen enemies. Plays like a swarm (in the spirit of the Zerg): overwhelm with numbers, sacrifice squads and immediately recruit new ones, with a small elite core.

- **Playstyle:** constant pressure with mass, trading cheap squads for expensive ones, curses and dead that rise from fallen enemies
- **Racial traits:**
  - **Horde** = **cheap units** (low cost, little food, most without armor) **+ wraiths and skeletons** that appear for free from abilities and fallen enemies. There is no separate swarm mechanic.
  - The dead: some abilities turn fallen enemies into **wraiths and skeletons** — **temporary** and **taking no food** **[mechanic: turning the fallen into units, temporary units]**
  - **Walls** — as for all factions
- **Strengths:** mass, necrotic damage and curses (poison, acid, anti-heal), siege trolls, harpies in the air
- **Weaknesses:** almost no armor — vulnerable to area damage and magic; strong only in numbers
- **Resources:** gold + wood
- **Visuals:** dark volcanic stone, black metal, porcelain-white Alkar aristocrats, greenish necrotic light; crude created creatures

### Buildings

| Building | Placement | What it does |
|---|---|---|
| **Keep** | map (player slot) | main building; goblins, heroes |
| **Orc Pits** | plot | orcs, berserkers, orc hunters, orc spearmen |
| **Dark Order** | plot | Alkar swordsmen, Alkar archers (Alkar elite) |
| **Harpy Nest** | free | harpies (air) |
| **Wolf Den** | free | goblin wolf riders |
| **Dark Cult** | free | dark acolytes |
| **Troll Arena** | free | siege trolls |
| **Workshop** | plot | upgrades (see below) |
| **Dark Order Enchantments** | plot | order upgrades (see below) |
| **Goblin Farm** | free | food |
| **Walls and Gates** | free, drag-placed | as for everyone |
| **Crossbow Tower** | plot near the keep / on the wall | piercing, ranged, **anti-air** |
| **Wraith Tower** | plot near the keep / on the wall | **necrotic** damage; with the "Invisibility Detection" upgrade (order enchantments) **sees invisible units** |

Placement is a proposal, to be adjusted.

### Units

| Unit | Where | Armor | Melee attack | Ranged attack | Special |
|---|---|---|---|---|---|
| **Goblin** | keep | **none** | — | — | builds, repairs, **captures** outposts |
| **Orc** | orc pits | **none** | slashing | — | cheap, large squad |
| **Berserker** | orc pits | **none** | slashing | — | **rage**: the lower the HP, the higher the damage **[mechanic: damage from missing health]** |
| **Orc Hunter** | orc pits | **none** | slashing | piercing (thrown spears) | cheap ranged infantry: throws spears at a short range; hits **ground and air** targets |
| **Orc Spearman** | orc pits | **heavy** | piercing | — | **bonus vs cavalry**; toggleable **"Shields"** mode: resistance to ranged attacks **[mechanic: toggleable modes]** |
| **Harpy** | harpy nest | **none** | slashing | piercing | **flying**; with upgrade — **acid bombs** (a separate attack against ground targets) |
| **Goblin Wolf Rider** | wolf den | **none** | slashing | — | fast cavalry |
| **Alkar Swordsman** | dark order | **heavy** | slashing | — | **Alkar elite**, very expensive |
| **Alkar Archer** | dark order | **heavy** | — | piercing | **Alkar elite**, very expensive; **anti-air** |
| **Dark Acolyte** | dark cult | **none** | — | necrotic | **Curse the Ground** (channeled): curses an area; an enemy who dies in it rises as a **skeleton (swordsman or archer) or wraith** under the player's control **[mechanic: channeled abilities]** |
| **Siege Troll** | troll arena | **none** | bludgeoning | (with upgrade) bludgeoning, **ground targets only** | siege |

**Workshop upgrades:**

| Upgrade | Applies to | Effect |
|---|---|---|
| **Burning Weapons** | orcs, berserkers | + **fire** to melee attack (burning) |
| **Sack of Rocks** | siege trolls | unlocks a **ranged attack**: bludgeoning damage, ground targets only |
| **Acid Bombs** | harpies | **new attack**: bombs against **ground** targets — acid (damage and −armor) |

**Dark Order Enchantments upgrades** (Alkar swordsmen and archers):

| Upgrade | Effect |
|---|---|
| **Wards Against Magic** | + **magic defense** for order units |
| **Cursed Weapons** | + **necrotic** damage (lifesteal) to order units' attacks |
| **Invisibility Detection** | **wraith towers** see invisible units |

### Summoned and Risen

All summoned and risen units are **temporary** (disappear after a time) and **take no food**.

| Unit | Source | Description |
|---|---|---|
| **Wraith** | Wraith Retinue, those killed by the Wraith King's abilities, the acolytes' curse | **ground** (does not fly), necrotic damage |
| **Skeleton Swordsman** | the acolytes' curse | cheap melee infantry |
| **Skeleton Archer** | the acolytes' curse | cheap ranged infantry, **anti-air** |

Stats (armor, damage, HP, lifetime) live in JSON and are tuned during balancing.
| **Demon** | Dark Mage's ultimate | melee: slashing + fire; heavy armor |

### Heroes

#### Dark Lord — lord of war

Ruler of the Alkars — the strongest of them (lore: [`lore/factions/alkars.md`](lore/factions/alkars.md)).

Heavy armor, bludgeoning damage, melee.

| Lvl | Ability | Type | Description | Starting values |
|---|---|---|---|---|
| 1 | **Scattering Wave** | ability, area in front / around | damage and **knockback** of units of size **medium and smaller** | radius 8 m, cooldown 30 s **[mechanic: unit size]** |
| 2 | **Dark Might** | passive | attacks deal **extra necrotic** damage (lifesteal) | +20% damage as necrotic |
| 3 | **Plague** | ability, area | curses an area: enemies inside take **poison** damage and get the **acid** status (−armor) | radius 10 m, 10 s, cooldown 45 s |
| 4 | **Cursed Presence** | aura | enemies nearby **cannot be healed** (anti-heal) | radius 15 m **[mechanic: heal prevention]** |
| 5 | **Dominion of Darkness** | ultimate, self | for a time **empowers "Cursed Presence"**: the aura grows wider, and enemies in it **constantly lose health** | radius 25 m, 2% max HP per second, 15 s, cooldown 150 s |

#### Dark Mage — fire and acid

Head of the order of mages based in the center of Alkaris; the order keeps the knowledge of how orcs are made.

**No armor**, ranged attack: fire + acid.

| Lvl | Ability | Type | Description | Starting values |
|---|---|---|---|---|
| 1 | **Cursed Blob** | ability, area | **necrotic** area damage and **acid**; also **reduces enemy damage** for a time | radius 6 m, −20% damage, 8 s, cooldown 25 s |
| 2 | **Rot** | aura (passive area) | enemies and **buildings** around take acid damage and the acid status | radius 8 m |
| 3 | **Scorched Earth** | ability, area | sets an area on fire (burning for everyone in it) | radius 8 m, 10 s, cooldown 40 s |
| 4 | **Fiery Death** | passive | burning enemies **explode on death** (fire damage around), if set on fire by the mage or his demon | explosion radius 3 m **[mechanic: on-death effects]** |
| 5 | **Summon Demon** | ultimate | summons a **demon**: melee, slashing + fire, heavy armor | 60 s, cooldown 180 s |

#### Wraith King — lord of the dead

Overseer of the Alkarian Necropolis, who has himself become one of the dead.

Heavy armor, damage: necrotic + slashing.

| Lvl | Ability | Type | Description | Starting values |
|---|---|---|---|---|
| 1 | **Wraith Retinue** | ability | summons a **squad of wraiths** | 1 squad, cooldown 60 s |
| 2 | **Chill of the Grave** | aura | wraiths and skeletons nearby (and the king himself) deal **necrotic** damage | radius 15 m |
| 3 | **Soul Harvest** | ability, area | **necrotic** area damage; **those killed by it** rise as **wraiths** | radius 8 m, cooldown 45 s |
| 4 | **Dominion over the Dead** | aura | wraiths and skeletons nearby deal **more damage** and **move faster** | radius 15 m, +20% damage, +20% speed |
| 5 | **Wraith Horde** | ultimate, rectangular area | a horde of wraiths sweeps along a **large rectangular** strip, dealing necrotic damage to everything in its path | 40×12 m, cooldown 180 s **[mechanic: rectangular areas, moving effects]** |

### Decisions

No open questions on the Alkars (summon numbers — during balancing, in JSON).

**Decided:** the crossbow tower hits air; invisibility detection for the wraith tower is in the order enchantments; wraiths do not fly; the Dark Lord's ultimate is an empowered Cursed Presence that drains enemy health; towers — crossbow and wraith (necrotic, invisibility detection with upgrade); food — goblin farm; goblins have no armor and capture; spearmen — bonus vs cavalry and toggleable ranged resistance; acid bombs are a separate ground attack; wraiths, skeletons and the demon are temporary and take no food; horde = cheap units + wraiths and skeletons, with no separate swarm mechanic.

## 6. Dwarves

**Concept.** A people from under the mountains: sturdy armor, gunpowder and machines, runes and golems, underground tunnels.

- **Playstyle:** slow but inexorable: sturdy infantry supported by mortars, airships and golems; fast redeployment through tunnels between their outposts
- **Racial traits:**
  - **Tunnels** — a unique building type: built on a **dedicated plot** at friendly **outposts**; the **keep** has a built-in tunnel (the keep itself is an entrance). The tunnel network connects all friendly entrances; a squad enters at one place and exits at another — **one squad at a time**. An entrance **can be destroyed**; **the enemy cannot pass through the tunnels** **[mechanic: tunnels — transfer between buildings]**
  - **Walls** — as for all factions
- **Strengths:** armor, machines and gunpowder, runes, defense, mobility via tunnels
- **Weaknesses:** **slow** (low movement speed on almost everything)
- **Resources:** metal + gold
- **Visuals:** stone, bronze, runes, machinery, steam

### Buildings

| Building | Placement | What it does |
|---|---|---|
| **Keep** | map (player slot) | main building; workers, heroes |
| **Barracks** | plot | axe dwarves, gunners |
| **Workshop** | free | airships, mortars |
| **Bear Grounds** | free | bear riders |
| **Golem Workshop** | free | golems — **maximum 1 per workshop**; choice of golem element (with runes) |
| **Forge** | plot | upgrades (see below) |
| **Rune Hall** | plot | rune upgrades (see below) |
| **Tunnel** | dedicated plot at a friendly outpost (built into the keep) | entrance / exit of the tunnel network; destructible |
| **Farm** | free | food |
| **Walls and Gates** | free, drag-placed | as for everyone |

Placement is a proposal, to be adjusted.

**Defensive structures** (plots near the keep and on walls):

| Tower | Damage | Targets |
|---|---|---|
| **Mortar Tower** | bludgeoning, area, ranged | **ground only** (does not hit air) |
| **Gunner Tower** | piercing, ranged | ground and **air** |
| **Rune Tower** | by the **selected rune** (fire / frost / lightning) with the rune's effect | ground and **air**; with a Rune Hall upgrade **sees invisible units** |

### Units

| Unit | Where | Armor | Melee attack | Ranged attack | Special |
|---|---|---|---|---|---|
| **Worker** | keep | **none** | — | — | builds, repairs, **captures** outposts |
| **Axe Dwarf** | barracks | heavy | slashing | — | — |
| **Gunner** | barracks | medium | slashing | piercing | firearms; **anti-air** |
| **Airship** | workshop | **heavy** | — | piercing — **against ground and air** | **flying**, machine (`mechanical`) |
| **Mortar** | workshop | **heavy** | — | bludgeoning **area**, **ground targets only** | machine |
| **Bear Rider** | bear grounds | heavy | bludgeoning | — | cavalry |
| **Golem** | golem workshop | heavy | bludgeoning | (with fire rune) fire | **1 per workshop**; machine / construct; **no element** by default — plain bludgeoning damage; the element is chosen in **its own** golem workshop and **applies immediately** to an already built golem |

**Forge upgrades:**

| Upgrade | Applies to | Effect |
|---|---|---|
| **Smoke Grenades** | gunners, mortars | ability: **smoke screen** — an area that **blocks the enemy's vision** **[mechanic: vision-blocking zones]** |
| **Insulated Armor** | axe dwarves, gunners, bear riders | **cold resistance** |
| **Incendiary Powder** | gunners, airships | + **fire** to attacks |
| **Phosphorus Shells** | mortars | **special attack**: sets an area on fire |

**Rune Hall upgrades:**

| Upgrade | Effect |
|---|---|
| **Rune Axes** | **melee units with weapons** deal extra **lightning** damage |
| **Frost Rune** | the golem workshop can give the golem the **frost** element: + cold damage and a **frost aura** (slows nearby enemies) |
| **Fire Rune** | the golem workshop can give the golem the **fire** element: + fire damage and a **ranged fire attack** |
| **Rune Sight** | **rune towers** see invisible units |

### Heroes

#### Dwarf King — unbreakable chieftain

The King under the Mountain, supreme ruler of the dwarves (lore: [`lore/factions/dwarves.md`](lore/factions/dwarves.md)). Heavy armor, bludgeoning damage, melee.

| Lvl | Ability | Type | Description | Starting values |
|---|---|---|---|---|
| 1 | **Wrath of the Ancestors** | ability, self | for a time attacks deal **radiant** damage | 10 s, cooldown 40 s |
| 2 | **Ancestral Shield Wall** | aura | friendly units nearby take **less damage from ranged attacks** | radius 15 m, −25% |
| 3 | **Stoneskin** | ability, self | for a time **immune to ranged attacks** | 8 s, cooldown 50 s |
| 4 | **Steadfast** | passive | **stun immunity** | — |
| 5 | **Earthquake** | ultimate, area | **bludgeoning** damage and **stun** on enemies in the area | radius 12 m, stun 2 s, cooldown 150 s |

#### Dwarf Engineer — master of machines

Medium armor. Melee — slashing, ranged — piercing.

| Lvl | Ability | Type | Description | Starting values |
|---|---|---|---|---|
| 1 | **Blinding Charges** | ability, area | units in the area **miss** for a time | radius 6 m, 5 s, cooldown 35 s |
| 2 | **Field Repair** | passive | automatically **repairs** the nearest mortar, airship or golem — **one at a time** | radius 10 m **[mechanic: automatic repair]** |
| 3 | **Recon Shell** | ability, point | **reveals an area** from the fog of war for a time | radius 15 m, 15 s, cooldown 45 s **[mechanic: fog of war, temporary vision]** |
| 4 | **Mechanism Tuning** | aura | +attack speed for **machines and towers** nearby | radius 15 m, +25% |
| 5 | **Undermine** | ultimate, structure | digs a **tunnel passage** into an **enemy or neutral** outpost or fortress — friendly squads exit right inside / at the walls; they pass **one squad at a time**; the passage **can be destroyed** | cooldown 240 s **[mechanic: tunnel into someone else's building]** |

#### Rune Master — lord of the elements

Medium armor. Melee and ranged attacks use the element of the **selected rune**: fire, frost or lightning.

| Lvl | Ability | Type | Description | Starting values |
|---|---|---|---|---|
| 1 | **Rune Swap** | toggle | selects the active rune: fire, frost or lightning | cooldown 5 s |
| 2 | **Rune Ward** | passive | **resistance to the elements that are not selected** (fire selected — resist to frost and lightning) | −40% |
| 3 | **Rune Trap** | ability, point | places a rune trap: damage and effect follow the **selected rune** (burning / slow / stun); **the enemy cannot see it** | up to 3 traps, cooldown 20 s |
| 4 | **Rune Aura** | aura | by selected rune: **fire** — friendly units attack faster; **frost** — less **physical** damage; **lightning** — less **magic** damage | radius 15 m, +20% / −20% / −20% |
| 5 | **Rune Fusion** | ultimate, self | for a time **all three runes** are active at once (attacks, resists, aura); during it "Rune Trap" places a **special rune with all three effects** | 20 s, cooldown 150 s |

### Decisions

No open questions on the Dwarves.

**Decided:** food — farm; tunnel — a building type on a dedicated plot at outposts, built into the keep, one squad at a time, destructible, the enemy cannot pass; workers have no armor and capture outposts; gunners and the gunner tower hit air; airships and mortars have heavy armor; a golem without a rune deals bludgeoning damage, the element is chosen in the golem workshop and applies immediately; rune tower — damage by selected rune, hits air, invisibility detection is a Rune Hall upgrade; the enemy cannot see rune traps; Undermine — one squad at a time, can be destroyed.

## 7. Celestials

**TODO** — the faction will be described later.

Draft idea (from the first sketch): a heavenly host — few squads, strong magic, flight, healing and summoning. Everything else — when it is described.

---

## 8. Matchup Matrix (draft)

Who has the edge over whom, by feel. To be refined during balancing.

| | Empire | High Elves | Dark Elves | Alkars | Dwarves | Celestials |
|---|---|---|---|---|---|---|
| **Empire** | = | siege and cavalry vs fragile elves | formation vs raids | upgrades vs mass | parity | crossbows vs air |
| **High Elves** | range | = | light magic vs shadows | kiting the slow horde | hard to pierce armor | parity |
| **Dark Elves** | economy raids | ambushes | = | poison vs mass | weak vs armor | debuffs vs elites |
| **Alkars** | mass before upgrades | charge into archers | brute force | = | trolls vs infantry | mass vs the few |
| **Dwarves** | armor | armor vs arrows | armor vs poison | cannons vs trolls | = | weak vs air |
| **Celestials** | air | parity | light vs darkness | too few | air over walls | = |

---

## 9. Required Mechanics (Summary)

Collected from the tags above. This feeds the "Common mechanics" block in `ROADMAP.md`.

| Mechanic | Who needs it | Priority |
|---|---|---|
| Damage types (piercing / slashing / bludgeoning, fire / lightning / ice / acid / poison / necrotic), armor classes (light / medium / heavy), building material, tag-based bonuses | everyone | high |
| Effects: burning, stun, slow (movement and attack), armor reduction, poison with spreading (up to 2), lifesteal; magic defense | all mages | high |
| Ability system: target (unit / point / area / self), cooldowns only, **autocast** (toggled per squad) | all heroes and mages | high |
| Buffs, debuffs, auras, damage over time (poison), healing | everyone | high |
| Requirements and tech tree | everyone | high |
| Upgrades (weapons, armor, special; purchased upgrades unlock abilities — fire arrows) | everyone | high |
| Experience and levels for squads and heroes (5 levels, ability / aura / ability / aura / ultimate pattern), hero recruitment and revival | everyone | high |
| Blindness (misses), true damage (bypasses all defenses), invulnerability, immunities | Empire (Paladin), everyone | high |
| Toggleable damage type and abilities that depend on the selected mode | Empire (Archmage) | medium |
| Auras on buildings (construction and recruitment speed), group teleport | Empire (Emperor) | medium |
| Attacking buildings (towers), incl. anti-air and minimum range | Empire, Dwarves, everyone | high |
| Drag-placed walls, gates (passable only for the owner), tower slots on walls | Empire, Dwarves | medium |
| Siege engines (range, buildings only, crew) | Empire, Alkars, Dwarves | medium |
| Toggleable modes and stances (shield wall, spear wall, fire arrows) | Empire, Dwarves | high |
| Damage that depends on the target's speed (spear wall vs charge) | Empire | medium |
| Squad formations (wedge) | Empire, all cavalry | medium |
| Flying units and anti-air defense | Elves, Dwarves, Celestials | done (flight without navigation, targeting, grounding) |
| Invisibility and fog of war | Dark Elves, High Elves | medium |
| Faction powers (points, power tree) | everyone | medium |
| Damage from missing health (berserkers) | Alkars | medium |
| Turning the fallen into units (skeletons, wraiths), temporary summoned units without food | Alkars, all summons | high |
| Heal prevention (anti-heal), unit size (who gets knocked back) | Alkars | medium |
| Rectangular and moving areas (wraith horde) | Alkars | medium |
| Summoning units and buildings | Dwarves, Celestials, powers | low |
| Tunnels: transfer between friendly outposts; undermining into someone else's building | Dwarves | medium |
| Vision-blocking zones (smoke), temporary vision (recon shell) | Dwarves | medium |
| Automatic repair of machines by a hero | Dwarves | low |
| Ground zones (storm: an effect on everyone inside for a time) | Empire (water mages), faction powers | high |
| Ground surfaces (web), traps and mines | Dark Elves | medium |
| Building sabotage (channeled ability, does not break invisibility), per-building unit limit (dragon) | Dark Elves | medium |
| Paralysis, death prevention (HP not below 1), damage reflection, on-death effects | Dark Elves | medium |
| Cones, grounding flyers, hero transformation | Dark Elves | medium |
| Chain effects (chain lightning: jumps between targets) | Empire (lightning mages) | medium |
| Repair of buildings and machines by builders; `living` / `mechanical` tags (who gets healed, who gets repaired) | all workers, Empire | medium |
| Multiple attacks per unit (melee + ranged, stances with different attacks) | High Elves, everyone | high |
| Day and night, vision, fog of war; night vision | everyone, High Elves | high |
| Territory ("own land") and regeneration on it | High Elves | medium |
| Trees as map objects: forest, cover, invisibility, burning / felling, regrowth, summoning ents | High Elves, everyone | medium |
| Zones that change passability (thickets), ignoring passability (agility) | High Elves | medium |
| Dispel (removing negative effects) | High Elves | medium |
| Replaceable upgrades (element choice), invisibility detection | High Elves | medium |
| Attacks with target restrictions (air only / ground only) | High Elves (griffins), everyone | high |
| Radiant damage — true damage, does not hit buildings | High Elves, Empire (Paladin) | high |
| Squad replenishment (reinforcement, resurrection) | Celestials, everyone | medium |

## 10. Open Questions

- How many factions do we bring to a playable state first? Proposal: **the Human Empire** (the baseline, tests all common mechanics) and the **Alkars** (contrast: mass vs quality).
- ~~A unique resource for some factions~~ — no, everyone shares the same set.
- ~~Mana for heroes~~ — no, cooldowns only.
- ~~Flying units later~~ — no, right away.
- Names: all original, nothing from other universes (see the notes in `ROADMAP.md`).
