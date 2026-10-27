package sim

import "testing"

func TestBuildOnPlot(t *testing.T) {
	w := newTestWorld(t)
	keep := w.Buildings[0] // player 0
	n := len(w.Buildings)
	w.Step([]Command{{Player: 0, Kind: CmdBuildOnPlot, TargetBuilding: keep.ID, Count: 1, Name: "barracks"}})
	if len(w.Buildings) != n+1 || !keep.PlotOccupied(1) {
		t.Fatalf("barracks not placed (buildings %d → %d)", n, len(w.Buildings))
	}
	b := w.Buildings[len(w.Buildings)-1]
	if !b.UnderConstruction() || b.HP >= b.Def.Health {
		t.Fatalf("new barracks: constructing=%v hp=%v", b.UnderConstruction(), b.HP)
	}
	if b.Pos.Dist(keep.PlotPos(1)) > 0 {
		t.Errorf("barracks at %v, plot at %v", b.Pos, keep.PlotPos(1))
	}

	// The same plot can't take a second building; the enemy can't use our plots.
	w.Step([]Command{
		{Player: 0, Kind: CmdBuildOnPlot, TargetBuilding: keep.ID, Count: 1, Name: "barracks"},
		{Player: 1, Kind: CmdBuildOnPlot, TargetBuilding: keep.ID, Count: 2, Name: "barracks"},
		{Player: 0, Kind: CmdBuildOnPlot, TargetBuilding: keep.ID, Count: 4, Name: "mage_guild"}, // too big for a small plot
	})
	if len(w.Buildings) != n+1 {
		t.Fatalf("invalid plot builds accepted: %d buildings", len(w.Buildings))
	}

	for i := 0; i < 20*21 && b.UnderConstruction(); i++ {
		w.Step(nil)
	}
	if b.UnderConstruction() || b.HP != b.Def.Health {
		t.Fatalf("after build time: constructing=%v hp=%v/%v", b.UnderConstruction(), b.HP, b.Def.Health)
	}

	// Destroyed → plot is free again.
	b.HP = 0
	w.Step(nil)
	if keep.PlotOccupied(1) {
		t.Error("plot still occupied after its building was destroyed")
	}
}

func TestCancelConstruction(t *testing.T) {
	w := newTestWorld(t)
	keep := w.Buildings[0]
	w.Step([]Command{{Player: 0, Kind: CmdBuildOnPlot, TargetBuilding: keep.ID, Count: 0, Name: "barracks"}})
	b := w.Buildings[len(w.Buildings)-1]
	w.Step([]Command{{Player: 1, Kind: CmdCancelBuild, TargetBuilding: b.ID}}) // not theirs
	if w.Building(b.ID) == nil {
		t.Fatal("enemy cancelled our construction")
	}
	w.Step([]Command{{Player: 0, Kind: CmdCancelBuild, TargetBuilding: b.ID}})
	if w.Building(b.ID) != nil || keep.PlotOccupied(0) {
		t.Fatal("cancel did not remove the foundation / free the plot")
	}
	if !w.Nav.Grid().WalkableAt(b.Pos) {
		t.Error("ground still blocked after cancel")
	}
}
