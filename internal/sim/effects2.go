package sim

import (
	"dnd-rts/internal/data"
	"dnd-rts/internal/fixed"
)

// Advanced ability effects (docs/ABILITIES.md): cone areas, chains, summons,
// teleports and transformations.

// Cast event kinds (visuals).
const (
	EventCast  = 0 // a circle at Pos with Radius (zones: Duration)
	EventChain = 1 // one jump of a chain: from Pos to To
	EventCone  = 2 // a cone from Pos, Radius long, Angle degrees wide, towards Facing
)

// buildingSnap: teleports to a building must end this close to an own one.
var buildingSnap = fixed.FromInt(12)

// area calls fn for every soldier an effect touches, by its shape.
func (w *World) area(owner PlayerID, affects string, src *Soldier, e *data.AbilityEffect, center fixed.Vec2, target *Horde, fn func(*Soldier)) {
	if e.Shape != data.ShapeCone || src == nil {
		w.affected(owner, affects, center, e.Radius, target, fn)
		return
	}
	origin := src.Pos
	facing := coneFacing(src, center)
	half := e.Angle.Mul(fixed.Pi).DivInt(360) // half the width, radians
	r2 := e.Radius.Mul(e.Radius)
	w.forEachSoldierNear(origin, e.Radius, func(s *Soldier) {
		if !s.Alive() || !touches(affects, owner, s.Horde.Owner) {
			return
		}
		d := s.Pos.Sub(origin)
		if d.LenSq() > r2 || (!d.IsZero() && fixed.NormalizeAngle(d.Angle()-facing).Abs() > half) {
			return
		}
		fn(s)
	})
}

// coneFacing: from the caster towards the aimed point (or where it faces).
func coneFacing(src *Soldier, center fixed.Vec2) fixed.Fixed {
	if d := center.Sub(src.Pos); !d.IsZero() {
		return d.Angle()
	}
	return src.Horde.Facing
}

// runChain hits the soldier nearest to the target point, then jumps to the
// nearest enemy not hit yet, losing Falloff of its power on every jump.
func (w *World) runChain(owner PlayerID, src *Soldier, a *data.Ability, e *data.AbilityEffect, center fixed.Vec2, target *Horde) {
	affects := a.AffectsOf(e)
	var hit []*Soldier
	valid := func(s *Soldier) bool {
		if !s.Alive() || !touches(affects, owner, s.Horde.Owner) {
			return false
		}
		for _, x := range hit {
			if x == s {
				return false
			}
		}
		return true
	}
	nearest := func(p fixed.Vec2, reach fixed.Fixed) *Soldier {
		var best *Soldier
		var bestSq fixed.Fixed
		w.forEachSoldierNear(p, reach, func(s *Soldier) {
			if !valid(s) {
				return
			}
			d := s.Pos.DistSq(p)
			if d <= reach.Mul(reach) && (best == nil || d < bestSq || (d == bestSq && s.ID < best.ID)) {
				best, bestSq = s, d
			}
		})
		return best
	}

	cur := nearest(center, fixed.FromInt(3))
	if target != nil && (cur == nil || cur.Horde != target) {
		for _, s := range target.Soldiers { // unit target: its soldier nearest to the point
			if valid(s) && (cur == nil || cur.Horde != target || s.Pos.DistSq(center) < cur.Pos.DistSq(center)) {
				cur = s
			}
		}
	}
	from := center
	if src != nil {
		from = src.Pos
	}
	power := fixed.One
	dmgMul := fixed.One
	if src != nil {
		dmgMul = mul(src.mods.damage)
	}
	for j := 0; j <= e.Jumps && cur != nil; j++ {
		w.event(a, CastEvent{Kind: EventChain, Owner: owner, Pos: from, To: cur.Pos})
		for k := range e.Effects {
			n := &e.Effects[k]
			if n.Mode != "" && n.Mode != casterMode(src) {
				continue
			}
			switch n.Type {
			case data.EffDamage:
				w.hitSoldier(src, owner, cur, n.Amount.Mul(power).Mul(dmgMul), n.DamageType, true)
			case data.EffStatus:
				w.applyStatus(cur, owner, n.Status, cooldownTicks(n.Duration))
			}
		}
		hit = append(hit, cur)
		power = power.Mul(fixed.One - e.Falloff)
		from = cur.Pos
		cur = nearest(from, e.JumpRange)
	}
}

// summon spawns the effect's hordes around the centre for the caster's owner.
func (w *World) summon(owner PlayerID, src *Soldier, a *data.Ability, e *data.AbilityEffect, center fixed.Vec2) {
	n := max(e.Count, 1)
	facing := fixed.Fixed(0)
	slot := -1
	if src != nil {
		facing = src.Horde.Facing
		slot = src.Horde.AbilityIndex(a.ID)
	}
	for k := 0; k < n; k++ {
		p := center
		if n > 1 {
			p = p.Add(fixed.FromAngle(fixed.Pi.MulInt(2 * k).DivInt(n)).Scale(fixed.FromInt(4)))
		}
		var ok bool
		def := w.Catalog.Hordes[e.Horde]
		if w.Catalog.Units[def.Unit].Flying {
			p, ok = w.clampToMap(p), true
		} else {
			p, ok = w.walkableTarget(p)
		}
		if !ok {
			continue
		}
		h, err := w.SpawnHorde(owner, e.Horde, p, facing)
		if err != nil {
			continue
		}
		h.summoned = true
		if src != nil {
			h.summoner, h.summonSlot = src.Horde, slot
		}
		if e.Duration > 0 {
			h.expires = w.Tick + uint32(cooldownTicks(e.Duration))
		}
	}
}

// summonAlive: something summoned by ability slot i of h is still alive.
func (w *World) summonAlive(h *Horde, i int) bool {
	for _, x := range w.Hordes {
		if x.summoner == h && x.summonSlot == i && x.Alive() {
			return true
		}
	}
	return false
}

// limitedSummon: the ability summons with a limit (one at a time).
func limitedSummon(a *data.Ability) bool {
	for _, e := range a.Effects {
		if e.Type == data.EffSummon && e.Limit {
			return true
		}
	}
	return false
}

// expireSummons dismisses temporary hordes whose time ran out (no XP for anyone).
func (w *World) expireSummons() {
	for _, h := range w.Hordes {
		if h.expires != 0 && w.Tick >= h.expires && !h.dismissed {
			h.dismissed = true
			for _, s := range h.Soldiers {
				s.HP = 0
			}
		}
	}
}

// teleportTarget is where a teleport to point ends, or false if it can't.
func (w *World) teleportTarget(owner PlayerID, e *data.AbilityEffect, point fixed.Vec2) (fixed.Vec2, bool) {
	if !e.ToBuilding {
		return point, true
	}
	var best *Building
	for _, b := range w.Buildings {
		if b.Owner == owner && !b.UnderConstruction() && b.DistTo(point) <= buildingSnap &&
			(best == nil || b.DistTo(point) < best.DistTo(point)) {
			best = b
		}
	}
	if best == nil {
		return point, false
	}
	return w.walkableTarget(best.approach(point, fixed.FromInt(4)))
}

// castBlocked: the ability can't be cast at point right now (a limited
// summon is alive, a teleport to a building without one there).
func (w *World) castBlocked(h *Horde, i int, point fixed.Vec2) bool {
	a := h.abilities[i]
	if limitedSummon(a) && w.summonAlive(h, i) {
		return true
	}
	for k := range a.Effects {
		if e := &a.Effects[k]; e.Type == data.EffTeleport {
			if _, ok := w.teleportTarget(h.Owner, e, point); !ok {
				return true
			}
		}
	}
	return false
}

// teleport moves the caster's horde (and own hordes within Radius of it) to the centre.
func (w *World) teleport(owner PlayerID, src *Soldier, e *data.AbilityEffect, center fixed.Vec2) {
	if src == nil {
		return
	}
	dest, ok := w.teleportTarget(owner, e, center)
	if !ok {
		return
	}
	lead := src.Horde
	group := []*Horde{lead}
	if e.Radius > 0 {
		for _, h := range w.Hordes {
			if h != lead && h.Owner == owner && h.Alive() && h.Pos.Dist(lead.Pos) <= e.Radius {
				group = append(group, h)
			}
		}
	}
	origin := lead.Pos
	for _, h := range group {
		g := w.navFor(h.Owner).Grid()
		np, ok := w.targetFor(h, dest.Add(h.Pos.Sub(origin)))
		if !ok {
			continue
		}
		h.clearOrders()
		h.stop()
		h.pending = nil
		h.wall = nil // teleported off the wall
		h.Pos = np
		for i, s := range h.Soldiers {
			p := h.slotPos(i)
			if !h.Unit.Flying && !g.WalkableAt(p) {
				p = np
			}
			s.Pos = p
			s.target = nil
		}
	}
}

// transform turns the caster's horde into another unit for a while.
func (w *World) transform(src *Soldier, e *data.AbilityEffect) {
	if src == nil {
		return
	}
	h := src.Horde
	h.setForm(w.Catalog.Units[e.Unit])
	h.formTicks = cooldownTicks(e.Duration)
}

// setForm swaps the horde's unit, keeping each soldier's health fraction.
func (h *Horde) setForm(u *data.Unit) {
	if u == nil || u == h.Unit {
		return
	}
	fracs := make([]fixed.Fixed, len(h.Soldiers))
	for i, s := range h.Soldiers {
		fracs[i] = s.HP.Div(s.MaxHP())
	}
	h.Unit = u
	for i, s := range h.Soldiers {
		s.HP = max(fracs[i].Mul(s.MaxHP()), fixed.FromFraction(1, 100)) // never kills by rounding
	}
	h.melee = false
}

// updateForm counts a transformation down and turns back at the end.
func (h *Horde) updateForm() {
	if h.formTicks > 0 {
		h.formTicks--
		if h.formTicks == 0 {
			h.setForm(h.base)
		}
	}
}

// Transformed reports whether the horde is in another form; Scale is the
// form's size relative to the base unit (for rendering).
func (h *Horde) Transformed() bool { return h.Unit != h.base }

func (h *Horde) FormScale() fixed.Fixed {
	if h.base == nil || h.base.Radius == 0 {
		return fixed.One
	}
	return h.Unit.Radius.Div(h.base.Radius)
}

// event records a visual cast event for the ability.
func (w *World) event(a *data.Ability, ev CastEvent) {
	if idx := w.abilityIndex(a); idx >= 0 {
		ev.Ability = idx
		w.Events = append(w.Events, ev)
	}
}

// Summoned reports whether an ability created the horde.
func (h *Horde) Summoned() bool { return h.summoned }

// Mode is the id of the horde's current mode (element), "" without one.
func (h *Horde) Mode() string {
	for i, a := range h.abilities {
		if a.Kind == data.AbilityMode && h.abilityUnlocked(i) {
			return a.Modes[h.modes[i]].ID
		}
	}
	return ""
}

func casterMode(src *Soldier) string {
	if src == nil {
		return ""
	}
	return src.Horde.Mode()
}

// forMode keeps the effects that run in mode (zones fix their pulses at cast).
func forMode(effects []data.AbilityEffect, mode string) []data.AbilityEffect {
	out := make([]data.AbilityEffect, 0, len(effects))
	for _, e := range effects {
		if e.Mode == "" || e.Mode == mode {
			out = append(out, e)
		}
	}
	return out
}

// firstDamageType: the first damage among effects that run in mode ("" = none).
func firstDamageType(effects []data.AbilityEffect, mode string) string {
	for _, e := range effects {
		if e.Mode != "" && e.Mode != mode {
			continue
		}
		if e.Type == data.EffDamage {
			return e.DamageType
		}
		if t := firstDamageType(e.Effects, mode); t != "" {
			return t
		}
	}
	return ""
}

// attackType: the damage type of an attack, unless a mode / buff replaces it.
func (s *Soldier) attackType(a *data.Attack) string {
	if s.mods.attackType != "" {
		return s.mods.attackType
	}
	return a.Type()
}
