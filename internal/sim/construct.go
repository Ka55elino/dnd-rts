package sim

import (
	"dnd-rts/internal/data"
	"dnd-rts/internal/fixed"
)

// TagWorker marks units that construct free buildings.
const TagWorker = "worker"

var (
	// buildReach: a worker horde builds when its anchor is this close to the footprint.
	buildReach = NavCellSize + fixed.One
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
	if len(workers) == 0 || !w.CanPlaceFree(c.Player, def, c.Target, c.Angle) || !w.pay(c.Player, def.Cost) {
		return
	}
	b, err := w.SpawnBuilding(c.Player, def.ID, c.Target, c.Angle)
	if err != nil {
		return
	}
	b.startConstruction()
	for _, h := range workers {
		h.constructOrder(w, b)
	}
}

// CanPlaceFree reports whether player may put a free building of type def
// at pos facing `facing`: allowed for the faction, on open ground inside the
// map, clear of other buildings and of every plot, outside every building's
// no-build circle, and with no building inside its own no-build circle.
func (w *World) CanPlaceFree(player PlayerID, def *data.Building, pos fixed.Vec2, facing fixed.Fixed) bool {
	if !def.Allows(data.PlacementFree) || !w.canBuild(player, def.ID) || !w.Meets(player, def.Requires) {
		return false
	}
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
	// Other buildings and plots (exact rectangle overlap), no-build circles both ways.
	for _, b := range w.Buildings {
		if r := b.Def.NoBuildRadius; r > 0 && tmp.DistTo(b.Pos) < r {
			return false // reaches into b's circle
		}
		if r := def.NoBuildRadius; r > 0 && b.DistTo(pos) < r {
			return false // b would stand inside the new building's circle
		}
		if ghost.overlaps(rect{center: b.Pos, facing: b.Facing, half: b.Def.Footprint.Scale(fixed.FromFraction(1, 2))}) {
			return false
		}
		for i, p := range b.Def.Plots {
			half := p.Size.DivInt(2)
			if ghost.overlaps(rect{center: b.PlotPos(i), facing: b.Facing, half: fixed.Vec2{X: half, Y: half}}) {
				return false
			}
		}
	}
	return true
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
