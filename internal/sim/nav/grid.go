package nav

import (
	"dnd-rts/internal/fixed"
	"dnd-rts/internal/terrain"
)

// Grid is the walkability grid of a map. Cell (c, r) covers world
// [-W/2 + c*CellSize, -W/2 + (c+1)*CellSize) on X (Y likewise).
type Grid struct {
	Cols, Rows    int
	CellSize      fixed.Fixed
	Width, Height int
	Blocked       []bool // row-major
}

// FromTerrain marks cells steeper than maxSlope (rise per unit) as blocked,
// plus a one-cell border so units stay inside the map.
func FromTerrain(hf *terrain.Heightfield, cellSize, maxSlope fixed.Fixed) *Grid {
	cols := fixed.FromInt(hf.Width).Div(cellSize).Ceil().Int()
	rows := fixed.FromInt(hf.Height).Div(cellSize).Ceil().Int()
	g := &Grid{
		Cols: cols, Rows: rows, CellSize: cellSize,
		Width: hf.Width, Height: hf.Height,
		Blocked: make([]bool, cols*rows),
	}

	half := cellSize.DivInt(2)
	probes := []fixed.Vec2{
		{}, {X: cellSize}, {Y: cellSize}, {X: cellSize, Y: cellSize}, {X: half, Y: half},
	}
	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			i := r*cols + c
			if c == 0 || r == 0 || c == cols-1 || r == rows-1 {
				g.Blocked[i] = true
				continue
			}
			origin := g.cellOrigin(c, r)
			lo, hi := fixed.Max, fixed.Min
			for _, p := range probes {
				h := hf.HeightAt(origin.Add(p))
				lo, hi = min(lo, h), max(hi, h)
			}
			g.Blocked[i] = (hi - lo).Div(cellSize) > maxSlope
		}
	}
	return g
}

// Cell returns the cell containing p (clamped to the grid).
func (g *Grid) Cell(p fixed.Vec2) (c, r int) {
	c = (p.X + fixed.FromInt(g.Width).DivInt(2)).Div(g.CellSize).Int()
	r = (p.Y + fixed.FromInt(g.Height).DivInt(2)).Div(g.CellSize).Int()
	return max(0, min(c, g.Cols-1)), max(0, min(r, g.Rows-1))
}

// Index returns the flat index of cell (c, r).
func (g *Grid) Index(c, r int) int { return r*g.Cols + c }

// IndexAt returns the flat index of the cell containing p.
func (g *Grid) IndexAt(p fixed.Vec2) int { return g.Index(g.Cell(p)) }

// Center returns the world center of cell index i.
func (g *Grid) Center(i int) fixed.Vec2 {
	half := g.CellSize.DivInt(2)
	return g.cellOrigin(i%g.Cols, i/g.Cols).Add(fixed.Vec2{X: half, Y: half})
}

// WalkableAt reports whether p lies in a walkable cell.
func (g *Grid) WalkableAt(p fixed.Vec2) bool { return !g.Blocked[g.IndexAt(p)] }

// LineWalkable reports whether a corridor along a→b is walkable. It samples
// every quarter cell on the center line and on two parallel lines a quarter
// cell to each side, so paths don't clip the corners of blocked cells.
func (g *Grid) LineWalkable(a, b fixed.Vec2) bool {
	d := b.Sub(a)
	dist := d.Len()
	q := g.CellSize.DivInt(4)
	var side fixed.Vec2
	if dist > 0 {
		side = fixed.Vec2{X: -d.Y, Y: d.X}.Scale(q.Div(dist))
	}
	n := dist.Div(q).Int() + 1
	for k := 0; k <= n; k++ {
		p := fixed.LerpV2(a, b, fixed.FromFraction(k, n))
		if !g.WalkableAt(p) || !g.WalkableAt(p.Add(side)) || !g.WalkableAt(p.Sub(side)) {
			return false
		}
	}
	return true
}

// NearestWalkable returns the walkable cell closest to index i (BFS order),
// or -1 if the whole grid is blocked.
func (g *Grid) NearestWalkable(i int) int {
	if !g.Blocked[i] {
		return i
	}
	seen := make([]bool, len(g.Blocked))
	queue := []int{i}
	seen[i] = true
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if !g.Blocked[cur] {
			return cur
		}
		c, r := cur%g.Cols, cur/g.Cols
		for _, d := range neighbors4 {
			nc, nr := c+d[0], r+d[1]
			if nc < 0 || nr < 0 || nc >= g.Cols || nr >= g.Rows {
				continue
			}
			n := g.Index(nc, nr)
			if !seen[n] {
				seen[n] = true
				queue = append(queue, n)
			}
		}
	}
	return -1
}

func (g *Grid) cellOrigin(c, r int) fixed.Vec2 {
	return fixed.Vec2{
		X: g.CellSize.MulInt(c) - fixed.FromInt(g.Width).DivInt(2),
		Y: g.CellSize.MulInt(r) - fixed.FromInt(g.Height).DivInt(2),
	}
}

var neighbors4 = [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}
