# Dev console

A developer console for testing mechanics right in a match. It grows as new mechanics appear: a new command — a new row in the table below.

## How to open

| Key | What it does |
|---|---|
| **F2** | open / close (works on any keyboard layout) |
| the key **left of 1** | same. On a Mac with an ISO keyboard (Italian, German…) this is the `\` / `<` key next to the left Shift — macOS swaps them |
| **Esc** | close |
| **Enter** | run the line |
| **Tab** | complete: command name, squad / building / resource id, player (`me`, `enemy`, `neutral`) |
| **↑ / ↓** | command history (lives as long as the app is open) |

While the console is open, the game keyboard (camera, hotkeys) doesn't work. Clicks on the console itself don't select units.

## How commands pick a target

- **Cursor** — the point on the ground under the mouse at the moment Enter is pressed. `spawn` and `build` appear there, and `kill` / `heal` / `hit` take the building under the cursor.
- **Selection** — the currently selected squads (`kill`, `heal`, `hit`, `xp`).
- **Player**:
  - `me` — the one you're playing as (default);
  - `enemy` — the opponent (1 if you're 0, otherwise 0);
  - `neutral` — neutral;
  - a number `0–254` — the player with that id.

Commands that change the world (`spawn`, `build`, `kill`, `heal`, `hit`, `xp`, `give`) go through the simulation as ordinary orders. They're recorded in the replay, and determinism isn't broken. Line parsing — `internal/devconsole`, execution — `internal/sim/dev.go`.

## Commands

### Help and info

| Command | What it does |
|---|---|
| `help` | list of all commands |
| `help <command>` | syntax of a single command |
| `list` / `list hordes` | live squads: id, owner, type, soldiers / size, position |
| `list buildings` | buildings: id, owner, type, HP, position |
| `list types` | ids of all squad and building types that can be `spawn`ed / `build`t |
| `res [player]` | a player's resources: gold, wood, metal and food (used / available) |
| `hash` | current tick and state hash (determinism check) |
| `clear` | clear the console window (client only) |

### Units and buildings

| Command | What it does |
|---|---|
| `spawn <horde> [count] [player]` | create squads at the cursor; count 1–50. Example: `spawn harpies 3 enemy` |
| `build <building> [player] [rotation°]` | place a **finished** building at the cursor without placement rules (not on occupied ground). Without rotation the building faces the map center. Example: `build barracks me 90` |
| `kill` | kill the selected squads and the building under the cursor |
| `kill all [player]` | kill all squads (or only a player's squads) |
| `heal` | heal the selected squads and the building under the cursor to full health |
| `heal all` | heal all squads and buildings |
| `hit <damageType> <amount>` | hit every soldier of the selected squads (and the building under the cursor) with 1–100000 damage of this type: with armor, resistances and effects. Types: `pierce`, `slash`, `crush`, `fire`, `lightning`, `ice`, `acid`, `poison`, `necrotic`, `radiant`. Example: `hit fire 30` — test burning |
| `xp <amount>` | give experience to the selected heroes (1–1000000). Level thresholds: 150 / 400 / 800 / 1400 total. `xp 1400` — straight to level 5 |

### Economy and players

| Command | What it does |
|---|---|
| `give <resource> <amount> [player]` | add a resource (or take it away if the number is negative), up to ±1000000. Resources: `gold`, `wood`, `metal`. Food is a cap, provided by farms. Example: `give gold 5000` |
| `control <player>` | play as another player: it stops being AI, your previous player becomes AI. Example: `control enemy` — play as the horde |

### Time

| Command | What it does |
|---|---|
| `pause` | pause the simulation |
| `resume` | resume |
| `step [ticks]` | while paused, advance by n ticks (1–1000, default 1); 20 ticks = 1 second |
| `speed` | show the current speed |
| `speed <x>` | speed multiplier 0.1–8 (1 = 20 ticks/s). Example: `speed 4` — speed up construction and recruitment |

## Other debug keys (not the console)

| Key | What it does |
|---|---|
| **F3** | show impassable ground (in red) |
| **F4** | debug panel: FPS, tick, simulation time, hash, LOD stats |

## Common scenarios

- **Test combat with flyers:** select your archers, point the cursor nearby → `spawn harpies 2 enemy`.
- **Level up a hero:** select the captain → `xp 400` (level 3, fire pillar unlocked).
- **Hero revival:** select a hero → `kill` → "Revive …" appears in the keep's ring.
- **Fast economy:** `give gold 10000`, `give wood 5000`, `give metal 2000`, then `speed 4`.
- **Damage effects:** select an enemy → `hit poison 20`, `hit acid 10`, `hit ice 15`.
- **New pack leader abilities** (frost fan, pack call, beast form): `control enemy` → select the pack leader → `xp 1400`.
- **Chain lightning and blink:** `spawn lightning_mages`, nearby `spawn raiders 2 enemy`.
- **Archmage and elements:** `spawn archmage` → select → `xp 1400`; the "Element" button switches fire / ice / lightning (LMB — next, RMB — previous); the elemental summon and meteor shower follow the element.
- **Paladin and Emperor:** `spawn paladin`, `spawn emperor` → select → `xp 1400`; the Emperor's oversight speeds up recruitment in nearby buildings.
- **Walls and gates:** click an own gatehouse → "Close the gate" / "Open the gate"; select archers → right-click an own wall (they'll climb the stairs); `spawn raiders 3 enemy` outside by the wall; `hit crush 3000` on the segment under the cursor → breach.
- **Siege:** `spawn trebuchets`, `spawn ballistas`; targets: `spawn swordsmen 2 enemy` farther away (the trebuchet has a 16 m dead zone), `spawn harpies enemy` for the ballistas; towers — click a tower slot on the wall; tower fire — after `give gold 2000` and research in the forge.
- **Spear wall vs. wedge:** `spawn spearmen` → turn on "Spear Wall"; `spawn knights enemy` farther away → `control enemy`, turn on "Wedge Formation" and attack the spearmen.
- **Watchtower:** `build watchtower` at the cursor → select archers → right-click the tower (they'll climb onto the platform).
- **Attack ground:** select archers (or a trebuchet) → `T` → click the ground where an enemy stands (`spawn swordsmen 2 enemy`).
- **Compare determinism:** `pause`, `hash` on two runs of the same replay.
