package sim

import (
	"testing"

	"dnd-rts/internal/fixed"
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
