package sim

import (
	"dnd-rts/internal/data"
	"dnd-rts/internal/fixed"
)

// Damage pipeline (docs/FACTIONS.md §1.1). Every hit in the game goes
// through hitSoldier / hitBuilding so damage types, armor classes, defenses,
// resistances and status effects apply the same way everywhere.

// status holds a soldier's timed effects. Durations count down every tick;
// damage over time is dealt every effect interval.
type status struct {
	burnTicks  int
	burnDamage fixed.Fixed // per interval
	burnOwner  PlayerID

	poisonTicks  int
	poisonDamage fixed.Fixed // per interval
	poisonOwner  PlayerID
	poisonSpread int // more units this poisoning may still infect

	slowTicks  int
	stunTicks  int
	blindTicks int // attacks miss
	acidStacks int
	acidTicks  int
}

// Status flags for the client (snapshot).
const (
	StatusBurning  = 1 << 0
	StatusPoisoned = 1 << 1
	StatusSlowed   = 1 << 2
	StatusStunned  = 1 << 3
	StatusAcid     = 1 << 4
	StatusGrounded = 1 << 5 // flying unit pulled down
	StatusOnWall   = 1 << 6 // standing on a wall walkway (drawn at its height)
)

// StatusFlags returns the soldier's active effects as bits.
func (s *Soldier) StatusFlags() uint16 {
	var f uint16
	if s.Horde.groundTicks > 0 {
		f |= StatusGrounded
	}
	if s.Horde.wall != nil {
		f |= StatusOnWall
	}
	if s.fx.burnTicks > 0 {
		f |= StatusBurning
	}
	if s.fx.poisonTicks > 0 {
		f |= StatusPoisoned
	}
	if s.fx.slowTicks > 0 {
		f |= StatusSlowed
	}
	if s.fx.stunTicks > 0 {
		f |= StatusStunned
	}
	if s.fx.acidStacks > 0 {
		f |= StatusAcid
	}
	return f
}

// Burning reports whether the building is on fire.
func (b *Building) Burning() bool { return b.burnTicks > 0 }

// Stunned reports whether the soldier can't move or act.
func (s *Soldier) Stunned() bool { return s.fx.stunTicks > 0 }

// chance rolls a deterministic chance in [0, 1] from the tick, two ids and a salt.
func (w *World) chance(p fixed.Fixed, a, b, salt uint64) bool {
	if p <= 0 {
		return false
	}
	if p >= fixed.One {
		return true
	}
	h := mix(uint64(w.Tick)*0x9E3779B97F4A7C15 ^ a*0xC2B2AE3D27D4EB4F ^ b*0x165667B19E3779F9 ^ salt)
	return int64(h&0xFFFF) < p.Raw()
}

// Salts so different rolls on the same tick are independent.
const (
	saltDodge = iota + 1
	saltPierce
	saltStun
	saltSpread
)

// attackBonus multiplies dmg by the attack's bonuses for the target's tags.
func attackBonus(a *data.Attack, target *data.Unit, dmg fixed.Fixed) fixed.Fixed {
	for _, tag := range target.Tags {
		if m, ok := a.Bonus[tag]; ok {
			dmg = dmg.Mul(m)
		}
	}
	return dmg
}

func (w *World) ticks(seconds fixed.Fixed) int { return max(1, seconds.MulInt(TickRate).Round().Int()) }

// mitigate applies flat defense with the minimum-damage floor.
func mitigate(dmg, defense, minFrac fixed.Fixed) fixed.Fixed {
	return max(dmg-max(defense, 0), dmg.Mul(minFrac))
}

// hitSoldier deals dmg of type dtype to t. src is the attacking soldier (may
// be nil: volleys, effects, abilities) and owner its player. effects=false for
// damage over time (no new effects from ticks). Returns the damage dealt.
func (w *World) hitSoldier(src *Soldier, owner PlayerID, t *Soldier, dmg fixed.Fixed, dtype string, effects bool) fixed.Fixed {
	return w.hitSoldierEx(src, owner, t, dmg, dtype, effects, false)
}

// hitSoldierEx is hitSoldier with ranged = true for volleys (ranged-damage modifiers apply).
func (w *World) hitSoldierEx(src *Soldier, owner PlayerID, t *Soldier, dmg fixed.Fixed, dtype string, effects, ranged bool) fixed.Fixed {
	if !t.Alive() || dmg <= 0 || t.mods.invulnerable {
		return 0
	}
	r := &w.Catalog.Combat
	u := t.Horde.Unit
	var srcKey uint64
	if src != nil {
		srcKey = uint64(src.ID)
	} else {
		srcKey = uint64(owner) + 1<<40
	}
	base := dmg

	if dtype != data.DmgRadiant {
		phys := data.IsPhysical(dtype)
		if phys && u.Class() == data.ArmorLight && w.chance(r.Armor.LightDodge, srcKey, uint64(t.ID), saltDodge) {
			return 0
		}
		if phys {
			noPierce := (src != nil && src.mods.noPierce) || (src == nil && w.pierceOff)
			ignore := dtype == data.DmgPierce && !noPierce && w.chance(r.Pierce.IgnoreArmorChance, srcKey, uint64(t.ID), saltPierce)
			if !ignore {
				armor := u.Armor + t.mods.armor - r.Acid.ArmorPerStack.MulInt(t.fx.acidStacks)
				dmg = mitigate(dmg, armor, r.MinDamage)
				if u.Class() == data.ArmorHeavy {
					dmg = dmg.Mul(fixed.One - r.Armor.HeavyResist)
				}
			}
		} else {
			dmg = mitigate(dmg, u.MagicDefense+t.mods.magicDefense, r.MinDamage)
		}
		if res, ok := u.Resist[dtype]; ok {
			dmg = dmg.Mul(fixed.One - res)
		}
		if t.mods.resistType == dtype {
			dmg = dmg.Mul(max(fixed.One-t.mods.resist, 0))
		}
		dmg = dmg.Mul(mul(t.mods.damageTaken))
		if ranged {
			dmg = dmg.Mul(mul(t.mods.rangedTaken))
		}
	}
	if dmg <= 0 {
		return 0
	}
	t.HP -= dmg

	if dtype == data.DmgNecrotic && src != nil && src.Alive() {
		src.HP = min(src.HP+dmg.Mul(r.Necrotic.LifeSteal), src.MaxHP())
	}
	if effects && t.Alive() {
		w.applyEffect(t, owner, srcKey, base, dtype)
	}
	return dmg
}

// applyEffect starts the status effect of a damage type on t.
func (w *World) applyEffect(t *Soldier, owner PlayerID, srcKey uint64, base fixed.Fixed, dtype string) {
	r := &w.Catalog.Combat
	interval := r.EffectInterval
	switch dtype {
	case data.DmgFire:
		if !t.immuneTo(data.EffectBurn) {
			t.fx.burnTicks = w.ticks(r.Fire.Duration)
			t.fx.burnDamage = max(t.fx.burnDamage, base.Mul(r.Fire.DotPerSecond).Mul(interval))
			t.fx.burnOwner = owner
		}
	case data.DmgLightning:
		if !t.immuneTo(data.EffectStun) && !t.mods.stunImmune && w.chance(r.Lightning.StunChance, srcKey, uint64(t.ID), saltStun) {
			t.fx.stunTicks = max(t.fx.stunTicks, w.ticks(r.Lightning.StunDuration))
		}
	case data.DmgIce:
		if !t.immuneTo(data.EffectSlow) {
			t.fx.slowTicks = max(t.fx.slowTicks, w.ticks(r.Ice.Duration))
		}
	case data.DmgAcid:
		if !t.immuneTo(data.EffectAcid) {
			t.fx.acidStacks = min(t.fx.acidStacks+1, r.Acid.MaxStacks)
			t.fx.acidTicks = w.ticks(r.Acid.Duration)
		}
	case data.DmgPoison:
		w.poison(t, owner, base.Mul(r.Poison.DotPerSecond).Mul(interval), r.Poison.MaxSpread)
	}
}

func (w *World) poison(t *Soldier, owner PlayerID, perInterval fixed.Fixed, spread int) {
	if t.immuneTo(data.EffectPoison) {
		return
	}
	t.fx.poisonTicks = w.ticks(w.Catalog.Combat.Poison.Duration)
	t.fx.poisonDamage = max(t.fx.poisonDamage, perInterval)
	t.fx.poisonOwner = owner
	t.fx.poisonSpread = max(t.fx.poisonSpread, spread)
}

// hitBuilding deals dmg of type dtype to b. Returns the damage dealt.
func (w *World) hitBuilding(owner PlayerID, b *Building, dmg fixed.Fixed, dtype string, effects bool) fixed.Fixed {
	if !b.Alive() || dmg <= 0 {
		return 0
	}
	r := &w.Catalog.Combat
	mat := b.Def.MaterialOf()
	switch dtype {
	case data.DmgRadiant:
		return 0
	case data.DmgLightning:
		if mat == data.MaterialWood || mat == data.MaterialStone {
			return 0
		}
	case data.DmgCrush:
		dmg = dmg.Mul(r.Crush.BuildingMultiplier)
	case data.DmgFire:
		switch mat {
		case data.MaterialWood:
			dmg = dmg.Mul(r.Fire.Wood)
		case data.MaterialStone:
			dmg = dmg.Mul(r.Fire.Stone)
		}
	}
	base := dmg
	if data.IsPhysical(dtype) {
		dmg = mitigate(dmg, b.Def.Armor, r.MinDamage)
	} else {
		dmg = mitigate(dmg, b.Def.MagicDefense, r.MinDamage)
	}
	b.HP -= dmg
	if effects && dtype == data.DmgFire && b.Alive() {
		b.burnTicks = w.ticks(r.Fire.Duration)
		b.burnDamage = max(b.burnDamage, base.Mul(r.Fire.DotPerSecond).Mul(r.EffectInterval))
		b.burnOwner = owner
	}
	return dmg
}

// updateEffects counts effect durations down and deals damage over time
// every effect interval. Poison may spread to nearby units of the victim's side.
func (w *World) updateEffects() {
	r := &w.Catalog.Combat
	interval := w.ticks(r.EffectInterval)
	tickNow := int(w.Tick)%interval == 0
	for _, s := range w.Soldiers {
		fx := &s.fx
		if !s.Alive() {
			continue
		}
		if fx.slowTicks > 0 {
			fx.slowTicks--
		}
		if fx.stunTicks > 0 {
			fx.stunTicks--
		}
		if fx.blindTicks > 0 {
			fx.blindTicks--
		}
		if fx.acidTicks > 0 {
			if fx.acidTicks--; fx.acidTicks == 0 {
				fx.acidStacks = 0
			}
		}
		if fx.burnTicks > 0 {
			fx.burnTicks--
			if tickNow {
				w.hitSoldier(nil, fx.burnOwner, s, fx.burnDamage, data.DmgFire, false)
			}
			if fx.burnTicks == 0 {
				fx.burnDamage = 0
			}
		}
		if fx.poisonTicks > 0 {
			fx.poisonTicks--
			if tickNow {
				w.hitSoldier(nil, fx.poisonOwner, s, fx.poisonDamage, data.DmgPoison, false)
				if fx.poisonSpread > 0 {
					w.spreadPoison(s)
				}
			}
			if fx.poisonTicks == 0 {
				fx.poisonDamage, fx.poisonSpread = 0, 0
			}
		}
	}
	for _, b := range w.Buildings {
		if b.burnTicks > 0 && b.Alive() {
			b.burnTicks--
			if tickNow {
				w.hitBuilding(b.burnOwner, b, b.burnDamage, data.DmgFire, false)
			}
			if b.burnTicks == 0 {
				b.burnDamage = 0
			}
		}
	}
}

// spreadPoison may infect the nearest healthy unit of the victim's side.
// Units infected this way don't spread further.
func (w *World) spreadPoison(s *Soldier) {
	r := &w.Catalog.Combat
	if !w.chance(r.Poison.SpreadChance, uint64(s.ID), uint64(s.fx.poisonOwner), saltSpread) {
		return
	}
	var best *Soldier
	bestSq := r.Poison.SpreadRadius.Mul(r.Poison.SpreadRadius)
	w.forEachSoldierNear(s.Pos, r.Poison.SpreadRadius, func(e *Soldier) {
		if e == s || !e.Alive() || e.Horde.Owner != s.Horde.Owner || e.fx.poisonTicks > 0 {
			return
		}
		if d := e.Pos.DistSq(s.Pos); d <= bestSq && (best == nil || d < bestSq || e.ID < best.ID) {
			best, bestSq = e, d
		}
	})
	if best == nil {
		return
	}
	w.poison(best, s.fx.poisonOwner, s.fx.poisonDamage, 0)
	s.fx.poisonSpread--
}

// slowFactor is the soldier's speed multiplier from slows and stuns.
func (s *Soldier) slowFactor(w *World) fixed.Fixed {
	f := mul(s.mods.moveSpeed)
	switch {
	case s.fx.stunTicks > 0:
		return 0
	case s.fx.slowTicks > 0:
		f = f.Mul(fixed.One - w.Catalog.Combat.Ice.Slow)
	}
	return f
}

// slowFactor of a horde: the average of its soldiers (formation moves at the pace of the group).
func (h *Horde) slowFactor(w *World) fixed.Fixed {
	if len(h.Soldiers) == 0 {
		return fixed.One
	}
	var sum fixed.Fixed
	for _, s := range h.Soldiers {
		sum += s.slowFactor(w)
	}
	return sum.DivInt(len(h.Soldiers))
}
