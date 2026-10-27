# Lore and world description

This is where the game world lives: history, geography, magic, peoples, heroes, campaign. It is the **canon** for in-game texts, unit descriptions, voice-over, the campaign and art. Mechanics and numbers live in [`../FACTIONS.md`](../FACTIONS.md) and [`../ABILITIES.md`](../ABILITIES.md); lore does not change them, it explains them.


## Rules

- **Tone: mixed.** Heroic peoples and epic battles, but every side has its own reasons, dark spots and a price for victory. No pure good and no pure evil: even the Alkars have their own logic, and the Empire has its own sins.
- **Language.** Lore is written in English. New lore texts the author sends in Russian are translated into English when they are inserted. Canonical names are in [`glossary.md`](glossary.md).
- **Status** for every section:
  - `draft` — a sketch, anything can change;
  - `accepted` — agreed on; other texts already rely on it;
  - `canon` — in the game (descriptions, campaign); changed only deliberately.
- **Link to mechanics.** If lore explains a mechanic (why elves heal on their own land, why dwarves travel by tunnels), add a link to the relevant section of FACTIONS.md.
- **Images** for the lore (map, coats of arms, reference images) live only in `docs/lore/images/`: `world/` for world maps, `<people>/` for one folder per people (`empire/`, `high-elves/`, …), which may contain `orders/`, `cities/` and so on. Links from `.md` files are relative (`images/...` from `docs/lore/`, `../images/...` from `factions/`). Sources for game models are kept separately, in `art/source/`.
- **Questions** in templates (`> ❓ …`) are things that still need to be decided. The answer is written in place of the question.

## Files

| File | What's inside |
|---|---|
| [`world.md`](world.md) | the world as a whole: name, essence, main conflict, eras, what makes the world special |
| [`history.md`](history.md) | chronology: eras and key events before the game starts |
| [`geography.md`](geography.md) | continents, regions, climate, important places; link to the game maps |
| [`magic.md`](magic.md) | magic, elements, gods and powers; how they relate to damage types and abilities |
| [`bestiary.md`](bestiary.md) | creatures outside the factions: elementals, monsters, neutral camps |
| [`campaign.md`](campaign.md) | the single-player campaign story: acts, missions, heroes |
| [`glossary.md`](glossary.md) | glossary: terms, names, place names (EN / RU) |
| [`factions/`](factions/) | one file per people: [`empire.md`](factions/empire.md), [`high-elves.md`](factions/high-elves.md), [`dark-elves.md`](factions/dark-elves.md), [`alkars.md`](factions/alkars.md) (the Alkars, formerly the "Dark Lord's army"), [`dwarves.md`](factions/dwarves.md), [`celestials.md`](factions/celestials.md) |

## Order of work

1. **World:** name, essence, main conflict (`world.md`).
2. **History:** 3–5 eras, key events, why the peoples are at odds (`history.md`).
3. **Peoples:** following the template in `factions/` — who they are, where they come from, what they want, their heroes.
4. **Geography:** who lives where, disputed lands (that is where the maps will be).
5. **Magic and gods** — explaining the elements, necrotic power, light.
6. **Campaign** — once the world and the peoples are accepted.
