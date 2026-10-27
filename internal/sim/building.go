package sim

import (
	"fmt"
	"slices"

	"dnd-rts/internal/data"
	"dnd-rts/internal/fixed"
)

// BuildingID identifies a building; IDs only grow.
type BuildingID uint32

// Neutral owns buildings no player controls (uncaptured outposts).
const Neutral PlayerID = 255

// Building is a structure on the map: a fortress, an outpost or a
// constructed building. It blocks movement under its footprint.
type Building struct {
	ID     BuildingID
	Owner  PlayerID
	Kind   int // index into Catalog.BuildingIDs (snapshot kind)
	Def    *data.Building
	Pos    fixed.Vec2
	Facing fixed.Fixed // radians; the front faces (cos, sin)
	HP     fixed.Fixed

	// Construction: buildTicks > 0 while unfinished; work counts up to it
	// (one per tick for plot buildings; for free buildings it depends on
	// how many workers are building).
	buildTicks int
	work       fixed.Fixed
	selfBuild  bool        // plot buildings construct themselves
	builders   int         // workers building it this tick (scratch, recomputed every tick)
	workSpeed  fixed.Fixed // + fraction from auras (scratch, recomputed every tick)
	workAcc    fixed.Fixed // training progress carried between ticks
	atkCd      int         // towers: ticks until the next shot
	gateOpen   bool        // gates: units pass through the passage
	gateWant   bool        // gates: what the owner asked for (closing waits for a clear passage)
	toggled    []bool      // towers: Def.Toggles switched on

	// Economy: income accumulators per effect, training queue, rally point.
	acc        []fixed.Fixed
	lastIncome uint32
	queue      []queued
	rally      fixed.Vec2
	hasRally   bool

	// Burning (fire damage over time).
	burnTicks  int
	burnDamage fixed.Fixed
	burnOwner  PlayerID

	// Outposts: who is capturing (Neutral = nobody) and for how long.
	captureBy    PlayerID
	captureTicks int

	// Fortresses / outposts: the building standing on each plot (nil = free).
	plotUsers []*Building
	// Plot buildings: the fortress / outpost and plot they stand on.
	host *Building
	plot int
}

// constructionStartHP: a new foundation has this fraction of full health.
var constructionStartHP = fixed.FromFraction(1, 10)

// constructionHP is the health of an undamaged foundation after `work` ticks of work.
func (b *Building) constructionHP(work fixed.Fixed) fixed.Fixed {
	start := b.Def.Health.Mul(constructionStartHP)
	return start + (b.Def.Health - start).Mul(work).DivInt(b.buildTicks)
}

// UnderConstruction reports whether the building is still being built.
func (b *Building) UnderConstruction() bool { return b.buildTicks > 0 }

// Progress is the construction progress in [0, 1] (1 = finished).
func (b *Building) Progress() fixed.Fixed {
	if b.buildTicks == 0 {
		return fixed.One
	}
	return b.work.DivInt(b.buildTicks).Min(fixed.One)
}

// PlotOccupied reports whether plot i of this fortress / outpost has a building.
func (b *Building) PlotOccupied(i int) bool { return i < len(b.plotUsers) && b.plotUsers[i] != nil }

// axes returns the building's right and forward unit vectors.
func (b *Building) axes() (right, forward fixed.Vec2) {
	forward = fixed.FromAngle(b.Facing)
	return fixed.Vec2{X: forward.Y, Y: -forward.X}, forward
}

// Local converts a building-local offset (x = right, y = front) to world space.
func (b *Building) Local(off fixed.Vec2) fixed.Vec2 {
	right, forward := b.axes()
	return b.Pos.Add(right.Scale(off.X)).Add(forward.Scale(off.Y))
}

// Alive reports whether the building still stands.
func (b *Building) Alive() bool { return b.HP > 0 }

// Attackable reports whether player p may attack the building
// (enemies only; neutral outposts are captured, not destroyed).
func (b *Building) Attackable(p PlayerID) bool {
	return b.Alive() && b.Owner != p && b.Owner != Neutral
}

// Closest returns the point of the footprint nearest to p (p itself if inside).
func (b *Building) Closest(p fixed.Vec2) fixed.Vec2 {
	right, forward := b.axes()
	d := p.Sub(b.Pos)
	hx := b.Def.Footprint.X.DivInt(2)
	hy := b.Def.Footprint.Y.DivInt(2)
	x := d.Dot(right).Clamp(-hx, hx)
	y := d.Dot(forward).Clamp(-hy, hy)
	return b.Pos.Add(right.Scale(x)).Add(forward.Scale(y))
}

// DistTo is the distance from p to the footprint (0 inside).
func (b *Building) DistTo(p fixed.Vec2) fixed.Fixed { return p.Dist(b.Closest(p)) }

// approach returns a point `dist` away from the footprint on the side facing from.
func (b *Building) approach(from fixed.Vec2, dist fixed.Fixed) fixed.Vec2 {
	c := b.Closest(from)
	dir := from.Sub(c).Normalize()
	if dir.IsZero() {
		dir = from.Sub(b.Pos).Normalize()
	}
	return c.Add(dir.Scale(dist))
}

// PlotPos returns the world position of plot i.
func (b *Building) PlotPos(i int) fixed.Vec2 { return b.Local(b.Def.Plots[i].Offset) }

// Covers reports whether p lies inside the footprint (a rotated rectangle),
// grown by margin on every side.
func (b *Building) Covers(p fixed.Vec2, margin fixed.Fixed) bool {
	right, forward := b.axes()
	d := p.Sub(b.Pos)
	hx := b.Def.Footprint.X.DivInt(2) + margin
	hy := b.Def.Footprint.Y.DivInt(2) + margin
	return d.Dot(right).Abs() <= hx && d.Dot(forward).Abs() <= hy
}

// SpawnBuilding places a finished building and blocks the nav cells whose
// centers lie under its footprint. Fails if the center is on blocked ground.
func (w *World) SpawnBuilding(owner PlayerID, buildingType string, pos fixed.Vec2, facing fixed.Fixed) (*Building, error) {
	def, ok := w.Catalog.Buildings[buildingType]
	if !ok {
		return nil, fmt.Errorf("sim: unknown building type %q", buildingType)
	}
	g := w.Nav.Grid()
	if !g.WalkableAt(pos) {
		return nil, fmt.Errorf("sim: building %q at %v is on blocked ground", buildingType, pos)
	}
	b := &Building{
		ID:     w.nextBuilding,
		Owner:  owner,
		Kind:   slices.Index(w.Catalog.BuildingIDs, buildingType),
		Def:    def,
		Pos:    pos,
		Facing: fixed.NormalizeAngle(facing),
		HP:     def.Health,

		captureBy: Neutral,
	}
	if len(def.Plots) > 0 {
		b.plotUsers = make([]*Building, len(def.Plots))
	}
	b.toggled = make([]bool, len(def.Toggles))
	w.nextBuilding++
	w.Buildings = append(w.Buildings, b)
	w.blockFootprint(b)
	if def.Garrison {
		w.spawnTowerRing(b)
	}
	return b, nil
}

// removeDestroyedBuildings drops buildings with no health left, frees their
// ground and checks whether a player lost their fortress.
func (w *World) removeDestroyedBuildings() {
	kept := w.Buildings[:0]
	var lost []PlayerID
	for _, b := range w.Buildings {
		if b.Alive() {
			kept = append(kept, b)
			continue
		}
		if b.Def.Capture != nil {
			// A destroyed outpost falls back to neutral and can be taken again.
			w.forgetBuilding(b)
			b.Owner = Neutral
			b.HP = b.Def.Health
			b.captureBy, b.captureTicks = Neutral, 0
			kept = append(kept, b)
			continue
		}
		if b.Def.Kind == data.BuildingFortress {
			lost = append(lost, b.Owner)
		}
		if b.Def.Garrison {
			w.forgetTowerRing(b)
		}
		w.forgetBuilding(b)
	}
	if len(kept) == len(w.Buildings) {
		return
	}
	for i := len(kept); i < len(w.Buildings); i++ {
		w.Buildings[i] = nil
	}
	w.Buildings = kept
	w.rebuildBlocked()
	for _, p := range lost {
		w.defeat(p)
	}
}

// buildOnPlot lays a foundation of buildingType on plot i of host.
func (w *World) buildOnPlot(player PlayerID, host *Building, i int, buildingType string) {
	def, ok := w.Catalog.Buildings[buildingType]
	if !ok || host.Owner != player || !host.Alive() || i < 0 || i >= len(host.Def.Plots) || host.PlotOccupied(i) {
		return
	}
	placement := data.PlacementPlot
	if host.Def.Plots[i].Class == data.PlotTower {
		placement = data.PlacementTower // a tower socket on the base wall
	}
	if !def.Allows(placement) || !def.FitsPlot(host.Def.Plots[i].Size) || !w.canBuild(player, buildingType) || !w.Meets(player, def.Requires) {
		return
	}
	if !w.pay(player, def.Cost) {
		return
	}
	var b *Building
	if placement == data.PlacementTower {
		b = w.placeBuilding(player, buildingType, host.PlotPos(i), host.Facing) // on the wall: no open ground under it
	} else {
		var err error
		if b, err = w.SpawnBuilding(player, buildingType, host.PlotPos(i), host.Facing); err != nil {
			return
		}
	}
	b.startConstruction()
	b.selfBuild = true
	b.host, b.plot = host, i
	host.plotUsers[i] = b
}

// startConstruction turns a freshly spawned building into a foundation.
func (b *Building) startConstruction() {
	b.buildTicks = max(1, cooldownTicks(b.Def.BuildTime))
	b.work = 0
	b.HP = b.constructionHP(0)
}

// workRate: construction work per tick with n workers. Each extra worker
// adds half the speed of the first (diminishing crowding).
func workRate(n int) fixed.Fixed {
	if n <= 0 {
		return 0
	}
	return fixed.One + fixed.FromInt(n-1).DivInt(2)
}

// cancelConstruction removes an unfinished building of the player.
func (w *World) cancelConstruction(player PlayerID, b *Building) {
	if b.Owner != player || !b.UnderConstruction() {
		return
	}
	w.refund(player, b.Def.Cost)
	w.RemoveBuilding(b.ID)
}

// canBuild reports whether the player's faction lists the building type.
func (w *World) canBuild(player PlayerID, buildingType string) bool {
	f := w.faction(player)
	return f != nil && slices.Contains(f.Buildings, buildingType)
}

// updateConstruction advances every unfinished building; health grows with progress.
func (w *World) updateConstruction() {
	for _, b := range w.Buildings {
		if b.buildTicks == 0 || !b.Alive() {
			continue
		}
		rate := workRate(b.builders)
		if b.selfBuild {
			rate = fixed.One
		}
		if rate == 0 {
			continue
		}
		rate = rate.Mul(mul(b.workSpeed))
		next := (b.work + rate).Min(fixed.FromInt(b.buildTicks))
		// Health follows the progress; damage taken meanwhile is kept
		// (the gain is the step between two progress targets, which sums to
		// exactly full health at the end).
		b.HP = min(b.HP+b.constructionHP(next)-b.constructionHP(b.work), b.Def.Health)
		b.work = next
		if b.work >= fixed.FromInt(b.buildTicks) {
			b.buildTicks, b.work = 0, 0
			for _, h := range w.Hordes {
				if h.construct == b { // done: workers stand by
					h.construct, h.constructing = nil, false
				}
			}
		}
	}
}

// RemoveBuilding deletes a building and frees the ground under it.
func (w *World) RemoveBuilding(id BuildingID) {
	i := slices.IndexFunc(w.Buildings, func(b *Building) bool { return b.ID == id })
	if i < 0 {
		return
	}
	b := w.Buildings[i]
	b.HP = 0
	if b.Def.Garrison {
		w.forgetTowerRing(b)
	}
	w.forgetBuilding(b)
	w.Buildings = slices.Delete(w.Buildings, i, i+1)
	w.rebuildBlocked()
}

// forgetBuilding stops every horde attacking b and frees its plot.
func (w *World) forgetBuilding(b *Building) {
	if h := b.host; h != nil && h.plotUsers[b.plot] == b {
		h.plotUsers[b.plot] = nil
	}
	for _, h := range w.Hordes {
		if h.siege == b {
			h.siege = nil
			h.fighting = false
			h.moving = false
		}
		if h.construct == b {
			h.construct = nil
			h.constructing = false
			h.moving = false
		}
	}
}

// rebuildBlocked resets the nav grid to terrain-only walkability and stamps
// every remaining building on it.
func (w *World) rebuildBlocked() {
	copy(w.Nav.Grid().Blocked, w.navTerrain)
	for _, b := range w.Buildings {
		w.blockFootprint(b)
	}
	w.Nav.Invalidate()
}

// blockFootprint marks the building's cells as blocked and drops cached flow fields.
func (w *World) blockFootprint(b *Building) {
	stamp(w.Nav, b)
}

// Building returns a building by ID (binary search; buildings are sorted by ID).
func (w *World) Building(id BuildingID) *Building {
	i, ok := slices.BinarySearchFunc(w.Buildings, id, func(b *Building, id BuildingID) int {
		return int(b.ID) - int(id)
	})
	if !ok {
		return nil
	}
	return w.Buildings[i]
}
