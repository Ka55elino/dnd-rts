package sim

import (
	"slices"

	"dnd-rts/internal/data"
	"dnd-rts/internal/fixed"
)

// Stat modifiers (docs/ABILITIES.md): toggles and passives of the own horde,
// timed buffs on soldiers and auras of nearby units add up into one mods
// value per soldier, recomputed at the start of every tick.

// mods is the sum of all active modifiers on a soldier.
type mods struct {
	armor, magicDefense      fixed.Fixed
	damage, attackSpeed      fixed.Fixed
	moveSpeed, rangeMul      fixed.Fixed
	damageTaken, rangedTaken fixed.Fixed
	invulnerable, stunImmune bool
	noPierce                 bool
	extraType                string
	extraDamage              fixed.Fixed
	attackType               string
	resistType               string
	resist                   fixed.Fixed
	abilityHaste             fixed.Fixed
	immune                   uint8 // bit i: data.StatusEffects[i]
	rooted                   bool
	speedDamage              fixed.Fixed
	formation                string
	charge                   fixed.Fixed
}

func (m *mods) add(x *data.Modifiers) {
	m.armor += x.Armor
	m.magicDefense += x.MagicDefense
	m.damage += x.Damage
	m.attackSpeed += x.AttackSpeed
	m.moveSpeed += x.MoveSpeed
	m.rangeMul += x.Range
	m.damageTaken += x.DamageTaken
	m.rangedTaken += x.RangedTaken
	m.invulnerable = m.invulnerable || x.Invulnerable
	m.stunImmune = m.stunImmune || x.StunImmune
	m.noPierce = m.noPierce || x.NoPierce
	m.abilityHaste += x.AbilityHaste
	m.rooted = m.rooted || x.Rooted
	m.speedDamage += x.SpeedDamage
	m.charge += x.Charge
	if x.Formation != "" {
		m.formation = x.Formation
	}
	if x.Immune != "" {
		m.immune |= 1 << slices.Index(data.StatusEffects, x.Immune)
	}
	if x.AttackType != "" {
		m.attackType = x.AttackType // the last one wins (own mode is added first)
	}
	if x.ResistType != "" {
		switch {
		case m.resistType == x.ResistType:
			m.resist += x.Resist
		case m.resistType == "" || x.Resist > m.resist:
			m.resistType, m.resist = x.ResistType, x.Resist // one ward at a time: the strongest
		}
	}
	if x.ExtraType != "" && x.ExtraDamage > 0 {
		// One extra damage type at a time: the strongest wins (ties: first).
		if m.extraType == "" || x.ExtraDamage > m.extraDamage {
			m.extraType, m.extraDamage = x.ExtraType, x.ExtraDamage
		}
	}
}

// mul returns 1 + f, never below zero.
func mul(f fixed.Fixed) fixed.Fixed { return max(fixed.One+f, 0) }

// buff is a timed modifier on one soldier.
type buff struct {
	m     *data.Modifiers
	ticks int
}

// updateMods recomputes every soldier's modifiers: own horde (toggles,
// passives), timed buffs (counted down here) and auras around casters.
func (w *World) updateMods() {
	for _, b := range w.Buildings {
		b.workSpeed = 0 // auras add it again below
	}
	for _, h := range w.Hordes {
		var own mods
		mode := h.Mode()
		for i, a := range h.abilities {
			if !w.Meets(h.Owner, a.Requires) || !h.abilityUnlocked(i) {
				continue
			}
			switch {
			case a.Kind == data.AbilityMode:
				own.add(&a.Modes[h.modes[i]].Modifiers)
			case a.Kind == data.AbilityPassive || (a.Kind == data.AbilityToggle && h.toggled[i]):
				own.add(a.ModsFor(mode))
			}
		}
		if hd := h.base.Hero; hd != nil {
			own.addScaled(&hd.PerLevel, h.Level()-1)
		}
		if e := w.economyOf(h.Owner); e != nil {
			for i, on := range e.upgrades {
				if g := w.Catalog.Upgrades[w.Catalog.UpgradeIDs[i]]; on && g.Applies(h.Unit) {
					own.add(&g.Modifiers)
				}
			}
		}
		for _, s := range h.Soldiers {
			s.mods = own
			kept := s.buffs[:0]
			for _, b := range s.buffs {
				if b.ticks--; b.ticks > 0 {
					s.mods.add(b.m)
					kept = append(kept, b)
				}
			}
			s.buffs = kept
		}
	}
	for _, h := range w.Hordes {
		if len(h.Soldiers) == 0 {
			continue
		}
		center := h.Soldiers[0].Pos
		mode := h.Mode()
		for i, a := range h.abilities {
			if a.Kind != data.AbilityAura || !h.abilityUnlocked(i) {
				continue
			}
			if ws := a.ModsFor(mode).WorkSpeed; ws != 0 {
				for _, b := range w.Buildings {
					if b.Owner == h.Owner && b.DistTo(center) <= a.Radius {
						b.workSpeed += ws
					}
				}
			}
			affects := a.Affects
			if affects == "" {
				affects = data.AffectsAllies
			}
			r2 := a.Radius.Mul(a.Radius)
			m := a.ModsFor(mode)
			w.forEachSoldierNear(center, a.Radius, func(s *Soldier) {
				if s.Alive() && s.Pos.DistSq(center) <= r2 && touches(affects, h.Owner, s.Horde.Owner) && hasAnyTag(s.Horde.Unit, a.Tags) {
					s.mods.add(m)
				}
			})
		}
	}
}

// touches reports whether an effect for `affects` from player `from` touches a unit of `owner`.
func touches(affects string, from, owner PlayerID) bool {
	switch affects {
	case data.AffectsAllies:
		return owner == from
	case data.AffectsAll:
		return true
	}
	return owner != from
}

// cooldownStep advances the soldier's attack-speed accumulator and returns how
// many cooldown ticks pass this tick (0 while stunned, fewer while slowed,
// more with attack speed buffs).
func (s *Soldier) cooldownStep(w *World) int {
	return stepRate(w, s, &s.cdAcc)
}

// volleyStep: the same for a horde's volley cooldown (paced by its first soldier).
func (h *Horde) volleyStep(w *World) int {
	return stepRate(w, h.Soldiers[0], &h.volleyAcc)
}

func stepRate(w *World, s *Soldier, acc *fixed.Fixed) int {
	if s.fx.stunTicks > 0 {
		return 0
	}
	rate := mul(s.mods.attackSpeed)
	if s.fx.slowTicks > 0 {
		rate = rate.Mul(fixed.One - w.Catalog.Combat.Ice.Slow)
	}
	*acc += rate
	n := acc.Int()
	*acc -= fixed.FromInt(n)
	return n
}

// hasAnyTag: no tags = everyone.
func hasAnyTag(u *data.Unit, tags []string) bool {
	if len(tags) == 0 {
		return true
	}
	for _, t := range tags {
		if u.HasTag(t) {
			return true
		}
	}
	return false
}

// immuneTo: the unit or one of its modifiers ignores the status effect.
func (s *Soldier) immuneTo(effect string) bool {
	if s.Horde.Unit.IsImmune(effect) {
		return true
	}
	i := slices.Index(data.StatusEffects, effect)
	return i >= 0 && s.mods.immune&(1<<i) != 0
}
