package sim

import (
	"testing"

	"dnd-rts/internal/fixed"
)

// runUntil steps the world until done() or the tick limit.
func runUntil(w *World, ticks int, cmds []Command, done func() bool) int {
	for t := 0; t < ticks; t++ {
		if t > 0 {
			cmds = nil
		}
		w.Step(cmds)
		if done() {
			return t
		}
	}
	return -1
}

func TestMeleeDestroysBuilding(t *testing.T) {
	w := newTestWorld(t)
	farm, err := w.SpawnBuilding(1, "farm", fixed.V2i(0, 40), 0)
	if err != nil {
		t.Fatal(err)
	}
	h, err := w.SpawnHorde(0, "swordsmen", fixed.V2i(-20, 30), 0)
	if err != nil {
		t.Fatal(err)
	}
	g := w.Nav.Grid()
	took := runUntil(w, 20*120, []Command{{Player: 0, Kind: CmdAttackBuilding, Hordes: []HordeID{h.ID}, TargetBuilding: farm.ID}},
		func() bool { return w.Building(farm.ID) == nil })
	if took < 0 {
		t.Fatalf("farm not destroyed in 2 minutes, hp %v", farm.HP)
	}
	if !g.WalkableAt(farm.Pos) {
		t.Error("ground under the destroyed farm is still blocked")
	}
	if h.SiegeTarget() != nil {
		t.Error("horde still targets the destroyed building")
	}
	t.Logf("destroyed in %.1fs", float64(took)/TickRate)
}

func TestArchersDamageBuilding(t *testing.T) {
	w := newTestWorld(t)
	farm, _ := w.SpawnBuilding(1, "farm", fixed.V2i(0, 40), 0)
	h, _ := w.SpawnHorde(0, "archers", fixed.V2i(-60, 40), 0)
	runUntil(w, 20*20, []Command{{Player: 0, Kind: CmdAttackBuilding, Hordes: []HordeID{h.ID}, TargetBuilding: farm.ID}},
		func() bool { return false })
	if farm.HP >= farm.Def.Health {
		t.Fatalf("archers did no damage in 20s (hp %v)", farm.HP)
	}
}

func TestCannotAttackOwnOrNeutralBuilding(t *testing.T) {
	w := newTestWorld(t)
	h := w.Hordes[0] // player 0
	own := w.Buildings[0]
	neutral := firstOutpost(w)
	w.Step([]Command{
		{Player: 0, Kind: CmdAttackBuilding, Hordes: []HordeID{h.ID}, TargetBuilding: own.ID},
		{Player: 0, Kind: CmdAttackBuilding, Hordes: []HordeID{h.ID}, TargetBuilding: neutral.ID},
	})
	if h.SiegeTarget() != nil {
		t.Fatal("own / neutral building accepted as a target")
	}
}

func TestLosingFortressEndsMatch(t *testing.T) {
	w := newTestWorld(t)
	enemyFort := fortressOf(w, 1)
	if enemyFort.Owner != 1 || w.Over {
		t.Fatal("unexpected setup")
	}
	enemyFort.HP = fixed.One
	h, _ := w.SpawnHorde(0, "swordsmen", enemyFort.approach(fixed.V2i(0, 0), fixed.FromInt(4)), 0)
	took := runUntil(w, 20*30, []Command{{Player: 0, Kind: CmdAttackBuilding, Hordes: []HordeID{h.ID}, TargetBuilding: enemyFort.ID}},
		func() bool { return w.Over })
	if took < 0 {
		t.Fatal("match did not end")
	}
	if w.Winner != 0 || len(w.Defeated) != 1 || w.Defeated[0] != 1 {
		t.Fatalf("winner %d defeated %v", w.Winner, w.Defeated)
	}
}
