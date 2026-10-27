package sim

import (
	"testing"

	"dnd-rts/internal/data"
	"dnd-rts/internal/fixed"
)

func TestSetupPlacesBuildings(t *testing.T) {
	w := newTestWorld(t)
	m := w.Catalog.Maps["plains"]
	var forts, outposts, walls, gates int
	for _, b := range w.Buildings {
		switch b.Def.Kind {
		case "fortress":
			forts++
		case "outpost":
			outposts++
			if b.Owner != Neutral {
				t.Errorf("outpost %d owned by %d, want neutral", b.ID, b.Owner)
			}
		case "wall":
			walls++
		case "gate":
			gates++
		}
	}
	if forts != 2 || outposts != len(m.Outposts) {
		t.Fatalf("fortresses %d, outposts %d", forts, outposts)
	}
	if len(w.Walls) != 2 || gates != 4 || walls+gates != 2*len(w.Walls[0].Nodes) {
		t.Errorf("walls: rings %d, wall segments %d, gate segments %d", len(w.Walls), walls, gates)
	}
	if b := w.Buildings[0]; b.Owner != 0 || b.Def.Kind != "fortress" {
		t.Errorf("first building: owner %d kind %q, want player 0's fortress", b.Owner, b.Def.Kind)
	}
}

func TestBuildingBlocksNav(t *testing.T) {
	w := newTestWorld(t)
	g := w.Nav.Grid()
	for _, b := range w.Buildings {
		if b.Def.Kind == "wall" || b.Def.Kind == "gate" {
			continue // segments next to each other; see the wall tests
		}
		if g.WalkableAt(b.Pos) {
			t.Errorf("%s %d: center is still walkable", b.Def.ID, b.ID)
		}
		// Just outside the footprint is free (the maps keep buildings on open ground).
		if p := b.Local(fixed.Vec2{Y: b.Def.Footprint.Y.DivInt(2) + NavCellSize.MulInt(2)}); !g.WalkableAt(p) {
			t.Errorf("%s %d: ground in front of it is blocked", b.Def.ID, b.ID)
		}
	}
}

// Every plot of every map building must be on open ground and clear of other buildings.
func TestPlotsAreBuildable(t *testing.T) {
	w := newTestWorld(t)
	g := w.Nav.Grid()
	for _, b := range w.Buildings {
		for i, plot := range b.Def.Plots {
			if plot.Class == data.PlotTower {
				continue // tower sockets sit on the wall
			}
			half := plot.Size.DivInt(2)
			for _, c := range []fixed.Vec2{{}, {X: -half, Y: -half}, {X: half, Y: -half}, {X: -half, Y: half}, {X: half, Y: half}} {
				q := b.Local(plot.Offset.Add(c))
				if !g.WalkableAt(q) {
					t.Errorf("%s %d plot %d: %v is blocked", b.Def.ID, b.ID, i, q)
				}
			}
		}
	}
}

func TestBuildingLocalAxes(t *testing.T) {
	w := newTestWorld(t)
	b, err := w.SpawnBuilding(0, "farm", fixed.V2i(0, 40), 0) // front faces +x
	if err != nil {
		t.Fatal(err)
	}
	if got := b.Local(fixed.V2i(0, 10)); got.Dist(fixed.V2i(10, 40)) > fixed.FromFraction(1, 100) {
		t.Errorf("front offset → %v, want (10, 40)", got)
	}
	if got := b.Local(fixed.V2i(10, 0)); got.Dist(fixed.V2i(0, 30)) > fixed.FromFraction(1, 100) {
		t.Errorf("right offset → %v, want (0, 30)", got)
	}
	if w.Building(b.ID) != b {
		t.Error("Building(id) lookup failed")
	}
}
