package sim

import (
	"testing"

	"dnd-rts/internal/data"
	"dnd-rts/internal/fixed"
)

// withUnit gives the horde its own copy of the unit definition to edit.
func withUnit(h *Horde, edit func(u *data.Unit)) {
	u := *h.Unit
	edit(&u)
	h.Unit = &u
}

func TestMeleeCantHitFlying(t *testing.T) {
	w := newTestWorld(t)
	foot, _ := w.SpawnHorde(0, "swordsmen", openSpot(w, -50, 60), 0)
	birds, _ := w.SpawnHorde(1, "raiders", openSpot(w, -40, 60), 0)
	withUnit(birds, func(u *data.Unit) { u.Flying = true })
	for i := 0; i < 20*5; i++ {
		w.Step(nil)
	}
	if foot.AttackTarget() == birds {
		t.Fatal("swordsmen targeted a flying horde")
	}
	w.Step([]Command{{Player: 0, Kind: CmdAttack, Hordes: []HordeID{foot.ID}, TargetHorde: birds.ID}})
	w.Step(nil)
	if foot.AttackTarget() != nil {
		t.Error("explicit attack on a flying horde accepted without anti-air")
	}
}

func TestAntiAirArchersHitFlying(t *testing.T) {
	w := newTestWorld(t)
	archers, _ := w.SpawnHorde(0, "archers", openSpot(w, -60, 60), 0) // ranged targets ground + air
	birds, _ := w.SpawnHorde(1, "raiders", openSpot(w, -40, 60), 0)
	withUnit(birds, func(u *data.Unit) { u.Flying = true })
	before := 0
	for _, s := range birds.Soldiers {
		before += s.HP.Int()
	}
	for i := 0; i < 20*10; i++ {
		w.Step(nil)
	}
	after := 0
	for _, s := range birds.Soldiers {
		after += s.HP.Int()
	}
	if archers.AttackTarget() != birds && after >= before {
		t.Fatalf("archers never shot the flying horde (hp %d → %d)", before, after)
	}
}

func TestBothAttacksSwitchByDistance(t *testing.T) {
	w := newTestWorld(t)
	h, _ := w.SpawnHorde(0, "swordsmen", openSpot(w, -60, 60), 0)
	withUnit(h, func(u *data.Unit) {
		u.Ranged = &data.Attack{
			Damage: fixed.FromInt(5), Range: fixed.FromInt(25), Cooldown: fixed.FromInt(2),
			Projectile: &data.Projectile{Speed: fixed.FromInt(30), Spread: fixed.FromInt(2)},
		}
	})
	far, _ := w.SpawnHorde(1, "raiders", h.Pos.Add(fixed.V2i(25, 0)), 0)
	if !h.selectMode(far, h.Pos.Dist(far.Pos)) || !h.Ranged() {
		t.Fatal("at 25 m a unit with both attacks should shoot")
	}
	if !h.selectMode(far, fixed.FromInt(5)) || !h.InMelee() {
		t.Fatal("at 5 m it should fight in melee")
	}
	// In play: it volleys first.
	h.melee = false
	shot := false
	for i := 0; i < 20*4 && !shot; i++ {
		w.Step(nil)
		shot = len(w.Projectiles) > 0
	}
	if !shot {
		t.Error("no volley from the ranged attack")
	}
}

func TestAirOnlyMeleeIgnoresGround(t *testing.T) {
	a := &data.Attack{Targets: []string{data.TargetAir}}
	if a.CanHit(false) || !a.CanHit(true) {
		t.Error("air-only attack")
	}
	var none *data.Attack
	if none.CanHit(false) {
		t.Error("nil attack hits")
	}
	if !(&data.Attack{}).CanHit(false) || (&data.Attack{}).CanHit(true) {
		t.Error("default targets should be ground only")
	}
}
