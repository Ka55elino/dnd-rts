package sim

import (
	"dnd-rts/internal/data"
	"dnd-rts/internal/fixed"
)

// ProjectileID identifies a volley in flight.
type ProjectileID uint32

// Projectile is one volley of a ranged horde: a single simulated object that
// stands for every arrow the horde fired. It flies to a ground point (not to a
// soldier), so moving targets can get out of the way, and on impact each of its
// arrows lands at its own deterministic spot inside the spread circle and hits
// whoever stands there. Dense formations therefore take more hits.
type Projectile struct {
	ID         ProjectileID
	Owner      PlayerID
	From, To   fixed.Vec2
	LaunchTick uint32
	ImpactTick uint32
	Arrows     int         // one per archer alive at launch
	Damage     fixed.Fixed // per arrow
	Spread     fixed.Fixed // landing radius
	Building   *Building   // aimed at this building: arrows landing on it damage it
	DamageType string
	ExtraType  string       // extra damage type from modifiers (e.g. fire arrows)
	Extra      fixed.Fixed  // fraction of each arrow's damage
	NoPierce   bool         // fire mode: pierce damage can't ignore armor
	FromAir    bool         // shot by a flying horde (visual: launch height)
	FromWall   bool         // shot from a wall or a tower (visual: launch height)
	Splash     fixed.Fixed  // area shot: hurts everyone around where it lands
	ToAir      bool         // aimed at a flying horde (visual: impact height)
	horde      *Horde       // shooter (bonuses); may be gone by impact — only its Def is read
	attack     *data.Attack // the attack fired (targets, bonuses): a horde's ranged attack or a tower's
}

var (
	// spreadPerUnit widens the landing circle with distance (less accurate at long range).
	spreadPerUnit = fixed.FromFraction(8, 100)
	// arrowHitRadius is added to the victim's radius when checking a landing spot.
	arrowHitRadius = fixed.FromFraction(2, 10)
	// minFlightTicks keeps very close volleys visible.
	minFlightTicks = 4
)

// fireVolley launches a volley from horde h at a ground point (the enemy
// horde's current position, or the center of a target building).
func (w *World) fireVolley(h *Horde, to fixed.Vec2, building *Building) {
	pr := h.Unit.Ranged.Projectile
	dist := h.Pos.Dist(to)
	flight := max(minFlightTicks, dist.Div(pr.Speed).MulInt(TickRate).Ceil().Int())

	p := &Projectile{
		ID:         w.nextProjectile,
		Owner:      h.Owner,
		From:       h.Pos,
		To:         to,
		LaunchTick: w.Tick,
		ImpactTick: w.Tick + uint32(flight),
		Arrows:     len(h.Soldiers),
		Damage:     h.Unit.Ranged.Damage.Mul(mul(h.Soldiers[0].mods.damage)),
		ExtraType:  h.Soldiers[0].mods.extraType,
		Extra:      h.Soldiers[0].mods.extraDamage,
		NoPierce:   h.Soldiers[0].mods.noPierce,
		Spread:     pr.Spread + dist.Mul(spreadPerUnit),
		Building:   building,
		DamageType: h.Soldiers[0].attackType(h.Unit.Ranged),
		FromAir:    h.Airborne(),
		FromWall:   h.wall != nil,
		ToAir:      building == nil && h.attack != nil && h.attack.Airborne(),
		horde:      h,
		attack:     h.Unit.Ranged,
		Splash:     h.Unit.Ranged.Splash,
	}
	w.nextProjectile++
	w.Projectiles = append(w.Projectiles, p)

	for _, s := range h.Soldiers {
		s.swing = swingTicks // draw/release animation
	}
}

// updateProjectiles resolves volleys that land this tick.
func (w *World) updateProjectiles() {
	kept := w.Projectiles[:0]
	for _, p := range w.Projectiles {
		if p.ImpactTick > w.Tick {
			kept = append(kept, p)
			continue
		}
		w.impact(p)
	}
	for i := len(kept); i < len(w.Projectiles); i++ {
		w.Projectiles[i] = nil
	}
	w.Projectiles = kept
}

func (w *World) impact(p *Projectile) {
	w.pierceOff = p.NoPierce
	defer func() { w.pierceOff = false }()
	for k := 0; k < p.Arrows; k++ {
		land := p.To.Add(arrowOffset(p.ID, k, p.Spread))
		if p.Splash > 0 {
			w.splash(p, land)
			continue
		}

		var victim *Soldier
		bestSq := fixed.Max
		w.forEachSoldierNear(land, fixed.FromInt(2), func(e *Soldier) {
			if e.Horde.Owner == p.Owner || !e.Alive() || (p.attack != nil && !p.attack.CanHit(e.Horde.Airborne())) {
				return
			}
			reach := e.Horde.Unit.Radius + arrowHitRadius
			d := land.DistSq(e.Pos)
			if d <= reach.Mul(reach) && d < bestSq {
				victim, bestSq = e, d
			}
		})
		if victim != nil {
			dmg := p.Damage
			if p.attack != nil {
				dmg = attackBonus(p.attack, victim.Horde.Unit, dmg)
			}
			if w.hitSoldierEx(nil, p.Owner, victim, dmg, p.DamageType, true, true) > 0 && p.ExtraType != "" {
				w.hitSoldierEx(nil, p.Owner, victim, dmg.Mul(p.Extra), p.ExtraType, true, true)
			}
		} else if b := p.Building; b != nil && b.Alive() && b.Covers(land, arrowHitRadius) {
			w.hitBuilding(p.Owner, b, p.Damage, p.DamageType, true)
		}
	}
}

// arrowOffset returns a deterministic, uniformly distributed point in a disk
// of the given radius for arrow k of a volley.
func arrowOffset(id ProjectileID, k int, radius fixed.Fixed) fixed.Vec2 {
	h := mix(uint64(id)*0x9E3779B97F4A7C15 ^ uint64(k)*0xD1B54A32D192ED03)
	angle := fixed.FromRaw(int64(h % uint64(fixed.TwoPi.Raw())))
	u := fixed.FromRaw(int64((h >> 32) & 0xFFFF)) // [0, 1)
	r := radius.Mul(u.Sqrt())                     // sqrt: uniform over the area
	return fixed.FromAngle(angle).Scale(r)
}

func mix(x uint64) uint64 {
	x ^= x >> 30
	x *= 0xBF58476D1CE4E5B9
	x ^= x >> 27
	x *= 0x94D049BB133111EB
	x ^= x >> 31
	return x
}

// updateTowers: defensive buildings shoot the nearest enemy horde in range.
func (w *World) updateTowers() {
	for _, b := range w.Buildings {
		a := b.Def.Attack
		if a == nil || !b.Alive() || b.UnderConstruction() || b.Owner == Neutral {
			continue
		}
		if b.atkCd > 0 {
			b.atkCd--
			continue
		}
		reach := a.Range
		onWall := b.host != nil && b.host.Def.Plots[b.plot].Class == data.PlotTower
		if onWall {
			reach = reach.Mul(mul(wallRangeBonus))
		}
		var target *Horde
		bestSq := reach.Mul(reach)
		minSq := a.MinRange.Mul(a.MinRange)
		for _, e := range w.Hordes {
			if e.Owner == b.Owner || !e.Alive() || !a.CanHit(e.Airborne()) {
				continue
			}
			if d := b.Pos.DistSq(e.Pos); d <= bestSq && d >= minSq {
				target, bestSq = e, d
			}
		}
		if target == nil {
			continue
		}
		m := w.towerMods(b)
		b.atkCd = max(1, fixed.FromInt(cooldownTicks(a.Cooldown)).Div(mul(m.attackSpeed)).Int())
		pr := a.Projectile
		dist := b.Pos.Dist(target.Pos)
		flight := max(minFlightTicks, dist.Div(pr.Speed).MulInt(TickRate).Ceil().Int())
		w.Projectiles = append(w.Projectiles, &Projectile{
			ID: w.nextProjectile, Owner: b.Owner, From: b.Pos, To: target.Pos,
			LaunchTick: w.Tick, ImpactTick: w.Tick + uint32(flight),
			Arrows: max(b.Def.Volley, 1), Damage: a.Damage.Mul(mul(m.damage)), Spread: pr.Spread + dist.Mul(spreadPerUnit),
			DamageType: a.Type(), FromWall: true, ToAir: target.Airborne(), attack: a, Splash: a.Splash,
			ExtraType: m.extraType, Extra: m.extraDamage, NoPierce: m.noPierce,
		})
		w.nextProjectile++
	}
}

// splash: an area shot lands — every enemy soldier in the radius takes damage
// (full at the centre, half at the edge), and so do buildings it reaches.
func (w *World) splash(p *Projectile, land fixed.Vec2) {
	r := p.Splash
	w.forEachSoldierNear(land, r+fixed.One, func(e *Soldier) {
		if e.Horde.Owner == p.Owner || !e.Alive() || (p.attack != nil && !p.attack.CanHit(e.Horde.Airborne())) {
			return
		}
		d := e.Pos.Dist(land)
		if d > r+e.Horde.Unit.Radius {
			return
		}
		f := fixed.One - d.Min(r).Div(r).DivInt(2) // 1 at the centre .. 0.5 at the edge
		dmg := p.Damage.Mul(f)
		if p.attack != nil {
			dmg = attackBonus(p.attack, e.Horde.Unit, dmg)
		}
		w.hitSoldierEx(nil, p.Owner, e, dmg, p.DamageType, true, true)
	})
	for _, b := range w.Buildings {
		if b.Alive() && b.Attackable(p.Owner) && b.DistTo(land) <= r {
			w.hitBuilding(p.Owner, b, p.Damage, p.DamageType, true)
		}
	}
}

// towerMods: a tower's switched-on toggles and its owner's upgrades for it.
func (w *World) towerMods(b *Building) mods {
	var m mods
	for i, id := range b.Def.Toggles {
		if a := w.Catalog.Abilities[id]; i < len(b.toggled) && b.toggled[i] && w.Meets(b.Owner, a.Requires) {
			m.add(&a.Modifiers)
		}
	}
	if e := w.economyOf(b.Owner); e != nil {
		for i, on := range e.upgrades {
			if g := w.Catalog.Upgrades[w.Catalog.UpgradeIDs[i]]; on && g.AppliesTo(b.Def) {
				m.add(&g.Modifiers)
			}
		}
	}
	return m
}

// toggleBuilding switches a tower's toggle ability (needs its requirements to turn on).
func (w *World) toggleBuilding(player PlayerID, b *Building, id string) {
	if b == nil || b.Owner != player || !b.Alive() {
		return
	}
	for i, t := range b.Def.Toggles {
		if t != id || i >= len(b.toggled) {
			continue
		}
		if !b.toggled[i] && !w.Meets(player, w.Catalog.Abilities[t].Requires) {
			return
		}
		b.toggled[i] = !b.toggled[i]
	}
}

// ToggleOn reports whether the building's first toggle (fire arrows) is on.
func (b *Building) ToggleOn() bool { return len(b.toggled) > 0 && b.toggled[0] }
