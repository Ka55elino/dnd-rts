package nav

import "container/heap"

const (
	costStraight = 10
	costDiagonal = 14
	unreachable  = int32(-1)
)

// Field is a flow field towards one target cell: for every cell, the next cell
// on a shortest path. Shared by every horde heading to the same cell.
type Field struct {
	Target int
	Cost   []int32 // path cost to target, -1 = unreachable
	Next   []int32 // next cell index towards target, -1 = none
}

// Reachable reports whether cell i has a path to the target.
func (f *Field) Reachable(i int) bool { return f.Cost[i] >= 0 }

// BuildField runs Dijkstra from target over walkable cells (8-connected, no
// corner cutting). Ties are broken by cell index, so the result is deterministic.
func (g *Grid) BuildField(target int) *Field {
	n := len(g.Blocked)
	f := &Field{Target: target, Cost: make([]int32, n), Next: make([]int32, n)}
	for i := range f.Cost {
		f.Cost[i] = unreachable
		f.Next[i] = unreachable
	}
	if g.Blocked[target] {
		return f
	}

	f.Cost[target] = 0
	q := &pq{{cost: 0, idx: int32(target)}}
	for q.Len() > 0 {
		cur := heap.Pop(q).(item)
		if cur.cost > f.Cost[cur.idx] {
			continue // stale entry
		}
		c, r := int(cur.idx)%g.Cols, int(cur.idx)/g.Cols
		for _, d := range neighbors8 {
			nc, nr := c+d.dc, r+d.dr
			if nc < 0 || nr < 0 || nc >= g.Cols || nr >= g.Rows {
				continue
			}
			ni := g.Index(nc, nr)
			if g.Blocked[ni] {
				continue
			}
			if d.diagonal && (g.Blocked[g.Index(c+d.dc, r)] || g.Blocked[g.Index(c, r+d.dr)]) {
				continue // don't squeeze between two blocked corners
			}
			nCost := cur.cost + d.cost
			if f.Cost[ni] < 0 || nCost < f.Cost[ni] {
				f.Cost[ni] = nCost
				f.Next[ni] = cur.idx // from ni, step to cur
				heap.Push(q, item{cost: nCost, idx: int32(ni)})
			}
		}
	}
	return f
}

type step struct {
	dc, dr   int
	cost     int32
	diagonal bool
}

var neighbors8 = [8]step{
	{1, 0, costStraight, false}, {-1, 0, costStraight, false},
	{0, 1, costStraight, false}, {0, -1, costStraight, false},
	{1, 1, costDiagonal, true}, {1, -1, costDiagonal, true},
	{-1, 1, costDiagonal, true}, {-1, -1, costDiagonal, true},
}

type item struct {
	cost int32
	idx  int32
}

type pq []item

func (q pq) Len() int { return len(q) }
func (q pq) Less(i, j int) bool {
	if q[i].cost != q[j].cost {
		return q[i].cost < q[j].cost
	}
	return q[i].idx < q[j].idx
}
func (q pq) Swap(i, j int) { q[i], q[j] = q[j], q[i] }
func (q *pq) Push(x any)   { *q = append(*q, x.(item)) }
func (q *pq) Pop() any {
	old := *q
	it := old[len(old)-1]
	*q = old[:len(old)-1]
	return it
}

// Cache keeps recently used fields. Lookups are by key only (never iterated),
// so it does not affect determinism.
type Cache struct {
	grid   *Grid
	fields map[int]*Field
	order  []int // FIFO for eviction
	limit  int
}

func NewCache(g *Grid, limit int) *Cache {
	return &Cache{grid: g, fields: map[int]*Field{}, limit: limit}
}

// Grid returns the underlying walkability grid.
func (c *Cache) Grid() *Grid { return c.grid }

// Invalidate drops all cached fields (call after the grid changes).
func (c *Cache) Invalidate() {
	clear(c.fields)
	c.order = c.order[:0]
}

// Field returns the flow field towards target cell, building it if needed.
func (c *Cache) Field(target int) *Field {
	if f, ok := c.fields[target]; ok {
		return f
	}
	f := c.grid.BuildField(target)
	if len(c.order) >= c.limit {
		delete(c.fields, c.order[0])
		c.order = c.order[1:]
	}
	c.fields[target] = f
	c.order = append(c.order, target)
	return f
}
