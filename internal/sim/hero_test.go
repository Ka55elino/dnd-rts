package sim

import (
	"testing"

	"dnd-rts/internal/fixed"
)

func findHorde(w *World, p PlayerID, id string) *Horde {
	for _, h := range w.Hordes {
		if h.Owner == p && h.Def.ID == id {
			return h
		}
	}
	return nil
}

func TestHeroLevelsUpFromKills(t *testing.T) {
	w := newTestWorld(t)
	hero, _ := w.SpawnHorde(0, "captain", openSpot(w, -60, 60), 0)
	if hero.Level() != 1 || hero.abilityUnlocked(1) {
		t.Fatalf("fresh hero: level %d, ability 2 unlocked %v", hero.Level(), hero.abilityUnlocked(1))
	}
	hp1 := hero.Soldiers[0].MaxHP()
	enemy, _ := w.SpawnHorde(1, "raiders", hero.Pos.Add(fixed.V2i(6, 0)), 0)
	for _, s := range enemy.Soldiers {
		s.HP = 0
	}
	w.Step(nil)
	if hero.XP() == 0 {
		t.Fatal("no XP for enemies dying nearby")
	}
	hero.gainXP(150 - hero.XP())
	if hero.Level() != 2 || !hero.abilityUnlocked(1) || hero.abilityUnlocked(2) {
		t.Fatalf("at 150 xp: level %d", hero.Level())
	}
	s := hero.Soldiers[0]
	if s.MaxHP() <= hp1 || s.HP != s.MaxHP() {
		t.Errorf("level up: max hp %v → %v, hp %v", hp1, s.MaxHP(), s.HP)
	}
	w.Step(nil)
	// +10% per level and +15% from battle_cry, which opened at level 2.
	if !approx(s.mods.damage, fixed.FromFraction(25, 100)) || s.mods.armor != fixed.One {
		t.Errorf("level 2 mods: damage %v armor %v", s.mods.damage, s.mods.armor)
	}
	hero.gainXP(100000)
	if hero.Level() != 5 {
		t.Errorf("max level %d, want 5", hero.Level())
	}
}

func TestLockedAbilityRejected(t *testing.T) {
	w := newTestWorld(t)
	hero, _ := w.SpawnHorde(0, "captain", openSpot(w, -60, 60), 0)
	w.Step([]Command{{Player: 0, Kind: CmdCast, Hordes: []HordeID{hero.ID}, Name: "storm", Target: hero.Pos.Add(fixed.V2i(10, 0))}})
	if len(w.Zones) != 0 || hero.pending != nil {
		t.Fatal("storm cast at level 1")
	}
	if st := hero.AbilityStates(); st[0].Locked || !st[3].Locked {
		t.Errorf("locked flags: %+v", st)
	}
}

func TestHeroUniqueAndRevive(t *testing.T) {
	w := newTestWorld(t)
	keep := w.Buildings[0]
	gold := res(t, w, "gold")
	w.Step([]Command{{Player: 0, Kind: CmdDevGive, Name: "gold", Count: 10000}})
	for _, h := range w.Hordes { // free food for the test
		if h.Owner == 0 && !h.IsHero() {
			for _, s := range h.Soldiers {
				s.HP = 0
			}
		}
	}
	w.Step(nil)
	hero := findHorde(w, 0, "paladin")
	if hero == nil {
		hero, _ = w.SpawnHorde(0, "paladin", openSpot(w, -60, 60), 0)
	}
	hero.gainXP(400) // level 3
	before := w.Stock(0, gold)
	w.Step([]Command{{Player: 0, Kind: CmdTrain, TargetBuilding: keep.ID, Name: "paladin"}})
	if keep.Queue() != nil || w.Stock(0, gold) != before {
		t.Fatal("hired a second paladin while one is alive")
	}
	for _, s := range hero.Soldiers {
		s.HP = 0
	}
	w.Step(nil)
	if findHorde(w, 0, "paladin") != nil {
		t.Fatal("hero did not die")
	}
	hs := w.Heroes(0)
	if len(hs) != 1 || hs[0].Horde != "paladin" || hs[0].Alive || hs[0].Level != 3 {
		t.Fatalf("hero records %+v", hs)
	}
	w.Step([]Command{{Player: 0, Kind: CmdTrain, TargetBuilding: keep.ID, Name: "paladin"},
		{Player: 0, Kind: CmdTrain, TargetBuilding: keep.ID, Name: "paladin"}})
	if q := keep.Queue(); q == nil || len(q.Hordes) != 1 {
		t.Fatalf("revive queue %+v", q)
	}
	for i := 0; i < 20*121 && findHorde(w, 0, "paladin") == nil; i++ {
		w.Step(nil)
	}
	back := findHorde(w, 0, "paladin")
	if back == nil {
		t.Fatal("not revived")
	}
	if back.Level() != 3 || back.XP() != 400 || back.Soldiers[0].HP != back.Soldiers[0].MaxHP() {
		t.Errorf("revived: level %d xp %d", back.Level(), back.XP())
	}
}

func TestCaptureGivesXP(t *testing.T) {
	w := newTestWorld(t)
	mine := outpost(t, w, "gold_mine")
	hero, _ := w.SpawnHorde(0, "captain", near(w, mine), 0)
	for i := 0; i < 20*20 && mine.Owner == Neutral; i++ {
		w.Step(nil)
	}
	if mine.Owner != 0 || hero.XP() < w.Catalog.Combat.XP.Capture {
		t.Errorf("capture: owner %d, hero xp %d", mine.Owner, hero.XP())
	}
}
