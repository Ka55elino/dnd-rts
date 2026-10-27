# Roadmap

Order matters: **data → game on top of data → editor for the data**.
Phases 0–1 test whether the project is viable. Until they are closed, nothing else starts.

## Phase 0 — Foundation

- [x] Rename the Go module (`changeme` → `dnd-rts`)
- [x] Create the folder structure per `ARCHITECTURE.md`, remove the demo (`greetservice.go`)
- [ ] **Benchmark**: Go generates N moving points → binary Wails Stream @ 20 Hz → `InstancedMesh` with interpolation
  - [x] measured on macOS (M3 Pro, 100k): 60 FPS (vsync), frame cpu 2.97 ms (interp 1.98), snapshot 1.17 MB, 22.9 MB/s, max gap 52 ms, go tick 3.93 ms
  - [x] VAT models on macOS (M3 Pro, 100k swordsmen, LOD Far): ~30 FPS
  - [ ] measure on Windows
  - [ ] measure on Linux (WebKitGTK)
- [x] `internal/fixed`: Fixed type, Vec2, arithmetic, sqrt, table-based sin/cos + tests (+ determinism golden test)
  - [ ] run `go test ./internal/fixed` on Windows/Linux amd64: the golden hash must match
- [ ] `internal/protocol` + `frontend/src/net`: protocol version, snapshot, commands (snapshot is done, game commands are in phase 1)
- [x] `internal/data`: minimal `unit`, `horde`, `faction`, `map` schemas + loading and validation from `content/`
- [x] Mode switching: menu → game / editor → back (no WebGL leaks)

**Done when:** the benchmark holds 60 FPS on all three OSes, and screen transitions don't leak memory.

## Phase 1 — Vertical Slice

Data is written by hand in JSON; there is no editor yet.

- [x] Heightmap terrain (procedural from a seed or a .r16 file), RTS camera (screen edges, WASD, zoom, Q/E rotation, map bounds)
- [x] Squad: formation, click and box selection, "move" order (Go session, binary snapshots/commands)
- [x] Navigation: passability grid (by slope) + flow fields with path smoothing, formation keeping, mass-based separation (F3 — passability overlay)
- [x] Melee combat: auto-aggro and attack order, targets, damage with armor, soldier deaths, formation tightening, corpses
- [x] Archers: a squad volley = one projectile (flight time, lands at a point, each arrow is its own point within the spread circle); drawn as a dot for now
- [x] Cavalry: charge through the formation on an attack order, trampling infantry (damage + knockback + knockdown), losing momentum in a crowd
- [x] VAT animations for soldiers (idle, walk, run, attack, die): `tools/vatbake`, 3 LODs, visibility culling (`engine/vat`)
- [x] Hero (a squad of 1 soldier) with the "Shockwave" ability: radial damage + knockback, cooldown, HUD button and key 1 (placeholder model; `SkinnedMesh` comes with the assets)
- [x] HUD: selected squad panel, HP bars above squads, ability panel, hotkeys
- [x] State hash every 20 ticks (in the snapshot, F4), replay recording (commands + checkpoint hashes) to `replays/last.json` and verification via `Replay.Verify`

**Done when:** a battle of 1000+ soldiers runs stably on Win / Mac / Linux.

### Phase 1.5 — Bases and buildings (on the standard map)

Everything is tested on the current `plains` map with data in JSON. Full maps and the editor start once the "build → recruit → fight → destroy the base" loop already works.

**Concept: a BFME1 + BFME2 hybrid.**

Three kinds of buildings:

1. **Keep (main tower).** Placed by the map into the player slot (position, rotation). For now it is just a target to destroy, with no functions: keep destroyed — the player loses. Placeholder: `keep-light.glb`.
2. **Buildings on plots (as in BFME1).** Around the keep there are several predefined plots. The player chooses what to build on a plot; the building builds itself, no workers needed. Placeholder: `mausoleum-light.glb`.
3. **Free buildings (as in BFME2).** Placed anywhere within the build zone, if slope and neighboring buildings allow. Built by workers. Buildings have different footprints. Placeholder: `witch-cottage-light.glb`.

Kinds 2 and 3 are one building catalog: each building has a list of allowed placement modes `placement: ["plot"] | ["free"] | ["plot", "free"]`. The full building list and balance are added gradually; factions have different building sets (`Faction.buildings`).

**Resources:** gold, wood, metal, food. All numbers are initial placeholders in JSON, balance comes later. Mechanics and data format — [`RESOURCES.md`](RESOURCES.md).

- Gold — gold mine (resource outpost); later additional sources, e.g. control points.
- Wood — lumber mill (resource outpost).
- Metal — ore mine (resource outpost).
- Food — farms at the base. Food is the army cap: squads take it up (like supply / command points).

**Outposts:** neutral mini-bases on the map. Capture — squads stand within the radius for a few seconds with no enemies nearby. An outpost can be retaken or destroyed (ruins can be occupied again).

- **Regular outpost:** a central building (placeholder — a scaled-down `keep`) and 1–3 build plots.
- **Resource outpost:** the central building — gold mine, lumber mill or ore mine — brings income to the owner; build plots around it. Mines, lumber mills and ore mines cannot be built: they exist only as outpost centers, so gold, wood and metal are obtained only by holding points on the map.
- Later: control-point outposts with bonus gold, special recruitment.

**Placeholder models:** the same open source as the unit models (grab3d), for development only, to be replaced later. While there are few models, each building's name is shown above it.

**Open questions (not blocking the start):**

- Metal: where is it mined from.
- Workers: a separate unit (individually or as a squad), where they are recruited (keep or a building), whether they can gather resources.
- Balance: a player without outposts has no gold, wood or metal income — is a larger starting stock needed, or a nearby "own" outpost at each base?

**Tasks:**

- [x] Assets: placeholders in `art/source/buildings/` (grab3d, development only); `tools/meshbake` (gltf-transform + meshoptimizer) → 3 LODs with shared materials, `engine/models/StaticModel`
- [x] `Building` schema: kind, `placement`, footprint, HP, armor, model, plots, `buildTime`, `cost`, `economy`, `trains`, `noBuildRadius`, `capture`
- [x] Map slots: keep (position, rotation), outposts, plots on the keep / outpost type
- [x] Buildings in the simulation: an entity with HP and an owner, navigation blocking, hash; building attack order (RMB), damage in melee and from volleys over the footprint, destruction with a collapse animation; ruins — together with outposts
- [x] Navigation: a building blocks passability grid cells, flow fields are recomputed on construction and destruction (deterministically)
- [x] Plots: click on a free plot → menu of the faction's buildings that fit by size; timed construction without workers (the building "grows", HP rises with progress), building panel with cancel; resource refund — in step 4
- [x] Workers: builder unit (`builders`, tag `worker`, one at start), "build here" order, progress advances only while workers are nearby (each additional one +50% speed), RMB on your own foundation — help; workers don't engage in combat on their own; worker recruitment — in step 4
- [x] Free placement: anywhere on the map except buildings' no-build circles (`noBuildRadius`: keep 30, outposts 20, farm 12 — farms can't be placed right next to each other), terrain check, map bounds, overlaps with buildings and plots (exact rotated-rectangle check); green / red ghost, no-build circles visible during placement, R to rotate; Go re-checks
- [x] Resources per [`RESOURCES.md`](RESOURCES.md): catalog `content/resources.json` (gold, wood, metal, food), `income` / `provide` effects, starting stock, outpost income, building and squad costs (refunded on cancel), food is the army cap (the queue reserves it); `storage` / `upkeep` / crowding — later
- [x] Outposts: resource ones (gold mine / lumber mill / ore mine + plots) on `plains`; capture 10 s, paused when contested, progress decays with no troops; a captured one can be attacked, a destroyed one becomes neutral again; income to the owner
- [x] Recruitment: queue of up to 5 in a building (barracks — army, keep — workers), cancel with refund, rally point (RMB on the ground while the building panel is open), the squad exits in front of the building
- [x] Protocol: buildings, plots, capture, players (faction, human / AI, defeat), local player's resources and queues; commands plot / build / help / cancel / train / untrain / rally
- [x] Player slots: each has a faction and a controller — client or AI (AI badge; the AI itself — in phase 4); `control <player>` in the dev console switches which player the client plays
- [x] Dev console (F2 or the key left of 1), all commands — [`DEV_CONSOLE.md`](DEV_CONSOLE.md): commands go through the simulation (end up in the replay), parsing in `internal/devconsole`; autocomplete and history
- [x] Labels with the name and an HP bar above buildings (while there are few placeholder models)
- [x] HUD ("stone and bronze" theme, Cinzel + Alegreya Sans fonts): player banners with You / AI badges; round minimap (terrain, buildings, squads, camera frame; LMB — camera, RMB — order); resource plaque with "+N"; portraits of selected squads with an HP ring and soldier count; hero column; ring menu for plots and buildings (recruitment, queue, cancel, parchment tooltip)
- [x] Command card (bottom right, 4×3 grid): Move (G), Stop (X), Hold (H), Attack (F: on an enemy — attack, on the ground — attack-move), Patrol (P); abilities in the bottom row (1–4); for workers — available free buildings (1–…); target selection mode (LMB — target, RMB / Esc — cancel, Shift — repeat)
- [x] Orders in the simulation: stop, hold (attacks within radius, doesn't pursue), attack-move (fights along the way and keeps moving), patrol (attack-move back and forth)
- [x] Victory condition: the enemy keep is destroyed (Victory / Defeat banner); the enemy has a static base for now (AI — in phase 4)
- [ ] Determinism: "build → recruit → destroy" golden test; hash and replay include buildings and resources

**Proposed path:**

1. ✅ Placeholder assets and rendering of static models with LOD; keeps and outposts on `plains`, plots.
2. ✅ Buildings in the simulation and navigation: the keep can be attacked and destroyed, victory condition.
3. ✅ Plots around the keep: choosing and constructing a building without workers.
4. ✅ Resources, food as the army cap, recruiting squads from buildings, player slots (human / AI, `control` in the console).
5. ✅ Workers and free construction with different footprints (done before step 4).
6. ✅ Outposts: capture, retaking, plots (income — with resources in step 4).
7. HUD ✅, command card and orders ✅; next — "build → recruit → destroy" golden test, performance measurement with bases, assets from the list below.

**Done when:** on the standard map you can build up the plots, place free buildings with workers, recruit an army within the food cap and destroy the enemy keep.

### Design of factions, units, heroes and abilities

- [ ] Lore and world description ([`lore/`](lore/README.md)): world, history, geography, magic, peoples, heroes, campaign — structure and templates are ready, fill them in

First on paper (`docs/FACTIONS.md`): the faction rosters determine which mechanics the simulation is missing. The map editor comes after this block.

- [ ] Think through all factions: concept, play style, strengths and weaknesses, visual style; two in detail, the rest as sketches. Described in `FACTIONS.md`: human Empire, High Elves, Dark Elves, orcs (the Dark Lord's army), Dwarves; **Celestials — TODO**
- [ ] Units of each faction: role, what it counters and what it loses to, squad size, cost and food, where it is recruited
- [ ] Buildings of each faction: plots or free placement, what they recruit and unlock, levels
- [x] Heroes: 3 per faction, 5 levels (skill / aura / skill / aura / ultimate), recruitment and revival — for five factions
- [ ] Abilities and skills: active (target, area, cooldown), passive, auras, buffs and debuffs
- [ ] Upgrades: weapons, armor, special squad and building upgrades
- [ ] Faction powers (analog of the BFME spell book): what earns points, what gets unlocked
- [x] Damage and armor types (section 1.1 of `FACTIONS.md`); counter table — during balancing
- [ ] Output: a list of mechanics missing from the simulation — the next block follows from it

**Next, based on the design results:**

- [ ] Shared mechanics in the simulation (data-driven):
  - [x] Damage types (piercing / slashing / bludgeoning, fire / lightning / ice / acid / poison / necrotic / radiant), armor classes, magic defense, resistances, immunities, building material, tag-based bonuses; parameters — `content/combat.json`
  - [x] Effects: burning (units and buildings), stun, movement and attack slow, acid (−armor, stacks), poison with spreading (up to 2), lifesteal; visuals — `StatusEffectsView` particle pool; dev console `hit <type> <damage>`
  - [x] Multiple attacks per unit: `melee` and / or `ranged` in the unit JSON; with both — shoots from range, switches to melee when the enemy closes in; attack targets `targets: ["ground", "air"]` (default — ground); `flying` flag on the unit (melee and charge don't hit flyers); stances (manual switching) — with the ability system
  - [x] Ability system ([`ABILITIES.md`](ABILITIES.md)): kinds active / toggle / passive / aura; targets self / point / unit with approach to range; effects damage / heal / status / knockback / dispel / buff / zone; modifiers (armor, damage, speed, range, damage taken, invulnerability, extra damage of a type); autocast; command card: buttons, cooldowns, toggling, autocast on RMB, target selection; cast and zone visuals
  - [x] Next-level abilities, part A ([`ABILITIES.md`](ABILITIES.md)): chain effects (`chain`), cones (`shape: cone`), summoning (`summon`, temporary and with a "one at a time" limit), teleport (`teleport`, including mass teleport to your own building), transformation (`transform`); visuals — chain lightning, cone fan, cone preview while aiming, shape size; examples — battle mages (Kingdom: chain lightning, blink), packleader (Warband: ice fan, call of the pack, beast form); faction `start` — the starting army, separate from the recruitment list
  - [x] Next-level abilities, part B: modes (`kind: mode` — elements; `byMode` on auras / passives, `mode` on effects), `attackType` / `resist` / `abilityHaste` modifiers, tag-based auras, ultimates (`ultimate`, gold frame); Kingdom archmage with elementals and meteor shower; command card — element button (LMB / RMB)
  - [x] Requirements (`requires` on buildings, squads, upgrades, abilities) and upgrades (`content/upgrades`: research in the building queue, modifiers on units by id / tags, levels via requirements, replacement groups, unlocking abilities); forge with weapons I–II, armor I–II and fire arrows (fire mode for archers); research buttons in the ring menu, "Requires: …" in tooltips
  - [x] Hero experience and levels ([`ABILITIES.md`](ABILITIES.md), "Heroes"): experience for enemy deaths nearby and outpost captures, 5 levels (ability i unlocks at level i+1, +damage / +armor / +health per level), one hero of each type, revival at the keep keeping the level; HUD: level badge and experience bar, fallen heroes grayed out, "Revive … (level N)" in the keep's ring, "Unlocks at hero level N" on buttons; dev console `xp <n>`
  - [x] Flying units: `"flying": true` on a unit — they fly in a straight line over terrain, water and buildings (no navigation), don't push against ground units, are hit only by attacks with `targets: ["air"]`; the `ground` status (landing) temporarily makes them ground units; rendering — flight height, bobbing, shadow and selection ring on the ground, arrows at air targets; content — harpies for the horde (barracks), griffin riders (only via `spawn` for now)
- [x] One faction fully playable on `plains` — the Empire (`empire`, formerly `kingdom`), step by step:
  - [x] 1. Roster and buildings: barracks (swordsmen, spearmen), archery range (archers), stables (knights, require barracks), mages' guild (fire, water, lightning mages, healers), forge, farm; heroes — Paladin, Archmage, King (keep); `immune`, `workSpeed` modifiers (auras on buildings: construction and recruitment speed), self buffs; armor classes per `FACTIONS.md`
  - [x] Keep base ([`BASES.md`](BASES.md)): `base` generator (circle / square / polygon outline, plot rings, gates, tower slots), S / M / L plots (`content/plots.json`), "plot and / or field" buildings, terrain leveling under the base; the wall outline is drawn on the ground for now; fix for icon flickering in the building ring and command card
  - [x] 2. Siege and towers: workshop (ballistas, trebuchet), minimum range (`minRange`), area damage (`splash`, half at the edge, hits buildings); archer / ballista / trebuchet towers on tower slots; fire mode for towers (`toggles`, button in the ring) after an upgrade; upgrades on towers (`buildings` on the upgrade): fire arrows for ballistas, improved mechanism (−25% reload for trebuchets and their towers); repair by builders (RMB on a damaged building, machines nearby repair themselves); machines rendered as static models (wheels, trebuchet arm with counterweight and sling, ballista recoil), heavy projectiles with a dust cloud
  - [x] 3. Stances and formations: spear wall (`rooted`, `speedDamage`: damage from enemy speed, the charge is absorbed, cannot be trampled), knights' wedge (`formation: wedge`, `charge`: +speed, +charge, deeper through the formation, +flank damage)
  - [x] 4a. Base wall ([`BASES.md`](BASES.md)): base radius 70 m, wall along the outline (`castle_wall` segments with HP, breaches), gates only for your own units (per-player navigation), infantry climbs stairs and walks on the wall (+25% range for ranged units, melee by levels), archer tower on tower slots, attacking buildings (`attack`, `volley`); `castle-wall.glb` model with LOD (rough simplification for far levels), wall instancing; soldiers path around walls via the navigation field
  - [x] Gatehouse: `gatehouse` model with LOD (leaves and portcullis separate), one building per 3 segments with a 5 m passage, BFME-style toggle gate (open for everyone / closed; closing waits for an empty passage), animation, button in the ring; the wall walkway is interrupted at the gates
  - [x] "Attack ground" order (`T`) for all squads with a ranged attack: approach to range (accounting for the dead zone), volleys at the point until a new order
  - [x] 4b. Instead of walls in the field — a watchtower (`watchtower`, `garrison: true`): built by workers in the field, one infantry squad climbs onto the platform, as on a wall (+25% range for ranged units, melee from the ground can't reach; tower destroyed — the squad falls down); RMB on your own tower — climb up

### Render performance

The current state is in `ARCHITECTURE.md`, "Render optimizations". What remains:

- [ ] Quality settings in the menu: render scale (50–100%), shadows on / off, particle count, LOD distance, HP bars (all / selected / none); save in settings
- [ ] Shared effects pool: statuses (burning, poison, ice, stun, acid, burning buildings) — ✅ `StatusEffectsView`; explosions, lightning, ability projectiles — with abilities
- [ ] Pool of HTML labels and culling of labels off-screen / beyond a set distance
- [ ] Instancing of trees and decor (with the map editor); LOD or impostors for trees
- [ ] Chunked terrain with LOD — for maps larger than 512×512
- [ ] Shadows (if needed): cascaded or baked, with a toggle in settings
- [ ] Impostors for the farthest soldier LOD — if weak hardware can't keep up
- [ ] Occlusion culling — only if scenes with heavy occlusion appear
- [ ] Measurements on Windows and Linux with bases, effects and 1000+ soldiers

### Art and assets for the HUD and game

Full list with technical requirements (polygon and vertex budgets, clips, node names) — [`ASSETS.md`](ASSETS.md). The code currently uses the idle / walk / run / attack / die clips; `cast`, `work`, `brace` and PNG icons by id are hooked up when the art appears.

Right now everything is drawn in code (CSS frames, SVG icons). Replace as assets appear (CC0, generated or our own):

- [ ] Squad and hero portraits: square PNGs of 256×256 or more for each type (swordsmen, archers, knights, worker, captain, raiders, wolf riders, packleader); for the large portrait of the selected unit — 512×512 or more
- [ ] Building icons for the ring menu and command card: barracks, farm, keep, gold mine, lumber mill, ore mine (128×128 or more, transparent background)
- [ ] Order icons: move, stop, hold, attack, patrol, rally, cancel, build
- [ ] Ability icons (currently all are a crown)
- [ ] Resource icons: gold, wood, metal, food (currently colored stones)
- [ ] Painted frames: minimap ring, resource plaque, command card frame, medallion, tooltip panel (PNG with transparency / 9-slice)
- [ ] Faction crests for player banners and the selection screen
- [x] Cursors — placeholders and logic ([`UX.md`](UX.md)): 28 states (select, inspect, scroll, move, attack, not allowed, climb, build, repair, capture, rally point, target modes, cast, plot, construction) — SVG → PNG
- [ ] Cursors — custom images instead of SVG placeholders
- [ ] Heading font with Cyrillic (Cinzel lacks it): e.g. Cormorant SC or Philosopher — if the UI will be in Russian
- [ ] Models: final units and buildings instead of grab3d placeholders; ruins, foundation / scaffolding for construction
- [ ] UI sounds: click, refusal, "unit ready", "building complete", base under attack


## Phase 2 — Map Editor (MVP)

The editor works with the same base and building schemas already proven in phase 1.5.

- [ ] Campaign bases ([`BASES.md`](BASES.md)): a `base` block on the player slot — the generator with different numbers or a hand-made outline, gates, tower slots and plots
- [ ] New map, saving and loading
  - [ ] Map parameters: id, name, author, description, size (multiple of the grid step), height step, max height, seed or flat terrain, number of players (2–8)
  - [ ] Format: `content/maps/<id>/map.json` + `height.r16` (+ later `splat.png`, `objects.json`); format version and migrations
  - [ ] Go service `MapService`: list, create, load, save (atomic write), validation via `internal/data`
  - [ ] Editor screens: "New map" (form), "Open" (list with thumbnails), "Save / Save as", unsaved-changes flag
  - [ ] Map thumbnail (top-down render → `preview.png`), also needed for map selection in the game
- [ ] "Base and buildings" layer
  - [ ] Player slots: keep (position, rotation), build zone, team (for 2v2), default color
  - [ ] Plots: placement, rotation, size, allowed categories, owner or neutral
  - [ ] Outposts and neutral buildings (watchtowers; later monster camps)
  - [ ] Default starting conditions: resources, command point cap, starting squads (can be overridden in the lobby)
  - [ ] Gizmo: dragging and rotation, zone display, error highlighting (impassable ground, overlaps)
- [ ] Terrain brushes: raise, lower, smooth, flatten; re-check bases and plots after changes
- [ ] Terrain painting (splat map, 4–8 layers)
- [ ] Object placement: trees, rocks, decor (instancing)
- [ ] Impassable zones (manual marking on top of slope)
- [ ] Undo/redo: a shared command stack for all tools
- [ ] "Test run" of the map from the editor

**Proposed path:**

1. `MapService` and switching the game to load maps through it.
2. Editor skeleton: "New" / "Open" / "Save" screens, terrain and camera from the game, undo/redo stack.
3. "Base and buildings" layer.
4. Terrain brushes, then painting and objects.
5. Test run.

## Phase 3 — Content Editor

- [ ] Forms based on `internal/data` schemas: unit, squad, hero, ability, faction
- [ ] 3D preview of the model and animations
- [ ] Reference validation (models, abilities, faction units)
- [ ] `tools/`: VAT baking from glTF

## Phase 4 — Game Expansion

- [ ] Base development: building levels, upgrades (forge: weapons and armor), walls around the keep
- [ ] Squad experience and veterancy (done for heroes)
- [ ] Volley visuals: a cloud of arrows instead of a dot; siege engines
- [ ] Walls and sieges
- [ ] Fog of war
- [ ] Faction powers (analog of the spell book)

## Phase 5 — Multiplayer

- [ ] Deterministic lockstep on top of `transport`
- [ ] Lobby, connecting, faction selection
- [ ] Desync detector based on the state hash
- [ ] Replays (seed + command log)

## Phase 6 — Release

- [ ] AI opponent (decided to do it right before release): recruitment, construction, attacks, base defense — to test factions and balance
- [ ] GitHub Actions: builds for Windows / macOS / Linux
- [ ] Installers: NSIS, `.dmg`, `.deb` / `.rpm` / AppImage
- [ ] Signing: Apple Developer ID + notarization, a Windows certificate
- [ ] Auto-update

## Notes

- Starting assets are free placeholders (Kenney, Quaternius). Pipeline: Blender → glTF → VAT.
- Tolkien's names, characters and locations are protected by copyright. A public release needs its own setting.
