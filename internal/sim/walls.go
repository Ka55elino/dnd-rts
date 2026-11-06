package sim

import (
	"slices"

	"dnd-rts/internal/data"
	"dnd-rts/internal/fixed"
)

// Walls built by workers (docs/BASES.md). The player drags a line: it becomes a row of wall
// segments (one footprint long each), laid as foundations and paid per segment; the workers put
// them up one after another, nearest first. A finished segment already blocks and can be walked
// on. Stairs come with the wall: at both ends and every stairEvery segments, on the inner side
// (the side of the owner's keep). A line started at an end of an own wall continues that wall
// (one walkway, turns allowed). A gatehouse replaces a stretch of finished wall.

const (
	maxWallSegments = 40 // one order
	stairEvery      = 8
)

// stairOut: how far from the wall's centre line the foot of a stair is (beyond half its depth).
var stairOut = fixed.FromFraction(5, 2)

// wallJoin: a new line that continues an own open wall from one of its ends.
type wallJoin struct {
	ring    *WallRing
	atStart bool // joins the first node (the new segments go before it), else the last
}

// joinAt: an end of an own open wall whose end segment is within reach of pos (a click on the last
// segment, or just past it), or nil.
func (w *World) joinAt(owner PlayerID, def *data.Building, pos fixed.Vec2) *wallJoin {
	reach := def.Footprint.X.MulInt(3).DivInt(4)
	var best *wallJoin
	bestD := reach
	for _, r := range w.Walls {
		if r.Owner != owner || !r.open || r.tower || len(r.Nodes) == 0 {
			continue
		}
		for _, end := range []int{0, len(r.Nodes) - 1} {
			if !r.Segs[end].Alive() || r.Segs[end].Def.Kind != data.BuildingWall {
				continue
			}
			if d := r.Nodes[end].Dist(pos); d <= bestD {
				bestD = d
				// A one-segment wall: its only node is the last one (the line goes on after it).
				best = &wallJoin{ring: r, atStart: end == 0 && len(r.Nodes) > 1}
			}
		}
	}
	return best
}

// WallPlan is what a wall order from a to b would lay: the segment centres, their facing (the
// front points out, away from the keep, or the same way as the wall it continues), how many of
// them fit (they stop at the first that doesn't), and the wall it continues (nil: a new line).
type WallPlan struct {
	Centres []fixed.Vec2
	Facing  fixed.Fixed
	Fits    int
	join    *wallJoin
}

// Joins reports whether the plan continues an existing wall.
func (p WallPlan) Joins() bool { return p.join != nil }

// PlanWall: see WallPlan. A line started on (or right past) an end segment of an own open wall
// continues that wall: its segments go on from the end, in the dragged direction (turns allowed).
func (w *World) PlanWall(owner PlayerID, def *data.Building, a, b fixed.Vec2) WallPlan {
	fx := def.Footprint.X
	var p WallPlan
	origin, first := a, fixed.One/2 // the first centre: half a segment from a new line's start
	var endOut fixed.Vec2
	if j := w.joinAt(owner, def, a); j != nil {
		p.join = j
		end := len(j.ring.Nodes) - 1
		if j.atStart {
			end = 0
		}
		// The new run pivots on the tip of the end segment's centre line: at a turn the two
		// segments meet there, and a round joint (wallJoints) fills the corner.
		along := j.ring.endAxis(end, b)
		origin = j.ring.Nodes[end].Add(along.Scale(fx / 2))
		endOut = j.ring.Out[end]
		if dir := b.Sub(origin).Normalize(); !dir.IsZero() && dir.Dot(along) < -fixed.One/2 {
			p.join = nil // turning back more than 120°: the wall would fold onto itself
			p.Fits = 0
			return p
		}
	}
	d := b.Sub(origin)
	length := d.Len()
	dir := d.Normalize()
	if dir.IsZero() {
		dir = fixed.Vec2{X: fixed.One}
	}
	n := (length + fx/2).Div(fx).Int()
	if p.join == nil {
		n = max(n, 1)
	}
	n = min(n, maxWallSegments)
	out := fixed.Vec2{X: -dir.Y, Y: dir.X} // left of the line
	if p.join != nil {
		if out.Dot(endOut) < 0 {
			out = out.Scale(-fixed.One) // the same side out as the wall it continues
		}
	} else {
		mid := origin.Add(dir.Scale(fx.MulInt(n).DivInt(2)))
		if f := w.fortressOf(owner); f != nil && out.Dot(f.Pos.Sub(mid)) > 0 {
			out = out.Scale(-fixed.One) // the front faces away from the keep
		}
	}
	p.Facing = out.Angle()
	// Checked a hair smaller: segments touch end to end.
	probe := *def
	probe.Footprint = def.Footprint.Sub(fixed.Vec2{X: fixed.FromFraction(1, 10), Y: fixed.FromFraction(1, 10)})
	if j := p.join; j != nil {
		// The first new segment may touch the end segment and, at a turn, the one before it.
		segs := j.ring.Segs
		if j.atStart {
			w.placeIgnore = segs[:min(2, len(segs))]
		} else {
			w.placeIgnore = segs[max(0, len(segs)-2):]
		}
		w.placeIgnoreStair.ring, w.placeIgnoreStair.node = j.ring, len(segs)-1
		if j.atStart {
			w.placeIgnoreStair.node = 0
		}
	}
	p.Fits = -1
	for k := 0; k < n; k++ {
		c := origin.Add(dir.Scale(fx.Mul(fixed.FromInt(k) + first)))
		p.Centres = append(p.Centres, c)
		if p.Fits < 0 && !w.fitsFree(&probe, c, p.Facing, false) {
			p.Fits = k
		}
		w.placeIgnore = nil // only the first new segment touches the end one
	}
	w.placeIgnore = nil
	w.placeIgnoreStair.ring = nil
	if p.Fits < 0 {
		p.Fits = n
	}
	return p
}

// WallLine is PlanWall's centres, facing and fit count (tests).
func (w *World) WallLine(owner PlayerID, def *data.Building, a, b fixed.Vec2) ([]fixed.Vec2, fixed.Fixed, int) {
	p := w.PlanWall(owner, def, a, b)
	return p.Centres, p.Facing, p.Fits
}

// buildWall: workers lay a wall line from a to b and start building it (stops at the first
// segment that doesn't fit or that the player can't pay for).
func (w *World) buildWall(c Command) {
	def, ok := w.Catalog.Buildings[c.Name]
	if !ok || !def.Allows(data.PlacementWall) || !w.canBuild(c.Player, def.ID) || !w.Meets(c.Player, def.Requires) {
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
	segs := w.layWall(c.Player, def, c.Target, c.To, false, true)
	if len(segs) == 0 {
		return
	}
	w.sendToBuild(workers, segs)
}

// sendToBuild: workers build the nearest of bs, then the rest (nearest first).
func (w *World) sendToBuild(workers []*Horde, bs []*Building) {
	for _, h := range workers {
		next := nearestBuilding(bs, h.Pos)
		if next == nil {
			continue
		}
		h.constructOrder(w, next)
		h.buildQueue = slices.Clone(bs)
	}
}

// layWall puts the segments of a wall line from a towards b (foundations, or finished ones): a new
// wall with its stairs, or more of the wall it continues. pay: each segment costs Def.Cost (stops
// when the player can't pay). Returns the new segments.
func (w *World) layWall(owner PlayerID, def *data.Building, a, b fixed.Vec2, finished, pay bool) []*Building {
	p := w.PlanWall(owner, def, a, b)
	out := fixed.FromAngle(p.Facing)
	var nodes, outs []fixed.Vec2
	var segs []*Building
	for k := 0; k < p.Fits; k++ {
		if pay && !w.pay(owner, def.Cost) {
			break
		}
		seg, err := w.SpawnBuilding(owner, def.ID, p.Centres[k], p.Facing)
		if err != nil {
			break
		}
		if !finished {
			seg.startConstruction()
		}
		nodes = append(nodes, seg.Pos)
		outs = append(outs, out)
		segs = append(segs, seg)
	}
	n := len(segs)
	if n == 0 {
		return nil
	}
	in := out.Scale(-(def.Footprint.Y/2 + stairOut))
	if j := p.join; j != nil {
		w.extendWall(j, nodes, outs, segs, in)
		w.rebuildBlocked() // the joint at the turn blocks too
		return segs
	}
	ring := &WallRing{Owner: owner, open: true, WalkHeight: def.WalkHeight, segLen: def.Footprint.X, Nodes: nodes, Out: outs, Segs: segs}
	for k := 0; k < n; k++ {
		if k == 0 || k == n-1 || k%stairEvery == 0 {
			ring.Stairs = append(ring.Stairs, wallStair{node: k, ground: nodes[k].Add(in)})
		}
	}
	w.Walls = append(w.Walls, ring)
	return segs
}

// extendWall adds new segments (in order away from the joined end) to a wall, with stairs every
// stairEvery segments and at the new end. Joining the first node puts them in front: everything
// that refers to the wall's node or stair numbers moves along.
func (w *World) extendWall(j *wallJoin, nodes, outs []fixed.Vec2, segs []*Building, in fixed.Vec2) {
	r := j.ring
	n := len(segs)
	// The stair at the joined end goes if the new wall covers its foot.
	end := len(r.Nodes) - 1
	if j.atStart {
		end = 0
	}
	for i := 0; i < len(r.Stairs); i++ {
		st := r.Stairs[i]
		if st.node != end || !slices.ContainsFunc(segs, func(s *Building) bool { return s.DistTo(st.ground) < stairClear }) {
			continue
		}
		w.dropStair(r, i)
		i--
	}
	var stairs []wallStair
	for k := 0; k < n; k++ {
		if k == n-1 || (k+1)%stairEvery == 0 {
			stairs = append(stairs, wallStair{node: k, ground: nodes[k].Add(in)})
		}
	}
	if !j.atStart {
		base := len(r.Nodes)
		r.Nodes = append(r.Nodes, nodes...)
		r.Out = append(r.Out, outs...)
		r.Segs = append(r.Segs, segs...)
		for _, st := range stairs {
			r.Stairs = append(r.Stairs, wallStair{node: base + st.node, ground: st.ground})
		}
		return
	}
	// In front: reverse the new run (it goes away from the first node), then shift the old one.
	nodes, outs, segs = slices.Clone(nodes), slices.Clone(outs), slices.Clone(segs)
	slices.Reverse(nodes)
	slices.Reverse(outs)
	slices.Reverse(segs)
	shift := fixed.FromInt(n)
	var front []wallStair
	for _, st := range stairs {
		front = append(front, wallStair{node: n - 1 - st.node, ground: st.ground})
	}
	slices.SortFunc(front, func(a, b wallStair) int { return a.node - b.node })
	for i := range r.Stairs {
		r.Stairs[i].node += n
	}
	r.Nodes = append(nodes, r.Nodes...)
	r.Out = append(outs, r.Out...)
	r.Segs = append(segs, r.Segs...)
	r.Stairs = append(front, r.Stairs...)
	for _, h := range w.Hordes {
		if h.wall == r {
			h.wallS += shift
			h.wallTo += shift
		}
		if e := h.wallEnter; e != nil && e.ring == r {
			e.goal += shift
			e.stair += len(front)
		}
		if x := h.wallExit; x != nil && h.wall == r {
			x.stair += len(front)
		}
	}
}

// GateSpot: where a gatehouse of type def goes for a click at pos — the wall of player whose
// walkway passes near pos, centred on the nearest node, if that stretch (as many segments as the
// gate is long) is finished wall. Returns the wall, the centre node and how many nodes it spans.
func (w *World) GateSpot(player PlayerID, def *data.Building, pos fixed.Vec2) (r *WallRing, centre, span int, ok bool) {
	if !def.Allows(data.PlacementGate) || !w.canBuild(player, def.ID) || !w.Meets(player, def.Requires) {
		return nil, 0, 0, false
	}
	ring, s, found := w.wallAt(player, pos)
	if !found || ring.tower {
		return nil, 0, 0, false
	}
	span = max((def.Footprint.X.Div(ring.segLen) + fixed.One/2).Int(), 1)
	centre = (s + fixed.One/2).Int()
	half := span / 2
	for k := centre - half; k < centre-half+span; k++ {
		if ring.open && (k < 0 || k >= len(ring.Segs)) {
			return nil, 0, 0, false
		}
		seg := ring.Segs[(k%len(ring.Segs)+len(ring.Segs))%len(ring.Segs)]
		if !seg.Alive() || seg.UnderConstruction() || seg.Def.Kind != data.BuildingWall || seg.onWallTower(w) {
			return nil, 0, 0, false
		}
	}
	return ring, centre, span, true
}

// onWallTower: a tower stands on this wall segment.
func (b *Building) onWallTower(w *World) bool {
	for _, t := range w.Buildings {
		if t.onWall && t.Alive() && b.DistTo(t.Pos) < fixed.One {
			return true
		}
	}
	return false
}

// buildGate replaces a stretch of finished wall with a gatehouse foundation; workers build it.
func (w *World) buildGate(player PlayerID, def *data.Building, pos fixed.Vec2, workers []*Horde) *Building {
	r, centre, span, ok := w.GateSpot(player, def, pos)
	if !ok || !w.pay(player, def.Cost) {
		return nil
	}
	gate := w.putGate(player, def, r, centre, span)
	gate.startConstruction()
	for _, h := range workers {
		h.constructOrder(w, gate)
	}
	return gate
}

// putGate swaps span segments of wall r around node centre for a (finished) gatehouse of type def.
func (w *World) putGate(player PlayerID, def *data.Building, r *WallRing, centre, span int) *Building {
	n := len(r.Segs)
	idx := func(k int) int { return (k%n + n) % n }
	at := r.Nodes[idx(centre)]
	facing := r.Out[idx(centre)].Angle()
	if span%2 == 0 { // an even span: between the two middle nodes
		at = at.Add(r.Nodes[idx(centre-1)]).Scale(fixed.One / 2)
	}
	half := span / 2
	var old []*Building
	for k := centre - half; k < centre-half+span; k++ {
		old = append(old, r.Segs[idx(k)])
	}
	for _, seg := range old {
		w.RemoveBuilding(seg.ID)
	}
	gate := w.placeBuilding(player, def.ID, at, facing)
	for k := centre - half; k < centre-half+span; k++ {
		r.Segs[idx(k)] = gate
	}
	w.rebuildBlocked()
	return gate
}

// nearestBuilding: the unfinished, standing building of bs closest to p (nil if none).
func nearestBuilding(bs []*Building, p fixed.Vec2) *Building {
	var best *Building
	for _, b := range bs {
		if b.Alive() && b.UnderConstruction() && (best == nil || b.DistTo(p) < best.DistTo(p)) {
			best = b
		}
	}
	return best
}

// continueBuildQueue: a worker horde that finished its foundation goes on to the nearest unfinished
// one of its queue (a wall line).
func (w *World) continueBuildQueue(h *Horde) {
	next := nearestBuilding(h.buildQueue, h.Pos)
	if next == nil {
		h.buildQueue = nil
		return
	}
	queue := h.buildQueue
	h.constructOrder(w, next)
	h.buildQueue = queue
}

// endAxis: the direction along the wall out of end node e (away from the rest of the wall; a
// one-segment wall: its axis, the way towards `towards`).
func (r *WallRing) endAxis(e int, towards fixed.Vec2) fixed.Vec2 {
	if len(r.Nodes) > 1 {
		prev := e - 1
		if e == 0 {
			prev = 1
		}
		return r.Nodes[e].Sub(r.Nodes[prev]).Normalize()
	}
	right, _ := r.Segs[e].axes()
	if right.Dot(towards.Sub(r.Nodes[e])) < 0 {
		right = right.Scale(-fixed.One)
	}
	return right
}

// WallJoint is a turn of a wall between two neighbouring standing segments: the point where their
// centre lines meet (drawn and blocked as a round tower-like post of the wall's thickness).
type WallJoint struct {
	Pos    fixed.Vec2
	Radius fixed.Fixed
}

// jointTurn: neighbouring segments turning more than ~10° get a joint.
var jointTurn = fixed.FromFraction(985, 1000)

// WallJoints lists the joints of all walls (see WallJoint).
func (w *World) WallJoints() []WallJoint {
	var out []WallJoint
	for _, r := range w.Walls {
		if r.tower {
			continue
		}
		n := len(r.Segs)
		last := n - 1
		if !r.open {
			last = n
		}
		for i := 0; i < last; i++ {
			a, b := r.Segs[i], r.Segs[(i+1)%n]
			if a == b || !a.Alive() || !b.Alive() || a.Def.Kind != data.BuildingWall || b.Def.Kind != data.BuildingWall {
				continue
			}
			ra, _ := a.axes()
			rb, _ := b.axes()
			if ra.Dot(rb).Abs() >= jointTurn {
				continue // (nearly) straight on
			}
			dir := b.Pos.Sub(a.Pos)
			if ra.Dot(dir) < 0 {
				ra = ra.Scale(-fixed.One)
			}
			out = append(out, WallJoint{Pos: a.Pos.Add(ra.Scale(a.Def.Footprint.X / 2)), Radius: a.Def.Footprint.Y / 2})
		}
	}
	return out
}

// stampWallJoints blocks the nav cells under the wall joints (rebuildBlocked).
func (w *World) stampWallJoints() {
	g := w.Nav.Grid()
	margin := g.CellSize.DivInt(4)
	for _, j := range w.WallJoints() {
		r := j.Radius + margin
		c0, r0 := g.Cell(j.Pos.Sub(fixed.Vec2{X: r, Y: r}))
		c1, r1 := g.Cell(j.Pos.Add(fixed.Vec2{X: r, Y: r}))
		for row := r0; row <= r1; row++ {
			for col := c0; col <= c1; col++ {
				i := g.Index(col, row)
				if g.Center(i).DistSq(j.Pos) <= r.Mul(r) {
					g.Blocked[i] = true
				}
			}
		}
	}
}

// wallOf: the wall a segment belongs to (nil if none).
func (w *World) wallOf(b *Building) *WallRing {
	for _, r := range w.Walls {
		if !r.tower && slices.Contains(r.Segs, b) {
			return r
		}
	}
	return nil
}

// dropStair removes stair i of r: hordes heading for it (or down it) forget that order, the
// others' stair numbers move down.
func (w *World) dropStair(r *WallRing, i int) {
	for _, h := range w.Hordes {
		if e := h.wallEnter; e != nil && e.ring == r {
			switch {
			case e.stair == i:
				h.wallEnter = nil
				h.stop()
			case e.stair > i:
				e.stair--
			}
		}
		if x := h.wallExit; x != nil && h.wall == r {
			switch {
			case x.stair == i:
				h.wallExit = nil
				h.wallTo = h.wallS
			case x.stair > i:
				x.stair--
			}
		}
	}
	r.Stairs = slices.Delete(r.Stairs, i, i+1)
}
