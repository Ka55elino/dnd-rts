package sim

import (
	"encoding/binary"
	"hash/fnv"
	"slices"

	"dnd-rts/internal/fixed"
)

// HashInterval: the session records a state hash every this many ticks.
const HashInterval = 20

// StateHash fingerprints the complete game state. Two machines running the
// same commands must produce the same hash at every tick; a mismatch is a
// desync. Everything that influences future ticks must be included.
func (w *World) StateHash() uint64 {
	h := fnv.New64a()
	var buf [8]byte
	u := func(v uint64) {
		binary.LittleEndian.PutUint64(buf[:], v)
		h.Write(buf[:])
	}
	f := func(v fixed.Fixed) { u(uint64(v.Raw())) }
	v2 := func(v fixed.Vec2) { f(v.X); f(v.Y) }
	b := func(v bool) {
		if v {
			u(1)
		} else {
			u(0)
		}
	}
	horde := func(x *Horde) {
		if x == nil {
			u(0)
		} else {
			u(uint64(x.ID))
		}
	}
	soldier := func(x *Soldier) {
		if x == nil {
			u(0)
		} else {
			u(uint64(x.ID))
		}
	}

	u(uint64(w.Tick))
	u(uint64(w.nextHorde))
	u(uint64(w.nextSoldier))
	u(uint64(w.nextProjectile))
	u(uint64(w.nextBuilding))

	u(uint64(len(w.Hordes)))
	for _, x := range w.Hordes {
		u(uint64(x.ID))
		u(uint64(x.Owner))
		v2(x.Pos)
		f(x.Facing)
		b(x.moving)
		b(x.ordered)
		v2(x.target)
		v2(x.waypt)
		u(uint64(x.repathAt))
		horde(x.attack)
		if x.construct != nil {
			u(uint64(x.construct.ID))
		} else {
			u(0)
		}
		b(x.constructing)
		b(x.hold)
		b(x.amove)
		v2(x.amTarget)
		b(x.patrol)
		v2(x.patrolFrom)
		if x.siege != nil {
			u(uint64(x.siege.ID))
		} else {
			u(0)
		}
		b(x.fighting)
		b(x.charging)
		u(uint64(x.volleyCooldown))
		f(x.volleyAcc)
		u(uint64(x.level))
		u(uint64(x.groundTicks))
		b(x.hasFireAt)
		v2(x.fireAt)
		u(uint64(slices.Index(w.Walls, x.wall) + 1))
		f(x.wallS)
		f(x.wallTo)
		if e := x.wallEnter; e != nil {
			u(uint64(slices.Index(w.Walls, e.ring) + 1))
			u(uint64(e.stair))
			f(e.goal)
		}
		if e := x.wallExit; e != nil {
			u(uint64(e.stair))
			v2(e.target)
		}
		u(uint64(slices.Index(w.Catalog.UnitIDs, x.Unit.ID)))
		u(uint64(x.formTicks))
		f(x.hasteAcc)
		b(x.summoned)
		b(x.dismissed)
		u(uint64(x.expires))
		u(uint64(x.summonSlot))
		if x.summoner != nil {
			u(uint64(x.summoner.ID))
		}
		u(uint64(x.xp))
		for i := range x.abilities {
			u(uint64(x.abilityCd[i]))
			u(uint64(x.modes[i]))
			b(x.toggled[i])
			b(x.autocast[i])
		}
		if p := x.pending; p != nil {
			u(uint64(p.index) + 1)
			v2(p.point)
			horde(p.target)
		} else {
			u(0)
		}
		u(uint64(len(x.Soldiers)))
	}

	u(uint64(len(w.Soldiers)))
	for _, s := range w.Soldiers {
		u(uint64(s.ID))
		u(uint64(s.Slot))
		v2(s.Pos)
		f(s.Rot)
		f(s.HP)
		soldier(s.target)
		u(uint64(s.cooldown))
		u(uint64(s.swing))
		f(s.momentum)
		v2(s.knock)
		u(uint64(s.knockTicks))
		u(uint64(s.immuneUntil))
		u(uint64(s.fx.burnTicks))
		f(s.fx.burnDamage)
		u(uint64(s.fx.poisonTicks))
		f(s.fx.poisonDamage)
		u(uint64(s.fx.poisonSpread))
		u(uint64(s.fx.slowTicks))
		u(uint64(s.fx.stunTicks))
		u(uint64(s.fx.acidStacks))
		u(uint64(s.fx.acidTicks))
		u(uint64(s.fx.blindTicks))
		f(s.cdAcc)
		b(s.hasDetour)
		u(uint64(s.spearUntil))
		v2(s.detour)
		u(uint64(len(s.buffs)))
		for _, bf := range s.buffs {
			u(uint64(bf.ticks))
		}
	}

	u(uint64(len(w.Projectiles)))
	for _, p := range w.Projectiles {
		u(uint64(p.ID))
		u(uint64(p.Owner))
		v2(p.From)
		v2(p.To)
		u(uint64(p.LaunchTick))
		u(uint64(p.ImpactTick))
		u(uint64(p.Arrows))
		f(p.Damage)
		f(p.Spread)
		if p.Building != nil {
			u(uint64(p.Building.ID))
		} else {
			u(0)
		}
	}
	for _, e := range w.econ {
		for _, n := range e.stock {
			u(uint64(n))
		}
		for _, on := range e.upgrades {
			b(on)
		}
		for _, r := range e.heroes {
			u(uint64(r.level))
			u(uint64(r.xp))
			b(r.alive)
		}
	}
	u(uint64(w.Winner))
	b(w.Over)
	u(uint64(len(w.Defeated)))
	for _, p := range w.Defeated {
		u(uint64(p))
	}

	u(uint64(w.nextZone))
	u(uint64(len(w.Zones)))
	for _, z := range w.Zones {
		u(uint64(z.ID))
		u(uint64(z.Owner))
		v2(z.Pos)
		f(z.Radius)
		u(uint64(z.ticks))
	}

	u(uint64(len(w.Buildings)))
	for _, x := range w.Buildings {
		u(uint64(x.ID))
		u(uint64(x.Owner))
		u(uint64(x.Kind))
		v2(x.Pos)
		f(x.Facing)
		f(x.HP)
		for _, a := range x.acc {
			f(a)
		}
		u(uint64(len(x.queue)))
		for _, q := range x.queue {
			if q.def != nil {
				u(uint64(slices.Index(w.Catalog.HordeIDs, q.def.ID)))
			} else {
				u(1<<32 + uint64(slices.Index(w.Catalog.UpgradeIDs, q.upgrade.ID)))
			}
			u(uint64(q.done))
		}
		b(x.hasRally)
		v2(x.rally)
		u(uint64(x.burnTicks))
		f(x.burnDamage)
		u(uint64(x.captureBy))
		u(uint64(x.captureTicks))
		u(uint64(x.buildTicks))
		f(x.work)
		f(x.workAcc)
		u(uint64(x.atkCd))
		b(x.gateOpen)
		b(x.gateWant)
		for _, on := range x.toggled {
			b(on)
		}
		b(x.selfBuild)
		if x.host != nil {
			u(uint64(x.host.ID))
			u(uint64(x.plot))
		} else {
			u(0)
		}
		for _, p := range x.plotUsers {
			if p != nil {
				u(uint64(p.ID))
			} else {
				u(0)
			}
		}
	}
	return h.Sum64()
}
