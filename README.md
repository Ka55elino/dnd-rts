# dnd-rts

A real-time strategy game in the spirit of *The Battle for Middle-earth II*: bases with plots and walls, squads (hordes) instead of single soldiers, heroes, abilities and up to ~100,000 soldiers on screen.

- **Go** owns all game logic: a deterministic fixed-point simulation (20 ticks per second), content loading, validation of every command.
- **TypeScript + three.js** only renders the game and shows UI hints.
- **Wails 3** packs both into one desktop app; Go streams binary snapshots to the frontend, the frontend sends commands back.

## Requirements

| Tool | Version | Why |
|---|---|---|
| [Go](https://go.dev/dl/) | 1.25+ | simulation, app |
| [Node.js](https://nodejs.org/) | 20.19+ or 22.12+ | frontend (Vite 8), bake tools |
| [Wails 3 CLI](https://v3.wails.io/) | `v3.0.0-beta.26` (same as in `go.mod`) | dev server, bindings, builds |

Install the Wails CLI and check the system:

```sh
go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.26
wails3 doctor
```

`wails3 doctor` lists what is missing for your OS:
- **macOS** — Xcode Command Line Tools (`xcode-select --install`);
- **Windows** — WebView2 runtime (preinstalled on Windows 10/11);
- **Linux** — GTK 3 and WebKitGTK 4.1 dev packages (e.g. `libgtk-3-dev libwebkit2gtk-4.1-dev` on Debian/Ubuntu).

## Running the project

```sh
git clone <repo> dnd-rts
cd dnd-rts
wails3 dev
```

`wails3 dev` installs the frontend packages, generates the Go → TypeScript bindings, starts Vite (port 9245, change with `WAILS_VITE_PORT`) and runs the app with hot reload: frontend changes apply instantly, Go changes rebuild and restart the app.

In the main menu:
- **Game** — a match on the test map `plains`;
- **Editor** — the map editor (early stage);
- **Options** — display mode and settings;
- **Benchmark** — a load test of the Go → JS snapshot pipeline (up to 100,000 entities).

In a match, **F2** opens the developer console (spawn units, give resources, set XP, kill, etc.) — all commands are in [`docs/DEV_CONSOLE.md`](docs/DEV_CONSOLE.md).

## Building

```sh
wails3 build      # release build for the current OS → bin/
wails3 task package   # platform package (.app, installer, AppImage…)
```

Other tasks are in `Taskfile.yml` (run them with `wails3 task <name>`): `build:server` / `run:server` (headless HTTP server mode), `setup:docker` (cross-compilation image), and the platform tasks in `build/<os>/Taskfile.yml`.

If you change exported Go service methods or their types, regenerate the bindings (dev mode does it automatically):

```sh
wails3 task common:generate:bindings
```

## Tests and checks

```sh
go test ./internal/...                      # simulation, data, protocol, terrain, dev console
go vet ./...
cd frontend && npx tsc --noEmit -p .        # frontend type check
```

- The simulation is **deterministic**: `internal/sim` has golden tests that hash the whole world state after a scripted match (`TestSimDeterminismGolden`, `TestCombatDeterminismGolden`, …). If you change game rules or content on purpose, the hash changes — check the behavior first, then update the constant in the test.
- Gameplay values live in `content/` (JSON); `internal/data` validates them at load time, so a broken content file fails `go test ./internal/data`.

## Tools

### `tools/meshbake` — static models (buildings, machines, props)

Takes a `.glb` from `art/source/`, cleans it, joins primitives by material, and makes 3 LODs with meshoptimizer.

```sh
cd tools/meshbake
npm install          # once
npm run bake         # all models
npm run bake -- keep # one model by id
```

- **Config:** `tools/meshbake/models.json` — one entry per model: `id`, `source` (path from the repo root) and optional settings:
  - `lods` — fraction of triangles kept per LOD (default `[1, 0.3, 0.08]`);
  - `error` — max simplification error for LOD1+;
  - `sloppy` — coarse, seam-ignoring simplification for flat-shaded models with UV seams;
  - `keepNamed` — keep named nodes separate (moving parts: gates, wheels, arms).
- **Output:** `frontend/public/models/<id>/lod0.glb`, `lod1.glb`, `lod2.glb`, `model.json`. The log prints triangles per LOD — compare them with the budgets in [`docs/ASSETS.md`](docs/ASSETS.md).

### `tools/vatbake` — animated units

Bakes a skinned character into a Vertex Animation Texture: every frame of every clip becomes vertex positions in a texture, so thousands of units of one type are drawn with one instanced draw call.

```sh
cd tools/vatbake
npm install
npm run bake             # all models
npm run bake -- swordsman
```

- **Config:** `tools/vatbake/models.json` — `fps` (15) and per model:
  - `id`, `source`;
  - `teamMaterials` — materials tinted with the player color;
  - `clips` — game clip name → clip in the `.glb` (`{"from": "walk", "loop": true}`) or a procedural placeholder (`@swing`, `@bow`, `@die`).
- **Output:** `frontend/public/models/<id>/model.json`, `geometry.bin` (3 LODs, vertex colors, team mask), `vat.bin`. The log prints vertices, triangles per LOD and the VAT size.

### `tools/gentrig` — trigonometry tables

Generates `internal/fixed/trig_tables.go` (sin / atan tables for the fixed-point math). The generated file is committed and is the source of truth; **do not regenerate it** unless you mean to change simulation results:

```sh
go run ./tools/gentrig > internal/fixed/trig_tables.go
```

## Adding art

1. Put the source file in `art/source/` following the layout in [`docs/ASSETS.md`](docs/ASSETS.md) (units: `art/source/<faction>/units/<unit>/<skin>.glb`).
2. Add an entry to `tools/vatbake/models.json` (animated) or `tools/meshbake/models.json` (static) and run the bake.
3. Point the content at the model: the `model` field in `content/units/*.json` or `content/buildings/*.json`.
4. Check the budgets in the bake log and the look in the game.

What has to be made: [`docs/ART_LIST.md`](docs/ART_LIST.md). Licenses of the current placeholders: [`art/README.md`](art/README.md).

## Project layout

| Path | What |
|---|---|
| `main.go` | app entry: services, game and benchmark streams, window |
| `internal/sim` | deterministic simulation: squads, combat, abilities, buildings, walls, heroes |
| `internal/fixed` | Q47.16 fixed-point math |
| `internal/data` | content schema, loading and validation, base generator |
| `internal/protocol` | binary snapshot and command format |
| `internal/game` | game session: tick loop, stream to the frontend |
| `internal/services` | Wails services: content, settings |
| `internal/terrain` | heightfield |
| `internal/devconsole` | dev console command parsing |
| `internal/bench` | snapshot pipeline load test |
| `content/` | game data (JSON): units, hordes, buildings, abilities, upgrades, factions, maps |
| `frontend/src` | three.js renderer, HUD, input, screens |
| `frontend/public/models` | baked models (output of the tools) |
| `art/source` | source art (not shipped) |
| `tools/` | bake tools and generators |
| `docs/` | design and technical docs |

## Documentation

| Doc | About |
|---|---|
| [`docs/ROADMAP.md`](docs/ROADMAP.md) | plan by phases and what is done |
| [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) | architecture: Go / JS split, protocol, determinism |
| [`docs/FACTIONS.md`](docs/FACTIONS.md) | factions, units, buildings, heroes, damage and armor |
| [`docs/ABILITIES.md`](docs/ABILITIES.md) | ability and modifier format |
| [`docs/RESOURCES.md`](docs/RESOURCES.md) | economy |
| [`docs/BASES.md`](docs/BASES.md) | bases, plots, walls, gates |
| [`docs/UX.md`](docs/UX.md) | cursors and interaction |
| [`docs/DEV_CONSOLE.md`](docs/DEV_CONSOLE.md) | developer console |
| [`docs/ASSETS.md`](docs/ASSETS.md) | art technical specification |
| [`docs/ART_LIST.md`](docs/ART_LIST.md) | art production list |
| [`docs/lore/`](docs/lore/) | world, history, religions, factions, geography |
