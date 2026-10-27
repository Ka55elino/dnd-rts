package sim

import (
	"testing"

	"dnd-rts/internal/fixed"
)

func res(t *testing.T, w *World, id string) int {
	t.Helper()
	r := w.Catalog.ResourceIndex(id)
	if r < 0 {
		t.Fatalf("no resource %q", id)
	}
	return r
}

func TestStartingStocksAndFood(t *testing.T) {
	w := newTestWorld(t)
	if got := w.Stock(0, res(t, w, "gold")); got != 500 {
		t.Errorf("gold = %d, want 500", got)
	}
	food := res(t, w, "food")
	if got := w.CapProvided(0, food); got != 20 { // keep
		t.Errorf("food provided = %d, want 20", got)
	}
	if w.CapUsed(0, food) == 0 {
		t.Error("starting army uses no food")
	}
}

func TestPlotBuildingCostsAndRefund(t *testing.T) {
	w := newTestWorld(t)
	gold, wood := res(t, w, "gold"), res(t, w, "wood")
	keep := w.Buildings[0]
	w.Step([]Command{{Player: 0, Kind: CmdBuildOnPlot, TargetBuilding: keep.ID, Count: 0, Name: "barracks"}})
	if w.Stock(0, gold) != 300 || w.Stock(0, wood) != 200 {
		t.Fatalf("after barracks: gold %d wood %d", w.Stock(0, gold), w.Stock(0, wood))
	}
	b := w.Buildings[len(w.Buildings)-1]
	w.Step([]Command{{Player: 0, Kind: CmdCancelBuild, TargetBuilding: b.ID}})
	if w.Stock(0, gold) != 500 || w.Stock(0, wood) != 300 {
		t.Fatalf("after cancel: gold %d wood %d", w.Stock(0, gold), w.Stock(0, wood))
	}
	// Not enough wood: refused, nothing taken.
	w.Step([]Command{{Player: 0, Kind: CmdDevGive, Name: "wood", Count: -250}})
	n := len(w.Buildings)
	w.Step([]Command{{Player: 0, Kind: CmdBuildOnPlot, TargetBuilding: keep.ID, Count: 0, Name: "barracks"}})
	if len(w.Buildings) != n || w.Stock(0, gold) != 500 {
		t.Fatal("barracks placed without enough wood")
	}
}

func TestOutpostIncome(t *testing.T) {
	w := newTestWorld(t)
	mine := outpost(t, w, "gold_mine")
	mine.Owner = 0
	gold := res(t, w, "gold")
	before := w.Stock(0, gold)
	for i := 0; i < 20*60; i++ {
		w.Step(nil)
	}
	if got := w.Stock(0, gold) - before; got != 15*12 { // 15 every 5 s for a minute
		t.Errorf("one minute of gold mine = %d, want 180", got)
	}
}

func TestTrainHorde(t *testing.T) {
	w := newTestWorld(t)
	keep := w.Buildings[0]
	food := res(t, w, "food")
	// Make room: the starting army is over the keep's 20 food.
	w.Step([]Command{{Player: 0, Kind: CmdDevSpawnBuilding, Name: "farm", Target: keep.Pos.Add(fixed.V2i(0, 30))},
		{Player: 0, Kind: CmdDevSpawnBuilding, Name: "farm", Target: keep.Pos.Add(fixed.V2i(0, -30))},
		{Player: 0, Kind: CmdDevSpawnBuilding, Name: "farm", Target: keep.Pos.Add(fixed.V2i(-30, 0))}})
	used := w.CapUsed(0, food)
	hordes := len(w.Hordes)
	rally := keep.Pos.Add(fixed.V2i(40, 0))
	w.Step([]Command{
		{Player: 0, Kind: CmdRally, TargetBuilding: keep.ID, Target: rally},
		{Player: 0, Kind: CmdTrain, TargetBuilding: keep.ID, Name: "builders"},
		{Player: 0, Kind: CmdTrain, TargetBuilding: keep.ID, Name: "swordsmen"}, // keep doesn't train swordsmen
	})
	if q := keep.Queue(); q == nil || len(q.Hordes) != 1 {
		t.Fatalf("queue %+v", q)
	}
	if w.CapUsed(0, food) != used+2 {
		t.Errorf("queued builder should reserve food: %d → %d", used, w.CapUsed(0, food))
	}
	for i := 0; i < 20*11; i++ {
		w.Step(nil)
	}
	if len(w.Hordes) != hordes+1 || keep.Queue() != nil {
		t.Fatalf("builder not trained (hordes %d → %d)", hordes, len(w.Hordes))
	}
	h := w.Hordes[len(w.Hordes)-1]
	if h.Def.ID != "builders" || h.Owner != 0 || (!h.IsMoving() && h.Pos.Dist(rally) > fixed.FromInt(6)) {
		t.Errorf("new horde %s owner %d moving %v pos %v", h.Def.ID, h.Owner, h.IsMoving(), h.Pos)
	}
}

func TestFoodLimitBlocksTraining(t *testing.T) {
	w := newTestWorld(t)
	keep := w.Buildings[0]
	gold := res(t, w, "gold")
	before := w.Stock(0, gold)
	// Starting army already uses more than the keep provides.
	w.Step([]Command{{Player: 0, Kind: CmdTrain, TargetBuilding: keep.ID, Name: "builders"}})
	if keep.Queue() != nil || w.Stock(0, gold) != before {
		t.Fatal("trained over the food limit")
	}
}

func TestCancelTrainingRefunds(t *testing.T) {
	w := newTestWorld(t)
	keep := w.Buildings[0]
	w.Step([]Command{{Player: 0, Kind: CmdDevSpawnBuilding, Name: "farm", Target: keep.Pos.Add(fixed.V2i(0, 30))},
		{Player: 0, Kind: CmdDevSpawnBuilding, Name: "farm", Target: keep.Pos.Add(fixed.V2i(0, -30))},
		{Player: 0, Kind: CmdDevSpawnBuilding, Name: "farm", Target: keep.Pos.Add(fixed.V2i(-30, 0))}})
	gold := res(t, w, "gold")
	before := w.Stock(0, gold)
	w.Step([]Command{{Player: 0, Kind: CmdTrain, TargetBuilding: keep.ID, Name: "builders"}})
	if w.Stock(0, gold) != before-50 {
		t.Fatalf("builder cost: %d → %d", before, w.Stock(0, gold))
	}
	w.Step([]Command{{Player: 0, Kind: CmdCancelTrain, TargetBuilding: keep.ID, Count: 0}})
	if w.Stock(0, gold) != before || keep.Queue() != nil {
		t.Fatal("cancel did not refund")
	}
}
