package sim

import (
	"dnd-rts/internal/data"
	"dnd-rts/internal/fixed"
)

// Soldier is one member of a horde.
type Soldier struct {
	ID    SoldierID
	Horde *Horde
	Slot  int
	Pos   fixed.Vec2
	Rot   fixed.Fixed // facing, radians
	HP    fixed.Fixed

	target   *Soldier // enemy being fought
	cooldown int      // ticks until the next attack
	swing    int      // ticks left of the attack animation (for the client)

	// Cavalry
	momentum fixed.Fixed // 1 = full speed; lost when trampling, regained over time

	// Knockdown (trampled)
	knock       fixed.Vec2 // knockback velocity, units per second
	knockTicks  int        // ticks left lying down / sliding (no actions)
	immuneUntil uint32     // can't be trampled again before this tick

	fx    status // burning, poison, slow, stun, acid, blind
	mods  mods   // modifiers this tick (toggles, passives, buffs, auras)
	buffs []buff // timed modifiers
	cdAcc fixed.Fixed

	// Detour: far from its slot with a wall in the way, the soldier follows the
	// horde's flow field instead of walking straight (refreshed every few ticks).
	detour    fixed.Vec2
	hasDetour bool

	spearUntil uint32 // can't be hit by a spear wall again before this tick
}

var (
	// catchUp lets soldiers move faster than the horde to reach their slot.
	catchUp = fixed.FromFraction(3, 2)
	// soldierTurnRate is how fast an idle soldier turns to the horde's facing (rad/s).
	soldierTurnRate = fixed.FromInt(6)
	// arriveEps: closer than this to the slot counts as "in position".
	arriveEps = fixed.FromFraction(1, 20)
	// meleeLeash: how far a melee soldier may stray from its formation slot to chase.
	meleeLeash = fixed.FromInt(10)
)

const (
	// swingTicks is how long the client shows the attack animation.
	swingTicks = 6
	// knockdownTicks: how long a trampled soldier is down.
	knockdownTicks = 20
	// trampleImmunity: ticks before the same soldier can be trampled again.
	trampleImmunity = 10
)

var (
	// chargeMinSpeed: fraction of full speed a rider needs to trample.
	chargeMinSpeed = fixed.FromFraction(6, 10)
	// momentumLoss per trampled victim; momentumMin floor; momentumGain per tick.
	momentumLoss = fixed.FromFraction(15, 100)
	momentumMin  = fixed.FromFraction(3, 10)
	momentumGain = fixed.FromFraction(4, 100)
	// knockFriction: knockback velocity kept per tick.
	knockFriction = fixed.FromFraction(85, 100)
	// trampleReach: extra distance beyond touching that still counts as a hit.
	trampleReach = fixed.FromFraction(1, 10)
)

// Alive reports whether the soldier has health left.
func (s *Soldier) Alive() bool { return s.HP > 0 }

// Swinging reports whether the soldier is in its attack animation.
func (s *Soldier) Swinging() bool { return s.swing > 0 }

// Knocked reports whether the soldier is down after being trampled.
func (s *Soldier) Knocked() bool { return s.knockTicks > 0 }

func (s *Soldier) update(w *World) {
	start := s.Pos
	defer s.stayOnWalkable(w, start)
	if s.swing > 0 {
		s.swing--
	}
	if s.cooldown > 0 {
		s.cooldown = max(s.cooldown-s.cooldownStep(w), 0)
	}
	if !s.Alive() {
		return // killed earlier this tick; removed at the end of the tick
	}
	if s.Stunned() {
		return // no movement, no actions
	}

	if s.knockTicks > 0 {
		// Down: slide with the knockback, no actions.
		s.knockTicks--
		s.Pos = s.Pos.Add(s.knock.Scale(TickDt))
		s.knock = s.knock.Scale(knockFriction)
		return
	}

	s.act(w)

	if s.Horde.Unit.Charge != nil {
		s.trample(w, start)
		s.momentum = min(s.momentum+momentumGain, fixed.One)
	}
}

func (s *Soldier) act(w *World) {
	if h := s.Horde; h.constructing && h.construct != nil {
		s.target = nil
		s.hammer(h.construct)
		return
	}
	if s.Horde.Ranged() || s.Horde.Unit.Melee == nil {
		// Archers shoot as a horde (volleys) and keep formation, facing the enemy.
		s.target = nil
		s.holdFormation(w)
		return
	}
	if s.Horde.fighting {
		if s.fight(w) {
			return
		}
	} else {
		s.target = nil
	}
	s.holdFormation(w)
}

// speedFactor scales movement (cavalry loses momentum while trampling).
func (s *Soldier) speedFactor(w *World) fixed.Fixed {
	f := s.slowFactor(w)
	if s.Horde.Unit.Charge != nil {
		f = f.Mul(s.momentum)
	}
	return f
}

// trample hits enemy infantry this rider rode into while moving fast.
func (s *Soldier) trample(w *World, start fixed.Vec2) {
	u := s.Horde.Unit
	moved := s.Pos.Sub(start)
	minStep := u.Speed.Mul(TickDt).Mul(chargeMinSpeed)
	if moved.LenSq() < minStep.Mul(minStep) {
		return
	}
	dir := moved.Normalize()

	w.forEachSoldierNear(s.Pos, u.Radius.MulInt(2)+trampleReach, func(e *Soldier) {
		if e.Horde.Owner == s.Horde.Owner || !e.Alive() || e.immuneUntil > w.Tick || e.Horde.Airborne() || e.Horde.wall != nil || e.Horde.Rooted() ||
			e.Horde.Unit.HasTag("cavalry") || e.Horde.Unit.HasTag("hero") {
			return
		}
		away := e.Pos.Sub(s.Pos)
		if away.Len() > u.Radius+e.Horde.Unit.Radius+trampleReach {
			return
		}
		dt := u.Charge.DamageType
		if dt == "" {
			dt = data.DmgCrush
		}
		w.hitSoldier(s, s.Horde.Owner, e, u.Charge.TrampleDamage.Mul(mul(s.mods.charge)), dt, true)
		// Thrown forward and to the side the rider hit them from.
		push := dir.Add(away.Normalize()).Normalize()
		if push.IsZero() {
			push = dir
		}
		e.knock = push.Scale(u.Charge.Knockback.Mul(mul(s.mods.charge)))
		e.knockTicks = knockdownTicks
		e.immuneUntil = w.Tick + trampleImmunity
		e.target = nil
		loss := momentumLoss
		if s.mods.formation == data.FormationWedge {
			loss = loss.DivInt(2) // a wedge drives deeper into the line
		}
		s.momentum = max(s.momentum-loss, momentumMin)
	})
}

func (s *Soldier) holdFormation(w *World) {
	h := s.Horde
	slot := h.slotPos(s.Slot)
	if p, ok := s.detourTo(w, slot); ok {
		slot = p
	}
	to := slot.Sub(s.Pos)
	dist := to.Len()

	if dist > arriveEps {
		step := h.Unit.Speed.Mul(catchUp).Mul(TickDt).Mul(s.speedFactor(w))
		s.Pos, _ = s.Pos.MoveTowards(slot, step)
		s.Rot = turnTowards(s.Rot, to.Angle(), soldierTurnRate.Mul(TickDt).MulInt(2))
		return
	}
	s.Pos = slot
	s.Rot = turnTowards(s.Rot, h.Facing, soldierTurnRate.Mul(TickDt))
}

// fight engages the nearest enemy in melee. Returns false if the soldier should
// keep formation instead (no target in reach of the leash).
func (s *Soldier) fight(w *World) bool {
	if b := s.Horde.siege; b != nil {
		return s.fightBuilding(w, b)
	}
	u := s.Horde.Unit
	search := meleeLeash

	if t := s.target; t == nil || !t.Alive() || s.Pos.Dist(t.Pos) > search+fixed.FromInt(2) {
		s.target = w.nearestEnemySoldier(s, search)
	}
	t := s.target
	if t == nil {
		return false
	}

	to := t.Pos.Sub(s.Pos)
	dist := to.Len()
	reach := u.Radius + t.Horde.Unit.Radius + u.Melee.Range
	if !to.IsZero() {
		s.Rot = turnTowards(s.Rot, to.Angle(), soldierTurnRate.Mul(TickDt).MulInt(2))
	}

	if dist > reach {
		// Close in, but don't leave the formation too far behind.
		if s.Pos.Dist(s.Horde.slotPos(s.Slot)) > meleeLeash {
			s.target = nil
			return false
		}
		step := u.Speed.Mul(TickDt).Mul(s.speedFactor(w))
		stopAt := t.Pos.Sub(to.Scale(reach.Div(dist)))
		s.Pos, _ = s.Pos.MoveTowards(stopAt, step)
		return true
	}

	if s.cooldown == 0 {
		if s.fx.blindTicks == 0 {
			dmg := attackBonus(u.Melee, t.Horde.Unit, u.Melee.Damage).Mul(mul(s.mods.damage))
			dealt := w.hitSoldier(s, s.Horde.Owner, t, dmg, s.attackType(u.Melee), true)
			if s.mods.extraType != "" && dealt > 0 {
				w.hitSoldier(s, s.Horde.Owner, t, dmg.Mul(s.mods.extraDamage), s.mods.extraType, true)
			}
		}
		s.cooldown = cooldownTicks(u.Melee.Cooldown)
		s.swing = swingTicks
	}
	return true
}

// fightBuilding walks to the nearest wall of the building and hits it.
func (s *Soldier) fightBuilding(w *World, b *Building) bool {
	u := s.Horde.Unit
	s.target = nil
	c := b.Closest(s.Pos)
	to := c.Sub(s.Pos)
	dist := to.Len()
	reach := u.Radius + u.Melee.Range
	if !to.IsZero() {
		s.Rot = turnTowards(s.Rot, to.Angle(), soldierTurnRate.Mul(TickDt).MulInt(2))
	}
	if dist > reach {
		if s.Pos.Dist(s.Horde.slotPos(s.Slot)) > meleeLeash {
			return false
		}
		step := u.Speed.Mul(TickDt).Mul(s.speedFactor(w))
		s.Pos, _ = s.Pos.MoveTowards(c.Sub(to.Scale(reach.Div(dist))), step)
		return true
	}
	if s.cooldown == 0 {
		if s.fx.blindTicks == 0 {
			w.hitBuilding(s.Horde.Owner, b, u.Melee.Damage.Mul(mul(s.mods.damage)), s.attackType(u.Melee), true)
		}
		s.cooldown = cooldownTicks(u.Melee.Cooldown)
		s.swing = swingTicks
	}
	return true
}

func cooldownTicks(seconds fixed.Fixed) int {
	return max(1, seconds.MulInt(TickRate).Round().Int())
}

// stayOnWalkable undoes a move that ended on blocked ground, sliding along
// the obstacle on one axis if possible.
func (s *Soldier) stayOnWalkable(w *World, prev fixed.Vec2) {
	if s.Horde.Unit.Flying || s.Horde.wall != nil {
		s.Pos = w.clampToMap(s.Pos)
		return
	}
	g := w.navFor(s.Horde.Owner).Grid()
	if g.WalkableAt(s.Pos) || !g.WalkableAt(prev) {
		return
	}
	if p := (fixed.Vec2{X: s.Pos.X, Y: prev.Y}); g.WalkableAt(p) {
		s.Pos = p
		return
	}
	if p := (fixed.Vec2{X: prev.X, Y: s.Pos.Y}); g.WalkableAt(p) {
		s.Pos = p
		return
	}
	s.Pos = prev
}

var (
	detourDist  = fixed.FromInt(6) // only soldiers this far from their slot look for a way around
	detourEvery = uint32(10)       // ticks between checks (staggered by soldier id)
)

// detourTo: a waypoint around an obstacle between the soldier and its slot.
func (s *Soldier) detourTo(w *World, slot fixed.Vec2) (fixed.Vec2, bool) {
	h := s.Horde
	if h.Unit.Flying || h.wall != nil || s.Pos.Dist(slot) <= detourDist {
		s.hasDetour = false
		return fixed.Vec2{}, false
	}
	if (w.Tick+uint32(s.ID))%detourEvery == 0 || (s.hasDetour && s.Pos.Dist(s.detour) < NavCellSize) {
		nv := w.navFor(h.Owner)
		g := nv.Grid()
		s.hasDetour = false
		if !g.LineWalkable(s.Pos, slot) {
			field := nv.Field(g.IndexAt(h.target))
			i := g.IndexAt(s.Pos)
			if field.Reachable(i) {
				// Farthest flow-field cell still in a straight line (a few steps ahead).
				best := g.Center(i)
				for k := 0; k < 6; k++ {
					n := int(field.Next[i])
					if n < 0 || !g.LineWalkable(s.Pos, g.Center(n)) {
						break
					}
					i, best = n, g.Center(n)
				}
				s.detour, s.hasDetour = best, true
			}
		}
	}
	return s.detour, s.hasDetour
}
