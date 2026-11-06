package sim

import (
	"os"
	"testing"

	"dnd-rts/internal/data"
	"dnd-rts/internal/fixed"
	"dnd-rts/internal/terrain"
)

// newWorldWith: a skirmish of two factions (fog off).
func newWorldWith(t *testing.T, f0, f1 string) *World {
	t.Helper()
	cat, err := data.Load(os.DirFS("../../content"))
	if err != nil {
		t.Fatal(err)
	}
	hf, err := terrain.ForMap(os.DirFS("../../content"), cat.Maps["plains"])
	if err != nil {
		t.Fatal(err)
	}
	w := NewWorld(cat, hf)
	if err := SetupSkirmish(w, "plains", []PlayerSetup{{0, f0}, {1, f1}}); err != nil {
		t.Fatal(err)
	}
	w.SetFog(false)
	return w
}

func steps(w *World, n int) {
	for range n {
		w.Step(nil)
	}
}

func TestTrapGoesOffOnEnemy(t *testing.T) {
	w := newWorldWith(t, "dark_elves", "empire")
	p := openSpot(w, -60, 60)
	weaver, _ := w.SpawnHorde(0, "webweavers", p, 0)
	w.Step([]Command{{Player: 0, Kind: CmdCast, Hordes: []HordeID{weaver.ID}, Name: "place_mine", Target: p.Add(fixed.V2i(8, 0))}})
	var mine *Horde
	for _, h := range w.Hordes {
		if h.IsTrap() {
			mine = h
		}
	}
	if mine == nil {
		t.Fatal("no mine placed")
	}
	w.Step(nil)
	if !mine.Hidden() || w.VisibleTo(mine, 1) {
		t.Fatal("the mine is visible to the enemy")
	}
	enemy, _ := w.SpawnHorde(1, "swordsmen", mine.Pos.Add(fixed.V2i(12, 0)), 0)
	hp := totalHP(enemy)
	w.Step([]Command{{Player: 1, Kind: CmdMove, Hordes: []HordeID{enemy.ID}, Target: mine.Pos.Add(fixed.V2i(-10, 0))}})
	steps(w, 4*TickRate)
	if mine.Alive() {
		t.Fatal("the mine did not go off")
	}
	if totalHP(enemy) >= hp {
		t.Error("the blast hurt nobody")
	}
}

func TestTrapMaxAlive(t *testing.T) {
	w := newWorldWith(t, "dark_elves", "empire")
	p := openSpot(w, -60, 60)
	weaver, _ := w.SpawnHorde(0, "webweavers", p, 0)
	for k := range 7 {
		w.Step([]Command{{Player: 0, Kind: CmdCast, Hordes: []HordeID{weaver.ID}, Name: "place_mine", Target: p.Add(fixed.V2i(6, 3*k-9))}})
		weaver.abilityCd[weaver.AbilityIndex("place_mine")] = 0
		w.Step(nil)
	}
	n := 0
	for _, h := range w.Hordes {
		if h.IsTrap() && h.Alive() {
			n++
		}
	}
	if n != 5 {
		t.Fatalf("%d mines alive, want 5 (maxAlive)", n)
	}
}

func TestCurseTheGroundRaisesTheDead(t *testing.T) {
	w := newWorldWith(t, "alkars", "empire")
	p := openSpot(w, -60, 60)
	acolytes, _ := w.SpawnHorde(0, "dark_acolytes", p, 0)
	victims, _ := w.SpawnHorde(1, "swordsmen", p.Add(fixed.V2i(14, 0)), 0)
	w.Step([]Command{{Player: 0, Kind: CmdCast, Hordes: []HordeID{acolytes.ID}, Name: "curse_the_ground", Target: victims.Pos}})
	before := len(w.Hordes)
	w.Step([]Command{{Player: 0, Kind: CmdDevKill, Hordes: []HordeID{victims.ID}}})
	w.Step(nil)
	risen := 0
	for _, h := range w.Hordes {
		if h.Owner == 0 && h.Def.ID == "risen_skeletons" {
			risen++
			if h.expires == 0 {
				t.Error("risen skeletons are not temporary")
			}
		}
	}
	if want := victims.Def.Size / 3; risen != want {
		t.Fatalf("risen hordes %d (hordes %d → %d), want %d", risen, before, len(w.Hordes), want)
	}
}

func TestChannelBreaksWhenTheCasterWalks(t *testing.T) {
	w := newWorldWith(t, "alkars", "empire")
	p := openSpot(w, -60, 60)
	acolytes, _ := w.SpawnHorde(0, "dark_acolytes", p, 0)
	w.Step([]Command{{Player: 0, Kind: CmdCast, Hordes: []HordeID{acolytes.ID}, Name: "curse_the_ground", Target: p.Add(fixed.V2i(10, 0))}})
	if len(w.Zones) != 1 {
		t.Fatalf("zones %d, want 1", len(w.Zones))
	}
	w.Step([]Command{{Player: 0, Kind: CmdMove, Hordes: []HordeID{acolytes.ID}, Target: p.Add(fixed.V2i(-30, 0))}})
	w.Step(nil)
	if len(w.Zones) != 0 {
		t.Error("the channelled zone outlived the caster walking away")
	}
}

func TestTreesBlockAndRegrow(t *testing.T) {
	w := newTestWorld(t)
	if len(w.Trees) < 100 {
		t.Fatalf("only %d trees from the map's forests", len(w.Trees))
	}
	tr := &w.Trees[0]
	g := w.Nav.Grid()
	if g.WalkableAt(tr.Pos) {
		t.Fatal("a grown tree does not block its cell")
	}
	w.damageTrees(tr.Pos, fixed.FromInt(1), fixed.FromInt(500), data.DmgFire)
	if tr.Grown {
		t.Fatal("fire did not fell the tree")
	}
	steps(w, treeRegrowTicks+TickRate)
	if !tr.Grown {
		t.Error("the tree did not grow back")
	}
}

func TestEntsNeedTrees(t *testing.T) {
	w := newWorldWith(t, "high_elves", "empire")
	king, _ := w.SpawnHorde(0, "elf_king", openSpot(w, -60, 60), 0)
	king.gainXP(1400)
	i := king.AbilityIndex("awakening_of_the_ents")
	if !w.castBlocked(king, i, king.Pos) {
		t.Fatal("ents can be woken with no tree around")
	}
	tree := w.Trees[0].Pos
	n := len(w.Hordes)
	w.cast(king, i, tree, nil)
	if got := len(w.Hordes) - n; got != 3 {
		t.Fatalf("%d ents woke, want 3", got)
	}
}

func TestMoonwellPlantsTrees(t *testing.T) {
	w := newWorldWith(t, "high_elves", "empire")
	p := openSpot(w, -60, 60)
	b, err := w.SpawnBuilding(0, "moonwell", p, 0)
	if err != nil {
		t.Fatal(err)
	}
	n := len(w.Trees)
	steps(w, 3*20*TickRate+TickRate)
	if len(w.Trees) <= n {
		t.Fatal("the moonwell planted nothing")
	}
	_ = b
}

func TestTunnelCarriesAHorde(t *testing.T) {
	w := newWorldWith(t, "dwarves", "empire")
	keep := w.fortressOf(0)
	if !w.IsTunnel(keep) {
		t.Fatal("the dwarf keep is not a tunnel entrance")
	}
	far := openSpot(w, 0, 60)
	exit, err := w.SpawnBuilding(0, "tunnel", far, 0)
	if err != nil {
		t.Fatal(err)
	}
	h, _ := w.SpawnHorde(0, "axe_dwarves", keep.approach(keep.Pos.Add(fixed.V2i(40, 0)), fixed.FromInt(3)), 0)
	w.Step([]Command{{Player: 0, Kind: CmdMove, Hordes: []HordeID{h.ID}, Target: exit.Pos}})
	under := false
	for range 30 * TickRate {
		w.Step(nil)
		under = under || h.InTunnel()
		if !h.InTunnel() && exit.DistTo(h.Pos) < fixed.FromInt(10) {
			break
		}
	}
	if !under {
		t.Fatal("the horde never went underground")
	}
	if exit.DistTo(h.Pos) >= fixed.FromInt(10) {
		t.Fatalf("the horde came out at %v, far from the exit tunnel", h.Pos)
	}
}

func TestFogHidesAndRevealShows(t *testing.T) {
	w := newTestWorld(t)
	w.SetFog(true)
	far, _ := w.SpawnHorde(1, "raiders", openSpot(w, 40, -60), 0)
	w.Step(nil)
	if w.Sees(0, far) {
		t.Fatal("player 0 sees an enemy far from its units")
	}
	if fs := w.FogState(0); fs == nil || fs[w.fogIndex(far.Pos)] != FogUnexplored {
		t.Fatal("the far cell is not unexplored")
	}
	w.addReveal(0, far.Pos, fixed.FromInt(15), fixed.FromInt(5))
	if !w.Sees(0, far) {
		t.Fatal("a reveal does not show the enemy")
	}
	steps(w, 6*TickRate)
	if w.Sees(0, far) {
		t.Fatal("the reveal did not run out")
	}
	if w.FogState(0)[w.fogIndex(far.Pos)] != FogExplored {
		t.Error("a revealed cell is not left explored")
	}
}

func TestSmokeHidesFromOutside(t *testing.T) {
	w := newWorldWith(t, "dwarves", "empire")
	p := openSpot(w, -60, 60)
	gunners, _ := w.SpawnHorde(0, "gunners", p, 0)
	enemy, _ := w.SpawnHorde(1, "swordsmen", p.Add(fixed.V2i(25, 0)), 0)
	w.Zones = append(w.Zones, &Zone{ID: 99, Owner: 0, Pos: p, Radius: fixed.FromInt(8), ticks: 100, interval: 20, smoke: true})
	w.Step(nil)
	if w.VisibleTo(gunners, 1) {
		t.Fatal("the enemy outside sees the hordes in the smoke")
	}
	_ = enemy
}

func TestUndermineDigsIntoAnEnemyBuilding(t *testing.T) {
	w := newWorldWith(t, "dwarves", "empire")
	eng, _ := w.SpawnHorde(0, "engineer", openSpot(w, 60, 40), 0)
	eng.gainXP(1400)
	i := eng.AbilityIndex("undermine")
	if !w.castBlocked(eng, i, openSpot(w, 0, 40)) {
		t.Fatal("undermine allowed with no foreign building")
	}
	enemyKeep := w.fortressOf(1)
	n := len(w.Buildings)
	w.cast(eng, i, enemyKeep.Closest(eng.Pos), nil)
	if len(w.Buildings) != n+1 {
		t.Fatal("no passage dug")
	}
	if b := w.Buildings[len(w.Buildings)-1]; b.Owner != 0 || !w.IsTunnel(b) || b.DistTo(enemyKeep.Pos) > fixed.FromInt(30) {
		t.Fatalf("passage %+v: not an own tunnel next to the enemy keep", b.Def.ID)
	}
}

func TestGroupUpgradeSwapIsFree(t *testing.T) {
	w := newWorldWith(t, "high_elves", "empire")
	p := openSpot(w, -60, 60)
	tower, _ := w.SpawnBuilding(0, "enchantment_tower", p, 0)
	w.Step([]Command{{Player: 0, Kind: CmdDevGive, Name: "gold", Count: 5000}, {Player: 0, Kind: CmdDevGive, Name: "wood", Count: 5000}})
	w.Step([]Command{{Player: 0, Kind: CmdResearch, TargetBuilding: tower.ID, Name: "elemental_blades_fire"}})
	steps(w, 40*TickRate)
	if !w.HasUpgrade(0, "elemental_blades_fire") {
		t.Fatal("fire blades not researched")
	}
	gold := w.economyOf(0).stock[0]
	w.Step([]Command{{Player: 0, Kind: CmdResearch, TargetBuilding: tower.ID, Name: "elemental_blades_ice"}})
	steps(w, 6*TickRate)
	if !w.HasUpgrade(0, "elemental_blades_ice") || w.HasUpgrade(0, "elemental_blades_fire") {
		t.Fatal("the swap to ice did not replace fire")
	}
	if w.economyOf(0).stock[0] < gold {
		t.Error("the swap cost gold")
	}
}
