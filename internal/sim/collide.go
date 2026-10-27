package sim

import "dnd-rts/internal/fixed"

// Soldier-vs-soldier separation using a uniform spatial hash.
//
// Overlapping soldiers are pushed apart, split by mass (a knight shoves
// infantry aside). Pairs are processed in soldier order, so the result is
// deterministic.

// separationCell must be >= the largest unit diameter so a 3x3 lookup finds every overlap.
var separationCell = fixed.FromInt(2)

// spatialHash buckets soldier indices by grid cell. Buffers are reused between
// ticks; it is scratch data, not game state.
type spatialHash struct {
	cell       fixed.Fixed
	cols, rows int
	head       []int32 // first soldier index per cell, -1 = empty
	next       []int32 // next soldier index in the same cell
	touched    []int32 // cells written last build (reset cheaply next time)
}

func newSpatialHash(w *World, cell fixed.Fixed) *spatialHash {
	sh := &spatialHash{
		cell: cell,
		cols: fixed.FromInt(w.Terrain.Width).Div(cell).Int() + 1,
		rows: fixed.FromInt(w.Terrain.Height).Div(cell).Int() + 1,
	}
	sh.head = make([]int32, sh.cols*sh.rows)
	for i := range sh.head {
		sh.head[i] = -1
	}
	return sh
}

// build re-buckets all soldiers. Each bucket lists soldiers in ascending index order.
func (sh *spatialHash) build(w *World) {
	for _, k := range sh.touched {
		sh.head[k] = -1
	}
	sh.touched = sh.touched[:0]
	if cap(sh.next) < len(w.Soldiers) {
		sh.next = make([]int32, len(w.Soldiers))
	}
	sh.next = sh.next[:len(w.Soldiers)]

	for i := len(w.Soldiers) - 1; i >= 0; i-- {
		c, r := sh.cellOf(w, w.Soldiers[i].Pos)
		k := r*sh.cols + c
		if sh.head[k] < 0 {
			sh.touched = append(sh.touched, int32(k))
		}
		sh.next[i] = sh.head[k]
		sh.head[k] = int32(i)
	}
}

func (sh *spatialHash) cellOf(w *World, p fixed.Vec2) (int, int) {
	c := (p.X + fixed.FromInt(w.Terrain.Width).DivInt(2)).Div(sh.cell).Int()
	r := (p.Y + fixed.FromInt(w.Terrain.Height).DivInt(2)).Div(sh.cell).Int()
	return max(0, min(c, sh.cols-1)), max(0, min(r, sh.rows-1))
}

func (w *World) separate() {
	n := len(w.Soldiers)
	if n < 2 {
		return
	}
	before := make([]fixed.Vec2, n)
	for i, s := range w.Soldiers {
		before[i] = s.Pos
	}

	// Two relaxation passes: one pass leaves chain overlaps in dense packs.
	for pass := 0; pass < 2; pass++ {
		w.separatePass()
	}
	for i, s := range w.Soldiers {
		s.stayOnWalkable(w, before[i]) // undo pushes into blocked ground
	}
}

func (w *World) separatePass() {
	sh := w.sepHash
	sh.build(w)
	for i, a := range w.Soldiers {
		ca, ra := sh.cellOf(w, a.Pos)
		for dr := -1; dr <= 1; dr++ {
			for dc := -1; dc <= 1; dc++ {
				c, r := ca+dc, ra+dr
				if c < 0 || r < 0 || c >= sh.cols || r >= sh.rows {
					continue
				}
				for j := sh.head[r*sh.cols+c]; j >= 0; j = sh.next[j] {
					if int(j) <= i {
						continue // each pair once, lower index first
					}
					resolve(a, w.Soldiers[j])
				}
			}
		}
	}
}

func resolve(a, b *Soldier) {
	if a.Horde.Unit.Flying != b.Horde.Unit.Flying || (a.Horde.wall != nil) != (b.Horde.wall != nil) {
		return // the air, the walls and the ground don't push each other
	}
	ra, rb := a.Horde.Unit.Radius, b.Horde.Unit.Radius
	minDist := ra + rb
	d := b.Pos.Sub(a.Pos)
	if d.X.Abs() >= minDist || d.Y.Abs() >= minDist {
		return // cheap reject
	}
	distSq := d.LenSq()
	if distSq >= minDist.Mul(minDist) {
		return
	}

	dist := distSq.Sqrt()
	var normal fixed.Vec2
	if dist == 0 {
		// Exactly on top of each other: pick a direction from the IDs.
		normal = fixed.FromAngle(fixed.FromInt(int(a.ID*7+b.ID*13) % 6))
	} else {
		normal = d.DivScalar(dist)
	}

	push := minDist - dist
	ma, mb := a.Horde.Unit.Mass, b.Horde.Unit.Mass
	total := ma + mb
	a.Pos = a.Pos.Sub(normal.Scale(push.Mul(mb).Div(total)))
	b.Pos = b.Pos.Add(normal.Scale(push.Mul(ma).Div(total)))
}
