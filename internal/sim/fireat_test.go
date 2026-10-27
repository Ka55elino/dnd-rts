package sim

import (
	"testing"

	"dnd-rts/internal/fixed"
)

func TestAttackGround(t *testing.T) {
	w := newTestWorld(t)
	p := openSpot(w, 0, 60)
	archers, _ := w.SpawnHorde(0, "archers", p, 0)
	spot := p.Add(fixed.V2i(55, 0)) // out of range: they walk up first
	enemy, _ := w.SpawnHorde(1, "swordsmen", spot, 0)
	hp := totalHP(enemy)
	w.Step([]Command{{Player: 0, Kind: CmdAttackGround, Hordes: []HordeID{archers.ID}, Target: spot}})
	volleys := 0
	for i := 0; i < 20*15; i++ {
		stillEnemy(enemy)
		w.Step(nil)
		for _, pr := range w.Projectiles {
			if pr.LaunchTick == w.Tick-1 && pr.Owner == 0 {
				volleys++
			}
		}
	}
	if d := archers.Pos.Dist(spot); d > archers.attackRange(archers.Unit.Ranged)+engageBase {
		t.Fatalf("archers stayed %v from the spot, out of range", d)
	}
	if volleys < 3 || totalHP(enemy) >= hp {
		t.Errorf("%d volleys at the spot, enemy hp %v → %v", volleys, hp, totalHP(enemy))
	}
	// A new order cancels it.
	w.Step([]Command{{Player: 0, Kind: CmdStop, Hordes: []HordeID{archers.ID}}})
	if archers.hasFireAt {
		t.Error("stop did not cancel the ground attack")
	}
	// Melee-only hordes ignore it.
	sw, _ := w.SpawnHorde(0, "swordsmen", openSpot(w, -40, 60), 0)
	w.Step([]Command{{Player: 0, Kind: CmdAttackGround, Hordes: []HordeID{sw.ID}, Target: spot}})
	if sw.hasFireAt {
		t.Error("swordsmen took a ground-attack order")
	}
}

func TestAttackGroundRespectsTheDeadZone(t *testing.T) {
	w := newTestWorld(t)
	p := openSpot(w, 0, 60)
	tr, _ := w.SpawnHorde(0, "trebuchets", p, 0)
	near := p.Add(fixed.V2i(6, 0)) // inside the 16 m dead zone: it backs off, then shoots
	w.Step([]Command{{Player: 0, Kind: CmdAttackGround, Hordes: []HordeID{tr.ID}, Target: near}})
	shot := false
	for i := 0; i < 20*30 && !shot; i++ {
		w.Step(nil)
		shot = len(w.Projectiles) > 0
	}
	if !shot || tr.Pos.Dist(near) < tr.Unit.Ranged.MinRange {
		t.Errorf("trebuchet: shot %v at %v from the spot (dead zone %v)", shot, tr.Pos.Dist(near), tr.Unit.Ranged.MinRange)
	}
}
