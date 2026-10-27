package sim

import (
	"testing"

	"dnd-rts/internal/fixed"
)

// open ground in the middle of plains, away from bases and outposts.
func openSpot(w *World, x, y int) fixed.Vec2 {
	p, _ := w.walkableTarget(fixed.V2i(x, y))
	return p
}

func TestStopOrder(t *testing.T) {
	w := newTestWorld(t)
	h, _ := w.SpawnHorde(0, "swordsmen", openSpot(w, -60, 40), 0)
	w.Step([]Command{{Player: 0, Kind: CmdMove, Hordes: []HordeID{h.ID}, Target: openSpot(w, 40, 40)}})
	for i := 0; i < 20; i++ {
		w.Step(nil)
	}
	w.Step([]Command{{Player: 0, Kind: CmdStop, Hordes: []HordeID{h.ID}}})
	at := h.Pos
	for i := 0; i < 40; i++ {
		w.Step(nil)
	}
	if h.IsMoving() || h.Pos.Dist(at) > fixed.One {
		t.Fatalf("still moving after stop: %v → %v", at, h.Pos)
	}
}

func TestHoldDoesNotChase(t *testing.T) {
	w := newTestWorld(t)
	base := openSpot(w, -40, 60)
	h, _ := w.SpawnHorde(0, "swordsmen", base, 0)
	w.Step([]Command{{Player: 0, Kind: CmdHold, Hordes: []HordeID{h.ID}}})
	// An enemy inside normal aggro range but outside melee reach.
	w.SpawnHorde(1, "raiders", base.Add(fixed.V2i(16, 0)), 0)
	for i := 0; i < 20*5; i++ {
		w.Step(nil)
	}
	if !h.IsHolding() || h.Pos.Dist(base) > fixed.FromInt(2) {
		t.Fatalf("holding horde moved %v (holding %v)", h.Pos.Dist(base), h.IsHolding())
	}
}

func TestAttackMoveFightsOnTheWay(t *testing.T) {
	w := newTestWorld(t)
	start := openSpot(w, -60, 60)
	h, _ := w.SpawnHorde(0, "swordsmen", start, 0)
	enemy, _ := w.SpawnHorde(1, "raiders", openSpot(w, -30, 60), 0)
	dest := openSpot(w, 20, 60)
	w.Step([]Command{{Player: 0, Kind: CmdAttackMove, Hordes: []HordeID{h.ID}, Target: dest}})
	engaged := false
	for i := 0; i < 20*60 && h.Alive(); i++ {
		w.Step(nil)
		if h.AttackTarget() == enemy {
			engaged = true
		}
		if !enemy.Alive() && h.Pos.Dist(dest) < fixed.FromInt(4) {
			break
		}
	}
	if !engaged {
		t.Fatal("attack-move walked past the enemy")
	}
	if h.Alive() && !enemy.Alive() && h.Pos.Dist(dest) > fixed.FromInt(4) {
		t.Errorf("did not continue to the destination after the fight (at %v)", h.Pos)
	}
}

func TestPatrolTurnsAround(t *testing.T) {
	w := newTestWorld(t)
	a := openSpot(w, -60, 60)
	b := openSpot(w, -30, 60)
	h, _ := w.SpawnHorde(0, "swordsmen", a, 0)
	w.Step([]Command{{Player: 0, Kind: CmdPatrol, Hordes: []HordeID{h.ID}, Target: b}})
	reachedB, backAtA := false, false
	for i := 0; i < 20*60; i++ {
		w.Step(nil)
		if h.Pos.Dist(b) < fixed.FromInt(2) {
			reachedB = true
		}
		if reachedB && h.Pos.Dist(a) < fixed.FromInt(2) {
			backAtA = true
			break
		}
	}
	if !reachedB || !backAtA {
		t.Fatalf("patrol: reached B %v, back at A %v", reachedB, backAtA)
	}
	if !h.IsMoving() {
		t.Error("patrol stopped instead of turning around")
	}
}
