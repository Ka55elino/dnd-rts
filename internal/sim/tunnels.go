package sim

import (
	"slices"

	"dnd-rts/internal/fixed"
)

// Tunnels (docs/FACTIONS.md, dwarves). Every finished own building with Def.Tunnel (and the
// fortress of a faction with FortressTunnel; Undermine's passage too) is an entrance of the
// owner's tunnel network. A horde ordered onto an own entrance walks to the entrance nearest to
// it, goes underground and comes out at the ordered one: the trip takes tunnelTravel, and an exit
// lets one horde out every tunnelGap. Underground hordes can't be seen, hit or ordered; if the
// exit is destroyed meanwhile they come out where they went in. Enemies can't use the tunnels.

const (
	tunnelTravel = 3 * TickRate
	tunnelGap    = 2 * TickRate
)

// tunnelEnter: a horde goes underground this close to the entrance.
var tunnelEnter = fixed.FromInt(5)

type tunnelTrip struct {
	entrance, exit *Building
	underground    bool
	arrive         uint32 // tick it comes out (underground)
}

// InTunnel reports whether the horde is inside the tunnel network.
func (h *Horde) InTunnel() bool { return h.tunnel != nil && h.tunnel.underground }

// IsTunnel reports whether b is an entrance of its owner's tunnel network.
func (w *World) IsTunnel(b *Building) bool {
	if !b.Alive() || b.UnderConstruction() || b.Owner == Neutral {
		return false
	}
	if b.Def.Tunnel {
		return true
	}
	if i := slices.Index(w.Players, b.Owner); i >= 0 && b == w.fortressOf(b.Owner) {
		return w.Catalog.Factions[w.factions[i]].FortressTunnel
	}
	return false
}

// fortressOf: player p's fortress, or nil.
func (w *World) fortressOf(p PlayerID) *Building {
	for _, b := range w.Buildings {
		if b.Owner == p && b.Def.Kind == "fortress" && b.Alive() {
			return b
		}
	}
	return nil
}

// tunnelAt: an own tunnel entrance whose footprint is at (or right next to) p.
func (w *World) tunnelAt(p PlayerID, pos fixed.Vec2) *Building {
	for _, b := range w.Buildings {
		if b.Owner == p && b.DistTo(pos) <= fixed.One && w.IsTunnel(b) {
			return b
		}
	}
	return nil
}

// orderTunnel sends h through the network to exit (or just walks there when the exit is the
// nearest entrance anyway).
func (w *World) orderTunnel(h *Horde, exit *Building) {
	if h.Unit.Flying || h.IsTrap() || h.Unit.Trap != nil {
		return
	}
	var entrance *Building
	for _, b := range w.Buildings {
		if b.Owner == h.Owner && w.IsTunnel(b) && (entrance == nil || b.DistTo(h.Pos) < entrance.DistTo(h.Pos)) {
			entrance = b
		}
	}
	h.clearOrders()
	if entrance == nil || entrance == exit {
		if p, ok := w.walkableTarget(exit.approach(h.Pos, fixed.FromInt(3))); ok {
			h.moveTo(p)
		}
		return
	}
	h.tunnel = &tunnelTrip{entrance: entrance, exit: exit}
	if p, ok := w.walkableTarget(entrance.approach(h.Pos, fixed.FromInt(2))); ok {
		h.ordered = true
		h.setDestination(p)
	}
}

// updateTunnels takes hordes in at their entrance and lets them out at the exit.
func (w *World) updateTunnels() {
	for _, h := range w.Hordes {
		t := h.tunnel
		if t == nil || !h.Alive() {
			continue
		}
		if !t.underground {
			if !w.IsTunnel(t.entrance) || !w.IsTunnel(t.exit) {
				h.tunnel = nil // an entrance fell: walk on foot
				continue
			}
			if t.entrance.DistTo(h.Pos) <= tunnelEnter {
				t.underground = true
				t.arrive = max(w.Tick+tunnelTravel, t.exit.tunnelFree)
				t.exit.tunnelFree = t.arrive + tunnelGap
				h.stop()
				h.attack, h.pending = nil, nil
			} else if !h.moving {
				h.tunnel = nil // stopped short (blocked)
			}
			continue
		}
		if w.Tick < t.arrive {
			continue
		}
		out := t.exit
		if !w.IsTunnel(out) {
			out = t.entrance // the exit fell: back out where it went in
			if !t.entrance.Alive() {
				out = nil
			}
		}
		h.tunnel = nil
		if out == nil {
			continue // both ends gone: it stays where it went in
		}
		w.placeHorde(h, out.approach(out.Pos.Add(fixed.FromAngle(out.Facing).Scale(fixed.FromInt(50))), fixed.FromInt(3)))
	}
}

// placeHorde puts h (all its soldiers) at p.
func (w *World) placeHorde(h *Horde, p fixed.Vec2) {
	np, ok := w.targetFor(h, p)
	if !ok {
		return
	}
	g := w.navFor(h.Owner).Grid()
	h.clearOrders()
	h.stop()
	h.wall = nil
	h.Pos = np
	for i, s := range h.Soldiers {
		sp := h.slotPos(i)
		if !g.WalkableAt(sp) {
			sp = np
		}
		s.Pos = sp
		s.target = nil
	}
}
