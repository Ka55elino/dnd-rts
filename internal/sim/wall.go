package sim

import (
	"slices"

	"dnd-rts/internal/data"
	"dnd-rts/internal/fixed"
	"dnd-rts/internal/sim/nav"
)

// Base walls (docs/BASES.md): a fortress with a walled base gets its wall
// segments and gates as buildings when it is placed. Walls block movement;
// gates block everyone but their owner (each player paths on its own copy of
// the grid). Infantry climbs the walls by the stairs inside the base and
// walks along the top: the walkway is the loop of segment centres.

// WallRing is the wall of one base.
type WallRing struct {
	Owner      PlayerID
	Fortress   *Building
	Nodes      []fixed.Vec2 // segment centres, around the base in order
	Out        []fixed.Vec2 // outward direction at each node
	Segs       []*Building  // the segment buildings (dead ones = gaps)
	Stairs     []wallStair
	WalkHeight fixed.Fixed
	segLen     fixed.Fixed // average distance between nodes
	tower      bool        // a garrison building (watchtower): one node, one horde, packed tight
}

type wallStair struct {
	node   int
	ground fixed.Vec2
}

var (
	wallRangeBonus = fixed.FromFraction(25, 100) // ranged units on walls shoot farther
	wallBand       = fixed.FromInt(5)            // a click this close to the walkway means "onto the wall"
	wallRows       = 4                           // soldiers abreast on the walkway
	wallSpacing    = fixed.FromFraction(13, 10)
	stairReach     = fixed.FromInt(3)
)

// spawnWalls builds the wall of fortress f from its base layout.
func (w *World) spawnWalls(f *Building) {
	l := f.Def.Base
	if l == nil || len(l.Segments) == 0 {
		return
	}
	right, forward := f.axes()
	ring := &WallRing{Owner: f.Owner, Fortress: f}
	for _, s := range l.Segments {
		id := l.Wall
		if s.Gate {
			id = l.Gate
		}
		out := right.Scale(s.Out.X).Add(forward.Scale(s.Out.Y))
		b := w.placeBuilding(f.Owner, id, f.Local(s.Offset), out.Angle())
		ring.Nodes = append(ring.Nodes, b.Pos)
		ring.Out = append(ring.Out, out)
		ring.Segs = append(ring.Segs, b)
		ring.WalkHeight = max(ring.WalkHeight, b.Def.WalkHeight)
	}
	for _, st := range l.StairList {
		ring.Stairs = append(ring.Stairs, wallStair{node: st.Node, ground: f.Local(st.Ground)})
	}
	per := fixed.Fixed(0)
	for i := range ring.Nodes {
		per += ring.Nodes[(i+1)%len(ring.Nodes)].Sub(ring.Nodes[i]).Len()
	}
	ring.segLen = per.DivInt(len(ring.Nodes))
	w.Walls = append(w.Walls, ring)
}

// placeBuilding spawns a building without the open-ground check (walls next
// to each other, towers on a wall).
func (w *World) placeBuilding(owner PlayerID, buildingType string, pos fixed.Vec2, facing fixed.Fixed) *Building {
	def := w.Catalog.Buildings[buildingType]
	b := &Building{
		ID: w.nextBuilding, Owner: owner, Kind: slices.Index(w.Catalog.BuildingIDs, buildingType), Def: def,
		Pos: pos, Facing: fixed.NormalizeAngle(facing), HP: def.Health, captureBy: Neutral,
		gateOpen: def.Kind == data.BuildingGate, gateWant: def.Kind == data.BuildingGate, // gates start open
	}
	if len(def.Plots) > 0 {
		b.plotUsers = make([]*Building, len(def.Plots))
	}
	b.toggled = make([]bool, len(def.Toggles))
	w.nextBuilding++
	w.Buildings = append(w.Buildings, b)
	w.blockFootprint(b)
	return b
}

// --- navigation and gates ---

// navFor is the grid hordes of player p path on (one shared grid: gates are
// open or closed for everyone, like in BFME).
func (w *World) navFor(p PlayerID) *nav.Cache { return w.Nav }

// stamp blocks b's cells. A gatehouse always blocks its sides; its passage
// only while closed.
func stamp(c *nav.Cache, b *Building) {
	g := c.Grid()
	margin := fixed.Fixed(0)
	wallLike := b.Def.Kind == data.BuildingWall || b.Def.Kind == data.BuildingGate
	if wallLike {
		margin = g.CellSize.DivInt(4) // wider than the footprint: no corner leaks through a 7 m wall, a 1-segment breach still opens
	}
	open := b.Def.Kind == data.BuildingGate && b.gateOpen
	right, _ := b.axes()
	r := b.Def.Footprint.X.DivInt(2) + b.Def.Footprint.Y.DivInt(2) + margin
	c0, r0 := g.Cell(b.Pos.Sub(fixed.Vec2{X: r, Y: r}))
	c1, r1 := g.Cell(b.Pos.Add(fixed.Vec2{X: r, Y: r}))
	for row := r0; row <= r1; row++ {
		for col := c0; col <= c1; col++ {
			i := g.Index(col, row)
			p := g.Center(i)
			if !b.Covers(p, margin) {
				continue
			}
			if open && p.Sub(b.Pos).Dot(right).Abs() <= b.Def.Passage.DivInt(2) {
				continue // the open passage
			}
			g.Blocked[i] = true
		}
	}
	c.Invalidate()
}

// setGate opens a gate now, or asks it to close (it closes once nobody stands in the passage).
func (w *World) setGate(player PlayerID, b *Building, open bool) {
	if b == nil || b.Owner != player || b.Def.Kind != data.BuildingGate || !b.Alive() {
		return
	}
	b.gateWant = open
	if open && !b.gateOpen {
		b.gateOpen = true
		w.rebuildBlocked()
	}
}

// updateGates closes the gates that were told to, once their passage is clear.
func (w *World) updateGates() {
	changed := false
	for _, b := range w.Buildings {
		if b.Def.Kind != data.BuildingGate || b.gateWant || !b.gateOpen || !b.Alive() {
			continue
		}
		clear := true
		for _, s := range w.Soldiers {
			if s.Alive() && s.Horde.wall == nil && !s.Horde.Airborne() && b.Covers(s.Pos, s.Horde.Unit.Radius) {
				clear = false
				break
			}
		}
		if clear {
			b.gateOpen = false
			changed = true
		}
	}
	if changed {
		w.rebuildBlocked()
	}
}

// GateOpen reports whether a gate lets units through.
func (b *Building) GateOpen() bool { return b.gateOpen }

// --- hordes on walls ---

// canClimb: foot soldiers only (no cavalry, monsters, machines or flyers).
func (h *Horde) canClimb() bool {
	u := h.Unit
	return u.HasTag("infantry") && !u.HasTag("cavalry") && !u.Flying && !u.HasTag("mechanical")
}

// OnWall reports whether the horde stands on a wall.
func (h *Horde) OnWall() bool { return h.wall != nil }

// layer: 0 ground, 1 on a wall, 2 in the air. Melee only reaches the same
// layer (flyers reach everyone); ranged attacks reach all.
func (h *Horde) layer() int {
	switch {
	case h.Airborne():
		return 2
	case h.wall != nil:
		return 1
	}
	return 0
}

// meleeReaches: can a's melee hit t (layers and air targeting)?
func meleeReaches(a, t *Horde) bool {
	if a.Unit.Melee == nil {
		return false
	}
	if t.Airborne() {
		return a.Unit.Melee.CanHit(true)
	}
	if !a.Unit.Melee.CanHit(false) {
		return false
	}
	return a.Airborne() || a.layer() == t.layer()
}

// wallAt: player p's own wall whose walkway passes within wallBand of pos,
// and the walkway position there.
func (w *World) wallAt(p PlayerID, pos fixed.Vec2) (*WallRing, fixed.Fixed, bool) {
	for _, r := range w.Walls {
		if r.Owner != p {
			continue
		}
		best, bestD := -1, fixed.Max
		for i, n := range r.Nodes {
			if d := n.Dist(pos); d < bestD {
				best, bestD = i, d
			}
		}
		if best >= 0 && bestD <= wallBand+r.segLen.DivInt(2) && r.walkable(best) {
			return r, fixed.FromInt(best), true
		}
	}
	return nil, 0, false
}

// ringPoint: the walkway point at position s (fractional node index).
func (r *WallRing) point(s fixed.Fixed) fixed.Vec2 {
	n := len(r.Nodes)
	i := s.Int() % n
	t := s - fixed.FromInt(s.Int())
	return r.Nodes[i].Add(r.Nodes[(i+1)%n].Sub(r.Nodes[i]).Scale(t))
}

func (r *WallRing) outAt(s fixed.Fixed) fixed.Vec2 {
	n := len(r.Nodes)
	i := s.Int() % n
	return r.Out[i].Add(r.Out[(i+1)%n]).Normalize()
}

// ringDelta: signed shortest way from s to t around the loop (in nodes).
func (r *WallRing) ringDelta(s, t fixed.Fixed) fixed.Fixed {
	n := fixed.FromInt(len(r.Nodes))
	d := t - s
	for d > n.DivInt(2) {
		d -= n
	}
	for d < -n.DivInt(2) {
		d += n
	}
	return d
}

func (r *WallRing) wrap(s fixed.Fixed) fixed.Fixed {
	n := fixed.FromInt(len(r.Nodes))
	for s < 0 {
		s += n
	}
	for s >= n {
		s -= n
	}
	return s
}

// walkable: units can stand on node i (a standing wall segment; gatehouses break the walkway).
func (r *WallRing) walkable(i int) bool {
	b := r.Segs[(i%len(r.Segs)+len(r.Segs))%len(r.Segs)]
	return b.Alive() && b.Def.Kind != data.BuildingGate && !b.UnderConstruction()
}

// path: signed walk from s to t along the walkway that never crosses a gap or
// a gatehouse (the shorter of the two ways if both are clear).
func (r *WallRing) path(s, t fixed.Fixed) (fixed.Fixed, bool) {
	n := len(r.Nodes)
	d := r.ringDelta(s, t)
	clear := func(d fixed.Fixed) bool {
		from := (s + fixed.One/2).Int()
		steps := (d.Abs() + fixed.One/2).Int()
		dir := 1
		if d < 0 {
			dir = -1
		}
		for k := 0; k <= steps; k++ {
			if !r.walkable(from + dir*k) {
				return false
			}
		}
		return true
	}
	if clear(d) {
		return d, true
	}
	other := d - fixed.FromInt(n)
	if d < 0 {
		other = d + fixed.FromInt(n)
	}
	if clear(other) {
		return other, true
	}
	return 0, false
}

// segAt: the segment under walkway position s (nearest node).
func (r *WallRing) segAt(s fixed.Fixed) *Building {
	i := (s + fixed.One/2).Int() % len(r.Nodes)
	return r.Segs[i]
}

// nearestStair: the stair minimizing ground distance from `ground` plus the
// walk along the wall to / from walkway position s.
func (r *WallRing) nearestStair(ground fixed.Vec2, s fixed.Fixed) int {
	best, bestCost := -1, fixed.Max
	for i, st := range r.Stairs {
		d, ok := r.path(fixed.FromInt(st.node), s)
		if !ok || !r.walkable(st.node) {
			continue
		}
		cost := st.ground.Dist(ground) + d.Abs().Mul(r.segLen)
		if cost < bestCost {
			best, bestCost = i, cost
		}
	}
	return best
}

// orderOntoWall: walk to the best stair, climb, then go to walkway position goal.
func (w *World) orderOntoWall(h *Horde, r *WallRing, goal fixed.Fixed) {
	if h.wall == r {
		h.clearOrders()
		h.wallTo, h.wallExit = goal, nil
		return
	}
	if r.tower && w.towerTaken(r, h) {
		return // a watchtower holds one horde
	}
	if h.wall != nil { // another wall: down first, then like from the ground
		w.orderOffWall(h, r.point(goal))
		return
	}
	si := r.nearestStair(h.Pos, goal)
	if si < 0 {
		return
	}
	h.clearOrders()
	h.wallEnter = &wallEnter{ring: r, stair: si, goal: goal}
	h.ordered = true
	h.setDestination(r.Stairs[si].ground)
}

// orderOffWall: walk the walkway to the stair nearest to target, go down, then move there.
func (w *World) orderOffWall(h *Horde, target fixed.Vec2) {
	r := h.wall
	si := r.nearestStair(target, h.wallS)
	if si < 0 {
		return
	}
	h.clearOrders()
	t := target
	h.wallExit = &wallExit{stair: si, target: t}
	h.wallTo = fixed.FromInt(r.Stairs[si].node)
}

type wallEnter struct {
	ring  *WallRing
	stair int
	goal  fixed.Fixed
}

type wallExit struct {
	stair  int
	target fixed.Vec2
}

// updateWall: climbing up, walking the walkway, going down, falling off broken segments.
// Returns true when the horde is on a wall (normal movement is skipped).
func (w *World) updateWall(h *Horde) bool {
	if e := h.wallEnter; e != nil && h.wall == nil {
		st := e.ring.Stairs[e.stair]
		if !e.ring.Segs[st.node].Alive() {
			h.wallEnter = nil
			return false
		}
		if h.Pos.Dist(st.ground) <= stairReach {
			h.wallEnter = nil
			h.wall, h.wallS, h.wallTo = e.ring, fixed.FromInt(st.node), e.goal
			h.moving, h.ordered = false, false
			h.Pos = e.ring.point(h.wallS)
		}
		return h.wall != nil
	}
	r := h.wall
	if r == nil {
		return false
	}
	// The segment underfoot is gone: everyone drops down on the inside.
	if !r.walkable((h.wallS + fixed.One/2).Int()) {
		w.dropOffWall(h)
		return false
	}
	d, ok := r.path(h.wallS, h.wallTo)
	if !ok {
		h.wallTo, h.wallExit = h.wallS, nil // cut off (a breach or a gatehouse): stay
	} else if d != 0 {
		step := h.Unit.Speed.Mul(TickDt).Mul(h.slowFactor(w)).Div(r.segLen)
		move := d.Clamp(-step, step)
		next := r.wrap(h.wallS + move)
		if !r.walkable((next + fixed.One/2).Int()) {
			h.wallTo = h.wallS // a gap: stop at the edge
		} else {
			h.wallS = next
		}
	}
	h.Pos = r.point(h.wallS)
	h.Facing = r.outAt(h.wallS).Angle() // watch the outside
	if x := h.wallExit; x != nil && r.ringDelta(h.wallS, h.wallTo).Abs() < fixed.FromFraction(1, 100) {
		st := r.Stairs[x.stair]
		h.wallExit = nil
		h.wall = nil
		h.Pos = st.ground
		if p, ok := w.targetFor(h, x.target); ok {
			h.moveTo(p)
		}
		return false
	}
	return true
}

// dropOffWall: the horde falls to the nearest open ground inside.
func (w *World) dropOffWall(h *Horde) {
	r := h.wall
	h.wall, h.wallExit, h.wallEnter = nil, nil, nil
	in := h.Pos.Sub(r.outAt(h.wallS).Scale(fixed.FromInt(6)))
	if p, ok := w.targetFor(h, in); ok {
		h.Pos = p
	}
	h.moving = false
}

// wallSlotCols: on a walkway the formation is wallRows deep at most.
func (h *Horde) wallSlotCols() int {
	n := max(len(h.Soldiers), 1)
	if h.wall != nil && h.wall.tower {
		return max(1, intSqrt(n)) // a square on the platform
	}
	return max(1, (n+wallRows-1)/wallRows)
}

func intSqrt(n int) int {
	r := 0
	for (r+1)*(r+1) <= n {
		r++
	}
	if r*r < n {
		r++
	}
	return r
}

// towerSpacing: soldiers on a watchtower platform stand close.
var towerSpacing = fixed.FromFraction(6, 10)

// spawnTowerRing makes a garrison building climbable: a one-node "wall" with
// a stair at its front.
func (w *World) spawnTowerRing(b *Building) {
	ring := &WallRing{
		Owner: b.Owner, Nodes: []fixed.Vec2{b.Pos}, Segs: []*Building{b}, WalkHeight: b.Def.WalkHeight, tower: true, segLen: max(b.Def.Footprint.X, fixed.One),
		Out: []fixed.Vec2{fixed.FromAngle(b.Facing)},
	}
	ring.Stairs = []wallStair{{node: 0, ground: b.Local(fixed.Vec2{Y: b.Def.Footprint.Y.DivInt(2) + fixed.FromInt(2)})}}
	w.Walls = append(w.Walls, ring)
}

// towerTaken: another horde is on (or heading up) the watchtower.
func (w *World) towerTaken(r *WallRing, h *Horde) bool {
	for _, x := range w.Hordes {
		if x != h && x.Alive() && (x.wall == r || (x.wallEnter != nil && x.wallEnter.ring == r)) {
			return true
		}
	}
	return false
}

// forgetTowerRing drops the ring of a destroyed garrison building (hordes on it fall off).
func (w *World) forgetTowerRing(b *Building) {
	w.Walls = slices.DeleteFunc(w.Walls, func(r *WallRing) bool { return r.tower && r.Segs[0] == b })
}
