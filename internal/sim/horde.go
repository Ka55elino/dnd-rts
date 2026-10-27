package sim

import (
	"dnd-rts/internal/data"
	"dnd-rts/internal/fixed"
)

// Horde is a group of soldiers that moves and fights as one.
// Pos/Facing describe the formation anchor; soldiers follow their slots.
type Horde struct {
	ID       HordeID
	Owner    PlayerID
	Kind     int // index into Catalog.HordeIDs (sent to the client)
	Def      *data.Horde
	Unit     *data.Unit
	Pos      fixed.Vec2
	Facing   fixed.Fixed // radians, direction the formation faces
	Soldiers []*Soldier  // alive soldiers, sorted by ID; slot = index

	moving   bool
	ordered  bool // moving because of an explicit move order (ignores enemies)
	target   fixed.Vec2
	waypt    fixed.Vec2 // current steering point (target or a flow-field shortcut)
	repathAt uint32     // tick at which the waypoint is recomputed

	attack *Horde    // enemy horde being attacked (explicit or auto-aggro)
	melee  bool      // units with melee + ranged: fighting in melee right now
	siege  *Building // enemy building being attacked (explicit order only)

	construct    *Building // workers: foundation being built
	constructing bool      // workers: in reach and working

	// Stances and orders.
	hold       bool       // hold position: fight what comes into range, never chase
	amove      bool       // attack-move: walk to amTarget, fighting whatever is met
	amTarget   fixed.Vec2 // attack-move / patrol destination
	patrol     bool       // patrol between patrolFrom and amTarget (attack-move both ways)
	patrolFrom fixed.Vec2
	fighting   bool // close enough to attack: soldiers break formation to fight
	charging   bool // cavalry riding through the enemy towards a point beyond it

	volleyCooldown int         // ranged hordes: ticks until the next volley
	volleyAcc      fixed.Fixed // attack-speed accumulator for volleys

	// Abilities of the unit (same order as Unit.Abilities).
	abilities []*data.Ability
	abilityCd []int // ticks until ready
	modes     []int // mode abilities: index of the chosen mode
	hasteAcc  fixed.Fixed
	toggled   []bool // toggle abilities: on
	autocast  []bool // active abilities: cast by themselves when useful
	pending   *pendingCast

	groundTicks int // flying horde pulled down (StatusGround)

	// Attack ground: ranged hordes shoot volleys at this point.
	fireAt    fixed.Vec2
	hasFireAt bool

	// Walls: on the walkway of `wall` at node position wallS, walking to wallTo.
	wall      *WallRing
	wallS     fixed.Fixed
	wallTo    fixed.Fixed
	wallEnter *wallEnter // on the ground, heading for a stair
	wallExit  *wallExit  // on the wall, heading for a stair down

	// Transformation: Unit is the current form, base the original unit.
	base      *data.Unit
	formTicks int

	// Summons.
	summoned   bool   // created by an ability: no food, may expire
	summoner   *Horde // horde whose ability summoned it
	summonSlot int    // that ability's slot
	expires    uint32 // tick it disappears (0 = never)
	dismissed  bool   // ran out of time: dies without giving XP

	// Heroes.
	level int // 1..5 (0 = not a hero)
	xp    int
}

// Ranged reports whether the horde is currently fighting with its ranged
// attack (volleys, soldiers keep formation) rather than in melee.
func (h *Horde) Ranged() bool { return h.Unit.Ranged != nil && !h.melee }

// InMelee reports whether the horde uses its melee attack right now.
func (h *Horde) InMelee() bool { return !h.Ranged() && h.Unit.Melee != nil }

// currentAttack is the attack of the current mode (nil: the unit has none).
func (h *Horde) currentAttack() *data.Attack {
	if h.Ranged() {
		return h.Unit.Ranged
	}
	if h.Unit.Melee != nil {
		return h.Unit.Melee
	}
	return h.Unit.Ranged
}

// canTarget reports whether any attack of the horde can hit the enemy horde.
func (h *Horde) canTarget(e *Horde) bool {
	return meleeReaches(h, e) || h.Unit.Ranged.CanHit(e.Airborne())
}

// longestReach: engage distance of the longest-range attack (for aggro).
func (h *Horde) longestReach() fixed.Fixed {
	r := fixed.Fixed(0)
	for _, a := range []*data.Attack{h.Unit.Melee, h.Unit.Ranged} {
		if a != nil {
			r = max(r, h.attackRange(a))
		}
	}
	return engageBase + r
}

// selectMode picks melee or ranged against an enemy at dist. Units with both
// shoot from afar and draw weapons when the enemy closes in. Returns false if
// no attack can hit the enemy.
func (h *Horde) selectMode(e *Horde, dist fixed.Fixed) bool {
	mc, rc := meleeReaches(h, e), h.Unit.Ranged.CanHit(e.Airborne())
	if rc && h.Unit.Ranged.MinRange > 0 && dist < h.Unit.Ranged.MinRange+engageBase {
		rc = false // too close for a trebuchet
	}
	switch {
	case mc && (!rc || dist <= engageBase+h.Unit.Melee.Range+fightSlack):
		h.melee = true
	case rc:
		h.melee = false
	default:
		return false
	}
	return true
}

// selectSiegeMode: buildings are ground targets; melee if possible (units walk
// up to the wall), else ranged. Returns false if the horde can't hit buildings.
func (h *Horde) selectSiegeMode() bool {
	switch {
	case h.Unit.Melee.CanHit(false):
		h.melee = true
	case h.Unit.Ranged.CanHit(false):
		h.melee = false
	default:
		return false
	}
	return true
}

const (
	// lookahead is how many flow-field steps a horde scans for a visible shortcut.
	lookahead = 12
	// repathEvery: waypoints are refreshed this often (ticks), staggered per horde.
	repathEvery = 5
)

var (
	// directCheckRange: beyond this distance the straight-to-target check is skipped
	// (too expensive); the flow field + lookahead handles long open paths.
	directCheckRange = fixed.FromInt(64)
	// turnRate is how fast a horde rotates its formation (rad/s).
	turnRate = fixed.FromInt(3)
	// engageBase: melee hordes stop chasing when their anchors are this close (+ attack range).
	engageBase = fixed.FromInt(6)
	// aggroExtra: idle hordes auto-attack enemies within engage distance + this.
	aggroExtra = fixed.FromInt(12)
	// fightSlack: hordes keep fighting while anchors stay within engage distance + this.
	fightSlack = fixed.FromInt(6)
	// chargeOvershoot: cavalry charges ride this far past the enemy anchor.
	chargeOvershoot = fixed.FromInt(20)
)

// Alive reports whether the horde still has soldiers.
func (h *Horde) Alive() bool { return len(h.Soldiers) > 0 }

// IsMoving reports whether the horde is moving (order or chase).
func (h *Horde) IsMoving() bool { return h.moving }

// IsFighting reports whether the horde's soldiers are engaged.
func (h *Horde) IsFighting() bool { return h.fighting }

// IsCharging reports whether a cavalry horde is mid-charge.
func (h *Horde) IsCharging() bool { return h.charging }

// Target returns the current move target.
func (h *Horde) Target() fixed.Vec2 { return h.target }

// AttackTarget returns the horde being attacked, or nil.
func (h *Horde) AttackTarget() *Horde { return h.attack }

// SiegeTarget returns the building being attacked, or nil.
func (h *Horde) SiegeTarget() *Building { return h.siege }

// moveTo is an explicit move order: disengage and go.
func (h *Horde) moveTo(target fixed.Vec2) {
	h.clearOrders()
	h.fighting = false
	h.charging = false
	h.ordered = true
	h.setDestination(target)
}

// attackHorde is an explicit attack order.
func (h *Horde) attackHorde(enemy *Horde) {
	h.clearOrders()
	h.attack = enemy
	h.ordered = false
	h.charging = false
	h.setDestination(enemy.Pos)
}

// charge is a cavalry attack order: ride through the enemy to point, then melee.
func (h *Horde) charge(enemy *Horde, point fixed.Vec2) {
	h.attackHorde(enemy)
	h.charging = true
	h.fighting = false
	h.setDestination(point)
}

// attackBuilding is an explicit order to attack an enemy building. The horde
// ignores enemy hordes until the building falls or it gets another order.
func (h *Horde) attackBuilding(w *World, b *Building) {
	if !h.selectSiegeMode() {
		return
	}
	h.clearOrders()
	h.siege = b
	h.ordered = false
	h.charging = false
	h.fighting = false
	if p, ok := w.walkableTarget(b.approach(h.Pos, h.engageDist()-fixed.One)); ok {
		h.setDestination(p)
	}
}

// clearOrders drops every target, stance and multi-step order.
func (h *Horde) clearOrders() {
	h.attack = nil
	h.siege = nil
	h.construct = nil
	h.constructing = false
	h.fighting = false
	h.charging = false
	h.hold = false
	h.amove = false
	h.patrol = false
	h.pending = nil
	h.wallEnter, h.wallExit = nil, nil
	h.hasFireAt = false
}

// stopOrder: stand still and forget all orders (auto-aggro still applies).
func (h *Horde) stopOrder() {
	h.clearOrders()
	h.moving = false
	h.ordered = false
}

// holdOrder: stand still; fight enemies in range but never chase.
func (h *Horde) holdOrder() {
	h.stopOrder()
	h.hold = true
}

// attackMove walks to target and fights every enemy met on the way.
func (h *Horde) attackMove(target fixed.Vec2) {
	h.clearOrders()
	h.ordered = false
	h.amove = true
	h.amTarget = target
	h.setDestination(target)
}

// patrolTo attack-moves back and forth between the current position and target.
func (h *Horde) patrolTo(target fixed.Vec2) {
	h.attackMove(target)
	h.patrol = true
	h.patrolFrom = h.Pos
}

// IsHolding reports whether the horde holds position.
func (h *Horde) IsHolding() bool { return h.hold }

func (h *Horde) setDestination(target fixed.Vec2) {
	h.target = target
	h.moving = !target.Sub(h.Pos).IsZero()
	h.repathAt = 0 // recompute immediately
}

// engageDist is how close the anchors must be for this horde to fight.
func (h *Horde) engageDist() fixed.Fixed {
	if a := h.currentAttack(); a != nil {
		return engageBase + h.attackRange(a)
	}
	return engageBase
}

// attackRange: ranged attacks get the range modifier of the horde's first soldier.
func (h *Horde) attackRange(a *data.Attack) fixed.Fixed {
	if a == h.Unit.Ranged && len(h.Soldiers) > 0 {
		bonus := h.Soldiers[0].mods.rangeMul
		if h.wall != nil {
			bonus += wallRangeBonus
		}
		return a.Range.Mul(mul(bonus))
	}
	return a.Range
}

func (h *Horde) update(w *World) {
	if h.groundTicks > 0 {
		h.groundTicks--
	}
	h.updateForm()
	w.updateAbilities(h)
	h.updateCombat(w)
	h.updateVolley(w)
	w.repairMachines(h)
	if w.updateWall(h) {
		return // on a wall: the walkway moves it
	}
	if h.Rooted() {
		h.moving = false // a spear wall stands its ground
		return
	}
	if !h.moving {
		return
	}
	if w.Tick >= h.repathAt || h.Pos == h.waypt {
		wp, ok := h.waypoint(w)
		if !ok {
			h.stop() // target unreachable from here
			return
		}
		h.waypt = wp
		h.repathAt = w.Tick + repathEvery
	}
	dest := h.waypt
	to := dest.Sub(h.Pos)
	if !to.IsZero() {
		h.Facing = turnTowards(h.Facing, to.Angle(), turnRate.Mul(TickDt))
	}

	// Slow down while the formation is still turning, so it doesn't drag sideways.
	speed := h.Unit.Speed.Mul(TickDt).Mul(h.momentum()).Mul(h.slowFactor(w))
	diff := fixed.NormalizeAngle(to.Angle() - h.Facing).Abs()
	if diff > fixed.Pi.DivInt(4) {
		speed = speed.DivInt(4)
	}

	prev := h.Pos
	var reached bool
	h.Pos, reached = h.Pos.MoveTowards(dest, speed)
	if h.Unit.Flying {
		h.Pos = w.clampToMap(h.Pos) // flies over anything, but stays on the map
	} else if g := w.navFor(h.Owner).Grid(); !g.WalkableAt(h.Pos) && g.WalkableAt(prev) {
		h.Pos = prev // safety net: never step onto blocked ground (but let a stuck horde walk out)
		h.repathAt = 0
		return
	}
	if reached && dest == h.target {
		switch {
		case h.patrol && h.attack == nil:
			// Turn around: the other end becomes the destination.
			h.amTarget, h.patrolFrom = h.patrolFrom, h.amTarget
			h.setDestination(h.amTarget)
		case h.amove && h.attack == nil && dest == h.amTarget:
			h.amove = false
			h.stop()
		default:
			h.stop()
		}
	}
}

func (h *Horde) stop() {
	h.moving = false
	h.ordered = false
	h.charging = false
}

// momentum is the average soldier momentum (cavalry slows down in a crowd).
func (h *Horde) momentum() fixed.Fixed {
	if h.Unit.Charge == nil || len(h.Soldiers) == 0 {
		return fixed.One
	}
	var sum fixed.Fixed
	for _, s := range h.Soldiers {
		sum += s.momentum
	}
	return sum.DivInt(len(h.Soldiers))
}

// updateCombat picks / chases the enemy horde and decides whether to fight.
func (h *Horde) updateCombat(w *World) {
	if h.attack != nil && !h.attack.Alive() {
		h.attack = nil
	}
	if h.construct != nil {
		h.updateConstruct(w)
		if h.construct != nil {
			h.fighting = false
			return
		}
	}
	if h.siege != nil {
		if h.siege.Alive() {
			h.updateSiege(w)
			return
		}
		h.siege = nil
		h.fighting = false
		h.moving = false
	}
	if h.charging {
		if h.moving {
			h.fighting = false // ride on in formation; riders trample what they hit
			return
		}
		h.charging = false // charge finished: fall through to normal melee
	}
	if h.hasFireAt {
		h.updateFireAt(w)
		return
	}
	// Auto-aggro: an idle (or attack-moving) horde attacks the nearest enemy
	// in range; holding hordes only what is already in reach; workers never.
	if h.attack == nil && !h.ordered && !h.IsWorker() {
		reach := h.longestReach() + aggroExtra
		if h.hold || h.wall != nil || h.Rooted() {
			reach = h.longestReach()
		}
		h.attack = w.nearestEnemyHorde(h, reach)
	}
	if h.attack != nil && !h.selectMode(h.attack, h.Pos.Dist(h.attack.Pos)) {
		h.attack = nil // nothing can hit it (e.g. flying, no anti-air)
	}
	if h.attack == nil {
		h.fighting = false
		// Attack-move / patrol: back on the way once the fight is over.
		if h.amove && (!h.moving || h.target != h.amTarget) {
			h.setDestination(h.amTarget)
		}
		return
	}

	dist := h.Pos.Dist(h.attack.Pos)
	switch {
	case dist <= h.engageDist():
		// In range: hold position (soldiers do the fighting) and face the enemy.
		h.fighting = true
		h.moving = false
		if to := h.attack.Pos.Sub(h.Pos); !to.IsZero() {
			h.Facing = turnTowards(h.Facing, to.Angle(), turnRate.Mul(TickDt))
		}
	case h.hold || h.wall != nil || h.Rooted():
		// Holding (or on a wall): never chase. Keep the fight going while it is close, else let go.
		h.moving = false
		h.fighting = dist <= h.engageDist()+fightSlack
		if !h.fighting {
			h.attack = nil
		}
	default:
		// Chase. Keep fighting while the gap is small so soldiers don't snap back.
		h.fighting = dist <= h.engageDist()+fightSlack
		if !h.moving || h.target.Dist(h.attack.Pos) > fixed.FromInt(2) {
			h.target = h.attack.Pos
			h.moving = true
		}
	}
}

// updateSiege approaches the target building and fights once close enough.
func (h *Horde) updateSiege(w *World) {
	b := h.siege
	if b.DistTo(h.Pos) <= h.engageDist() {
		h.fighting = true
		h.moving = false
		if to := b.Closest(h.Pos).Sub(h.Pos); !to.IsZero() {
			h.Facing = turnTowards(h.Facing, to.Angle(), turnRate.Mul(TickDt))
		}
		return
	}
	h.fighting = false
	if !h.moving {
		// Stopped short (blocked approach point, pushed away): head in again.
		if p, ok := w.walkableTarget(b.approach(h.Pos, h.engageDist()-fixed.One)); ok && p != h.Pos {
			h.setDestination(p)
		}
	}
}

// updateVolley fires at the attack target when in range and reloaded.
func (h *Horde) updateVolley(w *World) {
	if h.Unit.Ranged == nil {
		return
	}
	if h.volleyCooldown > 0 {
		if len(h.Soldiers) > 0 {
			h.volleyCooldown = max(h.volleyCooldown-h.volleyStep(w), 0)
		}
		return
	}
	if !h.fighting || !h.Ranged() {
		return
	}
	switch {
	case h.hasFireAt:
		if d := h.Pos.Dist(h.fireAt); d > h.attackRange(h.Unit.Ranged)+engageBase || d < h.Unit.Ranged.MinRange {
			return
		}
		w.fireVolley(h, h.fireAt, w.enemyBuildingAt(h.Owner, h.fireAt))
	case h.siege != nil:
		if d := h.siege.DistTo(h.Pos); d > h.engageDist() || d < h.Unit.Ranged.MinRange {
			return
		}
		w.fireVolley(h, h.siege.Pos, h.siege)
	case h.attack != nil:
		if h.Pos.Dist(h.attack.Pos) > h.engageDist() {
			return
		}
		w.fireVolley(h, h.attack.Pos, nil)
	default:
		return
	}
	h.volleyCooldown = cooldownTicks(h.Unit.Ranged.Cooldown)
}

// waypoint returns where the horde should head this tick: straight to the
// target if the line is clear, otherwise the farthest visible cell along the
// flow field (string pulling), which smooths the 8-direction grid path.
func (h *Horde) waypoint(w *World) (fixed.Vec2, bool) {
	if h.Unit.Flying {
		return h.target, true // straight line over terrain, water and buildings
	}
	nv := w.navFor(h.Owner)
	g := nv.Grid()
	if h.Pos.Dist(h.target) <= directCheckRange && g.LineWalkable(h.Pos, h.target) {
		return h.target, true
	}
	field := nv.Field(g.IndexAt(h.target))
	cur := g.IndexAt(h.Pos)
	if !field.Reachable(cur) {
		// Standing on a blocked cell (pushed there) or cut off: try from the nearest walkable cell.
		cur = g.NearestWalkable(cur)
		if cur < 0 || !field.Reachable(cur) {
			return fixed.Vec2{}, false
		}
		return g.Center(cur), true
	}

	best := g.Center(cur)
	next := cur
	for k := 0; k < lookahead; k++ {
		n := int(field.Next[next])
		if n < 0 {
			break // reached the target cell
		}
		next = n
		c := g.Center(next)
		if !g.LineWalkable(h.Pos, c) {
			break
		}
		best = c
	}
	if next == field.Target && g.LineWalkable(h.Pos, h.target) {
		return h.target, true
	}
	return best, true
}

// slotPos returns the world position of formation slot i.
//
// Slots form a grid `columns` wide, centered on Pos; row 0 is the front line.
// The formation shrinks as soldiers die (slots are re-packed).
func (h *Horde) slotPos(i int) fixed.Vec2 {
	n := max(len(h.Soldiers), 1)
	if h.wall == nil && len(h.Soldiers) > 0 && h.Soldiers[0].mods.formation == data.FormationWedge {
		return h.wedgeSlot(i, n)
	}
	cols := min(h.Def.Formation.Columns, n)
	sp := h.Def.Formation.Spacing
	if h.wall != nil { // a narrow walkway: long and few rows deep
		cols, sp = h.wallSlotCols(), wallSpacing
		if h.wall.tower {
			sp = towerSpacing
		}
	}
	rows := (n + cols - 1) / cols

	col, row := i%cols, i/cols
	// Center the last (possibly shorter) row.
	inRow := cols
	if row == rows-1 && n%cols != 0 {
		inRow = n % cols
	}

	// Doubled offsets keep the half-spacing exact: lateral = (2*col - (inRow-1)) * sp / 2.
	lateral := sp.MulInt(2*col - (inRow - 1)).DivInt(2)
	depth := sp.MulInt(2*row - (rows - 1)).DivInt(2)

	forward := fixed.FromAngle(h.Facing)
	right := fixed.Vec2{X: forward.Y, Y: -forward.X}
	return h.Pos.Add(right.Scale(lateral)).Sub(forward.Scale(depth))
}

// turnTowards rotates `from` towards `to` by at most `step` radians.
func turnTowards(from, to, step fixed.Fixed) fixed.Fixed {
	d := fixed.NormalizeAngle(to - from)
	if d.Abs() <= step {
		return fixed.NormalizeAngle(to)
	}
	if d > 0 {
		return fixed.NormalizeAngle(from + step)
	}
	return fixed.NormalizeAngle(from - step)
}

// Rooted: the horde stands its ground (spear wall).
func (h *Horde) Rooted() bool { return len(h.Soldiers) > 0 && h.Soldiers[0].mods.rooted }

// wedgeSlot: slot i of a wedge — row r holds r+1 soldiers, row 0 is the point.
func (h *Horde) wedgeSlot(i, n int) fixed.Vec2 {
	rows := 0
	for c := 0; c < n; rows++ {
		c += rows + 1
	}
	row, first := 0, 0
	for first+row+1 <= i {
		first += row + 1
		row++
	}
	inRow := min(row+1, n-first)
	col := i - first
	sp := h.Def.Formation.Spacing
	lateral := sp.MulInt(2*col - (inRow - 1)).DivInt(2)
	depth := sp.MulInt(2*row - (rows - 1)).DivInt(2)
	forward := fixed.FromAngle(h.Facing)
	right := fixed.Vec2{X: forward.Y, Y: -forward.X}
	return h.Pos.Add(right.Scale(lateral)).Sub(forward.Scale(depth))
}

// speedNow: how fast the horde is going (m/s) — for spear walls.
func (h *Horde) speedNow(w *World) fixed.Fixed {
	if !(h.moving || h.charging) || len(h.Soldiers) == 0 {
		return 0
	}
	return h.Unit.Speed.Mul(h.momentum()).Mul(h.slowFactor(w))
}

// orderAttackGround: shoot at a ground point (ranged hordes only), walking into range first.
func (h *Horde) orderAttackGround(w *World, p fixed.Vec2) {
	if h.Unit.Ranged == nil {
		return
	}
	h.clearOrders()
	h.fireAt, h.hasFireAt = p, true
	h.melee = false
	h.ordered = true // ignore enemies on the way
	h.updateFireAt(w)
}

// updateFireAt: walk into range of the fire point (not on walls / rooted), then stand and shoot.
func (h *Horde) updateFireAt(w *World) {
	h.melee = false
	reach := h.attackRange(h.Unit.Ranged) + engageBase
	d := h.Pos.Dist(h.fireAt)
	switch {
	case d <= reach && d >= h.Unit.Ranged.MinRange:
		h.fighting = true
		h.moving = false
		if to := h.fireAt.Sub(h.Pos); !to.IsZero() {
			h.Facing = turnTowards(h.Facing, to.Angle(), turnRate.Mul(TickDt))
		}
	case h.wall != nil || h.Rooted() || h.hold:
		h.fighting = false // can't walk: wait until it is in range
	case !h.moving:
		h.fighting = false
		// To the point at range − 2 m along the line from the target (or back out of the dead zone).
		dir := h.Pos.Sub(h.fireAt).Normalize()
		if dir.IsZero() {
			dir = fixed.FromAngle(h.Facing)
		}
		want := max(reach-fixed.FromInt(2), h.Unit.Ranged.MinRange+fixed.FromInt(2))
		if p, ok := w.targetFor(h, h.fireAt.Add(dir.Scale(want))); ok {
			h.setDestination(p)
		}
	}
}

// enemyBuildingAt: an enemy building covering p (arrows shot there damage it), or nil.
func (w *World) enemyBuildingAt(owner PlayerID, p fixed.Vec2) *Building {
	for _, b := range w.Buildings {
		if b.Alive() && b.Attackable(owner) && b.Covers(p, fixed.One) {
			return b
		}
	}
	return nil
}
