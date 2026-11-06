package sim

import (
	"testing"

	"dnd-rts/internal/data"
	"dnd-rts/internal/fixed"
)

// give puts a long timed modifier on every soldier of the horde.
func give(h *Horde, m *data.Modifiers) {
	for _, s := range h.Soldiers {
		s.buffs = append(s.buffs, buff{m: m, ticks: 1 << 20})
	}
}

func TestInvisibleHordeIsNotTargeted(t *testing.T) {
	w := newTestWorld(t)
	p := openSpot(w, -60, 60)
	ghosts, _ := w.SpawnHorde(0, "swordsmen", p, 0)
	enemy, _ := w.SpawnHorde(1, "swordsmen", p.Add(fixed.V2i(12, 0)), 0)
	give(ghosts, &data.Modifiers{Invisible: true})
	w.Step(nil)
	if !ghosts.Hidden() || w.VisibleTo(ghosts, 1) || !w.VisibleTo(ghosts, 0) {
		t.Fatal("invisible horde is seen by the enemy")
	}
	hp := totalHP(ghosts)
	for range 3 * TickRate {
		w.Step(nil)
	}
	if totalHP(ghosts) != hp {
		t.Error("enemy attacked an invisible horde")
	}
	_ = enemy
}

func TestDetectorSeesInvisible(t *testing.T) {
	w := newTestWorld(t)
	p := openSpot(w, -60, 60)
	ghosts, _ := w.SpawnHorde(0, "swordsmen", p, 0)
	seer, _ := w.SpawnHorde(1, "swordsmen", p.Add(fixed.V2i(20, 0)), 0)
	give(ghosts, &data.Modifiers{Invisible: true})
	give(seer, &data.Modifiers{Detect: fixed.FromInt(30)})
	w.Step(nil)
	if !ghosts.Hidden() || !w.VisibleTo(ghosts, 1) {
		t.Fatal("detector does not see the invisible horde")
	}
}

func TestAttackRevealsInvisible(t *testing.T) {
	w := newTestWorld(t)
	p := openSpot(w, -60, 60)
	ghosts, _ := w.SpawnHorde(0, "swordsmen", p, 0)
	enemy, _ := w.SpawnHorde(1, "swordsmen", p.Add(fixed.V2i(12, 0)), 0)
	give(ghosts, &data.Modifiers{Invisible: true})
	w.Step([]Command{{Player: 0, Kind: CmdAttack, Hordes: []HordeID{ghosts.ID}, TargetHorde: enemy.ID}})
	revealed := false
	for range 4 * TickRate {
		w.Step(nil)
		revealed = revealed || !ghosts.Hidden()
	}
	if !revealed {
		t.Error("attacking did not reveal the invisible horde")
	}
}

func TestCamouflageOnlyStandingStill(t *testing.T) {
	w := newTestWorld(t)
	p := openSpot(w, -60, 60)
	h, _ := w.SpawnHorde(0, "swordsmen", p, 0)
	give(h, &data.Modifiers{Camouflage: true})
	w.Step(nil)
	if !h.Hidden() {
		t.Fatal("standing camouflaged horde is not hidden")
	}
	w.Step([]Command{{Player: 0, Kind: CmdMove, Hordes: []HordeID{h.ID}, Target: p.Add(fixed.V2i(40, 0))}})
	w.Step(nil)
	if h.Hidden() {
		t.Error("moving camouflaged horde is hidden")
	}
}

func TestReflectDamage(t *testing.T) {
	w := newTestWorld(t)
	p := openSpot(w, -60, 60)
	a, _ := w.SpawnHorde(0, "swordsmen", p, 0)
	b, _ := w.SpawnHorde(1, "swordsmen", p.Add(fixed.V2i(6, 0)), 0)
	give(b, &data.Modifiers{Reflect: fixed.One, Invulnerable: false})
	a0 := totalHP(a)
	w.Step([]Command{{Player: 0, Kind: CmdAttack, Hordes: []HordeID{a.ID}, TargetHorde: b.ID}})
	for range 2 * TickRate {
		w.Step(nil)
	}
	if totalHP(a) >= a0 {
		t.Error("attackers took no reflected damage")
	}
}

func TestNoHealBlocksHealing(t *testing.T) {
	w := newTestWorld(t)
	h, _ := w.SpawnHorde(0, "swordsmen", openSpot(w, -60, 60), 0)
	s := h.Soldiers[0]
	s.HP = s.HP / 2
	give(h, &data.Modifiers{NoHeal: true})
	w.updateMods()
	before := s.HP
	w.heal(s, fixed.FromInt(10))
	if s.HP != before {
		t.Errorf("healed through noHeal: %v → %v", before, s.HP)
	}
}
