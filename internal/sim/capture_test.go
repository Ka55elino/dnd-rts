package sim

import (
	"testing"

	"dnd-rts/internal/fixed"
)

func outpost(t *testing.T, w *World, id string) *Building {
	t.Helper()
	for _, b := range w.Buildings {
		if b.Def.ID == id {
			return b
		}
	}
	t.Fatalf("no %s on the map", id)
	return nil
}

// near returns a walkable point just outside the outpost's footprint.
func near(w *World, b *Building) fixed.Vec2 {
	p, _ := w.walkableTarget(b.approach(fixed.Vec2{}, fixed.FromInt(5)))
	return p
}

func TestCaptureOutpost(t *testing.T) {
	w := newTestWorld(t)
	mine := outpost(t, w, "gold_mine")
	if _, err := w.SpawnHorde(0, "swordsmen", near(w, mine), 0); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 20*12 && mine.Owner == Neutral; i++ {
		w.Step(nil)
		if i == 20*5 {
			if by, p := mine.Capturing(); by != 0 || p <= 0 {
				t.Fatalf("after 5s: capturing by %d, progress %v", by, p)
			}
		}
	}
	if mine.Owner != 0 {
		t.Fatalf("not captured after 12s (owner %d)", mine.Owner)
	}
	if !mine.Attackable(1) {
		t.Error("captured outpost should be attackable by the enemy")
	}

	// Destroyed → neutral again, full health, still standing.
	mine.HP = 0
	w.Step(nil)
	if w.Building(mine.ID) == nil || mine.Owner != Neutral || mine.HP != mine.Def.Health {
		t.Fatalf("after destruction: present=%v owner=%d hp=%v", w.Building(mine.ID) != nil, mine.Owner, mine.HP)
	}
}

func TestContestedCaptureHolds(t *testing.T) {
	w := newTestWorld(t)
	mill := outpost(t, w, "sawmill")
	p := near(w, mill)
	w.SpawnHorde(0, "swordsmen", p, 0)
	for i := 0; i < 20*3; i++ {
		w.Step(nil)
	}
	_, before := mill.Capturing()
	w.SpawnHorde(1, "raiders", p, 0) // they will fight; while both stand there, progress holds
	w.Step(nil)
	w.Step(nil)
	if _, after := mill.Capturing(); after != before && mill.Owner == Neutral {
		// One step of progress may slip in before the raiders are bucketed.
		if after-before > fixed.FromFraction(2, 100) {
			t.Errorf("contested progress moved: %v → %v", before, after)
		}
	}
}

func TestCaptureDrainsWhenLeft(t *testing.T) {
	w := newTestWorld(t)
	ore := outpost(t, w, "ore_mine")
	h, _ := w.SpawnHorde(0, "swordsmen", near(w, ore), 0)
	for i := 0; i < 20*4; i++ {
		w.Step(nil)
	}
	_, mid := ore.Capturing()
	w.Step([]Command{{Player: 0, Kind: CmdMove, Hordes: []HordeID{h.ID}, Target: w.Buildings[0].Pos.Add(fixed.V2i(0, 50))}})
	for i := 0; i < 20*30; i++ { // walk out of range, then the progress drains
		w.Step(nil)
	}
	if by, p := ore.Capturing(); ore.Owner != Neutral || by != Neutral || p != 0 {
		t.Errorf("after leaving: owner %d, capturing %d at %v (was %v)", ore.Owner, by, p, mid)
	}
}
