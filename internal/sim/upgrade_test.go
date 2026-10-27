package sim

import (
	"testing"

	"dnd-rts/internal/fixed"
)

// forgeWorld: player 0 has barracks and a forge (spawned finished) and plenty of resources.
func forgeWorld(t *testing.T) (*World, *Building) {
	t.Helper()
	w := newTestWorld(t)
	keep := w.Buildings[0]
	if _, err := w.SpawnBuilding(0, "barracks", keep.PlotPos(0), keep.Facing); err != nil {
		t.Fatal(err)
	}
	forge, err := w.SpawnBuilding(0, "forge", keep.PlotPos(1), keep.Facing)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range []string{"gold", "wood", "metal"} {
		w.Step([]Command{{Player: 0, Kind: CmdDevGive, Name: r, Count: 5000}})
	}
	return w, forge
}

func research(t *testing.T, w *World, b *Building, id string) {
	t.Helper()
	w.Step([]Command{{Player: 0, Kind: CmdResearch, TargetBuilding: b.ID, Name: id}})
	for i := 0; i < 20*60 && b.Queue() != nil; i++ {
		w.Step(nil)
	}
}

func TestUpgradeAppliesToExistingUnits(t *testing.T) {
	w, forge := forgeWorld(t)
	h, _ := w.SpawnHorde(0, "swordsmen", openSpot(w, -50, 60), 0)
	archers, _ := w.SpawnHorde(0, "archers", openSpot(w, -50, 70), 0)
	research(t, w, forge, "forged_weapons_1")
	if !w.HasUpgrade(0, "forged_weapons_1") {
		t.Fatal("upgrade not researched")
	}
	w.Step(nil)
	if h.Soldiers[0].mods.damage != fixed.FromFraction(15, 100) {
		t.Errorf("swordsmen damage mod %v, want 0.15", h.Soldiers[0].mods.damage)
	}
	if archers.Soldiers[0].mods.damage != 0 {
		t.Error("weapons upgrade reached archers")
	}
}

func TestUpgradeRequirements(t *testing.T) {
	w, forge := forgeWorld(t)
	gold := w.Catalog.ResourceIndex("gold")
	before := w.Stock(0, gold)
	w.Step([]Command{{Player: 0, Kind: CmdResearch, TargetBuilding: forge.ID, Name: "forged_weapons_2"}})
	if forge.Queue() != nil || w.Stock(0, gold) != before {
		t.Fatal("level 2 researched without level 1")
	}
	research(t, w, forge, "forged_weapons_1")
	research(t, w, forge, "forged_weapons_2")
	if !w.HasUpgrade(0, "forged_weapons_2") {
		t.Fatal("level 2 not researched after level 1")
	}
	// Researching again is refused.
	before = w.Stock(0, gold)
	w.Step([]Command{{Player: 0, Kind: CmdResearch, TargetBuilding: forge.ID, Name: "forged_weapons_1"}})
	if w.Stock(0, gold) != before {
		t.Error("paid for an upgrade already owned")
	}
}

func TestBuildingAndHordeRequirements(t *testing.T) {
	w := newTestWorld(t)
	keep := w.Buildings[0]
	n := len(w.Buildings)
	w.Step([]Command{{Player: 0, Kind: CmdBuildOnPlot, TargetBuilding: keep.ID, Count: 0, Name: "forge"}})
	if len(w.Buildings) != n {
		t.Fatal("forge built without barracks")
	}
	// Knights need Forged Weapons I.
	barracks, _ := w.SpawnBuilding(0, "barracks", keep.PlotPos(1), keep.Facing)
	w.Step([]Command{{Player: 0, Kind: CmdDevGive, Name: "gold", Count: 5000}, {Player: 0, Kind: CmdDevGive, Name: "metal", Count: 5000}})
	for i := 0; i < 3; i++ {
		w.SpawnBuilding(0, "farm", keep.Pos.Add(fixed.V2i(-30, -30+i*30)), 0)
	}
	w.Step([]Command{{Player: 0, Kind: CmdTrain, TargetBuilding: barracks.ID, Name: "knights"}})
	if barracks.Queue() != nil {
		t.Fatal("knights trained without the upgrade")
	}
}

func TestFireModeNeedsUpgrade(t *testing.T) {
	w, forge := forgeWorld(t)
	archers, _ := w.SpawnHorde(0, "archers", openSpot(w, -50, 70), 0)
	w.Step([]Command{{Player: 0, Kind: CmdToggle, Hordes: []HordeID{archers.ID}, Name: "fire_mode"}})
	if archers.toggled[archers.AbilityIndex("fire_mode")] {
		t.Fatal("fire mode on without Fire Arrows")
	}
	research(t, w, forge, "fire_arrows")
	w.Step([]Command{{Player: 0, Kind: CmdToggle, Hordes: []HordeID{archers.ID}, Name: "fire_mode"}})
	w.Step(nil)
	s := archers.Soldiers[0]
	if s.mods.extraType != "fire" || !s.mods.noPierce {
		t.Fatalf("fire mode mods: %q noPierce=%v", s.mods.extraType, s.mods.noPierce)
	}
}

func TestCancelResearchRefunds(t *testing.T) {
	w, forge := forgeWorld(t)
	gold := w.Catalog.ResourceIndex("gold")
	before := w.Stock(0, gold)
	w.Step([]Command{{Player: 0, Kind: CmdResearch, TargetBuilding: forge.ID, Name: "reinforced_armor_1"}})
	if w.Stock(0, gold) != before-200 {
		t.Fatalf("cost: %d → %d", before, w.Stock(0, gold))
	}
	w.Step([]Command{{Player: 0, Kind: CmdCancelTrain, TargetBuilding: forge.ID, Count: 0}})
	if w.Stock(0, gold) != before || forge.Queue() != nil {
		t.Fatal("cancel did not refund the research")
	}
}
