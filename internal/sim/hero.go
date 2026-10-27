package sim

import (
	"dnd-rts/internal/data"
	"dnd-rts/internal/fixed"
)

// Heroes (docs/FACTIONS.md §1): experience from kills nearby and outpost
// captures, 5 levels (ability i unlocks at level i+1, stats grow per level),
// one per type per player, revived in the castle keeping their level.

// heroRecord remembers a unique horde of a player (alive or waiting for revival).
type heroRecord struct {
	def   *data.Horde
	level int
	xp    int
	alive bool
}

// IsHero reports whether the horde levels up.
func (h *Horde) IsHero() bool { return h.base.Hero != nil }

// Level returns the hero level (1 for everyone else).
func (h *Horde) Level() int { return max(h.level, 1) }

// XP returns the hero's total experience.
func (h *Horde) XP() int { return h.xp }

// MaxHP is the soldier's health limit (heroes grow per level).
func (s *Soldier) MaxHP() fixed.Fixed {
	u, hd := s.Horde.Unit, s.Horde.base.Hero // a transformed hero keeps growing by its own rules
	if hd == nil || s.Horde.level <= 1 {
		return u.Health
	}
	return u.Health.Mul(fixed.One + hd.HealthPerLevel.MulInt(s.Horde.level-1))
}

// abilityUnlocked: hero ability i opens at level i+1; other units have all.
func (h *Horde) abilityUnlocked(i int) bool {
	return !h.IsHero() || i < h.Level()
}

// xpValue is what a dying soldier is worth.
func xpValue(u *data.Unit) int {
	if u.XP > 0 {
		return u.XP
	}
	return max(1, u.Health.Int()/10)
}

// gainXP adds experience and levels the hero up (fully healed on level up).
func (h *Horde) gainXP(n int) {
	hd := h.base.Hero
	if hd == nil || n <= 0 {
		return
	}
	h.xp += n
	for h.level < hd.MaxLevel() && h.xp >= hd.XP[h.level] {
		h.level++
		for _, s := range h.Soldiers {
			s.HP = s.MaxHP()
		}
	}
}

// awardKills gives the XP of soldiers that died this tick to enemy heroes nearby.
func (w *World) awardKills() {
	r := w.Catalog.Combat.XP.Radius
	r2 := r.Mul(r)
	var heroes []*Horde
	for _, h := range w.Hordes {
		if h.IsHero() && h.Alive() {
			heroes = append(heroes, h)
		}
	}
	if len(heroes) == 0 {
		return
	}
	for _, s := range w.Soldiers {
		if s.Alive() || s.Horde.dismissed {
			continue
		}
		for _, h := range heroes {
			if h.Owner != s.Horde.Owner && h.Soldiers[0].Pos.DistSq(s.Pos) <= r2 {
				h.gainXP(xpValue(s.Horde.Unit))
			}
		}
	}
}

// awardCapture gives capture XP to the capturer's heroes near the outpost.
func (w *World) awardCapture(b *Building, p PlayerID) {
	reach := b.Def.Capture.Radius + w.Catalog.Combat.XP.Radius
	for _, h := range w.Hordes {
		if h.Owner == p && h.IsHero() && h.Alive() && b.DistTo(h.Soldiers[0].Pos) <= reach {
			h.gainXP(w.Catalog.Combat.XP.Capture)
		}
	}
}

// heroRecordOf returns the player's record for a unique horde type, or nil.
func (w *World) heroRecordOf(p PlayerID, hordeType string) *heroRecord {
	e := w.economyOf(p)
	if e == nil {
		return nil
	}
	for _, r := range e.heroes {
		if r.def.ID == hordeType {
			return r
		}
	}
	return nil
}

// registerUnique records a unique horde as it spawns; a revived one gets its level back.
func (w *World) registerUnique(h *Horde) {
	if !h.Def.Unique {
		return
	}
	e := w.economyOf(h.Owner)
	if e == nil {
		return
	}
	if r := w.heroRecordOf(h.Owner, h.Def.ID); r != nil {
		h.level, h.xp = r.level, r.xp
		r.alive = true
		for _, s := range h.Soldiers {
			s.HP = s.MaxHP()
		}
		return
	}
	h.level = 1
	e.heroes = append(e.heroes, &heroRecord{def: h.Def, level: 1, alive: true})
}

// recordFallen stores the level of unique hordes that were wiped out.
func (w *World) recordFallen(h *Horde) {
	if r := w.heroRecordOf(h.Owner, h.Def.ID); r != nil && h.Def.Unique {
		r.alive, r.level, r.xp = false, h.Level(), h.xp
	}
}

// HeroInfo is a player's unique horde for the client.
type HeroInfo struct {
	Horde string
	Level int
	XP    int
	Alive bool
}

// Heroes lists the player's unique hordes (alive or fallen), in hire order.
func (w *World) Heroes(p PlayerID) []HeroInfo {
	e := w.economyOf(p)
	if e == nil {
		return nil
	}
	out := make([]HeroInfo, len(e.heroes))
	for i, r := range e.heroes {
		out[i] = HeroInfo{Horde: r.def.ID, Level: r.level, XP: r.xp, Alive: r.alive}
		if r.alive {
			for _, h := range w.Hordes {
				if h.Owner == p && h.Def == r.def {
					out[i].Level, out[i].XP = h.Level(), h.xp
				}
			}
		}
	}
	return out
}

// addScaled adds k times the modifiers (hero growth per level).
func (m *mods) addScaled(x *data.Modifiers, k int) {
	if k <= 0 {
		return
	}
	f := fixed.FromInt(k)
	m.armor += x.Armor.Mul(f)
	m.magicDefense += x.MagicDefense.Mul(f)
	m.damage += x.Damage.Mul(f)
	m.attackSpeed += x.AttackSpeed.Mul(f)
	m.moveSpeed += x.MoveSpeed.Mul(f)
	m.rangeMul += x.Range.Mul(f)
	m.damageTaken += x.DamageTaken.Mul(f)
	m.rangedTaken += x.RangedTaken.Mul(f)
}
