package sim

import (
	"dnd-rts/internal/data"
	"dnd-rts/internal/fixed"
)

// Trees (docs/FACTIONS.md: trees as map objects). Forests come from the map; spirits and moonwells
// plant more. A grown tree blocks its nav cell, gives cover (Forest Shadow, elven camouflage), is
// the target of Sacred Tree and the source of ents. Fire and crushing area damage fells it; a
// felled tree (and a sapling) grows back after a while, faster near Power of Growth.

// Tree is a tree on the map.
type Tree struct {
	ID    int
	Pos   fixed.Vec2
	HP    fixed.Fixed
	Grown bool        // false: a sapling or a stump, growing (back)
	grow  fixed.Fixed // ticks until it is grown (growth auras count faster)
}

var (
	treeHealth = fixed.FromInt(120)
	// treeSnap: tree-targeted abilities find a tree this close to the clicked point.
	treeSnap = fixed.FromInt(8)
	// treeSpacing: no two trees closer than this.
	treeSpacing = fixed.FromInt(3)
)

const (
	treeRegrowTicks  = 90 * TickRate // a felled tree
	treeSaplingTicks = 25 * TickRate // a planted one
)

// nearTree reports whether a grown tree stands within r of p.
func (w *World) nearTree(p fixed.Vec2, r fixed.Fixed) bool { return w.nearestTree(p, r) != nil }

// nearestTree is the closest grown tree within r of p, or nil.
func (w *World) nearestTree(p fixed.Vec2, r fixed.Fixed) *Tree {
	var best *Tree
	bestD := r.Mul(r)
	for i := range w.Trees {
		if t := &w.Trees[i]; t.Grown {
			if d := t.Pos.DistSq(p); d <= bestD {
				best, bestD = t, d
			}
		}
	}
	return best
}

// treesNear returns up to n grown trees within r of p, nearest first.
func (w *World) treesNear(p fixed.Vec2, r fixed.Fixed, n int) []*Tree {
	var out []*Tree
	r2 := r.Mul(r)
	for i := range w.Trees {
		if t := &w.Trees[i]; t.Grown && t.Pos.DistSq(p) <= r2 {
			out = append(out, t)
		}
	}
	// insertion sort by distance, then ID (few trees)
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j].Pos.DistSq(p) < out[j-1].Pos.DistSq(p); j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	if len(out) > n {
		out = out[:n]
	}
	return out
}

// spawnForests scatters the map's forests (deterministic: a small LCG per forest).
func (w *World) spawnForests(forests []data.MapForest) {
	for fi, f := range forests {
		seed := uint32(f.Seed) ^ uint32(fi*7919+17)
		next := func() fixed.Fixed { // [0, 1)
			seed = seed*1664525 + 1013904223
			return fixed.Fixed(seed >> 16)
		}
		for k, tries := 0, 0; k < f.Count && tries < f.Count*8; tries++ {
			a := next().Mul(fixed.Pi.MulInt(2))
			d := f.Radius.Mul(next().Sqrt()) // uniform over the disc
			p := f.Pos.Add(fixed.FromAngle(a).Scale(d))
			if w.addTree(p, true) {
				k++
			}
		}
	}
	w.rebuildBlocked()
}

// addTree puts a tree at p if the ground is free (terrain walkable, no building, not crowded).
func (w *World) addTree(p fixed.Vec2, grown bool) bool {
	g := w.Nav.Grid()
	p = w.clampToMap(p)
	if w.navTerrain[g.IndexAt(p)] {
		return false
	}
	for _, b := range w.Buildings {
		if b.DistTo(p) < NavCellSize {
			return false
		}
	}
	s2 := treeSpacing.Mul(treeSpacing)
	for i := range w.Trees {
		if w.Trees[i].Pos.DistSq(p) < s2 {
			return false
		}
	}
	t := Tree{ID: w.nextTree, Pos: p, HP: treeHealth, Grown: grown}
	if !grown {
		t.grow = fixed.FromInt(treeSaplingTicks)
	}
	w.nextTree++
	w.Trees = append(w.Trees, t)
	return true
}

// plantTrees plants n saplings around center (within radius; at the centre when radius is 0).
func (w *World) plantTrees(center fixed.Vec2, n int, radius fixed.Fixed) {
	for k := 0; k < n; k++ {
		p := center
		if radius > 0 {
			// a golden-angle spiral: deterministic, spreads out
			a := fixed.FromFraction(2399, 1000).MulInt(int(w.nextTree) + k)
			p = center.Add(fixed.FromAngle(a).Scale(radius.Mul(fixed.FromFraction(k+1, n+1).Sqrt())))
		}
		if !w.soldierNear(p, fixed.FromInt(2)) {
			w.addTree(p, false)
		}
	}
}

// soldierNear: a soldier stands within r of p (don't grow a tree on top of anyone).
func (w *World) soldierNear(p fixed.Vec2, r fixed.Fixed) bool {
	found := false
	r2 := r.Mul(r)
	w.forEachSoldierNear(p, r, func(s *Soldier) {
		found = found || (s.Alive() && !s.Horde.Unit.Flying && s.Pos.DistSq(p) <= r2)
	})
	return found
}

// damageTrees: fire and crushing area damage hurts trees (fire twice as much); a tree at 0 HP
// is felled and starts growing back.
func (w *World) damageTrees(center fixed.Vec2, radius, dmg fixed.Fixed, dtype string) {
	switch dtype {
	case data.DmgFire:
		dmg = dmg.MulInt(2)
	case data.DmgCrush:
	default:
		return
	}
	r2 := radius.Mul(radius)
	felled := false
	for i := range w.Trees {
		if t := &w.Trees[i]; t.Grown && t.Pos.DistSq(center) <= r2 {
			if t.HP -= dmg; t.HP <= 0 {
				t.Grown, t.HP, t.grow = false, treeHealth, fixed.FromInt(treeRegrowTicks)
				felled = true
			}
		}
	}
	if felled {
		w.rebuildBlocked()
	}
}

// updateTrees grows saplings and stumps (Power of Growth auras speed it up) and lets moonwells
// plant new trees.
func (w *World) updateTrees() {
	if len(w.Trees) == 0 && !w.anyGrower() {
		return
	}
	// Growth auras.
	type grower struct {
		pos    fixed.Vec2
		r2     fixed.Fixed
		growth fixed.Fixed
	}
	var gs []grower
	for _, h := range w.Hordes {
		if len(h.Soldiers) == 0 {
			continue
		}
		for i, a := range h.abilities {
			if a.Kind == data.AbilityAura && h.abilityUnlocked(i) {
				if g := a.ModsFor(h.Mode()).TreeGrowth; g > 0 {
					gs = append(gs, grower{h.Pos, a.Radius.Mul(a.Radius), g})
				}
			}
		}
	}
	grew := false
	for i := range w.Trees {
		t := &w.Trees[i]
		if t.Grown {
			continue
		}
		step := fixed.One
		for _, g := range gs {
			if t.Pos.DistSq(g.pos) <= g.r2 {
				step += g.growth
			}
		}
		t.grow -= step
		if t.grow <= 0 {
			if w.soldierNear(t.Pos, fixed.FromInt(2)) {
				t.grow = fixed.FromInt(TickRate) // someone stands there: try again in a second
				continue
			}
			t.Grown, t.HP = true, treeHealth
			grew = true
		}
	}
	// Moonwells.
	for _, b := range w.Buildings {
		g := b.Def.GrowTrees
		if g == nil || !b.Alive() || b.UnderConstruction() || b.Owner == Neutral {
			continue
		}
		if b.growCd > 0 {
			b.growCd--
			continue
		}
		b.growCd = cooldownTicks(g.Every)
		r := g.Radius + max(b.Def.Footprint.X, b.Def.Footprint.Y)/2
		n := 0
		r2 := r.Mul(r)
		for i := range w.Trees {
			if w.Trees[i].Pos.DistSq(b.Pos) <= r2 {
				n++
			}
		}
		if n < g.Max {
			w.plantTrees(b.Pos, 1, r)
		}
	}
	if grew {
		w.rebuildBlocked()
	}
}

func (w *World) anyGrower() bool {
	for _, b := range w.Buildings {
		if b.Def.GrowTrees != nil {
			return true
		}
	}
	return false
}

// stampTrees blocks the nav cells of grown trees (rebuildBlocked).
func (w *World) stampTrees() {
	g := w.Nav.Grid()
	for i := range w.Trees {
		if t := &w.Trees[i]; t.Grown {
			g.Blocked[g.IndexAt(t.Pos)] = true
		}
	}
}
