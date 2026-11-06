package sim

import (
	"slices"

	"dnd-rts/internal/data"
	"dnd-rts/internal/fixed"
)

// Invisibility (docs/FACTIONS.md: camouflage, Shadow, Forest Shadow, traps). A horde with the
// Invisible modifier (or Camouflage while it stands still; with InTrees only near trees) is hidden
// from enemies: they don't target it, towers don't shoot it and the client of another player
// doesn't get it in the snapshot — unless a detector of that player is close enough (towers with
// True Sight / Rune Sight / invisibility detection, units with the Detect modifier). Attacking or
// casting reveals the horde for a few seconds. A stealthy horde doesn't pick fights on its own.
//
// Smoke (Smoke Screen): hordes inside a smoke cloud are hidden from everyone outside it; whoever
// has a horde inside the same cloud sees them.
//
// A horde inside the tunnel network is seen by nobody (not even its owner's client).

// revealTicks: how long an invisible horde stays visible after it attacks or casts.
const revealTicks = 3 * TickRate

// treeCover: how close to a tree a horde with InTrees camouflage must be.
var treeCover = fixed.FromInt(5)

// reveal shows an invisible horde for a while (it just attacked or cast).
func (h *Horde) reveal(w *World) { h.revealUntil = w.Tick + revealTicks }

// Hidden reports whether the horde is concealed this tick (before detection).
func (h *Horde) Hidden() bool { return h.hidden }

// VisibleTo reports whether player p sees horde h: own hordes always (unless in a tunnel);
// others when not concealed from p and, with the fog of war on, in a cell p sees now.
func (w *World) VisibleTo(h *Horde, p PlayerID) bool {
	if h.InTunnel() {
		return false
	}
	if h.Owner == p {
		return true
	}
	if h.hidden && (p >= 64 || h.seenBy&(1<<p) == 0) {
		return false
	}
	return w.fogVisible(p, h.Pos)
}

// Sees reports whether player p's client gets horde h in its snapshot.
func (w *World) Sees(p PlayerID, h *Horde) bool { return w.VisibleTo(h, p) }

// updateConcealment decides which hordes are hidden this tick and which players detect them.
func (w *World) updateConcealment() {
	any := false
	for _, h := range w.Hordes {
		h.hidden, h.stealth, h.seenBy = false, false, 0
		if len(h.Soldiers) == 0 || h.InTunnel() {
			continue
		}
		if w.Tick >= h.revealUntil {
			m := &h.Soldiers[0].mods
			if (m.invisible || (m.camouflage && !h.moving)) && (!m.inTrees || w.nearTree(h.Pos, treeCover)) {
				h.hidden, h.stealth = true, true
			}
		}
		any = any || h.hidden
	}
	// Smoke clouds: who is inside each one.
	for _, z := range w.Zones {
		if !z.smoke {
			continue
		}
		r2 := z.Radius.Mul(z.Radius)
		var inside uint64
		var in []*Horde
		for _, h := range w.Hordes {
			if len(h.Soldiers) > 0 && !h.InTunnel() && !h.Unit.Flying && h.Pos.DistSq(z.Pos) <= r2 {
				in = append(in, h)
				if h.Owner < 64 {
					inside |= 1 << h.Owner
				}
			}
		}
		for _, h := range in {
			if !h.hidden {
				h.hidden, h.seenBy = true, inside
			} else {
				h.seenBy |= inside
			}
			any = true
		}
	}
	if !any {
		return
	}
	type detector struct {
		owner  PlayerID
		pos    fixed.Vec2
		radius fixed.Fixed
	}
	var ds []detector
	for _, h := range w.Hordes {
		if len(h.Soldiers) > 0 && h.Soldiers[0].mods.detect > 0 && !h.InTunnel() {
			ds = append(ds, detector{h.Owner, h.Pos, h.Soldiers[0].mods.detect})
		}
	}
	for _, b := range w.Buildings {
		if !b.Alive() || b.UnderConstruction() || b.Owner == Neutral {
			continue
		}
		if r := w.towerMods(b).detect; r > 0 {
			ds = append(ds, detector{b.Owner, b.Pos, r})
		}
	}
	for _, h := range w.Hordes {
		if !h.hidden {
			continue
		}
		for _, d := range ds {
			if d.owner != h.Owner && d.owner < 64 && h.Pos.DistSq(d.pos) <= d.radius.Mul(d.radius) {
				h.seenBy |= 1 << d.owner
			}
		}
	}
}

// ---------------------------------------------------------------- fog of war

// Fog of war (docs/FACTIONS.md: vision and fog of war). The map is split into FogCell cells; a
// player sees the cells within the sight radius of own hordes and buildings (and of reveals:
// Recon Shell). Cells seen once stay explored (the client draws them dimmed, buildings stay on
// it). Enemy hordes in cells the player doesn't see now are neither targetable nor sent to its
// client. Off unless World.FogEnabled (skirmish turns it on; the dev console toggles it).

// FogCell is the size of one fog cell (world units).
var FogCell = fixed.FromInt(8)

// fogEvery: the fog is recomputed every this many ticks (units move ~1 m in that time).
const fogEvery = 4

// Fog cell states sent to the client.
const (
	FogUnexplored = 0
	FogExplored   = 1
	FogVisible    = 2
)

type fogGrid struct {
	cols, rows int
	visible    [][]bool  // per entry of Players
	explored   [][]bool  // per entry of Players
	reveals    []revealZ // temporary vision (Recon Shell)
}

type revealZ struct {
	owner  PlayerID
	pos    fixed.Vec2
	radius fixed.Fixed
	until  uint32
}

// SetFog turns the fog of war on or off.
func (w *World) SetFog(on bool) {
	w.FogEnabled = on
	w.fog.visible = nil // rebuilt on the next update
	w.updateFog(true)
}

// FogSize is the fog grid's size in cells.
func (w *World) FogSize() (cols, rows int) {
	return fogDim(w.Terrain.Width), fogDim(w.Terrain.Height)
}

func fogDim(size int) int { return fixed.FromInt(size).Div(FogCell).Ceil().Int() }

// fogIndex: the cell of a world point (clamped to the map).
func (w *World) fogIndex(p fixed.Vec2) int {
	f := &w.fog
	c := (p.X + fixed.FromInt(w.Terrain.Width)/2).Div(FogCell).Int()
	r := (p.Y + fixed.FromInt(w.Terrain.Height)/2).Div(FogCell).Int()
	return min(max(r, 0), f.rows-1)*f.cols + min(max(c, 0), f.cols-1)
}

// fogVisible: player p sees the point now (always without fog, for neutral and unknown players).
func (w *World) fogVisible(p PlayerID, pos fixed.Vec2) bool {
	if !w.FogEnabled || w.fog.visible == nil {
		return true
	}
	i := slices.Index(w.Players, p)
	if i < 0 {
		return true
	}
	return w.fog.visible[i][w.fogIndex(pos)]
}

// FogState returns player p's fog grid (FogUnexplored / FogExplored / FogVisible per cell, row by
// row from -x,-y), or nil without the fog of war.
func (w *World) FogState(p PlayerID) []byte {
	i := slices.Index(w.Players, p)
	if !w.FogEnabled || w.fog.visible == nil || i < 0 {
		return nil
	}
	out := make([]byte, len(w.fog.visible[i]))
	for k, v := range w.fog.visible[i] {
		switch {
		case v:
			out[k] = FogVisible
		case w.fog.explored[i][k]:
			out[k] = FogExplored
		}
	}
	return out
}

// addReveal lets owner see an area for a while (Recon Shell).
func (w *World) addReveal(owner PlayerID, pos fixed.Vec2, radius, seconds fixed.Fixed) {
	w.fog.reveals = append(w.fog.reveals, revealZ{owner, pos, radius, w.Tick + uint32(cooldownTicks(seconds))})
	w.updateFog(true)
}

// updateFog recomputes what every player sees (every fogEvery ticks, or now when forced).
func (w *World) updateFog(force bool) {
	kept := w.fog.reveals[:0]
	for _, r := range w.fog.reveals {
		if w.Tick < r.until {
			kept = append(kept, r)
		}
	}
	w.fog.reveals = kept
	if !w.FogEnabled || (!force && w.Tick%fogEvery != 0 && w.fog.visible != nil) {
		return
	}
	f := &w.fog
	if f.visible == nil || len(f.visible) != len(w.Players) {
		f.cols, f.rows = w.FogSize()
		f.visible = make([][]bool, len(w.Players))
		f.explored = make([][]bool, len(w.Players))
		for i := range w.Players {
			f.visible[i] = make([]bool, f.cols*f.rows)
			f.explored[i] = make([]bool, f.cols*f.rows)
		}
	}
	for i := range w.Players {
		clear(f.visible[i])
	}
	for _, h := range w.Hordes {
		if len(h.Soldiers) == 0 || h.InTunnel() {
			continue
		}
		w.see(h.Owner, h.Pos, unitSight(h.Unit))
	}
	for _, b := range w.Buildings {
		if !b.Alive() || b.Owner == Neutral {
			continue
		}
		sight := b.Def.Sight
		if sight == 0 {
			sight = data.DefaultSight
			if a := b.Def.Attack; a != nil {
				sight = max(sight, a.Range+fixed.FromInt(4))
			}
			sight += max(b.Def.Footprint.X, b.Def.Footprint.Y) / 2
		}
		w.see(b.Owner, b.Pos, sight)
	}
	for _, r := range f.reveals {
		w.see(r.owner, r.pos, r.radius)
	}
}

// unitSight: the unit's own sight, or the default — ranged units see as far as they shoot
// (siege engines don't: they need someone to spot for them).
func unitSight(u *data.Unit) fixed.Fixed {
	if u.Sight > 0 {
		return u.Sight
	}
	s := data.DefaultSight
	if u.Ranged != nil && !u.HasTag("siege") {
		s = max(s, u.Ranged.Range+fixed.FromInt(2))
	}
	return s
}

// see marks the cells within radius of pos as visible (and explored) for player p.
func (w *World) see(p PlayerID, pos fixed.Vec2, radius fixed.Fixed) {
	i := slices.Index(w.Players, p)
	if i < 0 {
		return
	}
	f := &w.fog
	vis, exp := f.visible[i], f.explored[i]
	hw, hh := fixed.FromInt(w.Terrain.Width)/2, fixed.FromInt(w.Terrain.Height)/2
	span := radius.Div(FogCell).Ceil().Int()
	c0 := (pos.X + hw).Div(FogCell).Int()
	r0 := (pos.Y + hh).Div(FogCell).Int()
	r2 := radius.Mul(radius)
	half := FogCell / 2
	for r := max(r0-span, 0); r <= min(r0+span, f.rows-1); r++ {
		for c := max(c0-span, 0); c <= min(c0+span, f.cols-1); c++ {
			centre := fixed.Vec2{X: FogCell.MulInt(c) + half - hw, Y: FogCell.MulInt(r) + half - hh}
			if centre.DistSq(pos) <= r2 {
				vis[r*f.cols+c] = true
				exp[r*f.cols+c] = true
			}
		}
	}
}
