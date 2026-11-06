package sim

import (
	"slices"
	"testing"

	"dnd-rts/internal/data"
	"dnd-rts/internal/fixed"
)

// wallLineSpot: two ends of an open stretch for a wall of n segments, 45 m in front of player 0's
// keep, running across (the keep stays behind it).
func wallLineSpot(t *testing.T, w *World, n int) (fixed.Vec2, fixed.Vec2) {
	t.Helper()
	keep := fortressOf(w, 0)
	_, forward, right := startFrame(keep.Pos)
	def := w.Catalog.Buildings["castle_wall"]
	for d := 45; d <= 120; d += 5 {
		c := keep.Pos.Add(forward.Scale(fixed.FromInt(d)))
		half := right.Scale(def.Footprint.X.MulInt(n).DivInt(2))
		a, b := c.Sub(half), c.Add(half)
		if _, _, fits := w.WallLine(0, def, a, b); fits == n {
			return a, b
		}
	}
	t.Fatal("no open stretch for a wall")
	return fixed.Vec2{}, fixed.Vec2{}
}

func TestWorkersBuildAWallLine(t *testing.T) {
	w := newTestWorld(t)
	w.Step([]Command{{Player: 0, Kind: CmdDevGive, Name: "gold", Count: 5000}, {Player: 0, Kind: CmdDevGive, Name: "wood", Count: 5000}})
	gold := w.Stock(0, res(t, w, "gold"))
	a, b := wallLineSpot(t, w, 6)
	workers := workersOf(t, w)
	w.Step([]Command{{Player: 0, Kind: CmdBuildWall, Hordes: []HordeID{workers.ID}, Name: "castle_wall", Target: a, To: b}})
	if len(w.Walls) != 1 {
		t.Fatalf("walls %d, want the new line", len(w.Walls))
	}
	r := w.Walls[0]
	if len(r.Segs) != 6 || !r.open {
		t.Fatalf("segments %d (open %v), want 6 in a line", len(r.Segs), r.open)
	}
	if spent := gold - w.Stock(0, res(t, w, "gold")); spent != 6*15 {
		t.Errorf("paid %d gold, want 15 per segment", spent)
	}
	// The front faces away from the keep; the stairs are on the keep's side, at both ends.
	keep := fortressOf(w, 0)
	if fixed.FromAngle(r.Segs[0].Facing).Dot(keep.Pos.Sub(r.Segs[0].Pos)) > 0 {
		t.Error("the wall faces the keep")
	}
	if len(r.Stairs) < 2 || r.Stairs[0].node != 0 || r.Stairs[len(r.Stairs)-1].node != 5 {
		t.Errorf("stairs %+v, want one at each end", r.Stairs)
	}
	for i := 0; i < 20*200 && slicesAnyUnfinished(r.Segs); i++ {
		w.Step(nil)
	}
	if slicesAnyUnfinished(r.Segs) {
		t.Fatal("the workers did not finish the wall")
	}
	// It blocks the ground and infantry walks on it.
	if w.Nav.Grid().WalkableAt(r.Segs[2].Pos) {
		t.Error("the finished wall does not block")
	}
	h, _ := w.SpawnHorde(0, "archers", keep.Pos.Add(r.Segs[0].Pos.Sub(keep.Pos).Scale(fixed.FromFraction(1, 2))), 0)
	w.Step([]Command{{Player: 0, Kind: CmdMove, Hordes: []HordeID{h.ID}, Target: r.Segs[3].Pos}})
	for i := 0; i < 20*40 && !h.OnWall(); i++ {
		w.Step(nil)
	}
	if !h.OnWall() || h.wall != r {
		t.Fatal("archers did not climb the built wall")
	}
	for i := 0; i < 20*20; i++ {
		w.Step(nil)
	}
	if d := h.Pos.Dist(r.Segs[3].Pos); d > fixed.FromInt(4) {
		t.Errorf("archers stopped %v from the clicked spot", d)
	}
}

func slicesAnyUnfinished(bs []*Building) bool {
	for _, b := range bs {
		if b.Alive() && b.UnderConstruction() {
			return true
		}
	}
	return false
}

func TestWallLineStopsAtAnObstacle(t *testing.T) {
	w := newTestWorld(t)
	keep := fortressOf(w, 0)
	def := w.Catalog.Buildings["castle_wall"]
	_, forward, right := startFrame(keep.Pos)
	a := keep.Pos.Add(right.Scale(fixed.FromInt(-30))).Add(forward.Scale(fixed.FromInt(2)))
	_, _, fits := w.WallLine(0, def, a, a.Add(right.Scale(fixed.FromInt(60))))
	if fits == 0 || fits >= 9 {
		t.Errorf("a line across the keep fits %d segments, want it to stop before the keep", fits)
	}
}

func TestGateOverAFinishedWall(t *testing.T) {
	w := newTestWorld(t)
	w.Step([]Command{{Player: 0, Kind: CmdDevGive, Name: "gold", Count: 5000}, {Player: 0, Kind: CmdDevGive, Name: "wood", Count: 5000}})
	a, b := wallLineSpot(t, w, 9)
	segs := slices.Clone(w.layWall(0, w.Catalog.Buildings["castle_wall"], a, b, false, false))
	gate := w.Catalog.Buildings["castle_gate"]
	if _, _, _, ok := w.GateSpot(0, gate, segs[4].Pos); ok {
		t.Fatal("a gate over an unfinished wall")
	}
	for _, s := range segs {
		s.buildTicks, s.HP = 0, s.Def.Health
	}
	workers := workersOf(t, w)
	w.Step([]Command{{Player: 0, Kind: CmdBuildFree, Hordes: []HordeID{workers.ID}, Name: "castle_gate", Target: segs[4].Pos}})
	var g *Building
	for _, x := range w.Buildings {
		if x.Def.Kind == data.BuildingGate && x.Owner == 0 {
			g = x
		}
	}
	if g == nil || !g.UnderConstruction() {
		t.Fatal("no gatehouse foundation on the wall")
	}
	if segs[3].Alive() || segs[4].Alive() || segs[5].Alive() || !segs[2].Alive() || !segs[6].Alive() {
		t.Error("the gate did not replace exactly the three middle segments")
	}
	for i := 0; i < 20*90 && g.UnderConstruction(); i++ {
		w.Step(nil)
	}
	if g.UnderConstruction() {
		t.Fatal("the workers did not finish the gate")
	}
	w.Step([]Command{{Player: 0, Kind: CmdGate, TargetBuilding: g.ID, Count: 0}})
	w.Step(nil)
	if g.GateOpen() {
		t.Error("the gate did not close")
	}
}

// A line started at an end of an own wall continues it: one wall, one walkway, turns allowed —
// at the last end and at the first.
func TestWallContinuesFromAnEnd(t *testing.T) {
	w := newTestWorld(t)
	w.Step([]Command{{Player: 0, Kind: CmdDevGive, Name: "gold", Count: 5000}, {Player: 0, Kind: CmdDevGive, Name: "wood", Count: 5000}})
	def := w.Catalog.Buildings["castle_wall"]
	a, b := wallLineSpot(t, w, 4)
	w.layWall(0, def, a, b, true, false)
	if len(w.Walls) != 1 {
		t.Fatalf("walls %d", len(w.Walls))
	}
	r := w.Walls[0]
	keep := fortressOf(w, 0)
	_, forward, _ := startFrame(keep.Pos)
	// From the last end, turning 90° towards the keep.
	last := r.Nodes[len(r.Nodes)-1]
	added := w.layWall(0, def, last, last.Sub(forward.Scale(fixed.FromInt(25))), true, false)
	if len(added) < 3 || len(w.Walls) != 1 || len(r.Segs) != 4+len(added) {
		t.Fatalf("continuing from the end: %d new, %d walls, %d segments", len(added), len(w.Walls), len(r.Segs))
	}
	// From the first end, straight on (the new segments go in front).
	first := r.Nodes[0]
	dir := r.Nodes[0].Sub(r.Nodes[1]).Normalize()
	before := len(r.Segs)
	front := w.layWall(0, def, first, first.Add(dir.Scale(fixed.FromInt(22))), true, false) // 3 segments past the tip
	if len(front) != 3 || len(w.Walls) != 1 || len(r.Segs) != before+3 || r.Segs[0] != front[len(front)-1] {
		t.Fatalf("continuing from the start: %d new, %d walls, %d segments", len(front), len(w.Walls), len(r.Segs))
	}
	for i := 1; i < len(r.Nodes); i++ {
		if d := r.Nodes[i].Dist(r.Nodes[i-1]); d > def.Footprint.X+fixed.One {
			t.Fatalf("nodes %d and %d are %v apart: the walkway is broken", i-1, i, d)
		}
	}
	if _, ok := r.path(0, fixed.FromInt(len(r.Nodes)-1)); !ok {
		t.Error("the walkway does not run end to end")
	}
	// The 90° turn has a round joint, and it blocks the corner.
	joints := w.WallJoints()
	if len(joints) != 1 {
		t.Fatalf("joints %d, want one at the turn", len(joints))
	}
	if w.Nav.Grid().WalkableAt(joints[0].Pos) {
		t.Error("the joint does not block")
	}
	for _, st := range r.Stairs {
		if st.node < 0 || st.node >= len(r.Nodes) {
			t.Errorf("stair at node %d of %d", st.node, len(r.Nodes))
		}
	}
	if !slices.IsSortedFunc(r.Stairs, func(a, b wallStair) int { return a.node - b.node }) {
		t.Error("stairs out of order")
	}
}

// Workers sent to one foundation of a wall go on with the rest of it.
func TestHelpBuildContinuesTheWall(t *testing.T) {
	w := newTestWorld(t)
	a, b := wallLineSpot(t, w, 5)
	segs := w.layWall(0, w.Catalog.Buildings["castle_wall"], a, b, false, false)
	workers := workersOf(t, w)
	w.Step([]Command{{Player: 0, Kind: CmdConstruct, Hordes: []HordeID{workers.ID}, TargetBuilding: segs[2].ID}})
	for i := 0; i < 20*150 && slicesAnyUnfinished(segs); i++ {
		w.Step(nil)
	}
	if slicesAnyUnfinished(segs) {
		t.Fatal("the workers stopped after one segment")
	}
}
