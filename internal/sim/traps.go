package sim

import "dnd-rts/internal/fixed"

// Traps (docs/FACTIONS.md: Dark Elf traps and mines, the Runemaster's Rune Trap). A trap is a
// horde of a unit with Unit.Trap, placed by an ability (a summon with maxAlive). It never moves,
// is invisible (its passive) — enemies see it only with detection — and goes off when an enemy
// ground soldier steps within its radius: the effects of its Trap.Ability at its spot, then it is
// gone (no XP for anyone).

// IsTrap reports whether the horde is a trap.
func (h *Horde) IsTrap() bool { return h.Unit.Trap != nil }

// updateTraps sets off the traps enemies stepped on.
func (w *World) updateTraps() {
	for _, h := range w.Hordes {
		t := h.Unit.Trap
		if t == nil || !h.Alive() || h.dismissed {
			continue
		}
		r2 := t.Radius.Mul(t.Radius)
		var hit *Soldier
		w.forEachSoldierNear(h.Pos, t.Radius, func(s *Soldier) {
			if hit == nil && s.Alive() && s.Horde.Owner != h.Owner && !s.Horde.Airborne() && !s.Horde.IsTrap() &&
				!s.Horde.InTunnel() && s.Pos.DistSq(h.Pos) <= r2 {
				hit = s
			}
		})
		if hit == nil {
			continue
		}
		w.triggerTrap(h, hit.Horde)
	}
}

// triggerTrap sets the trap off on the horde that stepped on it.
func (w *World) triggerTrap(h, victim *Horde) {
	a := w.Catalog.Abilities[h.Unit.Trap.Ability]
	for k := range a.Effects {
		e := &a.Effects[k]
		if e.Mode != "" {
			continue
		}
		w.runEffect(h.Owner, nil, a, e, h.Pos, victim)
	}
	w.event(a, CastEvent{Owner: h.Owner, Pos: h.Pos, Radius: abilityRadius(a), DamageType: firstDamageType(a.Effects, "")})
	h.dismissed = true
	for _, s := range h.Soldiers {
		s.HP = 0
	}
}

// buildAt puts up a finished building of owner next to point (Undermine's tunnel).
func (w *World) buildAt(owner PlayerID, id string, point fixed.Vec2) {
	def, ok := w.Catalog.Buildings[id]
	if !ok {
		return
	}
	facing := fixed.Fixed(0)
	if b := w.foreignBuildingNear(owner, point); b != nil {
		// just outside the target's footprint, on the caster's side of it
		facing = point.Sub(b.Pos).Angle()
		point = b.Closest(point).Add(fixed.FromAngle(facing).Scale(max(def.Footprint.X, def.Footprint.Y)/2 + fixed.FromInt(2)))
	}
	for k := 0; k < 12; k++ {
		p := point.Add(fixed.FromAngle(fixed.Pi.MulInt(2 * k).DivInt(12)).Scale(fixed.FromInt(3 * (k + 1) / 2)))
		if k == 0 {
			p = point
		}
		if w.fitsFree(def, p, facing, false) {
			if _, err := w.SpawnBuilding(owner, id, p, facing); err == nil {
				w.rebuildBlocked()
			}
			return
		}
	}
}
