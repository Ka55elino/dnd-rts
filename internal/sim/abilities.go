package sim

import (
	"slices"

	"dnd-rts/internal/data"
	"dnd-rts/internal/fixed"
)

// Abilities (docs/ABILITIES.md): active abilities are cast on command or by
// autocast, walk into range first if needed, then run their effects; toggles
// and passives change the own horde's modifiers (mods.go), auras those of
// units around the caster. Zones keep pulsing effects for a while.

// pendingCast is an ordered cast waiting until the horde is in range.
type pendingCast struct {
	index  int
	point  fixed.Vec2
	target *Horde // unit-targeted
}

// Zone is a lingering ability area (e.g. a storm).
type Zone struct {
	ID       uint32
	Owner    PlayerID
	Ability  *data.Ability
	Pos      fixed.Vec2
	Radius   fixed.Fixed
	ticks    int // left
	interval int
	effects  []data.AbilityEffect
}

// CastEvent tells the client an ability went off (for visuals only).
type CastEvent struct {
	Ability  int // index into Catalog.AbilityIDs
	Kind     int // EventCast / EventChain / EventCone
	Owner    PlayerID
	Pos      fixed.Vec2
	Radius   fixed.Fixed
	Duration fixed.Fixed // zones: how long it lasts (0 = instant)
	To       fixed.Vec2  // chain: end of the jump
	Angle    fixed.Fixed // cone: width, degrees
	Facing   fixed.Fixed // cone: direction, radians
	// DamageType of the first damage that went off (visual colour; follows the element).
	DamageType string
}

// abilityIndex is the ability's index in the catalog (snapshot kind).
func (w *World) abilityIndex(a *data.Ability) int { return slices.Index(w.Catalog.AbilityIDs, a.ID) }

// initAbilities resolves the unit's abilities when the horde spawns.
func (h *Horde) initAbilities(c *data.Catalog) {
	for _, id := range h.Unit.Abilities {
		h.abilities = append(h.abilities, c.Abilities[id])
	}
	n := len(h.abilities)
	h.abilityCd = make([]int, n)
	h.modes = make([]int, n)
	h.toggled = make([]bool, n)
	h.autocast = make([]bool, n)
}

// AbilityIndex returns the slot of an ability id in the horde, or -1.
func (h *Horde) AbilityIndex(id string) int {
	return slices.IndexFunc(h.abilities, func(a *data.Ability) bool { return a.ID == id })
}

// AbilityState is a horde's ability slot for the client.
type AbilityState struct {
	ID       string
	Cooldown int // ticks left
	Toggled  bool
	Autocast bool
	Pending  bool
	Locked   bool // hero level too low
	Mode     int  // mode abilities: chosen mode index
}

// AbilityStates lists the horde's abilities.
func (h *Horde) AbilityStates() []AbilityState {
	out := make([]AbilityState, len(h.abilities))
	for i, a := range h.abilities {
		out[i] = AbilityState{ID: a.ID, Cooldown: h.abilityCd[i], Toggled: h.toggled[i], Autocast: h.autocast[i],
			Pending: h.pending != nil && h.pending.index == i, Locked: !h.abilityUnlocked(i), Mode: h.modes[i]}
	}
	return out
}

// caster is the soldier abilities come from (the hero / the first soldier).
func (h *Horde) caster() *Soldier {
	if len(h.Soldiers) == 0 {
		return nil
	}
	return h.Soldiers[0]
}

// orderCast handles a cast command for one horde.
func (w *World) orderCast(h *Horde, id string, point fixed.Vec2, target *Horde) {
	i := h.AbilityIndex(id)
	if i < 0 || h.abilityCd[i] > 0 || !h.Alive() {
		return
	}
	a := h.abilities[i]
	if a.Kind != data.AbilityActive || !w.Meets(h.Owner, a.Requires) || !h.abilityUnlocked(i) {
		return
	}
	switch a.Target {
	case data.TargetSelf:
		if !w.castBlocked(h, i, h.caster().Pos) {
			w.cast(h, i, h.caster().Pos, nil)
		}
		return
	case data.TargetUnit:
		affects := a.Affects
		if affects == "" {
			affects = data.AffectsEnemies // default for unit targets
		}
		if target == nil || !target.Alive() || !touches(affects, h.Owner, target.Owner) {
			return
		}
		point = target.Pos
	}
	if w.castBlocked(h, i, point) {
		return
	}
	if h.caster().Pos.Dist(point) <= a.Range {
		w.cast(h, i, point, target)
		return
	}
	// Walk into range first.
	h.clearOrders()
	h.pending = &pendingCast{index: i, point: point, target: target}
	if p, ok := w.walkableTarget(point); ok {
		h.ordered = true
		h.setDestination(p)
	}
}

// updateAbilities counts cooldowns down, finishes pending casts and runs autocast.
func (w *World) updateAbilities(h *Horde) {
	// Ability haste (auras of focus) makes cooldowns run faster.
	steps := 1
	if c := h.caster(); c != nil && c.mods.abilityHaste != 0 {
		h.hasteAcc += mul(c.mods.abilityHaste)
		steps = h.hasteAcc.Int()
		h.hasteAcc -= fixed.FromInt(steps)
	}
	for i := range h.abilityCd {
		// A limited summon's cooldown waits until what it summoned is gone.
		if h.abilityCd[i] > 0 && !(limitedSummon(h.abilities[i]) && w.summonAlive(h, i)) {
			h.abilityCd[i] = max(h.abilityCd[i]-steps, 0)
		}
	}
	if p := h.pending; p != nil && h.Alive() {
		a := h.abilities[p.index]
		if p.target != nil {
			if !p.target.Alive() {
				h.pending = nil
				h.stop()
				return
			}
			p.point = p.target.Pos
			if !h.moving || h.target.Dist(p.point) > fixed.FromInt(4) {
				if q, ok := w.walkableTarget(p.point); ok {
					h.setDestination(q)
				}
			}
		}
		if h.caster().Pos.Dist(p.point) <= a.Range {
			h.pending = nil
			h.stop()
			if h.abilityCd[p.index] == 0 && !w.castBlocked(h, p.index, p.point) {
				w.cast(h, p.index, p.point, p.target)
			}
		}
		return
	}
	for i, a := range h.abilities {
		if h.autocast[i] && h.abilityCd[i] == 0 && h.Alive() && w.Meets(h.Owner, a.Requires) && h.abilityUnlocked(i) {
			w.tryAutocast(h, i, a)
		}
	}
}

// tryAutocast casts an ability when it has a good target in range: heals on
// the most wounded allied horde, damage on the nearest enemy horde.
func (w *World) tryAutocast(h *Horde, i int, a *data.Ability) {
	c := h.caster()
	reach := a.Range
	if a.Target == data.TargetSelf {
		reach = abilityRadius(a)
	}
	allies := a.Affects == data.AffectsAllies
	var best *Horde
	var bestScore fixed.Fixed
	for _, e := range w.Hordes {
		if !e.Alive() || (e.Owner == h.Owner) != allies || c.Pos.Dist(e.Pos) > reach {
			continue
		}
		var score fixed.Fixed
		if allies {
			score = missingHP(e) // most wounded
			if score <= 0 {
				continue
			}
		} else {
			score = -c.Pos.Dist(e.Pos) // nearest
		}
		if best == nil || score > bestScore {
			best, bestScore = e, score
		}
	}
	if best == nil || w.castBlocked(h, i, best.Pos) {
		return
	}
	switch a.Target {
	case data.TargetSelf:
		w.cast(h, i, c.Pos, nil)
	case data.TargetPoint:
		w.cast(h, i, best.Pos, nil)
	case data.TargetUnit:
		w.cast(h, i, best.Pos, best)
	}
}

func missingHP(h *Horde) fixed.Fixed {
	var m fixed.Fixed
	for _, s := range h.Soldiers {
		m += s.MaxHP() - s.HP
	}
	return m
}

// abilityRadius is the largest effect radius (for visuals and self autocast).
func abilityRadius(a *data.Ability) fixed.Fixed {
	r := fixed.Fixed(0)
	for _, e := range a.Effects {
		r = max(r, e.Radius)
	}
	return r
}

// cast runs an active ability now.
func (w *World) cast(h *Horde, i int, center fixed.Vec2, target *Horde) {
	a := h.abilities[i]
	// Commands run before this tick's hash rebuild; refresh it so indices match.
	w.combatHash.build(w)
	src := h.caster()
	mode := h.Mode()
	if a.Target == data.TargetSelf && target == nil {
		target = h // radius-0 effects of self abilities touch the own horde (shields, buffs)
	}
	for k := range a.Effects {
		if e := &a.Effects[k]; e.Mode == "" || e.Mode == mode {
			w.runEffect(h.Owner, src, a, e, center, target)
		}
	}
	h.abilityCd[i] = cooldownTicks(a.Cooldown)
	for _, s := range h.Soldiers {
		s.swing = swingTicks
	}
	// Visuals: one ring for circles; cones and chains send their own events.
	circle := false
	for k := range a.Effects {
		switch e := &a.Effects[k]; {
		case e.Shape == data.ShapeCone:
			w.event(a, CastEvent{Kind: EventCone, Owner: h.Owner, Pos: src.Pos, Radius: e.Radius, Angle: e.Angle, Facing: coneFacing(src, center)})
		case e.Type != data.EffChain && e.Type != data.EffZone:
			circle = true
		}
	}
	if circle {
		w.event(a, CastEvent{Owner: h.Owner, Pos: center, Radius: abilityRadius(a), DamageType: firstDamageType(a.Effects, mode)})
	}
}

// affected calls fn for every soldier the effect touches: the target horde
// when radius is 0, else everyone in the circle that `affects` allows.
func (w *World) affected(owner PlayerID, affects string, center fixed.Vec2, radius fixed.Fixed, target *Horde, fn func(*Soldier)) {
	if radius == 0 {
		if target != nil && touches(affects, owner, target.Owner) {
			for _, s := range target.Soldiers {
				if s.Alive() {
					fn(s)
				}
			}
		}
		return
	}
	r2 := radius.Mul(radius)
	w.forEachSoldierNear(center, radius, func(s *Soldier) {
		if s.Alive() && s.Pos.DistSq(center) <= r2 && touches(affects, owner, s.Horde.Owner) {
			fn(s)
		}
	})
}

// runEffect applies one ability effect at center.
func (w *World) runEffect(owner PlayerID, src *Soldier, a *data.Ability, e *data.AbilityEffect, center fixed.Vec2, target *Horde) {
	affects := a.AffectsOf(e)
	switch e.Type {
	case data.EffChain:
		w.runChain(owner, src, a, e, center, target)
	case data.EffSummon:
		w.summon(owner, src, a, e, center)
	case data.EffTeleport:
		w.teleport(owner, src, e, center)
	case data.EffTransform:
		w.transform(src, e)
	case data.EffDamage:
		dmg := e.Amount
		if src != nil {
			dmg = dmg.Mul(mul(src.mods.damage))
		}
		w.area(owner, affects, src, e, center, target, func(s *Soldier) {
			w.hitSoldier(src, owner, s, dmg, e.DamageType, true)
		})
		if e.Radius > 0 && e.Shape != data.ShapeCone {
			for _, b := range w.Buildings {
				if b.Owner != owner && b.Owner != Neutral && b.DistTo(center) <= e.Radius && affects != data.AffectsAllies {
					w.hitBuilding(owner, b, dmg, e.DamageType, true)
				}
			}
		}
	case data.EffHeal:
		w.area(owner, affects, src, e, center, target, func(s *Soldier) {
			if isLiving(s.Horde.Unit) {
				s.HP = min(s.HP+e.Amount, s.MaxHP())
			}
		})
	case data.EffStatus:
		ticks := cooldownTicks(e.Duration)
		w.area(owner, affects, src, e, center, target, func(s *Soldier) { w.applyStatus(s, owner, e.Status, ticks) })
	case data.EffKnockback:
		w.area(owner, affects, src, e, center, target, func(s *Soldier) { knockFrom(w, s, center, e.Radius, e.Force) })
	case data.EffDispel:
		w.area(owner, affects, src, e, center, target, func(s *Soldier) { s.fx = status{} })
	case data.EffBuff:
		ticks := cooldownTicks(e.Duration)
		w.area(owner, affects, src, e, center, target, func(s *Soldier) {
			s.buffs = append(s.buffs, buff{m: &e.Modifiers, ticks: ticks})
		})
	case data.EffZone:
		z := &Zone{
			ID: w.nextZone, Owner: owner, Ability: a, Pos: center, Radius: e.Radius,
			ticks: cooldownTicks(e.Duration), interval: cooldownTicks(e.Interval), effects: forMode(e.Effects, casterMode(src)),
		}
		w.nextZone++
		w.Zones = append(w.Zones, z)
		w.event(a, CastEvent{Owner: owner, Pos: center, Radius: e.Radius, Duration: e.Duration, DamageType: firstDamageType(z.effects, "")})
	}
}

// isLiving: heal and life effects skip machines, constructs and elementals.
func isLiving(u *data.Unit) bool {
	return !u.HasTag("mechanical") && !u.HasTag("construct") && !u.HasTag("elemental")
}

// applyStatus puts a status from an ability effect on s.
func (w *World) applyStatus(s *Soldier, owner PlayerID, st string, ticks int) {
	u := s.Horde.Unit
	switch st {
	case data.StatusStun:
		if !s.immuneTo(data.EffectStun) && !s.mods.stunImmune {
			s.fx.stunTicks = max(s.fx.stunTicks, ticks)
		}
	case data.StatusSlow:
		if !s.immuneTo(data.EffectSlow) {
			s.fx.slowTicks = max(s.fx.slowTicks, ticks)
		}
	case data.StatusBlind:
		s.fx.blindTicks = max(s.fx.blindTicks, ticks)
	case data.StatusBurn:
		if !s.immuneTo(data.EffectBurn) {
			s.fx.burnTicks = max(s.fx.burnTicks, ticks)
			s.fx.burnOwner = owner
			if s.fx.burnDamage == 0 {
				s.fx.burnDamage = fixed.FromInt(2)
			}
		}
	case data.StatusGround:
		if u.Flying {
			s.Horde.groundTicks = max(s.Horde.groundTicks, ticks)
		}
	case data.StatusAcid:
		if !s.immuneTo(data.EffectAcid) {
			s.fx.acidStacks = min(s.fx.acidStacks+1, w.Catalog.Combat.Acid.MaxStacks)
			s.fx.acidTicks = max(s.fx.acidTicks, ticks)
		}
	}
}

// knockFrom pushes s away from center, harder near the centre. Heroes resist.
func knockFrom(w *World, s *Soldier, center fixed.Vec2, radius, force fixed.Fixed) {
	if s.Horde.Unit.HasTag("hero") || s.Horde.Airborne() {
		return
	}
	away := s.Pos.Sub(center)
	dir := away.Normalize()
	if dir.IsZero() {
		dir = fixed.FromAngle(fixed.FromInt(int(s.ID) % 6))
	}
	k := fixed.One
	if radius > 0 {
		k = fixed.One - away.Len().Div(radius).DivInt(2) // full at the centre, half at the edge
	}
	s.knock = dir.Scale(force.Mul(k))
	s.knockTicks = knockdownTicks
	s.immuneUntil = w.Tick + trampleImmunity
	s.target = nil
}

// updateZones pulses every zone's effects and removes expired ones.
func (w *World) updateZones() {
	kept := w.Zones[:0]
	for _, z := range w.Zones {
		if z.ticks%max(z.interval, 1) == 0 {
			for k := range z.effects {
				e := z.effects[k]
				if e.Radius == 0 {
					e.Radius = z.Radius // pulses cover the whole zone by default
				}
				w.runEffect(z.Owner, nil, z.Ability, &e, z.Pos, nil)
			}
		}
		if z.ticks--; z.ticks > 0 {
			kept = append(kept, z)
		}
	}
	for i := len(kept); i < len(w.Zones); i++ {
		w.Zones[i] = nil
	}
	w.Zones = kept
}

// toggleAbility switches a toggle ability of the horde (needs its requirements to turn on).
func (w *World) toggleAbility(h *Horde, id string, value int) {
	if i := h.AbilityIndex(id); i >= 0 && h.abilities[i].Kind == data.AbilityMode {
		a := h.abilities[i]
		if value < 0 || value >= len(a.Modes) || value == h.modes[i] || h.abilityCd[i] > 0 ||
			!w.Meets(h.Owner, a.Requires) || !h.abilityUnlocked(i) {
			return
		}
		h.modes[i] = value
		h.abilityCd[i] = cooldownTicks(a.Cooldown)
		return
	}
	if i := h.AbilityIndex(id); i >= 0 && h.abilities[i].Kind == data.AbilityToggle {
		if !h.toggled[i] && (!w.Meets(h.Owner, h.abilities[i].Requires) || !h.abilityUnlocked(i)) {
			return
		}
		h.toggled[i] = !h.toggled[i]
	}
}

// setAutocast turns autocast of an ability on or off (if the ability allows it).
func (h *Horde) setAutocast(id string, on bool) {
	if i := h.AbilityIndex(id); i >= 0 && h.abilities[i].Autocast {
		h.autocast[i] = on
	}
}
