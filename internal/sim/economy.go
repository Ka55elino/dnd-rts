package sim

import (
	"slices"

	"dnd-rts/internal/data"
	"dnd-rts/internal/fixed"
)

// Economy (docs/RESOURCES.md). Stocks are whole numbers per resource in
// Catalog.ResourceIDs order; income accumulates exactly per effect; cap
// resources (food) are recomputed from buildings and hordes when needed.

const maxQueue = 5 // hordes queued per building

// queued is a horde being trained in a building.
type queued struct {
	def     *data.Horde   // a horde to train, or
	upgrade *data.Upgrade // an upgrade to research
	paid    data.Amounts  // what was paid (refund on cancel)
	ticks   int           // total training time
	done    int           // progress
}

// economy is a player's resource state.
type economy struct {
	stock    []int64 // per resource; cap resources stay 0 here
	upgrades []bool  // per Catalog.UpgradeIDs: researched
	heroes   []*heroRecord
}

func (w *World) initEconomy() {
	w.econ = make([]*economy, len(w.Players))
	for i := range w.econ {
		e := &economy{stock: make([]int64, len(w.Catalog.ResourceIDs)), upgrades: make([]bool, len(w.Catalog.UpgradeIDs))}
		for r, id := range w.Catalog.ResourceIDs {
			if res := w.Catalog.Resources[id]; res.Kind == data.ResourceStock {
				e.stock[r] = int64(res.Start)
			}
		}
		w.econ[i] = e
	}
}

func (w *World) economyOf(p PlayerID) *economy {
	i := slices.Index(w.Players, p)
	if i < 0 || i >= len(w.econ) {
		return nil
	}
	return w.econ[i]
}

// Stock returns a player's amount of a stock resource (by index into ResourceIDs).
func (w *World) Stock(p PlayerID, r int) int64 {
	if e := w.economyOf(p); e != nil {
		return e.stock[r]
	}
	return 0
}

// CapProvided is a cap resource's limit for the player: base + finished buildings, clamped to max.
func (w *World) CapProvided(p PlayerID, r int) int {
	id := w.Catalog.ResourceIDs[r]
	res := w.Catalog.Resources[id]
	total := fixed.FromInt(res.Start)
	for _, b := range w.Buildings {
		if b.Owner != p || !b.Alive() || b.UnderConstruction() {
			continue
		}
		for _, e := range b.Def.Economy {
			if e.Type == data.EffectProvide && e.Resource == id {
				total += e.Amount
			}
		}
	}
	n := total.Int()
	if res.Max > 0 {
		n = min(n, res.Max)
	}
	return n
}

// CapUsed is how much of a cap resource the player's hordes (alive and queued) take.
func (w *World) CapUsed(p PlayerID, r int) int {
	id := w.Catalog.ResourceIDs[r]
	n := 0
	for _, h := range w.Hordes {
		if h.Owner == p && !h.summoned {
			n += h.Def.Uses[id]
		}
	}
	for _, b := range w.Buildings {
		if b.Owner != p {
			continue
		}
		for _, q := range b.queue {
			if q.def != nil {
				n += q.def.Uses[id]
			}
		}
	}
	return n
}

// CanAfford reports whether the player has every stock resource of cost.
func (w *World) CanAfford(p PlayerID, cost data.Amounts) bool {
	e := w.economyOf(p)
	if e == nil {
		return len(cost) == 0
	}
	for r, id := range w.Catalog.ResourceIDs {
		if int64(cost[id]) > e.stock[r] {
			return false
		}
	}
	return true
}

// hasRoom reports whether the player's cap resources can take `uses` more.
func (w *World) hasRoom(p PlayerID, uses data.Amounts) bool {
	for r, id := range w.Catalog.ResourceIDs {
		if need := uses[id]; need > 0 && w.CapUsed(p, r)+need > w.CapProvided(p, r) {
			return false
		}
	}
	return true
}

// pay takes cost from the player (all or nothing). Returns false if it can't.
func (w *World) pay(p PlayerID, cost data.Amounts) bool {
	if !w.CanAfford(p, cost) {
		return false
	}
	if e := w.economyOf(p); e != nil {
		for r, id := range w.Catalog.ResourceIDs {
			e.stock[r] -= int64(cost[id])
		}
	}
	return true
}

// gain adds amounts to the player's stocks, respecting storage limits.
func (w *World) gain(p PlayerID, r int, n int64) {
	e := w.economyOf(p)
	if e == nil {
		return
	}
	e.stock[r] += n
	if m := w.Catalog.Resources[w.Catalog.ResourceIDs[r]].Max; m > 0 {
		e.stock[r] = min(e.stock[r], int64(m))
	}
}

func (w *World) refund(p PlayerID, cost data.Amounts) {
	for r, id := range w.Catalog.ResourceIDs {
		if n := cost[id]; n > 0 {
			w.gain(p, r, int64(n))
		}
	}
}

// updateEconomy pays out income effects of finished buildings to their owners.
// Each effect accumulates `amount` per tick and pays one unit per `every`
// seconds' worth of ticks, so integer incomes are exact.
func (w *World) updateEconomy() {
	for _, b := range w.Buildings {
		if b.Owner == Neutral || !b.Alive() || b.UnderConstruction() {
			continue
		}
		for i, e := range b.Def.Economy {
			if e.Type != data.EffectIncome {
				continue
			}
			if b.acc == nil {
				b.acc = make([]fixed.Fixed, len(b.Def.Economy))
			}
			period := cooldownTicks(e.Every)
			b.acc[i] += e.Amount
			if units := b.acc[i].DivInt(period).Int(); units > 0 {
				b.acc[i] -= fixed.FromInt(units).MulInt(period)
				w.gain(b.Owner, w.Catalog.ResourceIndex(e.Resource), int64(units))
				b.lastIncome = w.Tick
			}
		}
	}
}

// train queues a horde in a building after paying for it.
func (w *World) train(player PlayerID, b *Building, hordeType string) {
	def, ok := w.Catalog.Hordes[hordeType]
	if !ok || b.Owner != player || !b.Alive() || b.UnderConstruction() || len(b.queue) >= maxQueue {
		return
	}
	if !slices.Contains(b.Def.Trains, hordeType) {
		return
	}
	if f := w.faction(player); f == nil || !slices.Contains(f.Hordes, hordeType) {
		return
	}
	if !w.Meets(player, def.Requires) {
		return
	}
	cost, time := def.Cost, def.BuildTime
	if def.Unique {
		r := w.heroRecordOf(player, hordeType)
		if (r != nil && r.alive) || w.inAnyQueue(player, hordeType) {
			return // one at a time
		}
		if r != nil { // revive
			if len(def.ReviveCost) > 0 {
				cost = def.ReviveCost
			}
			time = def.BuildTime.MulInt(2)
			if def.ReviveTime > 0 {
				time = def.ReviveTime
			}
		}
	}
	if !w.hasRoom(player, def.Uses) || !w.pay(player, cost) {
		return
	}
	b.queue = append(b.queue, queued{def: def, paid: cost, ticks: max(1, cooldownTicks(time))})
}

// cancelTraining removes queue entry i and refunds it.
func (w *World) cancelTraining(player PlayerID, b *Building, i int) {
	if b.Owner != player || i < 0 || i >= len(b.queue) {
		return
	}
	if q := b.queue[i]; q.def != nil {
		w.refund(player, q.paid)
	} else {
		w.refund(player, q.upgrade.Cost)
	}
	b.queue = slices.Delete(b.queue, i, i+1)
}

// updateTraining advances the front of every queue and sends finished hordes out.
func (w *World) updateTraining() {
	for _, b := range w.Buildings {
		if len(b.queue) == 0 || !b.Alive() || b.UnderConstruction() || b.Owner == Neutral {
			continue
		}
		q := &b.queue[0]
		b.workAcc += mul(b.workSpeed) // auras speed training up
		n := b.workAcc.Int()
		b.workAcc -= fixed.FromInt(n)
		q.done += n
		if q.done < q.ticks {
			continue
		}
		def := q.def
		if q.upgrade != nil {
			w.completeUpgrade(b.Owner, q.upgrade)
			b.queue = slices.Delete(b.queue, 0, 1)
			continue
		}
		b.queue = slices.Delete(b.queue, 0, 1)
		exit, ok := w.walkableTarget(b.Local(fixed.Vec2{Y: b.Def.Footprint.Y.DivInt(2) + fixed.FromInt(6)}))
		if !ok {
			continue
		}
		h, err := w.SpawnHorde(b.Owner, def.ID, exit, b.Facing)
		if err == nil && b.hasRally {
			if p, ok := w.walkableTarget(b.rally); ok {
				h.moveTo(p)
			}
		}
	}
}

// QueueInfo describes a building's training queue for the client.
type QueueInfo struct {
	Hordes   []string    // per entry: horde type id, or "" for research (see Upgrades)
	Upgrades []string    // per entry: upgrade id, or "" for a horde
	Progress fixed.Fixed // of the front entry, [0, 1]
}

// Queue returns the building's training queue (nil if empty).
func (b *Building) Queue() *QueueInfo {
	if len(b.queue) == 0 {
		return nil
	}
	q := &QueueInfo{Progress: fixed.FromInt(b.queue[0].done).DivInt(b.queue[0].ticks)}
	for _, e := range b.queue {
		if e.def != nil {
			q.Hordes = append(q.Hordes, e.def.ID)
			q.Upgrades = append(q.Upgrades, "")
		} else {
			q.Hordes = append(q.Hordes, "")
			q.Upgrades = append(q.Upgrades, e.upgrade.ID)
		}
	}
	return q
}

// LastIncome returns the tick of the building's latest income payout (for client effects).
func (b *Building) LastIncome() uint32 { return b.lastIncome }

// Meets reports whether the player fulfils r: owns a finished building of
// every listed type and has every listed upgrade.
func (w *World) Meets(p PlayerID, r *data.Requires) bool {
	if r.IsEmpty() {
		return true
	}
	for _, id := range r.Buildings {
		if !slices.ContainsFunc(w.Buildings, func(b *Building) bool {
			return b.Owner == p && b.Def.ID == id && b.Alive() && !b.UnderConstruction()
		}) {
			return false
		}
	}
	for _, id := range r.Upgrades {
		if !w.HasUpgrade(p, id) {
			return false
		}
	}
	return true
}

// HasUpgrade reports whether the player researched the upgrade.
func (w *World) HasUpgrade(p PlayerID, id string) bool {
	e := w.economyOf(p)
	i := slices.Index(w.Catalog.UpgradeIDs, id)
	return e != nil && i >= 0 && e.upgrades[i]
}

// Upgrades returns the player's researched upgrade ids (sorted).
func (w *World) Upgrades(p PlayerID) []string {
	e := w.economyOf(p)
	if e == nil {
		return nil
	}
	var out []string
	for i, on := range e.upgrades {
		if on {
			out = append(out, w.Catalog.UpgradeIDs[i])
		}
	}
	return out
}

// researching reports whether the player has the upgrade in any queue.
func (w *World) researching(p PlayerID, id string) bool {
	for _, b := range w.Buildings {
		if b.Owner != p {
			continue
		}
		for _, q := range b.queue {
			if q.upgrade != nil && q.upgrade.ID == id {
				return true
			}
		}
	}
	return false
}

// research queues an upgrade in a building after paying for it.
func (w *World) research(player PlayerID, b *Building, id string) {
	g, ok := w.Catalog.Upgrades[id]
	if !ok || b.Owner != player || !b.Alive() || b.UnderConstruction() || len(b.queue) >= maxQueue {
		return
	}
	if !slices.Contains(b.Def.Research, id) || w.HasUpgrade(player, id) || w.researching(player, id) {
		return
	}
	if !w.Meets(player, g.Requires) || !w.pay(player, g.Cost) {
		return
	}
	b.queue = append(b.queue, queued{upgrade: g, ticks: max(1, cooldownTicks(g.ResearchTime))})
}

// completeUpgrade marks an upgrade researched; others of its group are replaced.
func (w *World) completeUpgrade(p PlayerID, g *data.Upgrade) {
	e := w.economyOf(p)
	if e == nil {
		return
	}
	for i, id := range w.Catalog.UpgradeIDs {
		if id == g.ID {
			e.upgrades[i] = true
		} else if g.Group != "" && w.Catalog.Upgrades[id].Group == g.Group {
			e.upgrades[i] = false
		}
	}
}

// inAnyQueue reports whether the player is training the horde type anywhere.
func (w *World) inAnyQueue(p PlayerID, hordeType string) bool {
	for _, b := range w.Buildings {
		if b.Owner != p {
			continue
		}
		for _, q := range b.queue {
			if q.def != nil && q.def.ID == hordeType {
				return true
			}
		}
	}
	return false
}
