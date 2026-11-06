package sim

import (
	"testing"

	"dnd-rts/internal/data"
	"dnd-rts/internal/fixed"
)

// baseSpot: the k-th place inside player 0's base wall (rings around the keep) where a building of
// type id fits.
func baseSpot(t *testing.T, w *World, id string, k int) fixed.Vec2 {
	t.Helper()
	keep := fortressOf(w, 0)
	def := w.Catalog.Buildings[id]
	for r := 16; r <= 60; r += 4 {
		for a := 0; a < 16; a++ {
			p := keep.Pos.Add(fixed.FromAngle(fixed.TwoPi.MulInt(a).DivInt(16)).Scale(fixed.FromInt(r)))
			if w.fitsFree(def, p, 0, true) {
				if k == 0 {
					return p
				}
				k--
			}
		}
	}
	t.Fatalf("no room for %s inside the base", id)
	return fixed.Vec2{}
}

// workersOf: player 0's builder horde.
func workersOf(t *testing.T, w *World) *Horde {
	t.Helper()
	for _, h := range w.Hordes {
		if h.Owner == 0 && h.IsWorker() {
			return h
		}
	}
	t.Fatal("no workers")
	return nil
}

// wallSegment: the k-th standing wall segment of player 0.
func wallSegment(t *testing.T, w *World, k int) *Building {
	t.Helper()
	for _, b := range w.Buildings {
		if b.Owner == 0 && b.Def.Kind == data.BuildingWall {
			if k == 0 {
				return b
			}
			k--
		}
	}
	t.Fatal("no wall segment")
	return nil
}

// towerOnWall has workers build a tower of type id on player 0's k-th wall segment and finishes it.
func towerOnWall(t *testing.T, w *World, id string, k int) *Building {
	t.Helper()
	w.Step([]Command{{Player: 0, Kind: CmdDevGive, Name: "gold", Count: 9000}, {Player: 0, Kind: CmdDevGive, Name: "wood", Count: 9000},
		{Player: 0, Kind: CmdDevGive, Name: "metal", Count: 9000}})
	def := w.Catalog.Buildings[id]
	for tries := 0; tries < 40; tries, k = tries+1, k+3 {
		seg := wallSegment(t, w, k)
		if _, _, ok := w.towerSpotAny(0, def, seg.Pos); !ok {
			continue
		}
		free := *def
		free.Requires = nil
		w.Catalog.Buildings[id] = &free // no tech needed for the test
		w.Step([]Command{{Player: 0, Kind: CmdBuildFree, Hordes: []HordeID{workersOf(t, w).ID}, Name: id, Target: seg.Pos}})
		w.Catalog.Buildings[id] = def
		b := w.Buildings[len(w.Buildings)-1]
		if b.Def.ID != id || !b.OnWall() {
			t.Fatalf("no %s on the wall", id)
		}
		b.buildTicks, b.HP = 0, b.Def.Health
		return b
	}
	t.Fatal("no free place on the wall")
	return nil
}

func TestTowerGoesOnTheWall(t *testing.T) {
	w := newWalledWorld(t)
	def := w.Catalog.Buildings["archer_tower"]
	seg := wallSegment(t, w, 5)
	// A click a little inside the wall snaps onto its centre line, facing out.
	click := seg.Pos.Sub(fixed.FromAngle(seg.Facing).Scale(fixed.FromInt(3))).Add(fixed.V2i(1, 0))
	spot, facing, ok := w.TowerSpot(0, def, click)
	if !ok {
		t.Fatal("no tower spot next to the wall")
	}
	if seg.DistTo(spot) > 0 || facing != seg.Facing {
		t.Errorf("spot %v (facing %v) is not on segment %v (facing %v)", spot, facing, seg.Pos, seg.Facing)
	}
	// Not on open ground, not on the enemy's wall, not by a gate.
	if _, _, ok := w.TowerSpot(0, def, fortressOf(w, 0).Pos.Add(fixed.V2i(25, 0))); ok {
		t.Error("a tower spot on open ground")
	}
	enemySeg := (*Building)(nil)
	for _, b := range w.Buildings {
		if b.Owner == 1 && b.Def.Kind == data.BuildingWall {
			enemySeg = b
			break
		}
	}
	if _, _, ok := w.TowerSpot(0, def, enemySeg.Pos); ok {
		t.Error("a tower spot on the enemy's wall")
	}
	for _, b := range w.Buildings {
		if b.Owner == 0 && b.Def.Kind == data.BuildingGate {
			if _, _, ok := w.TowerSpot(0, def, b.Pos); ok {
				t.Error("a tower spot on a gatehouse")
			}
		}
	}
	// Workers put it up; a second tower can't stand on top of it; a ground building can't go on the wall.
	tower := towerOnWall(t, w, "archer_tower", 5)
	if _, _, ok := w.towerSpotAny(0, def, tower.Pos); ok {
		t.Error("two towers in one place")
	}
	if w.CanPlaceFree(0, w.Catalog.Buildings["farm"], seg.Pos, 0) {
		t.Error("a farm on the wall")
	}
}

func TestWorkersBuildTheTower(t *testing.T) {
	w := newWalledWorld(t)
	w.Step([]Command{{Player: 0, Kind: CmdDevGive, Name: "gold", Count: 5000}, {Player: 0, Kind: CmdDevGive, Name: "wood", Count: 5000}})
	seg := wallSegment(t, w, 8)
	workers := workersOf(t, w)
	w.Step([]Command{{Player: 0, Kind: CmdBuildFree, Hordes: []HordeID{workers.ID}, Name: "archer_tower", Target: seg.Pos}})
	tower := w.Buildings[len(w.Buildings)-1]
	if tower.Def.ID != "archer_tower" || !tower.UnderConstruction() {
		t.Fatal("no tower foundation on the wall")
	}
	for i := 0; i < 20*120 && tower.UnderConstruction(); i++ {
		w.Step(nil)
	}
	if tower.UnderConstruction() {
		t.Fatalf("workers did not finish the tower (workers at %v, tower at %v)", workers.Pos, tower.Pos)
	}
}

// Nothing builds itself any more: a foundation without workers stays a foundation.
func TestNoSelfBuilding(t *testing.T) {
	w := newWalledWorld(t)
	b, err := w.SpawnBuilding(0, "barracks", baseSpot(t, w, "barracks", 0), 0)
	if err != nil {
		t.Fatal(err)
	}
	b.startConstruction()
	for i := 0; i < 20*10; i++ {
		w.Step(nil)
	}
	if b.Progress() > 0 {
		t.Error("a foundation built itself")
	}
}
