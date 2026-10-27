package services

import (
	"fmt"
	"io/fs"
	"slices"
	"strings"
	"sync"

	"dnd-rts/internal/data"
	"dnd-rts/internal/fixed"
	"dnd-rts/internal/sim"
	"dnd-rts/internal/terrain"
)

// ContentService exposes loaded game content to the frontend.
type ContentService struct {
	fsys    fs.FS
	catalog *data.Catalog

	mu      sync.Mutex
	terrain map[string]*terrain.Heightfield // cache per map
}

// NewContentService loads and validates all content. Invalid content is a startup error.
func NewContentService(fsys fs.FS) (*ContentService, error) {
	catalog, err := data.Load(fsys)
	if err != nil {
		return nil, fmt.Errorf("content is invalid:\n%w", err)
	}
	return &ContentService{fsys: fsys, catalog: catalog, terrain: map[string]*terrain.Heightfield{}}, nil
}

// Catalog returns the loaded content (Go-side use only; package-level to stay unbound).
func Catalog(s *ContentService) *data.Catalog { return s.catalog }

// Heightfield returns the cached heightfield of a map (Go-side use only).
func Heightfield(s *ContentService, mapID string) (*terrain.Heightfield, error) {
	m, ok := s.catalog.Maps[mapID]
	if !ok {
		return nil, fmt.Errorf("unknown map %q", mapID)
	}
	return s.heightfield(m)
}

// HordeType describes a horde type for the client. The slice index is the
// `kind` sent in game snapshots.
type HordeType struct {
	ID     string   `json:"id"`
	Name   string   `json:"name"`
	Size   int      `json:"size"`
	Radius float64  `json:"radius"` // unit collision radius, used to scale the placeholder model
	Tags   []string `json:"tags"`
	Hero   bool     `json:"hero"`
	// Cost / Uses: resource id → amount (training price, cap usage such as food).
	Cost      map[string]int `json:"cost"`
	Uses      map[string]int `json:"uses"`
	BuildTime float64        `json:"buildTime"` // training time, seconds
	Model     string         `json:"model"`     // baked VAT model id (frontend/public/models/<id>), "" = placeholder
	Speed     float64        `json:"speed"`     // units per second, used to match walk/run playback speed

	// Abilities: ability ids in slot order (see Abilities()).
	Abilities []string     `json:"abilities"`
	Flying    bool         `json:"flying"`
	Requires  RequiresInfo `json:"requires"`
	// Mesh: static model of a machine (frontend/public/models/<id>, meshbake), "" = none.
	Mesh string `json:"mesh"`
	// Ranged: has a ranged attack (can shoot at a ground point).
	Ranged bool `json:"ranged"`
	// AntiAir: an attack can hit flying units. Climbs: foot soldiers that go up walls and watchtowers.
	AntiAir bool `json:"antiAir"`
	Climbs  bool `json:"climbs"`

	// Heroes: one per player (unique); revived in the castle at their level.
	Unique     bool           `json:"unique"`
	ReviveCost map[string]int `json:"reviveCost"`
	ReviveTime float64        `json:"reviveTime"` // seconds
	HeroXP     []int          `json:"heroXp"`     // total XP needed for level i+1 (index 0 = level 1)
}

// RequiresInfo: buildings (finished, own) and upgrades a player needs.
type RequiresInfo struct {
	Buildings []string `json:"buildings"`
	Upgrades  []string `json:"upgrades"`
}

func requiresInfo(r *data.Requires) RequiresInfo {
	if r == nil {
		return RequiresInfo{Buildings: []string{}, Upgrades: []string{}}
	}
	return RequiresInfo{Buildings: append([]string{}, r.Buildings...), Upgrades: append([]string{}, r.Upgrades...)}
}

// UpgradeInfo describes an upgrade for the HUD. The slice index is the
// upgrade kind used in snapshots and commands.
type UpgradeInfo struct {
	ID           string         `json:"id"`
	Name         string         `json:"name"`
	Cost         map[string]int `json:"cost"`
	ResearchTime float64        `json:"researchTime"` // seconds
	Group        string         `json:"group"`
	Requires     RequiresInfo   `json:"requires"`
	Description  string         `json:"description"` // generated from the modifiers
}

// Upgrades returns all upgrades sorted by ID (index = snapshot kind).
func (s *ContentService) Upgrades() []UpgradeInfo {
	out := make([]UpgradeInfo, 0, len(s.catalog.UpgradeIDs))
	for _, id := range s.catalog.UpgradeIDs {
		g := s.catalog.Upgrades[id]
		out = append(out, UpgradeInfo{ID: g.ID, Name: g.Name, Cost: g.Cost, ResearchTime: g.ResearchTime.Float(),
			Group: g.Group, Requires: requiresInfo(g.Requires), Description: describeUpgrade(s.catalog, g)})
	}
	return out
}

// describeUpgrade: "+15% damage · swordsman, knight".
func describeUpgrade(c *data.Catalog, g *data.Upgrade) string {
	m := g.Modifiers
	var parts []string
	pct := func(name string, v float64) {
		if v != 0 {
			parts = append(parts, fmt.Sprintf("%+.0f%% %s", v*100, name))
		}
	}
	flat := func(name string, v float64) {
		if v != 0 {
			parts = append(parts, fmt.Sprintf("%+g %s", v, name))
		}
	}
	pct("damage", m.Damage.Float())
	pct("attack speed", m.AttackSpeed.Float())
	pct("speed", m.MoveSpeed.Float())
	pct("range", m.Range.Float())
	flat("armor", m.Armor.Float())
	flat("magic defense", m.MagicDefense.Float())
	pct("damage taken", m.DamageTaken.Float())
	if m.ExtraType != "" {
		parts = append(parts, fmt.Sprintf("+%.0f%% %s damage", m.ExtraDamage.Float()*100, m.ExtraType))
	}
	// Abilities unlocked by this upgrade.
	for _, aid := range c.AbilityIDs {
		if a := c.Abilities[aid]; a.Requires != nil && slices.Contains(a.Requires.Upgrades, g.ID) {
			parts = append(parts, "unlocks "+a.Name)
		}
	}
	who := "all units"
	if len(g.Units) > 0 || len(g.Tags) > 0 {
		var names []string
		for _, u := range g.Units {
			names = append(names, c.Units[u].Name)
		}
		names = append(names, g.Tags...)
		who = strings.Join(names, ", ")
	}
	if len(parts) == 0 {
		return who
	}
	return strings.Join(parts, " · ") + " — " + who
}

// AbilityInfo describes an ability for the HUD. The slice index is the
// ability kind used in snapshots and commands.
type AbilityInfo struct {
	ID         string       `json:"id"`
	Name       string       `json:"name"`
	Kind       string       `json:"kind"`   // active / toggle / passive / aura
	Target     string       `json:"target"` // active: self / point / unit
	Affects    string       `json:"affects"`
	Range      float64      `json:"range"`
	Cooldown   float64      `json:"cooldown"` // seconds
	Autocast   bool         `json:"autocast"`
	Radius     float64      `json:"radius"`     // area of the effect / aura (visuals, targeting circle)
	DamageType string       `json:"damageType"` // of the first damage effect (visual color), "" = none
	Requires   RequiresInfo `json:"requires"`
	// Cone: width in degrees of a cone-shaped effect (aiming preview), 0 = circles only.
	Cone float64 `json:"cone"`
	// Description: what the special effects do (chain, summon, teleport, transform), for the tooltip.
	Description string `json:"description"`
	// Modes: the choices of a mode ability (an element), in order.
	Modes []ModeInfo `json:"modes"`
	// ByMode: effects or modifiers follow the unit's current mode.
	ByMode   bool `json:"byMode"`
	Ultimate bool `json:"ultimate"`
}

// ModeInfo is one choice of a mode ability.
type ModeInfo struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	AttackType string `json:"attackType"` // "" = unchanged (visual colour)
}

// Abilities returns all abilities sorted by ID (index = snapshot kind).
func (s *ContentService) Abilities() []AbilityInfo {
	out := make([]AbilityInfo, 0, len(s.catalog.AbilityIDs))
	for _, id := range s.catalog.AbilityIDs {
		a := s.catalog.Abilities[id]
		info := AbilityInfo{ID: a.ID, Name: a.Name, Kind: a.Kind, Target: a.Target, Affects: a.Affects,
			Range: a.Range.Float(), Cooldown: a.Cooldown.Float(), Autocast: a.Autocast, Radius: a.Radius.Float(),
			Requires: requiresInfo(a.Requires)}
		var walk func(effects []data.AbilityEffect)
		walk = func(effects []data.AbilityEffect) {
			for _, e := range effects {
				if e.Type != data.EffTeleport { // a teleport's radius is around the caster, not the target
					info.Radius = max(info.Radius, e.Radius.Float())
				}
				if e.Type == data.EffDamage && info.DamageType == "" {
					info.DamageType = e.DamageType
				}
				if e.Shape == data.ShapeCone {
					info.Cone = max(info.Cone, e.Angle.Float())
				}
				if d := s.describeEffect(&e); d != "" {
					if info.Description != "" {
						info.Description += ". "
					}
					info.Description += d
				}
				if e.Type != data.EffChain {
					walk(e.Effects)
				}
			}
		}
		walk(a.Effects)
		info.Ultimate = a.Ultimate
		info.Modes = []ModeInfo{}
		for _, m := range a.Modes {
			info.Modes = append(info.Modes, ModeInfo{ID: m.ID, Name: m.Name, AttackType: m.Modifiers.AttackType})
		}
		info.ByMode = len(a.ByMode) > 0 || hasModeEffects(a.Effects)
		if info.ByMode {
			if info.Description != "" {
				info.Description += ". "
			}
			info.Description += "Follows the chosen element"
		}
		out = append(out, info)
	}
	return out
}

func hasModeEffects(effects []data.AbilityEffect) bool {
	for _, e := range effects {
		if e.Mode != "" || hasModeEffects(e.Effects) {
			return true
		}
	}
	return false
}

// describeEffect: a short English line for effects the generic tooltip can't show.
func (s *ContentService) describeEffect(e *data.AbilityEffect) string {
	switch e.Type {
	case data.EffChain:
		return fmt.Sprintf("Chain: up to %d targets, jumps %g m, −%g%% per jump", e.Jumps+1, e.JumpRange.Float(), e.Falloff.Float()*100)
	case data.EffSummon:
		name := e.Horde
		if h, ok := s.catalog.Hordes[e.Horde]; ok {
			name = h.Name
		}
		d := fmt.Sprintf("Summons %d× %s", max(e.Count, 1), name)
		if e.Duration > 0 {
			d += fmt.Sprintf(" for %gs", e.Duration.Float())
		}
		if e.Limit {
			d += "; one at a time, cooldown starts when they die"
		}
		return d
	case data.EffTeleport:
		d := "Teleports the horde"
		if e.Radius > 0 {
			d += fmt.Sprintf(" and own hordes within %g m", e.Radius.Float())
		}
		if e.ToBuilding {
			d += " next to an own building"
		}
		return d
	case data.EffTransform:
		name := e.Unit
		if u, ok := s.catalog.Units[e.Unit]; ok {
			name = u.Name
		}
		return fmt.Sprintf("Turns into %s for %gs", name, e.Duration.Float())
	}
	if e.Shape == data.ShapeCone && e.Type == data.EffDamage {
		return fmt.Sprintf("Cone %g°, %g m", e.Angle.Float(), e.Radius.Float())
	}
	return ""
}

// HordeTypes returns all horde types sorted by ID (index = snapshot kind).
func (s *ContentService) HordeTypes() []HordeType {
	out := make([]HordeType, 0, len(s.catalog.HordeIDs))
	for _, id := range s.catalog.HordeIDs {
		h := s.catalog.Hordes[id]
		u := s.catalog.Units[h.Unit]
		t := HordeType{ID: h.ID, Name: h.Name, Size: h.Size, Radius: u.Radius.Float(), Tags: u.Tags, Hero: u.HasTag("hero"), Model: u.Model, Speed: u.Speed.Float(),
			Cost: h.Cost, Uses: h.Uses, BuildTime: h.BuildTime.Float(), Requires: requiresInfo(h.Requires)}
		t.Abilities = u.Abilities
		t.Flying = u.Flying
		t.Mesh = u.Mesh
		t.Ranged = u.Ranged != nil
		t.AntiAir = u.Melee.CanHit(true) || u.Ranged.CanHit(true)
		t.Climbs = u.HasTag("infantry") && !u.HasTag("cavalry") && !u.Flying && !u.HasTag("mechanical")
		t.Unique = h.Unique
		t.ReviveCost, t.ReviveTime = h.Cost, h.BuildTime.Float()*2
		if len(h.ReviveCost) > 0 {
			t.ReviveCost = h.ReviveCost
		}
		if h.ReviveTime > 0 {
			t.ReviveTime = h.ReviveTime.Float()
		}
		t.HeroXP = []int{}
		if u.Hero != nil {
			t.HeroXP = u.Hero.XP
		}
		out = append(out, t)
	}
	return out
}

// BuildingPlot is a build site relative to its building: x along the
// building's right, y along its front.
type BuildingPlot struct {
	X     float64 `json:"x"`
	Y     float64 `json:"y"`
	Size  float64 `json:"size"`
	Class string  `json:"class"` // s / m / l ("" = custom size)
}

// Point2 is a local offset (x along the building's right, y along its front).
type Point2 struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// BaseInfo is a fortress's base for drawing: wall contour, gates, tower sockets.
type BaseInfo struct {
	Radius  float64  `json:"radius"`
	Outline []Point2 `json:"outline"` // closed loop
	Gates   []Point2 `json:"gates"`
	Towers  []Point2 `json:"towers"`
}

// BuildingType describes a building type for the client. The slice index is
// the `kind` sent in game snapshots.
type BuildingType struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Kind       string   `json:"kind"`  // "fortress" | "outpost" | "building"
	Model      string   `json:"model"` // static model id (frontend/public/models/<id>)
	ModelScale float64  `json:"modelScale"`
	Width      float64  `json:"width"` // footprint along the building's right
	Depth      float64  `json:"depth"` // footprint along its front
	Placement  []string `json:"placement"`
	BuildTime  float64  `json:"buildTime"` // seconds (constructible buildings)
	// NoBuildRadius: free buildings may not reach into this circle (0 = none).
	NoBuildRadius float64        `json:"noBuildRadius"`
	Plots         []BuildingPlot `json:"plots"`
	// Cost: resource id → amount; Trains: horde ids it can train; Research: upgrade ids.
	Cost     map[string]int `json:"cost"`
	Trains   []string       `json:"trains"`
	Research []string       `json:"research"`
	Requires RequiresInfo   `json:"requires"`
	Base     *BaseInfo      `json:"base"` // fortresses with a generated base, else null
	// WalkHeight: walls and gates — height of the walkway (units on the wall are drawn there).
	WalkHeight float64 `json:"walkHeight"`
	// Toggles: toggle abilities of a tower (fire arrows), see Abilities().
	Toggles []string `json:"toggles"`
	// Attacks: the building shoots (a tower).
	Attacks bool `json:"attacks"`
	// Garrison: foot soldiers stand on top (watchtower), at WalkHeight.
	Garrison bool `json:"garrison"`
}

func points(vs []fixed.Vec2) []Point2 {
	out := make([]Point2, len(vs))
	for i, v := range vs {
		out[i] = Point2{X: v.X.Float(), Y: v.Y.Float()}
	}
	return out
}

// BuildingTypes returns all building types sorted by ID (index = snapshot kind).
func (s *ContentService) BuildingTypes() []BuildingType {
	out := make([]BuildingType, 0, len(s.catalog.BuildingIDs))
	for _, id := range s.catalog.BuildingIDs {
		b := s.catalog.Buildings[id]
		t := BuildingType{
			ID: b.ID, Name: b.Name, Kind: b.Kind, Model: b.Model, ModelScale: b.Scale().Float(),
			Width: b.Footprint.X.Float(), Depth: b.Footprint.Y.Float(),
			Placement: b.Placement, BuildTime: b.BuildTime.Float(), NoBuildRadius: b.NoBuildRadius.Float(), Plots: make([]BuildingPlot, len(b.Plots)),
		}
		t.Cost, t.Trains, t.Research, t.Requires = b.Cost, b.Trains, b.Research, requiresInfo(b.Requires)
		t.WalkHeight = b.WalkHeight.Float()
		t.Toggles = append([]string{}, b.Toggles...)
		t.Attacks = b.Attack != nil
		t.Garrison = b.Garrison
		for i, p := range b.Plots {
			t.Plots[i] = BuildingPlot{X: p.Offset.X.Float(), Y: p.Offset.Y.Float(), Size: p.Size.Float(), Class: p.Class}
		}
		if l := b.Base; l != nil {
			t.Base = &BaseInfo{Radius: l.Radius.Float(), Outline: points(l.Outline), Gates: points(l.GatePos), Towers: points(l.TowerSockets)}
		}
		out = append(out, t)
	}
	return out
}

// FactionInfo describes a faction for the client.
type FactionInfo struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Color     string   `json:"color"`
	Fortress  string   `json:"fortress"`
	Buildings []string `json:"buildings"` // building ids the faction can construct
	Hordes    []string `json:"hordes"`
}

// Factions returns all factions sorted by ID.
func (s *ContentService) Factions() []FactionInfo {
	out := make([]FactionInfo, 0, len(s.catalog.FactionIDs))
	for _, id := range s.catalog.FactionIDs {
		f := s.catalog.Factions[id]
		out = append(out, FactionInfo{ID: f.ID, Name: f.Name, Color: f.Color, Fortress: f.Fortress, Buildings: f.Buildings, Hordes: f.Hordes})
	}
	return out
}

// ResourceInfo describes a resource for the HUD. The slice is in sorted id
// order, matching the resources section of game snapshots.
type ResourceInfo struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Icon  string `json:"icon"`
	Kind  string `json:"kind"` // "stock" | "cap"
	Order int    `json:"order"`
}

// Resources returns all resources sorted by ID (= snapshot order).
func (s *ContentService) Resources() []ResourceInfo {
	out := make([]ResourceInfo, 0, len(s.catalog.ResourceIDs))
	for _, id := range s.catalog.ResourceIDs {
		r := s.catalog.Resources[id]
		out = append(out, ResourceInfo{ID: r.ID, Name: r.Name, Icon: r.Icon, Kind: r.Kind, Order: r.Order})
	}
	return out
}

// MapSummary is a map entry for menus.
type MapSummary struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Width   int    `json:"width"`
	Height  int    `json:"height"`
	Players int    `json:"players"`
}

// ListMaps returns all maps sorted by ID.
func (s *ContentService) ListMaps() []MapSummary {
	out := make([]MapSummary, 0, len(s.catalog.MapIDs))
	for _, id := range s.catalog.MapIDs {
		m := s.catalog.Maps[id]
		out = append(out, MapSummary{ID: m.ID, Name: m.Name, Width: m.Width, Height: m.Height, Players: len(m.Players)})
	}
	return out
}

// Point is a 2D world position for the frontend (x → world X, y → world Z).
type Point struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// TerrainData is everything the renderer needs to build the terrain mesh.
type TerrainData struct {
	MapID     string  `json:"mapId"`
	Width     int     `json:"width"`
	Height    int     `json:"height"`
	Cols      int     `json:"cols"`
	Rows      int     `json:"rows"`
	CellSize  float64 `json:"cellSize"`
	MaxHeight float64 `json:"maxHeight"`
	// Samples: little-endian uint16, row-major (base64 in JSON).
	Samples []byte  `json:"samples"`
	Starts  []Point `json:"starts"`
}

// Terrain returns the heightfield of a map (generated or loaded on first request).
func (s *ContentService) Terrain(mapID string) (*TerrainData, error) {
	m, ok := s.catalog.Maps[mapID]
	if !ok {
		return nil, fmt.Errorf("unknown map %q", mapID)
	}
	h, err := s.heightfield(m)
	if err != nil {
		return nil, err
	}

	starts := make([]Point, len(m.Players))
	for i, p := range m.Players {
		starts[i] = Point{X: p.Start.X.Float(), Y: p.Start.Y.Float()}
	}
	return &TerrainData{
		MapID:     m.ID,
		Width:     h.Width,
		Height:    h.Height,
		Cols:      h.Cols,
		Rows:      h.Rows,
		CellSize:  h.CellSize.Float(),
		MaxHeight: h.MaxHeight.Float(),
		Samples:   h.R16(),
		Starts:    starts,
	}, nil
}

func (s *ContentService) heightfield(m *data.Map) (*terrain.Heightfield, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if h, ok := s.terrain[m.ID]; ok {
		return h, nil
	}
	h, err := terrain.ForMap(s.fsys, m)
	if err != nil {
		return nil, err
	}
	s.terrain[m.ID] = h
	return h, nil
}

// NavGridData is the walkability grid, for the debug overlay.
type NavGridData struct {
	Cols     int     `json:"cols"`
	Rows     int     `json:"rows"`
	CellSize float64 `json:"cellSize"`
	// Blocked: one byte per cell (1 = blocked), row-major (base64 in JSON).
	Blocked []byte `json:"blocked"`
}

// NavGrid returns the walkability grid the simulation uses for a map.
func (s *ContentService) NavGrid(mapID string) (*NavGridData, error) {
	h, err := Heightfield(s, mapID)
	if err != nil {
		return nil, err
	}
	g := sim.BuildNavGrid(h)
	blocked := make([]byte, len(g.Blocked))
	for i, b := range g.Blocked {
		if b {
			blocked[i] = 1
		}
	}
	return &NavGridData{Cols: g.Cols, Rows: g.Rows, CellSize: g.CellSize.Float(), Blocked: blocked}, nil
}
