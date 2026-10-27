package sim

import (
	"testing"

	"dnd-rts/internal/fixed"
)

func TestFlyersCrossBlockedGround(t *testing.T) {
	w := newTestWorld(t)
	keep := w.Buildings[0] // its footprint is blocked for walkers
	if w.Nav.Grid().WalkableAt(keep.Pos) {
		t.Fatal("test needs a blocked spot")
	}
	start := openSpot(w, int(keep.Pos.X.Int())+40, int(keep.Pos.Y.Int()))
	h, _ := w.SpawnHorde(1, "harpies", start, 0)
	w.Step([]Command{{Player: 1, Kind: CmdMove, Hordes: []HordeID{h.ID}, Target: keep.Pos}})
	for i := 0; i < 20*15 && h.Pos != keep.Pos; i++ {
		w.Step(nil)
	}
	if h.Pos != keep.Pos {
		t.Fatalf("harpies stopped at %v, want the keep centre %v", h.Pos, keep.Pos)
	}
	over := 0
	for _, s := range h.Soldiers {
		if !w.Nav.Grid().WalkableAt(s.Pos) {
			over++
		}
	}
	if over == 0 {
		t.Error("no harpy hovers over the keep")
	}
}

func TestGroundMeleeCantHitFlyersUntilGrounded(t *testing.T) {
	w := newTestWorld(t)
	p := openSpot(w, -60, 60)
	harpies, _ := w.SpawnHorde(1, "harpies", p, 0)
	swords, _ := w.SpawnHorde(0, "swordsmen", p.Add(fixed.V2i(3, 0)), 0)
	if swords.canTarget(harpies) || !harpies.canTarget(swords) {
		t.Fatal("swordsmen should not target flyers; harpies hit the ground")
	}
	// Flyers and walkers don't push each other.
	a, b := harpies.Soldiers[0], swords.Soldiers[0]
	a.Pos, b.Pos = p, p
	resolve(a, b)
	if a.Pos != p || b.Pos != p {
		t.Error("air and ground soldiers separated")
	}
	// Pulled down: ground attacks can hit it for a while.
	w.applyStatus(harpies.Soldiers[0], 0, "ground", 40)
	if harpies.Airborne() || !swords.canTarget(harpies) {
		t.Fatal("grounded harpies still airborne")
	}
	if harpies.Soldiers[0].StatusFlags()&StatusGrounded == 0 {
		t.Error("grounded flag missing")
	}
	for i := 0; i < 41; i++ {
		w.Step(nil)
	}
	if !harpies.Airborne() {
		t.Error("harpies did not take off again")
	}
}

func TestArchersShootFlyers(t *testing.T) {
	w := newTestWorld(t)
	p := openSpot(w, -60, 60)
	archers, _ := w.SpawnHorde(0, "archers", p, 0)
	harpies, _ := w.SpawnHorde(1, "harpies", p.Add(fixed.V2i(20, 0)), 0)
	harpies.ordered = true // sit still
	hp := totalHP(harpies)
	air := false
	for i := 0; i < 20*6; i++ {
		harpies.ordered, harpies.attack = true, nil
		w.Step(nil)
		for _, pr := range w.Projectiles {
			air = air || pr.ToAir
		}
	}
	if totalHP(harpies) >= hp {
		t.Error("archers did not hurt the harpies")
	}
	if !air {
		t.Error("volleys at flyers should be marked ToAir")
	}
	_ = archers
}
