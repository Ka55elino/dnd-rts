package sim

import (
	"testing"

	"dnd-rts/internal/data"
	"dnd-rts/internal/fixed"
)

func wallOf(t *testing.T, w *World, p PlayerID) *WallRing {
	t.Helper()
	for _, r := range w.Walls {
		if r.Owner == p {
			return r
		}
	}
	t.Fatal("no wall")
	return nil
}

func gateOf(r *WallRing) *Building {
	for _, b := range r.Segs {
		if b.Def.Kind == data.BuildingGate {
			return b
		}
	}
	return nil
}

func closeGates(w *World, p PlayerID) {
	var cmds []Command
	for _, b := range w.Buildings {
		if b.Owner == p && b.Def.Kind == data.BuildingGate {
			cmds = append(cmds, Command{Player: p, Kind: CmdGate, TargetBuilding: b.ID, Count: 0})
		}
	}
	w.Step(cmds)
}

func TestGateToggle(t *testing.T) {
	w := newTestWorld(t)
	r := wallOf(t, w, 0)
	gate := gateOf(r)
	if gate == nil || !gate.GateOpen() {
		t.Fatal("no gate, or it starts closed")
	}
	g := w.Nav.Grid()
	right, _ := gate.axes()
	side := gate.Pos.Add(right.Scale(gate.Def.Footprint.X.DivInt(2) - fixed.FromInt(2)))
	if !g.WalkableAt(gate.Pos) || g.WalkableAt(side) {
		t.Fatal("open gatehouse: the passage should be open, its sides blocked")
	}
	// The enemy can't toggle our gate.
	w.Step([]Command{{Player: 1, Kind: CmdGate, TargetBuilding: gate.ID, Count: 0}})
	if !gate.GateOpen() {
		t.Fatal("the enemy closed our gate")
	}
	// Someone in the passage: closing waits.
	h, _ := w.SpawnHorde(0, "swordsmen", gate.Pos, 0)
	h.ordered = true
	w.Step([]Command{{Player: 0, Kind: CmdGate, TargetBuilding: gate.ID, Count: 0}})
	if !gate.GateOpen() {
		t.Fatal("closed on top of the soldiers")
	}
	inside := fortressOf(w, 0).Pos.Add(gate.Pos.Sub(fortressOf(w, 0).Pos).Normalize().Scale(fixed.FromInt(40)))
	w.Step([]Command{{Player: 0, Kind: CmdMove, Hordes: []HordeID{h.ID}, Target: inside}})
	for i := 0; i < 20*20 && gate.GateOpen(); i++ {
		w.Step(nil)
	}
	if gate.GateOpen() || w.Nav.Grid().WalkableAt(gate.Pos) {
		t.Fatal("the gate did not close once the passage was clear")
	}
	w.Step([]Command{{Player: 0, Kind: CmdGate, TargetBuilding: gate.ID, Count: 1}})
	if !gate.GateOpen() || !w.Nav.Grid().WalkableAt(gate.Pos) {
		t.Fatal("the gate did not open")
	}
	seg := r.Segs[len(r.Segs)/4]
	if seg.Def.Kind != data.BuildingWall || w.Nav.Grid().WalkableAt(seg.Pos) {
		t.Error("a wall segment lets units through")
	}
}

func TestEnemyCantWalkIntoTheBase(t *testing.T) {
	w := newTestWorld(t)
	closeGates(w, 0)
	keep := fortressOf(w, 0)
	raiders, _ := w.SpawnHorde(1, "raiders", keep.Pos.Add(fixed.V2i(110, 0)), 0)
	inside := keep.Pos.Add(fixed.V2i(30, 30))
	w.Step([]Command{{Player: 1, Kind: CmdMove, Hordes: []HordeID{raiders.ID}, Target: inside}})
	for i := 0; i < 20*40; i++ {
		w.Step(nil)
	}
	if raiders.Pos.Dist(keep.Pos) < fixed.FromInt(70) {
		t.Fatalf("raiders got inside the walls: %v from the keep", raiders.Pos.Dist(keep.Pos))
	}
}

func TestWalkwayStopsAtGatehouses(t *testing.T) {
	w := newTestWorld(t)
	r := wallOf(t, w, 0)
	gi := -1
	for i, b := range r.Segs {
		if b.Def.Kind == data.BuildingGate {
			gi = i
			break
		}
	}
	// Two gatehouses split the walkway into two arcs: no way from one side of a gate to the other.
	if _, ok := r.path(fixed.FromInt(gi-2), fixed.FromInt(gi+2)); ok {
		t.Error("walkway crosses a gatehouse")
	}
	if _, ok := r.path(fixed.FromInt(gi+2), fixed.FromInt(gi+10)); !ok {
		t.Error("no way along one arc")
	}
}

func TestInfantryClimbsWalksAndLeavesTheWall(t *testing.T) {
	w := newTestWorld(t)
	r := wallOf(t, w, 0)
	keep := fortressOf(w, 0)
	h, _ := w.SpawnHorde(0, "archers", keep.Pos.Add(fixed.V2i(30, 0)), 0)
	goal := r.Nodes[len(r.Nodes)/3]
	w.Step([]Command{{Player: 0, Kind: CmdMove, Hordes: []HordeID{h.ID}, Target: goal}})
	for i := 0; i < 20*60 && !(h.OnWall() && h.wallS == h.wallTo); i++ {
		w.Step(nil)
	}
	if !h.OnWall() || h.Pos.Dist(goal) > r.segLen {
		t.Fatalf("archers: on wall %v at %v, want the walkway near %v", h.OnWall(), h.Pos, goal)
	}
	if h.Soldiers[0].StatusFlags()&StatusOnWall == 0 {
		t.Error("no on-wall status for the client")
	}
	if h.attackRange(h.Unit.Ranged) <= h.Unit.Ranged.Range {
		t.Error("no range bonus on the wall")
	}
	// Down again, to a spot inside.
	down := keep.Pos.Add(fixed.V2i(-30, 0))
	w.Step([]Command{{Player: 0, Kind: CmdMove, Hordes: []HordeID{h.ID}, Target: down}})
	for i := 0; i < 20*60 && (h.OnWall() || h.IsMoving()); i++ {
		w.Step(nil)
	}
	if h.OnWall() || h.Pos.Dist(down) > fixed.FromInt(2) {
		t.Errorf("after leaving: on wall %v, %v from the target", h.OnWall(), h.Pos.Dist(down))
	}
}

func TestCavalryCantClimb(t *testing.T) {
	w := newTestWorld(t)
	r := wallOf(t, w, 0)
	keep := fortressOf(w, 0)
	k, _ := w.SpawnHorde(0, "knights", keep.Pos.Add(fixed.V2i(30, 0)), 0)
	w.Step([]Command{{Player: 0, Kind: CmdMove, Hordes: []HordeID{k.ID}, Target: r.Nodes[5]}})
	for i := 0; i < 20*30; i++ {
		w.Step(nil)
	}
	if k.OnWall() || k.wallEnter != nil {
		t.Error("knights went up the wall")
	}
}

func TestWallLayersInCombat(t *testing.T) {
	w := newTestWorld(t)
	r := wallOf(t, w, 0)
	swords, _ := w.SpawnHorde(0, "swordsmen", r.Nodes[10], 0)
	swords.wall, swords.wallS, swords.wallTo = r, fixed.FromInt(10), fixed.FromInt(10)
	raiders, _ := w.SpawnHorde(1, "raiders", r.Nodes[10].Add(r.Out[10].Scale(fixed.FromInt(8))), 0)
	if raiders.canTarget(swords) || swords.canTarget(raiders) {
		t.Error("melee between the wall top and the ground")
	}
	archers, _ := w.SpawnHorde(1, "archers", r.Nodes[10].Add(r.Out[10].Scale(fixed.FromInt(20))), 0)
	if !archers.canTarget(swords) {
		t.Error("archers can't shoot at the wall")
	}
}

func TestBrokenSegmentDropsUnitsAndOpensAGap(t *testing.T) {
	w := newTestWorld(t)
	r := wallOf(t, w, 0)
	i := len(r.Segs) / 4
	seg := r.Segs[i]
	h, _ := w.SpawnHorde(0, "swordsmen", seg.Pos, 0)
	h.wall, h.wallS, h.wallTo = r, fixed.FromInt(i), fixed.FromInt(i)
	seg.HP = 0
	w.Step(nil)
	w.Step(nil)
	if h.OnWall() {
		t.Error("still on a destroyed segment")
	}
	if !w.navFor(1).Grid().WalkableAt(seg.Pos) {
		t.Error("the breach is still blocked")
	}
}

func TestTowerOnTheWallShoots(t *testing.T) {
	w := newTestWorld(t)
	keep := fortressOf(w, 0)
	w.Step([]Command{{Player: 0, Kind: CmdDevGive, Name: "gold", Count: 5000}, {Player: 0, Kind: CmdDevGive, Name: "wood", Count: 5000}})
	sock := -1
	for i, p := range keep.Def.Plots {
		if p.Class == data.PlotTower {
			sock = i
			break
		}
	}
	w.Step([]Command{{Player: 0, Kind: CmdBuildOnPlot, TargetBuilding: keep.ID, Count: sock, Name: "archer_tower"}})
	tower := w.Buildings[len(w.Buildings)-1]
	if tower.Def.ID != "archer_tower" {
		t.Fatalf("no tower on the socket (last building %s)", tower.Def.ID)
	}
	w.Step([]Command{{Player: 0, Kind: CmdBuildOnPlot, TargetBuilding: keep.ID, Count: 0, Name: "archer_tower"}})
	if w.Buildings[len(w.Buildings)-1] != tower {
		t.Error("a tower was built on a ground plot")
	}
	for i := 0; i < 20*31 && tower.UnderConstruction(); i++ {
		w.Step(nil)
	}
	out := tower.Pos.Add(tower.Pos.Sub(keep.Pos).Normalize().Scale(fixed.FromInt(20)))
	enemy, _ := w.SpawnHorde(1, "raiders", out, 0)
	hp := totalHP(enemy)
	for i := 0; i < 20*5; i++ {
		enemy.ordered, enemy.attack = true, nil
		w.Step(nil)
	}
	if totalHP(enemy) >= hp {
		t.Error("the tower did not shoot")
	}
}
