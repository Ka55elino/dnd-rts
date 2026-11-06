package data

import "dnd-rts/internal/fixed"

// Content schemas. Files live in content/<kind>/<id>.json (maps: content/maps/<id>/map.json).
// Simulation-relevant numbers are fixed.Fixed: written as plain decimals in JSON
// (e.g. "speed": 3.5) and converted deterministically on load.
// Times are in seconds; the simulation converts them to ticks.

// Unit is a single soldier / creature type.
type Unit struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Model string `json:"model"` // path relative to content/, e.g. "models/swordsman.glb"
	// Mesh: a static model (tools/meshbake) for machines instead of an animated
	// one; its parts (wheels, arm) are moved by the client.
	Mesh string `json:"mesh,omitempty"`

	Health fixed.Fixed `json:"health"`
	Armor  fixed.Fixed `json:"armor"` // flat reduction of physical damage
	Speed  fixed.Fixed `json:"speed"` // units per second

	// Defense (docs/FACTIONS.md §1.1).
	ArmorClass   string                 `json:"armorClass,omitempty"`   // light / medium (default) / heavy
	MagicDefense fixed.Fixed            `json:"magicDefense,omitempty"` // flat reduction of magic damage
	Resist       map[string]fixed.Fixed `json:"resist,omitempty"`       // damage type → fraction removed (0..1)
	Immune       []string               `json:"immune,omitempty"`       // status effects that don't apply
	Radius       fixed.Fixed            `json:"radius"`                 // collision radius
	Mass         fixed.Fixed            `json:"mass"`                   // for knockback / trample

	// Attacks: a unit may have a melee attack, a ranged attack, both or none.
	// With both, it shoots from afar and switches to melee when enemies close in.
	Melee     *Attack  `json:"melee,omitempty"`
	Ranged    *Attack  `json:"ranged,omitempty"`    // needs a projectile; may require an upgrade (Attack.Requires)
	Flying    bool     `json:"flying,omitempty"`    // only attacks that target "air" can hit it
	Charge    *Charge  `json:"charge,omitempty"`    // cavalry: tramples infantry at speed
	Abilities []string `json:"abilities,omitempty"` // Ability.IDs
	// Hero: levels and experience (heroes only). Ability i unlocks at level i+1.
	Hero *HeroDef `json:"hero,omitempty"`
	// XP: experience this unit is worth when it dies (0 = health / 10).
	XP int `json:"xp,omitempty"`
	// Sight: how far the unit sees through the fog of war (0 = the default, DefaultSight).
	Sight fixed.Fixed `json:"sight,omitempty"`
	// Trap: the unit is a ground trap (placed by an ability): it stands still and goes off when
	// an enemy steps within Radius — its Ability's effects at its spot, then it is gone.
	Trap *Trap    `json:"trap,omitempty"`
	Tags []string `json:"tags,omitempty"` // "infantry", "cavalry", "ranged", ...
}

// HasTag reports whether the unit has a tag.
func (u *Unit) HasTag(tag string) bool {
	for _, t := range u.Tags {
		if t == tag {
			return true
		}
	}
	return false
}

// DefaultSight: vision radius of units and buildings without their own (world units).
var DefaultSight = fixed.FromInt(36)

// Trap: see Unit.Trap.
type Trap struct {
	Radius  fixed.Fixed `json:"radius"`
	Ability string      `json:"ability"` // an active ability; its effects go off at the trap
}

// Charge describes trampling: a unit moving fast through enemy infantry
// damages and knocks them down.
type Charge struct {
	DamageType    string      `json:"damageType,omitempty"` // default crush
	TrampleDamage fixed.Fixed `json:"trampleDamage"`
	Knockback     fixed.Fixed `json:"knockback"` // initial knockback speed, units per second
}

// Attack describes a unit's basic attack.
type Attack struct {
	Damage   fixed.Fixed `json:"damage"`
	Range    fixed.Fixed `json:"range"`
	Cooldown fixed.Fixed `json:"cooldown"` // seconds between attacks

	// DamageType: slash by default, pierce for projectiles (see Type()).
	DamageType string `json:"damageType,omitempty"`
	// Bonus: damage multiplier against targets with a tag, e.g. {"cavalry": 1.5}.
	Bonus map[string]fixed.Fixed `json:"bonus,omitempty"`

	// Projectile: required for ranged attacks (the horde fires volleys).
	Projectile *Projectile `json:"projectile,omitempty"`

	// Targets: what the attack can hit — "ground", "air" or both. Default: ground.
	Targets []string `json:"targets,omitempty"`

	// Requires: the attack exists only once the player has these (a troll's sack of rocks).
	Requires *Requires `json:"requires,omitempty"`

	// MinRange: ranged attacks can't hit closer than this (trebuchets).
	MinRange fixed.Fixed `json:"minRange,omitempty"`
	// Splash: every shot hurts everyone within this radius of where it lands
	// (full damage at the centre, half at the edge) and the buildings there.
	Splash fixed.Fixed `json:"splash,omitempty"`
}

// Projectile describes a ranged volley.
type Projectile struct {
	Speed  fixed.Fixed `json:"speed"`  // units per second
	Spread fixed.Fixed `json:"spread"` // landing radius around the aim point at close range
}

// Horde is what the player builds and commands: N units of one type in formation.
type Horde struct {
	ID        string      `json:"id"`
	Name      string      `json:"name"`
	Unit      string      `json:"unit"` // Unit.ID
	Size      int         `json:"size"`
	Formation Formation   `json:"formation"`
	Cost      Amounts     `json:"cost"`           // stock resources paid when training starts
	Uses      Amounts     `json:"uses,omitempty"` // cap resources (e.g. food) taken while the horde lives
	BuildTime fixed.Fixed `json:"buildTime"`      // training time, seconds
	Requires  *Requires   `json:"requires,omitempty"`
	// Unique: only one of this horde per player at a time (heroes). A dead
	// unique horde is revived in the same building, keeping its level.
	Unique     bool        `json:"unique,omitempty"`
	ReviveTime fixed.Fixed `json:"reviveTime,omitempty"` // seconds (0 = 2 × buildTime)
	ReviveCost Amounts     `json:"reviveCost,omitempty"` // empty = cost
}

// HeroDef: experience needed for each level and growth per level.
type HeroDef struct {
	XP             []int       `json:"xp"`                       // total XP for level 1, 2, … (first is 0)
	PerLevel       Modifiers   `json:"perLevel,omitempty"`       // added once per level above 1
	HealthPerLevel fixed.Fixed `json:"healthPerLevel,omitempty"` // + fraction of health per level above 1
}

// MaxLevel returns the hero's top level.
func (h *HeroDef) MaxLevel() int { return len(h.XP) }

// Resource kinds (see docs/RESOURCES.md).
const (
	ResourceStock = "stock" // accumulates from income, spent on buildings and hordes
	ResourceCap   = "cap"   // a limit: buildings provide it, hordes use it (food)
)

// Resource is an entry of content/resources.json. The simulation knows
// resources only by id; nothing in code is specific to gold or food.
type Resource struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Icon  string `json:"icon"`
	Kind  string `json:"kind"`          // ResourceStock / ResourceCap
	Start int    `json:"start"`         // stock: starting amount; cap: base limit before buildings
	Max   int    `json:"max,omitempty"` // stock: storage limit; cap: hard limit (0 = none)
	Order int    `json:"order"`         // display order in the HUD
}

// Amounts maps resource ids to whole amounts, e.g. {"gold": 150, "wood": 50}.
// Simulation code must iterate Catalog.ResourceIDs, never the map.
type Amounts map[string]int

// Economy effect types.
const (
	EffectIncome  = "income"  // + Amount of a stock resource every Every seconds
	EffectProvide = "provide" // + Amount to a cap resource while the building stands (finished)
)

// Effect is an economic effect of a building (docs/RESOURCES.md §2).
type Effect struct {
	Type     string      `json:"type"`
	Resource string      `json:"resource"`
	Amount   fixed.Fixed `json:"amount"`
	Every    fixed.Fixed `json:"every,omitempty"` // seconds (income)
}

// Formation is the default rectangular layout of a horde.
type Formation struct {
	Columns int         `json:"columns"`
	Spacing fixed.Fixed `json:"spacing"`
}

// Faction groups everything a player can build.
type Faction struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Color     string   `json:"color"`               // UI / team color, "#rrggbb"
	Hordes    []string `json:"hordes"`              // Horde.IDs the faction may train
	Start     []string `json:"start,omitempty"`     // starting army (Horde.IDs); empty = one of each in Hordes
	Fortress  string   `json:"fortress"`            // Building.ID (kind fortress), placed in the player's map slot
	Buildings []string `json:"buildings,omitempty"` // Building.IDs (kind building) the player can construct
	// FortressTunnel: the fortress is also a tunnel entrance (dwarves).
	FortressTunnel bool `json:"fortressTunnel,omitempty"`
}

// Building kinds.
const (
	BuildingFortress = "fortress" // a player's main building, placed by the map; losing it loses the game
	BuildingOutpost  = "outpost"  // neutral building placed by the map, captured by standing near it
	BuildingRegular  = "building" // constructed by players
	BuildingWall     = "wall"     // a wall segment of a fortress base (placed with it); infantry walks on top
	BuildingGate     = "gate"     // a wall segment its owner's units pass through
)

// Placements (every building is put up by workers).
const (
	PlacementFree  = "free"  // anywhere on open ground
	PlacementTower = "tower" // anywhere along the owner's wall (defensive towers)
	PlacementWall  = "wall"  // wall segments: dragged as a line from a start to an end point
	PlacementGate  = "gate"  // a gatehouse over a stretch of the owner's finished wall
)

// Building is a structure type: fortress, outpost or a constructible building.
type Building struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Kind string `json:"kind"` // BuildingFortress / BuildingOutpost / BuildingRegular

	Model      string      `json:"model"`                // static model id (frontend/public/models/<id>)
	ModelScale fixed.Fixed `json:"modelScale,omitempty"` // 0 = 1
	// FactionModels: a shared building drawn differently per owner faction (the keep): faction id →
	// model id; other factions use Model.
	FactionModels map[string]string `json:"factionModels,omitempty"`

	// Footprint: width (x, along the building's right) and depth (y, along its
	// front) in world units. Blocks movement.
	Footprint fixed.Vec2  `json:"footprint"`
	Health    fixed.Fixed `json:"health"`
	Armor     fixed.Fixed `json:"armor"`

	Material     string      `json:"material,omitempty"`     // wood / stone (default) / magic
	MagicDefense fixed.Fixed `json:"magicDefense,omitempty"` // flat reduction of magic damage

	// Regular buildings: where they may be built (PlacementFree / PlacementTower)
	// and how long construction takes.
	Placement []string    `json:"placement,omitempty"`
	BuildTime fixed.Fixed `json:"buildTime,omitempty"` // seconds
	Cost      Amounts     `json:"cost,omitempty"`      // paid when the foundation is laid, refunded on cancel

	// Economy effects (income, food, ...) and the hordes it trains.
	Economy  []Effect  `json:"economy,omitempty"`
	Trains   []string  `json:"trains,omitempty"`   // Horde.IDs
	Research []string  `json:"research,omitempty"` // Upgrade.IDs researched here
	Requires *Requires `json:"requires,omitempty"`

	// Fortresses: the base around the building (the wall, gates, stairs).
	Base *BaseLayout `json:"base,omitempty"`

	// NoBuildRadius keeps free buildings away: no footprint may reach into
	// this circle around the building (0 = only the footprint itself).
	NoBuildRadius fixed.Fixed `json:"noBuildRadius,omitempty"`

	// Capture: outposts are taken by standing next to them (nil = can't be captured).
	Capture *Capture `json:"capture,omitempty"`

	// Walls and gates: height of the walkway on top (units walk there).
	// Garrison buildings (watchtowers): height of the platform.
	WalkHeight fixed.Fixed `json:"walkHeight,omitempty"`
	// Garrison: one horde of foot soldiers can stand on top, like on a wall
	// (ranged bonus, out of reach of melee from the ground).
	Garrison bool `json:"garrison,omitempty"`
	// Gates: width of the passage in the middle (open: anyone walks through;
	// the rest of the gatehouse always blocks).
	Passage fixed.Fixed `json:"passage,omitempty"`

	// Attack: defensive buildings (towers) shoot like a ranged horde of Volley archers.
	Attack *Attack `json:"attack,omitempty"`
	Volley int     `json:"volley,omitempty"` // arrows per shot (default 1)
	// Toggles: toggle abilities (e.g. fire arrows) a tower switches; their
	// modifiers change its shots while on.
	Toggles []string `json:"toggles,omitempty"`

	// Sight: vision radius through the fog of war (0 = DefaultSight).
	Sight fixed.Fixed `json:"sight,omitempty"`
	// Tunnel: an entrance of the owner's tunnel network (dwarves): an own horde sent to a tunnel
	// enters the nearest entrance and comes out of this one, one horde at a time.
	Tunnel bool `json:"tunnel,omitempty"`
	// GrowTrees: plants trees around the building over time (moonwell).
	GrowTrees *GrowTrees `json:"growTrees,omitempty"`
}

// GrowTrees: one tree every Every seconds within Radius of the building, up to Max of them.
type GrowTrees struct {
	Radius fixed.Fixed `json:"radius"`
	Every  fixed.Fixed `json:"every"`
	Max    int         `json:"max"`
}

// Capture describes how a neutral outpost is taken: units of exactly one
// player within Radius of its footprint for Time seconds. A captured outpost
// can be attacked; when destroyed it turns neutral again.
type Capture struct {
	Time   fixed.Fixed `json:"time"`   // seconds
	Radius fixed.Fixed `json:"radius"` // from the footprint
}

// Scale returns ModelScale with the default applied.
func (b *Building) Scale() fixed.Fixed {
	if b.ModelScale == 0 {
		return fixed.One
	}
	return b.ModelScale
}

// Allows reports whether the building may be placed that way.
func (b *Building) Allows(placement string) bool {
	for _, p := range b.Placement {
		if p == placement {
			return true
		}
	}
	return false
}

// Map is a playable map.
type Map struct {
	ID       string       `json:"id"`
	Name     string       `json:"name"`
	Width    int          `json:"width"`  // world units, map is centered at (0,0)
	Height   int          `json:"height"` // world units
	Terrain  MapTerrain   `json:"terrain"`
	Players  []MapPlayer  `json:"players"`
	Outposts []MapOutpost `json:"outposts,omitempty"`
	Forests  []MapForest  `json:"forests,omitempty"`
}

// MapForest is a patch of trees: Count trees scattered in a circle of Radius (deterministic
// from Seed). Trees block movement, give cover, burn, regrow.
type MapForest struct {
	Pos    fixed.Vec2  `json:"pos"`
	Radius fixed.Fixed `json:"radius"`
	Count  int         `json:"count"`
	Seed   int64       `json:"seed,omitempty"`
}

// MapTerrain describes the heightfield.
type MapTerrain struct {
	CellSize  fixed.Fixed `json:"cellSize"`  // distance between height samples
	MaxHeight fixed.Fixed `json:"maxHeight"` // height of sample value 65535
	// Heightmap is a raw little-endian uint16 file next to map.json
	// ((width/cellSize+1) x (height/cellSize+1) samples, row-major).
	// Empty = generate procedurally from Seed.
	Heightmap string `json:"heightmap,omitempty"`
	Seed      int64  `json:"seed,omitempty"`
}

// MapPlayer is a start slot on a map.
type MapPlayer struct {
	Start    fixed.Vec2   `json:"start"`    // where the starting army stands; {"x": .., "y": ..}, map is centered at 0,0
	Fortress MapPlacement `json:"fortress"` // the faction's fortress building goes here
}

// MapOutpost is a neutral outpost on a map.
type MapOutpost struct {
	Building string `json:"building"` // Building.ID (kind outpost)
	MapPlacement
}

// MapPlacement positions a building on a map.
type MapPlacement struct {
	Pos fixed.Vec2 `json:"pos"`
	// Rotation in degrees (0 = front faces +x, counter-clockwise towards +y).
	// Omitted = front faces the map center.
	Rotation *fixed.Fixed `json:"rotation,omitempty"`
}

// Facing returns the building's facing angle in radians.
func (p MapPlacement) Facing() fixed.Fixed {
	if p.Rotation != nil {
		return fixed.NormalizeAngle(p.Rotation.Mul(fixed.Pi).DivInt(180))
	}
	if p.Pos.IsZero() {
		return 0
	}
	return p.Pos.Neg().Angle()
}
