package sim

import (
	"testing"

	"dnd-rts/internal/fixed"
)

func maxHero(t *testing.T, w *World, owner PlayerID, id string, at fixed.Vec2) *Horde {
	t.Helper()
	h, err := w.SpawnHorde(owner, id, at, 0)
	if err != nil {
		t.Fatal(err)
	}
	h.gainXP(1400)
	return h
}

func TestPaladinPurityAndShield(t *testing.T) {
	w := newTestWorld(t)
	p := maxHero(t, w, 0, "paladin", openSpot(w, -60, 60))
	w.Step(nil)
	s := p.Soldiers[0]
	w.hitSoldier(nil, 1, s, fixed.FromInt(30), "poison", true)
	if s.fx.poisonTicks > 0 {
		t.Error("purity: paladin got poisoned")
	}
	w.Step([]Command{{Player: 0, Kind: CmdCast, Hordes: []HordeID{p.ID}, Name: "divine_shield"}})
	w.Step(nil)
	hp := s.HP
	w.hitSoldier(nil, 1, s, fixed.FromInt(300), "slash", false)
	if s.HP != hp {
		t.Error("divine shield: paladin took damage")
	}
}

func TestPaladinRadianceAndSmite(t *testing.T) {
	w := newTestWorld(t)
	p := maxHero(t, w, 0, "paladin", openSpot(w, -60, 60))
	enemy, _ := w.SpawnHorde(1, "warlord", p.Pos.Add(fixed.V2i(5, 0)), 0)
	w.Step([]Command{{Player: 0, Kind: CmdCast, Hordes: []HordeID{p.ID}, Name: "holy_radiance"}})
	if enemy.Soldiers[0].fx.blindTicks == 0 || p.Soldiers[0].fx.blindTicks != 0 {
		t.Error("radiance should blind enemies only")
	}
	hp := enemy.Soldiers[0].HP
	w.Step([]Command{{Player: 0, Kind: CmdCast, Hordes: []HordeID{p.ID}, Name: "divine_smite", TargetHorde: enemy.ID}})
	if got := hp - enemy.Soldiers[0].HP; got < fixed.FromInt(250) {
		t.Errorf("smite dealt %v, want the full 250 (radiant ignores defences)", got)
	}
}

func TestKingOversightSpeedsTraining(t *testing.T) {
	run := func(withEmperor bool) int {
		w := newTestWorld(t)
		keep := w.Buildings[0]
		for _, h := range w.Hordes { // free the food
			if h.Owner == 0 && !h.IsHero() {
				for _, s := range h.Soldiers {
					s.HP = 0
				}
			}
		}
		w.Step(nil)
		if withEmperor {
			maxHero(t, w, 0, "emperor", keep.approach(fixed.Vec2{}, fixed.FromInt(3)))
		}
		w.Step([]Command{{Player: 0, Kind: CmdTrain, TargetBuilding: keep.ID, Name: "builders"}})
		for i := 1; i < 20*60; i++ {
			if keep.Queue() == nil {
				return i
			}
			w.Step(nil)
		}
		return -1
	}
	plain, fast := run(false), run(true)
	if plain < 0 || fast < 0 || fast >= plain*85/100 {
		t.Errorf("training: %d ticks without the emperor, %d with (want ~30%% faster)", plain, fast)
	}
}

func TestKingInspireAndGuard(t *testing.T) {
	w := newTestWorld(t)
	k := maxHero(t, w, 0, "emperor", openSpot(w, -60, 60))
	near, _ := w.SpawnHorde(0, "swordsmen", k.Pos.Add(fixed.V2i(6, 0)), 0)
	w.Step([]Command{{Player: 0, Kind: CmdCast, Hordes: []HordeID{k.ID}, Name: "inspire"}})
	w.Step(nil)
	if !approx(near.Soldiers[0].mods.attackSpeed, fixed.FromFraction(3, 10)) {
		t.Errorf("inspire: ally attack speed %v", near.Soldiers[0].mods.attackSpeed)
	}
	if !approx(near.Soldiers[0].mods.moveSpeed, fixed.FromFraction(15, 100)) {
		t.Errorf("imperial march: ally move speed %v", near.Soldiers[0].mods.moveSpeed)
	}
	n := len(w.Hordes)
	w.Step([]Command{{Player: 0, Kind: CmdCast, Hordes: []HordeID{k.ID}, Name: "imperial_guard"}})
	if len(w.Hordes) != n+1 || w.Hordes[n].Def.ID != "imperial_guards" || len(w.Hordes[n].Soldiers) != 6 {
		t.Fatal("no imperial guard")
	}
}
