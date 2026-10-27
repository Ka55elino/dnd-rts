package sim

import (
	"dnd-rts/internal/data"
	"dnd-rts/internal/fixed"
)

// combatCell is the spatial hash cell for target searches (coarser than the
// separation hash, since search radii reach archer range).
var combatCell = fixed.FromInt(8)

// nearestEnemyHorde returns the closest living enemy horde whose anchor is
// within maxDist, or nil. Ties go to the lower ID (iteration order).
func (w *World) nearestEnemyHorde(h *Horde, maxDist fixed.Fixed) *Horde {
	var best *Horde
	bestSq := maxDist.Mul(maxDist)
	for _, e := range w.Hordes {
		if e.Owner == h.Owner || !e.Alive() || !h.canTarget(e) {
			continue
		}
		if m := h.Unit.Ranged; m != nil && m.MinRange > 0 && h.Unit.Melee == nil && h.Pos.DistSq(e.Pos) < (m.MinRange+engageBase).Mul(m.MinRange+engageBase) {
			continue // inside the dead zone: look for something farther
		}
		if d := h.Pos.DistSq(e.Pos); d <= bestSq {
			if best == nil || d < bestSq {
				best, bestSq = e, d
			}
		}
	}
	return best
}

// nearestEnemySoldier returns the closest living enemy soldier within radius.
// Ties go to the lower soldier index, so the result is deterministic.
func (w *World) nearestEnemySoldier(s *Soldier, radius fixed.Fixed) *Soldier {
	sh := w.combatHash
	c0, r0 := sh.cellOf(w, s.Pos)
	span := radius.Div(sh.cell).Ceil().Int()
	limitSq := radius.Mul(radius)

	var best *Soldier
	bestSq := limitSq
	bestIdx := int32(-1)
	for r := max(0, r0-span); r <= min(sh.rows-1, r0+span); r++ {
		for c := max(0, c0-span); c <= min(sh.cols-1, c0+span); c++ {
			for j := sh.head[r*sh.cols+c]; j >= 0; j = sh.next[j] {
				e := w.Soldiers[j]
				if e.Horde.Owner == s.Horde.Owner || !e.Alive() || !meleeReaches(s.Horde, e.Horde) {
					continue
				}
				d := s.Pos.DistSq(e.Pos)
				if d > limitSq {
					continue
				}
				if d < bestSq || (d == bestSq && (bestIdx < 0 || j < bestIdx)) {
					best, bestSq, bestIdx = e, d, j
				}
			}
		}
	}
	return best
}

// removeDead drops killed soldiers and empty hordes, keeping ID order, and
// re-packs formation slots.
func (w *World) removeDead() {
	w.awardKills()
	alive := w.Soldiers[:0]
	died := false
	for _, s := range w.Soldiers {
		if s.Alive() {
			alive = append(alive, s)
		} else {
			died = true
		}
	}
	// Clear the tail so dead soldiers can be garbage collected.
	for i := len(alive); i < len(w.Soldiers); i++ {
		w.Soldiers[i] = nil
	}
	w.Soldiers = alive
	if !died {
		return
	}

	hordes := w.Hordes[:0]
	for _, h := range w.Hordes {
		kept := h.Soldiers[:0]
		for _, s := range h.Soldiers {
			if s.Alive() {
				s.Slot = len(kept)
				kept = append(kept, s)
			}
		}
		h.Soldiers = kept
		if !h.Alive() {
			w.recordFallen(h)
		}
		if h.Alive() {
			hordes = append(hordes, h)
		}
	}
	for i := len(hordes); i < len(w.Hordes); i++ {
		w.Hordes[i] = nil
	}
	w.Hordes = hordes
}

// forEachSoldierNear calls fn for every soldier bucketed within radius of p
// (cell granularity; fn must check the exact distance). Order is deterministic.
func (w *World) forEachSoldierNear(p fixed.Vec2, radius fixed.Fixed, fn func(*Soldier)) {
	sh := w.combatHash
	c0, r0 := sh.cellOf(w, p)
	span := radius.Div(sh.cell).Ceil().Int()
	for r := max(0, r0-span); r <= min(sh.rows-1, r0+span); r++ {
		for c := max(0, c0-span); c <= min(sh.cols-1, c0+span); c++ {
			for j := sh.head[r*sh.cols+c]; j >= 0; j = sh.next[j] {
				fn(w.Soldiers[j])
			}
		}
	}
}

var (
	spearReach    = fixed.FromFraction(15, 10) // beyond the spearman's radius
	spearCooldown = uint32(TickRate)           // one spear hit per enemy per second
	spearMinSpeed = fixed.One                  // slower than this: no extra damage
)

// updateSpearWalls: enemies running into a spear wall take pierce damage that
// grows with their speed; a charge into it is broken.
func (w *World) updateSpearWalls() {
	for _, h := range w.Hordes {
		if !h.Rooted() || len(h.Soldiers) == 0 {
			continue
		}
		per := h.Soldiers[0].mods.speedDamage
		if per <= 0 {
			continue
		}
		for _, s := range h.Soldiers {
			if !s.Alive() {
				continue
			}
			w.forEachSoldierNear(s.Pos, h.Unit.Radius+spearReach+fixed.One, func(e *Soldier) {
				if e.Horde.Owner == h.Owner || !e.Alive() || e.spearUntil > w.Tick || !meleeReaches(h, e.Horde) {
					return
				}
				if e.Pos.Dist(s.Pos) > h.Unit.Radius+e.Horde.Unit.Radius+spearReach {
					return
				}
				v := e.Horde.speedNow(w)
				if v < spearMinSpeed {
					return
				}
				dmg := per.Mul(v)
				if a := h.Unit.Melee; a != nil {
					dmg = attackBonus(a, e.Horde.Unit, dmg)
				}
				w.hitSoldier(s, h.Owner, e, dmg, data.DmgPierce, true)
				e.spearUntil = w.Tick + spearCooldown
				e.momentum = momentumMin
				if e.Horde.charging { // the charge breaks on the spears
					e.Horde.charging = false
					e.Horde.moving = false
				}
			})
		}
	}
}
