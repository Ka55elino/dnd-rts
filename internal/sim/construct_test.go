package sim

import (
	"testing"

	"dnd-rts/internal/fixed"
)

func builder(t *testing.T, w *World, owner PlayerID) *Horde {
	t.Helper()
	for _, h := range w.Hordes {
		if h.Owner == owner && h.IsWorker() {
			return h
		}
	}
	t.Fatal("no builder")
	return nil
}

// A free spot inside player 0's build zone.
func freeSpot(t *testing.T, w *World) fixed.Vec2 {
	t.Helper()
	keep := w.Buildings[0]
	farm := w.Catalog.Buildings["farm"]
	for r := 85; r <= 140; r += 5 { // outside the base wall
		for a := 0; a < 12; a++ {
			p := keep.Pos.Add(fixed.FromAngle(fixed.TwoPi.MulInt(a).DivInt(12)).Scale(fixed.FromInt(r)))
			if w.CanPlaceFree(0, farm, p, 0) {
				return p
			}
		}
	}
	t.Fatal("no free spot in the build zone")
	return fixed.Vec2{}
}

func TestWorkerBuildsFreeBuilding(t *testing.T) {
	w := newTestWorld(t)
	h := builder(t, w, 0)
	spot := freeSpot(t, w)
	n := len(w.Buildings)
	w.Step([]Command{{Player: 0, Kind: CmdBuildFree, Hordes: []HordeID{h.ID}, Name: "farm", Target: spot}})
	if len(w.Buildings) != n+1 {
		t.Fatal("foundation not placed")
	}
	farm := w.Buildings[len(w.Buildings)-1]
	if h.ConstructTarget() != farm || !farm.UnderConstruction() {
		t.Fatal("worker not assigned / farm not a foundation")
	}
	took := -1
	for i := 0; i < 20*60; i++ {
		w.Step(nil)
		if !farm.UnderConstruction() {
			took = i
			break
		}
	}
	if took < 0 {
		t.Fatalf("farm not finished in 60s (progress %v)", farm.Progress())
	}
	if farm.HP != farm.Def.Health || h.ConstructTarget() != nil {
		t.Errorf("finished farm hp %v, worker still on it: %v", farm.HP, h.ConstructTarget() != nil)
	}
	t.Logf("built in %.1fs (build time %vs + walking)", float64(took)/TickRate, farm.Def.BuildTime)
}

func TestFoundationWaitsForWorkers(t *testing.T) {
	w := newTestWorld(t)
	h := builder(t, w, 0)
	spot := freeSpot(t, w)
	w.Step([]Command{{Player: 0, Kind: CmdBuildFree, Hordes: []HordeID{h.ID}, Name: "farm", Target: spot}})
	farm := w.Buildings[len(w.Buildings)-1]
	w.Step([]Command{{Player: 0, Kind: CmdMove, Hordes: []HordeID{h.ID}, Target: w.Buildings[0].Pos.Add(fixed.V2i(0, -40))}})
	before := farm.Progress()
	for i := 0; i < 100; i++ {
		w.Step(nil)
	}
	if farm.Progress() != before {
		t.Errorf("progress moved without workers: %v → %v", before, farm.Progress())
	}
	// Send it back to help.
	w.Step([]Command{{Player: 0, Kind: CmdConstruct, Hordes: []HordeID{h.ID}, TargetBuilding: farm.ID}})
	for i := 0; i < 20*40 && farm.UnderConstruction(); i++ {
		w.Step(nil)
	}
	if farm.UnderConstruction() {
		t.Error("construct order did not finish the farm")
	}
}

func TestFreePlacementRules(t *testing.T) {
	w := newTestWorld(t)
	keep := w.Buildings[0]
	farm := w.Catalog.Buildings["farm"]
	barracks := w.Catalog.Buildings["forge"] // plot only
	if w.CanPlaceFree(0, farm, keep.Pos, 0) {
		t.Error("placed on top of the keep")
	}
	if w.CanPlaceFree(0, farm, keep.PlotPos(0), 0) {
		t.Error("placed on a plot")
	}
	if w.CanPlaceFree(0, farm, keep.Local(fixed.V2i(0, 25)), 0) {
		t.Error("placed inside the keep's no-build circle")
	}
	if w.CanPlaceFree(0, barracks, freeSpot(t, w), 0) {
		t.Error("plot-only building placed freely")
	}
	// Farms keep their distance from each other.
	spot := freeSpot(t, w)
	if _, err := w.SpawnBuilding(0, "farm", spot, 0); err != nil {
		t.Fatal(err)
	}
	if w.CanPlaceFree(0, farm, spot.Add(fixed.V2i(14, 0)), 0) {
		t.Error("farm placed inside another farm's no-build circle")
	}
	// Non-workers can't build.
	soldiers := w.Hordes[0]
	n := len(w.Buildings)
	w.Step([]Command{{Player: 0, Kind: CmdBuildFree, Hordes: []HordeID{soldiers.ID}, Name: "farm", Target: freeSpot(t, w)}})
	if !soldiers.IsWorker() && len(w.Buildings) != n {
		t.Error("swordsmen laid a foundation")
	}
}

func TestRectOverlap(t *testing.T) {
	a := rect{center: fixed.V2i(0, 0), half: fixed.V2i(2, 2)}
	b := rect{center: fixed.V2i(3, 0), half: fixed.V2i(2, 2)}
	c := rect{center: fixed.V2i(5, 0), half: fixed.V2i(1, 1)}
	rot := rect{center: fixed.V2i(4, 4), facing: fixed.Pi.DivInt(4), half: fixed.V2i(1, 1)}
	if !a.overlaps(b) || a.overlaps(c) || a.overlaps(rot) {
		t.Errorf("overlaps: ab=%v ac=%v arot=%v", a.overlaps(b), a.overlaps(c), a.overlaps(rot))
	}
}
