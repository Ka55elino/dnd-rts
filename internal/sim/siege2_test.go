package sim

import (
	"slices"
	"testing"

	"dnd-rts/internal/data"
	"dnd-rts/internal/fixed"
)

func stillEnemy(h *Horde) { h.ordered, h.attack = true, nil }

func TestTrebuchetDeadZoneAndSplash(t *testing.T) {
	w := newTestWorld(t)
	p := openSpot(w, 0, 60)
	tr, _ := w.SpawnHorde(0, "trebuchets", p, 0)
	near, _ := w.SpawnHorde(1, "raiders", p.Add(fixed.V2i(8, 0)), 0)
	hp := totalHP(near)
	for i := 0; i < 20*10; i++ {
		stillEnemy(near)
		w.Step(nil)
	}
	if totalHP(near) < hp {
		t.Fatal("trebuchet hit something inside its dead zone")
	}
	far, _ := w.SpawnHorde(1, "swordsmen", p.Add(fixed.V2i(-40, 0)), 0)
	near.Soldiers = nil // out of the way
	fhp := totalHP(far)
	hurt := 0
	for i := 0; i < 20*12; i++ {
		stillEnemy(far)
		w.Step(nil)
	}
	for _, s := range far.Soldiers {
		if s.HP < s.MaxHP() {
			hurt++
		}
	}
	if totalHP(far) >= fhp || hurt < 3 {
		t.Errorf("trebuchet volley: hp %v → %v, %d soldiers hurt (want splash on several)", fhp, totalHP(far), hurt)
	}
	_ = tr
}

func TestBallistaShootsFlyers(t *testing.T) {
	w := newTestWorld(t)
	p := openSpot(w, 0, 60)
	w.SpawnHorde(0, "ballistas", p, 0)
	harpies, _ := w.SpawnHorde(1, "harpies", p.Add(fixed.V2i(25, 0)), 0)
	hp := totalHP(harpies)
	for i := 0; i < 20*8; i++ {
		stillEnemy(harpies)
		w.Step(nil)
	}
	if totalHP(harpies) >= hp {
		t.Error("ballistas did not hit the harpies")
	}
}

// towerOnSocket builds a tower of type id on the first free socket of player 0's keep and finishes it.
func towerOnSocket(t *testing.T, w *World, id string) *Building {
	t.Helper()
	keep := fortressOf(w, 0)
	w.Step([]Command{{Player: 0, Kind: CmdDevGive, Name: "gold", Count: 9000}, {Player: 0, Kind: CmdDevGive, Name: "wood", Count: 9000},
		{Player: 0, Kind: CmdDevGive, Name: "metal", Count: 9000}})
	for i, p := range keep.Def.Plots {
		if p.Class == data.PlotTower && !keep.PlotOccupied(i) {
			w.Step([]Command{{Player: 0, Kind: CmdBuildOnPlot, TargetBuilding: keep.ID, Count: i, Name: id}})
			b := w.Buildings[len(w.Buildings)-1]
			if b.Def.ID != id {
				t.Fatalf("no %s on socket %d", id, i)
			}
			b.buildTicks, b.HP = 0, b.Def.Health
			return b
		}
	}
	t.Fatal("no free socket")
	return nil
}

func TestTowerFireToggleNeedsTheUpgrade(t *testing.T) {
	w := newTestWorld(t)
	tower := towerOnSocket(t, w, "archer_tower")
	w.Step([]Command{{Player: 0, Kind: CmdToggle, TargetBuilding: tower.ID, Name: "fire_mode"}})
	if tower.ToggleOn() {
		t.Fatal("fire arrows on without the upgrade")
	}
	w.completeUpgrade(0, w.Catalog.Upgrades["fire_arrows"])
	w.Step([]Command{{Player: 1, Kind: CmdToggle, TargetBuilding: tower.ID, Name: "fire_mode"}})
	if tower.ToggleOn() {
		t.Fatal("the enemy switched our tower")
	}
	w.Step([]Command{{Player: 0, Kind: CmdToggle, TargetBuilding: tower.ID, Name: "fire_mode"}})
	if !tower.ToggleOn() {
		t.Fatal("fire arrows did not turn on")
	}
	keep := fortressOf(w, 0)
	w.SpawnHorde(1, "raiders", tower.Pos.Add(tower.Pos.Sub(keep.Pos).Normalize().Scale(fixed.FromInt(20))), 0)
	fire := false
	for i := 0; i < 20*3 && !fire; i++ {
		w.Step(nil)
		fire = slices.ContainsFunc(w.Projectiles, func(p *Projectile) bool { return p.ExtraType == "fire" && p.NoPierce })
	}
	if !fire {
		t.Error("tower shots carry no fire")
	}
}

func TestImprovedMechanismSpeedsTrebuchetTowers(t *testing.T) {
	w := newTestWorld(t)
	if _, err := w.SpawnBuilding(0, "forge", fortressOf(w, 0).PlotPos(0), 0); err != nil { // the tower needs a forge
		t.Fatal(err)
	}
	tower := towerOnSocket(t, w, "trebuchet_tower")
	keep := fortressOf(w, 0)
	w.SpawnHorde(1, "swordsmen", tower.Pos.Add(tower.Pos.Sub(keep.Pos).Normalize().Scale(fixed.FromInt(35))), 0)
	cdFor := func() int {
		tower.atkCd = 0
		for i := 0; i < 5 && tower.atkCd == 0; i++ {
			w.Step(nil)
		}
		return tower.atkCd
	}
	before := cdFor()
	w.completeUpgrade(0, w.Catalog.Upgrades["improved_mechanism"])
	after := cdFor()
	if before == 0 || after >= before*85/100 {
		t.Errorf("tower cooldown %d → %d ticks, want ~25%% shorter", before, after)
	}
}

func TestWorkersRepairBuildingsAndMachines(t *testing.T) {
	w := newTestWorld(t)
	keep := fortressOf(w, 0)
	keep.HP = keep.Def.Health.DivInt(2)
	b := builder(t, w, 0)
	w.Step([]Command{{Player: 0, Kind: CmdConstruct, TargetBuilding: keep.ID, Hordes: []HordeID{b.ID}}})
	hp := keep.HP
	for i := 0; i < 20*20; i++ {
		w.Step(nil)
	}
	if keep.HP <= hp {
		t.Fatalf("keep not repaired: %v → %v", hp, keep.HP)
	}
	// Idle workers next to a damaged machine fix it.
	ba, _ := w.SpawnHorde(0, "ballistas", b.Pos.Add(fixed.V2i(3, 0)), 0)
	b.construct, b.constructing, b.moving = nil, false, false
	s := ba.Soldiers[0]
	s.HP = s.MaxHP().DivInt(2)
	mhp := s.HP
	for i := 0; i < 20*3; i++ {
		w.Step(nil)
	}
	if s.HP <= mhp {
		t.Error("ballista not repaired by the idle builders")
	}
}
