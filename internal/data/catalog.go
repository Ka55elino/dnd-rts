package data

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"path"
	"regexp"
	"slices"
	"strings"

	"dnd-rts/internal/fixed"
)

// Catalog is all loaded content, indexed by ID.
//
// Never range over the maps in simulation code (Go map order is random, which
// breaks determinism) — use the sorted *IDs slices.
type Catalog struct {
	Units     map[string]*Unit
	Hordes    map[string]*Horde
	Factions  map[string]*Faction
	Buildings map[string]*Building
	Maps      map[string]*Map
	Resources map[string]*Resource
	Combat    CombatRules // content/combat.json
	Abilities map[string]*Ability
	Upgrades  map[string]*Upgrade

	UnitIDs     []string
	HordeIDs    []string
	FactionIDs  []string
	BuildingIDs []string
	MapIDs      []string
	ResourceIDs []string // sorted; the index is the resource's slot in player stocks
	AbilityIDs  []string
	UpgradeIDs  []string
	PlotSizes   PlotSizes
}

// Load reads and validates all content from fsys (e.g. os.DirFS("content")).
// It returns every problem found, not just the first one.
func Load(fsys fs.FS) (*Catalog, error) {
	c := &Catalog{
		Units:     map[string]*Unit{},
		Hordes:    map[string]*Horde{},
		Factions:  map[string]*Faction{},
		Buildings: map[string]*Building{},
		Maps:      map[string]*Map{},
		Resources: map[string]*Resource{},
		Abilities: map[string]*Ability{},
		Upgrades:  map[string]*Upgrade{},
	}
	var errs []error

	errs = append(errs, loadKind(fsys, "units/*.json", c.Units, func(u *Unit) string { return u.ID })...)
	errs = append(errs, loadKind(fsys, "hordes/*.json", c.Hordes, func(h *Horde) string { return h.ID })...)
	errs = append(errs, loadKind(fsys, "factions/*.json", c.Factions, func(f *Faction) string { return f.ID })...)
	errs = append(errs, loadKind(fsys, "abilities/*.json", c.Abilities, func(a *Ability) string { return a.ID })...)
	errs = append(errs, loadKind(fsys, "upgrades/*.json", c.Upgrades, func(u *Upgrade) string { return u.ID })...)
	errs = append(errs, loadKind(fsys, "buildings/*.json", c.Buildings, func(b *Building) string { return b.ID })...)
	errs = append(errs, loadKind(fsys, "maps/*/map.json", c.Maps, func(m *Map) string { return m.ID })...)

	c.UnitIDs = sortedKeys(c.Units)
	c.HordeIDs = sortedKeys(c.Hordes)
	c.FactionIDs = sortedKeys(c.Factions)
	errs = append(errs, loadResources(fsys, c.Resources)...)
	errs = append(errs, loadCombat(fsys, &c.Combat)...)
	errs = append(errs, loadPlotSizes(fsys, &c.PlotSizes)...)
	c.BuildingIDs = sortedKeys(c.Buildings)
	c.expandBases()
	c.AbilityIDs = sortedKeys(c.Abilities)
	c.UpgradeIDs = sortedKeys(c.Upgrades)
	c.ResourceIDs = sortedKeys(c.Resources)
	c.MapIDs = sortedKeys(c.Maps)

	errs = append(errs, c.Validate()...)
	if len(errs) > 0 {
		return c, errors.Join(errs...)
	}
	return c, nil
}

func loadKind[T any](fsys fs.FS, pattern string, into map[string]*T, id func(*T) string) []error {
	files, err := fs.Glob(fsys, pattern)
	if err != nil {
		return []error{err}
	}
	slices.Sort(files)

	var errs []error
	for _, file := range files {
		raw, err := fs.ReadFile(fsys, file)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		dec := json.NewDecoder(bytes.NewReader(raw))
		dec.DisallowUnknownFields() // catch typos like "heatlh"
		v := new(T)
		if err := dec.Decode(v); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", file, err))
			continue
		}

		key := id(v)
		if want := idFromPath(file); key != want {
			errs = append(errs, fmt.Errorf("%s: id %q must match file name %q", file, key, want))
			continue
		}
		if _, dup := into[key]; dup {
			errs = append(errs, fmt.Errorf("%s: duplicate id %q", file, key))
			continue
		}
		into[key] = v
	}
	return errs
}

// loadResources reads content/resources.json: {"resources": [...]}.
func loadResources(fsys fs.FS, into map[string]*Resource) []error {
	raw, err := fs.ReadFile(fsys, "resources.json")
	if err != nil {
		return []error{fmt.Errorf("resources.json: %w", err)}
	}
	var file struct {
		Resources []*Resource `json:"resources"`
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&file); err != nil {
		return []error{fmt.Errorf("resources.json: %w", err)}
	}
	var errs []error
	for _, r := range file.Resources {
		if _, dup := into[r.ID]; dup {
			errs = append(errs, fmt.Errorf("resources.json: duplicate id %q", r.ID))
			continue
		}
		into[r.ID] = r
	}
	return errs
}

// ResourceIndex returns the slot of a resource id in ResourceIDs, or -1.
func (c *Catalog) ResourceIndex(id string) int { return slices.Index(c.ResourceIDs, id) }

// idFromPath: "units/swordsman.json" → "swordsman", "maps/valley/map.json" → "valley".
func idFromPath(file string) string {
	if path.Base(file) == "map.json" {
		return path.Base(path.Dir(file))
	}
	return strings.TrimSuffix(path.Base(file), path.Ext(file))
}

var validID = regexp.MustCompile(`^[a-z0-9][a-z0-9_]*$`)
var validColor = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

// Validate checks values and cross references. Safe to call again after edits (editor).
func (c *Catalog) Validate() []error {
	var errs []error
	fail := func(kind, id, format string, args ...any) {
		errs = append(errs, fmt.Errorf("%s %q: %s", kind, id, fmt.Sprintf(format, args...)))
	}

	for _, id := range c.UnitIDs {
		u := c.Units[id]
		if !validID.MatchString(id) {
			fail("unit", id, "id must be lowercase snake_case")
		}
		if u.Health <= 0 {
			fail("unit", id, "health must be > 0")
		}
		if u.Speed <= 0 {
			fail("unit", id, "speed must be > 0")
		}
		if u.Radius <= 0 {
			fail("unit", id, "radius must be > 0")
		}
		if u.Mass <= 0 {
			fail("unit", id, "mass must be > 0")
		}
		if u.Melee != nil && u.Melee.Projectile != nil {
			fail("unit", id, "melee attack can't have a projectile")
		}
		if u.Ranged != nil && u.Ranged.Projectile == nil {
			fail("unit", id, "ranged attack needs a projectile")
		}
		for _, at := range []struct {
			name string
			a    *Attack
		}{{"melee", u.Melee}, {"ranged", u.Ranged}} {
			if at.a == nil {
				continue
			}
			if at.a.Cooldown <= 0 || at.a.Range <= 0 {
				fail("unit", id, "%s.cooldown and %s.range must be > 0", at.name, at.name)
			}
			if pr := at.a.Projectile; pr != nil && (pr.Speed <= 0 || pr.Spread <= 0) {
				fail("unit", id, "%s.projectile.speed and spread must be > 0", at.name)
			}
			for _, t := range at.a.Targets {
				if t != TargetGround && t != TargetAir {
					fail("unit", id, "%s.targets: %q must be ground or air", at.name, t)
				}
			}
			if !validDamageType(at.a.DamageType) {
				fail("unit", id, "%s.damageType %q is unknown", at.name, at.a.DamageType)
			}
			for _, tag := range slices.Sorted(maps.Keys(at.a.Bonus)) {
				if at.a.Bonus[tag] <= 0 {
					fail("unit", id, "%s.bonus %q must be > 0", at.name, tag)
				}
			}
		}
		if ch := u.Charge; ch != nil && (ch.TrampleDamage <= 0 || ch.Knockback <= 0) {
			fail("unit", id, "charge.trampleDamage and charge.knockback must be > 0")
		}
	}

	for _, id := range c.HordeIDs {
		h := c.Hordes[id]
		if !validID.MatchString(id) {
			fail("horde", id, "id must be lowercase snake_case")
		}
		if _, ok := c.Units[h.Unit]; !ok {
			fail("horde", id, "unknown unit %q", h.Unit)
		}
		if h.Size < 1 {
			fail("horde", id, "size must be >= 1")
		}
		if h.Formation.Columns < 1 {
			fail("horde", id, "formation.columns must be >= 1")
		}
		if h.Formation.Spacing <= 0 {
			fail("horde", id, "formation.spacing must be > 0")
		}
		if h.BuildTime < 0 {
			fail("horde", id, "buildTime must be >= 0")
		}
		c.checkAmounts(h.Cost, ResourceStock, func(m string) { fail("horde", id, "cost: %s", m) })
		c.checkAmounts(h.Uses, ResourceCap, func(m string) { fail("horde", id, "uses: %s", m) })
	}

	for _, id := range c.FactionIDs {
		f := c.Factions[id]
		if !validID.MatchString(id) {
			fail("faction", id, "id must be lowercase snake_case")
		}
		if !validColor.MatchString(f.Color) {
			fail("faction", id, "color must be #rrggbb, got %q", f.Color)
		}
		if len(f.Hordes) == 0 {
			fail("faction", id, "must list at least one horde")
		}
		for _, h := range append(append([]string{}, f.Hordes...), f.Start...) {
			if _, ok := c.Hordes[h]; !ok {
				fail("faction", id, "unknown horde %q", h)
			}
		}
		if b, ok := c.Buildings[f.Fortress]; !ok || b.Kind != BuildingFortress {
			fail("faction", id, "fortress %q must be a building of kind %q", f.Fortress, BuildingFortress)
		}
		for _, bid := range f.Buildings {
			if b, ok := c.Buildings[bid]; !ok || b.Kind != BuildingRegular {
				fail("faction", id, "building %q must be a building of kind %q", bid, BuildingRegular)
			}
		}
	}

	for _, id := range c.BuildingIDs {
		b := c.Buildings[id]
		if !validID.MatchString(id) {
			fail("building", id, "id must be lowercase snake_case")
		}
		switch b.Kind {
		case BuildingWall, BuildingGate:
			if b.WalkHeight <= 0 {
				fail("building", id, "walls and gates need walkHeight > 0")
			}
			if len(b.Placement) > 0 || len(b.Plots) > 0 || b.Base != nil {
				fail("building", id, "walls and gates are placed with their base")
			}
		case BuildingFortress, BuildingOutpost:
			if len(b.Placement) > 0 {
				fail("building", id, "placement is only for kind %q (fortresses and outposts are placed by the map)", BuildingRegular)
			}
		case BuildingRegular:
			if len(b.Placement) == 0 {
				fail("building", id, "placement must list %q and/or %q", PlacementPlot, PlacementFree)
			}
			for _, p := range b.Placement {
				if p != PlacementPlot && p != PlacementFree && p != PlacementTower {
					fail("building", id, "unknown placement %q", p)
				}
			}
			if len(b.Plots) > 0 {
				fail("building", id, "only fortresses and outposts have plots")
			}
			if b.BuildTime <= 0 {
				fail("building", id, "buildTime must be > 0")
			}
		default:
			fail("building", id, "kind must be %q, %q, %q, %q or %q", BuildingFortress, BuildingOutpost, BuildingRegular, BuildingWall, BuildingGate)
		}
		if b.Model == "" {
			fail("building", id, "model is required")
		}
		if b.ModelScale < 0 {
			fail("building", id, "modelScale must be > 0")
		}
		if b.Footprint.X <= 0 || b.Footprint.Y <= 0 {
			fail("building", id, "footprint.x and footprint.y must be > 0")
		}
		if b.Health <= 0 || b.Armor < 0 {
			fail("building", id, "health must be > 0 and armor >= 0")
		}
		c.checkAmounts(b.Cost, ResourceStock, func(m string) { fail("building", id, "cost: %s", m) })
		for i, e := range b.Economy {
			r, ok := c.Resources[e.Resource]
			switch {
			case !ok:
				fail("building", id, "economy %d: unknown resource %q", i, e.Resource)
			case e.Type == EffectIncome && (r.Kind != ResourceStock || e.Amount <= 0 || e.Every <= 0):
				fail("building", id, "economy %d: income needs a stock resource, amount > 0 and every > 0", i)
			case e.Type == EffectProvide && (r.Kind != ResourceCap || e.Amount <= 0):
				fail("building", id, "economy %d: provide needs a cap resource and amount > 0", i)
			case e.Type != EffectIncome && e.Type != EffectProvide:
				fail("building", id, "economy %d: unknown type %q", i, e.Type)
			}
		}
		for _, h := range b.Trains {
			if _, ok := c.Hordes[h]; !ok {
				fail("building", id, "trains unknown horde %q", h)
			}
		}
		if a := b.Attack; a != nil && (a.Projectile == nil || a.Range <= 0 || a.Cooldown <= 0 || a.Damage <= 0) {
			fail("building", id, "attack: buildings shoot (projectile), need range, cooldown and damage > 0")
		}
		if b.Garrison && b.WalkHeight <= 0 {
			fail("building", id, "garrison buildings need walkHeight > 0 (platform height)")
		}
		if b.Volley < 0 {
			fail("building", id, "volley must be >= 0")
		}
		for _, t := range b.Toggles {
			if a, ok := c.Abilities[t]; !ok || a.Kind != AbilityToggle {
				fail("building", id, "toggles: %q must be a toggle ability", t)
			}
		}
		if len(b.Toggles) > 1 {
			fail("building", id, "at most one toggle per building (one flag in the snapshot)")
		}
		if c := b.Capture; c != nil && (b.Kind != BuildingOutpost || c.Time <= 0 || c.Radius <= 0) {
			fail("building", id, "capture is for outposts and needs time > 0 and radius > 0")
		}
		if b.NoBuildRadius < 0 {
			fail("building", id, "noBuildRadius must be >= 0")
		}
		for i, p := range b.Plots {
			if p.Size <= 0 {
				fail("building", id, "plot %d: needs a size > 0 or a size class", i)
			}
		}
	}

	c.validateBases(fail)
	c.validateCombat(fail)
	c.validateAbilities(fail)
	c.validateUpgrades(fail)
	for _, id := range c.UnitIDs {
		u := c.Units[id]
		if u.ArmorClass != "" && u.ArmorClass != ArmorLight && u.ArmorClass != ArmorMedium && u.ArmorClass != ArmorHeavy {
			fail("unit", id, "armorClass must be light, medium or heavy")
		}
		if hd := u.Hero; hd != nil {
			if len(hd.XP) == 0 || hd.XP[0] != 0 || !slices.IsSorted(hd.XP) {
				fail("unit", id, "hero.xp must start at 0 and grow")
			}
		}
		if u.MagicDefense < 0 {
			fail("unit", id, "magicDefense must be >= 0")
		}
		if u.Charge != nil && !validDamageType(u.Charge.DamageType) {
			fail("unit", id, "charge.damageType %q is unknown", u.Charge.DamageType)
		}
		for _, t := range slices.Sorted(maps.Keys(u.Resist)) {
			if !slices.Contains(DamageTypes, t) || u.Resist[t] < 0 || u.Resist[t] > fixed.One {
				fail("unit", id, "resist %q: unknown type or not in 0..1", t)
			}
		}
		for _, e := range u.Immune {
			if !slices.Contains(StatusEffects, e) {
				fail("unit", id, "immune: unknown effect %q", e)
			}
		}
	}
	for _, id := range c.BuildingIDs {
		b := c.Buildings[id]
		if m := b.Material; m != "" && m != MaterialWood && m != MaterialStone && m != MaterialMagic {
			fail("building", id, "material must be wood, stone or magic")
		}
	}

	for _, id := range c.ResourceIDs {
		r := c.Resources[id]
		if !validID.MatchString(id) {
			fail("resource", id, "id must be lowercase snake_case")
		}
		if r.Kind != ResourceStock && r.Kind != ResourceCap {
			fail("resource", id, "kind must be %q or %q", ResourceStock, ResourceCap)
		}
		if r.Start < 0 || r.Max < 0 || (r.Max > 0 && r.Start > r.Max) {
			fail("resource", id, "start and max must be >= 0 and start <= max")
		}
	}

	for _, id := range c.MapIDs {
		m := c.Maps[id]
		if !validID.MatchString(id) {
			fail("map", id, "id must be lowercase snake_case")
		}
		if m.Width <= 0 || m.Height <= 0 {
			fail("map", id, "width and height must be > 0")
		}
		if t := m.Terrain; t.CellSize <= 0 || t.MaxHeight <= 0 {
			fail("map", id, "terrain.cellSize and terrain.maxHeight must be > 0")
		} else if fixed.FromInt(m.Width)%t.CellSize != 0 || fixed.FromInt(m.Height)%t.CellSize != 0 {
			fail("map", id, "width/height must be multiples of terrain.cellSize")
		}
		if len(m.Players) < 2 {
			fail("map", id, "needs at least 2 player start positions")
		}
		inside := func(p fixed.Vec2) bool {
			return p.X.Abs().Int()*2 <= m.Width && p.Y.Abs().Int()*2 <= m.Height
		}
		for i, p := range m.Players {
			if !inside(p.Start) {
				fail("map", id, "player %d start is outside the map (map is centered at 0,0)", i)
			}
			if !inside(p.Fortress.Pos) {
				fail("map", id, "player %d fortress is outside the map", i)
			}
		}
		for i, o := range m.Outposts {
			if b, ok := c.Buildings[o.Building]; !ok || b.Kind != BuildingOutpost {
				fail("map", id, "outpost %d: building %q must be of kind %q", i, o.Building, BuildingOutpost)
			}
			if !inside(o.Pos) {
				fail("map", id, "outpost %d is outside the map", i)
			}
		}
	}
	return errs
}

// checkAmounts validates a resource → amount map against the resource kind.
func (c *Catalog) checkAmounts(a Amounts, kind string, fail func(string)) {
	for _, id := range slices.Sorted(maps.Keys(a)) {
		r, ok := c.Resources[id]
		switch {
		case !ok:
			fail(fmt.Sprintf("unknown resource %q", id))
		case r.Kind != kind:
			fail(fmt.Sprintf("%q is a %s resource, expected %s", id, r.Kind, kind))
		case a[id] < 0:
			fail(fmt.Sprintf("%q must be >= 0", id))
		}
	}
}

func sortedKeys[T any](m map[string]*T) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}
