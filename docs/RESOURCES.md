# Resources: mechanics and data

Goal: every resource, income, price or cap is set in JSON. Giving a new building income or changing the balance requires no code changes.

## 1. Resource catalog

`content/resources.json` — the list of resources. The code knows nothing about "gold" or "food", only about ids from the catalog.

```json
{
  "resources": [
    { "id": "gold",  "name": "Gold",  "icon": "gold",  "kind": "stock", "start": 500 },
    { "id": "wood",  "name": "Wood",  "icon": "wood",  "kind": "stock", "start": 300 },
    { "id": "metal", "name": "Metal", "icon": "metal", "kind": "stock", "start": 100 },
    { "id": "food",  "name": "Food",  "icon": "food",  "kind": "cap",   "start": 20, "max": 300 }
  ]
}
```

There are two kinds of resources:

- **`stock`** — accumulating. Builds up from income and is spent on construction and recruitment. The optional `max` field caps storage.
- **`cap`** — a limit. Buildings *provide* it, squads *use* it. A squad can be recruited only if `used + needed ≤ provided`. When a squad dies, the cap is freed; when a building is destroyed, the cap shrinks. An army that already exceeds the cap doesn't disappear; you just can't recruit new ones. Food works this way.

`start` is the default value. The map and lobby can override it.

## 2. Effects on buildings

A building isn't "a farm" as such. It has a list of **economic effects**, and any effect can be put on any building.

```json
{
  "id": "farm",
  "placement": ["free", "plot"],
  "cost":  { "gold": 100, "wood": 50 },
  "buildTime": 20,
  "economy": [
    { "type": "provide", "resource": "food", "amount": 10 },
    { "type": "income",  "resource": "gold", "amount": 2, "every": 5,
      "crowding": { "radius": 15, "penalty": 0.25, "min": 0.25 } }
  ]
}
```

| Effect | What it does | Fields |
|---|---|---|
| `income` | Gives the owner the resource every `every` seconds | `resource`, `amount`, `every`, optional `crowding`, `requires` |
| `provide` | Increases the limit of a `cap` resource (while the building stands and is complete) | `resource`, `amount` |
| `storage` | Increases `max` of a `stock` resource | `resource`, `amount` |
| `upkeep` | Deducts a resource (maintenance), optional | `resource`, `amount`, `every` |
| `refund` | Fraction of the cost returned on demolition or cancel | `fraction` |

Prices and consumption use the same resource dictionary:

- **Building:** `cost` — resources deducted when construction is laid down.
- **Squad:** `cost` — same, on recruitment; `uses: { "food": 5 }` — how much cap it takes.

Examples: to make barracks give a little food, add `provide food` to them. To make the keep produce gold, add `income gold` to it. It's a JSON edit, no code.

## 3. Income modifiers

The final income of an effect is computed as:

```
total = amount × crowding × level × faction × map × lobby
```

- **`crowding`** — a crowding penalty, like farms in BFME2. For every building with the same effect within `radius`, income drops by `penalty`, but not below `min`.
- **`requires`** — a condition for the effect to work, e.g. `{ "ownerOf": "outpost" }` or "building is complete". If the condition isn't met, the effect doesn't apply.
- **Level** — multipliers by building level (phase 4): `"levels": [1.0, 1.5, 2.0]`.
- **Faction** — `factions/<id>.json`: `"economy": { "gold": 1.1 }`.
- **Map and lobby** — global multipliers (e.g. "rich map": ×1.5 to all income), set in `map.json` or the match settings.

All multipliers are stored in JSON as decimals and converted to fixed-point at load time.

## 4. Resource outposts

An outpost is an ordinary building with an `income` effect. Only one thing is special: its owner is whoever captured the point.

```json
{
  "id": "gold_mine",
  "placement": ["outpost"],
  "outpost": { "captureTime": 10, "captureRadius": 12, "plots": 3 },
  "economy": [
    { "type": "income", "resource": "gold", "amount": 15, "every": 5 }
  ]
}
```

The sawmill and ore mine differ from the gold mine only in the `resource` field. A new outpost type (e.g. a control point with gold) is a new JSON file.

## 5. How it works in the simulation

- **Player stockpile** — an `[]int64` array (whole resource units), index = the resource's ordinal in the catalog after sorting by id. No maps; the order is deterministic.
- **Income** — each effect has its own fixed-point accumulator. Every tick `total / (every × 20)` is added to it. When a whole unit accumulates, it's credited to the stockpile, and the fractional part stays in the accumulator. This way fractional income like 2.5 per 5 seconds isn't lost.
- **`cap` limits** aren't stored but recomputed from live completed buildings and squads on every change: construction, destruction, recruitment, death.
- **Deduction** is atomic: a construction or recruitment command checks all resources at once and either goes through entirely or is rejected.
- **State hash** includes players' stockpiles and accumulators.

## 6. What the player sees

- **Protocol:** the snapshot for the local player carries the stockpile and caps for each resource (in catalog order), plus income per minute (for the tooltip).
- **HUD:** icon + value for `stock` and "used / provided" for `cap`. On hover — where the income comes from.
- **Build menu:** price with shortfall highlighted, per resource.
- **Floating numbers** "+15" over a building at the moment of crediting — toggleable, useful for balancing.

## 7. Balancing tools

- **Hot reload of `content/`** in the dev build: a JSON edit is picked up on the next match start.
- **F4 panel:** income per resource per minute, list of sources.
- **Economy test:** a golden test of the "base + 2 outposts, 5 minutes" scenario pins the final stockpiles. The test fails when the formula changes, not when numbers are edited: it uses separate test data.
- Later — a table of all buildings and squads with prices and income in the content editor (phase 3).
