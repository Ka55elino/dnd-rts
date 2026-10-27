package sim

import (
	"encoding/json"
	"hash/fnv"
	"os"
	"testing"

	"dnd-rts/internal/data"
	"dnd-rts/internal/fixed"
	"dnd-rts/internal/terrain"
)

func newTestWorld(t *testing.T) *World {
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
	if err := SetupSkirmish(w, "plains", []PlayerSetup{{0, "empire"}, {1, "warband"}}); err != nil {
		t.Fatal(err)
	}
	return w
}

func TestSetupSpawnsFactions(t *testing.T) {
	w := newTestWorld(t)
	if len(w.Hordes) != 10 { // 5 empire + 5 warband (incl. heroes, builders, harpies)
		t.Fatalf("hordes = %d, want 10", len(w.Hordes))
	}
	total := 0
	for _, h := range w.Hordes {
		total += h.Def.Size
		if len(h.Soldiers) != h.Def.Size {
			t.Errorf("horde %d has %d soldiers", h.ID, len(h.Soldiers))
		}
	}
	if len(w.Soldiers) != total {
		t.Fatalf("soldiers = %d, want %d", len(w.Soldiers), total)
	}
}

func TestMoveReachesTarget(t *testing.T) {
	w := newTestWorld(t)
	h := w.Hordes[0]
	target := h.Pos.Add(fixed.V2i(30, 40))
	w.Step([]Command{{Player: h.Owner, Kind: CmdMove, Hordes: []HordeID{h.ID}, Target: target}})
	for i := 0; i < 20*30 && h.IsMoving(); i++ {
		w.Step(nil)
	}
	if h.IsMoving() || h.Pos != target {
		t.Fatalf("horde at %v, want %v (moving=%v)", h.Pos, target, h.IsMoving())
	}
	// Give soldiers time to settle, then check they are in formation.
	for i := 0; i < 20*5; i++ {
		w.Step(nil)
	}
	for _, s := range h.Soldiers {
		if d := s.Pos.Dist(h.slotPos(s.Slot)); d > fixed.FromFraction(1, 10) {
			t.Errorf("soldier %d is %v from its slot", s.ID, d)
		}
	}
}

func TestCannotCommandEnemy(t *testing.T) {
	w := newTestWorld(t)
	var enemy *Horde
	for _, h := range w.Hordes {
		if h.Owner == 1 {
			enemy = h
			break
		}
	}
	start := enemy.Pos
	w.Step([]Command{{Player: 0, Kind: CmdMove, Hordes: []HordeID{enemy.ID}, Target: fixed.V2i(0, 0)}})
	w.Step(nil)
	if enemy.Pos != start || enemy.IsMoving() {
		t.Fatal("player 0 moved an enemy horde")
	}
}

func TestMoveTargetClampedToMap(t *testing.T) {
	w := newTestWorld(t)
	h := w.Hordes[0]
	w.Step([]Command{{Player: h.Owner, Kind: CmdMove, Hordes: []HordeID{h.ID}, Target: fixed.V2i(10000, 0)}})
	if h.Target().X.Int() >= w.Terrain.Width/2 {
		t.Fatalf("target not clamped: %v", h.Target())
	}
}

// Same commands → same world on every platform.
func TestSimDeterminismGolden(t *testing.T) {
	const golden uint64 = 0x8b82c812895c9dd7

	w := newTestWorld(t)
	for tick := 0; tick < 600; tick++ {
		var cmds []Command
		if tick%100 == 0 {
			for _, h := range w.Hordes {
				target := fixed.V2i(int(h.ID)*17%120-60, tick%160-80)
				cmds = append(cmds, Command{Player: h.Owner, Kind: CmdMove, Hordes: []HordeID{h.ID}, Target: target})
			}
		}
		w.Step(cmds)
	}

	f := fnv.New64a()
	var b [8]byte
	put := func(v int64) {
		for i := range b {
			b[i] = byte(v >> (8 * i))
		}
		f.Write(b[:])
	}
	for _, s := range w.Soldiers {
		put(s.Pos.X.Raw())
		put(s.Pos.Y.Raw())
		put(s.Rot.Raw())
	}
	if got := f.Sum64(); got != golden {
		t.Fatalf("world hash = %#x, want %#x", got, golden)
	}
}

// wallWorld: flat 128x128 map with a steep ridge across X=0 and a gap near Y=+40.
func wallWorld(t *testing.T) *World {
	t.Helper()
	cat, err := data.Load(os.DirFS("../../content"))
	if err != nil {
		t.Fatal(err)
	}
	hf, err := terrain.New(128, 128, fixed.FromInt(2), fixed.FromInt(40))
	if err != nil {
		t.Fatal(err)
	}
	for r := 0; r < hf.Rows; r++ {
		for c := 0; c < hf.Cols; c++ {
			p := hf.SamplePos(c, r)
			inRidge := p.X.Abs() <= fixed.FromInt(4)
			inGap := p.Y >= fixed.FromInt(32) && p.Y <= fixed.FromInt(48)
			if inRidge && !inGap {
				hf.Samples[r*hf.Cols+c] = 65535
			}
		}
	}
	return NewWorld(cat, hf)
}

func TestPathAroundWall(t *testing.T) {
	w := wallWorld(t)
	h, err := w.SpawnHorde(0, "swordsmen", fixed.V2i(-30, -30), 0)
	if err != nil {
		t.Fatal(err)
	}
	target := fixed.V2i(30, -30)
	if w.Nav.Grid().LineWalkable(h.Pos, target) {
		t.Fatal("test setup: straight line should be blocked by the ridge")
	}
	w.Step([]Command{{Player: 0, Kind: CmdMove, Hordes: []HordeID{h.ID}, Target: target}})

	g := w.Nav.Grid()
	for i := 0; i < 20*60 && h.IsMoving(); i++ {
		w.Step(nil)
		if !g.WalkableAt(h.Pos) {
			t.Fatalf("tick %d: horde anchor on blocked ground at %v", w.Tick, h.Pos)
		}
		for _, s := range h.Soldiers {
			if !g.WalkableAt(s.Pos) {
				t.Fatalf("tick %d: soldier %d on blocked ground at %v", w.Tick, s.ID, s.Pos)
			}
		}
	}
	if h.IsMoving() || h.Pos.Dist(target) > fixed.One {
		t.Fatalf("horde did not reach target: at %v moving=%v", h.Pos, h.IsMoving())
	}
}

func TestSeparationPushesApart(t *testing.T) {
	w := wallWorld(t)
	a, _ := w.SpawnHorde(0, "swordsmen", fixed.V2i(-40, -40), 0)
	b, _ := w.SpawnHorde(1, "raiders", fixed.V2i(-40, -40), 0) // spawned on top of each other
	for i := 0; i < 20; i++ {
		w.Step(nil)
	}
	_ = a
	_ = b
	for i, s1 := range w.Soldiers {
		for _, s2 := range w.Soldiers[i+1:] {
			minD := s1.Horde.Unit.Radius + s2.Horde.Unit.Radius
			if s1.Pos.Dist(s2.Pos) < minD.Mul(fixed.FromFraction(9, 10)) {
				t.Fatalf("soldiers %d and %d still overlap: %v", s1.ID, s2.ID, s1.Pos.Dist(s2.Pos))
			}
		}
	}
}

func TestMeleeCombatKills(t *testing.T) {
	w := wallWorld(t)
	a, _ := w.SpawnHorde(0, "swordsmen", fixed.V2i(-40, -40), 0)
	b, _ := w.SpawnHorde(1, "raiders", fixed.V2i(-10, -40), fixed.Pi)
	w.Step([]Command{{Player: 0, Kind: CmdAttack, Hordes: []HordeID{a.ID}, TargetHorde: b.ID}})

	startA, startB := len(a.Soldiers), len(b.Soldiers)
	for i := 0; i < 20*120 && a.Alive() && b.Alive(); i++ {
		w.Step(nil)
	}
	if a.Alive() && b.Alive() {
		t.Fatalf("no horde was destroyed: %d/%d vs %d/%d", len(a.Soldiers), startA, len(b.Soldiers), startB)
	}
	// Dead hordes and soldiers are removed; slots stay packed.
	for _, h := range w.Hordes {
		if !h.Alive() {
			t.Fatal("empty horde left in world")
		}
		for i, s := range h.Soldiers {
			if s.Slot != i || !s.Alive() {
				t.Fatalf("horde %d: bad soldier %d slot=%d hp=%v", h.ID, s.ID, s.Slot, s.HP)
			}
		}
	}
	for i := 1; i < len(w.Soldiers); i++ {
		if w.Soldiers[i-1].ID >= w.Soldiers[i].ID {
			t.Fatal("soldiers not sorted by ID after deaths")
		}
	}
}

func TestIdleHordeAutoAggro(t *testing.T) {
	w := wallWorld(t)
	a, _ := w.SpawnHorde(0, "swordsmen", fixed.V2i(-40, -40), 0)
	b, _ := w.SpawnHorde(1, "raiders", fixed.V2i(-25, -40), fixed.Pi)
	w.Step(nil)
	if a.AttackTarget() != b || b.AttackTarget() != a {
		t.Fatal("idle hordes in range should target each other")
	}
	// An explicit move order disengages.
	w.Step([]Command{{Player: 0, Kind: CmdMove, Hordes: []HordeID{a.ID}, Target: fixed.V2i(-40, 40)}})
	if a.AttackTarget() != nil || a.IsFighting() {
		t.Fatal("move order should cancel the attack")
	}
}

func TestCannotAttackOwnHorde(t *testing.T) {
	w := wallWorld(t)
	a, _ := w.SpawnHorde(0, "swordsmen", fixed.V2i(-40, -40), 0)
	b, _ := w.SpawnHorde(0, "archers", fixed.V2i(-40, 20), 0)
	w.Step([]Command{{Player: 0, Kind: CmdAttack, Hordes: []HordeID{a.ID}, TargetHorde: b.ID}})
	if a.AttackTarget() != nil {
		t.Fatal("attacking an own horde must be ignored")
	}
}

func TestArmorReducesDamage(t *testing.T) {
	q := fixed.FromFraction(1, 4)
	if got := mitigate(fixed.FromInt(12), fixed.FromInt(2), q); got != fixed.FromInt(10) {
		t.Fatalf("12 dmg vs 2 armor = %v", got)
	}
	if got := mitigate(fixed.FromInt(8), fixed.FromInt(20), q); got != fixed.FromInt(2) {
		t.Fatalf("minimum damage should be a quarter, got %v", got)
	}
}

// A battle must play out identically everywhere (targets, damage, deaths).
func TestCombatDeterminismGolden(t *testing.T) {
	const golden uint64 = 0x25d266976df06777

	w := wallWorld(t)
	a, _ := w.SpawnHorde(0, "swordsmen", fixed.V2i(-50, -40), 0)
	kn, _ := w.SpawnHorde(0, "knights", fixed.V2i(-58, -20), 0)
	b, _ := w.SpawnHorde(1, "raiders", fixed.V2i(-36, -40), fixed.Pi)
	w.SpawnHorde(1, "wolf_riders", fixed.V2i(-36, -10), fixed.Pi)
	arch, _ := w.SpawnHorde(0, "archers", fixed.V2i(-60, -50), 0)
	w.Step([]Command{
		{Player: 0, Kind: CmdAttack, Hordes: []HordeID{arch.ID}, TargetHorde: b.ID}, // volleys
		{Player: 0, Kind: CmdAttack, Hordes: []HordeID{a.ID}, TargetHorde: b.ID},
		{Player: 0, Kind: CmdAttack, Hordes: []HordeID{kn.ID}, TargetHorde: b.ID}, // charge
	})
	for i := 0; i < 400; i++ {
		w.Step(nil)
	}

	f := fnv.New64a()
	var buf [8]byte
	put := func(v int64) {
		for i := range buf {
			buf[i] = byte(v >> (8 * i))
		}
		f.Write(buf[:])
	}
	put(int64(len(w.Soldiers)))
	for _, s := range w.Soldiers {
		put(int64(s.ID))
		put(s.Pos.X.Raw())
		put(s.Pos.Y.Raw())
		put(s.HP.Raw())
	}
	t.Logf("after 400 ticks: %d soldiers, %d hordes", len(w.Soldiers), len(w.Hordes))
	if got := f.Sum64(); got != golden {
		t.Fatalf("combat hash = %#x, want %#x", got, golden)
	}
}

func TestCavalryChargeTramplesInfantry(t *testing.T) {
	w := wallWorld(t)
	// Keep the charge point (20 past the enemy) west of the ridge at x=0.
	knights, _ := w.SpawnHorde(0, "knights", fixed.V2i(-58, -40), 0)
	foot, _ := w.SpawnHorde(1, "raiders", fixed.V2i(-36, -40), fixed.Pi)
	w.Step([]Command{{Player: 0, Kind: CmdAttack, Hordes: []HordeID{knights.ID}, TargetHorde: foot.ID}})
	if !knights.IsCharging() {
		t.Fatal("cavalry attack order should start a charge")
	}

	knockedEver := map[SoldierID]bool{}
	startHP := foot.Unit.Health.MulInt(len(foot.Soldiers))
	for i := 0; i < 20*8; i++ {
		w.Step(nil)
		for _, s := range foot.Soldiers {
			if s.Knocked() {
				knockedEver[s.ID] = true
			}
		}
	}
	if len(knockedEver) < 3 {
		t.Fatalf("charge knocked down only %d infantry", len(knockedEver))
	}
	var hp fixed.Fixed
	for _, s := range foot.Soldiers {
		hp += s.HP
	}
	if hp >= startHP {
		t.Fatal("trampling should deal damage")
	}
}

func TestCavalryDoesNotTrampleCavalry(t *testing.T) {
	w := wallWorld(t)
	knights, _ := w.SpawnHorde(0, "knights", fixed.V2i(-58, -40), 0)
	riders, _ := w.SpawnHorde(1, "wolf_riders", fixed.V2i(-36, -40), fixed.Pi)
	w.Step([]Command{{Player: 0, Kind: CmdAttack, Hordes: []HordeID{knights.ID}, TargetHorde: riders.ID}})
	for i := 0; i < 20*6; i++ {
		w.Step(nil)
		for _, s := range riders.Soldiers {
			if s.Knocked() {
				t.Fatal("cavalry must not be knocked down by a charge")
			}
		}
	}
}

func TestArcherVolleys(t *testing.T) {
	w := wallWorld(t)
	archers, _ := w.SpawnHorde(0, "archers", fixed.V2i(-55, -40), 0)
	foot, _ := w.SpawnHorde(1, "raiders", fixed.V2i(-28, -40), fixed.Pi) // 27 away: in range
	w.Step([]Command{{Player: 0, Kind: CmdAttack, Hordes: []HordeID{archers.ID}, TargetHorde: foot.ID}})
	// Keep the raiders from charging so we measure archery alone.
	foot.ordered = true
	foot.attack = nil

	var first *Projectile
	startHP := totalHP(foot)
	for i := 0; i < 20*3 && first == nil; i++ {
		w.Step(nil)
		if len(w.Projectiles) > 0 {
			first = w.Projectiles[0]
		}
	}
	if first == nil {
		t.Fatal("archers in range never fired")
	}
	if first.Arrows != len(archers.Soldiers) || first.ImpactTick <= first.LaunchTick {
		t.Fatalf("bad volley: %+v", first)
	}
	if totalHP(foot) != startHP {
		t.Fatal("damage must land on impact, not at launch")
	}
	for w.Tick <= first.ImpactTick {
		foot.ordered, foot.attack = true, nil
		w.Step(nil)
	}
	if totalHP(foot) >= startHP {
		t.Fatal("volley impact should damage the target horde")
	}
}

func TestVolleyMissesTargetThatMovedAway(t *testing.T) {
	w := wallWorld(t)
	archers, _ := w.SpawnHorde(0, "archers", fixed.V2i(-55, -40), 0)
	foot, _ := w.SpawnHorde(1, "raiders", fixed.V2i(-28, -40), fixed.Pi)
	w.Step([]Command{{Player: 0, Kind: CmdAttack, Hordes: []HordeID{archers.ID}, TargetHorde: foot.ID}})
	for len(w.Projectiles) == 0 {
		foot.ordered, foot.attack = true, nil
		w.Step(nil)
	}
	// Teleport the target far away before impact: every arrow must miss.
	startHP := totalHP(foot)
	shift := fixed.V2i(0, 40)
	foot.Pos = foot.Pos.Add(shift)
	for _, s := range foot.Soldiers {
		s.Pos = s.Pos.Add(shift)
	}
	impact := w.Projectiles[0].ImpactTick
	for w.Tick <= impact {
		foot.ordered, foot.attack = true, nil
		w.Step(nil)
	}
	if totalHP(foot) != startHP {
		t.Fatal("arrows land on the ground point; a target that left must not be hit")
	}
}

func totalHP(h *Horde) fixed.Fixed {
	var hp fixed.Fixed
	for _, s := range h.Soldiers {
		hp += s.HP
	}
	return hp
}

func TestHeroShockwave(t *testing.T) {
	w := wallWorld(t)
	hero, _ := w.SpawnHorde(0, "captain", fixed.V2i(-40, -40), 0)
	near, _ := w.SpawnHorde(1, "raiders", fixed.V2i(-35, -40), fixed.Pi) // within 9
	far, _ := w.SpawnHorde(1, "raiders", fixed.V2i(-40, 0), fixed.Pi)    // 40 away
	near.ordered, far.ordered = true, true                               // keep them still
	nearHP, farHP := totalHP(near), totalHP(far)

	w.Step([]Command{{Player: 0, Kind: CmdCast, Name: "shockwave", Hordes: []HordeID{hero.ID}}})
	if cd := hero.AbilityStates()[hero.AbilityIndex("shockwave")].Cooldown; cd == 0 {
		t.Fatal("cast not registered")
	}
	if totalHP(near) >= nearHP {
		t.Fatal("shockwave should damage nearby enemies")
	}
	knocked := 0
	for _, s := range near.Soldiers {
		if s.Knocked() {
			knocked++
		}
	}
	if knocked == 0 {
		t.Fatal("shockwave should knock enemies down")
	}
	if totalHP(far) != farHP {
		t.Fatal("enemies outside the radius must be untouched")
	}

	// On cooldown: a second cast does nothing.
	hpAfter := totalHP(near)
	w.Step([]Command{{Player: 0, Kind: CmdCast, Name: "shockwave", Hordes: []HordeID{hero.ID}}})
	if totalHP(near) < hpAfter-fixed.FromInt(200) {
		t.Fatal("ability must not fire again while on cooldown")
	}
}

func TestCannotCastEnemyAbility(t *testing.T) {
	w := wallWorld(t)
	enemyHero, _ := w.SpawnHorde(1, "warlord", fixed.V2i(-40, -40), 0)
	w.Step([]Command{{Player: 0, Kind: CmdCast, Name: "shockwave", Hordes: []HordeID{enemyHero.ID}}})
	if enemyHero.AbilityStates()[0].Cooldown != 0 {
		t.Fatal("player 0 cast an enemy hero's ability")
	}
}

func newPlainsWorld(t *testing.T) func(string, []PlayerSetup) (*World, error) {
	cat, err := data.Load(os.DirFS("../../content"))
	if err != nil {
		t.Fatal(err)
	}
	return func(mapID string, players []PlayerSetup) (*World, error) {
		hf, err := terrain.ForMap(os.DirFS("../../content"), cat.Maps[mapID])
		if err != nil {
			return nil, err
		}
		w := NewWorld(cat, hf)
		return w, SetupSkirmish(w, mapID, players)
	}
}

func TestStateHashChangesWithState(t *testing.T) {
	w := newTestWorld(t)
	h0 := w.StateHash()
	if w.StateHash() != h0 {
		t.Fatal("hash must be stable for the same state")
	}
	w.Soldiers[0].HP -= fixed.One
	if w.StateHash() == h0 {
		t.Fatal("hash must change when state changes")
	}
}

// A recorded match replays to exactly the same checkpoints.
func TestReplayVerifies(t *testing.T) {
	players := []PlayerSetup{{0, "empire"}, {1, "warband"}}
	newWorld := newPlainsWorld(t)
	w, err := newWorld("plains", players)
	if err != nil {
		t.Fatal(err)
	}
	rec := NewRecorder("plains", players)

	var own, enemy []HordeID
	for _, h := range w.Hordes {
		if h.Owner == 0 {
			own = append(own, h.ID)
		} else {
			enemy = append(enemy, h.ID)
		}
	}
	for w.Tick < 20*40 {
		var cmds []Command
		switch w.Tick {
		case 5:
			cmds = append(cmds, Command{Player: 0, Kind: CmdMove, Hordes: own, Target: fixed.V2i(0, 0)})
		case 300:
			cmds = append(cmds, Command{Player: 0, Kind: CmdAttack, Hordes: own, TargetHorde: enemy[0]})
		case 500:
			cmds = append(cmds, Command{Player: 0, Kind: CmdCast, Name: "shockwave", Hordes: own})
		}
		rec.Record(w.Tick, cmds)
		w.Step(cmds)
		rec.Checkpoint(w)
	}

	rp := rec.Replay()
	if len(rp.Checkpoints) != 40 {
		t.Fatalf("checkpoints = %d, want 40", len(rp.Checkpoints))
	}
	if err := rp.Verify(newWorld); err != nil {
		t.Fatal(err)
	}

	// Through JSON too (the saved file format).
	raw, err := json.Marshal(rp)
	if err != nil {
		t.Fatal(err)
	}
	var loaded Replay
	if err := json.Unmarshal(raw, &loaded); err != nil {
		t.Fatal(err)
	}
	if err := loaded.Verify(newWorld); err != nil {
		t.Fatalf("after JSON round trip: %v", err)
	}

	// A tampered checkpoint is reported as a desync.
	loaded.Checkpoints[10].Hash ^= 1
	if err := loaded.Verify(newWorld); err == nil {
		t.Fatal("tampered replay must fail verification")
	}
}

// fortressOf returns player p's fortress (walls come right after it in w.Buildings).
func fortressOf(w *World, p PlayerID) *Building {
	for _, b := range w.Buildings {
		if b.Owner == p && b.Def.Kind == "fortress" {
			return b
		}
	}
	return nil
}

func firstOutpost(w *World) *Building {
	for _, b := range w.Buildings {
		if b.Def.Kind == "outpost" {
			return b
		}
	}
	return nil
}
