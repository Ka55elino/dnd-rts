package data

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"slices"

	"dnd-rts/internal/fixed"
)

// Base layouts (docs/BASES.md). A fortress describes its base with a
// generator — a contour (circle / square / polygon) with a wall, gates,
// tower sockets on the wall and rings of build plots — which Load expands
// into concrete plots. PvP maps use the generator; campaign maps will place
// bases by hand in the map editor (same Plot / BaseLayout data).

// Plot size classes.
const (
	PlotSmall  = "s"
	PlotMedium = "m"
	PlotLarge  = "l"
)

// PlotSizes: side of a plot per size class (content/plots.json).
type PlotSizes map[string]fixed.Fixed

// BaseLayout is a fortress's base, relative to the fortress (x along its
// right, y along its front). Angles are degrees, clockwise from the front.
type BaseLayout struct {
	Shape  string      `json:"shape"`            // circle, square or polygon
	Sides  int         `json:"sides,omitempty"`  // polygon: number of sides (>= 5)
	Radius fixed.Fixed `json:"radius"`           // contour: circle radius, half the square's side, polygon circumradius
	Rotate fixed.Fixed `json:"rotate,omitempty"` // square / polygon: turn of the contour, degrees

	Rings  []PlotRing    `json:"rings,omitempty"`
	Gates  []fixed.Fixed `json:"gates,omitempty"`  // gate directions
	Towers int           `json:"towers,omitempty"` // tower sockets on the wall, evenly spaced, kept clear of gates

	// The wall: segments of building Wall around the contour; at each gate one
	// Gate building (a gatehouse) in place of GateWidth segments (default: its
	// footprint length over the wall's). Stairs up to the walkway on
	// the inside: next to every gate, plus Stairs more spread around.
	Wall      string `json:"wall,omitempty"`
	Gate      string `json:"gate,omitempty"`
	GateWidth int    `json:"gateWidth,omitempty"`
	Stairs    int    `json:"stairs,omitempty"`

	// Filled by Load.
	Outline      []fixed.Vec2 `json:"-"` // contour corners (circles: 32-gon)
	GatePos      []fixed.Vec2 `json:"-"`
	TowerSockets []fixed.Vec2 `json:"-"`
	Segments     []WallSlot   `json:"-"` // around the contour, in order (the walkway is this loop)
	StairList    []WallStair  `json:"-"`
}

// WallSlot is one segment of a base wall, relative to the fortress.
type WallSlot struct {
	Offset fixed.Vec2 // centre
	Out    fixed.Vec2 // unit vector pointing out of the base (the segment's front)
	Gate   bool       // a gatehouse: one building spanning Span wall segments
	Span   int
}

// WallStair connects the ground inside the base with walkway node Node.
type WallStair struct {
	Node   int
	Ground fixed.Vec2 // relative to the fortress
}

// PlotRing places Count plots evenly on a circle; sizes cycle through Sizes.
type PlotRing struct {
	Radius fixed.Fixed `json:"radius"`
	Count  int         `json:"count"`
	Offset fixed.Fixed `json:"offset,omitempty"` // angle of the first plot, degrees
	Sizes  []string    `json:"sizes"`            // size classes, repeated around the ring
}

// maxPlots: the snapshot carries a 48-bit mask of occupied plots per building.
const maxPlots = 48

// PlotTower: the class of tower sockets on a base wall (plots of the fortress).
const PlotTower = "tower"

// gateClear: tower sockets keep this many degrees away from a gate.
var gateClear = fixed.FromInt(12)

func loadPlotSizes(fsys fs.FS, into *PlotSizes) []error {
	raw, err := fs.ReadFile(fsys, "plots.json")
	if err != nil {
		return []error{fmt.Errorf("plots.json: %w", err)}
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(into); err != nil {
		return []error{fmt.Errorf("plots.json: %w", err)}
	}
	return nil
}

// polar: offset at angle a (degrees, clockwise from the front) and radius r.
func polar(a, r fixed.Fixed) fixed.Vec2 {
	rad := a.Mul(fixed.Pi).DivInt(180)
	return fixed.Vec2{X: fixed.Sin(rad).Mul(r), Y: fixed.Cos(rad).Mul(r)}
}

// expandBases turns base layouts into plots and contour points, and fills plot
// sizes from their class. Runs before Validate.
func (c *Catalog) expandBases() {
	for _, id := range c.BuildingIDs {
		b := c.Buildings[id]
		for i := range b.Plots { // explicit plots may name a class instead of a size
			if p := &b.Plots[i]; p.Size == 0 && p.Class != "" {
				p.Size = c.PlotSizes[p.Class]
			}
		}
		l := b.Base
		if l == nil {
			continue
		}
		for _, r := range l.Rings {
			for k := 0; k < r.Count && len(r.Sizes) > 0; k++ {
				a := r.Offset + fixed.FromInt(360).MulInt(k).DivInt(r.Count)
				class := r.Sizes[k%len(r.Sizes)]
				b.Plots = append(b.Plots, Plot{Offset: polar(a, r.Radius), Size: c.PlotSizes[class], Class: class})
			}
		}
		// Contour.
		n := 32
		switch l.Shape {
		case "square":
			n = 4
		case "polygon":
			n = l.Sides
		}
		l.Outline = nil
		if n >= 3 {
			r := l.Radius
			start := l.Rotate
			if l.Shape == "square" {
				r = l.Radius.Mul(fixed.FromFraction(14142, 10000)) // corners of a square of half side Radius
				start += fixed.FromInt(45)
			}
			for k := 0; k < n; k++ {
				l.Outline = append(l.Outline, polar(start+fixed.FromInt(360).MulInt(k).DivInt(n), r))
			}
		}
		l.GatePos = nil
		for _, g := range l.Gates {
			l.GatePos = append(l.GatePos, l.wallPoint(g))
		}
		l.TowerSockets = nil
		for k := 0; k < l.Towers; k++ {
			a := fixed.FromInt(360).MulInt(2*k + 1).DivInt(2 * l.Towers) // between evenly spaced spokes
			if l.nearGate(a) {
				continue
			}
			l.TowerSockets = append(l.TowerSockets, l.wallPoint(a))
		}
		c.expandWall(b, l)
		// Free buildings stay outside the wall.
		b.NoBuildRadius = max(b.NoBuildRadius, l.Radius+fixed.FromInt(4)+c.wallThickness(l))
	}
}

// wallPoint: where the wall runs at angle a (on the contour).
func (l *BaseLayout) wallPoint(a fixed.Fixed) fixed.Vec2 {
	if l.Shape == "circle" || len(l.Outline) < 3 {
		return polar(a, l.Radius)
	}
	// Ray from the centre against each contour edge.
	dir := polar(a, fixed.One)
	best := polar(a, l.Radius)
	for i := range l.Outline {
		p, q := l.Outline[i], l.Outline[(i+1)%len(l.Outline)]
		e := q.Sub(p)
		den := dir.Cross(e)
		if den == 0 {
			continue
		}
		t := p.Cross(e).Div(den)
		u := p.Cross(dir).Div(den)
		if t > 0 && u >= 0 && u <= fixed.One {
			best = dir.Scale(t)
		}
	}
	return best
}

func (l *BaseLayout) nearGate(a fixed.Fixed) bool {
	for _, g := range l.Gates {
		d := (a - g).Abs()
		d = min(d, fixed.FromInt(360)-d)
		if d < gateClear {
			return true
		}
	}
	return false
}

func (c *Catalog) validateBases(fail func(kind, id, format string, args ...any)) {
	for _, class := range []string{PlotSmall, PlotMedium, PlotLarge, PlotTower} {
		if c.PlotSizes[class] <= 0 {
			fail("plots", class, "plots.json needs a size > 0 for %q", class)
		}
	}
	if !(c.PlotSizes[PlotSmall] < c.PlotSizes[PlotMedium] && c.PlotSizes[PlotMedium] < c.PlotSizes[PlotLarge]) {
		fail("plots", "sizes", "plot sizes must grow: s < m < l")
	}
	for _, id := range c.BuildingIDs {
		b := c.Buildings[id]
		if l := b.Base; l != nil {
			if b.Kind != BuildingFortress {
				fail("building", id, "only fortresses have a base layout")
			}
			if !slices.Contains([]string{"circle", "square", "polygon"}, l.Shape) || (l.Shape == "polygon" && l.Sides < 5) {
				fail("building", id, "base.shape must be circle, square or polygon (sides >= 5)")
			}
			if l.Radius <= 0 {
				fail("building", id, "base.radius must be > 0")
			}
			if l.Wall != "" {
				if w, ok := c.Buildings[l.Wall]; !ok || w.Kind != BuildingWall {
					fail("building", id, "base.wall %q must be a building of kind %q", l.Wall, BuildingWall)
				}
				if g, ok := c.Buildings[l.Gate]; len(l.Gates) > 0 && (!ok || g.Kind != BuildingGate) {
					fail("building", id, "base.gate %q must be a building of kind %q", l.Gate, BuildingGate)
				}
				for _, s := range l.Segments {
					if g, ok := c.Buildings[l.Gate]; ok && s.Gate {
						want := c.Buildings[l.Wall].Footprint.X.MulInt(s.Span)
						if (g.Footprint.X - want).Abs() > fixed.One {
							fail("building", id, "base.gate %q is %v long but spans %d segments (%v)", l.Gate, g.Footprint.X, s.Span, want)
						}
					}
				}
			}
			for i, r := range l.Rings {
				if r.Radius <= 0 || r.Count <= 0 || len(r.Sizes) == 0 {
					fail("building", id, "base.rings[%d]: needs radius > 0, count > 0 and sizes", i)
				}
				for _, s := range r.Sizes {
					if _, ok := c.PlotSizes[s]; !ok {
						fail("building", id, "base.rings[%d]: unknown plot size %q (s, m, l)", i, s)
					}
				}
			}
		}
		for i, p := range b.Plots {
			if p.Class != "" && c.PlotSizes[p.Class] == 0 {
				fail("building", id, "plot %d: unknown size class %q", i, p.Class)
			}
		}
		if len(b.Plots) > maxPlots {
			fail("building", id, "at most %d plots (has %d)", maxPlots, len(b.Plots))
		}
		// Plots must not overlap each other or the building, and stay inside the wall.
		half := func(p Plot) fixed.Fixed { return p.Size.DivInt(2) }
		for i, p := range b.Plots {
			if p.Class == PlotTower {
				continue // on the wall, not inside the base
			}
			if (p.Offset.X.Abs() < b.Footprint.X.DivInt(2)+half(p)) && (p.Offset.Y.Abs() < b.Footprint.Y.DivInt(2)+half(p)) {
				fail("building", id, "plot %d overlaps the building", i)
			}
			for j := i + 1; j < len(b.Plots); j++ {
				q := b.Plots[j]
				if q.Class == PlotTower {
					continue
				}
				if (p.Offset.X-q.Offset.X).Abs() < half(p)+half(q) && (p.Offset.Y-q.Offset.Y).Abs() < half(p)+half(q) {
					fail("building", id, "plots %d and %d overlap", i, j)
				}
			}
			if l := b.Base; l != nil {
				// The plot's far corner must stay inside the contour's inner circle.
				corner := fixed.Vec2{X: p.Offset.X.Abs() + half(p), Y: p.Offset.Y.Abs() + half(p)}
				inner := l.Radius
				if l.Shape != "circle" && len(l.Outline) >= 3 {
					inner = l.Outline[0].Add(l.Outline[1]).Scale(fixed.One / 2).Len() // apothem
				}
				inner -= c.wallThickness(l) // the wall's inner face
				if corner.Len() > inner {
					fail("building", id, "plot %d reaches past the base wall", i)
				}
			}
		}
	}
}

func (c *Catalog) wallThickness(l *BaseLayout) fixed.Fixed {
	if w, ok := c.Buildings[l.Wall]; ok {
		return w.Footprint.Y.DivInt(2)
	}
	return 0
}

// expandWall lays wall segments around the contour, gates, stairs and the
// tower sockets (as plots of class "tower", after the building plots).
func (c *Catalog) expandWall(b *Building, l *BaseLayout) {
	l.Segments, l.StairList = nil, nil
	wall, ok := c.Buildings[l.Wall]
	if !ok || wall.Footprint.X <= 0 || len(l.Outline) < 3 {
		return
	}
	// Perimeter → number of segments (each at most the wall's footprint length).
	per := fixed.Fixed(0)
	for i := range l.Outline {
		per += l.Outline[(i+1)%len(l.Outline)].Sub(l.Outline[i]).Len()
	}
	n := per.Div(wall.Footprint.X).Ceil().Int()
	gate, hasGate := c.Buildings[l.Gate]
	if l.Shape == "circle" && hasGate && len(l.Gates) > 0 {
		// Gatehouses exactly at the gate directions; the arcs between them
		// filled with whole segments (chord <= footprint).
		c.circleWithGates(l, wall, gate)
	} else {
		if l.Shape == "circle" {
			// True circle: n segments on the circle itself (chord <= footprint).
			for k := 0; k < n; k++ {
				a := fixed.FromInt(360).MulInt(2*k + 1).DivInt(2 * n) // segment centres between spokes
				p := polar(a, l.Radius)
				l.Segments = append(l.Segments, WallSlot{Offset: p, Out: polar(a, fixed.One)})
			}
		} else {
			// Along each edge, evenly.
			for i := range l.Outline {
				p, q := l.Outline[i], l.Outline[(i+1)%len(l.Outline)]
				e := q.Sub(p)
				m := max(1, e.Len().Div(wall.Footprint.X).Ceil().Int())
				out := fixed.Vec2{X: e.Y, Y: -e.X}.Normalize() // clockwise loop: right of the edge is outside
				if out.Dot(p.Add(q)) < 0 {
					out = out.Scale(-fixed.One)
				}
				for k := 0; k < m; k++ {
					l.Segments = append(l.Segments, WallSlot{Offset: p.Add(e.Scale(fixed.FromInt(2*k + 1).DivInt(2 * m))), Out: out})
				}
			}
		}
		// Gates: the GateWidth segments closest to each gate direction.
		gw := l.GateWidth
		if gate, ok := c.Buildings[l.Gate]; ok && gw == 0 {
			gw = gate.Footprint.X.Div(wall.Footprint.X).Round().Int()
		}
		gw = max(gw, 1)
		for _, g := range l.GatePos {
			idx := make([]int, len(l.Segments))
			for i := range idx {
				idx[i] = i
			}
			slices.SortStableFunc(idx, func(a, b int) int {
				da, db := l.Segments[a].Offset.DistSq(g), l.Segments[b].Offset.DistSq(g)
				switch {
				case da < db:
					return -1
				case da > db:
					return 1
				}
				return 0
			})
			for _, i := range idx[:min(gw, len(idx))] {
				l.Segments[i].Gate = true
			}
		}
		l.Segments = mergeGates(l, l.Segments)
	}
	// Stairs: inside, next to each gate (on both sides), then spread evenly.
	inward := wall.Footprint.Y.DivInt(2) + fixed.FromInt(3)
	stair := func(i int) {
		i = (i%len(l.Segments) + len(l.Segments)) % len(l.Segments)
		if l.Segments[i].Gate || slices.ContainsFunc(l.StairList, func(s WallStair) bool { return s.Node == i }) {
			return
		}
		s := l.Segments[i]
		l.StairList = append(l.StairList, WallStair{Node: i, Ground: s.Offset.Sub(s.Out.Scale(inward))})
	}
	for i, s := range l.Segments {
		if !s.Gate {
			continue
		}
		prev, next := l.Segments[(i+len(l.Segments)-1)%len(l.Segments)], l.Segments[(i+1)%len(l.Segments)]
		if !prev.Gate {
			stair(i - 1)
		}
		if !next.Gate {
			stair(i + 1)
		}
	}
	for k := 0; k < l.Stairs; k++ {
		stair((2*k + 1) * len(l.Segments) / (2 * l.Stairs))
	}
	slices.SortFunc(l.StairList, func(a, b WallStair) int { return a.Node - b.Node })
	// Tower sockets on the wall become plots of class "tower".
	for _, t := range l.TowerSockets {
		b.Plots = append(b.Plots, Plot{Offset: t, Size: c.PlotSizes[PlotTower], Class: PlotTower})
	}
}

// mergeGates turns each run of gate segments into one gate slot (a gatehouse
// building) at their middle; the loop starts on a wall segment.
func mergeGates(l *BaseLayout, segs []WallSlot) []WallSlot {
	start := slices.IndexFunc(segs, func(s WallSlot) bool { return !s.Gate })
	if start < 0 {
		return segs
	}
	ordered := append(slices.Clone(segs[start:]), segs[:start]...)
	var out []WallSlot
	for i := 0; i < len(ordered); {
		if !ordered[i].Gate {
			out = append(out, ordered[i])
			i++
			continue
		}
		var sum, dir fixed.Vec2
		j := i
		for j < len(ordered) && ordered[j].Gate {
			sum, dir = sum.Add(ordered[j].Offset), dir.Add(ordered[j].Out)
			j++
		}
		n := j - i
		centre := sum.DivScalar(fixed.FromInt(n))
		if l.Shape == "circle" {
			centre = centre.Normalize().Scale(l.Radius) // on the circle, like the segments
		}
		out = append(out, WallSlot{Offset: centre, Out: dir.Normalize(), Gate: true, Span: n})
		i = j
	}
	return out
}

// circleWithGates lays a round wall: a gatehouse at each gate angle, then the
// arc to the next gate in evenly spaced wall segments, in loop order.
func (c *Catalog) circleWithGates(l *BaseLayout, wall, gate *Building) {
	full := fixed.FromInt(360)
	deg := func(length fixed.Fixed) fixed.Fixed { // arc length at the wall radius → degrees
		return length.Div(l.Radius).MulInt(180).Div(fixed.Pi)
	}
	gates := slices.Clone(l.Gates)
	for i := range gates {
		for gates[i] < 0 {
			gates[i] += full
		}
		for gates[i] >= full {
			gates[i] -= full
		}
	}
	slices.Sort(gates)
	half := deg(gate.Footprint.X).DivInt(2)
	span := max(1, gate.Footprint.X.Div(wall.Footprint.X).Round().Int())
	for i, g := range gates {
		l.Segments = append(l.Segments, WallSlot{Offset: polar(g, l.Radius), Out: polar(g, fixed.One), Gate: true, Span: span})
		next := gates[(i+1)%len(gates)]
		if i == len(gates)-1 {
			next += full
		}
		from, to := g+half, next-half
		if to <= from {
			continue
		}
		m := max(1, (to - from).Mul(fixed.Pi).DivInt(180).Mul(l.Radius).Div(wall.Footprint.X).Ceil().Int())
		for k := 0; k < m; k++ {
			a := from + (to - from).MulInt(2*k+1).DivInt(2*m)
			l.Segments = append(l.Segments, WallSlot{Offset: polar(a, l.Radius), Out: polar(a, fixed.One)})
		}
	}
}
