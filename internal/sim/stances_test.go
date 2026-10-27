package sim

import (
	"testing"

	"dnd-rts/internal/fixed"
)

func toggle(w *World, h *Horde, id string) {
	w.Step([]Command{{Player: h.Owner, Kind: CmdToggle, Hordes: []HordeID{h.ID}, Name: id}})
	w.Step(nil) // modifiers apply from the next tick
}

func TestSpearWallStandsItsGround(t *testing.T) {
	w := newTestWorld(t)
	sp, _ := w.SpawnHorde(0, "spearmen", openSpot(w, 0, 60), 0)
	toggle(w, sp, "spear_wall")
	if !sp.Rooted() {
		t.Fatal("spear wall did not root the horde")
	}
	at := sp.Pos
	w.Step([]Command{{Player: 0, Kind: CmdMove, Hordes: []HordeID{sp.ID}, Target: at.Add(fixed.V2i(20, 0))}})
	for i := 0; i < 40; i++ {
		w.Step(nil)
	}
	if sp.Pos != at {
		t.Errorf("rooted spearmen moved: %v → %v", at, sp.Pos)
	}
}

// A knight charge into a spear wall is broken and hurts the knights; into
// spearmen without the wall it tramples them.
func TestSpearWallBreaksTheCharge(t *testing.T) {
	charge := func(wall bool) (knightLoss, spearLoss fixed.Fixed, knocked int) {
		w := newTestWorld(t)
		p := openSpot(w, 0, 60)
		sp, _ := w.SpawnHorde(0, "spearmen", p, 0)
		if wall {
			toggle(w, sp, "spear_wall")
		}
		kn, _ := w.SpawnHorde(1, "knights", p.Add(fixed.V2i(-35, 0)), 0)
		k0, s0 := totalHP(kn), totalHP(sp)
		w.Step([]Command{{Player: 1, Kind: CmdAttack, Hordes: []HordeID{kn.ID}, TargetHorde: sp.ID}})
		for i := 0; i < 20*8; i++ {
			w.Step(nil)
			for _, s := range sp.Soldiers {
				if s.Knocked() {
					knocked++
				}
			}
		}
		return k0 - totalHP(kn), s0 - totalHP(sp), knocked
	}
	kWall, _, knWall := charge(true)
	kOpen, _, knOpen := charge(false)
	if knWall != 0 {
		t.Errorf("spearmen in a spear wall got trampled (%d knock-down ticks)", knWall)
	}
	if knOpen == 0 {
		t.Error("the control charge trampled nobody (test setup?)")
	}
	if kWall <= kOpen {
		t.Errorf("knights lost %v against the wall, %v against open spearmen: the wall should hurt more", kWall, kOpen)
	}
}

func TestWedgeFormation(t *testing.T) {
	w := newTestWorld(t)
	kn, _ := w.SpawnHorde(0, "knights", openSpot(w, 0, 60), 0)
	base := kn.Soldiers[0].slowFactor(w)
	toggle(w, kn, "wedge")
	fwd := fixed.FromAngle(kn.Facing)
	front := func(i int) fixed.Fixed { return kn.slotPos(i).Sub(kn.Pos).Dot(fwd) }
	if front(0) <= front(1) || front(1) != front(2) || front(2) <= front(3) {
		t.Errorf("not a wedge: depths %v %v %v %v", front(0), front(1), front(2), front(3))
	}
	if kn.Soldiers[0].slowFactor(w) <= base {
		t.Error("the wedge is not faster")
	}
	if kn.Soldiers[0].mods.charge <= 0 {
		t.Error("no charge bonus in the wedge")
	}
}
