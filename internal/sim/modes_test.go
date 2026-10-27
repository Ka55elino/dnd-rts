package sim

import (
	"testing"

	"dnd-rts/internal/fixed"
)

func archmage(t *testing.T, w *World) *Horde {
	t.Helper()
	h, err := w.SpawnHorde(0, "archmage", openSpot(w, -60, 60), 0)
	if err != nil {
		t.Fatal(err)
	}
	h.gainXP(1400)
	return h
}

func setMode(w *World, h *Horde, mode int) {
	w.Step([]Command{{Player: h.Owner, Kind: CmdToggle, Hordes: []HordeID{h.ID}, Name: "element", Count: mode}})
}

func TestElementModeSwitchesAttackType(t *testing.T) {
	w := newTestWorld(t)
	h := archmage(t, w)
	w.Step(nil)
	if h.Mode() != "fire" || h.Soldiers[0].attackType(h.Unit.Ranged) != "fire" {
		t.Fatalf("default mode %q", h.Mode())
	}
	setMode(w, h, 1)
	w.Step(nil)
	if h.Mode() != "ice" || h.Soldiers[0].attackType(h.Unit.Ranged) != "ice" {
		t.Fatalf("after switch: mode %q attack %q", h.Mode(), h.Soldiers[0].attackType(h.Unit.Ranged))
	}
	// Switching has a cooldown.
	setMode(w, h, 2)
	if h.Mode() != "ice" {
		t.Error("switched again during the cooldown")
	}
	if st := h.AbilityStates()[0]; st.Mode != 1 || st.Cooldown == 0 {
		t.Errorf("state %+v", st)
	}
}

func TestElementalWardFollowsMode(t *testing.T) {
	w := newTestWorld(t)
	h := archmage(t, w)
	ally, _ := w.SpawnHorde(0, "swordsmen", h.Pos.Add(fixed.V2i(5, 0)), 0)
	w.Step(nil)
	s := ally.Soldiers[0]
	if s.mods.resistType != "fire" || !approx(s.mods.resist, fixed.FromFraction(3, 10)) {
		t.Fatalf("ward: %q %v", s.mods.resistType, s.mods.resist)
	}
	hp := s.HP
	w.hitSoldier(nil, 1, s, fixed.FromInt(20), "fire", false)
	warded := hp - s.HP
	s.HP = hp
	setMode(w, h, 2)
	w.Step(nil)
	if s.mods.resistType != "lightning" {
		t.Fatalf("ward did not follow the element: %q", s.mods.resistType)
	}
	w.hitSoldier(nil, 1, s, fixed.FromInt(20), "fire", false)
	if hp-s.HP <= warded {
		t.Error("fire still warded after switching to lightning")
	}
}

func TestSummonElementalByMode(t *testing.T) {
	w := newTestWorld(t)
	h := archmage(t, w)
	setMode(w, h, 1)
	w.Step([]Command{{Player: 0, Kind: CmdCast, Hordes: []HordeID{h.ID}, Name: "summon_elemental"}})
	last := w.Hordes[len(w.Hordes)-1]
	if last.Def.ID != "ice_elementals" || !last.summoned {
		t.Fatalf("summoned %s", last.Def.ID)
	}
	n := 0
	for _, x := range w.Hordes {
		if x.summoned {
			n++
		}
	}
	if n != 1 {
		t.Errorf("summoned %d elementals, want 1", n)
	}
}

func TestMeteorShowerUsesTheElement(t *testing.T) {
	w := newTestWorld(t)
	h := archmage(t, w)
	setMode(w, h, 2) // lightning
	enemy, _ := w.SpawnHorde(1, "raiders", h.Pos.Add(fixed.V2i(15, 0)), 0)
	enemy.ordered = true
	w.Step([]Command{{Player: 0, Kind: CmdCast, Hordes: []HordeID{h.ID}, Name: "meteor_shower", Target: enemy.Pos}})
	if len(w.Zones) != 1 {
		t.Fatalf("zones %d", len(w.Zones))
	}
	z := w.Zones[0]
	if len(z.effects) != 2 || z.effects[0].DamageType != "lightning" {
		t.Fatalf("zone effects %+v", z.effects)
	}
	stunned := false
	for i := 0; i < 25; i++ {
		enemy.ordered, enemy.attack = true, nil
		w.Step(nil)
		for _, s := range enemy.Soldiers {
			stunned = stunned || s.fx.stunTicks > 0
		}
	}
	if !stunned {
		t.Error("lightning meteors did not stun")
	}
}

func TestArcaneFocusHastesMagesOnly(t *testing.T) {
	w := newTestWorld(t)
	h := archmage(t, w)
	mages, _ := w.SpawnHorde(0, "lightning_mages", h.Pos.Add(fixed.V2i(6, 0)), 0)
	swords, _ := w.SpawnHorde(0, "swordsmen", h.Pos.Add(fixed.V2i(-6, 0)), 0)
	w.Step(nil)
	if mages.Soldiers[0].mods.abilityHaste == 0 || swords.Soldiers[0].mods.abilityHaste != 0 {
		t.Fatalf("haste: mages %v swords %v", mages.Soldiers[0].mods.abilityHaste, swords.Soldiers[0].mods.abilityHaste)
	}
	i := mages.AbilityIndex("chain_lightning")
	mages.abilityCd[i] = 300
	for k := 0; k < 100; k++ {
		w.Step(nil)
	}
	if left := mages.abilityCd[i]; left > 170 || left < 160 { // ~1.33 ticks per tick
		t.Errorf("hasted cooldown left %d after 100 ticks, want ~167", left)
	}
}
