package sim

import (
	"os"
	"testing"

	"dnd-rts/internal/data"
	"dnd-rts/internal/fixed"
	"dnd-rts/internal/terrain"
)

func TestWatchtowerGarrison(t *testing.T) {
	w := newTestWorld(t)
	p := openSpot(w, 0, 60)
	tower, err := w.SpawnBuilding(0, "watchtower", p, 0)
	if err != nil {
		t.Fatal(err)
	}
	archers, _ := w.SpawnHorde(0, "archers", p.Add(fixed.V2i(15, 0)), 0)
	swords, _ := w.SpawnHorde(0, "swordsmen", p.Add(fixed.V2i(-15, 0)), 0)
	knights, _ := w.SpawnHorde(0, "knights", p.Add(fixed.V2i(0, 15)), 0)
	w.Step([]Command{{Player: 0, Kind: CmdMove, Hordes: []HordeID{archers.ID}, Target: tower.Pos}})
	w.Step([]Command{{Player: 0, Kind: CmdMove, Hordes: []HordeID{swords.ID, knights.ID}, Target: tower.Pos}})
	for i := 0; i < 20*15 && !archers.OnWall(); i++ {
		w.Step(nil)
	}
	if !archers.OnWall() || archers.wall.tower == false {
		t.Fatal("archers did not climb the watchtower")
	}
	for i := 0; i < 20*3; i++ { // soldiers walk up to their places on the platform
		w.Step(nil)
	}
	if swords.OnWall() || swords.wallEnter != nil || knights.OnWall() {
		t.Error("a second horde (or cavalry) went up a one-horde tower")
	}
	for _, s := range archers.Soldiers {
		if s.Pos.Dist(tower.Pos) > fixed.FromInt(3) {
			t.Fatalf("archer %v from the tower centre: the platform is small", s.Pos.Dist(tower.Pos))
		}
	}
	if archers.attackRange(archers.Unit.Ranged) <= archers.Unit.Ranged.Range {
		t.Error("no range bonus on the tower")
	}
	raiders, _ := w.SpawnHorde(1, "raiders", p.Add(fixed.V2i(0, -10)), 0)
	if raiders.canTarget(archers) {
		t.Error("ground melee reaches the tower")
	}
	// The tower falls: the archers drop to the ground.
	tower.HP = 0
	w.Step(nil)
	w.Step(nil)
	if archers.OnWall() {
		t.Error("still up a destroyed tower")
	}
}

// Only foot soldiers go up a watchtower — every foot unit of every faction does, nobody else does
// (cavalry, flyers, monsters, machines, traps). The Playground's tower has another building right in
// front of its stair: the climb must still find its way.
func TestWatchtowerOnlyFootUnits(t *testing.T) {
	w := newPlaygroundWorld(t)
	var tower *Building
	for _, b := range w.Buildings {
		if b.Def.Garrison && b.Owner == 0 {
			tower = b
		}
	}
	if tower == nil {
		t.Fatal("no watchtower in the playground")
	}
	// Clear the field so nothing else holds the tower or picks fights.
	for _, h := range w.Hordes {
		for _, s := range h.Soldiers {
			s.HP = 0
		}
	}
	w.Step(nil)
	start, ok := w.walkableTarget(tower.Pos.Add(fixed.V2i(0, -24)))
	if !ok {
		t.Fatal("no open ground near the tower")
	}
	climbed, tried := 0, 0
	for _, id := range w.Catalog.HordeIDs {
		tried++
		def := w.Catalog.Hordes[id]
		u := w.Catalog.Units[def.Unit]
		if u.Trap != nil {
			continue
		}
		h, err := w.SpawnHorde(0, id, start, 0)
		if err != nil {
			t.Fatal(err)
		}
		w.Step([]Command{{Player: 0, Kind: CmdMove, Hordes: []HordeID{h.ID}, Target: tower.Pos}})
		for i := 0; i < 25*TickRate && !h.OnWall(); i++ {
			w.Step(nil)
		}
		foot := u.HasTag("infantry") && !u.HasTag("cavalry") && !u.Flying && !u.HasTag("mechanical")
		if h.OnWall() {
			climbed++
		}
		if h.OnWall() != foot {
			t.Errorf("%s (tags %v, flying %v): on the tower = %v, want %v", id, u.Tags, u.Flying, h.OnWall(), foot)
		}
		h.dismissed = true
		for _, s := range h.Soldiers {
			s.HP = 0
		}
		w.Step(nil)
		w.Step(nil)
	}
	t.Logf("%d of %d horde types climbed", climbed, tried)
}

func newPlaygroundWorld(t *testing.T) *World {
	t.Helper()
	cat, err := data.Load(os.DirFS("../../content"))
	if err != nil {
		t.Fatal(err)
	}
	hf, err := terrain.ForMap(os.DirFS("../../content"), cat.Maps["plains"])
	if err != nil {
		t.Fatal(err)
	}
	w := NewWorld(cat, hf)
	if err := SetupPlayground(w, "plains", []PlayerSetup{{0, "empire"}, {1, "warband"}}); err != nil {
		t.Fatal(err)
	}
	return w
}
