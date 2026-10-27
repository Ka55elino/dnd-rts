package sim

import (
	"fmt"
	"slices"

	"dnd-rts/internal/data"
	"dnd-rts/internal/fixed"
	"dnd-rts/internal/sim/nav"
	"dnd-rts/internal/terrain"
)

// TickRate is the number of simulation steps per second.
const TickRate = 20

// TickDt is the duration of one tick in seconds.
var TickDt = fixed.FromFraction(1, TickRate)

var (
	// NavCellSize is the walkability grid resolution (world units).
	NavCellSize = fixed.FromInt(4)
	// MaxWalkableSlope: terrain steeper than this (rise per unit, ~24°) blocks movement.
	MaxWalkableSlope = fixed.FromFraction(45, 100)
)

type (
	PlayerID  uint8
	HordeID   uint32
	SoldierID uint32
)

// World is the complete, deterministic game state.
//
// Rules: no floats, no time.Now, no global rand, no map iteration.
// Hordes, Soldiers, Projectiles and Buildings are kept sorted by ID; IDs only grow.
type World struct {
	Tick        uint32
	Catalog     *data.Catalog
	Terrain     *terrain.Heightfield
	Nav         *nav.Cache // gates closed for everyone; each player paths on navFor(player)
	Walls       []*WallRing
	Hordes      []*Horde
	Soldiers    []*Soldier
	Projectiles []*Projectile // volleys in flight, sorted by ID
	Buildings   []*Building   // sorted by ID
	Zones       []*Zone       // lingering ability areas, sorted by ID
	Events      []CastEvent   // this tick's casts (for the client; not game state)

	nextHorde      HordeID
	nextSoldier    SoldierID
	nextProjectile ProjectileID
	nextBuilding   BuildingID
	nextZone       uint32

	// navTerrain is the walkability from terrain alone; buildings are
	// stamped on top of it (see rebuildBlocked).
	navTerrain []bool

	// pierceOff is set while a volley from a "no pierce" horde lands (fire mode); scratch.
	pierceOff bool

	// Match state.
	Players  []PlayerID // in the match, sorted
	factions []string   // faction id per entry of Players
	econ     []*economy // per entry of Players
	Defeated []PlayerID // lost their fortress, in order of defeat
	Winner   PlayerID   // Neutral until decided
	Over     bool       // the match has ended (Winner may stay Neutral: draw)

	// Scratch spatial hashes (not game state).
	sepHash    *spatialHash
	combatHash *spatialHash
}

// NewWorld creates an empty world for a map.
func NewWorld(catalog *data.Catalog, hf *terrain.Heightfield) *World {
	grid := BuildNavGrid(hf)
	w := &World{
		Catalog:        catalog,
		Terrain:        hf,
		Nav:            nav.NewCache(grid, 64),
		nextHorde:      1,
		nextSoldier:    1,
		nextProjectile: 1,
		nextBuilding:   1,
		nextZone:       1,
		Winner:         Neutral,
	}
	w.navTerrain = slices.Clone(grid.Blocked)
	w.sepHash = newSpatialHash(w, separationCell)
	w.combatHash = newSpatialHash(w, combatCell)
	return w
}

// Step applies the commands for this tick, then advances the world by one tick.
// Commands are applied in the given order; callers must make that order
// deterministic (e.g. sorted by player, then arrival).
func (w *World) Step(cmds []Command) {
	w.Events = w.Events[:0]
	w.combatHash.build(w)
	w.updateMods()
	for _, c := range cmds {
		w.apply(c)
	}
	for _, b := range w.Buildings {
		b.builders = 0 // counted again by worker hordes below
	}
	for _, h := range w.Hordes {
		h.update(w)
	}
	w.combatHash.build(w)
	for _, s := range w.Soldiers {
		s.update(w)
	}
	w.updateSpearWalls()
	w.updateTowers()
	w.updateProjectiles()
	w.updateCapture()
	w.updateEffects()
	w.updateZones()
	w.separate()
	w.updateGates()
	w.updateConstruction()
	w.updateRepairs()
	w.updateEconomy()
	w.updateTraining()
	w.expireSummons()
	w.removeDead()
	w.removeDestroyedBuildings()
	w.Tick++
}

// BuildNavGrid builds the walkability grid the simulation uses for a heightfield.
func BuildNavGrid(hf *terrain.Heightfield) *nav.Grid {
	return nav.FromTerrain(hf, NavCellSize, MaxWalkableSlope)
}

// primaryCooldown is the primary attack's cooldown in ticks (1 s without attacks).
func primaryCooldown(u *data.Unit) int {
	if a := u.Primary(); a != nil {
		return cooldownTicks(a.Cooldown)
	}
	return TickRate
}

// SpawnHorde creates a horde of the given type at pos, facing angle (radians).
func (w *World) SpawnHorde(owner PlayerID, hordeType string, pos fixed.Vec2, facing fixed.Fixed) (*Horde, error) {
	def, ok := w.Catalog.Hordes[hordeType]
	if !ok {
		return nil, fmt.Errorf("sim: unknown horde type %q", hordeType)
	}
	unit := w.Catalog.Units[def.Unit]
	kind := slices.Index(w.Catalog.HordeIDs, hordeType)

	h := &Horde{
		ID:     w.nextHorde,
		Owner:  owner,
		Kind:   kind,
		Def:    def,
		Unit:   unit,
		base:   unit,
		Pos:    pos,
		Facing: fixed.NormalizeAngle(facing),
	}
	w.nextHorde++

	cd := primaryCooldown(unit)
	h.volleyCooldown = int(h.ID) % cd // stagger volleys between hordes
	for i := 0; i < def.Size; i++ {
		s := &Soldier{
			ID:       w.nextSoldier,
			Horde:    h,
			Slot:     i,
			Rot:      h.Facing,
			HP:       unit.Health,
			momentum: fixed.One,
			// Stagger first attacks so a whole line doesn't swing on the same tick.
			cooldown: int(w.nextSoldier) % cd,
		}
		w.nextSoldier++
		h.Soldiers = append(h.Soldiers, s)
		w.Soldiers = append(w.Soldiers, s)
	}
	for i, s := range h.Soldiers {
		s.Pos = h.slotPos(i) // after all are added: slots depend on horde size
	}
	h.initAbilities(w.Catalog)
	w.Hordes = append(w.Hordes, h)
	if h.IsHero() {
		h.level = 1
	}
	w.registerUnique(h)
	return h, nil
}

// Horde returns a horde by ID (binary search; hordes are sorted by ID).
func (w *World) Horde(id HordeID) *Horde {
	i, ok := slices.BinarySearchFunc(w.Hordes, id, func(h *Horde, id HordeID) int {
		return int(h.ID) - int(id)
	})
	if !ok {
		return nil
	}
	return w.Hordes[i]
}

func (w *World) apply(c Command) {
	switch c.Kind {
	case CmdMove:
		for _, id := range c.Hordes {
			h := w.Horde(id)
			if h == nil || h.Owner != c.Player {
				continue // unknown or not yours: ignore (never trust the client)
			}
			// Onto an own wall (foot soldiers), along it, or down from it.
			if r, s, ok := w.wallAt(c.Player, c.Target); ok && h.canClimb() {
				w.orderOntoWall(h, r, s)
				continue
			}
			if h.wall != nil {
				w.orderOffWall(h, c.Target)
				continue
			}
			if target, ok := w.targetFor(h, c.Target); ok {
				h.moveTo(target)
			}
		}
	case CmdAttack:
		enemy := w.Horde(c.TargetHorde)
		if enemy == nil || enemy.Owner == c.Player {
			return
		}
		for _, id := range c.Hordes {
			h := w.Horde(id)
			if h == nil || h.Owner != c.Player {
				continue
			}
			if h.Unit.Charge != nil {
				dir := enemy.Pos.Sub(h.Pos).Normalize()
				if point, ok := w.walkableTarget(enemy.Pos.Add(dir.Scale(chargeOvershoot))); ok {
					h.charge(enemy, point)
					continue
				}
			}
			h.attackHorde(enemy)
		}
	case CmdCast, CmdToggle, CmdAutocast:
		if c.Kind == CmdToggle && c.TargetBuilding != 0 {
			w.toggleBuilding(c.Player, w.Building(c.TargetBuilding), c.Name) // a tower's fire arrows
			return
		}
		target := w.Horde(c.TargetHorde)
		for _, id := range c.Hordes {
			h := w.Horde(id)
			if h == nil || h.Owner != c.Player {
				continue
			}
			switch c.Kind {
			case CmdCast:
				w.orderCast(h, c.Name, c.Target, target)
			case CmdToggle:
				w.toggleAbility(h, c.Name, c.Count)
			case CmdAutocast:
				h.setAutocast(c.Name, c.Count != 0)
			}
		}
	case CmdAttackBuilding:
		b := w.Building(c.TargetBuilding)
		if b == nil || !b.Attackable(c.Player) {
			return
		}
		for _, id := range c.Hordes {
			if h := w.Horde(id); h != nil && h.Owner == c.Player {
				h.attackBuilding(w, b)
			}
		}
	case CmdBuildOnPlot:
		if host := w.Building(c.TargetBuilding); host != nil {
			w.buildOnPlot(c.Player, host, c.Count, c.Name)
		}
	case CmdBuildFree:
		w.buildFree(c)
	case CmdConstruct:
		b := w.Building(c.TargetBuilding)
		// Help build a foundation, or repair a damaged building (walls too).
		if b == nil || b.Owner != c.Player || !(b.UnderConstruction() && !b.selfBuild || b.Damaged()) {
			return
		}
		for _, id := range c.Hordes {
			if h := w.Horde(id); h != nil && h.Owner == c.Player && h.IsWorker() {
				h.constructOrder(w, b)
			}
		}
	case CmdStop, CmdHold, CmdAttackMove, CmdPatrol:
		for _, id := range c.Hordes {
			h := w.Horde(id)
			if h == nil || h.Owner != c.Player {
				continue
			}
			target, ok := c.Target, true
			if (c.Kind == CmdAttackMove || c.Kind == CmdPatrol) && h.wall != nil {
				w.orderOffWall(h, c.Target) // down first (attack-move / patrol are ground orders)
				continue
			}
			if c.Kind == CmdAttackMove || c.Kind == CmdPatrol {
				if target, ok = w.targetFor(h, c.Target); !ok {
					continue
				}
			}
			switch c.Kind {
			case CmdStop:
				h.stopOrder()
			case CmdHold:
				h.holdOrder()
			case CmdAttackMove:
				h.attackMove(target)
			case CmdPatrol:
				h.patrolTo(target)
			}
		}
	case CmdAttackGround:
		for _, id := range c.Hordes {
			if h := w.Horde(id); h != nil && h.Owner == c.Player {
				h.orderAttackGround(w, w.clampToMap(c.Target))
			}
		}
	case CmdGate:
		w.setGate(c.Player, w.Building(c.TargetBuilding), c.Count != 0)
	case CmdTrain:
		if b := w.Building(c.TargetBuilding); b != nil {
			w.train(c.Player, b, c.Name)
		}
	case CmdResearch:
		if b := w.Building(c.TargetBuilding); b != nil {
			w.research(c.Player, b, c.Name)
		}
	case CmdCancelTrain:
		if b := w.Building(c.TargetBuilding); b != nil {
			w.cancelTraining(c.Player, b, c.Count)
		}
	case CmdRally:
		if b := w.Building(c.TargetBuilding); b != nil && b.Owner == c.Player {
			b.rally, b.hasRally = c.Target, true
		}
	case CmdCancelBuild:
		if b := w.Building(c.TargetBuilding); b != nil {
			w.cancelConstruction(c.Player, b)
		}
	case CmdDevSpawnHorde, CmdDevSpawnBuilding, CmdDevKill, CmdDevHeal, CmdDevGive, CmdDevHit, CmdDevXP:
		w.applyDev(c)
	}
}

// FactionOf returns the player's faction, or nil.
func (w *World) FactionOf(p PlayerID) *data.Faction { return w.faction(p) }

// IsDefeated reports whether the player lost their fortress.
func (w *World) IsDefeated(p PlayerID) bool { return slices.Contains(w.Defeated, p) }

// faction returns the player's faction, or nil.
func (w *World) faction(p PlayerID) *data.Faction {
	i := slices.Index(w.Players, p)
	if i < 0 {
		return nil
	}
	return w.Catalog.Factions[w.factions[i]]
}

// defeat marks a player as beaten and decides the match when one player is left.
func (w *World) defeat(p PlayerID) {
	if p == Neutral || slices.Contains(w.Defeated, p) || !slices.Contains(w.Players, p) {
		return
	}
	w.Defeated = append(w.Defeated, p)
	var left []PlayerID
	for _, q := range w.Players {
		if !slices.Contains(w.Defeated, q) {
			left = append(left, q)
		}
	}
	if len(left) <= 1 && !w.Over {
		w.Over = true
		if len(left) == 1 {
			w.Winner = left[0]
		}
	}
}

// walkableTarget clamps p to the map and, if it lands on blocked ground,
// moves it to the center of the nearest walkable cell.
// clampToMap keeps a point a cell inside the map edges.
func (w *World) clampToMap(p fixed.Vec2) fixed.Vec2 {
	hw := fixed.FromInt(w.Terrain.Width).DivInt(2) - NavCellSize
	hh := fixed.FromInt(w.Terrain.Height).DivInt(2) - NavCellSize
	return fixed.Vec2{X: p.X.Clamp(-hw, hw), Y: p.Y.Clamp(-hh, hh)}
}

// targetFor is where horde h can go towards p: anywhere on the map for
// flyers, the nearest walkable point for everyone else.
func (w *World) targetFor(h *Horde, p fixed.Vec2) (fixed.Vec2, bool) {
	if h.Unit.Flying {
		return w.clampToMap(p), true
	}
	return walkableOn(w.navFor(h.Owner).Grid(), w.clampToMap(p))
}

func (w *World) walkableTarget(p fixed.Vec2) (fixed.Vec2, bool) {
	return walkableOn(w.Nav.Grid(), w.clampToMap(p))
}

func walkableOn(g *nav.Grid, p fixed.Vec2) (fixed.Vec2, bool) {
	if g.WalkableAt(p) {
		return p, true
	}
	i := g.NearestWalkable(g.IndexAt(p))
	if i < 0 {
		return p, false
	}
	return g.Center(i), true
}
