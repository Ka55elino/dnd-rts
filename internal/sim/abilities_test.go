package sim

import (
	"testing"

	"dnd-rts/internal/fixed"
)

func soldierHP(h *Horde) fixed.Fixed { return totalHP(h) }

func TestToggleShieldWall(t *testing.T) {
	w := newTestWorld(t)
	h, _ := w.SpawnHorde(0, "swordsmen", openSpot(w, -50, 60), 0)
	w.Step(nil)
	base := h.Soldiers[0].slowFactor(w)
	w.Step([]Command{{Player: 0, Kind: CmdToggle, Hordes: []HordeID{h.ID}, Name: "shield_wall"}})
	w.Step(nil)
	s := h.Soldiers[0]
	if s.mods.armor != fixed.FromInt(3) || !approx(s.slowFactor(w), base.Mul(fixed.FromFraction(6, 10))) {
		t.Fatalf("shield wall: armor %v speed %v", s.mods.armor, s.slowFactor(w))
	}
	// Ranged damage taken −60%.
	hp := s.HP
	w.hitSoldierEx(nil, 1, s, fixed.FromInt(50), "slash", false, true)
	got := hp - s.HP
	if got > fixed.FromInt(20) {
		t.Errorf("ranged hit through shield wall: %v", got)
	}
	w.Step([]Command{{Player: 0, Kind: CmdToggle, Hordes: []HordeID{h.ID}, Name: "shield_wall"}})
	w.Step(nil)
	if h.Soldiers[0].mods.armor != 0 {
		t.Error("toggle off did not remove the modifiers")
	}
}

func TestAuraBuffsAlliesInRange(t *testing.T) {
	w := newTestWorld(t)
	hero, _ := w.SpawnHorde(0, "captain", openSpot(w, -50, 60), 0)
	hero.gainXP(1400) // all abilities unlocked
	nearby, _ := w.SpawnHorde(0, "swordsmen", hero.Pos.Add(fixed.V2i(5, 0)), 0)
	far, _ := w.SpawnHorde(0, "swordsmen", openSpot(w, 40, -60), 0)
	enemy, _ := w.SpawnHorde(1, "raiders", hero.Pos.Add(fixed.V2i(0, 6)), 0)
	w.Step(nil)
	if !approx(nearby.Soldiers[0].mods.damage, fixed.FromFraction(15, 100)) {
		t.Errorf("ally near the captain: damage mod %v, want 0.15", nearby.Soldiers[0].mods.damage)
	}
	if far.Soldiers[0].mods.damage != 0 || enemy.Soldiers[0].mods.damage != 0 {
		t.Error("aura reached a far ally or an enemy")
	}
}

func TestPointAbilityWalksIntoRangeThenCasts(t *testing.T) {
	w := newTestWorld(t)
	hero, _ := w.SpawnHorde(0, "captain", openSpot(w, -60, 60), 0)
	hero.gainXP(1400)
	enemy, _ := w.SpawnHorde(1, "raiders", openSpot(w, -10, 60), 0) // 50 m: out of 30 m range
	enemy.ordered = true
	hp := soldierHP(enemy)
	w.Step([]Command{{Player: 0, Kind: CmdCast, Hordes: []HordeID{hero.ID}, Name: "fire_pillar", Target: enemy.Pos}})
	if hero.pending == nil {
		t.Fatal("out of range: cast should wait while walking")
	}
	cast := false
	for i := 0; i < 20*20 && !cast; i++ {
		enemy.ordered, enemy.attack = true, nil
		w.Step(nil)
		cast = hero.AbilityStates()[hero.AbilityIndex("fire_pillar")].Cooldown > 0
	}
	if !cast {
		t.Fatal("pillar never cast")
	}
	if soldierHP(enemy) >= hp {
		t.Error("pillar dealt no damage")
	}
}

func TestStormZonePulses(t *testing.T) {
	w := newTestWorld(t)
	hero, _ := w.SpawnHorde(0, "captain", openSpot(w, -60, 60), 0)
	hero.gainXP(1400)
	enemy, _ := w.SpawnHorde(1, "raiders", hero.Pos.Add(fixed.V2i(15, 0)), 0)
	enemy.ordered = true
	w.Step([]Command{{Player: 0, Kind: CmdCast, Hordes: []HordeID{hero.ID}, Name: "storm", Target: enemy.Pos}})
	if len(w.Zones) != 1 {
		t.Fatalf("zones = %d, want 1", len(w.Zones))
	}
	hp := soldierHP(enemy)
	slowed := false
	for i := 0; i < 20*3; i++ {
		enemy.ordered, enemy.attack = true, nil
		w.Step(nil)
		for _, s := range enemy.Soldiers {
			if s.fx.slowTicks > 0 {
				slowed = true
			}
		}
	}
	if soldierHP(enemy) >= hp || !slowed {
		t.Errorf("storm: hp %v → %v, slowed %v", hp, soldierHP(enemy), slowed)
	}
	for i := 0; i < 20*5; i++ {
		w.Step(nil)
	}
	if len(w.Zones) != 0 {
		t.Error("zone did not expire")
	}
}

func TestAutocastHeal(t *testing.T) {
	w := newTestWorld(t)
	hero, _ := w.SpawnHorde(0, "captain", openSpot(w, -60, 60), 0)
	hero.gainXP(1400)
	hurt, _ := w.SpawnHorde(0, "swordsmen", hero.Pos.Add(fixed.V2i(8, 0)), 0)
	for _, s := range hurt.Soldiers {
		s.HP = fixed.FromInt(20)
	}
	w.Step([]Command{{Player: 0, Kind: CmdAutocast, Hordes: []HordeID{hero.ID}, Name: "heal", Count: 1}})
	w.Step(nil)
	if soldierHP(hurt) <= fixed.FromInt(20).MulInt(len(hurt.Soldiers)) {
		t.Error("autocast heal did not heal the wounded allies")
	}
	// Autocast can't be set on abilities that don't allow it.
	w.Step([]Command{{Player: 0, Kind: CmdAutocast, Hordes: []HordeID{hero.ID}, Name: "storm", Count: 1}})
	if hero.autocast[hero.AbilityIndex("storm")] {
		t.Error("autocast set on an ability without autocast")
	}
}
