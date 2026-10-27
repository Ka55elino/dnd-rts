package sim

import (
	"testing"

	"dnd-rts/internal/data"
	"dnd-rts/internal/fixed"
)

func TestChainLightningJumps(t *testing.T) {
	w := newTestWorld(t)
	p := openSpot(w, -60, 60)
	mages, _ := w.SpawnHorde(0, "lightning_mages", p, 0)
	enemy, _ := w.SpawnHorde(1, "raiders", p.Add(fixed.V2i(15, 0)), 0)
	hp := make(map[SoldierID]fixed.Fixed)
	for _, s := range enemy.Soldiers {
		hp[s.ID] = s.HP
	}
	w.Step([]Command{{Player: 0, Kind: CmdCast, Hordes: []HordeID{mages.ID}, Name: "chain_lightning", TargetHorde: enemy.ID}})
	hit, chains := 0, 0
	var first, last fixed.Fixed
	for _, s := range enemy.Soldiers {
		if d := hp[s.ID] - s.HP; d > fixed.FromInt(5) {
			hit++
			if s.fx.stunTicks == 0 {
				t.Error("chain target not stunned")
			}
			if first == 0 || d > first {
				first = d
			}
			if last == 0 || d < last {
				last = d
			}
		}
	}
	for _, e := range w.Events {
		if e.Kind == EventChain {
			chains++
		}
	}
	if hit != 5 || chains != 5 {
		t.Fatalf("chain hit %d soldiers with %d links, want 5 / 5", hit, chains)
	}
	if last >= first {
		t.Errorf("no falloff: first %v, last %v", first, last)
	}
}

func TestFrostFanIsACone(t *testing.T) {
	w := newTestWorld(t)
	hero, _ := w.SpawnHorde(1, "warlord", openSpot(w, -60, 60), 0)
	hero.gainXP(1400)
	front, _ := w.SpawnHorde(0, "swordsmen", hero.Pos.Add(fixed.V2i(7, 0)), 0)
	behind, _ := w.SpawnHorde(0, "swordsmen", hero.Pos.Add(fixed.V2i(-7, 0)), 0)
	f0, b0 := totalHP(front), totalHP(behind)
	w.Step([]Command{{Player: 1, Kind: CmdCast, Hordes: []HordeID{hero.ID}, Name: "frost_fan", Target: hero.Pos.Add(fixed.V2i(10, 0))}})
	if totalHP(front) >= f0 {
		t.Error("cone missed the horde in front")
	}
	if totalHP(behind) != b0 {
		t.Error("cone hit the horde behind")
	}
	cone := false
	for _, e := range w.Events {
		cone = cone || e.Kind == EventCone
	}
	if !cone {
		t.Error("no cone event")
	}
}

func TestLimitedSummon(t *testing.T) {
	w := newTestWorld(t)
	hero, _ := w.SpawnHorde(1, "warlord", openSpot(w, -60, 60), 0)
	hero.gainXP(1400)
	n := len(w.Hordes)
	w.Step([]Command{{Player: 1, Kind: CmdCast, Hordes: []HordeID{hero.ID}, Name: "pack_call"}})
	if len(w.Hordes) != n+1 {
		t.Fatalf("hordes %d → %d, want one summoned", n, len(w.Hordes))
	}
	wolves := w.Hordes[len(w.Hordes)-1]
	if wolves.Def.ID != "wolf_riders" || wolves.Owner != 1 || !wolves.summoned {
		t.Fatalf("summoned %s for %d", wolves.Def.ID, wolves.Owner)
	}
	i := hero.AbilityIndex("pack_call")
	cd := hero.abilityCd[i]
	for k := 0; k < 40; k++ {
		w.Step(nil)
	}
	if hero.abilityCd[i] != cd {
		t.Errorf("cooldown ran while the pack lives: %d → %d", cd, hero.abilityCd[i])
	}
	for _, s := range wolves.Soldiers {
		s.HP = 0
	}
	w.Step(nil)
	w.Step(nil)
	if hero.abilityCd[i] >= cd {
		t.Error("cooldown did not start after the pack died")
	}
}

func TestSummonExpires(t *testing.T) {
	w := newTestWorld(t)
	hero, _ := w.SpawnHorde(1, "warlord", openSpot(w, -60, 60), 0)
	w.summon(1, hero.caster(), hero.abilities[0], &summonEffect, hero.Pos)
	sum := w.Hordes[len(w.Hordes)-1]
	if !sum.summoned || sum.expires == 0 {
		t.Fatal("no timed summon")
	}
	for i := 0; i < 20*3+2; i++ {
		w.Step(nil)
	}
	if w.Horde(sum.ID) != nil {
		t.Error("summon did not expire")
	}
}

func TestBlinkTeleportsTheHorde(t *testing.T) {
	w := newTestWorld(t)
	mages, _ := w.SpawnHorde(0, "lightning_mages", openSpot(w, -60, 60), 0)
	giveAbility(w, mages, "blink")
	dest := openSpot(w, -40, 60)
	w.Step([]Command{{Player: 0, Kind: CmdCast, Hordes: []HordeID{mages.ID}, Name: "blink", Target: dest}})
	if mages.Pos != dest {
		t.Fatalf("blink: at %v, want %v", mages.Pos, dest)
	}
	for _, s := range mages.Soldiers {
		if s.Pos.Dist(dest) > fixed.FromInt(5) {
			t.Errorf("soldier left behind at %v", s.Pos)
		}
	}
}

func TestMassRecallNeedsOwnBuilding(t *testing.T) {
	w := newTestWorld(t)
	keep := w.Buildings[0]
	hero, _ := w.SpawnHorde(0, "captain", openSpot(w, -60, 60), 0)
	ally, _ := w.SpawnHorde(0, "swordsmen", hero.Pos.Add(fixed.V2i(6, 0)), 0)
	giveAbility(w, hero, "mass_recall")
	hero.gainXP(1400)
	hero.level = 6 // slot 5 unlocked
	start := hero.Pos
	w.Step([]Command{{Player: 0, Kind: CmdCast, Hordes: []HordeID{hero.ID}, Name: "mass_recall", Target: openSpot(w, 0, 0)}})
	if hero.Pos != start {
		t.Fatal("recalled to open ground")
	}
	w.Step([]Command{{Player: 0, Kind: CmdCast, Hordes: []HordeID{hero.ID}, Name: "mass_recall", Target: keep.Pos}})
	if keep.DistTo(hero.Pos) > fixed.FromInt(8) || keep.DistTo(ally.Pos) > fixed.FromInt(16) {
		t.Errorf("recall: hero %v from the keep, ally %v", keep.DistTo(hero.Pos), keep.DistTo(ally.Pos))
	}
}

func TestBeastForm(t *testing.T) {
	w := newTestWorld(t)
	hero, _ := w.SpawnHorde(1, "warlord", openSpot(w, -60, 60), 0)
	hero.gainXP(1400)
	s := hero.Soldiers[0]
	s.HP = s.MaxHP().DivInt(2)
	w.Step([]Command{{Player: 1, Kind: CmdCast, Hordes: []HordeID{hero.ID}, Name: "beast_form"}})
	if hero.Unit.ID != "warbeast" || !hero.Transformed() || !hero.IsHero() || hero.Level() != 5 {
		t.Fatalf("form %s hero %v level %d", hero.Unit.ID, hero.IsHero(), hero.Level())
	}
	if !approx(s.HP.Div(s.MaxHP()), fixed.FromFraction(1, 2)) {
		t.Errorf("health fraction %v, want 0.5", s.HP.Div(s.MaxHP()))
	}
	if hero.FormScale() <= fixed.One {
		t.Error("beast should be bigger")
	}
	for i := 0; i < 20*30+1; i++ {
		w.Step(nil)
	}
	if hero.Unit.ID != "warlord" {
		t.Errorf("did not turn back: %s", hero.Unit.ID)
	}
}

var summonEffect = data.AbilityEffect{Type: data.EffSummon, Horde: "raiders", Duration: fixed.FromInt(3)}

// giveAbility adds an ability slot to a horde (tests of abilities no unit has).
func giveAbility(w *World, h *Horde, id string) {
	h.abilities = append(h.abilities, w.Catalog.Abilities[id])
	h.abilityCd = append(h.abilityCd, 0)
	h.modes = append(h.modes, 0)
	h.toggled = append(h.toggled, false)
	h.autocast = append(h.autocast, false)
}
