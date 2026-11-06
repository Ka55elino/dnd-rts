package sim

import (
	"slices"

	"dnd-rts/internal/data"
	"dnd-rts/internal/fixed"
)

// TagWorker marks units that construct free buildings.
const TagWorker = "worker"

var (
	// buildReach: a worker horde builds when its anchor is this close to the footprint (a cell and
	// a half: next to a wall the nearest open cell can be a little over one cell away).
	buildReach = NavCellSize.MulInt(3).DivInt(2) + fixed.One
	// buildApproach: where workers head, measured from the footprint.
	buildApproach = fixed.FromInt(2)
	// swingReach: a worker soldier steps this close to the wall to hammer it.
	swingReach = fixed.FromFraction(8, 10)
)

// IsWorker reports whether the horde's units can build.
func (h *Horde) IsWorker() bool { return h.Unit.HasTag(TagWorker) }

// ConstructTarget returns the building the horde is constructing, or nil.
func (h *Horde) ConstructTarget() *Building { return h.construct }

// constructOrder sends a worker horde to build (or help build) b.
func (h *Horde) constructOrder(w *World, b *Building) {
	h.clearOrders()
	h.construct = b
	h.ordered = false
	if p, ok := w.walkableTarget(b.approach(h.Pos, buildApproach)); ok {
		h.setDestination(p)
	}
}

// updateConstruct walks to the foundation and works on it while in reach.
func (h *Horde) updateConstruct(w *World) {
	b := h.construct
	if !b.Alive() || !(b.UnderConstruction() || b.Damaged()) {
		h.construct = nil // built, or repaired to full health
		h.constructing = false
		return
	}
	if b.DistTo(h.Pos) <= buildReach {
		h.constructing = true
		h.moving = false
		if to := b.Closest(h.Pos).Sub(h.Pos); !to.IsZero() {
			h.Facing = turnTowards(h.Facing, to.Angle(), turnRate.Mul(TickDt))
		}
		b.builders++
		return
	}
	h.constructing = false
	if !h.moving {
		if p, ok := w.walkableTarget(b.approach(h.Pos, buildApproach)); ok && p != h.Pos {
			h.setDestination(p)
		}
	}
}

// hammer: a worker soldier steps up to the wall and swings (the animation
// only; progress is counted per horde in updateConstruct).
func (s *Soldier) hammer(b *Building) {
	u := s.Horde.Unit
	c := b.Closest(s.Pos)
	to := c.Sub(s.Pos)
	dist := to.Len()
	if !to.IsZero() {
		s.Rot = turnTowards(s.Rot, to.Angle(), soldierTurnRate.Mul(TickDt).MulInt(2))
	}
	if dist > u.Radius+swingReach && s.Pos.Dist(s.Horde.slotPos(s.Slot)) <= meleeLeash {
		step := u.Speed.Mul(TickDt)
		s.Pos, _ = s.Pos.MoveTowards(c.Sub(to.Scale((u.Radius + swingReach).Div(dist))), step)
		return
	}
	if s.cooldown == 0 {
		s.cooldown = primaryCooldown(u)
		s.swing = swingTicks
	}
}

// buildFree lays a foundation for a free building and sends the workers to it.
func (w *World) buildFree(c Command) {
	def, ok := w.Catalog.Buildings[c.Name]
	if !ok {
		return
	}
	var workers []*Horde
	for _, id := range c.Hordes {
		if h := w.Horde(id); h != nil && h.Owner == c.Player && h.IsWorker() {
			workers = append(workers, h)
		}
	}
	if len(workers) == 0 {
		return
	}
	var b *Building
	if def.Allows(data.PlacementGate) {
		w.buildGate(c.Player, def, c.Target, workers) // over the own wall where the player clicked
		return
	}
	if def.Allows(data.PlacementWall) {
		return // walls are lines (CmdBuildWall)
	}
	if def.Allows(data.PlacementTower) {
		// A defensive tower: on the own wall, wherever the player clicked along it.
		spot, facing, ok := w.TowerSpot(c.Player, def, c.Target)
		if !ok || !w.pay(c.Player, def.Cost) {
			return
		}
		b = w.placeBuilding(c.Player, def.ID, spot, facing) // on the wall: no open ground under it
		b.onWall = true
	} else {
		if !w.CanPlaceFree(c.Player, def, c.Target, c.Angle) || !w.pay(c.Player, def.Cost) {
			return
		}
		var err error
		if b, err = w.SpawnBuilding(c.Player, def.ID, c.Target, c.Angle); err != nil {
			return
		}
	}
	b.startConstruction()
	for _, h := range workers {
		h.constructOrder(w, b)
	}
}

// CanPlaceFree reports whether player may put a free building of type def
// at pos facing `facing`: allowed for the faction, on open ground inside the
// map, clear of other buildings, trees and the feet of wall stairs, outside every
// building's no-build circle, and with no building inside its own no-build circle.
func (w *World) CanPlaceFree(player PlayerID, def *data.Building, pos fixed.Vec2, facing fixed.Fixed) bool {
	if !def.Allows(data.PlacementFree) || !w.canBuild(player, def.ID) || !w.Meets(player, def.Requires) {
		return false
	}
	return w.fitsFree(def, pos, facing, true)
}

// fitsFree: the footprint fits on open ground (no terrain block, tree or building under it);
// with noBuild, the no-build circles count too.
func (w *World) fitsFree(def *data.Building, pos fixed.Vec2, facing fixed.Fixed, noBuild bool) bool {
	ghost := rect{center: pos, facing: facing, half: def.Footprint.Scale(fixed.FromFraction(1, 2))}
	// Map bounds (keep a cell of margin) and terrain: every nav cell whose
	// center lies under the footprint must be walkable.
	hw := fixed.FromInt(w.Terrain.Width).DivInt(2) - NavCellSize
	hh := fixed.FromInt(w.Terrain.Height).DivInt(2) - NavCellSize
	r := ghost.half.X + ghost.half.Y
	if pos.X-r < -hw || pos.X+r > hw || pos.Y-r < -hh || pos.Y+r > hh {
		return false
	}
	g := w.Nav.Grid()
	tmp := &Building{Def: def, Pos: pos, Facing: facing}
	c0, r0 := g.Cell(pos.Sub(fixed.Vec2{X: r, Y: r}))
	c1, r1 := g.Cell(pos.Add(fixed.Vec2{X: r, Y: r}))
	for row := r0; row <= r1; row++ {
		for col := c0; col <= c1; col++ {
			i := g.Index(col, row)
			if (tmp.Covers(g.Center(i), 0) || tmp.Covers(g.Center(i), NavCellSize.DivInt(2))) && w.navTerrain[i] {
				return false
			}
		}
	}
	for i := range w.Trees {
		if t := &w.Trees[i]; t.Grown && tmp.Covers(t.Pos, fixed.One) {
			return false
		}
	}
	// The feet of wall stairs stay clear (units go up there).
	for _, r := range w.Walls {
		for _, st := range r.Stairs {
			if r == w.placeIgnoreStair.ring && st.node == w.placeIgnoreStair.node {
				continue
			}
			if !r.tower && tmp.DistTo(st.ground) < stairClear {
				return false
			}
		}
	}
	// Other buildings (exact rectangle overlap), no-build circles both ways.
	for _, b := range w.Buildings {
		if slices.Contains(w.placeIgnore, b) {
			continue // the wall end a new line joins
		}
		if r := b.Def.NoBuildRadius; noBuild && r > 0 && tmp.DistTo(b.Pos) < r {
			return false // reaches into b's circle
		}
		if r := def.NoBuildRadius; noBuild && r > 0 && b.DistTo(pos) < r {
			return false // b would stand inside the new building's circle
		}
		if ghost.overlaps(rect{center: b.Pos, facing: b.Facing, half: b.Def.Footprint.Scale(fixed.FromFraction(1, 2))}) {
			return false
		}
	}
	return true
}

// stairClear: free buildings keep this far from the foot of a wall stair.
var stairClear = fixed.FromInt(3)

// towerReach: a click this close to the centre line of an own wall segment puts a tower on it.
var towerReach = fixed.FromInt(8)

// TowerSpot is where a defensive tower of type def goes when player clicks at pos: on the centre
// line of the nearest own (finished, standing) wall segment, facing out, clear of gates, stairs and
// other towers. ok is false when there is no such place (or the player can't build it).
func (w *World) TowerSpot(player PlayerID, def *data.Building, pos fixed.Vec2) (spot fixed.Vec2, facing fixed.Fixed, ok bool) {
	if !def.Allows(data.PlacementTower) || !w.canBuild(player, def.ID) || !w.Meets(player, def.Requires) {
		return spot, 0, false
	}
	best := towerReach
	for _, s := range w.Buildings {
		if s.Owner != player || s.Def.Kind != data.BuildingWall || !s.Alive() || s.UnderConstruction() {
			continue
		}
		right, _ := s.axes()
		half := s.Def.Footprint.X.DivInt(2)
		along := pos.Sub(s.Pos).Dot(right).Clamp(-half, half)
		p := s.Pos.Add(right.Scale(along))
		if d := p.Dist(pos); d <= best {
			best, spot, facing, ok = d, p, s.Facing, true
		}
	}
	if !ok {
		return spot, 0, false
	}
	size := max(def.Footprint.X, def.Footprint.Y)
	for _, b := range w.Buildings {
		if !b.Alive() {
			continue
		}
		switch {
		case b.Def.Kind == data.BuildingGate:
			if b.DistTo(spot) < size/2+fixed.FromInt(2) {
				return spot, 0, false // keep the gatehouse clear
			}
		case b.onWall:
			if b.DistTo(spot) < size/2+fixed.One {
				return spot, 0, false // another tower here
			}
		}
	}
	for _, r := range w.Walls {
		for _, st := range r.Stairs {
			if !r.tower && st.ground.Dist(spot) < size/2+fixed.FromInt(5) {
				return spot, 0, false // a stair comes up here
			}
		}
	}
	return spot, facing, true
}

// rect is an oriented rectangle: half.X along its right, half.Y along its front.
type rect struct {
	center fixed.Vec2
	facing fixed.Fixed
	half   fixed.Vec2
}

func (r rect) axes() (right, forward fixed.Vec2) {
	forward = fixed.FromAngle(r.facing)
	return fixed.Vec2{X: forward.Y, Y: -forward.X}, forward
}

// overlaps: separating axis test on the four edge normals.
func (r rect) overlaps(o rect) bool {
	ar, af := r.axes()
	br, bf := o.axes()
	d := o.center.Sub(r.center)
	for _, axis := range []fixed.Vec2{ar, af, br, bf} {
		ra := r.half.X.Mul(ar.Dot(axis).Abs()) + r.half.Y.Mul(af.Dot(axis).Abs())
		rb := o.half.X.Mul(br.Dot(axis).Abs()) + o.half.Y.Mul(bf.Dot(axis).Abs())
		if d.Dot(axis).Abs() >= ra+rb {
			return false
		}
	}
	return true
}

var (
	// repairRate: health per tick restored by one worker horde, as a fraction of the maximum.
	repairRate = fixed.FromFraction(1, 1000) // ~2% a second
	// machineRepairReach: idle workers fix own machines this close.
	machineRepairReach = fixed.FromInt(8)
)

// Damaged: a finished building below full health (workers can repair it).
func (b *Building) Damaged() bool {
	return b.Alive() && !b.UnderConstruction() && b.HP < b.Def.Health
}

// updateRepairs: workers at damaged finished buildings restore their health.
func (w *World) updateRepairs() {
	for _, b := range w.Buildings {
		if b.builders > 0 && b.Damaged() {
			b.HP = min(b.HP+b.Def.Health.Mul(repairRate).Mul(workRate(b.builders)), b.Def.Health)
		}
	}
}

// repairMachines: an idle worker horde fixes the most damaged own machine nearby.
func (w *World) repairMachines(h *Horde) {
	if !h.IsWorker() || h.construct != nil || h.moving || h.attack != nil || !h.Alive() {
		return
	}
	var best *Soldier
	w.forEachSoldierNear(h.Pos, machineRepairReach, func(s *Soldier) {
		if s.Horde.Owner != h.Owner || !s.Alive() || !s.Horde.Unit.HasTag("mechanical") || s.HP >= s.MaxHP() {
			return
		}
		if s.Pos.Dist(h.Pos) <= machineRepairReach && (best == nil || s.HP.Div(s.MaxHP()) < best.HP.Div(best.MaxHP()) ||
			(s.HP.Div(s.MaxHP()) == best.HP.Div(best.MaxHP()) && s.ID < best.ID)) {
			best = s
		}
	})
	if best != nil {
		best.HP = min(best.HP+best.MaxHP().Mul(repairRate).MulInt(2), best.MaxHP())
	}
}
