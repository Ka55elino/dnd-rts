package sim

import (
	"testing"

	"dnd-rts/internal/data"
	"dnd-rts/internal/fixed"
)

// target spawns a single-soldier test horde and returns its soldier, with the
// unit definition replaced by a copy the test may edit.
func target(t *testing.T, w *World, owner PlayerID, pos fixed.Vec2) *Soldier {
	t.Helper()
	h, err := w.SpawnHorde(owner, "builders", pos, 0)
	if err != nil {
		t.Fatal(err)
	}
	u := *h.Unit
	u.Armor, u.MagicDefense, u.ArmorClass, u.Resist, u.Immune = 0, 0, data.ArmorMedium, nil, nil
	u.Health = fixed.FromInt(1000)
	h.Unit = &u
	s := h.Soldiers[0]
	s.HP = u.Health
	return s
}

func approx(a, b fixed.Fixed) bool { return (a - b).Abs() <= fixed.FromFraction(1, 100) }

func TestPhysicalArmorAndHeavy(t *testing.T) {
	w := newTestWorld(t)
	s := target(t, w, 1, fixed.V2i(0, 40))
	s.Horde.Unit.Armor = fixed.FromInt(4)
	before := s.HP
	w.hitSoldier(nil, 0, s, fixed.FromInt(20), data.DmgSlash, true)
	if got := before - s.HP; got != fixed.FromInt(16) {
		t.Errorf("slash 20 vs armor 4 = %v, want 16", got)
	}
	s.Horde.Unit.ArmorClass = data.ArmorHeavy
	before = s.HP
	w.hitSoldier(nil, 0, s, fixed.FromInt(20), data.DmgSlash, true)
	if got := before - s.HP; !approx(got, fixed.FromFraction(1040, 100)) { // 16 × 0.65
		t.Errorf("heavy armor: %v, want 10.4", got)
	}
	// Magic ignores armor and heavy resist, uses magic defense.
	s.Horde.Unit.MagicDefense = fixed.FromInt(5)
	before = s.HP
	w.hitSoldier(nil, 0, s, fixed.FromInt(20), data.DmgAcid, false)
	if got := before - s.HP; got != fixed.FromInt(15) {
		t.Errorf("acid 20 vs magic defense 5 = %v, want 15", got)
	}
}

func TestRadiantIgnoresEverythingAndSparesBuildings(t *testing.T) {
	w := newTestWorld(t)
	s := target(t, w, 1, fixed.V2i(0, 40))
	u := s.Horde.Unit
	u.Armor, u.MagicDefense, u.ArmorClass = fixed.FromInt(50), fixed.FromInt(50), data.ArmorHeavy
	u.Resist = map[string]fixed.Fixed{data.DmgRadiant: fixed.One}
	before := s.HP
	w.hitSoldier(nil, 0, s, fixed.FromInt(30), data.DmgRadiant, true)
	if before-s.HP != fixed.FromInt(30) {
		t.Errorf("radiant dealt %v, want 30", before-s.HP)
	}
	keep := fortressOf(w, 1)
	hp := keep.HP
	w.hitBuilding(0, keep, fixed.FromInt(100), data.DmgRadiant, true)
	w.hitBuilding(0, keep, fixed.FromInt(100), data.DmgLightning, true) // stone
	if keep.HP != hp {
		t.Errorf("radiant / lightning damaged a stone keep: %v → %v", hp, keep.HP)
	}
}

func TestCrushAndFireOnBuildings(t *testing.T) {
	w := newTestWorld(t)
	farm, _ := w.SpawnBuilding(1, "farm", fixed.V2i(0, 40), 0) // wood, armor 2
	hp := farm.HP
	w.hitBuilding(0, farm, fixed.FromInt(10), data.DmgCrush, true)
	if got := hp - farm.HP; got != fixed.FromInt(23) { // 10 × 2.5 − 2
		t.Errorf("crush on farm = %v, want 23", got)
	}
	w.hitBuilding(0, farm, fixed.FromInt(10), data.DmgFire, true)
	if farm.burnTicks == 0 {
		t.Fatal("fire did not set the farm burning")
	}
	hp = farm.HP
	for i := 0; i < 20*3; i++ {
		w.Step(nil)
	}
	if farm.HP >= hp || farm.burnTicks != 0 {
		t.Errorf("burning: hp %v → %v, ticks left %d", hp, farm.HP, farm.burnTicks)
	}
}

func TestBurnSlowStunAcid(t *testing.T) {
	w := newTestWorld(t)
	s := target(t, w, 1, fixed.V2i(0, 40))
	w.hitSoldier(nil, 0, s, fixed.FromInt(20), data.DmgFire, true)
	w.hitSoldier(nil, 0, s, fixed.FromInt(1), data.DmgIce, true)
	w.hitSoldier(nil, 0, s, fixed.FromInt(1), data.DmgAcid, true)
	w.hitSoldier(nil, 0, s, fixed.FromInt(1), data.DmgAcid, true)
	f := s.StatusFlags()
	if f&StatusBurning == 0 || f&StatusSlowed == 0 || f&StatusAcid == 0 || s.fx.acidStacks != 2 {
		t.Fatalf("flags %b, acid stacks %d", f, s.fx.acidStacks)
	}
	if got := s.slowFactor(w); !approx(got, fixed.FromFraction(7, 10)) {
		t.Errorf("slow factor %v, want 0.7", got)
	}
	hp := s.HP
	for i := 0; i < 20*6+1; i++ {
		w.Step(nil)
	}
	if s.HP >= hp {
		t.Error("burning dealt no damage")
	}
	if s.StatusFlags() != 0 {
		t.Errorf("effects did not expire: %b", s.StatusFlags())
	}
	// Immunity.
	s.Horde.Unit.Immune = []string{data.EffectBurn}
	w.hitSoldier(nil, 0, s, fixed.FromInt(20), data.DmgFire, true)
	if s.fx.burnTicks != 0 {
		t.Error("immune unit started burning")
	}
	// Stun eventually (15% per hit, deterministic rolls).
	stunned := false
	for i := 0; i < 60 && !stunned; i++ {
		w.Step(nil)
		w.hitSoldier(nil, 0, s, fixed.One, data.DmgLightning, true)
		stunned = s.Stunned()
	}
	if !stunned {
		t.Error("60 lightning hits never stunned")
	}
}

func TestPoisonSpreadsAtMostTwice(t *testing.T) {
	w := newTestWorld(t)
	first := target(t, w, 1, fixed.V2i(0, 40))
	var others []*Soldier
	for i := 1; i <= 5; i++ {
		others = append(others, target(t, w, 1, fixed.V2i(i, 40)))
	}
	w.combatHash.build(w)
	w.hitSoldier(nil, 0, first, fixed.FromInt(20), data.DmgPoison, true)
	for i := 0; i < 20*5; i++ {
		w.Step(nil)
		for _, s := range append(others, first) { // keep them in place and alive
			s.HP = s.Horde.Unit.Health
		}
	}
	infected := 0
	for _, s := range others {
		if s.fx.poisonTicks > 0 || s.fx.poisonDamage > 0 {
			infected++
		}
	}
	if infected > 2 {
		t.Errorf("one poisoning infected %d others, max 2", infected)
	}
}

func TestNecroticLifeSteal(t *testing.T) {
	w := newTestWorld(t)
	attacker := target(t, w, 0, fixed.V2i(0, 40))
	attacker.HP = fixed.FromInt(500)
	victim := target(t, w, 1, fixed.V2i(1, 40))
	w.hitSoldier(attacker, 0, victim, fixed.FromInt(100), data.DmgNecrotic, true)
	if !approx(attacker.HP, fixed.FromInt(530)) {
		t.Errorf("attacker hp %v, want 530 (30%% of 100)", attacker.HP)
	}
}

func TestBonusVsTag(t *testing.T) {
	a := &data.Attack{Bonus: map[string]fixed.Fixed{"cavalry": fixed.FromFraction(3, 2)}}
	cav := &data.Unit{Tags: []string{"cavalry"}}
	inf := &data.Unit{Tags: []string{"infantry"}}
	if attackBonus(a, cav, fixed.FromInt(10)) != fixed.FromInt(15) || attackBonus(a, inf, fixed.FromInt(10)) != fixed.FromInt(10) {
		t.Error("tag bonus wrong")
	}
}
